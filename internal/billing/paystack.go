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
	"net/url"
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
	// BaseURL is the Paystack API root. Overridden in tests to point at a
	// httptest server; empty means the live API.
	BaseURL string
	// RetryDelays is the backoff schedule for VerifyTransaction. Tests set it
	// to zeros so the suite does not sit through the real backoff.
	RetryDelays []time.Duration

	http *http.Client
}

// defaultVerifyDelays: retry twice after the first attempt. Paystack can
// briefly report a just-completed transaction as pending.
var defaultVerifyDelays = []time.Duration{0, 1 * time.Second, 2 * time.Second}

const paystackAPIBase = "https://api.paystack.co"

func NewPaystackProvider(secretKey, webhookKey string, plans PaystackPlans) *PaystackProvider {
	return &PaystackProvider{
		SecretKey:  secretKey,
		WebhookKey: webhookKey,
		Plans:      plans,
		BaseURL:    paystackAPIBase,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *PaystackProvider) baseURL() string {
	if p.BaseURL != "" {
		return p.BaseURL
	}
	return paystackAPIBase
}

func (p *PaystackProvider) client() *http.Client {
	if p.http != nil {
		return p.http
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (p *PaystackProvider) Name() string { return "paystack" }

func (p *PaystackProvider) CreateCheckout(ctx context.Context, params CheckoutParams) (*CheckoutResult, error) {
	planCode := p.Plans.ProMonthly
	if params.Interval == "year" {
		planCode = p.Plans.ProAnnual
	}

	// No start_date. Paystack has no native free trial: passing a plan to
	// transaction/initialize overrides the amount and charges the plan price
	// immediately, so a future start_date only shifts the *next* debit while
	// the customer is billed today. Sending it implied a trial that never
	// existed.
	reqBody, _ := json.Marshal(map[string]interface{}{
		"email":        params.Email,
		"plan":         planCode,
		"callback_url": params.SuccessURL,
		"metadata": map[string]interface{}{
			"user_id":    params.UserID,
			"cancel_url": params.CancelURL,
		},
	})

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		p.baseURL()+"/transaction/initialize",
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
			// Paystack spells these both ways across payloads.
			UpdatedAtCamel string `json:"updatedAt"`
			UpdatedAtSnake string `json:"updated_at"`
			CreatedAtCamel string `json:"createdAt"`
			CreatedAtSnake string `json:"created_at"`
			Customer       struct {
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
			EventAt: firstParsedTime(
				sub.UpdatedAtCamel, sub.UpdatedAtSnake,
				sub.CreatedAtCamel, sub.CreatedAtSnake,
			),
		}, nil
	}
	return nil, nil
}

// firstParsedTime returns the first value that parses as RFC3339, as a Unix
// timestamp, or 0 if none do. Paystack spells its timestamp fields
// inconsistently across payloads, so several candidates are tried in order.
func firstParsedTime(candidates ...string) int64 {
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, c); err == nil {
			return t.Unix()
		}
	}
	return 0
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

// IntervalForPlanCode reports the billing interval for one of our configured
// plan codes. The second return is false for any code that is not ours.
//
// This is the authority on interval — the client-supplied value is not
// trusted, since it drives current_period_end.
func (p *PaystackProvider) IntervalForPlanCode(code string) (string, bool) {
	switch {
	case code != "" && code == p.Plans.ProMonthly:
		return "month", true
	case code != "" && code == p.Plans.ProAnnual:
		return "year", true
	}
	return "", false
}

// PaystackTransaction is the subset of a verified transaction we act on.
type PaystackTransaction struct {
	ID        int64
	Status    string
	Amount    int // kobo
	Currency  string
	Reference string
	Email     string
	// CustomerCode is Paystack's CUS_ identifier.
	CustomerCode string
	// Plan carries the plan code and, when Paystack told us, its amount.
	Plan PaystackPlanInfo
}

// PaystackPlanInfo is the plan attached to a transaction.
type PaystackPlanInfo struct {
	Code string
	// Amount in kobo. Zero means Paystack did not tell us the plan's price
	// (it returned a bare plan code with no plan object), NOT that it is free.
	Amount int
	// AmountKnown distinguishes "the plan costs nothing" from "we do not know".
	AmountKnown bool
}

// paystackVerifyResponse mirrors GET /transaction/verify/{reference}.
//
// Plan is json.RawMessage because Paystack returns it as a plan code string on
// some transactions and as a full plan object on others — decoding it into a
// fixed shape aborts the whole parse mid-way, which is what originally made
// legitimate payments fail verification.
type paystackVerifyResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID        int64  `json:"id"`
		Status    string `json:"status"`
		Amount    int    `json:"amount"`
		Currency  string `json:"currency"`
		Reference string `json:"reference"`
		Customer  struct {
			CustomerCode string `json:"customer_code"`
			Email        string `json:"email"`
		} `json:"customer"`
		Plan       json.RawMessage `json:"plan"`
		PlanObject json.RawMessage `json:"plan_object"`
	} `json:"data"`
}

// ParsePaystackPlan extracts the plan code and amount from the two shapes
// Paystack uses. `plan` may be a bare code string or a full object;
// `plan_object` carries the object form when `plan` is a string.
func ParsePaystackPlan(plan, planObject json.RawMessage) (PaystackPlanInfo, bool) {
	var info PaystackPlanInfo

	type planShape struct {
		PlanCode string `json:"plan_code"`
		Amount   int    `json:"amount"`
		Interval string `json:"interval"`
	}

	// `plan` as a bare code string
	var asString string
	if len(plan) > 0 && json.Unmarshal(plan, &asString) == nil {
		info.Code = strings.TrimSpace(asString)
	}

	// `plan` as an object
	if info.Code == "" && len(plan) > 0 {
		var obj planShape
		if json.Unmarshal(plan, &obj) == nil && obj.PlanCode != "" {
			info.Code = obj.PlanCode
			info.Amount, info.AmountKnown = obj.Amount, true
		}
	}

	// `plan_object` fills in the amount when `plan` was only a code
	if len(planObject) > 0 && !info.AmountKnown {
		var obj planShape
		if json.Unmarshal(planObject, &obj) == nil && obj.PlanCode != "" {
			if info.Code == "" {
				info.Code = obj.PlanCode
			}
			if obj.PlanCode == info.Code {
				info.Amount, info.AmountKnown = obj.Amount, true
			}
		}
	}

	return info, info.Code != ""
}

// VerifyTransaction verifies a transaction reference with Paystack, retrying
// transient failures. Exhausting the retries is an error: a transaction that
// cannot be verified must never be treated as paid.
func (p *PaystackProvider) VerifyTransaction(
	ctx context.Context,
	reference string,
) (*PaystackTransaction, error) {
	if reference == "" {
		return nil, fmt.Errorf("paystack: empty reference")
	}

	var lastErr error
	delays := p.RetryDelays
	if delays == nil {
		delays = defaultVerifyDelays
	}

	for i, delay := range delays {
		if delay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET",
			p.baseURL()+"/transaction/verify/"+url.PathEscape(reference), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+p.SecretKey)

		resp, err := p.client().Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", i+1, err)
			continue
		}

		var out paystackVerifyResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()

		if decodeErr != nil {
			lastErr = fmt.Errorf("attempt %d: decode: %w", i+1, decodeErr)
			continue
		}

		if !out.Status || out.Data.Status != "success" {
			// Not transient in most cases, but Paystack can briefly report a
			// just-completed transaction as pending, which is why we retry.
			lastErr = fmt.Errorf("attempt %d: status=%v data.status=%q message=%q",
				i+1, out.Status, out.Data.Status, out.Message)
			continue
		}

		planInfo, ok := ParsePaystackPlan(out.Data.Plan, out.Data.PlanObject)
		if !ok {
			// No plan means this was a one-off charge, not a subscription.
			return nil, fmt.Errorf("paystack: transaction %s has no plan attached", reference)
		}

		return &PaystackTransaction{
			ID:           out.Data.ID,
			Status:       out.Data.Status,
			Amount:       out.Data.Amount,
			Currency:     out.Data.Currency,
			Reference:    out.Data.Reference,
			Email:        out.Data.Customer.Email,
			CustomerCode: out.Data.Customer.CustomerCode,
			Plan:         planInfo,
		}, nil
	}

	return nil, fmt.Errorf("paystack verify %s failed after %d attempts: %w",
		reference, len(delays), lastErr)
}
