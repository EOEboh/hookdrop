package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
