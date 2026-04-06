package httpapi

import (
	"encoding/base64"
	"fmt"
	"strings"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

type rawSkillSyncRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	AdapterType   string `json:"adapter_type"`
	WorkflowID    string `json:"workflow_id"`
}

type rawUpdateWorkflowSkillsRequest struct {
	Skills []string `json:"skills"`
}

type rawSkillBundleFileRequest struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
	MediaType     string `json:"media_type"`
	IsExecutable  bool   `json:"is_executable"`
}

type rawCreateSkillRequest struct {
	Name        string `json:"name"`
	Content     string `json:"content"`
	Description string `json:"description"`
	CreatedBy   string `json:"created_by"`
	IsEnabled   *bool  `json:"is_enabled"`
}

type rawImportSkillBundleRequest struct {
	Name      string                      `json:"name"`
	CreatedBy string                      `json:"created_by"`
	IsEnabled *bool                       `json:"is_enabled"`
	Files     []rawSkillBundleFileRequest `json:"files"`
}

type rawUpdateSkillRequest struct {
	Content     *string                     `json:"content,omitempty"`
	Description string                      `json:"description"`
	Files       []rawSkillBundleFileRequest `json:"files,omitempty"`
}

type rawUpdateSkillBindingsRequest struct {
	WorkflowIDs []string `json:"workflow_ids"`
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
	var workflowID *uuid.UUID
	if trimmed := strings.TrimSpace(raw.WorkflowID); trimmed != "" {
		parsed, err := uuid.Parse(trimmed)
		if err != nil {
			return workflowservice.RefreshSkillsInput{}, fmt.Errorf("workflow_id must be a UUID")
		}
		workflowID = &parsed
	}

	return workflowservice.RefreshSkillsInput{
		ProjectID:     projectID,
		WorkspaceRoot: workspaceRoot,
		AdapterType:   adapterType,
		WorkflowID:    workflowID,
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

func parseCreateSkillRequest(projectID uuid.UUID, raw rawCreateSkillRequest) (workflowservice.CreateSkillInput, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return workflowservice.CreateSkillInput{}, fmt.Errorf("name must not be empty")
	}
	if strings.TrimSpace(raw.Content) == "" {
		return workflowservice.CreateSkillInput{}, fmt.Errorf("content must not be empty")
	}

	return workflowservice.CreateSkillInput{
		ProjectID:   projectID,
		Name:        name,
		Content:     raw.Content,
		Description: strings.TrimSpace(raw.Description),
		CreatedBy:   strings.TrimSpace(raw.CreatedBy),
		Enabled:     raw.IsEnabled,
	}, nil
}

func parseImportSkillBundleRequest(
	projectID uuid.UUID,
	raw rawImportSkillBundleRequest,
) (workflowservice.CreateSkillBundleInput, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return workflowservice.CreateSkillBundleInput{}, fmt.Errorf("name must not be empty")
	}
	if len(raw.Files) == 0 {
		return workflowservice.CreateSkillBundleInput{}, fmt.Errorf("files must not be empty")
	}

	files := make([]workflowservice.SkillBundleFileInput, 0, len(raw.Files))
	for _, item := range raw.Files {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			return workflowservice.CreateSkillBundleInput{}, fmt.Errorf("files.path must not be empty")
		}
		content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(item.ContentBase64))
		if err != nil {
			return workflowservice.CreateSkillBundleInput{}, fmt.Errorf("files.content_base64 must be valid base64")
		}
		files = append(files, workflowservice.SkillBundleFileInput{
			Path:         path,
			Content:      content,
			IsExecutable: item.IsExecutable,
			MediaType:    strings.TrimSpace(item.MediaType),
		})
	}

	return workflowservice.CreateSkillBundleInput{
		ProjectID: projectID,
		Name:      name,
		Files:     files,
		CreatedBy: strings.TrimSpace(raw.CreatedBy),
		Enabled:   raw.IsEnabled,
	}, nil
}

type parsedUpdateSkillRequest struct {
	SingleFile  *workflowservice.UpdateSkillInput
	BundleFiles *workflowservice.UpdateSkillBundleInput
}

func parseUpdateSkillRequest(skillID uuid.UUID, raw rawUpdateSkillRequest) (parsedUpdateSkillRequest, error) {
	if len(raw.Files) > 0 {
		files := make([]workflowservice.SkillBundleFileInput, 0, len(raw.Files))
		for _, item := range raw.Files {
			path := strings.TrimSpace(item.Path)
			if path == "" {
				return parsedUpdateSkillRequest{}, fmt.Errorf("files.path must not be empty")
			}
			content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(item.ContentBase64))
			if err != nil {
				return parsedUpdateSkillRequest{}, fmt.Errorf("files.content_base64 must be valid base64")
			}
			files = append(files, workflowservice.SkillBundleFileInput{
				Path:         path,
				Content:      content,
				IsExecutable: item.IsExecutable,
				MediaType:    strings.TrimSpace(item.MediaType),
			})
		}

		input := workflowservice.UpdateSkillBundleInput{
			SkillID:      skillID,
			Files:        files,
			Description:  strings.TrimSpace(raw.Description),
			ReplaceEntry: raw.Content != nil,
		}
		if raw.Content != nil {
			input.Content = *raw.Content
		}

		return parsedUpdateSkillRequest{BundleFiles: &input}, nil
	}

	if raw.Content == nil || strings.TrimSpace(*raw.Content) == "" {
		return parsedUpdateSkillRequest{}, fmt.Errorf("content must not be empty")
	}

	return parsedUpdateSkillRequest{
		SingleFile: &workflowservice.UpdateSkillInput{
			SkillID:     skillID,
			Content:     *raw.Content,
			Description: strings.TrimSpace(raw.Description),
		},
	}, nil
}

func parseUpdateSkillBindingsRequest(
	skillID uuid.UUID,
	raw rawUpdateSkillBindingsRequest,
) (workflowservice.UpdateSkillBindingsInput, error) {
	if len(raw.WorkflowIDs) == 0 {
		return workflowservice.UpdateSkillBindingsInput{}, fmt.Errorf("workflow_ids must not be empty")
	}

	workflowIDs := make([]uuid.UUID, 0, len(raw.WorkflowIDs))
	for _, item := range raw.WorkflowIDs {
		parsed, err := uuid.Parse(strings.TrimSpace(item))
		if err != nil {
			return workflowservice.UpdateSkillBindingsInput{}, fmt.Errorf("workflow_ids must contain UUID values")
		}
		workflowIDs = append(workflowIDs, parsed)
	}

	return workflowservice.UpdateSkillBindingsInput{
		SkillID:     skillID,
		WorkflowIDs: workflowIDs,
	}, nil
}

func parseSkillBundleFileRequests(
	items []rawSkillBundleFileRequest,
) ([]workflowservice.SkillBundleFileInput, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("files must not be empty")
	}

	files := make([]workflowservice.SkillBundleFileInput, 0, len(items))
	for _, item := range items {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			return nil, fmt.Errorf("files.path must not be empty")
		}
		content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(item.ContentBase64))
		if err != nil {
			return nil, fmt.Errorf("files.content_base64 must be valid base64")
		}
		files = append(files, workflowservice.SkillBundleFileInput{
			Path:         path,
			Content:      content,
			IsExecutable: item.IsExecutable,
			MediaType:    strings.TrimSpace(item.MediaType),
		})
	}
	return files, nil
}
