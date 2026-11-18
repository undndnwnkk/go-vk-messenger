package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

func (r *messageRepoStub) SetReaction(_ context.Context, chat, user string, id int64, reaction string, add bool) (*model.MessageReactions, bool, error) {
	r.called, r.chat, r.sender, r.reactionID, r.reactionName, r.reactionAdd = true, chat, user, id, reaction, add
	return &model.MessageReactions{ChatID: chat, MessageID: id, Reactions: []model.ReactionSummary{}}, r.reactionChanged, r.err
}

func TestReactionService(t *testing.T) {
	for _, add := range []bool{false, true} {
		for _, tc := range []struct {
			name, chat, reaction     string
			id                       int64
			accessErr, repoErr, want error
			called                   bool
		}{
			{name: "Success", chat: messageChat, id: 42, reaction: "like", called: true},
			{name: "InvalidChat", chat: "bad", id: 42, reaction: "like", want: ErrInvalidID},
			{name: "InvalidMessage", chat: messageChat, id: 0, reaction: "like", want: ErrInvalidMessageID},
			{name: "InvalidReaction", chat: messageChat, id: 42, reaction: "unknown", want: ErrInvalidReaction},
			{name: "Outsider", chat: messageChat, id: 42, reaction: "like", accessErr: repository.ErrChatNotFound, want: ErrChatNotFound},
			{name: "DeletedMissingOtherChat", chat: messageChat, id: 42, reaction: "like", repoErr: repository.ErrMessageNotFound, want: ErrMessageNotFound, called: true},
		} {
			t.Run(tc.name+map[bool]string{true: "Add", false: "Remove"}[add], func(t *testing.T) {
				repo := &messageRepoStub{err: tc.repoErr, reactionChanged: true}
				state, changed, err := messageTestService(repo, tc.accessErr).SetReaction(context.Background(), strings.ToUpper(messageUser), strings.ToUpper(tc.chat), tc.id, tc.reaction, add)
				if !errors.Is(err, tc.want) || repo.called != tc.called {
					t.Fatalf("err=%v called=%v", err, repo.called)
				}
				if err == nil && (!changed || state.MessageID != 42 || repo.sender != messageUser || repo.chat != messageChat || repo.reactionAdd != add || repo.reactionName != tc.reaction) {
					t.Fatalf("state=%+v repo=%+v", state, repo)
				}
			})
		}
	}
}
