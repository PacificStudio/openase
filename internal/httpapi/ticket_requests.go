package httpapi

import (
	"fmt"
	"strings"

	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type rawCreateTicketRequest struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	StatusID       *string  `json:"status_id"`
	Priority       *string  `json:"priority"`
	Type           *string  `json:"type"`
	WorkflowID     *string  `json:"workflow_id"`
	CreatedBy      *string  `json:"created_by"`
	ParentTicketID *string  `json:"parent_ticket_id"`
	ExternalRef    *string  `json:"external_ref"`
	BudgetUSD      *float64 `json:"budget_usd"`
}

type rawUpdateTicketRequest struct {
	Title          *string  `json:"title"`
	Description    *string  `json:"description"`
	StatusID       *string  `json:"status_id"`
	Priority       *string  `json:"priority"`
	Type           *string  `json:"type"`
	WorkflowID     *string  `json:"workflow_id"`
	CreatedBy      *string  `json:"created_by"`
	ParentTicketID *string  `json:"parent_ticket_id"`
	ExternalRef    *string  `json:"external_ref"`
	BudgetUSD      *float64 `json:"budget_usd"`
}

type rawAddDependencyRequest struct {
	TargetTicketID string `json:"target_ticket_id"`
	Type           string `json:"type"`
}

func parseCreateTicketRequest(projectID uuid.UUID, raw rawCreateTicketRequest) (ticketservice.CreateInput, error) {
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return ticketservice.CreateInput{}, fmt.Errorf("title must not be empty")
	}

	statusID, err := parseOptionalUUIDString("status_id", raw.StatusID)
	if err != nil {
		return ticketservice.CreateInput{}, err
	}
	workflowID, err := parseOptionalUUIDString("workflow_id", raw.WorkflowID)
	if err != nil {
		return ticketservice.CreateInput{}, err
	}
	parentTicketID, err := parseOptionalUUIDString("parent_ticket_id", raw.ParentTicketID)
	if err != nil {
		return ticketservice.CreateInput{}, err
	}

	priority := entticket.DefaultPriority
	if raw.Priority != nil {
		priority, err = parseTicketPriority(*raw.Priority)
		if err != nil {
			return ticketservice.CreateInput{}, err
		}
	}

	ticketType := entticket.DefaultType
	if raw.Type != nil {
		ticketType, err = parseTicketType(*raw.Type)
		if err != nil {
			return ticketservice.CreateInput{}, err
		}
	}

	input := ticketservice.CreateInput{
		ProjectID:      projectID,
		Title:          title,
		Description:    strings.TrimSpace(raw.Description),
		StatusID:       statusID,
		Priority:       priority,
		Type:           ticketType,
		WorkflowID:     workflowID,
		ParentTicketID: parentTicketID,
	}
	if raw.CreatedBy != nil {
		input.CreatedBy = strings.TrimSpace(*raw.CreatedBy)
	}
	if raw.ExternalRef != nil {
		input.ExternalRef = strings.TrimSpace(*raw.ExternalRef)
	}
	if raw.BudgetUSD != nil {
		if *raw.BudgetUSD < 0 {
			return ticketservice.CreateInput{}, fmt.Errorf("budget_usd must be greater than or equal to zero")
		}
		input.BudgetUSD = *raw.BudgetUSD
	}

	return input, nil
}

func parseUpdateTicketRequest(ticketID uuid.UUID, raw rawUpdateTicketRequest) (ticketservice.UpdateInput, error) {
	input := ticketservice.UpdateInput{TicketID: ticketID}

	if raw.Title != nil {
		title := strings.TrimSpace(*raw.Title)
		if title == "" {
			return ticketservice.UpdateInput{}, fmt.Errorf("title must not be empty")
		}
		input.Title = ticketservice.Some(title)
	}
	if raw.Description != nil {
		input.Description = ticketservice.Some(strings.TrimSpace(*raw.Description))
	}
	if raw.StatusID != nil {
		statusID, err := parseUUIDString("status_id", *raw.StatusID)
		if err != nil {
			return ticketservice.UpdateInput{}, err
		}
		input.StatusID = ticketservice.Some(statusID)
	}
	if raw.Priority != nil {
		priority, err := parseTicketPriority(*raw.Priority)
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
	if raw.CreatedBy != nil {
		input.CreatedBy = ticketservice.Some(strings.TrimSpace(*raw.CreatedBy))
	}
	if raw.ParentTicketID != nil {
		parentTicketID, err := parseOptionalUUIDString("parent_ticket_id", raw.ParentTicketID)
		if err != nil {
			return ticketservice.UpdateInput{}, err
		}
		input.ParentTicketID = ticketservice.Some(parentTicketID)
	}
	if raw.ExternalRef != nil {
		input.ExternalRef = ticketservice.Some(strings.TrimSpace(*raw.ExternalRef))
	}
	if raw.BudgetUSD != nil {
		if *raw.BudgetUSD < 0 {
			return ticketservice.UpdateInput{}, fmt.Errorf("budget_usd must be greater than or equal to zero")
		}
		input.BudgetUSD = ticketservice.Some(*raw.BudgetUSD)
	}

	return input, nil
}

func parseAddDependencyRequest(ticketID uuid.UUID, raw rawAddDependencyRequest) (ticketservice.AddDependencyInput, error) {
	targetTicketID, err := parseUUIDString("target_ticket_id", raw.TargetTicketID)
	if err != nil {
		return ticketservice.AddDependencyInput{}, err
	}

	dependencyType, err := parseDependencyType(raw.Type)
	if err != nil {
		return ticketservice.AddDependencyInput{}, err
	}

	return ticketservice.AddDependencyInput{
		TicketID:       ticketID,
		TargetTicketID: targetTicketID,
		Type:           dependencyType,
	}, nil
}

func parseTicketID(c echo.Context) (uuid.UUID, error) {
	return parseUUIDPathParamValue(c, "ticketId")
}

func parseDependencyID(c echo.Context) (uuid.UUID, error) {
	return parseUUIDPathParamValue(c, "dependencyId")
}

func parseTicketPriority(raw string) (entticket.Priority, error) {
	priority := entticket.Priority(strings.ToLower(strings.TrimSpace(raw)))
	if err := entticket.PriorityValidator(priority); err != nil {
		return "", fmt.Errorf("priority must be one of urgent, high, medium, low")
	}

	return priority, nil
}

func parseTicketType(raw string) (entticket.Type, error) {
	ticketType := entticket.Type(strings.ToLower(strings.TrimSpace(raw)))
	if err := entticket.TypeValidator(ticketType); err != nil {
		return "", fmt.Errorf("type must be one of feature, bugfix, refactor, chore, epic")
	}

	return ticketType, nil
}

func parseDependencyType(raw string) (entticketdependency.Type, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "blocks":
		return entticketdependency.TypeBlocks, nil
	case "sub_issue", "sub-issue":
		return entticketdependency.TypeSubIssue, nil
	default:
		return "", fmt.Errorf("type must be one of blocks, sub_issue")
	}
}

func parseCSVQueryValues(c echo.Context, name string) []string {
	values := c.QueryParams()[name]
	if len(values) == 0 {
		return nil
	}

	parsed := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			parsed = append(parsed, trimmed)
		}
	}

	return parsed
}
