package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"strings"
	"unicode/utf8"
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
	msgRepo     MessageRepositoryInterface
	chatService ChatService
}

func NewMessageService(
	msgRepo MessageRepositoryInterface,
	chatService ChatService,
) *MessageService {
	return &MessageService{msgRepo: msgRepo, chatService: chatService}
}

func (s *MessageService) Send(ctx context.Context, senderID, chatID string, req model.CreateMessageRequest) (*model.Message, error) {
	if len(strings.TrimSpace(req.Content)) == 0 {
		return nil, ErrEmptyMessage
	}

	if utf8.RuneCountInString(req.Content) > 4000 {
		return nil, ErrMessageTooLong
	}

	if _, err := s.chatService.GetChatByID(ctx, senderID, chatID); err != nil {
		return nil, fmt.Errorf("check message chat access: %w", err)
	}

	msg, err := s.msgRepo.Create(ctx, chatID, senderID, req.Content)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *MessageService) History(ctx context.Context, userID, chatID string, beforeID *int64, limit int) (*model.MessagePage, error) {
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	if limit > 100 {
		limit = 100
	}

	if beforeID != nil && *beforeID <= 0 {
		return nil, ErrInvalidCursor
	}

	if _, err := s.chatService.GetChatByID(ctx, userID, chatID); err != nil {
		return nil, fmt.Errorf("check message chat access: %w", err)
	}

	messages, err := s.msgRepo.ListBefore(ctx, chatID, beforeID, limit+1)
	if err != nil {
		return nil, err
	}

	page := &model.MessagePage{Messages: messages}
	if len(messages) > limit {
		page.Messages = messages[:limit]
		cursor := page.Messages[limit-1].ID
		page.NextCursor = &cursor
	}
	return page, nil
}

func (s *MessageService) Search(ctx context.Context, userID, chatID, query string) ([]model.Message, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptySearchQuery
	}

	if _, err := s.chatService.GetChatByID(ctx, userID, chatID); err != nil {
		return nil, fmt.Errorf("check message chat access: %w", err)
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
)
