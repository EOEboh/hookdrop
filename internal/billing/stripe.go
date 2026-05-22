package billing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	stripesubscription "github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

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

	s, err := checkoutsession.New(&stripe.CheckoutSessionParams{
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
			TrialPeriodDays: stripe.Int64(14),
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
	// Cancel at period end — user keeps access until their billing cycle ends
	_, err := stripesubscription.Update(subID, &stripe.SubscriptionParams{
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
		return "", fmt.Errorf("stripe portal: %w", err)
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
			return nil, fmt.Errorf("unmarshal subscription: %w", err)
		}

		// Customer field on Subscription is an expandable object
		// It may be just an ID string or a full object depending on expansion
		customerID := ""
		if sub.Customer != nil {
			customerID = sub.Customer.ID
		}

		interval := ""
		plan := "free"
		if len(sub.Items.Data) > 0 {
			item := sub.Items.Data[0]
			if item.Price != nil {
				if item.Price.Recurring != nil {
					interval = string(item.Price.Recurring.Interval)
				}
				plan = p.planFromPriceID(item.Price.ID)
			}
		}

		return &WebhookEvent{
			Type:           normaliseStripeEvent(string(event.Type)),
			CustomerID:     customerID,
			SubscriptionID: sub.ID,
			Plan:           plan,
			Status:         string(sub.Status),
			Currency:       string(sub.Currency),
			Interval:       interval,
			PeriodEnd:      sub.CurrentPeriodEnd,
			TrialEnd:       sub.TrialEnd,
			CancelAtEnd:    sub.CancelAtPeriodEnd,
			UserID:         sub.Metadata["user_id"],
		}, nil
	}

	// Unhandled event type — not an error, just ignore
	return nil, nil
}

func (p *StripeProvider) planFromPriceID(priceID string) string {
	if priceID == p.Prices.ProMonthly || priceID == p.Prices.ProAnnual {
		return "pro"
	}
	return "free"
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
