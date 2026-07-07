package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
	"github.com/google/uuid"
)

type SecretsHandler struct {
	Store *store.Store
}

func (h *SecretsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.save(w, r)
	case http.MethodDelete:
		h.delete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SecretsHandler) list(w http.ResponseWriter, r *http.Request) {
	endpointID := extractEndpointID(r.URL.Path)
	if endpointID == "" {
		http.Error(w, "missing endpoint id", http.StatusBadRequest)
		return
	}

	// Verify the endpoint belongs to this user
	user := middleware.GetUser(r)
	if !h.Store.EndpointBelongsToUser(endpointID, user.ID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Returns secrets WITHOUT the actual secret value
	secrets, err := h.Store.GetWebhookSecrets(endpointID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	if secrets == nil {
		secrets = []*models.WebhookSecret{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secrets)
}

func (h *SecretsHandler) save(w http.ResponseWriter, r *http.Request) {
	endpointID := extractEndpointID(r.URL.Path)
	if endpointID == "" {
		http.Error(w, "missing endpoint id", http.StatusBadRequest)
		return
	}

	user := middleware.GetUser(r)
	if !h.Store.EndpointBelongsToUser(endpointID, user.ID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var body struct {
		Provider string `json:"provider"`
		Secret   string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	validProviders := map[string]bool{
		"stripe": true, "paystack": true,
		"github": true, "generic": true,
	}
	if !validProviders[body.Provider] {
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	body.Secret = strings.TrimSpace(body.Secret)
	if body.Secret == "" {
		http.Error(w, "secret is required", http.StatusBadRequest)
		return
	}

	ws := &models.WebhookSecret{
		ID:         uuid.NewString(),
		EndpointID: endpointID,
		Provider:   body.Provider,
		Secret:     body.Secret,
		CreatedAt:  time.Now().UTC(),
	}

	if err := h.Store.SaveWebhookSecret(ws); err != nil {
		http.Error(w, "failed to save secret", http.StatusInternalServerError)
		return
	}

	// Return without the secret value
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          ws.ID,
		"endpoint_id": ws.EndpointID,
		"provider":    ws.Provider,
		"created_at":  ws.CreatedAt,
	})
}

func (h *SecretsHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	// Path: /endpoints/{endpointID}/secrets/{secretID}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	endpointID := parts[1]
	secretID := parts[3]

	if !h.Store.EndpointBelongsToUser(endpointID, user.ID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := h.Store.DeleteWebhookSecret(secretID, endpointID); err != nil {
		http.Error(w, "failed to delete secret", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func extractEndpointID(path string) string {
	// /endpoints/{id}/secrets
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
