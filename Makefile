DB_DRIVER = postgres
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_HOST ?= 127.0.0.1
DB_PORT ?= 5432
DB_NAME ?= messenger_db
DB_SSLMODE ?= disable
MIGRATIONS_DIR ?= ./backend/migrations

DB_URL = user=$(DB_USER) password=$(DB_PASSWORD) host=$(DB_HOST) port=$(DB_PORT) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)
GOOSE = go run github.com/pressly/goose/v3/cmd/goose@latest
BACKEND = cd backend &&
COMPOSE = docker compose $(if $(wildcard backend/.env),--env-file backend/.env)

.PHONY: run test test-race cover vet migrate-up migrate-down migrate-status migrate-create docker-build docker-up docker-down docker-logs docker-ps docker-migrate docker-reset

run:
	$(BACKEND) go run ./cmd/app

test:
	$(BACKEND) go test ./...

test-race:
	$(BACKEND) go test -race ./...

cover:
	$(BACKEND) go test -coverprofile=cover.out ./...
	$(BACKEND) go tool cover -func=cover.out | tail -1

vet:
	$(BACKEND) go vet ./...

migrate-up:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$(DB_URL)" up

migrate-down:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$(DB_URL)" down

migrate-status:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$(DB_URL)" status

migrate-create:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" create "$(name)" sql

docker-build:
	$(COMPOSE) build

docker-up:
	$(COMPOSE) up -d --build

docker-migrate:
	$(COMPOSE) run --rm migrate

docker-down:
	$(COMPOSE) down

docker-reset:
	$(COMPOSE) down -v

docker-logs:
	$(COMPOSE) logs -f app

docker-ps:
	$(COMPOSE) ps
