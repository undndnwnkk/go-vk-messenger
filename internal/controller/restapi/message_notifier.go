package restapi

import (
	"context"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

type MessageNotifier struct {
	chat service.ChatService
	hub  *realtime.Hub
}

func NewMessageNotifier(chat service.ChatService, hub *realtime.Hub) *MessageNotifier {
	return &MessageNotifier{chat: chat, hub: hub}
}

func (n *MessageNotifier) NotifyMessageCreated(ctx context.Context, currentUserID string, msg *model.Message) error {
	members, err := n.chat.GetMembersByChatID(ctx, currentUserID, msg.ChatID)
	if err != nil {
		return err
	}

	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	n.hub.SendToUsers(userIDs, realtime.MustEventJSON(realtime.EventMessageCreated, msg))
	return nil
}
