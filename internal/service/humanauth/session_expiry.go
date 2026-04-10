package humanauth

import (
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
)

var neverExpireSessionAt = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

func sessionDeadline(now time.Time, ttl time.Duration) time.Time {
	if ttl <= 0 {
		return neverExpireSessionAt
	}
	return now.UTC().Add(ttl)
}

func sessionRefreshAbsoluteDeadline(current time.Time, now time.Time, ttl time.Duration) time.Time {
	if ttl <= 0 {
		return sessionDeadline(now, ttl)
	}
	return current.UTC()
}

func sessionHasNoExpiry(deadline time.Time) bool {
	return deadline.IsZero() || deadline.Year() >= neverExpireSessionAt.Year()
}

func sessionDeadlineExpired(now time.Time, deadline time.Time) bool {
	if sessionHasNoExpiry(deadline) {
		return false
	}
	return now.After(deadline.UTC())
}

func browserSessionExpired(now time.Time, session domain.BrowserSession) bool {
	return sessionDeadlineExpired(now, session.ExpiresAt) ||
		sessionDeadlineExpired(now, session.IdleExpiresAt)
}
