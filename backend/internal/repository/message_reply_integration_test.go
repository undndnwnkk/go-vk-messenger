package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func testMessageRepliesPostgres(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	repo, chats := NewMessageRepository(pool), NewChatRepository(pool)
	var userID, otherUserID string
	if err := pool.QueryRow(ctx, "INSERT INTO users(username,password_hash) VALUES('reply_author','hash') RETURNING id").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO users(username,password_hash) VALUES('reply_reader','hash') RETURNING id").Scan(&otherUserID); err != nil {
		t.Fatal(err)
	}
	chat, err := chats.CreateGroup(ctx, userID, "Replies", []string{otherUserID})
	if err != nil {
		t.Fatal(err)
	}
	other, err := chats.GetOrCreateDirect(ctx, userID, otherUserID)
	if err != nil {
		t.Fatal(err)
	}
	create := func(chatID, sender, text string, replyID *int64) *model.Message {
		t.Helper()
		msg, err := repo.Create(ctx, chatID, sender, text, replyID)
		if err != nil {
			t.Fatal(err)
		}
		return msg
	}
	parent := create(chat.ID, userID, "original", nil)
	if parent.ReplyToMessageID != nil || parent.ReplyTo != nil {
		t.Fatalf("plain message=%+v", parent)
	}
	reply := create(chat.ID, otherUserID, "answer", &parent.ID)
	assertReply := func(msg *model.Message, id int64, content string, edited, deleted bool) {
		t.Helper()
		if msg.ReplyToMessageID == nil || *msg.ReplyToMessageID != id || msg.ReplyTo == nil || msg.ReplyTo.ID != id || msg.ReplyTo.Content != content || msg.ReplyTo.Edited != edited || msg.ReplyTo.Deleted != deleted {
			t.Fatalf("message=%+v preview=%+v", msg, msg.ReplyTo)
		}
	}
	assertReply(reply, parent.ID, "original", false, false)
	if reply.ReplyTo.SenderID != userID {
		t.Fatalf("parent author=%s", reply.ReplyTo.SenderID)
	}
	// The parent is outside this page, but the reply preview is still available.
	page, err := repo.ListBefore(ctx, chat.ID, nil, 1)
	if err != nil || len(page) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	assertReply(&page[0], parent.ID, "original", false, false)
	chain := create(chat.ID, userID, "reply to reply", &reply.ID)
	assertReply(chain, reply.ID, "answer", false, false)
	create(chat.ID, userID, "self reply", &parent.ID)
	if _, err := repo.Edit(ctx, chat.ID, userID, parent.ID, "corrected"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repo.GetByID(ctx, chat.ID, reply.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertReply(reloaded, parent.ID, "corrected", true, false)
	if reloaded.ReplyTo.EditedAt == nil {
		t.Fatal("missing parent edit timestamp")
	}
	search, err := repo.Search(ctx, chat.ID, "answer", 50)
	if err != nil || len(search) != 1 {
		t.Fatalf("search=%+v err=%v", search, err)
	}
	assertReply(&search[0], parent.ID, "corrected", true, false)
	updated, err := repo.Edit(ctx, chat.ID, otherUserID, reply.ID, "updated answer")
	if err != nil {
		t.Fatal(err)
	}
	assertReply(updated, parent.ID, "corrected", true, false)
	if _, err := repo.Delete(ctx, chat.ID, userID, parent.ID); err != nil {
		t.Fatal(err)
	}
	reloaded, err = repo.GetByID(ctx, chat.ID, reply.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertReply(reloaded, parent.ID, "", true, true)
	if reloaded.Deleted || reloaded.Content != "updated answer" || reloaded.ReplyTo.DeletedAt == nil {
		t.Fatalf("reply after parent deletion=%+v", reloaded)
	}
	missing := int64(9223372036854775807)
	for _, tc := range []struct {
		name, chat string
		target     int64
	}{
		{"Missing", chat.ID, missing}, {"Deleted", chat.ID, parent.ID}, {"OtherChat", other.ID, reply.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := repo.Create(ctx, tc.chat, userID, "invalid reply", &tc.target); !errors.Is(err, ErrMessageNotFound) {
				t.Fatalf("create: %v", err)
			}
		})
	}
	// Even SQL callers cannot bypass the chat boundary or create dangling references.
	for _, target := range []int64{reply.ID, missing} {
		_, err := pool.Exec(ctx, "INSERT INTO messages(chat_id,sender_id,content,reply_to_message_id) VALUES($1,$2,'invalid',$3)", other.ID, userID, target)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
			t.Fatalf("expected foreign key violation, got %v", err)
		}
	}
	deletedReply, err := repo.Delete(ctx, chat.ID, otherUserID, reply.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertReply(deletedReply, parent.ID, "", true, true)

	t.Run("MigrationRollbackAndUpgrade", func(t *testing.T) {
		data, err := os.ReadFile("../../migrations/20261007000200_add_message_replies.sql")
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
			t.Fatalf("down: %v", err)
		}
		if _, err := tx.Exec(ctx, parts[0]); err != nil {
			t.Fatalf("up: %v", err)
		}
		msg, err := scanMessage(tx.QueryRow(ctx, "SELECT "+messageColumns+" FROM messages WHERE id = $1", chain.ID))
		if err != nil || msg.Content != chain.Content || msg.ReplyTo != nil || msg.ReplyToMessageID != nil {
			t.Fatalf("upgraded message=%+v err=%v", msg, err)
		}
	})
	// A self-referencing FK must not prevent the chat's existing cascade deletion.
	if _, err := pool.Exec(ctx, "DELETE FROM chats WHERE id = $1", chat.ID); err != nil {
		t.Fatalf("delete chat with replies: %v", err)
	}
}
