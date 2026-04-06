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
	Archived       *bool    `json:"archived"`
	Priority       *string  `json:"priority"`
	Type           *string  `json:"type"`
	WorkflowID     *string  `json:"workflow_id"`
	ParentTicketID *string  `json:"parent_ticket_id"`
	ExternalRef    *string  `json:"external_ref"`
	BudgetUSD      *float64 `json:"budget_usd"`
}

type rawAgentUpdateTicketRequest struct {
	Title          *string  `json:"title"`
	Description    *string  `json:"description"`
	ExternalRef    *string  `json:"external_ref"`
	StatusID       *string  `json:"status_id"`
	StatusName     *string  `json:"status_name"`
	Archived       *bool    `json:"archived"`
	Priority       *string  `json:"priority"`
	Type           *string  `json:"type"`
	WorkflowID     *string  `json:"workflow_id"`
	ParentTicketID *string  `json:"parent_ticket_id"`
	BudgetUSD      *float64 `json:"budget_usd"`
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
	Name                   *string   `json:"name"`
	Slug                   *string   `json:"slug"`
	Description            *string   `json:"description"`
	Status                 *string   `json:"status"`
	DefaultAgentProviderID *string   `json:"default_agent_provider_id"`
	AccessibleMachineIDs   *[]string `json:"accessible_machine_ids"`
	MaxConcurrentAgents    *int      `json:"max_concurrent_agents"`
	AgentRunSummaryPrompt  *string   `json:"agent_run_summary_prompt"`
}

type rawAgentCreateProjectUpdateThreadRequest struct {
	Status string  `json:"status"`
	Title  *string `json:"title"`
	Body   string  `json:"body"`
}

type rawAgentUpdateProjectUpdateThreadRequest struct {
	Status     string  `json:"status"`
	Title      *string `json:"title"`
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
	return parseCreateTicketRequest(projectID, "", rawCreateTicketRequest{
		Title:          raw.Title,
		Description:    raw.Description,
		StatusID:       raw.StatusID,
		Archived:       raw.Archived,
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
	if raw.Archived != nil {
		input.Archived = ticketservice.Some(*raw.Archived)
	}
	if raw.Priority != nil {
		priority, err := parseOptionalTicketPriority(*raw.Priority)
		if err != nil {
			return ticketservice.UpdateInput{}, err
		}
		input.Priority = ticketservice.Some(priority)
	}
	if raw.Type != nil {
		ticketType, err := parseTicketType(*raw.Type)
		if err != nil {
			return ticketservice.UpdateInput{}, err
		}
		input.Type = ticketservice.Some(ticketType)
	}
	if raw.WorkflowID != nil {
		workflowID, err := parseOptionalUUIDString("workflow_id", raw.WorkflowID)
		if err != nil {
			return ticketservice.UpdateInput{}, err
		}
		input.WorkflowID = ticketservice.Some(workflowID)
	}
	if raw.ParentTicketID != nil {
		parentTicketID, err := parseOptionalUUIDString("parent_ticket_id", raw.ParentTicketID)
		if err != nil {
			return ticketservice.UpdateInput{}, err
		}
		input.ParentTicketID = ticketservice.Some(parentTicketID)
	}
	if raw.BudgetUSD != nil {
		if *raw.BudgetUSD < 0 {
			return ticketservice.UpdateInput{}, writeableError("budget_usd must be greater than or equal to zero")
		}
		input.BudgetUSD = ticketservice.Some(*raw.BudgetUSD)
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
	return parseCreateTicketCommentRequest(ticketID, createdBy, rawCreateTicketCommentRequest(raw))
}

func parseAgentUpdateTicketCommentRequest(ticketID uuid.UUID, commentID uuid.UUID, editedBy string, raw rawAgentTicketCommentRequest) (ticketservice.UpdateCommentInput, error) {
	return parseUpdateTicketCommentRequest(ticketID, commentID, editedBy, rawUpdateTicketCommentRequest{
		Body:       raw.Body,
		EditReason: stringPointer("agent_workpad_update"),
	})
}

func parseAgentProjectPatchRequest(
	projectID uuid.UUID,
	current domain.Project,
	raw rawAgentProjectPatchRequest,
) (domain.UpdateProject, error) {
	if raw.Name == nil &&
		raw.Slug == nil &&
		raw.Description == nil &&
		raw.Status == nil &&
		raw.DefaultAgentProviderID == nil &&
		raw.AccessibleMachineIDs == nil &&
		raw.MaxConcurrentAgents == nil &&
		raw.AgentRunSummaryPrompt == nil {
		return domain.UpdateProject{}, writeableError("at least one project field must be provided")
	}

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
	if raw.Name != nil {
		request.Name = strings.TrimSpace(*raw.Name)
	}
	if raw.Slug != nil {
		request.Slug = strings.TrimSpace(*raw.Slug)
	}
	if raw.Description != nil {
		request.Description = strings.TrimSpace(*raw.Description)
	}
	if raw.Status != nil {
		request.Status = strings.TrimSpace(*raw.Status)
	}
	if raw.DefaultAgentProviderID != nil {
		request.DefaultAgentProviderID = raw.DefaultAgentProviderID
	}
	if raw.AccessibleMachineIDs != nil {
		request.AccessibleMachineIDs = cloneStringSlice(*raw.AccessibleMachineIDs)
	}
	if raw.MaxConcurrentAgents != nil {
		request.MaxConcurrentAgents = raw.MaxConcurrentAgents
	}
	if raw.AgentRunSummaryPrompt != nil {
		request.AgentRunSummaryPrompt = raw.AgentRunSummaryPrompt
	}

	return domain.ParseUpdateProject(projectID, current.OrganizationID, request)
}

func parseAgentCreateProjectUpdateThreadRequest(
	projectID uuid.UUID,
	createdBy string,
	raw rawAgentCreateProjectUpdateThreadRequest,
) (projectupdateservice.AddThreadInput, error) {
	return parseCreateProjectUpdateThreadRequest(projectID, createdBy, rawCreateProjectUpdateThreadRequest(raw))
}

func parseAgentUpdateProjectUpdateThreadRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	editedBy string,
	raw rawAgentUpdateProjectUpdateThreadRequest,
) (projectupdateservice.UpdateThreadInput, error) {
	return parseUpdateProjectUpdateThreadRequest(projectID, threadID, editedBy, rawUpdateProjectUpdateThreadRequest(raw))
}

func parseAgentCreateProjectUpdateCommentRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	createdBy string,
	raw rawAgentCreateProjectUpdateCommentRequest,
) (projectupdateservice.AddCommentInput, error) {
	return parseCreateProjectUpdateCommentRequest(projectID, threadID, createdBy, rawCreateProjectUpdateCommentRequest(raw))
}

func parseAgentUpdateProjectUpdateCommentRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	commentID uuid.UUID,
	editedBy string,
	raw rawAgentUpdateProjectUpdateCommentRequest,
) (projectupdateservice.UpdateCommentInput, error) {
	return parseUpdateProjectUpdateCommentRequest(projectID, threadID, commentID, editedBy, rawUpdateProjectUpdateCommentRequest(raw))
}
