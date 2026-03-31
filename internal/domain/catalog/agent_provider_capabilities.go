package catalog

type AgentProviderCapabilities struct {
	EphemeralChat AgentProviderCapability
}

type AgentProviderCapability struct {
	State  AgentProviderCapabilityState
	Reason *string
}

func DeriveAgentProviderCapabilities(item AgentProvider) AgentProvider {
	item.Capabilities = ResolveAgentProviderCapabilities(item)
	return item
}

func ResolveAgentProviderCapabilities(item AgentProvider) AgentProviderCapabilities {
	return AgentProviderCapabilities{
		EphemeralChat: ResolveAgentProviderEphemeralChatCapability(item),
	}
}

func ResolveAgentProviderEphemeralChatCapability(item AgentProvider) AgentProviderCapability {
	if !adapterSupportsEphemeralChat(item.AdapterType) {
		return AgentProviderCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: cloneStringPointer(availabilityReasonPointer(providerReasonUnsupportedAdapter)),
		}
	}

	if item.Available {
		return AgentProviderCapability{
			State: AgentProviderCapabilityStateAvailable,
		}
	}

	reason := cloneStringPointer(item.AvailabilityReason)
	if reason == nil {
		reason = cloneStringPointer(availabilityReasonPointer(providerReasonNotReady))
	}

	return AgentProviderCapability{
		State:  AgentProviderCapabilityStateUnavailable,
		Reason: reason,
	}
}

func adapterSupportsEphemeralChat(adapterType AgentProviderAdapterType) bool {
	switch adapterType {
	case AgentProviderAdapterTypeClaudeCodeCLI,
		AgentProviderAdapterTypeCodexAppServer,
		AgentProviderAdapterTypeGeminiCLI:
		return true
	default:
		return false
	}
}
