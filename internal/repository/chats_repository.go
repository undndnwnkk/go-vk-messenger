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
		return nil, ErrTransactionAction
	}
	defer tx.Rollback(ctx)

	var chatID string
	if err = tx.QueryRow(ctx, insertChatQuery, "group", title, creatorID).Scan(&chatID); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chatID, creatorID, "admin"); err != nil {
		return nil, ErrInsertMember
	}

	for _, cur := range memberIDs {
		if _, err := tx.Exec(ctx, insertMemberQuery, chatID, cur, "member"); err != nil {
			return nil, ErrInsertMember
		}
	}

	var res model.Chat

	if err := tx.QueryRow(ctx, getChatByIDQuery, chatID).Scan(&res); err != nil {
		return nil, ErrChatNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, ErrTransactionAction
	}

	return &res, nil
}

func (r *ChatRepository) GetOrCreateDirect(
	ctx context.Context,
	userID1 string,
	userID2 string,
) (*model.Chat, error) {
	var chatID string
	userID1, userID2 = normalizeUserPair(userID1, userID2)
	err := r.db.QueryRow(ctx, getChatIDFromDirectChat, userID1, userID2).Scan(&chatID)
	if err == nil && chatID != "" {
		var chat model.Chat
		if err = r.db.QueryRow(ctx, getChatByIDQuery, chatID).Scan(&chat); err != nil {
			return nil, mapChatsPostgresError(err)
		}

		return &chat, nil
	}
	if err != nil && !errors.Is(mapChatsPostgresError(err), ErrChatNotFound) {
		return nil, mapChatsPostgresError(err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, ErrTransactionAction
	}
	defer tx.Rollback(ctx)
	var chat model.Chat
	if err := tx.QueryRow(ctx, insertChatQueryFullReturning, "direct", "", userID1).Scan(&chat); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	if _, err := tx.Exec(ctx, insertDirectChatQuery, chatID, userID1, userID2); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chatID, userID1, "admin"); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chatID, userID2, "admin"); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, ErrTransactionAction
	}

	return &chat, nil
}

func (r *ChatRepository) GetByID(
	ctx context.Context,
	chatID string,
) (*model.Chat, error) {
	var chat model.Chat

	if err := r.db.QueryRow(ctx, getChatByIDQuery, chatID).Scan(&chat); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	return &chat, nil
}

func (r *ChatRepository) GetByIDForUser(
	ctx context.Context,
	chatID string,
	userID string,
) (*model.Chat, error) {
	var chat model.Chat

	if err := r.db.QueryRow(ctx, getByIDForUserQuery, chatID, userID).Scan(&chat); err != nil {
		return nil, mapChatsPostgresError(err)
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
		return nil, mapChatsPostgresError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var chat model.Chat
		if err := rows.Scan(&chat.ID, &chat.Type, &chat.Title, &chat.CreatedBy, &chat.CreatedAt); err != nil {
			return nil, mapChatsPostgresError(err)
		}

		chats = append(chats, chat)
	}

	return chats, nil
}

func (r *ChatRepository) GetMember(
	ctx context.Context,
	chatID string,
	userID string,
) (*model.ChatMember, error) {
	var member model.ChatMember

	if err := r.db.QueryRow(ctx, getChatMemberQuery, chatID, userID).Scan(&member); err != nil {
		return nil, mapChatsPostgresError(err)
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
		return nil, mapChatsPostgresError(err)
	}

	for rows.Next() {
		var m model.ChatMember

		if err := rows.Scan(&m.ChatID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, mapChatsPostgresError(err)
		}

		members = append(members, m)
	}

	return members, nil
}

func (r *ChatRepository) AddMember(
	ctx context.Context,
	chatID string,
	userID string,
	role model.ChatRole,
) error {
	if _, err := r.db.Exec(ctx, insertMemberQuery, chatID, userID, role); err != nil {
		return mapChatsPostgresError(err)
	}

	return nil
}

func (r *ChatRepository) RemoveMember(
	ctx context.Context,
	chatID string,
	userID string,
) error {
	if _, err := r.db.Exec(ctx, deleteMemberQuery, chatID, userID); err != nil {
		return mapChatsPostgresError(err)
	}

	return nil
}

func mapChatsPostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return ErrChatAlreadyExists
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrChatNotFound
	}

	return fmt.Errorf("error chat repository: %w", err)
}

func normalizeUserPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

var (
	// errors
	ErrTransactionAction = errors.New("error creating/commiting transaction")
	ErrInsertChat        = errors.New("error while inserting chat")
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat not found")
	ErrInsertMember      = errors.New("error insert chat member")
	// queries
	insertMemberQuery            = "INSERT INTO chat_members(chat_id, user_id, role) VALUES ($1, $2, $3)"
	insertChatQuery              = "INSERT INTO chats (type, title, created_by) VALUES ($1, $2, $3) RETURNING id"
	insertChatQueryFullReturning = "INSERT INTO chats (type, title, created_by) VALUES ($1, $2, $3) RETURNING id, type, title, created_by, created_at"
	getChatByIDQuery             = "SELECT id, type, title, created_by, created_at FROM chats WHERE id = $1"
	insertDirectChatQuery        = "INSERT INTO direct_chats (chat_id, user1_id, user2_id) VALUES ($1, $2, $3)"
	getDirectChatQuery           = "SELECT chat_id, user1_id, user2_id FROM direct_chats WHERE chat_id = $1"
	getChatIDFromDirectChat      = "SELECT id FROM direct_chats WHERE user1_id = $1 AND user2_id = $2"
	getByIDForUserQuery          = `
		SELECT id, type, title, created_by, created_at 
		FROM chats WHERE id = $1
		JOIN chat_members ON chat_members.chat_id = id AND chat_members.user_id = $2 
	`
	getChatsByUserIDQuery = `
		SELECT id, type, title, created_by, created_at 
		FROM chats
		JOIN chat_members ON chat_members.chat_id = id AND chat_members.user_id = $1
	`
	getChatMemberQuery = `
		SELECT chat_id, user_id, role, joined_at FROM chat_members WHERE chat_id = $1 AND user_id = $2
	`
	getChatMembersQuery = "SELECT chat_id, user_id, role, joined_at FROM chat_members WHERE chat_id = $1"
	deleteMemberQuery   = "DELETE FROM chat_members WHERE chat_id = $1 AND user_id = $2"
)
