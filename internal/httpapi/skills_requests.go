package httpapi

import (
	"fmt"
	"strings"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

type rawSkillSyncRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	AdapterType   string `json:"adapter_type"`
}

type rawUpdateWorkflowSkillsRequest struct {
	Skills []string `json:"skills"`
}

func parseRefreshSkillsRequest(projectID uuid.UUID, raw rawSkillSyncRequest) (workflowservice.RefreshSkillsInput, error) {
	workspaceRoot := strings.TrimSpace(raw.WorkspaceRoot)
	if workspaceRoot == "" {
		return workflowservice.RefreshSkillsInput{}, fmt.Errorf("workspace_root must not be empty")
	}
	adapterType := strings.TrimSpace(raw.AdapterType)
	if adapterType == "" {
		return workflowservice.RefreshSkillsInput{}, fmt.Errorf("adapter_type must not be empty")
	}

	return workflowservice.RefreshSkillsInput{
		ProjectID:     projectID,
		WorkspaceRoot: workspaceRoot,
		AdapterType:   adapterType,
	}, nil
}

func parseHarvestSkillsRequest(projectID uuid.UUID, raw rawSkillSyncRequest) (workflowservice.HarvestSkillsInput, error) {
	workspaceRoot := strings.TrimSpace(raw.WorkspaceRoot)
	if workspaceRoot == "" {
		return workflowservice.HarvestSkillsInput{}, fmt.Errorf("workspace_root must not be empty")
	}
	adapterType := strings.TrimSpace(raw.AdapterType)
	if adapterType == "" {
		return workflowservice.HarvestSkillsInput{}, fmt.Errorf("adapter_type must not be empty")
	}

	return workflowservice.HarvestSkillsInput{
		ProjectID:     projectID,
		WorkspaceRoot: workspaceRoot,
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
