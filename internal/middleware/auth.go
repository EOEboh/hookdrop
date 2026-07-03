package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/EOEboh/hookdrop/internal/auth"
)

type contextKey string

const UserContextKey contextKey = "user"

type UserContext struct {
	ID    string
	Email string
}

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""

			// Header first
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				tokenStr = strings.TrimPrefix(header, "Bearer ")
			}

			// Fall back to query param (SSE connections)
			if tokenStr == "" {
				tokenStr = r.URL.Query().Get("token")
			}

			if tokenStr == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
				ID:    claims.UserID,
				Email: claims.Email,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser pulls the authenticated user out of the request context
func GetUser(r *http.Request) *UserContext {
	user, _ := r.Context().Value(UserContextKey).(*UserContext)
	return user
}
