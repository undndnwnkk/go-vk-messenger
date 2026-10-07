package restapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
)

func (h *Handler) setChatMuteHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req model.SetChatMuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Muted == nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "muted must be a boolean")
		return
	}
	state, changed, err := h.chat.SetMute(r.Context(), userID, chi.URLParam(r, "chatID"), *req.Muted)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	if changed {
		h.hub.SendToUser(userID, realtime.MustEventJSON(realtime.EventChatMuteUpdated, state))
	}
	WriteJSON(w, http.StatusOK, state)
}
