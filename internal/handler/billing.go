package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	h.handleWebhook(w, r, webhookRoute{
		provider:        "lemonsqueezy",
		signatureHeader: "X-Signature",
		eventNameHeader: "X-Event-Name",
		handler:         h.LemonSqueezy,
	})
}

// POST /billing/webhook/paystack
func (h *BillingHandler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, webhookRoute{
		provider:        "paystack",
		signatureHeader: "X-Paystack-Signature",
		handler:         h.Paystack,
	})
}

type webhookRoute struct {
	provider        string
	signatureHeader string
	// eventNameHeader is the header carrying the provider's event name, where
	// one exists. Empty means fall back to parsing the payload.
	eventNameHeader string
	handler         billing.Provider
}

// handleWebhook is the shared inbound path for both providers: verify, record,
// deduplicate, reject stale deliveries, then process.
func (h *BillingHandler) handleWebhook(
	w http.ResponseWriter,
	r *http.Request,
	route webhookRoute,
) {
	// MaxBytesReader errors on an oversized body rather than silently
	// truncating it. A truncated payload would fail signature verification
	// and be reported as a forgery.
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		log.Printf("%s webhook: read error: %v", route.provider, err)
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	// Signature first, always. Nothing unverified is ever recorded.
	event, err := route.handler.HandleWebhook(payload, r.Header.Get(route.signatureHeader))
	if err != nil {
		log.Printf("%s webhook error: %v", route.provider, err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	eventType := r.Header.Get(route.eventNameHeader)
	if eventType == "" {
		eventType = providerEventName(payload)
	}

	// Neither provider sends a unique event ID, but retries are byte-identical
	// (the signature is computed over the raw body), so the payload hash is a
	// stable dedupe key for both.
	sum := sha256.Sum256(payload)
	eventKey := hex.EncodeToString(sum[:])

	record := &models.BillingEvent{
		Provider:  route.provider,
		EventType: eventType,
		Payload:   string(payload),
		EventKey:  eventKey,
	}
	if event != nil {
		record.UserID = event.UserID
		// Scope the ordering check by subscription where there is one, else by
		// customer, charge.success identifies only the customer.
		record.ObjectID = event.SubscriptionID
		if record.ObjectID == "" {
			record.ObjectID = event.CustomerID
		}
		if event.EventAt > 0 {
			t := time.Unix(event.EventAt, 0).UTC()
			record.EventAt = &t
		}
	}

	switch err := h.Store.RecordBillingEvent(record); {
	case errors.Is(err, store.ErrDuplicateBillingEvent):
		// The provider retried something we already have. 200 stops the
		// retries; re-processing would be wasted work at best.
		log.Printf("%s webhook: duplicate delivery %s (%s). Already recorded",
			route.provider, eventType, eventKey[:12])
		writeWebhookOK(w)
		return
	case err != nil:
		// Failing to record means we cannot guarantee idempotency, so do not
		// process. Non-2xx asks the provider to try again.
		log.Printf("%s webhook: could not record event: %v", route.provider, err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	if event == nil {
		// Recorded for the audit trail, but not an event we act on.
		writeWebhookOK(w)
		return
	}

	// Out-of-order guard: a delivery older than one already applied to this
	// subscription must not be replayed over it. Without this a retried
	// subscription_updated landing after subscription_expired resurrects Pro.
	if record.EventAt != nil {
		latest, found, err := h.Store.LatestProcessedEventAt(route.provider, record.ObjectID)
		if err != nil {
			log.Printf("%s webhook: ordering check failed: %v", route.provider, err)
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if found && record.EventAt.Before(latest) {
			log.Printf("%s webhook: SKIPPING stale %s for %s (event_at=%s, already applied %s)",
				route.provider, eventType, record.ObjectID,
				record.EventAt.Format(time.RFC3339), latest.Format(time.RFC3339))
			// Left processed=0 on purpose: processed means "applied to the
			// subscription", and this one deliberately was not.
			writeWebhookOK(w)
			return
		}
	} else {
		log.Printf("%s webhook: %s carries no timestamp. No ordering protection",
			route.provider, eventType)
	}

	if err := h.processWebhookEvent(event, route.provider); err != nil {
		// Non-2xx so the provider retries and the failure shows up in its
		// dashboard, not only in our logs.
		log.Printf("process %s event error: %v", route.provider, err)
		http.Error(w, "processing failed", http.StatusInternalServerError)
		return
	}

	if err := h.Store.MarkBillingEventProcessed(eventKey, event.UserID); err != nil {
		log.Printf("%s webhook: could not mark %s processed: %v",
			route.provider, eventKey[:12], err)
	}

	writeWebhookOK(w)
}

func writeWebhookOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"received": "true"})
}

// providerEventName pulls the event name out of a payload for the audit trail,
// covering both providers' envelopes without committing to either shape.
func providerEventName(payload []byte) string {
	var envelope struct {
		Event string `json:"event"` // Paystack
		Meta  struct {
			EventName string `json:"event_name"` // Lemon Squeezy
		} `json:"meta"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "unknown"
	}
	if envelope.Event != "" {
		return envelope.Event
	}
	if envelope.Meta.EventName != "" {
		return envelope.Meta.EventName
	}
	return "unknown"
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
	// stored rather than dropping the event on the floor. A dropped
	// cancellation leaves a customer on Pro forever.
	if userID == "" {
		existing, via, err := h.resolveWebhookUser(event)
		if err != nil {
			return err
		}
		if existing == nil {
			return fmt.Errorf(
				"%s webhook %s: no user_id in the payload and no subscription row for sub=%s customer=%s",
				providerName, event.Type, event.SubscriptionID, event.CustomerID)
		}
		userID = existing.UserID
		log.Printf("WARNING: %s webhook %s carried no user_id. Resolved user=%s via %s",
			providerName, event.Type, userID, via)
	}

	var periodEnd *time.Time
	if event.PeriodEnd > 0 {
		t := time.Unix(event.PeriodEnd, 0)
		periodEnd = &t
	} else if existing, err := h.Store.GetSubscription(userID); err == nil {
		// Not every event moves the period. A failed renewal reports no new
		// date. The upsert writes every column, so passing nil would blank
		// current_period_end, and a nil period grants access indefinitely.
		// That would turn the expiry gate off for exactly the subscriptions it
		// exists to catch.
		periodEnd = existing.CurrentPeriodEnd
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

	// Derive the plan from the status, not the event type. Lemon Squeezy fires
	// subscription_expired and subscription_updated together at end of life;
	// the updated one normalises to subscription.updated while carrying
	// status "expired", which used to write plan=pro alongside status=canceled.
	// IsActive also keeps past_due on Pro through its grace period.
	plan := event.Plan
	if !billing.IsEntitledStatus(event.Status) {
		plan = "free"
	}

	// charge.success names no subscription, so renewals carry an empty
	// SubscriptionID. The upsert writes every column, so passing it through
	// would blank the stored subscription code and break cancellation.
	//
	// auto_renews is decided by VerifyPaystack and the repair pass, which see
	// the authorization. A webhook does not, so it must never overwrite it.
	providerSubID := event.SubscriptionID
	autoRenews := true
	if existing, err := h.Store.GetSubscription(userID); err == nil {
		autoRenews = existing.AutoRenews
		if providerSubID == "" {
			providerSubID = existing.ProviderSubID
		}
	}

	sub := &models.Subscription{
		UserID:             userID,
		Plan:               plan,
		Provider:           providerName,
		ProviderCustomerID: event.CustomerID,
		ProviderSubID:      providerSubID,
		Status:             event.Status,
		CurrentPeriodEnd:   periodEnd,
		TrialEnd:           trialEnd,
		Currency:           event.Currency,
		Interval:           event.Interval,
		CancelAtPeriodEnd:  event.CancelAtEnd,
		AutoRenews:         autoRenews,
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
		log.Printf("VerifyPaystack: client claimed interval=%q, plan %s says %q. Using %q",
			body.Interval, tx.Plan.Code, interval, interval)
	}

	// ── 5. Amount must match the plan ────────────────────────────────────
	//
	// There is no zero-amount exemption: Paystack has no native free trial,
	// so a ₦0 charge against a priced plan is never legitimate here.
	switch {
	case !tx.Plan.AmountKnown:
		// Paystack returned a bare plan code with no plan object. The plan
		// code check above already carries the weight here. Attaching our
		// plan code makes Paystack charge that plan's price, so accept, but
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
	// Same user re-verifying must not fail. A double submit or a retried
	// handlePaystackSuccess should not lock a paying customer out, but it must
	// not be *applied* twice either: the period is extended from whatever is
	// left, so replaying a reference would hand out free time.
	//
	// Recording the verification claims the reference. The UNIQUE index on
	// event_key decides, which is stronger than comparing against
	// provider_sub_id. That only remembers the most recent reference and
	// would let an older one be replayed after a renewal.
	claim := sha256.Sum256([]byte("paystack-verify:" + body.Reference))
	claimKey := hex.EncodeToString(claim[:])
	switch err := h.Store.RecordBillingEvent(&models.BillingEvent{
		UserID:    user.ID,
		Provider:  "paystack",
		EventType: "verify",
		Payload:   body.Reference,
		EventKey:  claimKey,
		ObjectID:  tx.CustomerCode,
	}); {
	case errors.Is(err, store.ErrDuplicateBillingEvent):
		log.Printf("VerifyPaystack: reference %s already applied for user=%s. Returning current state",
			body.Reference, user.ID)
		current, err := h.Store.GetSubscription(user.ID)
		if err != nil {
			http.Error(w, "store error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plan":      current.Plan,
			"status":    current.Status,
			"is_trial":  false,
			"trial_end": nil,
		})
		return
	case err != nil:
		log.Printf("VerifyPaystack: could not claim reference %s: %v", body.Reference, err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	log.Printf("VerifyPaystack: verified user=%s ref=%s plan=%s interval=%s amount=%d card_country=%s channel=%s reusable=%t payer_ip=%s",
		user.ID, body.Reference, tx.Plan.Code, interval, tx.Amount,
		tx.CardCountry, tx.Channel, tx.Reusable, tx.PayerIP)

	// Paystack subscriptions require an authorization that can be charged
	// again. A bank transfer cannot, so Paystack takes the plan amount once
	// and creates no subscription. The customer has bought a single period.
	// They keep what they paid for; the UI has to stop calling it a
	// subscription.
	if !tx.Reusable {
		log.Printf("VerifyPaystack: %s payment is not reusable. User=%s gets one period and will NOT auto-renew",
			tx.Channel, user.ID)
	}

	// NGN pricing is substantially cheaper than USD, and the currency is
	// chosen client-side. LocalStorage hookdrop_currency is enough to pick
	// it. The card's country is the only server-side evidence of where the
	// payer really is.
	//
	// Flagged, not blocked: a Nigerian customer paying with a foreign-issued
	// card is a real case, and refusing them after they have been charged is
	// a worse failure than the mispricing. This makes the exposure visible so
	// it can be judged on evidence.
	if tx.CardCountry != "" && !billing.PaystackCountries[tx.CardCountry] {
		log.Printf("WARNING: VerifyPaystack: NGN pricing paid with a %s card. User=%s ref=%s customer=%s payer_ip=%s (review: possible currency mispricing)",
			tx.CardCountry, user.ID, body.Reference, tx.CustomerCode, tx.PayerIP)
	}

	now := time.Now().UTC()

	// Paystack has no native free trial, so the customer has just been
	// charged and is active from now. trial_end stays nil; only the Lemon
	// Squeezy path produces trialing subscriptions.
	subStatus := "active"

	// Stack onto whatever is left rather than restarting from now. A
	// non-recurring subscription is renewed by paying again, and someone who
	// renews early should not silently lose the days they already had.
	// An expired period is not carried forward. That would backdate the
	// renewal to a date already passed.
	base := now
	if current, err := h.Store.GetSubscription(user.ID); err == nil &&
		current.CurrentPeriodEnd != nil &&
		current.CurrentPeriodEnd.After(now) {
		base = *current.CurrentPeriodEnd
	}
	pe := base.Add(billing.PaystackBillingPeriod(interval))
	periodEnd := &pe

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
		Currency:           "ngn",
		Interval:           interval,
		CancelAtPeriodEnd:  false,
		AutoRenews:         tx.Reusable,
		CreatedAt:          now,
	}

	log.Printf("VerifyPaystack: upserting. User=%s plan=pro status=%s customer=%s",
		user.ID, subStatus, customerCode)

	if err := h.Store.UpsertSubscription(sub); err != nil {
		log.Printf("VerifyPaystack: upsert FAILED: %v", err)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	log.Printf("VerifyPaystack: upsert SUCCESS. User=%s is now Pro (status=%s)",
		user.ID, subStatus)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plan":   "pro",
		"status": subStatus,
		// Retained for the client contract; Paystack never yields a trial.
		"is_trial":  false,
		"trial_end": nil,
	})
}

// resolveWebhookUser finds the local user for an event whose payload carried no
// user_id, and reports which key resolved it.
//
// Lemon Squeezy puts user_id in meta.custom_data on every subscription event,
// so this is a safety net there. On Paystack it is load-bearing: renewals are
// charged against a stored authorization and carry none of the transaction
// metadata from the original checkout, but they always name the customer.
func (h *BillingHandler) resolveWebhookUser(
	event *billing.WebhookEvent,
) (*models.Subscription, string, error) {
	if sub, err := h.Store.GetSubscriptionByProviderSubID(event.SubscriptionID); err != nil {
		return nil, "", fmt.Errorf("lookup by provider_sub_id %s: %w", event.SubscriptionID, err)
	} else if sub != nil {
		return sub, "provider_sub_id=" + event.SubscriptionID, nil
	}

	if sub, err := h.Store.GetSubscriptionByProviderCustomerID(event.CustomerID); err != nil {
		return nil, "", fmt.Errorf("lookup by provider_customer_id %s: %w", event.CustomerID, err)
	} else if sub != nil {
		return sub, "provider_customer_id=" + event.CustomerID, nil
	}

	return nil, "", nil
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
		// Already scheduled for cancellation. Return success idempotently
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
		// provider_sub_id holds a real SUB_ code once subscription.create has
		// been applied. Rows created before that fix hold the transaction
		// reference instead, which Paystack cannot cancel.
		if strings.HasPrefix(sub.ProviderSubID, "SUB_") {
			if err := h.Paystack.CancelSubscription(r.Context(), sub.ProviderSubID); err != nil {
				log.Printf("CancelSubscription: Paystack cancel failed for user=%s sub=%s: %v",
					user.ID, sub.ProviderSubID, err)
				http.Error(w, "could not cancel subscription with the payment provider",
					http.StatusBadGateway)
				return
			}
			log.Printf("CancelSubscription: Paystack subscription %s disabled", sub.ProviderSubID)
		} else {
			// There is nothing callable. Honour the intent locally, but say
			// plainly that the provider was not told and is still billing.
			log.Printf("WARNING: CancelSubscription: user=%s has no Paystack subscription code (provider_sub_id=%q). Cancelled locally only; Paystack was NOT notified and will keep billing",
				user.ID, sub.ProviderSubID)
		}
	} else {
		// LemonSqueezy. Use stored sub ID directly.
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
