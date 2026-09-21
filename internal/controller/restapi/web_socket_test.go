package restapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
)

func TestWebSocketWithoutJWT(t *testing.T) {
	handler, _ := newTestHandler(t, newFakeUserRepo())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestWebSocketMissingUserIDBeforeUpgrade(t *testing.T) {
	h := &Handler{}
	server := httptest.NewServer(http.HandlerFunc(h.webSocketHandler))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if conn != nil {
		defer conn.CloseNow()
	}
	if err == nil {
		t.Fatal("upgrade succeeded without user ID in context")
	}
	if resp == nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("response = %v, want HTTP 500 before upgrade; error: %v", resp, err)
	}
}

func TestWebSocketWithJWTAndDisconnect(t *testing.T) {
	for _, abrupt := range []bool{false, true} {
		name := "normal_close"
		if abrupt {
			name = "abrupt_disconnect"
		}
		t.Run(name, func(t *testing.T) {
			hub := realtime.NewHub()
			handler, jwtService := newTestHandlerWithHub(t, newFakeUserRepo(), hub)
			done := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/ws" {
					defer close(done)
				}
				handler.ServeHTTP(w, r)
			}))
			defer server.Close()
			token, err := jwtService.GenerateToken("user-1")
			if err != nil {
				t.Fatalf("generate token: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", &websocket.DialOptions{
				HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
			})
			if err != nil {
				t.Fatalf("dial with JWT: %v", err)
			}
			defer conn.CloseNow()
			if resp.StatusCode != http.StatusSwitchingProtocols {
				t.Fatalf("status = %d, want 101", resp.StatusCode)
			}

			waitForConnectionCount(t, ctx, hub, "user-1", 1)
			if abrupt {
				err = conn.CloseNow()
			} else {
				err = conn.Close(websocket.StatusNormalClosure, "done")
			}
			if err != nil {
				t.Fatalf("close connection: %v", err)
			}
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatal("handler did not return after disconnect")
			}
			if got := hub.ConnectionCount("user-1"); got != 0 {
				t.Fatalf("connections after disconnect = %d, want 0", got)
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/health", nil)
			if err != nil {
				t.Fatal(err)
			}
			healthResp, err := server.Client().Do(req)
			if err != nil {
				t.Fatalf("server unavailable after disconnect: %v", err)
			}
			defer healthResp.Body.Close()
			if healthResp.StatusCode != http.StatusOK {
				t.Fatalf("health status after disconnect = %d, want 200", healthResp.StatusCode)
			}
		})
	}
}

func TestWebSocketPingPongAndEventErrors(t *testing.T) {
	hub := realtime.NewHub()
	handler, jwtService := newTestHandlerWithHub(t, newFakeUserRepo(), hub)
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ws" {
			defer close(done)
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	token, err := jwtService.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
	})
	if err != nil {
		t.Fatalf("dial with JWT: %v", err)
	}
	defer conn.CloseNow()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}

	writeTextMessage(t, ctx, conn, `{"type":"ping"}`)
	assertWebSocketEvent(t, ctx, conn, "pong", "")

	writeTextMessage(t, ctx, conn, `{asd`)
	assertWebSocketEvent(t, ctx, conn, "error", "invalid_event")

	writeTextMessage(t, ctx, conn, `{"type":"wtf"}`)
	assertWebSocketEvent(t, ctx, conn, "error", "unsupported_event")

	writeTextMessage(t, ctx, conn, `{"type":"ping"}`)
	assertWebSocketEvent(t, ctx, conn, "pong", "")

	if err := conn.Close(websocket.StatusNormalClosure, "done"); err != nil {
		t.Fatalf("close connection: %v", err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("handler did not return after disconnect")
	}
}

func TestWebSocketReceivesHubMessage(t *testing.T) {
	hub := realtime.NewHub()
	handler, jwtService := newTestHandlerWithHub(t, newFakeUserRepo(), hub)
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ws" {
			defer close(done)
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	token, err := jwtService.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
	})
	if err != nil {
		t.Fatalf("dial with JWT: %v", err)
	}
	defer conn.CloseNow()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}

	waitForConnectionCount(t, ctx, hub, "user-1", 1)
	hub.SendToUser("user-1", []byte("hello"))

	msgType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read hub message: %v", err)
	}
	if msgType != websocket.MessageText {
		t.Fatalf("message type = %v, want %v", msgType, websocket.MessageText)
	}
	if string(data) != "hello" {
		t.Fatalf("message = %q, want %q", data, "hello")
	}

	if err := conn.Close(websocket.StatusNormalClosure, "done"); err != nil {
		t.Fatalf("close connection: %v", err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("handler did not return after disconnect")
	}
}

func TestWebSocketHubShutdownClosesActiveConnection(t *testing.T) {
	hub := realtime.NewHub()
	handler, jwtService := newTestHandlerWithHub(t, newFakeUserRepo(), hub)
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ws" {
			defer close(done)
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	token, err := jwtService.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
	})
	if err != nil {
		t.Fatalf("dial with JWT: %v", err)
	}
	defer conn.CloseNow()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}
	waitForConnectionCount(t, ctx, hub, "user-1", 1)

	hub.Shutdown()
	_, _, err = conn.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
		t.Fatalf("close status = %v, want %v; err=%v", websocket.CloseStatus(err), websocket.StatusNormalClosure, err)
	}

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("handler did not return after hub shutdown")
	}
	if got := hub.ConnectionCount("user-1"); got != 0 {
		t.Fatalf("connections after shutdown = %d, want 0", got)
	}
}

func TestWebSocketHubShutdownClosesAllUserConnections(t *testing.T) {
	hub := realtime.NewHub()
	handler, jwtService := newTestHandlerWithHub(t, newFakeUserRepo(), hub)
	done := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ws" {
			defer func() { done <- struct{}{} }()
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	token, err := jwtService.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn1 := dialWebSocket(t, ctx, server.URL, token)
	defer conn1.CloseNow()
	conn2 := dialWebSocket(t, ctx, server.URL, token)
	defer conn2.CloseNow()
	waitForConnectionCount(t, ctx, hub, "user-1", 2)

	hub.Shutdown()
	assertClosedWithStatus(t, ctx, conn1, websocket.StatusNormalClosure)
	assertClosedWithStatus(t, ctx, conn2, websocket.StatusNormalClosure)
	hub.Shutdown()

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("handler did not return after hub shutdown")
		}
	}
	if got := hub.ConnectionCount("user-1"); got != 0 {
		t.Fatalf("connections after shutdown = %d, want 0", got)
	}
}

func writeTextMessage(t *testing.T, ctx context.Context, conn *websocket.Conn, message string) {
	t.Helper()

	if err := conn.Write(ctx, websocket.MessageText, []byte(message)); err != nil {
		t.Fatalf("write websocket message %q: %v", message, err)
	}
}

func assertWebSocketEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, wantType, wantCode string) {
	t.Helper()

	msgType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read websocket event: %v", err)
	}
	if msgType != websocket.MessageText {
		t.Fatalf("message type = %v, want %v", msgType, websocket.MessageText)
	}

	var event struct {
		Type  string `json:"type"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode websocket event %q: %v", data, err)
	}
	if event.Type != wantType {
		t.Fatalf("event type = %q, want %q; body: %s", event.Type, wantType, data)
	}
	if event.Error.Code != wantCode {
		t.Fatalf("event error code = %q, want %q; body: %s", event.Error.Code, wantCode, data)
	}
}

func assertClosedWithStatus(t *testing.T, ctx context.Context, conn *websocket.Conn, want websocket.StatusCode) {
	t.Helper()

	_, _, err := conn.Read(ctx)
	if websocket.CloseStatus(err) != want {
		t.Fatalf("close status = %v, want %v; err=%v", websocket.CloseStatus(err), want, err)
	}
}

func waitForConnectionCount(t *testing.T, ctx context.Context, hub *realtime.Hub, userID string, want int) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if got := hub.ConnectionCount(userID); got == want {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("ConnectionCount(%q) did not become %d before timeout", userID, want)
		}
	}
}
