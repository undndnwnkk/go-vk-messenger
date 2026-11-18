package infrastructure

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
)

// This opt-in smoke test creates disposable users/chats in a running deployment.
// Both URLs must address DIFFERENT replicas sharing one PostgreSQL and Redis.
func TestReplicas(t *testing.T) {
	a, b := os.Getenv("TEST_APP_A_URL"), os.Getenv("TEST_APP_B_URL")
	if a == "" || b == "" {
		t.Skip("set TEST_APP_A_URL and TEST_APP_B_URL for a disposable deployment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	request := func(base, method, path, token string, body any, want int, out any) {
		t.Helper()
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequestWithContext(ctx, method, base+path, bytes.NewReader(encoded))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != want {
			data, _ := io.ReadAll(res.Body)
			t.Fatalf("%s %s: %d %s", method, path, res.StatusCode, data)
		}
		if out != nil {
			if err := json.NewDecoder(res.Body).Decode(out); err != nil {
				t.Fatal(err)
			}
		}
	}
	register := func(base string) (string, string) {
		var token struct {
			AccessToken string `json:"access_token"`
		}
		request(base, "POST", "/api/v1/auth/register", "", map[string]string{"username": "smoke_" + strings.ToLower(rand.Text()[:12]), "password": "smoke-test-password"}, 201, &token)
		var me struct {
			ID string `json:"id"`
		}
		request(base, "GET", "/api/v1/me", token.AccessToken, nil, 200, &me)
		return token.AccessToken, me.ID
	}
	alice, _ := register(a)
	bob, bobID := register(b)
	var chat model.Chat
	request(a, "POST", "/api/v1/chats/direct", alice, map[string]string{"user2_id": bobID}, 200, &chat)
	path := "/api/v1/chats/" + chat.ID
	connect := func(base, token string) (*websocket.Conn, <-chan realtime.Event) {
		t.Helper()
		conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(base, "http")+"/api/v1/ws", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": {"Bearer " + token}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = conn.CloseNow() })
		events := make(chan realtime.Event, 64)
		go func() {
			defer close(events)
			for {
				_, data, err := conn.Read(ctx)
				if err != nil {
					return
				}
				var event realtime.Event
				if json.Unmarshal(data, &event) == nil {
					events <- event
				}
			}
		}()
		return conn, events
	}
	expect := func(events <-chan realtime.Event, typ realtime.EventType) realtime.Event {
		t.Helper()
		select {
		case event, ok := <-events:
			if !ok || event.Type != typ {
				t.Fatalf("event = %s, want %s", event.Type, typ)
			}
			return event
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for %s", typ)
			return realtime.Event{}
		}
	}
	aliceConn, aliceEvents := connect(a, alice)
	bobConn, bobEvents := connect(b, bob)
	aliceOtherConn, aliceOtherDevice := connect(b, alice)
	// Ping confirms local registration before sending through another replica.
	for _, device := range []struct {
		conn   *websocket.Conn
		events <-chan realtime.Event
	}{{aliceConn, aliceEvents}, {bobConn, bobEvents}, {aliceOtherConn, aliceOtherDevice}} {
		if err := device.conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`)); err != nil {
			t.Fatal(err)
		}
		expect(device.events, realtime.EventPong)
	}
	var message model.Message
	request(a, "POST", path+"/messages", alice, map[string]string{"content": "cross-replica smoke"}, 201, &message)
	for _, events := range []<-chan realtime.Event{aliceEvents, bobEvents, aliceOtherDevice} {
		event := expect(events, realtime.EventMessageCreated)
		var received model.Message
		if err := json.Unmarshal(event.Data, &received); err != nil || received.ID != message.ID {
			t.Fatalf("wrong relayed message: %s", event.Data)
		}
	}
	messagePath := fmt.Sprintf("%s/messages/%d", path, message.ID)
	request(b, "PUT", messagePath+"/reactions/like", bob, nil, 200, nil)
	for _, events := range []<-chan realtime.Event{aliceEvents, bobEvents, aliceOtherDevice} {
		expect(events, realtime.EventReactionsUpdated)
	}
	request(a, "PATCH", path+"/mute", alice, map[string]bool{"muted": true}, 200, nil)
	expect(aliceEvents, realtime.EventChatMuteUpdated)
	expect(aliceOtherDevice, realtime.EventChatMuteUpdated)
	select {
	case event := <-bobEvents:
		t.Fatalf("private mute leaked to peer: %s", event.Type)
	case <-time.After(150 * time.Millisecond):
	}
	request(b, "GET", path, alice, nil, 200, &chat)
	if !chat.Muted {
		t.Fatal("mute was not persisted across replicas")
	}
	request(a, "PATCH", messagePath, alice, map[string]string{"content": "edited across replicas"}, 200, nil)
	for _, events := range []<-chan realtime.Event{aliceEvents, bobEvents, aliceOtherDevice} {
		expect(events, realtime.EventMessageEdited)
	}
	request(a, "DELETE", messagePath, alice, nil, 200, nil)
	for _, events := range []<-chan realtime.Event{aliceEvents, bobEvents, aliceOtherDevice} {
		expect(events, realtime.EventMessageDeleted)
	}

	// A fresh user has an empty window. Split a concurrent burst across replicas.
	limited, _ := register(a)
	request(a, "POST", "/api/v1/chats/direct", limited, map[string]string{"user2_id": bobID}, 200, &chat)
	statuses := make(chan int, 24)
	var wg sync.WaitGroup
	start := time.Now()
	for i := range 24 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			base := a
			if i%2 == 1 {
				base = b
			}
			req, _ := http.NewRequestWithContext(ctx, "POST", base+"/api/v1/chats/"+chat.ID+"/messages", strings.NewReader(`{"content":"rate limit smoke"}`))
			req.Header.Set("Authorization", "Bearer "+limited)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				statuses <- 0
				return
			}
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			statuses <- res.StatusCode
		}(i)
	}
	wg.Wait()
	close(statuses)
	if time.Since(start) >= 5*time.Second {
		t.Fatal("burst exceeded the rate-limit window; test deployment is too slow")
	}
	accepted, rejected := 0, 0
	for status := range statuses {
		switch status {
		case 201:
			accepted++
		case 429:
			rejected++
		default:
			t.Errorf("unexpected burst status: %d", status)
		}
	}
	if accepted != 10 || rejected != 14 {
		t.Fatalf("accepted=%d rejected=%d, want shared 10/14", accepted, rejected)
	}
}
