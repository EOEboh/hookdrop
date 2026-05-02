package handler

import (
	"encoding/json"
	"net/http"

	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/replay"
	"github.com/EOEboh/hookdrop/internal/store"
)

type ReplayHandler struct {
	Store  *store.Store
	Engine *replay.Engine
}

func (h *ReplayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Decode the replay instruction from the frontend
	var req models.ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RequestID == "" || req.TargetURL == "" {
		http.Error(w, "request_id and target_url are required", http.StatusBadRequest)
		return
	}

	// 2. Fetch the original captured request from the store
	original, err := h.Store.GetRequest(req.RequestID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	if original == nil {
		http.Error(w, "request not found", http.StatusNotFound)
		return
	}

	// 3. Fire the replay
	result, err := h.Engine.Replay(r.Context(), original, &req)
	if err != nil {
		// Replay error means the target was unreachable or timed out
		// Return a structured error so the frontend can show it cleanly
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// 4. Return the result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
