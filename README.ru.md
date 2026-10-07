# Go VK Messenger

Backend небольшого мессенджера на Go. В проекте есть регистрация и логин, JWT-аутентификация, direct/group chats, PostgreSQL persistence, REST history, поиск, редактирование и удаление сообщений, WebSocket realtime delivery, read status, unread counters, rate limiting, Docker startup и SQL migrations.

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
- персональный mute с сохранением в БД и синхронизацией между подключениями пользователя.

Messages:

- отправка через REST;
- отправка через WebSocket;
- хранение в PostgreSQL;
- cursor pagination для history;
- поиск внутри чата.
- редактирование своих сообщений с отметками `edited` / `edited_at`;
- удаление своих сообщений для всех с отметками `deleted` / `deleted_at`, с сохранением ID и курсоров прочтения.
- ответы на сообщения внутри чата через REST и WebSocket с актуальной цитатой оригинала.
- реакции: like, heart, laugh, wow, sad, angry; добавление и снятие своих реакций.

Realtime:

- WebSocket endpoint;
- несколько подключений одного пользователя;
- realtime `message_created`;
- realtime `message_edited` и `message_deleted`;
- realtime `read_updated`;
- realtime `reactions_updated` и персональное событие `chat_mute_updated`;
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
↓
message-edit-delete
↓
message-replies
↓
message-reactions
↓
chat-mute
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

После сохранения online-участники чата получают `message_created`.

Все объекты сообщений в REST и WebSocket также содержат `edited`, `edited_at`, `deleted`, `deleted_at`. Для нового сообщения флаги равны `false`, даты — `null`.

### Ответ на сообщение (Reply)

Для ответа используй существующий endpoint отправки с необязательным полем `reply_to_message_id`:

```http
POST /api/v1/chats/{chatID}/messages
Authorization: Bearer <JWT>
Content-Type: application/json

{"content":"Да, договорились","reply_to_message_id":123}
```

Через WebSocket:

```json
{
  "type": "send_message",
  "data": {
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "content": "Да, договорились",
    "reply_to_message_id": 123
  }
}
```

REST возвращает `201`, WebSocket — `message_ack`; участники получают обычный `message_created`. Во всех объектах сообщений, включая историю, поиск и события, добавлены два поля. Для ответа они выглядят так:

```json
{
  "reply_to_message_id": 123,
  "reply_to": {
    "id": 123,
    "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "content": "Встретимся завтра?",
    "edited": false,
    "edited_at": null,
    "deleted": false,
    "deleted_at": null
  }
}
```

Если не передавать `reply_to_message_id` или указать `null`, сообщение будет обычным: оба поля ответа равны `null`. Оригинал должен существовать, быть неудалённым и относиться к тому же чату. Можно отвечать на свои и чужие сообщения, в том числе на ответы. Текст ответа обязателен; действуют обычные ограничения на длину и частоту отправки. Ответ учитывается как новое непрочитанное сообщение для других участников. Редактирование ответа меняет только текст, но не ссылку на оригинал.

`reply_to` содержит актуальную цитату на один уровень, без рекурсивной цепочки. Она доступна даже тогда, когда оригинал не попал на текущую страницу истории. После правки оригинала последующие загрузки возвращают исправленную цитату. После удаления оригинала ответы сохраняются, а у цитаты текст пустой и `deleted: true`. Новый ответ на удалённый оригинал запрещён (`404 message_not_found`). Поиск ищет по собственному тексту ответа, а не по цитате.

Неположительный ID даёт `400 invalid_message_id`, неверный JSON-тип или дробное число — `400 invalid_request` (WebSocket: `invalid_payload`). Несуществующий оригинал или сообщение другого чата — `404 message_not_found`; отсутствие доступа к чату — `404 chat_not_found`. При ошибке сообщение не создаётся, ack и рассылки нет.

При `message_edited` / `message_deleted` клиент обновляет цитаты, чей `reply_to_message_id` совпадает с ID изменённого сообщения. Сервер рассылает событие оригинала, без отдельного события для каждого ответа. После reconnect нужно перезагрузить соответствующие страницы истории; старый текст удалённого оригинала в цитатах сохранять нельзя.

Нужна миграция `20261007000200_add_message_replies.sql`: примените `make migrate-up` или пересоберите Docker-образы и запустите миграции командами из раздела ниже. Миграция добавляет ссылку на оригинал и ограничение БД, запрещающее ссылки между чатами. У существующих сообщений ссылка изначально `NULL`. Откат сохраняет сообщения, но удаляет связи ответов.

### Редактирование сообщения

```http
PATCH /api/v1/chats/{chatID}/messages/123
Authorization: Bearer <JWT>
Content-Type: application/json

{"content":"Привет, исправленный текст"}
```

Ответ: `200 OK` с обновлённым сообщением:

```json
{
  "id": 123,
  "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "content": "Привет, исправленный текст",
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

Редактировать сообщение может только автор, который по-прежнему состоит в чате. Администратор группы не может редактировать чужие сообщения. Текст не должен состоять только из пробелов; максимум — 4000 Unicode-символов. Каждое успешное редактирование обновляет `edited_at`, даже если текст совпадает с прежним. ID, автор и `created_at` не меняются, счётчик непрочитанных не увеличивается. История и поиск возвращают актуальный текст.

### Удаление сообщения

```http
DELETE /api/v1/chats/{chatID}/messages/123
Authorization: Bearer <JWT>
```

Ответ: `200 OK` с объектом сообщения, у которого `content: ""`, `deleted: true`, `deleted_at` — время удаления. Отметки предыдущего редактирования сохраняются. Удаление доступно только автору — участнику чата и применяется для всех.

Текст очищается в БД, но запись с исходными ID, автором и датой создания сохраняется. История возвращает её как заглушку для отображения «Сообщение удалено». Поиск и счётчик непрочитанных не учитывают удалённые сообщения. Курсор прочтения не сбрасывается; его можно продвинуть до ID удалённого сообщения.

Ошибки обоих endpoints: `400 invalid_message_id` для некорректного или неположительного ID; `404 chat_not_found` для постороннего или исключённого участника; `404 message_not_found` для отсутствующего, удалённого или относящегося к другому чату сообщения; `403 forbidden` для чужого сообщения. Повторное удаление и редактирование после удаления возвращают `404 message_not_found`.

Для обновления существующей Docker-установки сначала пересоберите образы и примените миграцию `20261007000100_add_message_edit_delete.sql`:

```bash
docker compose build migrate app
docker compose run --rm migrate
docker compose up -d app
```

Без Docker: `make migrate-up` перед запуском новой версии. Миграция добавляет `edited_at` и `deleted_at`; у существующих сообщений обе даты изначально `NULL`. Откат заменяет удалённый текст на `[deleted]`, сохраняя ID и курсоры; исходный текст восстановить нельзя.

### Реакции

Код реакции указывается в URL, тело запроса не требуется:

| Код | Отображение |
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

Для снятия своей реакции — `DELETE` на тот же URL. Оба запроса возвращают `200 OK` с полным текущим состоянием:

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

Участник чата может реагировать на свои и чужие неудалённые сообщения. Можно поставить несколько разных реакций, но каждую — только один раз. Повторные PUT/DELETE идемпотентны: если состояние не изменилось, счётчик и версия не меняются, рассылки нет. Пользователь определяется по JWT, чужие реакции снять нельзя. После исключения участник не может менять реакции, но ранее поставленные реакции сохраняются.

Все объекты сообщений в истории, поиске и WebSocket содержат `reactions` (пустой массив при отсутствии реакций) и `reactions_version`. В каждой группе есть код, количество и ID пользователей. Редактирование сохраняет реакции, удаление сообщения очищает их. Реакции не влияют на прочитанность и счётчик непрочитанных.

Ошибки: `400 invalid_reaction`, `400 invalid_message_id`; для постороннего — `404 chat_not_found`; для отсутствующего, удалённого или принадлежащего другому чату сообщения — `404 message_not_found`. Изменения выполняются через REST, события доставляются через WebSocket.

### Mute чата

```http
PATCH /api/v1/chats/{chatID}/mute
Authorization: Bearer <JWT>
Content-Type: application/json

{"muted":true}
```

Чтобы включить уведомления обратно, передай `{"muted":false}`. Поле обязательно; отсутствие, `null` или неверный тип дают `400 invalid_request`. Ответ (`200 OK`):

```json
{
  "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "muted": true,
  "mute_version": 1
}
```

Mute — личная настройка участника, сохраняемая в `chat_members`; работает для личных и групповых чатов. Права администратора не нужны. Менять можно только свою настройку. Посторонний получает `404 chat_not_found`. Список чатов, `GET /api/v1/chats/{chatID}` и повторное открытие существующего direct chat возвращают `muted` и `mute_version` текущего пользователя. Чужие настройки не выдаются в списке участников.

Backend хранит флаг, а клиент использует его для отключения звука и всплывающих уведомлений. Доставка WebSocket-сообщений, реакций, статусов прочтения, история и счётчики непрочитанных продолжают работать. Отдельного push-сервиса в проекте нет. Новое участие в чате начинается с `muted: false`, `mute_version: 0`; после удаления и повторного добавления участника настройка сбрасывается. Повтор того же состояния не создаёт событие.

Для этих функций нужны миграции `20261007000300_add_message_reactions.sql` и `20261007000400_add_chat_mute.sql`. Применить до запуска новой версии:

```bash
docker compose build migrate app
docker compose run --rm migrate
docker compose up -d app
```

Без Docker: `make migrate-up`. У существующих сообщений реакции изначально отсутствуют, у участников mute выключен. Откат удаляет реакции и настройки mute, сохраняя сообщения и участников.

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

Рассылается online-участникам чата после успешного INSERT.

### message_edited / message_deleted

Редактирование и удаление выполняются через REST (`PATCH` / `DELETE`). WebSocket-команд для этих операций нет. После успешного сохранения все online-участники чата, включая все подключения автора, получают событие с полным обновлённым объектом сообщения:

```json
{
  "type": "message_edited",
  "data": {
    "id": 123,
    "chat_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    "sender_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "content": "Привет, исправленный текст",
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

При удалении тип — `message_deleted`, `data.content` — пустая строка, `data.deleted` — `true`, `data.deleted_at` — время удаления. Клиент заменяет сообщение по ID, скрывает удалённый текст и обновляет счётчики чатов. При одновременных запросах события могут прийти не по порядку: удаление окончательно, последующие события редактирования такого ID нужно игнорировать; среди редактирований выбирать самое новое по `edited_at`.

Ошибка рассылки не отменяет сохранённое изменение. Офлайн-очереди событий нет: после reconnect нужно перезагрузить страницы истории с закешированными сообщениями. Загрузки только новых ID недостаточно для получения пропущенных правок и удалений.

### reactions_updated

После реального изменения реакций все online-участники, включая все подключения автора действия, получают:

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

Клиент заменяет список реакций по ID сообщения. Если ответы или события пришли не по порядку, сохраняй состояние с большей `reactions_version`, в том числе при получении реакций внутри других событий сообщения. Удаление окончательно: поздние реакции удалённого сообщения игнорируются. Ошибка рассылки не отменяет сохранение; после reconnect актуальные реакции доступны через историю.

### chat_mute_updated

Событие получают только подключения пользователя, изменившего настройку, включая его текущее устройство:

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

`mute_version` позволяет игнорировать запоздавшие изменения в рамках текущего участия в чате. После reconnect или повторного вступления загрузи настройки из списка чатов. Другие участники не получают событие о твоём mute.

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
- Alice/Bob ставят одинаковую реакцию: счётчик 2; повторный PUT не меняет счётчик или версию.
- Alice снимает свою реакцию, реакция Bob остаётся; участники получают `reactions_updated`, посторонние — нет.
- Удалённое сообщение не принимает реакции, история и поиск показывают актуальный список.
- Bob включает mute: событие получают только его подключения; сообщения по-прежнему доставляются, непрочитанные считаются.
- После reconnect mute сохраняется; `muted: false` выключает его. Посторонний не может менять настройки или реакции.
- Alice отправляет WebSocket `send_message`, Alice получает `message_ack`, Bob получает `message_created`.
- Bob отвечает на сообщение Alice через REST и WebSocket: ответ/ack и `message_created` содержат `reply_to_message_id` и цитату оригинала.
- Страница истории с ответом содержит цитату, даже если оригинала на этой странице нет.
- После правки/удаления оригинала повторная загрузка показывает обновлённую/пустую цитату; собственный текст ответа сохраняется.
- Ответ на отсутствующее, удалённое или принадлежащее другому чату сообщение отклоняется.
- Без `reply_to_message_id` или с `null` обычная отправка работает как прежде.
- Alice редактирует своё сообщение через `PATCH`: `200`, `edited = true`, участники получают `message_edited`.
- Bob не может изменить или удалить сообщение Alice (`403 forbidden`), Carol вне чата получает `404 chat_not_found`.
- История и поиск показывают исправленный текст; поиск по старому тексту больше не находит сообщение.
- Alice удаляет сообщение через `DELETE`: `200`, участники получают `message_deleted`, в истории остаётся пустая заглушка.
- Удалённое сообщение исчезает из поиска и счётчика непрочитанных; курсор прочтения сохраняется.
- Повторное удаление или редактирование удалённого сообщения возвращает `404 message_not_found`.
- После reconnect перезагрузка соответствующих страниц истории показывает правки и удаления.
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

В коллекции есть создание direct chat, создание group chat, обычный message flow и папка `Rate Limit`. Папку `Rate Limit` запускай через Collection Runner без задержек между запросами; перед повторным запуском подожди минимум 5 секунд.

Папка `Reactions and Mute` проверяет добавление, повтор и снятие реакции, включение и выключение mute. Сначала выполни запросы авторизации, создания direct chat и отправки сообщения; `message_id` должен указывать на неудалённое сообщение из `chat_id`.

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
