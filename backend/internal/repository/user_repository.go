package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user model.User) (string, error) {
	sql := "INSERT INTO users (username, password_hash) VALUES($1, $2) RETURNING id"

	var id string

	err := r.db.QueryRow(ctx, sql, user.Username, user.PasswordHash).Scan(&id)
	if err != nil {
		if mappedErr := mapPostgresError(err); mappedErr != nil {
			return "", mappedErr
		}
		return "", fmt.Errorf("create user: %w", err)
	}

	return id, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	sql := `
		SELECT id, username, password_hash, created_at FROM users WHERE username = $1
	`
	var user model.User

	if err := r.db.QueryRow(
		ctx,
		sql,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt); err != nil {
		if mappedErr := mapPostgresError(err); mappedErr != nil {
			return nil, mappedErr
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	sql := `
		SELECT id, username, password_hash, created_at FROM users WHERE id = $1
	`
	var user model.User

	if err := r.db.QueryRow(
		ctx,
		sql,
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt); err != nil {
		if mappedErr := mapPostgresError(err); mappedErr != nil {
			return nil, mappedErr
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Search(ctx context.Context, excludeID, prefix string, limit int) ([]model.UserSummary, error) {
	return r.listSummaries(ctx, searchUsersQuery, excludeID, likePrefix(prefix), limit)
}

func (r *UserRepository) ListContacts(ctx context.Context, userID string, limit int) ([]model.UserSummary, error) {
	return r.listSummaries(ctx, listContactsQuery, userID, limit)
}

func (r *UserRepository) listSummaries(ctx context.Context, query string, args ...any) ([]model.UserSummary, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]model.UserSummary, 0)
	for rows.Next() {
		var user model.UserSummary
		if err := rows.Scan(&user.ID, &user.Username); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func likePrefix(value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(value)
	return escaped + "%"
}

func mapPostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return ErrUsernameTaken
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}

	return nil
}

const (
	searchUsersQuery = `
 SELECT id, username FROM users
 WHERE id <> $1 AND username LIKE $2 ESCAPE '\'
 ORDER BY username
 LIMIT $3`
	listContactsQuery = `
 SELECT u.id, u.username FROM users u
 WHERE u.id <> $1
   AND EXISTS (
       SELECT 1 FROM chat_members me
       JOIN chat_members other ON other.chat_id = me.chat_id
       WHERE me.user_id = $1 AND other.user_id = u.id
   )
 ORDER BY u.username
 LIMIT $2`
)

var (
	ErrUsernameTaken = errors.New("username must be unique")
	ErrUserNotFound  = errors.New("user not found")
)
