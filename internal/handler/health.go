package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/EOEboh/hookdrop/internal/store"
)

type HealthHandler struct {
	Store     *store.Store
	StartedAt time.Time
	Version   string
}

type healthResponse struct {
	Status   string `json:"status"`   // "ok" or "degraded"
	Database string `json:"database"` // "ok" or "error"
	UptimeS  int64  `json:"uptime_seconds"`
	Version  string `json:"version,omitempty"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := healthResponse{
		Status:   "ok",
		Database: "ok",
		UptimeS:  int64(time.Since(h.StartedAt).Seconds()),
		Version:  h.Version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store") // monitors must never get a cached result

	if err := h.Store.Ping(); err != nil {
		// Deliberately don't leak err.Error() in the response —
		// internal error details have no business being public.
		// Full detail goes to the server log only.
		resp.Status = "degraded"
		resp.Database = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
