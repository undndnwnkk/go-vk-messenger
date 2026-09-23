DB_DRIVER = postgres
DB_USER = postgres
DB_PASSWORD = postgres
DB_HOST = 127.0.0.1
DB_PORT = 5432
DB_NAME = messenger_db
DB_SSLMODE = disable
MIGRATIONS_DIR = ./migrations

DB_URL = user=$(DB_USER) password=$(DB_PASSWORD) host=$(DB_HOST) port=$(DB_PORT) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)
GOOSE = go run github.com/pressly/goose/v3/cmd/goose@latest

run:
	cmd /C "go run ./cmd/app"

test:
	cmd /C "go test ./..."

test-race:
	cmd /C "go test -race ./..."

vet:
	cmd /C "go vet ./..."

migrate-up:
	cmd /C "$(GOOSE) -dir \"$(MIGRATIONS_DIR)\" \"$(DB_DRIVER)\" \"$(DB_URL)\" up"

migrate-down:
	cmd /C "$(GOOSE) -dir \"$(MIGRATIONS_DIR)\" \"$(DB_DRIVER)\" \"$(DB_URL)\" down"

migrate-status:
	cmd /C "$(GOOSE) -dir \"$(MIGRATIONS_DIR)\" \"$(DB_DRIVER)\" \"$(DB_URL)\" status"

migrate-create:
	cmd /C "$(GOOSE) -dir \"$(MIGRATIONS_DIR)\" create \"$(name)\" sql"

docker-build:
	cmd /C "docker compose build"

docker-up:
	cmd /C "docker compose up -d --build"

docker-down:
	cmd /C "docker compose down"

docker-logs:
	cmd /C "docker compose logs -f app"

docker-ps:
	cmd /C "docker compose ps"
