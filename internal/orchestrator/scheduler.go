package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/provider"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

const (
	skipReasonBlocked        = "blocked"
	skipReasonNoAgent        = "no_agent"
	skipReasonMaxConcurrency = "max_concurrency"
)

// TickReport summarizes the work done during one scheduler tick.
type TickReport struct {
	ScheduledJobsScanned    int            `json:"scheduled_jobs_scanned"`
	ScheduledTicketsCreated int            `json:"scheduled_tickets_created"`
	WorkflowsScanned        int            `json:"workflows_scanned"`
	CandidatesScanned       int            `json:"candidates_scanned"`
	TicketsDispatched       int            `json:"tickets_dispatched"`
	TicketsSkipped          map[string]int `json:"tickets_skipped"`
}

// Scheduler claims runnable tickets and advances orchestrator work.
type Scheduler struct {
	client        *ent.Client
	logger        *slog.Logger
	events        provider.EventProvider
	scheduledJobs *scheduledjobservice.Service
	now           func() time.Time
}

// NewScheduler constructs the orchestrator scheduler.
func NewScheduler(client *ent.Client, logger *slog.Logger, events provider.EventProvider) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		client:        client,
		logger:        logger.With("component", "scheduler"),
		events:        events,
		scheduledJobs: scheduledjobservice.NewService(client, ticketservice.NewService(client), logger),
		now:           time.Now,
	}
}

// RunTick executes one scheduling pass.
func (s *Scheduler) RunTick(ctx context.Context) (TickReport, error) {
	report := TickReport{
		TicketsSkipped: map[string]int{},
	}
	if s == nil || s.client == nil {
		return report, fmt.Errorf("scheduler unavailable")
	}

	if s.scheduledJobs != nil {
		s.scheduledJobs.SetNowFunc(s.now)
		dueReport, err := s.scheduledJobs.RunDue(ctx)
		if err != nil {
			return report, fmt.Errorf("run scheduled jobs: %w", err)
		}
		report.ScheduledJobsScanned = dueReport.JobsScanned
		report.ScheduledTicketsCreated = dueReport.TicketsCreated
	}

	now := s.now().UTC()
	workflows, err := s.client.Workflow.Query().
		Where(
			entworkflow.IsActive(true),
			entworkflow.HasProjectWith(entproject.StatusEQ(entproject.StatusActive)),
		).
		WithProject().
		Order(ent.Asc(entworkflow.FieldName)).
		All(ctx)
	if err != nil {
		return report, fmt.Errorf("list active workflows: %w", err)
	}

	for _, workflow := range workflows {
		workflowReport, err := s.runWorkflowTick(ctx, workflow, now)
		if err != nil {
			return report, err
		}
		report.WorkflowsScanned++
		report.CandidatesScanned += workflowReport.CandidatesScanned
		report.TicketsDispatched += workflowReport.TicketsDispatched
		mergeSkipCounts(report.TicketsSkipped, workflowReport.TicketsSkipped)
	}

	return report, nil
}

func (s *Scheduler) runWorkflowTick(ctx context.Context, workflow *ent.Workflow, now time.Time) (TickReport, error) {
	report := TickReport{
		TicketsSkipped: map[string]int{},
	}

	candidates, err := s.client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(workflow.ProjectID),
			entticket.StatusIDEQ(workflow.PickupStatusID),
			entticket.AssignedAgentIDIsNil(),
			entticket.RetryPaused(false),
			entticket.Or(
				entticket.NextRetryAtIsNil(),
				entticket.NextRetryAtLTE(now),
			),
		).
		Order(ent.Asc(entticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return report, fmt.Errorf("list pickup candidates for workflow %s: %w", workflow.ID, err)
	}

	sortTicketsByPriorityAndAge(candidates)
	report.CandidatesScanned = len(candidates)

	for _, candidate := range candidates {
		blocked, err := s.isTicketBlocked(ctx, candidate.ID)
		if err != nil {
			return report, fmt.Errorf("check ticket %s dependencies: %w", candidate.ID, err)
		}
		if blocked {
			report.TicketsSkipped[skipReasonBlocked]++
			continue
		}

		dispatched, reason, err := s.tryDispatch(ctx, workflow, candidate, now)
		if err != nil {
			return report, fmt.Errorf("dispatch ticket %s: %w", candidate.ID, err)
		}
		if !dispatched {
			report.TicketsSkipped[reason]++
			continue
		}
		report.TicketsDispatched++
	}

	return report, nil
}

func (s *Scheduler) tryDispatch(ctx context.Context, workflow *ent.Workflow, ticket *ent.Ticket, now time.Time) (bool, string, error) {
	project := workflow.Edges.Project
	if project == nil {
		return false, "", fmt.Errorf("workflow %s missing project edge", workflow.ID)
	}
	if workflow.MaxConcurrent <= 0 || project.MaxConcurrentAgents <= 0 {
		return false, skipReasonMaxConcurrency, nil
	}

	agents, err := s.listIdleAgents(ctx, workflow.ProjectID)
	if err != nil {
		return false, "", fmt.Errorf("list idle agents: %w", err)
	}
	if len(agents) == 0 {
		return false, skipReasonNoAgent, nil
	}

	sortAgentsForDispatch(agents)
	for _, agent := range agents {
		outcome, err := s.claimTicketWithAgent(ctx, workflow, ticket, agent, project.MaxConcurrentAgents, now)
		if err != nil {
			return false, "", err
		}
		if outcome == "" {
			s.logger.Info(
				"ticket dispatched",
				"ticket_id", ticket.ID,
				"workflow_id", workflow.ID,
				"agent_id", agent.ID,
			)
			return true, "", nil
		}
		if outcome == skipReasonMaxConcurrency {
			return false, skipReasonMaxConcurrency, nil
		}
	}

	return false, skipReasonNoAgent, nil
}

func (s *Scheduler) listIdleAgents(ctx context.Context, projectID uuid.UUID) ([]*ent.Agent, error) {
	agents, err := s.client.Agent.Query().
		Where(
			entagent.ProjectIDEQ(projectID),
			entagent.StatusEQ(entagent.StatusIdle),
			entagent.CurrentTicketIDIsNil(),
		).
		Order(ent.Asc(entagent.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return agents, nil
}

func (s *Scheduler) claimTicketWithAgent(
	ctx context.Context,
	workflow *ent.Workflow,
	ticket *ent.Ticket,
	agent *ent.Agent,
	projectMaxConcurrent int,
	now time.Time,
) (string, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return "", fmt.Errorf("start dispatch tx: %w", err)
	}
	defer rollback(tx)

	workflowActive, err := tx.Ticket.Query().
		Where(
			entticket.WorkflowIDEQ(workflow.ID),
			entticket.AssignedAgentIDNotNil(),
		).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("count workflow concurrency: %w", err)
	}
	if workflowActive >= workflow.MaxConcurrent {
		return skipReasonMaxConcurrency, nil
	}

	projectActive, err := tx.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(workflow.ProjectID),
			entticket.AssignedAgentIDNotNil(),
		).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("count project concurrency: %w", err)
	}
	if projectActive >= projectMaxConcurrent {
		return skipReasonMaxConcurrency, nil
	}

	claimedAgents, err := tx.Agent.Update().
		Where(
			entagent.IDEQ(agent.ID),
			entagent.StatusEQ(entagent.StatusIdle),
			entagent.CurrentTicketIDIsNil(),
		).
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticket.ID).
		ClearSessionID().
		SetRuntimePhase(entagent.RuntimePhaseNone).
		ClearRuntimeStartedAt().
		SetLastError("").
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("claim agent %s: %w", agent.ID, err)
	}
	if claimedAgents == 0 {
		return skipReasonNoAgent, nil
	}

	claimedTickets, err := tx.Ticket.Update().
		Where(
			entticket.IDEQ(ticket.ID),
			entticket.StatusIDEQ(workflow.PickupStatusID),
			entticket.AssignedAgentIDIsNil(),
			entticket.RetryPaused(false),
			entticket.Or(
				entticket.NextRetryAtIsNil(),
				entticket.NextRetryAtLTE(now),
			),
		).
		SetAssignedAgentID(agent.ID).
		SetWorkflowID(workflow.ID).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("claim ticket %s: %w", ticket.ID, err)
	}
	if claimedTickets == 0 {
		return skipReasonNoAgent, nil
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit dispatch tx: %w", err)
	}

	claimedAgent, err := loadAgentLifecycleState(ctx, s.client, agent.ID)
	if err != nil {
		return "", err
	}
	if err := publishAgentLifecycleEvent(
		ctx,
		s.client,
		s.events,
		agentClaimedType,
		claimedAgent,
		lifecycleMessage(agentClaimedType, claimedAgent.Name),
		runtimeEventMetadata(claimedAgent),
		now,
	); err != nil {
		return "", err
	}

	return "", nil
}

func (s *Scheduler) isTicketBlocked(ctx context.Context, ticketID uuid.UUID) (bool, error) {
	dependencies, err := s.client.TicketDependency.Query().
		Where(
			entticketdependency.TargetTicketIDEQ(ticketID),
			entticketdependency.TypeEQ(entticketdependency.TypeBlocks),
		).
		WithSourceTicket(func(query *ent.TicketQuery) {
			query.WithWorkflow()
			query.WithStatus()
		}).
		All(ctx)
	if err != nil {
		return false, err
	}

	for _, dependency := range dependencies {
		sourceTicket := dependency.Edges.SourceTicket
		if sourceTicket == nil {
			return true, fmt.Errorf("dependency %s missing source ticket", dependency.ID)
		}
		if !isDependencyResolved(sourceTicket) {
			return true, nil
		}
	}

	return false, nil
}

func isDependencyResolved(ticket *ent.Ticket) bool {
	if ticket.CompletedAt != nil {
		return true
	}

	if workflow := ticket.Edges.Workflow; workflow != nil && workflow.FinishStatusID != nil && ticket.StatusID == *workflow.FinishStatusID {
		return true
	}

	if status := ticket.Edges.Status; status != nil && strings.EqualFold(status.Name, "Done") {
		return true
	}

	return false
}

func sortTicketsByPriorityAndAge(tickets []*ent.Ticket) {
	slices.SortFunc(tickets, func(left, right *ent.Ticket) int {
		leftRank := priorityRank(left.Priority)
		rightRank := priorityRank(right.Priority)
		if leftRank != rightRank {
			return leftRank - rightRank
		}
		if left.CreatedAt.Before(right.CreatedAt) {
			return -1
		}
		if left.CreatedAt.After(right.CreatedAt) {
			return 1
		}
		return strings.Compare(left.Identifier, right.Identifier)
	})
}

func sortAgentsForDispatch(agents []*ent.Agent) {
	slices.SortFunc(agents, func(left, right *ent.Agent) int {
		if left.TotalTicketsCompleted != right.TotalTicketsCompleted {
			return left.TotalTicketsCompleted - right.TotalTicketsCompleted
		}
		return strings.Compare(left.Name, right.Name)
	})
}

func priorityRank(priority entticket.Priority) int {
	switch priority {
	case entticket.PriorityUrgent:
		return 0
	case entticket.PriorityHigh:
		return 1
	case entticket.PriorityMedium:
		return 2
	case entticket.PriorityLow:
		return 3
	default:
		return 4
	}
}

func mergeSkipCounts(target map[string]int, incoming map[string]int) {
	for reason, count := range incoming {
		target[reason] += count
	}
}

func rollback(tx *ent.Tx) {
	if tx == nil {
		return
	}
	_ = tx.Rollback()
}
