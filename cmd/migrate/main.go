package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dsn := buildDSNFromEnv()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	dir := getEnv("MIGRATIONS_DIR", "./migrations")
	if err := migrateUp(ctx, pool, dir); err != nil {
		log.Fatalf("migrate up: %v", err)
	}
	log.Println("migrations applied")
}

func migrateUp(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, createVersionTableSQL); err != nil {
		return fmt.Errorf("create version table: %w", err)
	}
	if _, err := pool.Exec(ctx, insertZeroVersionSQL); err != nil {
		return fmt.Errorf("insert zero version: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", dir)
	}
	sort.Strings(files)

	for _, file := range files {
		version, err := migrationVersion(file)
		if err != nil {
			return err
		}
		applied, err := isApplied(ctx, pool, version)
		if err != nil {
			return err
		}
		if applied {
			log.Printf("migration %d already applied", version)
			continue
		}

		upSQL, err := readUpSQL(file)
		if err != nil {
			return err
		}
		if strings.TrimSpace(upSQL) == "" {
			return fmt.Errorf("migration %s has empty up section", file)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}
		if _, err := tx.Exec(ctx, upSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
		if _, err := tx.Exec(ctx, insertVersionSQL, version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %d: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
		log.Printf("migration %d applied", version)
	}

	return nil
}

func migrationVersion(file string) (int64, error) {
	base := filepath.Base(file)
	prefix := strings.SplitN(base, "_", 2)[0]
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse migration version from %s: %w", file, err)
	}
	return version, nil
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, version int64) (bool, error) {
	var applied bool
	err := pool.QueryRow(ctx, latestVersionSQL, version).Scan(&applied)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read migration version %d: %w", version, err)
	}
	return applied, nil
}

func readUpSQL(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read migration %s: %w", file, err)
	}
	text := string(data)
	parts := strings.SplitN(text, "-- +goose Down", 2)
	up := strings.Replace(parts[0], "-- +goose Up", "", 1)
	return strings.TrimSpace(up), nil
}

func buildDSNFromEnv() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_HOST", "127.0.0.1"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "messenger_db"),
		getEnv("DB_SSLMODE", "disable"),
	)
}

func getEnv(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

const createVersionTableSQL = `
CREATE TABLE IF NOT EXISTS goose_db_version (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL,
    is_applied BOOLEAN NOT NULL,
    tstamp TIMESTAMPTZ NOT NULL DEFAULT now()
);`

const insertZeroVersionSQL = `
INSERT INTO goose_db_version(version_id, is_applied)
SELECT 0, true
WHERE NOT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id = 0);`

const latestVersionSQL = `
SELECT is_applied
FROM goose_db_version
WHERE version_id = $1
ORDER BY id DESC
LIMIT 1;`

const insertVersionSQL = `INSERT INTO goose_db_version(version_id, is_applied) VALUES ($1, true);`
