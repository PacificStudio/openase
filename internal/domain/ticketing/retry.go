package ticketing

import "time"

const (
	initialRetryBackoff = 10 * time.Second
	maxRetryBackoff     = 30 * time.Minute
)

type PauseReason string

const (
	PauseReasonBudgetExhausted PauseReason = "budget_exhausted"
	PauseReasonRepeatedStalls  PauseReason = "repeated_stalls"
	PauseReasonUserPaused      PauseReason = "user_paused"
)

func ComputeRetryBackoff(attemptCount int) time.Duration {
	if attemptCount <= 1 {
		return initialRetryBackoff
	}

	backoff := initialRetryBackoff
	for attempt := 1; attempt < attemptCount; attempt++ {
		backoff *= 2
		if backoff >= maxRetryBackoff {
			return maxRetryBackoff
		}
	}

	return backoff
}

func ShouldPauseForBudget(costAmount float64, budgetUSD float64) bool {
	return budgetUSD > 0 && costAmount >= budgetUSD
}

func (r PauseReason) String() string {
	return string(r)
}
