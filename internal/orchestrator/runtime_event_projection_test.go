package orchestrator

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	entagentactivityinstance "github.com/BetterAndBetterII/openase/ent/agentactivityinstance"
	entagentrawevent "github.com/BetterAndBetterII/openase/ent/agentrawevent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagenttranscriptentry "github.com/BetterAndBetterII/openase/ent/agenttranscriptentry"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestProjectRuntimeEventPersistsClaudeRawEventWithoutNULDedupKey(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, time.April, 15, 10, 30, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Claude raw events").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-RAW-CLAUDE").
		SetTitle("Persist Claude raw events safely").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "claude-raw-01", 0)
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	rawPayload, err := json.Marshal(map[string]any{
		"type":        "task_progress",
		"session_id":  "claude-session-raw-1",
		"turn_id":     "turn-raw-1",
		"tool_use_id": "toolu-raw-1",
		"stream":      "command",
		"command":     "pwd",
		"text":        "/repo\n",
		"snapshot":    true,
	})
	if err != nil {
		t.Fatalf("marshal raw payload: %v", err)
	}

	raw := rawClaudeProviderEvent(provider.ClaudeCodeEvent{
		Kind:      provider.ClaudeCodeEventKindTaskProgress,
		SessionID: "claude-session-raw-1",
		UUID:      "event-raw-1",
		Raw:       rawPayload,
	})
	if raw == nil {
		t.Fatal("expected raw Claude provider event")
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil, nil)
	if err := launcher.projectRuntimeEvent(ctx, runtimeEventProjectionInput{
		ProjectID:  fixture.projectID,
		AgentID:    agentItem.ID,
		TicketID:   ticketItem.ID,
		RunID:      runItem.ID,
		Provider:   "claude",
		ObservedAt: now,
		Event: agentEvent{
			Raw: raw,
		},
	}); err != nil {
		t.Fatalf("projectRuntimeEvent() error = %v", err)
	}

	rawEvent, err := client.AgentRawEvent.Query().
		Where(entagentrawevent.AgentRunIDEQ(runItem.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query raw event: %v", err)
	}
	if strings.Contains(rawEvent.DedupKey, "\x00") {
		t.Fatalf("dedup key must not contain NUL bytes: %q", rawEvent.DedupKey)
	}
	if rawEvent.Provider != "claude" {
		t.Fatalf("provider = %q, want claude", rawEvent.Provider)
	}
}

func TestProjectRuntimeEventSanitizesCommandOutputBeforePersistence(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, time.April, 19, 14, 0, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Command output projection").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-RUNTIME-OUTPUT").
		SetTitle("Sanitize command output persistence").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "codex-output-01", 0)
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil, nil)

	events := []runtimeEventProjectionInput{
		{
			ProjectID:  fixture.projectID,
			AgentID:    agentItem.ID,
			TicketID:   ticketItem.ID,
			RunID:      runItem.ID,
			Provider:   "codex",
			ObservedAt: now,
			Event: agentEvent{
				Type: agentEventTypeOutputProduced,
				Output: &agentOutputEvent{
					ThreadID: "thread-output-1",
					TurnID:   "turn-output-1",
					ItemID:   "command-output-1",
					Stream:   "command",
					Text:     string([]byte{'o', 'k', 0, 'b', 'a', 'd', 0xff, '\n'}),
				},
			},
		},
		{
			ProjectID:  fixture.projectID,
			AgentID:    agentItem.ID,
			TicketID:   ticketItem.ID,
			RunID:      runItem.ID,
			Provider:   "codex",
			ObservedAt: now.Add(time.Second),
			Event: agentEvent{
				Type: agentEventTypeOutputProduced,
				Output: &agentOutputEvent{
					ThreadID: "thread-output-1",
					TurnID:   "turn-output-1",
					ItemID:   "command-output-1",
					Stream:   "command",
					Text:     string([]byte{'f', 'i', 'n', 'a', 'l', 0, '-', 'o', 'u', 't', 0xfe, '\n'}),
					Snapshot: true,
				},
			},
		},
	}
	for _, input := range events {
		if err := launcher.projectRuntimeEvent(ctx, input); err != nil {
			t.Fatalf("projectRuntimeEvent() error = %v", err)
		}
	}

	activity, err := client.AgentActivityInstance.Query().
		Where(
			entagentactivityinstance.AgentRunIDEQ(runItem.ID),
			entagentactivityinstance.ActivityKindEQ(catalogdomain.AgentActivityKindCommandExecution),
			entagentactivityinstance.ActivityIDEQ("command-output-1"),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query activity: %v", err)
	}
	if got, want := stringOrEmpty(activity.LiveText), "final-out"; got != want {
		t.Fatalf("live_text = %q, want %q", got, want)
	}
	if got, want := stringOrEmpty(activity.FinalText), "final-out"; got != want {
		t.Fatalf("final_text = %q, want %q", got, want)
	}
	if got, want := stringOrEmpty(activity.Title), "okbad"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	for fieldName, value := range map[string]string{
		"live_text":  stringOrEmpty(activity.LiveText),
		"final_text": stringOrEmpty(activity.FinalText),
		"title":      stringOrEmpty(activity.Title),
	} {
		if strings.Contains(value, "\x00") {
			t.Fatalf("%s must not contain NUL bytes: %q", fieldName, value)
		}
		if !utf8.ValidString(value) {
			t.Fatalf("%s must be valid UTF-8: %q", fieldName, value)
		}
	}

	entry, err := client.AgentTranscriptEntry.Query().
		Where(
			entagenttranscriptentry.AgentRunIDEQ(runItem.ID),
			entagenttranscriptentry.EntryKeyEQ("command_completed:command-output-1"),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query transcript entry: %v", err)
	}
	if got, want := stringOrEmpty(entry.BodyText), "final-out"; got != want {
		t.Fatalf("body_text = %q, want %q", got, want)
	}
	if got, want := stringOrEmpty(entry.Summary), "final-out"; got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
	if got, want := stringOrEmpty(entry.Title), "final-out"; got != want {
		t.Fatalf("transcript title = %q, want %q", got, want)
	}
	for fieldName, value := range map[string]string{
		"body_text": stringOrEmpty(entry.BodyText),
		"summary":   stringOrEmpty(entry.Summary),
		"title":     stringOrEmpty(entry.Title),
	} {
		if strings.Contains(value, "\x00") {
			t.Fatalf("transcript %s must not contain NUL bytes: %q", fieldName, value)
		}
		if !utf8.ValidString(value) {
			t.Fatalf("transcript %s must be valid UTF-8: %q", fieldName, value)
		}
	}
}
