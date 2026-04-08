package humanauth

import (
	"context"
	"errors"
	"testing"
)

func TestCountApprovalPoliciesReturnsAuthDisabledWhenOIDCDisabled(t *testing.T) {
	service := NewService(nil, nil, nil)

	count, err := service.CountApprovalPolicies(context.Background())
	if !errors.Is(err, ErrAuthDisabled) {
		t.Fatalf("expected ErrAuthDisabled, got count=%d err=%v", count, err)
	}
	if count != 0 {
		t.Fatalf("expected zero approval policies when auth is disabled, got %d", count)
	}
}
