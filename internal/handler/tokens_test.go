package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/store"
)

func newTokenTestHandler(t *testing.T, perHour int) (*TokensHandler, string) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user, err := st.GetOrCreateUser("mint@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return &TokensHandler{Store: st, MintLimiter: middleware.NewEmailRateLimiter(perHour)}, user.ID
}

// jwtContext simulates the auth middleware placing a JWT-authenticated user
// on the request (token management is JWT-only).
func jwtRequest(method, target, body, userID string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, &middleware.UserContext{
		ID:         userID,
		Email:      "mint@example.com",
		AuthMethod: middleware.AuthMethodJWT,
	})
	return req.WithContext(ctx)
}

func TestTokenMintRateLimited(t *testing.T) {
	h, userID := newTokenTestHandler(t, 10)

	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, jwtRequest(http.MethodPost, "/tokens", `{"name":"cli"}`, userID))
		if rec.Code != http.StatusCreated {
			t.Fatalf("mint %d: got %d, want 201", i+1, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, jwtRequest(http.MethodPost, "/tokens", `{"name":"cli"}`, userID))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("11th mint: got %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429")
	}
}

func TestApiTokenCannotManageTokens(t *testing.T) {
	h, userID := newTokenTestHandler(t, 10)

	req := httptest.NewRequest(http.MethodPost, "/tokens", strings.NewReader(`{"name":"x"}`))
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, &middleware.UserContext{
		ID:         userID,
		AuthMethod: middleware.AuthMethodAPIToken,
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("api-token mint: got %d, want 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "api_token_cannot_manage_tokens") {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}
