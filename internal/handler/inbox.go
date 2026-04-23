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
	Broadcast func(sessionID string, req *models.CapturedRequest) // SSE hook (wired up later)
}

func (h *InboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Extract session ID from URL: /i/{sessionID}
	sessionID := strings.TrimPrefix(r.URL.Path, "/i/")
	sessionID = strings.Trim(sessionID, "/")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	// 2. Verify session exists
	if !h.Store.SessionExists(sessionID) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// 3. Read body — cap at 1MB to prevent abuse
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// 4. Flatten headers (deduplicate multi-value into comma-joined)
	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[http.CanonicalHeaderKey(key)] = strings.Join(values, ", ")
	}

	// 5. Build the captured request
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

	// 6. Persist it
	if err := h.Store.SaveRequest(captured); err != nil {
		log.Printf("failed to save request: %v", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	// 7. Broadcast to any connected React clients (no-op until SSE is wired)
	if h.Broadcast != nil {
		go h.Broadcast(sessionID, captured)
	}

	// 8. Acknowledge the webhook sender — always 200, always fast
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received", "id": captured.ID})
}

// extractIP handles X-Forwarded-For from proxies/load balancers
func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}
