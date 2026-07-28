package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

	resp, err := p.client().Do(req)
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

// subscriptionEmailToken fetches the email_token Paystack requires to disable
// a subscription. It is only present on the subscription object, so it has to
// be read at cancel time.
func (p *PaystackProvider) subscriptionEmailToken(ctx context.Context, subCode string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		p.baseURL()+"/subscription/"+url.PathEscape(subCode), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)

	resp, err := p.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Status bool `json:"status"`
		Data   struct {
			EmailToken string `json:"email_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if !out.Status || out.Data.EmailToken == "" {
		return "", fmt.Errorf("paystack: no email_token for %s", subCode)
	}
	return out.Data.EmailToken, nil
}

func (p *PaystackProvider) CancelSubscription(ctx context.Context, subCode string) error {
	if subCode == "" {
		return fmt.Errorf("paystack: empty subscription code")
	}
	// Disabling needs a subscription code. A transaction reference cannot be
	// cancelled, and pretending otherwise reports success while the customer
	// keeps being billed.
	if !strings.HasPrefix(subCode, "SUB_") {
		return fmt.Errorf("paystack: %q is a transaction reference, not a subscription code", subCode)
	}

	// /subscription/disable rejects an empty token with
	// `"token" is not allowed to be empty`, so this is required, not optional.
	token, err := p.subscriptionEmailToken(ctx, subCode)
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]string{"code": subCode, "token": token})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		p.baseURL()+"/subscription/disable",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("paystack disable subscription %d: %s", resp.StatusCode, msg)
	}
	// Paystack answers 200 with status:false for logical failures, so the
	// status code alone is not enough.
	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(msg, &out); err == nil && !out.Status {
		return fmt.Errorf("paystack disable subscription failed: %s", out.Message)
	}
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
			// plan is an OBJECT on subscription events, though it is a bare
			// code string on transaction payloads. Decoding it as a string
			// fails the whole parse, which took subscription.create with it.
			Plan            json.RawMessage `json:"plan"`
			EmailToken      string          `json:"email_token"`
			NextPaymentDate string          `json:"next_payment_date"`
			// Transaction metadata, where our user_id is passed at checkout.
			// Paystack returns this as an object, an empty string, or the
			// integer 0, so it cannot be decoded into a fixed shape.
			Metadata json.RawMessage `json:"metadata"`
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

		// Handles both shapes; on subscription events this is the object form.
		planInfo, _ := ParsePaystackPlan(sub.Plan, nil)
		// Without this the upsert wrote an empty interval over the stored one.
		interval, _ := p.IntervalForPlanCode(planInfo.Code)

		return &WebhookEvent{
			Type:           normalisePaystackEvent(event.Event),
			CustomerID:     sub.Customer.CustomerCode,
			SubscriptionID: sub.SubscriptionCode,
			Plan:           p.planFromCode(planInfo.Code),
			Status:         normalisePaystackStatus(sub.Status),
			Currency:       "ngn",
			Interval:       interval,
			PeriodEnd:      periodEnd,
			// subscription.not_renew means the customer cancelled but keeps
			// access until the period ends. Without this the upsert wrote
			// cancel_at_period_end=false and silently undid the cancellation.
			CancelAtEnd: event.Event == "subscription.not_renew" ||
				event.Event == "subscription.disable",
			// Transaction metadata first; customer metadata is a different
			// field and is usually null.
			UserID: firstNonEmpty(
				paystackMetadataUserID(sub.Metadata),
				sub.Customer.Metadata.UserID,
			),
			EventAt: firstParsedTime(
				sub.UpdatedAtCamel, sub.UpdatedAtSnake,
				sub.CreatedAtCamel, sub.CreatedAtSnake,
			),
		}, nil

	case "charge.success":
		// Subscription renewals arrive as charge.success — Paystack sends no
		// subscription.* event when a recurring payment goes through. Without
		// handling it, current_period_end is written once at signup and then
		// silently goes stale.
		var charge struct {
			Reference   string          `json:"reference"`
			Amount      int             `json:"amount"`
			Currency    string          `json:"currency"`
			Status      string          `json:"status"`
			Plan        json.RawMessage `json:"plan"`        // code string or object
			PlanObject  json.RawMessage `json:"plan_object"` // object form
			Metadata    json.RawMessage `json:"metadata"`
			PaidAtSnake string          `json:"paid_at"`
			PaidAtCamel string          `json:"paidAt"`
			CreatedAt   string          `json:"createdAt"`
			Customer    struct {
				CustomerCode string `json:"customer_code"`
				Email        string `json:"email"`
			} `json:"customer"`
		}
		if err := json.Unmarshal(event.Data, &charge); err != nil {
			return nil, err
		}

		planInfo, ok := ParsePaystackPlan(charge.Plan, charge.PlanObject)
		if !ok {
			// A one-off charge, not a subscription payment. Nothing to apply.
			return nil, nil
		}
		interval, ours := p.IntervalForPlanCode(planInfo.Code)
		if !ours {
			// Someone else's plan on this integration.
			return nil, nil
		}

		paidAt := firstParsedTime(charge.PaidAtSnake, charge.PaidAtCamel, charge.CreatedAt)
		if paidAt == 0 {
			paidAt = time.Now().Unix()
		}

		return &WebhookEvent{
			Type:       "subscription.updated",
			CustomerID: charge.Customer.CustomerCode,
			// charge.success names no subscription, so this cannot set
			// provider_sub_id — the user resolves by customer code.
			Plan:     "pro",
			Status:   "active",
			Currency: "ngn",
			Interval: interval,
			// Paystack does not report the next payment date here, so the
			// period is advanced one interval from the payment.
			PeriodEnd: time.Unix(paidAt, 0).Add(PaystackBillingPeriod(interval)).Unix(),
			UserID:    paystackMetadataUserID(charge.Metadata),
			EventAt:   paidAt,
		}, nil

	case "invoice.payment_failed", "invoice.update":
		// A renewal charge failed. Paystack does not retry, so this is the
		// only notice we get that the subscription has stopped paying.
		//
		// This exists to make the downgrade prompt. It is NOT what guarantees
		// correctness — IsActive expires access once the period lapses, which
		// holds whether or not this event ever arrives or is shaped as
		// expected. Parsed defensively for that reason: only fields present in
		// every Paystack payload observed so far are relied on.
		var inv struct {
			Status   string          `json:"status"`
			Paid     *bool           `json:"paid"`
			Plan     json.RawMessage `json:"plan"`
			Sub      json.RawMessage `json:"subscription"`
			Created  string          `json:"createdAt"`
			Updated  string          `json:"updatedAt"`
			Customer struct {
				CustomerCode string `json:"customer_code"`
			} `json:"customer"`
		}
		if err := json.Unmarshal(event.Data, &inv); err != nil {
			return nil, err
		}

		// invoice.update reports the final outcome, success included. Only a
		// failure is actionable here; a success is already covered by
		// charge.success, which carries the payment date.
		failed := event.Event == "invoice.payment_failed" ||
			(inv.Paid != nil && !*inv.Paid) ||
			strings.EqualFold(inv.Status, "failed")
		if !failed {
			return nil, nil
		}

		planInfo, ok := ParsePaystackPlan(inv.Plan, nil)
		if !ok {
			return nil, nil
		}
		if _, ours := p.IntervalForPlanCode(planInfo.Code); !ours {
			return nil, nil
		}

		return &WebhookEvent{
			Type:       "subscription.updated",
			CustomerID: inv.Customer.CustomerCode,
			// The subscription code is nested and its shape is unconfirmed, so
			// it is deliberately not read — processWebhookEvent preserves the
			// stored one and the user resolves by customer code.
			Plan:     "pro",
			Status:   "past_due",
			Currency: "ngn",
			// PeriodEnd is deliberately left at zero: a failed charge moves no
			// period, and processWebhookEvent keeps the stored value.
			EventAt: firstParsedTime(inv.Updated, inv.Created),
		}, nil
	}
	return nil, nil
}

// PaystackBillingPeriod is one billing period for an interval. Paystack
// reports no next-payment date on either the verify response or
// charge.success, so the period end is derived from this.
func PaystackBillingPeriod(interval string) time.Duration {
	if interval == "year" {
		return 365 * 24 * time.Hour
	}
	return 30 * 24 * time.Hour
}

// paystackMetadataUserID extracts user_id from Paystack transaction metadata.
//
// Paystack returns metadata as an object when set, but as an empty string or
// the integer 0 when it is not, so decoding it into a struct fails outright on
// those transactions. Returns "" for anything that is not an object.
func paystackMetadataUserID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var obj struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	return strings.TrimSpace(obj.UserID)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
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

// PaystackCustomerSubscription is a subscription as reported on the customer
// object, which is the only route from a customer *code* to a subscription.
//
// GET /subscription?customer= expects the numeric customer id, not the code,
// and silently returns zero rows when given a code — which is what made the
// old cancel path fail without complaining.
type PaystackCustomerSubscription struct {
	SubscriptionCode string
	Status           string
	NextPaymentAt    int64 // Unix; 0 when Paystack reported none
}

// LookupCustomerSubscription finds the subscription belonging to a customer
// code. It prefers an active subscription, falling back to the most recently
// listed one so a cancelled subscription can still be reconciled.
func (p *PaystackProvider) LookupCustomerSubscription(
	ctx context.Context,
	customerCode string,
) (*PaystackCustomerSubscription, error) {
	if customerCode == "" {
		return nil, fmt.Errorf("paystack: empty customer code")
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		p.baseURL()+"/customer/"+url.PathEscape(customerCode), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)

	resp, err := p.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			ID            int64 `json:"id"`
			Subscriptions []struct {
				SubscriptionCode string `json:"subscription_code"`
				Status           string `json:"status"`
				NextPaymentDate  string `json:"next_payment_date"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack customer %s: %s", customerCode, out.Message)
	}
	if len(out.Data.Subscriptions) == 0 {
		return nil, fmt.Errorf("paystack customer %s has no subscriptions", customerCode)
	}

	chosen := out.Data.Subscriptions[0]
	for _, s := range out.Data.Subscriptions {
		if s.Status == "active" {
			chosen = s
			break
		}
	}

	return &PaystackCustomerSubscription{
		SubscriptionCode: chosen.SubscriptionCode,
		Status:           normalisePaystackStatus(chosen.Status),
		NextPaymentAt:    firstParsedTime(chosen.NextPaymentDate),
	}, nil
}
