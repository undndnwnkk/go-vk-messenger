package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func TestChatRepositoryPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("chat_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.MaxConns = 24
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	files, err := filepath.Glob("../../migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("migrations: %v %v", files, err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(string(data), "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("migration %s: %v", file, err)
		}
	}
	repo := NewChatRepository(pool)
	users := make([]string, 4)
	for i := range users {
		if err := pool.QueryRow(ctx, "INSERT INTO users(username,password_hash) VALUES($1,'hash') RETURNING id", fmt.Sprintf("user%d", i)).Scan(&users[i]); err != nil {
			t.Fatal(err)
		}
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	var group *model.Chat
	t.Run("CreateGroupSuccess", func(t *testing.T) {
		group, err = repo.CreateGroup(ctx, users[0], "Group", users[1:3])
		if err != nil {
			t.Fatal(err)
		}
		if group.Type != model.Group || group.Title == nil || *group.Title != "Group" || group.CreatedBy != users[0] || group.CreatedAt.IsZero() {
			t.Fatalf("group: %+v", group)
		}
		members, err := repo.ListMembers(ctx, group.ID)
		if err != nil || len(members) != 3 {
			t.Fatalf("members: %+v %v", members, err)
		}
		for _, user := range users[:3] {
			m, err := repo.GetMember(ctx, group.ID, user)
			if err != nil {
				t.Fatal(err)
			}
			want := model.Member
			if user == users[0] {
				want = model.Admin
			}
			if m.Role != want || m.JoinedAt.IsZero() {
				t.Fatalf("member: %+v", m)
			}
		}
	})
	if group == nil {
		t.Fatal("group creation failed")
	}
	t.Run("CreateGroupRollback", func(t *testing.T) {
		before := count("SELECT count(*) FROM chats")
		if _, err := repo.CreateGroup(ctx, users[0], "Rollback", []string{users[1], "00000000-0000-0000-0000-000000000000"}); err == nil {
			t.Fatal("expected error")
		}
		if count("SELECT count(*) FROM chats") != before {
			t.Fatal("orphan chat")
		}
	})
	t.Run("DirectCreateExistingReverse", func(t *testing.T) {
		low, high := normalizeUserPair(users[0], users[1])
		chat, err := repo.GetOrCreateDirect(ctx, high, low)
		if err != nil {
			t.Fatal(err)
		}
		if chat.Type != model.Direct || chat.Title != nil || chat.CreatedBy != high {
			t.Fatalf("direct: %+v", chat)
		}
		if count("SELECT count(*) FROM chat_members WHERE chat_id=$1", chat.ID) != 2 {
			t.Fatal("member count")
		}
		for _, pair := range [][2]string{{high, low}, {low, high}} {
			existing, err := repo.GetOrCreateDirect(ctx, pair[0], pair[1])
			if err != nil || existing.ID != chat.ID {
				t.Fatalf("existing: %+v %v", existing, err)
			}
		}
	})
	t.Run("GetOrCreateDirectConcurrent", func(t *testing.T) {
		// Hold a table lock so all callers miss the fast path before any insert can commit.
		blocker, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer blocker.Rollback(ctx)
		if _, err := blocker.Exec(ctx, "LOCK TABLE direct_chats IN SHARE MODE"); err != nil {
			t.Fatal(err)
		}
		const n = 20
		results := make(chan *model.Chat, n)
		errs := make(chan error, n)
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				a, b := users[2], users[3]
				if i%2 == 0 {
					a, b = b, a
				}
				chat, err := repo.GetOrCreateDirect(ctx, a, b)
				results <- chat
				errs <- err
			}(i)
		}
		deadline := time.Now().Add(10 * time.Second)
		for count("SELECT count(*) FROM pg_locks WHERE relation='direct_chats'::regclass AND NOT granted") < n {
			if time.Now().After(deadline) {
				t.Fatal("callers did not reach concurrent inserts")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err := blocker.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		close(results)
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		id := ""
		for chat := range results {
			if chat == nil {
				t.Fatal("nil chat")
			}
			if id == "" {
				id = chat.ID
			}
			if chat.ID != id {
				t.Fatal("different chat IDs")
			}
		}
		if count("SELECT count(*) FROM direct_chats WHERE chat_id=$1", id) != 1 || count("SELECT count(*) FROM chat_members WHERE chat_id=$1", id) != 2 || count("SELECT count(*) FROM chats WHERE type='direct'") != 2 {
			t.Fatal("duplicate or orphan rows")
		}
	})
	t.Run("AccessAndLists", func(t *testing.T) {
		if _, err := repo.GetByIDForUser(ctx, group.ID, users[1]); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.GetByIDForUser(ctx, group.ID, users[3]); !errors.Is(err, ErrChatNotFound) {
			t.Fatalf("outsider: %v", err)
		}
		if _, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrChatNotFound) {
			t.Fatalf("missing chat: %v", err)
		}
		chats, err := repo.ListByUser(ctx, users[0])
		if err != nil || len(chats) != 2 {
			t.Fatalf("list: %+v %v", chats, err)
		}
		for i, chat := range chats {
			if _, err := repo.GetMember(ctx, chat.ID, users[0]); err != nil {
				t.Fatal(err)
			}
			if i > 0 && chats[i-1].CreatedAt.Before(chat.CreatedAt) {
				t.Fatal("wrong order")
			}
		}
	})
	t.Run("ReadStatus", func(t *testing.T) {
		messages := NewMessageRepository(pool)
		own, err := messages.Create(ctx, group.ID, users[0], "own")
		if err != nil {
			t.Fatal(err)
		}
		incoming, err := messages.Create(ctx, group.ID, users[1], "incoming")
		if err != nil {
			t.Fatal(err)
		}
		otherChat, err := repo.CreateGroup(ctx, users[0], "Other", nil)
		if err != nil {
			t.Fatal(err)
		}
		other, err := messages.Create(ctx, otherChat.ID, users[0], "other chat")
		if err != nil {
			t.Fatal(err)
		}

		chats, err := repo.ListByUser(ctx, users[0])
		if err != nil {
			t.Fatal(err)
		}
		var groupListItem *model.Chat
		for i := range chats {
			if chats[i].ID == group.ID {
				groupListItem = &chats[i]
				break
			}
		}
		if groupListItem == nil {
			t.Fatal("group not listed")
		}
		if groupListItem.UnreadCount != 1 {
			t.Fatalf("unread count before read = %d, want 1", groupListItem.UnreadCount)
		}

		state, advanced, err := repo.MarkRead(ctx, group.ID, users[0], incoming.ID)
		if err != nil || !advanced || state.ChatID != group.ID || state.UserID != users[0] || state.LastReadMessageID != incoming.ID {
			t.Fatalf("mark read state=%+v advanced=%v err=%v", state, advanced, err)
		}
		state, advanced, err = repo.MarkRead(ctx, group.ID, users[0], own.ID)
		if err != nil || advanced || state.LastReadMessageID != incoming.ID {
			t.Fatalf("stale mark read state=%+v advanced=%v err=%v", state, advanced, err)
		}
		if _, _, err := repo.MarkRead(ctx, group.ID, users[0], other.ID); !errors.Is(err, ErrMessageNotFound) {
			t.Fatalf("wrong-chat message err=%v", err)
		}
		if _, _, err := repo.MarkRead(ctx, group.ID, users[3], incoming.ID); !errors.Is(err, ErrMemberNotFound) {
			t.Fatalf("outsider err=%v", err)
		}

		chats, err = repo.ListByUser(ctx, users[0])
		if err != nil {
			t.Fatal(err)
		}
		for i := range chats {
			if chats[i].ID == group.ID {
				groupListItem = &chats[i]
				break
			}
		}
		if groupListItem.UnreadCount != 0 || groupListItem.LastReadMessageID == nil || *groupListItem.LastReadMessageID != incoming.ID {
			t.Fatalf("list read state after read = %+v", groupListItem)
		}

		if err := repo.AddMember(ctx, group.ID, users[3], model.Member); err != nil {
			t.Fatal(err)
		}
		member, err := repo.GetMember(ctx, group.ID, users[3])
		if err != nil {
			t.Fatal(err)
		}
		if member.LastReadMessageID == nil || *member.LastReadMessageID != incoming.ID {
			t.Fatalf("new member read state = %+v, want %d", member, incoming.ID)
		}
		if err := repo.RemoveMember(ctx, group.ID, users[3]); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("MembershipErrors", func(t *testing.T) {
		if err := repo.RemoveMember(ctx, group.ID, users[3]); !errors.Is(err, ErrMemberNotFound) {
			t.Fatal(err)
		}
		if _, err := repo.GetMember(ctx, group.ID, users[3]); !errors.Is(err, ErrMemberNotFound) {
			t.Fatal(err)
		}
		if err := repo.AddMember(ctx, group.ID, users[3], model.Member); err != nil {
			t.Fatal(err)
		}
		if err := repo.AddMember(ctx, group.ID, users[3], model.Member); !errors.Is(err, ErrMemberAlreadyExists) {
			t.Fatal(err)
		}
		if err := repo.RemoveMember(ctx, group.ID, users[3]); err != nil {
			t.Fatal(err)
		}
	})
	for i := len(files) - 1; i >= 0; i-- {
		data, err := os.ReadFile(files[i])
		if err != nil {
			t.Fatal(err)
		}
		down := strings.Split(string(data), "-- +goose Down")[1]
		if _, err := pool.Exec(ctx, down); err != nil {
			t.Fatalf("down migration: %v", err)
		}
	}
}
