package restapi

import (
	"context"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
	"net/http"
	"strings"
)

type contextKey string

const userIDKey contextKey = "userID"

func JWTMiddleware(jwtService *service.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, http.StatusUnauthorized, "missing_header", "you dont have token")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				WriteError(w, http.StatusUnauthorized, "invalid_authorization", "you need Bearer <token>")
				return
			}
			tokenString := parts[1]

			claims, err := jwtService.ValidateToken(tokenString)
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
