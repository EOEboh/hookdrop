package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PaystackPlans struct {
	ProMonthly string // e.g. PLN_xxxxxxxxxxxx
	ProAnnual  string
}

type PaystackProvider struct {
	SecretKey  string
	WebhookKey string
	Plans      PaystackPlans
}

func NewPaystackProvider(secretKey, webhookKey string, plans PaystackPlans) *PaystackProvider {
	return &PaystackProvider{
		SecretKey:  secretKey,
		WebhookKey: webhookKey,
		Plans:      plans,
	}
}

func (p *PaystackProvider) Name() string { return "paystack" }

func (p *PaystackProvider) CreateCheckout(ctx context.Context, params CheckoutParams) (*CheckoutResult, error) {
	planCode := p.Plans.ProMonthly
	if params.Interval == "year" {
		planCode = p.Plans.ProAnnual
	}

	// Trial via Paystack: start_date 14 days from now
	startDate := time.Now().Add(14 * 24 * time.Hour).Format("2006-01-02")

	reqBody, _ := json.Marshal(map[string]interface{}{
		"email":        params.Email,
		"plan":         planCode,
		"callback_url": params.SuccessURL,
		"metadata": map[string]interface{}{
			"user_id":    params.UserID,
			"cancel_url": params.CancelURL,
		},
		"start_date": startDate,
	})

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		"https://api.paystack.co/transaction/initialize",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paystack initialize: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status bool `json:"status"`
		Data   struct {
			AuthorizationURL string `json:"authorization_url"`
			AccessCode       string `json:"access_code"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Status {
		return nil, fmt.Errorf("paystack initialization failed")
	}

	return &CheckoutResult{
		RedirectURL: result.Data.AuthorizationURL,
		AccessCode:  result.Data.AccessCode,
	}, nil
}

func (p *PaystackProvider) CancelSubscription(ctx context.Context, subCode string) error {
	req, err := http.NewRequestWithContext(ctx,
		"POST",
		fmt.Sprintf("https://api.paystack.co/subscription/disable"),
		strings.NewReader(fmt.Sprintf(
			`{"code":"%s","token":""}`, subCode,
		)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (p *PaystackProvider) GetPortalURL(ctx context.Context, customerCode, returnURL string) (string, error) {

	return returnURL + "/settings/billing", nil
}

func (p *PaystackProvider) HandleWebhook(payload []byte, signature string) (*WebhookEvent, error) {
	// Verify HMAC-SHA512
	mac := hmac.New(sha512.New, []byte(p.WebhookKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, fmt.Errorf("paystack webhook signature invalid")
	}

	var event struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	switch event.Event {
	case "subscription.create",
		"subscription.not_renew",
		"subscription.disable":
		var sub struct {
			SubscriptionCode string `json:"subscription_code"`
			Status           string `json:"status"`
			PlanCode         string `json:"plan"`
			EmailToken       string `json:"email_token"`
			NextPaymentDate  string `json:"next_payment_date"`
			Customer         struct {
				CustomerCode string `json:"customer_code"`
				Email        string `json:"email"`
				Metadata     struct {
					UserID string `json:"user_id"`
				} `json:"metadata"`
			} `json:"customer"`
		}
		if err := json.Unmarshal(event.Data, &sub); err != nil {
			return nil, err
		}

		periodEnd := int64(0)
		if sub.NextPaymentDate != "" {
			if t, err := time.Parse(time.RFC3339, sub.NextPaymentDate); err == nil {
				periodEnd = t.Unix()
			}
		}

		return &WebhookEvent{
			Type:           normalisePaystackEvent(event.Event),
			CustomerID:     sub.Customer.CustomerCode,
			SubscriptionID: sub.SubscriptionCode,
			Plan:           p.planFromCode(sub.PlanCode),
			Status:         normalisePaystackStatus(sub.Status),
			Currency:       "ngn",
			PeriodEnd:      periodEnd,
			UserID:         sub.Customer.Metadata.UserID,
		}, nil
	}
	return nil, nil
}

func normalisePaystackEvent(e string) string {
	switch e {
	case "subscription.create":
		return "subscription.created"
	case "subscription.not_renew":
		return "subscription.updated"
	case "subscription.disable":
		return "subscription.canceled"
	}
	return e
}

func normalisePaystackStatus(s string) string {
	switch s {
	case "active":
		return "active"
	case "non-renewing":
		return "active" // still active, just won't renew
	case "attention", "cancelled":
		return "canceled"
	case "completed":
		return "canceled"
	}
	return s
}

func (p *PaystackProvider) planFromCode(code string) string {
	if code == p.Plans.ProMonthly || code == p.Plans.ProAnnual {
		return "pro"
	}
	return "free"
}
