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

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
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
	ProjectID  *string `json:"project_id"`
	WorkflowID *string `json:"workflow_id"`
	TicketID   *string `json:"ticket_id"`
}

type Context struct {
	ProjectID  uuid.UUID
	WorkflowID *uuid.UUID
	TicketID   *uuid.UUID
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
}

type Service struct {
	logger       *slog.Logger
	runtime      Runtime
	catalog      catalogReader
	tickets      ticketReader
	workflows    workflowReader
	workingDir   provider.AbsolutePath
	maxTurns     int
	maxBudgetUSD float64
	sessions     sessionRegistry
	userLocks    userLockRegistry
}

type donePayload struct {
	SessionID      string   `json:"session_id"`
	CostUSD        *float64 `json:"cost_usd,omitempty"`
	TurnsUsed      int      `json:"turns_used"`
	TurnsRemaining *int     `json:"turns_remaining,omitempty"`
}

type sessionPayload struct {
	SessionID string `json:"session_id"`
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

func NewService(
	logger *slog.Logger,
	runtime Runtime,
	catalog catalogReader,
	tickets ticketReader,
	workflows workflowReader,
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
		workingDir:   workingDir,
		maxTurns:     DefaultMaxTurns,
		maxBudgetUSD: DefaultMaxBudgetUSD,
	}
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
	if err := validateSourceContext(source, workflowID, ticketID); err != nil {
		return StartInput{}, err
	}

	sessionID, err := parseOptionalSessionID(raw.SessionID)
	if err != nil {
		return StartInput{}, err
	}

	return StartInput{
		Message:    message,
		Source:     source,
		ProviderID: providerID,
		Context: Context{
			ProjectID:  projectID,
			WorkflowID: workflowID,
			TicketID:   ticketID,
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
	providerItem, err := s.resolveProvider(ctx, project, input.ProviderID, existingSession, sessionID)
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
	}

	stream, err := s.runtime.StartTurn(ctx, RuntimeTurnInput{
		SessionID:        sessionID,
		Provider:         providerItem,
		Message:          input.Message,
		SystemPrompt:     systemPrompt,
		WorkingDirectory: s.workingDir,
		MaxTurns:         policy.MaxTurns,
		MaxBudgetUSD:     policy.MaxBudgetUSD,
	})
	if err != nil {
		if created {
			s.sessions.Delete(sessionID)
		}
		return TurnStream{}, err
	}

	if created {
		if _, ok := s.sessions.ResolveForUser(userID, sessionID); !ok {
			s.runtime.CloseSession(sessionID)
			return TurnStream{}, ErrSessionNotFound
		}
	}

	events := make(chan StreamEvent, 1)
	go func() {
		defer close(events)

		events <- StreamEvent{
			Event:   "session",
			Payload: sessionPayload{SessionID: sessionID.String()},
		}

		for event := range stream.Events {
			events <- event
			s.handleTerminalEvent(sessionID, event)
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
	if !ok {
		return false
	}

	s.sessions.Delete(sessionID)
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

	firstUnavailable := (*providerResolutionIssue)(nil)
	if resolvedProviderID != nil {
		providerItem, ok := findProvider(providers, *resolvedProviderID)
		if !ok {
			return catalogdomain.AgentProvider{}, ErrProviderNotFound
		}
		capability := resolveEphemeralChatCapability(providerItem)
		switch capability.State {
		case catalogdomain.AgentProviderCapabilityStateUnsupported:
			return catalogdomain.AgentProvider{}, providerResolutionError(ErrProviderUnsupported, providerItem, capability)
		case catalogdomain.AgentProviderCapabilityStateUnavailable:
			return catalogdomain.AgentProvider{}, providerResolutionError(ErrProviderUnavailable, providerItem, capability)
		}
		if !s.runtime.Supports(providerItem) {
			return catalogdomain.AgentProvider{}, fmt.Errorf("%w: provider=%s reason=runtime_missing", ErrProviderUnsupported, providerItem.Name)
		}
		return providerItem, nil
	}

	if project.DefaultAgentProviderID != nil {
		if providerItem, ok := findProvider(providers, *project.DefaultAgentProviderID); ok {
			capability := resolveEphemeralChatCapability(providerItem)
			if capability.State == catalogdomain.AgentProviderCapabilityStateAvailable && s.runtime.Supports(providerItem) {
				return providerItem, nil
			}
			if capability.State == catalogdomain.AgentProviderCapabilityStateUnavailable && firstUnavailable == nil {
				firstUnavailable = &providerResolutionIssue{
					provider:   providerItem,
					capability: capability,
				}
			}
		}
	}

	for _, providerItem := range providers {
		capability := resolveEphemeralChatCapability(providerItem)
		if capability.State == catalogdomain.AgentProviderCapabilityStateAvailable && s.runtime.Supports(providerItem) {
			return providerItem, nil
		}
		if capability.State == catalogdomain.AgentProviderCapabilityStateUnavailable && firstUnavailable == nil {
			firstUnavailable = &providerResolutionIssue{
				provider:   providerItem,
				capability: capability,
			}
		}
	}

	if firstUnavailable != nil {
		return catalogdomain.AgentProvider{}, providerResolutionError(
			ErrProviderUnavailable,
			firstUnavailable.provider,
			firstUnavailable.capability,
		)
	}

	return catalogdomain.AgentProvider{}, fmt.Errorf("%w: project=%s session=%s", ErrProviderNotFound, project.ID, sessionID)
}

type providerResolutionIssue struct {
	provider   catalogdomain.AgentProvider
	capability catalogdomain.AgentProviderCapability
}

func resolveEphemeralChatCapability(providerItem catalogdomain.AgentProvider) catalogdomain.AgentProviderCapability {
	providerItem = catalogdomain.DeriveAgentProviderCapabilities(providerItem)
	return providerItem.Capabilities.EphemeralChat
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
	if !ok {
		return
	}

	state, deleted := s.sessions.Delete(previousSessionID)
	if !deleted || state.Released {
		return
	}

	s.runtime.CloseSession(previousSessionID)
}

func (s *Service) handleTerminalEvent(sessionID SessionID, event StreamEvent) {
	if event.Event == "error" {
		state, ok := s.sessions.Delete(sessionID)
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
	s.runtime.CloseSession(sessionID)
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
	sb.WriteString("请基于下面的上下文回答。不要声称已经执行了任何平台写操作。")
	sb.WriteString("如果用户请求创建/修改工单等平台操作，请输出一个 action_proposal JSON 对象，等待前端确认后再执行。\n\n")

	switch input.Source {
	case SourceHarnessEditor:
		if err := s.writeHarnessEditorContext(ctx, &sb, project, input); err != nil {
			return "", err
		}
	case SourceProjectSidebar:
		if err := s.writeProjectSidebarContext(ctx, &sb, project); err != nil {
			return "", err
		}
	case SourceTicketDetail:
		if err := s.writeTicketDetailContext(ctx, &sb, project, input); err != nil {
			return "", err
		}
	default:
		return "", ErrSourceUnsupported
	}

	sb.WriteString("\n## action_proposal 协议\n")
	sb.WriteString("当且仅当用户明确要求平台写操作时，请只输出一个 JSON 对象，不要添加解释文本。格式如下：\n")
	sb.WriteString("{\"type\":\"action_proposal\",\"summary\":\"一句话总结\",\"actions\":[{\"method\":\"POST|PATCH|DELETE\",\"path\":\"/api/v1/...\",\"body\":{}}]}\n")
	sb.WriteString("适合使用 action_proposal 的请求包括：拆分子工单、创建 ticket、修改 ticket 状态、绑定 workflow。\n")

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
	_, _ = fmt.Fprintf(sb, "Workflow: %s (%s)\n\n", workflowItem.Name, workflowItem.Type)
	sb.WriteString("### 当前 Harness\n")
	sb.WriteString("```markdown\n")
	sb.WriteString(workflowItem.HarnessContent)
	if !strings.HasSuffix(workflowItem.HarnessContent, "\n") {
		sb.WriteByte('\n')
	}
	sb.WriteString("```\n\n")
	sb.WriteString("### 可用模板变量\n")
	sb.WriteString(renderHarnessVariableDictionary())
	sb.WriteByte('\n')
	sb.WriteString("\n### Harness 编辑器回复要求\n")
	sb.WriteString("- 当用户请求修改 Harness 时，优先输出一个结构化 diff JSON 对象，供编辑器直接安全应用。\n")
	sb.WriteString("- diff JSON 格式如下：{\"type\":\"diff\",\"file\":\"harness content\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"# Title\"},{\"op\":\"add\",\"text\":\"新增内容\"}]}]}\n")
	sb.WriteString("- `file` 固定写 `harness content`，`hunks` 使用 1-based 行号，`lines[].op` 只能是 `context` / `add` / `remove`。\n")
	sb.WriteString("- 如果无法可靠地产出结构化 diff，才回退为简要说明加完整 Harness markdown 代码块。\n")
	sb.WriteString("- 只有在用户明确要求平台写操作时才输出 action_proposal；普通 Harness 建议不要输出 action_proposal。\n")
	return nil
}

func (s *Service) writeProjectSidebarContext(
	ctx context.Context,
	sb *strings.Builder,
	project catalogdomain.Project,
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
	if project.Description != "" {
		sb.WriteString(project.Description)
		sb.WriteString("\n")
	}
	sb.WriteString("\n### 工单统计\n")
	_, _ = fmt.Fprintf(sb, "- 总数: %d\n", total)
	_, _ = fmt.Fprintf(sb, "- 进行中: %d\n", inProgress)
	_, _ = fmt.Fprintf(sb, "- 已完成: %d\n", completed)
	_, _ = fmt.Fprintf(sb, "- 失败/暂停: %d\n", failing)
	sb.WriteString("\n### 最近活动\n")
	sb.WriteString(renderActivityLines(activityItems))
	return nil
}

func (s *Service) writeTicketDetailContext(
	ctx context.Context,
	sb *strings.Builder,
	project catalogdomain.Project,
	input StartInput,
) error {
	ticketID := uuidPtrValue(input.Context.TicketID)
	ticketItem, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("get ticket for chat context: %w", err)
	}
	repoScopes, err := s.catalog.ListTicketRepoScopes(ctx, project.ID, ticketID)
	if err != nil {
		return fmt.Errorf("list repo scopes for chat context: %w", err)
	}
	activityItems, err := s.listRecentActivity(ctx, project.ID, &ticketID, defaultActivityLimit)
	if err != nil {
		return err
	}

	sb.WriteString("## 来源: 工单详情页\n")
	_, _ = fmt.Fprintf(sb, "项目: %s\n", project.Name)
	_, _ = fmt.Fprintf(sb, "工单: %s - %s\n", ticketItem.Identifier, ticketItem.Title)
	_, _ = fmt.Fprintf(sb, "状态: %s | 优先级: %s | 尝试次数: %d\n", ticketItem.StatusName, ticketItem.Priority, ticketItem.AttemptCount)
	if ticketItem.Description != "" {
		sb.WriteString("\n### 描述\n")
		sb.WriteString(ticketItem.Description)
		sb.WriteString("\n")
	}
	if len(ticketItem.Dependencies) > 0 {
		sb.WriteString("\n### 依赖工单\n")
		for _, dependency := range ticketItem.Dependencies {
			_, _ = fmt.Fprintf(sb, "- [%s] %s (%s)\n", dependency.Target.Identifier, dependency.Target.Title, dependency.Type)
		}
	}
	if len(repoScopes) > 0 {
		sb.WriteString("\n### 仓库范围\n")
		for _, scope := range repoScopes {
			_, _ = fmt.Fprintf(sb, "- repo=%s branch=%s pr_status=%s ci_status=%s\n", scope.RepoID, scope.BranchName, scope.PrStatus, scope.CiStatus)
		}
	}
	sb.WriteString("\n### 活动日志\n")
	sb.WriteString(renderActivityLines(activityItems))

	hookHistory := filterHookActivityEvents(activityItems)
	if len(hookHistory) > 0 {
		sb.WriteString("\n### Hook 历史\n")
		sb.WriteString(renderActivityLines(hookHistory))
	}
	return nil
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
	case SourceHarnessEditor, SourceProjectSidebar, SourceTicketDetail:
		return source, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrSourceUnsupported, raw)
	}
}

func validateSourceContext(source Source, workflowID *uuid.UUID, ticketID *uuid.UUID) error {
	switch source {
	case SourceHarnessEditor:
		if workflowID == nil {
			return fmt.Errorf("context.workflow_id is required for source %s", source)
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
		_, _ = fmt.Fprintf(&sb, "- [%s] %s (%s)\n", item.CreatedAt.UTC().Format(time.RFC3339), item.Message, item.EventType)
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
	if strings.Contains(strings.ToLower(item.EventType), "hook") {
		return true
	}
	for _, key := range []string{"hook", "hook_name", "hook_stage", "hook_result", "hook_outcome"} {
		if _, ok := item.Metadata[key]; ok {
			return true
		}
	}
	return false
}

func uuidPtrValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.UUID{}
	}
	return *value
}
