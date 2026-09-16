ifneq (,$(wildcard .env))
    include .env
    export
endif

DB_DRIVER ?= postgres
DB_URL ?= "user=$(DB_USER) password=$(DB_PASSWORD) host=$(DB_HOST) port=$(DB_PORT) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)"
MIGRATIONS_DIR ?= ./migrations

.PHONY: migrate-status migrate-up migrate-down migrate-create

migrate-status:
	@goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_URL) status

migrate-up:
	@goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_URL) up

migrate-down:
	@goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_URL) down

migrate-create:
ifndef name
	$(error name is required. Usage: make migrate-create name=your_migration_name)
endif
	@goose -dir $(MIGRATIONS_DIR) create $(name) sql
