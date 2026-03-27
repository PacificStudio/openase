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

func TestSkillHelpers(t *testing.T) {
	skills := Skills()
	if len(skills) == 0 {
		t.Fatal("Skills() expected built-in skills")
	}

	originalTitle := skills[0].Title
	skills[0].Title = "mutated"
	refreshed := Skills()
	if len(refreshed) == 0 || refreshed[0].Title != originalTitle {
		t.Fatalf("Skills() should clone templates, got %+v", refreshed)
	}

	if !IsBuiltinSkill("commit") {
		t.Fatal("IsBuiltinSkill(commit) expected true")
	}
	if IsBuiltinSkill("missing-skill") {
		t.Fatal("IsBuiltinSkill(missing-skill) expected false")
	}
	if _, ok := SkillByName("missing-skill"); ok {
		t.Fatal("SkillByName(missing-skill) expected false")
	}
}
