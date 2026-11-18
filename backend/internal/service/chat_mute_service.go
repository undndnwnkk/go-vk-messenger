package service

import (
	"context"
	"errors"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

func (s *ChatService) SetMute(ctx context.Context, userID, chatID string, muted bool) (*model.ChatMuteState, bool, error) {
	if err := normalizeChatIDs(&userID, &chatID); err != nil {
		return nil, false, err
	}
	state, changed, err := s.repo.SetMute(ctx, chatID, userID, muted)
	if errors.Is(err, repository.ErrMemberNotFound) {
		return nil, false, ErrChatNotFound
	}
	return state, changed, chatRepoError(err)
}
