DB_DRIVER = postgres
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_HOST ?= 127.0.0.1
DB_PORT ?= 5432
DB_NAME ?= messenger_db
DB_SSLMODE ?= disable
MIGRATIONS_DIR ?= ./migrations

DB_URL = user=$(DB_USER) password=$(DB_PASSWORD) host=$(DB_HOST) port=$(DB_PORT) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)
GOOSE = go run github.com/pressly/goose/v3/cmd/goose@latest

.PHONY: run test test-race vet migrate-up migrate-down migrate-status migrate-create docker-build docker-up docker-down docker-logs docker-ps docker-migrate docker-reset

run:
	go run ./cmd/app

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

migrate-up:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$(DB_URL)" up

migrate-down:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$(DB_URL)" down

migrate-status:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$(DB_URL)" status

migrate-create:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" create "$(name)" sql

docker-build:
	docker compose build

docker-up:
	docker compose up -d --build

docker-migrate:
	docker compose run --rm migrate

docker-down:
	docker compose down

docker-reset:
	docker compose down -v

docker-logs:
	docker compose logs -f app

docker-ps:
	docker compose ps
