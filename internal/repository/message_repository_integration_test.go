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

func TestMessageRepositoryPostgres(t *testing.T) {
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
	schema := fmt.Sprintf("message_test_%d", time.Now().UnixNano())
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
	config.MaxConns = 2
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

	repo := NewMessageRepository(pool)
	var userID string
	if err := pool.QueryRow(ctx, "INSERT INTO users(username,password_hash) VALUES('sender','hash') RETURNING id").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	newChat := func() string {
		t.Helper()
		chat, err := NewChatRepository(pool).CreateGroup(ctx, userID, "Messages", nil)
		if err != nil {
			t.Fatal(err)
		}
		return chat.ID
	}
	create := func(chatID, content string) *model.Message {
		t.Helper()
		msg, err := repo.Create(ctx, chatID, userID, content)
		if err != nil {
			t.Fatal(err)
		}
		if msg.ID <= 0 || msg.ChatID != chatID || msg.SenderID != userID || msg.Content != content || msg.CreatedAt.IsZero() {
			t.Fatalf("unexpected message: %+v", msg)
		}
		return msg
	}
	assertIDs := func(t *testing.T, messages []model.Message, want []int64) {
		t.Helper()
		if len(messages) != len(want) {
			t.Fatalf("got %d messages, want %d: %+v", len(messages), len(want), messages)
		}
		for i, id := range want {
			if messages[i].ID != id {
				t.Fatalf("message %d: got ID %d, want %d", i, messages[i].ID, id)
			}
		}
	}
	chatID, otherChatID, emptyChatID := newChat(), newChat(), newChat()
	ids := make([]int64, 10)
	t.Run("Create", func(t *testing.T) {
		for i := range ids {
			ids[i] = create(chatID, fmt.Sprintf("Message %d", i+1)).ID
		}
		var persisted model.Message
		if err := pool.QueryRow(ctx, "SELECT id, chat_id, sender_id, content, created_at FROM messages WHERE id=$1", ids[0]).Scan(&persisted.ID, &persisted.ChatID, &persisted.SenderID, &persisted.Content, &persisted.CreatedAt); err != nil {
			t.Fatal(err)
		}
		if persisted.ChatID != chatID || persisted.SenderID != userID || persisted.Content != "Message 1" || persisted.CreatedAt.IsZero() {
			t.Fatalf("persisted: %+v", persisted)
		}
	})
	if ids[9] == 0 {
		t.Fatal("message creation failed")
	}
	other := create(otherChatID, "MESSAGE from another chat")
	t.Run("FirstPage", func(t *testing.T) {
		page, err := repo.ListBefore(ctx, chatID, nil, 3)
		if err != nil {
			t.Fatal(err)
		}
		assertIDs(t, page, []int64{ids[9], ids[8], ids[7]})
	})
	t.Run("SecondPage", func(t *testing.T) {
		page, err := repo.ListBefore(ctx, chatID, &ids[7], 3)
		if err != nil {
			t.Fatal(err)
		}
		assertIDs(t, page, []int64{ids[6], ids[5], ids[4]})
	})
	t.Run("PagesDescendingWithoutDuplicates", func(t *testing.T) {
		var cursor *int64
		seen := make(map[int64]bool)
		for pageIndex, want := range [][]int64{
			{ids[9], ids[8], ids[7]}, {ids[6], ids[5], ids[4]}, {ids[3], ids[2], ids[1]}, {ids[0]}, {},
		} {
			page, err := repo.ListBefore(ctx, chatID, cursor, 3)
			if err != nil {
				t.Fatal(err)
			}
			assertIDs(t, page, want)
			t.Logf("page %d: %v", pageIndex+1, want)
			for _, msg := range page {
				if seen[msg.ID] {
					t.Fatalf("duplicate ID %d", msg.ID)
				}
				seen[msg.ID] = true
			}
			if len(page) > 0 {
				id := page[len(page)-1].ID
				cursor = &id
			}
		}
		if len(seen) != 10 {
			t.Fatalf("received %d unique messages", len(seen))
		}
	})
	t.Run("EmptyChat", func(t *testing.T) {
		page, err := repo.ListBefore(ctx, emptyChatID, nil, 3)
		if err != nil {
			t.Fatal(err)
		}
		if page == nil || len(page) != 0 {
			t.Fatalf("expected empty non-nil slice: %+v", page)
		}
	})
	t.Run("ChatIsolation", func(t *testing.T) {
		page, err := repo.ListBefore(ctx, otherChatID, nil, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertIDs(t, page, []int64{other.ID})
		for _, msg := range page {
			if msg.ChatID != otherChatID {
				t.Fatal("message from another chat")
			}
		}
	})
	t.Run("SearchCaseInsensitiveDescendingAndLimit", func(t *testing.T) {
		for _, query := range []string{"message", "MESSAGE", "mEsSaGe"} {
			page, err := repo.Search(ctx, chatID, query, 3)
			if err != nil {
				t.Fatal(err)
			}
			assertIDs(t, page, []int64{ids[9], ids[8], ids[7]})
		}
	})
	t.Run("SearchChatIsolation", func(t *testing.T) {
		page, err := repo.Search(ctx, otherChatID, "message", 50)
		if err != nil {
			t.Fatal(err)
		}
		assertIDs(t, page, []int64{other.ID})
		page, err = repo.Search(ctx, chatID, "another chat", 50)
		if err != nil {
			t.Fatal(err)
		}
		assertIDs(t, page, nil)
		page, err = repo.Search(ctx, emptyChatID, "message", 50)
		if err != nil {
			t.Fatal(err)
		}
		if page == nil || len(page) != 0 {
			t.Fatalf("expected empty search: %+v", page)
		}
	})
}
