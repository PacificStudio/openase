package catalog

import (
	"fmt"
	"strings"
)

const (
	providerReasonReasoningUnsupported = "reasoning_unsupported"
	providerReasonUnknownModel         = "unknown_model"
)

var agentProviderReasoningEffortOrder = []AgentProviderReasoningEffort{
	AgentProviderReasoningEffortMinimal,
	AgentProviderReasoningEffortLow,
	AgentProviderReasoningEffortMedium,
	AgentProviderReasoningEffortHigh,
	AgentProviderReasoningEffortXHigh,
	AgentProviderReasoningEffortMax,
}

type AgentProviderReasoningCapability struct {
	State                  AgentProviderCapabilityState
	Reason                 *string
	SupportedEfforts       []AgentProviderReasoningEffort
	DefaultEffort          *AgentProviderReasoningEffort
	SelectedEffort         *AgentProviderReasoningEffort
	EffectiveEffort        *AgentProviderReasoningEffort
	SupportsProviderPreset bool
	SupportsModelOverride  bool
}

type AgentProviderModelReasoningCapability struct {
	State                  AgentProviderCapabilityState
	Reason                 *string
	SupportedEfforts       []AgentProviderReasoningEffort
	DefaultEffort          *AgentProviderReasoningEffort
	SupportsProviderPreset bool
	SupportsModelOverride  bool
}

var builtinAgentProviderModelReasoning = map[AgentProviderAdapterType]map[string]AgentProviderModelReasoningCapability{
	AgentProviderAdapterTypeCodexAppServer: {
		"gpt-5.4": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortXHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortMedium),
			SupportsProviderPreset: true,
		},
		"gpt-5.4-mini": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortXHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortMedium),
			SupportsProviderPreset: true,
		},
		"gpt-5.3-codex": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortXHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortMedium),
			SupportsProviderPreset: true,
		},
		"gpt-5.3-codex-spark": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortXHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortHigh),
			SupportsProviderPreset: true,
		},
		"gpt-5.2-codex": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortXHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortMedium),
			SupportsProviderPreset: true,
		},
		"gpt-5.2": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortXHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortMedium),
			SupportsProviderPreset: true,
		},
		"gpt-5.1-codex-max": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortXHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortMedium),
			SupportsProviderPreset: true,
		},
		"gpt-5.1-codex-mini": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh),
			DefaultEffort:          reasoningEffortPointer(AgentProviderReasoningEffortMedium),
			SupportsProviderPreset: true,
		},
	},
	AgentProviderAdapterTypeClaudeCodeCLI: {
		"claude-opus-4-6": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh, AgentProviderReasoningEffortMax),
			SupportsProviderPreset: true,
		},
		"claude-sonnet-4-6": {
			State:                  AgentProviderCapabilityStateAvailable,
			SupportedEfforts:       newReasoningEffortSlice(AgentProviderReasoningEffortLow, AgentProviderReasoningEffortMedium, AgentProviderReasoningEffortHigh),
			SupportsProviderPreset: true,
		},
		"claude-haiku-4-5": {
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: availabilityReasonPointer(providerReasonReasoningUnsupported),
		},
	},
}

func ResolveAgentProviderReasoningCapability(item AgentProvider) AgentProviderReasoningCapability {
	modelCapability := BuiltinAgentProviderModelReasoningCapability(item.AdapterType, item.ModelName)
	capability := AgentProviderReasoningCapability{
		State:                  modelCapability.State,
		Reason:                 cloneStringPointer(modelCapability.Reason),
		SupportedEfforts:       cloneReasoningEffortSlice(modelCapability.SupportedEfforts),
		DefaultEffort:          cloneReasoningEffortPointer(modelCapability.DefaultEffort),
		SupportsProviderPreset: modelCapability.SupportsProviderPreset,
		SupportsModelOverride:  modelCapability.SupportsModelOverride,
	}
	if capability.State != AgentProviderCapabilityStateAvailable {
		return capability
	}

	capability.SelectedEffort = cloneReasoningEffortPointer(item.ReasoningEffort)
	if capability.SelectedEffort != nil {
		capability.EffectiveEffort = cloneReasoningEffortPointer(capability.SelectedEffort)
		return capability
	}

	capability.EffectiveEffort = cloneReasoningEffortPointer(capability.DefaultEffort)
	return capability
}

func BuiltinAgentProviderModelReasoningCapability(
	adapterType AgentProviderAdapterType,
	modelID string,
) AgentProviderModelReasoningCapability {
	normalizedModelID := strings.TrimSpace(modelID)
	if normalizedModelID == "" {
		return AgentProviderModelReasoningCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: availabilityReasonPointer(providerReasonUnknownModel),
		}
	}

	adapterModels := builtinAgentProviderModelReasoning[adapterType]
	if len(adapterModels) == 0 {
		return AgentProviderModelReasoningCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: availabilityReasonPointer(providerReasonReasoningUnsupported),
		}
	}

	capability, ok := adapterModels[normalizedModelID]
	if !ok {
		return AgentProviderModelReasoningCapability{
			State:  AgentProviderCapabilityStateUnsupported,
			Reason: availabilityReasonPointer(providerReasonUnknownModel),
		}
	}

	capability.Reason = cloneStringPointer(capability.Reason)
	capability.SupportedEfforts = cloneReasoningEffortSlice(capability.SupportedEfforts)
	capability.DefaultEffort = cloneReasoningEffortPointer(capability.DefaultEffort)
	return capability
}

func builtinModelReasoningCapabilityPointer(
	adapterType AgentProviderAdapterType,
	modelID string,
) *AgentProviderModelReasoningCapability {
	capability := BuiltinAgentProviderModelReasoningCapability(adapterType, modelID)
	return &capability
}

func parseAgentProviderReasoningEffort(
	adapterType AgentProviderAdapterType,
	modelName string,
	raw *string,
) (*AgentProviderReasoningEffort, error) {
	if raw == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}

	effort := AgentProviderReasoningEffort(strings.ToLower(trimmed))
	if !effort.IsValid() {
		return nil, fmt.Errorf("reasoning_effort must be one of %s", strings.Join(reasoningEffortStrings(agentProviderReasoningEffortOrder), ", "))
	}

	capability := BuiltinAgentProviderModelReasoningCapability(adapterType, modelName)
	if capability.State != AgentProviderCapabilityStateAvailable || !capability.SupportsProviderPreset {
		reason := providerReasonReasoningUnsupported
		if capability.Reason != nil && strings.TrimSpace(*capability.Reason) != "" {
			reason = strings.TrimSpace(*capability.Reason)
		}
		return nil, fmt.Errorf("reasoning_effort is not supported for adapter_type %s model_name %q (%s)", adapterType, strings.TrimSpace(modelName), reason)
	}
	if !reasoningEffortSupported(capability.SupportedEfforts, effort) {
		return nil, fmt.Errorf(
			"reasoning_effort %q is not supported for adapter_type %s model_name %q; allowed values: %s",
			effort,
			adapterType,
			strings.TrimSpace(modelName),
			strings.Join(reasoningEffortStrings(capability.SupportedEfforts), ", "),
		)
	}

	return &effort, nil
}

func ParseStoredAgentProviderReasoningEffort(raw *string) *AgentProviderReasoningEffort {
	if raw == nil {
		return nil
	}

	trimmed := strings.TrimSpace(strings.ToLower(*raw))
	if trimmed == "" {
		return nil
	}

	effort := AgentProviderReasoningEffort(trimmed)
	if !effort.IsValid() {
		return nil
	}
	return &effort
}

func cloneReasoningEffortPointer(
	value *AgentProviderReasoningEffort,
) *AgentProviderReasoningEffort {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneReasoningEffortSlice(
	values []AgentProviderReasoningEffort,
) []AgentProviderReasoningEffort {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]AgentProviderReasoningEffort, len(values))
	copy(cloned, values)
	return cloned
}

func newReasoningEffortSlice(
	values ...AgentProviderReasoningEffort,
) []AgentProviderReasoningEffort {
	return cloneReasoningEffortSlice(values)
}

func reasoningEffortPointer(value AgentProviderReasoningEffort) *AgentProviderReasoningEffort {
	copied := value
	return &copied
}

func reasoningEffortSupported(
	values []AgentProviderReasoningEffort,
	target AgentProviderReasoningEffort,
) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func reasoningEffortStrings(values []AgentProviderReasoningEffort) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.String())
	}
	return result
}
