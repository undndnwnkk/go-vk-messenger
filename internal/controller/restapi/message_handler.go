package restapi

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
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
	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	res, err := h.message.Send(r.Context(), userID, req)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
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

	var req model.HistoryRequest
	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	limit := chi.URLParam(r, "limit")
	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_limit", "limit must be number")
		return
	}

	beforeID := chi.URLParam(r, "before_id")
	var beforeIDRes *int64
	if n, err := strconv.Atoi(beforeID); err != nil {
		beforeIDRes = nil
	} else {
		tmp := int64(n)
		beforeIDRes = &tmp
	}

	messages, err := h.message.History(r.Context(), userID, req.ChatID, beforeIDRes, limitNum)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_server_error", err.Error())
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

	var req model.HistoryRequest
	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	query := chi.URLParam(r, "query")

	messages, err := h.message.Search(r.Context(), userID, req.ChatID, query)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, messages)
}
