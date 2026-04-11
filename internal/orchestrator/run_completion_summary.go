package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagentstepevent "github.com/BetterAndBetterII/openase/ent/agentstepevent"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	runtimesecretenv "github.com/BetterAndBetterII/openase/internal/runtime/secretenv"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

var (
	runCompletionSummaryLongRunningThreshold = 2 * time.Minute
	runCompletionSummaryRiskyCommandHints    = []string{
		"rm -rf", "sudo ", "chmod 777", "chown ", "curl ", "wget ", "ssh ", "scp ",
		"git push --force", "git reset --hard", "git clean -fd", "docker system prune",
		"terraform destroy", "kubectl delete", "mv ", "cp ",
	}
	runCompletionSummaryRiskyPathHints = []string{
		".github/workflows", ".env", ".env.", "secrets", "auth", "credential", "token", "ssh", "deploy", "script",
	}
	ticketRunSummaryStreamTopic = provider.MustParseTopic("ticket.run.events")
	ticketRunSummaryStreamType  = provider.MustParseEventType("ticket.run.summary")
)

const (
	runCompletionSummaryTurnInputMaxBytes             = 1024 * 1024
	runCompletionSummaryTurnReservedBytes             = 64 * 1024
	runCompletionSummaryDeveloperInstructionsMaxBytes = 32 * 1024
	runCompletionSummaryUserPromptTargetBytes         = 256 * 1024
	runCompletionSummaryUserPromptMinimumBytes        = 8 * 1024
)

var runCompletionSummaryPromptStages = []runCompletionSummaryPromptCaps{
	{
		StringBytes:      400,
		Steps:            40,
		Commands:         80,
		ToolCalls:        40,
		Approvals:        20,
		OutputExcerpts:   8,
		Repos:            12,
		FilesPerRepo:     20,
		LongRunningSteps: 20,
		RepeatedCommands: 20,
		RepeatedFailures: 20,
		RiskyCommands:    20,
		RiskyFiles:       20,
		IncludeRepoFiles: true,
	},
	{
		StringBytes:      240,
		Steps:            24,
		Commands:         40,
		ToolCalls:        20,
		Approvals:        10,
		OutputExcerpts:   6,
		Repos:            8,
		FilesPerRepo:     10,
		LongRunningSteps: 12,
		RepeatedCommands: 12,
		RepeatedFailures: 12,
		RiskyCommands:    12,
		RiskyFiles:       12,
		IncludeRepoFiles: true,
	},
	{
		StringBytes:      160,
		Steps:            12,
		Commands:         16,
		ToolCalls:        10,
		Approvals:        6,
		OutputExcerpts:   6,
		Repos:            5,
		FilesPerRepo:     5,
		LongRunningSteps: 8,
		RepeatedCommands: 8,
		RepeatedFailures: 8,
		RiskyCommands:    8,
		RiskyFiles:       8,
		IncludeRepoFiles: true,
	},
	{
		StringBytes:      120,
		Steps:            8,
		Commands:         8,
		ToolCalls:        6,
		Approvals:        4,
		OutputExcerpts:   4,
		Repos:            3,
		FilesPerRepo:     0,
		LongRunningSteps: 5,
		RepeatedCommands: 5,
		RepeatedFailures: 5,
		RiskyCommands:    5,
		RiskyFiles:       5,
		IncludeRepoFiles: false,
	},
}

type runtimeCompletionSummaryCoordinator struct {
	client         *ent.Client
	logger         *slog.Logger
	events         provider.EventProvider
	adapters       *agentAdapterRegistry
	processManager provider.AgentCLIProcessManager
	sshPool        *sshinfra.Pool
	transports     *machinetransport.Resolver
	workflow       *workflowservice.Service
	secretResolver runtimesecretenv.Resolver
	secretManager  runtimeSecretManager
	now            func() time.Time
	timeout        time.Duration
	runs           *runtimeRunTracker
}

func newRuntimeCompletionSummaryCoordinator(
	client *ent.Client,
	logger *slog.Logger,
	events provider.EventProvider,
	adapters *agentAdapterRegistry,
	processManager provider.AgentCLIProcessManager,
	sshPool *sshinfra.Pool,
	workflow *workflowservice.Service,
	now func() time.Time,
	timeout time.Duration,
) *runtimeCompletionSummaryCoordinator {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &runtimeCompletionSummaryCoordinator{
		client:         client,
		logger:         logger.With("component", "runtime-completion-summary"),
		events:         events,
		adapters:       adapters,
		processManager: processManager,
		sshPool:        sshPool,
		transports:     machinetransport.NewResolver(processManager, sshPool),
		workflow:       workflow,
		now:            now,
		timeout:        timeout,
		runs:           newRuntimeRunTracker(),
	}
}

func (c *runtimeCompletionSummaryCoordinator) ConfigureSecretManager(manager runtimeSecretManager) {
	if c == nil {
		return
	}
	c.secretManager = manager
}

type runCompletionSummaryContext struct {
	run          *ent.AgentRun
	agent        *ent.Agent
	project      *ent.Project
	ticket       *ent.Ticket
	provider     *ent.AgentProvider
	machine      catalogdomain.Machine
	traceEntries []*ent.AgentTraceEvent
	stepEntries  []*ent.AgentStepEvent
	workspaces   []*ent.TicketRepoWorkspace
}

type runCompletionSummaryInputPayload struct {
	Metadata       map[string]any                 `json:"metadata"`
	Steps          []map[string]any               `json:"steps"`
	Commands       []map[string]any               `json:"commands"`
	ToolCalls      []map[string]any               `json:"tool_calls"`
	Approvals      []map[string]any               `json:"approvals"`
	OutputExcerpts []map[string]any               `json:"output_excerpts"`
	FileSnapshot   runCompletionWorkspaceSnapshot `json:"file_snapshot"`
	Heuristics     runCompletionSummaryHeuristics `json:"heuristics"`
}

type runCompletionSummaryHeuristics struct {
	LongRunningSteps []map[string]any `json:"long_running_steps"`
	RepeatedCommands []map[string]any `json:"repeated_commands"`
	RepeatedFailures []map[string]any `json:"repeated_failures"`
	RiskyCommands    []map[string]any `json:"risky_commands"`
	RiskyFiles       []map[string]any `json:"risky_files"`
}

type runCompletionSummaryPromptCaps struct {
	StringBytes      int
	Steps            int
	Commands         int
	ToolCalls        int
	Approvals        int
	OutputExcerpts   int
	Repos            int
	FilesPerRepo     int
	LongRunningSteps int
	RepeatedCommands int
	RepeatedFailures int
	RiskyCommands    int
	RiskyFiles       int
	IncludeRepoFiles bool
}

type runCompletionWorkspaceSnapshot struct {
	WorkspacePath string                  `json:"workspace_path"`
	Dirty         bool                    `json:"dirty"`
	ReposChanged  int                     `json:"repos_changed"`
	FilesChanged  int                     `json:"files_changed"`
	Added         int                     `json:"added"`
	Removed       int                     `json:"removed"`
	Repos         []runCompletionRepoDiff `json:"repos"`
}

type runCompletionRepoDiff struct {
	Name         string                  `json:"name"`
	Path         string                  `json:"path"`
	Branch       string                  `json:"branch"`
	Dirty        bool                    `json:"dirty"`
	FilesChanged int                     `json:"files_changed"`
	Added        int                     `json:"added"`
	Removed      int                     `json:"removed"`
	Files        []runCompletionFileDiff `json:"files"`
}

type runCompletionFileDiff struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

type runCompletionGitStatusEntry struct {
	code    string
	path    string
	oldPath string
}

type runCompletionGitNumstat struct {
	path    string
	added   int
	removed int
}

type ticketRunCompletionSummaryStreamPayload struct {
	ProjectID         string                                 `json:"project_id"`
	TicketID          string                                 `json:"ticket_id"`
	RunID             string                                 `json:"run_id"`
	CompletionSummary ticketRunCompletionSummaryStreamRecord `json:"completion_summary"`
}

type ticketRunCompletionSummaryStreamRecord struct {
	Status      string         `json:"status"`
	Markdown    *string        `json:"markdown,omitempty"`
	JSON        map[string]any `json:"json,omitempty"`
	GeneratedAt *string        `json:"generated_at,omitempty"`
	Error       *string        `json:"error,omitempty"`
}

func (c *runtimeCompletionSummaryCoordinator) reconcileRunCompletionSummaries(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}

	runs, err := c.client.AgentRun.Query().
		Where(
			entagentrun.StatusIn(
				entagentrun.StatusCompleted,
				entagentrun.StatusErrored,
				entagentrun.StatusInterrupted,
				entagentrun.StatusTerminated,
			),
		).
		Order(entagentrun.ByTerminalAt(sql.OrderAsc())).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list pending run completion summaries: %w", err)
	}

	for _, run := range runs {
		if run == nil {
			continue
		}
		switch {
		case run.CompletionSummaryStatus == nil:
			c.prepareRunCompletionSummaryBestEffort(ctx, run.ID)
			c.scheduleRunCompletionSummary(run.ID)
		case *run.CompletionSummaryStatus == entagentrun.CompletionSummaryStatusPending:
			c.scheduleRunCompletionSummary(run.ID)
		}
	}
	return nil
}

func (c *runtimeCompletionSummaryCoordinator) prepareRunCompletionSummaryBestEffort(ctx context.Context, runID uuid.UUID) {
	if c == nil || runID == uuid.Nil {
		return
	}

	if err := c.prepareRunCompletionSummary(ctx, runID); err != nil {
		c.logger.Warn("prepare run completion summary", "run_id", runID, "error", err)
		c.markRunCompletionSummaryFailed(context.Background(), runID, err)
	}
}

func (c *runtimeCompletionSummaryCoordinator) prepareRunCompletionSummary(ctx context.Context, runID uuid.UUID) error {
	summaryCtx, err := c.loadRunCompletionSummaryContext(ctx, runID)
	if err != nil {
		return err
	}

	snapshot, err := c.captureRunCompletionWorkspaceSnapshot(ctx, summaryCtx.machine, summaryCtx.workspaces)
	if err != nil {
		return err
	}

	input := buildRunCompletionSummaryInput(summaryCtx, snapshot)
	if _, err := c.client.AgentRun.UpdateOneID(runID).
		SetCompletionSummaryStatus(entagentrun.CompletionSummaryStatusPending).
		SetCompletionSummaryInput(input).
		SetCompletionSummaryJSON(map[string]any{}).
		ClearCompletionSummaryMarkdown().
		ClearCompletionSummaryGeneratedAt().
		ClearCompletionSummaryError().
		Save(ctx); err != nil {
		return fmt.Errorf("persist run completion summary input: %w", err)
	}

	pendingStatus := entagentrun.CompletionSummaryStatusPending
	summaryCtx.run.CompletionSummaryStatus = &pendingStatus
	summaryCtx.run.CompletionSummaryMarkdown = nil
	summaryCtx.run.CompletionSummaryGeneratedAt = nil
	summaryCtx.run.CompletionSummaryError = nil
	summaryCtx.run.CompletionSummaryJSON = map[string]any{}
	if err := c.publishRunCompletionSummaryEvent(ctx, summaryCtx.project.ID, summaryCtx.run); err != nil {
		c.logger.Warn("publish run completion summary pending", "run_id", runID, "error", err)
	}

	return nil
}

func (c *runtimeCompletionSummaryCoordinator) scheduleRunCompletionSummary(runID uuid.UUID) {
	if c == nil || runID == uuid.Nil {
		return
	}
	if !c.beginRunCompletionSummary(runID) {
		return
	}

	go func() {
		defer c.endRunCompletionSummary(runID)

		timeout := c.timeout
		if timeout <= 0 {
			timeout = defaultCompletionSummaryTimeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := c.generateRunCompletionSummary(ctx, runID); err != nil {
			c.logger.Warn("generate run completion summary", "run_id", runID, "error", err)
			c.markRunCompletionSummaryFailed(context.Background(), runID, err)
		}
	}()
}

func (c *runtimeCompletionSummaryCoordinator) beginRunCompletionSummary(runID uuid.UUID) bool {
	if c == nil {
		return false
	}
	return c.runs.begin(runID)
}

func (c *runtimeCompletionSummaryCoordinator) endRunCompletionSummary(runID uuid.UUID) {
	if c == nil {
		return
	}
	c.runs.finish(runID)
}

func (c *runtimeCompletionSummaryCoordinator) generateRunCompletionSummary(ctx context.Context, runID uuid.UUID) error {
	summaryCtx, err := c.loadRunCompletionSummaryContext(ctx, runID)
	if err != nil {
		return err
	}
	if summaryCtx.run.CompletionSummaryStatus == nil ||
		*summaryCtx.run.CompletionSummaryStatus != entagentrun.CompletionSummaryStatusPending {
		return nil
	}
	if len(summaryCtx.run.CompletionSummaryInput) == 0 {
		return fmt.Errorf("run %s is missing completion summary input", runID)
	}

	workingDirectory, err := c.resolveRunCompletionSummaryWorkingDirectory(summaryCtx.machine, summaryCtx.project.ID)
	if err != nil {
		return err
	}
	processManager, err := c.resolveRunCompletionSummaryProcessManager(summaryCtx.machine)
	if err != nil {
		return err
	}

	commandString := strings.TrimSpace(summaryCtx.provider.CliCommand)
	if summaryCtx.machine.AgentCLIPath != nil && strings.TrimSpace(*summaryCtx.machine.AgentCLIPath) != "" {
		commandString = strings.TrimSpace(*summaryCtx.machine.AgentCLIPath)
	}
	command, err := provider.ParseAgentCLICommand(commandString)
	if err != nil {
		return fmt.Errorf("parse summary provider command: %w", err)
	}

	environment, err := runtimesecretenv.AppendResolvedProviderSecrets(ctx, c.secretResolver, runtimesecretenv.ResolveInput{
		ProjectID:          summaryCtx.project.ID,
		ProviderAuthConfig: summaryCtx.provider.AuthConfig,
		BaseEnvironment:    buildAgentCLIEnvironment(summaryCtx.machine.EnvVars, summaryCtx.provider.AuthConfig),
		TicketID:           &summaryCtx.ticket.ID,
		WorkflowID:         optionalUUIDPointer(summaryCtx.run.WorkflowID),
		AgentID:            &summaryCtx.agent.ID,
	})
	if err != nil {
		return err
	}
	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		append([]string(nil), summaryCtx.provider.CliArgs...),
		&workingDirectory,
		environment,
	)
	if err != nil {
		return fmt.Errorf("build summary provider process spec: %w", err)
	}

	adapter, err := c.adapters.adapterFor(summaryCtx.provider.AdapterType)
	if err != nil {
		return err
	}

	developerInstructions := buildRunCompletionSummaryDeveloperInstructions(summaryCtx.project)
	session, err := adapter.Start(ctx, agentSessionStartSpec{
		Process:               processSpec,
		ProcessManager:        processManager,
		WorkingDirectory:      workingDirectory.String(),
		Model:                 summaryCtx.provider.ModelName,
		ReasoningEffort:       catalogdomain.ParseStoredAgentProviderReasoningEffort(summaryCtx.provider.ReasoningEffort),
		PermissionProfile:     catalogdomain.AgentProviderPermissionProfileStandard,
		DeveloperInstructions: developerInstructions,
		TurnTitle:             fmt.Sprintf("%s run summary", summaryCtx.ticket.Identifier),
	})
	if err != nil {
		return fmt.Errorf("start completion summary session: %w", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer stopCancel()
		_ = session.Stop(stopCtx)
	}()

	userPromptBudget := runCompletionSummaryUserPromptTargetBytes
	if available := runCompletionSummaryTurnInputMaxBytes - runCompletionSummaryTurnReservedBytes - len(developerInstructions); available < userPromptBudget {
		userPromptBudget = available
	}
	if userPromptBudget < runCompletionSummaryUserPromptMinimumBytes {
		userPromptBudget = runCompletionSummaryUserPromptMinimumBytes
	}

	userPrompt, err := buildRunCompletionSummaryUserPrompt(summaryCtx.run.CompletionSummaryInput, userPromptBudget)
	if err != nil {
		return err
	}
	if _, err := session.SendPrompt(ctx, userPrompt); err != nil {
		return fmt.Errorf("send completion summary prompt: %w", err)
	}

	markdown, err := collectRunCompletionSummaryMarkdown(ctx, session)
	if err != nil {
		return err
	}
	if markdown == "" {
		return fmt.Errorf("completion summary provider returned empty markdown")
	}

	result := map[string]any{
		"markdown":     markdown,
		"provider_id":  summaryCtx.provider.ID.String(),
		"provider":     summaryCtx.provider.Name,
		"adapter_type": string(summaryCtx.provider.AdapterType),
		"model":        summaryCtx.provider.ModelName,
	}
	generatedAt := c.now().UTC()
	if _, err := c.client.AgentRun.UpdateOneID(runID).
		SetCompletionSummaryStatus(entagentrun.CompletionSummaryStatusCompleted).
		SetCompletionSummaryMarkdown(markdown).
		SetCompletionSummaryJSON(result).
		SetCompletionSummaryGeneratedAt(generatedAt).
		ClearCompletionSummaryError().
		Save(ctx); err != nil {
		return fmt.Errorf("persist completion summary result: %w", err)
	}

	completedStatus := entagentrun.CompletionSummaryStatusCompleted
	summaryCtx.run.CompletionSummaryStatus = &completedStatus
	summaryCtx.run.CompletionSummaryMarkdown = &markdown
	summaryCtx.run.CompletionSummaryJSON = result
	summaryCtx.run.CompletionSummaryGeneratedAt = &generatedAt
	summaryCtx.run.CompletionSummaryError = nil
	if err := c.publishRunCompletionSummaryEvent(ctx, summaryCtx.project.ID, summaryCtx.run); err != nil {
		c.logger.Warn("publish run completion summary completed", "run_id", runID, "error", err)
	}

	return nil
}

func (c *runtimeCompletionSummaryCoordinator) markRunCompletionSummaryFailed(ctx context.Context, runID uuid.UUID, cause error) {
	if c == nil || c.client == nil || runID == uuid.Nil || cause == nil {
		return
	}
	message := strings.TrimSpace(cause.Error())
	if message == "" {
		message = "completion summary failed"
	}
	update := c.client.AgentRun.UpdateOneID(runID).
		SetCompletionSummaryStatus(entagentrun.CompletionSummaryStatusFailed).
		SetCompletionSummaryError(message).
		ClearCompletionSummaryMarkdown().
		ClearCompletionSummaryGeneratedAt()
	if _, err := update.Save(ctx); err != nil {
		c.logger.Warn("persist completion summary failure", "run_id", runID, "error", err)
		return
	}
	summaryCtx, err := c.loadRunCompletionSummaryContext(ctx, runID)
	if err != nil {
		c.logger.Warn("reload completion summary failure state", "run_id", runID, "error", err)
		return
	}
	if err := c.publishRunCompletionSummaryEvent(ctx, summaryCtx.project.ID, summaryCtx.run); err != nil {
		c.logger.Warn("publish run completion summary failed", "run_id", runID, "error", err)
	}
}

func (c *runtimeCompletionSummaryCoordinator) publishRunCompletionSummaryEvent(
	ctx context.Context,
	projectID uuid.UUID,
	run *ent.AgentRun,
) error {
	if c == nil || c.events == nil || projectID == uuid.Nil || run == nil {
		return nil
	}
	payload, ok := buildRunCompletionSummaryStreamPayload(projectID, run)
	if !ok {
		return nil
	}
	event, err := provider.NewJSONEvent(
		ticketRunSummaryStreamTopic,
		ticketRunSummaryStreamType,
		payload,
		c.now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("construct run completion summary event: %w", err)
	}
	if err := c.events.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish run completion summary event: %w", err)
	}
	return nil
}

func buildRunCompletionSummaryStreamPayload(
	projectID uuid.UUID,
	run *ent.AgentRun,
) (ticketRunCompletionSummaryStreamPayload, bool) {
	if projectID == uuid.Nil || run == nil || run.TicketID == uuid.Nil || run.ID == uuid.Nil || run.CompletionSummaryStatus == nil {
		return ticketRunCompletionSummaryStreamPayload{}, false
	}
	return ticketRunCompletionSummaryStreamPayload{
		ProjectID: projectID.String(),
		TicketID:  run.TicketID.String(),
		RunID:     run.ID.String(),
		CompletionSummary: ticketRunCompletionSummaryStreamRecord{
			Status:      run.CompletionSummaryStatus.String(),
			Markdown:    cloneSummaryStringPointer(run.CompletionSummaryMarkdown),
			JSON:        cloneSummaryAnyMap(run.CompletionSummaryJSON),
			GeneratedAt: cloneSummaryTimePointer(run.CompletionSummaryGeneratedAt),
			Error:       cloneSummaryStringPointer(run.CompletionSummaryError),
		},
	}, true
}

func cloneSummaryStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneSummaryTimePointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func cloneSummaryAnyMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func (c *runtimeCompletionSummaryCoordinator) loadRunCompletionSummaryContext(ctx context.Context, runID uuid.UUID) (runCompletionSummaryContext, error) {
	runItem, err := c.client.AgentRun.Query().
		Where(entagentrun.IDEQ(runID)).
		WithAgent(func(query *ent.AgentQuery) {
			query.WithProject(func(projectQuery *ent.ProjectQuery) {
				projectQuery.WithOrganization()
			})
		}).
		WithProvider().
		WithTicket().
		Only(ctx)
	if err != nil {
		return runCompletionSummaryContext{}, fmt.Errorf("load run %s for completion summary: %w", runID, err)
	}
	if runItem.Edges.Agent == nil || runItem.Edges.Agent.Edges.Project == nil {
		return runCompletionSummaryContext{}, fmt.Errorf("run %s is missing agent project context", runID)
	}
	if runItem.Edges.Provider == nil {
		return runCompletionSummaryContext{}, fmt.Errorf("run %s is missing provider context", runID)
	}
	if runItem.Edges.Ticket == nil {
		return runCompletionSummaryContext{}, fmt.Errorf("run %s is missing ticket context", runID)
	}

	machineItem, err := c.client.Machine.Query().
		Where(entmachine.IDEQ(runItem.Edges.Provider.MachineID)).
		Only(ctx)
	if err != nil {
		return runCompletionSummaryContext{}, fmt.Errorf("load machine for run %s summary: %w", runID, err)
	}

	traceEntries, err := c.client.AgentTraceEvent.Query().
		Where(entagenttraceevent.AgentRunIDEQ(runID)).
		Order(entagenttraceevent.BySequence(sql.OrderAsc()), entagenttraceevent.ByID(sql.OrderAsc())).
		All(ctx)
	if err != nil {
		return runCompletionSummaryContext{}, fmt.Errorf("list trace entries for run %s summary: %w", runID, err)
	}
	stepEntries, err := c.client.AgentStepEvent.Query().
		Where(entagentstepevent.AgentRunIDEQ(runID)).
		Order(entagentstepevent.ByCreatedAt(sql.OrderAsc()), entagentstepevent.ByID(sql.OrderAsc())).
		All(ctx)
	if err != nil {
		return runCompletionSummaryContext{}, fmt.Errorf("list step entries for run %s summary: %w", runID, err)
	}
	workspaces, err := c.client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runID)).
		Order(entticketrepoworkspace.ByRepoPath(sql.OrderAsc()), entticketrepoworkspace.ByID(sql.OrderAsc())).
		All(ctx)
	if err != nil {
		return runCompletionSummaryContext{}, fmt.Errorf("list workspaces for run %s summary: %w", runID, err)
	}

	return runCompletionSummaryContext{
		run:          runItem,
		agent:        runItem.Edges.Agent,
		project:      runItem.Edges.Agent.Edges.Project,
		ticket:       runItem.Edges.Ticket,
		provider:     runItem.Edges.Provider,
		machine:      mapRuntimeMachine(machineItem),
		traceEntries: traceEntries,
		stepEntries:  stepEntries,
		workspaces:   workspaces,
	}, nil
}

func (c *runtimeCompletionSummaryCoordinator) resolveRunCompletionSummaryWorkingDirectory(machine catalogdomain.Machine, projectID uuid.UUID) (provider.AbsolutePath, error) {
	if machine.WorkspaceRoot != nil && strings.TrimSpace(*machine.WorkspaceRoot) != "" {
		return provider.ParseAbsolutePath(strings.TrimSpace(*machine.WorkspaceRoot))
	}
	if machine.Host == catalogdomain.LocalMachineHost && c != nil && c.workflow != nil {
		root, err := c.workflow.ProjectControlRoot(projectID)
		if err != nil {
			return "", fmt.Errorf("resolve project control root for completion summary: %w", err)
		}
		if root := strings.TrimSpace(root); root != "" {
			return provider.ParseAbsolutePath(root)
		}
	}
	if machine.Host == catalogdomain.LocalMachineHost {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve summary working directory: %w", err)
		}
		return provider.ParseAbsolutePath(cwd)
	}
	return "", fmt.Errorf("machine %s is missing workspace_root for completion summary", machine.Name)
}

func (c *runtimeCompletionSummaryCoordinator) resolveRunCompletionSummaryProcessManager(machine catalogdomain.Machine) (provider.AgentCLIProcessManager, error) {
	if c == nil || c.transports == nil {
		return nil, fmt.Errorf("machine transport resolver unavailable for completion summary on machine %s", machine.Name)
	}
	transport, err := c.transports.Resolve(machine)
	if err != nil {
		return nil, err
	}
	return machinetransport.NewProcessManager(transport, machine), nil
}

func buildRunCompletionSummaryDeveloperInstructions(project *ent.Project) string {
	rawOverride := ""
	if project != nil {
		rawOverride = project.AgentRunSummaryPrompt
	}
	prompt, _ := catalogdomain.EffectiveAgentRunSummaryPrompt(rawOverride)

	instructions := strings.TrimSpace(`
You are OpenASE's post-run summarizer.
Use only the structured run facts that the platform provides.
Do not run commands, call tools, request approval, or ask for extra input.
Respond with Markdown only.
` + "\n\n" + prompt)

	if len(instructions) <= runCompletionSummaryDeveloperInstructionsMaxBytes {
		return instructions
	}
	const note = "\n\n[OpenASE note: project summary instructions were trimmed for length.]"
	limit := runCompletionSummaryDeveloperInstructionsMaxBytes - len(note)
	if limit <= 0 {
		return strings.TrimSpace(truncateUTF8Bytes(note, runCompletionSummaryDeveloperInstructionsMaxBytes))
	}
	truncated := strings.TrimSpace(truncateUTF8Bytes(instructions, limit))
	if truncated == "" {
		return strings.TrimSpace(note)
	}
	return truncated + note
}

func buildRunCompletionSummaryUserPrompt(input map[string]any, maxBytes int) (string, error) {
	parsed, err := parseRunCompletionSummaryInputPayload(input)
	if err != nil {
		return "", err
	}
	if maxBytes <= 0 {
		maxBytes = runCompletionSummaryUserPromptTargetBytes
	}
	if maxBytes < runCompletionSummaryUserPromptMinimumBytes {
		maxBytes = runCompletionSummaryUserPromptMinimumBytes
	}

	for _, stage := range runCompletionSummaryPromptStages {
		prompt, err := formatRunCompletionSummaryUserPrompt(buildRunCompletionSummaryPromptInput(parsed, stage))
		if err != nil {
			return "", err
		}
		if len(prompt) <= maxBytes {
			return prompt, nil
		}
	}

	prompt, err := formatRunCompletionSummaryUserPrompt(buildMinimalRunCompletionSummaryPromptInput(parsed))
	if err != nil {
		return "", err
	}
	if len(prompt) > maxBytes {
		return "", fmt.Errorf("compact completion summary prompt still exceeds %d bytes", maxBytes)
	}
	return prompt, nil
}

func parseRunCompletionSummaryInputPayload(input map[string]any) (runCompletionSummaryInputPayload, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return runCompletionSummaryInputPayload{}, fmt.Errorf("marshal completion summary input: %w", err)
	}

	var parsed runCompletionSummaryInputPayload
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return runCompletionSummaryInputPayload{}, fmt.Errorf("parse completion summary input: %w", err)
	}
	return parsed, nil
}

func formatRunCompletionSummaryUserPrompt(promptInput map[string]any) (string, error) {
	payload, err := json.MarshalIndent(promptInput, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal completion summary prompt input: %w", err)
	}

	return strings.TrimSpace(`
Summarize this ticket agent run from structured facts only.
Return Markdown only.

Structured run facts:
` + "```json\n" + string(payload) + "\n```"), nil
}

func buildRunCompletionSummaryPromptInput(
	input runCompletionSummaryInputPayload,
	caps runCompletionSummaryPromptCaps,
) map[string]any {
	steps, omittedSteps := compactRunCompletionSummaryMaps(input.Steps, caps.Steps, caps.StringBytes)
	commands, omittedCommands := compactRunCompletionSummaryMaps(input.Commands, caps.Commands, caps.StringBytes)
	toolCalls, omittedToolCalls := compactRunCompletionSummaryMaps(input.ToolCalls, caps.ToolCalls, caps.StringBytes)
	approvals, omittedApprovals := compactRunCompletionSummaryMaps(input.Approvals, caps.Approvals, caps.StringBytes)
	outputExcerpts, omittedOutputExcerpts := compactRunCompletionSummaryMaps(input.OutputExcerpts, caps.OutputExcerpts, caps.StringBytes)

	fileSnapshot, omittedRepos, omittedFiles := compactRunCompletionSummarySnapshot(input.FileSnapshot, caps)
	heuristics, heuristicOmitted := compactRunCompletionSummaryHeuristics(input.Heuristics, caps)

	omittedCounts := map[string]any{}
	addRunCompletionSummaryOmittedCount(omittedCounts, "steps", omittedSteps)
	addRunCompletionSummaryOmittedCount(omittedCounts, "commands", omittedCommands)
	addRunCompletionSummaryOmittedCount(omittedCounts, "tool_calls", omittedToolCalls)
	addRunCompletionSummaryOmittedCount(omittedCounts, "approvals", omittedApprovals)
	addRunCompletionSummaryOmittedCount(omittedCounts, "output_excerpts", omittedOutputExcerpts)
	addRunCompletionSummaryOmittedCount(omittedCounts, "repos", omittedRepos)
	addRunCompletionSummaryOmittedCount(omittedCounts, "files", omittedFiles)
	for key, count := range heuristicOmitted {
		addRunCompletionSummaryOmittedCount(omittedCounts, key, count)
	}

	promptContext := map[string]any{
		"compacted_for_size": true,
		"selection_strategy": "Representative head/tail items plus heuristic highlights when sections are too large.",
		"input_overview": map[string]any{
			"steps_total":           len(input.Steps),
			"commands_total":        len(input.Commands),
			"tool_calls_total":      len(input.ToolCalls),
			"approvals_total":       len(input.Approvals),
			"output_excerpts_total": len(input.OutputExcerpts),
			"repos_total":           len(input.FileSnapshot.Repos),
			"files_changed_total":   input.FileSnapshot.FilesChanged,
		},
	}
	if len(omittedCounts) > 0 {
		promptContext["omitted_counts"] = omittedCounts
	}

	return map[string]any{
		"prompt_context":  promptContext,
		"metadata":        compactRunCompletionSummaryItem(input.Metadata, caps.StringBytes),
		"steps":           steps,
		"commands":        commands,
		"tool_calls":      toolCalls,
		"approvals":       approvals,
		"output_excerpts": outputExcerpts,
		"file_snapshot":   fileSnapshot,
		"heuristics":      heuristics,
	}
}

func buildMinimalRunCompletionSummaryPromptInput(input runCompletionSummaryInputPayload) map[string]any {
	caps := runCompletionSummaryPromptCaps{
		StringBytes:      96,
		OutputExcerpts:   3,
		LongRunningSteps: 3,
		RepeatedCommands: 3,
		RepeatedFailures: 3,
		RiskyCommands:    3,
		RiskyFiles:       3,
	}
	heuristics, _ := compactRunCompletionSummaryHeuristics(input.Heuristics, caps)
	outputExcerpts, omittedOutputExcerpts := compactRunCompletionSummaryMaps(input.OutputExcerpts, caps.OutputExcerpts, caps.StringBytes)
	metadata := compactRunCompletionSummaryItem(input.Metadata, caps.StringBytes)

	promptContext := map[string]any{
		"compacted_for_size": true,
		"selection_strategy": "Minimal fallback preserves run identity, outcome signals, excerpts, and heuristics.",
		"input_overview": map[string]any{
			"steps_total":           len(input.Steps),
			"commands_total":        len(input.Commands),
			"tool_calls_total":      len(input.ToolCalls),
			"approvals_total":       len(input.Approvals),
			"output_excerpts_total": len(input.OutputExcerpts),
			"repos_total":           len(input.FileSnapshot.Repos),
			"files_changed_total":   input.FileSnapshot.FilesChanged,
		},
	}
	if omittedOutputExcerpts > 0 {
		promptContext["omitted_counts"] = map[string]any{
			"output_excerpts": omittedOutputExcerpts,
		}
	}

	return map[string]any{
		"prompt_context":  promptContext,
		"metadata":        metadata,
		"output_excerpts": outputExcerpts,
		"heuristics":      heuristics,
		"file_snapshot": map[string]any{
			"workspace_path": compactRunCompletionSummaryString(input.FileSnapshot.WorkspacePath, caps.StringBytes),
			"dirty":          input.FileSnapshot.Dirty,
			"repos_changed":  input.FileSnapshot.ReposChanged,
			"files_changed":  input.FileSnapshot.FilesChanged,
			"added":          input.FileSnapshot.Added,
			"removed":        input.FileSnapshot.Removed,
		},
	}
}

func compactRunCompletionSummaryHeuristics(
	heuristics runCompletionSummaryHeuristics,
	caps runCompletionSummaryPromptCaps,
) (map[string]any, map[string]int) {
	longRunningSteps, omittedLongRunningSteps := compactRunCompletionSummaryMaps(heuristics.LongRunningSteps, caps.LongRunningSteps, caps.StringBytes)
	repeatedCommands, omittedRepeatedCommands := compactRunCompletionSummaryMaps(heuristics.RepeatedCommands, caps.RepeatedCommands, caps.StringBytes)
	repeatedFailures, omittedRepeatedFailures := compactRunCompletionSummaryMaps(heuristics.RepeatedFailures, caps.RepeatedFailures, caps.StringBytes)
	riskyCommands, omittedRiskyCommands := compactRunCompletionSummaryMaps(heuristics.RiskyCommands, caps.RiskyCommands, caps.StringBytes)
	riskyFiles, omittedRiskyFiles := compactRunCompletionSummaryMaps(heuristics.RiskyFiles, caps.RiskyFiles, caps.StringBytes)

	return map[string]any{
			"long_running_steps": longRunningSteps,
			"repeated_commands":  repeatedCommands,
			"repeated_failures":  repeatedFailures,
			"risky_commands":     riskyCommands,
			"risky_files":        riskyFiles,
		}, map[string]int{
			"long_running_steps": omittedLongRunningSteps,
			"repeated_commands":  omittedRepeatedCommands,
			"repeated_failures":  omittedRepeatedFailures,
			"risky_commands":     omittedRiskyCommands,
			"risky_files":        omittedRiskyFiles,
		}
}

func compactRunCompletionSummaryMaps(
	items []map[string]any,
	maxItems int,
	stringBytes int,
) ([]map[string]any, int) {
	selected, omitted := selectRunCompletionSummaryMaps(items, maxItems)
	compacted := make([]map[string]any, 0, len(selected))
	for _, item := range selected {
		compacted = append(compacted, compactRunCompletionSummaryItem(item, stringBytes))
	}
	return compacted, omitted
}

func selectRunCompletionSummaryMaps(items []map[string]any, maxItems int) ([]map[string]any, int) {
	if len(items) == 0 {
		return []map[string]any{}, 0
	}
	if maxItems <= 0 {
		return []map[string]any{}, len(items)
	}
	if len(items) <= maxItems {
		return append([]map[string]any(nil), items...), 0
	}

	headCount := maxItems / 2
	tailCount := maxItems - headCount
	if headCount == 0 {
		headCount = 1
		tailCount = maxItems - 1
	}
	if tailCount == 0 {
		tailCount = 1
		headCount = maxItems - 1
	}

	selected := make([]map[string]any, 0, maxItems)
	selected = append(selected, items[:headCount]...)
	startTail := len(items) - tailCount
	if startTail < headCount {
		startTail = headCount
	}
	selected = append(selected, items[startTail:]...)
	return selected, len(items) - len(selected)
}

func compactRunCompletionSummaryItem(item map[string]any, stringBytes int) map[string]any {
	if len(item) == 0 {
		return map[string]any{}
	}

	compacted := make(map[string]any, len(item))
	for key, value := range item {
		switch key {
		case "arguments", "payload":
			if excerpt := compactRunCompletionSummaryJSONExcerpt(value, stringBytes*2); excerpt != "" {
				compacted[key+"_excerpt"] = excerpt
			}
		default:
			compactedValue, ok := compactRunCompletionSummaryValue(value, stringBytes)
			if ok {
				compacted[key] = compactedValue
			}
		}
	}
	return compacted
}

func compactRunCompletionSummaryValue(value any, stringBytes int) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, false
	case string:
		trimmed := compactRunCompletionSummaryString(typed, stringBytes)
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	case bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return typed, true
	default:
		excerpt := compactRunCompletionSummaryJSONExcerpt(typed, stringBytes*2)
		if excerpt == "" {
			return "", false
		}
		return excerpt, true
	}
}

func compactRunCompletionSummaryJSONExcerpt(value any, maxBytes int) string {
	if maxBytes <= 0 || value == nil {
		return ""
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return compactRunCompletionSummaryString(string(payload), maxBytes)
}

func compactRunCompletionSummaryString(raw string, maxBytes int) string {
	trimmed := strings.TrimSpace(strings.ToValidUTF8(raw, ""))
	if trimmed == "" {
		return ""
	}
	if maxBytes <= 0 || len(trimmed) <= maxBytes {
		return trimmed
	}
	const ellipsis = "..."
	limit := maxBytes - len(ellipsis)
	if limit <= 0 {
		return ellipsis
	}
	truncated := strings.TrimSpace(truncateUTF8Bytes(trimmed, limit))
	if truncated == "" {
		return ellipsis
	}
	return truncated + ellipsis
}

func compactRunCompletionSummarySnapshot(
	snapshot runCompletionWorkspaceSnapshot,
	caps runCompletionSummaryPromptCaps,
) (map[string]any, int, int) {
	repos, omittedRepos := selectRunCompletionSummaryRepos(snapshot.Repos, caps.Repos)
	compactedRepos := make([]map[string]any, 0, len(repos))
	omittedFiles := 0

	for _, repo := range repos {
		repoMap := map[string]any{
			"name":          compactRunCompletionSummaryString(repo.Name, caps.StringBytes),
			"path":          compactRunCompletionSummaryString(repo.Path, caps.StringBytes),
			"branch":        compactRunCompletionSummaryString(repo.Branch, caps.StringBytes),
			"dirty":         repo.Dirty,
			"files_changed": repo.FilesChanged,
			"added":         repo.Added,
			"removed":       repo.Removed,
		}
		if caps.IncludeRepoFiles {
			files, repoOmittedFiles := compactRunCompletionSummaryFiles(repo.Files, caps.FilesPerRepo, caps.StringBytes)
			repoMap["files"] = files
			omittedFiles += repoOmittedFiles
		} else {
			omittedFiles += len(repo.Files)
		}
		compactedRepos = append(compactedRepos, repoMap)
	}

	if omittedRepos > 0 {
		for _, repo := range snapshot.Repos[len(snapshot.Repos)-omittedRepos:] {
			omittedFiles += len(repo.Files)
		}
	}

	return map[string]any{
		"workspace_path": compactRunCompletionSummaryString(snapshot.WorkspacePath, caps.StringBytes),
		"dirty":          snapshot.Dirty,
		"repos_changed":  snapshot.ReposChanged,
		"files_changed":  snapshot.FilesChanged,
		"added":          snapshot.Added,
		"removed":        snapshot.Removed,
		"repos":          compactedRepos,
	}, omittedRepos, omittedFiles
}

func selectRunCompletionSummaryRepos(repos []runCompletionRepoDiff, maxRepos int) ([]runCompletionRepoDiff, int) {
	if len(repos) == 0 {
		return []runCompletionRepoDiff{}, 0
	}
	if maxRepos <= 0 {
		return []runCompletionRepoDiff{}, len(repos)
	}
	if len(repos) <= maxRepos {
		return append([]runCompletionRepoDiff(nil), repos...), 0
	}

	selected := append([]runCompletionRepoDiff(nil), repos[:maxRepos]...)
	return selected, len(repos) - len(selected)
}

func compactRunCompletionSummaryFiles(
	files []runCompletionFileDiff,
	maxFiles int,
	stringBytes int,
) ([]map[string]any, int) {
	totalFiles := len(files)
	if totalFiles == 0 {
		return []map[string]any{}, 0
	}
	if maxFiles <= 0 {
		return []map[string]any{}, totalFiles
	}
	if totalFiles > maxFiles {
		files = append([]runCompletionFileDiff(nil), files[:maxFiles]...)
	}

	compacted := make([]map[string]any, 0, len(files))
	for _, file := range files {
		compacted = append(compacted, map[string]any{
			"path":    compactRunCompletionSummaryString(file.Path, stringBytes),
			"status":  compactRunCompletionSummaryString(file.Status, stringBytes),
			"added":   file.Added,
			"removed": file.Removed,
		})
	}
	return compacted, totalFiles - len(compacted)
}

func addRunCompletionSummaryOmittedCount(counts map[string]any, key string, count int) {
	if count <= 0 {
		return
	}
	counts[key] = count
}

func collectRunCompletionSummaryMarkdown(ctx context.Context, session agentSession) (string, error) {
	if session == nil {
		return "", fmt.Errorf("summary session is unavailable")
	}

	var assistantSnapshot string
	var assistantDelta strings.Builder

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case event, ok := <-session.Events():
			if !ok {
				if err := session.Err(); err != nil {
					return "", err
				}
				if trimmed := strings.TrimSpace(assistantSnapshot); trimmed != "" {
					return trimmed, nil
				}
				if trimmed := strings.TrimSpace(assistantDelta.String()); trimmed != "" {
					return trimmed, nil
				}
				return "", fmt.Errorf("summary session ended without assistant output")
			}

			switch event.Type {
			case agentEventTypeToolCallRequested:
				return "", fmt.Errorf("summary session attempted tool use")
			case agentEventTypeApprovalRequested, agentEventTypeUserInputRequested:
				return "", fmt.Errorf("summary session requested approval or user input")
			case agentEventTypeTurnFailed:
				if event.Turn != nil && event.Turn.Error != nil && strings.TrimSpace(event.Turn.Error.Message) != "" {
					return "", errors.New(strings.TrimSpace(event.Turn.Error.Message))
				}
				return "", fmt.Errorf("summary turn failed")
			case agentEventTypeOutputProduced:
				if event.Output == nil || strings.TrimSpace(event.Output.Stream) != "assistant" {
					continue
				}
				if event.Output.Snapshot {
					assistantSnapshot = event.Output.Text
					continue
				}
				assistantDelta.WriteString(event.Output.Text)
			case agentEventTypeTurnCompleted:
				if trimmed := strings.TrimSpace(assistantSnapshot); trimmed != "" {
					return trimmed, nil
				}
				if trimmed := strings.TrimSpace(assistantDelta.String()); trimmed != "" {
					return trimmed, nil
				}
				return "", fmt.Errorf("summary turn completed without assistant output")
			}
		}
	}
}

func buildRunCompletionSummaryInput(
	summaryCtx runCompletionSummaryContext,
	snapshot runCompletionWorkspaceSnapshot,
) map[string]any {
	metadata := map[string]any{
		"run_id":              summaryCtx.run.ID.String(),
		"ticket_id":           summaryCtx.ticket.ID.String(),
		"ticket_identifier":   summaryCtx.ticket.Identifier,
		"ticket_title":        summaryCtx.ticket.Title,
		"agent_id":            summaryCtx.agent.ID.String(),
		"agent_name":          summaryCtx.agent.Name,
		"provider_id":         summaryCtx.provider.ID.String(),
		"provider_name":       summaryCtx.provider.Name,
		"provider_adapter":    string(summaryCtx.provider.AdapterType),
		"provider_model":      summaryCtx.provider.ModelName,
		"run_status":          string(summaryCtx.run.Status),
		"created_at":          summaryCtx.run.CreatedAt.UTC().Format(time.RFC3339),
		"runtime_started_at":  timePointerString(summaryCtx.run.RuntimeStartedAt),
		"terminal_at":         timePointerString(summaryCtx.run.TerminalAt),
		"duration_seconds":    runDurationSeconds(summaryCtx.run.RuntimeStartedAt, summaryCtx.run.TerminalAt),
		"input_tokens":        summaryCtx.run.InputTokens,
		"output_tokens":       summaryCtx.run.OutputTokens,
		"cached_input_tokens": summaryCtx.run.CachedInputTokens,
		"reasoning_tokens":    summaryCtx.run.ReasoningTokens,
		"total_tokens":        summaryCtx.run.TotalTokens,
	}

	steps := buildRunCompletionSummarySteps(summaryCtx.stepEntries, summaryCtx.run.TerminalAt)
	commands, repeatedCommands, riskyCommands := buildRunCompletionSummaryCommands(summaryCtx.traceEntries)
	toolCalls := buildRunCompletionSummaryToolCalls(summaryCtx.traceEntries)
	approvals := buildRunCompletionSummaryApprovals(summaryCtx.traceEntries)
	outputExcerpts, repeatedFailures := buildRunCompletionSummaryOutputExcerpts(summaryCtx.traceEntries)
	riskyFiles := buildRunCompletionSummaryRiskyFiles(snapshot)
	longRunningSteps := buildRunCompletionSummaryLongRunningSteps(steps)

	return map[string]any{
		"metadata":        metadata,
		"steps":           steps,
		"commands":        commands,
		"tool_calls":      toolCalls,
		"approvals":       approvals,
		"output_excerpts": outputExcerpts,
		"file_snapshot":   snapshot,
		"heuristics": map[string]any{
			"long_running_steps": longRunningSteps,
			"repeated_commands":  repeatedCommands,
			"repeated_failures":  repeatedFailures,
			"risky_commands":     riskyCommands,
			"risky_files":        riskyFiles,
		},
	}
}

func buildRunCompletionSummarySteps(stepEntries []*ent.AgentStepEvent, terminalAt *time.Time) []map[string]any {
	steps := make([]map[string]any, 0, len(stepEntries))
	for index, entry := range stepEntries {
		step := map[string]any{
			"step_status": entry.StepStatus,
			"summary":     strings.TrimSpace(entry.Summary),
			"started_at":  entry.CreatedAt.UTC().Format(time.RFC3339),
		}
		var nextAt *time.Time
		if index+1 < len(stepEntries) {
			nextAt = &stepEntries[index+1].CreatedAt
		} else {
			nextAt = terminalAt
		}
		durationSeconds := runDurationSeconds(&entry.CreatedAt, nextAt)
		if durationSeconds > 0 {
			step["duration_seconds"] = durationSeconds
		}
		steps = append(steps, step)
	}
	return steps
}

func buildRunCompletionSummaryLongRunningSteps(steps []map[string]any) []map[string]any {
	items := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		durationSeconds, ok := step["duration_seconds"].(int64)
		if !ok || time.Duration(durationSeconds)*time.Second < runCompletionSummaryLongRunningThreshold {
			continue
		}
		items = append(items, map[string]any{
			"step_status":      step["step_status"],
			"summary":          step["summary"],
			"duration_seconds": durationSeconds,
		})
	}
	return items
}

func buildRunCompletionSummaryCommands(
	traceEntries []*ent.AgentTraceEvent,
) ([]map[string]any, []map[string]any, []map[string]any) {
	commands := make([]map[string]any, 0)
	counts := make(map[string]int)
	risky := make([]map[string]any, 0)

	for _, entry := range traceEntries {
		if entry.Kind != catalogdomain.AgentTraceKindCommandDelta && entry.Kind != catalogdomain.AgentTraceKindCommandSnapshot {
			continue
		}
		command := strings.TrimSpace(stringMapValue(entry.Payload, "command"))
		if command == "" {
			command = strings.TrimSpace(entry.Text)
		}
		if command == "" {
			continue
		}

		counts[command]++
		commands = append(commands, map[string]any{
			"command":    command,
			"kind":       entry.Kind,
			"created_at": entry.CreatedAt.UTC().Format(time.RFC3339),
		})
		if isRunCompletionRiskyCommand(command) {
			risky = append(risky, map[string]any{
				"command": command,
				"reason":  "matches risky command heuristic",
			})
		}
	}

	repeated := make([]map[string]any, 0)
	for command, count := range counts {
		if count < 2 {
			continue
		}
		repeated = append(repeated, map[string]any{
			"command": command,
			"count":   count,
		})
	}
	sort.Slice(repeated, func(i, j int) bool {
		if repeated[i]["count"].(int) == repeated[j]["count"].(int) {
			return repeated[i]["command"].(string) < repeated[j]["command"].(string)
		}
		return repeated[i]["count"].(int) > repeated[j]["count"].(int)
	})
	return commands, repeated, risky
}

func buildRunCompletionSummaryToolCalls(traceEntries []*ent.AgentTraceEvent) []map[string]any {
	items := make([]map[string]any, 0, len(traceEntries))
	for _, entry := range traceEntries {
		if entry.Kind != catalogdomain.AgentTraceKindToolCallStarted {
			continue
		}
		item := map[string]any{
			"tool":       firstNonEmpty(strings.TrimSpace(entry.Text), stringMapValue(entry.Payload, "tool")),
			"created_at": entry.CreatedAt.UTC().Format(time.RFC3339),
		}
		if callID := stringMapValue(entry.Payload, "call_id"); callID != "" {
			item["call_id"] = callID
		}
		if arguments, ok := entry.Payload["arguments"]; ok {
			item["arguments"] = arguments
		}
		items = append(items, item)
	}
	return items
}

func buildRunCompletionSummaryApprovals(traceEntries []*ent.AgentTraceEvent) []map[string]any {
	items := make([]map[string]any, 0, len(traceEntries))
	for _, entry := range traceEntries {
		switch entry.Kind {
		case catalogdomain.AgentTraceKindApprovalRequested, catalogdomain.AgentTraceKindUserInputRequested:
		default:
			continue
		}
		item := map[string]any{
			"kind":       entry.Kind,
			"summary":    strings.TrimSpace(entry.Text),
			"created_at": entry.CreatedAt.UTC().Format(time.RFC3339),
		}
		if len(entry.Payload) > 0 {
			item["payload"] = cloneResourceMap(entry.Payload)
		}
		items = append(items, item)
	}
	return items
}

func buildRunCompletionSummaryOutputExcerpts(
	traceEntries []*ent.AgentTraceEvent,
) ([]map[string]any, []map[string]any) {
	excerpts := make([]map[string]any, 0, 6)
	failureCounts := make(map[string]int)
	lastAssistant := ""

	for _, entry := range traceEntries {
		text := strings.TrimSpace(entry.Text)
		if text == "" {
			continue
		}
		if entry.Kind == catalogdomain.AgentTraceKindAssistantDelta || entry.Kind == catalogdomain.AgentTraceKindAssistantSnapshot {
			lastAssistant = text
		}
		if !looksLikeRunCompletionExcerpt(entry.Kind, text) {
			continue
		}
		excerpt := trimRunCompletionExcerpt(text, 280)
		excerpts = append(excerpts, map[string]any{
			"kind":       entry.Kind,
			"stream":     entry.Stream,
			"text":       excerpt,
			"created_at": entry.CreatedAt.UTC().Format(time.RFC3339),
		})
		if isRunCompletionFailureText(excerpt) {
			failureCounts[excerpt]++
		}
		if len(excerpts) == 6 {
			break
		}
	}
	if lastAssistant != "" {
		excerpts = append(excerpts, map[string]any{
			"kind":   "assistant_conclusion",
			"text":   trimRunCompletionExcerpt(lastAssistant, 400),
			"stream": "assistant",
		})
	}

	repeatedFailures := make([]map[string]any, 0)
	for text, count := range failureCounts {
		if count < 2 {
			continue
		}
		repeatedFailures = append(repeatedFailures, map[string]any{
			"text":  text,
			"count": count,
		})
	}
	sort.Slice(repeatedFailures, func(i, j int) bool {
		return repeatedFailures[i]["count"].(int) > repeatedFailures[j]["count"].(int)
	})
	return excerpts, repeatedFailures
}

func buildRunCompletionSummaryRiskyFiles(snapshot runCompletionWorkspaceSnapshot) []map[string]any {
	items := make([]map[string]any, 0)
	for _, repo := range snapshot.Repos {
		for _, file := range repo.Files {
			if !isRunCompletionRiskyPath(file.Path) {
				continue
			}
			items = append(items, map[string]any{
				"repo":    repo.Name,
				"path":    file.Path,
				"status":  file.Status,
				"added":   file.Added,
				"removed": file.Removed,
			})
		}
	}
	return items
}

func (c *runtimeCompletionSummaryCoordinator) captureRunCompletionWorkspaceSnapshot(
	ctx context.Context,
	machine catalogdomain.Machine,
	workspaces []*ent.TicketRepoWorkspace,
) (runCompletionWorkspaceSnapshot, error) {
	if len(workspaces) == 0 {
		return runCompletionWorkspaceSnapshot{Repos: []runCompletionRepoDiff{}}, nil
	}

	workspaceRoot := strings.TrimSpace(workspaces[0].WorkspaceRoot)
	snapshot := runCompletionWorkspaceSnapshot{
		WorkspacePath: workspaceRoot,
		Repos:         make([]runCompletionRepoDiff, 0, len(workspaces)),
	}

	for _, workspace := range workspaces {
		repoSummary, err := c.captureRunCompletionWorkspaceRepo(ctx, machine, workspaceRoot, workspace)
		if err != nil {
			return runCompletionWorkspaceSnapshot{}, err
		}
		if !repoSummary.Dirty {
			continue
		}
		snapshot.Dirty = true
		snapshot.ReposChanged++
		snapshot.FilesChanged += repoSummary.FilesChanged
		snapshot.Added += repoSummary.Added
		snapshot.Removed += repoSummary.Removed
		snapshot.Repos = append(snapshot.Repos, repoSummary)
	}

	sort.Slice(snapshot.Repos, func(i, j int) bool {
		return snapshot.Repos[i].Path < snapshot.Repos[j].Path
	})
	return snapshot, nil
}

func (c *runtimeCompletionSummaryCoordinator) captureRunCompletionWorkspaceRepo(
	ctx context.Context,
	machine catalogdomain.Machine,
	workspaceRoot string,
	workspace *ent.TicketRepoWorkspace,
) (runCompletionRepoDiff, error) {
	branch, err := workspaceinfra.ReadWorkspaceGitBranch(ctx, workspace.RepoPath, func(
		ctx context.Context,
		args []string,
		allowExitCodeOne bool,
	) ([]byte, error) {
		return c.runCompletionSummaryGitCommand(ctx, machine, args, allowExitCodeOne)
	})
	if err != nil {
		if errors.Is(err, workspaceinfra.ErrGitWorkspaceUnavailable) {
			return runCompletionRepoDiff{}, nil
		}
		return runCompletionRepoDiff{}, fmt.Errorf("read workspace branch for %s: %w", workspace.RepoPath, err)
	}
	statusOutput, err := c.runCompletionSummaryGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", workspace.RepoPath, "status", "--porcelain=v1", "-z", "--untracked-files=all"},
		false,
	)
	if err != nil {
		return runCompletionRepoDiff{}, fmt.Errorf("read workspace status for %s: %w", workspace.RepoPath, err)
	}
	statuses, err := parseRunCompletionGitStatusEntries(statusOutput)
	if err != nil {
		return runCompletionRepoDiff{}, fmt.Errorf("parse workspace status for %s: %w", workspace.RepoPath, err)
	}
	if len(statuses) == 0 {
		return runCompletionRepoDiff{}, nil
	}

	numstatOutput, err := workspaceinfra.ReadWorkspaceGitNumstat(ctx, workspace.RepoPath, func(
		ctx context.Context,
		args []string,
		allowExitCodeOne bool,
	) ([]byte, error) {
		return c.runCompletionSummaryGitCommand(ctx, machine, args, allowExitCodeOne)
	})
	if err != nil {
		return runCompletionRepoDiff{}, fmt.Errorf("read workspace diff stats for %s: %w", workspace.RepoPath, err)
	}
	numstats, err := parseRunCompletionGitNumstat(numstatOutput)
	if err != nil {
		return runCompletionRepoDiff{}, fmt.Errorf("parse workspace diff stats for %s: %w", workspace.RepoPath, err)
	}
	fileStats := make(map[string]runCompletionGitNumstat, len(numstats))
	for _, item := range numstats {
		fileStats[item.path] = item
	}

	relativeRepoPath := strings.TrimSpace(workspace.RepoPath)
	if workspaceRoot != "" {
		if rel, relErr := filepath.Rel(workspaceRoot, workspace.RepoPath); relErr == nil {
			relativeRepoPath = filepath.ToSlash(rel)
		}
	}

	files := make([]runCompletionFileDiff, 0, len(statuses))
	repoSummary := runCompletionRepoDiff{
		Name:   filepath.Base(workspace.RepoPath),
		Path:   relativeRepoPath,
		Branch: branch,
		Dirty:  true,
	}
	for _, status := range statuses {
		stat := fileStats[status.path]
		if status.code == "??" {
			stat, err = c.readRunCompletionUntrackedNumstat(ctx, machine, workspace.RepoPath, status.path)
			if err != nil {
				return runCompletionRepoDiff{}, fmt.Errorf("read untracked numstat for %s: %w", status.path, err)
			}
		}
		file := runCompletionFileDiff{
			Path:    status.path,
			Status:  mapRunCompletionWorkspaceFileStatus(status.code),
			Added:   stat.added,
			Removed: stat.removed,
		}
		files = append(files, file)
		repoSummary.Added += file.Added
		repoSummary.Removed += file.Removed
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	repoSummary.Files = files
	repoSummary.FilesChanged = len(files)
	return repoSummary, nil
}

func (c *runtimeCompletionSummaryCoordinator) readRunCompletionUntrackedNumstat(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	filePath string,
) (runCompletionGitNumstat, error) {
	output, err := c.runCompletionSummaryGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "diff", "--no-index", "--numstat", "-z", "/dev/null", filePath},
		true,
	)
	if err != nil {
		return runCompletionGitNumstat{}, err
	}
	stats, err := parseRunCompletionGitNumstat(output)
	if err != nil {
		return runCompletionGitNumstat{}, err
	}
	if len(stats) == 0 {
		return runCompletionGitNumstat{path: filePath}, nil
	}
	return runCompletionGitNumstat{
		path:    filePath,
		added:   stats[0].added,
		removed: stats[0].removed,
	}, nil
}

func (c *runtimeCompletionSummaryCoordinator) runCompletionSummaryGitCommand(
	ctx context.Context,
	machine catalogdomain.Machine,
	args []string,
	allowExitCodeOne bool,
) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("git command is empty")
	}
	if machine.Host == catalogdomain.LocalMachineHost {
		command := exec.CommandContext(ctx, args[0], args[1:]...) // #nosec G204
		output, err := command.CombinedOutput()
		if err != nil && (!allowExitCodeOne || !runCompletionSummaryCommandExitedWithCode(err, 1)) {
			return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return output, nil
	}
	if c == nil || c.transports == nil {
		return nil, fmt.Errorf("machine transport resolver unavailable for machine %s", machine.Name)
	}
	resolved, err := c.transports.ResolveRuntime(machine)
	if err != nil {
		return nil, err
	}
	commandSessionExecutor := resolved.CommandSessionExecutor()
	if commandSessionExecutor == nil {
		return nil, fmt.Errorf("%w: remote command session unavailable for machine %s", machinetransport.ErrTransportUnavailable, machine.Name)
	}
	session, err := commandSessionExecutor.OpenCommandSession(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("open remote command session for run completion summary: %w", err)
	}
	defer func() { _ = session.Close() }()

	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, sshinfra.ShellQuote(arg))
	}
	output, err := session.CombinedOutput("sh -lc " + sshinfra.ShellQuote(strings.Join(quoted, " ")))
	if err != nil && (!allowExitCodeOne || !strings.Contains(err.Error(), "exit status 1")) {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func parseRunCompletionGitStatusEntries(raw []byte) ([]runCompletionGitStatusEntry, error) {
	parts := bytes.Split(raw, []byte{0})
	entries := make([]runCompletionGitStatusEntry, 0, len(parts))
	for index := 0; index < len(parts); index++ {
		entry := parts[index]
		if len(entry) == 0 {
			continue
		}
		if len(entry) < 4 {
			return nil, fmt.Errorf("status entry %d is truncated", index)
		}
		status := string(entry[:2])
		path := string(entry[3:])
		item := runCompletionGitStatusEntry{
			code: status,
			path: filepath.ToSlash(path),
		}
		if strings.Contains(status, "R") || strings.Contains(status, "C") {
			index++
			if index >= len(parts) || len(parts[index]) == 0 {
				return nil, fmt.Errorf("status entry %q is missing original path", status)
			}
			item.oldPath = filepath.ToSlash(string(parts[index]))
		}
		entries = append(entries, item)
	}
	return entries, nil
}

func parseRunCompletionGitNumstat(raw []byte) ([]runCompletionGitNumstat, error) {
	parts := bytes.Split(raw, []byte{0})
	stats := make([]runCompletionGitNumstat, 0, len(parts))
	for index := 0; index < len(parts); index++ {
		entry := parts[index]
		if len(entry) == 0 {
			continue
		}
		fields := strings.SplitN(string(entry), "\t", 3)
		if len(fields) != 3 {
			return nil, fmt.Errorf("numstat entry %d is malformed", index)
		}
		added, err := parseRunCompletionGitNumstatCount(fields[0])
		if err != nil {
			return nil, err
		}
		removed, err := parseRunCompletionGitNumstatCount(fields[1])
		if err != nil {
			return nil, err
		}

		path := fields[2]
		if path == "" {
			if index+2 >= len(parts) {
				return nil, fmt.Errorf("rename numstat entry %d is truncated", index)
			}
			path = string(parts[index+2])
			index += 2
		}
		stats = append(stats, runCompletionGitNumstat{
			path:    filepath.ToSlash(path),
			added:   added,
			removed: removed,
		})
	}
	return stats, nil
}

func parseRunCompletionGitNumstatCount(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "-" {
		return 0, nil
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse numstat count %q: %w", raw, err)
	}
	return count, nil
}

func mapRunCompletionWorkspaceFileStatus(code string) string {
	switch {
	case code == "??":
		return "untracked"
	case strings.Contains(code, "R") || strings.Contains(code, "C"):
		return "renamed"
	case strings.Contains(code, "D"):
		return "deleted"
	case strings.Contains(code, "A"):
		return "added"
	default:
		return "modified"
	}
}

func runCompletionSummaryCommandExitedWithCode(err error, code int) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == code
}

func runDurationSeconds(start *time.Time, end *time.Time) int64 {
	if start == nil || end == nil {
		return 0
	}
	duration := end.UTC().Sub(start.UTC())
	if duration <= 0 {
		return 0
	}
	return int64(duration / time.Second)
}

func timePointerString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func optionalUUIDPointer(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	copied := value
	return &copied
}

func stringMapValue(raw map[string]any, key string) string {
	if len(raw) == 0 {
		return ""
	}
	value, ok := raw[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func looksLikeRunCompletionExcerpt(kind string, text string) bool {
	if isRunCompletionFailureText(text) {
		return true
	}
	switch kind {
	case catalogdomain.AgentTraceKindApprovalRequested,
		catalogdomain.AgentTraceKindUserInputRequested,
		catalogdomain.AgentTraceKindTurnDiffUpdated,
		catalogdomain.AgentTraceKindReasoningUpdated:
		return true
	default:
		return false
	}
}

func isRunCompletionFailureText(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(normalized, "error") ||
		strings.Contains(normalized, "failed") ||
		strings.Contains(normalized, "panic") ||
		strings.Contains(normalized, "exception")
}

func trimRunCompletionExcerpt(text string, maxLen int) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= maxLen || maxLen <= 3 {
		return trimmed
	}
	return trimmed[:maxLen-3] + "..."
}

func isRunCompletionRiskyCommand(command string) bool {
	normalized := strings.ToLower(strings.TrimSpace(command))
	for _, hint := range runCompletionSummaryRiskyCommandHints {
		if strings.Contains(normalized, hint) {
			return true
		}
	}
	return false
}

func isRunCompletionRiskyPath(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	for _, hint := range runCompletionSummaryRiskyPathHints {
		if strings.Contains(normalized, hint) {
			return true
		}
	}
	return false
}
