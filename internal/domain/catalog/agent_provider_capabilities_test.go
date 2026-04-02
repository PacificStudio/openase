package catalog

import "testing"

func TestResolveAgentProviderEphemeralChatCapability(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		item       AgentProvider
		wantState  AgentProviderCapabilityState
		wantReason *string
	}{
		{
			name: "available",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeCodexAppServer,
				Available:   true,
			},
			wantState: AgentProviderCapabilityStateAvailable,
		},
		{
			name: "unavailable",
			item: AgentProvider{
				AdapterType:        AgentProviderAdapterTypeClaudeCodeCLI,
				Available:          false,
				AvailabilityReason: stringPtr(providerReasonMachineOffline),
			},
			wantState:  AgentProviderCapabilityStateUnavailable,
			wantReason: stringPtr(providerReasonMachineOffline),
		},
		{
			name: "unsupported",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeCustom,
				Available:   true,
			},
			wantState:  AgentProviderCapabilityStateUnsupported,
			wantReason: stringPtr(providerReasonUnsupportedAdapter),
		},
		{
			name: "unavailable-fallback-reason",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeGeminiCLI,
			},
			wantState:  AgentProviderCapabilityStateUnavailable,
			wantReason: stringPtr(providerReasonNotReady),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveAgentProviderEphemeralChatCapability(tc.item)
			if got.State != tc.wantState {
				t.Fatalf("state = %q, want %q", got.State, tc.wantState)
			}
			switch {
			case tc.wantReason == nil && got.Reason != nil:
				t.Fatalf("reason = %v, want nil", *got.Reason)
			case tc.wantReason != nil && got.Reason == nil:
				t.Fatalf("reason = nil, want %q", *tc.wantReason)
			case tc.wantReason != nil && got.Reason != nil && *got.Reason != *tc.wantReason:
				t.Fatalf("reason = %q, want %q", *got.Reason, *tc.wantReason)
			}
		})
	}
}

func TestResolveAgentProviderHarnessAICapability(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		item       AgentProvider
		wantState  AgentProviderCapabilityState
		wantReason *string
	}{
		{
			name: "available-local-claude",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeClaudeCodeCLI,
				MachineHost: LocalMachineHost,
				Available:   true,
			},
			wantState: AgentProviderCapabilityStateAvailable,
		},
		{
			name: "unavailable-local-gemini",
			item: AgentProvider{
				AdapterType:        AgentProviderAdapterTypeGeminiCLI,
				MachineHost:        LocalMachineHost,
				Available:          false,
				AvailabilityReason: stringPtr(providerReasonMachineOffline),
			},
			wantState:  AgentProviderCapabilityStateUnavailable,
			wantReason: stringPtr(providerReasonMachineOffline),
		},
		{
			name: "unsupported-remote",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeCodexAppServer,
				MachineHost: "gpu-01.internal",
				Available:   true,
			},
			wantState:  AgentProviderCapabilityStateUnsupported,
			wantReason: stringPtr(providerReasonRemoteMachineNotSupported),
		},
		{
			name: "unsupported-adapter",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeCustom,
				MachineHost: LocalMachineHost,
				Available:   true,
			},
			wantState:  AgentProviderCapabilityStateUnsupported,
			wantReason: stringPtr(providerReasonUnsupportedAdapter),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveAgentProviderHarnessAICapability(tc.item)
			if got.State != tc.wantState {
				t.Fatalf("state = %q, want %q", got.State, tc.wantState)
			}
			switch {
			case tc.wantReason == nil && got.Reason != nil:
				t.Fatalf("reason = %v, want nil", *got.Reason)
			case tc.wantReason != nil && got.Reason == nil:
				t.Fatalf("reason = nil, want %q", *tc.wantReason)
			case tc.wantReason != nil && got.Reason != nil && *got.Reason != *tc.wantReason:
				t.Fatalf("reason = %q, want %q", *got.Reason, *tc.wantReason)
			}
		})
	}
}

func TestResolveAgentProviderSkillAICapability(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		item       AgentProvider
		wantState  AgentProviderCapabilityState
		wantReason *string
	}{
		{
			name: "available-local-codex",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeCodexAppServer,
				MachineHost: LocalMachineHost,
				Available:   true,
			},
			wantState: AgentProviderCapabilityStateAvailable,
		},
		{
			name: "unsupported-claude",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeClaudeCodeCLI,
				MachineHost: LocalMachineHost,
				Available:   true,
			},
			wantState:  AgentProviderCapabilityStateUnsupported,
			wantReason: stringPtr(providerReasonSkillAIRequiresCodex),
		},
		{
			name: "unsupported-remote-codex",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeCodexAppServer,
				MachineHost: "10.0.0.10",
				Available:   true,
			},
			wantState:  AgentProviderCapabilityStateUnsupported,
			wantReason: stringPtr(providerReasonRemoteMachineNotSupported),
		},
		{
			name: "unavailable-fallback-reason",
			item: AgentProvider{
				AdapterType: AgentProviderAdapterTypeCodexAppServer,
				MachineHost: LocalMachineHost,
			},
			wantState:  AgentProviderCapabilityStateUnavailable,
			wantReason: stringPtr(providerReasonNotReady),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveAgentProviderSkillAICapability(tc.item)
			if got.State != tc.wantState {
				t.Fatalf("state = %q, want %q", got.State, tc.wantState)
			}
			switch {
			case tc.wantReason == nil && got.Reason != nil:
				t.Fatalf("reason = %v, want nil", *got.Reason)
			case tc.wantReason != nil && got.Reason == nil:
				t.Fatalf("reason = nil, want %q", *tc.wantReason)
			case tc.wantReason != nil && got.Reason != nil && *got.Reason != *tc.wantReason:
				t.Fatalf("reason = %q, want %q", *got.Reason, *tc.wantReason)
			}
		})
	}
}

func TestDeriveAgentProviderCapabilities(t *testing.T) {
	t.Parallel()

	item := DeriveAgentProviderCapabilities(AgentProvider{
		AdapterType: AgentProviderAdapterTypeCodexAppServer,
		MachineHost: LocalMachineHost,
		Available:   true,
	})
	if item.Capabilities.EphemeralChat.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("ephemeral chat capability = %+v", item.Capabilities.EphemeralChat)
	}
	if item.Capabilities.HarnessAI.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("harness ai capability = %+v", item.Capabilities.HarnessAI)
	}
	if item.Capabilities.SkillAI.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("skill ai capability = %+v", item.Capabilities.SkillAI)
	}
}
