package chat

import (
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestProviderSurfaceSupportMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		surface    providerSurface
		provider   catalogdomain.AgentProvider
		wantState  catalogdomain.AgentProviderCapabilityState
		wantReason *string
	}{
		{
			name:    "harness/local-codex",
			surface: providerSurfaceHarnessAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
				MachineHost: catalogdomain.LocalMachineHost,
				Available:   true,
			},
			wantState: catalogdomain.AgentProviderCapabilityStateAvailable,
		},
		{
			name:    "harness/local-claude",
			surface: providerSurfaceHarnessAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
				MachineHost: catalogdomain.LocalMachineHost,
				Available:   true,
			},
			wantState: catalogdomain.AgentProviderCapabilityStateAvailable,
		},
		{
			name:    "harness/local-gemini",
			surface: providerSurfaceHarnessAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeGeminiCLI,
				MachineHost: catalogdomain.LocalMachineHost,
				Available:   true,
			},
			wantState: catalogdomain.AgentProviderCapabilityStateAvailable,
		},
		{
			name:    "harness/remote-codex",
			surface: providerSurfaceHarnessAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
				MachineHost: "10.0.0.12",
				Available:   true,
			},
			wantState:  catalogdomain.AgentProviderCapabilityStateUnsupported,
			wantReason: testStringPointer("remote_machine_not_supported"),
		},
		{
			name:    "skill/local-codex",
			surface: providerSurfaceSkillAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
				MachineHost: catalogdomain.LocalMachineHost,
				Available:   true,
			},
			wantState: catalogdomain.AgentProviderCapabilityStateAvailable,
		},
		{
			name:    "skill/local-claude",
			surface: providerSurfaceSkillAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
				MachineHost: catalogdomain.LocalMachineHost,
				Available:   true,
			},
			wantState:  catalogdomain.AgentProviderCapabilityStateUnsupported,
			wantReason: testStringPointer("skill_ai_requires_codex"),
		},
		{
			name:    "skill/local-gemini",
			surface: providerSurfaceSkillAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeGeminiCLI,
				MachineHost: catalogdomain.LocalMachineHost,
				Available:   true,
			},
			wantState:  catalogdomain.AgentProviderCapabilityStateUnsupported,
			wantReason: testStringPointer("skill_ai_requires_codex"),
		},
		{
			name:    "skill/remote-codex",
			surface: providerSurfaceSkillAI,
			provider: catalogdomain.AgentProvider{
				AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
				MachineHost: "10.0.0.13",
				Available:   true,
			},
			wantState:  catalogdomain.AgentProviderCapabilityStateUnsupported,
			wantReason: testStringPointer("remote_machine_not_supported"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capability := resolveProviderCapabilityForSurface(tc.provider, tc.surface)
			if capability.State != tc.wantState {
				t.Fatalf("state = %q, want %q", capability.State, tc.wantState)
			}
			switch {
			case tc.wantReason == nil && capability.Reason != nil:
				t.Fatalf("reason = %q, want nil", *capability.Reason)
			case tc.wantReason != nil && capability.Reason == nil:
				t.Fatalf("reason = nil, want %q", *tc.wantReason)
			case tc.wantReason != nil && capability.Reason != nil && *capability.Reason != *tc.wantReason:
				t.Fatalf("reason = %q, want %q", *capability.Reason, *tc.wantReason)
			}
		})
	}
}

func testStringPointer(value string) *string {
	return &value
}
