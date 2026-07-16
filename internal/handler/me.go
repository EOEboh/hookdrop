package handler

import (
	"encoding/json"
	"net/http"

	"github.com/EOEboh/hookdrop/internal/billing"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/store"
)

// MeHandler returns the authenticated user's identity, plan, and limits.
// Works with both JWTs and API tokens — the CLI uses it for `hookdrop whoami`
// and to validate a token at login.
type MeHandler struct {
	Store *store.Store
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetUser(r)
	user, err := h.Store.GetUserByID(userCtx.ID)
	if err != nil || user == nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	sub, err := h.Store.GetSubscription(userCtx.ID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	// Inactive subscriptions fall back to free limits
	plan := sub.Plan
	if !billing.IsActive(sub.Status, sub.CurrentPeriodEnd) {
		plan = string(billing.PlanFree)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":        user,
		"plan":        plan,
		"auth_method": userCtx.AuthMethod,
		"limits":      billing.GetLimits(plan),
	})
}
