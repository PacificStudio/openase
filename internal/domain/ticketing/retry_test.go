package ticketing

import (
	"testing"
	"time"
)

func TestComputeRetryBackoff(t *testing.T) {
	testCases := []struct {
		name        string
		attempt     int
		wantBackoff time.Duration
	}{
		{name: "non_positive_attempt_defaults_to_initial", attempt: 0, wantBackoff: 10 * time.Second},
		{name: "first_attempt_uses_initial_backoff", attempt: 1, wantBackoff: 10 * time.Second},
		{name: "second_attempt_doubles_backoff", attempt: 2, wantBackoff: 20 * time.Second},
		{name: "sixth_attempt_keeps_exponential_backoff", attempt: 6, wantBackoff: 320 * time.Second},
		{name: "seventh_attempt_caps_at_thirty_minutes", attempt: 7, wantBackoff: 30 * time.Minute},
		{name: "later_attempts_stay_capped", attempt: 12, wantBackoff: 30 * time.Minute},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := ComputeRetryBackoff(testCase.attempt); got != testCase.wantBackoff {
				t.Fatalf("ComputeRetryBackoff(%d) = %s, want %s", testCase.attempt, got, testCase.wantBackoff)
			}
		})
	}
}

func TestShouldPauseForBudget(t *testing.T) {
	testCases := []struct {
		name       string
		costAmount float64
		budgetUSD  float64
		wantPause  bool
	}{
		{name: "zero_budget_never_pauses", costAmount: 5, budgetUSD: 0, wantPause: false},
		{name: "under_budget_keeps_running", costAmount: 4.99, budgetUSD: 5, wantPause: false},
		{name: "exact_budget_pauses", costAmount: 5, budgetUSD: 5, wantPause: true},
		{name: "over_budget_pauses", costAmount: 5.01, budgetUSD: 5, wantPause: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := ShouldPauseForBudget(testCase.costAmount, testCase.budgetUSD); got != testCase.wantPause {
				t.Fatalf(
					"ShouldPauseForBudget(%v, %v) = %t, want %t",
					testCase.costAmount,
					testCase.budgetUSD,
					got,
					testCase.wantPause,
				)
			}
		})
	}
}
