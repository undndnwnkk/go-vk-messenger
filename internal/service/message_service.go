package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

type MessageRepositoryInterface interface {
	SetReaction(ctx context.Context, chatID, userID string, messageID int64, reaction string, add bool) (*model.MessageReactions, bool, error)
	GetByID(ctx context.Context, chatID string, messageID int64) (*model.Message, error)
	Edit(ctx context.Context, chatID, senderID string, messageID int64, content string) (*model.Message, error)
	Delete(ctx context.Context, chatID, senderID string, messageID int64) (*model.Message, error)

	Create(
		ctx context.Context,
		chatID string,
		senderID string,
		content string,
		replyToMessageID *int64,
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
	limiter     *MessageRateLimiter
}

func NewMessageService(
	msgRepo MessageRepositoryInterface,
	chatService ChatService,
) *MessageService {
	return NewMessageServiceWithLimiter(msgRepo, chatService, NewMessageRateLimiter())
}

func NewMessageServiceWithLimiter(
	msgRepo MessageRepositoryInterface,
	chatService ChatService,
	limiter *MessageRateLimiter,
) *MessageService {
	return &MessageService{msgRepo: msgRepo, chatService: chatService, limiter: limiter}
}

func (s *MessageService) Send(ctx context.Context, senderID, chatID string, req model.CreateMessageRequest) (*model.Message, error) {
	if err := validateMessageContent(req.Content); err != nil {
		return nil, err
	}
	if req.ReplyToMessageID != nil && *req.ReplyToMessageID <= 0 {
		return nil, ErrInvalidMessageID
	}

	if !s.limiter.Allow(senderID) {
		return nil, ErrRateLimited
	}

	if _, err := s.chatService.GetChatByID(ctx, senderID, chatID); err != nil {
		return nil, fmt.Errorf("check message chat access: %w", err)
	}

	msg, err := s.msgRepo.Create(ctx, chatID, senderID, req.Content, req.ReplyToMessageID)
	if err != nil {
		return nil, messageRepoError(err)
	}

	return msg, nil
}

func (s *MessageService) History(ctx context.Context, userID, chatID string, beforeID *int64, limit int) (*model.MessagePage, error) {
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	// Cap oversized pages; omitted limits are defaulted to 50 by HTTP.
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
	query = strings.TrimSpace(query)
	if query == "" {
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

func validateMessageContent(content string) error {
	if len(strings.TrimSpace(content)) == 0 {
		return ErrEmptyMessage
	}
	if utf8.RuneCountInString(content) > 4000 {
		return ErrMessageTooLong
	}
	return nil
}

func (s *MessageService) Edit(ctx context.Context, userID, chatID string, messageID int64, req model.EditMessageRequest) (*model.Message, error) {
	if err := validateMessageContent(req.Content); err != nil {
		return nil, err
	}
	if err := normalizeChatIDs(&userID, &chatID); err != nil {
		return nil, err
	}
	if err := s.requireMessageAuthor(ctx, userID, chatID, messageID); err != nil {
		return nil, err
	}
	msg, err := s.msgRepo.Edit(ctx, chatID, userID, messageID, req.Content)
	return msg, messageRepoError(err)
}

func (s *MessageService) Delete(ctx context.Context, userID, chatID string, messageID int64) (*model.Message, error) {
	if err := normalizeChatIDs(&userID, &chatID); err != nil {
		return nil, err
	}
	if err := s.requireMessageAuthor(ctx, userID, chatID, messageID); err != nil {
		return nil, err
	}
	msg, err := s.msgRepo.Delete(ctx, chatID, userID, messageID)
	return msg, messageRepoError(err)
}

func (s *MessageService) requireMessageAuthor(ctx context.Context, userID, chatID string, messageID int64) error {
	if messageID <= 0 {
		return ErrInvalidMessageID
	}
	if _, err := s.chatService.GetChatByID(ctx, userID, chatID); err != nil {
		return fmt.Errorf("check message chat access: %w", err)
	}
	msg, err := s.msgRepo.GetByID(ctx, chatID, messageID)
	if err != nil {
		return messageRepoError(err)
	}
	if msg.Deleted {
		return ErrMessageNotFound
	}
	if msg.SenderID != userID {
		return ErrNotMessageAuthor
	}
	return nil
}

func messageRepoError(err error) error {
	if errors.Is(err, repository.ErrMessageNotFound) {
		return ErrMessageNotFound
	}
	return err
}

var (
	ErrNotMessageAuthor = errors.New("only the message author can edit or delete it")
	ErrEmptyMessage     = errors.New("message cannot be empty")
	ErrMessageTooLong   = errors.New("message is too long")
	ErrInvalidLimit     = errors.New("invalid limit")
	ErrInvalidCursor    = errors.New("invalid cursor")
	ErrEmptySearchQuery = errors.New("search query cannot be empty")
	ErrRateLimited      = errors.New("too many messages")
)
