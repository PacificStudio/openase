package catalog

type AgentProviderModelOption struct {
	ID          string
	Label       string
	Description string
	Recommended bool
	Preview     bool
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
		},
		{
			ID:          "gpt-5.4-mini",
			Label:       "gpt-5.4-mini",
			Description: "Smaller frontier agentic coding model.",
		},
		{
			ID:          "gpt-5.3-codex",
			Label:       "gpt-5.3-codex",
			Description: "Frontier Codex-optimized agentic coding model.",
		},
		{
			ID:          "gpt-5.3-codex-spark",
			Label:       "gpt-5.3-codex-spark",
			Description: "Ultra-fast coding model.",
		},
		{
			ID:          "gpt-5.2-codex",
			Label:       "gpt-5.2-codex",
			Description: "Frontier agentic coding model.",
		},
		{
			ID:          "gpt-5.2",
			Label:       "gpt-5.2",
			Description: "Optimized for professional work and long-running agents.",
		},
		{
			ID:          "gpt-5.1-codex-max",
			Label:       "gpt-5.1-codex-max",
			Description: "Codex-optimized model for deep and fast reasoning.",
		},
		{
			ID:          "gpt-5.1-codex-mini",
			Label:       "gpt-5.1-codex-mini",
			Description: "Optimized for Codex. Cheaper, faster, but less capable.",
		},
	},
	AgentProviderAdapterTypeClaudeCodeCLI: {
		{
			ID:          "claude-opus-4-6",
			Label:       "Default",
			Description: "Opus 4.6 with 1M context. Most capable for complex work.",
			Recommended: true,
		},
		{
			ID:          "claude-sonnet-4-6",
			Label:       "Sonnet",
			Description: "Sonnet 4.6. Best for everyday tasks.",
		},
		{
			ID:          "claude-haiku-4-5",
			Label:       "Haiku",
			Description: "Haiku 4.5. Fastest for quick answers.",
		},
	},
	AgentProviderAdapterTypeGeminiCLI: {
		{
			ID:          "auto-gemini-2.5",
			Label:       "Auto (Gemini 2.5)",
			Description: "Let Gemini CLI decide between gemini-2.5-pro and gemini-2.5-flash.",
			Recommended: true,
		},
		{
			ID:          "gemini-2.5-pro",
			Label:       "gemini-2.5-pro",
			Description: "Stable Gemini 2.5 Pro model.",
		},
		{
			ID:          "gemini-2.5-flash",
			Label:       "gemini-2.5-flash",
			Description: "Stable Gemini 2.5 Flash model.",
		},
		{
			ID:          "gemini-2.5-flash-lite",
			Label:       "gemini-2.5-flash-lite",
			Description: "Stable Gemini 2.5 Flash Lite model.",
		},
		{
			ID:          "auto-gemini-3",
			Label:       "Auto (Gemini 3)",
			Description: "Preview routing for Gemini 3 models such as gemini-3.1-pro-preview and gemini-3-flash-preview.",
			Preview:     true,
		},
		{
			ID:          "gemini-3-flash-preview",
			Label:       "gemini-3-flash-preview",
			Description: "Preview Gemini 3 Flash model.",
			Preview:     true,
		},
		{
			ID:          "gemini-3.1-pro-preview",
			Label:       "gemini-3.1-pro-preview",
			Description: "Preview Gemini 3.1 Pro model.",
			Preview:     true,
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
	return cloned
}

func BuiltinAgentProviderAdaptersWithModelOptions() []AgentProviderAdapterType {
	cloned := make([]AgentProviderAdapterType, len(builtinAgentProviderModelOptionAdapters))
	copy(cloned, builtinAgentProviderModelOptionAdapters)
	return cloned
}
