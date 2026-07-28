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

// IsActive returns true if the subscription is usable right now
// Handles grace period: past_due users keep access for 7 days
func IsActive(status string, periodEnd *time.Time) bool {
	switch status {
	case "active", "trialing":
		return true
	case "past_due":
		// Grace period: keep access for 7 days after period end
		if periodEnd != nil {
			return time.Now().Before(periodEnd.Add(7 * 24 * time.Hour))
		}
		return true
	default:
		return false
	}
}
