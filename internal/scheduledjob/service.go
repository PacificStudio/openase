package scheduledjob

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entscheduledjob "github.com/BetterAndBetterII/openase/ent/scheduledjob"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/robfig/cron/v3"
)

const defaultCreatedBy = "system:scheduled-job"

var (
	ErrUnavailable           = errors.New("scheduled job service unavailable")
	ErrProjectNotFound       = errors.New("project not found")
	ErrWorkflowNotFound      = errors.New("workflow not found")
	ErrScheduledJobNotFound  = errors.New("scheduled job not found")
	ErrScheduledJobConflict  = errors.New("scheduled job conflict")
	ErrStatusNotFound        = errors.New("ticket status not found")
	ErrInvalidCronExpression = errors.New("scheduled job cron expression is invalid")
	ErrInvalidTicketTemplate = errors.New("scheduled job ticket template is invalid")
)

type Optional[T any] struct {
	Set   bool
	Value T
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{Set: true, Value: value}
}

type TicketTemplate struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Status      string             `json:"status,omitempty"`
	Priority    entticket.Priority `json:"priority"`
	Type        entticket.Type     `json:"type"`
	CreatedBy   string             `json:"created_by"`
	BudgetUSD   float64            `json:"budget_usd,omitempty"`
}

type ScheduledJob struct {
	ID             uuid.UUID      `json:"id"`
	ProjectID      uuid.UUID      `json:"project_id"`
	Name           string         `json:"name"`
	CronExpression string         `json:"cron_expression"`
	TicketTemplate TicketTemplate `json:"ticket_template"`
	IsEnabled      bool           `json:"is_enabled"`
	LastRunAt      *time.Time     `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time     `json:"next_run_at,omitempty"`
}

type CreateInput struct {
	ProjectID      uuid.UUID
	Name           string
	CronExpression string
	TicketTemplate TicketTemplate
	IsEnabled      bool
}

type UpdateInput struct {
	JobID          uuid.UUID
	Name           Optional[string]
	CronExpression Optional[string]
	TicketTemplate Optional[TicketTemplate]
	IsEnabled      Optional[bool]
}

type DeleteResult struct {
	DeletedJobID uuid.UUID `json:"deleted_job_id"`
}

type TriggerResult struct {
	Job    ScheduledJob         `json:"job"`
	Ticket ticketservice.Ticket `json:"ticket"`
}

type RunDueReport struct {
	JobsScanned    int `json:"jobs_scanned"`
	TicketsCreated int `json:"tickets_created"`
}

type ticketCreator interface {
	Create(context.Context, ticketservice.CreateInput) (ticketservice.Ticket, error)
}

type Service struct {
	client     *ent.Client
	tickets    ticketCreator
	logger     *slog.Logger
	now        func() time.Time
	cronParser cron.Parser
}

func NewService(client *ent.Client, tickets ticketCreator, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		client:  client,
		tickets: tickets,
		logger:  logger.With("component", "scheduled-job-service"),
		now:     time.Now,
		cronParser: cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		),
	}
}

func (s *Service) SetNowFunc(now func() time.Time) {
	if s == nil || now == nil {
		return
	}

	s.now = now
}

func ParseRawTicketTemplate(raw map[string]any) (TicketTemplate, error) {
	if len(raw) == 0 {
		return TicketTemplate{}, fmt.Errorf("%w: ticket_template.title must not be empty", ErrInvalidTicketTemplate)
	}

	for key := range raw {
		switch key {
		case "title", "description", "status", "priority", "type", "created_by", "budget_usd":
		default:
			return TicketTemplate{}, fmt.Errorf("%w: ticket_template.%s is not supported", ErrInvalidTicketTemplate, key)
		}
	}

	title, err := parseRequiredString(raw, "title")
	if err != nil {
		return TicketTemplate{}, err
	}
	description, err := parseOptionalString(raw, "description")
	if err != nil {
		return TicketTemplate{}, err
	}
	status, err := parseOptionalString(raw, "status")
	if err != nil {
		return TicketTemplate{}, err
	}

	priority := entticket.DefaultPriority
	if value, ok := raw["priority"]; ok {
		priorityRaw, ok := value.(string)
		if !ok {
			return TicketTemplate{}, fmt.Errorf("%w: ticket_template.priority must be a string", ErrInvalidTicketTemplate)
		}
		priority = entticket.Priority(strings.ToLower(strings.TrimSpace(priorityRaw)))
		if err := entticket.PriorityValidator(priority); err != nil {
			return TicketTemplate{}, fmt.Errorf("%w: ticket_template.priority must be one of urgent, high, medium, low", ErrInvalidTicketTemplate)
		}
	}

	ticketType := entticket.DefaultType
	if value, ok := raw["type"]; ok {
		typeRaw, ok := value.(string)
		if !ok {
			return TicketTemplate{}, fmt.Errorf("%w: ticket_template.type must be a string", ErrInvalidTicketTemplate)
		}
		ticketType = entticket.Type(strings.ToLower(strings.TrimSpace(typeRaw)))
		if err := entticket.TypeValidator(ticketType); err != nil {
			return TicketTemplate{}, fmt.Errorf("%w: ticket_template.type must be one of feature, bugfix, refactor, chore, epic", ErrInvalidTicketTemplate)
		}
	}

	createdBy := defaultCreatedBy
	if value, ok := raw["created_by"]; ok {
		createdByValue, ok := value.(string)
		if !ok {
			return TicketTemplate{}, fmt.Errorf("%w: ticket_template.created_by must be a string", ErrInvalidTicketTemplate)
		}
		createdBy = strings.TrimSpace(createdByValue)
		if createdBy == "" {
			return TicketTemplate{}, fmt.Errorf("%w: ticket_template.created_by must not be empty", ErrInvalidTicketTemplate)
		}
	}

	budgetUSD := 0.0
	if value, ok := raw["budget_usd"]; ok {
		budgetUSD, err = parseBudgetUSD(value)
		if err != nil {
			return TicketTemplate{}, err
		}
	}

	return TicketTemplate{
		Title:       title,
		Description: description,
		Status:      status,
		Priority:    priority,
		Type:        ticketType,
		CreatedBy:   createdBy,
		BudgetUSD:   budgetUSD,
	}, nil
}

func (t TicketTemplate) Raw() map[string]any {
	raw := map[string]any{
		"title":      t.Title,
		"priority":   string(t.Priority),
		"type":       string(t.Type),
		"created_by": t.CreatedBy,
	}
	if t.Description != "" {
		raw["description"] = t.Description
	}
	if t.Status != "" {
		raw["status"] = t.Status
	}
	if t.BudgetUSD > 0 {
		raw["budget_usd"] = t.BudgetUSD
	}

	return raw
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]ScheduledJob, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}

	items, err := s.client.ScheduledJob.Query().
		Where(entscheduledjob.ProjectIDEQ(projectID)).
		Order(ent.Asc(entscheduledjob.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scheduled jobs: %w", err)
	}

	jobs := make([]ScheduledJob, 0, len(items))
	for _, item := range items {
		job, err := mapScheduledJob(item)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (ScheduledJob, error) {
	if s == nil || s.client == nil {
		return ScheduledJob{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return ScheduledJob{}, err
	}
	if strings.TrimSpace(input.TicketTemplate.Status) == "" {
		return ScheduledJob{}, fmt.Errorf("%w: ticket_template.status must not be empty", ErrInvalidTicketTemplate)
	}

	schedule, err := s.parseCron(input.CronExpression)
	if err != nil {
		return ScheduledJob{}, err
	}

	builder := s.client.ScheduledJob.Create().
		SetProjectID(input.ProjectID).
		SetName(strings.TrimSpace(input.Name)).
		SetCronExpression(strings.TrimSpace(input.CronExpression)).
		SetTicketTemplate(input.TicketTemplate.Raw()).
		SetIsEnabled(input.IsEnabled)

	if input.IsEnabled {
		nextRunAt := schedule.Next(s.now().UTC())
		builder.SetNextRunAt(nextRunAt)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return ScheduledJob{}, s.mapWriteError("create scheduled job", err)
	}

	return mapScheduledJob(item)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (ScheduledJob, error) {
	if s == nil || s.client == nil {
		return ScheduledJob{}, ErrUnavailable
	}

	current, err := s.client.ScheduledJob.Get(ctx, input.JobID)
	if err != nil {
		return ScheduledJob{}, s.mapReadError("get scheduled job", err)
	}

	effectiveCron := current.CronExpression
	cronChanged := false
	if input.CronExpression.Set {
		effectiveCron = strings.TrimSpace(input.CronExpression.Value)
		cronChanged = effectiveCron != current.CronExpression
	}
	schedule, err := s.parseCron(effectiveCron)
	if err != nil {
		return ScheduledJob{}, err
	}

	effectiveEnabled := current.IsEnabled
	if input.IsEnabled.Set {
		effectiveEnabled = input.IsEnabled.Value
	}

	builder := s.client.ScheduledJob.UpdateOneID(input.JobID)
	if input.Name.Set {
		builder.SetName(strings.TrimSpace(input.Name.Value))
	}
	if input.CronExpression.Set {
		builder.SetCronExpression(effectiveCron)
	}
	if input.TicketTemplate.Set {
		if strings.TrimSpace(input.TicketTemplate.Value.Status) == "" {
			return ScheduledJob{}, fmt.Errorf("%w: ticket_template.status must not be empty", ErrInvalidTicketTemplate)
		}
		builder.SetTicketTemplate(input.TicketTemplate.Value.Raw())
	}
	if input.IsEnabled.Set {
		builder.SetIsEnabled(input.IsEnabled.Value)
	}

	nextRunAt := s.nextRunAfterUpdate(current, schedule, effectiveEnabled, cronChanged)
	if nextRunAt == nil {
		builder.ClearNextRunAt()
	} else {
		builder.SetNextRunAt(*nextRunAt)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return ScheduledJob{}, s.mapWriteError("update scheduled job", err)
	}

	return mapScheduledJob(item)
}

func (s *Service) Delete(ctx context.Context, jobID uuid.UUID) (DeleteResult, error) {
	if s == nil || s.client == nil {
		return DeleteResult{}, ErrUnavailable
	}

	if err := s.client.ScheduledJob.DeleteOneID(jobID).Exec(ctx); err != nil {
		return DeleteResult{}, s.mapWriteError("delete scheduled job", err)
	}

	return DeleteResult{DeletedJobID: jobID}, nil
}

func (s *Service) Trigger(ctx context.Context, jobID uuid.UUID) (TriggerResult, error) {
	if s == nil || s.client == nil || s.tickets == nil {
		return TriggerResult{}, ErrUnavailable
	}

	jobItem, err := s.client.ScheduledJob.Query().
		Where(entscheduledjob.IDEQ(jobID)).
		WithWorkflow().
		Only(ctx)
	if err != nil {
		return TriggerResult{}, s.mapReadError("get scheduled job", err)
	}

	now := s.now().UTC()
	ticketItem, err := s.createTicketForJob(ctx, jobItem, now)
	if err != nil {
		return TriggerResult{}, err
	}

	updated, err := s.client.ScheduledJob.UpdateOneID(jobID).
		SetLastRunAt(now).
		Save(ctx)
	if err != nil {
		return TriggerResult{}, s.mapWriteError("update scheduled job trigger timestamps", err)
	}

	if updated.IsEnabled {
		schedule, parseErr := s.parseCron(updated.CronExpression)
		if parseErr != nil {
			return TriggerResult{}, parseErr
		}
		if updated.NextRunAt == nil || !updated.NextRunAt.After(now) {
			updated, err = s.client.ScheduledJob.UpdateOneID(jobID).
				SetNextRunAt(schedule.Next(now)).
				Save(ctx)
			if err != nil {
				return TriggerResult{}, s.mapWriteError("refresh scheduled job next run", err)
			}
		}
	}

	job, err := mapScheduledJob(updated)
	if err != nil {
		return TriggerResult{}, err
	}

	return TriggerResult{
		Job:    job,
		Ticket: ticketItem,
	}, nil
}

func (s *Service) RunDue(ctx context.Context) (RunDueReport, error) {
	report := RunDueReport{}
	if s == nil || s.client == nil || s.tickets == nil {
		return report, ErrUnavailable
	}

	now := s.now().UTC()
	items, err := s.client.ScheduledJob.Query().
		Where(
			entscheduledjob.IsEnabled(true),
			entscheduledjob.NextRunAtNotNil(),
			entscheduledjob.NextRunAtLTE(now),
			entscheduledjob.HasProjectWith(entproject.StatusEQ("In Progress")),
		).
		WithWorkflow().
		Order(ent.Asc(entscheduledjob.FieldNextRunAt), ent.Asc(entscheduledjob.FieldName)).
		All(ctx)
	if err != nil {
		return report, fmt.Errorf("list due scheduled jobs: %w", err)
	}

	report.JobsScanned = len(items)
	for _, item := range items {
		jobLogger := s.logger.With("job_id", item.ID, "job_name", item.Name, "workflow_id", item.WorkflowID)

		if _, err := s.createTicketForJob(ctx, item, now); err != nil {
			jobLogger.Error("create scheduled job ticket", "error", err)
			continue
		}

		schedule, err := s.parseCron(item.CronExpression)
		if err != nil {
			jobLogger.Error("parse scheduled job cron", "error", err)
			continue
		}
		nextRunAt := schedule.Next(now)
		if _, err := s.client.ScheduledJob.UpdateOneID(item.ID).
			SetLastRunAt(now).
			SetNextRunAt(nextRunAt).
			Save(ctx); err != nil {
			jobLogger.Error("advance scheduled job", "error", s.mapWriteError("advance scheduled job", err))
			continue
		}

		report.TicketsCreated++
		jobLogger.Info(
			"scheduled job triggered",
			"next_run_at", nextRunAt.Format(time.RFC3339),
		)
	}

	return report, nil
}

func (s *Service) createTicketForJob(ctx context.Context, item *ent.ScheduledJob, now time.Time) (ticketservice.Ticket, error) {
	template, err := ParseRawTicketTemplate(item.TicketTemplate)
	if err != nil {
		return ticketservice.Ticket{}, err
	}

	workflowItem := item.Edges.Workflow
	if workflowItem == nil && item.WorkflowID != nil {
		workflowItem, err = s.loadWorkflow(ctx, item.ProjectID, *item.WorkflowID)
		if err != nil {
			return ticketservice.Ticket{}, err
		}
	}
	if workflowItem != nil && strings.TrimSpace(template.Status) == "" && len(workflowItem.Edges.PickupStatuses) == 0 {
		workflowItem, err = s.loadWorkflow(ctx, item.ProjectID, *item.WorkflowID)
		if err != nil {
			return ticketservice.Ticket{}, err
		}
	}

	title, err := renderScheduledJobTemplateField(template.Title, scheduledJobTemplateContext(item, workflowItem, now))
	if err != nil {
		return ticketservice.Ticket{}, err
	}
	if title == "" {
		return ticketservice.Ticket{}, fmt.Errorf("%w: rendered ticket title must not be empty", ErrInvalidTicketTemplate)
	}
	description, err := renderScheduledJobTemplateField(template.Description, scheduledJobTemplateContext(item, workflowItem, now))
	if err != nil {
		return ticketservice.Ticket{}, err
	}

	if template.Status != "" {
		statusID, err := s.resolveStatusIDByName(ctx, item.ProjectID, template.Status)
		if err != nil {
			return ticketservice.Ticket{}, err
		}

		return s.tickets.Create(ctx, ticketservice.CreateInput{
			ProjectID:   item.ProjectID,
			Title:       title,
			Description: description,
			StatusID:    &statusID,
			Priority:    template.Priority,
			Type:        template.Type,
			CreatedBy:   template.CreatedBy,
			BudgetUSD:   template.BudgetUSD,
		})
	}

	if workflowItem == nil {
		return ticketservice.Ticket{}, fmt.Errorf("%w: ticket_template.status must not be empty", ErrInvalidTicketTemplate)
	}

	statusID, err := resolveScheduledJobPickupStatus(template.Status, workflowItem)
	if err != nil {
		return ticketservice.Ticket{}, err
	}

	return s.tickets.Create(ctx, ticketservice.CreateInput{
		ProjectID:   item.ProjectID,
		Title:       title,
		Description: description,
		StatusID:    &statusID,
		Priority:    template.Priority,
		Type:        template.Type,
		CreatedBy:   template.CreatedBy,
		BudgetUSD:   template.BudgetUSD,
	})
}

func (s *Service) ensureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := s.client.Project.Query().
		Where(entproject.IDEQ(projectID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func (s *Service) loadWorkflow(ctx context.Context, projectID uuid.UUID, workflowID uuid.UUID) (*ent.Workflow, error) {
	item, err := s.client.Workflow.Query().
		Where(
			entworkflow.IDEQ(workflowID),
			entworkflow.ProjectIDEQ(projectID),
		).
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("get workflow: %w", err)
	}

	return item, nil
}

func (s *Service) resolveStatusIDByName(ctx context.Context, projectID uuid.UUID, statusName string) (uuid.UUID, error) {
	item, err := s.client.TicketStatus.Query().
		Where(
			entticketstatus.ProjectIDEQ(projectID),
			entticketstatus.NameEQ(strings.TrimSpace(statusName)),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.UUID{}, ErrStatusNotFound
		}
		return uuid.UUID{}, fmt.Errorf("get ticket status: %w", err)
	}

	return item.ID, nil
}

func resolveScheduledJobPickupStatus(templateStatus string, workflowItem *ent.Workflow) (uuid.UUID, error) {
	if strings.TrimSpace(templateStatus) != "" {
		return uuid.UUID{}, nil
	}

	pickupStatuses := workflowItem.Edges.PickupStatuses
	switch len(pickupStatuses) {
	case 0:
		return uuid.UUID{}, fmt.Errorf("workflow %s has no pickup statuses configured", workflowItem.ID)
	case 1:
		return pickupStatuses[0].ID, nil
	default:
		return uuid.UUID{}, fmt.Errorf(
			"%w: scheduled job ticket template must specify status when workflow %s has multiple pickup statuses",
			ErrInvalidTicketTemplate,
			workflowItem.ID,
		)
	}
}

func (s *Service) parseCron(raw string) (cron.Schedule, error) {
	schedule, err := s.cronParser.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidCronExpression, err)
	}

	return schedule, nil
}

func (s *Service) nextRunAfterUpdate(current *ent.ScheduledJob, schedule cron.Schedule, enabled bool, cronChanged bool) *time.Time {
	if !enabled {
		return nil
	}

	now := s.now().UTC()
	if cronChanged || !current.IsEnabled || current.NextRunAt == nil || !current.NextRunAt.After(now) {
		next := schedule.Next(now)
		return &next
	}

	next := current.NextRunAt.UTC()
	return &next
}

func (s *Service) mapReadError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrScheduledJobNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func (s *Service) mapWriteError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrScheduledJobNotFound
	case ent.IsConstraintError(err):
		return ErrScheduledJobConflict
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func mapScheduledJob(item *ent.ScheduledJob) (ScheduledJob, error) {
	template, err := ParseRawTicketTemplate(item.TicketTemplate)
	if err != nil {
		return ScheduledJob{}, err
	}

	return ScheduledJob{
		ID:             item.ID,
		ProjectID:      item.ProjectID,
		Name:           item.Name,
		CronExpression: item.CronExpression,
		TicketTemplate: template,
		IsEnabled:      item.IsEnabled,
		LastRunAt:      cloneTime(item.LastRunAt),
		NextRunAt:      cloneTime(item.NextRunAt),
	}, nil
}

func parseRequiredString(raw map[string]any, fieldName string) (string, error) {
	value, ok := raw[fieldName]
	if !ok {
		return "", fmt.Errorf("%w: ticket_template.%s must not be empty", ErrInvalidTicketTemplate, fieldName)
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%w: ticket_template.%s must be a string", ErrInvalidTicketTemplate, fieldName)
	}
	trimmed := strings.TrimSpace(stringValue)
	if trimmed == "" {
		return "", fmt.Errorf("%w: ticket_template.%s must not be empty", ErrInvalidTicketTemplate, fieldName)
	}

	return trimmed, nil
}

func parseOptionalString(raw map[string]any, fieldName string) (string, error) {
	value, ok := raw[fieldName]
	if !ok {
		return "", nil
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%w: ticket_template.%s must be a string", ErrInvalidTicketTemplate, fieldName)
	}

	return strings.TrimSpace(stringValue), nil
}

func parseBudgetUSD(value any) (float64, error) {
	floatValue, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%w: ticket_template.budget_usd must be a number", ErrInvalidTicketTemplate)
	}
	if floatValue < 0 {
		return 0, fmt.Errorf("%w: ticket_template.budget_usd must be greater than or equal to zero", ErrInvalidTicketTemplate)
	}

	return floatValue, nil
}

func renderScheduledJobTemplateField(content string, data map[string]any) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}

	template, err := gonja.FromString(content)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidTicketTemplate, err)
	}

	rendered, err := template.ExecuteToString(exec.NewContext(data))
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidTicketTemplate, err)
	}

	return strings.TrimSpace(rendered), nil
}

func scheduledJobTemplateContext(item *ent.ScheduledJob, workflowItem *ent.Workflow, now time.Time) map[string]any {
	workflowID := ""
	workflowName := ""
	workflowType := ""
	if workflowItem != nil {
		workflowID = workflowItem.ID.String()
		workflowName = workflowItem.Name
		workflowType = workflowItem.Type.String()
	}

	return map[string]any{
		"date":      now.Format("2006-01-02"),
		"time":      now.Format("15:04:05"),
		"timestamp": now.Format(time.RFC3339),
		"job": map[string]any{
			"id":              item.ID.String(),
			"name":            item.Name,
			"cron_expression": item.CronExpression,
		},
		"workflow": map[string]any{
			"id":   workflowID,
			"name": workflowName,
			"type": workflowType,
		},
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}
