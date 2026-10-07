package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

type muteRepoStub struct {
	ChatRepositoryInterface
	chat, user    string
	muted, called bool
	err           error
}

func (r *muteRepoStub) SetMute(_ context.Context, chat, user string, muted bool) (*model.ChatMuteState, bool, error) {
	r.chat, r.user, r.muted, r.called = chat, user, muted, true
	return &model.ChatMuteState{ChatID: chat, Muted: muted, Version: 1}, true, r.err
}

func TestChatMuteService(t *testing.T) {
	failure := errors.New("database failure")
	for _, muted := range []bool{true, false} {
		for _, tc := range []struct {
			name, chat    string
			repoErr, want error
			called        bool
		}{
			{name: "Success", chat: messageChat, called: true},
			{name: "InvalidChat", chat: "bad", want: ErrInvalidID},
			{name: "NonMember", chat: messageChat, repoErr: repository.ErrMemberNotFound, want: ErrChatNotFound, called: true},
			{name: "DatabaseError", chat: messageChat, repoErr: failure, want: failure, called: true},
		} {
			t.Run(tc.name+map[bool]string{true: "Mute", false: "Unmute"}[muted], func(t *testing.T) {
				repo := &muteRepoStub{err: tc.repoErr}
				svc := NewChatService(repo, UserService{})
				state, changed, err := svc.SetMute(context.Background(), strings.ToUpper(messageUser), strings.ToUpper(tc.chat), muted)
				if !errors.Is(err, tc.want) || repo.called != tc.called {
					t.Fatalf("err=%v called=%v", err, repo.called)
				}
				if err == nil && (!changed || state.Muted != muted || repo.user != messageUser || repo.chat != messageChat) {
					t.Fatalf("state=%+v repo=%+v", state, repo)
				}
			})
		}
	}
}
