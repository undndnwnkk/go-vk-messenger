package restapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

type contextKey string

const userIDKey contextKey = "userID"

const (
	// WebSocketSubprotocol is the application subprotocol of /api/v1/ws. The
	// server echoes it on a successful handshake.
	WebSocketSubprotocol = "messenger.v1"
	// bearerSubprotocolPrefix marks the subprotocol entry that carries the JWT
	// for browser clients: new WebSocket(url, ["messenger.v1", "bearer." + jwt]).
	// The server never echoes it back.
	bearerSubprotocolPrefix = "bearer."
)

// authError is a 401/400 response produced while extracting a token.
type authError struct {
	status        int
	code, message string
}

// JWTMiddleware authenticates REST requests with "Authorization: Bearer <JWT>".
func JWTMiddleware(jwtService *service.JWTService) func(http.Handler) http.Handler {
	return jwtMiddleware(jwtService, false)
}

// WebSocketJWTMiddleware authenticates the WebSocket handshake. It accepts the
// Authorization header (non-browser clients) or, when that header is absent,
// a "bearer.<JWT>" entry in Sec-WebSocket-Protocol (browsers). A present but
// invalid header is rejected without falling back to the subprotocol.
func WebSocketJWTMiddleware(jwtService *service.JWTService) func(http.Handler) http.Handler {
	return jwtMiddleware(jwtService, true)
}

func jwtMiddleware(jwtService *service.JWTService, allowSubprotocol bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string
			var authErr *authError
			if r.Header.Get("Authorization") == "" && allowSubprotocol {
				token, authErr = bearerFromSubprotocol(r)
			} else {
				token, authErr = bearerFromHeader(r)
			}
			if authErr != nil {
				WriteError(w, authErr.status, authErr.code, authErr.message)
				return
			}

			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
				return
			}

			if claims.UserID == "" {
				WriteError(w, http.StatusUnauthorized, "missing_user_id", "token missing user id")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerFromHeader(r *http.Request) (string, *authError) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", &authError{http.StatusUnauthorized, "missing_header", "you dont have token"}
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", &authError{http.StatusUnauthorized, "invalid_authorization", "you need Bearer <token>"}
	}
	return parts[1], nil
}

func bearerFromSubprotocol(r *http.Request) (string, *authError) {
	var tokens []string
	offersApp := false
	for _, header := range r.Header.Values("Sec-WebSocket-Protocol") {
		for _, protocol := range strings.Split(header, ",") {
			protocol = strings.TrimSpace(protocol)
			switch {
			case strings.HasPrefix(protocol, bearerSubprotocolPrefix):
				tokens = append(tokens, strings.TrimPrefix(protocol, bearerSubprotocolPrefix))
			case strings.EqualFold(protocol, WebSocketSubprotocol):
				offersApp = true
			}
		}
	}

	switch {
	case len(tokens) == 0:
		return "", &authError{http.StatusUnauthorized, "missing_header", "you dont have token"}
	case len(tokens) > 1 || tokens[0] == "":
		return "", &authError{http.StatusUnauthorized, "invalid_authorization", "send exactly one bearer.<token> subprotocol"}
	case !offersApp:
		// Without an agreed subprotocol a browser aborts the handshake with an
		// opaque error; fail early with a readable one instead.
		return "", &authError{http.StatusBadRequest, "invalid_request", "offer the " + WebSocketSubprotocol + " subprotocol together with bearer.<token>"}
	}
	return tokens[0], nil
}
