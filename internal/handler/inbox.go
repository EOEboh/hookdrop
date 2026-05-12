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

	// First check if it's a named endpoint slug
	ep, err := h.Store.GetEndpointBySlug(identifier)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	if ep != nil {
		// Named endpoint: use its ID as the session identifier
		sessionID = ep.ID
	} else {
		// Fall back to temporary session lookup
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

// extractIP handles X-Forwarded-For from proxies/load balancers
func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}
