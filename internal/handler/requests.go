package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
)

type RequestsHandler struct {
	Store *store.Store
}

func (h *RequestsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identifier := strings.TrimPrefix(r.URL.Path, "/requests/")
	identifier = strings.Trim(identifier, "/")

	if identifier == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	// Resolve slug/endpoint/session to the canonical ID requests are stored
	// under, and enforce ownership — foreign resources return the same 404 as
	// missing ones.
	user := middleware.GetUser(r)
	sessionID, ok := h.Store.ResolveIdentifierForUser(identifier, user.ID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Parse filters from query string
	filter := models.FilterFromQuery(r.URL.Query())

	requests, err := h.Store.GetRequests(sessionID, filter)
	if err != nil {
		http.Error(w, "failed to fetch requests", http.StatusInternalServerError)
		return
	}
	if requests == nil {
		requests = []*models.CapturedRequest{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}
