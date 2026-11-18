package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func testMessageMutationsPostgres(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	repo, chats := NewMessageRepository(pool), NewChatRepository(pool)
	users := make([]string, 2)
	for i, name := range []string{"edit_author", "edit_reader"} {
		if err := pool.QueryRow(ctx, "INSERT INTO users(username,password_hash) VALUES($1,'hash') RETURNING id", name).Scan(&users[i]); err != nil {
			t.Fatal(err)
		}
	}
	chat, err := chats.CreateGroup(ctx, users[0], "Mutations", users[1:])
	if err != nil {
		t.Fatal(err)
	}
	other, err := chats.GetOrCreateDirect(ctx, users[0], users[1])
	if err != nil {
		t.Fatal(err)
	}
	create := func(content string) *model.Message {
		t.Helper()
		msg, err := repo.Create(ctx, chat.ID, users[0], content, nil)
		if err != nil {
			t.Fatal(err)
		}
		return msg
	}
	assertUnread := func(want int64, cursor *int64) {
		t.Helper()
		list, err := chats.ListByUser(ctx, users[1])
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range list {
			if c.ID != chat.ID {
				continue
			}
			if c.UnreadCount != want {
				t.Fatalf("unread=%d want=%d", c.UnreadCount, want)
			}
			if cursor != nil && (c.LastReadMessageID == nil || *c.LastReadMessageID != *cursor) {
				t.Fatalf("read cursor=%v want=%d", c.LastReadMessageID, *cursor)
			}
			return
		}
		t.Fatal("chat missing")
	}
	first := create("original text")
	if first.Edited || first.EditedAt != nil || first.Deleted || first.DeletedAt != nil {
		t.Fatalf("new message=%+v", first)
	}
	if _, err := repo.GetByID(ctx, other.ID, first.ID); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("cross-chat lookup: %v", err)
	}
	for _, op := range []string{"edit", "delete"} {
		for _, ids := range [][2]string{{other.ID, users[0]}, {chat.ID, users[1]}} {
			if op == "edit" {
				_, err = repo.Edit(ctx, ids[0], ids[1], first.ID, "forbidden")
			} else {
				_, err = repo.Delete(ctx, ids[0], ids[1], first.ID)
			}
			if !errors.Is(err, ErrMessageNotFound) {
				t.Fatalf("%s wrong chat/author: %v", op, err)
			}
		}
	}
	edited, err := repo.Edit(ctx, chat.ID, users[0], first.ID, "исправлено 😀")
	if err != nil {
		t.Fatal(err)
	}
	if !edited.Edited || edited.EditedAt == nil || !edited.CreatedAt.Equal(first.CreatedAt) || edited.Deleted || edited.Content != "исправлено 😀" {
		t.Fatalf("edited=%+v", edited)
	}
	page, err := repo.ListBefore(ctx, chat.ID, nil, 10)
	if err != nil || len(page) != 1 || !page[0].Edited || page[0].Content != edited.Content {
		t.Fatalf("history=%+v err=%v", page, err)
	}
	found, err := repo.Search(ctx, chat.ID, "original", 50)
	if err != nil || len(found) != 0 {
		t.Fatalf("old text found=%+v err=%v", found, err)
	}
	found, err = repo.Search(ctx, chat.ID, "исправлено", 50)
	if err != nil || len(found) != 1 || !found[0].Edited {
		t.Fatalf("new text found=%+v err=%v", found, err)
	}
	assertUnread(1, nil)
	if _, _, err := chats.MarkRead(ctx, chat.ID, users[1], first.ID); err != nil {
		t.Fatal(err)
	}
	second := create("unread text")
	assertUnread(1, &first.ID)
	deleted, err := repo.Delete(ctx, chat.ID, users[0], first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted.Deleted || deleted.DeletedAt == nil || deleted.Content != "" || !deleted.Edited || deleted.EditedAt == nil {
		t.Fatalf("deleted=%+v", deleted)
	}
	assertUnread(1, &first.ID)
	if _, err := repo.Delete(ctx, chat.ID, users[0], second.ID); err != nil {
		t.Fatal(err)
	}
	assertUnread(0, &first.ID)
	found, err = repo.Search(ctx, chat.ID, "%", 50)
	if err != nil || len(found) != 0 {
		t.Fatalf("deleted search=%+v err=%v", found, err)
	}
	page, err = repo.ListBefore(ctx, chat.ID, &second.ID, 1)
	if err != nil || len(page) != 1 || page[0].ID != first.ID || !page[0].Deleted || page[0].Content != "" {
		t.Fatalf("deleted history=%+v err=%v", page, err)
	}
	if _, err := repo.Edit(ctx, chat.ID, users[0], first.ID, "resurrect"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("edit deleted: %v", err)
	}
	if _, err := repo.Delete(ctx, chat.ID, users[0], first.ID); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("repeat delete: %v", err)
	}
	// Deleted IDs still support read advancement after reconnect.
	if _, advanced, err := chats.MarkRead(ctx, chat.ID, users[1], second.ID); err != nil || !advanced {
		t.Fatalf("read tombstone: advanced=%v err=%v", advanced, err)
	}
	assertUnread(0, &second.ID)

	t.Run("ConcurrentEditDelete", func(t *testing.T) {
		msg := create("concurrent")
		var editErr, deleteErr error
		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() { defer wg.Done(); <-start; _, editErr = repo.Edit(ctx, chat.ID, users[0], msg.ID, "changed") }()
		go func() { defer wg.Done(); <-start; _, deleteErr = repo.Delete(ctx, chat.ID, users[0], msg.ID) }()
		close(start)
		wg.Wait()
		if deleteErr != nil || (editErr != nil && !errors.Is(editErr, ErrMessageNotFound)) {
			t.Fatalf("edit=%v delete=%v", editErr, deleteErr)
		}
		final, err := repo.GetByID(ctx, chat.ID, msg.ID)
		if err != nil || !final.Deleted || final.Content != "" {
			t.Fatalf("final=%+v err=%v", final, err)
		}
	})

	t.Run("MigrationRollbackAndUpgrade", func(t *testing.T) {
		data, err := os.ReadFile("../../migrations/20261007000100_add_message_edit_delete.sql")
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.SplitN(string(data), "-- +goose Down", 2)
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, parts[1]); err != nil {
			t.Fatalf("migration down: %v", err)
		}
		var oldID int64
		if err := tx.QueryRow(ctx, "INSERT INTO messages(chat_id, sender_id, content) VALUES($1, $2, 'legacy message') RETURNING id", chat.ID, users[0]).Scan(&oldID); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, parts[0]); err != nil {
			t.Fatalf("migration up with existing data: %v", err)
		}
		legacy, err := scanMessage(tx.QueryRow(ctx, `SELECT `+messageColumns+` FROM messages WHERE id = $1`, oldID))
		if err != nil || legacy.Content != "legacy message" || legacy.Edited || legacy.Deleted {
			t.Fatalf("legacy message=%+v err=%v", legacy, err)
		}
		var content string
		var readID int64
		if err := tx.QueryRow(ctx, "SELECT content FROM messages WHERE id = $1", first.ID).Scan(&content); err != nil || content != "[deleted]" {
			t.Fatalf("rollback placeholder=%q err=%v", content, err)
		}
		if err := tx.QueryRow(ctx, "SELECT last_read_message_id FROM chat_members WHERE chat_id = $1 AND user_id = $2", chat.ID, users[1]).Scan(&readID); err != nil || readID != second.ID {
			t.Fatalf("rollback cursor=%d err=%v", readID, err)
		}
	})
}
