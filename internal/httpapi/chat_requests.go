package httpapi

import (
	"strings"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/google/uuid"
)

type rawCreateConversationRequest struct {
	Source     string `json:"source"`
	ProviderID string `json:"provider_id"`
	Context    struct {
		ProjectID string `json:"project_id"`
	} `json:"context"`
}

type rawConversationTurnRequest struct {
	Message            string                                           `json:"message"`
	Focus              *chatservice.RawProjectConversationFocus         `json:"focus"`
	WorkspaceFileDraft *rawProjectConversationWorkspaceFileDraftContext `json:"workspace_file_draft,omitempty"`
}

type rawInterruptResponseRequest struct {
	Decision *string        `json:"decision"`
	Answer   map[string]any `json:"answer"`
}

type createProjectConversationRequest struct {
	Source     chatdomain.Source
	ProjectID  uuid.UUID
	ProviderID uuid.UUID
}

type projectConversationTurnRequest struct {
	Message            string
	Focus              *chatservice.ProjectConversationFocus
	WorkspaceFileDraft *chatservice.ProjectConversationWorkspaceFileDraftContext
}

type projectConversationWorkspaceTreeRequest struct {
	RepoPath string
	Path     string
}

type projectConversationWorkspaceFileRequest struct {
	RepoPath string
	Path     string
}

type rawUpdateProjectConversationWorkspaceFileRequest struct {
	RepoPath     string `json:"repo_path"`
	Path         string `json:"path"`
	BaseRevision string `json:"base_revision"`
	Content      string `json:"content"`
	Encoding     string `json:"encoding"`
	LineEnding   string `json:"line_ending"`
}

type updateProjectConversationWorkspaceFileRequest struct {
	File chatservice.ProjectConversationWorkspaceFileSaveInput
}

type rawProjectConversationWorkspaceFileDraftContext struct {
	RepoPath   string `json:"repo_path"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	Encoding   string `json:"encoding"`
	LineEnding string `json:"line_ending"`
}

type rawCreateProjectConversationTerminalSessionRequest struct {
	Mode     string  `json:"mode"`
	RepoPath *string `json:"repo_path"`
	CWDPath  *string `json:"cwd_path"`
	Cols     *int    `json:"cols"`
	Rows     *int    `json:"rows"`
}

type createProjectConversationTerminalSessionRequest struct {
	Terminal chatdomain.OpenTerminalSessionInput
}

func parseCreateProjectConversationRequest(raw rawCreateConversationRequest) (createProjectConversationRequest, error) {
	source, err := chatdomain.ParseSource(raw.Source)
	if err != nil {
		return createProjectConversationRequest{}, err
	}
	projectID, err := parseUUIDString("context.project_id", raw.Context.ProjectID)
	if err != nil {
		return createProjectConversationRequest{}, err
	}
	providerID, err := parseUUIDString("provider_id", raw.ProviderID)
	if err != nil {
		return createProjectConversationRequest{}, err
	}

	return createProjectConversationRequest{
		Source:     source,
		ProjectID:  projectID,
		ProviderID: providerID,
	}, nil
}

func parseProjectConversationTurnRequest(raw rawConversationTurnRequest) (projectConversationTurnRequest, error) {
	message := strings.TrimSpace(raw.Message)
	if message == "" {
		return projectConversationTurnRequest{}, writeableError("message must not be empty")
	}
	focus, err := chatservice.ParseProjectConversationFocus(raw.Focus)
	if err != nil {
		return projectConversationTurnRequest{}, writeableError(err.Error())
	}
	workspaceFileDraft, err := parseProjectConversationWorkspaceFileDraftContext(raw.WorkspaceFileDraft)
	if err != nil {
		return projectConversationTurnRequest{}, writeableError(err.Error())
	}
	return projectConversationTurnRequest{
		Message:            message,
		Focus:              focus,
		WorkspaceFileDraft: workspaceFileDraft,
	}, nil
}

func parseInterruptResponseRequest(raw rawInterruptResponseRequest) chatdomain.InterruptResponse {
	return chatdomain.InterruptResponse{
		Decision: raw.Decision,
		Answer:   raw.Answer,
	}
}

func parseProjectConversationWorkspaceTreeRequest(
	repoPath string,
	path string,
) (projectConversationWorkspaceTreeRequest, error) {
	trimmedRepoPath := strings.TrimSpace(repoPath)
	if trimmedRepoPath == "" {
		return projectConversationWorkspaceTreeRequest{}, writeableError("repo_path must not be empty")
	}
	return projectConversationWorkspaceTreeRequest{
		RepoPath: trimmedRepoPath,
		Path:     strings.TrimSpace(path),
	}, nil
}

func parseProjectConversationWorkspaceFileRequest(
	repoPath string,
	path string,
) (projectConversationWorkspaceFileRequest, error) {
	trimmedRepoPath := strings.TrimSpace(repoPath)
	if trimmedRepoPath == "" {
		return projectConversationWorkspaceFileRequest{}, writeableError("repo_path must not be empty")
	}
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return projectConversationWorkspaceFileRequest{}, writeableError("path must not be empty")
	}
	return projectConversationWorkspaceFileRequest{
		RepoPath: trimmedRepoPath,
		Path:     trimmedPath,
	}, nil
}

func parseUpdateProjectConversationWorkspaceFileRequest(
	raw rawUpdateProjectConversationWorkspaceFileRequest,
) (updateProjectConversationWorkspaceFileRequest, error) {
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return updateProjectConversationWorkspaceFileRequest{}, writeableError(err.Error())
	}
	filePath, err := chatservice.ParseWorkspaceFilePath(raw.Path)
	if err != nil {
		return updateProjectConversationWorkspaceFileRequest{}, writeableError(err.Error())
	}
	baseRevision, err := chatservice.ParseWorkspaceFileRevision(raw.BaseRevision)
	if err != nil {
		return updateProjectConversationWorkspaceFileRequest{}, writeableError(err.Error())
	}
	content, err := chatservice.ParseWorkspaceTextContent(raw.Content)
	if err != nil {
		return updateProjectConversationWorkspaceFileRequest{}, writeableError(err.Error())
	}
	encoding, err := chatservice.ParseWorkspaceEncoding(raw.Encoding)
	if err != nil {
		return updateProjectConversationWorkspaceFileRequest{}, writeableError(err.Error())
	}
	lineEnding, err := chatservice.ParseWorkspaceLineEnding(raw.LineEnding)
	if err != nil {
		return updateProjectConversationWorkspaceFileRequest{}, writeableError(err.Error())
	}
	return updateProjectConversationWorkspaceFileRequest{
		File: chatservice.ProjectConversationWorkspaceFileSaveInput{
			RepoPath:     repoPath,
			Path:         filePath,
			BaseRevision: baseRevision,
			Content:      content,
			Encoding:     encoding,
			LineEnding:   lineEnding,
		},
	}, nil
}

func parseProjectConversationWorkspaceFileDraftContext(
	raw *rawProjectConversationWorkspaceFileDraftContext,
) (*chatservice.ProjectConversationWorkspaceFileDraftContext, error) {
	if raw == nil {
		return nil, nil
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return nil, err
	}
	filePath, err := chatservice.ParseWorkspaceFilePath(raw.Path)
	if err != nil {
		return nil, err
	}
	content, err := chatservice.ParseWorkspaceTextContent(raw.Content)
	if err != nil {
		return nil, err
	}
	encoding, err := chatservice.ParseWorkspaceEncoding(raw.Encoding)
	if err != nil {
		return nil, err
	}
	lineEnding, err := chatservice.ParseWorkspaceLineEnding(raw.LineEnding)
	if err != nil {
		return nil, err
	}
	return &chatservice.ProjectConversationWorkspaceFileDraftContext{
		RepoPath:   repoPath,
		Path:       filePath,
		Content:    content,
		Encoding:   encoding,
		LineEnding: lineEnding,
	}, nil
}

func parseCreateProjectConversationTerminalSessionRequest(
	raw rawCreateProjectConversationTerminalSessionRequest,
) (createProjectConversationTerminalSessionRequest, error) {
	parsed, err := chatdomain.ParseOpenTerminalSessionInput(chatdomain.OpenTerminalSessionRawInput{
		Mode:     raw.Mode,
		RepoPath: raw.RepoPath,
		CWDPath:  raw.CWDPath,
		Cols:     raw.Cols,
		Rows:     raw.Rows,
	})
	if err != nil {
		return createProjectConversationTerminalSessionRequest{}, writeableError(err.Error())
	}
	return createProjectConversationTerminalSessionRequest{Terminal: parsed}, nil
}

type writeableError string

func (e writeableError) Error() string { return string(e) }
