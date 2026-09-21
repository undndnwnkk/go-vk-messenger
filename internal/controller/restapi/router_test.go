package restapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

type fakeUserRepo struct {
	usersByID       map[string]model.User
	usersByUsername map[string]model.User
	nextID          int
}

type fakeHealthChecker struct {
	err error
}

func (h fakeHealthChecker) Ping(_ context.Context) error {
	return h.err
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		usersByID:       make(map[string]model.User),
		usersByUsername: make(map[string]model.User),
	}
}

func (r *fakeUserRepo) Create(_ context.Context, user model.User) (string, error) {
	if _, ok := r.usersByUsername[user.Username]; ok {
		return "", repository.ErrUsernameTaken
	}

	r.nextID++
	user.ID = "user-" + strconv.Itoa(r.nextID)
	user.CreatedAt = time.Now().UTC()
	r.usersByID[user.ID] = user
	r.usersByUsername[user.Username] = user
	return user.ID, nil
}

func (r *fakeUserRepo) GetByUsername(_ context.Context, username string) (*model.User, error) {
	user, ok := r.usersByUsername[username]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return &user, nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	user, ok := r.usersByID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return &user, nil
}

func newTestHandler(t *testing.T, repo *fakeUserRepo) (http.Handler, *service.JWTService) {
	t.Helper()
	return newTestHandlerWithHub(t, repo, realtime.NewHub())
}

func newTestHandlerWithHub(t *testing.T, repo *fakeUserRepo, hub *realtime.Hub) (http.Handler, *service.JWTService) {
	t.Helper()

	jwtService := service.NewJWTService("test-secret", 15*time.Minute)
	userService := service.NewUserService(repo, jwtService)
	return NewHandler(*userService, jwtService, fakeHealthChecker{}, service.ChatService{}, service.MessageService{}, hub), jwtService
}

func postJSON(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestRegisterHandler(t *testing.T) {
	repo := newFakeUserRepo()
	handler, jwtService := newTestHandler(t, repo)

	rec := postJSON(t, handler, "/api/v1/auth/register", `{"username":"Alice","password":"verysecret"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	saved, ok := repo.usersByUsername["alice"]
	if !ok {
		t.Fatal("registered user was not saved")
	}
	if saved.PasswordHash == "verysecret" || saved.PasswordHash == "" {
		t.Fatalf("password hash was not stored correctly: %q", saved.PasswordHash)
	}

	var registerResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&registerResponse); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registerResponse.TokenType != "Bearer" {
		t.Fatalf("token_type = %q, want %q", registerResponse.TokenType, "Bearer")
	}
	if _, err := jwtService.ValidateToken(registerResponse.AccessToken); err != nil {
		t.Fatalf("register returned invalid token: %v", err)
	}

	rec = postJSON(t, handler, "/api/v1/auth/register", `{"username":"alice","password":"verysecret"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want %d", rec.Code, http.StatusConflict)
	}

	rec = postJSON(t, handler, "/api/v1/auth/register", `{"username":"","password":"verysecret"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid register status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLoginHandler(t *testing.T) {
	repo := newFakeUserRepo()
	handler, jwtService := newTestHandler(t, repo)
	if rec := postJSON(t, handler, "/api/v1/auth/register", `{"username":"bob","password":"verysecret"}`); rec.Code != http.StatusCreated {
		t.Fatalf("register fixture status = %d; body: %s", rec.Code, rec.Body.String())
	}

	rec := postJSON(t, handler, "/api/v1/auth/login", `{"username":"BOB","password":"verysecret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var loginResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResponse.TokenType != "Bearer" {
		t.Fatalf("token_type = %q, want %q", loginResponse.TokenType, "Bearer")
	}
	if _, err := jwtService.ValidateToken(loginResponse.AccessToken); err != nil {
		t.Fatalf("login returned invalid token: %v", err)
	}

	wrongPassword := postJSON(t, handler, "/api/v1/auth/login", `{"username":"bob","password":"wrongpassword"}`)
	wrongUsername := postJSON(t, handler, "/api/v1/auth/login", `{"username":"missing","password":"verysecret"}`)
	if wrongPassword.Code != http.StatusUnauthorized || wrongUsername.Code != http.StatusUnauthorized {
		t.Fatalf("invalid login statuses = %d/%d, want %d/%d", wrongPassword.Code, wrongUsername.Code, http.StatusUnauthorized, http.StatusUnauthorized)
	}
	if wrongPassword.Body.String() != wrongUsername.Body.String() {
		t.Fatalf("invalid login responses differ: %q vs %q", wrongPassword.Body.String(), wrongUsername.Body.String())
	}
}

func TestHealthHandler(t *testing.T) {
	repo := newFakeUserRepo()
	jwtService := service.NewJWTService("test-secret", 15*time.Minute)
	userService := service.NewUserService(repo, jwtService)
	handler := NewHandler(*userService, jwtService, fakeHealthChecker{}, service.ChatService{}, service.MessageService{}, realtime.NewHub())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	handler = NewHandler(*userService, jwtService, fakeHealthChecker{err: errors.New("database down")}, service.ChatService{}, service.MessageService{}, realtime.NewHub())
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("health unavailable status = %d, want %d; body: %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestMeHandlerAndJWTMiddleware(t *testing.T) {
	repo := newFakeUserRepo()
	handler, jwtService := newTestHandler(t, repo)
	if rec := postJSON(t, handler, "/api/v1/auth/register", `{"username":"carol","password":"verysecret"}`); rec.Code != http.StatusCreated {
		t.Fatalf("register fixture status = %d; body: %s", rec.Code, rec.Body.String())
	}

	token, err := jwtService.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me without token status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("me with token status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "password_hash") || strings.Contains(body, "PasswordHash") || strings.Contains(body, repo.usersByID["user-1"].PasswordHash) {
		t.Fatalf("me response leaked password hash: %s", body)
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, service.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	expiredTokenString, err := expiredToken.SignedString(jwtService.Secret)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredTokenString)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me with expired token status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token+"tampered")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me with tampered token status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
