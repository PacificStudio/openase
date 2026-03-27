package catalog

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseCreateAgentProviderRequiresMachineID(t *testing.T) {
	_, err := ParseCreateAgentProvider(uuid.New(), AgentProviderInput{
		Name:        "Codex",
		AdapterType: "codex-app-server",
		ModelName:   "gpt-5.4",
	})
	if err == nil || err.Error() != "machine_id must not be empty" {
		t.Fatalf("expected machine_id validation error, got %v", err)
	}
}

func TestParseCreateAgentProviderParsesMachineID(t *testing.T) {
	machineID := uuid.New()
	input, err := ParseCreateAgentProvider(uuid.New(), AgentProviderInput{
		MachineID:   machineID.String(),
		Name:        "Codex",
		AdapterType: "codex-app-server",
		CliCommand:  "codex",
		ModelName:   "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("ParseCreateAgentProvider returned error: %v", err)
	}
	if input.MachineID != machineID {
		t.Fatalf("expected machine_id %s, got %s", machineID, input.MachineID)
	}
}

func TestBuildAgentRuntimeSummaryAggregatesConcurrentRuns(t *testing.T) {
	startedAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	heartbeatAt := startedAt.Add(2 * time.Minute)
	stepStatus := "executing"
	stepSummary := "Applying patch"
	stepChangedAt := startedAt.Add(30 * time.Second)
	runtime := BuildAgentRuntimeSummary(
		[]AgentRun{
			{
				ID:               uuid.New(),
				TicketID:         uuid.New(),
				Status:           AgentRunStatusReady,
				SessionID:        "session-ready",
				RuntimeStartedAt: &startedAt,
				LastHeartbeatAt:  &heartbeatAt,
				CreatedAt:        startedAt,
			},
			{
				ID:                   uuid.New(),
				TicketID:             uuid.New(),
				Status:               AgentRunStatusExecuting,
				SessionID:            "session-executing",
				RuntimeStartedAt:     &startedAt,
				LastHeartbeatAt:      &heartbeatAt,
				CurrentStepStatus:    &stepStatus,
				CurrentStepSummary:   &stepSummary,
				CurrentStepChangedAt: &stepChangedAt,
				CreatedAt:            startedAt.Add(time.Minute),
			},
		},
		AgentRuntimeControlStateActive,
	)
	if runtime == nil {
		t.Fatalf("expected runtime summary, got nil")
	}
	if runtime.ActiveRunCount != 2 {
		t.Fatalf("expected active run count 2, got %+v", runtime)
	}
	if runtime.CurrentRunID != nil || runtime.CurrentTicketID != nil {
		t.Fatalf("expected aggregate summary to clear singular run/ticket ids, got %+v", runtime)
	}
	if runtime.SessionID != "" {
		t.Fatalf("expected aggregate summary to clear singular session, got %+v", runtime)
	}
	if runtime.CurrentStepStatus != nil || runtime.CurrentStepSummary != nil || runtime.CurrentStepChangedAt != nil {
		t.Fatalf("expected aggregate summary to clear singular step fields, got %+v", runtime)
	}
	if runtime.Status != AgentStatusRunning || runtime.RuntimePhase != AgentRuntimePhaseExecuting {
		t.Fatalf("expected executing aggregate summary, got %+v", runtime)
	}
}
