package service

import (
	"context"
	"errors"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
	"strings"
)

type MessageRepositoryInterface interface {
	Create(
		ctx context.Context,
		chatID string,
		senderID string,
		content string,
	) (*model.Message, error)

	ListBefore(
		ctx context.Context,
		chatID string,
		beforeID *int64,
		limit int,
	) ([]model.Message, error)

	Search(
		ctx context.Context,
		chatID string,
		query string,
		limit int,
	) ([]model.Message, error)
}

type MessageService struct {
	msgRepo     repository.MessageRepository
	chatService ChatService
}

func NewMessageService(
	msgRepo repository.MessageRepository,
	chatService ChatService,
) *MessageService {
	return &MessageService{msgRepo: msgRepo, chatService: chatService}
}

func (s *MessageService) Send(ctx context.Context, senderID string, req model.CreateMessageRequest) (*model.Message, error) {
	if len(strings.TrimSpace(req.Content)) == 0 {
		return nil, ErrEmptyMessage
	}

	if len(req.Content) > 4000 {
		return nil, ErrMessageTooLong
	}

	if _, err := s.chatService.GetChatByID(ctx, senderID, req.ChatID); err != nil {
		return nil, ErrChatNotFound
	}

	msg, err := s.msgRepo.Create(ctx, req.ChatID, senderID, req.Content)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *MessageService) History(ctx context.Context, userID, chatID string, beforeID *int64, limit int) (*model.MessagePage, error) {
	if limit < 0 {
		return nil, ErrInvalidLimit
	}
	if limit < 50 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}

	if *beforeID < 0 {
		return nil, ErrInvalidCursor
	}

	if _, err := s.chatService.GetChatByID(ctx, userID, chatID); err != nil {
		return nil, ErrChatNotFound
	}

	if beforeID == nil {
		id, err := s.msgRepo.GetMaxMessageIDByChatID(ctx, chatID)
		if err != nil || id == nil {
			return nil, ErrChatNotFound
		}
		beforeID = id
	}

	messages, err := s.msgRepo.ListBefore(ctx, chatID, beforeID, limit)
	if err != nil {
		return nil, err
	}

	cursor := messages[len(messages)-1].ID

	response := model.MessagePage{Messages: messages, NextCursor: &cursor}

	return &response, nil
}

func (s *MessageService) Search(ctx context.Context, userID, chatID, query string) ([]model.Message, error) {
	if len(query) == 0 {
		return nil, ErrEmptyQuery
	}

	if _, err := s.chatService.GetChatByID(ctx, userID, chatID); err != nil {
		return nil, ErrChatNotFound
	}

	messages, err := s.msgRepo.Search(ctx, chatID, query, 50)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

var (
	ErrEmptyMessage     = errors.New("message cannot be empty")
	ErrMessageTooLong   = errors.New("message is too long")
	ErrInvalidLimit     = errors.New("invalid limit")
	ErrInvalidCursor    = errors.New("invalid cursor")
	ErrEmptySearchQuery = errors.New("search query cannot be empty")
	ErrEmptyQuery       = errors.New("query cannot be empty")
)
