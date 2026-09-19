package restapi

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
	"net/http"
	"strconv"
)

func (h *Handler) createMessageHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	var req model.CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	res, err := h.message.Send(r.Context(), userID, chi.URLParam(r, "chatID"), req)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, res)
}

func (h *Handler) getMessagesHistoryHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	limitNum := 50
	params := r.URL.Query()
	if params.Has("limit") {
		n, err := strconv.Atoi(params.Get("limit"))
		if err != nil {
			WriteServiceError(w, service.ErrInvalidLimit)
			return
		}
		limitNum = n
	}
	var beforeIDRes *int64
	if params.Has("before_id") {
		n, err := strconv.ParseInt(params.Get("before_id"), 10, 64)
		if err != nil {
			WriteServiceError(w, service.ErrInvalidCursor)
			return
		}
		beforeIDRes = &n
	}

	messages, err := h.message.History(r.Context(), userID, chi.URLParam(r, "chatID"), beforeIDRes, limitNum)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, messages)
}

func (h *Handler) searchMessagesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	query := r.URL.Query().Get("q")

	messages, err := h.message.Search(r.Context(), userID, chi.URLParam(r, "chatID"), query)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, messages)
}
