package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/auth"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
	"github.com/google/uuid"
)

// TokensHandler manages long-lived API tokens (used by the CLI).
// All routes are JWT-only: a leaked API token must not be able to mint
// replacements or revoke the user's other tokens.
type TokensHandler struct {
	Store *store.Store
}

func (h *TokensHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.AuthMethod != middleware.AuthMethodJWT {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "api_token_cannot_manage_tokens",
		})
		return
	}

	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/tokens"), "/")

	switch {
	case id == "" && r.Method == http.MethodGet:
		h.list(w, user.ID)
	case id == "" && r.Method == http.MethodPost:
		h.create(w, r, user.ID)
	case id == "" && r.Method == http.MethodDelete:
		h.revokeAll(w, user.ID)
	case id != "" && r.Method == http.MethodDelete:
		h.revoke(w, id, user.ID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TokensHandler) create(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Name          string `json:"name"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if body.ExpiresInDays < 0 || body.ExpiresInDays > 3650 {
		http.Error(w, "expires_in_days must be between 0 and 3650", http.StatusBadRequest)
		return
	}

	token, hash, prefix, err := auth.GenerateAPIToken()
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	var expiresAt *time.Time
	if body.ExpiresInDays > 0 {
		t := time.Now().UTC().AddDate(0, 0, body.ExpiresInDays)
		expiresAt = &t
	}

	apiToken := &models.APIToken{
		ID:          uuid.NewString(),
		UserID:      userID,
		Name:        body.Name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt,
	}
	if err := h.Store.CreateAPIToken(apiToken); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	// The only response that ever contains the full token
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           apiToken.ID,
		"name":         apiToken.Name,
		"token":        token,
		"token_prefix": apiToken.TokenPrefix,
		"created_at":   apiToken.CreatedAt,
		"expires_at":   apiToken.ExpiresAt,
	})
}

func (h *TokensHandler) list(w http.ResponseWriter, userID string) {
	tokens, err := h.Store.ListAPITokens(userID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	if tokens == nil {
		tokens = []*models.APIToken{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

func (h *TokensHandler) revoke(w http.ResponseWriter, id, userID string) {
	if err := h.Store.RevokeAPIToken(id, userID); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TokensHandler) revokeAll(w http.ResponseWriter, userID string) {
	if err := h.Store.RevokeAllAPITokens(userID); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
