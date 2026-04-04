package httpapi

import (
	"testing"
	"time"
)

func TestSSEKeepaliveInterval(t *testing.T) {
	if sseKeepaliveInterval != 5*time.Second {
		t.Fatalf("sseKeepaliveInterval = %s, want 5s", sseKeepaliveInterval)
	}
}
