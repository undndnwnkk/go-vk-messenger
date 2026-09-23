# Go VK Messenger

Backend небольшого мессенджера на Go. В проекте есть регистрация и логин, JWT-аутентификация, direct/group chats, PostgreSQL persistence, REST history, поиск, WebSocket realtime delivery, read status, unread counters, rate limiting, Docker startup и SQL migrations.

English documentation: [README.md](README.md).

## Возможности

Authentication:

- регистрация;
- логин;
- JWT access tokens;
- `GET /api/v1/me`.

Chats:

- direct chats;
- group chats;
- membership;
- group admins;
- hidden access policy для приватных ресурсов чата.

Messages:

- отправка через REST;
- отправка через WebSocket;
- хранение в PostgreSQL;
- cursor pagination для history;
- поиск внутри чата.

Realtime:

- WebSocket endpoint;
- несколько подключений одного пользователя;
- realtime `message_created`;
- realtime `read_updated`;
- восстановление после reconnect через REST history.

Read status:

- `chat_members.last_read_message_id`;
- `unread_count` в `GET /api/v1/chats`;
- read-position двигается только вперёд.

Protection and lifecycle:

- per-user message rate limit;
- slow WebSocket clients удаляются из Hub;
- graceful shutdown закрывает WebSocket clients и PostgreSQL pool.

## Архитектура

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

Проект сделан как modular monolith. Это не microservices, потому что для MVP достаточно одного deployment unit, меньше operational complexity, а границы controller/service/repository уже есть внутри кода.

PostgreSQL — source of truth. Realtime — только доставка:

```text
INSERT message
↓
message_ack отправителю
↓
message_created online-участникам чата
```

Если broadcast упал после успешного INSERT, сообщение всё равно считается созданным. Клиент может восстановить состояние через REST history.

WebSocket не хранит offline queue. Если Bob offline, пока Alice отправляет сообщение, Bob не получит realtime event. После reconnect Bob вызывает `GET /api/v1/chats/{chatID}/messages` и забирает пропущенные сообщения из PostgreSQL.

Read status хранится через:

```text
chat_members.last_read_message_id
```

а не через таблицу вида:

```text
message_reads(user_id, message_id)
```

Redis/NATS сейчас не используется, потому что приложение single-instance. Если появится несколько Go instances, in-memory Hub понадобится заменить или дополнить external Pub/Sub.

## Структура проекта

```text
cmd/app/                 entrypoint приложения
internal/config/         загрузка и валидация environment config
internal/controller/     HTTP и WebSocket handlers
internal/model/          DTOs и domain models
internal/realtime/       Client, Hub, WebSocket events
internal/repository/     PostgreSQL repositories
internal/service/        business logic
migrations/              goose SQL migrations
.github/workflows/       CI workflow
```

## Требования

- Go версии из [go.mod](go.mod).
- Docker и Docker Compose.
- PostgreSQL для запуска без Docker.
- GNU Make для удобных команд или прямые Go/Docker команды.

Локальный Makefile запускает goose через:

```bash
go run github.com/pressly/goose/v3/cmd/goose@latest
```

Глобально установленный goose не нужен.

## Конфигурация

Создай локальный env файл:

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

Для Docker Compose app container переопределяет `DB_HOST=db`, потому что внутри docker network PostgreSQL доступен по имени service, а не через `localhost`. Для PostgreSQL 18 named volume монтируется в `/var/lib/postgresql`, что соответствует актуальному data root layout официального image PostgreSQL 18+.

Нельзя коммитить реальные production passwords, JWT secrets, access tokens, deployment credentials, dumps, logs и `.env`. `.env` находится в `.gitignore`.

Startup validation:

- без `JWT_SECRET` приложение падает при старте с понятной ошибкой;
- некорректный `JWT_TTL` ломает startup;
- недоступный PostgreSQL ломает startup на `Ping`, до обработки запросов.

## Запуск через Docker

```bash
git clone <repository-url>
cd go-vk-messenger
cp .env.example .env
docker compose up -d --build
curl http://localhost:8080/health
```

Ожидаемый ответ:

```json
{
  "status": "ok"
}
```

Полезные команды:

```bash
make docker-ps
make docker-logs
make docker-down
```

Docker Compose запускает три service: `db`, одноразовый `migrate` и `app`. App ждёт, пока PostgreSQL станет healthy и migrations успешно завершатся. Этот сценарий требует только Docker; локальные Go и make не нужны.

Если поменял database credentials, `migrate` service получает те же значения через compose variable substitution. Для уже запущенного окружения миграции можно повторно запустить явно:

```bash
docker compose run --rm migrate
```

## Запуск без Docker

Подними PostgreSQL сам и настрой `.env`.

```bash
make migrate-up
make run
```

Эквивалентные прямые команды:

```bash
go run github.com/pressly/goose/v3/cmd/goose@latest -dir ./migrations postgres "user=postgres password=postgres host=127.0.0.1 port=5432 dbname=messenger_db sslmode=disable" up
go run ./cmd/app
```

Local API URL:

```text
http://localhost:8080
```

## Миграции

Миграции лежат в [migrations](migrations). Ожидаемый порядок:

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

Команды:

```bash
make migrate-status
make migrate-up
make migrate-down
```

Бинарник приложения не применяет миграции сам. В Docker Compose одноразовый `migrate` service применяет миграции до старта `app`. В production сохраняй тот же порядок: миграции до запуска версии приложения, которой нужна новая схема.

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

Makefile написан под GNU Make на Linux/macOS. Docker startup не зависит от Makefile; на любой машине с Docker Compose можно использовать `docker compose up -d --build`.

## CI

GitHub Actions workflow: [.github/workflows/ci.yml](.github/workflows/ci.yml).

Он запускается на pull request в `main` и на push в `main`.

Проверки:

- чистый `go mod tidy`;
- `go test ./...`;
- repository integration tests с PostgreSQL и `TEST_DATABASE_URL`;
- `go test -race ./...`;
- `go vet ./...`.

## Quality gate

Перед release:

```bash
go test ./...
go vet ./...
go test -race ./...
```

Race tests требуют cgo и C compiler. Если на Windows видишь:

```text
cgo: C compiler "gcc" not found
```

установи C compiler или полагайся на Linux CI job.

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

Остальные `/api/v1` endpoints требуют:

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

### Direct chat

```json
{
  "user2_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
}
```

Повторное создание direct chat в обратном порядке возвращает тот же чат.

### Group chat

```json
{
  "title": "Backend team",
  "user_ids": [
    "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
  ]
}
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

После сохранения online-участники чата получают `message_created`.

### History

```http
GET /api/v1/chats/{chatID}/messages?limit=50
GET /api/v1/chats/{chatID}/messages?limit=50&before_id=120
```

Response:

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

### Search

```http
GET /api/v1/chats/{chatID}/messages/search?q=hello
```

Search case-insensitive и ограничен текущим чатом.

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

Read position двигается только вперёд.

## Error format

```json
{
  "error": {
    "code": "chat_not_found",
    "message": "chat not found"
  }
}
```

Важные codes:

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

JWT обязателен:

```text
Authorization: Bearer <JWT>
```

Local URL:

```text
ws://localhost:8080/api/v1/ws
```

HTTPS deployment:

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

Sender всегда берётся из JWT. `sender_id` из payload игнорируется.

### message_ack

Отправляется только WebSocket-клиенту отправителя после сохранения сообщения.

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

Рассылается online-участникам чата после успешного INSERT.

### read_updated

Рассылается другим online-участникам чата, когда read position реально продвинулся.

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

Malformed JSON и unsupported event type не закрывают connection. Сервер отправляет error event и продолжает читать.

## Manual smoke checklist

Используй Alice, Bob и Carol.

- Register Alice/Bob.
- Login Alice/Bob.
- `GET /api/v1/me`.
- Invalid login возвращает `invalid_credentials`.
- Alice создаёт direct chat с Bob.
- Bob создаёт direct chat с Alice, возвращается тот же chat.
- Alice создаёт group с Bob.
- Alice admin, Bob member.
- Alice может add/remove Carol.
- Bob не может выполнять admin mutations.
- Alice отправляет REST message, Bob получает `message_created`.
- Alice отправляет WebSocket `send_message`, Alice получает `message_ack`, Bob получает `message_created`.
- History работает с `limit`, `before_id`, `next_cursor`.
- Search `q=hello` не возвращает сообщения другого чата.
- Offline Bob не получает realtime, после reconnect забирает пропущенное через REST history.
- Unread count уменьшается после mark read.
- Alice получает `read_updated`, когда Bob marks read.
- Carol outsider не может читать, искать, отправлять, mark read или получать members чужого чата.
- Rate limit возвращает REST `429 rate_limited` и WebSocket `error / rate_limited`.
- Graceful shutdown закрывает WebSocket clients и PostgreSQL pool без panic.

Не логируй passwords, JWT, Authorization headers и secrets.

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

WebSocket удобнее проверить двумя WebSocket tabs в Postman: Alice и Bob.

## Deployment notes

Конфиг должен приходить только через environment variables. Не кладите `.env` и secrets внутрь image.

Deployment flow:

```text
provision PostgreSQL
↓
set production environment variables
↓
run migrations через one-shot job/container
↓
deploy/start app
↓
check /health
↓
run REST and WebSocket smoke tests
```

Для HTTPS deployment обязательно проверить WebSocket через `wss://`.