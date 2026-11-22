# 0006. Chat list summaries and user search

Status: accepted (2026-11-22)

## Context

The chat list design needs, per row, the other user's name for direct chats, a
preview of the latest message with its author, and the time of last activity.
The API returned only chat rows, so the client would have needed N extra requests
(members, users, messages) per list. Starting a chat needs a way to find users;
the API only accepted a user UUID.

## Decision

- `GET /chats` and `GET /chats/{chatID}` embed `peer` ({id, username}, direct
  chats only) and `last_message` ({id, sender_id, sender_username, content,
  deleted, created_at}). One query: `direct_chats` gives the peer, a `LATERAL`
  lookup on the existing `(chat_id, id DESC)` index gives the latest message.
- Deleted messages stay eligible as `last_message` with `deleted: true`, so the
  list can show "Message deleted" instead of jumping to an older message.
- `GET /chats` is ordered by last activity: latest message time, or chat creation
  time when empty. This replaces ordering by creation time.
- `GET /users/search?q=` is a username prefix search (`LIKE` with `%`, `_` and `\`
  escaped). A blank `q` returns contacts, users who share a chat with the caller,
  which fills the "Contacts" section of the New chat dialog. The caller is never
  returned. There is no global user listing.
- Create responses (`POST /chats/direct`, `POST /chats/group`) do not include the
  new fields; the client refetches the list after creating a chat.
- The frontend marks a chat read up to its newest loaded message when the chat is
  opened, so unread badges clear without a realtime connection.

## Consequences

- The chat list stays one request regardless of the number of chats.
- Prefix search can enumerate usernames to any signed-in user. Acceptable for a
  messenger where usernames are public handles; add rate limiting before a public
  launch.
- The list does not update live yet: it refetches on window focus and after the
  user's own actions. Realtime updates come with the WebSocket client.
