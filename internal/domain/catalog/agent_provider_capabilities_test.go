package catalog

import "testing"

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
