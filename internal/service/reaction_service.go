package service

import (
	"context"
	"errors"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

var ErrInvalidReaction = errors.New("reaction must be like, heart, laugh, wow, sad or angry")

func (s *MessageService) SetReaction(ctx context.Context, userID, chatID string, messageID int64, reaction string, add bool) (*model.MessageReactions, bool, error) {
	if err := normalizeChatIDs(&userID, &chatID); err != nil {
		return nil, false, err
	}
	if messageID <= 0 {
		return nil, false, ErrInvalidMessageID
	}
	switch reaction {
	case "like", "heart", "laugh", "wow", "sad", "angry":
	default:
		return nil, false, ErrInvalidReaction
	}
	if _, err := s.chatService.GetChatByID(ctx, userID, chatID); err != nil {
		return nil, false, err
	}
	state, changed, err := s.msgRepo.SetReaction(ctx, chatID, userID, messageID, reaction, add)
	return state, changed, messageRepoError(err)
}
