# 0002. WebSocket auth: JWT in a `bearer.<JWT>` subprotocol

Status: accepted (2026-11-18)

## Context

`/api/v1/ws` authenticated only with `Authorization: Bearer <JWT>`. The browser
`WebSocket` API cannot set request headers, so the frontend could not connect.
Separately, `websocket.Accept` used its default origin check, which rejects every
cross-origin handshake, so a dev server on another port got 403 regardless of auth.

Options considered:

| Option | Pros | Cons |
|---|---|---|
| `?access_token=<JWT>` | simplest, stateless | the JWT lands in access logs, proxy logs, browser history |
| One-time ticket (`POST /ws/ticket`, then `?ticket=`) | JWT never leaves the header; leaked ticket is useless | new endpoint, shared state (Redis across replicas), extra round trip per reconnect |
| **`bearer.<JWT>` subprotocol** | stateless, works on any replica with or without Redis, stays out of URLs | unusual; server must echo a real subprotocol; some proxies may log the header |

## Decision

- Browsers connect with `new WebSocket(url, ["messenger.v1", "bearer." + jwt])`
  (the same approach the Kubernetes API server uses for browser clients).
- The server selects and echoes only `messenger.v1`, never the token entry.
  A `bearer.*` entry without `messenger.v1` gets 400: otherwise the browser would
  abort the handshake with an opaque error.
- Exactly one `bearer.*` entry is accepted. JWT characters (base64url and `.`)
  are valid in subprotocol tokens.
- The `Authorization` header keeps working for non-browser clients. If it is
  present it takes precedence, and an invalid header is not retried with the
  subprotocol, so the result never depends on which credential happened to be valid.
- The subprotocol is accepted only on `/api/v1/ws`. REST stays header-only.
- Cross-origin handshakes are allowed only from `CORS_ALLOWED_ORIGINS` (same list
  as CORS, ADR 0001); same-host handshakes are always allowed.

## Consequences

- Frontend: always offer both subprotocols; check `ws.protocol === "messenger.v1"`.
- Reconnecting with a fresh token needs no extra request.
- If a proxy in front of the API logs `Sec-WebSocket-Protocol`, exclude that
  header from its logs.
