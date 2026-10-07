package restapi

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

func (h *Handler) setReactionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	messageID, err := strconv.ParseInt(chi.URLParam(r, "messageID"), 10, 64)
	if err != nil {
		WriteServiceError(w, service.ErrInvalidMessageID)
		return
	}
	state, changed, err := h.message.SetReaction(r.Context(), userID, chi.URLParam(r, "chatID"), messageID, chi.URLParam(r, "reaction"), r.Method == http.MethodPut)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	if changed {
		if err := h.notifier.NotifyReactionsUpdated(r.Context(), userID, state); err != nil {
			log.Printf("reactions broadcast failed: %v", err)
		}
	}
	WriteJSON(w, http.StatusOK, state)
}
