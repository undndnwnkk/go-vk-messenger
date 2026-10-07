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

func testMessageReactionsPostgres(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	repo, chats := NewMessageRepository(pool), NewChatRepository(pool)
	users := make([]string, 2)
	for i, name := range []string{"reaction_alice", "reaction_bob"} {
		if err := pool.QueryRow(ctx, "INSERT INTO users(username,password_hash) VALUES($1,'hash') RETURNING id", name).Scan(&users[i]); err != nil {
			t.Fatal(err)
		}
	}
	chat, err := chats.CreateGroup(ctx, users[0], "Reactions", users[1:])
	if err != nil {
		t.Fatal(err)
	}
	other, err := chats.GetOrCreateDirect(ctx, users[0], users[1])
	if err != nil {
		t.Fatal(err)
	}
	msg, err := repo.Create(ctx, chat.ID, users[0], "react here", nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Reactions == nil || len(msg.Reactions) != 0 || msg.ReactionsVersion != 0 {
		t.Fatalf("initial=%+v", msg)
	}
	set := func(user, reaction string, add, wantChanged bool, version int64) *model.MessageReactions {
		t.Helper()
		state, changed, err := repo.SetReaction(ctx, chat.ID, user, msg.ID, reaction, add)
		if err != nil || changed != wantChanged || state.Version != version {
			t.Fatalf("state=%+v changed=%v err=%v", state, changed, err)
		}
		return state
	}
	set(users[0], "like", true, true, 1)
	set(users[0], "like", true, false, 1)
	state := set(users[1], "like", true, true, 2)
	if len(state.Reactions) != 1 || state.Reactions[0].Count != 2 || len(state.Reactions[0].UserIDs) != 2 {
		t.Fatalf("aggregation=%+v", state)
	}
	set(users[0], "heart", true, true, 3)
	state = set(users[0], "like", false, true, 4)
	if len(state.Reactions) != 2 || state.Reactions[1].UserIDs[0] != users[1] {
		t.Fatalf("removed another user's reaction: %+v", state)
	}
	set(users[0], "like", false, false, 4)
	for _, read := range []func() ([]model.Message, error){
		func() ([]model.Message, error) { return repo.ListBefore(ctx, chat.ID, nil, 10) },
		func() ([]model.Message, error) { return repo.Search(ctx, chat.ID, "react", 50) },
	} {
		list, err := read()
		if err != nil || len(list) != 1 || len(list[0].Reactions) != 2 || list[0].ReactionsVersion != 4 {
			t.Fatalf("read=%+v err=%v", list, err)
		}
	}
	edited, err := repo.Edit(ctx, chat.ID, users[0], msg.ID, "edited reaction message")
	if err != nil || len(edited.Reactions) != 2 || edited.ReactionsVersion != 4 {
		t.Fatalf("edited=%+v err=%v", edited, err)
	}
	for _, target := range []struct {
		chat string
		id   int64
	}{{other.ID, msg.ID}, {chat.ID, 9223372036854775807}} {
		for _, add := range []bool{false, true} {
			if _, _, err := repo.SetReaction(ctx, target.chat, users[0], target.id, "like", add); !errors.Is(err, ErrMessageNotFound) {
				t.Fatalf("wrong target: %v", err)
			}
		}
	}
	// Repeated concurrent PUTs result in exactly one new reaction and version increment.
	var changedCount atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, changed, err := repo.SetReaction(ctx, chat.ID, users[1], msg.ID, "laugh", true)
			if err != nil {
				t.Error(err)
			}
			if changed {
				changedCount.Add(1)
			}
		}()
	}
	wg.Wait()
	if changedCount.Load() != 1 {
		t.Fatalf("concurrent changes=%d", changedCount.Load())
	}

	// The message lock also protects against adding reactions after concurrent deletion.
	var deleteErr, reactionErr error
	var deleted *model.Message
	wg.Add(2)
	go func() { defer wg.Done(); deleted, deleteErr = repo.Delete(ctx, chat.ID, users[0], msg.ID) }()
	go func() {
		defer wg.Done()
		_, _, reactionErr = repo.SetReaction(ctx, chat.ID, users[1], msg.ID, "wow", true)
	}()
	wg.Wait()
	if deleteErr != nil || (reactionErr != nil && !errors.Is(reactionErr, ErrMessageNotFound)) {
		t.Fatalf("delete=%v reaction=%v", deleteErr, reactionErr)
	}
	if !deleted.Deleted || len(deleted.Reactions) != 0 || deleted.Reactions == nil {
		t.Fatalf("deleted=%+v", deleted)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM message_reactions WHERE message_id=$1", msg.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("remaining reactions=%d err=%v", count, err)
	}
	for _, add := range []bool{true, false} {
		if _, _, err := repo.SetReaction(ctx, chat.ID, users[0], msg.ID, "like", add); !errors.Is(err, ErrMessageNotFound) {
			t.Fatalf("deleted message reaction: %v", err)
		}
	}
}
