package billing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/billingportal/session"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

// Price IDs from your Stripe dashboard
// Create these in Stripe and paste the IDs here
type StripePrices struct {
	ProMonthly string
	ProAnnual  string
}

type StripeProvider struct {
	SecretKey  string
	WebhookKey string
	Prices     StripePrices
}

func NewStripeProvider(secretKey, webhookKey string, prices StripePrices) *StripeProvider {
	stripe.Key = secretKey
	return &StripeProvider{
		SecretKey:  secretKey,
		WebhookKey: webhookKey,
		Prices:     prices,
	}
}

func (p *StripeProvider) Name() string { return "stripe" }

func (p *StripeProvider) CreateCheckout(ctx context.Context, params CheckoutParams) (*CheckoutResult, error) {
	priceID := p.Prices.ProMonthly
	if params.Interval == "year" {
		priceID = p.Prices.ProAnnual
	}

	// Trial period — 14 days
	trialDays := int64(14)

	s, err := session.New(&stripe.CheckoutSessionParams{
		CustomerEmail: stripe.String(params.Email),
		Mode:          stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:    stripe.String(params.SuccessURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:     stripe.String(params.CancelURL),
		Metadata: map[string]string{
			"user_id": params.UserID,
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(trialDays),
			Metadata: map[string]string{
				"user_id": params.UserID,
			},
		},
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: stripe.Bool(true),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("stripe checkout: %w", err)
	}

	return &CheckoutResult{RedirectURL: s.URL}, nil
}

func (p *StripeProvider) CancelSubscription(ctx context.Context, subID string) error {
	_, err := sub.Update(subID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	})
	return err
}

func (p *StripeProvider) GetPortalURL(ctx context.Context, customerID, returnURL string) (string, error) {
	s, err := portalsession.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	})
	if err != nil {
		return "", err
	}
	return s.URL, nil
}

func (p *StripeProvider) HandleWebhook(payload []byte, signature string) (*WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, p.WebhookKey)
	if err != nil {
		return nil, fmt.Errorf("stripe webhook signature: %w", err)
	}

	switch event.Type {
	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return nil, err
		}
		return &WebhookEvent{
			Type:           normaliseStripeEvent(event.Type),
			CustomerID:     sub.Customer.ID,
			SubscriptionID: sub.ID,
			Plan:           planFromStripePrices(sub, p.Prices),
			Status:         string(sub.Status),
			Currency:       string(sub.Currency),
			Interval:       string(sub.Items.Data[0].Price.Recurring.Interval),
			PeriodEnd:      sub.CurrentPeriodEnd,
			TrialEnd:       sub.TrialEnd,
			CancelAtEnd:    sub.CancelAtPeriodEnd,
			UserID:         sub.Metadata["user_id"],
		}, nil
	}
	return nil, nil // unhandled event type — not an error
}

func normaliseStripeEvent(t string) string {
	switch t {
	case "customer.subscription.created":
		return "subscription.created"
	case "customer.subscription.updated":
		return "subscription.updated"
	case "customer.subscription.deleted":
		return "subscription.canceled"
	}
	return t
}

func planFromStripePrices(sub stripe.Subscription, prices StripePrices) string {
	if len(sub.Items.Data) == 0 {
		return "free"
	}
	priceID := sub.Items.Data[0].Price.ID
	if priceID == prices.ProMonthly || priceID == prices.ProAnnual {
		return "pro"
	}
	return "free"
}
