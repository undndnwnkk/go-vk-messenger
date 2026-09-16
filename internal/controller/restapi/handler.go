package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	user       service.UserService
	jwtService *service.JWTService
	health     HealthChecker
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

func NewHandler(user service.UserService, jwtService *service.JWTService, health HealthChecker) http.Handler {
	h := &Handler{user: user, jwtService: jwtService, health: health}
	r := chi.NewRouter()

	r.Get("/health", h.healthHandler)
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.registerHandler)
			r.Post("/login", h.loginHandler)
		})
		r.Group(func(r chi.Router) {
			r.Use(JWTMiddleware(h.jwtService))
			r.Get("/me", h.meHandler)
		})
	})

	return r
}

func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.health.Ping(r.Context()); err != nil {
		log.Printf("health check failed: %v", err)
		WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "database unavailable")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid arguments: "+err.Error())
		return
	}

	token, err := h.user.Register(r.Context(), req)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, NewTokenResponse(token))
}

func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	token, err := h.user.Login(r.Context(), req)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, NewTokenResponse(token))
}

func NewTokenResponse(token string) tokenResponse {
	return tokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}
	user, err := h.user.Me(r.Context(), userID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	res := struct {
		ID        string    `json:"id"`
		Username  string    `json:"username"`
		CreatedAt time.Time `json:"created_at"`
	}{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}

	WriteJSON(w, http.StatusOK, res)
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(&data)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	res := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}

	res.Error.Code = code
	res.Error.Message = message

	_ = json.NewEncoder(w).Encode(&res)
}

func getUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

func WriteServiceError(w http.ResponseWriter, err error) {
	status, code, message := publicError(err)
	if status == http.StatusInternalServerError {
		log.Printf("request failed: %v", err)
	}

	WriteError(w, status, code, message)
}

func publicError(err error) (int, string, string) {
	if errors.Is(err, service.ErrInvalidUsername) || errors.Is(err, service.ErrUserNotFound) {
		return http.StatusBadRequest, "invalid_username", err.Error()
	}

	if errors.Is(err, service.ErrUserAlreadyExists) {
		return http.StatusConflict, "user_exists", err.Error()
	}

	if errors.Is(err, service.ErrShortPassword) || errors.Is(err, service.ErrLongPassword) {
		return http.StatusBadRequest, "invalid_password", err.Error()
	}

	if errors.Is(err, service.ErrInvalidCredentials) {
		return http.StatusUnauthorized, "invalid_credentials", err.Error()
	}

	return http.StatusInternalServerError, "internal_error", "internal server error"
}
