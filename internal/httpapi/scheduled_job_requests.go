package httpapi

import (
	"fmt"
	"strings"

	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	"github.com/google/uuid"
)

type rawCreateScheduledJobRequest struct {
	Name           string         `json:"name"`
	CronExpression string         `json:"cron_expression"`
	TicketTemplate map[string]any `json:"ticket_template"`
	IsEnabled      *bool          `json:"is_enabled"`
}

type rawUpdateScheduledJobRequest struct {
	Name           *string         `json:"name"`
	CronExpression *string         `json:"cron_expression"`
	TicketTemplate *map[string]any `json:"ticket_template"`
	IsEnabled      *bool           `json:"is_enabled"`
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

	ticketTemplate, err := scheduledjobservice.ParseRawTicketTemplate(raw.TicketTemplate)
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
		ticketTemplate, err := scheduledjobservice.ParseRawTicketTemplate(*raw.TicketTemplate)
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
