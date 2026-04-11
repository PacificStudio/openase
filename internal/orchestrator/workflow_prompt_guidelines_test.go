package orchestrator

import (
	"strings"
	"testing"
)

func TestComposeWorkflowDeveloperInstructionsIncludesSharedGuidance(t *testing.T) {
	instructions := composeWorkflowDeveloperInstructions(
		"# Workflow\n\nImplement the ticket using the current workspace.",
		"## OpenASE Platform Capability Contract",
	)

	if !strings.Contains(instructions, "Implement the ticket using the current workspace.") {
		t.Fatalf("expected rendered harness in instructions, got %q", instructions)
	}
	if !strings.Contains(instructions, "## Shared Workflow Execution Rules") {
		t.Fatalf("expected shared workflow guidance in instructions, got %q", instructions)
	}
	if !strings.Contains(instructions, "## OpenASE Platform Capability Contract") {
		t.Fatalf("expected platform contract in instructions, got %q", instructions)
	}
}

func TestComposeWorkflowDeveloperInstructionsFallsBackToSharedGuidance(t *testing.T) {
	instructions := composeWorkflowDeveloperInstructions("", "")

	if !strings.Contains(instructions, "## Shared Workflow Execution Rules") {
		t.Fatalf("expected shared workflow guidance without optional sections, got %q", instructions)
	}
}
