package billing

import (
	"encoding/json"
	"sort"
	"testing"
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
