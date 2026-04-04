package catalog

import (
	"strings"
	"testing"
)

func TestEffectiveAgentRunSummaryPromptUsesBuiltInPromptByDefault(t *testing.T) {
	prompt, source := EffectiveAgentRunSummaryPrompt("  ")
	if source != AgentRunSummaryPromptSourceBuiltin {
		t.Fatalf("EffectiveAgentRunSummaryPrompt() source = %q, want builtin", source)
	}
	if prompt != strings.TrimSpace(DefaultAgentRunSummaryPrompt) {
		t.Fatalf("EffectiveAgentRunSummaryPrompt() prompt = %q, want built-in prompt", prompt)
	}
}

func TestEffectiveAgentRunSummaryPromptPrefersProjectOverride(t *testing.T) {
	prompt, source := EffectiveAgentRunSummaryPrompt("  Summarize retries first.  ")
	if source != AgentRunSummaryPromptSourceProjectOverride {
		t.Fatalf("EffectiveAgentRunSummaryPrompt() source = %q, want project_override", source)
	}
	if prompt != "Summarize retries first." {
		t.Fatalf("EffectiveAgentRunSummaryPrompt() prompt = %q, want trimmed override", prompt)
	}
}

func TestAgentRunSummaryPromptSourceString(t *testing.T) {
	if got := AgentRunSummaryPromptSourceBuiltin.String(); got != "builtin" {
		t.Fatalf("AgentRunSummaryPromptSourceBuiltin.String() = %q", got)
	}
	if got := AgentRunSummaryPromptSourceProjectOverride.String(); got != "project_override" {
		t.Fatalf("AgentRunSummaryPromptSourceProjectOverride.String() = %q", got)
	}
}
