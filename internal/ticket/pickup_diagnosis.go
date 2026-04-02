package ticket

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

type PickupDiagnosisState string

const (
	PickupDiagnosisStateRunnable    PickupDiagnosisState = "runnable"
	PickupDiagnosisStateWaiting     PickupDiagnosisState = "waiting"
	PickupDiagnosisStateBlocked     PickupDiagnosisState = "blocked"
	PickupDiagnosisStateRunning     PickupDiagnosisState = "running"
	PickupDiagnosisStateCompleted   PickupDiagnosisState = "completed"
	PickupDiagnosisStateUnavailable PickupDiagnosisState = "unavailable"
)

type PickupDiagnosisReasonCode string

const (
	PickupDiagnosisReasonReadyForPickup            PickupDiagnosisReasonCode = "ready_for_pickup"
	PickupDiagnosisReasonCompleted                 PickupDiagnosisReasonCode = "completed"
	PickupDiagnosisReasonRunningCurrentRun         PickupDiagnosisReasonCode = "running_current_run"
	PickupDiagnosisReasonRetryBackoff              PickupDiagnosisReasonCode = "retry_backoff"
	PickupDiagnosisReasonRetryPausedRepeatedStalls PickupDiagnosisReasonCode = "retry_paused_repeated_stalls"
	PickupDiagnosisReasonRetryPausedBudget         PickupDiagnosisReasonCode = "retry_paused_budget"
	PickupDiagnosisReasonRetryPausedUser           PickupDiagnosisReasonCode = "retry_paused_user"
	PickupDiagnosisReasonBlockedDependency         PickupDiagnosisReasonCode = "blocked_dependency"
	PickupDiagnosisReasonNoMatchingActiveWorkflow  PickupDiagnosisReasonCode = "no_matching_active_workflow"
	PickupDiagnosisReasonWorkflowInactive          PickupDiagnosisReasonCode = "workflow_inactive"
	PickupDiagnosisReasonWorkflowMissingAgent      PickupDiagnosisReasonCode = "workflow_missing_agent"
	PickupDiagnosisReasonAgentMissing              PickupDiagnosisReasonCode = "agent_missing"
	PickupDiagnosisReasonAgentPaused               PickupDiagnosisReasonCode = "agent_paused"
	PickupDiagnosisReasonAgentPauseRequested       PickupDiagnosisReasonCode = "agent_pause_requested"
	PickupDiagnosisReasonProviderMissing           PickupDiagnosisReasonCode = "provider_missing"
	PickupDiagnosisReasonMachineMissing            PickupDiagnosisReasonCode = "machine_missing"
	PickupDiagnosisReasonMachineOffline            PickupDiagnosisReasonCode = "machine_offline"
	PickupDiagnosisReasonProviderUnknown           PickupDiagnosisReasonCode = "provider_unknown"
	PickupDiagnosisReasonProviderStale             PickupDiagnosisReasonCode = "provider_stale"
	PickupDiagnosisReasonProviderUnavailable       PickupDiagnosisReasonCode = "provider_unavailable"
	PickupDiagnosisReasonWorkflowConcurrencyFull   PickupDiagnosisReasonCode = "workflow_concurrency_full"
	PickupDiagnosisReasonProjectConcurrencyFull    PickupDiagnosisReasonCode = "project_concurrency_full"
	PickupDiagnosisReasonProviderConcurrencyFull   PickupDiagnosisReasonCode = "provider_concurrency_full"
	PickupDiagnosisReasonStatusCapacityFull        PickupDiagnosisReasonCode = "status_capacity_full"
	PickupDiagnosisReasonSchedulerUnavailable      PickupDiagnosisReasonCode = "scheduler_unavailable"
)

type PickupDiagnosisReasonSeverity string

const (
	PickupDiagnosisReasonSeverityInfo    PickupDiagnosisReasonSeverity = "info"
	PickupDiagnosisReasonSeverityWarning PickupDiagnosisReasonSeverity = "warning"
	PickupDiagnosisReasonSeverityError   PickupDiagnosisReasonSeverity = "error"
)

type PickupDiagnosis struct {
	State                PickupDiagnosisState           `json:"state"`
	PrimaryReasonCode    PickupDiagnosisReasonCode      `json:"primary_reason_code"`
	PrimaryReasonMessage string                         `json:"primary_reason_message"`
	NextActionHint       string                         `json:"next_action_hint,omitempty"`
	Reasons              []PickupDiagnosisReason        `json:"reasons"`
	Workflow             *PickupDiagnosisWorkflow       `json:"workflow,omitempty"`
	Agent                *PickupDiagnosisAgent          `json:"agent,omitempty"`
	Provider             *PickupDiagnosisProvider       `json:"provider,omitempty"`
	Retry                PickupDiagnosisRetry           `json:"retry"`
	Capacity             PickupDiagnosisCapacity        `json:"capacity"`
	BlockedBy            []PickupDiagnosisBlockedTicket `json:"blocked_by"`
}

type PickupDiagnosisReason struct {
	Code     PickupDiagnosisReasonCode     `json:"code"`
	Message  string                        `json:"message"`
	Severity PickupDiagnosisReasonSeverity `json:"severity"`
}

type PickupDiagnosisWorkflow struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	IsActive          bool      `json:"is_active"`
	PickupStatusMatch bool      `json:"pickup_status_match"`
}

type PickupDiagnosisAgent struct {
	ID                  uuid.UUID                              `json:"id"`
	Name                string                                 `json:"name"`
	RuntimeControlState catalogdomain.AgentRuntimeControlState `json:"runtime_control_state"`
}

type PickupDiagnosisProvider struct {
	ID                 uuid.UUID                                    `json:"id"`
	Name               string                                       `json:"name"`
	MachineID          uuid.UUID                                    `json:"machine_id"`
	MachineName        string                                       `json:"machine_name"`
	MachineStatus      catalogdomain.MachineStatus                  `json:"machine_status"`
	AvailabilityState  catalogdomain.AgentProviderAvailabilityState `json:"availability_state"`
	AvailabilityReason *string                                      `json:"availability_reason,omitempty"`
}

type PickupDiagnosisRetry struct {
	AttemptCount int        `json:"attempt_count"`
	RetryPaused  bool       `json:"retry_paused"`
	PauseReason  string     `json:"pause_reason,omitempty"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
}

type PickupDiagnosisCapacity struct {
	Workflow PickupDiagnosisCapacityBucket `json:"workflow"`
	Project  PickupDiagnosisCapacityBucket `json:"project"`
	Provider PickupDiagnosisCapacityBucket `json:"provider"`
	Status   PickupDiagnosisStatusCapacity `json:"status"`
}

type PickupDiagnosisCapacityBucket struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   int  `json:"capacity"`
}

type PickupDiagnosisStatusCapacity struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   *int `json:"capacity"`
}

type PickupDiagnosisBlockedTicket struct {
	ID         uuid.UUID `json:"id"`
	Identifier string    `json:"identifier"`
	Title      string    `json:"title"`
	StatusID   uuid.UUID `json:"status_id"`
	StatusName string    `json:"status_name"`
}

type pickupDiagnosisBuildContext struct {
	now                time.Time
	ticket             *ent.Ticket
	project            *ent.Project
	matchingWorkflows  []*ent.Workflow
	activeWorkflow     *ent.Workflow
	agent              *ent.Agent
	provider           *ent.AgentProvider
	machine            *ent.Machine
	providerState      *catalogdomain.AgentProvider
	blockedBy          []PickupDiagnosisBlockedTicket
	workflowActiveRuns int
	projectActiveRuns  int
	providerActiveRuns int
	statusActiveRuns   int
}

func (s *Service) GetPickupDiagnosis(ctx context.Context, ticketID uuid.UUID) (PickupDiagnosis, error) {
	if s == nil || s.client == nil {
		return newPickupDiagnosis(
			PickupDiagnosisStateUnavailable,
			PickupDiagnosisReasonSchedulerUnavailable,
			"Scheduler state is unavailable.",
			"Retry once the ticket service is available again.",
			nil,
		), ErrUnavailable
	}

	ctxData, err := s.loadPickupDiagnosisContext(ctx, ticketID)
	if err != nil {
		return PickupDiagnosis{}, err
	}

	return buildPickupDiagnosis(ctxData), nil
}

func (s *Service) loadPickupDiagnosisContext(ctx context.Context, ticketID uuid.UUID) (pickupDiagnosisBuildContext, error) {
	now := timeNowUTC()
	ticketItem, err := s.client.Ticket.Query().
		Where(entticket.ID(ticketID)).
		WithStatus().
		Only(ctx)
	if err != nil {
		return pickupDiagnosisBuildContext{}, s.mapTicketReadError("get pickup diagnosis ticket", err)
	}

	projectItem, err := s.client.Project.Get(ctx, ticketItem.ProjectID)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("get pickup diagnosis project: %w", err)
	}

	matchingWorkflows, err := s.client.Workflow.Query().
		Where(
			entworkflow.ProjectIDEQ(ticketItem.ProjectID),
			entworkflow.HasPickupStatusesWith(entticketstatus.IDEQ(ticketItem.StatusID)),
		).
		Order(ent.Asc(entworkflow.FieldName)).
		All(ctx)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("list pickup diagnosis workflows: %w", err)
	}

	blockedBy, err := s.loadBlockedByTickets(ctx, ticketItem.ID)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("list pickup diagnosis blockers: %w", err)
	}

	buildCtx := pickupDiagnosisBuildContext{
		now:               now,
		ticket:            ticketItem,
		project:           projectItem,
		matchingWorkflows: matchingWorkflows,
		blockedBy:         blockedBy,
	}
	buildCtx.activeWorkflow = firstActiveWorkflow(matchingWorkflows)
	if buildCtx.activeWorkflow == nil {
		return buildCtx, nil
	}

	agentItem, err := s.loadWorkflowDiagnosisAgent(ctx, buildCtx.activeWorkflow)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}
	buildCtx.agent = agentItem
	if agentItem == nil {
		return buildCtx, nil
	}

	providerItem, machineItem, providerState, err := s.loadWorkflowDiagnosisProvider(ctx, projectItem.OrganizationID, agentItem, now)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}
	buildCtx.provider = providerItem
	buildCtx.machine = machineItem
	buildCtx.providerState = providerState

	buildCtx.workflowActiveRuns, buildCtx.projectActiveRuns, buildCtx.providerActiveRuns, buildCtx.statusActiveRuns, err = s.loadPickupDiagnosisActiveRuns(
		ctx,
		buildCtx.activeWorkflow,
		projectItem,
		providerItem,
		ticketItem.StatusID,
	)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}

	return buildCtx, nil
}

func (s *Service) loadBlockedByTickets(ctx context.Context, ticketID uuid.UUID) ([]PickupDiagnosisBlockedTicket, error) {
	dependencies, err := s.client.TicketDependency.Query().
		Where(
			entticketdependency.TargetTicketIDEQ(ticketID),
			entticketdependency.TypeEQ(entticketdependency.TypeBlocks),
		).
		WithSourceTicket(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		Order(ent.Asc(entticketdependency.FieldSourceTicketID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	blockedBy := make([]PickupDiagnosisBlockedTicket, 0, len(dependencies))
	for _, dependency := range dependencies {
		sourceTicket := dependency.Edges.SourceTicket
		if sourceTicket == nil || pickupDiagnosisDependencyResolved(sourceTicket) {
			continue
		}
		blockedBy = append(blockedBy, PickupDiagnosisBlockedTicket{
			ID:         sourceTicket.ID,
			Identifier: sourceTicket.Identifier,
			Title:      sourceTicket.Title,
			StatusID:   sourceTicket.StatusID,
			StatusName: sourceTicket.Edges.Status.Name,
		})
	}

	return blockedBy, nil
}

func firstActiveWorkflow(workflows []*ent.Workflow) *ent.Workflow {
	for _, workflowItem := range workflows {
		if workflowItem.IsActive {
			return workflowItem
		}
	}
	return nil
}

func (s *Service) loadWorkflowDiagnosisAgent(ctx context.Context, workflowItem *ent.Workflow) (*ent.Agent, error) {
	if workflowItem == nil || workflowItem.AgentID == nil {
		return nil, nil
	}

	agentItem, err := s.client.Agent.Query().
		Where(
			entagent.IDEQ(*workflowItem.AgentID),
			entagent.ProjectIDEQ(workflowItem.ProjectID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get pickup diagnosis agent: %w", err)
	}

	return agentItem, nil
}

func (s *Service) loadWorkflowDiagnosisProvider(
	ctx context.Context,
	organizationID uuid.UUID,
	agentItem *ent.Agent,
	now time.Time,
) (*ent.AgentProvider, *ent.Machine, *catalogdomain.AgentProvider, error) {
	if agentItem == nil {
		return nil, nil, nil, nil
	}

	providerItem, err := s.client.AgentProvider.Get(ctx, agentItem.ProviderID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("get pickup diagnosis provider: %w", err)
	}

	machineItem, err := s.client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(organizationID),
			entmachine.IDEQ(providerItem.MachineID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return providerItem, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("get pickup diagnosis machine: %w", err)
	}

	state := catalogdomain.DeriveAgentProviderAvailability(catalogdomain.AgentProvider{
		ID:                   providerItem.ID,
		OrganizationID:       providerItem.OrganizationID,
		MachineID:            providerItem.MachineID,
		MachineName:          machineItem.Name,
		MachineHost:          machineItem.Host,
		MachineStatus:        catalogdomain.MachineStatus(machineItem.Status),
		MachineSSHUser:       optionalStringPointer(machineItem.SSHUser),
		MachineWorkspaceRoot: optionalStringPointer(machineItem.WorkspaceRoot),
		MachineAgentCLIPath:  optionalStringPointer(machineItem.AgentCliPath),
		MachineResources:     cloneAnyMap(machineItem.Resources),
		Name:                 providerItem.Name,
		AdapterType:          catalogdomain.AgentProviderAdapterType(providerItem.AdapterType),
		CliCommand:           providerItem.CliCommand,
		CliArgs:              append([]string(nil), providerItem.CliArgs...),
		AuthConfig:           cloneAnyMap(providerItem.AuthConfig),
		ModelName:            providerItem.ModelName,
		ModelTemperature:     providerItem.ModelTemperature,
		ModelMaxTokens:       providerItem.ModelMaxTokens,
		MaxParallelRuns:      providerItem.MaxParallelRuns,
		CostPerInputToken:    providerItem.CostPerInputToken,
		CostPerOutputToken:   providerItem.CostPerOutputToken,
	}, now)

	return providerItem, machineItem, &state, nil
}

func (s *Service) loadPickupDiagnosisActiveRuns(
	ctx context.Context,
	workflowItem *ent.Workflow,
	projectItem *ent.Project,
	providerItem *ent.AgentProvider,
	statusID uuid.UUID,
) (int, int, int, int, error) {
	workflowActiveRuns := 0
	if workflowItem != nil {
		count, err := s.client.Ticket.Query().
			Where(
				entticket.WorkflowIDEQ(workflowItem.ID),
				entticket.CurrentRunIDNotNil(),
			).
			Count(ctx)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis workflow active runs: %w", err)
		}
		workflowActiveRuns = count
	}

	projectActiveRuns := 0
	if projectItem != nil {
		count, err := s.client.Ticket.Query().
			Where(
				entticket.ProjectIDEQ(projectItem.ID),
				entticket.CurrentRunIDNotNil(),
			).
			Count(ctx)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis project active runs: %w", err)
		}
		projectActiveRuns = count
	}

	providerActiveRuns := 0
	if providerItem != nil {
		count, err := s.client.AgentRun.Query().
			Where(
				entagentrun.ProviderIDEQ(providerItem.ID),
				entagentrun.HasCurrentForTicket(),
			).
			Count(ctx)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis provider active runs: %w", err)
		}
		providerActiveRuns = count
	}

	statusActiveRuns, err := s.client.Ticket.Query().
		Where(
			entticket.StatusIDEQ(statusID),
			entticket.CurrentRunIDNotNil(),
		).
		Count(ctx)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis status active runs: %w", err)
	}

	return workflowActiveRuns, projectActiveRuns, providerActiveRuns, statusActiveRuns, nil
}

func buildPickupDiagnosis(ctx pickupDiagnosisBuildContext) PickupDiagnosis {
	diagnosis := newPickupDiagnosis(
		PickupDiagnosisStateRunnable,
		PickupDiagnosisReasonReadyForPickup,
		"Ticket is ready for pickup.",
		"Wait for the scheduler to claim the ticket.",
		nil,
	)
	diagnosis.Workflow = mapPickupDiagnosisWorkflow(ctx.activeWorkflow, ctx.matchingWorkflows)
	diagnosis.Agent = mapPickupDiagnosisAgent(ctx.agent)
	diagnosis.Provider = mapPickupDiagnosisProvider(ctx.provider, ctx.machine, ctx.providerState)
	diagnosis.Retry = PickupDiagnosisRetry{
		AttemptCount: ctx.ticket.AttemptCount,
		RetryPaused:  ctx.ticket.RetryPaused,
		PauseReason:  ctx.ticket.PauseReason,
		NextRetryAt:  cloneTime(ctx.ticket.NextRetryAt),
	}
	diagnosis.Capacity = buildPickupDiagnosisCapacity(ctx)
	diagnosis.BlockedBy = append(diagnosis.BlockedBy, ctx.blockedBy...)

	statusStage := ticketing.StatusStage(ctx.ticket.Edges.Status.Stage)
	if ctx.ticket.CompletedAt != nil || (statusStage.IsValid() && statusStage.IsTerminal()) {
		return newPickupDiagnosis(
			PickupDiagnosisStateCompleted,
			PickupDiagnosisReasonCompleted,
			"Ticket is already in a terminal state.",
			"No pickup is needed.",
			diagnosisSummary(ctx, PickupDiagnosisReasonCompleted, PickupDiagnosisReasonSeverityInfo, "Ticket is already completed or otherwise terminal."),
		)
	}

	if ctx.ticket.CurrentRunID != nil {
		diagnosis.State = PickupDiagnosisStateRunning
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonRunningCurrentRun
		diagnosis.PrimaryReasonMessage = "Ticket already has an active run."
		diagnosis.NextActionHint = "Wait for the current run to finish or inspect the active runtime."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonRunningCurrentRun, PickupDiagnosisReasonSeverityInfo, "Current run is still attached to the ticket.")
		return diagnosis
	}

	if ctx.ticket.RetryPaused {
		diagnosis.State = PickupDiagnosisStateBlocked
		switch ticketing.PauseReason(ctx.ticket.PauseReason) {
		case ticketing.PauseReasonRepeatedStalls:
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonRetryPausedRepeatedStalls
			diagnosis.PrimaryReasonMessage = "Retries are paused after repeated stalls."
			diagnosis.NextActionHint = "Review the last failed attempt, then continue retry when ready."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonRetryPausedRepeatedStalls, PickupDiagnosisReasonSeverityWarning, "Manual retry is required after repeated stalls.")
		case ticketing.PauseReasonBudgetExhausted:
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonRetryPausedBudget
			diagnosis.PrimaryReasonMessage = "Retries are paused because the ticket budget is exhausted."
			diagnosis.NextActionHint = "Increase the budget or reduce runtime cost before retrying."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonRetryPausedBudget, PickupDiagnosisReasonSeverityWarning, "Budget exhaustion paused further retries.")
		default:
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonRetryPausedUser
			diagnosis.PrimaryReasonMessage = "Retries are paused manually."
			diagnosis.NextActionHint = "Resume retries when you want the ticket to become schedulable again."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonRetryPausedUser, PickupDiagnosisReasonSeverityWarning, "Retries stay paused until they are resumed manually.")
		}
		return diagnosis
	}

	if ctx.ticket.NextRetryAt != nil && ctx.ticket.NextRetryAt.After(ctx.now) {
		nextRetry := ctx.ticket.NextRetryAt.UTC().Format(time.RFC3339)
		diagnosis.State = PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonRetryBackoff
		diagnosis.PrimaryReasonMessage = "Waiting for retry backoff to expire."
		diagnosis.NextActionHint = "The ticket will become schedulable automatically after the retry window expires."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonRetryBackoff, PickupDiagnosisReasonSeverityInfo, "Next retry is scheduled for "+nextRetry+".")
		return diagnosis
	}

	if len(ctx.blockedBy) > 0 {
		diagnosis.State = PickupDiagnosisStateBlocked
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonBlockedDependency
		diagnosis.PrimaryReasonMessage = "Waiting for blocking tickets to finish."
		diagnosis.NextActionHint = "Resolve the blocking tickets or move them to a terminal status."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonBlockedDependency, PickupDiagnosisReasonSeverityWarning, blockedDependencyMessage(ctx.blockedBy))
		return diagnosis
	}

	if len(ctx.matchingWorkflows) == 0 {
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonNoMatchingActiveWorkflow
		diagnosis.PrimaryReasonMessage = "No workflow picks up the ticket's current status."
		diagnosis.NextActionHint = "Add an active workflow for this status or move the ticket into a pickup status."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonNoMatchingActiveWorkflow, PickupDiagnosisReasonSeverityError, "No workflow in this project picks up status "+ctx.ticket.Edges.Status.Name+".")
		return diagnosis
	}

	if ctx.activeWorkflow == nil {
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.Workflow = mapPickupDiagnosisWorkflow(ctx.matchingWorkflows[0], ctx.matchingWorkflows)
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonWorkflowInactive
		diagnosis.PrimaryReasonMessage = "Matching workflow is inactive."
		diagnosis.NextActionHint = "Reactivate the workflow or move the ticket into a status handled by an active workflow."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonWorkflowInactive, PickupDiagnosisReasonSeverityError, "Workflow "+ctx.matchingWorkflows[0].Name+" matches the current status but is inactive.")
		return diagnosis
	}

	if ctx.activeWorkflow.AgentID == nil {
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonWorkflowMissingAgent
		diagnosis.PrimaryReasonMessage = "Workflow has no bound agent."
		diagnosis.NextActionHint = "Bind an agent to the workflow before expecting pickup."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonWorkflowMissingAgent, PickupDiagnosisReasonSeverityError, "Workflow "+ctx.activeWorkflow.Name+" does not have a bound agent.")
		return diagnosis
	}

	if ctx.agent == nil {
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonAgentMissing
		diagnosis.PrimaryReasonMessage = "Workflow agent is missing."
		diagnosis.NextActionHint = "Rebind the workflow to an existing agent."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonAgentMissing, PickupDiagnosisReasonSeverityError, "The workflow's bound agent record could not be found.")
		return diagnosis
	}

	switch ctx.agent.RuntimeControlState {
	case entagent.RuntimeControlStatePauseRequested:
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonAgentPauseRequested
		diagnosis.PrimaryReasonMessage = "Agent pause has been requested."
		diagnosis.NextActionHint = "Wait for the runtime to settle or resume the agent once it reaches paused."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonAgentPauseRequested, PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is transitioning toward a paused state.")
		return diagnosis
	case entagent.RuntimeControlStatePaused:
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonAgentPaused
		diagnosis.PrimaryReasonMessage = "Agent is paused."
		diagnosis.NextActionHint = "Resume the agent to allow pickup."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonAgentPaused, PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is paused.")
		return diagnosis
	}

	if ctx.provider == nil {
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonProviderMissing
		diagnosis.PrimaryReasonMessage = "Agent provider is missing."
		diagnosis.NextActionHint = "Reconnect the agent to a valid provider."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonProviderMissing, PickupDiagnosisReasonSeverityError, "The agent's provider record could not be found.")
		return diagnosis
	}

	if ctx.machine == nil {
		diagnosis.State = PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonMachineMissing
		diagnosis.PrimaryReasonMessage = "Provider machine is missing."
		diagnosis.NextActionHint = "Reconnect the provider to an existing machine."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonMachineMissing, PickupDiagnosisReasonSeverityError, "Provider "+ctx.provider.Name+" is bound to a machine that could not be found.")
		return diagnosis
	}

	if ctx.machine.Status != entmachine.StatusOnline {
		diagnosis.State = PickupDiagnosisStateUnavailable
		if ctx.machine.Status == entmachine.StatusOffline {
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonMachineOffline
			diagnosis.PrimaryReasonMessage = "Provider machine is offline."
			diagnosis.NextActionHint = "Bring the machine back online before retrying pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonMachineOffline, PickupDiagnosisReasonSeverityError, "Machine "+ctx.machine.Name+" is offline.")
		} else {
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonProviderUnavailable
			diagnosis.PrimaryReasonMessage = "Provider machine is not available."
			diagnosis.NextActionHint = "Recover the machine before expecting pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonProviderUnavailable, PickupDiagnosisReasonSeverityError, "Machine "+ctx.machine.Name+" is "+strings.ToLower(string(ctx.machine.Status))+".")
		}
		return diagnosis
	}

	if ctx.providerState != nil {
		switch ctx.providerState.AvailabilityState {
		case catalogdomain.AgentProviderAvailabilityStateUnknown:
			diagnosis.State = PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonProviderUnknown
			diagnosis.PrimaryReasonMessage = "Provider availability is unknown."
			diagnosis.NextActionHint = "Refresh machine health so the provider can be probed again."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonProviderUnknown, PickupDiagnosisReasonSeverityWarning, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		case catalogdomain.AgentProviderAvailabilityStateStale:
			diagnosis.State = PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonProviderStale
			diagnosis.PrimaryReasonMessage = "Provider health information is stale."
			diagnosis.NextActionHint = "Refresh provider health to confirm availability."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonProviderStale, PickupDiagnosisReasonSeverityWarning, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		case catalogdomain.AgentProviderAvailabilityStateUnavailable:
			diagnosis.State = PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = PickupDiagnosisReasonProviderUnavailable
			diagnosis.PrimaryReasonMessage = "Provider is unavailable."
			diagnosis.NextActionHint = "Fix the provider health issue before expecting pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonProviderUnavailable, PickupDiagnosisReasonSeverityError, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		}
	}

	if ctx.activeWorkflow.MaxConcurrent > 0 && ctx.workflowActiveRuns >= ctx.activeWorkflow.MaxConcurrent {
		diagnosis.State = PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonWorkflowConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Workflow concurrency is full."
		diagnosis.NextActionHint = "Wait for an active run in this workflow to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonWorkflowConcurrencyFull, PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Workflow %s is using %d of %d allowed runs.", ctx.activeWorkflow.Name, ctx.workflowActiveRuns, ctx.activeWorkflow.MaxConcurrent))
		return diagnosis
	}

	if ctx.project.MaxConcurrentAgents > 0 && ctx.projectActiveRuns >= ctx.project.MaxConcurrentAgents {
		diagnosis.State = PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonProjectConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Project concurrency is full."
		diagnosis.NextActionHint = "Wait for another active run in this project to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonProjectConcurrencyFull, PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Project is using %d of %d allowed runs.", ctx.projectActiveRuns, ctx.project.MaxConcurrentAgents))
		return diagnosis
	}

	if ctx.provider.MaxParallelRuns > 0 && ctx.providerActiveRuns >= ctx.provider.MaxParallelRuns {
		diagnosis.State = PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonProviderConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Provider concurrency is full."
		diagnosis.NextActionHint = "Wait for another run on this provider to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonProviderConcurrencyFull, PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Provider %s is using %d of %d allowed runs.", ctx.provider.Name, ctx.providerActiveRuns, ctx.provider.MaxParallelRuns))
		return diagnosis
	}

	if ctx.ticket.Edges.Status.MaxActiveRuns != nil && ctx.statusActiveRuns >= *ctx.ticket.Edges.Status.MaxActiveRuns {
		diagnosis.State = PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = PickupDiagnosisReasonStatusCapacityFull
		diagnosis.PrimaryReasonMessage = "Status capacity is full."
		diagnosis.NextActionHint = "Wait for another active run in this status to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonStatusCapacityFull, PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Status %s is using %d of %d allowed runs.", ctx.ticket.Edges.Status.Name, ctx.statusActiveRuns, *ctx.ticket.Edges.Status.MaxActiveRuns))
		return diagnosis
	}

	diagnosis.Reasons = diagnosisSummary(ctx, PickupDiagnosisReasonReadyForPickup, PickupDiagnosisReasonSeverityInfo, "The scheduler can claim this ticket on the next tick.")
	return diagnosis
}

func newPickupDiagnosis(
	state PickupDiagnosisState,
	code PickupDiagnosisReasonCode,
	message string,
	hint string,
	reasons []PickupDiagnosisReason,
) PickupDiagnosis {
	return PickupDiagnosis{
		State:                state,
		PrimaryReasonCode:    code,
		PrimaryReasonMessage: message,
		NextActionHint:       hint,
		Reasons:              append([]PickupDiagnosisReason{}, reasons...),
		BlockedBy:            []PickupDiagnosisBlockedTicket{},
	}
}

func diagnosisSummary(
	ctx pickupDiagnosisBuildContext,
	code PickupDiagnosisReasonCode,
	severity PickupDiagnosisReasonSeverity,
	message string,
) []PickupDiagnosisReason {
	reasons := []PickupDiagnosisReason{{
		Code:     code,
		Message:  message,
		Severity: severity,
	}}

	if len(ctx.blockedBy) > 1 && code == PickupDiagnosisReasonBlockedDependency {
		for _, blocker := range ctx.blockedBy {
			reasons = append(reasons, PickupDiagnosisReason{
				Code:     PickupDiagnosisReasonBlockedDependency,
				Message:  blocker.Identifier + " " + blocker.Title,
				Severity: PickupDiagnosisReasonSeverityWarning,
			})
		}
	}

	return reasons
}

func blockedDependencyMessage(blockers []PickupDiagnosisBlockedTicket) string {
	if len(blockers) == 0 {
		return "Blocking dependency is unresolved."
	}

	parts := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		parts = append(parts, blocker.Identifier+" "+blocker.Title)
	}
	return "Blocked by " + strings.Join(parts, ", ") + "."
}

func mapPickupDiagnosisWorkflow(activeWorkflow *ent.Workflow, matchingWorkflows []*ent.Workflow) *PickupDiagnosisWorkflow {
	if activeWorkflow != nil {
		return &PickupDiagnosisWorkflow{
			ID:                activeWorkflow.ID,
			Name:              activeWorkflow.Name,
			IsActive:          activeWorkflow.IsActive,
			PickupStatusMatch: true,
		}
	}
	if len(matchingWorkflows) == 0 {
		return nil
	}
	workflowItem := matchingWorkflows[0]
	return &PickupDiagnosisWorkflow{
		ID:                workflowItem.ID,
		Name:              workflowItem.Name,
		IsActive:          workflowItem.IsActive,
		PickupStatusMatch: true,
	}
}

func mapPickupDiagnosisAgent(agentItem *ent.Agent) *PickupDiagnosisAgent {
	if agentItem == nil {
		return nil
	}
	return &PickupDiagnosisAgent{
		ID:                  agentItem.ID,
		Name:                agentItem.Name,
		RuntimeControlState: catalogdomain.AgentRuntimeControlState(agentItem.RuntimeControlState),
	}
}

func mapPickupDiagnosisProvider(
	providerItem *ent.AgentProvider,
	machineItem *ent.Machine,
	providerState *catalogdomain.AgentProvider,
) *PickupDiagnosisProvider {
	if providerItem == nil {
		return nil
	}

	response := &PickupDiagnosisProvider{
		ID:   providerItem.ID,
		Name: providerItem.Name,
	}
	if machineItem != nil {
		response.MachineID = machineItem.ID
		response.MachineName = machineItem.Name
		response.MachineStatus = catalogdomain.MachineStatus(machineItem.Status)
	}
	if providerState != nil {
		response.AvailabilityState = providerState.AvailabilityState
		response.AvailabilityReason = cloneString(providerState.AvailabilityReason)
	}
	return response
}

func buildPickupDiagnosisCapacity(ctx pickupDiagnosisBuildContext) PickupDiagnosisCapacity {
	capacity := PickupDiagnosisCapacity{}
	if ctx.activeWorkflow != nil {
		capacity.Workflow = PickupDiagnosisCapacityBucket{
			Limited:    ctx.activeWorkflow.MaxConcurrent > 0,
			ActiveRuns: ctx.workflowActiveRuns,
			Capacity:   ctx.activeWorkflow.MaxConcurrent,
		}
	}
	if ctx.project != nil {
		capacity.Project = PickupDiagnosisCapacityBucket{
			Limited:    ctx.project.MaxConcurrentAgents > 0,
			ActiveRuns: ctx.projectActiveRuns,
			Capacity:   ctx.project.MaxConcurrentAgents,
		}
	}
	if ctx.provider != nil {
		capacity.Provider = PickupDiagnosisCapacityBucket{
			Limited:    ctx.provider.MaxParallelRuns > 0,
			ActiveRuns: ctx.providerActiveRuns,
			Capacity:   ctx.provider.MaxParallelRuns,
		}
	}
	if ctx.ticket != nil && ctx.ticket.Edges.Status != nil {
		capacity.Status = PickupDiagnosisStatusCapacity{
			Limited:    ctx.ticket.Edges.Status.MaxActiveRuns != nil,
			ActiveRuns: ctx.statusActiveRuns,
			Capacity:   cloneInt(ctx.ticket.Edges.Status.MaxActiveRuns),
		}
	}
	return capacity
}

func providerAvailabilityMessage(reason *string) string {
	switch strings.TrimSpace(stringValue(reason)) {
	case "machine_offline":
		return "Provider machine is offline."
	case "machine_degraded":
		return "Provider machine is degraded."
	case "machine_maintenance":
		return "Provider machine is in maintenance mode."
	case "l4_snapshot_missing":
		return "Provider health has not been probed yet."
	case "stale_l4_snapshot":
		return "Provider health snapshot is stale."
	case "cli_missing":
		return "Provider CLI is missing on the machine."
	case "not_logged_in":
		return "Provider CLI is not authenticated."
	case "not_ready":
		return "Provider CLI is not ready."
	case "config_incomplete":
		return "Provider launch configuration is incomplete."
	case "unsupported_adapter":
		return "Provider adapter is not supported by health checks."
	default:
		return "Provider health is blocking pickup."
	}
}

func pickupDiagnosisDependencyResolved(ticketItem *ent.Ticket) bool {
	if ticketItem == nil {
		return false
	}
	if ticketItem.CompletedAt != nil {
		return true
	}
	if ticketItem.Edges.Status == nil {
		return false
	}

	stage := ticketing.StatusStage(ticketItem.Edges.Status.Stage)
	return stage.IsValid() && stage.IsTerminal()
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := strings.TrimSpace(*value)
	return &copied
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
