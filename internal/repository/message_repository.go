package repository

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}
func (r *MessageRepository) Create(
	ctx context.Context,
	chatID string,
	senderID string,
	content string,
) (*model.Message, error) {
	var res model.Message

	if err := r.db.QueryRow(
		ctx,
		createMessageQuery,
		chatID, senderID, content,
	).Scan(&res.ID, &res.ChatID, &res.SenderID, &res.Content, &res.CreatedAt); err != nil {
		return nil, fmt.Errorf("error creating message: %w", err)
	}

	return &res, nil
}

func (r *MessageRepository) ListBefore(
	ctx context.Context,
	chatID string,
	beforeID *int64,
	limit int,
) ([]model.Message, error) {
	messages := make([]model.Message, 0)

	rows, err := r.db.Query(ctx,
		beforeIDQuery,
		chatID,
		beforeID,
		limit,
	)

	if err != nil {
		return nil, fmt.Errorf("error list before: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("error rows next: %w", err)
		}
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return messages, nil
}

func (r *MessageRepository) Search(
	ctx context.Context,
	chatID string,
	query string,
	limit int,
) ([]model.Message, error) {
	messages := make([]model.Message, 0)

	rows, err := r.db.Query(ctx, searchMessagesQuery, chatID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("error searching messages: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("error reading messages: %w", err)
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return messages, nil
}

var (
	createMessageQuery = `
		INSERT INTO messages (chat_id, sender_id, content) 
		VALUES ($1, $2, $3)
		RETURNING id, chat_id, sender_id, content, created_at`
	beforeIDQuery = `
		SELECT id, chat_id, sender_id, content, created_at 
		FROM messages
		WHERE chat_id = $1 AND ($2::bigint IS NULL OR id < $2)
		ORDER BY id DESC
		LIMIT $3
	`
	searchMessagesQuery = `
		SELECT
    		id,
    		chat_id,
    		sender_id,
    		content,
    		created_at
		FROM messages
		WHERE chat_id = $1
  			AND content ILIKE '%' || $2 || '%'
		ORDER BY id DESC
		LIMIT $3;
	`
)
