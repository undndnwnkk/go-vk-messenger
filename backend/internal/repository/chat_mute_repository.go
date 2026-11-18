package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func (r *ChatRepository) SetMute(ctx context.Context, chatID, userID string, muted bool) (*model.ChatMuteState, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin mute: %w", err)
	}
	defer tx.Rollback(ctx)
	var state model.ChatMuteState
	err = tx.QueryRow(ctx, `SELECT chat_id, muted, mute_version FROM chat_members
		WHERE chat_id = $1 AND user_id = $2 FOR UPDATE`, chatID, userID).Scan(&state.ChatID, &state.Muted, &state.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, ErrMemberNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("read mute: %w", err)
	}
	changed := state.Muted != muted
	if changed {
		err = tx.QueryRow(ctx, `UPDATE chat_members SET muted = $3, mute_version = mute_version + 1
			WHERE chat_id = $1 AND user_id = $2 RETURNING chat_id, muted, mute_version`, chatID, userID, muted).Scan(&state.ChatID, &state.Muted, &state.Version)
		if err != nil {
			return nil, false, fmt.Errorf("set mute: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit mute: %w", err)
	}
	return &state, changed, nil
}
