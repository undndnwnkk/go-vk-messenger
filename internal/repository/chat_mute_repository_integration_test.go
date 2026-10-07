package repository

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func testChatMutePostgres(t *testing.T, ctx context.Context, pool *pgxpool.Pool, users []string) {
	repo := NewChatRepository(pool)
	group, err := repo.CreateGroup(ctx, users[0], "Mute", users[1:2])
	if err != nil {
		t.Fatal(err)
	}
	direct, err := repo.GetOrCreateDirect(ctx, users[0], users[1])
	if err != nil {
		t.Fatal(err)
	}
	for _, chat := range []*model.Chat{group, direct} {
		initial, err := repo.GetByIDForUser(ctx, chat.ID, users[1])
		if err != nil || initial.Muted || initial.MuteVersion != 0 {
			t.Fatalf("default=%+v err=%v", initial, err)
		}
		state, changed, err := repo.SetMute(ctx, chat.ID, users[1], true)
		if err != nil || !changed || !state.Muted || state.Version != 1 {
			t.Fatalf("mute=%+v changed=%v err=%v", state, changed, err)
		}
		state, changed, err = repo.SetMute(ctx, chat.ID, users[1], true)
		if err != nil || changed || state.Version != 1 {
			t.Fatalf("repeat=%+v changed=%v err=%v", state, changed, err)
		}
		own, err := repo.GetByIDForUser(ctx, chat.ID, users[1])
		if err != nil || !own.Muted || own.MuteVersion != 1 {
			t.Fatalf("own=%+v err=%v", own, err)
		}
		other, err := repo.GetByIDForUser(ctx, chat.ID, users[0])
		if err != nil || other.Muted || other.MuteVersion != 0 {
			t.Fatalf("other=%+v err=%v", other, err)
		}
		if chat.Type == model.Direct {
			found, err := repo.GetOrCreateDirect(ctx, users[1], users[0])
			if err != nil || !found.Muted || found.MuteVersion != 1 {
				t.Fatalf("existing direct=%+v err=%v", found, err)
			}
		}
	}
	msg, err := NewMessageRepository(pool).Create(ctx, group.ID, users[0], "while muted", nil)
	if err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListByUser(ctx, users[1])
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range list {
		if c.ID == group.ID {
			found = true
			if !c.Muted || c.UnreadCount != 1 {
				t.Fatalf("muted unread=%+v", c)
			}
		}
	}
	if !found {
		t.Fatal("muted group missing")
	}
	if _, advanced, err := repo.MarkRead(ctx, group.ID, users[1], msg.ID); err != nil || !advanced {
		t.Fatalf("muted mark read: %v", err)
	}
	if _, _, err := repo.SetMute(ctx, group.ID, users[2], true); !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("outsider: %v", err)
	}
	if _, _, err := repo.SetMute(ctx, "00000000-0000-0000-0000-000000000000", users[1], true); !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("missing chat: %v", err)
	}
	var changes atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state, changed, err := repo.SetMute(ctx, group.ID, users[1], false)
			if err != nil {
				t.Error(err)
				return
			}
			if state.Version != 2 || state.Muted {
				t.Errorf("unmute=%+v", state)
			}
			if changed {
				changes.Add(1)
			}
		}()
	}
	wg.Wait()
	if changes.Load() != 1 {
		t.Fatalf("concurrent changes=%d", changes.Load())
	}
	if _, _, err := repo.SetMute(ctx, group.ID, users[1], true); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveMember(ctx, group.ID, users[1]); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.SetMute(ctx, group.ID, users[1], false); !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("removed member: %v", err)
	}
	if err := repo.AddMember(ctx, group.ID, users[1], model.Member); err != nil {
		t.Fatal(err)
	}
	rejoined, err := repo.GetByIDForUser(ctx, group.ID, users[1])
	if err != nil || rejoined.Muted || rejoined.MuteVersion != 0 {
		t.Fatalf("rejoined=%+v err=%v", rejoined, err)
	}
}
