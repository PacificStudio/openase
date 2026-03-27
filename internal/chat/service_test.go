package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestParseStartInputRequiresTicketForTicketDetail(t *testing.T) {
	_, err := ParseStartInput(RawStartInput{
		Message: "Why did it fail?",
		Source:  string(SourceTicketDetail),
		Context: RawChatContext{
			ProjectID: stringPointer("550e8400-e29b-41d4-a716-446655440000"),
		},
	})
	if err == nil || err.Error() != "context.ticket_id is required for source ticket_detail" {
		t.Fatalf("expected missing ticket_id error, got %v", err)
	}
}

func TestMapClaudeEventPromotesActionProposalJSON(t *testing.T) {
	events := mapClaudeEvent(SessionID("session-1"), DefaultMaxTurns, provider.ClaudeCodeEvent{
		Kind: provider.ClaudeCodeEventKindAssistant,
		Message: []byte("{\n" +
			"  \"role\":\"assistant\",\n" +
			"  \"content\":[\n" +
			"    {\n" +
			"      \"type\":\"text\",\n" +
			"      \"text\":\"```json\\n{\\\"type\\\":\\\"action_proposal\\\",\\\"summary\\\":\\\"Create 2 child tickets\\\",\\\"actions\\\":[{\\\"method\\\":\\\"POST\\\",\\\"path\\\":\\\"/api/v1/projects/p/tickets\\\",\\\"body\\\":{\\\"title\\\":\\\"Child\\\"}}]}\\n```\"\n" +
			"    }\n" +
			"  ]\n" +
			"}"),
	})
	if len(events) != 1 {
		t.Fatalf("expected one mapped event, got %d", len(events))
	}

	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected action proposal payload map, got %#v", events[0].Payload)
	}
	if payload["type"] != "action_proposal" || payload["summary"] != "Create 2 child tickets" {
		t.Fatalf("unexpected action proposal payload: %#v", payload)
	}
}

func TestParseActionProposalTextAcceptsCodeFenceWithWhitespace(t *testing.T) {
	payload, ok := parseActionProposalText("```json \n {\"type\":\"action_proposal\",\"actions\":[]} \n```")
	if !ok {
		t.Fatalf("expected action proposal to parse")
	}
	if payload["type"] != "action_proposal" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestBuildSystemPromptGuidesHarnessEditorReplies(t *testing.T) {
	workflowID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	service := NewService(nil, nil, nil, nil, harnessWorkflowReader{
		detail: workflowservice.WorkflowDetail{
			Workflow: workflowservice.Workflow{
				Name: "Coding Workflow",
				Type: "coding",
			},
			HarnessContent: "---\ntype: coding\n---\n\nWrite code.\n",
		},
	}, "")

	prompt, err := service.buildSystemPrompt(
		context.Background(),
		StartInput{
			Source: SourceHarnessEditor,
			Context: Context{
				ProjectID:  uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				WorkflowID: &workflowID,
			},
		},
		catalogdomain.Project{Name: "OpenASE"},
	)
	if err != nil {
		t.Fatalf("build system prompt: %v", err)
	}
	if !containsAll(prompt,
		"Harness 编辑器回复要求",
		"完整 Harness 必须放在一个 ```markdown 代码块中",
		"普通 Harness 建议不要输出 action_proposal",
	) {
		t.Fatalf("expected harness-editor response instructions in prompt, got %q", prompt)
	}
}

func TestStartTurnStreamsProjectSidebarContext(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	runtime := &fakeRuntime{
		streamEvents: []StreamEvent{
			{Event: "message", Payload: map[string]any{"type": "task_started"}},
			{Event: "message", Payload: textPayload{Type: "text", Content: "Project summary ready."}},
			{Event: "done", Payload: donePayload{SessionID: "sess-project-1", TurnsUsed: 2, TurnsRemaining: 8, CostUSD: floatPointer(0.12)}},
		},
	}
	catalog := fakeCatalogReader{
		project: catalogdomain.Project{
			ID:             projectID,
			OrganizationID: orgID,
			Name:           "OpenASE",
			Description:    "Issue-driven automation",
		},
		activityEvents: []catalogdomain.ActivityEvent{
			{
				CreatedAt: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
				EventType: "ticket.updated",
				Message:   "Updated issue status",
			},
		},
		providers: []catalogdomain.AgentProvider{
			{
				ID:             uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
				OrganizationID: orgID,
				Name:           "Claude Code",
				AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
				CliCommand:     "claude",
				CliArgs:        []string{"chat"},
				AuthConfig:     map[string]any{"anthropic_api_key": "test-key"},
				ModelName:      "claude-sonnet-4-6",
				Available:      true,
			},
		},
	}
	tickets := fakeTicketReader{
		items: []ticketservice.Ticket{
			{StatusName: "In Progress"},
			{StatusName: "Done"},
			{StatusName: "Todo", RetryPaused: true},
		},
	}
	service := NewService(nil, runtime, catalog, tickets, harnessWorkflowReader{}, "")

	stream, err := service.StartTurn(context.Background(), StartInput{
		Message: "Summarize project",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	collected := collectStreamEvents(stream.Events)
	if len(collected) != 3 {
		t.Fatalf("stream event count = %d, want 3: %+v", len(collected), collected)
	}
	if payload, ok := collected[0].Payload.(map[string]any); !ok || payload["type"] != "task_started" {
		t.Fatalf("first payload = %#v, want task_started", collected[0].Payload)
	}
	if payload, ok := collected[1].Payload.(textPayload); !ok || payload.Content != "Project summary ready." {
		t.Fatalf("second payload = %#v, want assistant text", collected[1].Payload)
	}
	done, ok := collected[2].Payload.(donePayload)
	if !ok || done.SessionID != "sess-project-1" || done.TurnsUsed != 2 || done.TurnsRemaining != 8 {
		t.Fatalf("done payload = %#v", collected[2].Payload)
	}

	if runtime.lastInput.Provider.CliCommand != "claude" {
		t.Fatalf("runtime provider command = %q, want claude", runtime.lastInput.Provider.CliCommand)
	}
	if strings.Join(runtime.lastInput.Provider.CliArgs, " ") != "chat" {
		t.Fatalf("runtime provider cli args = %v", runtime.lastInput.Provider.CliArgs)
	}
	if runtime.lastInput.Message != "Summarize project" {
		t.Fatalf("runtime message = %q, want Summarize project", runtime.lastInput.Message)
	}
	if !containsAll(runtime.lastInput.SystemPrompt,
		"## 来源: 项目侧栏",
		"- 总数: 3",
		"- 进行中: 1",
		"- 已完成: 1",
		"- 失败/暂停: 1",
		"Updated issue status",
	) {
		t.Fatalf("project sidebar prompt = %q", runtime.lastInput.SystemPrompt)
	}
	if runtime.lastInput.SessionID == "" {
		t.Fatal("runtime session id should be assigned")
	}
	if service.CloseSession(SessionID("sess-project-1")) {
		t.Fatal("CloseSession() after completion should return false")
	}
}

func TestBuildSystemPromptIncludesTicketDetailAndHookHistory(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
	ticketID := uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000")
	service := NewService(
		nil,
		nil,
		fakeCatalogReader{
			repoScopes: []catalogdomain.TicketRepoScope{
				{
					RepoID:     uuid.MustParse("bb0e8400-e29b-41d4-a716-446655440000"),
					BranchName: "feat/openase-278-coverage",
					PrStatus:   "open",
					CiStatus:   "passing",
				},
			},
			activityEvents: []catalogdomain.ActivityEvent{
				{
					CreatedAt: time.Date(2026, 3, 27, 12, 30, 0, 0, time.UTC),
					EventType: "agent.output",
					Message:   "Collected failing test output",
					Metadata:  map[string]any{"stream": "stdout"},
				},
				{
					CreatedAt: time.Date(2026, 3, 27, 12, 31, 0, 0, time.UTC),
					EventType: "hook.failed",
					Message:   "go test ./... failed in auth package",
					Metadata:  map[string]any{"hook_name": "ticket.on_complete"},
				},
			},
		},
		fakeTicketReader{
			ticket: ticketservice.Ticket{
				ID:           ticketID,
				Identifier:   "ASE-278",
				Title:        "Finish backend coverage rollout",
				Description:  "Raise the gate.",
				StatusName:   "In Review",
				Priority:     "high",
				AttemptCount: 3,
				Dependencies: []ticketservice.Dependency{
					{
						Type: entticketdependency.TypeBlocks,
						Target: ticketservice.TicketReference{
							Identifier: "ASE-100",
							Title:      "Primary blocker",
						},
					},
				},
			},
		},
		harnessWorkflowReader{},
		"",
	)

	prompt, err := service.buildSystemPrompt(context.Background(), StartInput{
		Source: SourceTicketDetail,
		Context: Context{
			ProjectID: projectID,
			TicketID:  &ticketID,
		},
	}, catalogdomain.Project{ID: projectID, Name: "OpenASE"})
	if err != nil {
		t.Fatalf("buildSystemPrompt() error = %v", err)
	}
	if !containsAll(prompt,
		"## 来源: 工单详情页",
		"工单: ASE-278 - Finish backend coverage rollout",
		"状态: In Review | 优先级: high | 尝试次数: 3",
		"### 依赖工单",
		"repo=bb0e8400-e29b-41d4-a716-446655440000 branch=feat/openase-278-coverage pr_status=open ci_status=passing",
		"### Hook 历史",
		"go test ./... failed in auth package",
		"\"type\":\"action_proposal\"",
	) {
		t.Fatalf("ticket detail prompt = %q", prompt)
	}

	if _, err := service.buildSystemPrompt(context.Background(), StartInput{
		Source: Source("unknown"),
		Context: Context{
			ProjectID: projectID,
		},
	}, catalogdomain.Project{ID: projectID, Name: "OpenASE"}); !errors.Is(err, ErrSourceUnsupported) {
		t.Fatalf("buildSystemPrompt() unsupported source error = %v, want %v", err, ErrSourceUnsupported)
	}
}

func TestChatHelperCoverageAndRegistry(t *testing.T) {
	t.Parallel()

	if _, err := ParseCloseSessionID("   "); err == nil {
		t.Fatal("ParseCloseSessionID() expected validation error")
	}
	if parsed, err := ParseCloseSessionID(" sess-1 "); err != nil || parsed.String() != "sess-1" {
		t.Fatalf("ParseCloseSessionID() = %q, %v", parsed, err)
	}

	if _, err := parseOptionalSessionID(nil); err != nil {
		t.Fatalf("parseOptionalSessionID(nil) error = %v", err)
	}
	if _, err := parseOptionalSessionID(stringPointer(" ")); err != nil {
		t.Fatalf("parseOptionalSessionID(blank) error = %v", err)
	}
	if _, err := parseOptionalSessionID(stringPointer("bad")); err != nil {
		t.Fatalf("parseOptionalSessionID() unexpected error = %v", err)
	}

	if got := buildBaseArgs([]string{"chat"}, "claude-sonnet-4-6"); strings.Join(got, " ") != "chat --model claude-sonnet-4-6" {
		t.Fatalf("buildBaseArgs() = %v", got)
	}
	if !hasModelFlag([]string{"chat", "--model=claude-haiku"}) {
		t.Fatal("hasModelFlag() should detect --model=value")
	}
	if !hasModelFlag([]string{"chat", "--model", "claude-haiku"}) {
		t.Fatal("hasModelFlag() should detect --model value")
	}
	if hasModelFlag([]string{"chat", "--temperature", "0"}) {
		t.Fatal("hasModelFlag() should ignore unrelated flags")
	}

	rawText := mustMarshalJSON(t, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": " first "},
			{"type": "image", "text": "ignore"},
			{"type": "text", "text": "second"},
		},
	})
	if got := extractAssistantTextBlocks(rawText); len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("extractAssistantTextBlocks() = %#v", got)
	}
	if got := extractAssistantTextBlocks(json.RawMessage("{")); got != nil {
		t.Fatalf("extractAssistantTextBlocks(invalid) = %#v, want nil", got)
	}

	if got := decodeRawJSON(json.RawMessage(`{"ok":true}`)); got.(map[string]any)["ok"] != true {
		t.Fatalf("decodeRawJSON(json) = %#v", got)
	}
	if got := decodeRawJSON(json.RawMessage("{")); got != "{" {
		t.Fatalf("decodeRawJSON(invalid) = %#v, want raw string", got)
	}
	if got := renderActivityLines(nil); got != "- 无\n" {
		t.Fatalf("renderActivityLines(nil) = %q", got)
	}

	activityItems := []catalogdomain.ActivityEvent{
		{EventType: "hook.completed", Message: "done", Metadata: map[string]any{}},
		{EventType: "ticket.updated", Message: "updated", Metadata: map[string]any{"hook_name": "ticket.on_start"}},
		{EventType: "ticket.updated", Message: "plain", Metadata: map[string]any{}},
	}
	if got := filterHookActivityEvents(activityItems); len(got) != 2 {
		t.Fatalf("filterHookActivityEvents() len = %d, want 2", len(got))
	}
	if isHookActivityEvent(activityItems[2]) {
		t.Fatal("isHookActivityEvent() should be false without hook markers")
	}

	if _, err := NewService(nil, nil, nil, nil, nil, "").StartTurn(context.Background(), StartInput{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("StartTurn() unavailable error = %v, want %v", err, ErrUnavailable)
	}

	var registry sessionProviderRegistry
	providerID := uuid.New()
	sessionID := SessionID("sess-registry")
	registry.Register(sessionID, providerID)
	if got, ok := registry.Resolve(sessionID); !ok || got != providerID {
		t.Fatalf("Resolve() = %v, %v", got, ok)
	}
	registry.Delete(sessionID)
	if _, ok := registry.Resolve(sessionID); ok {
		t.Fatal("Resolve() should fail after deletion")
	}
}

func TestBuildBaseArgsAddsModelFlagWhenMissing(t *testing.T) {
	args := buildBaseArgs([]string{"chat"}, "claude-sonnet-4-5")
	if len(args) != 3 {
		t.Fatalf("expected model flag to be appended, got %#v", args)
	}
	if args[1] != "--model" || args[2] != "claude-sonnet-4-5" {
		t.Fatalf("expected model flag args, got %#v", args)
	}
}

func TestBuildCodexArgsDoesNotAppendModelFlag(t *testing.T) {
	args := buildCodexArgs([]string{"app-server", "--listen", "stdio://"})
	if len(args) != 3 {
		t.Fatalf("expected codex args to remain unchanged, got %#v", args)
	}
	if strings.Contains(strings.Join(args, " "), "--model") {
		t.Fatalf("expected codex args without model flag, got %#v", args)
	}
}

type harnessWorkflowReader struct {
	detail workflowservice.WorkflowDetail
}

func (r harnessWorkflowReader) Get(context.Context, uuid.UUID) (workflowservice.WorkflowDetail, error) {
	return r.detail, nil
}

func stringPointer(value string) *string {
	return &value
}

func containsAll(value string, expected ...string) bool {
	for _, item := range expected {
		if !strings.Contains(value, item) {
			return false
		}
	}
	return true
}

type fakeCatalogReader struct {
	project          catalogdomain.Project
	projectErr       error
	activityEvents   []catalogdomain.ActivityEvent
	activityErr      error
	repoScopes       []catalogdomain.TicketRepoScope
	repoScopeErr     error
	providers        []catalogdomain.AgentProvider
	providerErr      error
	providerByID     map[uuid.UUID]catalogdomain.AgentProvider
	getProviderError error
}

func (r fakeCatalogReader) GetProject(context.Context, uuid.UUID) (catalogdomain.Project, error) {
	return r.project, r.projectErr
}

func (r fakeCatalogReader) ListActivityEvents(context.Context, catalogdomain.ListActivityEvents) ([]catalogdomain.ActivityEvent, error) {
	return r.activityEvents, r.activityErr
}

func (r fakeCatalogReader) ListTicketRepoScopes(context.Context, uuid.UUID, uuid.UUID) ([]catalogdomain.TicketRepoScope, error) {
	return r.repoScopes, r.repoScopeErr
}

func (r fakeCatalogReader) ListAgentProviders(context.Context, uuid.UUID) ([]catalogdomain.AgentProvider, error) {
	return r.providers, r.providerErr
}

func (r fakeCatalogReader) GetAgentProvider(_ context.Context, id uuid.UUID) (catalogdomain.AgentProvider, error) {
	if r.getProviderError != nil {
		return catalogdomain.AgentProvider{}, r.getProviderError
	}
	if item, ok := r.providerByID[id]; ok {
		return item, nil
	}
	return catalogdomain.AgentProvider{}, errors.New("provider not found")
}

type fakeTicketReader struct {
	ticket  ticketservice.Ticket
	items   []ticketservice.Ticket
	getErr  error
	listErr error
}

func (r fakeTicketReader) Get(context.Context, uuid.UUID) (ticketservice.Ticket, error) {
	return r.ticket, r.getErr
}

func (r fakeTicketReader) List(context.Context, ticketservice.ListInput) ([]ticketservice.Ticket, error) {
	return r.items, r.listErr
}

type fakeRuntime struct {
	streamEvents []StreamEvent
	lastInput    RuntimeTurnInput
	closeResult  bool
	startErr     error
}

func (r *fakeRuntime) Supports(catalogdomain.AgentProvider) bool {
	return true
}

func (r *fakeRuntime) StartTurn(_ context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if r.startErr != nil {
		return TurnStream{}, r.startErr
	}
	r.lastInput = input
	events := make(chan StreamEvent, len(r.streamEvents))
	for _, event := range r.streamEvents {
		events <- event
	}
	close(events)
	return TurnStream{Events: events}, nil
}

func (r *fakeRuntime) CloseSession(SessionID) bool {
	return r.closeResult
}

func collectStreamEvents(events <-chan StreamEvent) []StreamEvent {
	collected := make([]StreamEvent, 0)
	for event := range events {
		collected = append(collected, event)
	}
	return collected
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return body
}

func floatPointer(value float64) *float64 {
	return &value
}
