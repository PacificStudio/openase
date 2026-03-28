package hradvisor

import (
	"strings"
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

func TestAnalyzeRecommendsTechnicalWriterFromDocumentationDriftTrend(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "OpenASE",
			Status: "In Progress",
		},
		RecentActivityCount: 4,
		RecentTrends: []domain.ActivityTrendContext{
			{
				Kind:  domain.ActivityTrendDocumentationDrift,
				Count: 4,
				Evidence: []string{
					"Recent merge-like activity events: 4.",
					"Recent documentation update events: 0.",
				},
			},
		},
	})

	var writerRecommendation *domain.Recommendation
	for index := range analysis.Recommendations {
		recommendation := &analysis.Recommendations[index]
		if recommendation.RoleSlug == "technical-writer" {
			writerRecommendation = recommendation
			break
		}
	}
	if writerRecommendation == nil {
		t.Fatalf("expected technical writer recommendation, got %+v", analysis.Recommendations)
	}
	if writerRecommendation.Priority != "high" {
		t.Fatalf("expected high-priority documentation trend recommendation, got %+v", writerRecommendation)
	}
	evidence := strings.Join(writerRecommendation.Evidence, " ")
	if !strings.Contains(evidence, "merge-like activity events: 4") || !strings.Contains(evidence, "documentation update events: 0") {
		t.Fatalf("expected documentation drift evidence, got %+v", writerRecommendation.Evidence)
	}
	if analysis.Staffing.Docs != 1 {
		t.Fatalf("expected docs staffing to reflect documentation drift, got %+v", analysis.Staffing)
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

func TestAnalyzeRecommendsDispatcherFromBacklogPressure(t *testing.T) {
	tickets := make([]domain.TicketContext, 0, 11)
	for index := 0; index < 11; index++ {
		tickets = append(tickets, domain.TicketContext{
			Identifier: "ASE-backlog",
			Type:       "feature",
			StatusName: "Backlog",
		})
	}

	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "OpenASE",
			Status: "In Progress",
		},
		Tickets: tickets,
		Workflows: []domain.WorkflowContext{
			{
				Name:              "Coding Workflow",
				Type:              "coding",
				IsActive:          true,
				PickupStatusNames: []string{"Todo"},
				FinishStatusNames: []string{"Done"},
			},
		},
	})

	var dispatcherRecommendation *domain.Recommendation
	for index := range analysis.Recommendations {
		recommendation := &analysis.Recommendations[index]
		if recommendation.RoleSlug == "dispatcher" {
			dispatcherRecommendation = recommendation
			break
		}
	}
	if dispatcherRecommendation == nil {
		t.Fatalf("expected dispatcher recommendation, got %+v", analysis.Recommendations)
	}
	if dispatcherRecommendation.SuggestedWorkflowName != "Dispatcher" {
		t.Fatalf("expected dispatcher workflow name, got %+v", dispatcherRecommendation)
	}
	if !strings.Contains(dispatcherRecommendation.Reason, "Backlog") {
		t.Fatalf("expected backlog reason, got %+v", dispatcherRecommendation)
	}
	if len(dispatcherRecommendation.Evidence) < 2 || !strings.Contains(strings.Join(dispatcherRecommendation.Evidence, " "), "pick up and finish Backlog") {
		t.Fatalf("expected dispatcher evidence to mention backlog lane coverage, got %+v", dispatcherRecommendation.Evidence)
	}
}

func TestAnalyzeRecommendsMissingLaneEvenWhenRoleIsAlreadyActiveElsewhere(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "OpenASE",
			Status: "In Progress",
		},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "feature", StatusName: "Ready for Test"},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Ready for Test"},
		},
		Workflows: []domain.WorkflowContext{
			{
				Name:              "Coding Workflow",
				Type:              "coding",
				IsActive:          true,
				PickupStatusNames: []string{"Todo"},
				FinishStatusNames: []string{"Ready for Test"},
			},
			{
				Name:              "QA Regression",
				Type:              "test",
				RoleSlug:          "qa-engineer",
				IsActive:          true,
				PickupStatusNames: []string{"Regression Queue"},
				FinishStatusNames: []string{"Done"},
			},
		},
		ActiveRoleSlugs: []string{"qa-engineer"},
	})

	var qaRecommendation *domain.Recommendation
	for index := range analysis.Recommendations {
		recommendation := &analysis.Recommendations[index]
		if recommendation.RoleSlug == "qa-engineer" {
			qaRecommendation = recommendation
			break
		}
	}
	if qaRecommendation == nil {
		t.Fatalf("expected qa recommendation for missing lane, got %+v", analysis.Recommendations)
	}
	if qaRecommendation.SuggestedWorkflowName != "QA Engineer - Ready for Test" {
		t.Fatalf("expected lane-specific workflow suggestion, got %+v", qaRecommendation)
	}
	if !strings.Contains(qaRecommendation.Reason, "Ready for Test") {
		t.Fatalf("expected lane-specific reason, got %+v", qaRecommendation)
	}
	if !strings.Contains(strings.Join(qaRecommendation.Evidence, " "), "Coding Workflow") {
		t.Fatalf("expected evidence to mention upstream workflow finish binding, got %+v", qaRecommendation.Evidence)
	}
}
