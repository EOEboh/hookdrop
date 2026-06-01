package handler

import (
	"encoding/json"
	"fmt"
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
		log.Printf("GetSubscription error for user %s: %v", user.ID, err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	limits := billing.GetLimits(sub.Plan)
	isActive := billing.IsActive(sub.Status, sub.CurrentPeriodEnd)

	log.Printf("GetSubscription: user=%s plan=%s status=%s is_active=%v",
		user.ID, sub.Plan, sub.Status, isActive)

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
		Interval string `json:"interval"`
		Currency string `json:"currency"`
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
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	// Guard: no active paid subscription
	if sub.Plan == "free" || sub.ID == "" {
		http.Error(w, "no active subscription", http.StatusBadRequest)
		return
	}

	// Route by provider name stored on the subscription
	// Falls back to currency-based routing if provider name is missing
	var providerName string
	switch sub.Provider {
	case "paystack":
		providerName = "paystack"
	case "lemonsqueezy":
		providerName = "lemonsqueezy"
	default:
		// Legacy rows may have event type stored instead of provider name
		// Fall back to currency
		if sub.Currency == "ngn" {
			providerName = "paystack"
		} else {
			providerName = "lemonsqueezy"
		}
	}

	var provider billing.Provider
	if providerName == "paystack" {
		provider = h.Paystack
	} else {
		provider = h.LemonSqueezy
	}

	url, err := provider.GetPortalURL(r.Context(), sub.ProviderCustomerID, h.AppURL)
	if err != nil {
		log.Printf("GetPortal error: %v", err)
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

	sig := r.Header.Get("X-Signature")
	event, err := h.LemonSqueezy.HandleWebhook(payload, sig)
	if err != nil {
		log.Printf("lemonsqueezy webhook error: %v", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if event != nil {
		if err := h.processWebhookEvent(event, "lemonsqueezy"); err != nil {
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
		if err := h.processWebhookEvent(event, "paystack"); err != nil {
			log.Printf("process paystack event error: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"received": "true"})
}

// processWebhookEvent now takes an explicit providerName so Provider is
// stored correctly ("paystack" or "lemonsqueezy") instead of the event type
// string ("subscription.created" etc.) which broke GetPortal routing.
func (h *BillingHandler) processWebhookEvent(
	event *billing.WebhookEvent,
	providerName string,
) error {
	if event.UserID == "" {
		log.Printf("webhook event missing user_id — skipping")
		return nil
	}

	var periodEnd *time.Time
	if event.PeriodEnd > 0 {
		t := time.Unix(event.PeriodEnd, 0)
		periodEnd = &t
	}

	plan := event.Plan
	if event.Type == "subscription.canceled" {
		plan = "free"
	}

	sub := &models.Subscription{
		UserID:             event.UserID,
		Plan:               plan,
		Provider:           providerName,
		ProviderCustomerID: event.CustomerID,
		ProviderSubID:      event.SubscriptionID,
		Status:             event.Status,
		CurrentPeriodEnd:   periodEnd,
		Currency:           event.Currency,
		Interval:           event.Interval,
		CancelAtPeriodEnd:  event.CancelAtEnd,
		CreatedAt:          time.Now().UTC(),
	}

	log.Printf("processWebhookEvent: user=%s plan=%s provider=%s status=%s",
		sub.UserID, sub.Plan, sub.Provider, sub.Status)

	return h.Store.UpsertSubscription(sub)
}

// POST /billing/verify-paystack
func (h *BillingHandler) VerifyPaystack(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	log.Printf("VerifyPaystack called: user_id=%s", user.ID)

	var body struct {
		Reference string `json:"reference"`
		Plan      string `json:"plan"`
		Interval  string `json:"interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("VerifyPaystack decode error: %v", err)
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	log.Printf("VerifyPaystack body: reference=%s plan=%s interval=%s",
		body.Reference, body.Plan, body.Interval)

	if body.Reference == "" {
		http.Error(w, "reference required", http.StatusBadRequest)
		return
	}

	ps, ok := h.Paystack.(*billing.PaystackProvider)
	if !ok {
		log.Printf("VerifyPaystack: provider type assertion failed")
		http.Error(w, "provider error", http.StatusInternalServerError)
		return
	}

	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status   string `json:"status"`
			Amount   int    `json:"amount"` // 0 = trial charge
			Customer struct {
				CustomerCode string `json:"customer_code"`
				Email        string `json:"email"`
			} `json:"customer"`
			Plan json.RawMessage `json:"plan"` // string or object
		} `json:"data"`
	}

	var lastErr error
	delays := []time.Duration{0, 1 * time.Second, 2 * time.Second}

	for i, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}

		req, err := http.NewRequestWithContext(
			r.Context(), "GET",
			"https://api.paystack.co/transaction/verify/"+body.Reference,
			nil,
		)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Authorization", "Bearer "+ps.SecretKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("VerifyPaystack: attempt %d HTTP error: %v", i+1, err)
			lastErr = err
			continue
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if decodeErr != nil {
			log.Printf("VerifyPaystack: attempt %d decode error: %v", i+1, decodeErr)
			lastErr = decodeErr
			continue
		}

		log.Printf("VerifyPaystack: attempt %d — status=%v data.status=%s amount=%d customer=%s",
			i+1, result.Status, result.Data.Status,
			result.Data.Amount, result.Data.Customer.CustomerCode)

		if result.Status && result.Data.Status == "success" {
			lastErr = nil
			break
		}

		lastErr = fmt.Errorf("paystack: status=%v data.status=%s message=%s",
			result.Status, result.Data.Status, result.Message)
	}

	if lastErr != nil {
		log.Printf("VerifyPaystack: retries exhausted: %v — proceeding with upsert", lastErr)
	}

	now := time.Now().UTC()

	// Detect trial: Paystack charges ₦0 for the first transaction on a plan with a trial
	isTrial := result.Data.Amount == 0

	// Set subscription status and trial_end based on whether this is a trial
	subStatus := "active"
	var trialEnd *time.Time
	var periodEnd *time.Time

	if isTrial {
		subStatus = "trialing"
		// Trial is 14 days: set trial_end
		te := now.Add(14 * 24 * time.Hour)
		trialEnd = &te
		// Period end is after trial ends + one billing cycle
		var pe time.Time
		if body.Interval == "year" {
			pe = te.Add(365 * 24 * time.Hour)
		} else {
			pe = te.Add(30 * 24 * time.Hour)
		}
		periodEnd = &pe
		log.Printf("VerifyPaystack: detected trial — trial_end=%s", te.Format(time.RFC3339))
	} else {
		// Paid charge: set period end from now
		var pe time.Time
		if body.Interval == "year" {
			pe = now.Add(365 * 24 * time.Hour)
		} else {
			pe = now.Add(30 * 24 * time.Hour)
		}
		periodEnd = &pe
		log.Printf("VerifyPaystack: detected paid charge — amount=%d", result.Data.Amount)
	}

	customerCode := result.Data.Customer.CustomerCode
	if customerCode == "" {
		customerCode = "paystack_" + body.Reference
		log.Printf("VerifyPaystack: using fallback customer code for ref=%s", body.Reference)
	}

	sub := &models.Subscription{
		UserID:             user.ID,
		Plan:               "pro",
		Provider:           "paystack",
		ProviderCustomerID: customerCode,
		ProviderSubID:      body.Reference,
		Status:             subStatus,
		CurrentPeriodEnd:   periodEnd,
		TrialEnd:           trialEnd,
		Currency:           "ngn",
		Interval:           body.Interval,
		CancelAtPeriodEnd:  false,
		CreatedAt:          now,
	}

	log.Printf("VerifyPaystack: upserting — user=%s plan=pro status=%s customer=%s",
		user.ID, subStatus, customerCode)

	if err := h.Store.UpsertSubscription(sub); err != nil {
		log.Printf("VerifyPaystack: upsert FAILED: %v", err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	log.Printf("VerifyPaystack: upsert SUCCESS — user=%s is now Pro (status=%s)",
		user.ID, subStatus)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plan":      "pro",
		"status":    subStatus,
		"is_trial":  isTrial,
		"trial_end": trialEnd,
	})
}

// POST /billing/cancel
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	sub, err := h.Store.GetSubscription(user.ID)
	if err != nil || sub.Plan == "free" || sub.ID == "" {
		http.Error(w, "no active subscription", http.StatusBadRequest)
		return
	}

	// Cancel at period end: user keeps access till then
	if err := h.getProvider(sub.Currency).CancelSubscription(
		r.Context(), sub.ProviderSubID,
	); err != nil {
		log.Printf("CancelSubscription error: %v", err)
		http.Error(w, "cancellation failed", http.StatusInternalServerError)
		return
	}

	// Mark cancel_at_period_end in the database
	sub.CancelAtPeriodEnd = true
	sub.UpdatedAt = time.Now().UTC()
	if err := h.Store.UpsertSubscription(sub); err != nil {
		log.Printf("CancelSubscription upsert error: %v", err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	log.Printf("CancelSubscription: user=%s scheduled cancel at period end", user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cancelled":            true,
		"cancel_at_period_end": true,
		"access_until":         sub.CurrentPeriodEnd,
	})
}
