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

type writeableError string

func (e writeableError) Error() string { return string(e) }
