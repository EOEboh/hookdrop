package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

const (
	testSecret     = "test-signing-secret"
	variantMonthly = "111111"
	variantAnnual  = "222222"
)

func testProvider() *LemonSqueezyProvider {
	return NewLemonSqueezyProvider(
		"key", testSecret, "99999",
		LemonSqueezyVariants{ProMonthly: variantMonthly, ProAnnual: variantAnnual},
		true,
	)
}

func sign(t *testing.T, payload string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func handle(t *testing.T, payload string) (*WebhookEvent, error) {
	t.Helper()
	return testProvider().HandleWebhook([]byte(payload), sign(t, payload))
}

// subscriptionPayload mirrors the shape Lemonsqueezy actually sends for
// subscription events. Note first_subscription_item is null. That is what a
// subscription on a free trial looks like.
const trialCreatedPayload = `{
  "meta": {
    "event_name": "subscription_created",
    "test_mode": true,
    "custom_data": { "user_id": "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f" }
  },
  "data": {
    "type": "subscriptions",
    "id": "878123",
    "attributes": {
      "store_id": 99999,
      "customer_id": 4210987,
      "order_id": 5551234,
      "order_item_id": 777,
      "product_id": 333,
      "variant_id": 111111,
      "product_name": "hookdrop Pro",
      "variant_name": "Monthly",
      "user_name": "Test User",
      "user_email": "test@example.com",
      "status": "on_trial",
      "status_formatted": "On Trial",
      "card_brand": "visa",
      "card_last_four": "4242",
      "pause": null,
      "cancelled": false,
      "trial_ends_at": "2026-08-10T12:00:00.000000Z",
      "billing_anchor": 27,
      "first_subscription_item": null,
      "urls": {
        "update_payment_method": "https://s.lemonsqueezy.com/subscription/878123/payment-details?expires=1&signature=a",
        "customer_portal": "https://s.lemonsqueezy.com/billing?expires=1&signature=b"
      },
      "renews_at": "2026-08-10T12:00:00.000000Z",
      "ends_at": null,
      "created_at": "2026-07-27T12:00:00.000000Z",
      "updated_at": "2026-07-27T12:00:00.000000Z",
      "test_mode": true
    }
  }
}`

func TestHandleWebhook_TrialSubscriptionCreated(t *testing.T) {
	ev, err := handle(t, trialCreatedPayload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev == nil {
		t.Fatal("expected an event, got nil")
	}

	if ev.Type != "subscription.created" {
		t.Errorf("Type = %q, want subscription.created", ev.Type)
	}
	if ev.UserID != "9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f" {
		t.Errorf("UserID = %q, want the custom_data value", ev.UserID)
	}
	if ev.Plan != "pro" {
		t.Errorf("Plan = %q, want pro", ev.Plan)
	}
	if ev.Status != "trialing" {
		t.Errorf("Status = %q, want trialing", ev.Status)
	}
	// The customer ID must be the LS customer, not the order or subscription.
	// GetPortalURL calls GET /customers/{id} with it.
	if ev.CustomerID != "4210987" {
		t.Errorf("CustomerID = %q, want 4210987 (customer_id, not order_id)", ev.CustomerID)
	}
	if ev.SubscriptionID != "878123" {
		t.Errorf("SubscriptionID = %q, want 878123 (data.id)", ev.SubscriptionID)
	}
	if ev.Interval != "month" {
		t.Errorf("Interval = %q, want month", ev.Interval)
	}
	if ev.TrialEnd == 0 {
		t.Error("TrialEnd = 0, want the parsed trial_ends_at. A nil trial_end renders 'Trial ends soon' forever")
	}
	if want := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC).Unix(); ev.TrialEnd != want {
		t.Errorf("TrialEnd = %d, want %d", ev.TrialEnd, want)
	}
	if ev.PeriodEnd == 0 {
		t.Error("PeriodEnd = 0, want the parsed renews_at")
	}
	if ev.CancelAtEnd {
		t.Error("CancelAtEnd = true on a fresh subscription")
	}
}

// The annual variant is the only way to know the interval: the payload has no
// interval field anywhere.
func TestHandleWebhook_AnnualVariantYieldsYearInterval(t *testing.T) {
	payload := `{
      "meta": {"event_name":"subscription_created","custom_data":{"user_id":"u1"}},
      "data": {"id":"1","attributes":{"customer_id":1,"variant_id":222222,"status":"active",
        "cancelled":false,"trial_ends_at":null,"first_subscription_item":null,
        "renews_at":"2027-07-27T12:00:00.000000Z","ends_at":null}}
    }`
	ev, err := handle(t, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Interval != "year" {
		t.Errorf("Interval = %q, want year", ev.Interval)
	}
	if ev.Plan != "pro" {
		t.Errorf("Plan = %q, want pro", ev.Plan)
	}
}

// order_created carries an Order, not a Subscription: no top-level variant_id
// and status "paid". Parsing it as a subscription used to overwrite a paying
// customer with plan=free / status=paid / provider_sub_id=<order id>.
func TestHandleWebhook_IgnoresNonSubscriptionEvents(t *testing.T) {
	cases := map[string]string{
		"order_created": `{
          "meta":{"event_name":"order_created","custom_data":{"user_id":"u1"}},
          "data":{"type":"orders","id":"5551234","attributes":{"store_id":99999,
            "customer_id":4210987,"status":"paid","total":900,
            "first_order_item":{"order_id":5551234,"variant_id":111111}}}}`,
		"subscription_payment_success": `{
          "meta":{"event_name":"subscription_payment_success"},
          "data":{"type":"subscription-invoices","id":"7001","attributes":{
            "store_id":99999,"customer_id":4210987,"subscription_id":878123,"status":"paid"}}}`,
		"license_key_created": `{
          "meta":{"event_name":"license_key_created","custom_data":{"user_id":"u1"}},
          "data":{"type":"license-keys","id":"1","attributes":{"store_id":99999}}}`,
	}

	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			ev, err := handle(t, payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ev != nil {
				t.Fatalf("expected nil event, got %+v. This would corrupt the subscriptions row", ev)
			}
		})
	}
}

// Cancelling starts a grace period; access continues until ends_at. It must not
// map to subscription.canceled, which downgrades the plan to free.
func TestHandleWebhook_CancelKeepsAccessUntilEndsAt(t *testing.T) {
	payload := `{
      "meta":{"event_name":"subscription_cancelled","custom_data":{"user_id":"u1"}},
      "data":{"id":"878123","attributes":{"customer_id":4210987,"variant_id":111111,
        "status":"cancelled","cancelled":true,"trial_ends_at":null,
        "first_subscription_item":{"id":1,"subscription_id":878123,"price_id":9,"quantity":1},
        "renews_at":"2026-08-27T12:00:00.000000Z","ends_at":"2026-08-27T12:00:00.000000Z"}}}`

	ev, err := handle(t, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type == "subscription.canceled" {
		t.Error("subscription_cancelled mapped to subscription.canceled. That drops the customer to free during a grace period they paid for")
	}
	if ev.Type != "subscription.updated" {
		t.Errorf("Type = %q, want subscription.updated", ev.Type)
	}
	if ev.Plan != "pro" {
		t.Errorf("Plan = %q, want pro during the grace period", ev.Plan)
	}
	if !ev.CancelAtEnd {
		t.Error("CancelAtEnd = false, want true")
	}
	if want := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC).Unix(); ev.PeriodEnd != want {
		t.Errorf("PeriodEnd = %d, want ends_at %d", ev.PeriodEnd, want)
	}
	if ev.Status != "active" {
		t.Errorf("Status = %q, want active until the period ends", ev.Status)
	}
}

// `cancelled` stays true after expiry, so it must not mask the expired status.
func TestHandleWebhook_ExpiredDowngrades(t *testing.T) {
	payload := `{
      "meta":{"event_name":"subscription_expired","custom_data":{"user_id":"u1"}},
      "data":{"id":"878123","attributes":{"customer_id":4210987,"variant_id":111111,
        "status":"expired","cancelled":true,"trial_ends_at":null,
        "first_subscription_item":null,
        "renews_at":"2026-08-27T12:00:00.000000Z","ends_at":"2026-08-27T12:00:00.000000Z"}}}`

	ev, err := handle(t, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != "subscription.canceled" {
		t.Errorf("Type = %q, want subscription.canceled", ev.Type)
	}
	if ev.Status != "canceled" {
		t.Errorf("Status = %q, want canceled. An expired subscription reported as active keeps Pro forever", ev.Status)
	}
}

func TestHandleWebhook_SignatureVerification(t *testing.T) {
	p := testProvider()

	t.Run("valid", func(t *testing.T) {
		if _, err := p.HandleWebhook([]byte(trialCreatedPayload), sign(t, trialCreatedPayload)); err != nil {
			t.Fatalf("valid signature rejected: %v", err)
		}
	})

	t.Run("wrong signature", func(t *testing.T) {
		_, err := p.HandleWebhook([]byte(trialCreatedPayload), "deadbeef")
		if err == nil {
			t.Fatal("forged signature accepted")
		}
	})

	t.Run("tampered payload", func(t *testing.T) {
		sig := sign(t, trialCreatedPayload)
		_, err := p.HandleWebhook([]byte(trialCreatedPayload+" "), sig)
		if err == nil {
			t.Fatal("tampered payload accepted")
		}
	})

	t.Run("missing signature header", func(t *testing.T) {
		if _, err := p.HandleWebhook([]byte(trialCreatedPayload), ""); err == nil {
			t.Fatal("empty signature accepted")
		}
	})

	t.Run("unconfigured secret rejects everything", func(t *testing.T) {
		unset := NewLemonSqueezyProvider("key", "", "99999", LemonSqueezyVariants{}, true)
		// An empty key still produces a valid HMAC, so without the guard this
		// forged payload would verify.
		mac := hmac.New(sha256.New, nil)
		mac.Write([]byte(trialCreatedPayload))
		forged := hex.EncodeToString(mac.Sum(nil))

		if _, err := unset.HandleWebhook([]byte(trialCreatedPayload), forged); err == nil {
			t.Fatal("webhook accepted with an unconfigured signing secret")
		}
	})
}

func TestNormaliseLSStatus(t *testing.T) {
	future := time.Now().Add(24 * time.Hour).Unix()

	cases := []struct {
		status    string
		cancelled bool
		trialEnd  int64
		want      string
	}{
		{"on_trial", false, future, "trialing"},
		{"active", false, 0, "active"},
		{"cancelled", true, 0, "active"},        // grace period
		{"cancelled", true, future, "trialing"}, // cancelled mid-trial
		{"expired", true, 0, "canceled"},
		{"past_due", false, 0, "past_due"},
		{"unpaid", true, 0, "past_due"},
		{"paused", false, 0, "past_due"},
	}

	for _, c := range cases {
		if got := normaliseLSStatus(c.status, c.cancelled, c.trialEnd); got != c.want {
			t.Errorf("normaliseLSStatus(%q, %t, trial=%d) = %q, want %q",
				c.status, c.cancelled, c.trialEnd, got, c.want)
		}
	}
}
