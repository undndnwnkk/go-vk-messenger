package service

import (
	"context"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

type MessageRepository interface {
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
