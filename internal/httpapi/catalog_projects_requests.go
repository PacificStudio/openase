package httpapi

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type projectPatchRequest struct {
	Name                   *string   `json:"name"`
	Slug                   *string   `json:"slug"`
	Description            *string   `json:"description"`
	Status                 *string   `json:"status"`
	DefaultAgentProviderID *string   `json:"default_agent_provider_id"`
	AccessibleMachineIDs   *[]string `json:"accessible_machine_ids"`
	MaxConcurrentAgents    *int      `json:"max_concurrent_agents"`
	AgentRunSummaryPrompt  *string   `json:"agent_run_summary_prompt"`
}

func parseProjectPatchRequest(
	projectID uuid.UUID,
	current domain.Project,
	patch projectPatchRequest,
) (domain.UpdateProject, error) {
	request := domain.ProjectInput{
		Name:                   current.Name,
		Slug:                   current.Slug,
		Description:            current.Description,
		Status:                 current.Status.String(),
		DefaultAgentProviderID: uuidToStringPointer(current.DefaultAgentProviderID),
		AccessibleMachineIDs:   uuidSliceToStrings(current.AccessibleMachineIDs),
		MaxConcurrentAgents:    intPointer(current.MaxConcurrentAgents),
		AgentRunSummaryPrompt:  stringPointerOrNil(current.AgentRunSummaryPrompt),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.Slug != nil {
		request.Slug = *patch.Slug
	}
	if patch.Description != nil {
		request.Description = *patch.Description
	}
	if patch.Status != nil {
		request.Status = *patch.Status
	}
	if patch.DefaultAgentProviderID != nil {
		request.DefaultAgentProviderID = patch.DefaultAgentProviderID
	}
	if patch.AccessibleMachineIDs != nil {
		request.AccessibleMachineIDs = cloneStringSlice(*patch.AccessibleMachineIDs)
	}
	if patch.MaxConcurrentAgents != nil {
		request.MaxConcurrentAgents = patch.MaxConcurrentAgents
	}
	if patch.AgentRunSummaryPrompt != nil {
		request.AgentRunSummaryPrompt = patch.AgentRunSummaryPrompt
	}

	return domain.ParseUpdateProject(projectID, current.OrganizationID, request)
}

func stringPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
