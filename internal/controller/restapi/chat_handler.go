package restapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func (h *Handler) createDirectChatHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateDirectChat

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	chat, err := h.chat.CreateDirectChat(r.Context(), userID, req.UserID2)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, chat)
}

func (h *Handler) createGroupChatHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateGroupChat

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	chat, err := h.chat.CreateGroupChat(r.Context(), userID, req)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, chat)
}

func (h *Handler) getChatsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	chats, err := h.chat.GetChats(r.Context(), userID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, chats)
}

func (h *Handler) getChatByIDHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	chat, err := h.chat.GetChatByID(r.Context(), userID, chi.URLParam(r, "chatID"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, chat)
}

func (h *Handler) getChatMembersHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	members, err := h.chat.GetMembersByChatID(r.Context(), userID, chi.URLParam(r, "chatID"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, members)
}

func (h *Handler) addChatMemberHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	err := h.chat.AddMember(r.Context(), userID, chi.URLParam(r, "chatID"), chi.URLParam(r, "userID"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteChatMemberHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	err := h.chat.RemoveMember(r.Context(), userID, chi.URLParam(r, "chatID"), chi.URLParam(r, "userID"))
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
