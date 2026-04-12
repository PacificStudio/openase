package ticket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/google/uuid"
)

var ErrUnavailable = errors.New("ticket service unavailable")

type ticketHookSSHPool interface {
	Get(ctx context.Context, machine catalogdomain.Machine) (sshinfra.Client, error)
}

type ticketHookAgentPlatform interface {
	IssueToken(ctx context.Context, input agentplatform.IssueInput) (agentplatform.IssuedToken, error)
}

type ticketHookTransportResolver interface {
	Resolve(machine catalogdomain.Machine) (machinetransport.Transport, error)
}

type RunLifecycleHookInput struct {
	TicketID   uuid.UUID
	RunID      uuid.UUID
	HookName   infrahook.TicketHookName
	WorkflowID *uuid.UUID
	Blocking   bool
}

type Service struct {
	activityRepo    ActivityRepository
	queryRepo       QueryRepository
	commandRepo     CommandRepository
	linkRepo        LinkRepository
	commentRepo     CommentRepository
	usageRepo       UsageRepository
	runtimeRepo     RuntimeRepository
	logger          *slog.Logger
	sshPool         ticketHookSSHPool
	transport       ticketHookTransportResolver
	agentPlatform   ticketHookAgentPlatform
	activityEmitter *activitysvc.Emitter
	platformAPIURL  string
}

type loadedTicketHookRuntime struct {
	ticketID    uuid.UUID
	projectID   uuid.UUID
	identifier  string
	definitions []infrahook.Definition
	executor    infrahook.Executor
	env         infrahook.Env
}

func NewService(deps Dependencies) *Service {
	return &Service{
		activityRepo: deps.Activity,
		queryRepo:    deps.Query,
		commandRepo:  deps.Command,
		linkRepo:     deps.Link,
		commentRepo:  deps.Comment,
		usageRepo:    deps.Usage,
		runtimeRepo:  deps.Runtime,
		logger:       slog.Default().With("component", "ticket-service"),
	}
}

func (s *Service) ConfigureSSHPool(pool ticketHookSSHPool) {
	if s == nil {
		return
	}
	s.sshPool = pool
}

func (s *Service) ConfigureTransportResolver(resolver ticketHookTransportResolver) {
	if s == nil {
		return
	}
	s.transport = resolver
}

func (s *Service) ConfigurePlatformEnvironment(apiURL string, agentPlatform ticketHookAgentPlatform) {
	if s == nil {
		return
	}
	s.platformAPIURL = strings.TrimSpace(apiURL)
	s.agentPlatform = agentPlatform
}

func (s *Service) ConfigureActivityEmitter(emitter *activitysvc.Emitter) {
	if s == nil {
		return
	}
	s.activityEmitter = emitter
}

func (s *Service) RecordActivityEvent(ctx context.Context, input RecordActivityEventInput) (catalogdomain.ActivityEvent, error) {
	if s == nil || s.activityRepo == nil {
		return catalogdomain.ActivityEvent{}, ErrUnavailable
	}
	if input.ProjectID == uuid.Nil {
		return catalogdomain.ActivityEvent{}, fmt.Errorf("activity event project id must not be empty")
	}
	if _, err := activityevent.ParseRawType(input.EventType.String()); err != nil {
		return catalogdomain.ActivityEvent{}, err
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = timeNowUTC()
	} else {
		input.CreatedAt = input.CreatedAt.UTC()
	}
	input.Message = strings.TrimSpace(input.Message)
	input.Metadata = cloneAnyMap(input.Metadata)

	return s.activityRepo.RecordActivityEvent(ctx, input)
}

func (s *Service) List(ctx context.Context, input ListInput) ([]Ticket, error) {
	if s == nil || s.queryRepo == nil {
		return nil, ErrUnavailable
	}
	return s.queryRepo.List(ctx, input)
}

func (s *Service) ListArchived(ctx context.Context, input ArchivedListInput) (ArchivedListResult, error) {
	if s == nil || s.queryRepo == nil {
		return ArchivedListResult{}, ErrUnavailable
	}
	return s.queryRepo.ListArchived(ctx, input)
}

func (s *Service) Get(ctx context.Context, ticketID uuid.UUID) (Ticket, error) {
	if s == nil || s.queryRepo == nil {
		return Ticket{}, ErrUnavailable
	}
	return s.queryRepo.Get(ctx, ticketID)
}

func (s *Service) GetPickupDiagnosis(ctx context.Context, ticketID uuid.UUID) (PickupDiagnosis, error) {
	if s == nil || s.runtimeRepo == nil {
		return PickupDiagnosis{
			State:                PickupDiagnosisStateUnavailable,
			PrimaryReasonCode:    PickupDiagnosisReasonSchedulerUnavailable,
			PrimaryReasonMessage: "Scheduler state is unavailable.",
			NextActionHint:       "Retry once the ticket service is available again.",
			BlockedBy:            []PickupDiagnosisBlockedTicket{},
		}, ErrUnavailable
	}
	return s.runtimeRepo.GetPickupDiagnosis(ctx, ticketID)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Ticket, error) {
	if s == nil || s.commandRepo == nil {
		return Ticket{}, ErrUnavailable
	}
	return s.commandRepo.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Ticket, error) {
	if s == nil || s.commandRepo == nil {
		return Ticket{}, ErrUnavailable
	}
	result, err := s.commandRepo.Update(ctx, input)
	if err != nil {
		return Ticket{}, err
	}
	if result.DeferredHook != nil {
		hookName := infrahook.TicketHookName(result.DeferredHook.HookName)
		s.RunLifecycleHookBestEffort(ctx, RunLifecycleHookInput{
			TicketID:   result.Ticket.ID,
			RunID:      result.DeferredHook.RunID,
			HookName:   hookName,
			WorkflowID: result.DeferredHook.WorkflowID,
		})
	}
	return result.Ticket, nil
}

func (s *Service) ResumeRetry(ctx context.Context, input ResumeRetryInput) (Ticket, error) {
	if s == nil || s.commandRepo == nil {
		return Ticket{}, ErrUnavailable
	}
	return s.commandRepo.ResumeRetry(ctx, input)
}

func (s *Service) AddDependency(ctx context.Context, input AddDependencyInput) (Dependency, error) {
	if s == nil || s.linkRepo == nil {
		return Dependency{}, ErrUnavailable
	}
	return s.linkRepo.AddDependency(ctx, input)
}

func (s *Service) RemoveDependency(ctx context.Context, ticketID uuid.UUID, dependencyID uuid.UUID) (DeleteDependencyResult, error) {
	if s == nil || s.linkRepo == nil {
		return DeleteDependencyResult{}, ErrUnavailable
	}
	return s.linkRepo.RemoveDependency(ctx, ticketID, dependencyID)
}

func (s *Service) AddExternalLink(ctx context.Context, input AddExternalLinkInput) (ExternalLink, error) {
	if s == nil || s.linkRepo == nil {
		return ExternalLink{}, ErrUnavailable
	}
	return s.linkRepo.AddExternalLink(ctx, input)
}

func (s *Service) RemoveExternalLink(ctx context.Context, ticketID uuid.UUID, externalLinkID uuid.UUID) (DeleteExternalLinkResult, error) {
	if s == nil || s.linkRepo == nil {
		return DeleteExternalLinkResult{}, ErrUnavailable
	}
	return s.linkRepo.RemoveExternalLink(ctx, ticketID, externalLinkID)
}

func (s *Service) ListComments(ctx context.Context, ticketID uuid.UUID) ([]Comment, error) {
	if s == nil || s.commentRepo == nil {
		return nil, ErrUnavailable
	}
	return s.commentRepo.ListComments(ctx, ticketID)
}

func (s *Service) ListCommentRevisions(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) ([]CommentRevision, error) {
	if s == nil || s.commentRepo == nil {
		return nil, ErrUnavailable
	}
	return s.commentRepo.ListCommentRevisions(ctx, ticketID, commentID)
}

func (s *Service) AddComment(ctx context.Context, input AddCommentInput) (Comment, error) {
	if s == nil || s.commentRepo == nil {
		return Comment{}, ErrUnavailable
	}
	return s.commentRepo.AddComment(ctx, input)
}

func (s *Service) UpdateComment(ctx context.Context, input UpdateCommentInput) (Comment, error) {
	if s == nil || s.commentRepo == nil {
		return Comment{}, ErrUnavailable
	}
	return s.commentRepo.UpdateComment(ctx, input)
}

func (s *Service) RemoveComment(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) (DeleteCommentResult, error) {
	if s == nil || s.commentRepo == nil {
		return DeleteCommentResult{}, ErrUnavailable
	}
	return s.commentRepo.RemoveComment(ctx, ticketID, commentID)
}

func (s *Service) RunLifecycleHook(ctx context.Context, input RunLifecycleHookInput) error {
	if s == nil || s.runtimeRepo == nil {
		return ErrUnavailable
	}
	if input.TicketID == uuid.Nil {
		return fmt.Errorf("ticket hook ticket id must not be empty")
	}
	if input.RunID == uuid.Nil {
		return fmt.Errorf("ticket hook run id must not be empty")
	}

	runtime, err := s.loadHookRuntime(ctx, input)
	if err != nil {
		return err
	}
	if len(runtime.definitions) == 0 {
		return nil
	}

	results, err := runtime.executor.RunAll(ctx, input.HookName, runtime.definitions, runtime.env)
	s.logHookResults(input.HookName, runtime.ticketID, input.RunID, results, err)
	s.emitHookActivityEvents(ctx, runtime, input.HookName, results, err)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) RunLifecycleHookBestEffort(ctx context.Context, input RunLifecycleHookInput) {
	if err := s.RunLifecycleHook(ctx, input); err != nil {
		logger := slog.Default()
		if s != nil && s.logger != nil {
			logger = s.logger
		}
		logger.Warn(
			"ticket lifecycle hook failed",
			"hook_name", input.HookName,
			"ticket_id", input.TicketID,
			"run_id", input.RunID,
			"error", err,
		)
	}
}

func (s *Service) loadHookRuntime(ctx context.Context, input RunLifecycleHookInput) (loadedTicketHookRuntime, error) {
	if s == nil || s.runtimeRepo == nil {
		return loadedTicketHookRuntime{}, ErrUnavailable
	}

	data, err := s.runtimeRepo.LoadLifecycleHookRuntimeData(ctx, input.TicketID, input.RunID, input.WorkflowID)
	if err != nil {
		return loadedTicketHookRuntime{}, err
	}

	parsedHooks, err := infrahook.ParseTicketHooks(data.Hooks)
	if err != nil {
		return loadedTicketHookRuntime{}, fmt.Errorf("parse ticket hooks for lifecycle hook: %w", err)
	}
	definitions := selectTicketHookDefinitions(parsedHooks, input.HookName)
	if len(definitions) == 0 {
		return loadedTicketHookRuntime{ticketID: data.TicketID}, nil
	}
	if len(data.Workspaces) == 0 || strings.TrimSpace(data.WorkspaceRoot) == "" {
		return loadedTicketHookRuntime{}, fmt.Errorf("ticket hook workspace is unavailable for run %s", input.RunID)
	}

	repos := make([]infrahook.Repo, 0, len(data.Workspaces))
	for _, workspace := range data.Workspaces {
		repoName := strings.TrimSpace(workspace.RepoPath)
		if strings.TrimSpace(workspace.RepoName) != "" {
			repoName = strings.TrimSpace(workspace.RepoName)
		}
		repos = append(repos, infrahook.Repo{
			Name: repoName,
			Path: strings.TrimSpace(workspace.RepoPath),
		})
	}

	env := infrahook.Env{
		TicketID:         data.TicketID,
		ProjectID:        data.ProjectID,
		TicketIdentifier: data.TicketIdentifier,
		Workspace:        strings.TrimSpace(data.WorkspaceRoot),
		Repos:            repos,
		AgentName:        data.AgentName,
		WorkflowType:     data.WorkflowType,
		WorkflowFamily:   data.WorkflowFamily,
		Attempt:          data.Attempt,
		APIURL:           s.platformAPIURL,
	}
	if s.agentPlatform != nil {
		issued, issueErr := s.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
			AgentID:   data.AgentID,
			ProjectID: data.ProjectID,
			TicketID:  data.TicketID,
			ScopeWhitelist: agentplatform.ScopeWhitelist{
				Configured: len(data.PlatformAccessAllowed) > 0,
				Scopes:     append([]string(nil), data.PlatformAccessAllowed...),
			},
		})
		if issueErr != nil {
			return loadedTicketHookRuntime{}, fmt.Errorf("issue ticket hook agent token: %w", issueErr)
		}
		env.AgentToken = issued.Token
	}

	executor, err := s.ticketHookExecutor(data.Machine, data.Machine.Host != catalogdomain.LocalMachineHost)
	if err != nil {
		return loadedTicketHookRuntime{}, err
	}

	return loadedTicketHookRuntime{
		ticketID:    data.TicketID,
		projectID:   data.ProjectID,
		identifier:  data.TicketIdentifier,
		definitions: definitions,
		executor:    executor,
		env:         env,
	}, nil
}

func (s *Service) emitHookActivityEvents(
	ctx context.Context,
	runtime loadedTicketHookRuntime,
	hookName infrahook.TicketHookName,
	results []infrahook.Result,
	runErr error,
) {
	if s == nil || s.activityEmitter == nil || runtime.projectID == uuid.Nil || runtime.ticketID == uuid.Nil {
		return
	}

	for _, result := range results {
		eventType := activityevent.TypeHookPassed
		message := fmt.Sprintf("%s hook %s passed", runtime.identifier, hookName)
		if result.Outcome != infrahook.OutcomePass {
			eventType = activityevent.TypeHookFailed
			message = fmt.Sprintf("%s hook %s failed", runtime.identifier, hookName)
		}

		metadata := map[string]any{
			"ticket_identifier": runtime.identifier,
			"hook_name":         hookName,
			"command":           result.Command,
			"policy":            result.Policy,
			"outcome":           result.Outcome,
			"workdir":           result.WorkingDirectory,
		}
		if result.ExitCode != nil {
			metadata["exit_code"] = *result.ExitCode
		}
		if strings.TrimSpace(result.Error) != "" {
			metadata["error"] = result.Error
		}
		if strings.TrimSpace(result.Stderr) != "" {
			metadata["stderr"] = result.Stderr
		}
		if strings.TrimSpace(result.Stdout) != "" {
			metadata["stdout"] = result.Stdout
		}

		if _, emitErr := s.activityEmitter.Emit(ctx, activitysvc.RecordInput{
			ProjectID: runtime.projectID,
			TicketID:  &runtime.ticketID,
			EventType: eventType,
			Message:   message,
			Metadata:  metadata,
		}); emitErr != nil {
			s.logger.Warn("emit ticket hook activity", "hook_name", hookName, "ticket_id", runtime.ticketID, "error", emitErr)
		}
	}

	if runErr != nil && len(results) == 0 {
		if _, emitErr := s.activityEmitter.Emit(ctx, activitysvc.RecordInput{
			ProjectID: runtime.projectID,
			TicketID:  &runtime.ticketID,
			EventType: activityevent.TypeHookFailed,
			Message:   fmt.Sprintf("%s hook %s failed before command execution", runtime.identifier, hookName),
			Metadata: map[string]any{
				"ticket_identifier": runtime.identifier,
				"hook_name":         hookName,
				"error":             strings.TrimSpace(runErr.Error()),
			},
		}); emitErr != nil {
			s.logger.Warn("emit ticket hook preflight activity", "hook_name", hookName, "ticket_id", runtime.ticketID, "error", emitErr)
		}
	}
}

func (s *Service) ticketHookExecutor(machine catalogdomain.Machine, remote bool) (infrahook.Executor, error) {
	if !remote {
		return infrahook.NewShellExecutor(), nil
	}
	if s.transport != nil {
		transport, err := s.transport.Resolve(machine)
		if err != nil {
			return nil, err
		}
		return infrahook.NewRemoteShellExecutor(transport, machine), nil
	}
	if s.sshPool == nil {
		return nil, fmt.Errorf("ticket hook ssh pool unavailable for machine %s", machine.Name)
	}
	return infrahook.NewRemoteShellExecutor(sshSessionFactory{s.sshPool}, machine), nil
}

type sshSessionFactory struct {
	pool ticketHookSSHPool
}

func (f sshSessionFactory) OpenCommandSession(ctx context.Context, machine catalogdomain.Machine) (machinetransport.CommandSession, error) {
	if f.pool == nil {
		return nil, fmt.Errorf("ticket hook ssh pool unavailable for machine %s", machine.Name)
	}
	client, err := f.pool.Get(ctx, machine)
	if err != nil {
		return nil, err
	}
	return client.NewSession()
}

func (s *Service) logHookResults(
	hookName infrahook.TicketHookName,
	ticketID uuid.UUID,
	runID uuid.UUID,
	results []infrahook.Result,
	runErr error,
) {
	logger := slog.Default()
	if s != nil && s.logger != nil {
		logger = s.logger
	}

	for _, result := range results {
		attrs := []any{
			"hook_name", hookName,
			"ticket_id", ticketID,
			"run_id", runID,
			"command", result.Command,
			"policy", result.Policy,
			"outcome", result.Outcome,
			"duration", result.Duration,
			"workdir", result.WorkingDirectory,
		}
		if result.ExitCode != nil {
			attrs = append(attrs, "exit_code", *result.ExitCode)
		}
		if strings.TrimSpace(result.Stdout) != "" {
			attrs = append(attrs, "stdout", result.Stdout)
		}
		if strings.TrimSpace(result.Stderr) != "" {
			attrs = append(attrs, "stderr", result.Stderr)
		}
		if strings.TrimSpace(result.Error) != "" {
			attrs = append(attrs, "error", result.Error)
		}

		switch result.Outcome {
		case infrahook.OutcomePass:
			logger.Info("ticket lifecycle hook succeeded", attrs...)
		default:
			logger.Warn("ticket lifecycle hook finished with error", attrs...)
		}
	}
	if runErr != nil && len(results) == 0 {
		logger.Warn(
			"ticket lifecycle hook failed before command execution",
			"hook_name", hookName,
			"ticket_id", ticketID,
			"run_id", runID,
			"error", runErr,
		)
	}
}
