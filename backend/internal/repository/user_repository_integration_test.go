package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func TestUserRepositoryPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL or DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		t.Fatalf("prepare users table: %v", err)
	}

	username := "repo_test_" + time.Now().UTC().Format("20060102150405.000000000")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM users WHERE username = $1", username)
	})

	repo := NewUserRepository(pool)
	id, err := repo.Create(ctx, model.User{
		Username:     username,
		PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if id == "" {
		t.Fatal("Create returned empty id")
	}

	user, err := repo.GetByUsername(ctx, username)
	if err != nil {
		t.Fatalf("GetByUsername returned error: %v", err)
	}
	if user.ID != id || user.Username != username || user.PasswordHash != "hashed-password" {
		t.Fatalf("GetByUsername returned unexpected user: %#v", user)
	}

	if _, err := repo.Create(ctx, model.User{Username: username, PasswordHash: "another-hash"}); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate Create error = %v, want %v", err, ErrUsernameTaken)
	}
}
