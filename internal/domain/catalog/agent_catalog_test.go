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

func TestParseCreateAgentProviderParsesPricingConfig(t *testing.T) {
	machineID := uuid.New()
	input, err := ParseCreateAgentProvider(uuid.New(), AgentProviderInput{
		MachineID:   machineID.String(),
		Name:        "Codex",
		AdapterType: "codex-app-server",
		CliCommand:  "codex",
		ModelName:   "gpt-5.4",
		PricingConfig: map[string]any{
			"source_kind":  "official",
			"pricing_mode": "flat",
			"rates": map[string]any{
				"input_per_token":  0.75,
				"output_per_token": 4.5,
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseCreateAgentProvider returned error: %v", err)
	}
	if input.CostPerInputToken != 0.75 || input.CostPerOutputToken != 4.5 {
		t.Fatalf("expected pricing summary rates from pricing_config, got %+v", input)
	}
	if input.PricingConfig.SourceKind != "official" || input.PricingConfig.PricingMode != "flat" {
		t.Fatalf("expected parsed pricing config, got %+v", input.PricingConfig)
	}
}

func TestParseCreateAgentProviderRejectsInvalidPricingConfig(t *testing.T) {
	machineID := uuid.New()
	_, err := ParseCreateAgentProvider(uuid.New(), AgentProviderInput{
		MachineID:   machineID.String(),
		Name:        "Codex",
		AdapterType: "codex-app-server",
		CliCommand:  "codex",
		ModelName:   "gpt-5.4",
		PricingConfig: map[string]any{
			"rates": map[string]any{
				"input_per_token": -1,
			},
		},
	})
	if err == nil || err.Error() != "pricing_config.rates.input_per_token must be greater than or equal to zero" {
		t.Fatalf("expected pricing_config validation error, got %v", err)
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
