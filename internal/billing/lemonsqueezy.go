package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Variant IDs from your Lemonsqueezy dashboard
// Products → your product → Variants → copy the numeric ID
type LemonSqueezyVariants struct {
	ProMonthly string // e.g. "123456"
	ProAnnual  string // e.g. "123457"
}

type LemonSqueezyProvider struct {
	APIKey     string
	WebhookKey string
	StoreID    string
	Variants   LemonSqueezyVariants
}

func NewLemonSqueezyProvider(
	apiKey, webhookKey, storeID string,
	variants LemonSqueezyVariants,
) *LemonSqueezyProvider {
	return &LemonSqueezyProvider{
		APIKey:     apiKey,
		WebhookKey: webhookKey,
		StoreID:    storeID,
		Variants:   variants,
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("lemonsqueezy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("lemonsqueezy API %d: %v", resp.StatusCode, errBody)
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

	// Build the checkout payload
	// Lemonsqueezy uses JSON:API format
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "checkouts",
			"attributes": map[string]interface{}{
				"store_id":   p.StoreID,
				"variant_id": variantID,

				// Pre-fill customer email
				"checkout_data": map[string]interface{}{
					"email": params.Email,
					"custom": map[string]interface{}{
						// Pass user ID through custom data
						// Available in webhook as custom_data.user_id
						"user_id": params.UserID,
					},
				},

				// Redirect URLs
				"product_options": map[string]interface{}{
					"redirect_url":     params.SuccessURL,
					"receipt_link_url": params.SuccessURL,
				},

				// 14-day free trial
				// Lemonsqueezy trial is set on the variant in the dashboard
				// but can be overridden here per checkout
				"checkout_options": map[string]interface{}{
					"embed":        false,
					"media":        false,
					"logo":         true,
					"button_color": "#10b981",
				},

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
	// DELETE /subscriptions/{id} cancels at period end
	return p.doRequest(ctx, "DELETE", "/subscriptions/"+subID, nil, nil)
}

func (p *LemonSqueezyProvider) GetPortalURL(
	ctx context.Context,
	customerID, returnURL string,
) (string, error) {
	// Lemonsqueezy customer portal
	// customerID here is the Lemonsqueezy customer ID (numeric string)
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
		// Fallback to billing settings page in the app
		return returnURL + "/settings/billing", nil
	}
	return url, nil
}

// HandleWebhook verifies and parses incoming Lemonsqueezy webhook events
func (p *LemonSqueezyProvider) HandleWebhook(
	payload []byte,
	signature string,
) (*WebhookEvent, error) {
	// Verify HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(p.WebhookKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, fmt.Errorf("lemonsqueezy webhook signature invalid")
	}

	// Parse the event envelope
	var envelope struct {
		Meta struct {
			EventName  string `json:"event_name"`
			CustomData struct {
				UserID string `json:"user_id"`
			} `json:"custom_data"`
		} `json:"meta"`
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				StoreID               int         `json:"store_id"`
				CustomerID            int         `json:"customer_id"`
				OrderID               int         `json:"order_id"`
				ProductID             int         `json:"product_id"`
				VariantID             int         `json:"variant_id"`
				Status                string      `json:"status"`
				Cancelled             bool        `json:"cancelled"`
				Pause                 interface{} `json:"pause"`
				BillingAnchor         int         `json:"billing_anchor"`
				RenewsAt              string      `json:"renews_at"`
				EndsAt                string      `json:"ends_at"`
				CreatedAt             string      `json:"created_at"`
				UpdatedAt             string      `json:"updated_at"`
				FirstSubscriptionItem struct {
					SubscriptionID int    `json:"subscription_id"`
					PriceID        int    `json:"price_id"`
					Quantity       int    `json:"quantity"`
					Interval       string `json:"interval"`
					IntervalCount  int    `json:"interval_count"`
				} `json:"first_subscription_item"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("parse webhook payload: %w", err)
	}

	attr := envelope.Data.Attributes

	// Parse period end time
	periodEnd := int64(0)
	if attr.RenewsAt != "" {
		if t, err := time.Parse(time.RFC3339, attr.RenewsAt); err == nil {
			periodEnd = t.Unix()
		}
	}
	// If cancelled, use ends_at as the period end
	if attr.EndsAt != "" && attr.Cancelled {
		if t, err := time.Parse(time.RFC3339, attr.EndsAt); err == nil {
			periodEnd = t.Unix()
		}
	}

	// Determine interval from subscription item
	interval := "month"
	if attr.FirstSubscriptionItem.Interval == "year" {
		interval = "year"
	}

	event := &WebhookEvent{
		Type:           normaliseLSEvent(envelope.Meta.EventName),
		UserID:         envelope.Meta.CustomData.UserID,
		CustomerID:     fmt.Sprintf("%d", attr.CustomerID),
		SubscriptionID: envelope.Data.ID,
		Plan:           p.planFromVariant(attr.VariantID),
		Status:         normaliseLSStatus(attr.Status, attr.Cancelled),
		Currency:       "usd",
		Interval:       interval,
		PeriodEnd:      periodEnd,
		CancelAtEnd:    attr.Cancelled,
	}

	return event, nil
}

func normaliseLSEvent(e string) string {
	switch e {
	case "subscription_created":
		return "subscription.created"
	case "subscription_updated":
		return "subscription.updated"
	case "subscription_cancelled",
		"subscription_expired":
		return "subscription.canceled"
	case "subscription_resumed":
		return "subscription.updated"
	case "subscription_paused":
		return "subscription.updated"
	default:
		return e
	}
}

func normaliseLSStatus(status string, cancelled bool) string {
	if cancelled {
		return "active" // still active until period ends
	}
	switch status {
	case "active":
		return "active"
	case "on_trial":
		return "trialing"
	case "past_due", "unpaid":
		return "past_due"
	case "cancelled", "expired":
		return "canceled"
	case "paused":
		return "past_due" // treat paused as at-risk
	default:
		return status
	}
}

func (p *LemonSqueezyProvider) planFromVariant(variantID int) string {
	id := fmt.Sprintf("%d", variantID)
	if id == p.Variants.ProMonthly || id == p.Variants.ProAnnual {
		return "pro"
	}
	return "free"
}
