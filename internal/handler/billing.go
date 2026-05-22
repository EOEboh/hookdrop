package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/EOEboh/hookdrop/internal/billing"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
)

type BillingHandler struct {
	Store        *store.Store
	LemonSqueezy billing.Provider
	Paystack     billing.Provider
	AppURL       string
}

// getProvider routes to the correct billing provider based on currency
func (h *BillingHandler) getProvider(currency string) billing.Provider {
	if currency == "ngn" {
		return h.Paystack
	}
	return h.LemonSqueezy
}

// GET /billing/subscription
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	sub, err := h.Store.GetSubscription(user.ID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	limits := billing.GetLimits(sub.Plan)
	isActive := billing.IsActive(sub.Status, sub.CurrentPeriodEnd)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscription": sub,
		"limits":       limits,
		"is_active":    isActive,
	})
}

// POST /billing/checkout
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var body struct {
		Interval string `json:"interval"` // "month" or "year"
		Currency string `json:"currency"` // "ngn" or "usd"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Interval == "" {
		body.Interval = "month"
	}
	if body.Currency == "" {
		body.Currency = "usd"
	}

	provider := h.getProvider(body.Currency)

	result, err := provider.CreateCheckout(r.Context(), billing.CheckoutParams{
		UserID:     user.ID,
		Email:      user.Email,
		Plan:       "pro",
		Interval:   body.Interval,
		Currency:   body.Currency,
		SuccessURL: h.AppURL + "/settings/billing?success=true",
		CancelURL:  h.AppURL + "/settings/billing?canceled=true",
	})
	if err != nil {
		log.Printf("checkout error: %v", err)
		http.Error(w, "checkout failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /billing/portal
func (h *BillingHandler) GetPortal(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	sub, err := h.Store.GetSubscription(user.ID)
	if err != nil || sub.Provider == "" {
		http.Error(w, "no active subscription", http.StatusBadRequest)
		return
	}

	provider := h.getProvider(sub.Currency)
	url, err := provider.GetPortalURL(r.Context(), sub.ProviderCustomerID, h.AppURL)
	if err != nil {
		http.Error(w, "portal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// POST /billing/webhook/lemonsqueezy
func (h *BillingHandler) LemonSqueezyWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	// Lemonsqueezy sends the signature as X-Signature
	sig := r.Header.Get("X-Signature")
	event, err := h.LemonSqueezy.HandleWebhook(payload, sig)
	if err != nil {
		log.Printf("lemonsqueezy webhook error: %v", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if event != nil {
		if err := h.processWebhookEvent(event); err != nil {
			log.Printf("process lemonsqueezy event error: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"received": "true"})
}

// POST /billing/webhook/paystack
func (h *BillingHandler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Paystack-Signature")
	event, err := h.Paystack.HandleWebhook(payload, sig)
	if err != nil {
		log.Printf("paystack webhook error: %v", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if event != nil {
		if err := h.processWebhookEvent(event); err != nil {
			log.Printf("process paystack event error: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"received": "true"})
}

func (h *BillingHandler) processWebhookEvent(event *billing.WebhookEvent) error {
	if event.UserID == "" {
		log.Printf("webhook event missing user_id — skipping")
		return nil
	}

	var periodEnd *time.Time
	if event.PeriodEnd > 0 {
		t := time.Unix(event.PeriodEnd, 0)
		periodEnd = &t
	}

	sub := &models.Subscription{
		UserID:             event.UserID,
		Plan:               event.Plan,
		Provider:           event.Type,
		ProviderCustomerID: event.CustomerID,
		ProviderSubID:      event.SubscriptionID,
		Status:             event.Status,
		CurrentPeriodEnd:   periodEnd,
		Currency:           event.Currency,
		Interval:           event.Interval,
		CancelAtPeriodEnd:  event.CancelAtEnd,
		CreatedAt:          time.Now().UTC(),
	}

	if event.Type == "subscription.canceled" {
		sub.Plan = "free"
	}

	return h.Store.UpsertSubscription(sub)
}
