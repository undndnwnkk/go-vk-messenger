package restapi

import (
	"context"
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

			// CloseRead rejects application messages, so Ping also verifies that
			// the server consumed our message without sending an echo.
			readCtx := conn.CloseRead(ctx)
			if err := conn.Write(ctx, websocket.MessageText, []byte("ignored message")); err != nil {
				t.Fatalf("write message: %v", err)
			}
			if err := conn.Ping(ctx); err != nil {
				t.Fatalf("ping after message: %v", err)
			}
			if readCtx.Err() != nil {
				t.Fatalf("connection closed after message: %v", readCtx.Err())
			}
			if got := hub.ConnectionCount("user-1"); got != 1 {
				t.Fatalf("connections after ping = %d, want 1", got)
			}
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
