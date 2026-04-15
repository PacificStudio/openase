package orchestrator

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	entagentrawevent "github.com/BetterAndBetterII/openase/ent/agentrawevent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
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
