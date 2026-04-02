package httpapi

import (
	"context"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

type rawAgentCreateTicketRequest struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	StatusID       *string  `json:"status_id"`
	Priority       *string  `json:"priority"`
	Type           *string  `json:"type"`
	WorkflowID     *string  `json:"workflow_id"`
	ParentTicketID *string  `json:"parent_ticket_id"`
	ExternalRef    *string  `json:"external_ref"`
	BudgetUSD      *float64 `json:"budget_usd"`
}

type rawAgentUpdateTicketRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ExternalRef *string `json:"external_ref"`
	StatusID    *string `json:"status_id"`
	StatusName  *string `json:"status_name"`
}

type rawAgentReportUsageRequest struct {
	InputTokens  *int64   `json:"input_tokens"`
	OutputTokens *int64   `json:"output_tokens"`
	CostUSD      *float64 `json:"cost_usd"`
}

type rawAgentTicketCommentRequest struct {
	Body string `json:"body"`
}

type rawAgentProjectPatchRequest struct {
	Description *string `json:"description"`
}

type rawAgentCreateProjectUpdateThreadRequest struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type rawAgentUpdateProjectUpdateThreadRequest struct {
	Status     string  `json:"status"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	EditReason *string `json:"edit_reason"`
}

type rawAgentCreateProjectUpdateCommentRequest struct {
	Body string `json:"body"`
}

type rawAgentUpdateProjectUpdateCommentRequest struct {
	Body       string  `json:"body"`
	EditReason *string `json:"edit_reason"`
}

type agentStatusNameResolutionError struct {
	err error
}

func (e agentStatusNameResolutionError) Error() string { return e.err.Error() }

func (e agentStatusNameResolutionError) Unwrap() error { return e.err }

func parseAgentCreateTicketRequest(projectID uuid.UUID, raw rawAgentCreateTicketRequest) (ticketservice.CreateInput, error) {
	return parseCreateTicketRequest(projectID, rawCreateTicketRequest{
		Title:          raw.Title,
		Description:    raw.Description,
		StatusID:       raw.StatusID,
		Priority:       raw.Priority,
		Type:           raw.Type,
		WorkflowID:     raw.WorkflowID,
		ParentTicketID: raw.ParentTicketID,
		ExternalRef:    raw.ExternalRef,
		BudgetUSD:      raw.BudgetUSD,
	})
}

func parseAgentUpdateTicketRequest(
	ctx context.Context,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	createdBy string,
	raw rawAgentUpdateTicketRequest,
	resolveStatusID func(context.Context, uuid.UUID, string) (uuid.UUID, error),
) (ticketservice.UpdateInput, error) {
	if raw.StatusID != nil && raw.StatusName != nil {
		return ticketservice.UpdateInput{}, writeableError("status_id and status_name cannot be provided together")
	}

	input := ticketservice.UpdateInput{TicketID: ticketID}
	if raw.Title != nil {
		title := strings.TrimSpace(*raw.Title)
		if title == "" {
			return ticketservice.UpdateInput{}, writeableError("title must not be empty")
		}
		input.Title = ticketservice.Some(title)
	}
	if raw.Description != nil {
		input.Description = ticketservice.Some(strings.TrimSpace(*raw.Description))
	}
	if raw.ExternalRef != nil {
		input.ExternalRef = ticketservice.Some(strings.TrimSpace(*raw.ExternalRef))
	}
	if raw.StatusID != nil {
		statusID, err := parseUUIDString("status_id", *raw.StatusID)
		if err != nil {
			return ticketservice.UpdateInput{}, err
		}
		input.StatusID = ticketservice.Some(statusID)
	}
	if raw.StatusName != nil {
		statusID, err := resolveStatusID(ctx, projectID, *raw.StatusName)
		if err != nil {
			return ticketservice.UpdateInput{}, agentStatusNameResolutionError{err: err}
		}
		input.StatusID = ticketservice.Some(statusID)
	}
	input.CreatedBy = ticketservice.Some(createdBy)
	return input, nil
}

func parseAgentReportUsageRequest(raw rawAgentReportUsageRequest) ticketing.RawUsageDelta {
	return ticketing.RawUsageDelta{
		InputTokens:  raw.InputTokens,
		OutputTokens: raw.OutputTokens,
		CostUSD:      raw.CostUSD,
	}
}

func parseAgentCreateTicketCommentRequest(ticketID uuid.UUID, createdBy string, raw rawAgentTicketCommentRequest) (ticketservice.AddCommentInput, error) {
	return parseCreateTicketCommentRequest(ticketID, rawCreateTicketCommentRequest{
		Body:      raw.Body,
		CreatedBy: stringPointer(createdBy),
	})
}

func parseAgentUpdateTicketCommentRequest(ticketID uuid.UUID, commentID uuid.UUID, editedBy string, raw rawAgentTicketCommentRequest) (ticketservice.UpdateCommentInput, error) {
	return parseUpdateTicketCommentRequest(ticketID, commentID, rawUpdateTicketCommentRequest{
		Body:       raw.Body,
		EditedBy:   stringPointer(editedBy),
		EditReason: stringPointer("agent_workpad_update"),
	})
}

func parseAgentProjectPatchRequest(
	projectID uuid.UUID,
	current domain.Project,
	raw rawAgentProjectPatchRequest,
) (domain.UpdateProject, error) {
	if raw.Description == nil {
		return domain.UpdateProject{}, writeableError("description is required")
	}

	return domain.ParseUpdateProject(projectID, current.OrganizationID, domain.ProjectInput{
		Name:                   current.Name,
		Slug:                   current.Slug,
		Description:            strings.TrimSpace(*raw.Description),
		Status:                 current.Status.String(),
		DefaultAgentProviderID: uuidToStringPointer(current.DefaultAgentProviderID),
		AccessibleMachineIDs:   uuidSliceToStrings(current.AccessibleMachineIDs),
		MaxConcurrentAgents:    intPointer(current.MaxConcurrentAgents),
	})
}

func parseAgentCreateProjectUpdateThreadRequest(
	projectID uuid.UUID,
	createdBy string,
	raw rawAgentCreateProjectUpdateThreadRequest,
) (projectupdateservice.AddThreadInput, error) {
	return parseCreateProjectUpdateThreadRequest(projectID, rawCreateProjectUpdateThreadRequest{
		Status:    raw.Status,
		Title:     raw.Title,
		Body:      raw.Body,
		CreatedBy: stringPointer(createdBy),
	})
}

func parseAgentUpdateProjectUpdateThreadRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	editedBy string,
	raw rawAgentUpdateProjectUpdateThreadRequest,
) (projectupdateservice.UpdateThreadInput, error) {
	return parseUpdateProjectUpdateThreadRequest(projectID, threadID, rawUpdateProjectUpdateThreadRequest{
		Status:     raw.Status,
		Title:      raw.Title,
		Body:       raw.Body,
		EditedBy:   stringPointer(editedBy),
		EditReason: raw.EditReason,
	})
}

func parseAgentCreateProjectUpdateCommentRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	createdBy string,
	raw rawAgentCreateProjectUpdateCommentRequest,
) (projectupdateservice.AddCommentInput, error) {
	return parseCreateProjectUpdateCommentRequest(projectID, threadID, rawCreateProjectUpdateCommentRequest{
		Body:      raw.Body,
		CreatedBy: stringPointer(createdBy),
	})
}

func parseAgentUpdateProjectUpdateCommentRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	commentID uuid.UUID,
	editedBy string,
	raw rawAgentUpdateProjectUpdateCommentRequest,
) (projectupdateservice.UpdateCommentInput, error) {
	return parseUpdateProjectUpdateCommentRequest(projectID, threadID, commentID, rawUpdateProjectUpdateCommentRequest{
		Body:       raw.Body,
		EditedBy:   stringPointer(editedBy),
		EditReason: raw.EditReason,
	})
}
