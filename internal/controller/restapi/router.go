package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
	"log"
	"net/http"
)

type Handler struct {
	user       service.UserService
	chat       service.ChatService
	message    service.MessageService
	jwtService *service.JWTService
	health     HealthChecker
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

func NewHandler(user service.UserService, jwtService *service.JWTService, health HealthChecker, chat service.ChatService, message service.MessageService) http.Handler {
	h := &Handler{user: user, jwtService: jwtService, health: health, chat: chat, message: message}
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

			r.Get("/chats", h.getChatsHandler)
			r.Route("/chats", func(r chi.Router) {
				r.Post("/direct", h.createDirectChatHandler)
				r.Post("/group", h.createGroupChatHandler)

				r.Get("/", h.getChatsHandler)
				r.Get("/{chatID}", h.getChatByIDHandler)

				r.Get("/{chatID}/members", h.getChatMembersHandler)
				r.Route("/{chatID}/members", func(r chi.Router) {
					r.Get("/", h.getChatMembersHandler)
					r.Post("/{userID}", h.addChatMemberHandler)
					r.Delete("/{userID}", h.deleteChatMemberHandler)
				})

				r.Route("/{chatID}/messages", func(r chi.Router) {
					r.Post("/", h.createMessageHandler)
					r.Get("/", h.getMessagesHistoryHandler)
					r.Get("/search", h.searchMessagesHandler)
				})
			})
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

func GetUserIDFromContext(ctx context.Context) (string, bool) {
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
	switch {
	case errors.Is(err, service.ErrEmptyMessage):
		return http.StatusBadRequest, "empty_message", service.ErrEmptyMessage.Error()
	case errors.Is(err, service.ErrMessageTooLong):
		return http.StatusBadRequest, "message_too_long", service.ErrMessageTooLong.Error()
	case errors.Is(err, service.ErrInvalidLimit):
		return http.StatusBadRequest, "invalid_limit", service.ErrInvalidLimit.Error()
	case errors.Is(err, service.ErrInvalidCursor):
		return http.StatusBadRequest, "invalid_cursor", service.ErrInvalidCursor.Error()
	case errors.Is(err, service.ErrEmptySearchQuery):
		return http.StatusBadRequest, "empty_search_query", service.ErrEmptySearchQuery.Error()
	case errors.Is(err, service.ErrInvalidID):
		return http.StatusBadRequest, "invalid_id", service.ErrInvalidID.Error()
	case errors.Is(err, service.ErrTargetUserNotFound):
		return http.StatusNotFound, "user_not_found", service.ErrTargetUserNotFound.Error()
	case errors.Is(err, service.ErrCannotRemoveSelf):
		return http.StatusConflict, "cannot_remove_self", service.ErrCannotRemoveSelf.Error()
	case errors.Is(err, service.ErrChatNotFound):
		return http.StatusNotFound, "chat_not_found", service.ErrChatNotFound.Error()
	case errors.Is(err, service.ErrMemberNotFound):
		return http.StatusNotFound, "member_not_found", service.ErrMemberNotFound.Error()
	case errors.Is(err, service.ErrNotAdmin):
		return http.StatusForbidden, "forbidden", service.ErrNotAdmin.Error()
	case errors.Is(err, service.ErrAlreadyMember):
		return http.StatusConflict, "already_member", service.ErrAlreadyMember.Error()
	case errors.Is(err, service.ErrInvalidTitle):
		return http.StatusBadRequest, "invalid_title", service.ErrInvalidTitle.Error()
	case errors.Is(err, service.ErrCannotModifyDirect):
		return http.StatusBadRequest, "cannot_modify_direct", service.ErrCannotModifyDirect.Error()
	case errors.Is(err, service.ErrCommonUser):
		return http.StatusBadRequest, "same_user", service.ErrCommonUser.Error()
	}
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
