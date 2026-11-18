package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
)

func TestMetricsNormalizeRoutes(t *testing.T) {
	hub := realtime.NewHub()
	m := New(hub)
	r := chi.NewRouter()
	r.Use(m.Middleware)
	r.Get("/chats/{chatID}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, path := range []string{"/chats/private-a?text=secret", "/chats/private-b"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	}
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PRIVATE", "/unknown-secret", nil))
	hub.SetPublisher(func([]string, []byte) error { return errors.New("offline") })
	hub.SendToUser("private-a", []byte(`{}`))
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()
	for _, want := range []string{
		`messenger_http_requests_total{method="GET",route="/chats/{chatID}",status="204"} 2`,
		`messenger_http_requests_total{method="OTHER",route="unmatched",status="405"} 1`,
		`messenger_redis_publish_errors_total 1`,
		`messenger_websocket_connections 0`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing metric %s", want)
		}
	}
	for _, secret := range []string{"private-a", "private-b", "unknown-secret", "text=secret", "PRIVATE"} {
		if strings.Contains(body, secret) {
			t.Errorf("unbounded label leaked: %s", secret)
		}
	}
}

func TestMiddlewarePreservesWebSocketUpgrade(t *testing.T) {
	m := New(realtime.NewHub())
	r := chi.NewRouter()
	r.Use(m.Middleware)
	done := make(chan struct{})
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.CloseNow()
		_ = conn.Write(r.Context(), websocket.MessageText, []byte("hello"))
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.ServeHTTP(w, req)
		close(done)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	_, data, err := conn.Read(ctx)
	if err != nil || string(data) != "hello" {
		t.Fatalf("read: %s, %v", data, err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(rec.Body.String(), `messenger_http_requests_total{method="GET",route="/ws",status="101"} 1`) {
		t.Fatal("upgrade was not counted")
	}
	if strings.Contains(rec.Body.String(), `messenger_http_duration_seconds_count{method="GET",route="/ws"}`) {
		t.Fatal("session duration counted as HTTP latency")
	}
}
