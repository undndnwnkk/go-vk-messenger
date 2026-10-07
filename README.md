# Go VK Messenger

Backend for a small messenger written in Go. It supports registration and login, JWT authentication, direct and group chats, PostgreSQL persistence, REST message history, search, message editing and deletion, WebSocket realtime delivery, read status, unread counters, rate limiting, Docker startup, and SQL migrations.

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
- personal mute settings, synchronized across the user's connections.

Messages:

- send through REST;
- send through WebSocket;
- PostgreSQL persistence;
- cursor history pagination;
- chat-scoped search.
- author-only editing with `edited` / `edited_at`;
- deletion for everyone with `deleted` / `deleted_at`, preserving message IDs and read cursors.
- replies within a chat through REST and WebSocket, with a live preview of the original message.
- reactions: like, heart, laugh, wow, sad, angry; add/remove your own reactions.

Realtime:

- WebSocket endpoint;
- multiple connections per user;
- realtime `message_created`;
- realtime `message_edited` and `message_deleted`;
- realtime `read_updated`;
- realtime `reactions_updated` and private `chat_mute_updated`;
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
↓
message-edit-delete
↓
message-replies
↓
message-reactions
↓
chat-mute
```

Commands:

```bash
make migrate-status
make migrate-up
make migrate-down
```

The app binary does not run migrations itself. In Docker Compose, the one-shot `migrate` service applies migrations before `app` starts. In production, keep the same order: run migrations before starting the app version that needs the new schema.

Message editing/deletion requires `20261007000100_add_message_edit_delete.sql`. Rebuild the migration image when upgrading an existing Docker environment so it includes the new SQL:

```bash
docker compose build migrate app
docker compose run --rm migrate
docker compose up -d app
```

The migration adds nullable `edited_at` and `deleted_at` columns; existing messages start unedited and undeleted. Deletion clears the stored content. Rolling this migration back replaces deleted content with `[deleted]` to preserve message IDs and read cursors; it cannot restore the original text.

Replies additionally require `20261007000200_add_message_replies.sql`, applied by the same commands. It adds nullable `reply_to_message_id` and a composite foreign key that keeps replies within the same chat. Existing messages have no reply target. Rolling back this migration preserves messages but removes their reply links.

Reactions and mute require `20261007000300_add_message_reactions.sql` and `20261007000400_add_chat_mute.sql`. Apply them with the same Docker upgrade commands above or `make migrate-up` before starting the new binary. Existing messages start with no reactions; existing memberships start unmuted. Rollbacks remove reactions/mute preferences but preserve messages and memberships.

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
PATCH  /api/v1/chats/{chatID}/messages/{messageID}
DELETE /api/v1/chats/{chatID}/messages/{messageID}
PUT    /api/v1/chats/{chatID}/messages/{messageID}/reactions/{reaction}
DELETE /api/v1/chats/{chatID}/messages/{messageID}/reactions/{reaction}

POST /api/v1/chats/{chatID}/read
PATCH /api/v1/chats/{chatID}/mute

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
  "unread_count": 0,
  "muted": false,
  "mute_version": 0
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
    "unread_count": 2,
    "muted": false,
    "mute_version": 0
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
  "created_at": "2026-09-24T10:00:00Z",
  "edited": false,
  "edited_at": null,
  "deleted": false,
  "deleted_at": null,
  "reply_to_message_id": null,
  "reply_to": null,
  "reactions_version": 0,
  "reactions": []
}
```

After saving, online chat members receive `message_created`.

All message objects (REST and WebSocket) also contain `edited`, `edited_at`, `deleted`, and `deleted_at`. New messages have both flags set to `false` and both timestamps set to `null`.

### Reply to a message

Use the existing send endpoint with the optional `reply_to_message_id`:

```http
POST /api/v1/chats/{chatID}/messages
Authorization: Bearer <JWT>
Content-Type: application/json

{"content":"Yes, agreed","reply_to_message_id":123}
```

The WebSocket equivalent uses the existing `send_message` event:

```json
{
  "type": "send_message",
  "data": {
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "content": "Yes, agreed",
    "reply_to_message_id": 123
  }
}
```

REST returns `201` and WebSocket returns `message_ack`, followed by the usual `message_created` broadcast. All message responses, history, search, and message events include two additional fields. For a reply they look like:

```json
{
  "reply_to_message_id": 123,
  "reply_to": {
    "id": 123,
    "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "content": "Shall we meet tomorrow?",
    "edited": false,
    "edited_at": null,
    "deleted": false,
    "deleted_at": null
  }
}
```

Omit `reply_to_message_id` or send `null` for an ordinary message; both response fields will be `null`. The target must exist, be undeleted, and belong to the same chat. Members may reply to their own or other members' messages, including replies. A reply still needs non-empty content, follows the normal 4000-character limit and rate limit, and counts as a new unread message for other members. Editing a reply changes only its content, not its target.

`reply_to` is a one-level preview, without a nested reply chain. It is fetched from the current original message, including when that message is outside the requested history page. Editing the original updates subsequent previews; deleting it preserves existing replies but returns an empty preview content with `deleted: true`. New replies to deleted originals return `404 message_not_found`. Search matches the reply's own content, not its preview.

Invalid non-positive target IDs return `400 invalid_message_id`; non-integer JSON values return `400 invalid_request` (WebSocket: `invalid_payload`). Missing or different-chat targets also return `404 message_not_found`. Non-members get `404 chat_not_found`; failed sends produce no ack or broadcast.

Clients should update cached previews whose `reply_to_message_id` matches an incoming `message_edited` / `message_deleted` event. The server emits the original message's event, not a separate event for each reply. After reconnect, reload relevant history pages to refresh previews; do not retain a quote's old text after the original is deleted.

### Edit message

```http
PATCH /api/v1/chats/{chatID}/messages/123
Authorization: Bearer <JWT>
Content-Type: application/json

{"content":"hello bob, corrected"}
```

Response: `200 OK` with the updated message:

```json
{
  "id": 123,
  "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "content": "hello bob, corrected",
  "created_at": "2026-09-24T10:00:00Z",
  "edited": true,
  "edited_at": "2026-10-07T10:00:00Z",
  "deleted": false,
  "deleted_at": null,
  "reply_to_message_id": null,
  "reply_to": null,
  "reactions_version": 0,
  "reactions": []
}
```

Only the author, who must still be a chat member, can edit a message. Group admins cannot edit another author's message. Content must not be whitespace-only and must contain at most 4000 Unicode characters, as for sending. Each successful edit sets `edited_at` to the latest edit time, even if the content is unchanged. `id`, `sender_id`, and `created_at` remain unchanged; edits do not increase unread counts. History and search return the current text.

### Delete message

```http
DELETE /api/v1/chats/{chatID}/messages/123
Authorization: Bearer <JWT>
```

Response: `200 OK` with the message object: `content: ""`, `deleted: true`, and a non-null `deleted_at`. Existing edit metadata is preserved. The same author/member restrictions apply as for editing; deletion applies to everyone in the chat.

The database keeps a tombstone with the original ID, sender, and creation time, but clears the text. History includes this tombstone so clients can display “Message deleted”, including after reconnect. Search and unread counts exclude deleted messages. Read cursors remain valid and can advance to a deleted message ID.

Both endpoints return `400 invalid_message_id` for a non-positive or malformed ID, `404 chat_not_found` for non-members, `404 message_not_found` for a missing, deleted, or different-chat message, and `403 forbidden` for another author's message. Repeated deletion and editing after deletion return `404 message_not_found`.

### Reactions

Use the reaction code in the path; there is no request body:

| Code | Display |
| --- | --- |
| `like` | 👍 |
| `heart` | ❤️ |
| `laugh` | 😂 |
| `wow` | 😮 |
| `sad` | 😢 |
| `angry` | 😡 |

```http
PUT /api/v1/chats/{chatID}/messages/123/reactions/like
Authorization: Bearer <JWT>
```

Use `DELETE` on the same URL to remove your own reaction. Both return `200 OK` with the complete current state:

```json
{
  "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "message_id": 123,
  "reactions_version": 1,
  "reactions": [
    {"reaction":"like","count":1,"user_ids":["aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"]}
  ]
}
```

Any current chat member can react to their own or another member's undeleted message. One user may add several different reactions, but each kind appears at most once per user/message. Repeated PUT or DELETE is idempotent: no duplicate, version increment, or broadcast if nothing changed. The acting user always comes from JWT; you cannot remove someone else's reaction. Removing a member prevents new reaction changes but preserves their existing reactions.

All message objects in REST history/search and WebSocket message events contain `reactions` (empty array when none) and `reactions_version`. Each group includes its count and user IDs, so clients can identify the signed-in user's selections. Editing preserves reactions. Deletion removes them and returns `reactions: []`; deleted messages reject further reaction mutations. Reactions do not mark messages read or change unread counts.

Errors: `400 invalid_reaction`, `400 invalid_message_id`, `404 chat_not_found` for non-members, `404 message_not_found` for missing/deleted/different-chat messages. Reaction mutations use REST; updates are delivered through WebSocket.

### Mute a chat

```http
PATCH /api/v1/chats/{chatID}/mute
Authorization: Bearer <JWT>
Content-Type: application/json

{"muted":true}
```

Send `{"muted":false}` to unmute. The boolean is required: missing, null, or a wrong type returns `400 invalid_request`. Response (`200 OK`):

```json
{
  "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "muted": true,
  "mute_version": 1
}
```

Mute is a personal setting for direct and group chats, persisted in `chat_members`. Any member may change their own setting; admin rights are unnecessary. Non-members get `404 chat_not_found`. Other users' settings cannot be changed or read through this endpoint. `GET /api/v1/chats`, `GET /api/v1/chats/{chatID}`, and reopening an existing direct chat return the caller's `muted` and `mute_version`.

This backend stores the notification flag; the client uses it to suppress sound/pop-up notifications. WebSocket messages, reactions, read updates, history, and unread counts continue working while muted. There is no push-notification service in this project. New memberships default to `muted: false`, `mute_version: 0`; leaving and rejoining resets the setting. Repeating the same state is idempotent and emits no event.

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
      "created_at": "2026-09-24T10:00:00Z",
      "edited": false,
      "edited_at": null,
      "deleted": false,
      "deleted_at": null,
      "reply_to_message_id": null,
      "reply_to": null,
      "reactions_version": 0,
      "reactions": []
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
    "created_at": "2026-09-24T10:00:00Z",
    "edited": false,
    "edited_at": null,
    "deleted": false,
    "deleted_at": null,
    "reply_to_message_id": null,
    "reply_to": null,
    "reactions_version": 0,
    "reactions": []
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
    "created_at": "2026-09-24T10:00:00Z",
    "edited": false,
    "edited_at": null,
    "deleted": false,
    "deleted_at": null,
    "reply_to_message_id": null,
    "reply_to": null,
    "reactions_version": 0,
    "reactions": []
  }
}
```

### message_edited / message_deleted

Editing and deletion are requested through REST (`PATCH` / `DELETE`); there are no WebSocket mutation commands for them. After the database change succeeds, all online chat members, including every connection of the author, receive an event with the full updated message:

```json
{
  "type": "message_edited",
  "data": {
    "id": 123,
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "content": "hello bob, corrected",
    "created_at": "2026-09-24T10:00:00Z",
    "edited": true,
    "edited_at": "2026-10-07T10:00:00Z",
    "deleted": false,
    "deleted_at": null,
    "reply_to_message_id": null,
    "reply_to": null,
    "reactions_version": 0,
    "reactions": []
  }
}
```

For deletion, `type` is `message_deleted`, `data.content` is empty, `data.deleted` is `true`, and `data.deleted_at` is the deletion time. Clients should replace their cached message by ID, hide deleted content, and refresh chat unread counts after deletion. Concurrent requests can deliver events out of order: a deletion is final, so ignore later edit events for a deleted ID; for edits, keep the newest `edited_at`.

As with creation, a broadcast failure does not roll back a saved change. There is no offline event queue: after reconnect, reload history pages containing cached messages to reconcile edits and deletions (fetching only newer IDs is insufficient).

### reactions_updated

After an actual reaction change, all online chat members (including every connection of the actor) receive:

```json
{
  "type": "reactions_updated",
  "data": {
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "message_id": 123,
    "reactions_version": 1,
    "reactions": [
      {"reaction":"like","count":1,"user_ids":["aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"]}
    ]
  }
}
```

Replace the cached reaction state by message ID. For out-of-order responses/events, keep the largest `reactions_version`, including reaction data embedded in other message events. A deleted message stays deleted; ignore late reaction events for it. A failed broadcast does not roll back a saved reaction; reload history after reconnect.

### chat_mute_updated

Only the user's own online connections receive this event, including the connection on the device that initiated the REST request:

```json
{
  "type": "chat_mute_updated",
  "data": {
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "muted": true,
    "mute_version": 1
  }
}
```

Use `mute_version` to ignore stale settings events within a membership. After reconnect or rejoining a chat, reload settings from the chat list. Other members never receive your mute event.

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

Reactions and mute:

- Alice and Bob add the same reaction: count is 2; Alice repeats PUT: count/version stay unchanged;
- Alice removes her reaction: Bob's remains; history/search show the current count;
- participants receive `reactions_updated`, outsiders do not; a deleted message rejects reactions;
- Bob mutes a chat: all Bob's connections receive `chat_mute_updated`, Alice receives nothing;
- Bob still receives messages while muted, and unread counts increase normally;
- the chat list retains Bob's setting after reconnect; unmute restores `muted: false`;
- non-members cannot change mute or reactions.

Replies:

- Bob sends `reply_to_message_id` referencing Alice's message through REST and WebSocket;
- response/ack and `message_created` include the reply ID and the server's original-message preview;
- load a history page without the original: the reply preview is still present;
- edit/delete the original and reload history: its preview changes/clears, while the reply's own text remains;
- missing, deleted, and different-chat targets are rejected without creating a message;
- omit the target or send `null`: ordinary sending works as before.

Editing and deletion:

- Alice edits her message with `PATCH`; response is `200`, `edited = true`, and `edited_at` is set;
- Alice and Bob receive `message_edited`; Carol, outside the chat, receives nothing;
- Bob cannot edit/delete Alice's message (`403 forbidden`); Carol gets `404 chat_not_found`;
- history/search show the edited content; the old text no longer matches;
- Alice deletes the message with `DELETE`; response is `200` and members receive `message_deleted`;
- history contains an empty tombstone; search excludes it; deleting an unread message reduces unread count;
- deleting a message used as a read cursor preserves the cursor and does not resurrect old unread messages;
- editing/deleting the tombstone returns `404 message_not_found`;
- reconnect and reload the relevant history pages to see edits and tombstones.

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
carol_username=carol
carol_token=
alice_id=
bob_id=
carol_id=
chat_id=
group_chat_id=
group_message_id=
message_id=
```

The collection includes direct chat creation, group chat creation, regular message flow, and a `Rate Limit` folder. Run the `Rate Limit` folder in Collection Runner without delay between requests; wait at least 5 seconds before rerunning it.

The `Reactions and Mute` folder checks reaction add/repeat/remove and mute/unmute. Run the authentication, direct-chat, and message setup requests first; `message_id` must refer to an undeleted message in `chat_id`.

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
