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

func TestDeriveAgentProviderCapabilities(t *testing.T) {
	t.Parallel()

	item := DeriveAgentProviderCapabilities(AgentProvider{
		AdapterType: AgentProviderAdapterTypeCodexAppServer,
		Available:   true,
	})
	if item.Capabilities.EphemeralChat.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("ephemeral chat capability = %+v", item.Capabilities.EphemeralChat)
	}
}
