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
	Message string                                   `json:"message"`
	Focus   *chatservice.RawProjectConversationFocus `json:"focus"`
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
	Message string
	Focus   *chatservice.ProjectConversationFocus
}

type projectConversationWorkspaceTreeRequest struct {
	RepoPath string
	Path     string
}

type projectConversationWorkspaceFileRequest struct {
	RepoPath string
	Path     string
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
	return projectConversationTurnRequest{
		Message: message,
		Focus:   focus,
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
