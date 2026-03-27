package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
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
	codeFencePattern           = regexp.MustCompile("(?s)^```(?:json)?\\s*(\\{.*\\})\\s*```$")
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
	logger           *slog.Logger
	runtime          Runtime
	catalog          catalogReader
	tickets          ticketReader
	workflows        workflowReader
	workingDir       provider.AbsolutePath
	maxTurns         int
	maxBudgetUSD     float64
	sessionProviders sessionProviderRegistry
}

type donePayload struct {
	SessionID      string   `json:"session_id"`
	CostUSD        *float64 `json:"cost_usd,omitempty"`
	TurnsUsed      int      `json:"turns_used"`
	TurnsRemaining int      `json:"turns_remaining"`
}

type errorPayload struct {
	Message string `json:"message"`
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

func (s *Service) StartTurn(ctx context.Context, input StartInput) (TurnStream, error) {
	if s == nil || s.runtime == nil || s.catalog == nil || s.tickets == nil || s.workflows == nil {
		return TurnStream{}, ErrUnavailable
	}

	project, err := s.catalog.GetProject(ctx, input.Context.ProjectID)
	if err != nil {
		return TurnStream{}, fmt.Errorf("get project for chat: %w", err)
	}

	sessionID, created := s.resolveSessionID(input.SessionID)
	providerItem, err := s.resolveProvider(ctx, project, input.ProviderID, input.SessionID, sessionID)
	if err != nil {
		return TurnStream{}, err
	}

	systemPrompt, err := s.buildSystemPrompt(ctx, input, project)
	if err != nil {
		return TurnStream{}, err
	}

	if created {
		s.sessionProviders.Register(sessionID, providerItem.ID)
	}

	stream, err := s.runtime.StartTurn(ctx, RuntimeTurnInput{
		SessionID:        sessionID,
		Provider:         providerItem,
		Message:          input.Message,
		SystemPrompt:     systemPrompt,
		WorkingDirectory: s.workingDir,
		MaxTurns:         s.maxTurns,
		MaxBudgetUSD:     s.maxBudgetUSD,
	})
	if err != nil {
		if created {
			s.sessionProviders.Delete(sessionID)
		}
		return TurnStream{}, err
	}

	return stream, nil
}

func (s *Service) CloseSession(sessionID SessionID) bool {
	if s == nil || s.runtime == nil {
		return false
	}
	s.sessionProviders.Delete(sessionID)
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
	rawSessionID *SessionID,
	sessionID SessionID,
) (catalogdomain.AgentProvider, error) {
	providers, err := s.catalog.ListAgentProviders(ctx, project.OrganizationID)
	if err != nil {
		return catalogdomain.AgentProvider{}, fmt.Errorf("list project agent providers for chat: %w", err)
	}

	resolvedProviderID := rawProviderID
	if rawSessionID != nil {
		sessionProviderID, ok := s.sessionProviders.Resolve(*rawSessionID)
		if !ok {
			return catalogdomain.AgentProvider{}, ErrSessionNotFound
		}
		if resolvedProviderID != nil && *resolvedProviderID != sessionProviderID {
			return catalogdomain.AgentProvider{}, ErrSessionProviderMismatch
		}
		resolvedProviderID = &sessionProviderID
	}

	if resolvedProviderID != nil {
		providerItem, ok := findProvider(providers, *resolvedProviderID)
		if !ok {
			return catalogdomain.AgentProvider{}, ErrProviderNotFound
		}
		if !s.runtime.Supports(providerItem) {
			return catalogdomain.AgentProvider{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, providerItem.AdapterType)
		}
		if !providerItem.Available {
			return catalogdomain.AgentProvider{}, fmt.Errorf("%w: %s", ErrProviderUnavailable, providerItem.Name)
		}
		return providerItem, nil
	}

	if project.DefaultAgentProviderID != nil {
		if providerItem, ok := findProvider(providers, *project.DefaultAgentProviderID); ok && providerItem.Available && s.runtime.Supports(providerItem) {
			return providerItem, nil
		}
	}

	for _, providerItem := range providers {
		if providerItem.Available && s.runtime.Supports(providerItem) {
			return providerItem, nil
		}
	}

	return catalogdomain.AgentProvider{}, fmt.Errorf("%w: project=%s session=%s", ErrProviderNotFound, project.ID, sessionID)
}

func findProvider(items []catalogdomain.AgentProvider, want uuid.UUID) (catalogdomain.AgentProvider, bool) {
	for _, item := range items {
		if item.ID == want {
			return item, true
		}
	}

	return catalogdomain.AgentProvider{}, false
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
	sb.WriteString("- 当用户请求修改 Harness 时，先简要说明修改意图，再给出完整的更新后 Harness。\n")
	sb.WriteString("- 完整 Harness 必须放在一个 ```markdown 代码块中，便于编辑器提取 diff 并应用。\n")
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

func extractAssistantTextBlocks(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil
	}

	items := make([]string, 0, len(message.Content))
	for _, block := range message.Content {
		if block.Type != "text" {
			continue
		}
		text := strings.TrimSpace(block.Text)
		if text == "" {
			continue
		}
		items = append(items, text)
	}
	return items
}

func parseActionProposalText(text string) (map[string]any, bool) {
	trimmed := extractJSONObjectCandidate(text)
	if trimmed == "" {
		return nil, false
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, false
	}
	if strings.TrimSpace(stringValue(payload["type"])) != "action_proposal" {
		return nil, false
	}
	if _, ok := payload["actions"]; !ok {
		return nil, false
	}
	return payload, true
}

func extractJSONObjectCandidate(text string) string {
	trimmed := strings.TrimSpace(text)
	if matches := codeFencePattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	return trimmed
}

func decodeRawJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return string(raw)
	}
	return decoded
}

func stringValue(value any) string {
	typed, _ := value.(string)
	return typed
}

func uuidPtrValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.UUID{}
	}
	return *value
}
