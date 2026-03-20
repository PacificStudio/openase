package httpapi

import (
	"fmt"
	"strings"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

type rawSkillSyncRequest struct {
	WorkspacePath string `json:"workspace_path"`
	AdapterType   string `json:"adapter_type"`
}

type rawUpdateWorkflowSkillsRequest struct {
	Skills []string `json:"skills"`
}

func parseRefreshSkillsRequest(projectID uuid.UUID, raw rawSkillSyncRequest) (workflowservice.RefreshSkillsInput, error) {
	workspacePath := strings.TrimSpace(raw.WorkspacePath)
	if workspacePath == "" {
		return workflowservice.RefreshSkillsInput{}, fmt.Errorf("workspace_path must not be empty")
	}
	adapterType := strings.TrimSpace(raw.AdapterType)
	if adapterType == "" {
		return workflowservice.RefreshSkillsInput{}, fmt.Errorf("adapter_type must not be empty")
	}

	return workflowservice.RefreshSkillsInput{
		ProjectID:     projectID,
		WorkspacePath: workspacePath,
		AdapterType:   adapterType,
	}, nil
}

func parseHarvestSkillsRequest(projectID uuid.UUID, raw rawSkillSyncRequest) (workflowservice.HarvestSkillsInput, error) {
	workspacePath := strings.TrimSpace(raw.WorkspacePath)
	if workspacePath == "" {
		return workflowservice.HarvestSkillsInput{}, fmt.Errorf("workspace_path must not be empty")
	}
	adapterType := strings.TrimSpace(raw.AdapterType)
	if adapterType == "" {
		return workflowservice.HarvestSkillsInput{}, fmt.Errorf("adapter_type must not be empty")
	}

	return workflowservice.HarvestSkillsInput{
		ProjectID:     projectID,
		WorkspacePath: workspacePath,
		AdapterType:   adapterType,
	}, nil
}

func parseUpdateWorkflowSkillsRequest(workflowID uuid.UUID, raw rawUpdateWorkflowSkillsRequest) (workflowservice.UpdateWorkflowSkillsInput, error) {
	if len(raw.Skills) == 0 {
		return workflowservice.UpdateWorkflowSkillsInput{}, fmt.Errorf("skills must not be empty")
	}

	return workflowservice.UpdateWorkflowSkillsInput{
		WorkflowID: workflowID,
		Skills:     append([]string(nil), raw.Skills...),
	}, nil
}
