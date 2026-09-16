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

var (
	ErrUsernameTaken = errors.New("username must be unique")
	ErrUserNotFound  = errors.New("user not found")
)
