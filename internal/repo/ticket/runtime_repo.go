package ticket

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

func (r *RuntimeRepository) LoadLifecycleHookRuntimeData(
	ctx context.Context,
	ticketID uuid.UUID,
	runID uuid.UUID,
	workflowID *uuid.UUID,
) (LifecycleHookRuntimeData, error) {
	runItem, err := r.client.AgentRun.Query().
		Where(entagentrun.IDEQ(runID)).
		WithAgent(func(query *ent.AgentQuery) {
			query.WithProvider()
		}).
		Only(ctx)
	if err != nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("load ticket hook run %s: %w", runID, err)
	}
	if runItem.Edges.Agent == nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("ticket hook run %s is missing agent", runID)
	}
	if runItem.Edges.Agent.Edges.Provider == nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("ticket hook run %s agent is missing provider", runID)
	}

	ticketItem, err := r.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return LifecycleHookRuntimeData{}, mapTicketReadError("load ticket for lifecycle hook", err)
	}

	resolvedWorkflowID := ticketItem.WorkflowID
	if workflowID != nil {
		resolvedWorkflowID = workflowID
	}
	if resolvedWorkflowID == nil {
		return LifecycleHookRuntimeData{TicketID: ticketItem.ID}, nil
	}

	workflowItem, err := r.client.Workflow.Query().
		Where(entworkflow.IDEQ(*resolvedWorkflowID)).
		WithCurrentVersion().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return LifecycleHookRuntimeData{TicketID: ticketItem.ID}, nil
		}
		return LifecycleHookRuntimeData{}, fmt.Errorf("load workflow %s for lifecycle hook: %w", *resolvedWorkflowID, err)
	}

	workspaces, err := r.client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runID)).
		Order(ent.Asc(entticketrepoworkspace.FieldRepoPath)).
		WithRepo(func(query *ent.ProjectRepoQuery) {
			query.Order(entprojectrepo.ByName())
		}).
		All(ctx)
	if err != nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("load ticket repo workspaces for run %s: %w", runID, err)
	}

	machineItem, err := r.client.Machine.Get(ctx, runItem.Edges.Agent.Edges.Provider.MachineID)
	if err != nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("load machine for ticket hook run %s: %w", runID, err)
	}

	repos := make([]domain.HookWorkspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		repoName := strings.TrimSpace(workspace.RepoPath)
		if workspace.Edges.Repo != nil && strings.TrimSpace(workspace.Edges.Repo.Name) != "" {
			repoName = strings.TrimSpace(workspace.Edges.Repo.Name)
		}
		repos = append(repos, domain.HookWorkspace{
			RepoName: repoName,
			RepoPath: strings.TrimSpace(workspace.RepoPath),
		})
	}

	workspaceRoot := ""
	if len(workspaces) > 0 {
		workspaceRoot = strings.TrimSpace(workspaces[0].WorkspaceRoot)
	}

	typeLabel, parseErr := workflowdomain.ParseTypeLabel(workflowItem.Type)
	if parseErr != nil {
		typeLabel = workflowdomain.MustParseTypeLabel("unknown")
	}
	harnessContent := ""
	if workflowItem.Edges.CurrentVersion != nil {
		harnessContent = workflowItem.Edges.CurrentVersion.ContentMarkdown
	}
	workflowFamily := workflowdomain.ClassifyWorkflow(workflowdomain.WorkflowClassificationInput{
		TypeLabel:      typeLabel,
		WorkflowName:   workflowItem.Name,
		HarnessPath:    workflowItem.HarnessPath,
		HarnessContent: harnessContent,
	}).Family

	return LifecycleHookRuntimeData{
		TicketID:              ticketItem.ID,
		ProjectID:             ticketItem.ProjectID,
		AgentID:               runItem.AgentID,
		TicketIdentifier:      ticketItem.Identifier,
		AgentName:             runItem.Edges.Agent.Name,
		WorkflowType:          workflowItem.Type,
		WorkflowFamily:        string(workflowFamily),
		PlatformAccessAllowed: append([]string(nil), workflowItem.PlatformAccessAllowed...),
		Attempt:               ticketItem.AttemptCount + 1,
		WorkspaceRoot:         workspaceRoot,
		Hooks:                 cloneAnyMap(workflowItem.Hooks),
		Machine:               mapTicketHookMachine(machineItem),
		Workspaces:            repos,
	}, nil
}

func mapTicketHookMachine(item *ent.Machine) catalogdomain.Machine {
	if item == nil {
		return catalogdomain.Machine{}
	}

	return catalogdomain.Machine{
		ID:                 item.ID,
		OrganizationID:     item.OrganizationID,
		Name:               item.Name,
		Host:               item.Host,
		Port:               item.Port,
		SSHUser:            cloneOptionalText(item.SSHUser),
		SSHKeyPath:         cloneOptionalText(item.SSHKeyPath),
		Status:             catalogdomain.MachineStatus(item.Status),
		ConnectionMode:     mapStoredTicketMachineConnectionMode(item),
		AdvertisedEndpoint: cloneOptionalText(item.AdvertisedEndpoint),
		WorkspaceRoot:      cloneOptionalText(item.WorkspaceRoot),
		AgentCLIPath:       cloneOptionalText(item.AgentCliPath),
		EnvVars:            slices.Clone(item.EnvVars),
		Resources:          cloneMap(item.Resources),
		DaemonStatus: catalogdomain.MachineDaemonStatus{
			Registered:       item.DaemonRegistered,
			LastRegisteredAt: cloneOptionalTime(item.DaemonLastRegisteredAt),
			CurrentSessionID: cloneOptionalText(item.DaemonSessionID),
			SessionState:     catalogdomain.MachineTransportSessionState(item.DaemonSessionState),
		},
		ChannelCredential: catalogdomain.MachineChannelCredential{
			Kind:          catalogdomain.MachineChannelCredentialKind(item.ChannelCredentialKind),
			TokenID:       cloneOptionalText(item.ChannelTokenID),
			CertificateID: cloneOptionalText(item.ChannelCertificateID),
		},
	}
}

func mapStoredTicketMachineConnectionMode(item *ent.Machine) catalogdomain.MachineConnectionMode {
	if item == nil {
		return ""
	}
	if item.Host == catalogdomain.LocalMachineHost || item.Name == catalogdomain.LocalMachineName {
		return catalogdomain.MachineConnectionModeLocal
	}
	mode, err := catalogdomain.ParseStoredMachineConnectionMode(string(item.ConnectionMode), item.Host)
	if err != nil {
		return catalogdomain.MachineConnectionModeWSListener
	}
	return mode
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
	blockedBy          []domain.PickupDiagnosisBlockedTicket
	workflowActiveRuns int
	projectActiveRuns  int
	providerActiveRuns int
	statusActiveRuns   int
}

func (r *RuntimeRepository) GetPickupDiagnosis(ctx context.Context, ticketID uuid.UUID) (domain.PickupDiagnosis, error) {
	if r == nil || r.client == nil {
		return newPickupDiagnosis(
			domain.PickupDiagnosisStateUnavailable,
			domain.PickupDiagnosisReasonSchedulerUnavailable,
			"Scheduler state is unavailable.",
			"Retry once the ticket service is available again.",
			nil,
		), errUnavailable
	}

	ctxData, err := r.loadPickupDiagnosisContext(ctx, ticketID)
	if err != nil {
		return domain.PickupDiagnosis{}, err
	}

	return buildPickupDiagnosis(ctxData), nil
}

func (r *RuntimeRepository) loadPickupDiagnosisContext(ctx context.Context, ticketID uuid.UUID) (pickupDiagnosisBuildContext, error) {
	now := timeNowUTC()
	ticketItem, err := r.client.Ticket.Query().
		Where(entticket.ID(ticketID)).
		WithStatus().
		Only(ctx)
	if err != nil {
		return pickupDiagnosisBuildContext{}, mapTicketReadError("get pickup diagnosis ticket", err)
	}

	projectItem, err := r.client.Project.Get(ctx, ticketItem.ProjectID)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("get pickup diagnosis project: %w", err)
	}

	matchingWorkflows, err := r.client.Workflow.Query().
		Where(
			entworkflow.ProjectIDEQ(ticketItem.ProjectID),
			entworkflow.HasPickupStatusesWith(entticketstatus.IDEQ(ticketItem.StatusID)),
		).
		Order(ent.Asc(entworkflow.FieldName)).
		All(ctx)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("list pickup diagnosis workflows: %w", err)
	}

	blockedBy, err := r.loadBlockedByTickets(ctx, ticketItem.ID)
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

	agentItem, err := r.loadWorkflowDiagnosisAgent(ctx, buildCtx.activeWorkflow)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}
	buildCtx.agent = agentItem
	if agentItem == nil {
		return buildCtx, nil
	}

	providerItem, machineItem, providerState, err := r.loadWorkflowDiagnosisProvider(
		ctx,
		projectItem.OrganizationID,
		agentItem,
		now,
	)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}
	buildCtx.provider = providerItem
	buildCtx.machine = machineItem
	buildCtx.providerState = providerState

	buildCtx.workflowActiveRuns, buildCtx.projectActiveRuns, buildCtx.providerActiveRuns, buildCtx.statusActiveRuns, err = r.loadPickupDiagnosisActiveRuns(
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

func (r *RuntimeRepository) loadBlockedByTickets(ctx context.Context, ticketID uuid.UUID) ([]domain.PickupDiagnosisBlockedTicket, error) {
	dependencies, err := r.client.TicketDependency.Query().
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

	blockedBy := make([]domain.PickupDiagnosisBlockedTicket, 0, len(dependencies))
	for _, dependency := range dependencies {
		sourceTicket := dependency.Edges.SourceTicket
		if sourceTicket == nil || pickupDiagnosisDependencyResolved(sourceTicket) {
			continue
		}
		blockedBy = append(blockedBy, domain.PickupDiagnosisBlockedTicket{
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

func (r *RuntimeRepository) loadWorkflowDiagnosisAgent(ctx context.Context, workflowItem *ent.Workflow) (*ent.Agent, error) {
	if workflowItem == nil || workflowItem.AgentID == nil {
		return nil, nil
	}

	agentItem, err := r.client.Agent.Query().
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

func (r *RuntimeRepository) loadWorkflowDiagnosisProvider(
	ctx context.Context,
	organizationID uuid.UUID,
	agentItem *ent.Agent,
	now time.Time,
) (*ent.AgentProvider, *ent.Machine, *catalogdomain.AgentProvider, error) {
	if agentItem == nil {
		return nil, nil, nil, nil
	}

	providerItem, err := r.client.AgentProvider.Get(ctx, agentItem.ProviderID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("get pickup diagnosis provider: %w", err)
	}

	machineItem, err := r.client.Machine.Query().
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

func (r *RuntimeRepository) loadPickupDiagnosisActiveRuns(
	ctx context.Context,
	workflowItem *ent.Workflow,
	projectItem *ent.Project,
	providerItem *ent.AgentProvider,
	statusID uuid.UUID,
) (int, int, int, int, error) {
	workflowActiveRuns := 0
	if workflowItem != nil {
		count, err := r.client.Ticket.Query().
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
		count, err := r.client.Ticket.Query().
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
		count, err := r.client.AgentRun.Query().
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

	statusActiveRuns, err := r.client.Ticket.Query().
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

func buildPickupDiagnosis(ctx pickupDiagnosisBuildContext) domain.PickupDiagnosis {
	diagnosis := newPickupDiagnosis(
		domain.PickupDiagnosisStateRunnable,
		domain.PickupDiagnosisReasonReadyForPickup,
		"Ticket is ready for pickup.",
		"Wait for the scheduler to claim the ticket.",
		nil,
	)
	diagnosis.Workflow = mapPickupDiagnosisWorkflow(ctx.activeWorkflow, ctx.matchingWorkflows)
	diagnosis.Agent = mapPickupDiagnosisAgent(ctx.agent)
	diagnosis.Provider = mapPickupDiagnosisProvider(ctx.provider, ctx.machine, ctx.providerState)
	diagnosis.Retry = domain.PickupDiagnosisRetry{
		AttemptCount: ctx.ticket.AttemptCount,
		RetryPaused:  ctx.ticket.RetryPaused,
		PauseReason:  ctx.ticket.PauseReason,
		NextRetryAt:  cloneTime(ctx.ticket.NextRetryAt),
	}
	diagnosis.Capacity = buildPickupDiagnosisCapacity(ctx)
	diagnosis.BlockedBy = append(diagnosis.BlockedBy, ctx.blockedBy...)

	if ctx.ticket.Archived {
		return newPickupDiagnosis(
			domain.PickupDiagnosisStateCompleted,
			domain.PickupDiagnosisReasonArchived,
			"Ticket is archived.",
			"Unarchive the ticket before pickup can resume.",
			diagnosisSummary(ctx, domain.PickupDiagnosisReasonArchived, domain.PickupDiagnosisReasonSeverityInfo, "Archived tickets are excluded from pickup."),
		)
	}

	statusStage := ticketing.StatusStage(ctx.ticket.Edges.Status.Stage)
	if ctx.ticket.CompletedAt != nil || (statusStage.IsValid() && statusStage.IsTerminal()) {
		return newPickupDiagnosis(
			domain.PickupDiagnosisStateCompleted,
			domain.PickupDiagnosisReasonCompleted,
			"Ticket is already in a terminal state.",
			"No pickup is needed.",
			diagnosisSummary(ctx, domain.PickupDiagnosisReasonCompleted, domain.PickupDiagnosisReasonSeverityInfo, "Ticket is already completed or otherwise terminal."),
		)
	}

	if ctx.ticket.CurrentRunID != nil {
		diagnosis.State = domain.PickupDiagnosisStateRunning
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRunningCurrentRun
		diagnosis.PrimaryReasonMessage = "Ticket already has an active run."
		diagnosis.NextActionHint = "Wait for the current run to finish or inspect the active runtime."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRunningCurrentRun, domain.PickupDiagnosisReasonSeverityInfo, "Current run is still attached to the ticket.")
		return diagnosis
	}

	if ctx.ticket.RetryPaused {
		diagnosis.State = domain.PickupDiagnosisStateBlocked
		switch ticketing.PauseReason(ctx.ticket.PauseReason) {
		case ticketing.PauseReasonRepeatedStalls:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedRepeatedStalls
			diagnosis.PrimaryReasonMessage = "Retries are paused after repeated stalls."
			diagnosis.NextActionHint = "Review the last failed attempt, then continue retry when ready."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedRepeatedStalls, domain.PickupDiagnosisReasonSeverityWarning, "Manual retry is required after repeated stalls.")
		case ticketing.PauseReasonBudgetExhausted:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedBudget
			diagnosis.PrimaryReasonMessage = "Retries are paused because the ticket budget is exhausted."
			diagnosis.NextActionHint = "Increase the budget or reduce runtime cost before retrying."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedBudget, domain.PickupDiagnosisReasonSeverityWarning, "Budget exhaustion paused further retries.")
		case ticketing.PauseReasonUserInterrupted:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedInterrupted
			diagnosis.PrimaryReasonMessage = "Retries are paused because the current run was interrupted."
			diagnosis.NextActionHint = "Resume retries when you want the agent to pick the ticket up again."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedInterrupted, domain.PickupDiagnosisReasonSeverityWarning, "The last active run was interrupted by an operator.")
		default:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedUser
			diagnosis.PrimaryReasonMessage = "Retries are paused manually."
			diagnosis.NextActionHint = "Resume retries when you want the ticket to become schedulable again."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedUser, domain.PickupDiagnosisReasonSeverityWarning, "Retries stay paused until they are resumed manually.")
		}
		return diagnosis
	}

	if ctx.ticket.NextRetryAt != nil && ctx.ticket.NextRetryAt.After(ctx.now) {
		nextRetry := ctx.ticket.NextRetryAt.UTC().Format(time.RFC3339)
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryBackoff
		diagnosis.PrimaryReasonMessage = "Waiting for retry backoff to expire."
		diagnosis.NextActionHint = "The ticket will become schedulable automatically after the retry window expires."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryBackoff, domain.PickupDiagnosisReasonSeverityInfo, "Next retry is scheduled for "+nextRetry+".")
		return diagnosis
	}

	if len(ctx.blockedBy) > 0 {
		diagnosis.State = domain.PickupDiagnosisStateBlocked
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonBlockedDependency
		diagnosis.PrimaryReasonMessage = "Waiting for blocking tickets to finish."
		diagnosis.NextActionHint = "Resolve the blocking tickets or move them to a terminal status."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonBlockedDependency, domain.PickupDiagnosisReasonSeverityWarning, blockedDependencyMessage(ctx.blockedBy))
		return diagnosis
	}

	if len(ctx.matchingWorkflows) == 0 {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonNoMatchingActiveWorkflow
		diagnosis.PrimaryReasonMessage = "No workflow picks up the ticket's current status."
		diagnosis.NextActionHint = "Add an active workflow for this status or move the ticket into a pickup status."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonNoMatchingActiveWorkflow, domain.PickupDiagnosisReasonSeverityError, "No workflow in this project picks up status "+ctx.ticket.Edges.Status.Name+".")
		return diagnosis
	}

	if ctx.activeWorkflow == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.Workflow = mapPickupDiagnosisWorkflow(ctx.matchingWorkflows[0], ctx.matchingWorkflows)
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonWorkflowInactive
		diagnosis.PrimaryReasonMessage = "Matching workflow is inactive."
		diagnosis.NextActionHint = "Reactivate the workflow or move the ticket into a status handled by an active workflow."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonWorkflowInactive, domain.PickupDiagnosisReasonSeverityError, "Workflow "+ctx.matchingWorkflows[0].Name+" matches the current status but is inactive.")
		return diagnosis
	}

	if ctx.activeWorkflow.AgentID == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonWorkflowMissingAgent
		diagnosis.PrimaryReasonMessage = "Workflow has no bound agent."
		diagnosis.NextActionHint = "Bind an agent to the workflow before expecting pickup."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonWorkflowMissingAgent, domain.PickupDiagnosisReasonSeverityError, "Workflow "+ctx.activeWorkflow.Name+" does not have a bound agent.")
		return diagnosis
	}

	if ctx.agent == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentMissing
		diagnosis.PrimaryReasonMessage = "Workflow agent is missing."
		diagnosis.NextActionHint = "Rebind the workflow to an existing agent."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentMissing, domain.PickupDiagnosisReasonSeverityError, "The workflow's bound agent record could not be found.")
		return diagnosis
	}

	switch ctx.agent.RuntimeControlState {
	case entagent.RuntimeControlStateInterruptRequested:
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentInterruptRequested
		diagnosis.PrimaryReasonMessage = "Agent interrupt has been requested."
		diagnosis.NextActionHint = "Wait for the active runtime to stop before retrying or reassigning work."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentInterruptRequested, domain.PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is being interrupted.")
		return diagnosis
	case entagent.RuntimeControlStatePauseRequested:
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentPauseRequested
		diagnosis.PrimaryReasonMessage = "Agent pause has been requested."
		diagnosis.NextActionHint = "Wait for the runtime to settle or resume the agent once it reaches paused."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentPauseRequested, domain.PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is transitioning toward a paused state.")
		return diagnosis
	case entagent.RuntimeControlStatePaused:
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentPaused
		diagnosis.PrimaryReasonMessage = "Agent is paused."
		diagnosis.NextActionHint = "Resume the agent to allow pickup."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentPaused, domain.PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is paused.")
		return diagnosis
	}

	if ctx.provider == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderMissing
		diagnosis.PrimaryReasonMessage = "Agent provider is missing."
		diagnosis.NextActionHint = "Reconnect the agent to a valid provider."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderMissing, domain.PickupDiagnosisReasonSeverityError, "The agent's provider record could not be found.")
		return diagnosis
	}

	if ctx.machine == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonMachineMissing
		diagnosis.PrimaryReasonMessage = "Provider machine is missing."
		diagnosis.NextActionHint = "Reconnect the provider to an existing machine."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonMachineMissing, domain.PickupDiagnosisReasonSeverityError, "Provider "+ctx.provider.Name+" is bound to a machine that could not be found.")
		return diagnosis
	}

	if ctx.machine.Status != entmachine.StatusOnline {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		if ctx.machine.Status == entmachine.StatusOffline {
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonMachineOffline
			diagnosis.PrimaryReasonMessage = "Provider machine is offline."
			diagnosis.NextActionHint = "Bring the machine back online before retrying pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonMachineOffline, domain.PickupDiagnosisReasonSeverityError, "Machine "+ctx.machine.Name+" is offline.")
		} else {
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderUnavailable
			diagnosis.PrimaryReasonMessage = "Provider machine is not available."
			diagnosis.NextActionHint = "Recover the machine before expecting pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderUnavailable, domain.PickupDiagnosisReasonSeverityError, "Machine "+ctx.machine.Name+" is "+strings.ToLower(string(ctx.machine.Status))+".")
		}
		return diagnosis
	}

	if ctx.providerState != nil {
		switch ctx.providerState.AvailabilityState {
		case catalogdomain.AgentProviderAvailabilityStateUnknown:
			diagnosis.State = domain.PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderUnknown
			diagnosis.PrimaryReasonMessage = "Provider availability is unknown."
			diagnosis.NextActionHint = "Refresh machine health so the provider can be probed again."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderUnknown, domain.PickupDiagnosisReasonSeverityWarning, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		case catalogdomain.AgentProviderAvailabilityStateStale:
			diagnosis.State = domain.PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderStale
			diagnosis.PrimaryReasonMessage = "Provider health information is stale."
			diagnosis.NextActionHint = "Refresh provider health to confirm availability."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderStale, domain.PickupDiagnosisReasonSeverityWarning, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		case catalogdomain.AgentProviderAvailabilityStateUnavailable:
			diagnosis.State = domain.PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderUnavailable
			diagnosis.PrimaryReasonMessage = "Provider is unavailable."
			diagnosis.NextActionHint = "Fix the provider health issue before expecting pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderUnavailable, domain.PickupDiagnosisReasonSeverityError, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		}
	}

	if ctx.activeWorkflow.MaxConcurrent > 0 && ctx.workflowActiveRuns >= ctx.activeWorkflow.MaxConcurrent {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonWorkflowConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Workflow concurrency is full."
		diagnosis.NextActionHint = "Wait for an active run in this workflow to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonWorkflowConcurrencyFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Workflow %s is using %d of %d allowed runs.", ctx.activeWorkflow.Name, ctx.workflowActiveRuns, ctx.activeWorkflow.MaxConcurrent))
		return diagnosis
	}

	if ctx.project.MaxConcurrentAgents > 0 && ctx.projectActiveRuns >= ctx.project.MaxConcurrentAgents {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProjectConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Project concurrency is full."
		diagnosis.NextActionHint = "Wait for another active run in this project to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProjectConcurrencyFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Project is using %d of %d allowed runs.", ctx.projectActiveRuns, ctx.project.MaxConcurrentAgents))
		return diagnosis
	}

	if ctx.provider.MaxParallelRuns > 0 && ctx.providerActiveRuns >= ctx.provider.MaxParallelRuns {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Provider concurrency is full."
		diagnosis.NextActionHint = "Wait for another run on this provider to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderConcurrencyFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Provider %s is using %d of %d allowed runs.", ctx.provider.Name, ctx.providerActiveRuns, ctx.provider.MaxParallelRuns))
		return diagnosis
	}

	if ctx.ticket.Edges.Status.MaxActiveRuns != nil && ctx.statusActiveRuns >= *ctx.ticket.Edges.Status.MaxActiveRuns {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonStatusCapacityFull
		diagnosis.PrimaryReasonMessage = "Status capacity is full."
		diagnosis.NextActionHint = "Wait for another active run in this status to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonStatusCapacityFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Status %s is using %d of %d allowed runs.", ctx.ticket.Edges.Status.Name, ctx.statusActiveRuns, *ctx.ticket.Edges.Status.MaxActiveRuns))
		return diagnosis
	}

	diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonReadyForPickup, domain.PickupDiagnosisReasonSeverityInfo, "The scheduler can claim this ticket on the next tick.")
	return diagnosis
}

func newPickupDiagnosis(
	state domain.PickupDiagnosisState,
	code domain.PickupDiagnosisReasonCode,
	message string,
	hint string,
	reasons []domain.PickupDiagnosisReason,
) domain.PickupDiagnosis {
	return domain.PickupDiagnosis{
		State:                state,
		PrimaryReasonCode:    code,
		PrimaryReasonMessage: message,
		NextActionHint:       hint,
		Reasons:              append([]domain.PickupDiagnosisReason{}, reasons...),
		BlockedBy:            []domain.PickupDiagnosisBlockedTicket{},
	}
}

func diagnosisSummary(
	ctx pickupDiagnosisBuildContext,
	code domain.PickupDiagnosisReasonCode,
	severity domain.PickupDiagnosisReasonSeverity,
	message string,
) []domain.PickupDiagnosisReason {
	reasons := []domain.PickupDiagnosisReason{{
		Code:     code,
		Message:  message,
		Severity: severity,
	}}

	if len(ctx.blockedBy) > 1 && code == domain.PickupDiagnosisReasonBlockedDependency {
		for _, blocker := range ctx.blockedBy {
			reasons = append(reasons, domain.PickupDiagnosisReason{
				Code:     domain.PickupDiagnosisReasonBlockedDependency,
				Message:  blocker.Identifier + " " + blocker.Title,
				Severity: domain.PickupDiagnosisReasonSeverityWarning,
			})
		}
	}

	return reasons
}

func blockedDependencyMessage(blockers []domain.PickupDiagnosisBlockedTicket) string {
	if len(blockers) == 0 {
		return "Blocking dependency is unresolved."
	}

	parts := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		parts = append(parts, blocker.Identifier+" "+blocker.Title)
	}
	return "Blocked by " + strings.Join(parts, ", ") + "."
}

func mapPickupDiagnosisWorkflow(activeWorkflow *ent.Workflow, matchingWorkflows []*ent.Workflow) *domain.PickupDiagnosisWorkflow {
	if activeWorkflow != nil {
		return &domain.PickupDiagnosisWorkflow{
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
	return &domain.PickupDiagnosisWorkflow{
		ID:                workflowItem.ID,
		Name:              workflowItem.Name,
		IsActive:          workflowItem.IsActive,
		PickupStatusMatch: true,
	}
}

func mapPickupDiagnosisAgent(agentItem *ent.Agent) *domain.PickupDiagnosisAgent {
	if agentItem == nil {
		return nil
	}
	return &domain.PickupDiagnosisAgent{
		ID:                  agentItem.ID,
		Name:                agentItem.Name,
		RuntimeControlState: catalogdomain.AgentRuntimeControlState(agentItem.RuntimeControlState),
	}
}

func mapPickupDiagnosisProvider(
	providerItem *ent.AgentProvider,
	machineItem *ent.Machine,
	providerState *catalogdomain.AgentProvider,
) *domain.PickupDiagnosisProvider {
	if providerItem == nil {
		return nil
	}

	response := &domain.PickupDiagnosisProvider{
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

func buildPickupDiagnosisCapacity(ctx pickupDiagnosisBuildContext) domain.PickupDiagnosisCapacity {
	capacity := domain.PickupDiagnosisCapacity{}
	if ctx.activeWorkflow != nil {
		capacity.Workflow = domain.PickupDiagnosisCapacityBucket{
			Limited:    ctx.activeWorkflow.MaxConcurrent > 0,
			ActiveRuns: ctx.workflowActiveRuns,
			Capacity:   ctx.activeWorkflow.MaxConcurrent,
		}
	}
	if ctx.project != nil {
		capacity.Project = domain.PickupDiagnosisCapacityBucket{
			Limited:    ctx.project.MaxConcurrentAgents > 0,
			ActiveRuns: ctx.projectActiveRuns,
			Capacity:   ctx.project.MaxConcurrentAgents,
		}
	}
	if ctx.provider != nil {
		capacity.Provider = domain.PickupDiagnosisCapacityBucket{
			Limited:    ctx.provider.MaxParallelRuns > 0,
			ActiveRuns: ctx.providerActiveRuns,
			Capacity:   ctx.provider.MaxParallelRuns,
		}
	}
	if ctx.ticket != nil && ctx.ticket.Edges.Status != nil {
		capacity.Status = domain.PickupDiagnosisStatusCapacity{
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
