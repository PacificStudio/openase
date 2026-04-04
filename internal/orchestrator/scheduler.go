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
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	domaincatalog "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/provider"
	scheduledjobrepo "github.com/BetterAndBetterII/openase/internal/repo/scheduledjob"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

const (
	skipReasonBlocked             = "blocked"
	skipReasonNoAgent             = "no_agent"
	skipReasonNoMachine           = "no_machine"
	skipReasonProviderUnknown     = "provider_unknown"
	skipReasonProviderUnavailable = "provider_unavailable"
	skipReasonProviderStale       = "provider_stale"
	skipReasonProviderBusy        = "provider_busy"
	skipReasonMaxConcurrency      = "max_concurrency"
	skipReasonStatusCapacity      = "status_capacity"
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
		scheduledJobs: scheduledjobservice.NewService(scheduledjobrepo.NewEntRepository(client), ticketservice.NewService(ticketrepo.NewEntRepository(client)), logger),
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
		).
		WithProject().
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithFinishStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
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
			entticket.Archived(false),
			entticket.StatusIDIn(ticketStatusIDs(workflow.Edges.PickupStatuses)...),
			entticket.CurrentRunIDIsNil(),
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
	agent, err := s.resolveWorkflowAgent(ctx, workflow)
	if err != nil {
		return false, "", fmt.Errorf("resolve workflow agent: %w", err)
	}
	if agent == nil {
		s.logDispatchSkip(workflow, ticket, skipReasonNoAgent)
		return false, skipReasonNoAgent, nil
	}
	machine, providerItem, reason, err := s.resolveExecutionMachine(ctx, project.OrganizationID, agent, now)
	if err != nil {
		return false, "", fmt.Errorf("resolve execution machine: %w", err)
	}
	if machine == nil {
		s.logDispatchSkip(workflow, ticket, reason, "agent_id", agent.ID, "provider_id", agent.ProviderID)
		return false, reason, nil
	}
	outcome, err := s.claimTicketWithAgent(ctx, workflow, ticket, machine, agent, providerItem, project.MaxConcurrentAgents, now)
	if err != nil {
		return false, "", err
	}
	if outcome == "" {
		s.logger.Info(
			"ticket dispatched to workflow-bound agent",
			"ticket_id", ticket.ID,
			"workflow_id", workflow.ID,
			"agent_id", agent.ID,
		)
		return true, "", nil
	}

	return false, outcome, nil
}

func (s *Scheduler) resolveWorkflowAgent(ctx context.Context, workflow *ent.Workflow) (*ent.Agent, error) {
	if workflow.AgentID == nil {
		return nil, nil
	}

	agentItem, err := s.client.Agent.Query().
		Where(
			entagent.IDEQ(*workflow.AgentID),
			entagent.ProjectIDEQ(workflow.ProjectID),
			entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateActive),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return agentItem, nil
}

func (s *Scheduler) resolveExecutionMachine(
	ctx context.Context,
	organizationID uuid.UUID,
	agent *ent.Agent,
	now time.Time,
) (*ent.Machine, *ent.AgentProvider, string, error) {
	providerItem, err := s.client.AgentProvider.Get(ctx, agent.ProviderID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, skipReasonNoMachine, nil
		}
		return nil, nil, "", err
	}
	if providerItem.OrganizationID != organizationID {
		return nil, nil, skipReasonNoMachine, nil
	}

	machine, err := s.client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(organizationID),
			entmachine.IDEQ(providerItem.MachineID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, skipReasonNoMachine, nil
		}
		return nil, nil, "", err
	}
	if machine.Status != entmachine.StatusOnline {
		return nil, nil, skipReasonNoMachine, nil
	}

	providerState := domaincatalog.DeriveAgentProviderAvailability(domaincatalog.AgentProvider{
		ID:                   providerItem.ID,
		OrganizationID:       providerItem.OrganizationID,
		MachineID:            providerItem.MachineID,
		MachineName:          machine.Name,
		MachineHost:          machine.Host,
		MachineStatus:        domaincatalog.MachineStatus(machine.Status),
		MachineSSHUser:       schedulerOptionalString(machine.SSHUser),
		MachineWorkspaceRoot: schedulerOptionalString(machine.WorkspaceRoot),
		MachineAgentCLIPath:  schedulerOptionalString(machine.AgentCliPath),
		MachineResources:     cloneResourceMap(machine.Resources),
		Name:                 providerItem.Name,
		AdapterType:          domaincatalog.AgentProviderAdapterType(providerItem.AdapterType),
		CliCommand:           providerItem.CliCommand,
		CliArgs:              append([]string(nil), providerItem.CliArgs...),
		AuthConfig:           cloneResourceMap(providerItem.AuthConfig),
		ModelName:            providerItem.ModelName,
		ModelTemperature:     providerItem.ModelTemperature,
		ModelMaxTokens:       providerItem.ModelMaxTokens,
		MaxParallelRuns:      providerItem.MaxParallelRuns,
		CostPerInputToken:    providerItem.CostPerInputToken,
		CostPerOutputToken:   providerItem.CostPerOutputToken,
	}, now.UTC())
	if providerState.AvailabilityState != domaincatalog.AgentProviderAvailabilityStateAvailable {
		return nil, nil, skipReasonForProviderAvailability(providerState.AvailabilityState), nil
	}

	return machine, providerItem, "", nil
}

func schedulerOptionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copied := value
	return &copied
}

func skipReasonForProviderAvailability(state domaincatalog.AgentProviderAvailabilityState) string {
	switch state {
	case domaincatalog.AgentProviderAvailabilityStateStale:
		return skipReasonProviderStale
	case domaincatalog.AgentProviderAvailabilityStateAvailable:
		return ""
	case domaincatalog.AgentProviderAvailabilityStateUnknown:
		return skipReasonProviderUnknown
	default:
		return skipReasonProviderUnavailable
	}
}

func (s *Scheduler) claimTicketWithAgent(ctx context.Context, workflow *ent.Workflow, ticket *ent.Ticket, machine *ent.Machine, agent *ent.Agent, providerItem *ent.AgentProvider, projectMaxConcurrent int, now time.Time) (string, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return "", fmt.Errorf("start dispatch tx: %w", err)
	}
	defer rollback(tx)

	workflowActive, err := tx.Ticket.Query().
		Where(
			entticket.WorkflowIDEQ(workflow.ID),
			entticket.CurrentRunIDNotNil(),
		).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("count workflow concurrency: %w", err)
	}
	if workflow.MaxConcurrent > 0 && workflowActive >= workflow.MaxConcurrent {
		s.logDispatchSkip(workflow, ticket, skipReasonMaxConcurrency,
			"scope", "workflow",
			"active_runs", workflowActive,
			"capacity", workflow.MaxConcurrent,
			"agent_id", agent.ID,
			"provider_id", providerItem.ID,
		)
		return skipReasonMaxConcurrency, nil
	}

	projectActive, err := tx.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(workflow.ProjectID),
			entticket.CurrentRunIDNotNil(),
		).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("count project concurrency: %w", err)
	}
	if projectMaxConcurrent > 0 && projectActive >= projectMaxConcurrent {
		s.logDispatchSkip(workflow, ticket, skipReasonMaxConcurrency,
			"scope", "project",
			"active_runs", projectActive,
			"capacity", projectMaxConcurrent,
			"agent_id", agent.ID,
			"provider_id", providerItem.ID,
		)
		return skipReasonMaxConcurrency, nil
	}

	providerActive, err := tx.AgentRun.Query().
		Where(
			entagentrun.ProviderIDEQ(providerItem.ID),
			entagentrun.HasCurrentForTicket(),
		).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("count provider concurrency: %w", err)
	}
	if providerItem.MaxParallelRuns > 0 && providerActive >= providerItem.MaxParallelRuns {
		s.logDispatchSkip(workflow, ticket, skipReasonProviderBusy,
			"scope", "provider",
			"active_runs", providerActive,
			"capacity", providerItem.MaxParallelRuns,
			"agent_id", agent.ID,
			"provider_id", providerItem.ID,
			"machine_id", machine.ID,
		)
		return skipReasonProviderBusy, nil
	}

	pickupStatus, err := tx.TicketStatus.Query().
		Where(entticketstatus.IDEQ(ticket.StatusID)).
		Only(ctx)
	if err != nil {
		return "", fmt.Errorf("load workflow pickup status: %w", err)
	}
	if pickupStatus.MaxActiveRuns != nil {
		statusActive, err := tx.Ticket.Query().
			Where(
				entticket.ProjectIDEQ(workflow.ProjectID),
				entticket.CurrentRunIDNotNil(),
				entticket.StatusIDEQ(pickupStatus.ID),
			).
			Count(ctx)
		if err != nil {
			return "", fmt.Errorf("count status concurrency: %w", err)
		}
		if statusActive >= *pickupStatus.MaxActiveRuns {
			s.logDispatchSkip(workflow, ticket, skipReasonStatusCapacity,
				"scope", "status",
				"status_id", pickupStatus.ID,
				"status_name", pickupStatus.Name,
				"active_runs", statusActive,
				"capacity", *pickupStatus.MaxActiveRuns,
				"agent_id", agent.ID,
				"provider_id", providerItem.ID,
			)
			return skipReasonStatusCapacity, nil
		}
	}

	claimedAgents, err := tx.Agent.Update().
		Where(
			entagent.IDEQ(agent.ID),
			entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateActive),
		).
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("claim agent %s: %w", agent.ID, err)
	}
	if claimedAgents == 0 {
		s.logDispatchSkip(workflow, ticket, skipReasonNoAgent,
			"scope", "agent_claim",
			"agent_id", agent.ID,
			"provider_id", providerItem.ID,
		)
		return skipReasonNoAgent, nil
	}

	runItem, err := tx.AgentRun.Create().
		SetAgentID(agent.ID).
		SetWorkflowID(workflow.ID).
		SetTicketID(ticket.ID).
		SetProviderID(agent.ProviderID).
		SetStatus(entagentrun.StatusLaunching).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("create agent run for ticket %s: %w", ticket.ID, err)
	}

	claimPredicates := []predicate.Ticket{
		entticket.IDEQ(ticket.ID),
		entticket.Archived(false),
		entticket.StatusIDIn(ticketStatusIDs(workflow.Edges.PickupStatuses)...),
		entticket.CurrentRunIDIsNil(),
		entticket.RetryPaused(false),
		entticket.Or(
			entticket.NextRetryAtIsNil(),
			entticket.NextRetryAtLTE(now),
		),
	}
	if ticket.NextRetryAt != nil && strings.TrimSpace(ticket.RetryToken) != "" {
		claimPredicates = append(claimPredicates, entticket.RetryTokenEQ(ticket.RetryToken))
	}

	claimedTickets, err := tx.Ticket.Update().
		Where(claimPredicates...).
		SetCurrentRunID(runItem.ID).
		SetWorkflowID(workflow.ID).
		SetTargetMachineID(machine.ID).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("claim ticket %s: %w", ticket.ID, err)
	}
	if claimedTickets == 0 {
		s.logDispatchSkip(workflow, ticket, skipReasonNoAgent,
			"scope", "ticket_claim",
			"agent_id", agent.ID,
			"provider_id", providerItem.ID,
			"machine_id", machine.ID,
			"ticket_retry_token", ticket.RetryToken,
		)
		return skipReasonNoAgent, nil
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit dispatch tx: %w", err)
	}

	claimedAgent, err := loadAgentLifecycleState(ctx, s.client, agent.ID, &runItem.ID)
	if err != nil {
		return "", err
	}
	if err := publishAgentLifecycleEvent(
		ctx,
		s.client,
		s.events,
		agentClaimedType,
		claimedAgent,
		lifecycleMessage(agentClaimedType, claimedAgent.agent.Name),
		schedulerRuntimeEventMetadata(claimedAgent, machine),
		now,
	); err != nil {
		return "", err
	}

	return "", nil
}

func (s *Scheduler) logDispatchSkip(workflow *ent.Workflow, ticket *ent.Ticket, reason string, attrs ...any) {
	if workflow == nil || ticket == nil || reason == "" {
		return
	}

	baseAttrs := make([]any, 0, 12+len(attrs))
	baseAttrs = append(baseAttrs,
		"operation", "schedule_ticket",
		"project_id", workflow.ProjectID,
		"workflow_id", workflow.ID,
		"ticket_id", ticket.ID,
		"ticket_status_id", ticket.StatusID,
		"skip_reason", reason,
	)
	s.logger.Info("scheduler skipped ticket", append(baseAttrs, attrs...)...)
}

func (s *Scheduler) isTicketBlocked(ctx context.Context, ticketID uuid.UUID) (bool, error) {
	dependencies, err := s.client.TicketDependency.Query().
		Where(
			entticketdependency.TargetTicketIDEQ(ticketID),
			entticketdependency.TypeEQ(entticketdependency.TypeBlocks),
		).
		WithSourceTicket(func(query *ent.TicketQuery) {
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

	if status := ticket.Edges.Status; status != nil {
		stage := ticketing.StatusStage(status.Stage)
		if stage.IsValid() && stage.IsTerminal() {
			return true
		}
	}

	return false
}

func machineHasAllLabels(machineLabels []string, requiredLabels []string) bool {
	if len(requiredLabels) == 0 {
		return true
	}

	available := make(map[string]struct{}, len(machineLabels))
	for _, label := range machineLabels {
		available[label] = struct{}{}
	}
	for _, label := range requiredLabels {
		if _, ok := available[label]; !ok {
			return false
		}
	}

	return true
}

func schedulerRuntimeEventMetadata(agentState agentLifecycleState, machine *ent.Machine) map[string]any {
	metadata := runtimeEventMetadataForState(agentState)
	if machine == nil {
		return metadata
	}
	metadata["target_machine_id"] = machine.ID.String()
	metadata["target_machine_name"] = machine.Name
	return metadata
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

func ticketStatusIDs(statuses []*ent.TicketStatus) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(statuses))
	for _, status := range statuses {
		ids = append(ids, status.ID)
	}
	return ids
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
