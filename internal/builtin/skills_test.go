package builtin

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"go.yaml.in/yaml/v3"
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
	if skills[0].Title == "" {
		t.Fatal("Skills() expected parsed titles")
	}
}

func TestLoadBuiltinSkillsFromBundles(t *testing.T) {
	entries, err := fs.ReadDir(builtinSkillFS, "skills")
	if err != nil {
		t.Fatalf("ReadDir(skills): %v", err)
	}
	directoryCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			directoryCount++
		}
	}
	if directoryCount != len(Skills()) {
		t.Fatalf("embedded skill directories=%d, Skills()=%d", directoryCount, len(Skills()))
	}
}

func TestLoadBuiltinSkillsParsesBundledMarkdown(t *testing.T) {
	templates, err := loadBuiltinSkills(fstest.MapFS{
		"skills/sample/SKILL.md": {
			Data: []byte("---\nname: \"sample\"\ndescription: \"Sample description\"\n---\n\n# Sample Title\n\nBody\n"),
		},
	})
	if err != nil {
		t.Fatalf("loadBuiltinSkills() error = %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("loadBuiltinSkills() len = %d, want 1", len(templates))
	}
	if templates[0].Name != "sample" {
		t.Fatalf("template name = %q", templates[0].Name)
	}
	if templates[0].Title != "Sample Title" {
		t.Fatalf("template title = %q", templates[0].Title)
	}
	if templates[0].Description != "Sample description" {
		t.Fatalf("template description = %q", templates[0].Description)
	}
}

func TestLoadBuiltinSkillsRejectsDirectoryFrontmatterMismatch(t *testing.T) {
	_, err := loadBuiltinSkills(fstest.MapFS{
		"skills/sample/SKILL.md": {
			Data: []byte("---\nname: \"other\"\ndescription: \"Sample description\"\n---\n\n# Sample Title\n"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "does not match directory") {
		t.Fatalf("loadBuiltinSkills() error = %v, want directory mismatch", err)
	}
}

func TestBuiltinSkillsIncludeCodexFrontmatter(t *testing.T) {
	for _, skill := range Skills() {
		if !strings.HasPrefix(skill.Content, "---\n") {
			t.Fatalf("skill %q missing YAML frontmatter prefix:\n%s", skill.Name, skill.Content)
		}

		lines := strings.Split(strings.ReplaceAll(skill.Content, "\r\n", "\n"), "\n")
		end := -1
		for index := 1; index < len(lines); index++ {
			if strings.TrimSpace(lines[index]) == "---" {
				end = index
				break
			}
		}
		if end == -1 {
			t.Fatalf("skill %q missing YAML frontmatter closing delimiter", skill.Name)
		}

		var document struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &document); err != nil {
			t.Fatalf("unmarshal frontmatter for %q: %v", skill.Name, err)
		}
		if document.Name != skill.Name {
			t.Fatalf("frontmatter name for %q = %q", skill.Name, document.Name)
		}
		if document.Description != skill.Description {
			t.Fatalf("frontmatter description for %q = %q, want %q", skill.Name, document.Description, skill.Description)
		}
	}
}

func TestOpenASEPlatformSkillDocumentsCoreCLIFlows(t *testing.T) {
	skill, ok := SkillByName("openase-platform")
	if !ok {
		t.Fatal("expected openase-platform skill to exist")
	}

	for _, snippet := range []string{
		"OPENASE_API_URL",
		"OPENASE_AGENT_TOKEN",
		"OPENASE_PROJECT_ID",
		"OPENASE_TICKET_ID",
		"./.openase/bin/openase ticket report-usage",
		"./.openase/bin/openase project add-repo",
		"./.openase/bin/openase workflow harness get $WORKFLOW_ID",
		"./.openase/bin/openase machine refresh-health $MACHINE_ID",
		"./.openase/bin/openase api GET /api/v1/tickets/$OPENASE_TICKET_ID",
		"`ticket-workpad` skill",
	} {
		if !strings.Contains(skill.Content, snippet) {
			t.Fatalf("expected openase-platform skill to contain %q, got:\n%s", snippet, skill.Content)
		}
	}
	foundScript := false
	for _, file := range skill.Files {
		if file.Path != "scripts/upsert_workpad.sh" {
			continue
		}
		foundScript = true
		if !file.IsExecutable {
			t.Fatalf("expected openase-platform script to be executable, got %+v", file)
		}
	}
	if !foundScript {
		t.Fatal("expected openase-platform to include scripts/upsert_workpad.sh")
	}
}

func TestTicketWorkpadSkillUsesGenericWorkpadTerminology(t *testing.T) {
	skill, ok := SkillByName("ticket-workpad")
	if !ok {
		t.Fatal("expected ticket-workpad skill to exist")
	}
	if strings.Contains(skill.Description, "Workpad") {
		t.Fatalf("ticket-workpad description should avoid Codex-specific naming: %q", skill.Description)
	}
	if strings.Contains(skill.Content, "## Workpad") {
		t.Fatalf("ticket-workpad content should avoid Codex-specific heading: %s", skill.Content)
	}
	for _, snippet := range []string{
		"Workpad 是当前工单唯一的持久化进度板",
		"openase-platform",
		"upsert_workpad.sh",
		"复用或更新那条持久化评论",
		"跨 runtime",
		"绑定到需要持续执行和续跑的 ticket workflow",
	} {
		if !strings.Contains(skill.Content, snippet) {
			t.Fatalf("expected ticket-workpad skill to contain %q, got:\n%s", snippet, skill.Content)
		}
	}
}
