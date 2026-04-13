package chat

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	ticketstatusservice "github.com/BetterAndBetterII/openase/internal/ticketstatus"
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

func TestMapClaudeEventLeavesUnsupportedStructuredJSONAsText(t *testing.T) {
	events := mapClaudeEvent(SessionID("session-1"), DefaultMaxTurns, provider.ClaudeCodeEvent{
		Kind: provider.ClaudeCodeEventKindAssistant,
		Message: []byte("{\n" +
			"  \"role\":\"assistant\",\n" +
			"  \"content\":[\n" +
			"    {\n" +
			"      \"type\":\"text\",\n" +
			"      \"text\":\"```json\\n{\\\"type\\\":\\\"custom_structured_payload\\\",\\\"summary\\\":\\\"Create 2 child tickets\\\",\\\"items\\\":[{\\\"name\\\":\\\"Child\\\"}]}\\n```\"\n" +
			"    }\n" +
			"  ]\n" +
			"}"),
	}, nil)
	if len(events) != 1 {
		t.Fatalf("expected one mapped event, got %d", len(events))
	}

	payload, ok := events[0].Payload.(textPayload)
	if !ok {
		t.Fatalf("expected unsupported JSON to remain plain text, got %#v", events[0].Payload)
	}
	if !strings.Contains(payload.Content, "\"type\":\"custom_structured_payload\"") ||
		!strings.Contains(payload.Content, "Create 2 child tickets") {
		t.Fatalf("unexpected plain-text structured payload: %#v", payload)
	}
}

func TestNormalizeAssistantTextLeavesUnsupportedStructuredCodeFenceAsText(t *testing.T) {
	events := normalizeAssistantText("```json \n {\"type\":\"custom_structured_payload\",\"items\":[]} \n```")
	if len(events) != 1 {
		t.Fatalf("expected one text event, got %+v", events)
	}
	payload, ok := events[0].Payload.(textPayload)
	if !ok || !strings.Contains(payload.Content, "\"type\":\"custom_structured_payload\"") {
		t.Fatalf("unexpected payload: %#v", events[0].Payload)
	}
}

func TestParseDiffPayloadTextAcceptsStructuredJSON(t *testing.T) {
	payload, ok := parseDiffPayloadText("```json\n{\"type\":\"diff\",\"file\":\"harness content\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"---\"},{\"op\":\"add\",\"text\":\"new line\"}]}]}\n```")
	if !ok {
		t.Fatalf("expected diff payload to parse")
	}
	if payload.Type != chatMessageTypeDiff || payload.File != "harness content" || len(payload.Hunks) != 1 {
		t.Fatalf("unexpected diff payload: %#v", payload)
	}
}

func TestParseBundleDiffPayloadTextAcceptsStructuredJSON(t *testing.T) {
	payload, ok := parseBundleDiffPayloadText("```json\n{\"type\":\"bundle_diff\",\"files\":[{\"file\":\"SKILL.md\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"---\"},{\"op\":\"add\",\"text\":\"new line\"}]}]},{\"file\":\"scripts/redeploy.sh\",\"hunks\":[{\"old_start\":1,\"old_lines\":0,\"new_start\":1,\"new_lines\":1,\"lines\":[{\"op\":\"add\",\"text\":\"#!/usr/bin/env bash\"}]}]}]}\n```")
	if !ok {
		t.Fatalf("expected bundle diff payload to parse")
	}
	if payload.Type != chatMessageTypeBundleDiff || len(payload.Files) != 2 {
		t.Fatalf("unexpected bundle diff payload: %#v", payload)
	}
}

func TestNormalizeAssistantTextSplitsTrailingDiffFromMixedText(t *testing.T) {
	events := normalizeAssistantText(
		"I will first locate the editable area from the current harness and project topology, then return an applicable structured diff.\n" +
			"{\"type\":\"diff\",\"file\":\"harness content\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"---\"},{\"op\":\"add\",\"text\":\"new line\"}]}]}",
	)
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2: %+v", len(events), events)
	}

	text, ok := events[0].Payload.(textPayload)
	if !ok {
		t.Fatalf("first payload = %#v, want text payload", events[0].Payload)
	}
	if !strings.Contains(text.Content, "structured diff") {
		t.Fatalf("first text payload = %#v, want explanatory prose", text)
	}

	diff, ok := events[1].Payload.(diffPayload)
	if !ok {
		t.Fatalf("second payload = %#v, want diff payload", events[1].Payload)
	}
	if diff.File != "harness content" || len(diff.Hunks) != 1 {
		t.Fatalf("unexpected diff payload: %#v", diff)
	}
}

func TestNormalizeAssistantTextCollapsesRepeatedTrailingDiffJSON(t *testing.T) {
	diffJSON := "{\"type\":\"diff\",\"file\":\"harness content\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"---\"},{\"op\":\"add\",\"text\":\"new line\"}]}]}"
	events := normalizeAssistantText("Explanation first." + diffJSON + diffJSON)
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2: %+v", len(events), events)
	}

	diff, ok := events[1].Payload.(diffPayload)
	if !ok {
		t.Fatalf("second payload = %#v, want diff payload", events[1].Payload)
	}
	if diff.File != "harness content" || len(diff.Hunks) != 1 {
		t.Fatalf("unexpected diff payload: %#v", diff)
	}
}

func TestMapClaudeEventPromotesDiffJSON(t *testing.T) {
	events := mapClaudeEvent(SessionID("session-1"), DefaultMaxTurns, provider.ClaudeCodeEvent{
		Kind: provider.ClaudeCodeEventKindAssistant,
		Message: []byte("{\n" +
			"  \"role\":\"assistant\",\n" +
			"  \"content\":[\n" +
			"    {\n" +
			"      \"type\":\"text\",\n" +
			"      \"text\":\"```json\\n{\\\"type\\\":\\\"diff\\\",\\\"file\\\":\\\"harness content\\\",\\\"hunks\\\":[{\\\"old_start\\\":1,\\\"old_lines\\\":1,\\\"new_start\\\":1,\\\"new_lines\\\":2,\\\"lines\\\":[{\\\"op\\\":\\\"context\\\",\\\"text\\\":\\\"---\\\"},{\\\"op\\\":\\\"add\\\",\\\"text\\\":\\\"new line\\\"}]}]}\\n```\"\n" +
			"    }\n" +
			"  ]\n" +
			"}"),
	}, nil)
	if len(events) != 1 {
		t.Fatalf("expected one mapped event, got %d", len(events))
	}

	payload, ok := events[0].Payload.(diffPayload)
	if !ok {
		t.Fatalf("expected diff payload, got %#v", events[0].Payload)
	}
	if payload.Type != chatMessageTypeDiff || payload.File != "harness content" {
		t.Fatalf("unexpected diff payload: %#v", payload)
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
			{Event: "done", Payload: donePayload{SessionID: "sess-project-1", TurnsUsed: 2, TurnsRemaining: nil, CostUSD: floatPointer(0.12)}},
		},
	}
	catalog := fakeCatalogReader{
		project: catalogdomain.Project{
			ID:             projectID,
			OrganizationID: orgID,
			Name:           "OpenASE",
			Slug:           "openase",
			Description:    "Issue-driven automation",
		},
		activityEvents: []catalogdomain.ActivityEvent{
			{
				CreatedAt: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
				EventType: activityevent.TypeTicketUpdated,
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
	projectsRoot := t.TempDir()
	service := NewService(nil, runtime, catalog, tickets, harnessWorkflowReader{}, fakeStatusReader{
		result: ticketstatusservice.ListResult{
			Statuses: []ticketstatusservice.Status{
				{ID: uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"), Name: "Todo"},
				{ID: uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"), Name: "In Progress"},
				{ID: uuid.MustParse("bb0e8400-e29b-41d4-a716-446655440000"), Name: "Done"},
			},
		},
	}, provider.MustParseAbsolutePath(projectsRoot))

	stream, err := service.StartTurn(context.Background(), UserID("user:test"), StartInput{
		Message: "Summarize project",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	collected := collectStreamEvents(stream.Events)
	if len(collected) != 4 {
		t.Fatalf("stream event count = %d, want 4: %+v", len(collected), collected)
	}
	session, ok := collected[0].Payload.(sessionPayload)
	if !ok || session.SessionID == "" {
		t.Fatalf("first payload = %#v, want session payload", collected[0].Payload)
	}
	if collected[0].Event != "session" {
		t.Fatalf("first event = %q, want session", collected[0].Event)
	}
	if payload, ok := collected[1].Payload.(map[string]any); !ok || payload["type"] != "task_started" {
		t.Fatalf("second payload = %#v, want task_started", collected[1].Payload)
	}
	if payload, ok := collected[2].Payload.(textPayload); !ok || payload.Content != "Project summary ready." {
		t.Fatalf("third payload = %#v, want assistant text", collected[2].Payload)
	}
	done, ok := collected[3].Payload.(donePayload)
	if !ok || done.SessionID != "sess-project-1" || done.TurnsUsed != 2 || done.TurnsRemaining != nil {
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
	wantWorkingDirectory, err := workspaceinfra.ProjectChatPath(projectsRoot, projectID.String())
	if err != nil {
		t.Fatalf("ProjectChatPath() error = %v", err)
	}
	if runtime.lastInput.WorkingDirectory.String() != wantWorkingDirectory {
		t.Fatalf("runtime working directory = %q, want %q", runtime.lastInput.WorkingDirectory, wantWorkingDirectory)
	}
	if runtime.lastInput.MaxTurns != 0 {
		t.Fatalf("runtime max turns = %d, want unlimited project sidebar policy", runtime.lastInput.MaxTurns)
	}
	if !containsAll(runtime.lastInput.SystemPrompt,
		"## Source: Project Sidebar",
		"project_id: 660e8400-e29b-41d4-a716-446655440000",
		"project_slug: openase",
		"- Total: 3",
		"- In progress: 1",
		"- Completed: 1",
		"- Failed/paused: 1",
		"### Platform Command References",
		"- current_project_name: OpenASE",
		"- statuses:",
		"Todo => 990e8400-e29b-41d4-a716-446655440000",
		"Do not claim that you have already performed platform write operations.",
		"Do not output proposal JSON.",
		"Updated issue status",
	) {
		t.Fatalf("project sidebar prompt = %q", runtime.lastInput.SystemPrompt)
	}
	if runtime.lastInput.SessionID == "" {
		t.Fatal("runtime session id should be assigned")
	}
	if service.CloseSession(UserID("user:test"), SessionID("sess-project-1")) {
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
				},
			},
			activityEvents: []catalogdomain.ActivityEvent{
				{
					CreatedAt: time.Date(2026, 3, 27, 12, 30, 0, 0, time.UTC),
					EventType: activityevent.TypeAgentFailed,
					Message:   "Collected failing test output",
					Metadata:  map[string]any{"stream": "stdout"},
				},
				{
					CreatedAt: time.Date(2026, 3, 27, 12, 31, 0, 0, time.UTC),
					EventType: activityevent.TypeHookFailed,
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
						Type: ticketservice.DependencyTypeBlocks,
						Target: ticketservice.TicketReference{
							Identifier: "ASE-100",
							Title:      "Primary blocker",
						},
					},
				},
			},
		},
		harnessWorkflowReader{},
		nil,
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
		"## Source: Ticket Detail Page",
		"Ticket: ASE-278 - Finish backend coverage rollout",
		"Status: In Review | Priority: high | Attempts: 3",
		"### Dependent Tickets",
		"repo=bb0e8400-e29b-41d4-a716-446655440000 branch=feat/openase-278-coverage",
		"### Hook History",
		"go test ./... failed in auth package",
		"Do not claim that you have already performed platform write operations.",
		"Do not output proposal JSON.",
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
	if got := renderActivityLines(nil); got != "- none\n" {
		t.Fatalf("renderActivityLines(nil) = %q", got)
	}

	activityItems := []catalogdomain.ActivityEvent{
		{EventType: activityevent.TypeHookPassed, Message: "done", Metadata: map[string]any{}},
		{EventType: activityevent.TypeTicketUpdated, Message: "updated", Metadata: map[string]any{"hook_name": "ticket.on_start"}},
		{EventType: activityevent.TypeTicketUpdated, Message: "plain", Metadata: map[string]any{}},
	}
	if got := filterHookActivityEvents(activityItems); len(got) != 1 {
		t.Fatalf("filterHookActivityEvents() len = %d, want 1", len(got))
	}
	if isHookActivityEvent(activityItems[2]) {
		t.Fatal("isHookActivityEvent() should be false for non-hook event types")
	}

	if _, err := NewService(nil, nil, nil, nil, nil, nil, "").StartTurn(context.Background(), UserID("browser-session:test"), StartInput{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("StartTurn() unavailable error = %v, want %v", err, ErrUnavailable)
	}

	var registry sessionRegistry
	providerID := uuid.New()
	userID := UserID("user:registry")
	sessionID := SessionID("sess-registry")
	registry.Register(userID, sessionID, providerID, DefaultMaxTurns, DefaultMaxBudgetUSD)
	if got, ok := registry.ResolveForUser(userID, sessionID); !ok || got.ProviderID != providerID {
		t.Fatalf("ResolveForUser() = %#v, %v", got, ok)
	}
	registry.Delete(sessionID)
	if _, ok := registry.Resolve(sessionID); ok {
		t.Fatal("Resolve() should fail after deletion")
	}
}

func TestStartTurnReplacesExistingSessionForSameUser(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	userID := UserID("user:alice")
	runtime := &fakeRuntime{
		closeResult: true,
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{
				{Event: "message", Payload: textPayload{Type: "text", Content: "ready"}},
				{
					Event: "done",
					Payload: donePayload{
						SessionID:      input.SessionID.String(),
						TurnsUsed:      1,
						TurnsRemaining: nil,
					},
				},
			}
		},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
					OrganizationID: orgID,
					Name:           "Claude Code",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
					CliCommand:     "claude",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	firstStream, err := service.StartTurn(context.Background(), userID, StartInput{
		Message: "first",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("first StartTurn() error = %v", err)
	}
	firstEvents := collectStreamEvents(firstStream.Events)
	firstSession := firstEvents[0].Payload.(sessionPayload).SessionID

	secondStream, err := service.StartTurn(context.Background(), userID, StartInput{
		Message: "second",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("second StartTurn() error = %v", err)
	}
	secondEvents := collectStreamEvents(secondStream.Events)
	secondSession := secondEvents[0].Payload.(sessionPayload).SessionID

	if firstSession == secondSession {
		t.Fatalf("expected replacement session, got same session id %q", firstSession)
	}
	if len(runtime.closeCalls) == 0 || runtime.closeCalls[0] != SessionID(firstSession) {
		t.Fatalf("expected previous session %q to be closed, got %+v", firstSession, runtime.closeCalls)
	}
}

func TestStartTurnRejectsResumeAcrossUsers(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	runtime := &fakeRuntime{
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{
				{
					Event: "done",
					Payload: donePayload{
						SessionID:      input.SessionID.String(),
						TurnsUsed:      1,
						TurnsRemaining: nil,
					},
				},
			}
		},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
					OrganizationID: orgID,
					Name:           "Claude Code",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
					CliCommand:     "claude",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	stream, err := service.StartTurn(context.Background(), UserID("user:alice"), StartInput{
		Message: "first",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("seed StartTurn() error = %v", err)
	}
	events := collectStreamEvents(stream.Events)
	sessionID := SessionID(events[0].Payload.(sessionPayload).SessionID)

	if _, err := service.StartTurn(context.Background(), UserID("user:bob"), StartInput{
		Message:   "resume",
		Source:    SourceProjectSidebar,
		Context:   Context{ProjectID: projectID},
		SessionID: &sessionID,
	}); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("resume across users error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestStartTurnSessionPayloadDeclaresProcessLocalResumeContract(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	runtime := &fakeRuntime{
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{{
				Event:   "done",
				Payload: donePayload{SessionID: input.SessionID.String(), TurnsUsed: 1},
			}}
		},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{{
				ID:             uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
				OrganizationID: orgID,
				Name:           "Codex",
				AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
				CliCommand:     "codex",
				Available:      true,
			}},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	stream, err := service.StartTurn(context.Background(), UserID("user:alice"), StartInput{
		Message: "hello",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	events := collectStreamEvents(stream.Events)
	session, ok := events[0].Payload.(sessionPayload)
	if !ok {
		t.Fatalf("expected session payload, got %#v", events[0].Payload)
	}
	if session.ProviderResumeSupported {
		t.Fatalf("expected provider resume unsupported for ephemeral chat, got %+v", session)
	}
	if session.ResumeScope != "process_local" {
		t.Fatalf("expected process_local resume scope, got %+v", session)
	}
}

func TestStartTurnClaudeSessionResumesFromDurableStoreAcrossRestart(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	storePath := filepath.Join(t.TempDir(), "ephemeral-sessions.json")

	buildService := func(runtime *fakeRuntime) *Service {
		service := NewService(
			nil,
			runtime,
			fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             projectID,
					OrganizationID: orgID,
					Name:           "OpenASE",
				},
				providers: []catalogdomain.AgentProvider{{
					ID:             providerID,
					OrganizationID: orgID,
					Name:           "Claude Code",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
					CliCommand:     "claude",
					Available:      true,
				}},
			},
			fakeTicketReader{},
			harnessWorkflowReader{},
			nil,
			"",
		)
		service.EnableDurableSessions(storePath)
		return service
	}

	firstRuntime := &fakeRuntime{
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{
				{
					Event: "session_anchor",
					Payload: RuntimeSessionAnchor{
						ProviderThreadID:      "claude-session-1",
						ProviderAnchorID:      "claude-session-1",
						ProviderAnchorKind:    "session",
						ProviderTurnSupported: false,
					},
				},
				{
					Event: "session_state",
					Payload: runtimeSessionStatePayload{
						Status:      "requires_action",
						ActiveFlags: []string{"requires_action"},
					},
				},
				{
					Event: "done",
					Payload: donePayload{
						SessionID: input.SessionID.String(),
						TurnsUsed: 1,
					},
				},
			}
		},
	}
	firstService := buildService(firstRuntime)

	firstStream, err := firstService.StartTurn(context.Background(), UserID("user:alice"), StartInput{
		Message: "Investigate",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("first StartTurn() error = %v", err)
	}
	firstEvents := collectStreamEvents(firstStream.Events)
	firstSession, ok := firstEvents[0].Payload.(sessionPayload)
	if !ok {
		t.Fatalf("expected session payload, got %#v", firstEvents[0].Payload)
	}
	if !firstSession.ProviderResumeSupported || firstSession.ResumeScope != "host_local" {
		t.Fatalf("unexpected durable session payload: %+v", firstSession)
	}
	sessionID := SessionID(firstSession.SessionID)

	secondRuntime := &fakeRuntime{
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{{
				Event: "done",
				Payload: donePayload{
					SessionID: input.SessionID.String(),
					TurnsUsed: 2,
				},
			}}
		},
	}
	secondService := buildService(secondRuntime)

	secondStream, err := secondService.StartTurn(context.Background(), UserID("user:alice"), StartInput{
		Message:   "Continue",
		Source:    SourceProjectSidebar,
		Context:   Context{ProjectID: projectID},
		SessionID: &sessionID,
	})
	if err != nil {
		t.Fatalf("second StartTurn() error = %v", err)
	}
	_ = collectStreamEvents(secondStream.Events)

	if secondRuntime.lastInput.ResumeProviderThreadID != "claude-session-1" {
		t.Fatalf("resume provider thread id = %q, want claude-session-1", secondRuntime.lastInput.ResumeProviderThreadID)
	}
}

func TestStartTurnRejectsExhaustedSession(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	userID := UserID("user:alice")
	sessionID := SessionID("sess-exhausted")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	service := NewService(
		nil,
		&fakeRuntime{},
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             providerID,
					OrganizationID: orgID,
					Name:           "Claude Code",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
					CliCommand:     "claude",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	service.sessions.Register(userID, sessionID, providerID, DefaultMaxTurns, DefaultMaxBudgetUSD)
	service.sessions.MarkUsage(sessionID, DefaultMaxTurns, nil)
	service.sessions.MarkReleased(sessionID, ErrSessionTurnLimitReached.Error())

	if _, err := service.StartTurn(context.Background(), userID, StartInput{
		Message:   "resume",
		Source:    SourceProjectSidebar,
		Context:   Context{ProjectID: projectID},
		SessionID: &sessionID,
	}); !errors.Is(err, ErrSessionTurnLimitReached) {
		t.Fatalf("StartTurn() exhausted error = %v, want %v", err, ErrSessionTurnLimitReached)
	}
}

func TestStartTurnAllowsUnlimitedProjectSidebarResume(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	userID := UserID("user:alice")
	sessionID := SessionID("sess-project-sidebar")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	runtime := &fakeRuntime{
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{{
				Event: "done",
				Payload: donePayload{
					SessionID:      input.SessionID.String(),
					TurnsUsed:      DefaultMaxTurns + 2,
					TurnsRemaining: nil,
				},
			}}
		},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             providerID,
					OrganizationID: orgID,
					Name:           "Codex",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					CliCommand:     "codex",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	service.sessions.Register(userID, sessionID, providerID, 0, DefaultMaxBudgetUSD)
	service.sessions.MarkUsage(sessionID, DefaultMaxTurns+1, nil)

	stream, err := service.StartTurn(context.Background(), userID, StartInput{
		Message:   "resume",
		Source:    SourceProjectSidebar,
		Context:   Context{ProjectID: projectID},
		SessionID: &sessionID,
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	events := collectStreamEvents(stream.Events)
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2: %+v", len(events), events)
	}
	if runtime.lastInput.MaxTurns != 0 {
		t.Fatalf("runtime max turns = %d, want unlimited resume policy", runtime.lastInput.MaxTurns)
	}
}

func TestStartTurnUsesProjectDefaultEphemeralChatProvider(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440101")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440101")
	defaultProviderID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440101")
	fallbackProviderID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440101")
	runtime := &fakeRuntime{
		streamEvents: []StreamEvent{{
			Event: "done",
			Payload: donePayload{
				SessionID: "sess-default-provider",
				TurnsUsed: 1,
			},
		}},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:                     projectID,
				OrganizationID:         orgID,
				Name:                   "OpenASE",
				DefaultAgentProviderID: &defaultProviderID,
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             defaultProviderID,
					OrganizationID: orgID,
					Name:           "Codex",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					CliCommand:     "codex",
					Available:      true,
				},
				{
					ID:             fallbackProviderID,
					OrganizationID: orgID,
					Name:           "Claude",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
					CliCommand:     "claude",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	stream, err := service.StartTurn(context.Background(), UserID("user:test"), StartInput{
		Message: "Summarize project",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	_ = collectStreamEvents(stream.Events)

	if runtime.lastInput.Provider.ID != defaultProviderID {
		t.Fatalf("provider id = %s, want default %s", runtime.lastInput.Provider.ID, defaultProviderID)
	}
}

func TestStartTurnFallsBackWhenDefaultProviderDoesNotSupportEphemeralChat(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440102")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440102")
	defaultProviderID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440102")
	fallbackProviderID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440102")
	runtime := &fakeRuntime{
		streamEvents: []StreamEvent{{
			Event: "done",
			Payload: donePayload{
				SessionID: "sess-fallback-provider",
				TurnsUsed: 1,
			},
		}},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:                     projectID,
				OrganizationID:         orgID,
				Name:                   "OpenASE",
				DefaultAgentProviderID: &defaultProviderID,
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             defaultProviderID,
					OrganizationID: orgID,
					Name:           "Custom",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeCustom,
					CliCommand:     "custom-chat",
					Available:      true,
				},
				{
					ID:             fallbackProviderID,
					OrganizationID: orgID,
					Name:           "Gemini",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
					CliCommand:     "gemini",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	stream, err := service.StartTurn(context.Background(), UserID("user:test"), StartInput{
		Message: "Summarize project",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	_ = collectStreamEvents(stream.Events)

	if runtime.lastInput.Provider.ID != fallbackProviderID {
		t.Fatalf("provider id = %s, want fallback %s", runtime.lastInput.Provider.ID, fallbackProviderID)
	}
}

func TestStartTurnRejectsExplicitUnsupportedEphemeralChatProvider(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440103")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440103")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440103")
	service := NewService(
		nil,
		&fakeRuntime{},
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             providerID,
					OrganizationID: orgID,
					Name:           "Custom",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeCustom,
					CliCommand:     "custom-chat",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	_, err := service.StartTurn(context.Background(), UserID("user:test"), StartInput{
		Message:    "Summarize project",
		Source:     SourceProjectSidebar,
		ProviderID: &providerID,
		Context:    Context{ProjectID: projectID},
	})
	if !errors.Is(err, ErrProviderUnsupported) {
		t.Fatalf("StartTurn() error = %v, want %v", err, ErrProviderUnsupported)
	}
	if !strings.Contains(err.Error(), "reason=unsupported_adapter") {
		t.Fatalf("expected unsupported reason in error, got %v", err)
	}
}

func TestStartTurnRejectsExplicitUnavailableEphemeralChatProvider(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440104")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440104")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440104")
	service := NewService(
		nil,
		&fakeRuntime{},
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:                 providerID,
					OrganizationID:     orgID,
					Name:               "Codex",
					AdapterType:        catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					CliCommand:         "codex",
					Available:          false,
					AvailabilityReason: stringPointer("machine_offline"),
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)

	_, err := service.StartTurn(context.Background(), UserID("user:test"), StartInput{
		Message:    "Summarize project",
		Source:     SourceProjectSidebar,
		ProviderID: &providerID,
		Context:    Context{ProjectID: projectID},
	})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("StartTurn() error = %v, want %v", err, ErrProviderUnavailable)
	}
	if !strings.Contains(err.Error(), "reason=machine_offline") {
		t.Fatalf("expected unavailable reason in error, got %v", err)
	}
}

func TestStartTurnRejectsResumeAfterBudgetExceeded(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440010")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440010")
	userID := UserID("user:budget")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440010")
	runtime := &fakeRuntime{
		closeResult: true,
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{{
				Event: "done",
				Payload: donePayload{
					SessionID:      input.SessionID.String(),
					CostUSD:        floatPointer(0.11),
					TurnsUsed:      1,
					TurnsRemaining: remainingTurns(input.MaxTurns, 1),
				},
			}}
		},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             providerID,
					OrganizationID: orgID,
					Name:           "Codex",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					CliCommand:     "codex",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)
	service.maxBudgetUSD = 0.10

	stream, err := service.StartTurn(context.Background(), userID, StartInput{
		Message: "first",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	events := collectStreamEvents(stream.Events)
	sessionID := SessionID(events[0].Payload.(sessionPayload).SessionID)

	if len(runtime.closeCalls) != 1 || runtime.closeCalls[0] != sessionID {
		t.Fatalf("runtime close calls = %+v, want session %q to close on budget exhaustion", runtime.closeCalls, sessionID)
	}

	if _, err := service.StartTurn(context.Background(), userID, StartInput{
		Message:   "resume",
		Source:    SourceProjectSidebar,
		Context:   Context{ProjectID: projectID},
		SessionID: &sessionID,
	}); !errors.Is(err, ErrSessionBudgetExceeded) {
		t.Fatalf("resume after budget exhaustion error = %v, want %v", err, ErrSessionBudgetExceeded)
	}
}

func TestStartTurnAllowsResumeWhenProviderSpendUnavailable(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440011")
	orgID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440011")
	userID := UserID("user:no-cost")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440011")
	runtime := &fakeRuntime{
		startFn: func(input RuntimeTurnInput) []StreamEvent {
			return []StreamEvent{{
				Event: "done",
				Payload: donePayload{
					SessionID:      input.SessionID.String(),
					TurnsUsed:      1,
					TurnsRemaining: remainingTurns(input.MaxTurns, 1),
				},
			}}
		},
	}
	service := NewService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: orgID,
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             providerID,
					OrganizationID: orgID,
					Name:           "Gemini",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
					CliCommand:     "gemini",
					Available:      true,
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		"",
	)
	service.maxBudgetUSD = 0.01

	stream, err := service.StartTurn(context.Background(), userID, StartInput{
		Message: "first",
		Source:  SourceProjectSidebar,
		Context: Context{ProjectID: projectID},
	})
	if err != nil {
		t.Fatalf("first StartTurn() error = %v", err)
	}

	events := collectStreamEvents(stream.Events)
	sessionID := SessionID(events[0].Payload.(sessionPayload).SessionID)

	if len(runtime.closeCalls) != 0 {
		t.Fatalf("runtime close calls = %+v, want no budget-driven close without cost telemetry", runtime.closeCalls)
	}

	if _, err := service.StartTurn(context.Background(), userID, StartInput{
		Message:   "resume",
		Source:    SourceProjectSidebar,
		Context:   Context{ProjectID: projectID},
		SessionID: &sessionID,
	}); err != nil {
		t.Fatalf("resume with spend-unavailable provider error = %v, want nil", err)
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

func TestBuildClaudeArgsAddsReasoningEffortWhenMissing(t *testing.T) {
	reasoning := catalogdomain.AgentProviderReasoningEffortMax
	args := buildClaudeArgs(
		[]string{"chat"},
		"claude-sonnet-4-6",
		&reasoning,
		catalogdomain.AgentProviderPermissionProfileStandard,
	)
	if strings.Join(args, " ") != "chat --model claude-sonnet-4-6 --effort max" {
		t.Fatalf("buildClaudeArgs() = %v", args)
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
	detail      workflowservice.WorkflowDetail
	list        []workflowservice.Workflow
	skillDetail workflowservice.SkillDetail
}

func (r harnessWorkflowReader) Get(context.Context, uuid.UUID) (workflowservice.WorkflowDetail, error) {
	return r.detail, nil
}

func (r harnessWorkflowReader) List(context.Context, uuid.UUID) ([]workflowservice.Workflow, error) {
	return r.list, nil
}

func (r harnessWorkflowReader) GetSkill(context.Context, uuid.UUID) (workflowservice.SkillDetail, error) {
	return r.skillDetail, nil
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
	agents           []catalogdomain.Agent
	agentErr         error
	projectRepos     []catalogdomain.ProjectRepo
	projectRepoErr   error
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

func (r fakeCatalogReader) ListActivityEvents(context.Context, catalogdomain.ListActivityEvents) (catalogdomain.ActivityEventPage, error) {
	return catalogdomain.ActivityEventPage{Events: r.activityEvents}, r.activityErr
}

func (r fakeCatalogReader) ListAgents(context.Context, uuid.UUID) ([]catalogdomain.Agent, error) {
	return r.agents, r.agentErr
}

func (r fakeCatalogReader) ListProjectRepos(context.Context, uuid.UUID) ([]catalogdomain.ProjectRepo, error) {
	return r.projectRepos, r.projectRepoErr
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

func (r fakeCatalogReader) CreateAgent(_ context.Context, input catalogdomain.CreateAgent) (catalogdomain.Agent, error) {
	if r.agentErr != nil {
		return catalogdomain.Agent{}, r.agentErr
	}
	return catalogdomain.Agent{
		ID:                  uuid.New(),
		ProviderID:          input.ProviderID,
		ProjectID:           input.ProjectID,
		Name:                input.Name,
		RuntimeControlState: input.RuntimeControlState,
	}, nil
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

type fakeStatusReader struct {
	result ticketstatusservice.ListResult
	err    error
}

func (r fakeStatusReader) List(context.Context, uuid.UUID) (ticketstatusservice.ListResult, error) {
	return r.result, r.err
}

type fakeRuntime struct {
	streamEvents []StreamEvent
	lastInput    RuntimeTurnInput
	closeResult  bool
	startErr     error
	closeCalls   []SessionID
	startFn      func(RuntimeTurnInput) []StreamEvent
	supportFn    func(catalogdomain.AgentProvider) bool
}

func (r *fakeRuntime) Supports(provider catalogdomain.AgentProvider) bool {
	if r.supportFn != nil {
		return r.supportFn(provider)
	}
	return true
}

func (r *fakeRuntime) StartTurn(_ context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if r.startErr != nil {
		return TurnStream{}, r.startErr
	}
	r.lastInput = input
	streamEvents := r.streamEvents
	if r.startFn != nil {
		streamEvents = r.startFn(input)
	}
	events := make(chan StreamEvent, len(streamEvents))
	for _, event := range streamEvents {
		events <- event
	}
	close(events)
	return TurnStream{Events: events}, nil
}

func (r *fakeRuntime) CloseSession(sessionID SessionID) bool {
	r.closeCalls = append(r.closeCalls, sessionID)
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
