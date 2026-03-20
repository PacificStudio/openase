package builtin

import (
	"strings"
	"testing"
)

func TestEnvironmentProvisionerSkillsExist(t *testing.T) {
	testCases := []struct {
		name    string
		title   string
		snippet string
	}{
		{name: "install-claude-code", title: "Install Claude Code", snippet: "`claude --version`"},
		{name: "install-codex", title: "Install Codex CLI", snippet: "`codex --version`"},
		{name: "setup-git", title: "Setup Git", snippet: "`git --version`"},
		{name: "setup-gh-cli", title: "Setup GitHub CLI", snippet: "`gh auth status`"},
	}

	for _, tt := range testCases {
		skill, ok := SkillByName(tt.name)
		if !ok {
			t.Fatalf("expected skill %q to exist", tt.name)
		}
		if skill.Title != tt.title {
			t.Fatalf("skill %q title=%q, want %q", tt.name, skill.Title, tt.title)
		}
		if !strings.Contains(skill.Content, tt.snippet) {
			t.Fatalf("expected skill %q to contain %q, got:\n%s", tt.name, tt.snippet, skill.Content)
		}
	}
}
