package restapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

func corsTestHandler(origins []string) (http.Handler, *int) {
	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTeapot)
	})
	return CORSMiddleware(origins)(next), &calls
}

func TestCORSAllowedOriginSimpleRequest(t *testing.T) {
	handler, calls := corsTestHandler([]string{"http://localhost:5173"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if *calls != 1 || rec.Code != http.StatusTeapot {
		t.Fatalf("request must reach next handler: calls=%d code=%d", *calls, rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("credentials must not be allowed, got %q", got)
	}
}

func TestCORSOriginMatchIsCaseInsensitive(t *testing.T) {
	handler, _ := corsTestHandler([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://App.Example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://App.Example.com" {
		t.Fatalf("Allow-Origin = %q", got)
	}
}

func TestCORSDisallowedOriginGetsNoHeaders(t *testing.T) {
	handler, calls := corsTestHandler([]string{"http://localhost:5173"})
	for _, origin := range []string{"http://evil.example", "http://localhost:5174", "https://localhost:5173", "null"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("origin %q: Allow-Origin = %q, want none", origin, got)
		}
	}
	if *calls != 4 {
		t.Fatalf("non-preflight requests must still reach the handler, calls=%d", *calls)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler, calls := corsTestHandler([]string{"http://localhost:5173"})
	tests := []struct {
		origin      string
		wantAllowed bool
	}{
		{"http://localhost:5173", true},
		{"http://evil.example", false},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/chats/1/messages/2", nil)
		req.Header.Set("Origin", tt.origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
		req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("%s: status = %d, want 204", tt.origin, rec.Code)
		}
		h := rec.Header()
		if tt.wantAllowed {
			if h.Get("Access-Control-Allow-Origin") != tt.origin ||
				h.Get("Access-Control-Allow-Methods") != corsAllowMethods ||
				h.Get("Access-Control-Allow-Headers") != corsAllowHeaders ||
				h.Get("Access-Control-Max-Age") != corsMaxAge {
				t.Fatalf("%s: missing preflight headers: %v", tt.origin, h)
			}
		} else if h.Get("Access-Control-Allow-Origin") != "" || h.Get("Access-Control-Allow-Methods") != "" {
			t.Fatalf("%s: disallowed preflight must not get CORS headers: %v", tt.origin, h)
		}
	}
	if *calls != 0 {
		t.Fatalf("preflight must not reach the router, calls=%d", *calls)
	}
}

func TestCORSPlainOptionsIsNotPreflight(t *testing.T) {
	handler, calls := corsTestHandler([]string{"http://localhost:5173"})
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/me", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if *calls != 1 {
		t.Fatalf("OPTIONS without Access-Control-Request-Method must reach the handler")
	}
}

func TestCORSEmptyAllowlistIsNoop(t *testing.T) {
	handler, calls := corsTestHandler(nil)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/me", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if *calls != 1 || len(rec.Header()) != 0 {
		t.Fatalf("empty allowlist must pass through untouched: calls=%d headers=%v", *calls, rec.Header())
	}
}

func TestCORSThroughRouterPreflightOnUnknownMethod(t *testing.T) {
	// chi would answer OPTIONS with 405; the middleware must run before routing.
	router := NewHandler(service.UserService{}, nil, fakeHealthChecker{}, service.ChatService{}, service.MessageService{}, realtime.NewHub(),
		CORSMiddleware([]string{"http://localhost:5173"}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/chats", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("router preflight: code=%d headers=%v", rec.Code, rec.Header())
	}

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("router GET: code=%d headers=%v", rec.Code, rec.Header())
	}
}
