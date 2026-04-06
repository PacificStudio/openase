package catalog

import "testing"

func TestDeriveAgentProviderCapabilitiesCopiesResolvedState(t *testing.T) {
	reason := providerReasonConfigIncomplete
	item := DeriveAgentProviderCapabilities(AgentProvider{
		AdapterType:        AgentProviderAdapterTypeCodexAppServer,
		AvailabilityReason: &reason,
	})

	if item.Capabilities.EphemeralChat.State != AgentProviderCapabilityStateUnavailable {
		t.Fatalf("ephemeral chat capability = %+v", item.Capabilities.EphemeralChat)
	}
	if item.Capabilities.EphemeralChat.Reason == nil || *item.Capabilities.EphemeralChat.Reason != reason {
		t.Fatalf("ephemeral chat reason = %+v", item.Capabilities.EphemeralChat.Reason)
	}
}

func TestResolveAgentProviderCapabilitiesUsesEphemeralChatOnly(t *testing.T) {
	item := ResolveAgentProviderCapabilities(AgentProvider{
		AdapterType: AgentProviderAdapterTypeCodexAppServer,
		Available:   true,
	})

	if item.EphemeralChat.State != AgentProviderCapabilityStateAvailable {
		t.Fatalf("ephemeral chat capability = %+v", item.EphemeralChat)
	}
}

func TestResolveAgentProviderCapabilitiesMarksUnsupportedAdapter(t *testing.T) {
	item := ResolveAgentProviderCapabilities(AgentProvider{
		AdapterType: AgentProviderAdapterTypeCustom,
	})

	if item.EphemeralChat.State != AgentProviderCapabilityStateUnsupported {
		t.Fatalf("ephemeral chat capability = %+v", item.EphemeralChat)
	}
	if item.EphemeralChat.Reason == nil || *item.EphemeralChat.Reason != providerReasonUnsupportedAdapter {
		t.Fatalf("ephemeral chat reason = %+v", item.EphemeralChat.Reason)
	}
}

func TestResolveAgentProviderCapabilitiesDefaultsUnavailableReason(t *testing.T) {
	item := ResolveAgentProviderCapabilities(AgentProvider{
		AdapterType: AgentProviderAdapterTypeGeminiCLI,
	})

	if item.EphemeralChat.State != AgentProviderCapabilityStateUnavailable {
		t.Fatalf("ephemeral chat capability = %+v", item.EphemeralChat)
	}
	if item.EphemeralChat.Reason == nil || *item.EphemeralChat.Reason != providerReasonNotReady {
		t.Fatalf("ephemeral chat reason = %+v", item.EphemeralChat.Reason)
	}
}
