package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
	"github.com/EOEboh/hookdrop/internal/verify"
	"github.com/google/uuid"
)

type InboxHandler struct {
	Store     *store.Store
	Broadcast func(sessionID string, req *models.CapturedRequest)
}

func (h *InboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identifier := strings.TrimPrefix(r.URL.Path, "/i/")
	identifier = strings.Trim(identifier, "/")

	if identifier == "" {
		http.Error(w, "missing endpoint identifier", http.StatusBadRequest)
		return
	}

	var sessionID string
	var endpointID string // only set for named endpoints

	ep, err := h.Store.GetEndpointBySlug(identifier)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	if ep != nil {
		sessionID = ep.ID
		endpointID = ep.ID
	} else {
		if !h.Store.SessionExists(identifier) {
			http.Error(w, "endpoint not found", http.StatusNotFound)
			return
		}
		sessionID = identifier
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[http.CanonicalHeaderKey(key)] = strings.Join(values, ", ")
	}

	captured := &models.CapturedRequest{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		Method:     r.Method,
		Headers:    headers,
		Body:       body,
		BodySize:   len(body),
		RemoteIP:   extractIP(r),
		ReceivedAt: time.Now().UTC(),
		Verified:   "unverified",
	}

	// Verify signature if this is a named endpoint with a secret configured
	if endpointID != "" {
		secrets, err := h.Store.GetWebhookSecretsWithValues(endpointID)
		if err != nil {
			log.Printf("failed to fetch secrets for endpoint %s: %v", endpointID, err)
		} else if len(secrets) > 0 {
			result := verify.Verify(captured, secrets)
			captured.Verified = result.Status
			captured.Provider = result.Provider
			log.Printf("verification: endpoint=%s status=%s provider=%s reason=%s",
				endpointID, result.Status, result.Provider, result.Reason)
		}
	}

	if err := h.Store.SaveRequest(captured); err != nil {
		log.Printf("failed to save request: %v", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	if h.Broadcast != nil {
		go h.Broadcast(sessionID, captured)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "received",
		"id":     captured.ID,
	})
}

func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}
