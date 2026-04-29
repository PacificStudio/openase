package httpapi

import (
	"slices"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestParseProjectPatchRequestFiltersLegacyProjectAIScopesOnUnrelatedUpdates(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	runSummaryPrompt := "Summarize blockers first."

	input, err := parseProjectPatchRequest(projectID, domain.Project{
		ID:                             projectID,
		OrganizationID:                 orgID,
		Name:                           "OpenASE",
		Slug:                           "openase",
		Status:                         domain.ProjectStatusInProgress,
		ProjectAIPlatformAccessAllowed: []string{"projects.update", "tickets.report_usage"},
		MaxConcurrentAgents:            1,
	}, projectPatchRequest{
		AgentRunSummaryPrompt: &runSummaryPrompt,
	})
	if err != nil {
		t.Fatalf("parseProjectPatchRequest() error = %v", err)
	}

	if input.AgentRunSummaryPrompt != runSummaryPrompt {
		t.Fatalf("agent run summary prompt = %q, want %q", input.AgentRunSummaryPrompt, runSummaryPrompt)
	}
	if got, want := input.ProjectAIPlatformAccessAllowed, []string{"projects.update"}; !slices.Equal(got, want) {
		t.Fatalf("project ai platform access allowed = %v, want %v", got, want)
	}
}
