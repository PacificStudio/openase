package hradvisor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/builtin"
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
			{Identifier: "ASE-1", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
			{Identifier: "ASE-3", Type: "bugfix", StatusName: "Todo", StatusStage: "unstarted", ConsecutiveErrors: 1},
			{Identifier: "ASE-4", Type: "refactor", StatusName: "In Progress", StatusStage: "started", ConsecutiveErrors: 1},
			{Identifier: "ASE-5", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
			{Identifier: "ASE-6", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
			{Identifier: "ASE-7", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
			{Identifier: "ASE-8", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
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
			{Identifier: "ASE-1", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
			{Identifier: "ASE-3", Type: "feature", StatusName: "Todo", StatusStage: "unstarted"},
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

func TestAnalyzeAcceptsEquivalentProjectStatuses(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Greenfield",
			Status: "planning",
		},
	})

	if len(analysis.Recommendations) != 1 || analysis.Recommendations[0].RoleSlug != "product-manager" {
		t.Fatalf("expected planning to map to product-manager recommendation, got %+v", analysis.Recommendations)
	}
	if analysis.Staffing.Product != 1 || analysis.Staffing.Developers != 0 {
		t.Fatalf("expected planning to drive product staffing only, got %+v", analysis.Staffing)
	}
}

func TestAnalyzeRecommendsDispatcherFromBacklogPressure(t *testing.T) {
	tickets := make([]domain.TicketContext, 0, 11)
	for index := 0; index < 11; index++ {
		tickets = append(tickets, domain.TicketContext{
			Identifier:  fmt.Sprintf("ASE-backlog-%d", index),
			Type:        "feature",
			StatusName:  "Backlog",
			StatusStage: "backlog",
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
				Name:     "Coding Workflow",
				Type:     "coding",
				IsActive: true,
				PickupStatuses: []domain.StatusBindingContext{
					{Name: "Todo", Stage: "unstarted"},
				},
				FinishStatuses: []domain.StatusBindingContext{
					{Name: "Done", Stage: "completed"},
				},
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
	if len(dispatcherRecommendation.Evidence) < 2 || !strings.Contains(strings.Join(dispatcherRecommendation.Evidence, " "), "pick up Backlog and finish into downstream non-backlog work statuses") {
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
			{Identifier: "ASE-1", Type: "feature", StatusName: "Ready for Test", StatusStage: "started"},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Ready for Test", StatusStage: "started"},
		},
		Workflows: []domain.WorkflowContext{
			{
				Name:     "Coding Workflow",
				Type:     "coding",
				IsActive: true,
				PickupStatuses: []domain.StatusBindingContext{
					{Name: "Todo", Stage: "unstarted"},
				},
				FinishStatuses: []domain.StatusBindingContext{
					{Name: "Ready for Test", Stage: "started"},
				},
			},
			{
				Name:     "QA Regression",
				Type:     "test",
				RoleSlug: "qa-engineer",
				IsActive: true,
				PickupStatuses: []domain.StatusBindingContext{
					{Name: "Regression Queue", Stage: "started"},
				},
				FinishStatuses: []domain.StatusBindingContext{
					{Name: "Done", Stage: "completed"},
				},
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

func TestRoleRecommendationSupportMatrixCoversBuiltinRoles(t *testing.T) {
	roles := builtin.Roles()
	if len(roleRecommendationSupportMatrix) != len(roles) {
		t.Fatalf("expected support matrix to cover %d roles, got %d", len(roles), len(roleRecommendationSupportMatrix))
	}

	for _, role := range roles {
		support, ok := recommendationSupport(role.Slug)
		if !ok {
			t.Fatalf("missing recommendation support entry for %s", role.Slug)
		}
		if support.RoleSlug != role.Slug {
			t.Fatalf("support entry for %s has mismatched slug %+v", role.Slug, support)
		}
		if support.Reason == "" {
			t.Fatalf("support entry for %s should include a reason", role.Slug)
		}
	}

	for _, roleSlug := range []string{
		"frontend-engineer",
		"backend-engineer",
		"devops-engineer",
		"code-reviewer",
		"env-provisioner",
		"harness-optimizer",
	} {
		support, ok := recommendationSupport(roleSlug)
		if !ok || support.Status != recommendationSupportSupportedNow {
			t.Fatalf("expected %s to be supported now, got %+v (ok=%t)", roleSlug, support, ok)
		}
	}

	marketAnalyst, _ := recommendationSupport("market-analyst")
	if marketAnalyst.Status != recommendationSupportIntentionallyDisabled {
		t.Fatalf("expected market-analyst to be intentionally unsupported, got %+v", marketAnalyst)
	}
	dataAnalyst, _ := recommendationSupport("data-analyst")
	if dataAnalyst.Status != recommendationSupportPlanned {
		t.Fatalf("expected data-analyst to be planned, got %+v", dataAnalyst)
	}
}

func TestAnalyzeRecommendsFrontendAndBackendLanesSeparately(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Split Delivery",
			Status: "In Progress",
		},
		Agents: []domain.AgentContext{{Status: "running"}},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "feature", StatusName: "Frontend Ready", StatusStage: "unstarted"},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Frontend Ready", StatusStage: "unstarted"},
			{Identifier: "ASE-3", Type: "feature", StatusName: "API Queue", StatusStage: "unstarted"},
			{Identifier: "ASE-4", Type: "feature", StatusName: "API Queue", StatusStage: "unstarted"},
		},
		Workflows: []domain.WorkflowContext{
			{Name: "Dispatcher", RoleSlug: "dispatcher", IsActive: true},
		},
	})

	got := make(map[string]string)
	for _, recommendation := range analysis.Recommendations {
		got[recommendation.RoleSlug] = recommendation.SuggestedWorkflowName
	}

	if got["frontend-engineer"] != "Frontend Engineer - Frontend Ready" {
		t.Fatalf("expected frontend lane recommendation, got %+v", analysis.Recommendations)
	}
	if got["backend-engineer"] != "Backend Engineer - API Queue" {
		t.Fatalf("expected backend lane recommendation, got %+v", analysis.Recommendations)
	}
}

func TestAnalyzeRecommendsDevopsForMissingDeployLane(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Release Train",
			Status: "In Progress",
		},
		Agents: []domain.AgentContext{{Status: "running"}},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "chore", StatusName: "Ready for Deploy", StatusStage: "started"},
			{Identifier: "ASE-2", Type: "chore", StatusName: "Ready for Deploy", StatusStage: "started"},
		},
		Workflows: []domain.WorkflowContext{
			{
				Name:     "QA Regression",
				Type:     "test",
				RoleSlug: "qa-engineer",
				IsActive: true,
				FinishStatuses: []domain.StatusBindingContext{
					{Name: "Ready for Deploy", Stage: "started"},
				},
			},
		},
	})

	var recommendation *domain.Recommendation
	for index := range analysis.Recommendations {
		if analysis.Recommendations[index].RoleSlug == "devops-engineer" {
			recommendation = &analysis.Recommendations[index]
			break
		}
	}
	if recommendation == nil {
		t.Fatalf("expected devops recommendation, got %+v", analysis.Recommendations)
	}
	if !strings.Contains(strings.Join(recommendation.Evidence, " "), "Upstream workflow families") {
		t.Fatalf("expected deploy recommendation evidence to mention upstream workflow families, got %+v", recommendation.Evidence)
	}
}

func TestAnalyzeSkipsDeployRecommendationWhenEquivalentWorkflowExists(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Release Train",
			Status: "In Progress",
		},
		Agents: []domain.AgentContext{{Status: "running"}},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "chore", StatusName: "Ready for Deploy", StatusStage: "started"},
		},
		Workflows: []domain.WorkflowContext{
			{
				Name:     "Release Workflow",
				Type:     "deploy",
				RoleSlug: "devops-engineer",
				IsActive: true,
				PickupStatuses: []domain.StatusBindingContext{
					{Name: "Ready for Deploy", Stage: "started"},
				},
			},
		},
	})

	for _, recommendation := range analysis.Recommendations {
		if recommendation.RoleSlug == "devops-engineer" {
			t.Fatalf("did not expect duplicate deploy recommendation when workflow already covers the lane: %+v", analysis.Recommendations)
		}
	}
}

func TestAnalyzeRecommendsEnvProvisionerFromRepeatedStalls(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Broken Runner",
			Status: "In Progress",
		},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "bugfix", StatusName: "Todo", StatusStage: "unstarted", ConsecutiveErrors: 3, RetryPaused: true},
			{Identifier: "ASE-2", Type: "bugfix", StatusName: "Todo", StatusStage: "unstarted", ConsecutiveErrors: 2, RetryPaused: true},
		},
		Workflows: []domain.WorkflowContext{
			{Name: "Backend", Type: "coding", RoleSlug: "backend-engineer", IsActive: true},
		},
		RecentTrends: []domain.ActivityTrendContext{
			{Kind: domain.ActivityTrendFailureBurst, Count: 1, Evidence: []string{"Recent failure burst on machine bootstrap."}},
		},
	})

	var recommendation *domain.Recommendation
	for index := range analysis.Recommendations {
		if analysis.Recommendations[index].RoleSlug == "env-provisioner" {
			recommendation = &analysis.Recommendations[index]
			break
		}
	}
	if recommendation == nil {
		t.Fatalf("expected env provisioner recommendation, got %+v", analysis.Recommendations)
	}
	if recommendation.Priority != "high" {
		t.Fatalf("expected high-priority env recommendation, got %+v", recommendation)
	}
	if !strings.Contains(strings.Join(recommendation.Evidence, " "), "Blocked tickets with paused retries") {
		t.Fatalf("expected env recommendation evidence to mention paused retries, got %+v", recommendation.Evidence)
	}
}

func TestAnalyzeRecommendsHarnessOptimizerFromRetryDrift(t *testing.T) {
	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Prompt Drift",
			Status: "In Progress",
		},
		Tickets: []domain.TicketContext{
			{Identifier: "ASE-1", Type: "feature", StatusName: "Todo", StatusStage: "unstarted", ConsecutiveErrors: 1, RetryPaused: true},
			{Identifier: "ASE-2", Type: "feature", StatusName: "Todo", StatusStage: "unstarted", ConsecutiveErrors: 2},
			{Identifier: "ASE-3", Type: "feature", StatusName: "Todo", StatusStage: "unstarted", ConsecutiveErrors: 1},
		},
		Workflows: []domain.WorkflowContext{
			{Name: "Coding Workflow", Type: "coding", RoleSlug: "fullstack-developer", IsActive: true},
			{Name: "QA Workflow", Type: "test", RoleSlug: "qa-engineer", IsActive: true},
		},
		RecentTrends: []domain.ActivityTrendContext{
			{Kind: domain.ActivityTrendFailureBurst, Count: 2, Evidence: []string{"Repeated retries across the same workflow lane."}},
		},
	})

	var recommendation *domain.Recommendation
	for index := range analysis.Recommendations {
		if analysis.Recommendations[index].RoleSlug == "harness-optimizer" {
			recommendation = &analysis.Recommendations[index]
			break
		}
	}
	if recommendation == nil {
		t.Fatalf("expected harness optimizer recommendation, got %+v", analysis.Recommendations)
	}
	if recommendation.Priority != "high" {
		t.Fatalf("expected high-priority harness recommendation, got %+v", recommendation)
	}
}

func TestAnalyzeSupportsMultipleSameFamilyLaneGaps(t *testing.T) {
	tickets := make([]domain.TicketContext, 0, 4)
	for index, statusName := range []string{"Ready for Test", "Regression Queue"} {
		for offset := 0; offset < 2; offset++ {
			tickets = append(tickets, domain.TicketContext{
				Identifier:  fmt.Sprintf("ASE-%d-%d", index, offset),
				Type:        "feature",
				StatusName:  statusName,
				StatusStage: "started",
			})
		}
	}

	analysis := Analyze(domain.Snapshot{
		Project: domain.ProjectContext{
			Name:   "Multi QA",
			Status: "In Progress",
		},
		Agents:  []domain.AgentContext{{Status: "running"}},
		Tickets: tickets,
		Workflows: []domain.WorkflowContext{
			{
				Name:     "Coding Workflow",
				Type:     "coding",
				RoleSlug: "fullstack-developer",
				IsActive: true,
				FinishStatuses: []domain.StatusBindingContext{
					{Name: "Ready for Test", Stage: "started"},
					{Name: "Regression Queue", Stage: "started"},
				},
			},
		},
	})

	var qaRecommendations []domain.Recommendation
	for _, recommendation := range analysis.Recommendations {
		if recommendation.RoleSlug == "qa-engineer" {
			qaRecommendations = append(qaRecommendations, recommendation)
		}
	}
	if len(qaRecommendations) != 2 {
		t.Fatalf("expected two qa lane recommendations, got %+v", analysis.Recommendations)
	}
	if qaRecommendations[0].SuggestedWorkflowName == qaRecommendations[1].SuggestedWorkflowName {
		t.Fatalf("expected qa recommendations to remain lane-specific, got %+v", qaRecommendations)
	}
}
