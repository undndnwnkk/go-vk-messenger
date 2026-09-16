package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
	"net/http"
	"time"
)

type Handler struct {
	user service.UserService
}

func NewHandler(user service.UserService) http.Handler {
	h := &Handler{user: user}
	r := chi.NewRouter()

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.registerHandler)
		r.Post("/login", h.loginHandler)
	})
	r.Group(func(r chi.Router) {
		r.Use(JWTMiddleware)
		r.Get("/me", h.meHandler)
	})

	return r
}

func (h *Handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid arguments: "+err.Error())
		return
	}

	token, err := h.user.Register(r.Context(), req)
	if err != nil {
		status, code := errorListener(err)
		WriteError(w, status, code, err.Error())
		return
	}

	res := make(map[string]string, 1)
	res["token"] = token
	WriteJSON(w, http.StatusCreated, res)
}

func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	token, err := h.user.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrIncorrectPassword) {
			WriteError(w, http.StatusUnauthorized, "invalid_credentials", service.ErrIncorrectPassword.Error())
			return
		}
		status, code := errorListener(err)
		WriteError(w, status, code, err.Error())
		return
	}

	res := make(map[string]string, 1)
	res["token"] = token
	WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}
	user, err := h.user.Me(r.Context(), userID)
	if err != nil {
		status, code := errorListener(err)
		WriteError(w, status, code, err.Error())
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

func errorListener(err error) (int, string) {
	if errors.Is(err, service.ErrInvalidUsername) || errors.Is(err, service.ErrUserNotFound) {
		return http.StatusBadRequest, "invalid_username"
	}

	if errors.Is(err, service.ErrUserAlreadyExists) {
		return http.StatusConflict, "user_exists"
	}

	if errors.Is(err, service.ErrShortPassword) {
		return http.StatusBadRequest, "short_password"
	}

	if errors.Is(err, service.ErrIncorrectPassword) {
		return http.StatusUnauthorized, "invalid_credentials"
	}

	return http.StatusInternalServerError, "internal_server_error"
}
