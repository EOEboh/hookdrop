package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Variant IDs from your Lemonsqueezy dashboard
// Products → your product → Variants → copy the numeric ID
// Test mode variants have DIFFERENT IDs to live mode variants.
type LemonSqueezyVariants struct {
	ProMonthly string // e.g. "123456"
	ProAnnual  string // e.g. "123457"
}

type LemonSqueezyProvider struct {
	APIKey     string
	WebhookKey string
	StoreID    string
	Variants   LemonSqueezyVariants
	// TestMode marks created checkouts as test mode. A test-mode API key only
	// ever sees test-mode data, so this must line up with the key in use.
	TestMode bool

	http *http.Client
}

func NewLemonSqueezyProvider(
	apiKey, webhookKey, storeID string,
	variants LemonSqueezyVariants,
	testMode bool,
) *LemonSqueezyProvider {
	return &LemonSqueezyProvider{
		APIKey:     apiKey,
		WebhookKey: webhookKey,
		StoreID:    storeID,
		Variants:   variants,
		TestMode:   testMode,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *LemonSqueezyProvider) Name() string { return "lemonsqueezy" }

// doRequest is a small helper for Lemonsqueezy API calls
func (p *LemonSqueezyProvider) doRequest(
	ctx context.Context,
	method, path string,
	body interface{},
	out interface{},
) error {
	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(
		ctx, method,
		"https://api.lemonsqueezy.com/v1"+path,
		reqBody,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")

	resp, err := p.http.Do(req)
	if err != nil {
		return fmt.Errorf("lemonsqueezy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Return the raw body: Lemonsqueezy validation errors name the exact
		// offending field, which is the only useful thing when a payload is
		// rejected. Decoding it into a map loses that.
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("lemonsqueezy API %s %s: %d: %s",
			method, path, resp.StatusCode, string(errBody))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (p *LemonSqueezyProvider) CreateCheckout(
	ctx context.Context,
	params CheckoutParams,
) (*CheckoutResult, error) {
	variantID := p.Variants.ProMonthly
	if params.Interval == "year" {
		variantID = p.Variants.ProAnnual
	}
	if variantID == "" {
		return nil, fmt.Errorf("no lemonsqueezy variant configured for interval %q", params.Interval)
	}

	// Lemonsqueezy uses JSON:API format.
	//
	// store_id and variant_id are READ-ONLY response attributes. The only
	// attributes accepted on create are custom_price, product_options,
	// checkout_options, checkout_data, preview, test_mode and expires_at.
	// The store and variant are addressed through relationships.
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "checkouts",
			"attributes": map[string]interface{}{
				"checkout_data": map[string]interface{}{
					// Pre-fill customer email
					"email": params.Email,
					"custom": map[string]interface{}{
						// Pass user ID through custom data. Comes back on
						// Order / Subscription webhooks as
						// meta.custom_data.user_id (as a string).
						"user_id": params.UserID,
					},
				},

				// Redirect URLs
				"product_options": map[string]interface{}{
					"redirect_url":     params.SuccessURL,
					"receipt_link_url": params.SuccessURL,
				},

				// The free trial is configured on the variant in the
				// Lemonsqueezy dashboard. There is no per-checkout trial
				// override. checkout_options.skip_trial would REMOVE it, so
				// it is deliberately not set here.
				"checkout_options": map[string]interface{}{
					"embed":        false,
					"media":        false,
					"logo":         true,
					"button_color": "#10b981",
				},

				"test_mode": p.TestMode,

				"expires_at": time.Now().UTC().
					Add(24 * time.Hour).
					Format(time.RFC3339),
			},
			"relationships": map[string]interface{}{
				"store": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "stores",
						"id":   p.StoreID,
					},
				},
				"variant": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "variants",
						"id":   variantID,
					},
				},
			},
		},
	}

	var result struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := p.doRequest(ctx, "POST", "/checkouts", payload, &result); err != nil {
		return nil, fmt.Errorf("create checkout: %w", err)
	}

	if result.Data.Attributes.URL == "" {
		return nil, fmt.Errorf("lemonsqueezy returned empty checkout URL")
	}

	return &CheckoutResult{
		RedirectURL: result.Data.Attributes.URL,
	}, nil
}

func (p *LemonSqueezyProvider) CancelSubscription(
	ctx context.Context,
	subID string,
) error {
	if subID == "" {
		return fmt.Errorf("lemonsqueezy: empty subscription id")
	}
	// DELETE /subscriptions/{id} cancels at period end: the subscription moves
	// to status "cancelled" with ends_at set, and expires from there.
	return p.doRequest(ctx, "DELETE", "/subscriptions/"+subID, nil, nil)
}

func (p *LemonSqueezyProvider) GetPortalURL(
	ctx context.Context,
	customerID, returnURL string,
) (string, error) {
	if customerID == "" {
		return "", fmt.Errorf("lemonsqueezy: no customer id stored for this subscription")
	}

	// customerID here is the Lemonsqueezy customer ID (numeric string), taken
	// from subscription webhook attributes.customer_id.
	//
	// The returned portal URL is pre-signed and expires 24h after it is
	// issued, so it is fetched per request and must never be cached.
	var result struct {
		Data struct {
			Attributes struct {
				URLs struct {
					CustomerPortal string `json:"customer_portal"`
				} `json:"urls"`
			} `json:"attributes"`
		} `json:"data"`
	}

	err := p.doRequest(
		ctx, "GET",
		"/customers/"+customerID,
		nil,
		&result,
	)
	if err != nil {
		return "", err
	}

	url := result.Data.Attributes.URLs.CustomerPortal
	if url == "" {
		// Do not silently bounce the user back to the page they came from.
		// That hides a real failure behind a no-op redirect.
		return "", fmt.Errorf(
			"lemonsqueezy customer %s returned no customer_portal URL", customerID)
	}
	return url, nil
}

// lsSubscriptionEvents are the only events we act on.
//
// Every other Lemonsqueezy event carries a DIFFERENT object shape:
// order_created sends an Order (no top-level variant_id, status "paid"),
// subscription_payment_* send a Subscription invoice. Parsing either as a
// subscription writes a garbage row. Plan "free", status "paid" and an order
// ID in provider_sub_id, over a paying customer.
var lsSubscriptionEvents = map[string]bool{
	"subscription_created":   true,
	"subscription_updated":   true,
	"subscription_cancelled": true,
	"subscription_resumed":   true,
	"subscription_expired":   true,
	"subscription_paused":    true,
	"subscription_unpaused":  true,
}

// HandleWebhook verifies and parses incoming Lemonsqueezy webhook events
func (p *LemonSqueezyProvider) HandleWebhook(
	payload []byte,
	signature string,
) (*WebhookEvent, error) {
	// An empty signing secret still produces a valid HMAC, which would let
	// anyone forge a webhook. Refuse rather than verify vacuously.
	if p.WebhookKey == "" {
		return nil, fmt.Errorf("lemonsqueezy webhook secret not configured")
	}
	if signature == "" {
		return nil, fmt.Errorf("lemonsqueezy webhook missing X-Signature header")
	}

	// Verify HMAC-SHA256 hex digest of the raw payload against X-Signature
	mac := hmac.New(sha256.New, []byte(p.WebhookKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, fmt.Errorf("lemonsqueezy webhook signature invalid")
	}

	// Parse the event envelope.
	//
	// Note there is deliberately no first_subscription_item here: it is null
	// while the subscription is on trial, and it carries no interval field.
	// The interval is derived from variant_id instead.
	var envelope struct {
		Meta struct {
			EventName  string `json:"event_name"`
			TestMode   bool   `json:"test_mode"`
			CustomData struct {
				UserID string `json:"user_id"`
			} `json:"custom_data"`
		} `json:"meta"`
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				StoreID       int    `json:"store_id"`
				CustomerID    int    `json:"customer_id"`
				OrderID       int    `json:"order_id"`
				ProductID     int    `json:"product_id"`
				VariantID     int    `json:"variant_id"`
				Status        string `json:"status"`
				Cancelled     bool   `json:"cancelled"`
				BillingAnchor int    `json:"billing_anchor"`
				TrialEndsAt   string `json:"trial_ends_at"`
				RenewsAt      string `json:"renews_at"`
				EndsAt        string `json:"ends_at"`
				CreatedAt     string `json:"created_at"`
				UpdatedAt     string `json:"updated_at"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("parse webhook payload: %w", err)
	}

	if !lsSubscriptionEvents[envelope.Meta.EventName] {
		// Not a subscription event, so there is nothing to persist.
		return nil, nil
	}

	attr := envelope.Data.Attributes

	// renews_at is the end of the current billing cycle. On a trialing
	// subscription it is the date of the first real charge.
	periodEnd := parseLSTime(attr.RenewsAt)
	// ends_at is only populated for cancelled/expired subscriptions and is the
	// date access actually stops, so it wins when present.
	if end := parseLSTime(attr.EndsAt); end > 0 {
		periodEnd = end
	}

	trialEnd := parseLSTime(attr.TrialEndsAt)

	event := &WebhookEvent{
		Type:           normaliseLSEvent(envelope.Meta.EventName),
		UserID:         envelope.Meta.CustomData.UserID,
		CustomerID:     fmt.Sprintf("%d", attr.CustomerID),
		SubscriptionID: envelope.Data.ID,
		Plan:           p.planFromVariant(attr.VariantID),
		Status:         normaliseLSStatus(attr.Status, attr.Cancelled, trialEnd),
		Currency:       "usd",
		Interval:       p.intervalFromVariant(attr.VariantID),
		PeriodEnd:      periodEnd,
		TrialEnd:       trialEnd,
		CancelAtEnd:    attr.Cancelled,
		EventAt:        parseLSTime(attr.UpdatedAt),
	}

	return event, nil
}

// parseLSTime parses a Lemonsqueezy ISO 8601 timestamp
// (e.g. "2021-08-11T13:47:28.000000Z"). Returns 0 for empty/null/unparseable.
func parseLSTime(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func normaliseLSEvent(e string) string {
	switch e {
	case "subscription_created":
		return "subscription.created"
	case "subscription_expired":
		// The grace period has run out (or dunning finished). This is the
		// point access actually ends.
		return "subscription.canceled"
	default:
		// subscription_updated / _cancelled / _resumed / _paused / _unpaused.
		//
		// subscription_cancelled deliberately does NOT map to
		// subscription.canceled: Lemonsqueezy fires it the moment the customer
		// cancels, but they keep the access they paid for until ends_at.
		// Downgrading here would cut off paid access early. cancel_at_period_end
		// and current_period_end carry the cancellation; subscription_expired
		// does the downgrade.
		return "subscription.updated"
	}
}

func normaliseLSStatus(status string, cancelled bool, trialEnd int64) string {
	switch status {
	case "expired":
		return "canceled"
	case "past_due", "unpaid":
		return "past_due"
	case "paused":
		return "past_due" // treat paused as at-risk
	case "on_trial":
		return "trialing"
	case "active":
		return "active"
	case "cancelled":
		// Cancelled but not yet expired: still a valid grace period, access
		// runs until ends_at. Keep reporting the trial if it is still running
		// so the UI keeps rendering the trial card.
		if trialEnd > 0 && time.Now().Unix() < trialEnd {
			return "trialing"
		}
		return "active"
	}

	// Unknown status. `cancelled` is checked last so it can never mask a
	// terminal status above (it stays true after a subscription expires).
	if cancelled {
		return "active"
	}
	return status
}

func (p *LemonSqueezyProvider) planFromVariant(variantID int) string {
	id := fmt.Sprintf("%d", variantID)
	if id == p.Variants.ProMonthly || id == p.Variants.ProAnnual {
		return "pro"
	}
	return "free"
}

// intervalFromVariant derives the billing interval from the variant.
//
// The webhook payload has no usable interval field:
// first_subscription_item carries only id / subscription_id / price_id /
// quantity / timestamps, and is null entirely during a free trial.
func (p *LemonSqueezyProvider) intervalFromVariant(variantID int) string {
	if fmt.Sprintf("%d", variantID) == p.Variants.ProAnnual {
		return "year"
	}
	return "month"
}
