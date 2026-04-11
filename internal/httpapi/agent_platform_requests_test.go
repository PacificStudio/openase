package httpapi

import (
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestParseAgentProjectPatchRequestSupportsFullProjectSurface(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	defaultProviderID := uuid.New()
	accessibleMachineID := uuid.New()
	maxConcurrentAgents := 5
	name := "OpenASE Automation"
	slug := "openase-automation"
	description := "Updated by agent platform API"
	status := domain.ProjectStatusInProgress.String()
	runSummaryPrompt := "Summarize blockers first."
	projectAIPlatformAccessAllowed := []string{"projects.update", "tickets.list"}

	input, err := parseAgentProjectPatchRequest(projectID, domain.Project{
		ID:                             projectID,
		OrganizationID:                 orgID,
		Name:                           "Legacy",
		Slug:                           "legacy",
		Description:                    "Legacy project",
		Status:                         domain.ProjectStatusCompleted,
		DefaultAgentProviderID:         &defaultProviderID,
		ProjectAIPlatformAccessAllowed: []string{"projects.update"},
		AccessibleMachineIDs:           []uuid.UUID{uuid.New()},
		MaxConcurrentAgents:            1,
		AgentRunSummaryPrompt:          "Old prompt",
	}, rawAgentProjectPatchRequest{
		Name:                           &name,
		Slug:                           &slug,
		Description:                    &description,
		Status:                         &status,
		DefaultAgentProviderID:         stringPointer(defaultProviderID.String()),
		ProjectAIPlatformAccessAllowed: &projectAIPlatformAccessAllowed,
		AccessibleMachineIDs:           &[]string{accessibleMachineID.String()},
		MaxConcurrentAgents:            &maxConcurrentAgents,
		AgentRunSummaryPrompt:          &runSummaryPrompt,
	})
	if err != nil {
		t.Fatalf("parseAgentProjectPatchRequest() error = %v", err)
	}

	if input.Name != name ||
		input.Slug != slug ||
		input.Description != description ||
		input.Status != domain.ProjectStatusInProgress ||
		input.MaxConcurrentAgents != maxConcurrentAgents ||
		input.AgentRunSummaryPrompt != runSummaryPrompt {
		t.Fatalf("parseAgentProjectPatchRequest() = %+v", input)
	}
	if input.DefaultAgentProviderID == nil || *input.DefaultAgentProviderID != defaultProviderID {
		t.Fatalf("default agent provider = %v, want %s", input.DefaultAgentProviderID, defaultProviderID)
	}
	if len(input.AccessibleMachineIDs) != 1 || input.AccessibleMachineIDs[0] != accessibleMachineID {
		t.Fatalf("accessible machine ids = %+v, want [%s]", input.AccessibleMachineIDs, accessibleMachineID)
	}
	if len(input.ProjectAIPlatformAccessAllowed) != 2 || input.ProjectAIPlatformAccessAllowed[1] != "tickets.list" {
		t.Fatalf("project ai platform access allowed = %+v", input.ProjectAIPlatformAccessAllowed)
	}
}
