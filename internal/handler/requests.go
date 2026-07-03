package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
)

type RequestsHandler struct {
	Store *store.Store
}

func (h *RequestsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/requests/")
	sessionID = strings.Trim(sessionID, "/")

	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	if !h.Store.IdentifierExists(sessionID) {
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
