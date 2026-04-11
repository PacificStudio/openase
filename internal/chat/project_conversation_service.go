package chat

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	claudecodeadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

var (
	ErrConversationNotFound         = chatrepo.ErrNotFound
	ErrConversationConflict         = chatrepo.ErrConflict
	ErrConversationTurnActive       = chatrepo.ErrTurnAlreadyActive
	ErrConversationTurnNotActive    = fmt.Errorf("%w: project conversation does not have an active turn", domain.ErrConflict)
	ErrConversationInterruptPending = domain.ErrInterruptPending
	ErrPendingInterruptNotFound     = chatrepo.ErrNotFound
	ErrConversationRuntimeAbsent    = fmt.Errorf("chat conversation runtime is unavailable")
)

type projectConversationCatalog interface {
	ListOrganizations(ctx context.Context) ([]catalogdomain.Organization, error)
	ListProjects(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.Project, error)
	GetProject(ctx context.Context, id uuid.UUID) (catalogdomain.Project, error)
	GetMachine(ctx context.Context, id uuid.UUID) (catalogdomain.Machine, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (catalogdomain.AgentProvider, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.AgentProvider, error)
	ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]catalogdomain.ProjectRepo, error)
	ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]catalogdomain.TicketRepoScope, error)
	ListActivityEvents(ctx context.Context, input catalogdomain.ListActivityEvents) (catalogdomain.ActivityEventPage, error)
}

type projectConversationSkillSync interface {
	RefreshSkills(ctx context.Context, input workflowservice.RefreshSkillsInput) (workflowservice.RefreshSkillsResult, error)
}

type projectConversationAgentPlatform interface {
	IssueToken(ctx context.Context, input agentplatform.IssueInput) (agentplatform.IssuedToken, error)
}

type projectConversationSecretManager interface {
	ResolveBoundForRuntime(context.Context, secretsservice.ResolveBoundRuntimeInput) ([]secretsdomain.ResolvedSecret, error)
}

type projectConversationActivityEmitter interface {
	Emit(context.Context, activitysvc.RecordInput) (*catalogdomain.ActivityEvent, error)
}

type liveProjectConversation struct {
	principal domain.ProjectConversationPrincipal
	provider  catalogdomain.AgentProvider
	machine   catalogdomain.Machine
	runtime   Runtime
	codex     projectConversationCodexRuntime
	interrupt projectConversationInterruptRuntime
	turnStop  projectConversationTurnStopRuntime
	workspace provider.AbsolutePath
}

type projectConversationUsageHighWater struct {
	inputTokens         int64
	outputTokens        int64
	cachedInputTokens   int64
	cacheCreationTokens int64
	reasoningTokens     int64
	promptTokens        int64
	candidateTokens     int64
	toolTokens          int64
	totalTokens         int64
	costAmount          float64
	hasCostAmount       bool
}

type projectConversationCodexRuntime interface {
	Runtime
	EnsureSession(ctx context.Context, input RuntimeTurnInput) error
	RespondInterrupt(ctx context.Context, input RuntimeInterruptResponseInput) (TurnStream, error)
	InterruptTurn(ctx context.Context, sessionID SessionID) (RuntimeSessionAnchor, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationInterruptRuntime interface {
	Runtime
	RespondInterrupt(ctx context.Context, input RuntimeInterruptResponseInput) (TurnStream, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationTurnStopRuntime interface {
	Runtime
	InterruptTurn(ctx context.Context, sessionID SessionID) (RuntimeSessionAnchor, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationSessionAnchorer interface {
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type ProjectConversationService struct {
	logger *slog.Logger

	conversations projectConversationConversationStore
	entries       projectConversationEntryStore
	interrupts    projectConversationInterruptStore
	runtimeStore  projectConversationRuntimeStore
	catalog       projectConversationCatalog
	tickets       ticketReader
	workflows     workflowReader
	skillSync     projectConversationSkillSync

	localProcessManager provider.AgentCLIProcessManager
	sshPool             *sshinfra.Pool
	platformAPIURL      string
	agentPlatform       projectConversationAgentPlatform
	githubAuth          githubauthservice.TokenResolver
	secretResolver      RuntimeEnvironmentResolver
	secretManager       projectConversationSecretManager
	activityEmitter     projectConversationActivityEmitter

	streamBroker    *projectConversationStreamBroker
	muxBroker       *projectConversationMuxBroker
	runtimeManager  *projectConversationRuntimeManager
	turnLocks       userLockRegistry
	promptBuilder   *Service
	newCodexRuntime func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error)
}

func NewProjectConversationService(
	logger *slog.Logger,
	stores projectConversationStoreSource,
	catalog projectConversationCatalog,
	tickets ticketReader,
	workflows workflowReader,
	localProcessManager provider.AgentCLIProcessManager,
	sshPool *sshinfra.Pool,
) *ProjectConversationService {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	service := &ProjectConversationService{
		logger:              logger.With("component", "project-conversation-service"),
		conversations:       stores,
		entries:             stores,
		interrupts:          stores,
		runtimeStore:        stores,
		catalog:             catalog,
		tickets:             tickets,
		workflows:           workflows,
		localProcessManager: localProcessManager,
		sshPool:             sshPool,
		streamBroker:        newProjectConversationStreamBroker(),
		muxBroker:           newProjectConversationMuxBroker(),
	}
	if syncer, ok := workflows.(projectConversationSkillSync); ok {
		service.skillSync = syncer
	}
	service.promptBuilder = &Service{
		logger:    service.logger,
		catalog:   catalog,
		tickets:   tickets,
		workflows: workflows,
	}
	service.newCodexRuntime = func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		adapter, err := newCodexAdapterForManager(manager)
		if err != nil {
			return nil, err
		}
		runtime := NewCodexRuntime(adapter)
		runtime.ConfigureSecretResolver(service.secretResolver)
		return runtime, nil
	}
	service.runtimeManager = newProjectConversationRuntimeManager(
		service.logger,
		catalog,
		service.runtimeStore,
		localProcessManager,
		sshPool,
		service.newCodexRuntime,
	)
	service.runtimeManager.ConfigureSkillSync(service.skillSync)
	return service
}

func (s *ProjectConversationService) ConfigurePlatformEnvironment(
	apiURL string,
	platform projectConversationAgentPlatform,
) {
	if s == nil {
		return
	}
	s.platformAPIURL = strings.TrimSpace(apiURL)
	s.agentPlatform = platform
}

func (s *ProjectConversationService) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	if s == nil {
		return
	}
	s.githubAuth = resolver
	if s.runtimeManager != nil {
		s.runtimeManager.ConfigureGitHubCredentials(resolver)
	}
}

func (s *ProjectConversationService) ConfigureSecretResolver(resolver RuntimeEnvironmentResolver) {
	if s == nil {
		return
	}
	s.secretResolver = resolver
	if s.runtimeManager != nil {
		s.runtimeManager.ConfigureSecretResolver(resolver)
	}
}

func (s *ProjectConversationService) ConfigureSecretManager(manager projectConversationSecretManager) {
	if s == nil {
		return
	}
	s.secretManager = manager
}

func (s *ProjectConversationService) ConfigureActivityEmitter(emitter projectConversationActivityEmitter) {
	if s == nil {
		return
	}
	s.activityEmitter = emitter
}

func projectConversationTurnLockKey(conversation domain.Conversation) UserID {
	return UserID("conversation:" + conversation.ID.String())
}

func isStableLocalProjectConversationUser(userID UserID) bool {
	return strings.TrimSpace(userID.String()) == strings.TrimSpace(LocalProjectConversationUserID.String())
}

func (s *ProjectConversationService) normalizeConversationUser(
	ctx context.Context,
	conversation domain.Conversation,
	userID UserID,
) (domain.Conversation, error) {
	if !isStableLocalProjectConversationUser(userID) || conversation.UserID == userID.String() {
		return conversation, nil
	}
	return s.conversations.UpdateConversationUser(ctx, conversation.ID, userID.String())
}

func (s *ProjectConversationService) CreateConversation(
	ctx context.Context,
	userID UserID,
	projectID uuid.UUID,
	providerID uuid.UUID,
) (domain.Conversation, error) {
	project, err := s.catalog.GetProject(ctx, projectID)
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("get project for chat conversation: %w", err)
	}
	providerItem, err := s.catalog.GetAgentProvider(ctx, providerID)
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("get provider for chat conversation: %w", err)
	}
	if providerItem.OrganizationID != project.OrganizationID {
		return domain.Conversation{}, fmt.Errorf("%w: provider is outside the project organization", ErrConversationConflict)
	}

	return s.conversations.CreateConversation(ctx, domain.CreateConversation{
		ProjectID:  projectID,
		UserID:     userID.String(),
		Source:     domain.SourceProjectSidebar,
		ProviderID: providerID,
	})
}

func (s *ProjectConversationService) ListConversations(
	ctx context.Context,
	userID UserID,
	projectID uuid.UUID,
	providerID *uuid.UUID,
) ([]domain.Conversation, error) {
	source := domain.SourceProjectSidebar
	filter := domain.ListConversationsFilter{
		ProjectID:  projectID,
		UserID:     userID.String(),
		Source:     &source,
		ProviderID: providerID,
	}
	if isStableLocalProjectConversationUser(userID) {
		filter.UserID = ""
	}
	conversations, err := s.conversations.ListConversations(ctx, filter)
	if err != nil {
		return nil, err
	}
	if !isStableLocalProjectConversationUser(userID) {
		return conversations, nil
	}
	for index, conversation := range conversations {
		normalized, normalizeErr := s.normalizeConversationUser(ctx, conversation, userID)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		conversations[index] = normalized
	}
	return conversations, nil
}

func (s *ProjectConversationService) GetConversation(ctx context.Context, userID UserID, conversationID uuid.UUID) (domain.Conversation, error) {
	conversation, err := s.conversations.GetConversation(ctx, conversationID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if conversation.UserID != userID.String() && !isStableLocalProjectConversationUser(userID) {
		return domain.Conversation{}, ErrConversationNotFound
	}
	return s.normalizeConversationUser(ctx, conversation, userID)
}

func (s *ProjectConversationService) GetPrincipal(ctx context.Context, userID UserID, conversationID uuid.UUID) (domain.ProjectConversationPrincipal, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.ProjectConversationPrincipal{}, err
	}
	return s.runtimeStore.EnsurePrincipal(ctx, domain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      conversation.ProjectID,
		ProviderID:     conversation.ProviderID,
		Name:           projectConversationPrincipalName(conversation.ID),
	})
}

func (s *ProjectConversationService) ListEntries(ctx context.Context, userID UserID, conversationID uuid.UUID) ([]domain.Entry, error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return s.entries.ListEntries(ctx, conversationID)
}

func (s *ProjectConversationService) WatchConversation(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (<-chan StreamEvent, func(), error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, nil, err
	}
	live, hasLive := s.runtimeManager.Get(conversationID)

	state := s.watchConversationRuntimeState(ctx, conversationID, live, hasLive)
	sessionPayload := map[string]any{
		"conversation_id": conversationID.String(),
		"runtime_state":   state,
	}
	var sessionProvider *catalogdomain.AgentProvider
	if live != nil {
		sessionProvider = &live.provider
	}
	if sessionProvider == nil && s.catalog != nil {
		if providerItem, providerErr := s.catalog.GetAgentProvider(ctx, conversation.ProviderID); providerErr == nil {
			sessionProvider = &providerItem
		}
	}
	mergeConversationSessionPayload(sessionPayload, conversation, sessionProvider)
	if hasLive && live != nil {
		anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
		mergeConversationSessionPayload(sessionPayload, domain.Conversation{
			ID:                        conversationID,
			ProviderThreadID:          optionalString(anchor.ProviderThreadID),
			LastTurnID:                optionalString(anchor.LastTurnID),
			ProviderThreadStatus:      optionalString(anchor.ProviderThreadStatus),
			ProviderThreadActiveFlags: append([]string(nil), anchor.ProviderThreadActiveFlags...),
		}, sessionProvider)
	}
	events, cleanup := s.streamBroker.Watch(
		conversationID,
		StreamEvent{Event: "session", Payload: sessionPayload},
	)
	return events, cleanup, nil
}

func (s *ProjectConversationService) WatchProjectConversations(
	ctx context.Context,
	userID UserID,
	projectID uuid.UUID,
) (<-chan ProjectConversationMuxEvent, func(), error) {
	conversations, err := s.ListConversations(ctx, userID, projectID, nil)
	if err != nil {
		return nil, nil, err
	}

	providersByID := map[uuid.UUID]catalogdomain.AgentProvider{}
	initial := make([]ProjectConversationMuxEvent, 0, len(conversations))
	for _, conversation := range conversations {
		live, hasLive := s.runtimeManager.Get(conversation.ID)
		state := s.watchConversationRuntimeState(ctx, conversation.ID, live, hasLive)
		var providerItem *catalogdomain.AgentProvider
		if hasLive && live != nil {
			providerItem = &live.provider
			anchor := liveRuntimeSessionAnchor(live, SessionID(conversation.ID.String()))
			conversation.ProviderThreadID = optionalString(anchor.ProviderThreadID)
			conversation.LastTurnID = optionalString(anchor.LastTurnID)
			conversation.ProviderThreadStatus = optionalString(anchor.ProviderThreadStatus)
			conversation.ProviderThreadActiveFlags = append(
				[]string(nil),
				anchor.ProviderThreadActiveFlags...,
			)
		}
		if providerItem == nil && s.catalog != nil && conversation.ProviderID != uuid.Nil {
			if cached, ok := providersByID[conversation.ProviderID]; ok {
				providerItem = &cached
			} else if resolved, resolveErr := s.catalog.GetAgentProvider(ctx, conversation.ProviderID); resolveErr == nil {
				providersByID[conversation.ProviderID] = resolved
				providerItem = &resolved
			}
		}

		initial = append(initial, newProjectConversationMuxEvent(conversation, StreamEvent{
			Event:   "session",
			Payload: conversationSessionPayload(conversation.ID, state, conversation, providerItem),
		}))
	}

	key := projectConversationMuxWatchKey{ProjectID: projectID, UserID: userID}
	events, cleanup := s.muxBroker.Watch(key, initial)
	return events, cleanup, nil
}

func (s *ProjectConversationService) StartTurn(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	message string,
	focus *ProjectConversationFocus,
) (domain.Turn, error) {
	return s.startTurn(ctx, userID, conversationID, message, focus, nil)
}

func (s *ProjectConversationService) StartTurnWithWorkspaceFileDraft(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	message string,
	focus *ProjectConversationFocus,
	workspaceFileDraft *ProjectConversationWorkspaceFileDraftContext,
) (domain.Turn, error) {
	return s.startTurn(ctx, userID, conversationID, message, focus, workspaceFileDraft)
}

func (s *ProjectConversationService) startTurn(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	message string,
	focus *ProjectConversationFocus,
	workspaceFileDraft *ProjectConversationWorkspaceFileDraftContext,
) (domain.Turn, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.Turn{}, err
	}

	unlock := s.turnLocks.Lock(projectConversationTurnLockKey(conversation))
	defer unlock()

	project, err := s.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return domain.Turn{}, fmt.Errorf("get project for chat turn: %w", err)
	}
	providerItem, err := s.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return domain.Turn{}, fmt.Errorf("get provider for chat turn: %w", err)
	}

	_, hadLive := s.runtimeManager.Get(conversation.ID)
	if err := s.recoverStaleActiveTurnBeforeStart(ctx, conversation, hadLive); err != nil {
		return domain.Turn{}, err
	}

	live, hadLive, err := s.ensureLiveRuntime(ctx, conversation, project, providerItem)
	if err != nil {
		return domain.Turn{}, err
	}

	promptFocus := focus

	includeRecovery := !hadLive
	resumeThreadID := strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID))
	resumeTurnID := strings.TrimSpace(stringPointerValue(conversation.LastTurnID))
	if !hadLive && resumeThreadID != "" {
		switch providerItem.AdapterType {
		case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
			if live.codex == nil {
				break
			}
			resumePrompt, resumePromptErr := s.buildProjectConversationPromptWithDraft(
				ctx,
				conversation,
				project,
				promptFocus,
				workspaceFileDraft,
				false,
			)
			if resumePromptErr != nil {
				return domain.Turn{}, resumePromptErr
			}
			runtimeInput, inputErr := s.buildConversationRuntimeInput(
				ctx,
				conversation,
				project,
				providerItem,
				live.workspace,
				resumePrompt,
				resumeThreadID,
				resumeTurnID,
				promptFocus,
			)
			if inputErr != nil {
				return domain.Turn{}, inputErr
			}
			resumeErr := live.codex.EnsureSession(ctx, runtimeInput)
			switch {
			case resumeErr == nil:
				includeRecovery = false
			case codexadapter.IsThreadNotFoundError(resumeErr):
				resumeThreadID = ""
				resumeTurnID = ""
				emptyFlags := []string{}
				_, _ = s.conversations.UpdateConversationAnchors(ctx, conversationID, conversation.Status, domain.ConversationAnchors{
					ProviderThreadID:          optionalString(""),
					LastTurnID:                optionalString(""),
					ProviderThreadStatus:      optionalString("notLoaded"),
					ProviderThreadActiveFlags: &emptyFlags,
					RollingSummary:            conversation.RollingSummary,
				})
				includeRecovery = true
			default:
				return domain.Turn{}, resumeErr
			}
		case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
			if _, parseErr := provider.ParseClaudeCodeSessionID(resumeThreadID); parseErr == nil {
				includeRecovery = false
			}
		}
	}

	systemPrompt, err := s.buildProjectConversationPromptWithDraft(
		ctx,
		conversation,
		project,
		promptFocus,
		workspaceFileDraft,
		includeRecovery,
	)
	if err != nil {
		return domain.Turn{}, err
	}

	turn, _, err := s.entries.CreateTurnWithUserEntry(ctx, conversationID, strings.TrimSpace(message))
	if err != nil {
		return domain.Turn{}, err
	}
	if _, err := s.entries.AppendEntry(
		ctx,
		conversationID,
		&turn.ID,
		domain.EntryKindSystem,
		serializeProjectConversationFocus(promptFocus),
	); err != nil {
		return domain.Turn{}, err
	}

	runNow := time.Now().UTC()
	run, err := s.runtimeStore.CreateRun(ctx, domain.CreateRunInput{
		RunID:                uuid.New(),
		PrincipalID:          live.principal.ID,
		ConversationID:       conversation.ID,
		ProjectID:            conversation.ProjectID,
		ProviderID:           providerItem.ID,
		TurnID:               &turn.ID,
		Status:               domain.RunStatusLaunching,
		SessionID:            optionalString(conversation.ID.String()),
		WorkspacePath:        optionalString(live.workspace.String()),
		RuntimeStartedAt:     &runNow,
		LastHeartbeatAt:      &runNow,
		CurrentStepStatus:    optionalString("turn_launching"),
		CurrentStepSummary:   optionalString("Starting project conversation turn."),
		CurrentStepChangedAt: &runNow,
	})
	if err != nil {
		return domain.Turn{}, err
	}
	if principal, runtimeErr := s.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
		PrincipalID:          live.principal.ID,
		RuntimeState:         domain.RuntimeStateExecuting,
		CurrentSessionID:     optionalString(conversation.ID.String()),
		CurrentWorkspacePath: optionalString(live.workspace.String()),
		CurrentRunID:         &run.ID,
		LastHeartbeatAt:      &runNow,
		CurrentStepStatus:    optionalString("turn_launching"),
		CurrentStepSummary:   optionalString("Starting project conversation turn."),
		CurrentStepChangedAt: &runNow,
	}); runtimeErr == nil {
		live.principal = principal
	}

	environment, err := s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem, promptFocus)
	if err != nil {
		return domain.Turn{}, err
	}
	stream, err := live.runtime.StartTurn(ctx, RuntimeTurnInput{
		SessionID:              SessionID(conversationID.String()),
		Provider:               providerItem,
		Message:                strings.TrimSpace(message),
		SystemPrompt:           systemPrompt,
		WorkingDirectory:       live.workspace,
		Environment:            environment,
		ResumeProviderThreadID: resumeThreadID,
		ResumeProviderTurnID:   resumeTurnID,
		MaxTurns:               0,
		MaxBudgetUSD:           0,
		PersistentConversation: true,
	})
	if err != nil {
		failedStatus := domain.RunStatusFailed
		_, _ = s.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
			RunID:                run.ID,
			Status:               &failedStatus,
			TerminalAt:           &runNow,
			LastError:            optionalString(err.Error()),
			LastHeartbeatAt:      &runNow,
			CurrentStepStatus:    optionalString("turn_failed"),
			CurrentStepSummary:   optionalString("Project conversation turn failed to start."),
			CurrentStepChangedAt: &runNow,
		})
		if principal, runtimeErr := s.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
			PrincipalID:          live.principal.ID,
			RuntimeState:         domain.RuntimeStateReady,
			CurrentSessionID:     optionalString(conversation.ID.String()),
			CurrentWorkspacePath: optionalString(live.workspace.String()),
			CurrentRunID:         &run.ID,
			LastHeartbeatAt:      &runNow,
			CurrentStepStatus:    optionalString("runtime_ready"),
			CurrentStepSummary:   optionalString("Project conversation runtime ready."),
			CurrentStepChangedAt: &runNow,
		}); runtimeErr == nil {
			live.principal = principal
		}
		return domain.Turn{}, err
	}
	executingStatus := domain.RunStatusExecuting
	_, _ = s.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
		RunID:                run.ID,
		Status:               &executingStatus,
		LastHeartbeatAt:      &runNow,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &runNow,
	})

	go s.consumeTurn(context.WithoutCancel(ctx), conversation, turn, live, run, stream)
	return turn, nil
}

func (s *ProjectConversationService) RespondInterrupt(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	interruptID uuid.UUID,
	response domain.InterruptResponse,
) (domain.PendingInterrupt, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	interrupt, err := s.interrupts.GetPendingInterrupt(ctx, interruptID)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	if interrupt.ConversationID != conversation.ID {
		return domain.PendingInterrupt{}, ErrPendingInterruptNotFound
	}
	live, _ := s.runtimeManager.Get(conversationID)
	if live != nil && live.interrupt == nil && live.codex != nil {
		live.interrupt = live.codex
	}
	var project catalogdomain.Project
	var providerItem catalogdomain.AgentProvider
	if live != nil {
		providerItem = live.provider
	}
	if s.catalog != nil {
		var projectErr error
		project, projectErr = s.catalog.GetProject(ctx, conversation.ProjectID)
		if projectErr != nil {
			return domain.PendingInterrupt{}, fmt.Errorf("get project for interrupt response: %w", projectErr)
		}
		if providerItem.ID == uuid.Nil {
			var providerErr error
			providerItem, providerErr = s.catalog.GetAgentProvider(ctx, conversation.ProviderID)
			if providerErr != nil {
				return domain.PendingInterrupt{}, fmt.Errorf("get provider for interrupt response: %w", providerErr)
			}
		}
	}
	if live == nil || live.interrupt == nil {
		if s.catalog == nil || providerItem.ID == uuid.Nil {
			return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
		}
		storedFocus, focusErr := s.loadLatestConversationFocus(ctx, conversationID)
		if focusErr != nil {
			return domain.PendingInterrupt{}, focusErr
		}
		var ensureErr error
		live, _, ensureErr = s.ensureLiveRuntime(ctx, conversation, project, providerItem)
		if ensureErr != nil {
			return domain.PendingInterrupt{}, ensureErr
		}
		if live.interrupt == nil {
			return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
		}
		if live.codex != nil {
			systemPrompt, promptErr := s.buildProjectConversationPrompt(ctx, conversation, project, storedFocus, false)
			if promptErr != nil {
				return domain.PendingInterrupt{}, promptErr
			}
			runtimeInput, inputErr := s.buildConversationRuntimeInput(
				ctx,
				conversation,
				project,
				providerItem,
				live.workspace,
				systemPrompt,
				strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID)),
				strings.TrimSpace(stringPointerValue(conversation.LastTurnID)),
				storedFocus,
			)
			if inputErr != nil {
				return domain.PendingInterrupt{}, inputErr
			}
			ensureErr = live.codex.EnsureSession(ctx, runtimeInput)
			if ensureErr != nil {
				if codexadapter.IsThreadNotFoundError(ensureErr) {
					return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
				}
				return domain.PendingInterrupt{}, ensureErr
			}
		}
	}

	runtimeKind := runtimeInterruptKind(interrupt.Kind)
	storedFocus, err := s.loadLatestConversationFocus(ctx, conversationID)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	environment, err := s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem, storedFocus)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	stream, err := live.interrupt.RespondInterrupt(ctx, RuntimeInterruptResponseInput{
		SessionID:              SessionID(conversationID.String()),
		ProjectID:              project.ID,
		TicketID:               conversationFocusTicketID(storedFocus),
		Provider:               live.provider,
		RequestID:              interrupt.ProviderRequestID,
		Kind:                   runtimeKind,
		Decision:               stringPointerValue(response.Decision),
		Answer:                 cloneMapAny(response.Answer),
		Payload:                cloneMapAny(interrupt.Payload),
		WorkingDirectory:       live.workspace,
		Environment:            environment,
		ResumeProviderThreadID: strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID)),
		ResumeProviderTurnID:   strings.TrimSpace(stringPointerValue(conversation.LastTurnID)),
		PersistentConversation: true,
	})
	if err != nil {
		return domain.PendingInterrupt{}, err
	}

	resolved, _ /* entry */, err := s.interrupts.ResolvePendingInterrupt(ctx, interruptID, response)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	anchor := RuntimeSessionAnchor{}
	if live.interrupt != nil {
		anchor = live.interrupt.SessionAnchor(SessionID(conversationID.String()))
	}
	_, _ = s.conversations.UpdateConversationAnchors(
		ctx,
		conversationID,
		domain.ConversationStatusActive,
		conversationAnchorsFromRuntimeAnchor(anchor, ""),
	)
	s.broadcastConversationEvent(conversation, StreamEvent{
		Event: "interrupt_resolved",
		Payload: map[string]any{
			"interrupt_id": resolved.ID.String(),
			"decision":     stringPointerValue(resolved.Decision),
		},
	})
	if stream.Events != nil {
		run, runErr := s.runtimeStore.GetRunByTurnID(ctx, interrupt.TurnID)
		if runErr == nil {
			now := time.Now().UTC()
			executingStatus := domain.RunStatusExecuting
			_, _ = s.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
				RunID:                run.ID,
				Status:               &executingStatus,
				LastHeartbeatAt:      &now,
				CurrentStepStatus:    optionalString("interrupt_resolved"),
				CurrentStepSummary:   optionalString("Project conversation interrupt resolved."),
				CurrentStepChangedAt: &now,
			})
			if principal, runtimeErr := s.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
				PrincipalID:          live.principal.ID,
				RuntimeState:         domain.RuntimeStateExecuting,
				CurrentSessionID:     optionalString(conversation.ID.String()),
				CurrentWorkspacePath: optionalString(live.workspace.String()),
				CurrentRunID:         &run.ID,
				LastHeartbeatAt:      &now,
				CurrentStepStatus:    optionalString("turn_executing"),
				CurrentStepSummary:   optionalString("Project conversation turn executing."),
				CurrentStepChangedAt: &now,
			}); runtimeErr == nil {
				live.principal = principal
			}
			go s.consumeTurn(context.WithoutCancel(ctx), conversation, domain.Turn{
				ID:             interrupt.TurnID,
				ConversationID: conversationID,
			}, live, run, stream)
			return resolved, nil
		}
		go s.consumeTurn(context.WithoutCancel(ctx), conversation, domain.Turn{
			ID:             interrupt.TurnID,
			ConversationID: conversationID,
		}, live, domain.ProjectConversationRun{}, stream)
	}
	return resolved, nil
}

func (s *ProjectConversationService) AppendSystemEntry(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	turnID *uuid.UUID,
	payload map[string]any,
) (domain.Entry, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.Entry{}, err
	}
	entryPayload := cloneMapAny(payload)
	entry, err := s.entries.AppendEntry(ctx, conversationID, turnID, domain.EntryKindSystem, entryPayload)
	if err != nil {
		return domain.Entry{}, err
	}
	s.broadcastConversationEvent(conversation, StreamEvent{Event: "message", Payload: cloneMapAny(payload)})
	return entry, nil
}

func (s *ProjectConversationService) CloseRuntime(ctx context.Context, userID UserID, conversationID uuid.UUID) error {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	return s.closeConversationRuntime(ctx, conversation)
}

func (s *ProjectConversationService) InterruptTurn(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) error {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}

	unlock := s.turnLocks.Lock(projectConversationTurnLockKey(conversation))
	defer unlock()

	activeTurn, err := s.entries.GetActiveTurn(ctx, conversationID)
	switch {
	case err == nil:
	case errors.Is(err, ErrConversationNotFound):
		return ErrConversationTurnNotActive
	default:
		return err
	}

	if activeTurn.Status != domain.TurnStatusRunning && activeTurn.Status != domain.TurnStatusPending {
		return ErrConversationTurnNotActive
	}

	pendingInterrupts, err := s.interrupts.ListPendingInterrupts(ctx, conversationID)
	if err != nil {
		return err
	}
	for _, interrupt := range pendingInterrupts {
		if interrupt.TurnID == activeTurn.ID && interrupt.Status == domain.InterruptStatusPending {
			return ErrConversationInterruptPending
		}
	}

	live, ok := s.runtimeManager.Get(conversationID)
	if !ok || live == nil || live.turnStop == nil {
		return ErrConversationRuntimeAbsent
	}

	anchor, err := live.turnStop.InterruptTurn(ctx, SessionID(conversationID.String()))
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	statusMessage := "Turn stopped by user."
	reason := "stopped_by_user"

	anchorThreadID := ""
	anchorTurnID := ""
	if live.provider.AdapterType != catalogdomain.AgentProviderAdapterTypeCodexAppServer {
		anchorThreadID = firstNonEmptyTrimmed(
			anchor.ProviderThreadID,
			stringPointerValue(conversation.ProviderThreadID),
		)
		anchorTurnID = firstNonEmptyTrimmed(
			anchor.LastTurnID,
			stringPointerValue(conversation.LastTurnID),
		)
	}
	if _, err := s.entries.CompleteTurn(ctx, activeTurn.ID, domain.TurnStatusInterrupted, optionalNonEmptyString(anchorTurnID)); err != nil {
		return err
	}
	if _, err := s.appendConversationEntryWithConflictRetry(ctx, conversationID, &activeTurn.ID, domain.EntryKindSystem, map[string]any{
		"type":    "turn_interrupted",
		"reason":  reason,
		"message": statusMessage,
	}); err != nil {
		return err
	}
	entries, err := s.entries.ListEntries(ctx, conversationID)
	if err != nil {
		return err
	}
	summary := buildRollingSummary(entries)
	emptyFlags := []string{}
	updatedConversation, err := s.conversations.UpdateConversationAnchors(
		ctx,
		conversationID,
		domain.ConversationStatusActive,
		domain.ConversationAnchors{
			ProviderThreadID:          optionalNonEmptyString(anchorThreadID),
			LastTurnID:                optionalNonEmptyString(anchorTurnID),
			ProviderThreadStatus:      optionalString("notLoaded"),
			ProviderThreadActiveFlags: &emptyFlags,
			RollingSummary:            summary,
		},
	)
	if err != nil {
		return err
	}

	if live.principal.ID != uuid.Nil {
		clearRunID := uuid.Nil
		if principal, principalErr := s.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
			PrincipalID:          live.principal.ID,
			RuntimeState:         domain.RuntimeStateReady,
			CurrentSessionID:     optionalString(conversation.ID.String()),
			CurrentWorkspacePath: optionalString(live.workspace.String()),
			CurrentRunID:         &clearRunID,
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_interrupted"),
			CurrentStepSummary:   optionalString(statusMessage),
			CurrentStepChangedAt: &now,
		}); principalErr == nil {
			live.principal = principal
		}
	}

	var run domain.ProjectConversationRun
	if storedRun, runErr := s.runtimeStore.GetRunByTurnID(ctx, activeTurn.ID); runErr == nil {
		run = storedRun
		interruptedStatus := domain.RunStatusInterrupted
		_, _ = s.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
			RunID:                run.ID,
			Status:               &interruptedStatus,
			ProviderThreadID:     optionalNonEmptyString(anchorThreadID),
			ProviderTurnID:       optionalNonEmptyString(anchorTurnID),
			TerminalAt:           &now,
			LastError:            optionalString(""),
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_interrupted"),
			CurrentStepSummary:   optionalString(statusMessage),
			CurrentStepChangedAt: &now,
		})
	}

	if run.ID != uuid.Nil {
		s.recordConversationTrace(ctx, live, run, "interrupted", map[string]any{
			"message": statusMessage,
			"reason":  reason,
		}, "runtime")
	}

	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event: "message",
		Payload: map[string]any{
			"type": "turn_interrupted",
			"raw": map[string]any{
				"message": statusMessage,
				"reason":  reason,
			},
		},
	})
	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event: "interrupted",
		Payload: map[string]any{
			"conversation_id": conversationID.String(),
			"turn_id":         activeTurn.ID.String(),
			"message":         statusMessage,
			"reason":          reason,
		},
	})
	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event:   "session",
		Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateReady), updatedConversation, &live.provider),
	})
	return nil
}

func (s *ProjectConversationService) appendConversationEntryWithConflictRetry(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID *uuid.UUID,
	kind domain.EntryKind,
	payload map[string]any,
) (domain.Entry, error) {
	const maxAttempts = 3

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		entry, err := s.entries.AppendEntry(ctx, conversationID, turnID, kind, payload)
		if err == nil {
			return entry, nil
		}
		if !errors.Is(err, ErrConversationConflict) {
			return domain.Entry{}, err
		}
		lastErr = err
		if ctx.Err() != nil {
			return domain.Entry{}, ctx.Err()
		}
		if attempt == maxAttempts-1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return domain.Entry{}, lastErr
}

func (s *ProjectConversationService) recoverStaleActiveTurnBeforeStart(
	ctx context.Context,
	conversation domain.Conversation,
	hadLive bool,
) error {
	activeTurn, err := s.entries.GetActiveTurn(ctx, conversation.ID)
	switch {
	case err == nil:
	case errors.Is(err, ErrConversationNotFound):
		return nil
	default:
		return err
	}

	pendingInterrupts, err := s.interrupts.ListPendingInterrupts(ctx, conversation.ID)
	if err != nil {
		return err
	}
	hasPendingInterrupt := false
	for _, interrupt := range pendingInterrupts {
		if interrupt.TurnID == activeTurn.ID && interrupt.Status == domain.InterruptStatusPending {
			hasPendingInterrupt = true
			break
		}
	}

	if activeTurn.Status == domain.TurnStatusInterrupted && hasPendingInterrupt {
		return ErrConversationInterruptPending
	}
	if hadLive {
		return ErrConversationTurnActive
	}
	return s.terminateStaleActiveTurn(ctx, conversation, activeTurn)
}

func (s *ProjectConversationService) terminateStaleActiveTurn(
	ctx context.Context,
	conversation domain.Conversation,
	activeTurn domain.Turn,
) error {
	now := time.Now().UTC()
	statusSummary := fmt.Sprintf(
		"Recovered stale %s turn after runtime became unavailable.",
		strings.TrimSpace(string(activeTurn.Status)),
	)
	if _, err := s.entries.CompleteTurn(ctx, activeTurn.ID, domain.TurnStatusTerminated, activeTurn.ProviderTurnID); err != nil {
		return err
	}
	if _, err := s.entries.AppendEntry(ctx, conversation.ID, &activeTurn.ID, domain.EntryKindSystem, map[string]any{
		"type":                 "turn_recovered_after_runtime_unavailable",
		"previous_turn_id":     activeTurn.ID.String(),
		"previous_turn_status": string(activeTurn.Status),
		"recovery_reason":      "runtime_unavailable",
	}); err != nil {
		return err
	}

	if run, err := s.runtimeStore.GetRunByTurnID(ctx, activeTurn.ID); err == nil {
		terminatedStatus := domain.RunStatusTerminated
		_, _ = s.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
			RunID:                run.ID,
			Status:               &terminatedStatus,
			TerminalAt:           &now,
			LastError:            optionalString("project conversation runtime became unavailable before the turn completed"),
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_recovered"),
			CurrentStepSummary:   optionalString(statusSummary),
			CurrentStepChangedAt: &now,
		})
	}
	if principal, err := s.runtimeStore.GetPrincipal(ctx, conversation.ID); err == nil {
		clearRunID := uuid.Nil
		_, _ = s.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
			PrincipalID:          principal.ID,
			RuntimeState:         domain.RuntimeStateInactive,
			CurrentSessionID:     optionalString(""),
			CurrentRunID:         &clearRunID,
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_recovered"),
			CurrentStepSummary:   optionalString(statusSummary),
			CurrentStepChangedAt: &now,
		})
	}

	emptyFlags := []string{}
	updatedConversation, err := s.conversations.UpdateConversationAnchors(
		ctx,
		conversation.ID,
		domain.ConversationStatusActive,
		domain.ConversationAnchors{
			ProviderThreadID:          conversation.ProviderThreadID,
			LastTurnID:                conversation.LastTurnID,
			ProviderThreadStatus:      optionalString("notLoaded"),
			ProviderThreadActiveFlags: &emptyFlags,
			RollingSummary:            conversation.RollingSummary,
		},
	)
	if err != nil {
		return err
	}
	if s.catalog != nil {
		if providerItem, providerErr := s.catalog.GetAgentProvider(ctx, conversation.ProviderID); providerErr == nil {
			s.broadcastConversationEvent(updatedConversation, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversation.ID, "inactive", updatedConversation, &providerItem),
			})
		}
	}
	s.broadcastConversationEvent(conversation, StreamEvent{
		Event: "turn_recovered",
		Payload: map[string]any{
			"turn_id":         activeTurn.ID.String(),
			"previous_status": string(activeTurn.Status),
			"recovery_reason": "runtime_unavailable",
		},
	})
	return nil
}

func (s *ProjectConversationService) consumeTurn(
	ctx context.Context,
	conversation domain.Conversation,
	turn domain.Turn,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	stream TurnStream,
) {
	conversationID := conversation.ID
	usageHighWater := projectConversationUsageHighWater{
		inputTokens:         run.InputTokens,
		outputTokens:        run.OutputTokens,
		cachedInputTokens:   run.CachedInputTokens,
		cacheCreationTokens: run.CacheCreationTokens,
		reasoningTokens:     run.ReasoningTokens,
		promptTokens:        run.PromptTokens,
		candidateTokens:     run.CandidateTokens,
		toolTokens:          run.ToolTokens,
		totalTokens:         run.TotalTokens,
		costAmount:          run.CostAmount,
		hasCostAmount:       run.CostAmount > 0,
	}
	for event := range stream.Events {
		switch event.Event {
		case "message":
			if normalized, ok := s.handleConversationMessage(ctx, conversationID, turn.ID, event.Payload); ok {
				s.recordConversationTrace(ctx, live, run, "message", normalized.Payload, "assistant")
				s.broadcastConversationEvent(conversation, normalized)
				continue
			}
			s.recordConversationTrace(ctx, live, run, "message", event.Payload, "assistant")
			s.broadcastConversationEvent(conversation, event)
		case "interrupt_requested":
			payload, ok := event.Payload.(RuntimeInterruptEvent)
			if !ok {
				continue
			}
			interruptKind := mapDomainInterruptKind(payload.Kind)
			interruptProvider := providerInterruptProviderName(live.provider)
			interruptPayload := map[string]any{
				"provider": interruptProvider,
				"kind":     string(interruptKind),
				"payload":  cloneMapAny(payload.Payload),
			}
			if len(payload.Options) > 0 {
				options := make([]map[string]any, 0, len(payload.Options))
				for _, option := range payload.Options {
					options = append(options, map[string]any{
						"id":    option.ID,
						"label": option.Label,
					})
				}
				interruptPayload["options"] = options
			}
			pending, _, err := s.interrupts.CreatePendingInterrupt(ctx, conversationID, turn.ID, payload.RequestID, interruptKind, interruptPayload)
			if err != nil {
				s.logger.Error("persist chat interrupt", "conversation_id", conversationID, "error", err)
				continue
			}
			s.recordConversationTrace(ctx, live, run, "interrupt_requested", interruptPayload, "runtime")
			anchor := RuntimeSessionAnchor{}
			if live.codex != nil {
				anchor = live.codex.SessionAnchor(SessionID(conversationID.String()))
			}
			_, _ = s.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusInterrupted,
				conversationAnchorsFromRuntimeAnchor(anchor, ""),
			)
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateInterrupted, domain.RunStatusInterrupted, "interrupt_requested", "Project conversation is waiting for user interrupt resolution.", nil, nil, "")
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "interrupt_requested",
				Payload: map[string]any{
					"interrupt_id": pending.ID.String(),
					"provider":     interruptProvider,
					"kind":         string(interruptKind),
					"options":      interruptPayload["options"],
					"payload":      interruptPayload["payload"],
				},
			})
		case "done":
			done, ok := event.Payload.(donePayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "done", map[string]any{
				"cost_usd": done.CostUSD,
			}, "runtime")
			anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
			_, _ = s.entries.CompleteTurn(ctx, turn.ID, domain.TurnStatusCompleted, optionalNonEmptyString(anchor.LastTurnID))
			entries, _ := s.entries.ListEntries(ctx, conversationID)
			summary := buildRollingSummary(entries)
			updatedConversation := conversation
			if storedConversation, err := s.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				conversationAnchorsFromRuntimeAnchor(anchor, summary),
			); err == nil {
				updatedConversation = storedConversation
			}
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateReady, domain.RunStatusCompleted, "turn_completed", "Project conversation turn completed.", optionalNonEmptyString(anchor.ProviderThreadID), done.CostUSD, "")
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "turn_done",
				Payload: map[string]any{
					"conversation_id": conversationID.String(),
					"turn_id":         turn.ID.String(),
					"cost_usd":        done.CostUSD,
				},
			})
			s.autoReleaseCompletedRuntime(ctx, updatedConversation, live)
		case "token_usage_updated":
			payload, ok := event.Payload.(runtimeTokenUsagePayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "token_usage_updated", map[string]any{
				"input_tokens":          payload.TotalInputTokens,
				"output_tokens":         payload.TotalOutputTokens,
				"cached_input_tokens":   payload.TotalCachedInputTokens,
				"cache_creation_tokens": payload.TotalCacheCreationTokens,
				"reasoning_tokens":      payload.TotalReasoningTokens,
				"prompt_tokens":         payload.TotalPromptTokens,
				"candidate_tokens":      payload.TotalCandidateTokens,
				"tool_tokens":           payload.TotalToolTokens,
				"total_tokens":          payload.TotalTokens,
				"cost_usd":              payload.CostUSD,
				"model_context_window":  payload.ModelContextWindow,
			}, "runtime")
			run = s.recordConversationUsage(ctx, live, run, payload, &usageHighWater)
		case "rate_limit_updated":
			payload, ok := event.Payload.(runtimeRateLimitPayload)
			if !ok || payload.RateLimit == nil {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "rate_limit_updated", map[string]any{
				"observed_at": payload.ObservedAt.UTC().Format(time.RFC3339),
			}, "runtime")
			s.recordConversationProviderRateLimit(ctx, live, payload)
		case "thread_status":
			payload, ok := event.Payload.(runtimeThreadStatusPayload)
			if !ok {
				continue
			}
			activeFlags := append([]string(nil), payload.ActiveFlags...)
			s.recordConversationTrace(ctx, live, run, "thread_status", map[string]any{
				"thread_id":    payload.ThreadID,
				"status":       payload.Status,
				"active_flags": activeFlags,
			}, "runtime")
			updated, _ := s.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				domain.ConversationAnchors{
					ProviderThreadStatus:      optionalString(payload.Status),
					ProviderThreadActiveFlags: &activeFlags,
				},
			)
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateExecuting, domain.RunStatusExecuting, payload.Status, "Conversation provider thread status updated.", optionalNonEmptyString(payload.ThreadID), nil, "")
			entry, _ := s.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":         "thread_status",
				"anchor_kind":  "thread",
				"thread_id":    payload.ThreadID,
				"status":       payload.Status,
				"active_flags": append([]string(nil), payload.ActiveFlags...),
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "message",
				Payload: map[string]any{
					"type": "thread_status",
					"raw": map[string]any{
						"anchor_kind":  "thread",
						"thread_id":    payload.ThreadID,
						"status":       payload.Status,
						"active_flags": append([]string(nil), payload.ActiveFlags...),
						"entry_id":     entry.ID.String(),
					},
				},
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "thread_status",
				Payload: map[string]any{
					"thread_id":    payload.ThreadID,
					"status":       payload.Status,
					"active_flags": append([]string(nil), payload.ActiveFlags...),
					"entry_id":     entry.ID.String(),
				},
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateExecuting), updated, &live.provider),
			})
		case "session_anchor":
			anchor, ok := event.Payload.(RuntimeSessionAnchor)
			if !ok || strings.TrimSpace(anchor.ProviderThreadID) == "" {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "session_anchor", map[string]any{
				"provider_thread_id":     anchor.ProviderThreadID,
				"last_turn_id":           anchor.LastTurnID,
				"provider_thread_status": anchor.ProviderThreadStatus,
				"active_flags":           append([]string(nil), anchor.ProviderThreadActiveFlags...),
			}, "runtime")
			updated, _ := s.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				conversationAnchorsFromRuntimeAnchor(anchor, ""),
			)
			s.broadcastConversationEvent(updated, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateExecuting), updated, &live.provider),
			})
		case "session_state":
			payload, ok := event.Payload.(runtimeSessionStatePayload)
			if !ok {
				continue
			}
			activeFlags := append([]string(nil), payload.ActiveFlags...)
			s.recordConversationTrace(ctx, live, run, "session_state", map[string]any{
				"status":       payload.Status,
				"active_flags": activeFlags,
				"detail":       payload.Detail,
				"raw":          cloneAnyMap(payload.Raw),
			}, "runtime")
			updated, _ := s.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				domain.ConversationAnchors{
					ProviderThreadStatus:      optionalString(payload.Status),
					ProviderThreadActiveFlags: &activeFlags,
				},
			)
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateExecuting, domain.RunStatusExecuting, payload.Status, payload.Detail, nil, nil, "")
			entry, _ := s.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":         "session_state",
				"anchor_kind":  "session",
				"status":       payload.Status,
				"active_flags": append([]string(nil), payload.ActiveFlags...),
				"detail":       payload.Detail,
				"raw":          cloneAnyMap(payload.Raw),
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "message",
				Payload: map[string]any{
					"type": "session_state",
					"raw": map[string]any{
						"anchor_kind":  "session",
						"status":       payload.Status,
						"active_flags": append([]string(nil), payload.ActiveFlags...),
						"detail":       payload.Detail,
						"entry_id":     entry.ID.String(),
					},
				},
			})
			s.broadcastConversationEvent(updated, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateExecuting), updated, &live.provider),
			})
		case "thread_compacted":
			payload, ok := event.Payload.(runtimeThreadCompactedPayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "thread_compacted", map[string]any{
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
			}, "runtime")
			entry, _ := s.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":      "thread_compacted",
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "thread_compacted",
				Payload: map[string]any{
					"thread_id": payload.ThreadID,
					"turn_id":   payload.TurnID,
					"entry_id":  entry.ID.String(),
				},
			})
		case "plan_updated":
			payload, ok := event.Payload.(runtimePlanUpdatedPayload)
			if !ok {
				continue
			}
			rawPlan := make([]map[string]any, 0, len(payload.Plan))
			for _, item := range payload.Plan {
				rawPlan = append(rawPlan, map[string]any{
					"step":   item.Step,
					"status": item.Status,
				})
			}
			s.recordConversationTrace(ctx, live, run, "plan_updated", map[string]any{
				"thread_id":   payload.ThreadID,
				"turn_id":     payload.TurnID,
				"explanation": payload.Explanation,
				"plan":        rawPlan,
			}, "runtime")
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateExecuting, domain.RunStatusExecuting, "plan_updated", stringPointerValue(payload.Explanation), nil, nil, "")
			entry, _ := s.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":        "turn_plan_updated",
				"thread_id":   payload.ThreadID,
				"turn_id":     payload.TurnID,
				"explanation": payload.Explanation,
				"plan":        rawPlan,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "plan_updated",
				Payload: map[string]any{
					"thread_id":   payload.ThreadID,
					"turn_id":     payload.TurnID,
					"explanation": payload.Explanation,
					"plan":        rawPlan,
					"entry_id":    entry.ID.String(),
				},
			})
		case "diff_updated":
			payload, ok := event.Payload.(runtimeDiffUpdatedPayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "diff_updated", map[string]any{
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
				"diff":      payload.Diff,
			}, "runtime")
			entry, _ := s.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":      "turn_diff_updated",
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
				"diff":      payload.Diff,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "diff_updated",
				Payload: map[string]any{
					"thread_id": payload.ThreadID,
					"turn_id":   payload.TurnID,
					"diff":      payload.Diff,
					"entry_id":  entry.ID.String(),
				},
			})
		case "reasoning_updated":
			payload, ok := event.Payload.(runtimeReasoningUpdatedPayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "reasoning_updated", map[string]any{
				"thread_id":     payload.ThreadID,
				"turn_id":       payload.TurnID,
				"item_id":       payload.ItemID,
				"kind":          payload.Kind,
				"delta":         payload.Delta,
				"summary_index": payload.SummaryIndex,
				"content_index": payload.ContentIndex,
			}, "runtime")
			entry, _ := s.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":          "turn_reasoning_updated",
				"thread_id":     payload.ThreadID,
				"turn_id":       payload.TurnID,
				"item_id":       payload.ItemID,
				"kind":          payload.Kind,
				"delta":         payload.Delta,
				"summary_index": payload.SummaryIndex,
				"content_index": payload.ContentIndex,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "reasoning_updated",
				Payload: map[string]any{
					"thread_id":     payload.ThreadID,
					"turn_id":       payload.TurnID,
					"item_id":       payload.ItemID,
					"kind":          payload.Kind,
					"delta":         payload.Delta,
					"summary_index": payload.SummaryIndex,
					"content_index": payload.ContentIndex,
					"entry_id":      entry.ID.String(),
				},
			})
		case "error":
			payload, ok := event.Payload.(errorPayload)
			if ok {
				s.recordConversationTrace(ctx, live, run, "error", map[string]any{"message": payload.Message}, "runtime")
				anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
				_, _ = s.entries.CompleteTurn(ctx, turn.ID, domain.TurnStatusFailed, optionalNonEmptyString(anchor.LastTurnID))
				_, _ = s.conversations.UpdateConversationAnchors(
					ctx,
					conversationID,
					domain.ConversationStatusActive,
					conversationAnchorsFromRuntimeAnchor(anchor, ""),
				)
				s.recordConversationStep(ctx, live, run, domain.RuntimeStateReady, domain.RunStatusFailed, "turn_failed", payload.Message, optionalNonEmptyString(anchor.ProviderThreadID), nil, payload.Message)
				s.broadcastConversationEvent(conversation, StreamEvent{
					Event: "error",
					Payload: map[string]any{
						"message": payload.Message,
					},
				})
			}
		case "interrupted":
			payload, ok := event.Payload.(errorPayload)
			if ok {
				s.recordConversationTrace(ctx, live, run, "interrupted", map[string]any{"message": payload.Message}, "runtime")
				anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
				_, _ = s.entries.CompleteTurn(ctx, turn.ID, domain.TurnStatusInterrupted, optionalNonEmptyString(anchor.LastTurnID))
				_, _ = s.conversations.UpdateConversationAnchors(
					ctx,
					conversationID,
					domain.ConversationStatusInterrupted,
					conversationAnchorsFromRuntimeAnchor(anchor, ""),
				)
				s.recordConversationStep(ctx, live, run, domain.RuntimeStateInterrupted, domain.RunStatusInterrupted, "turn_interrupted", payload.Message, optionalNonEmptyString(anchor.ProviderThreadID), nil, "")
				s.broadcastConversationEvent(conversation, StreamEvent{
					Event: "interrupted",
					Payload: map[string]any{
						"message": payload.Message,
					},
				})
			}
		}
	}
}

func (s *ProjectConversationService) recordConversationTrace(
	ctx context.Context,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	kind string,
	payload any,
	stream string,
) {
	if s == nil || s.runtimeStore == nil || live == nil || live.principal.ID == uuid.Nil || run.ID == uuid.Nil {
		return
	}
	now := time.Now().UTC()
	tracePayload := mapConversationTracePayload(payload)
	trace, err := s.runtimeStore.AppendTraceEvent(ctx, domain.AppendTraceEventInput{
		RunID:          run.ID,
		PrincipalID:    live.principal.ID,
		ConversationID: live.principal.ConversationID,
		ProjectID:      live.principal.ProjectID,
		Provider:       providerInterruptProviderName(live.provider),
		Kind:           kind,
		Stream:         stream,
		Text:           optionalNonEmptyString(extractConversationTraceText(tracePayload)),
		Payload:        tracePayload,
	})
	if err != nil {
		s.logger.Warn("append project conversation trace event failed", "conversation_id", live.principal.ConversationID, "run_id", run.ID, "error", err)
		return
	}
	_ = trace
	_, _ = s.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
		RunID:                run.ID,
		LastHeartbeatAt:      &now,
		CurrentStepChangedAt: &now,
	})
}

func (s *ProjectConversationService) recordConversationStep(
	ctx context.Context,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	runtimeState domain.RuntimeState,
	runStatus domain.RunStatus,
	stepStatus string,
	summary string,
	providerThreadID *string,
	costUSD *float64,
	lastError string,
) {
	if s == nil || s.runtimeStore == nil || live == nil || live.principal.ID == uuid.Nil || run.ID == uuid.Nil {
		return
	}
	now := time.Now().UTC()
	summaryPtr := optionalNonEmptyString(summary)
	step, err := s.runtimeStore.AppendStepEvent(ctx, domain.AppendStepEventInput{
		RunID:          run.ID,
		PrincipalID:    live.principal.ID,
		ConversationID: live.principal.ConversationID,
		ProjectID:      live.principal.ProjectID,
		StepStatus:     stepStatus,
		Summary:        summaryPtr,
	})
	if err != nil {
		s.logger.Warn("append project conversation step event failed", "conversation_id", live.principal.ConversationID, "run_id", run.ID, "error", err)
	}
	_ = step
	runStatusCopy := runStatus
	updateInput := domain.UpdateRunInput{
		RunID:                run.ID,
		Status:               &runStatusCopy,
		ProviderThreadID:     providerThreadID,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString(stepStatus),
		CurrentStepSummary:   summaryPtr,
		CurrentStepChangedAt: &now,
	}
	if costUSD != nil {
		updateInput.CostAmount = costUSD
	}
	if strings.TrimSpace(lastError) != "" {
		updateInput.LastError = optionalString(lastError)
	}
	if runStatus == domain.RunStatusCompleted || runStatus == domain.RunStatusFailed || runStatus == domain.RunStatusTerminated {
		updateInput.TerminalAt = &now
	}
	updatedRun, err := s.runtimeStore.UpdateRun(ctx, updateInput)
	if err == nil {
		run = updatedRun
	}
	currentRunID := run.ID
	updatedPrincipal, err := s.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
		PrincipalID:          live.principal.ID,
		RuntimeState:         runtimeState,
		CurrentSessionID:     optionalString(live.principal.ConversationID.String()),
		CurrentWorkspacePath: optionalString(live.workspace.String()),
		CurrentRunID:         &currentRunID,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString(stepStatus),
		CurrentStepSummary:   summaryPtr,
		CurrentStepChangedAt: &now,
	})
	if err == nil {
		live.principal = updatedPrincipal
	}
}

func (s *ProjectConversationService) recordConversationUsage(
	ctx context.Context,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	payload runtimeTokenUsagePayload,
	highWater *projectConversationUsageHighWater,
) domain.ProjectConversationRun {
	if s == nil || s.runtimeStore == nil || live == nil || run.ID == uuid.Nil || highWater == nil {
		return run
	}

	delta := domain.RunUsageSnapshot{
		InputTokens:         clampUsageDelta(payload.TotalInputTokens - highWater.inputTokens),
		OutputTokens:        clampUsageDelta(payload.TotalOutputTokens - highWater.outputTokens),
		CachedInputTokens:   clampUsageDelta(payload.TotalCachedInputTokens - highWater.cachedInputTokens),
		CacheCreationTokens: clampUsageDelta(payload.TotalCacheCreationTokens - highWater.cacheCreationTokens),
		ReasoningTokens:     clampUsageDelta(payload.TotalReasoningTokens - highWater.reasoningTokens),
		PromptTokens:        clampUsageDelta(payload.TotalPromptTokens - highWater.promptTokens),
		CandidateTokens:     clampUsageDelta(payload.TotalCandidateTokens - highWater.candidateTokens),
		ToolTokens:          clampUsageDelta(payload.TotalToolTokens - highWater.toolTokens),
		TotalTokens:         clampUsageDelta(payload.TotalTokens - highWater.totalTokens),
	}
	if payload.CostUSD != nil {
		costDelta := *payload.CostUSD
		if highWater.hasCostAmount {
			costDelta -= highWater.costAmount
		}
		if costDelta > 0 {
			delta.CostAmount = cloneCostUSD(&costDelta)
		}
	}

	updatedRun, err := s.runtimeStore.RecordRunUsage(ctx, domain.RecordRunUsageInput{
		RunID:      run.ID,
		ProjectID:  live.principal.ProjectID,
		ProviderID: live.principal.ProviderID,
		RecordedAt: time.Now().UTC(),
		Totals: domain.RunUsageSnapshot{
			InputTokens:         maxRunUsageTotal(highWater.inputTokens, payload.TotalInputTokens),
			OutputTokens:        maxRunUsageTotal(highWater.outputTokens, payload.TotalOutputTokens),
			CachedInputTokens:   maxRunUsageTotal(highWater.cachedInputTokens, payload.TotalCachedInputTokens),
			CacheCreationTokens: maxRunUsageTotal(highWater.cacheCreationTokens, payload.TotalCacheCreationTokens),
			ReasoningTokens:     maxRunUsageTotal(highWater.reasoningTokens, payload.TotalReasoningTokens),
			PromptTokens:        maxRunUsageTotal(highWater.promptTokens, payload.TotalPromptTokens),
			CandidateTokens:     maxRunUsageTotal(highWater.candidateTokens, payload.TotalCandidateTokens),
			ToolTokens:          maxRunUsageTotal(highWater.toolTokens, payload.TotalToolTokens),
			TotalTokens:         maxRunUsageTotal(highWater.totalTokens, payload.TotalTokens),
			CostAmount:          cloneCostUSD(payload.CostUSD),
			ModelContextWindow:  cloneInt64Pointer(payload.ModelContextWindow),
		},
		Delta: delta,
	})
	if err != nil {
		s.logger.Warn("record project conversation usage failed", "conversation_id", live.principal.ConversationID, "run_id", run.ID, "error", err)
		return run
	}

	highWater.inputTokens = maxRunUsageTotal(highWater.inputTokens, payload.TotalInputTokens)
	highWater.outputTokens = maxRunUsageTotal(highWater.outputTokens, payload.TotalOutputTokens)
	highWater.cachedInputTokens = maxRunUsageTotal(highWater.cachedInputTokens, payload.TotalCachedInputTokens)
	highWater.cacheCreationTokens = maxRunUsageTotal(highWater.cacheCreationTokens, payload.TotalCacheCreationTokens)
	highWater.reasoningTokens = maxRunUsageTotal(highWater.reasoningTokens, payload.TotalReasoningTokens)
	highWater.promptTokens = maxRunUsageTotal(highWater.promptTokens, payload.TotalPromptTokens)
	highWater.candidateTokens = maxRunUsageTotal(highWater.candidateTokens, payload.TotalCandidateTokens)
	highWater.toolTokens = maxRunUsageTotal(highWater.toolTokens, payload.TotalToolTokens)
	highWater.totalTokens = maxRunUsageTotal(highWater.totalTokens, payload.TotalTokens)
	if payload.CostUSD != nil && (!highWater.hasCostAmount || *payload.CostUSD > highWater.costAmount) {
		highWater.costAmount = *payload.CostUSD
		highWater.hasCostAmount = true
	}

	return updatedRun
}

func (s *ProjectConversationService) recordConversationProviderRateLimit(
	ctx context.Context,
	live *liveProjectConversation,
	payload runtimeRateLimitPayload,
) {
	if s == nil || s.runtimeStore == nil || live == nil || live.provider.ID == uuid.Nil || payload.RateLimit == nil {
		return
	}

	rateLimitPayload, err := marshalProjectConversationRateLimitPayload(payload.RateLimit)
	if err != nil {
		s.logger.Warn("marshal project conversation provider rate limit failed", "conversation_id", live.principal.ConversationID, "provider_id", live.provider.ID, "error", err)
		return
	}
	if err := s.runtimeStore.UpdateProviderRateLimit(ctx, domain.UpdateProviderRateLimitInput{
		ProjectID:        live.principal.ProjectID,
		ProviderID:       live.provider.ID,
		ObservedAt:       payload.ObservedAt.UTC(),
		RateLimitPayload: rateLimitPayload,
	}); err != nil {
		s.logger.Warn("update project conversation provider rate limit failed", "conversation_id", live.principal.ConversationID, "provider_id", live.provider.ID, "error", err)
	}
}

func marshalProjectConversationRateLimitPayload(rateLimit *provider.CLIRateLimit) (map[string]any, error) {
	if rateLimit == nil {
		return nil, nil
	}

	payload, err := json.Marshal(rateLimit)
	if err != nil {
		return nil, err
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func clampUsageDelta(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func maxRunUsageTotal(left int64, right int64) int64 {
	if right > left {
		return right
	}
	return left
}

func mapConversationTracePayload(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMapAny(typed)
	case nil:
		return map[string]any{}
	default:
		return map[string]any{"value": fmt.Sprintf("%v", typed)}
	}
}

func extractConversationTraceText(payload map[string]any) string {
	for _, key := range []string{"content", "message", "summary", "detail"} {
		if trimmed := strings.TrimSpace(stringValue(payload[key])); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s *ProjectConversationService) buildConversationRuntimeInput(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	workspace provider.AbsolutePath,
	systemPrompt string,
	resumeThreadID string,
	resumeTurnID string,
	focus *ProjectConversationFocus,
) (RuntimeTurnInput, error) {
	environment, err := s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem, focus)
	if err != nil {
		return RuntimeTurnInput{}, err
	}
	return RuntimeTurnInput{
		SessionID:              SessionID(conversation.ID.String()),
		ProjectID:              project.ID,
		TicketID:               conversationFocusTicketID(focus),
		Provider:               providerItem,
		Message:                "",
		SystemPrompt:           systemPrompt,
		WorkingDirectory:       workspace,
		Environment:            environment,
		ResumeProviderThreadID: strings.TrimSpace(resumeThreadID),
		ResumeProviderTurnID:   strings.TrimSpace(resumeTurnID),
		MaxTurns:               0,
		MaxBudgetUSD:           0,
		PersistentConversation: true,
	}, nil
}

func conversationFocusTicketID(focus *ProjectConversationFocus) *uuid.UUID {
	if focus == nil || focus.Ticket == nil || focus.Ticket.ID == uuid.Nil {
		return nil
	}
	copied := focus.Ticket.ID
	return &copied
}

func conversationAnchorsFromRuntimeAnchor(anchor RuntimeSessionAnchor, rollingSummary string) domain.ConversationAnchors {
	activeFlags := append([]string(nil), anchor.ProviderThreadActiveFlags...)
	return domain.ConversationAnchors{
		ProviderThreadID:          optionalString(anchor.ProviderThreadID),
		LastTurnID:                optionalString(anchor.LastTurnID),
		ProviderThreadStatus:      optionalString(anchor.ProviderThreadStatus),
		ProviderThreadActiveFlags: &activeFlags,
		RollingSummary:            rollingSummary,
	}
}

func liveRuntimeSessionAnchor(live *liveProjectConversation, sessionID SessionID) RuntimeSessionAnchor {
	if live == nil {
		return RuntimeSessionAnchor{}
	}
	if live.runtime != nil {
		if anchorer, ok := live.runtime.(projectConversationSessionAnchorer); ok {
			return anchorer.SessionAnchor(sessionID)
		}
	}
	if live.codex != nil {
		return live.codex.SessionAnchor(sessionID)
	}
	return RuntimeSessionAnchor{}
}

func (s *ProjectConversationService) autoReleaseCompletedRuntime(
	ctx context.Context,
	conversation domain.Conversation,
	live *liveProjectConversation,
) {
	if s == nil || s.runtimeManager == nil {
		return
	}

	closedLive, _ := s.runtimeManager.Close(conversation.ID)
	if closedLive == nil {
		closedLive = live
	}

	if s.runtimeStore != nil {
		principalID := uuid.Nil
		if closedLive != nil && closedLive.principal.ID != uuid.Nil {
			principalID = closedLive.principal.ID
		} else if principal, err := s.runtimeStore.GetPrincipal(ctx, conversation.ID); err == nil {
			principalID = principal.ID
		}
		if principalID != uuid.Nil {
			if _, err := s.runtimeStore.ClosePrincipal(ctx, domain.ClosePrincipalInput{PrincipalID: principalID}); err != nil {
				s.logger.Warn("close completed project conversation principal failed", "conversation_id", conversation.ID, "principal_id", principalID, "error", err)
			}
		}
	}

	emptyFlags := []string{}
	updatedConversation, err := s.conversations.UpdateConversationAnchors(
		ctx,
		conversation.ID,
		domain.ConversationStatusActive,
		domain.ConversationAnchors{
			ProviderThreadID:          conversation.ProviderThreadID,
			LastTurnID:                conversation.LastTurnID,
			ProviderThreadStatus:      optionalString("notLoaded"),
			ProviderThreadActiveFlags: &emptyFlags,
			RollingSummary:            conversation.RollingSummary,
		},
	)
	if err != nil {
		s.logger.Warn("persist completed project conversation auto-release failed", "conversation_id", conversation.ID, "error", err)
		return
	}

	var providerItem *catalogdomain.AgentProvider
	switch {
	case closedLive != nil:
		providerItem = &closedLive.provider
	case live != nil:
		providerItem = &live.provider
	case s.catalog != nil:
		if resolved, resolveErr := s.catalog.GetAgentProvider(ctx, updatedConversation.ProviderID); resolveErr == nil {
			providerItem = &resolved
		}
	}

	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event:   "session",
		Payload: conversationSessionPayload(conversation.ID, string(domain.RuntimeStateInactive), updatedConversation, providerItem),
	})
}

func (s *ProjectConversationService) watchConversationRuntimeState(
	ctx context.Context,
	conversationID uuid.UUID,
	live *liveProjectConversation,
	hasLive bool,
) string {
	if live != nil {
		if state := strings.TrimSpace(string(live.principal.RuntimeState)); state != "" {
			return state
		}
	}
	if s != nil && s.runtimeStore != nil {
		if principal, err := s.runtimeStore.GetPrincipal(ctx, conversationID); err == nil {
			if state := strings.TrimSpace(string(principal.RuntimeState)); state != "" {
				return state
			}
		}
	}
	if hasLive {
		return string(domain.RuntimeStateReady)
	}
	return string(domain.RuntimeStateInactive)
}

func conversationSessionPayload(
	conversationID uuid.UUID,
	runtimeState string,
	conversation domain.Conversation,
	providerItem *catalogdomain.AgentProvider,
) map[string]any {
	payload := map[string]any{
		"conversation_id": conversationID.String(),
		"runtime_state":   strings.TrimSpace(runtimeState),
	}
	mergeConversationSessionPayload(payload, conversation, providerItem)
	return payload
}

func mergeConversationSessionPayload(
	payload map[string]any,
	conversation domain.Conversation,
	providerItem *catalogdomain.AgentProvider,
) {
	if payload == nil {
		return
	}
	if providerItem != nil {
		switch providerItem.AdapterType {
		case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
			payload["provider_anchor_kind"] = "thread"
			payload["provider_turn_supported"] = true
		case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
			payload["provider_anchor_kind"] = "session"
			payload["provider_turn_supported"] = false
		}
	}
	if title := conversation.Title.String(); title != "" {
		payload["title"] = title
	}
	if summary := strings.TrimSpace(conversation.RollingSummary); summary != "" {
		payload["rolling_summary"] = summary
	}
	if conversation.ProviderThreadID != nil && strings.TrimSpace(*conversation.ProviderThreadID) != "" {
		anchorID := strings.TrimSpace(*conversation.ProviderThreadID)
		payload["provider_thread_id"] = anchorID
		payload["provider_anchor_id"] = anchorID
	}
	if conversation.LastTurnID != nil && strings.TrimSpace(*conversation.LastTurnID) != "" {
		turnID := strings.TrimSpace(*conversation.LastTurnID)
		payload["last_turn_id"] = turnID
		payload["provider_turn_id"] = turnID
	}
	if conversation.ProviderThreadStatus != nil && strings.TrimSpace(*conversation.ProviderThreadStatus) != "" {
		status := strings.TrimSpace(*conversation.ProviderThreadStatus)
		payload["provider_thread_status"] = status
		payload["provider_status"] = status
	}
	if len(conversation.ProviderThreadActiveFlags) > 0 {
		flags := append([]string(nil), conversation.ProviderThreadActiveFlags...)
		payload["provider_thread_active_flags"] = flags
		payload["provider_active_flags"] = append([]string(nil), flags...)
	}
}

func (s *ProjectConversationService) handleConversationMessage(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID uuid.UUID,
	payload any,
) (StreamEvent, bool) {
	switch typed := payload.(type) {
	case textPayload:
		_, _ = s.entries.AppendEntry(ctx, conversationID, &turnID, domain.EntryKindAssistantTextDelta, map[string]any{
			"role":    "assistant",
			"content": typed.Content,
		})
		return StreamEvent{
			Event: "message",
			Payload: map[string]any{
				"type":    chatMessageTypeText,
				"content": typed.Content,
			},
		}, true
	case map[string]any:
		kind := domain.EntryKindSystem
		switch typed["type"] {
		case chatMessageTypeDiff:
			kind = domain.EntryKindDiff
		case chatMessageTypeTaskStarted, chatMessageTypeTaskNotification, chatMessageTypeTaskProgress:
			kind = domain.EntryKindSystem
		}
		entry, _ := s.entries.AppendEntry(ctx, conversationID, &turnID, kind, cloneMapAny(typed))
		normalized := cloneMapAny(typed)
		if kind == domain.EntryKindDiff {
			normalized["entry_id"] = entry.ID.String()
		}
		return StreamEvent{Event: "message", Payload: normalized}, true
	}
	return StreamEvent{}, false
}

func (s *ProjectConversationService) ensureLiveRuntime(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
) (*liveProjectConversation, bool, error) {
	s.runtimeManager.newCodexRuntime = s.newCodexRuntime
	s.runtimeManager.ConfigureGitHubCredentials(s.githubAuth)
	s.runtimeManager.ConfigureSkillSync(s.skillSync)
	return s.runtimeManager.ensureLiveRuntime(ctx, conversation, project, providerItem)
}

func (s *ProjectConversationService) ensureConversationWorkspace(
	ctx context.Context,
	machine catalogdomain.Machine,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	conversationID uuid.UUID,
) (provider.AbsolutePath, error) {
	s.runtimeManager.ConfigureGitHubCredentials(s.githubAuth)
	s.runtimeManager.ConfigureSkillSync(s.skillSync)
	return s.runtimeManager.ensureConversationWorkspace(ctx, machine, project, providerItem, conversationID)
}

func (s *ProjectConversationService) buildProjectConversationPrompt(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
	includeRecovery bool,
) (string, error) {
	return s.buildProjectConversationPromptWithDraft(
		ctx,
		conversation,
		project,
		focus,
		nil,
		includeRecovery,
	)
}

func (s *ProjectConversationService) buildProjectConversationPromptWithDraft(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
	workspaceFileDraft *ProjectConversationWorkspaceFileDraftContext,
	includeRecovery bool,
) (string, error) {
	basePrompt, err := s.promptBuilder.buildSystemPrompt(ctx, StartInput{
		Message: "",
		Source:  SourceProjectSidebar,
		Context: Context{
			ProjectID:    project.ID,
			ProjectFocus: focus,
		},
	}, project)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(basePrompt)
	platformContract := agentplatform.BuildCapabilityContract(
		s.projectConversationPlatformContractInput(
			conversation,
			project,
			focus,
			"<runtime-injected>",
			projectConversationPlatformAccessAllowed(project),
		),
	)
	if strings.TrimSpace(platformContract) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(platformContract)
	}
	if ticketFocus := focusTicket(focus); ticketFocus != nil {
		ticketCapsule, capsuleErr := s.renderProjectConversationTicketCapsule(ctx, project, ticketFocus)
		if capsuleErr != nil {
			return "", capsuleErr
		}
		if strings.TrimSpace(ticketCapsule) != "" {
			builder.WriteString("\n\n")
			builder.WriteString(ticketCapsule)
		}
	}
	if workspaceDraftPrompt := renderProjectConversationWorkspaceFileDraftContext(workspaceFileDraft); strings.TrimSpace(workspaceDraftPrompt) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(workspaceDraftPrompt)
	}
	if !includeRecovery {
		return builder.String(), nil
	}

	entries, err := s.entries.ListEntries(ctx, conversation.ID)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 && strings.TrimSpace(conversation.RollingSummary) == "" {
		return builder.String(), nil
	}

	builder.WriteString("\n\n## Previous conversation\n")
	if strings.TrimSpace(conversation.RollingSummary) != "" {
		builder.WriteString("Rolling summary:\n")
		builder.WriteString(strings.TrimSpace(conversation.RollingSummary))
		builder.WriteString("\n\n")
	}
	builder.WriteString("Recent transcript:\n")
	for _, line := range renderRecoveryLines(entries, 12) {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	builder.WriteString("\nContinue from this conversation state without restating the entire history.")
	return builder.String(), nil
}

func renderProjectConversationWorkspaceFileDraftContext(
	draft *ProjectConversationWorkspaceFileDraftContext,
) string {
	if draft == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Active workspace file draft\n")
	builder.WriteString("This draft is request-scoped context only. It has not been saved to the workspace yet.\n")
	_, _ = fmt.Fprintf(&builder, "- repo_path: %s\n", draft.RepoPath)
	_, _ = fmt.Fprintf(&builder, "- path: %s\n", draft.Path)
	_, _ = fmt.Fprintf(&builder, "- encoding: %s\n", draft.Encoding)
	_, _ = fmt.Fprintf(&builder, "- line_ending: %s\n", draft.LineEnding)
	builder.WriteString("\nDraft content:\n```text\n")
	builder.WriteString(draft.Content.String())
	if !strings.HasSuffix(draft.Content.String(), "\n") {
		builder.WriteByte('\n')
	}
	builder.WriteString("```\n")
	return builder.String()
}

func (s *ProjectConversationService) projectConversationPlatformContractInput(
	conversation domain.Conversation,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
	token string,
	scopes []string,
) agentplatform.RuntimeContractInput {
	input := agentplatform.RuntimeContractInput{
		PrincipalKind:  agentplatform.PrincipalKindProjectConversation,
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		APIURL:         s.platformAPIURL,
		Token:          token,
		Scopes:         scopes,
	}
	if ticketFocus := focusTicket(focus); ticketFocus != nil {
		input.TicketID = ticketFocus.ID
	}
	return input
}

func projectConversationPlatformAccessAllowed(project catalogdomain.Project) []string {
	if len(project.ProjectAIPlatformAccessAllowed) > 0 {
		return append([]string(nil), project.ProjectAIPlatformAccessAllowed...)
	}
	return agentplatform.SupportedScopesForPrincipalKind(agentplatform.PrincipalKindProjectConversation)
}

func (s *ProjectConversationService) broadcastConversationEvent(
	conversation domain.Conversation,
	event StreamEvent,
) {
	s.streamBroker.Broadcast(conversation.ID, event)
	if conversation.ID == uuid.Nil || conversation.ProjectID == uuid.Nil || strings.TrimSpace(conversation.UserID) == "" {
		return
	}
	s.muxBroker.Broadcast(
		projectConversationMuxWatchKey{
			ProjectID: conversation.ProjectID,
			UserID:    UserID(conversation.UserID),
		},
		newProjectConversationMuxEvent(conversation, event),
	)
}

func (s *ProjectConversationService) broadcast(conversationID uuid.UUID, event StreamEvent) {
	if s == nil || s.conversations == nil {
		return
	}
	conversation, err := s.conversations.GetConversation(context.Background(), conversationID)
	if err != nil {
		return
	}
	s.broadcastConversationEvent(conversation, event)
}

func newProjectConversationMuxEvent(
	conversation domain.Conversation,
	event StreamEvent,
) ProjectConversationMuxEvent {
	return ProjectConversationMuxEvent{
		Event:          event.Event,
		ConversationID: conversation.ID,
		Payload:        event.Payload,
		SentAt:         time.Now().UTC(),
	}
}

func runtimeInterruptKind(kind domain.InterruptKind) string {
	switch kind {
	case domain.InterruptKindCommandExecutionApproval:
		return "command_execution"
	case domain.InterruptKindFileChangeApproval:
		return "file_change"
	default:
		return "user_input"
	}
}

func mapDomainInterruptKind(kind string) domain.InterruptKind {
	switch strings.TrimSpace(kind) {
	case codexadapterApprovalKindCommandExecution():
		return domain.InterruptKindCommandExecutionApproval
	case codexadapterApprovalKindFileChange():
		return domain.InterruptKindFileChangeApproval
	default:
		return domain.InterruptKindUserInput
	}
}

func codexadapterApprovalKindCommandExecution() string { return "command_execution" }
func codexadapterApprovalKindFileChange() string       { return "file_change" }

func providerInterruptProviderName(providerItem catalogdomain.AgentProvider) string {
	switch providerItem.AdapterType {
	case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
		return "claude"
	case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
		return "codex"
	case catalogdomain.AgentProviderAdapterTypeGeminiCLI:
		return "gemini"
	default:
		if name := strings.TrimSpace(providerItem.Name); name != "" {
			return name
		}
		if command := strings.TrimSpace(providerItem.CliCommand); command != "" {
			return command
		}
		if adapterType := strings.TrimSpace(string(providerItem.AdapterType)); adapterType != "" {
			return adapterType
		}
		return "provider"
	}
}

func renderRecoveryLines(entries []domain.Entry, limit int) []string {
	if len(entries) == 0 {
		return nil
	}
	if limit <= 0 || len(entries) < limit {
		limit = len(entries)
	}
	selected := entries[len(entries)-limit:]
	lines := make([]string, 0, len(selected))
	for _, entry := range selected {
		switch entry.Kind {
		case domain.EntryKindUserMessage:
			lines = append(lines, "user: "+strings.TrimSpace(stringValue(entry.Payload["content"])))
		case domain.EntryKindAssistantTextDelta:
			lines = append(lines, "assistant: "+strings.TrimSpace(stringValue(entry.Payload["content"])))
		case domain.EntryKindInterrupt:
			lines = append(lines, "system: turn paused for interrupt")
		case domain.EntryKindSystem:
			if stringValue(entry.Payload["type"]) == "turn_interrupted" {
				message := strings.TrimSpace(stringValue(entry.Payload["message"]))
				if message == "" {
					message = "Turn stopped by user."
				}
				lines = append(lines, "system: "+message)
			}
		}
	}
	return lines
}

func buildRollingSummary(entries []domain.Entry) string {
	lines := renderRecoveryLines(entries, 10)
	summary := strings.TrimSpace(strings.Join(lines, "\n"))
	if len(summary) > 4000 {
		return summary[len(summary)-4000:]
	}
	return summary
}

func optionalNonEmptyString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func uuidPointer(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	return &value
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneMapAny(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

func (s *ProjectConversationService) buildConversationRuntimeEnvironment(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	focus *ProjectConversationFocus,
) ([]string, error) {
	environment := make([]string, 0, 6)
	if providerItem.MachineHost == "" || providerItem.MachineHost == catalogdomain.LocalMachineHost {
		if executable, err := os.Executable(); err == nil && strings.TrimSpace(executable) != "" {
			environment = append(environment, "OPENASE_REAL_BIN="+executable)
		}
	}

	if s != nil && s.agentPlatform != nil && s.catalog != nil && strings.TrimSpace(s.platformAPIURL) != "" {
		principal, err := s.runtimeStore.EnsurePrincipal(ctx, domain.EnsurePrincipalInput{
			ConversationID: conversation.ID,
			ProjectID:      conversation.ProjectID,
			ProviderID:     conversation.ProviderID,
			Name:           projectConversationPrincipalName(conversation.ID),
		})
		if err != nil {
			s.logger.Warn("ensure project conversation principal failed", "conversation_id", conversation.ID, "error", err)
		} else {
			scopes := projectConversationPlatformAccessAllowed(project)
			issued, issueErr := s.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
				PrincipalKind:  agentplatform.PrincipalKindProjectConversation,
				PrincipalID:    principal.ID,
				PrincipalName:  principal.Name,
				ProjectID:      project.ID,
				ConversationID: conversation.ID,
				Scopes:         scopes,
			})
			if issueErr != nil {
				s.logger.Warn("issue project conversation platform token failed", "conversation_id", conversation.ID, "error", issueErr)
			} else {
				contractScopes := issued.Scopes
				if len(contractScopes) == 0 {
					contractScopes = scopes
				}
				environment = append(environment, agentplatform.BuildRuntimeEnvironment(
					s.projectConversationPlatformContractInput(conversation, project, focus, issued.Token, contractScopes),
				)...)
			}
		}
	}

	runtimeSecrets, err := s.buildConversationSecretEnvironment(ctx, project, focus)
	if err != nil {
		return nil, err
	}
	environment = append(environment, runtimeSecrets...)
	return environment, nil
}

func (s *ProjectConversationService) buildConversationSecretEnvironment(
	ctx context.Context,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
) ([]string, error) {
	if s == nil || s.secretManager == nil {
		return nil, nil
	}

	var ticketID *uuid.UUID
	if focus != nil && focus.Kind == ProjectConversationFocusTicket && focus.Ticket != nil {
		ticketID = uuidPointer(focus.Ticket.ID)
	}

	resolved, err := s.secretManager.ResolveBoundForRuntime(ctx, secretsservice.ResolveBoundRuntimeInput{
		ProjectID: project.ID,
		TicketID:  ticketID,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve conversation secret bindings: %w", err)
	}
	environment, err := secretsservice.BuildRuntimeEnvironment(resolved)
	if err != nil {
		return nil, fmt.Errorf("build conversation secret environment: %w", err)
	}
	return environment, nil
}

func conversationWorkspaceArtifactPaths(workspaceRoot string, adapterType string) []string {
	paths := []string{}
	for _, relative := range []string{".openase", skillTargetRelativePath(workspaceRoot, adapterType)} {
		if strings.TrimSpace(relative) == "" {
			continue
		}
		absolute := filepath.Join(workspaceRoot, relative)
		if _, err := os.Stat(absolute); err == nil {
			paths = append(paths, relative)
		}
	}
	return paths
}

func skillTargetRelativePath(workspaceRoot string, adapterType string) string {
	target, err := workflowservice.ResolveSkillTargetForRuntime(workspaceRoot, adapterType)
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(workspaceRoot, target.SkillsDir)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(relative)
}

func writeConversationWorkspaceArchive(
	writer *tar.Writer,
	root string,
	relativePaths []string,
) (returnErr error) {
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("open workspace root: %w", err)
	}
	defer func() {
		if closeErr := rootFS.Close(); returnErr == nil && closeErr != nil {
			returnErr = fmt.Errorf("close workspace root: %w", closeErr)
		}
	}()
	for _, relative := range relativePaths {
		absolute := filepath.Join(root, relative)
		if _, err := os.Stat(absolute); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat workspace artifact %s: %w", relative, err)
		}
		if err := filepath.Walk(absolute, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info == nil {
				var err error
				info, err = os.Lstat(path)
				if err != nil {
					return err
				}
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			relativeName, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(relativeName)
			if info.IsDir() {
				header.Name += "/"
			}
			if err := writer.WriteHeader(header); err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			file, err := rootFS.Open(relativeName)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(writer, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}); err != nil {
			return fmt.Errorf("archive workspace artifact %s: %w", relative, err)
		}
	}
	return nil
}

func mapConversationWorkspaceRepos(items []catalogdomain.ProjectRepo) []workspaceinfra.RepoInput {
	repos := make([]workspaceinfra.RepoInput, 0, len(items))
	for _, item := range items {
		repo := workspaceinfra.RepoInput{
			Name:          item.Name,
			RepositoryURL: item.RepositoryURL,
			DefaultBranch: item.DefaultBranch,
		}
		if branchName := strings.TrimSpace(item.DefaultBranch); branchName != "" {
			value := branchName
			repo.BranchName = &value
		}
		if workspaceDirname := strings.TrimSpace(item.WorkspaceDirname); workspaceDirname != "" && workspaceDirname != strings.TrimSpace(item.Name) {
			value := workspaceDirname
			repo.WorkspaceDirname = &value
		}
		repos = append(repos, repo)
	}
	return repos
}

func (s *ProjectConversationService) applyGitHubWorkspaceAuth(
	ctx context.Context,
	projectID uuid.UUID,
	request workspaceinfra.SetupRequest,
) (workspaceinfra.SetupRequest, error) {
	if s == nil {
		return request, nil
	}
	return githubauthservice.ApplyWorkspaceAuth(ctx, s.githubAuth, projectID, request)
}

func projectConversationPrincipalName(conversationID uuid.UUID) string {
	return "project-conversation:" + strings.TrimSpace(conversationID.String())
}

func projectConversationWorkspaceName(conversationID uuid.UUID) string {
	return "conv-" + strings.TrimSpace(conversationID.String())
}

func newCodexAdapterForManager(manager provider.AgentCLIProcessManager) (*codexadapter.Adapter, error) {
	return codexadapter.NewAdapter(codexadapter.AdapterOptions{ProcessManager: manager})
}

func newClaudeAdapterForManager(manager provider.AgentCLIProcessManager) provider.ClaudeCodeAdapter {
	return claudecodeadapter.NewAdapter(manager)
}
