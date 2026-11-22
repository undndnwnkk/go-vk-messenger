# 0005. Frontend access token storage and 401 handling

Status: accepted (2026-11-18)

## Context

The backend issues a short-lived JWT (`JWT_TTL`, default 15m) and has no refresh
endpoint. The REST API reads it from `Authorization: Bearer`, the WebSocket from a
`bearer.<JWT>` subprotocol (ADR 0002), so cookies are not an option without
backend changes.

## Decision

- `AuthProvider` keeps `access_token` in `localStorage` when "Keep me signed in" is
  checked and in `sessionStorage` otherwise. Storage errors (private mode, blocked
  site data) are ignored; the user simply stays signed out after a reload.
- An `openapi-fetch` middleware adds the header to every request. If a request
  that carried a token gets 401, the provider clears the token and the query
  cache, and `RequireAuth` redirects to `/login`, remembering the page to return to.
  A 401 on a request without a token (wrong password on `/auth/login`) is shown as
  a form error and does not sign anyone out.

## Consequences

- A token in Web Storage is readable by any script on the page, so XSS means token
  theft. Keep third-party scripts off the app and add a CSP before production.
- Users are signed out when the token expires. Silent renewal needs a refresh
  endpoint on the backend.
