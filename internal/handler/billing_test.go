package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	return verifyBodyFrom(status, email, planCode, amount, planAmount, "NG")
}

// verifyBodyReusable builds a response for a given payment channel.
// reusable=false is what a Paystack bank transfer produces: the plan is
// charged once and no subscription is created.
func verifyBodyReusable(planCode string, amount, planAmount int, channel string, reusable bool) string {
	return fmt.Sprintf(`{
      "status": true,
      "data": {
        "id": 1, "status": "success", "amount": %d, "currency": "NGN",
        "reference": "T_ref", "ip_address": "102.90.1.1",
        "authorization": {"country_code": "NG", "channel": %q, "reusable": %t},
        "customer": {"customer_code": "CUS_1", "email": "buyer@example.com"},
        "plan": %q, "plan_object": {"plan_code": %q, "amount": %d}
      }
    }`, amount, channel, reusable, planCode, planCode, planAmount)
}

// verifyBodyFrom lets a test set the card's issuing country.
func verifyBodyFrom(status, email, planCode string, amount, planAmount int, cardCountry string) string {
	return fmt.Sprintf(`{
      "status": true,
      "data": {
        "id": 1,
        "status": %q,
        "amount": %d,
        "currency": "NGN",
        "reference": "T_ref",
        "ip_address": "102.90.1.1",
        "authorization": {"country_code": %q},
        "customer": {"customer_code": "CUS_1", "email": %q},
        "plan": %q,
        "plan_object": {"plan_code": %q, "amount": %d}
      }
    }`, status, amount, cardCountry, email, planCode, planCode, planAmount)
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

// Paystack has no native free trial, so a ₦0 charge against a priced plan is
// never legitimate — it used to be treated as a trial and granted Pro.
func TestVerifyPaystack_RejectsZeroAmountCharge(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanMonthly, 0, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free", sub.Plan)
	}
}

// A paid Paystack subscription is active immediately, never trialing.
func TestVerifyPaystack_NeverProducesATrial(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBody("success", "buyer@example.com", testPlanMonthly, monthlyKobo, monthlyKobo))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status   string      `json:"status"`
		IsTrial  bool        `json:"is_trial"`
		TrialEnd interface{} `json:"trial_end"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.IsTrial || resp.Status != "active" {
		t.Errorf("is_trial=%t status=%q, want false/active", resp.IsTrial, resp.Status)
	}
	if resp.TrialEnd != nil {
		t.Errorf("trial_end = %v, want null", resp.TrialEnd)
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.TrialEnd != nil {
		t.Errorf("stored trial_end = %v, want nil — Paystack grants no trial", sub.TrialEnd)
	}
	if sub.Status != "active" {
		t.Errorf("stored status = %q, want active", sub.Status)
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

// ── Webhook dedupe and ordering ──────────────────────────────────────────

const lsWebhookSecret = "ls-test-secret"

func newWebhookTestHandler(t *testing.T) (*BillingHandler, *models.User) {
	t.Helper()

	st, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user, err := st.GetOrCreateUser("subscriber@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ls := billing.NewLemonSqueezyProvider(
		"key", lsWebhookSecret, "99999",
		billing.LemonSqueezyVariants{ProMonthly: "111111", ProAnnual: "222222"},
		true,
	)

	return &BillingHandler{Store: st, LemonSqueezy: ls, AppURL: "http://localhost:5173"}, user
}

func lsSubscriptionPayload(userID, status, updatedAt string) string {
	return fmt.Sprintf(`{
      "meta": {"event_name":"subscription_updated","custom_data":{"user_id":%q}},
      "data": {"id":"878123","attributes":{
        "customer_id":4210987,"variant_id":111111,"status":%q,"cancelled":false,
        "trial_ends_at":null,"first_subscription_item":null,
        "renews_at":"2026-09-27T12:00:00.000000Z","ends_at":null,
        "updated_at":%q}}}`, userID, status, updatedAt)
}

// postLSWebhook signs and delivers a payload, returning the recorder and the
// payload's dedupe key.
func postLSWebhook(t *testing.T, h *BillingHandler, payload string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(lsWebhookSecret))
	mac.Write([]byte(payload))
	sum := sha256.Sum256([]byte(payload))

	req := httptest.NewRequest(http.MethodPost, "/billing/webhook/lemonsqueezy",
		strings.NewReader(payload))
	req.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Event-Name", "subscription_updated")

	rec := httptest.NewRecorder()
	h.LemonSqueezyWebhook(rec, req)
	return rec, hex.EncodeToString(sum[:])
}

// requireEvent asserts a delivery was recorded, and returns it.
func requireEvent(t *testing.T, h *BillingHandler, key string) *models.BillingEvent {
	t.Helper()
	ev, err := h.Store.GetBillingEventByKey(key)
	if err != nil {
		t.Fatalf("lookup event: %v", err)
	}
	if ev == nil {
		t.Fatal("delivery was not recorded")
	}
	return ev
}

// A retried delivery is byte-identical, so it must be absorbed, not reapplied.
func TestWebhook_DuplicateDeliveryProcessedOnce(t *testing.T) {
	h, user := newWebhookTestHandler(t)
	payload := lsSubscriptionPayload(user.ID, "active", "2026-07-27T12:00:00.000000Z")

	var key string
	for i := 1; i <= 3; i++ {
		rec, k := postLSWebhook(t, h, payload)
		if rec.Code != http.StatusOK {
			t.Fatalf("delivery %d: got %d, want 200: %s", i, rec.Code, rec.Body.String())
		}
		key = k
	}

	// One row for the three deliveries: re-recording the key still collides.
	requireEvent(t, h, key)
	if err := h.Store.RecordBillingEvent(&models.BillingEvent{
		Provider: "lemonsqueezy", EventType: "x", Payload: "{}", EventKey: key,
	}); !errors.Is(err, store.ErrDuplicateBillingEvent) {
		t.Errorf("re-record err = %v, want ErrDuplicateBillingEvent", err)
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "pro" {
		t.Errorf("plan = %q, want pro", sub.Plan)
	}
}

// The scenario the ordering check exists for: an expiry lands, then a retried
// earlier update arrives. It must not put the customer back on Pro.
func TestWebhook_StaleDeliveryDoesNotResurrectPro(t *testing.T) {
	h, user := newWebhookTestHandler(t)

	// Newest event first: the subscription has expired.
	expired := lsSubscriptionPayload(user.ID, "expired", "2026-07-27T12:00:05.000000Z")
	if rec, _ := postLSWebhook(t, h, expired); rec.Code != http.StatusOK {
		t.Fatalf("expired delivery: got %d: %s", rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Fatalf("after expiry plan = %q, want free", sub.Plan)
	}

	// Now a stale active update, retried from before the expiry.
	stale := lsSubscriptionPayload(user.ID, "active", "2026-07-27T12:00:00.000000Z")
	if rec, _ := postLSWebhook(t, h, stale); rec.Code != http.StatusOK {
		t.Fatalf("stale delivery: got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sub, _ = h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free — a stale delivery resurrected Pro", sub.Plan)
	}
	if sub.Status != "canceled" {
		t.Errorf("status = %q, want canceled", sub.Status)
	}
}

// An expired subscription must not be left as plan=pro with status=canceled.
func TestWebhook_ExpiredStatusDropsPlanToFree(t *testing.T) {
	h, user := newWebhookTestHandler(t)

	payload := lsSubscriptionPayload(user.ID, "expired", "2026-07-27T12:00:00.000000Z")
	if rec, _ := postLSWebhook(t, h, payload); rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free — subscription_updated carrying status=expired left the row inconsistent", sub.Plan)
	}
}

// Unverified payloads must never reach the audit trail.
func TestWebhook_BadSignatureIsNotRecorded(t *testing.T) {
	h, user := newWebhookTestHandler(t)
	payload := lsSubscriptionPayload(user.ID, "active", "2026-07-27T12:00:00.000000Z")

	req := httptest.NewRequest(http.MethodPost, "/billing/webhook/lemonsqueezy",
		strings.NewReader(payload))
	req.Header.Set("X-Signature", "deadbeef")
	rec := httptest.NewRecorder()
	h.LemonSqueezyWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}

	sum := sha256.Sum256([]byte(payload))
	ev, err := h.Store.GetBillingEventByKey(hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if ev != nil {
		t.Error("an unverified payload was written to the audit trail")
	}
}

// Ignored event types are still recorded, for the audit trail.
func TestWebhook_IgnoredEventRecordedButNotApplied(t *testing.T) {
	h, user := newWebhookTestHandler(t)

	payload := fmt.Sprintf(`{
      "meta":{"event_name":"order_created","custom_data":{"user_id":%q}},
      "data":{"type":"orders","id":"5551234","attributes":{"status":"paid"}}}`, user.ID)

	rec, key := postLSWebhook(t, h, payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	ev := requireEvent(t, h, key)
	if ev.Processed {
		t.Error("an ignored event was marked processed")
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free — order_created must not touch the subscription", sub.Plan)
	}
}

// ── Paystack renewals ────────────────────────────────────────────────────

const paystackSecret = "paystack-hook-secret"

func newPaystackWebhookHandler(t *testing.T) (*BillingHandler, *models.User) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user, err := st.GetOrCreateUser("ngn@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ps := billing.NewPaystackProvider("sk", paystackSecret, billing.PaystackPlans{
		ProMonthly: testPlanMonthly, ProAnnual: testPlanAnnual,
	})
	return &BillingHandler{Store: st, Paystack: ps, AppURL: "http://localhost:5173"}, user
}

func postPaystackWebhook(t *testing.T, h *BillingHandler, payload string) *httptest.ResponseRecorder {
	t.Helper()
	mac := hmac.New(sha512.New, []byte(paystackSecret))
	mac.Write([]byte(payload))
	req := httptest.NewRequest(http.MethodPost, "/billing/webhook/paystack",
		strings.NewReader(payload))
	req.Header.Set("X-Paystack-Signature", hex.EncodeToString(mac.Sum(nil)))
	rec := httptest.NewRecorder()
	h.PaystackWebhook(rec, req)
	return rec
}

// A renewal must move current_period_end forward and must not blank the
// stored subscription code, which charge.success does not carry.
func TestPaystackRenewalAdvancesPeriodWithoutLosingSubID(t *testing.T) {
	h, user := newPaystackWebhookHandler(t)

	original := time.Now().UTC().Add(2 * 24 * time.Hour)
	if err := h.Store.UpsertSubscription(&models.Subscription{
		UserID:             user.ID,
		Plan:               "pro",
		Provider:           "paystack",
		ProviderCustomerID: "CUS_renew",
		ProviderSubID:      "SUB_original",
		Status:             "active",
		CurrentPeriodEnd:   &original,
		Currency:           "ngn",
		Interval:           "month",
		CreatedAt:          time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	paidAt := time.Now().UTC().Truncate(time.Second)
	payload := fmt.Sprintf(`{"event":"charge.success","data":{
      "reference":"T_renewal","amount":%d,"currency":"NGN","status":"success",
      "paid_at":%q,"metadata":0,
      "customer":{"customer_code":"CUS_renew","email":"ngn@example.com"},
      "plan":%q,
      "plan_object":{"plan_code":%q,"amount":%d,"interval":"monthly"}}}`,
		monthlyKobo, paidAt.Format(time.RFC3339), testPlanMonthly, testPlanMonthly, monthlyKobo)

	if rec := postPaystackWebhook(t, h, payload); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.ProviderSubID != "SUB_original" {
		t.Errorf("provider_sub_id = %q, want SUB_original — the renewal blanked it", sub.ProviderSubID)
	}
	if sub.Plan != "pro" || sub.Status != "active" {
		t.Errorf("plan/status = %s/%s, want pro/active", sub.Plan, sub.Status)
	}
	if sub.CurrentPeriodEnd == nil || !sub.CurrentPeriodEnd.After(original) {
		t.Errorf("current_period_end = %v, want later than %v", sub.CurrentPeriodEnd, original)
	}
}

// A one-off charge with no plan is not a subscription payment.
func TestPaystackChargeWithoutPlanIsIgnored(t *testing.T) {
	h, user := newPaystackWebhookHandler(t)

	payload := `{"event":"charge.success","data":{
      "reference":"T_oneoff","amount":50000,"currency":"NGN","status":"success",
      "metadata":0,"plan":"",
      "customer":{"customer_code":"CUS_x","email":"ngn@example.com"}}}`

	if rec := postPaystackWebhook(t, h, payload); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free — a plan-less charge granted Pro", sub.Plan)
	}
}

// Someone else's plan on the same integration must not grant our Pro.
func TestPaystackForeignPlanChargeIgnored(t *testing.T) {
	h, user := newPaystackWebhookHandler(t)

	payload := `{"event":"charge.success","data":{
      "reference":"T_other","amount":100000,"currency":"NGN","status":"success",
      "metadata":0,"plan":"PLN_not_ours",
      "plan_object":{"plan_code":"PLN_not_ours","amount":100000},
      "customer":{"customer_code":"CUS_x","email":"ngn@example.com"}}}`

	if rec := postPaystackWebhook(t, h, payload); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free", sub.Plan)
	}
}

// A failed renewal must mark the subscription past_due WITHOUT blanking the
// period — a nil period grants access indefinitely, which would disable the
// very expiry gate this event exists to trigger.
func TestPaystackInvoiceFailureMarksPastDueAndKeepsPeriod(t *testing.T) {
	h, user := newPaystackWebhookHandler(t)

	period := time.Now().UTC().Add(2 * 24 * time.Hour).Truncate(time.Second)
	if err := h.Store.UpsertSubscription(&models.Subscription{
		UserID:             user.ID,
		Plan:               "pro",
		Provider:           "paystack",
		ProviderCustomerID: "CUS_fail",
		ProviderSubID:      "SUB_keepme",
		Status:             "active",
		CurrentPeriodEnd:   &period,
		Currency:           "ngn",
		Interval:           "month",
		CreatedAt:          time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	payload := fmt.Sprintf(`{"event":"invoice.payment_failed","data":{
      "status":"failed","paid":false,"updatedAt":"2026-07-28T00:00:00.000Z",
      "customer":{"customer_code":"CUS_fail"},
      "plan":{"plan_code":%q,"amount":%d,"interval":"monthly"}}}`,
		testPlanMonthly, monthlyKobo)

	if rec := postPaystackWebhook(t, h, payload); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Status != "past_due" {
		t.Errorf("status = %q, want past_due", sub.Status)
	}
	if sub.CurrentPeriodEnd == nil {
		t.Fatal("current_period_end was blanked — the expiry gate can no longer fire")
	}
	if !sub.CurrentPeriodEnd.Equal(period) {
		t.Errorf("current_period_end = %v, want %v unchanged", sub.CurrentPeriodEnd, period)
	}
	if sub.ProviderSubID != "SUB_keepme" {
		t.Errorf("provider_sub_id = %q, want SUB_keepme", sub.ProviderSubID)
	}
}

// invoice.update reporting a successful charge is not actionable here.
func TestPaystackInvoiceSuccessIgnored(t *testing.T) {
	h, user := newPaystackWebhookHandler(t)

	payload := fmt.Sprintf(`{"event":"invoice.update","data":{
      "status":"success","paid":true,
      "customer":{"customer_code":"CUS_ok"},
      "plan":{"plan_code":%q,"amount":%d}}}`, testPlanMonthly, monthlyKobo)

	if rec := postPaystackWebhook(t, h, payload); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "free" {
		t.Errorf("plan = %q, want free — a paid invoice must not create a subscription", sub.Plan)
	}
}

// A foreign card on NGN pricing is recorded and flagged, never refused —
// blocking a real customer after charging them is worse than the mispricing.
func TestVerifyPaystack_ForeignCardOnNgnPricingIsAllowedNotBlocked(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBodyFrom("success", "buyer@example.com", testPlanMonthly, monthlyKobo, monthlyKobo, "US"))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 — a foreign card must not be refused: %s",
			rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "pro" {
		t.Errorf("plan = %q, want pro", sub.Plan)
	}
}

// A bank transfer cannot be charged again, so Paystack creates no
// subscription. The customer still gets the period they paid for, but the row
// must record that it will not renew.
func TestVerifyPaystack_NonReusablePaymentIsGrantedButNotRecurring(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBodyReusable(testPlanMonthly, monthlyKobo, monthlyKobo, "bank_transfer", false))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 — they paid, they get the period: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.Plan != "pro" || sub.Status != "active" {
		t.Errorf("plan/status = %s/%s, want pro/active", sub.Plan, sub.Status)
	}
	if sub.AutoRenews {
		t.Error("auto_renews = true for a bank transfer — it cannot be charged again")
	}
}

func TestVerifyPaystack_CardPaymentRecurs(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBodyReusable(testPlanMonthly, monthlyKobo, monthlyKobo, "card", true))

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if !sub.AutoRenews {
		t.Error("auto_renews = false for a reusable card authorization")
	}
}

// A webhook cannot see the authorization, so it must never flip the flag.
func TestPaystackWebhookPreservesAutoRenews(t *testing.T) {
	h, user := newPaystackWebhookHandler(t)

	period := time.Now().UTC().Add(10 * 24 * time.Hour).Truncate(time.Second)
	if err := h.Store.UpsertSubscription(&models.Subscription{
		UserID:             user.ID,
		Plan:               "pro",
		Provider:           "paystack",
		ProviderCustomerID: "CUS_keep",
		ProviderSubID:      "SUB_keep",
		Status:             "active",
		CurrentPeriodEnd:   &period,
		Currency:           "ngn",
		Interval:           "month",
		AutoRenews:         false, // paid by transfer
		CreatedAt:          time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	payload := fmt.Sprintf(`{"event":"charge.success","data":{
      "reference":"T_later","amount":%d,"currency":"NGN","status":"success",
      "paid_at":%q,"metadata":0,
      "customer":{"customer_code":"CUS_keep","email":"ngn@example.com"},
      "plan":%q,"plan_object":{"plan_code":%q,"amount":%d,"interval":"monthly"}}}`,
		monthlyKobo, time.Now().UTC().Format(time.RFC3339), testPlanMonthly, testPlanMonthly, monthlyKobo)

	if rec := postPaystackWebhook(t, h, payload); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}
	sub, _ := h.Store.GetSubscription(user.ID)
	if sub.AutoRenews {
		t.Error("a webhook flipped auto_renews to true — it cannot see the authorization")
	}
}

// Re-verifying the same reference must succeed but must not be applied twice.
// Once the period stacks, a replay would otherwise hand out free time.
func TestVerifyPaystack_ReplayingAReferenceDoesNotApplyTwice(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBodyReusable(testPlanMonthly, monthlyKobo, monthlyKobo, "card", true))

	post := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
			`{"reference":"T_ref","plan":"pro","interval":"month"}`))
		return rec
	}

	if rec := post(); rec.Code != http.StatusOK {
		t.Fatalf("first: got %d, want 200: %s", rec.Code, rec.Body.String())
	}
	first, _ := h.Store.GetSubscription(user.ID)
	if first.CurrentPeriodEnd == nil {
		t.Fatal("no period after the first verification")
	}
	firstEnd := *first.CurrentPeriodEnd

	for i := 2; i <= 4; i++ {
		if rec := post(); rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: got %d, want 200 — a retry must not lock the customer out: %s",
				i, rec.Code, rec.Body.String())
		}
	}

	after, _ := h.Store.GetSubscription(user.ID)
	if !after.CurrentPeriodEnd.Equal(firstEnd) {
		t.Errorf("current_period_end moved from %v to %v — replaying a reference granted extra time",
			firstEnd, after.CurrentPeriodEnd)
	}
}

// Renewing before expiry stacks onto the remaining time. Restarting from now
// would silently take back days the customer had already paid for.
func TestVerifyPaystack_RenewalExtendsFromTheCurrentExpiry(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBodyReusable(testPlanMonthly, monthlyKobo, monthlyKobo, "bank_transfer", false))

	// 20 days still to run.
	existing := time.Now().UTC().Add(20 * 24 * time.Hour).Truncate(time.Second)
	if err := h.Store.UpsertSubscription(&models.Subscription{
		UserID: user.ID, Plan: "pro", Provider: "paystack",
		ProviderCustomerID: "CUS_1", ProviderSubID: "T_earlier",
		Status: "active", CurrentPeriodEnd: &existing,
		Currency: "ngn", Interval: "month", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	want := existing.Add(30 * 24 * time.Hour)
	if sub.CurrentPeriodEnd == nil || sub.CurrentPeriodEnd.Sub(want).Abs() > time.Minute {
		t.Errorf("current_period_end = %v, want ~%v (old expiry + one month)",
			sub.CurrentPeriodEnd, want)
	}
	// Restarting from now would land ~20 days earlier.
	if sub.CurrentPeriodEnd.Before(existing) {
		t.Error("the renewal took back time the customer had already paid for")
	}
}

// An expired period must not be carried forward — that would backdate the
// renewal to a date already gone.
func TestVerifyPaystack_RenewalAfterExpiryStartsFromNow(t *testing.T) {
	h, user := newBillingTestHandler(t,
		verifyBodyReusable(testPlanMonthly, monthlyKobo, monthlyKobo, "bank_transfer", false))

	lapsed := time.Now().UTC().Add(-10 * 24 * time.Hour).Truncate(time.Second)
	if err := h.Store.UpsertSubscription(&models.Subscription{
		UserID: user.ID, Plan: "pro", Provider: "paystack",
		ProviderCustomerID: "CUS_1", ProviderSubID: "T_earlier",
		Status: "active", CurrentPeriodEnd: &lapsed,
		Currency: "ngn", Interval: "month", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rec := httptest.NewRecorder()
	h.VerifyPaystack(rec, verifyRequest(user.ID, user.Email,
		`{"reference":"T_ref","plan":"pro","interval":"month"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sub, _ := h.Store.GetSubscription(user.ID)
	want := time.Now().UTC().Add(30 * 24 * time.Hour)
	if sub.CurrentPeriodEnd.Sub(want).Abs() > time.Minute {
		t.Errorf("current_period_end = %v, want ~%v (a full period from today)",
			sub.CurrentPeriodEnd, want)
	}
}
