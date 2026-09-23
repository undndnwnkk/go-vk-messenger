# Go VK Messenger

Backend for a small messenger written in Go. It supports registration and login, JWT authentication, direct and group chats, PostgreSQL persistence, REST message history, search, WebSocket realtime delivery, read status, unread counters, rate limiting, Docker startup, and SQL migrations.

Russian documentation is available in [README.ru.md](README.ru.md).

## Features

Authentication:

- registration;
- login;
- JWT access tokens;
- `GET /api/v1/me`.

Chats:

- direct chats;
- group chats;
- chat membership;
- group admins;
- hidden access policy for private chat resources.

Messages:

- send through REST;
- send through WebSocket;
- PostgreSQL persistence;
- cursor history pagination;
- chat-scoped search.

Realtime:

- WebSocket endpoint;
- multiple connections per user;
- realtime `message_created`;
- realtime `read_updated`;
- reconnect through REST history.

Read status:

- `chat_members.last_read_message_id`;
- unread counts in `GET /api/v1/chats`;
- high-water mark semantics: read position only moves forward.

Protection and lifecycle:

- per-user message rate limit;
- slow WebSocket clients are removed;
- graceful shutdown closes active WebSocket clients and the PostgreSQL pool.

## Architecture

```text
Client / Postman / WebSocket client
      │
      ├── HTTP REST
      │
      └── WebSocket
            │
            ▼
      Go HTTP Server
            │
      Controllers / Handlers
            │
         Services
            │
       Repositories
            │
            ▼
       PostgreSQL

Realtime:
Service / Notifier
        │
        ▼
       Hub
        │
        ▼
WebSocket Clients
```

The project is a modular monolith. It is not split into microservices because this is an MVP with one deployment unit, lower operational complexity, and clear service/repository boundaries inside the codebase.

PostgreSQL is the source of truth. Realtime is delivery only:

```text
INSERT message
↓
message_ack to sender
↓
message_created to online chat members
```

If broadcast fails after a successful insert, the message is still created. Clients recover state through REST history.

WebSocket does not store an offline queue. If Bob is offline while Alice sends a message, Bob receives no realtime event. After reconnect, Bob calls `GET /api/v1/chats/{chatID}/messages` and gets missed messages from PostgreSQL.

Read status uses:

```text
chat_members.last_read_message_id
```

instead of a per-message table such as:

```text
message_reads(user_id, message_id)
```

Redis/NATS is not used because the app is currently single-instance. If several Go instances are introduced, the in-memory Hub will need an external Pub/Sub layer.

## Project structure

```text
cmd/app/                 application entrypoint
internal/config/         environment config loading and validation
internal/controller/     HTTP and WebSocket handlers
internal/model/          DTOs and domain models
internal/realtime/       Client, Hub, WebSocket events
internal/repository/     PostgreSQL repositories
internal/service/        business logic
migrations/              goose SQL migrations
.github/workflows/       CI workflow
```

## Requirements

- Go version from [go.mod](go.mod).
- Docker and Docker Compose.
- PostgreSQL for non-Docker local development.
- GNU Make for convenience targets, or direct Go/Docker commands.

The local Makefile runs goose through:

```bash
go run github.com/pressly/goose/v3/cmd/goose@latest
```

so a globally installed goose binary is not required.

## Configuration

Create a local env file:

```bash
cp .env.example .env
```

Development defaults:

```env
DB_DRIVER=postgres
DB_USER=postgres
DB_PASSWORD=postgres
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=messenger_db
DB_SSLMODE=disable
MIGRATIONS_DIR=./migrations

HTTP_PORT=8080

JWT_SECRET=dev-secret-change-me
JWT_TTL=15m
```

For Docker Compose, the app container overrides `DB_HOST` to `db`, because containers must connect to PostgreSQL by compose service name, not by `localhost`. PostgreSQL 18 data is persisted by mounting the named volume at `/var/lib/postgresql`, which matches the official image data root layout for PostgreSQL 18+.

Do not commit real production passwords, JWT secrets, access tokens, deployment credentials, local dumps, logs, or `.env` files. `.env` is ignored by Git.

Startup validation:

- missing `JWT_SECRET` fails startup with a clear error;
- invalid `JWT_TTL` fails startup;
- unavailable PostgreSQL fails startup during `Ping`, before serving requests.

## Run with Docker

```bash
git clone <repository-url>
cd go-vk-messenger
cp .env.example .env
docker compose up -d --build
curl http://localhost:8080/health
```

Expected health response:

```json
{
  "status": "ok"
}
```

Useful commands:

```bash
make docker-ps
make docker-logs
make docker-down
```

Docker Compose starts three services: `db`, one-shot `migrate`, and `app`. The app waits for PostgreSQL to become healthy and for migrations to finish successfully. This path needs Docker only; local Go and local make are not required.

If you changed database credentials, the `migrate` service receives the same values through compose variable substitution. For an already running environment, rerun migrations explicitly with:

```bash
docker compose run --rm migrate
```

## Run without Docker

Start PostgreSQL yourself and configure `.env` with the correct connection values.

```bash
make migrate-up
make run
```

Equivalent direct commands:

```bash
go run github.com/pressly/goose/v3/cmd/goose@latest -dir ./migrations postgres "user=postgres password=postgres host=127.0.0.1 port=5432 dbname=messenger_db sslmode=disable" up
go run ./cmd/app
```

Local API URL:

```text
http://localhost:8080
```

## Migrations

Migrations live in [migrations](migrations). Expected order:

```text
users
↓
chats
↓
chat_members
↓
direct_chats
↓
messages
↓
read-status
```

Commands:

```bash
make migrate-status
make migrate-up
make migrate-down
```

The app binary does not run migrations itself. In Docker Compose, the one-shot `migrate` service applies migrations before `app` starts. In production, keep the same order: run migrations before starting the app version that needs the new schema.

## Makefile

```bash
make run
make test
make test-race
make vet

make migrate-up
make migrate-down
make migrate-status
make migrate-create name=add_some_table

make docker-build
make docker-up
make docker-migrate
make docker-down
make docker-reset
make docker-logs
make docker-ps
```

The Makefile is written for GNU Make on Linux/macOS. Docker startup does not depend on Makefile; use `docker compose up -d --build` on any machine with Docker Compose.

## CI

GitHub Actions workflow: [.github/workflows/ci.yml](.github/workflows/ci.yml).

It runs on pull requests to `main` and pushes to `main`.

Checks:

- `go mod tidy` cleanliness;
- `go test ./...`;
- repository integration tests with PostgreSQL and `TEST_DATABASE_URL`;
- `go test -race ./...`;
- `go vet ./...`.

## Quality gate

Before release:

```bash
go test ./...
go vet ./...
go test -race ./...
```

Race tests require cgo and a C compiler. On Windows, if you see:

```text
cgo: C compiler "gcc" not found
```

install a C compiler or rely on the Linux CI race job.

Repository integration tests:

```bash
set TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/messenger_test?sslmode=disable
go test ./internal/repository/...
```

Unix-like shell:

```bash
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/messenger_test?sslmode=disable go test ./internal/repository/...
```

## REST API

Public endpoints:

```text
GET  /health
POST /api/v1/auth/register
POST /api/v1/auth/login
```

All other `/api/v1` endpoints require:

```text
Authorization: Bearer <JWT>
```

Endpoint list:

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/me

GET  /api/v1/chats
POST /api/v1/chats/direct
POST /api/v1/chats/group
GET  /api/v1/chats/{chatID}

GET    /api/v1/chats/{chatID}/members
POST   /api/v1/chats/{chatID}/members/{userID}
DELETE /api/v1/chats/{chatID}/members/{userID}

POST /api/v1/chats/{chatID}/messages
GET  /api/v1/chats/{chatID}/messages
GET  /api/v1/chats/{chatID}/messages/search?q=hello

POST /api/v1/chats/{chatID}/read

GET /api/v1/ws
```

### Register

```json
{
  "username": "alice",
  "password": "password123"
}
```

Response:

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer"
}
```

### Login

```json
{
  "username": "alice",
  "password": "password123"
}
```

Response:

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer"
}
```

### Current user

```http
GET /api/v1/me
Authorization: Bearer <JWT>
```

```json
{
  "id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "username": "alice",
  "created_at": "2026-09-24T10:00:00Z"
}
```

### Create direct chat

```json
{
  "user2_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
}
```

```json
{
  "id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "type": "direct",
  "title": null,
  "created_by": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "created_at": "2026-09-24T10:00:00Z",
  "unread_count": 0
}
```

Creating the same direct chat in reverse order returns the existing direct chat.

### Create group chat

```json
{
  "title": "Backend team",
  "user_ids": [
    "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
  ]
}
```

### List chats

```json
[
  {
    "id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "type": "direct",
    "title": null,
    "created_by": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "created_at": "2026-09-24T10:00:00Z",
    "last_read_message_id": 123,
    "unread_count": 2
  }
]
```

### Members

```http
GET /api/v1/chats/{chatID}/members
Authorization: Bearer <JWT>
```

```json
[
  {
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "user_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "role": "admin",
    "joined_at": "2026-09-24T10:00:00Z",
    "last_read_message_id": 123
  }
]
```

Group admins can manage members:

```text
POST   /api/v1/chats/{chatID}/members/{userID}
DELETE /api/v1/chats/{chatID}/members/{userID}
```

### Send message

```json
{
  "content": "hello bob"
}
```

Response:

```json
{
  "id": 123,
  "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "content": "hello bob",
  "created_at": "2026-09-24T10:00:00Z"
}
```

After saving, online chat members receive `message_created`.

### History

```http
GET /api/v1/chats/{chatID}/messages?limit=50
```

```json
{
  "messages": [
    {
      "id": 123,
      "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
      "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
      "content": "hello bob",
      "created_at": "2026-09-24T10:00:00Z"
    }
  ],
  "next_cursor": 120
}
```

Next page:

```http
GET /api/v1/chats/{chatID}/messages?limit=50&before_id=120
```

### Search

```http
GET /api/v1/chats/{chatID}/messages/search?q=hello
```

Search is case-insensitive and scoped to the current chat.

### Mark read

```json
{
  "message_id": 123
}
```

Response:

```json
{
  "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "user_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
  "last_read_message_id": 123
}
```

Read position only moves forward.

## Error format

```json
{
  "error": {
    "code": "chat_not_found",
    "message": "chat not found"
  }
}
```

Important codes:

```text
invalid_credentials
invalid_request
invalid_id
chat_not_found
message_not_found
forbidden
rate_limited
internal_error
```

## WebSocket API

Endpoint:

```text
GET /api/v1/ws
```

JWT is required:

```text
Authorization: Bearer <JWT>
```

Local URL:

```text
ws://localhost:8080/api/v1/ws
```

HTTPS deployment URL:

```text
wss://<your-domain>/api/v1/ws
```

### ping

Client:

```json
{
  "type": "ping"
}
```

Server:

```json
{
  "type": "pong"
}
```

### send_message

Client:

```json
{
  "type": "send_message",
  "data": {
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "content": "hello"
  }
}
```

The sender is always taken from JWT. `sender_id` from the payload is ignored.

### message_ack

Sent to the sending WebSocket client after the message is saved.

```json
{
  "type": "message_ack",
  "data": {
    "id": 123,
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "content": "hello",
    "created_at": "2026-09-24T10:00:00Z"
  }
}
```

### message_created

Broadcast to online chat members after a successful insert.

```json
{
  "type": "message_created",
  "data": {
    "id": 123,
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "content": "hello",
    "created_at": "2026-09-24T10:00:00Z"
  }
}
```

### read_updated

Broadcast to other online chat members when a read position advances.

```json
{
  "type": "read_updated",
  "data": {
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "user_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
    "last_read_message_id": 123
  }
}
```

### error

```json
{
  "type": "error",
  "error": {
    "code": "rate_limited",
    "message": "too many messages"
  }
}
```

Malformed JSON and unsupported event types do not close the connection. The server sends an error event and keeps reading.

## Manual smoke checklist

Use Alice, Bob, and Carol.

Authentication:

- register Alice and Bob;
- login Alice and Bob;
- call `GET /api/v1/me`;
- verify invalid login returns `invalid_credentials`.

Direct chat:

- Alice creates a direct chat with Bob;
- Bob creates a direct chat with Alice;
- both calls return the same chat.

Group chat:

- Alice creates a group with Bob;
- Alice is admin;
- Bob is member;
- Alice can add/remove Carol;
- Bob cannot perform admin mutations.

Messages and realtime:

- Alice sends through REST;
- REST returns `201`;
- Bob receives `message_created`;
- Alice sends through WebSocket;
- Alice receives `message_ack`;
- Bob receives `message_created`.

History and search:

- create several messages;
- check `limit`, `before_id`, and `next_cursor`;
- verify no duplicate messages between pages;
- search `q=hello`;
- verify search does not return messages from other chats.

Offline reconnect:

```text
Bob disconnects WebSocket
Alice sends a message
Bob receives no realtime event
Bob reconnects
Bob calls REST history
The missed message exists
```

Read status:

- Alice sends three messages to Bob;
- Bob has `unread_count = 3`;
- Bob reads up to the second message;
- Bob has `unread_count = 1`;
- Bob reads up to the third message;
- Bob has `unread_count = 0`;
- Bob's own messages do not increase Bob's unread count;
- Alice receives `read_updated` when Bob marks read;
- stale reads do not roll state back.

Outsider:

- Carol is not a member of Alice/Bob chat;
- Carol cannot receive realtime events, read history, search, send, mark read, or get members for that chat;
- hidden chat endpoints should return `404 chat_not_found` where applicable.

Rate limit:

- exceed Alice's message limit;
- REST returns `429 rate_limited`;
- WebSocket returns `error / rate_limited`;
- no DB insert, `message_ack`, or `message_created` happens for a limited send;
- after the window expires, sending works again.

Graceful shutdown:

- keep Alice and Bob connected through WebSocket;
- stop the server with Ctrl+C or SIGTERM;
- HTTP stops accepting new requests;
- WebSocket clients close;
- handlers finish;
- PostgreSQL pool closes;
- no panic.

Do not log passwords, JWTs, Authorization headers, or secrets.

## Postman collection

REST smoke collection:

[docs/postman/go-vk-messenger.postman_collection.json](docs/postman/go-vk-messenger.postman_collection.json)

Suggested variables:

```text
base_url=http://localhost:8080
alice_token=
bob_token=
alice_id=
bob_id=
chat_id=
message_id=
```

For WebSocket testing, use two WebSocket tabs in Postman: Alice and Bob. Connect to `ws://localhost:8080/api/v1/ws` with the matching `Authorization` header.

## Deployment notes

Configuration must come from environment variables. Do not bake `.env` or secrets into the image.

Deployment flow:

```text
provision PostgreSQL
↓
set production environment variables
↓
run migrations with a one-shot job/container
↓
deploy/start app
↓
check /health
↓
run REST and WebSocket smoke tests
```

For HTTPS deployments, verify WebSocket through `wss://`.