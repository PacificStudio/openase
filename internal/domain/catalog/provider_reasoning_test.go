package catalog

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuiltinAgentProviderModelReasoningCapabilityCodex(t *testing.T) {
	capability := BuiltinAgentProviderModelReasoningCapability(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
	)
	if capability.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("capability state = %q, want available", capability.State)
	}
	if capability.DefaultEffort == nil || *capability.DefaultEffort != AgentProviderReasoningEffortMedium {
		t.Fatalf("default effort = %+v, want medium", capability.DefaultEffort)
	}
	if got := reasoningEffortStrings(capability.SupportedEfforts); strings.Join(got, ",") != "low,medium,high,xhigh" {
		t.Fatalf("supported efforts = %v", got)
	}
}

func TestBuiltinAgentProviderModelReasoningCapabilityUnknownModel(t *testing.T) {
	capability := BuiltinAgentProviderModelReasoningCapability(
		AgentProviderAdapterTypeCodexAppServer,
		"custom-model",
	)
	if capability.State != AgentProviderCapabilityStateUnsupported {
		t.Fatalf("capability state = %q, want unsupported", capability.State)
	}
	if capability.Reason == nil || *capability.Reason != providerReasonUnknownModel {
		t.Fatalf("capability reason = %+v, want %q", capability.Reason, providerReasonUnknownModel)
	}
}

func TestParseAgentProviderReasoningEffort(t *testing.T) {
	raw := " high "
	parsed, err := parseAgentProviderReasoningEffort(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
		&raw,
	)
	if err != nil {
		t.Fatalf("parseAgentProviderReasoningEffort() error = %v", err)
	}
	if parsed == nil || *parsed != AgentProviderReasoningEffortHigh {
		t.Fatalf("parseAgentProviderReasoningEffort() = %+v, want high", parsed)
	}
}

func TestParseAgentProviderReasoningEffortRejectsUnsupportedCombination(t *testing.T) {
	raw := "max"
	if _, err := parseAgentProviderReasoningEffort(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
		&raw,
	); err == nil || !strings.Contains(err.Error(), `allowed values: low, medium, high, xhigh`) {
		t.Fatalf("parseAgentProviderReasoningEffort() error = %v", err)
	}
}

func TestResolveAgentProviderReasoningCapabilityUsesSelectedEffort(t *testing.T) {
	selected := AgentProviderReasoningEffortHigh
	capability := ResolveAgentProviderReasoningCapability(AgentProvider{
		ID:              uuid.New(),
		AdapterType:     AgentProviderAdapterTypeClaudeCodeCLI,
		ModelName:       "claude-opus-4-6",
		ReasoningEffort: &selected,
	})
	if capability.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("capability state = %q, want available", capability.State)
	}
	if capability.SelectedEffort == nil || *capability.SelectedEffort != AgentProviderReasoningEffortHigh {
		t.Fatalf("selected effort = %+v, want high", capability.SelectedEffort)
	}
	if capability.EffectiveEffort == nil || *capability.EffectiveEffort != AgentProviderReasoningEffortHigh {
		t.Fatalf("effective effort = %+v, want high", capability.EffectiveEffort)
	}
}

func TestParseStoredAgentProviderReasoningEffort(t *testing.T) {
	raw := " XHIGH "
	if got := ParseStoredAgentProviderReasoningEffort(&raw); got == nil || *got != AgentProviderReasoningEffortXHigh {
		t.Fatalf("ParseStoredAgentProviderReasoningEffort() = %+v, want xhigh", got)
	}
	invalid := "invalid"
	if got := ParseStoredAgentProviderReasoningEffort(&invalid); got != nil {
		t.Fatalf("ParseStoredAgentProviderReasoningEffort(invalid) = %+v, want nil", got)
	}
}
