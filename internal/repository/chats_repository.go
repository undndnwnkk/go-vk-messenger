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
) (*model.DirectChat, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, ErrTransactionAction
	}
	defer tx.Rollback(ctx)

	// getOrCreateDirectQuery
	var chatID string
	chatFound := false
	if err := tx.QueryRow(ctx, insertChatQuery, "direct", "", userID1).Scan(&chatID); err != nil {
		if !errors.Is(mapChatsPostgresError(err), ErrChatAlreadyExists) {
			return nil, mapChatsPostgresError(err)
		}
		
		chatFound = true
	}

	if chatFound {

	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chatID, userID1, "admin"); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	if _, err := tx.Exec(ctx, insertMemberQuery, chatID, userID2, "admin"); err != nil {
		return nil, mapChatsPostgresError(err)
	}

	var chat model.DirectChat
	if err := tx.QueryRow(ctx, getOrCreateDirectQuery, userID1, userID2).Scan(&chat); err != nil {
		if errors.Is(mapChatsPostgresError(err), ErrChatA)
	}

	return &chat, nil
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

var (
	// errors
	ErrTransactionAction = errors.New("error creating/commiting transaction")
	ErrInsertChat        = errors.New("error while inserting chat")
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat not found")
	ErrInsertMember      = errors.New("error insert chat member")
	// queries
	insertMemberQuery      = "INSERT INTO chat_members(chat_id, user_id, role) VALUES ($1, $2, $3)"
	insertChatQuery        = "INSERT INTO chats (type, title, created_by) VALUES ($1, $2, $3) RETURNING id"
	getChatByIDQuery       = "SELECT id, type, title, created_by, created_at FROM chats WHERE id = $1"
	insertDirectChatQuery = "INSERT INTO direct_chats (user1_id, user2_id) VALUES ($1, $2) RETURNING chat_id, user1_id, user2_id"
	getDirectChatQuery = "SELECT chat_id, user1_id, user2_id FROM direct_chats WHERE chat_id = $1"
	getDirectChatIDFromChatQuery = `
		SELECT chat_id, user1_id, user2_id
	`
)
