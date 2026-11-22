package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func newMigratedSchema(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := fmt.Sprintf("summary_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
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
	return pool
}

func TestChatSummariesAndUserSearchPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool := newMigratedSchema(t, ctx)
	chats := NewChatRepository(pool)
	users := NewUserRepository(pool)

	ids := map[string]string{}
	for _, name := range []string{"me", "alice", "bob", "al_x", "alpha"} {
		id, err := users.Create(ctx, model.User{Username: name, PasswordHash: "hash"})
		if err != nil {
			t.Fatal(err)
		}
		ids[name] = id
	}

	withAlice, err := chats.GetOrCreateDirect(ctx, ids["me"], ids["alice"])
	if err != nil {
		t.Fatal(err)
	}
	withBob, err := chats.GetOrCreateDirect(ctx, ids["bob"], ids["me"])
	if err != nil {
		t.Fatal(err)
	}
	group, err := chats.CreateGroup(ctx, ids["me"], "Team", []string{ids["bob"]})
	if err != nil {
		t.Fatal(err)
	}

	future := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	insert := func(chatID, sender, content string, at time.Time) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, "INSERT INTO messages(chat_id, sender_id, content, created_at) VALUES($1,$2,$3,$4) RETURNING id", chatID, sender, content, at).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	insert(withBob.ID, ids["bob"], "old news", future)
	insert(group.ID, ids["bob"], "standup at ten", future.Add(time.Minute))
	deletedID := insert(withBob.ID, ids["me"], "oops", future.Add(2*time.Minute))
	if _, err := pool.Exec(ctx, "UPDATE messages SET deleted_at = now(), content = '' WHERE id = $1", deletedID); err != nil {
		t.Fatal(err)
	}

	t.Run("ListByUser", func(t *testing.T) {
		list, err := chats.ListByUser(ctx, ids["me"])
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 3 || list[0].ID != withBob.ID || list[1].ID != group.ID || list[2].ID != withAlice.ID {
			t.Fatalf("order = %v, want withBob, group, withAlice", chatIDs(list))
		}

		bobChat, groupChat, aliceChat := list[0], list[1], list[2]
		if bobChat.Peer == nil || *bobChat.Peer != (model.UserSummary{ID: ids["bob"], Username: "bob"}) {
			t.Fatalf("direct peer = %+v", bobChat.Peer)
		}
		last := bobChat.LastMessage
		if last == nil || last.ID != deletedID || !last.Deleted || last.Content != "" || last.SenderID != ids["me"] || last.SenderUsername != "me" || !last.CreatedAt.Equal(future.Add(2*time.Minute)) {
			t.Fatalf("deleted last message = %+v", last)
		}
		if bobChat.UnreadCount != 1 {
			t.Fatalf("direct unread = %d, want 1", bobChat.UnreadCount)
		}

		if groupChat.Peer != nil {
			t.Fatalf("group peer = %+v, want nil", groupChat.Peer)
		}
		if groupChat.LastMessage == nil || groupChat.LastMessage.Content != "standup at ten" || groupChat.LastMessage.SenderUsername != "bob" || groupChat.LastMessage.Deleted {
			t.Fatalf("group last message = %+v", groupChat.LastMessage)
		}
		if groupChat.UnreadCount != 1 || groupChat.Title == nil || *groupChat.Title != "Team" {
			t.Fatalf("group = %+v", groupChat)
		}

		if aliceChat.LastMessage != nil || aliceChat.Peer == nil || aliceChat.Peer.Username != "alice" || aliceChat.UnreadCount != 0 {
			t.Fatalf("empty direct chat = %+v", aliceChat)
		}
	})

	t.Run("GetByIDForUserPeerPerspective", func(t *testing.T) {
		fromAlice, err := chats.GetByIDForUser(ctx, withAlice.ID, ids["alice"])
		if err != nil {
			t.Fatal(err)
		}
		if fromAlice.Peer == nil || fromAlice.Peer.ID != ids["me"] || fromAlice.LastMessage != nil {
			t.Fatalf("alice's view = %+v", fromAlice)
		}
		fromMe, err := chats.GetByIDForUser(ctx, group.ID, ids["me"])
		if err != nil {
			t.Fatal(err)
		}
		if fromMe.Peer != nil || fromMe.LastMessage == nil || fromMe.LastMessage.SenderUsername != "bob" {
			t.Fatalf("group view = %+v", fromMe)
		}
	})

	t.Run("Search", func(t *testing.T) {
		cases := []struct {
			caller, prefix string
			limit          int
			want           []string
		}{
			{"me", "al", 10, []string{"al_x", "alice", "alpha"}},
			{"me", "al_", 10, []string{"al_x"}},
			{"me", "%", 10, []string{}},
			{"me", "al", 2, []string{"al_x", "alice"}},
			{"alice", "ali", 10, []string{}},
			{"me", "m", 10, []string{}},
		}
		for _, tc := range cases {
			got, err := users.Search(ctx, ids[tc.caller], tc.prefix, tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			if names := usernames(got); strings.Join(names, ",") != strings.Join(tc.want, ",") {
				t.Errorf("Search(%s, %q, %d) = %v, want %v", tc.caller, tc.prefix, tc.limit, names, tc.want)
			}
		}
	})

	t.Run("ListContacts", func(t *testing.T) {
		got, err := users.ListContacts(ctx, ids["me"], 10)
		if err != nil {
			t.Fatal(err)
		}
		if names := usernames(got); strings.Join(names, ",") != "alice,bob" {
			t.Fatalf("contacts = %v, want alice,bob", names)
		}
		got, err = users.ListContacts(ctx, ids["alpha"], 10)
		if err != nil || len(got) != 0 {
			t.Fatalf("contacts of user without chats = %v %v", got, err)
		}
	})
}

func chatIDs(chats []model.Chat) []string {
	ids := make([]string, len(chats))
	for i, chat := range chats {
		ids[i] = chat.ID
	}
	return ids
}

func usernames(users []model.UserSummary) []string {
	names := make([]string, len(users))
	for i, user := range users {
		names[i] = user.Username
	}
	return names
}
