package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EOEboh/hookdrop/internal/billing"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
)

const (
	testPlanMonthly = "PLN_monthly"
	testPlanAnnual  = "PLN_annual"
	monthlyKobo     = 350000
	annualKobo      = 3360000
)

// verifyBody builds a Paystack verify response.
func verifyBody(status, email, planCode string, amount, planAmount int) string {
	return fmt.Sprintf(`{
      "status": true,
      "data": {
        "id": 1,
        "status": %q,
        "amount": %d,
        "currency": "NGN",
        "reference": "T_ref",
        "customer": {"customer_code": "CUS_1", "email": %q},
        "plan": %q,
        "plan_object": {"plan_code": %q, "amount": %d}
      }
    }`, status, amount, email, planCode, planCode, planAmount)
}

// newBillingTestHandler wires a BillingHandler against a temp DB and a stubbed
// Paystack API, and returns the handler plus the authenticated user.
func newBillingTestHandler(t *testing.T, paystackResponse string) (*BillingHandler, *models.User) {
	t.Helper()

	st, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user, err := st.GetOrCreateUser("buyer@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(paystackResponse))
	}))
	t.Cleanup(srv.Close)

	ps := billing.NewPaystackProvider("sk_test", "wh", billing.PaystackPlans{
		ProMonthly: testPlanMonthly,
		ProAnnual:  testPlanAnnual,
	})
	ps.BaseURL = srv.URL
	ps.RetryDelays = []time.Duration{0, 0, 0}

	return &BillingHandler{
		Store:    st,
		Paystack: ps,
		AppURL:   "http://localhost:5173",
	}, user
}

func verifyRequest(userID, email, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/billing/verify-paystack",
		strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, &middleware.UserContext{
		ID:         userID,
		Email:      email,
		AuthMethod: middleware.AuthMethodJWT,
	})
	return req.WithContext(ctx)
}

func TestVerifyPaystack_GrantsProOnValidTransaction(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanMonthly, monthlyKobo, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sub, err := h.Store.GetSubscription(user.ID)
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	if sub.Plan != "pro" || sub.Status != "active" {
		t.Errorf("plan/status = %s/%s, want pro/active", sub.Plan, sub.Status)
	}
	if sub.Provider != "paystack" {
		t.Errorf("provider = %q, want paystack", sub.Provider)
	}
	if sub.Interval != "month" {
		t.Errorf("interval = %q, want month", sub.Interval)
	}
}

// The core escalation: an unverifiable reference must not grant anything.
func TestVerifyPaystack_RejectsUnverifiableTransaction(t *testing.T) {
	cases := map[string]string{
		"paystack says status false": `{"status": false, "message": "Invalid reference", "data": {}}`,
		"transaction abandoned":      verifyBody("abandoned", "buyer@example.com", testPlanMonthly, monthlyKobo, monthlyKobo),
		"fabricated reference":       `{"status": false, "message": "Transaction reference not found", "data": {}}`,
	}

	for name, resp := range cases {
		t.Run(name, func(t *testing.T) {
			h, user := newBillingTestHandler(t, resp)

			rec := httptest.NewRecorder()
			h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
				`{"reference":"T_fake","plan":"pro","interval":"month"}`))

			if rec.Code == http.StatusOK {
				t.Fatalf("got 200 — unverified transaction granted Pro")
			}

			sub, _ := h.Store.GetSubscription(user.ID)
			if sub.Plan != "free" {
				t.Errorf("plan = %q, want free — nothing may be written on a failed verify", sub.Plan)
			}
		})
	}
}

// Binding to the caller is what stops replaying someone else's valid reference.
func TestVerifyPaystack_RejectsMismatchedEmail(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "someone.else@example.com", testPlanMonthly, monthlyKobo, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free", sub.Plan)
	}
}

// Case and whitespace differences are not impersonation.
func TestVerifyPaystack_EmailComparisonIsForgiving(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "  BUYER@Example.COM ", testPlanMonthly, monthlyKobo, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 — a case/whitespace difference locked out a real customer: %s",
			rec.Code, rec.Body.String())
	}
}

// A reference already redeemed by one account cannot be redeemed by another.
//
// The email check (6) runs first and catches the everyday version of this,
// since email is the account identity here. This test drives the replay check
// (7) in isolation as the backstop it is: the transaction's email matches the
// caller, but the reference is already owned by a different user ID.
func TestVerifyPaystack_RejectsReplayByAnotherUser(t *testing.T) {
	// The stub reports the *claimant's* email, so check 6 passes and the
	// rejection can only come from the replay check.
	h, claimant := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanMonthly, monthlyKobo, monthlyKobo))

	owner, err := h.Store.GetOrCreateUser("original.owner@example.com")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if owner.ID == claimant.ID {
		t.Fatal("owner and claimant must be distinct users")
	}

	// The reference is already redeemed by someone else.
	if err := h.Store.UpsertSubscription(&models.Subscription{
		UserID:        owner.ID,
		Plan:          "pro",
		Provider:      "paystack",
		ProviderSubID: "T_ref",
		Status:        "active",
		Currency:      "ngn",
		Interval:      "month",
		CreatedAt:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed owner subscription: %v", err)
	}

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(claimant.ID, claimant.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d, want 409 — a redeemed reference was accepted again: %s",
			rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(claimant.ID)
	if sub.Plan != "free" {
		t.Errorf("claimant plan = %q, want free", sub.Plan)
	}
	// The rightful owner keeps their subscription.
	ownerSub, _ := h.Store.GetSubscription(owner.ID)
	if ownerSub.Plan != "pro" {
		t.Errorf("owner plan = %q, want pro — the replay must not disturb them", ownerSub.Plan)
	}
}

// A double-submit from the legitimate buyer must not lock them out.
func TestVerifyPaystack_SameUserRetryIsIdempotent(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanMonthly, monthlyKobo, monthlyKobo))

	for i := 1; i <= 2; i++ {
		rec := httptest.NewRecorder()
		h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
			`{"reference":"T_ref","plan":"pro","interval":"month"}`))
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: got %d, want 200: %s", i, rec.Code, rec.Body.String())
		}
	}
}

func TestVerifyPaystack_RejectsForeignPlanCode(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", "PLN_not_ours", monthlyKobo, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

// A ₦100 charge must not buy the ₦33,600 annual plan.
func TestVerifyPaystack_RejectsAmountBelowPlanPrice(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanAnnual, 10000, annualKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"year"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free", sub.Plan)
	}
}

// The trial branch is dormant today but must not be broken by the amount check.
func TestVerifyPaystack_ZeroAmountTrialStillSucceeds(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanMonthly, 0, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 — the amount check rejected a genuine ₦0 trial: %s",
			rec.Code, rec.Body.String())
	}

	var resp struct {
		Status   string `json:"status"`
		IsTrial  bool   `json:"is_trial"`
		TrialEnd string `json:"trial_end"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.IsTrial || resp.Status != "trialing" {
		t.Errorf("is_trial=%t status=%q, want true/trialing", resp.IsTrial, resp.Status)
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.TrialEnd == nil {
		t.Error("trial_end is nil on a trial subscription")
	}
}

// The client asks for a year while paying for a month. The plan code decides.
func TestVerifyPaystack_IgnoresClientSuppliedInterval(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanMonthly, monthlyKobo, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"year"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Interval != "month" {
		t.Errorf("interval = %q, want month — the client's claim was trusted", sub.Interval)
	}
	// A month's access, not a year's.
	if sub.CurrentPeriodEnd == nil {
		t.Fatal("current_period_end is nil")
	}
	if got := time.Until(*sub.CurrentPeriodEnd); got > 60*24*time.Hour {
		t.Errorf("current_period_end is %v away, want ~30 days", got)
	}
}

func TestVerifyPaystack_RejectsNonNGNCurrency(t *testing.T) {
	body := fmt.Sprintf(`{"status":true,"data":{"id":1,"status":"success","amount":%d,
	  "currency":"USD","reference":"T_ref",
	  "customer":{"customer_code":"CUS_1","email":"buyer@example.com"},
	  "plan":%q,"plan_object":{"plan_code":%q,"amount":%d}}}`,
		monthlyKobo, testPlanMonthly, testPlanMonthly, monthlyKobo)

	h, user := newBillingTestHandler(t, body)

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body.String())
	}
}
