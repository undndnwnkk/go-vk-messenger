package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

// Locking the message serializes reactions with other reactions and message deletion.
func (r *MessageRepository) SetReaction(ctx context.Context, chatID, userID string, messageID int64, reaction string, add bool) (*model.MessageReactions, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin reaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var deleted bool
	err = tx.QueryRow(ctx, "SELECT deleted_at IS NOT NULL FROM messages WHERE chat_id = $1 AND id = $2 FOR UPDATE", chatID, messageID).Scan(&deleted)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && deleted) {
		return nil, false, ErrMessageNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("lock reaction message: %w", err)
	}
	query := "DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND reaction = $3"
	if add {
		query = "INSERT INTO message_reactions(message_id, user_id, reaction) VALUES($1,$2,$3) ON CONFLICT DO NOTHING"
	}
	tag, err := tx.Exec(ctx, query, messageID, userID, reaction)
	if err != nil {
		return nil, false, fmt.Errorf("set reaction: %w", err)
	}
	changed := tag.RowsAffected() > 0
	if changed {
		if _, err := tx.Exec(ctx, "UPDATE messages SET reactions_version = reactions_version + 1 WHERE id = $1", messageID); err != nil {
			return nil, false, fmt.Errorf("advance reactions version: %w", err)
		}
	}
	var state model.MessageReactions
	err = tx.QueryRow(ctx, "SELECT chat_id, id, reactions_version, "+messageReactionsJSON+" FROM messages WHERE id = $1", messageID).Scan(&state.ChatID, &state.MessageID, &state.Version, &state.Reactions)
	if err != nil {
		return nil, false, fmt.Errorf("read reactions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit reaction: %w", err)
	}
	return &state, changed, nil
}

const messageReactionsJSON = `CASE WHEN messages.deleted_at IS NOT NULL THEN '[]'::jsonb ELSE
	COALESCE((SELECT jsonb_agg(summary ORDER BY summary.reaction) FROM (
		SELECT reaction, count(*) AS count, array_agg(user_id ORDER BY user_id) AS user_ids
		FROM message_reactions WHERE message_id = messages.id GROUP BY reaction
	) summary), '[]'::jsonb) END AS reactions`
