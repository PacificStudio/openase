package chat

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	claudecodeadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

var (
	ErrConversationNotFound      = chatrepo.ErrNotFound
	ErrConversationConflict      = chatrepo.ErrConflict
	ErrConversationTurnActive    = chatrepo.ErrTurnAlreadyActive
	ErrPendingInterruptNotFound  = chatrepo.ErrNotFound
	ErrConversationRuntimeAbsent = fmt.Errorf("chat conversation runtime is unavailable")
)

type projectConversationCatalog interface {
	GetProject(ctx context.Context, id uuid.UUID) (catalogdomain.Project, error)
	GetMachine(ctx context.Context, id uuid.UUID) (catalogdomain.Machine, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (catalogdomain.AgentProvider, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.AgentProvider, error)
	ListAgents(ctx context.Context, projectID uuid.UUID) ([]catalogdomain.Agent, error)
	ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]catalogdomain.ProjectRepo, error)
	ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]catalogdomain.TicketRepoScope, error)
	ListActivityEvents(ctx context.Context, input catalogdomain.ListActivityEvents) ([]catalogdomain.ActivityEvent, error)
	CreateAgent(ctx context.Context, input catalogdomain.CreateAgent) (catalogdomain.Agent, error)
}

type projectConversationSkillSync interface {
	RefreshSkills(ctx context.Context, input workflowservice.RefreshSkillsInput) (workflowservice.RefreshSkillsResult, error)
}

type projectConversationAgentPlatform interface {
	IssueToken(ctx context.Context, input agentplatform.IssueInput) (agentplatform.IssuedToken, error)
}

type projectConversationRepository interface {
	CreateConversation(ctx context.Context, input domain.CreateConversation) (domain.Conversation, error)
	ListConversations(ctx context.Context, filter domain.ListConversationsFilter) ([]domain.Conversation, error)
	GetConversation(ctx context.Context, id uuid.UUID) (domain.Conversation, error)
	CreateTurnWithUserEntry(ctx context.Context, conversationID uuid.UUID, message string) (domain.Turn, domain.Entry, error)
	AppendEntry(ctx context.Context, conversationID uuid.UUID, turnID *uuid.UUID, kind domain.EntryKind, payload map[string]any) (domain.Entry, error)
	ListEntries(ctx context.Context, conversationID uuid.UUID) ([]domain.Entry, error)
	CreatePendingInterrupt(ctx context.Context, conversationID uuid.UUID, turnID uuid.UUID, providerRequestID string, kind domain.InterruptKind, payload map[string]any) (domain.PendingInterrupt, domain.Entry, error)
	GetPendingInterrupt(ctx context.Context, interruptID uuid.UUID) (domain.PendingInterrupt, error)
	ListPendingInterrupts(ctx context.Context, conversationID uuid.UUID) ([]domain.PendingInterrupt, error)
	ResolvePendingInterrupt(ctx context.Context, interruptID uuid.UUID, response domain.InterruptResponse) (domain.PendingInterrupt, domain.Entry, error)
	CompleteTurn(ctx context.Context, turnID uuid.UUID, status domain.TurnStatus, providerTurnID *string) (domain.Turn, error)
	UpdateConversationAnchors(ctx context.Context, conversationID uuid.UUID, status domain.ConversationStatus, anchors domain.ConversationAnchors) (domain.Conversation, error)
	CloseConversationRuntime(ctx context.Context, conversationID uuid.UUID) (domain.Conversation, error)
}

type liveProjectConversation struct {
	provider  catalogdomain.AgentProvider
	machine   catalogdomain.Machine
	runtime   Runtime
	codex     projectConversationCodexRuntime
	interrupt projectConversationInterruptRuntime
	workspace provider.AbsolutePath
}

type projectConversationCodexRuntime interface {
	Runtime
	EnsureSession(ctx context.Context, input RuntimeTurnInput) error
	RespondInterrupt(ctx context.Context, input RuntimeInterruptResponseInput) (TurnStream, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationInterruptRuntime interface {
	Runtime
	RespondInterrupt(ctx context.Context, input RuntimeInterruptResponseInput) (TurnStream, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationSessionAnchorer interface {
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type ProjectConversationService struct {
	logger *slog.Logger

	repo      projectConversationRepository
	catalog   projectConversationCatalog
	tickets   ticketReader
	workflows workflowReader
	skillSync projectConversationSkillSync

	localProcessManager provider.AgentCLIProcessManager
	sshPool             *sshinfra.Pool
	platformAPIURL      string
	agentPlatform       projectConversationAgentPlatform
	githubAuth          githubauthservice.TokenResolver

	liveMu          sync.Mutex
	live            map[uuid.UUID]*liveProjectConversation
	watchers        map[uuid.UUID]map[int]chan StreamEvent
	nextWatcher     int
	turnLocks       userLockRegistry
	promptBuilder   *Service
	newCodexRuntime func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error)
}

func NewProjectConversationService(
	logger *slog.Logger,
	repo projectConversationRepository,
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
		repo:                repo,
		catalog:             catalog,
		tickets:             tickets,
		workflows:           workflows,
		localProcessManager: localProcessManager,
		sshPool:             sshPool,
		live:                map[uuid.UUID]*liveProjectConversation{},
		watchers:            map[uuid.UUID]map[int]chan StreamEvent{},
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
		return NewCodexRuntime(adapter), nil
	}
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
}

func projectConversationTurnLockKey(conversation domain.Conversation) UserID {
	return UserID("conversation:" + conversation.ID.String())
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

	return s.repo.CreateConversation(ctx, domain.CreateConversation{
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
	return s.repo.ListConversations(ctx, domain.ListConversationsFilter{
		ProjectID:  projectID,
		UserID:     userID.String(),
		Source:     &source,
		ProviderID: providerID,
	})
}

func (s *ProjectConversationService) GetConversation(ctx context.Context, userID UserID, conversationID uuid.UUID) (domain.Conversation, error) {
	conversation, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if conversation.UserID != userID.String() {
		return domain.Conversation{}, ErrConversationNotFound
	}
	return conversation, nil
}

func (s *ProjectConversationService) ListEntries(ctx context.Context, userID UserID, conversationID uuid.UUID) ([]domain.Entry, error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return s.repo.ListEntries(ctx, conversationID)
}

func (s *ProjectConversationService) WatchConversation(
	ctx context.Context,
	conversationID uuid.UUID,
) (<-chan StreamEvent, func()) {
	events := make(chan StreamEvent, 32)

	s.liveMu.Lock()
	if s.watchers[conversationID] == nil {
		s.watchers[conversationID] = map[int]chan StreamEvent{}
	}
	watcherID := s.nextWatcher
	s.nextWatcher++
	s.watchers[conversationID][watcherID] = events
	hasLive := s.live[conversationID] != nil
	s.liveMu.Unlock()

	state := "inactive"
	if hasLive {
		state = "ready"
	}
	sessionPayload := map[string]any{
		"conversation_id": conversationID.String(),
		"runtime_state":   state,
	}
	var sessionProvider *catalogdomain.AgentProvider
	s.liveMu.Lock()
	live := s.live[conversationID]
	s.liveMu.Unlock()
	if live != nil {
		sessionProvider = &live.provider
	}
	if conversation, err := s.repo.GetConversation(ctx, conversationID); err == nil {
		if sessionProvider == nil && s.catalog != nil {
			if providerItem, providerErr := s.catalog.GetAgentProvider(ctx, conversation.ProviderID); providerErr == nil {
				sessionProvider = &providerItem
			}
		}
		mergeConversationSessionPayload(sessionPayload, conversation, sessionProvider)
	}
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
	events <- StreamEvent{Event: "session", Payload: sessionPayload}

	return events, func() {
		s.liveMu.Lock()
		defer s.liveMu.Unlock()

		if watchers := s.watchers[conversationID]; watchers != nil {
			delete(watchers, watcherID)
			if len(watchers) == 0 {
				delete(s.watchers, conversationID)
			}
		}
		close(events)
	}
}

func (s *ProjectConversationService) StartTurn(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	message string,
	focus *ProjectConversationFocus,
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

	live, hadLive, err := s.ensureLiveRuntime(ctx, conversation, project, providerItem)
	if err != nil {
		return domain.Turn{}, err
	}

	includeRecovery := !hadLive
	resumeThreadID := strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID))
	resumeTurnID := strings.TrimSpace(stringPointerValue(conversation.LastTurnID))
	if !hadLive && resumeThreadID != "" {
		switch providerItem.AdapterType {
		case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
			if live.codex == nil {
				break
			}
			resumePrompt, resumePromptErr := s.buildProjectConversationPrompt(ctx, conversation, project, focus, false)
			if resumePromptErr != nil {
				return domain.Turn{}, resumePromptErr
			}
			resumeErr := live.codex.EnsureSession(ctx, s.buildConversationRuntimeInput(
				ctx,
				conversation,
				project,
				providerItem,
				live.workspace,
				resumePrompt,
				resumeThreadID,
				resumeTurnID,
			))
			switch {
			case resumeErr == nil:
				includeRecovery = false
			case codexadapter.IsThreadNotFoundError(resumeErr):
				resumeThreadID = ""
				resumeTurnID = ""
				emptyFlags := []string{}
				_, _ = s.repo.UpdateConversationAnchors(ctx, conversationID, conversation.Status, domain.ConversationAnchors{
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

	systemPrompt, err := s.buildProjectConversationPrompt(ctx, conversation, project, focus, includeRecovery)
	if err != nil {
		return domain.Turn{}, err
	}

	turn, _, err := s.repo.CreateTurnWithUserEntry(ctx, conversationID, strings.TrimSpace(message))
	if err != nil {
		return domain.Turn{}, err
	}

	stream, err := live.runtime.StartTurn(ctx, RuntimeTurnInput{
		SessionID:              SessionID(conversationID.String()),
		Provider:               providerItem,
		Message:                strings.TrimSpace(message),
		SystemPrompt:           systemPrompt,
		WorkingDirectory:       live.workspace,
		Environment:            s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem),
		ResumeProviderThreadID: resumeThreadID,
		ResumeProviderTurnID:   resumeTurnID,
		MaxTurns:               0,
		MaxBudgetUSD:           0,
		PersistentConversation: true,
	})
	if err != nil {
		return domain.Turn{}, err
	}

	go s.consumeTurn(context.WithoutCancel(ctx), conversationID, turn, live, stream)
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
	interrupt, err := s.repo.GetPendingInterrupt(ctx, interruptID)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	if interrupt.ConversationID != conversation.ID {
		return domain.PendingInterrupt{}, ErrPendingInterruptNotFound
	}
	s.liveMu.Lock()
	live := s.live[conversationID]
	s.liveMu.Unlock()
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
		var ensureErr error
		live, _, ensureErr = s.ensureLiveRuntime(ctx, conversation, project, providerItem)
		if ensureErr != nil {
			return domain.PendingInterrupt{}, ensureErr
		}
		if live.interrupt == nil {
			return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
		}
		if live.codex != nil {
			systemPrompt, promptErr := s.buildProjectConversationPrompt(ctx, conversation, project, nil, false)
			if promptErr != nil {
				return domain.PendingInterrupt{}, promptErr
			}
			ensureErr = live.codex.EnsureSession(ctx, s.buildConversationRuntimeInput(
				ctx,
				conversation,
				project,
				providerItem,
				live.workspace,
				systemPrompt,
				strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID)),
				strings.TrimSpace(stringPointerValue(conversation.LastTurnID)),
			))
			if ensureErr != nil {
				if codexadapter.IsThreadNotFoundError(ensureErr) {
					return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
				}
				return domain.PendingInterrupt{}, ensureErr
			}
		}
	}

	runtimeKind := runtimeInterruptKind(interrupt.Kind)
	stream, err := live.interrupt.RespondInterrupt(ctx, RuntimeInterruptResponseInput{
		SessionID:              SessionID(conversationID.String()),
		Provider:               live.provider,
		RequestID:              interrupt.ProviderRequestID,
		Kind:                   runtimeKind,
		Decision:               stringPointerValue(response.Decision),
		Answer:                 cloneMapAny(response.Answer),
		Payload:                cloneMapAny(interrupt.Payload),
		WorkingDirectory:       live.workspace,
		Environment:            s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem),
		ResumeProviderThreadID: strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID)),
		ResumeProviderTurnID:   strings.TrimSpace(stringPointerValue(conversation.LastTurnID)),
		PersistentConversation: true,
	})
	if err != nil {
		return domain.PendingInterrupt{}, err
	}

	resolved, _ /* entry */, err := s.repo.ResolvePendingInterrupt(ctx, interruptID, response)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	anchor := RuntimeSessionAnchor{}
	if live.interrupt != nil {
		anchor = live.interrupt.SessionAnchor(SessionID(conversationID.String()))
	}
	_, _ = s.repo.UpdateConversationAnchors(
		ctx,
		conversationID,
		domain.ConversationStatusActive,
		conversationAnchorsFromRuntimeAnchor(anchor, ""),
	)
	s.broadcast(conversationID, StreamEvent{
		Event: "interrupt_resolved",
		Payload: map[string]any{
			"interrupt_id": resolved.ID.String(),
			"decision":     stringPointerValue(resolved.Decision),
		},
	})
	if stream.Events != nil {
		go s.consumeTurn(context.WithoutCancel(ctx), conversationID, domain.Turn{
			ID:             interrupt.TurnID,
			ConversationID: conversationID,
		}, live, stream)
	}
	return resolved, nil
}

func (s *ProjectConversationService) AppendActionExecutionResult(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	turnID *uuid.UUID,
	payload map[string]any,
) (domain.Entry, error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return domain.Entry{}, err
	}
	entry, err := s.repo.AppendEntry(ctx, conversationID, turnID, domain.EntryKindActionResult, payload)
	if err != nil {
		return domain.Entry{}, err
	}
	s.broadcast(conversationID, StreamEvent{
		Event: "message",
		Payload: map[string]any{
			"type":    "action_result",
			"payload": cloneMapAny(payload),
		},
	})
	return entry, nil
}

func (s *ProjectConversationService) CloseRuntime(ctx context.Context, userID UserID, conversationID uuid.UUID) error {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return err
	}

	s.liveMu.Lock()
	live := s.live[conversationID]
	delete(s.live, conversationID)
	s.liveMu.Unlock()

	if live != nil && live.runtime != nil {
		live.runtime.CloseSession(SessionID(conversationID.String()))
	}
	conversation, err := s.repo.CloseConversationRuntime(ctx, conversationID)
	if err == nil {
		var providerItem *catalogdomain.AgentProvider
		if live != nil {
			providerItem = &live.provider
		} else if s.catalog != nil {
			if resolved, resolveErr := s.catalog.GetAgentProvider(ctx, conversation.ProviderID); resolveErr == nil {
				providerItem = &resolved
			}
		}
		s.broadcast(conversationID, StreamEvent{
			Event:   "session",
			Payload: conversationSessionPayload(conversationID, "inactive", conversation, providerItem),
		})
	}
	return err
}

func (s *ProjectConversationService) consumeTurn(
	ctx context.Context,
	conversationID uuid.UUID,
	turn domain.Turn,
	live *liveProjectConversation,
	stream TurnStream,
) {
	for event := range stream.Events {
		switch event.Event {
		case "message":
			if normalized, ok := s.handleConversationMessage(ctx, conversationID, turn.ID, event.Payload); ok {
				s.broadcast(conversationID, normalized)
				continue
			}
			s.broadcast(conversationID, event)
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
			pending, _, err := s.repo.CreatePendingInterrupt(ctx, conversationID, turn.ID, payload.RequestID, interruptKind, interruptPayload)
			if err != nil {
				s.logger.Error("persist chat interrupt", "conversation_id", conversationID, "error", err)
				continue
			}
			anchor := RuntimeSessionAnchor{}
			if live.codex != nil {
				anchor = live.codex.SessionAnchor(SessionID(conversationID.String()))
			}
			_, _ = s.repo.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusInterrupted,
				conversationAnchorsFromRuntimeAnchor(anchor, ""),
			)
			s.broadcast(conversationID, StreamEvent{
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
			anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
			_, _ = s.repo.CompleteTurn(ctx, turn.ID, domain.TurnStatusCompleted, optionalNonEmptyString(anchor.LastTurnID))
			entries, _ := s.repo.ListEntries(ctx, conversationID)
			summary := buildRollingSummary(entries)
			_, _ = s.repo.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				conversationAnchorsFromRuntimeAnchor(anchor, summary),
			)
			s.broadcast(conversationID, StreamEvent{
				Event: "turn_done",
				Payload: map[string]any{
					"conversation_id": conversationID.String(),
					"turn_id":         turn.ID.String(),
					"cost_usd":        done.CostUSD,
				},
			})
		case "thread_status":
			payload, ok := event.Payload.(runtimeThreadStatusPayload)
			if !ok {
				continue
			}
			activeFlags := append([]string(nil), payload.ActiveFlags...)
			updated, _ := s.repo.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				domain.ConversationAnchors{
					ProviderThreadStatus:      optionalString(payload.Status),
					ProviderThreadActiveFlags: &activeFlags,
				},
			)
			entry, _ := s.repo.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":         "thread_status",
				"anchor_kind":  "thread",
				"thread_id":    payload.ThreadID,
				"status":       payload.Status,
				"active_flags": append([]string(nil), payload.ActiveFlags...),
			})
			s.broadcast(conversationID, StreamEvent{
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
			s.broadcast(conversationID, StreamEvent{
				Event: "thread_status",
				Payload: map[string]any{
					"thread_id":    payload.ThreadID,
					"status":       payload.Status,
					"active_flags": append([]string(nil), payload.ActiveFlags...),
					"entry_id":     entry.ID.String(),
				},
			})
			s.broadcast(conversationID, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, "ready", updated, &live.provider),
			})
		case "session_anchor":
			anchor, ok := event.Payload.(RuntimeSessionAnchor)
			if !ok || strings.TrimSpace(anchor.ProviderThreadID) == "" {
				continue
			}
			updated, _ := s.repo.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				conversationAnchorsFromRuntimeAnchor(anchor, ""),
			)
			s.broadcast(conversationID, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, "ready", updated, &live.provider),
			})
		case "session_state":
			payload, ok := event.Payload.(runtimeSessionStatePayload)
			if !ok {
				continue
			}
			activeFlags := append([]string(nil), payload.ActiveFlags...)
			updated, _ := s.repo.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				domain.ConversationAnchors{
					ProviderThreadStatus:      optionalString(payload.Status),
					ProviderThreadActiveFlags: &activeFlags,
				},
			)
			entry, _ := s.repo.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":         "session_state",
				"anchor_kind":  "session",
				"status":       payload.Status,
				"active_flags": append([]string(nil), payload.ActiveFlags...),
				"detail":       payload.Detail,
				"raw":          cloneAnyMap(payload.Raw),
			})
			s.broadcast(conversationID, StreamEvent{
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
			s.broadcast(conversationID, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, "ready", updated, &live.provider),
			})
		case "thread_compacted":
			payload, ok := event.Payload.(runtimeThreadCompactedPayload)
			if !ok {
				continue
			}
			entry, _ := s.repo.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":      "thread_compacted",
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
			})
			s.broadcast(conversationID, StreamEvent{
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
			entry, _ := s.repo.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":        "turn_plan_updated",
				"thread_id":   payload.ThreadID,
				"turn_id":     payload.TurnID,
				"explanation": payload.Explanation,
				"plan":        rawPlan,
			})
			s.broadcast(conversationID, StreamEvent{
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
			entry, _ := s.repo.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":      "turn_diff_updated",
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
				"diff":      payload.Diff,
			})
			s.broadcast(conversationID, StreamEvent{
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
			entry, _ := s.repo.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":          "turn_reasoning_updated",
				"thread_id":     payload.ThreadID,
				"turn_id":       payload.TurnID,
				"item_id":       payload.ItemID,
				"kind":          payload.Kind,
				"delta":         payload.Delta,
				"summary_index": payload.SummaryIndex,
				"content_index": payload.ContentIndex,
			})
			s.broadcast(conversationID, StreamEvent{
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
				anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
				_, _ = s.repo.CompleteTurn(ctx, turn.ID, domain.TurnStatusFailed, optionalNonEmptyString(anchor.LastTurnID))
				_, _ = s.repo.UpdateConversationAnchors(
					ctx,
					conversationID,
					domain.ConversationStatusActive,
					conversationAnchorsFromRuntimeAnchor(anchor, ""),
				)
				s.broadcast(conversationID, StreamEvent{
					Event: "error",
					Payload: map[string]any{
						"message": payload.Message,
					},
				})
			}
		case "interrupted":
			payload, ok := event.Payload.(errorPayload)
			if ok {
				anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
				_, _ = s.repo.CompleteTurn(ctx, turn.ID, domain.TurnStatusInterrupted, optionalNonEmptyString(anchor.LastTurnID))
				_, _ = s.repo.UpdateConversationAnchors(
					ctx,
					conversationID,
					domain.ConversationStatusInterrupted,
					conversationAnchorsFromRuntimeAnchor(anchor, ""),
				)
				s.broadcast(conversationID, StreamEvent{
					Event: "interrupted",
					Payload: map[string]any{
						"message": payload.Message,
					},
				})
			}
		}
	}
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
) RuntimeTurnInput {
	return RuntimeTurnInput{
		SessionID:              SessionID(conversation.ID.String()),
		Provider:               providerItem,
		Message:                "",
		SystemPrompt:           systemPrompt,
		WorkingDirectory:       workspace,
		Environment:            s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem),
		ResumeProviderThreadID: strings.TrimSpace(resumeThreadID),
		ResumeProviderTurnID:   strings.TrimSpace(resumeTurnID),
		MaxTurns:               0,
		MaxBudgetUSD:           0,
		PersistentConversation: true,
	}
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
		_, _ = s.repo.AppendEntry(ctx, conversationID, &turnID, domain.EntryKindAssistantTextDelta, map[string]any{
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
		case chatMessageTypeActionProposal:
			kind = domain.EntryKindActionProposal
		case chatMessageTypeDiff:
			kind = domain.EntryKindDiff
		case chatMessageTypeTaskStarted, chatMessageTypeTaskNotification, chatMessageTypeTaskProgress:
			kind = domain.EntryKindSystem
		}
		entry, _ := s.repo.AppendEntry(ctx, conversationID, &turnID, kind, cloneMapAny(typed))
		normalized := cloneMapAny(typed)
		if kind == domain.EntryKindActionProposal || kind == domain.EntryKindDiff {
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
	conversationID := conversation.ID

	s.liveMu.Lock()
	if existing := s.live[conversationID]; existing != nil {
		s.liveMu.Unlock()
		return existing, true, nil
	}
	s.liveMu.Unlock()

	machine, err := s.catalog.GetMachine(ctx, providerItem.MachineID)
	if err != nil {
		return nil, false, fmt.Errorf("get chat provider machine: %w", err)
	}

	workspacePath, err := s.ensureConversationWorkspace(ctx, machine, project, providerItem, conversationID)
	if err != nil {
		return nil, false, err
	}

	manager, err := s.resolveProcessManager(machine)
	if err != nil {
		return nil, false, err
	}

	var runtime Runtime
	var codexRuntime projectConversationCodexRuntime
	var interruptRuntime projectConversationInterruptRuntime
	switch providerItem.AdapterType {
	case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
		codexRuntime, err = s.newCodexRuntime(manager)
		if err != nil {
			return nil, false, err
		}
		runtime = codexRuntime
		interruptRuntime = codexRuntime
	case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
		claudeRuntime := NewClaudeRuntime(newClaudeAdapterForManager(manager))
		runtime = claudeRuntime
		interruptRuntime = claudeRuntime
	case catalogdomain.AgentProviderAdapterTypeGeminiCLI:
		runtime = NewGeminiRuntime(manager)
	default:
		return nil, false, fmt.Errorf("%w: provider=%s", ErrProviderUnsupported, providerItem.AdapterType)
	}

	live := &liveProjectConversation{
		provider:  providerItem,
		machine:   machine,
		runtime:   runtime,
		codex:     codexRuntime,
		interrupt: interruptRuntime,
		workspace: workspacePath,
	}
	s.liveMu.Lock()
	s.live[conversationID] = live
	s.liveMu.Unlock()
	return live, false, nil
}

func (s *ProjectConversationService) buildProjectConversationPrompt(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
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
	if !includeRecovery {
		return basePrompt, nil
	}

	entries, err := s.repo.ListEntries(ctx, conversation.ID)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 && strings.TrimSpace(conversation.RollingSummary) == "" {
		return basePrompt, nil
	}

	var builder strings.Builder
	builder.WriteString(basePrompt)
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

func (s *ProjectConversationService) ensureConversationWorkspace(
	ctx context.Context,
	machine catalogdomain.Machine,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	conversationID uuid.UUID,
) (provider.AbsolutePath, error) {
	root := ""
	if machine.WorkspaceRoot != nil && strings.TrimSpace(*machine.WorkspaceRoot) != "" {
		root = strings.TrimSpace(*machine.WorkspaceRoot)
	} else if machine.Host == catalogdomain.LocalMachineHost {
		localRoot, err := workspaceinfra.LocalWorkspaceRoot()
		if err != nil {
			return "", err
		}
		root = localRoot
	}
	if root == "" {
		return "", fmt.Errorf("chat provider machine %s is missing workspace_root", machine.Name)
	}

	projectRepos, err := s.catalog.ListProjectRepos(ctx, project.ID)
	if err != nil {
		return "", fmt.Errorf("list project repos for conversation workspace: %w", err)
	}
	request, err := workspaceinfra.ParseSetupRequest(workspaceinfra.SetupInput{
		WorkspaceRoot:    root,
		OrganizationSlug: project.OrganizationID.String(),
		ProjectSlug:      project.Slug,
		AgentName:        projectConversationAgentName(conversationID),
		TicketIdentifier: projectConversationWorkspaceName(conversationID),
		Repos:            mapConversationWorkspaceRepos(projectRepos),
	})
	if err != nil {
		return "", fmt.Errorf("build project conversation workspace request: %w", err)
	}
	request, err = s.applyGitHubWorkspaceAuth(ctx, project.ID, request)
	if err != nil {
		return "", fmt.Errorf("prepare chat workspace auth: %w", err)
	}
	var workspaceItem workspaceinfra.Workspace
	if machine.Host == catalogdomain.LocalMachineHost {
		workspaceItem, err = workspaceinfra.NewManager().Prepare(ctx, request)
		if err != nil {
			return "", fmt.Errorf("prepare local chat workspace: %w", err)
		}
	} else {
		if s.sshPool == nil {
			return "", fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
		}
		workspaceItem, err = workspaceinfra.NewRemoteManager(s.sshPool).Prepare(ctx, machine, request)
		if err != nil {
			return "", fmt.Errorf("prepare remote chat workspace: %w", err)
		}
	}

	if err := s.syncConversationWorkspaceSkills(ctx, machine, project.ID, workspaceItem.Path, string(providerItem.AdapterType)); err != nil {
		return "", err
	}

	return provider.ParseAbsolutePath(filepath.Clean(workspaceItem.Path))
}

func (s *ProjectConversationService) resolveProcessManager(
	machine catalogdomain.Machine,
) (provider.AgentCLIProcessManager, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		if s.localProcessManager == nil {
			return nil, fmt.Errorf("local chat process manager unavailable")
		}
		return s.localProcessManager, nil
	}
	if s.sshPool == nil {
		return nil, fmt.Errorf("ssh process manager unavailable")
	}
	return sshinfra.NewProcessManager(s.sshPool, machine), nil
}

func (s *ProjectConversationService) broadcast(conversationID uuid.UUID, event StreamEvent) {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()

	for _, watcher := range s.watchers[conversationID] {
		select {
		case watcher <- event:
		default:
		}
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
	default:
		return strings.TrimSpace(providerItem.Name)
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
		case domain.EntryKindActionProposal:
			lines = append(lines, "assistant: proposed platform action")
		case domain.EntryKindInterrupt:
			lines = append(lines, "system: turn paused for interrupt")
		case domain.EntryKindActionResult:
			lines = append(lines, "system: action proposal executed")
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

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
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
) []string {
	environment := make([]string, 0, 6)
	if providerItem.MachineHost == "" || providerItem.MachineHost == catalogdomain.LocalMachineHost {
		if executable, err := os.Executable(); err == nil && strings.TrimSpace(executable) != "" {
			environment = append(environment, "OPENASE_REAL_BIN="+executable)
		}
	}

	if s == nil || s.agentPlatform == nil || s.catalog == nil || strings.TrimSpace(s.platformAPIURL) == "" {
		return environment
	}

	agentID, err := s.ensureConversationAgent(ctx, conversation.ProjectID, providerItem.ID, conversation.ID)
	if err != nil {
		s.logger.Warn("ensure project conversation runtime agent failed", "conversation_id", conversation.ID, "error", err)
		return environment
	}
	scopes := append(agentplatform.DefaultScopes(), agentplatform.PrivilegedScopes()...)
	scopes = slices.Compact(scopes)
	issued, err := s.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: project.ID,
		TicketID:  conversation.ID,
		Scopes:    scopes,
	})
	if err != nil {
		s.logger.Warn("issue project conversation platform token failed", "conversation_id", conversation.ID, "error", err)
		return environment
	}

	return append(environment,
		"OPENASE_API_URL="+s.platformAPIURL,
		"OPENASE_AGENT_TOKEN="+issued.Token,
		"OPENASE_PROJECT_ID="+project.ID.String(),
	)
}

func (s *ProjectConversationService) ensureConversationAgent(
	ctx context.Context,
	projectID uuid.UUID,
	providerID uuid.UUID,
	conversationID uuid.UUID,
) (uuid.UUID, error) {
	agentName := projectConversationAgentName(conversationID)
	agents, err := s.catalog.ListAgents(ctx, projectID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("list project agents for conversation runtime: %w", err)
	}
	for _, item := range agents {
		if item.Name == agentName {
			return item.ID, nil
		}
	}

	created, err := s.catalog.CreateAgent(ctx, catalogdomain.CreateAgent{
		ProjectID:           projectID,
		ProviderID:          providerID,
		Name:                agentName,
		RuntimeControlState: catalogdomain.AgentRuntimeControlStateActive,
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("create project conversation runtime agent: %w", err)
	}
	return created.ID, nil
}

func (s *ProjectConversationService) syncConversationWorkspaceSkills(
	ctx context.Context,
	machine catalogdomain.Machine,
	projectID uuid.UUID,
	workspaceRoot string,
	adapterType string,
) error {
	if s == nil || s.skillSync == nil {
		return nil
	}

	if machine.Host == catalogdomain.LocalMachineHost {
		_, err := s.skillSync.RefreshSkills(ctx, workflowservice.RefreshSkillsInput{
			ProjectID:     projectID,
			WorkspaceRoot: workspaceRoot,
			AdapterType:   adapterType,
		})
		if err != nil {
			return fmt.Errorf("refresh local project conversation skills: %w", err)
		}
		return nil
	}

	tempRoot, err := os.MkdirTemp("", "openase-project-conversation-skills-*")
	if err != nil {
		return fmt.Errorf("create temp skills workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()

	_, err = s.skillSync.RefreshSkills(ctx, workflowservice.RefreshSkillsInput{
		ProjectID:     projectID,
		WorkspaceRoot: tempRoot,
		AdapterType:   adapterType,
	})
	if err != nil {
		return fmt.Errorf("refresh remote project conversation skills snapshot: %w", err)
	}
	if err := s.copyConversationWorkspaceArtifactsRemote(ctx, machine, tempRoot, workspaceRoot, adapterType); err != nil {
		return fmt.Errorf("sync remote project conversation skills: %w", err)
	}
	return nil
}

func (s *ProjectConversationService) copyConversationWorkspaceArtifactsRemote(
	ctx context.Context,
	machine catalogdomain.Machine,
	localRoot string,
	remoteWorkspaceRoot string,
	adapterType string,
) error {
	if s == nil || s.sshPool == nil {
		return fmt.Errorf("ssh pool unavailable for remote machine %s", machine.Name)
	}

	target, err := workflowservice.ResolveSkillTargetForRuntime(remoteWorkspaceRoot, adapterType)
	if err != nil {
		return err
	}
	relativePaths := conversationWorkspaceArtifactPaths(localRoot, adapterType)

	client, err := s.sshPool.Get(ctx, machine)
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("open ssh session for project conversation skill sync: %w", err)
	}
	defer func() { _ = session.Close() }()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("open ssh stdin for project conversation skill sync: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("open ssh stderr for project conversation skill sync: %w", err)
	}
	var stderrBuffer bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stderrBuffer, stderr)
		close(stderrDone)
	}()

	command := strings.Join([]string{
		"set -eu",
		"rm -rf " + sshinfra.ShellQuote(target.SkillsDir),
		"rm -rf " + sshinfra.ShellQuote(filepath.Join(remoteWorkspaceRoot, ".openase", "bin")),
		"mkdir -p " + sshinfra.ShellQuote(remoteWorkspaceRoot),
		"tar -C " + sshinfra.ShellQuote(remoteWorkspaceRoot) + " -xf -",
	}, " && ")
	if err := session.Start(command); err != nil {
		_ = stdin.Close()
		<-stderrDone
		return fmt.Errorf("start ssh skill sync command: %w", err)
	}

	tarWriter := tar.NewWriter(stdin)
	writeErr := writeConversationWorkspaceArchive(tarWriter, localRoot, relativePaths)
	closeErr := tarWriter.Close()
	stdinCloseErr := stdin.Close()
	waitErr := session.Wait()
	<-stderrDone
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if stdinCloseErr != nil {
		return stdinCloseErr
	}
	if waitErr != nil {
		return fmt.Errorf("%w: %s", waitErr, strings.TrimSpace(stderrBuffer.String()))
	}
	return nil
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

func projectConversationAgentName(conversationID uuid.UUID) string {
	return "project-ai-" + strings.TrimSpace(conversationID.String())
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
