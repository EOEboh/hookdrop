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
type CheckoutResult struct {
	// Stripe returns a redirect URL to hosted checkout
	// Paystack returns an authorization URL
	RedirectURL string
	// Paystack inline: access code for inline popup
	AccessCode string
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
}

// Paystack-supported countries
var PaystackCountries = map[string]bool{
	"NG": true, "GH": true, "ZA": true, "KE": true,
	"CI": true, "RW": true, "TZ": true, "EG": true,
	"UG": true, "CM": true, "ZM": true, "SN": true,
	"ET": true, "MZ": true,
}

func ProviderForCurrency(currency string) string {
	if currency == "ngn" {
		return "paystack"
	}
	return "stripe"
}

func ProviderForCountry(countryCode string) string {
	if PaystackCountries[countryCode] {
		return "paystack"
	}
	return "stripe"
}
