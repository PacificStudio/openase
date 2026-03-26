package catalog

import entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"

type BuiltinAgentProviderTemplate struct {
	ID          string
	Name        string
	Command     string
	AdapterType entagentprovider.AdapterType
	ModelName   string
	CliArgs     []string
}

func BuiltinAgentProviderTemplates() []BuiltinAgentProviderTemplate {
	return []BuiltinAgentProviderTemplate{
		{
			ID:          "claude-code",
			Name:        "Claude Code",
			Command:     "claude",
			AdapterType: entagentprovider.AdapterTypeClaudeCodeCli,
			ModelName:   "claude-sonnet-4-5",
		},
		{
			ID:          "codex",
			Name:        "OpenAI Codex",
			Command:     "codex",
			AdapterType: entagentprovider.AdapterTypeCodexAppServer,
			ModelName:   "gpt-5.3-codex",
			CliArgs:     []string{"app-server", "--listen", "stdio://"},
		},
		{
			ID:          "gemini",
			Name:        "Gemini CLI",
			Command:     "gemini",
			AdapterType: entagentprovider.AdapterTypeGeminiCli,
			ModelName:   "gemini-2.5-pro",
		},
	}
}
