package restapi

import (
	"encoding/json"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"net/http"
)

func (h *Handler) createDirectChatHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateDirectChat

	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	chat, err := h.chat.CreateDirectChat(r.Context(), userID, req.UserID2, "direct")
	if err != nil {
		publicError(err)
		return
	}

	WriteJSON(w, http.StatusCreated, chat)
}

func (h *Handler) createGroupChatHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateGroupChat

	if err := json.NewEncoder(w).Encode(&req); err != nil {
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
		publicError(err)
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
		publicError(err)
		return
	}

	WriteJSON(w, http.StatusCreated, chats)
}

func (h *Handler) getChatByIDHandler(w http.ResponseWriter, r *http.Request) {
	var req model.GetChatByID

	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	chat, err := h.chat.GetChatByID(r.Context(), userID, req.ChatID)
	if err != nil {
		publicError(err)
		return
	}

	WriteJSON(w, http.StatusCreated, chat)
}

func (h *Handler) getChatMembersHandler(w http.ResponseWriter, r *http.Request) {
	var req model.GetChatByID

	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	members, err := h.chat.GetMembersByChatID(r.Context(), req.ChatID)
	if err != nil {
		publicError(err)
		return
	}

	WriteJSON(w, http.StatusCreated, members)
}

func (h *Handler) addChatMemberHandler(w http.ResponseWriter, r *http.Request) {
	var req model.AddMember

	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	err := h.chat.AddMember(r.Context(), userID, req.ChatID, req.UserID, req.Role)
	if err != nil {
		publicError(err)
		return
	}

	resp := make(map[string]string, 1)
	resp["message"] = "user successfully added"
	WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) deleteChatMemberHandler(w http.ResponseWriter, r *http.Request) {
	var req model.AddMember

	if err := json.NewEncoder(w).Encode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	err := h.chat.RemoveMember(r.Context(), userID, req.ChatID, req.UserID)
	if err != nil {
		publicError(err)
		return
	}

	resp := make(map[string]string, 1)
	resp["message"] = "user successfully deleted"
	WriteJSON(w, http.StatusCreated, resp)
}
