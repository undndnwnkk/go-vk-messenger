package repository

import (
	"context"
	"errors"
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
		var pgErr *pgconn.PgError
		return "", pgErrorHandler(pgErr, err)
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
		var pgErr *pgconn.PgError
		return nil, pgErrorHandler(pgErr, err)
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
		var pgErr *pgconn.PgError
		return nil, pgErrorHandler(pgErr, err)
	}

	return &user, nil
}

func pgErrorHandler(pgErr *pgconn.PgError, err error) error {
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return ErrUsernameTaken
		}
		if pgErr.Code == "23502" {
			return ErrUsernameNull
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}

	return err
}

var (
	ErrUsernameTaken = errors.New("username must be unique")
	ErrUsernameNull  = errors.New("username must be not null")
	ErrUserNotFound  = errors.New("user not found")
)
