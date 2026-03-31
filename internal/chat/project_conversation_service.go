package chat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	claudecodeadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	"github.com/google/uuid"
)

var (
	ErrConversationNotFound      = chatrepo.ErrNotFound
	ErrConversationConflict      = chatrepo.ErrConflict
	ErrPendingInterruptNotFound  = chatrepo.ErrNotFound
	ErrConversationRuntimeAbsent = fmt.Errorf("chat conversation runtime is unavailable")
)

type projectConversationCatalog interface {
	GetProject(ctx context.Context, id uuid.UUID) (catalogdomain.Project, error)
	GetMachine(ctx context.Context, id uuid.UUID) (catalogdomain.Machine, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (catalogdomain.AgentProvider, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.AgentProvider, error)
	ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]catalogdomain.TicketRepoScope, error)
	ListActivityEvents(ctx context.Context, input catalogdomain.ListActivityEvents) ([]catalogdomain.ActivityEvent, error)
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
	UpdateConversationAnchors(ctx context.Context, conversationID uuid.UUID, status domain.ConversationStatus, providerThreadID *string, lastTurnID *string, rollingSummary string) (domain.Conversation, error)
	CloseConversationRuntime(ctx context.Context, conversationID uuid.UUID) (domain.Conversation, error)
}

type liveProjectConversation struct {
	provider  catalogdomain.AgentProvider
	machine   catalogdomain.Machine
	runtime   Runtime
	codex     projectConversationCodexRuntime
	workspace provider.AbsolutePath
}

type projectConversationCodexRuntime interface {
	Runtime
	RespondInterrupt(
		ctx context.Context,
		sessionID SessionID,
		requestID string,
		kind string,
		decision string,
		answer map[string]any,
	) error
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type ProjectConversationService struct {
	logger *slog.Logger

	repo      projectConversationRepository
	catalog   projectConversationCatalog
	tickets   ticketReader
	workflows workflowReader

	localProcessManager provider.AgentCLIProcessManager
	sshPool             *sshinfra.Pool

	liveMu        sync.Mutex
	live          map[uuid.UUID]*liveProjectConversation
	watchers      map[uuid.UUID]map[int]chan StreamEvent
	nextWatcher   int
	turnLocks     userLockRegistry
	promptBuilder *Service
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
	service.promptBuilder = &Service{
		logger:    service.logger,
		catalog:   catalog,
		tickets:   tickets,
		workflows: workflows,
	}
	return service
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
	_ context.Context,
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
	events <- StreamEvent{
		Event: "session",
		Payload: map[string]any{
			"conversation_id": conversationID.String(),
			"runtime_state":   state,
		},
	}

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
) (domain.Turn, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.Turn{}, err
	}

	unlock := s.turnLocks.Lock(UserID(conversationID.String()))
	defer unlock()

	project, err := s.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return domain.Turn{}, fmt.Errorf("get project for chat turn: %w", err)
	}
	providerItem, err := s.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return domain.Turn{}, fmt.Errorf("get provider for chat turn: %w", err)
	}

	live, hadLive, err := s.ensureLiveRuntime(ctx, conversationID, project, providerItem)
	if err != nil {
		return domain.Turn{}, err
	}

	systemPrompt, err := s.buildProjectConversationPrompt(ctx, conversation, project, !hadLive)
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
	if live == nil || live.codex == nil {
		return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
	}

	runtimeKind := runtimeInterruptKind(interrupt.Kind)
	if err := live.codex.RespondInterrupt(
		ctx,
		SessionID(conversationID.String()),
		interrupt.ProviderRequestID,
		runtimeKind,
		stringPointerValue(response.Decision),
		response.Answer,
	); err != nil {
		return domain.PendingInterrupt{}, err
	}

	resolved, _ /* entry */, err := s.repo.ResolvePendingInterrupt(ctx, interruptID, response)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	s.broadcast(conversationID, StreamEvent{
		Event: "interrupt_resolved",
		Payload: map[string]any{
			"interrupt_id": resolved.ID.String(),
			"decision":     stringPointerValue(resolved.Decision),
		},
	})
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
	_, err := s.repo.CloseConversationRuntime(ctx, conversationID)
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
			interruptPayload := map[string]any{
				"provider": "codex",
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
			s.broadcast(conversationID, StreamEvent{
				Event: "interrupt_requested",
				Payload: map[string]any{
					"interrupt_id": pending.ID.String(),
					"provider":     "codex",
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
			_, _ = s.repo.CompleteTurn(ctx, turn.ID, domain.TurnStatusCompleted, nil)
			entries, _ := s.repo.ListEntries(ctx, conversationID)
			summary := buildRollingSummary(entries)
			anchor := RuntimeSessionAnchor{}
			if live.codex != nil {
				anchor = live.codex.SessionAnchor(SessionID(conversationID.String()))
			}
			_, _ = s.repo.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				optionalNonEmptyString(anchor.ProviderThreadID),
				optionalNonEmptyString(anchor.LastTurnID),
				summary,
			)
			s.broadcast(conversationID, StreamEvent{
				Event: "turn_done",
				Payload: map[string]any{
					"conversation_id": conversationID.String(),
					"turn_id":         turn.ID.String(),
					"cost_usd":        done.CostUSD,
				},
			})
		case "error":
			payload, ok := event.Payload.(errorPayload)
			if ok {
				_, _ = s.repo.CompleteTurn(ctx, turn.ID, domain.TurnStatusFailed, nil)
				s.broadcast(conversationID, StreamEvent{
					Event: "error",
					Payload: map[string]any{
						"message": payload.Message,
					},
				})
			}
		}
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
	conversationID uuid.UUID,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
) (*liveProjectConversation, bool, error) {
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

	workspacePath, err := s.ensureConversationWorkspace(ctx, machine, project, conversationID)
	if err != nil {
		return nil, false, err
	}

	manager, err := s.resolveProcessManager(machine)
	if err != nil {
		return nil, false, err
	}

	var runtime Runtime
	var codexRuntime projectConversationCodexRuntime
	switch providerItem.AdapterType {
	case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
		adapter, err := newCodexAdapterForManager(manager)
		if err != nil {
			return nil, false, err
		}
		codexRuntime = NewCodexRuntime(adapter)
		runtime = codexRuntime
	case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
		runtime = NewClaudeRuntime(newClaudeAdapterForManager(manager))
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
	includeRecovery bool,
) (string, error) {
	basePrompt, err := s.promptBuilder.buildSystemPrompt(ctx, StartInput{
		Message: "",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: project.ID},
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

	path, err := workspaceinfra.TicketWorkspacePath(root, project.OrganizationID.String(), project.Slug, "conv-"+conversationID.String())
	if err != nil {
		return "", fmt.Errorf("derive project conversation workspace: %w", err)
	}

	if machine.Host == catalogdomain.LocalMachineHost {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return "", fmt.Errorf("create local chat workspace: %w", err)
		}
	} else {
		if s.sshPool == nil {
			return "", fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
		}
		client, err := s.sshPool.Get(ctx, machine)
		if err != nil {
			return "", err
		}
		session, err := client.NewSession()
		if err != nil {
			return "", fmt.Errorf("open ssh session for chat workspace: %w", err)
		}
		defer func() { _ = session.Close() }()
		if _, err := session.CombinedOutput("mkdir -p " + sshinfra.ShellQuote(path)); err != nil {
			return "", fmt.Errorf("create remote chat workspace: %w", err)
		}
	}

	return provider.ParseAbsolutePath(filepath.Clean(path))
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

func newCodexAdapterForManager(manager provider.AgentCLIProcessManager) (*codexadapter.Adapter, error) {
	return codexadapter.NewAdapter(codexadapter.AdapterOptions{ProcessManager: manager})
}

func newClaudeAdapterForManager(manager provider.AgentCLIProcessManager) provider.ClaudeCodeAdapter {
	return claudecodeadapter.NewAdapter(manager)
}
