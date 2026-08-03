package billing

import "time"

type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
)

// Limits is serialised straight onto /billing/subscription and /me, so the
// json tags have to match ui/src/types/index.ts PlanLimits. Without them Go
// emits Go field names and every lookup on the client reads undefined.
type Limits struct {
	MaxNamedEndpoints   int  `json:"max_named_endpoints"` // -1 = unlimited
	MaxRequestsPerMonth int  `json:"max_requests_per_month"`
	HistoryDays         int  `json:"history_days"`
	MaxSecrets          int  `json:"max_secrets"`
	HasFiltering        bool `json:"has_filtering"`
	HasPrioritySupport  bool `json:"has_priority_support"`
}

var PlanLimits = map[Plan]Limits{
	PlanFree: {
		MaxNamedEndpoints:   1,
		MaxRequestsPerMonth: 500,
		HistoryDays:         7,
		MaxSecrets:          1,
		HasFiltering:        false,
		HasPrioritySupport:  false,
	},
	PlanPro: {
		MaxNamedEndpoints:   -1,
		MaxRequestsPerMonth: 50000,
		HistoryDays:         90,
		MaxSecrets:          -1,
		HasFiltering:        true,
		HasPrioritySupport:  true,
	},
}

func GetLimits(plan string) Limits {
	if l, ok := PlanLimits[Plan(plan)]; ok {
		return l
	}
	return PlanLimits[PlanFree]
}

// AccessGrace is how long access continues past current_period_end before a
// subscription is treated as lapsed.
//
// It absorbs a late or retried renewal webhook without handing out a free
// billing cycle. Both providers renew on the period end date, so anything
// still unrenewed three days later has genuinely stopped paying.
const AccessGrace = 3 * 24 * time.Hour

// PastDueGrace is the longer allowance for a subscription already known to be
// in trouble, where the provider may still recover the payment.
const PastDueGrace = 7 * 24 * time.Hour

// IsActive reports whether a subscription is usable right now.
//
// The period is authoritative, not the status. Paystack does not retry a
// failed subscription charge: "when a payment attempt fails, it will not be
// attempted again", so a lapsed subscription simply stops producing events
// and sits at status "active" forever. Trusting the status alone handed those
// customers Pro indefinitely.
func IsActive(status string, periodEnd *time.Time) bool {
	switch status {
	case "active", "trialing":
		// A period that ran out means no access, whatever the status claims.
		// A nil period is not evidence of anything, so it keeps access.
		if periodEnd != nil && time.Now().After(periodEnd.Add(AccessGrace)) {
			return false
		}
		return true
	case "past_due":
		if periodEnd != nil {
			return time.Now().Before(periodEnd.Add(PastDueGrace))
		}
		return true
	default:
		return false
	}
}

// IsEntitledStatus reports whether a status on its own grants access,
// ignoring dates entirely.
//
// This is for the webhook write path, where the period is the value being
// written: gating on it there would be circular, and would misread events
// that legitimately carry a past period.
func IsEntitledStatus(status string) bool {
	switch status {
	case "active", "trialing", "past_due":
		return true
	default:
		return false
	}
}
