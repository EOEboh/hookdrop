package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testSecret512 = "paystack-test-secret"

func sign512(t *testing.T, payload string) string {
	t.Helper()
	mac := hmac.New(sha512.New, []byte(testSecret512))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestParsePaystackPlan(t *testing.T) {
	cases := []struct {
		name        string
		plan        string
		planObject  string
		wantCode    string
		wantAmount  int
		wantKnown   bool
		wantSuccess bool
	}{
		{
			// Paystack returns the plan as a bare code string on some
			// transactions. This is the shape that used to abort the decode.
			name:        "plan as bare code string",
			plan:        `"PLN_monthly"`,
			wantCode:    "PLN_monthly",
			wantKnown:   false,
			wantSuccess: true,
		},
		{
			name:        "plan as object",
			plan:        `{"plan_code":"PLN_annual","amount":3360000,"interval":"annually"}`,
			wantCode:    "PLN_annual",
			wantAmount:  3360000,
			wantKnown:   true,
			wantSuccess: true,
		},
		{
			name:        "code string plus plan_object supplies the amount",
			plan:        `"PLN_monthly"`,
			planObject:  `{"plan_code":"PLN_monthly","amount":350000,"interval":"monthly"}`,
			wantCode:    "PLN_monthly",
			wantAmount:  350000,
			wantKnown:   true,
			wantSuccess: true,
		},
		{
			name:        "plan_object only",
			planObject:  `{"plan_code":"PLN_monthly","amount":350000}`,
			wantCode:    "PLN_monthly",
			wantAmount:  350000,
			wantKnown:   true,
			wantSuccess: true,
		},
		{
			// A plain charge with no subscription plan attached.
			name:        "no plan",
			plan:        `""`,
			wantSuccess: false,
		},
		{
			name:        "null plan",
			plan:        `null`,
			wantSuccess: false,
		},
		{
			// plan_object for a DIFFERENT plan must not donate its amount.
			name:        "mismatched plan_object does not supply amount",
			plan:        `"PLN_monthly"`,
			planObject:  `{"plan_code":"PLN_other","amount":999}`,
			wantCode:    "PLN_monthly",
			wantKnown:   false,
			wantSuccess: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			info, ok := ParsePaystackPlan(
				json.RawMessage(c.plan), json.RawMessage(c.planObject))
			if ok != c.wantSuccess {
				t.Fatalf("ok = %t, want %t (info=%+v)", ok, c.wantSuccess, info)
			}
			if !c.wantSuccess {
				return
			}
			if info.Code != c.wantCode {
				t.Errorf("Code = %q, want %q", info.Code, c.wantCode)
			}
			if info.AmountKnown != c.wantKnown {
				t.Errorf("AmountKnown = %t, want %t", info.AmountKnown, c.wantKnown)
			}
			if c.wantKnown && info.Amount != c.wantAmount {
				t.Errorf("Amount = %d, want %d", info.Amount, c.wantAmount)
			}
		})
	}
}

func TestIntervalForPlanCode(t *testing.T) {
	p := NewPaystackProvider("sk", "wh", PaystackPlans{
		ProMonthly: "PLN_monthly",
		ProAnnual:  "PLN_annual",
	})

	if iv, ok := p.IntervalForPlanCode("PLN_monthly"); !ok || iv != "month" {
		t.Errorf("monthly = (%q, %t), want (month, true)", iv, ok)
	}
	if iv, ok := p.IntervalForPlanCode("PLN_annual"); !ok || iv != "year" {
		t.Errorf("annual = (%q, %t), want (year, true)", iv, ok)
	}
	if _, ok := p.IntervalForPlanCode("PLN_someone_elses"); ok {
		t.Error("a foreign plan code was accepted as ours")
	}
	// An unconfigured provider must not match the empty code.
	empty := NewPaystackProvider("sk", "wh", PaystackPlans{})
	if _, ok := empty.IntervalForPlanCode(""); ok {
		t.Error("empty plan code matched an unconfigured plan")
	}
}

// paystackStub serves a canned verify response and counts requests.
func paystackStub(t *testing.T, body string, status int) (*PaystackProvider, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	p := NewPaystackProvider("sk_test", "wh", PaystackPlans{
		ProMonthly: "PLN_monthly",
		ProAnnual:  "PLN_annual",
	})
	p.BaseURL = srv.URL
	p.RetryDelays = []time.Duration{0, 0, 0} // same 3 attempts, no waiting
	return p, &calls
}

const okVerifyBody = `{
  "status": true,
  "message": "Verification successful",
  "data": {
    "id": 4099260516,
    "status": "success",
    "amount": 350000,
    "currency": "NGN",
    "reference": "T_abc123",
    "customer": {"customer_code": "CUS_xyz", "email": "buyer@example.com"},
    "plan": "PLN_monthly",
    "plan_object": {"plan_code": "PLN_monthly", "amount": 350000, "interval": "monthly"}
  }
}`

func TestVerifyTransaction_Success(t *testing.T) {
	p, calls := paystackStub(t, okVerifyBody, 200)

	tx, err := p.VerifyTransaction(context.Background(), "T_abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *calls != 1 {
		t.Errorf("made %d requests, want 1 — a success must not retry", *calls)
	}
	if tx.Amount != 350000 || tx.Currency != "NGN" {
		t.Errorf("amount/currency = %d/%s", tx.Amount, tx.Currency)
	}
	if tx.Email != "buyer@example.com" || tx.CustomerCode != "CUS_xyz" {
		t.Errorf("customer = %s / %s", tx.Email, tx.CustomerCode)
	}
	if tx.Plan.Code != "PLN_monthly" || !tx.Plan.AmountKnown || tx.Plan.Amount != 350000 {
		t.Errorf("plan = %+v", tx.Plan)
	}
}

// The whole point of the change: a transaction that will not verify must be an
// error, never a pass-through that grants Pro.
func TestVerifyTransaction_FailuresAreErrors(t *testing.T) {
	cases := map[string]string{
		"status false": `{"status": false, "message": "Invalid key", "data": {}}`,
		"not success":  `{"status": true, "data": {"status": "abandoned", "amount": 350000, "plan": "PLN_monthly"}}`,
		"pending":      `{"status": true, "data": {"status": "pending", "amount": 350000, "plan": "PLN_monthly"}}`,
		"garbage":      `not json at all`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			p, calls := paystackStub(t, body, 200)
			tx, err := p.VerifyTransaction(context.Background(), "T_bad")
			if err == nil {
				t.Fatalf("expected an error, got tx=%+v — this is the free-Pro escalation", tx)
			}
			if *calls != 3 {
				t.Errorf("made %d attempts, want 3 (retries preserved)", *calls)
			}
		})
	}
}

// A transaction with no plan is a one-off charge and cannot grant a subscription.
func TestVerifyTransaction_NoPlanRejected(t *testing.T) {
	body := `{"status": true, "data": {"status":"success","amount":100,"currency":"NGN",
	  "customer":{"customer_code":"CUS_x","email":"a@b.c"},"plan":""}}`
	p, _ := paystackStub(t, body, 200)

	if _, err := p.VerifyTransaction(context.Background(), "T_noplan"); err == nil {
		t.Fatal("a transaction with no plan was accepted")
	} else if !strings.Contains(err.Error(), "no plan") {
		t.Errorf("error = %v, want it to mention the missing plan", err)
	}
}

func TestVerifyTransaction_EmptyReference(t *testing.T) {
	p, calls := paystackStub(t, okVerifyBody, 200)
	if _, err := p.VerifyTransaction(context.Background(), ""); err == nil {
		t.Fatal("empty reference accepted")
	}
	if *calls != 0 {
		t.Errorf("made %d requests for an empty reference, want 0", *calls)
	}
}

func TestPaystackMetadataUserID(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want string
	}{
		// Paystack sends the integer 0 when no metadata was set. Decoding that
		// into a struct fails, which must not take the whole event with it.
		"integer zero":     {`0`, ""},
		"empty string":     {`""`, ""},
		"null":             {`null`, ""},
		"absent":           {``, ""},
		"object":           {`{"user_id":"user-123"}`, "user-123"},
		"object padded":    {`{"user_id":"  user-123  "}`, "user-123"},
		"object other key": {`{"cancel_url":"https://x"}`, ""},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := paystackMetadataUserID(json.RawMessage(c.raw)); got != c.want {
				t.Errorf("paystackMetadataUserID(%s) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

// The user_id lives in transaction metadata; customer.metadata is a separate
// field that is usually null.
func TestHandleWebhook_ReadsUserIDFromTransactionMetadata(t *testing.T) {
	p := NewPaystackProvider("sk", testSecret512, PaystackPlans{ProMonthly: "PLN_monthly"})

	payload := `{"event":"subscription.create","data":{
      "subscription_code":"SUB_abc","status":"active","plan":"PLN_monthly",
      "next_payment_date":"2026-08-27T00:00:00.000Z",
      "metadata":{"user_id":"user-from-transaction"},
      "createdAt":"2026-07-27T00:00:00.000Z",
      "customer":{"customer_code":"CUS_1","email":"a@b.c","metadata":null}}}`

	ev, err := p.HandleWebhook([]byte(payload), sign512(t, payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.UserID != "user-from-transaction" {
		t.Errorf("UserID = %q, want user-from-transaction", ev.UserID)
	}
	if ev.SubscriptionID != "SUB_abc" {
		t.Errorf("SubscriptionID = %q, want SUB_abc", ev.SubscriptionID)
	}
	if ev.EventAt == 0 {
		t.Error("EventAt = 0, want the parsed createdAt")
	}
}

// metadata:0 must not abort the parse — the event still has to resolve by
// customer code downstream.
func TestHandleWebhook_SurvivesIntegerMetadata(t *testing.T) {
	p := NewPaystackProvider("sk", testSecret512, PaystackPlans{ProMonthly: "PLN_monthly"})

	payload := `{"event":"subscription.create","data":{
      "subscription_code":"SUB_abc","status":"active","plan":"PLN_monthly",
      "metadata":0,
      "customer":{"customer_code":"CUS_1","email":"a@b.c","metadata":null}}}`

	ev, err := p.HandleWebhook([]byte(payload), sign512(t, payload))
	if err != nil {
		t.Fatalf("integer metadata aborted the parse: %v", err)
	}
	if ev.UserID != "" {
		t.Errorf("UserID = %q, want empty", ev.UserID)
	}
	if ev.CustomerID != "CUS_1" {
		t.Errorf("CustomerID = %q, want CUS_1 — the fallback key", ev.CustomerID)
	}
}
