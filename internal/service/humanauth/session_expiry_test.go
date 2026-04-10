package humanauth

import (
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
)

func TestSessionDeadlineUsesFarFutureForNonExpiringSessions(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)

	if deadline := sessionDeadline(now, 0); !sessionHasNoExpiry(deadline) {
		t.Fatalf("sessionDeadline(0) = %s, want no-expiry sentinel", deadline)
	}
	if deadline := sessionDeadline(now, -1*time.Minute); !sessionHasNoExpiry(deadline) {
		t.Fatalf("sessionDeadline(negative) = %s, want no-expiry sentinel", deadline)
	}
}

func TestBrowserSessionExpiredIgnoresNoExpiryDeadlines(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	session := domain.BrowserSession{
		ExpiresAt:     sessionDeadline(now, 0),
		IdleExpiresAt: sessionDeadline(now, 0),
	}

	if browserSessionExpired(now.Add(365*24*time.Hour), session) {
		t.Fatal("browserSessionExpired() = true, want false for non-expiring session")
	}
}

func TestSessionRefreshAbsoluteDeadlineKeepsFiniteAbsoluteTimeout(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	original := now.Add(8 * time.Hour)

	if got := sessionRefreshAbsoluteDeadline(original, now.Add(30*time.Minute), 8*time.Hour); !got.Equal(original) {
		t.Fatalf("sessionRefreshAbsoluteDeadline() = %s, want %s", got, original)
	}
	if got := sessionRefreshAbsoluteDeadline(original, now, 0); !sessionHasNoExpiry(got) {
		t.Fatalf("sessionRefreshAbsoluteDeadline(no-expiry) = %s, want no-expiry sentinel", got)
	}
}
