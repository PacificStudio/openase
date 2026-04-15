package orchestrator

import (
	"strings"
	"testing"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
)

func TestComposeWorkflowDeveloperInstructionsIncludesSharedGuidance(t *testing.T) {
	instructions := composeWorkflowDeveloperInstructions(
		"# Workflow\n\nImplement the ticket using the current workspace.",
		"## Ticket Execution Context\nTicket: ASE-42 - Example",
		"## OpenASE Platform Capability Contract",
	)

	if !strings.Contains(instructions, "Implement the ticket using the current workspace.") {
		t.Fatalf("expected rendered harness in instructions, got %q", instructions)
	}
	if !strings.Contains(instructions, "## Shared Workflow Execution Rules") {
		t.Fatalf("expected shared workflow guidance in instructions, got %q", instructions)
	}
	if !strings.Contains(instructions, "## Ticket Execution Context") {
		t.Fatalf("expected ticket context in instructions, got %q", instructions)
	}
	if !strings.Contains(instructions, "## OpenASE Platform Capability Contract") {
		t.Fatalf("expected platform contract in instructions, got %q", instructions)
	}
}

func TestComposeWorkflowDeveloperInstructionsFallsBackToSharedGuidance(t *testing.T) {
	instructions := composeWorkflowDeveloperInstructions("", "", "")

	if !strings.Contains(instructions, "## Shared Workflow Execution Rules") {
		t.Fatalf("expected shared workflow guidance without optional sections, got %q", instructions)
	}
}

func TestComposeWorkflowTicketContextIncludesTicketDetailsEvenWithoutHarness(t *testing.T) {
	context := composeWorkflowTicketContext(workflowservice.HarnessTemplateData{
		Ticket: workflowservice.HarnessTicketData{
			Identifier:       "ASE-42",
			Title:            "Fix login flow",
			Description:      "Users cannot sign in after reset.",
			Status:           "Todo",
			Priority:         "high",
			Type:             "bugfix",
			AttemptCount:     1,
			MaxAttempts:      3,
			ParentIdentifier: "ASE-30",
			Links: []workflowservice.HarnessTicketLinkData{{
				Type:   "github_issue",
				Title:  "Login broken",
				Status: "open",
				URL:    "https://github.com/acme/app/issues/42",
			}},
			Dependencies: []workflowservice.HarnessTicketDependencyData{{
				Identifier: "ASE-40",
				Title:      "Auth contract",
				Type:       "blocks",
				Status:     "Done",
			}},
		},
		Repos: []workflowservice.HarnessRepoData{{
			Name:   "backend",
			Path:   "/workspaces/ASE-42/backend",
			Branch: "agent/ASE-42",
			Labels: []string{"go", "api"},
		}},
		Workspace: "/workspaces/ASE-42",
	})

	for _, snippet := range []string{
		"## Ticket Execution Context",
		"Ticket: ASE-42 - Fix login flow",
		"Status: Todo | Priority: high | Type: bugfix | Attempts: 1/3",
		"Workspace: /workspaces/ASE-42",
		"Parent Ticket: ASE-30",
		"### Ticket Description",
		"Users cannot sign in after reset.",
		"### Dependencies",
		"[ASE-40] Auth contract (blocks, status=Done)",
		"### External Links",
		"github_issue | Login broken | status=open | url=https://github.com/acme/app/issues/42",
		"### Scoped Repositories",
		"backend path=/workspaces/ASE-42/backend branch=agent/ASE-42 labels=go, api",
	} {
		if !strings.Contains(context, snippet) {
			t.Fatalf("expected ticket context to contain %q, got:\n%s", snippet, context)
		}
	}
}
