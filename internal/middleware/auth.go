package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/auth"
	"github.com/EOEboh/hookdrop/internal/store"
)

type contextKey string

const UserContextKey contextKey = "user"

// Auth methods — token-management routes are JWT-only so a leaked API token
// cannot mint or revoke tokens.
const (
	AuthMethodJWT      = "jwt"
	AuthMethodAPIToken = "api_token"
)

type UserContext struct {
	ID         string
	Email      string
	AuthMethod string // AuthMethodJWT or AuthMethodAPIToken
}

func Auth(jwtSecret string, st *store.Store) func(http.Handler) http.Handler {
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

			// API tokens (hkdp_...) are looked up by hash; anything else is a JWT
			if strings.HasPrefix(tokenStr, auth.APITokenPrefix) {
				apiToken, err := st.GetAPITokenByHash(auth.HashAPIToken(tokenStr))
				if err != nil || apiToken == nil {
					http.Error(w, "invalid token", http.StatusUnauthorized)
					return
				}
				user, err := st.GetUserByID(apiToken.UserID)
				if err != nil || user == nil {
					http.Error(w, "invalid token", http.StatusUnauthorized)
					return
				}
				go st.TouchAPIToken(apiToken.ID, time.Now().UTC())

				ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
					ID:         user.ID,
					Email:      user.Email,
					AuthMethod: AuthMethodAPIToken,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
				ID:         claims.UserID,
				Email:      claims.Email,
				AuthMethod: AuthMethodJWT,
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
