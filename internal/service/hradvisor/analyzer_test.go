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
			Status:      "In Progress",
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
			Status: "In Progress",
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

func TestAnalyzeCanonicalProjectStatuses(t *testing.T) {
	testCases := []struct {
		name           string
		status         string
		wantRoles      []string
		wantDevelopers int
		wantProduct    int
	}{
		{name: "backlog", status: "Backlog"},
		{name: "planned", status: "Planned", wantRoles: []string{"product-manager"}, wantProduct: 1},
		{name: "in progress", status: "In Progress", wantRoles: []string{"fullstack-developer"}, wantDevelopers: 1},
		{name: "completed", status: "Completed"},
		{name: "canceled", status: "Canceled"},
		{name: "archived", status: "Archived"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			analysis := Analyze(domain.Snapshot{
				Project: domain.ProjectContext{
					Name:   "Greenfield",
					Status: testCase.status,
				},
			})

			gotRoles := make([]string, 0, len(analysis.Recommendations))
			for _, recommendation := range analysis.Recommendations {
				gotRoles = append(gotRoles, recommendation.RoleSlug)
			}

			if len(gotRoles) != len(testCase.wantRoles) {
				t.Fatalf("status %q: expected roles %v, got %v", testCase.status, testCase.wantRoles, gotRoles)
			}
			for i := range gotRoles {
				if gotRoles[i] != testCase.wantRoles[i] {
					t.Fatalf("status %q: expected roles %v, got %v", testCase.status, testCase.wantRoles, gotRoles)
				}
			}
			if analysis.Staffing.Developers != testCase.wantDevelopers {
				t.Fatalf("status %q: expected developer staffing %d, got %+v", testCase.status, testCase.wantDevelopers, analysis.Staffing)
			}
			if analysis.Staffing.Product != testCase.wantProduct {
				t.Fatalf("status %q: expected product staffing %d, got %+v", testCase.status, testCase.wantProduct, analysis.Staffing)
			}
		})
	}
}

func TestAnalyzeDoesNotAcceptLegacyProjectStatuses(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Greenfield",
			Status: "planning",
		},
	})

	if len(analysis.Recommendations) != 0 {
		t.Fatalf("expected no legacy-status recommendation, got %+v", analysis.Recommendations)
	}
	if analysis.Staffing.Product != 0 || analysis.Staffing.Developers != 0 {
		t.Fatalf("expected no legacy-status staffing bump, got %+v", analysis.Staffing)
	}
}
