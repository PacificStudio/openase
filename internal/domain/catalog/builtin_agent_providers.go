package catalog

type BuiltinAgentProviderTemplate struct {
	ID          string
	Name        string
	Command     string
	AdapterType AgentProviderAdapterType
	ModelName   string
	CliArgs     []string
}

func BuiltinAgentProviderTemplates() []BuiltinAgentProviderTemplate {
	return []BuiltinAgentProviderTemplate{
		{
			ID:          "claude-code",
			Name:        "Claude Code",
			Command:     "claude",
			AdapterType: AgentProviderAdapterTypeClaudeCodeCLI,
			ModelName:   "claude-sonnet-4-5",
		},
		{
			ID:          "codex",
			Name:        "OpenAI Codex",
			Command:     "codex",
			AdapterType: AgentProviderAdapterTypeCodexAppServer,
			ModelName:   "gpt-5.3-codex",
			CliArgs:     []string{"app-server", "--listen", "stdio://"},
		},
		{
			ID:          "gemini",
			Name:        "Gemini CLI",
			Command:     "gemini",
			AdapterType: AgentProviderAdapterTypeGeminiCLI,
			ModelName:   "gemini-2.5-pro",
		},
	}
}
