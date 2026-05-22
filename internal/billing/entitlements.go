package billing

import "time"

type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
)

type Limits struct {
	MaxNamedEndpoints   int // -1 = unlimited
	MaxRequestsPerMonth int
	HistoryDays         int
	MaxSecrets          int
	HasFiltering        bool
	HasPrioritySupport  bool
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
