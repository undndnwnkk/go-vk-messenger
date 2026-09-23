package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

type ChatRepository struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateGroup(
	ctx context.Context,
	creatorID string,
	title string,
	memberIDs []string,
) (*model.Chat, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("chat transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var chatID string
	if err = tx.QueryRow(ctx, insertChatQuery, "group", title, creatorID).Scan(&chatID); err != nil {
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chatID, creatorID, "admin"); err != nil {
		return nil, fmt.Errorf("insert group member: %w", err)
	}

	for _, cur := range memberIDs {
		if _, err := tx.Exec(ctx, insertMemberQuery, chatID, cur, "member"); err != nil {
			return nil, fmt.Errorf("insert group member: %w", err)
		}
	}

	var res model.Chat

	if err := tx.QueryRow(ctx, getChatByIDQuery, chatID).Scan(&res.ID, &res.Type, &res.Title, &res.CreatedBy, &res.CreatedAt); err != nil {
		return nil, fmt.Errorf("read created group: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("chat transaction: %w", err)
	}

	return &res, nil
}

func (r *ChatRepository) GetOrCreateDirect(
	ctx context.Context,
	userID1 string,
	userID2 string,
) (*model.Chat, error) {
	originalCreatorID := userID1
	var chatID string
	userID1, userID2 = normalizeUserPair(userID1, userID2)
	err := r.db.QueryRow(ctx, getChatIDFromDirectChat, userID1, userID2).Scan(&chatID)
	if err == nil {
		return r.GetByID(ctx, chatID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("find direct chat: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("chat transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var chat model.Chat
	if err := tx.QueryRow(ctx, insertChatQueryFullReturning, model.Direct, nil, originalCreatorID).Scan(&chat.ID, &chat.Type, &chat.Title, &chat.CreatedBy, &chat.CreatedAt); err != nil {
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	if _, err := tx.Exec(ctx, insertDirectChatQuery, chat.ID, userID1, userID2); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "direct_chats_user1_id_user2_id_key" {
			// Release the failed transaction (including its speculative chat) before reading the winner.
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return nil, fmt.Errorf("rollback direct chat: %w", rollbackErr)
			}
			if err := r.db.QueryRow(ctx, getChatIDFromDirectChat, userID1, userID2).Scan(&chatID); err != nil {
				return nil, fmt.Errorf("find winning direct chat: %w", err)
			}
			return r.GetByID(ctx, chatID)
		}
		return nil, fmt.Errorf("insert direct chat: %w", err)
	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chat.ID, userID1, model.Member); err != nil {
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chat.ID, userID2, model.Member); err != nil {
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("chat transaction: %w", err)
	}

	return &chat, nil
}

func (r *ChatRepository) GetByID(
	ctx context.Context,
	chatID string,
) (*model.Chat, error) {
	var chat model.Chat

	if err := r.db.QueryRow(ctx, getChatByIDQuery, chatID).Scan(&chat.ID, &chat.Type, &chat.Title, &chat.CreatedBy, &chat.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatNotFound
		}
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	return &chat, nil
}

func (r *ChatRepository) GetByIDForUser(
	ctx context.Context,
	chatID string,
	userID string,
) (*model.Chat, error) {
	var chat model.Chat

	if err := r.db.QueryRow(ctx, getByIDForUserQuery, chatID, userID).Scan(&chat.ID, &chat.Type, &chat.Title, &chat.CreatedBy, &chat.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatNotFound
		}
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	return &chat, nil
}

func (r *ChatRepository) ListByUser(
	ctx context.Context,
	userID string,
) ([]model.Chat, error) {
	chats := make([]model.Chat, 0)

	rows, err := r.db.Query(ctx, getChatsByUserIDQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("chat repository: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var chat model.Chat
		if err := rows.Scan(&chat.ID, &chat.Type, &chat.Title, &chat.CreatedBy, &chat.CreatedAt, &chat.LastReadMessageID, &chat.UnreadCount); err != nil {
			return nil, fmt.Errorf("chat repository: %w", err)
		}

		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	return chats, nil
}

func (r *ChatRepository) GetMember(
	ctx context.Context,
	chatID string,
	userID string,
) (*model.ChatMember, error) {
	var member model.ChatMember

	if err := r.db.QueryRow(ctx, getChatMemberQuery, chatID, userID).Scan(&member.ChatID, &member.UserID, &member.Role, &member.JoinedAt, &member.LastReadMessageID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	return &member, nil
}

func (r *ChatRepository) ListMembers(
	ctx context.Context,
	chatID string,
) ([]model.ChatMember, error) {
	members := make([]model.ChatMember, 0)

	rows, err := r.db.Query(ctx, getChatMembersQuery, chatID)
	if err != nil {
		return nil, fmt.Errorf("chat repository: %w", err)
	}

	defer rows.Close()
	for rows.Next() {
		var m model.ChatMember

		if err := rows.Scan(&m.ChatID, &m.UserID, &m.Role, &m.JoinedAt, &m.LastReadMessageID); err != nil {
			return nil, fmt.Errorf("chat repository: %w", err)
		}

		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	return members, nil
}

func (r *ChatRepository) AddMember(
	ctx context.Context,
	chatID string,
	userID string,
	role model.ChatRole,
) error {
	if _, err := r.db.Exec(ctx, insertMemberAtCurrentReadQuery, chatID, userID, role); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "chat_members_pkey" {
			return ErrMemberAlreadyExists
		}
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *ChatRepository) MarkRead(ctx context.Context, chatID, userID string, messageID int64) (*model.ChatReadState, bool, error) {
	var state model.ChatReadState
	err := r.db.QueryRow(ctx, advanceReadQuery, chatID, userID, messageID).Scan(&state.ChatID, &state.UserID, &state.LastReadMessageID)
	if err == nil {
		return &state, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("mark read: %w", err)
	}

	var currentLastRead *int64
	if err := r.db.QueryRow(ctx, getReadStateQuery, chatID, userID).Scan(&state.ChatID, &state.UserID, &currentLastRead); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrMemberNotFound
		}
		return nil, false, fmt.Errorf("read state: %w", err)
	}

	var messageChatID string
	if err := r.db.QueryRow(ctx, getMessageChatIDQuery, messageID).Scan(&messageChatID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrMessageNotFound
		}
		return nil, false, fmt.Errorf("read message chat: %w", err)
	}
	if messageChatID != chatID {
		return nil, false, ErrMessageNotFound
	}

	if currentLastRead == nil {
		return nil, false, ErrMessageNotFound
	}
	state.LastReadMessageID = *currentLastRead
	return &state, false, nil
}

func (r *ChatRepository) RemoveMember(
	ctx context.Context,
	chatID string,
	userID string,
) error {
	tag, err := r.db.Exec(ctx, deleteMemberQuery, chatID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func normalizeUserPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

var (
	// errors
	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberAlreadyExists = errors.New("member already exists")
	ErrChatNotFound        = errors.New("chat not found")
	ErrMessageNotFound     = errors.New("message not found")
	// queries
	insertMemberQuery              = "INSERT INTO chat_members(chat_id, user_id, role) VALUES ($1, $2, $3)"
	insertMemberAtCurrentReadQuery = `
 INSERT INTO chat_members(chat_id, user_id, role, last_read_message_id)
 VALUES ($1, $2, $3, (SELECT max(id) FROM messages WHERE chat_id = $1))`
	insertChatQuery              = "INSERT INTO chats (type, title, created_by) VALUES ($1, $2, $3) RETURNING id"
	insertChatQueryFullReturning = "INSERT INTO chats (type, title, created_by) VALUES ($1, $2, $3) RETURNING id, type, title, created_by, created_at"
	getChatByIDQuery             = "SELECT id, type, title, created_by, created_at FROM chats WHERE id = $1"
	insertDirectChatQuery        = "INSERT INTO direct_chats (chat_id, user1_id, user2_id) VALUES ($1, $2, $3)"
	getChatIDFromDirectChat      = "SELECT chat_id FROM direct_chats WHERE user1_id = $1 AND user2_id = $2"
	getByIDForUserQuery          = `
 SELECT c.id, c.type, c.title, c.created_by, c.created_at
 FROM chats c JOIN chat_members cm ON cm.chat_id = c.id
 WHERE c.id = $1 AND cm.user_id = $2`
	getChatsByUserIDQuery = `
 SELECT c.id, c.type, c.title, c.created_by, c.created_at, cm.last_read_message_id,
        count(m.id) AS unread_count
 FROM chats c JOIN chat_members cm ON cm.chat_id = c.id
 LEFT JOIN messages m ON m.chat_id = c.id
      AND m.sender_id <> $1
      AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
 WHERE cm.user_id = $1
 GROUP BY c.id, c.type, c.title, c.created_by, c.created_at, cm.last_read_message_id
 ORDER BY c.created_at DESC`
	getChatMemberQuery = `
		SELECT chat_id, user_id, role, joined_at, last_read_message_id FROM chat_members WHERE chat_id = $1 AND user_id = $2
	`
	getChatMembersQuery = "SELECT chat_id, user_id, role, joined_at, last_read_message_id FROM chat_members WHERE chat_id = $1"
	deleteMemberQuery   = "DELETE FROM chat_members WHERE chat_id = $1 AND user_id = $2"
	advanceReadQuery    = `
 UPDATE chat_members
 SET last_read_message_id = $3
 WHERE chat_id = $1
   AND user_id = $2
   AND EXISTS (SELECT 1 FROM messages WHERE id = $3 AND chat_id = $1)
   AND (last_read_message_id IS NULL OR last_read_message_id < $3)
 RETURNING chat_id, user_id, last_read_message_id`
	getReadStateQuery = `
 SELECT chat_id, user_id, last_read_message_id
 FROM chat_members
 WHERE chat_id = $1 AND user_id = $2`
	getMessageChatIDQuery = "SELECT chat_id FROM messages WHERE id = $1"
)
