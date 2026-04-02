package catalog

import "strings"

type AgentProviderCapabilities struct {
	EphemeralChat AgentProviderCapability
	HarnessAI     AgentProviderCapability
	SkillAI       AgentProviderCapability
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
		HarnessAI:     ResolveAgentProviderHarnessAICapability(item),
		SkillAI:       ResolveAgentProviderSkillAICapability(item),
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

func ResolveAgentProviderHarnessAICapability(item AgentProvider) AgentProviderCapability {
	if !adapterSupportsEphemeralChat(item.AdapterType) {
		return AgentProviderCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: cloneStringPointer(availabilityReasonPointer(providerReasonUnsupportedAdapter)),
		}
	}
	if !providerRunsOnLocalMachine(item) {
		return AgentProviderCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: cloneStringPointer(availabilityReasonPointer(providerReasonRemoteMachineNotSupported)),
		}
	}
	return availabilityBackedCapability(item)
}

func ResolveAgentProviderSkillAICapability(item AgentProvider) AgentProviderCapability {
	if item.AdapterType != AgentProviderAdapterTypeCodexAppServer {
		return AgentProviderCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: cloneStringPointer(availabilityReasonPointer(providerReasonSkillAIRequiresCodex)),
		}
	}
	if !providerRunsOnLocalMachine(item) {
		return AgentProviderCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: cloneStringPointer(availabilityReasonPointer(providerReasonRemoteMachineNotSupported)),
		}
	}
	return availabilityBackedCapability(item)
}

func availabilityBackedCapability(item AgentProvider) AgentProviderCapability {
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

func providerRunsOnLocalMachine(item AgentProvider) bool {
	host := strings.TrimSpace(strings.ToLower(item.MachineHost))
	switch host {
	case "", LocalMachineHost, "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
