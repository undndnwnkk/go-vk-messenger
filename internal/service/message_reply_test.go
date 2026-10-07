package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

func TestMessageReply(t *testing.T) {
	zero, negative, target := int64(0), int64(-1), int64(42)
	for _, tc := range []struct {
		name                     string
		id                       *int64
		accessErr, repoErr, want error
		called                   bool
	}{
		{name: "PlainMessage", called: true},
		{name: "Reply", id: &target, called: true},
		{name: "Zero", id: &zero, want: ErrInvalidMessageID},
		{name: "Negative", id: &negative, want: ErrInvalidMessageID},
		{name: "Outsider", id: &target, accessErr: repository.ErrChatNotFound, want: ErrChatNotFound},
		{name: "MissingDeletedOrOtherChat", id: &target, repoErr: repository.ErrMessageNotFound, want: ErrMessageNotFound, called: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &messageRepoStub{err: tc.repoErr}
			if tc.id != nil {
				repo.reply = &model.MessageReply{ID: *tc.id, SenderID: "other author", Content: "original"}
			}
			msg, err := messageTestService(repo, tc.accessErr).Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "answer", ReplyToMessageID: tc.id})
			if !errors.Is(err, tc.want) || repo.called != tc.called {
				t.Fatalf("err=%v called=%v", err, repo.called)
			}
			if err == nil && (!reflect.DeepEqual(msg.ReplyToMessageID, tc.id) || !reflect.DeepEqual(msg.ReplyTo, repo.reply)) {
				t.Fatalf("reply=%+v", msg)
			}
		})
	}
}
