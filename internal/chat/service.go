package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	ticketstatusservice "github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

const (
	DefaultMaxTurns      = 10
	DefaultMaxBudgetUSD  = 2.0
	defaultActivityLimit = 30
)

var (
	ErrUnavailable             = errors.New("chat service unavailable")
	ErrSourceUnsupported       = errors.New("chat source is unsupported")
	ErrProviderNotFound        = errors.New("chat-capable provider not found for project")
	ErrProviderUnsupported     = errors.New("chat provider does not support ephemeral chat")
	ErrProviderUnavailable     = errors.New("chat provider is unavailable")
	ErrSessionNotFound         = errors.New("chat session not found")
	ErrSessionProviderMismatch = errors.New("chat session cannot resume across providers")
	ErrSessionTurnLimitReached = errors.New("chat session reached the turn limit; please create a ticket for further work")
	ErrSessionBudgetExceeded   = errors.New("chat session reached the budget cap; please create a ticket for further work")
)

type Source string

const (
	SourceHarnessEditor  Source = "harness_editor"
	SourceSkillEditor    Source = "skill_editor"
	SourceProjectSidebar Source = "project_sidebar"
	SourceTicketDetail   Source = "ticket_detail"
)

type RawStartInput struct {
	Message    string         `json:"message"`
	Source     string         `json:"source"`
	ProviderID *string        `json:"provider_id"`
	Context    RawChatContext `json:"context"`
	SessionID  *string        `json:"session_id"`
}

type RawChatContext struct {
	ProjectID      *string `json:"project_id"`
	WorkflowID     *string `json:"workflow_id"`
	TicketID       *string `json:"ticket_id"`
	HarnessDraft   *string `json:"harness_draft"`
	SkillID        *string `json:"skill_id"`
	SkillFilePath  *string `json:"skill_file_path"`
	SkillFileDraft *string `json:"skill_file_draft"`
}

type Context struct {
	ProjectID      uuid.UUID
	WorkflowID     *uuid.UUID
	TicketID       *uuid.UUID
	HarnessDraft   *string
	SkillID        *uuid.UUID
	SkillFilePath  *string
	SkillFileDraft *string
	ProjectFocus   *ProjectConversationFocus
}

type StartInput struct {
	Message    string
	Source     Source
	ProviderID *uuid.UUID
	Context    Context
	SessionID  *SessionID
}

type StreamEvent struct {
	Event   string
	Payload any
}

type TurnStream struct {
	Events <-chan StreamEvent
}

type catalogReader interface {
	GetProject(ctx context.Context, id uuid.UUID) (catalogdomain.Project, error)
	ListActivityEvents(ctx context.Context, input catalogdomain.ListActivityEvents) ([]catalogdomain.ActivityEvent, error)
	ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]catalogdomain.ProjectRepo, error)
	ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]catalogdomain.TicketRepoScope, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.AgentProvider, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (catalogdomain.AgentProvider, error)
}

type ticketReader interface {
	Get(ctx context.Context, ticketID uuid.UUID) (ticketservice.Ticket, error)
	List(ctx context.Context, input ticketservice.ListInput) ([]ticketservice.Ticket, error)
}

type workflowReader interface {
	Get(ctx context.Context, workflowID uuid.UUID) (workflowservice.WorkflowDetail, error)
	List(ctx context.Context, projectID uuid.UUID) ([]workflowservice.Workflow, error)
	GetSkill(ctx context.Context, skillID uuid.UUID) (workflowservice.SkillDetail, error)
}

type statusReader interface {
	List(ctx context.Context, projectID uuid.UUID) (ticketstatusservice.ListResult, error)
}

type Service struct {
	logger       *slog.Logger
	runtime      Runtime
	catalog      catalogReader
	tickets      ticketReader
	workflows    workflowReader
	statuses     statusReader
	workingDir   provider.AbsolutePath
	maxTurns     int
	maxBudgetUSD float64
	sessions     sessionRegistry
	sessionStore sessionStore
	userLocks    userLockRegistry
}

type donePayload struct {
	SessionID      string   `json:"session_id"`
	CostUSD        *float64 `json:"cost_usd,omitempty"`
	TurnsUsed      int      `json:"turns_used"`
	TurnsRemaining *int     `json:"turns_remaining,omitempty"`
}

type sessionPayload struct {
	SessionID               string `json:"session_id"`
	ProviderResumeSupported bool   `json:"provider_resume_supported"`
	ResumeScope             string `json:"resume_scope"`
}

type errorPayload struct {
	Message string `json:"message"`
}

type sessionPolicy struct {
	MaxTurns     int
	MaxBudgetUSD float64
}

type textPayload struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ticketPromptContext struct {
	Ticket        ticketservice.Ticket
	RepoScopes    []catalogdomain.TicketRepoScope
	ActivityItems []catalogdomain.ActivityEvent
	HookHistory   []catalogdomain.ActivityEvent
}

func NewService(
	logger *slog.Logger,
	runtime Runtime,
	catalog catalogReader,
	tickets ticketReader,
	workflows workflowReader,
	statuses statusReader,
	workingDir provider.AbsolutePath,
) *Service {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	return &Service{
		logger:       logger.With("component", "chat-service"),
		runtime:      runtime,
		catalog:      catalog,
		tickets:      tickets,
		workflows:    workflows,
		statuses:     statuses,
		workingDir:   workingDir,
		maxTurns:     DefaultMaxTurns,
		maxBudgetUSD: DefaultMaxBudgetUSD,
	}
}

func (s *Service) EnableDurableSessions(path string) {
	if s == nil {
		return
	}
	s.sessionStore = newFileSessionStore(strings.TrimSpace(path))
}

func ParseStartInput(raw RawStartInput) (StartInput, error) {
	message := strings.TrimSpace(raw.Message)
	if message == "" {
		return StartInput{}, fmt.Errorf("message must not be empty")
	}

	source, err := parseSource(raw.Source)
	if err != nil {
		return StartInput{}, err
	}

	providerID, err := parseOptionalUUIDPointer("provider_id", raw.ProviderID)
	if err != nil {
		return StartInput{}, err
	}

	projectID, err := parseRequiredUUIDPointer("context.project_id", raw.Context.ProjectID)
	if err != nil {
		return StartInput{}, err
	}
	workflowID, err := parseOptionalUUIDPointer("context.workflow_id", raw.Context.WorkflowID)
	if err != nil {
		return StartInput{}, err
	}
	ticketID, err := parseOptionalUUIDPointer("context.ticket_id", raw.Context.TicketID)
	if err != nil {
		return StartInput{}, err
	}
	skillID, err := parseOptionalUUIDPointer("context.skill_id", raw.Context.SkillID)
	if err != nil {
		return StartInput{}, err
	}
	if err := validateSourceContext(source, workflowID, ticketID, skillID); err != nil {
		return StartInput{}, err
	}
	harnessDraft := cloneOptionalString(raw.Context.HarnessDraft)
	skillFilePath := cloneOptionalString(raw.Context.SkillFilePath)
	skillFileDraft := cloneOptionalString(raw.Context.SkillFileDraft)

	sessionID, err := parseOptionalSessionID(raw.SessionID)
	if err != nil {
		return StartInput{}, err
	}

	return StartInput{
		Message:    message,
		Source:     source,
		ProviderID: providerID,
		Context: Context{
			ProjectID:      projectID,
			WorkflowID:     workflowID,
			TicketID:       ticketID,
			HarnessDraft:   harnessDraft,
			SkillID:        skillID,
			SkillFilePath:  skillFilePath,
			SkillFileDraft: skillFileDraft,
		},
		SessionID: sessionID,
	}, nil
}

func ParseCloseSessionID(raw string) (SessionID, error) {
	return ParseSessionID(raw)
}

func (s *Service) StartTurn(ctx context.Context, userID UserID, input StartInput) (TurnStream, error) {
	if s == nil || s.runtime == nil || s.catalog == nil || s.tickets == nil || s.workflows == nil {
		return TurnStream{}, ErrUnavailable
	}

	unlockUser := s.userLocks.Lock(userID)
	defer unlockUser()

	project, err := s.catalog.GetProject(ctx, input.Context.ProjectID)
	if err != nil {
		return TurnStream{}, fmt.Errorf("get project for chat: %w", err)
	}

	existingSession, err := s.resolveExistingSession(userID, input.SessionID)
	if err != nil {
		return TurnStream{}, err
	}
	if input.SessionID == nil {
		s.closeReplacedSession(userID)
	}

	sessionID, created := s.resolveSessionID(input.SessionID)
	providerItem, err := s.resolveProvider(
		ctx,
		project,
		input.Source,
		input.ProviderID,
		existingSession,
		sessionID,
	)
	if err != nil {
		return TurnStream{}, err
	}

	systemPrompt, err := s.buildSystemPrompt(ctx, input, project)
	if err != nil {
		return TurnStream{}, err
	}

	policy := s.policyForSource(input.Source)
	if existingSession != nil {
		policy = sessionPolicy{
			MaxTurns:     existingSession.MaxTurns,
			MaxBudgetUSD: existingSession.MaxBudgetUSD,
		}
	}

	if created {
		s.sessions.Register(userID, sessionID, providerItem.ID, policy.MaxTurns, policy.MaxBudgetUSD)
		if err := s.persistSessionState(sessionID); err != nil {
			s.sessions.Delete(sessionID)
			return TurnStream{}, err
		}
	}

	stream, err := s.runtime.StartTurn(ctx, RuntimeTurnInput{
		SessionID:              sessionID,
		Provider:               providerItem,
		Message:                input.Message,
		SystemPrompt:           systemPrompt,
		WorkingDirectory:       s.workingDir,
		ResumeProviderThreadID: resumeProviderThreadID(existingSession),
		MaxTurns:               policy.MaxTurns,
		MaxBudgetUSD:           policy.MaxBudgetUSD,
	})
	if err != nil {
		if created {
			s.sessions.Delete(sessionID)
			_ = s.deletePersistedSession(sessionID)
		}
		return TurnStream{}, err
	}

	if created {
		if _, ok := s.sessions.ResolveForUser(userID, sessionID); !ok {
			s.runtime.CloseSession(sessionID)
			return TurnStream{}, ErrSessionNotFound
		}
	}

	providerResumeSupported, resumeScope := s.providerResumeContract(providerItem)

	events := make(chan StreamEvent, 1)
	go func() {
		defer close(events)

		events <- StreamEvent{
			Event: "session",
			Payload: sessionPayload{
				SessionID:               sessionID.String(),
				ProviderResumeSupported: providerResumeSupported,
				ResumeScope:             resumeScope,
			},
		}

		for event := range stream.Events {
			events <- event
			s.handleRuntimeEvent(sessionID, event)
		}
	}()

	return TurnStream{Events: events}, nil
}

func (s *Service) CloseSession(userID UserID, sessionID SessionID) bool {
	if s == nil || s.runtime == nil {
		return false
	}

	unlockUser := s.userLocks.Lock(userID)
	defer unlockUser()

	state, ok := s.sessions.ResolveForUser(userID, sessionID)
	if !ok && s.sessionStore != nil {
		persisted, persistedOK, err := s.sessionStore.LoadForUser(userID, sessionID)
		if err == nil && persistedOK {
			s.sessions.Remember(sessionID, persisted)
			state = persisted
			ok = true
		}
	}
	if !ok {
		return false
	}

	s.sessions.Delete(sessionID)
	_ = s.deletePersistedSession(sessionID)
	if state.Released {
		return true
	}
	return s.runtime.CloseSession(sessionID)
}

func (s *Service) resolveSessionID(raw *SessionID) (SessionID, bool) {
	if raw != nil {
		return *raw, false
	}

	return SessionID(uuid.NewString()), true
}

func (s *Service) resolveProvider(
	ctx context.Context,
	project catalogdomain.Project,
	source Source,
	rawProviderID *uuid.UUID,
	existingSession *sessionState,
	sessionID SessionID,
) (catalogdomain.AgentProvider, error) {
	providers, err := s.catalog.ListAgentProviders(ctx, project.OrganizationID)
	if err != nil {
		return catalogdomain.AgentProvider{}, fmt.Errorf("list project agent providers for chat: %w", err)
	}

	resolvedProviderID := rawProviderID
	if existingSession != nil {
		sessionProviderID := existingSession.ProviderID
		if resolvedProviderID != nil && *resolvedProviderID != sessionProviderID {
			return catalogdomain.AgentProvider{}, ErrSessionProviderMismatch
		}
		resolvedProviderID = &sessionProviderID
	}

	providerItem, err := resolveProviderForSurface(
		providers,
		project.DefaultAgentProviderID,
		resolvedProviderID,
		chatProviderSurfaceForSource(source),
		s.runtime.Supports,
	)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			return catalogdomain.AgentProvider{}, fmt.Errorf("%w: project=%s session=%s", ErrProviderNotFound, project.ID, sessionID)
		}
		return catalogdomain.AgentProvider{}, err
	}
	return providerItem, nil
}

func providerResolutionError(
	base error,
	providerItem catalogdomain.AgentProvider,
	capability catalogdomain.AgentProviderCapability,
) error {
	details := []string{fmt.Sprintf("provider=%s", providerItem.Name)}
	if capability.Reason != nil && strings.TrimSpace(*capability.Reason) != "" {
		details = append(details, "reason="+strings.TrimSpace(*capability.Reason))
	}
	return fmt.Errorf("%w: %s", base, strings.Join(details, " "))
}

func (s *Service) resolveExistingSession(userID UserID, rawSessionID *SessionID) (*sessionState, error) {
	if rawSessionID == nil {
		return nil, nil
	}

	state, ok := s.sessions.ResolveForUser(userID, *rawSessionID)
	if !ok && s.sessionStore != nil {
		persisted, persistedOK, err := s.sessionStore.LoadForUser(userID, *rawSessionID)
		if err != nil {
			return nil, err
		}
		if persistedOK {
			s.sessions.Remember(*rawSessionID, persisted)
			state = persisted
			ok = true
		}
	}
	if !ok {
		return nil, ErrSessionNotFound
	}
	if err := s.validateSessionBudget(state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (s *Service) validateSessionBudget(state sessionState) error {
	switch state.ExhaustedMessage {
	case ErrSessionBudgetExceeded.Error():
		return ErrSessionBudgetExceeded
	case ErrSessionTurnLimitReached.Error():
		return ErrSessionTurnLimitReached
	}
	if state.MaxTurns > 0 && state.TurnsUsed >= state.MaxTurns {
		return ErrSessionTurnLimitReached
	}
	if state.MaxBudgetUSD > 0 && state.HasCostUSD && state.CostUSD >= state.MaxBudgetUSD {
		return ErrSessionBudgetExceeded
	}

	return nil
}

func (s *Service) closeReplacedSession(userID UserID) {
	if userID == "" {
		return
	}

	previousSessionID, ok := s.sessions.ResolveUserSession(userID)
	if !ok && s.sessionStore != nil {
		persistedSessionID, persistedOK, err := s.sessionStore.ResolveUserSession(userID)
		if err == nil && persistedOK {
			previousSessionID = persistedSessionID
			ok = true
		}
	}
	if !ok {
		return
	}

	state, deleted := s.sessions.Delete(previousSessionID)
	_ = s.deletePersistedSession(previousSessionID)
	if !deleted || state.Released {
		return
	}

	s.runtime.CloseSession(previousSessionID)
}

func (s *Service) handleRuntimeEvent(sessionID SessionID, event StreamEvent) {
	if anchor, ok := event.Payload.(RuntimeSessionAnchor); event.Event == "session_anchor" && ok {
		if _, tracked := s.sessions.UpdateProviderAnchor(sessionID, anchor.ProviderThreadID, anchor.ProviderThreadStatus, anchor.ProviderThreadActiveFlags); tracked {
			_ = s.persistSessionState(sessionID)
		}
		return
	}

	if payload, ok := event.Payload.(runtimeSessionStatePayload); event.Event == "session_state" && ok {
		if _, tracked := s.sessions.UpdateProviderAnchor(sessionID, "", payload.Status, payload.ActiveFlags); tracked {
			_ = s.persistSessionState(sessionID)
		}
		return
	}

	if event.Event == "error" {
		state, ok := s.sessions.Delete(sessionID)
		_ = s.deletePersistedSession(sessionID)
		if ok && !state.Released {
			s.runtime.CloseSession(sessionID)
		}
		return
	}

	done, ok := event.Payload.(donePayload)
	if !ok {
		return
	}

	state, tracked := s.sessions.MarkUsage(sessionID, done.TurnsUsed, done.CostUSD)
	if !tracked {
		return
	}
	_ = s.persistSessionState(sessionID)

	exhaustedMessage := ""
	switch {
	case state.MaxBudgetUSD > 0 && state.HasCostUSD && state.CostUSD >= state.MaxBudgetUSD:
		exhaustedMessage = ErrSessionBudgetExceeded.Error()
	case state.MaxTurns > 0 && state.TurnsUsed >= state.MaxTurns:
		exhaustedMessage = ErrSessionTurnLimitReached.Error()
	}
	if exhaustedMessage == "" {
		return
	}

	s.sessions.MarkReleased(sessionID, exhaustedMessage)
	_ = s.persistSessionState(sessionID)
	s.runtime.CloseSession(sessionID)
}

func (s *Service) persistSessionState(sessionID SessionID) error {
	if s == nil || s.sessionStore == nil || sessionID == "" {
		return nil
	}
	state, ok := s.sessions.Resolve(sessionID)
	if !ok {
		return nil
	}
	return s.sessionStore.Save(sessionID, state)
}

func (s *Service) deletePersistedSession(sessionID SessionID) error {
	if s == nil || s.sessionStore == nil || sessionID == "" {
		return nil
	}
	return s.sessionStore.Delete(sessionID)
}

func resumeProviderThreadID(state *sessionState) string {
	if state == nil {
		return ""
	}
	return strings.TrimSpace(state.ResumeProviderThreadID)
}

func (s *Service) providerResumeContract(providerItem catalogdomain.AgentProvider) (bool, string) {
	if s == nil || s.sessionStore == nil {
		return false, "process_local"
	}

	switch providerItem.AdapterType {
	case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
		return true, "host_local"
	default:
		return false, "process_local"
	}
}

func findProvider(items []catalogdomain.AgentProvider, want uuid.UUID) (catalogdomain.AgentProvider, bool) {
	for _, item := range items {
		if item.ID == want {
			return item, true
		}
	}

	return catalogdomain.AgentProvider{}, false
}

func (s *Service) policyForSource(source Source) sessionPolicy {
	policy := sessionPolicy{
		MaxTurns:     s.maxTurns,
		MaxBudgetUSD: s.maxBudgetUSD,
	}

	if source == SourceProjectSidebar {
		policy.MaxTurns = 0
	}

	return policy
}

func (s *Service) buildSystemPrompt(
	ctx context.Context,
	input StartInput,
	project catalogdomain.Project,
) (string, error) {
	var sb strings.Builder
	sb.WriteString("你是 OpenASE 平台的内嵌 AI 助手。你正在帮助用户理解或操作 OpenASE，而不是替代编排引擎执行工单。\n\n")
	sb.WriteString("请基于下面的上下文回答。不要声称已经执行了任何平台写操作。需要实际执行平台或仓库操作时，直接使用运行时里可用的 skill、CLI 和工具完成，不要输出 `action_proposal` 或 `platform_command_proposal` 之类的结构化提案 JSON。\n\n")

	switch input.Source {
	case SourceHarnessEditor:
		if err := s.writeHarnessEditorContext(ctx, &sb, project, input); err != nil {
			return "", err
		}
	case SourceSkillEditor:
		if err := s.writeSkillEditorContext(ctx, &sb, project, input); err != nil {
			return "", err
		}
	case SourceProjectSidebar:
		if err := s.writeProjectSidebarContext(ctx, &sb, project, input.Context.ProjectFocus); err != nil {
			return "", err
		}
	case SourceTicketDetail:
		if err := s.writeTicketDetailContext(ctx, &sb, project, input); err != nil {
			return "", err
		}
	default:
		return "", ErrSourceUnsupported
	}

	if input.Source == SourceProjectSidebar {
		sb.WriteString("\n## 项目侧栏执行约束\n")
		sb.WriteString("- 需要改平台数据时，直接使用运行时可用的 skill / CLI / tool 完成，不要先生成 proposal 再等待确认。\n")
		sb.WriteString("- 优先使用 project slug/name、ticket identifier、status name 这类人类可读引用；如果对象无法唯一确定，先提一个定向澄清问题，不要猜。\n")
	}

	return sb.String(), nil
}

func (s *Service) writeHarnessEditorContext(
	ctx context.Context,
	sb *strings.Builder,
	project catalogdomain.Project,
	input StartInput,
) error {
	workflowID := uuidPtrValue(input.Context.WorkflowID)
	workflowItem, err := s.workflows.Get(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("get workflow for chat context: %w", err)
	}

	sb.WriteString("## 来源: Harness 编辑器\n")
	_, _ = fmt.Fprintf(sb, "项目: %s\n", project.Name)
	_, _ = fmt.Fprintf(sb, "Workflow: %s (%s)\n", workflowItem.Name, workflowItem.Type)
	_, _ = fmt.Fprintf(sb, "Harness Path: %s | Active: %t | Version: %d\n", workflowItem.HarnessPath, workflowItem.IsActive, workflowItem.Version)
	_, _ = fmt.Fprintf(sb, "并发: %d | 最大重试: %d | 超时: %d 分钟 | 卡住超时: %d 分钟\n\n", workflowItem.MaxConcurrent, workflowItem.MaxRetryAttempts, workflowItem.TimeoutMinutes, workflowItem.StallTimeoutMinutes)
	sb.WriteString("### 当前 Harness\n")
	sb.WriteString("```markdown\n")
	sb.WriteString(workflowItem.HarnessContent)
	if !strings.HasSuffix(workflowItem.HarnessContent, "\n") {
		sb.WriteByte('\n')
	}
	sb.WriteString("```\n\n")
	if draft := input.Context.HarnessDraft; draft != nil && *draft != workflowItem.HarnessContent {
		sb.WriteString("### 当前编辑器草稿（未保存）\n")
		if *draft == "" {
			sb.WriteString("（当前草稿为空）\n\n")
		} else {
			sb.WriteString("```markdown\n")
			sb.WriteString(*draft)
			if !strings.HasSuffix(*draft, "\n") {
				sb.WriteByte('\n')
			}
			sb.WriteString("```\n\n")
		}
	}

	statusLines, statusNamesByID, err := s.renderHarnessStatusContext(ctx, project.ID)
	if err != nil {
		return err
	}
	if statusLines != "" {
		sb.WriteString("### 项目状态拓扑\n")
		sb.WriteString(statusLines)
		sb.WriteByte('\n')
	}

	workflowLines, workflowNamesByID, err := s.renderHarnessWorkflowTopology(ctx, project.ID, workflowID, statusNamesByID)
	if err != nil {
		return err
	}
	if workflowLines != "" {
		sb.WriteString("### 项目 Workflow 拓扑\n")
		sb.WriteString(workflowLines)
		sb.WriteByte('\n')
	}

	repoLines, err := s.renderHarnessRepoContext(ctx, project.ID)
	if err != nil {
		return err
	}
	if repoLines != "" {
		sb.WriteString("### 项目仓库边界\n")
		sb.WriteString(repoLines)
		sb.WriteByte('\n')
	}

	ticketLines, err := s.renderHarnessTicketSamples(ctx, project.ID, workflowNamesByID)
	if err != nil {
		return err
	}
	if ticketLines != "" {
		sb.WriteString("### 最近工单样本\n")
		sb.WriteString(ticketLines)
		sb.WriteByte('\n')
	}

	activityItems, err := s.listRecentActivity(ctx, project.ID, nil, 15)
	if err != nil {
		return err
	}
	sb.WriteString("### 最近活动样本\n")
	sb.WriteString(renderActivityLines(activityItems))
	sb.WriteByte('\n')

	sb.WriteString("### 可用模板变量\n")
	sb.WriteString(renderHarnessVariableDictionary())
	sb.WriteByte('\n')
	sb.WriteString("\n### 专业 Workflow 设计基线\n")
	sb.WriteString("- Harness 必须准确贴合当前项目真实的状态流转，不要默认使用 `Todo -> Done`，除非上下文明确如此。\n")
	sb.WriteString("- 产物应明确职责边界、接单状态、交付状态、完成定义、repo 作用域、验证要求、失败/阻塞处理和 handoff 规则。\n")
	sb.WriteString("- 优先复用当前项目已有 workflows 的分工，避免写出和现有 lane 冲突或重复负责的 workflow。\n")
	sb.WriteString("- 如果用户要的是“专业 workflow”，默认要写成可执行 SOP，而不是泛泛而谈的角色描述。\n")
	sb.WriteString("- 不要虚构平台能力；需要平台写操作时，直接使用运行时可用工具完成，无法安全确定目标时先澄清。\n")
	sb.WriteString("\n### 先推断，缺失再澄清\n")
	sb.WriteString("在给出 harness diff 前，先基于上下文判断下面 7 项是否已经明确；缺任何关键项时，先问定向澄清问题，不要直接产出 workflow 文本：\n")
	sb.WriteString("- 1. 这个 workflow 的职责边界是什么。\n")
	sb.WriteString("- 2. 它从哪个状态接单（pickup status）。\n")
	sb.WriteString("- 3. 它把工单推进到哪个状态（finish status）。\n")
	sb.WriteString("- 4. 它的完成定义是什么，例如代码提交、PR 创建、CI 通过还是已 merge。\n")
	sb.WriteString("- 5. 它允许主动做哪些平台写操作，例如改状态、建子工单、更新 repo scope。\n")
	sb.WriteString("- 6. 它默认覆盖哪些 repo 或 repo scope。\n")
	sb.WriteString("- 7. 遇到失败、阻塞、缺信息、CI 红灯时应该怎么处理。\n")
	sb.WriteString("\n### Harness 编辑器回复要求\n")
	sb.WriteString("- 当用户请求修改 Harness 且上下文已足够时，默认只输出 1 个结构化 diff JSON 对象，供编辑器直接安全应用。\n")
	sb.WriteString("- 除非你是在提澄清问题，或你明确无法可靠地产出 diff；否则不要输出解释文字、前言、后记、markdown 列表、代码块围栏、多个 JSON 对象，或不完整 JSON 片段。\n")
	sb.WriteString("- 输出必须是单个合法 JSON object，从第一个 `{` 开始，到最后一个 `}` 结束，中间不要夹杂任何自然语言。\n")
	sb.WriteString("- 顶层字段固定为：`type`、`file`、`hunks`。不要添加额外顶层字段。\n")
	sb.WriteString("- `type` 必须恒等于 `diff`；`file` 必须恒等于 `harness content`。\n")
	sb.WriteString("- `hunks` 必须是非空数组；每个 hunk 必须同时包含 `old_start`、`old_lines`、`new_start`、`new_lines`、`lines`。\n")
	sb.WriteString("- 行号使用 1-based 正整数；`old_lines` / `new_lines` 必须与 `lines` 中 `context` / `remove` / `add` 的数量严格一致。\n")
	sb.WriteString("- `lines[].op` 只能是 `context` / `add` / `remove`；`lines[].text` 必须是单行文本，不要把换行符写进一个 `text` 值里。\n")
	sb.WriteString("- 字段名必须使用 snake_case：`old_start`、`old_lines`、`new_start`、`new_lines`。不要输出 camelCase 变体。\n")
	sb.WriteString("- JSON Schema（简化）如下：\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"type\": \"object\",\n")
	sb.WriteString("  \"required\": [\"type\", \"file\", \"hunks\"],\n")
	sb.WriteString("  \"additionalProperties\": false,\n")
	sb.WriteString("  \"properties\": {\n")
	sb.WriteString("    \"type\": {\"const\": \"diff\"},\n")
	sb.WriteString("    \"file\": {\"const\": \"harness content\"},\n")
	sb.WriteString("    \"hunks\": {\n")
	sb.WriteString("      \"type\": \"array\",\n")
	sb.WriteString("      \"minItems\": 1,\n")
	sb.WriteString("      \"items\": {\n")
	sb.WriteString("        \"type\": \"object\",\n")
	sb.WriteString("        \"required\": [\"old_start\", \"old_lines\", \"new_start\", \"new_lines\", \"lines\"],\n")
	sb.WriteString("        \"additionalProperties\": false,\n")
	sb.WriteString("        \"properties\": {\n")
	sb.WriteString("          \"old_start\": {\"type\": \"integer\", \"minimum\": 1},\n")
	sb.WriteString("          \"old_lines\": {\"type\": \"integer\", \"minimum\": 0},\n")
	sb.WriteString("          \"new_start\": {\"type\": \"integer\", \"minimum\": 1},\n")
	sb.WriteString("          \"new_lines\": {\"type\": \"integer\", \"minimum\": 0},\n")
	sb.WriteString("          \"lines\": {\n")
	sb.WriteString("            \"type\": \"array\",\n")
	sb.WriteString("            \"minItems\": 1,\n")
	sb.WriteString("            \"items\": {\n")
	sb.WriteString("              \"type\": \"object\",\n")
	sb.WriteString("              \"required\": [\"op\", \"text\"],\n")
	sb.WriteString("              \"additionalProperties\": false,\n")
	sb.WriteString("              \"properties\": {\n")
	sb.WriteString("                \"op\": {\"enum\": [\"context\", \"add\", \"remove\"]},\n")
	sb.WriteString("                \"text\": {\"type\": \"string\"}\n")
	sb.WriteString("              }\n")
	sb.WriteString("            }\n")
	sb.WriteString("          }\n")
	sb.WriteString("        }\n")
	sb.WriteString("      }\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n")
	sb.WriteString("- 合法示例：{\"type\":\"diff\",\"file\":\"harness content\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"# Title\"},{\"op\":\"add\",\"text\":\"新增内容\"}]}]}\n")
	sb.WriteString("- 如果上下文已足够，就直接给出贴合当前项目状态与拓扑的 diff；如果上下文不足，就先提最少但足够的澄清问题。\n")
	sb.WriteString("- 如果无法可靠地产出结构化 diff，才回退为简要说明加完整 Harness markdown 代码块。\n")
	sb.WriteString("- 如果用户要求平台写操作，优先基于上下文给出可执行的实现或先澄清缺失信息，不要输出 proposal JSON。\n")
	return nil
}

func (s *Service) writeSkillEditorContext(
	ctx context.Context,
	sb *strings.Builder,
	project catalogdomain.Project,
	input StartInput,
) error {
	skillID := uuidPtrValue(input.Context.SkillID)
	skillItem, err := s.workflows.GetSkill(ctx, skillID)
	if err != nil {
		return fmt.Errorf("get skill for chat context: %w", err)
	}

	selectedPath := "SKILL.md"
	if input.Context.SkillFilePath != nil && strings.TrimSpace(*input.Context.SkillFilePath) != "" {
		selectedPath = strings.TrimSpace(*input.Context.SkillFilePath)
	}

	var selectedFile *workflowservice.SkillBundleFile
	for index := range skillItem.Files {
		if skillItem.Files[index].Path == selectedPath {
			selectedFile = &skillItem.Files[index]
			break
		}
	}

	sb.WriteString("## 来源: Skill 编辑器\n")
	_, _ = fmt.Fprintf(sb, "项目: %s\n", project.Name)
	_, _ = fmt.Fprintf(sb, "Skill: %s | 版本: %d | 启用: %t\n", skillItem.Name, skillItem.CurrentVersion, skillItem.IsEnabled)
	_, _ = fmt.Fprintf(sb, "Path: %s | Bundle Hash: %s | Files: %d\n", skillItem.Path, skillItem.BundleHash, skillItem.FileCount)
	if skillItem.Description != "" {
		_, _ = fmt.Fprintf(sb, "Description: %s\n", skillItem.Description)
	}
	if len(skillItem.BoundWorkflows) > 0 {
		sb.WriteString("\n### 绑定的 Workflows\n")
		for _, binding := range skillItem.BoundWorkflows {
			_, _ = fmt.Fprintf(sb, "- %s (%s)\n", binding.Name, binding.HarnessPath)
		}
	}

	sb.WriteString("\n### Skill Bundle 文件清单\n")
	for _, file := range skillItem.Files {
		_, _ = fmt.Fprintf(
			sb,
			"- %s [kind=%s, encoding=%s, size=%d]",
			file.Path,
			file.FileKind,
			file.Encoding,
			file.SizeBytes,
		)
		if file.IsExecutable {
			sb.WriteString(" executable=true")
		}
		sb.WriteByte('\n')
	}

	_, _ = fmt.Fprintf(sb, "\n### 当前选中文件\n- path: %s\n", selectedPath)
	if selectedFile == nil {
		sb.WriteString("- 当前选中文件不存在于已发布 bundle 中，按未保存的新文件处理。\n")
	} else {
		_, _ = fmt.Fprintf(sb, "- kind: %s\n- encoding: %s\n- media_type: %s\n", selectedFile.FileKind, selectedFile.Encoding, selectedFile.MediaType)
	}

	publishedContent := ""
	if selectedFile != nil && selectedFile.Encoding == "utf8" {
		publishedContent = string(selectedFile.Content)
	}
	if publishedContent != "" {
		sb.WriteString("\n### 已发布文件内容\n")
		sb.WriteString("```text\n")
		sb.WriteString(publishedContent)
		if !strings.HasSuffix(publishedContent, "\n") {
			sb.WriteByte('\n')
		}
		sb.WriteString("```\n")
	}

	otherTextFiles := 0
	for _, file := range skillItem.Files {
		if file.Path == selectedPath || file.Encoding != "utf8" || len(file.Content) == 0 {
			continue
		}
		if otherTextFiles == 0 {
			sb.WriteString("\n### 其他可编辑文本文件内容\n")
		}
		otherTextFiles++
		_, _ = fmt.Fprintf(sb, "\n#### %s\n", file.Path)
		sb.WriteString("```text\n")
		sb.WriteString(string(file.Content))
		if !strings.HasSuffix(string(file.Content), "\n") {
			sb.WriteByte('\n')
		}
		sb.WriteString("```\n")
	}

	if draft := input.Context.SkillFileDraft; draft != nil {
		sb.WriteString("\n### 当前编辑器草稿（未保存）\n")
		if strings.TrimSpace(*draft) == "" {
			sb.WriteString("（当前草稿为空）\n")
		} else {
			sb.WriteString("```text\n")
			sb.WriteString(*draft)
			if !strings.HasSuffix(*draft, "\n") {
				sb.WriteByte('\n')
			}
			sb.WriteString("```\n")
		}
	}

	sb.WriteString("\n### Skill 编辑要求\n")
	sb.WriteString("- 优先围绕当前选中的文件给建议；只有在需求天然跨文件时，才同时改动多个 bundle 文件。\n")
	sb.WriteString("- 优先保留现有 skill 的职责边界、frontmatter name、描述和目录结构。\n")
	sb.WriteString("- 如果用户请求的是脚本或参考文档，保持对应语言/格式的语法正确，不要强行改成 markdown。\n")
	sb.WriteString("- 当用户请求直接修改文件时，优先输出结构化 diff JSON，供编辑器直接安全应用。\n")
	sb.WriteString("- 单文件改动使用：{\"type\":\"diff\",\"file\":\"相对文件路径\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"原行\"},{\"op\":\"add\",\"text\":\"新增行\"}]}]}\n")
	sb.WriteString("- 多文件改动使用：{\"type\":\"bundle_diff\",\"files\":[{\"file\":\"SKILL.md\",\"hunks\":[...]},{\"file\":\"scripts/redeploy.sh\",\"hunks\":[...]}]}\n")
	sb.WriteString("- 单文件 `diff.file` 必须精确等于目标文件路径；多文件 `bundle_diff.files[].file` 必须是 bundle 内相对文件路径，允许创建新的 UTF-8 文本文件。\n")
	sb.WriteString("- 所有 `hunks` 使用 1-based 行号，`lines[].op` 只能是 `context` / `add` / `remove`。\n")
	sb.WriteString("- 如果无法可靠地产出结构化 diff，才回退为简要说明加完整文件代码块。\n")
	sb.WriteString("- 普通 skill 编辑优先给可应用 diff 或说明；不要输出 proposal JSON。\n")
	return nil
}

func (s *Service) writeProjectSidebarContext(
	ctx context.Context,
	sb *strings.Builder,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
) error {
	tickets, err := s.tickets.List(ctx, ticketservice.ListInput{ProjectID: project.ID})
	if err != nil {
		return fmt.Errorf("list tickets for chat context: %w", err)
	}
	activityItems, err := s.listRecentActivity(ctx, project.ID, nil, 20)
	if err != nil {
		return err
	}

	total := len(tickets)
	inProgress := 0
	completed := 0
	failing := 0
	for _, item := range tickets {
		status := strings.ToLower(strings.TrimSpace(item.StatusName))
		switch {
		case strings.Contains(status, "progress") || strings.Contains(status, "review") || strings.Contains(status, "merge"):
			inProgress++
		case strings.Contains(status, "done"):
			completed++
		}
		if item.ConsecutiveErrors > 0 || item.RetryPaused {
			failing++
		}
	}

	sb.WriteString("## 来源: 项目侧栏\n")
	_, _ = fmt.Fprintf(sb, "项目: %s\n", project.Name)
	_, _ = fmt.Fprintf(sb, "project_id: %s\n", project.ID)
	_, _ = fmt.Fprintf(sb, "project_slug: %s\n", project.Slug)
	if project.Description != "" {
		sb.WriteString(project.Description)
		sb.WriteString("\n")
	}
	sb.WriteString("\n### 工单统计\n")
	_, _ = fmt.Fprintf(sb, "- 总数: %d\n", total)
	_, _ = fmt.Fprintf(sb, "- 进行中: %d\n", inProgress)
	_, _ = fmt.Fprintf(sb, "- 已完成: %d\n", completed)
	_, _ = fmt.Fprintf(sb, "- 失败/暂停: %d\n", failing)
	if focus != nil {
		sb.WriteString("\n### 当前用户关注区域\n")
		sb.WriteString(renderProjectConversationFocus(focus))
	}
	sb.WriteString("\n### 平台命令引用\n")
	_, _ = fmt.Fprintf(sb, "- current_project_id: %s\n", project.ID)
	_, _ = fmt.Fprintf(sb, "- current_project_name: %s\n", project.Name)
	_, _ = fmt.Fprintf(sb, "- current_project_slug: %s\n", project.Slug)
	statusLines, err := s.renderProjectSidebarStatusReferences(ctx, project.ID)
	if err != nil {
		return err
	}
	sb.WriteString(statusLines)
	sb.WriteString(renderProjectSidebarTicketReferences(tickets))
	sb.WriteString("\n### 最近活动\n")
	sb.WriteString(renderActivityLines(activityItems))
	return nil
}

func (s *Service) renderProjectSidebarStatusReferences(ctx context.Context, projectID uuid.UUID) (string, error) {
	if s.statuses == nil {
		return "", nil
	}
	result, err := s.statuses.List(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("list ticket statuses for project sidebar context: %w", err)
	}
	if len(result.Statuses) == 0 {
		return "- statuses: 无\n", nil
	}
	var sb strings.Builder
	sb.WriteString("- statuses:\n")
	for _, item := range result.Statuses {
		_, _ = fmt.Fprintf(&sb, "  - %s => %s\n", item.Name, item.ID)
	}
	return sb.String(), nil
}

func renderProjectSidebarTicketReferences(tickets []ticketservice.Ticket) string {
	if len(tickets) == 0 {
		return "- tickets: 无\n"
	}
	var sb strings.Builder
	sb.WriteString("- tickets:\n")
	for _, item := range tickets {
		_, _ = fmt.Fprintf(&sb, "  - %s => %s [%s]\n", item.Identifier, item.ID, item.StatusName)
	}
	return sb.String()
}

func renderProjectConversationFocus(focus *ProjectConversationFocus) string {
	if focus == nil {
		return "- 无\n"
	}

	var sb strings.Builder
	switch focus.Kind {
	case ProjectConversationFocusWorkflow:
		if focus.Workflow == nil {
			return "- 无\n"
		}
		_, _ = fmt.Fprintf(&sb, "- 类型: workflow\n")
		_, _ = fmt.Fprintf(&sb, "- 名称: %s\n", focus.Workflow.Name)
		_, _ = fmt.Fprintf(&sb, "- workflow_id: %s\n", focus.Workflow.ID)
		_, _ = fmt.Fprintf(&sb, "- workflow_type: %s\n", focus.Workflow.Type)
		_, _ = fmt.Fprintf(&sb, "- harness_path: %s\n", focus.Workflow.HarnessPath)
		_, _ = fmt.Fprintf(&sb, "- active: %t\n", focus.Workflow.IsActive)
		if focus.Workflow.SelectedArea != "" {
			_, _ = fmt.Fprintf(&sb, "- selected_area: %s\n", focus.Workflow.SelectedArea)
		}
		_, _ = fmt.Fprintf(&sb, "- has_dirty_draft: %t\n", focus.Workflow.HasDirtyDraft)
	case ProjectConversationFocusSkill:
		if focus.Skill == nil {
			return "- 无\n"
		}
		_, _ = fmt.Fprintf(&sb, "- 类型: skill\n")
		_, _ = fmt.Fprintf(&sb, "- 名称: %s\n", focus.Skill.Name)
		_, _ = fmt.Fprintf(&sb, "- skill_id: %s\n", focus.Skill.ID)
		_, _ = fmt.Fprintf(&sb, "- selected_file_path: %s\n", focus.Skill.SelectedFilePath)
		if len(focus.Skill.BoundWorkflowNames) > 0 {
			_, _ = fmt.Fprintf(&sb, "- bound_workflows: %s\n", strings.Join(focus.Skill.BoundWorkflowNames, ", "))
		} else {
			sb.WriteString("- bound_workflows: 无\n")
		}
		_, _ = fmt.Fprintf(&sb, "- has_dirty_draft: %t\n", focus.Skill.HasDirtyDraft)
	case ProjectConversationFocusTicket:
		if focus.Ticket == nil {
			return "- 无\n"
		}
		_, _ = fmt.Fprintf(&sb, "- 类型: ticket\n")
		_, _ = fmt.Fprintf(&sb, "- 标识: %s\n", focus.Ticket.Identifier)
		_, _ = fmt.Fprintf(&sb, "- ticket_id: %s\n", focus.Ticket.ID)
		_, _ = fmt.Fprintf(&sb, "- 标题: %s\n", focus.Ticket.Title)
		_, _ = fmt.Fprintf(&sb, "- 状态: %s\n", focus.Ticket.Status)
		if focus.Ticket.SelectedArea != "" {
			_, _ = fmt.Fprintf(&sb, "- selected_area: %s\n", focus.Ticket.SelectedArea)
		}
	case ProjectConversationFocusMachine:
		if focus.Machine == nil {
			return "- 无\n"
		}
		_, _ = fmt.Fprintf(&sb, "- 类型: machine\n")
		_, _ = fmt.Fprintf(&sb, "- 名称: %s\n", focus.Machine.Name)
		_, _ = fmt.Fprintf(&sb, "- machine_id: %s\n", focus.Machine.ID)
		_, _ = fmt.Fprintf(&sb, "- host: %s\n", focus.Machine.Host)
		if focus.Machine.Status != "" {
			_, _ = fmt.Fprintf(&sb, "- 状态: %s\n", focus.Machine.Status)
		}
		if focus.Machine.SelectedArea != "" {
			_, _ = fmt.Fprintf(&sb, "- selected_area: %s\n", focus.Machine.SelectedArea)
		}
		if focus.Machine.HealthSummary != "" {
			_, _ = fmt.Fprintf(&sb, "- health_summary: %s\n", focus.Machine.HealthSummary)
		}
	}
	return sb.String()
}

func (s *Service) writeTicketDetailContext(
	ctx context.Context,
	sb *strings.Builder,
	project catalogdomain.Project,
	input StartInput,
) error {
	ticketID := uuidPtrValue(input.Context.TicketID)
	contextItem, err := s.loadTicketPromptContext(ctx, project.ID, ticketID)
	if err != nil {
		return err
	}
	sb.WriteString("## 来源: 工单详情页\n")
	_, _ = fmt.Fprintf(sb, "项目: %s\n", project.Name)
	s.writeTicketPromptContext(sb, contextItem)
	return nil
}

func (s *Service) loadTicketPromptContext(
	ctx context.Context,
	projectID uuid.UUID,
	ticketID uuid.UUID,
) (ticketPromptContext, error) {
	ticketItem, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		return ticketPromptContext{}, fmt.Errorf("get ticket for chat context: %w", err)
	}
	repoScopes, err := s.catalog.ListTicketRepoScopes(ctx, projectID, ticketID)
	if err != nil {
		return ticketPromptContext{}, fmt.Errorf("list repo scopes for chat context: %w", err)
	}
	activityItems, err := s.listRecentActivity(ctx, projectID, &ticketID, defaultActivityLimit)
	if err != nil {
		return ticketPromptContext{}, err
	}
	return ticketPromptContext{
		Ticket:        ticketItem,
		RepoScopes:    repoScopes,
		ActivityItems: activityItems,
		HookHistory:   filterHookActivityEvents(activityItems),
	}, nil
}

func (s *Service) writeTicketPromptContext(sb *strings.Builder, contextItem ticketPromptContext) {
	_, _ = fmt.Fprintf(sb, "工单: %s - %s\n", contextItem.Ticket.Identifier, contextItem.Ticket.Title)
	_, _ = fmt.Fprintf(sb, "状态: %s | 优先级: %s | 尝试次数: %d\n", contextItem.Ticket.StatusName, contextItem.Ticket.Priority, contextItem.Ticket.AttemptCount)
	if contextItem.Ticket.Description != "" {
		sb.WriteString("\n### 描述\n")
		sb.WriteString(contextItem.Ticket.Description)
		sb.WriteString("\n")
	}
	if len(contextItem.Ticket.Dependencies) > 0 {
		sb.WriteString("\n### 依赖工单\n")
		for _, dependency := range contextItem.Ticket.Dependencies {
			_, _ = fmt.Fprintf(sb, "- [%s] %s (%s)\n", dependency.Target.Identifier, dependency.Target.Title, dependency.Type)
		}
	}
	if len(contextItem.RepoScopes) > 0 {
		sb.WriteString("\n### 仓库范围\n")
		for _, scope := range contextItem.RepoScopes {
			_, _ = fmt.Fprintf(sb, "- repo=%s branch=%s", scope.RepoID, scope.BranchName)
			if scope.PullRequestURL != nil && *scope.PullRequestURL != "" {
				_, _ = fmt.Fprintf(sb, " pr_url=%s", *scope.PullRequestURL)
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n### 活动日志\n")
	sb.WriteString(renderActivityLines(contextItem.ActivityItems))

	if len(contextItem.HookHistory) > 0 {
		sb.WriteString("\n### Hook 历史\n")
		sb.WriteString(renderActivityLines(contextItem.HookHistory))
	}
}

func (s *Service) listRecentActivity(
	ctx context.Context,
	projectID uuid.UUID,
	ticketID *uuid.UUID,
	limit int,
) ([]catalogdomain.ActivityEvent, error) {
	rawInput := catalogdomain.ActivityEventListInput{
		Limit: strconv.Itoa(limit),
	}
	if ticketID != nil {
		rawInput.TicketID = ticketID.String()
	}

	input, err := catalogdomain.ParseListActivityEvents(projectID, rawInput)
	if err != nil {
		return nil, err
	}
	items, err := s.catalog.ListActivityEvents(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("list activity for chat context: %w", err)
	}
	return items, nil
}

func buildBaseArgs(cliArgs []string, modelName string) []string {
	args := append([]string(nil), cliArgs...)
	if strings.TrimSpace(modelName) == "" || hasModelFlag(args) {
		return args
	}

	return append(args, "--model", modelName)
}

func buildCodexArgs(cliArgs []string) []string {
	return append([]string(nil), cliArgs...)
}

func hasModelFlag(args []string) bool {
	for index, arg := range args {
		if arg == "--model" && index+1 < len(args) {
			return true
		}
		if strings.HasPrefix(arg, "--model=") {
			return true
		}
	}
	return false
}

func parseSource(raw string) (Source, error) {
	source := Source(strings.TrimSpace(raw))
	switch source {
	case SourceHarnessEditor, SourceSkillEditor, SourceProjectSidebar, SourceTicketDetail:
		return source, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrSourceUnsupported, raw)
	}
}

func validateSourceContext(source Source, workflowID *uuid.UUID, ticketID *uuid.UUID, skillID *uuid.UUID) error {
	switch source {
	case SourceHarnessEditor:
		if workflowID == nil {
			return fmt.Errorf("context.workflow_id is required for source %s", source)
		}
	case SourceSkillEditor:
		if skillID == nil {
			return fmt.Errorf("context.skill_id is required for source %s", source)
		}
	case SourceTicketDetail:
		if ticketID == nil {
			return fmt.Errorf("context.ticket_id is required for source %s", source)
		}
	}
	return nil
}

func parseRequiredUUIDPointer(field string, raw *string) (uuid.UUID, error) {
	if raw == nil {
		return uuid.UUID{}, fmt.Errorf("%s is required", field)
	}
	return parseRequiredUUID(field, *raw)
}

func parseRequiredUUID(field string, raw string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return uuid.UUID{}, fmt.Errorf("%s must not be empty", field)
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s must be a valid UUID", field)
	}
	return parsed, nil
}

func parseOptionalUUIDPointer(field string, raw *string) (*uuid.UUID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	parsed, err := parseRequiredUUID(field, *raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalSessionID(raw *string) (*SessionID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	parsed, err := ParseSessionID(*raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func cloneOptionalString(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := *raw
	return &value
}

func renderHarnessVariableDictionary() string {
	var sb strings.Builder
	for _, group := range workflowservice.HarnessVariableDictionary() {
		_, _ = fmt.Fprintf(&sb, "#### %s\n", group.Name)
		for _, variable := range group.Variables {
			_, _ = fmt.Fprintf(&sb, "- `%s` (%s): %s", variable.Path, variable.Type, variable.Description)
			if variable.Example != "" {
				_, _ = fmt.Fprintf(&sb, " 示例: `%s`", variable.Example)
			}
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func renderActivityLines(items []catalogdomain.ActivityEvent) string {
	if len(items) == 0 {
		return "- 无\n"
	}

	var sb strings.Builder
	for _, item := range items {
		_, _ = fmt.Fprintf(
			&sb,
			"- [%s] %s (%s)\n",
			item.CreatedAt.UTC().Format(time.RFC3339),
			item.Message,
			item.EventType.String(),
		)
	}
	return sb.String()
}

func filterHookActivityEvents(items []catalogdomain.ActivityEvent) []catalogdomain.ActivityEvent {
	filtered := make([]catalogdomain.ActivityEvent, 0, len(items))
	for _, item := range items {
		if !isHookActivityEvent(item) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func isHookActivityEvent(item catalogdomain.ActivityEvent) bool {
	switch item.EventType {
	case activityevent.TypeHookStarted, activityevent.TypeHookPassed, activityevent.TypeHookFailed:
		return true
	default:
		return false
	}
}

func uuidPtrValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.UUID{}
	}
	return *value
}

func (s *Service) renderHarnessStatusContext(
	ctx context.Context,
	projectID uuid.UUID,
) (string, map[uuid.UUID]string, error) {
	if s.statuses == nil {
		return "", map[uuid.UUID]string{}, nil
	}

	result, err := s.statuses.List(ctx, projectID)
	if err != nil {
		return "", nil, fmt.Errorf("list ticket statuses for chat context: %w", err)
	}

	statusNamesByID := make(map[uuid.UUID]string, len(result.Statuses))
	if len(result.Statuses) == 0 {
		return "- 无\n", statusNamesByID, nil
	}

	var sb strings.Builder
	for _, item := range result.Statuses {
		statusNamesByID[item.ID] = item.Name
		_, _ = fmt.Fprintf(&sb, "- %d. %s [stage=%s", item.Position, item.Name, item.Stage)
		if item.IsDefault {
			sb.WriteString(", default=true")
		}
		if item.MaxActiveRuns != nil {
			_, _ = fmt.Fprintf(&sb, ", max_active_runs=%d", *item.MaxActiveRuns)
		}
		sb.WriteString("]\n")
	}
	return sb.String(), statusNamesByID, nil
}

func (s *Service) renderHarnessWorkflowTopology(
	ctx context.Context,
	projectID uuid.UUID,
	currentWorkflowID uuid.UUID,
	statusNamesByID map[uuid.UUID]string,
) (string, map[uuid.UUID]string, error) {
	items, err := s.workflows.List(ctx, projectID)
	if err != nil {
		return "", nil, fmt.Errorf("list workflows for chat context: %w", err)
	}

	workflowNamesByID := make(map[uuid.UUID]string, len(items))
	if len(items) == 0 {
		return "- 无\n", workflowNamesByID, nil
	}

	var sb strings.Builder
	for _, item := range items {
		workflowNamesByID[item.ID] = item.Name
		_, _ = fmt.Fprintf(&sb, "- %s [%s]", item.Name, item.Type)
		if item.ID == currentWorkflowID {
			sb.WriteString(" (current)")
		}
		_, _ = fmt.Fprintf(
			&sb,
			" pickup=%s finish=%s active=%t harness=%s retry=%d timeout=%d concurrent=%d\n",
			renderStatusBindingNames(item.PickupStatusIDs, statusNamesByID),
			renderStatusBindingNames(item.FinishStatusIDs, statusNamesByID),
			item.IsActive,
			item.HarnessPath,
			item.MaxRetryAttempts,
			item.TimeoutMinutes,
			item.MaxConcurrent,
		)
	}
	return sb.String(), workflowNamesByID, nil
}

func (s *Service) renderHarnessRepoContext(ctx context.Context, projectID uuid.UUID) (string, error) {
	repos, err := s.catalog.ListProjectRepos(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("list project repos for chat context: %w", err)
	}
	if len(repos) == 0 {
		return "- 无\n", nil
	}

	var sb strings.Builder
	for _, repo := range repos {
		_, _ = fmt.Fprintf(
			&sb,
			"- %s default_branch=%s workspace=%s url=%s",
			repo.Name,
			repo.DefaultBranch,
			repo.WorkspaceDirname,
			repo.RepositoryURL,
		)
		if len(repo.Labels) > 0 {
			_, _ = fmt.Fprintf(&sb, " labels=%s", strings.Join(repo.Labels, ", "))
		}
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

func (s *Service) renderHarnessTicketSamples(
	ctx context.Context,
	projectID uuid.UUID,
	workflowNamesByID map[uuid.UUID]string,
) (string, error) {
	items, err := s.tickets.List(ctx, ticketservice.ListInput{
		ProjectID: projectID,
		Limit:     12,
	})
	if err != nil {
		return "", fmt.Errorf("list tickets for chat context: %w", err)
	}
	if len(items) == 0 {
		return "- 无\n", nil
	}

	var sb strings.Builder
	for _, item := range items {
		workflowName := "unassigned"
		if item.WorkflowID != nil {
			if name, ok := workflowNamesByID[*item.WorkflowID]; ok {
				workflowName = name
			} else {
				workflowName = item.WorkflowID.String()
			}
		}
		_, _ = fmt.Fprintf(
			&sb,
			"- %s %s | status=%s | workflow=%s | attempts=%d | paused=%t | consecutive_errors=%d\n",
			item.Identifier,
			item.Title,
			item.StatusName,
			workflowName,
			item.AttemptCount,
			item.RetryPaused,
			item.ConsecutiveErrors,
		)
	}
	return sb.String(), nil
}

func renderStatusBindingNames(statusIDs []uuid.UUID, statusNamesByID map[uuid.UUID]string) string {
	if len(statusIDs) == 0 {
		return "none"
	}

	names := make([]string, 0, len(statusIDs))
	for _, statusID := range statusIDs {
		if name, ok := statusNamesByID[statusID]; ok {
			names = append(names, name)
			continue
		}
		names = append(names, statusID.String())
	}
	return strings.Join(names, ", ")
}
