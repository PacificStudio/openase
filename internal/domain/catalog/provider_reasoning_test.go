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

func TestBuiltinAgentProviderModelReasoningCapabilityClaudeSonnet(t *testing.T) {
	capability := BuiltinAgentProviderModelReasoningCapability(
		AgentProviderAdapterTypeClaudeCodeCLI,
		"claude-sonnet-4-6",
	)
	if capability.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("capability state = %q, want available", capability.State)
	}
	if capability.DefaultEffort != nil {
		t.Fatalf("default effort = %+v, want nil because Claude defaults depend on account plan", capability.DefaultEffort)
	}
	if got := reasoningEffortStrings(capability.SupportedEfforts); strings.Join(got, ",") != "low,medium,high" {
		t.Fatalf("supported efforts = %v", got)
	}
}

func TestBuiltinAgentProviderModelReasoningCapabilityClaudeHaikuUnsupported(t *testing.T) {
	capability := BuiltinAgentProviderModelReasoningCapability(
		AgentProviderAdapterTypeClaudeCodeCLI,
		"claude-haiku-4-5",
	)
	if capability.State != AgentProviderCapabilityStateUnsupported {
		t.Fatalf("capability state = %q, want unsupported", capability.State)
	}
	if capability.Reason == nil || *capability.Reason != providerReasonReasoningUnsupported {
		t.Fatalf("capability reason = %+v, want %q", capability.Reason, providerReasonReasoningUnsupported)
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

func TestParseAgentProviderReasoningEffortHandlesEmptyInput(t *testing.T) {
	if got, err := parseAgentProviderReasoningEffort(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
		nil,
	); err != nil || got != nil {
		t.Fatalf("parseAgentProviderReasoningEffort(nil) = (%v, %v), want (nil, nil)", got, err)
	}

	raw := "   "
	if got, err := parseAgentProviderReasoningEffort(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
		&raw,
	); err != nil || got != nil {
		t.Fatalf("parseAgentProviderReasoningEffort(blank) = (%v, %v), want (nil, nil)", got, err)
	}
}

func TestParseAgentProviderReasoningEffortRejectsInvalidValue(t *testing.T) {
	raw := "turbo"
	if _, err := parseAgentProviderReasoningEffort(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
		&raw,
	); err == nil || !strings.Contains(err.Error(), "reasoning_effort must be one of minimal, low, medium, high, xhigh, max") {
		t.Fatalf("parseAgentProviderReasoningEffort() error = %v", err)
	}
}

func TestParseAgentProviderReasoningEffortRejectsUnsupportedProviderPreset(t *testing.T) {
	raw := "high"
	if _, err := parseAgentProviderReasoningEffort(
		AgentProviderAdapterTypeCustom,
		"custom-model",
		&raw,
	); err == nil || !strings.Contains(err.Error(), "(reasoning_unsupported)") {
		t.Fatalf("parseAgentProviderReasoningEffort(custom) error = %v", err)
	}

	if _, err := parseAgentProviderReasoningEffort(
		AgentProviderAdapterTypeCodexAppServer,
		"missing-model",
		&raw,
	); err == nil || !strings.Contains(err.Error(), "(unknown_model)") {
		t.Fatalf("parseAgentProviderReasoningEffort(unknown model) error = %v", err)
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

func TestResolveAgentProviderReasoningCapabilityLeavesUnknownClaudeDefaultUnset(t *testing.T) {
	capability := ResolveAgentProviderReasoningCapability(AgentProvider{
		ID:          uuid.New(),
		AdapterType: AgentProviderAdapterTypeClaudeCodeCLI,
		ModelName:   "claude-sonnet-4-6",
	})
	if capability.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("capability state = %q, want available", capability.State)
	}
	if capability.DefaultEffort != nil {
		t.Fatalf("default effort = %+v, want nil", capability.DefaultEffort)
	}
	if capability.EffectiveEffort != nil {
		t.Fatalf("effective effort = %+v, want nil when no preset is selected", capability.EffectiveEffort)
	}
}

func TestParseStoredAgentProviderReasoningEffort(t *testing.T) {
	raw := " XHIGH "
	if got := ParseStoredAgentProviderReasoningEffort(&raw); got == nil || *got != AgentProviderReasoningEffortXHigh {
		t.Fatalf("ParseStoredAgentProviderReasoningEffort() = %+v, want xhigh", got)
	}
	blank := "   "
	if got := ParseStoredAgentProviderReasoningEffort(&blank); got != nil {
		t.Fatalf("ParseStoredAgentProviderReasoningEffort(blank) = %+v, want nil", got)
	}
	invalid := "invalid"
	if got := ParseStoredAgentProviderReasoningEffort(&invalid); got != nil {
		t.Fatalf("ParseStoredAgentProviderReasoningEffort(invalid) = %+v, want nil", got)
	}
	if got := ParseStoredAgentProviderReasoningEffort(nil); got != nil {
		t.Fatalf("ParseStoredAgentProviderReasoningEffort(nil) = %+v, want nil", got)
	}
}

func TestReasoningEffortStringsHandlesEmptySlice(t *testing.T) {
	if got := reasoningEffortStrings(nil); got != nil {
		t.Fatalf("reasoningEffortStrings(nil) = %v, want nil", got)
	}
}
