package httpapi

import (
	"fmt"
	"net/url"
	"strings"

	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type rawCreateTicketRequest struct {
	Title          string                            `json:"title"`
	Description    string                            `json:"description"`
	StatusID       *string                           `json:"status_id"`
	Archived       *bool                             `json:"archived"`
	Priority       *string                           `json:"priority"`
	Type           *string                           `json:"type"`
	WorkflowID     *string                           `json:"workflow_id"`
	RepoScopes     []rawCreateTicketRepoScopeRequest `json:"repo_scopes"`
	CreatedBy      *string                           `json:"created_by"`
	ParentTicketID *string                           `json:"parent_ticket_id"`
	ExternalRef    *string                           `json:"external_ref"`
	BudgetUSD      *float64                          `json:"budget_usd"`
}

type rawCreateTicketRepoScopeRequest struct {
	RepoID     string  `json:"repo_id"`
	BranchName *string `json:"branch_name"`
}

type rawUpdateTicketRequest struct {
	Title          *string  `json:"title"`
	Description    *string  `json:"description"`
	StatusID       *string  `json:"status_id"`
	Archived       *bool    `json:"archived"`
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

type parsedAddDependencyRequest struct {
	Input ticketservice.AddDependencyInput
}

type rawAddExternalLinkRequest struct {
	Type       string  `json:"type"`
	URL        string  `json:"url"`
	ExternalID string  `json:"external_id"`
	Title      *string `json:"title"`
	Status     *string `json:"status"`
	Relation   *string `json:"relation"`
}

type rawCreateTicketCommentRequest struct {
	Body      string  `json:"body"`
	CreatedBy *string `json:"created_by"`
}

type rawUpdateTicketCommentRequest struct {
	Body       string  `json:"body"`
	EditedBy   *string `json:"edited_by"`
	EditReason *string `json:"edit_reason"`
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

	var priority *ticketservice.Priority
	if raw.Priority != nil {
		priority, err = parseOptionalTicketPriority(*raw.Priority)
		if err != nil {
			return ticketservice.CreateInput{}, err
		}
	}

	ticketType := ticketservice.DefaultType
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
		Archived:       raw.Archived != nil && *raw.Archived,
		Priority:       priority,
		Type:           ticketType,
		WorkflowID:     workflowID,
		ParentTicketID: parentTicketID,
	}
	if len(raw.RepoScopes) > 0 {
		input.RepoScopes = make([]ticketservice.CreateRepoScopeInput, 0, len(raw.RepoScopes))
		for index, scope := range raw.RepoScopes {
			repoID, err := parseUUIDString(fmt.Sprintf("repo_scopes[%d].repo_id", index), scope.RepoID)
			if err != nil {
				return ticketservice.CreateInput{}, err
			}
			var branchName *string
			if scope.BranchName != nil {
				trimmed := strings.TrimSpace(*scope.BranchName)
				if trimmed != "" {
					branchName = &trimmed
				}
			}
			input.RepoScopes = append(input.RepoScopes, ticketservice.CreateRepoScopeInput{
				RepoID:     repoID,
				BranchName: branchName,
			})
		}
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

func parseAddDependencyRequest(ticketID uuid.UUID, raw rawAddDependencyRequest) (parsedAddDependencyRequest, error) {
	targetTicketID, err := parseUUIDString("target_ticket_id", raw.TargetTicketID)
	if err != nil {
		return parsedAddDependencyRequest{}, err
	}

	switch strings.ToLower(strings.TrimSpace(raw.Type)) {
	case "blocks":
		return parsedAddDependencyRequest{
			Input: ticketservice.AddDependencyInput{
				TicketID:       ticketID,
				TargetTicketID: targetTicketID,
				Type:           ticketservice.DependencyTypeBlocks,
			},
		}, nil
	case "blocked_by", "blocked-by":
		return parsedAddDependencyRequest{
			Input: ticketservice.AddDependencyInput{
				TicketID:       targetTicketID,
				TargetTicketID: ticketID,
				Type:           ticketservice.DependencyTypeBlocks,
			},
		}, nil
	case "sub_issue", "sub-issue":
		return parsedAddDependencyRequest{
			Input: ticketservice.AddDependencyInput{
				TicketID:       ticketID,
				TargetTicketID: targetTicketID,
				Type:           ticketservice.DependencyTypeSubIssue,
			},
		}, nil
	default:
		return parsedAddDependencyRequest{}, fmt.Errorf("type must be one of blocks, blocked_by, sub_issue")
	}
}

func parseAddExternalLinkRequest(ticketID uuid.UUID, raw rawAddExternalLinkRequest) (ticketservice.AddExternalLinkInput, error) {
	linkType, err := parseExternalLinkType(raw.Type)
	if err != nil {
		return ticketservice.AddExternalLinkInput{}, err
	}

	trimmedURL := strings.TrimSpace(raw.URL)
	parsedURL, err := url.ParseRequestURI(trimmedURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ticketservice.AddExternalLinkInput{}, fmt.Errorf("url must be a valid absolute URL")
	}

	externalID := strings.TrimSpace(raw.ExternalID)
	if externalID == "" {
		return ticketservice.AddExternalLinkInput{}, fmt.Errorf("external_id must not be empty")
	}

	relation := ticketservice.DefaultExternalLinkRelation
	if raw.Relation != nil {
		relation, err = parseExternalLinkRelation(*raw.Relation)
		if err != nil {
			return ticketservice.AddExternalLinkInput{}, err
		}
	}

	input := ticketservice.AddExternalLinkInput{
		TicketID:   ticketID,
		LinkType:   linkType,
		URL:        trimmedURL,
		ExternalID: externalID,
		Relation:   relation,
	}
	if raw.Title != nil {
		input.Title = strings.TrimSpace(*raw.Title)
	}
	if raw.Status != nil {
		input.Status = strings.TrimSpace(*raw.Status)
	}

	return input, nil
}

func parseCreateTicketCommentRequest(ticketID uuid.UUID, raw rawCreateTicketCommentRequest) (ticketservice.AddCommentInput, error) {
	body := strings.TrimSpace(raw.Body)
	if body == "" {
		return ticketservice.AddCommentInput{}, fmt.Errorf("body must not be empty")
	}

	input := ticketservice.AddCommentInput{
		TicketID: ticketID,
		Body:     body,
	}
	if raw.CreatedBy != nil {
		input.CreatedBy = strings.TrimSpace(*raw.CreatedBy)
	}

	return input, nil
}

func parseUpdateTicketCommentRequest(ticketID uuid.UUID, commentID uuid.UUID, raw rawUpdateTicketCommentRequest) (ticketservice.UpdateCommentInput, error) {
	body := strings.TrimSpace(raw.Body)
	if body == "" {
		return ticketservice.UpdateCommentInput{}, fmt.Errorf("body must not be empty")
	}

	input := ticketservice.UpdateCommentInput{
		TicketID:  ticketID,
		CommentID: commentID,
		Body:      body,
	}
	if raw.EditedBy != nil {
		input.EditedBy = strings.TrimSpace(*raw.EditedBy)
	}
	if raw.EditReason != nil {
		input.EditReason = strings.TrimSpace(*raw.EditReason)
	}

	return input, nil
}

func parseTicketID(c echo.Context) (uuid.UUID, error) {
	return parseUUIDPathParamValue(c, "ticketId")
}

func parseDependencyID(c echo.Context) (uuid.UUID, error) {
	return parseUUIDPathParamValue(c, "dependencyId")
}

func parseCommentID(c echo.Context) (uuid.UUID, error) {
	return parseUUIDPathParamValue(c, "commentId")
}

func parseExternalLinkID(c echo.Context) (uuid.UUID, error) {
	return parseUUIDPathParamValue(c, "externalLinkId")
}

func parseTicketPriority(raw string) (ticketservice.Priority, error) {
	return ticketservice.ParsePriority(raw)
}

func parseOptionalTicketPriority(raw string) (*ticketservice.Priority, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	priority, err := parseTicketPriority(trimmed)
	if err != nil {
		return nil, err
	}
	return &priority, nil
}

func parseTicketType(raw string) (ticketservice.Type, error) {
	return ticketservice.ParseType(raw)
}

func parseExternalLinkType(raw string) (ticketservice.ExternalLinkType, error) {
	return ticketservice.ParseExternalLinkType(raw)
}

func parseExternalLinkRelation(raw string) (ticketservice.ExternalLinkRelation, error) {
	return ticketservice.ParseExternalLinkRelation(raw)
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
