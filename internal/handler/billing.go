package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	// MaxBytesReader errors on an oversized body rather than silently
	// truncating it — a truncated payload would fail signature verification
	// and be reported as a forgery.
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		log.Printf("lemonsqueezy webhook: read error: %v", err)
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
			// Return non-2xx so Lemonsqueezy retries and the failure is visible
			// in the dashboard's webhook log, not just ours.
			log.Printf("process lemonsqueezy event error: %v", err)
			http.Error(w, "processing failed", http.StatusInternalServerError)
			return
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
	userID := event.UserID

	// custom_data.user_id should always be present on subscription events, but
	// if it ever isn't, recover the user from the subscription ID we already
	// stored rather than dropping the event on the floor — a dropped
	// cancellation leaves a customer on Pro forever.
	if userID == "" {
		existing, err := h.Store.GetSubscriptionByProviderSubID(event.SubscriptionID)
		if err != nil {
			return fmt.Errorf("lookup subscription %s: %w", event.SubscriptionID, err)
		}
		if existing == nil {
			return fmt.Errorf(
				"%s webhook %s: custom_data.user_id missing and no subscription row for provider_sub_id=%s",
				providerName, event.Type, event.SubscriptionID)
		}
		userID = existing.UserID
		log.Printf(
			"WARNING: %s webhook %s had no custom_data.user_id — resolved user=%s via provider_sub_id=%s",
			providerName, event.Type, userID, event.SubscriptionID)
	}

	var periodEnd *time.Time
	if event.PeriodEnd > 0 {
		t := time.Unix(event.PeriodEnd, 0)
		periodEnd = &t
	}

	var trialEnd *time.Time
	if event.TrialEnd > 0 {
		t := time.Unix(event.TrialEnd, 0)
		trialEnd = &t
	} else {
		// The upsert overwrites trial_end unconditionally, so an event that
		// carries no trial date would wipe one we already know about (Paystack
		// records its trial via VerifyPaystack, not via the webhook). Carry the
		// stored value forward instead of nulling it.
		existing, err := h.Store.GetSubscription(userID)
		if err != nil {
			return fmt.Errorf("read existing subscription for %s: %w", userID, err)
		}
		trialEnd = existing.TrialEnd
	}

	plan := event.Plan
	if event.Type == "subscription.canceled" {
		plan = "free"
	}

	sub := &models.Subscription{
		UserID:             userID,
		Plan:               plan,
		Provider:           providerName,
		ProviderCustomerID: event.CustomerID,
		ProviderSubID:      event.SubscriptionID,
		Status:             event.Status,
		CurrentPeriodEnd:   periodEnd,
		TrialEnd:           trialEnd,
		Currency:           event.Currency,
		Interval:           event.Interval,
		CancelAtPeriodEnd:  event.CancelAtEnd,
		CreatedAt:          time.Now().UTC(),
	}

	log.Printf("processWebhookEvent: user=%s plan=%s provider=%s status=%s interval=%s trial_end=%v cancel_at_end=%t",
		sub.UserID, sub.Plan, sub.Provider, sub.Status, sub.Interval, sub.TrialEnd, sub.CancelAtPeriodEnd)

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

	// ── 1. The transaction must actually verify ──────────────────────────
	// Exhausting the retries is a hard failure. Granting Pro on an
	// unverifiable reference let any authenticated user upgrade themselves.
	tx, err := ps.VerifyTransaction(r.Context(), body.Reference)
	if err != nil {
		log.Printf("VerifyPaystack: verification failed for user=%s ref=%s: %v",
			user.ID, body.Reference, err)
		http.Error(w, "could not verify payment", http.StatusBadGateway)
		return
	}

	// ── 2. Currency ──────────────────────────────────────────────────────
	if !strings.EqualFold(tx.Currency, "NGN") {
		log.Printf("VerifyPaystack: rejecting non-NGN transaction user=%s ref=%s currency=%s",
			user.ID, body.Reference, tx.Currency)
		http.Error(w, "unsupported currency", http.StatusBadRequest)
		return
	}

	// ── 3+4. The plan must be one of ours, and it decides the interval ───
	// The client-supplied body.Interval is ignored: it drives
	// current_period_end, so trusting it let a monthly subscriber claim a
	// year of access.
	interval, known := ps.IntervalForPlanCode(tx.Plan.Code)
	if !known {
		log.Printf("VerifyPaystack: rejecting unknown plan user=%s ref=%s plan=%q",
			user.ID, body.Reference, tx.Plan.Code)
		http.Error(w, "transaction is not for a hookdrop plan", http.StatusBadRequest)
		return
	}
	if body.Interval != "" && body.Interval != interval {
		log.Printf("VerifyPaystack: client claimed interval=%q, plan %s says %q — using %q",
			body.Interval, tx.Plan.Code, interval, interval)
	}

	// ── 5. Amount must match the plan (or be a zero-charge trial) ────────
	isTrial := tx.Amount == 0
	switch {
	case isTrial:
		// Paystack charges ₦0 for the first transaction on a plan with a
		// trial. Explicitly allowed.
	case !tx.Plan.AmountKnown:
		// Paystack returned a bare plan code with no plan object. The plan
		// code check above already carries the weight here — attaching our
		// plan code makes Paystack charge that plan's price — so accept, but
		// say so.
		log.Printf("WARNING: VerifyPaystack: plan %s amount unknown, cannot cross-check charge of %d (user=%s ref=%s)",
			tx.Plan.Code, tx.Amount, user.ID, body.Reference)
	case tx.Amount != tx.Plan.Amount:
		log.Printf("VerifyPaystack: rejecting amount mismatch user=%s ref=%s charged=%d plan=%s expects=%d",
			user.ID, body.Reference, tx.Amount, tx.Plan.Code, tx.Plan.Amount)
		http.Error(w, "payment amount does not match the plan", http.StatusBadRequest)
		return
	}

	// ── 6. The transaction must belong to the caller ─────────────────────
	// Without this, any authenticated user could replay someone else's
	// valid reference.
	if !strings.EqualFold(
		strings.TrimSpace(tx.Email),
		strings.TrimSpace(user.Email),
	) {
		log.Printf("VerifyPaystack: REJECTED email mismatch (probable abuse) user=%s ref=%s",
			user.ID, body.Reference)
		http.Error(w, "this payment belongs to a different account", http.StatusForbidden)
		return
	}

	// ── 7. A reference may not be redeemed by a second account ───────────
	existing, err := h.Store.GetSubscriptionByProviderSubID(body.Reference)
	if err != nil {
		log.Printf("VerifyPaystack: replay lookup failed: %v", err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	if existing != nil && existing.UserID != user.ID {
		log.Printf("VerifyPaystack: REJECTED replayed reference (probable abuse) ref=%s owner=%s claimant=%s",
			body.Reference, existing.UserID, user.ID)
		http.Error(w, "this payment has already been redeemed", http.StatusConflict)
		return
	}
	// Same user re-verifying is idempotent: a double submit or a retried
	// handlePaystackSuccess must not lock a paying customer out.

	log.Printf("VerifyPaystack: verified user=%s ref=%s plan=%s interval=%s amount=%d is_trial=%v",
		user.ID, body.Reference, tx.Plan.Code, interval, tx.Amount, isTrial)

	now := time.Now().UTC()

	// Set subscription status and trial_end based on whether this is a trial
	subStatus := "active"
	var trialEnd *time.Time
	var periodEnd *time.Time

	cycle := 30 * 24 * time.Hour
	if interval == "year" {
		cycle = 365 * 24 * time.Hour
	}

	if isTrial {
		subStatus = "trialing"
		// Trial is 14 days: set trial_end
		te := now.Add(14 * 24 * time.Hour)
		trialEnd = &te
		// Period end is after trial ends + one billing cycle
		pe := te.Add(cycle)
		periodEnd = &pe
		log.Printf("VerifyPaystack: detected trial — trial_end=%s", te.Format(time.RFC3339))
	} else {
		// Paid charge: set period end from now
		pe := now.Add(cycle)
		periodEnd = &pe
		log.Printf("VerifyPaystack: detected paid charge — amount=%d", tx.Amount)
	}

	customerCode := tx.CustomerCode
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
		Interval:           interval,
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUser(r)
	log.Printf("CancelSubscription: user=%s", user.ID)

	sub, err := h.Store.GetSubscription(user.ID)
	if err != nil {
		log.Printf("CancelSubscription: store error: %v", err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	if sub.Plan == "free" || sub.ID == "" {
		http.Error(w, "no active subscription", http.StatusBadRequest)
		return
	}

	if sub.CancelAtPeriodEnd {
		// Already scheduled for cancellation — return success idempotently
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"cancelled":            true,
			"cancel_at_period_end": true,
			"access_until":         sub.CurrentPeriodEnd,
		})
		return
	}

	// For Paystack: ProviderSubID may be a transaction reference (T_xxx)
	// rather than a subscription code (SUB_xxx).
	// Attempt to find the real subscription code via Paystack API.
	// If it fails, we still honour the cancellation intent in our DB.
	if sub.Provider == "paystack" {
		ps, ok := h.Paystack.(*billing.PaystackProvider)
		if ok {
			subCode := h.resolvePaystackSubscriptionCode(
				r.Context(), ps, sub.ProviderCustomerID,
			)
			if subCode != "" {
				log.Printf("CancelSubscription: resolved subscription code %s for customer %s",
					subCode, sub.ProviderCustomerID)
				// Attempt Paystack API cancellation — non-fatal if it fails
				if err := h.Paystack.CancelSubscription(r.Context(), subCode); err != nil {
					log.Printf("CancelSubscription: Paystack API error (non-fatal): %v", err)
				}
			} else {
				log.Printf("CancelSubscription: could not resolve subscription code — marking cancelled in DB only")
			}
		}
	} else {
		// LemonSqueezy — use stored sub ID directly.
		//
		// This one is NOT best-effort: if the DELETE fails, Lemonsqueezy keeps
		// billing the customer. Marking them cancelled locally would show them
		// "cancelled" in the UI while their card is still being charged, so
		// fail loudly and leave the row untouched for a retry.
		if err := h.LemonSqueezy.CancelSubscription(r.Context(), sub.ProviderSubID); err != nil {
			log.Printf("CancelSubscription: LemonSqueezy cancel failed for user=%s sub=%s: %v",
				user.ID, sub.ProviderSubID, err)
			http.Error(w, "could not cancel subscription with the payment provider", http.StatusBadGateway)
			return
		}
	}

	// Mark cancel_at_period_end. For Paystack this is intent-only (the API call
	// above is best-effort); for LemonSqueezy the provider has already confirmed.
	sub.CancelAtPeriodEnd = true
	sub.UpdatedAt = time.Now().UTC()

	if err := h.Store.UpsertSubscription(sub); err != nil {
		log.Printf("CancelSubscription: upsert error: %v", err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	log.Printf("CancelSubscription: user=%s scheduled cancel at period end=%v",
		user.ID, sub.CurrentPeriodEnd)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cancelled":            true,
		"cancel_at_period_end": true,
		"access_until":         sub.CurrentPeriodEnd,
	})
}

// resolvePaystackSubscriptionCode looks up the active subscription code
// for a customer because ProviderSubID may be a transaction reference.
func (h *BillingHandler) resolvePaystackSubscriptionCode(
	ctx context.Context,
	ps *billing.PaystackProvider,
	customerCode string,
) string {
	if customerCode == "" || strings.HasPrefix(customerCode, "paystack_") {
		return "" // fallback customer code — can't look up
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.paystack.co/subscription?customer="+customerCode+"&status=active",
		nil,
	)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+ps.SecretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Status bool `json:"status"`
		Data   []struct {
			SubscriptionCode string `json:"subscription_code"`
			Status           string `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	for _, s := range result.Data {
		if s.Status == "active" && s.SubscriptionCode != "" {
			return s.SubscriptionCode
		}
	}
	return ""
}
