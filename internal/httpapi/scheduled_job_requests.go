package httpapi

import (
	"fmt"
	"strings"

	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	"github.com/google/uuid"
)

type rawCreateScheduledJobRequest struct {
	Name           string                               `json:"name"`
	CronExpression string                               `json:"cron_expression"`
	TicketTemplate rawScheduledJobTicketTemplateRequest `json:"ticket_template"`
	IsEnabled      *bool                                `json:"is_enabled"`
}

type rawUpdateScheduledJobRequest struct {
	Name           *string                               `json:"name"`
	CronExpression *string                               `json:"cron_expression"`
	TicketTemplate *rawScheduledJobTicketTemplateRequest `json:"ticket_template"`
	IsEnabled      *bool                                 `json:"is_enabled"`
}

type rawScheduledJobTicketTemplateRequest struct {
	Title       string                            `json:"title"`
	Description string                            `json:"description"`
	Status      string                            `json:"status"`
	Priority    string                            `json:"priority"`
	Type        string                            `json:"type"`
	CreatedBy   string                            `json:"created_by"`
	BudgetUSD   *float64                          `json:"budget_usd"`
	RepoScopes  []rawCreateTicketRepoScopeRequest `json:"repo_scopes"`
}

func parseCreateScheduledJobRequest(projectID uuid.UUID, raw rawCreateScheduledJobRequest) (scheduledjobservice.CreateInput, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return scheduledjobservice.CreateInput{}, fmt.Errorf("name must not be empty")
	}
	cronExpression := strings.TrimSpace(raw.CronExpression)
	if cronExpression == "" {
		return scheduledjobservice.CreateInput{}, fmt.Errorf("cron_expression must not be empty")
	}

	ticketTemplate, err := parseScheduledJobTicketTemplateRequest(raw.TicketTemplate)
	if err != nil {
		return scheduledjobservice.CreateInput{}, err
	}
	if strings.TrimSpace(ticketTemplate.Status) == "" {
		return scheduledjobservice.CreateInput{}, fmt.Errorf("ticket_template.status must not be empty")
	}

	input := scheduledjobservice.CreateInput{
		ProjectID:      projectID,
		Name:           name,
		CronExpression: cronExpression,
		TicketTemplate: ticketTemplate,
		IsEnabled:      true,
	}
	if raw.IsEnabled != nil {
		input.IsEnabled = *raw.IsEnabled
	}

	return input, nil
}

func parseUpdateScheduledJobRequest(jobID uuid.UUID, raw rawUpdateScheduledJobRequest) (scheduledjobservice.UpdateInput, error) {
	input := scheduledjobservice.UpdateInput{JobID: jobID}

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return scheduledjobservice.UpdateInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = scheduledjobservice.Some(name)
	}
	if raw.CronExpression != nil {
		cronExpression := strings.TrimSpace(*raw.CronExpression)
		if cronExpression == "" {
			return scheduledjobservice.UpdateInput{}, fmt.Errorf("cron_expression must not be empty")
		}
		input.CronExpression = scheduledjobservice.Some(cronExpression)
	}
	if raw.TicketTemplate != nil {
		ticketTemplate, err := parseScheduledJobTicketTemplateRequest(*raw.TicketTemplate)
		if err != nil {
			return scheduledjobservice.UpdateInput{}, err
		}
		if strings.TrimSpace(ticketTemplate.Status) == "" {
			return scheduledjobservice.UpdateInput{}, fmt.Errorf("ticket_template.status must not be empty")
		}
		input.TicketTemplate = scheduledjobservice.Some(ticketTemplate)
	}
	if raw.IsEnabled != nil {
		input.IsEnabled = scheduledjobservice.Some(*raw.IsEnabled)
	}

	return input, nil
}

func parseScheduledJobTicketTemplateRequest(
	raw rawScheduledJobTicketTemplateRequest,
) (scheduledjobservice.TicketTemplate, error) {
	template := map[string]any{
		"title": raw.Title,
	}
	if strings.TrimSpace(raw.Description) != "" {
		template["description"] = raw.Description
	}
	if strings.TrimSpace(raw.Status) != "" {
		template["status"] = raw.Status
	}
	if strings.TrimSpace(raw.Priority) != "" {
		template["priority"] = raw.Priority
	}
	if strings.TrimSpace(raw.Type) != "" {
		template["type"] = raw.Type
	}
	if strings.TrimSpace(raw.CreatedBy) != "" {
		template["created_by"] = raw.CreatedBy
	}
	if raw.BudgetUSD != nil {
		template["budget_usd"] = *raw.BudgetUSD
	}
	if len(raw.RepoScopes) > 0 {
		repoScopes := make([]map[string]any, 0, len(raw.RepoScopes))
		for _, scope := range raw.RepoScopes {
			item := map[string]any{"repo_id": scope.RepoID}
			if scope.BranchName != nil {
				item["branch_name"] = *scope.BranchName
			}
			repoScopes = append(repoScopes, item)
		}
		template["repo_scopes"] = repoScopes
	}

	return scheduledjobservice.ParseRawTicketTemplate(template)
}
