# 0001. CORS: exact-origin allowlist, no credentials

Status: accepted (2026-11-18)

## Context

The browser frontend runs on a different origin than the API (Vite dev server on
`http://localhost:5173`, API on `:8080`), so the API must answer CORS requests.

## Decision

- Allowed origins come from `CORS_ALLOWED_ORIGINS`, a comma-separated list of exact
  `scheme://host[:port]` values. Wildcards (`*`, `*.example.com`), paths and
  non-http(s) schemes fail startup, so a typo can't silently open the API.
- `Access-Control-Allow-Credentials` is not sent. The API authenticates with the
  `Authorization: Bearer` header, which needs no credentials mode; staying without it
  keeps the browser from attaching cookies to cross-origin calls.
- The middleware wraps the whole router, so preflights get `204` before chi routing
  (chi would otherwise answer `OPTIONS` with `405`). A preflight from an unlisted origin
  also gets `204`, but without CORS headers, so the browser blocks the request.
- Empty value (the default) disables CORS, so existing deployments behave as before.
- Hand-written (about 50 lines) instead of a dependency such as `rs/cors`: the policy is
  small and fixed, and tests cover it.

## Consequences

- Every frontend origin (dev, staging, prod) must be listed explicitly per environment.
- CORS does not cover WebSocket handshakes. `/api/v1/ws` needs its own origin check
  and a browser-compatible way to pass the token, see ADR 0002.
