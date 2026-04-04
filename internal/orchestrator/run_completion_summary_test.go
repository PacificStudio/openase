package orchestrator

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/google/uuid"
)

func TestBuildRunCompletionSummaryInputIncludesStructuredFacts(t *testing.T) {
	startedAt := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	terminalAt := startedAt.Add(5 * time.Minute)
	summaryCtx := runCompletionSummaryContext{
		run: &ent.AgentRun{
			ID:                uuid.New(),
			Status:            entagentrun.StatusErrored,
			CreatedAt:         startedAt.Add(-30 * time.Second),
			RuntimeStartedAt:  &startedAt,
			TerminalAt:        &terminalAt,
			InputTokens:       111,
			OutputTokens:      222,
			CachedInputTokens: 12,
			ReasoningTokens:   34,
			TotalTokens:       379,
		},
		agent: &ent.Agent{
			ID:   uuid.New(),
			Name: "Ticket Runner",
		},
		project: &ent.Project{
			ID:   uuid.New(),
			Name: "OpenASE",
		},
		ticket: &ent.Ticket{
			ID:         uuid.New(),
			Identifier: "ASE-471",
			Title:      "Add async post-run summaries",
		},
		provider: &ent.AgentProvider{
			ID:          uuid.New(),
			Name:        "Codex",
			AdapterType: entagentprovider.AdapterTypeCodexAppServer,
			ModelName:   "gpt-5.4",
		},
		stepEntries: []*ent.AgentStepEvent{
			{
				StepStatus: "inspect",
				Summary:    "Inspect runtime traces.",
				CreatedAt:  startedAt,
			},
			{
				StepStatus: "retry",
				Summary:    "Retry backend checks after failure.",
				CreatedAt:  startedAt.Add(3 * time.Minute),
			},
		},
		traceEntries: []*ent.AgentTraceEvent{
			{
				Kind:      catalogdomain.AgentTraceKindCommandDelta,
				Text:      "go test ./internal/httpapi",
				Payload:   map[string]any{"command": "go test ./internal/httpapi"},
				CreatedAt: startedAt.Add(10 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindCommandDelta,
				Text:      "go test ./internal/httpapi",
				Payload:   map[string]any{"command": "go test ./internal/httpapi"},
				CreatedAt: startedAt.Add(40 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindCommandDelta,
				Text:      "rm -rf tmp/cache",
				Payload:   map[string]any{"command": "rm -rf tmp/cache"},
				CreatedAt: startedAt.Add(50 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindToolCallStarted,
				Text:      "functions.exec_command",
				Payload:   map[string]any{"tool": "functions.exec_command", "call_id": "call-1", "arguments": map[string]any{"cmd": "go test ./internal/httpapi"}},
				CreatedAt: startedAt.Add(55 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindApprovalRequested,
				Text:      "Approval requested for forceful cleanup.",
				Payload:   map[string]any{"decision": "pending"},
				CreatedAt: startedAt.Add(70 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindReasoningUpdated,
				Text:      "Retrying after transient failure.",
				Stream:    "reasoning",
				CreatedAt: startedAt.Add(80 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindTurnDiffUpdated,
				Text:      "diff --git a/internal/orchestrator/run_completion_summary.go b/internal/orchestrator/run_completion_summary.go",
				Stream:    "diff",
				CreatedAt: startedAt.Add(90 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindCommandDelta,
				Text:      "error: tests failed",
				Stream:    "command",
				CreatedAt: startedAt.Add(100 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindCommandDelta,
				Text:      "error: tests failed",
				Stream:    "command",
				CreatedAt: startedAt.Add(110 * time.Second),
			},
			{
				Kind:      catalogdomain.AgentTraceKindAssistantSnapshot,
				Text:      "Implemented the summary flow but provider startup still failed.",
				Stream:    "assistant",
				CreatedAt: startedAt.Add(115 * time.Second),
			},
		},
	}
	snapshot := runCompletionWorkspaceSnapshot{
		WorkspacePath: "/tmp/openase/workspace",
		Dirty:         true,
		ReposChanged:  1,
		FilesChanged:  2,
		Added:         25,
		Removed:       4,
		Repos: []runCompletionRepoDiff{
			{
				Name:         "openase",
				Path:         "openase",
				Branch:       "feat/openase-471-run-summary",
				Dirty:        true,
				FilesChanged: 2,
				Added:        25,
				Removed:      4,
				Files: []runCompletionFileDiff{
					{Path: ".github/workflows/test.yml", Status: "modified", Added: 3, Removed: 1},
					{Path: "internal/orchestrator/run_completion_summary.go", Status: "modified", Added: 22, Removed: 3},
				},
			},
		},
	}

	input := buildRunCompletionSummaryInput(summaryCtx, snapshot)

	metadata := input["metadata"].(map[string]any)
	if metadata["ticket_identifier"] != "ASE-471" || metadata["provider_adapter"] != string(entagentprovider.AdapterTypeCodexAppServer) {
		t.Fatalf("expected metadata to include ticket/provider facts, got %+v", metadata)
	}

	steps := input["steps"].([]map[string]any)
	if len(steps) != 2 {
		t.Fatalf("expected two ordered steps, got %+v", steps)
	}
	if steps[0]["duration_seconds"] != int64(180) {
		t.Fatalf("expected first step duration to be captured, got %+v", steps[0])
	}

	commands := input["commands"].([]map[string]any)
	if len(commands) < 3 {
		t.Fatalf("expected normalized command facts, got %+v", commands)
	}
	if commands[0]["command"] != "go test ./internal/httpapi" || commands[2]["command"] != "rm -rf tmp/cache" {
		t.Fatalf("expected primary and risky commands to be preserved, got %+v", commands)
	}
	toolCalls := input["tool_calls"].([]map[string]any)
	if len(toolCalls) != 1 || toolCalls[0]["tool"] != "functions.exec_command" {
		t.Fatalf("expected tool call facts, got %+v", toolCalls)
	}
	approvals := input["approvals"].([]map[string]any)
	if len(approvals) != 1 {
		t.Fatalf("expected approval facts, got %+v", approvals)
	}

	outputExcerpts := input["output_excerpts"].([]map[string]any)
	if len(outputExcerpts) == 0 {
		t.Fatalf("expected output excerpts, got %+v", outputExcerpts)
	}

	heuristics := input["heuristics"].(map[string]any)
	if len(heuristics["long_running_steps"].([]map[string]any)) != 2 {
		t.Fatalf("expected long-running step heuristic, got %+v", heuristics["long_running_steps"])
	}
	repeatedCommands := heuristics["repeated_commands"].([]map[string]any)
	if len(repeatedCommands) != 2 || !containsRunCompletionCommand(repeatedCommands, "go test ./internal/httpapi") {
		t.Fatalf("expected repeated command heuristic, got %+v", repeatedCommands)
	}
	repeatedFailures := heuristics["repeated_failures"].([]map[string]any)
	if len(repeatedFailures) != 1 || repeatedFailures[0]["count"] != 2 {
		t.Fatalf("expected repeated failure heuristic, got %+v", repeatedFailures)
	}
	riskyCommands := heuristics["risky_commands"].([]map[string]any)
	if len(riskyCommands) != 1 || riskyCommands[0]["command"] != "rm -rf tmp/cache" {
		t.Fatalf("expected risky command heuristic, got %+v", riskyCommands)
	}
	riskyFiles := heuristics["risky_files"].([]map[string]any)
	if len(riskyFiles) != 1 || riskyFiles[0]["path"] != ".github/workflows/test.yml" {
		t.Fatalf("expected risky file heuristic, got %+v", riskyFiles)
	}

	if input["file_snapshot"].(runCompletionWorkspaceSnapshot).FilesChanged != 2 {
		t.Fatalf("expected stable file snapshot to be embedded, got %+v", input["file_snapshot"])
	}
}

func TestBuildRunCompletionSummaryDeveloperInstructionsUsesProjectOverride(t *testing.T) {
	defaultInstructions := buildRunCompletionSummaryDeveloperInstructions(&ent.Project{})
	if !strings.Contains(defaultInstructions, "## Overview") {
		t.Fatalf("expected built-in default prompt sections, got %q", defaultInstructions)
	}

	override := "Summarize retries first."
	overrideInstructions := buildRunCompletionSummaryDeveloperInstructions(&ent.Project{
		AgentRunSummaryPrompt: override,
	})
	if !strings.Contains(overrideInstructions, override) {
		t.Fatalf("expected project prompt override to be used, got %q", overrideInstructions)
	}
	if strings.Contains(overrideInstructions, "## Overview") {
		t.Fatalf("expected override instructions to replace the default prompt body, got %q", overrideInstructions)
	}
}

func TestPublishRunCompletionSummaryEventEmitsTicketRunSummaryPayload(t *testing.T) {
	bus := eventinfra.NewChannelBus()
	coordinator := newRuntimeCompletionSummaryCoordinator(
		nil,
		nil,
		bus,
		nil,
		nil,
		nil,
		nil,
		func() time.Time { return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC) },
		0,
	)
	projectID := uuid.New()
	ticketID := uuid.New()
	runID := uuid.New()
	status := entagentrun.CompletionSummaryStatusCompleted
	markdown := "## Overview\n\nDone."
	generatedAt := time.Date(2026, 4, 3, 12, 1, 0, 0, time.UTC)
	run := &ent.AgentRun{
		ID:                           runID,
		TicketID:                     ticketID,
		CompletionSummaryStatus:      &status,
		CompletionSummaryMarkdown:    &markdown,
		CompletionSummaryGeneratedAt: &generatedAt,
		CompletionSummaryJSON:        map[string]any{"provider": "Codex"},
	}

	stream, err := bus.Subscribe(context.Background(), ticketRunSummaryStreamTopic)
	if err != nil {
		t.Fatalf("subscribe summary stream: %v", err)
	}
	if err := coordinator.publishRunCompletionSummaryEvent(context.Background(), projectID, run); err != nil {
		t.Fatalf("publish run completion summary event: %v", err)
	}

	select {
	case event := <-stream:
		if event.Topic != ticketRunSummaryStreamTopic {
			t.Fatalf("event topic = %s, want %s", event.Topic, ticketRunSummaryStreamTopic)
		}
		if event.Type != ticketRunSummaryStreamType {
			t.Fatalf("event type = %s, want %s", event.Type, ticketRunSummaryStreamType)
		}
		var payload ticketRunCompletionSummaryStreamPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode summary payload: %v", err)
		}
		if payload.ProjectID != projectID.String() || payload.TicketID != ticketID.String() || payload.RunID != runID.String() {
			t.Fatalf("unexpected ids in payload: %+v", payload)
		}
		if payload.CompletionSummary.Status != "completed" || payload.CompletionSummary.Markdown == nil || *payload.CompletionSummary.Markdown != markdown {
			t.Fatalf("unexpected completion summary payload: %+v", payload.CompletionSummary)
		}
		if payload.CompletionSummary.GeneratedAt == nil || *payload.CompletionSummary.GeneratedAt != generatedAt.Format(time.RFC3339) {
			t.Fatalf("generated_at = %+v, want %s", payload.CompletionSummary.GeneratedAt, generatedAt.Format(time.RFC3339))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for summary event")
	}
}

func containsRunCompletionCommand(items []map[string]any, command string) bool {
	for _, item := range items {
		if item["command"] == command {
			return true
		}
	}
	return false
}
