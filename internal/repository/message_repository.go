package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	replyToMessageID *int64,
) (*model.Message, error) {
	res, err := scanMessage(r.db.QueryRow(
		ctx,
		createMessageQuery,
		chatID, senderID, content, replyToMessageID,
	))
	if err != nil {
		return nil, fmt.Errorf("error creating message: %w", err)
	}

	return res, nil
}

func scanMessage(row pgx.Row) (*model.Message, error) {
	var msg model.Message
	err := row.Scan(&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Content, &msg.CreatedAt, &msg.EditedAt, &msg.DeletedAt, &msg.ReplyToMessageID, &msg.ReplyTo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	msg.Edited = msg.EditedAt != nil
	msg.Deleted = msg.DeletedAt != nil
	return &msg, nil
}

func (r *MessageRepository) GetByID(ctx context.Context, chatID string, messageID int64) (*model.Message, error) {
	msg, err := scanMessage(r.db.QueryRow(ctx, `SELECT `+messageColumns+` FROM messages WHERE chat_id = $1 AND id = $2`, chatID, messageID))
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}
	return msg, nil
}

func (r *MessageRepository) Edit(ctx context.Context, chatID, senderID string, messageID int64, content string) (*model.Message, error) {
	msg, err := scanMessage(r.db.QueryRow(ctx, `
		UPDATE messages SET content = $4, edited_at = clock_timestamp()
		WHERE chat_id = $1 AND sender_id = $2 AND id = $3 AND deleted_at IS NULL
		RETURNING `+messageColumns, chatID, senderID, messageID, content))
	if err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}
	return msg, nil
}

func (r *MessageRepository) Delete(ctx context.Context, chatID, senderID string, messageID int64) (*model.Message, error) {
	// Retain the row so existing read cursors and history pagination remain valid.
	msg, err := scanMessage(r.db.QueryRow(ctx, `
		UPDATE messages SET content = '', deleted_at = clock_timestamp()
		WHERE chat_id = $1 AND sender_id = $2 AND id = $3 AND deleted_at IS NULL
		RETURNING `+messageColumns, chatID, senderID, messageID))
	if err != nil {
		return nil, fmt.Errorf("delete message: %w", err)
	}
	return msg, nil
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
		m, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("error rows next: %w", err)
		}
		messages = append(messages, *m)
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
		m, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("error reading messages: %w", err)
		}

		messages = append(messages, *m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return messages, nil
}

// Fetch the current parent in the same query/snapshot, including for RETURNING.
// No recursive reply expansion and no additional query per message.
const messageColumns = `id, chat_id, sender_id, content, created_at, edited_at, deleted_at, reply_to_message_id,
	(SELECT jsonb_build_object(
		'id', parent.id, 'sender_id', parent.sender_id, 'content', parent.content,
		'edited', parent.edited_at IS NOT NULL, 'edited_at', parent.edited_at,
		'deleted', parent.deleted_at IS NOT NULL, 'deleted_at', parent.deleted_at
	) FROM messages parent
	WHERE parent.chat_id = messages.chat_id AND parent.id = messages.reply_to_message_id) AS reply_to`

var (
	createMessageQuery = `
		INSERT INTO messages (chat_id, sender_id, content, reply_to_message_id)
		SELECT $1, $2, $3, $4::bigint
		WHERE $4::bigint IS NULL OR EXISTS (
			SELECT 1 FROM messages
			WHERE chat_id = $1 AND id = $4 AND deleted_at IS NULL
		)
		RETURNING ` + messageColumns
	beforeIDQuery = `
		SELECT ` + messageColumns + `
		FROM messages
		WHERE chat_id = $1 AND ($2::bigint IS NULL OR id < $2)
		ORDER BY id DESC
		LIMIT $3
	`
	searchMessagesQuery = `
		SELECT ` + messageColumns + `
		FROM messages
		WHERE chat_id = $1 AND deleted_at IS NULL
  			AND content ILIKE '%' || $2 || '%'
		ORDER BY id DESC
		LIMIT $3;
	`
)
