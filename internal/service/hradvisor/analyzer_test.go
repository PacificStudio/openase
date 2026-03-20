package hradvisor

import (
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
)

func TestAnalyzeRecommendsQADocsAndSecurityFromWorkloadShape(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:        "OpenASE",
			Description: "Automation control plane",
			Status:      "active",
		},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "feature", StatusName: "Todo"},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Todo"},
			{Identifier: "ASE-3", Type: "bugfix", StatusName: "Todo", ConsecutiveErrors: 1},
			{Identifier: "ASE-4", Type: "refactor", StatusName: "In Progress", ConsecutiveErrors: 1},
			{Identifier: "ASE-5", Type: "feature", StatusName: "Todo"},
			{Identifier: "ASE-6", Type: "feature", StatusName: "Todo"},
			{Identifier: "ASE-7", Type: "feature", StatusName: "Todo"},
			{Identifier: "ASE-8", Type: "feature", StatusName: "Todo"},
		},
		Workflows: []domain.WorkflowContext{
			{Name: "Coding Workflow", Type: "coding", IsActive: true},
		},
	})

	if analysis.Summary.OpenTickets != 8 {
		t.Fatalf("expected 8 open tickets, got %+v", analysis.Summary)
	}
	if analysis.Staffing.Developers != 2 || analysis.Staffing.QA != 2 {
		t.Fatalf("unexpected staffing plan: %+v", analysis.Staffing)
	}

	got := map[string]string{}
	for _, recommendation := range analysis.Recommendations {
		got[recommendation.RoleSlug] = recommendation.Priority
	}

	for roleSlug, priority := range map[string]string{
		"qa-engineer":       "high",
		"technical-writer":  "medium",
		"security-engineer": "medium",
	} {
		if got[roleSlug] != priority {
			t.Fatalf("expected %s priority %s, got %q in %+v", roleSlug, priority, got[roleSlug], analysis.Recommendations)
		}
	}
}

func TestAnalyzeSkipsRolesThatAreAlreadyActive(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "OpenASE",
			Status: "active",
		},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "feature", StatusName: "Todo"},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Todo"},
			{Identifier: "ASE-3", Type: "feature", StatusName: "Todo"},
		},
		ActiveRoleSlugs: []string{"qa-engineer"},
	})

	for _, recommendation := range analysis.Recommendations {
		if recommendation.RoleSlug == "qa-engineer" {
			t.Fatalf("did not expect qa-engineer recommendation when role is already active: %+v", analysis.Recommendations)
		}
	}
}

func TestAnalyzeRecommendsProductManagerForEmptyPlanningProject(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Greenfield",
			Status: "planning",
		},
	})

	if len(analysis.Recommendations) == 0 || analysis.Recommendations[0].RoleSlug != "product-manager" {
		t.Fatalf("expected product-manager recommendation, got %+v", analysis.Recommendations)
	}
	if analysis.Staffing.Product != 1 {
		t.Fatalf("expected product staffing suggestion, got %+v", analysis.Staffing)
	}
}
