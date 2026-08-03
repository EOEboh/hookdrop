package billing

import "context"

// CheckoutParams is what the frontend sends to initiate checkout
type CheckoutParams struct {
	UserID     string
	Email      string
	Plan       string // "pro"
	Interval   string // "month" or "year"
	Currency   string // "ngn" or "usd"
	SuccessURL string
	CancelURL  string
}

// CheckoutResult: returned to the frontend
// The json tags are load-bearing: the UI reads redirect_url / access_code
// (see ui/src/api/client.ts and BillingContext.startCheckout)
type CheckoutResult struct {
	// Lemonsqueezy returns a hosted checkout URL
	// Paystack returns an authorization URL
	RedirectURL string `json:"redirect_url"`
	// Paystack inline: access code for inline popup
	AccessCode string `json:"access_code,omitempty"`
}

// Provider: interface both Stripe and Paystack implement
type Provider interface {
	Name() string
	CreateCheckout(ctx context.Context, params CheckoutParams) (*CheckoutResult, error)
	CancelSubscription(ctx context.Context, subID string) error
	GetPortalURL(ctx context.Context, customerID, returnURL string) (string, error)
	HandleWebhook(payload []byte, signature string) (*WebhookEvent, error)
}

// WebhookEvent: normalised event from either provider
type WebhookEvent struct {
	Type           string // "subscription.created", "subscription.updated", etc.
	UserID         string
	CustomerID     string
	SubscriptionID string
	Plan           string
	Status         string
	Currency       string
	Interval       string
	PeriodEnd      int64 // Unix timestamp
	TrialEnd       int64
	CancelAtEnd    bool
	// EventAt is the provider's own timestamp for this event, used to reject
	// deliveries that arrive out of order. Zero when the payload carries none.
	EventAt int64
}

// PaystackCountries are the countries Paystack can charge. Everywhere else
// goes through Lemon Squeezy.
//
// Currency selection happens on the client (detectCurrency in
// ui/src/context/BillingContext.tsx), which keeps its own timezone list
// derived from this map. Keep the two in step.
var PaystackCountries = map[string]bool{
	"NG": true, "GH": true, "ZA": true, "KE": true,
	"CI": true, "RW": true, "TZ": true, "EG": true,
	"UG": true, "CM": true, "ZM": true, "SN": true,
	"ET": true, "MZ": true,
}
