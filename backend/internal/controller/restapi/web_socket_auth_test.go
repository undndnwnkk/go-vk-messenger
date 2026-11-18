package restapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

func newWebSocketAuthHandler(t *testing.T, origins []string) (http.Handler, string) {
	t.Helper()
	jwtService := service.NewJWTService("test-secret", 15*time.Minute)
	userService := service.NewUserService(newFakeUserRepo(), jwtService)
	handler := NewHandlerWithOptions(*userService, jwtService, fakeHealthChecker{}, service.ChatService{}, service.MessageService{}, realtime.NewHub(),
		Options{WebSocketOrigins: origins})
	token, err := jwtService.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return handler, token
}

func dialWebSocketAuth(t *testing.T, handler http.Handler, opts *websocket.DialOptions) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", opts)
	if conn != nil {
		t.Cleanup(func() { _ = conn.CloseNow() })
	}
	return conn, resp, err
}

func TestWebSocketSubprotocolAuth(t *testing.T) {
	handler, token := newWebSocketAuthHandler(t, nil)
	conn, resp, err := dialWebSocketAuth(t, handler, &websocket.DialOptions{
		Subprotocols: []string{WebSocketSubprotocol, bearerSubprotocolPrefix + token},
	})
	if err != nil {
		t.Fatalf("dial with bearer subprotocol: %v", err)
	}
	if got := conn.Subprotocol(); got != WebSocketSubprotocol {
		t.Fatalf("negotiated subprotocol = %q, want %q", got, WebSocketSubprotocol)
	}
	if got := resp.Header.Get("Sec-WebSocket-Protocol"); strings.Contains(got, token) {
		t.Fatalf("server echoed the token back: %q", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`)); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read pong: %v", err)
	}
	var event realtime.Event
	if err := json.Unmarshal(data, &event); err != nil || event.Type != realtime.EventPong {
		t.Fatalf("got %s, want pong", data)
	}
}

func TestWebSocketAuthorizationHeaderStillWorks(t *testing.T) {
	handler, token := newWebSocketAuthHandler(t, nil)
	conn, _, err := dialWebSocketAuth(t, handler, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
	})
	if err != nil {
		t.Fatalf("dial with Authorization header: %v", err)
	}
	if got := conn.Subprotocol(); got != "" {
		t.Fatalf("subprotocol = %q, want none when the client offered none", got)
	}
}

func TestWebSocketAuthRejections(t *testing.T) {
	handler, token := newWebSocketAuthHandler(t, nil)
	tests := []struct {
		name          string
		path          string
		authorization string
		protocols     string
		wantStatus    int
		wantCode      string
	}{
		{"no credentials", "/api/v1/ws", "", WebSocketSubprotocol, 401, "missing_header"},
		{"invalid token", "/api/v1/ws", "", WebSocketSubprotocol + ", bearer.not-a-jwt", 401, "invalid_token"},
		{"empty token", "/api/v1/ws", "", WebSocketSubprotocol + ", bearer.", 401, "invalid_authorization"},
		{"two tokens", "/api/v1/ws", "", WebSocketSubprotocol + ", bearer." + token + ", bearer." + token, 401, "invalid_authorization"},
		{"token without app subprotocol", "/api/v1/ws", "", "bearer." + token, 400, "invalid_request"},
		{"bad header does not fall back", "/api/v1/ws", "Token abc", WebSocketSubprotocol + ", bearer." + token, 401, "invalid_authorization"},
		{"REST ignores subprotocol", "/api/v1/me", "", WebSocketSubprotocol + ", bearer." + token, 401, "missing_header"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			req.Header.Set("Sec-WebSocket-Protocol", tt.protocols)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if code := errorCode(t, rec.Body.Bytes()); code != tt.wantCode {
				t.Fatalf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

func TestWebSocketSubprotocolsAcrossHeaders(t *testing.T) {
	// Proxies may split the list into several header lines.
	handler, token := newWebSocketAuthHandler(t, nil)
	conn, _, err := dialWebSocketAuth(t, handler, &websocket.DialOptions{
		HTTPHeader: http.Header{"Sec-WebSocket-Protocol": {"bearer." + token}},
		// DialOptions.Subprotocols adds a second header line.
		Subprotocols: []string{WebSocketSubprotocol},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if conn.Subprotocol() != WebSocketSubprotocol {
		t.Fatalf("subprotocol = %q", conn.Subprotocol())
	}
}

func TestWebSocketOriginAllowlist(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		origin  string
		wantOK  bool
	}{
		{"listed origin", []string{"http://localhost:5173"}, "http://localhost:5173", true},
		{"unlisted origin", []string{"http://localhost:5173"}, "http://evil.example", false},
		{"other port", []string{"http://localhost:5173"}, "http://localhost:5174", false},
		{"other scheme", []string{"http://localhost:5173"}, "https://localhost:5173", false},
		{"empty allowlist keeps cross-origin blocked", nil, "http://localhost:5173", false},
		{"ipv6 origin", []string{"http://[::1]:5173"}, "http://[::1]:5173", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, token := newWebSocketAuthHandler(t, tt.allowed)
			_, resp, err := dialWebSocketAuth(t, handler, &websocket.DialOptions{
				Subprotocols: []string{WebSocketSubprotocol, bearerSubprotocolPrefix + token},
				HTTPHeader:   http.Header{"Origin": {tt.origin}},
			})
			if tt.wantOK {
				if err != nil {
					t.Fatalf("dial from %s: %v", tt.origin, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("dial from %s succeeded, want rejection", tt.origin)
			}
			if resp == nil || resp.StatusCode != http.StatusForbidden {
				t.Fatalf("dial from %s: resp=%v err=%v, want 403", tt.origin, resp, err)
			}
		})
	}
}

func TestWebSocketOriginPatternsEscapeMatchMetacharacters(t *testing.T) {
	patterns := webSocketOriginPatterns([]string{"http://[::1]:5173"})
	if ok, err := path.Match(patterns[0], "http://[::1]:5173"); err != nil || !ok {
		t.Fatalf("pattern %q must match its own origin: ok=%v err=%v", patterns[0], ok, err)
	}
	if ok, _ := path.Match(patterns[0], "http://:"); ok {
		t.Fatalf("pattern %q must not behave as a character class", patterns[0])
	}
	if webSocketOriginPatterns(nil) != nil {
		t.Fatal("empty origins must give nil patterns")
	}
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var res struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("decode error body %q: %v", body, err)
	}
	return res.Error.Code
}
