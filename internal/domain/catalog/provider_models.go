package catalog

import "github.com/BetterAndBetterII/openase/internal/domain/pricing"

type AgentProviderModelOption struct {
	ID            string
	Label         string
	Description   string
	Recommended   bool
	Preview       bool
	PricingConfig *pricing.ProviderModelPricingConfig
	Reasoning     *AgentProviderModelReasoningCapability
}

var builtinAgentProviderModelOptionAdapters = []AgentProviderAdapterType{
	AgentProviderAdapterTypeCodexAppServer,
	AgentProviderAdapterTypeClaudeCodeCLI,
	AgentProviderAdapterTypeGeminiCLI,
}

var builtinAgentProviderModelOptions = map[AgentProviderAdapterType][]AgentProviderModelOption{
	AgentProviderAdapterTypeCodexAppServer: {
		{
			ID:          "gpt-5.4",
			Label:       "gpt-5.4",
			Description: "Latest frontier agentic coding model.",
			Recommended: true,
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.4",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.4",
			),
		},
		{
			ID:          "gpt-5.4-mini",
			Label:       "gpt-5.4-mini",
			Description: "Smaller frontier agentic coding model.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.4-mini",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.4-mini",
			),
		},
		{
			ID:          "gpt-5.3-codex",
			Label:       "gpt-5.3-codex",
			Description: "Frontier Codex-optimized agentic coding model.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.3-codex",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.3-codex",
			),
		},
		{
			ID:          "gpt-5.3-codex-spark",
			Label:       "gpt-5.3-codex-spark",
			Description: "Ultra-fast coding model.",
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.3-codex-spark",
			),
		},
		{
			ID:          "gpt-5.2-codex",
			Label:       "gpt-5.2-codex",
			Description: "Frontier agentic coding model.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.2-codex",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.2-codex",
			),
		},
		{
			ID:          "gpt-5.2",
			Label:       "gpt-5.2",
			Description: "Optimized for professional work and long-running agents.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.2",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.2",
			),
		},
		{
			ID:          "gpt-5.1-codex-max",
			Label:       "gpt-5.1-codex-max",
			Description: "Codex-optimized model for deep and fast reasoning.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.1-codex-max",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.1-codex-max",
			),
		},
		{
			ID:          "gpt-5.1-codex-mini",
			Label:       "gpt-5.1-codex-mini",
			Description: "Optimized for Codex. Cheaper, faster, but less capable.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.1-codex-mini",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeCodexAppServer,
				"gpt-5.1-codex-mini",
			),
		},
	},
	AgentProviderAdapterTypeClaudeCodeCLI: {
		{
			ID:          "claude-opus-4-6",
			Label:       "Default",
			Description: "Opus 4.6 with 1M context. Most capable for complex work.",
			Recommended: true,
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeClaudeCodeCLI,
				"claude-opus-4-6",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeClaudeCodeCLI,
				"claude-opus-4-6",
			),
		},
		{
			ID:          "claude-sonnet-4-6",
			Label:       "Sonnet",
			Description: "Sonnet 4.6. Best for everyday tasks.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeClaudeCodeCLI,
				"claude-sonnet-4-6",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeClaudeCodeCLI,
				"claude-sonnet-4-6",
			),
		},
		{
			ID:          "claude-haiku-4-5",
			Label:       "Haiku",
			Description: "Haiku 4.5. Fastest for quick answers.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeClaudeCodeCLI,
				"claude-haiku-4-5",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeClaudeCodeCLI,
				"claude-haiku-4-5",
			),
		},
	},
	AgentProviderAdapterTypeGeminiCLI: {
		{
			ID:          "auto-gemini-2.5",
			Label:       "Auto (Gemini 2.5)",
			Description: "Let Gemini CLI decide between gemini-2.5-pro and gemini-2.5-flash.",
			Recommended: true,
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"auto-gemini-2.5",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"auto-gemini-2.5",
			),
		},
		{
			ID:          "gemini-2.5-pro",
			Label:       "gemini-2.5-pro",
			Description: "Stable Gemini 2.5 Pro model.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-2.5-pro",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-2.5-pro",
			),
		},
		{
			ID:          "gemini-2.5-flash",
			Label:       "gemini-2.5-flash",
			Description: "Stable Gemini 2.5 Flash model.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-2.5-flash",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-2.5-flash",
			),
		},
		{
			ID:          "gemini-2.5-flash-lite",
			Label:       "gemini-2.5-flash-lite",
			Description: "Stable Gemini 2.5 Flash Lite model.",
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-2.5-flash-lite",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-2.5-flash-lite",
			),
		},
		{
			ID:          "auto-gemini-3",
			Label:       "Auto (Gemini 3)",
			Description: "Preview routing for Gemini 3 models such as gemini-3.1-pro-preview and gemini-3-flash-preview.",
			Preview:     true,
			PricingConfig: builtinPricingConfigPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"auto-gemini-3",
			),
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"auto-gemini-3",
			),
		},
		{
			ID:          "gemini-3-flash-preview",
			Label:       "gemini-3-flash-preview",
			Description: "Preview Gemini 3 Flash model.",
			Preview:     true,
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-3-flash-preview",
			),
		},
		{
			ID:          "gemini-3.1-pro-preview",
			Label:       "gemini-3.1-pro-preview",
			Description: "Preview Gemini 3.1 Pro model.",
			Preview:     true,
			Reasoning: builtinModelReasoningCapabilityPointer(
				AgentProviderAdapterTypeGeminiCLI,
				"gemini-3.1-pro-preview",
			),
		},
	},
}

func BuiltinAgentProviderModelOptions(adapterType AgentProviderAdapterType) []AgentProviderModelOption {
	options := builtinAgentProviderModelOptions[adapterType]
	if len(options) == 0 {
		return nil
	}

	cloned := make([]AgentProviderModelOption, len(options))
	copy(cloned, options)
	for index := range cloned {
		if cloned[index].PricingConfig != nil {
			config := cloned[index].PricingConfig.Clone()
			cloned[index].PricingConfig = &config
		}
		if cloned[index].Reasoning != nil {
			reasoning := *cloned[index].Reasoning
			reasoning.Reason = cloneStringPointer(reasoning.Reason)
			reasoning.SupportedEfforts = cloneReasoningEffortSlice(reasoning.SupportedEfforts)
			reasoning.DefaultEffort = cloneReasoningEffortPointer(reasoning.DefaultEffort)
			cloned[index].Reasoning = &reasoning
		}
	}
	return cloned
}

func BuiltinAgentProviderAdaptersWithModelOptions() []AgentProviderAdapterType {
	cloned := make([]AgentProviderAdapterType, len(builtinAgentProviderModelOptionAdapters))
	copy(cloned, builtinAgentProviderModelOptionAdapters)
	return cloned
}

func builtinPricingConfigPointer(
	adapterType AgentProviderAdapterType,
	modelID string,
) *pricing.ProviderModelPricingConfig {
	config, ok := BuiltinAgentProviderPricingConfig(adapterType, modelID)
	if !ok {
		return nil
	}
	return &config
}
