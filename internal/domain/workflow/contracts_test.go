package workflow

import (
	"errors"
	"testing"
)

func TestWorkflowImpactConflictWrapsUnderlyingError(t *testing.T) {
	conflict := &WorkflowImpactConflict{Err: ErrWorkflowHistoricalAgentRuns}
	if !errors.Is(conflict, ErrWorkflowHistoricalAgentRuns) {
		t.Fatal("WorkflowImpactConflict should wrap its underlying error")
	}
	if conflict.Error() != ErrWorkflowHistoricalAgentRuns.Error() {
		t.Fatalf("WorkflowImpactConflict.Error() = %q", conflict.Error())
	}
}

func TestWorkflowImpactConflictNilReceiver(t *testing.T) {
	var conflict *WorkflowImpactConflict
	if got := conflict.Error(); got != "" {
		t.Fatalf("nil WorkflowImpactConflict.Error() = %q", got)
	}
	if conflict.Unwrap() != nil {
		t.Fatal("nil WorkflowImpactConflict.Unwrap() should be nil")
	}
}

func TestWorkflowImpactConflictNilError(t *testing.T) {
	conflict := &WorkflowImpactConflict{}
	if got := conflict.Error(); got != "" {
		t.Fatalf("WorkflowImpactConflict.Error() with nil Err = %q", got)
	}
	if conflict.Unwrap() != nil {
		t.Fatal("WorkflowImpactConflict.Unwrap() with nil Err should be nil")
	}
}
