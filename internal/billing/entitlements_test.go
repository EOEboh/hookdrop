package billing

import (
	"encoding/json"
	"sort"
	"testing"
	"time"
)

// The wire format is a contract with ui/src/types/index.ts PlanLimits.
// Renaming a field here silently breaks the client, so pin the key names.
func TestLimitsJSONKeysMatchClientContract(t *testing.T) {
	raw, err := json.Marshal(GetLimits("pro"))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := []string{
		"has_filtering",
		"has_priority_support",
		"history_days",
		"max_named_endpoints",
		"max_requests_per_month",
		"max_secrets",
	}

	var got []string
	for k := range decoded {
		got = append(got, k)
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("keys = %v, want %v", got, want)
			break
		}
	}

	// Spot-check a value survived the tagging.
	if decoded["max_named_endpoints"] != float64(-1) {
		t.Errorf("pro max_named_endpoints = %v, want -1 (unlimited)", decoded["max_named_endpoints"])
	}
	if decoded["has_filtering"] != true {
		t.Errorf("pro has_filtering = %v, want true", decoded["has_filtering"])
	}
}

func TestIsActiveRespectsThePeriod(t *testing.T) {
	ago := func(d time.Duration) *time.Time { t := time.Now().Add(-d); return &t }
	ahead := func(d time.Duration) *time.Time { t := time.Now().Add(d); return &t }

	cases := []struct {
		name      string
		status    string
		periodEnd *time.Time
		want      bool
	}{
		// The bug: a lapsed subscription sits at "active" forever because
		// Paystack stops sending events once a charge fails.
		{"active, period lapsed well past grace", "active", ago(10 * 24 * time.Hour), false},
		{"active, lapsed just past grace", "active", ago(AccessGrace + time.Hour), false},
		{"active, lapsed but inside grace", "active", ago(AccessGrace - time.Hour), true},
		{"active, period still running", "active", ahead(20 * 24 * time.Hour), true},
		// A missing period is not evidence of lapsing.
		{"active, no period recorded", "active", nil, true},

		{"trialing, trial period lapsed", "trialing", ago(10 * 24 * time.Hour), false},
		{"trialing, still running", "trialing", ahead(5 * 24 * time.Hour), true},

		// past_due keeps its longer allowance.
		{"past_due inside its grace", "past_due", ago(PastDueGrace - time.Hour), true},
		{"past_due beyond its grace", "past_due", ago(PastDueGrace + time.Hour), false},
		{"past_due, no period", "past_due", nil, true},

		{"canceled", "canceled", ahead(30 * 24 * time.Hour), false},
		{"unknown status", "banana", ahead(30 * 24 * time.Hour), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsActive(c.status, c.periodEnd); got != c.want {
				t.Errorf("IsActive(%q, %v) = %t, want %t", c.status, c.periodEnd, got, c.want)
			}
		})
	}
}

// The write path must not inherit the date gate.
func TestIsEntitledStatusIgnoresDates(t *testing.T) {
	for _, s := range []string{"active", "trialing", "past_due"} {
		if !IsEntitledStatus(s) {
			t.Errorf("IsEntitledStatus(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"canceled", "expired", "", "paid"} {
		if IsEntitledStatus(s) {
			t.Errorf("IsEntitledStatus(%q) = true, want false", s)
		}
	}
}
