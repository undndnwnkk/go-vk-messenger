package restapi

import (
	"encoding/json"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"net/http"
	"time"
)

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

func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}
	user, err := h.user.GetUserByID(r.Context(), userID)
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
