package workflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	"github.com/google/uuid"
	"github.com/nikolalohinski/gonja/v2/exec"
)

func TestValidateHarnessContentAndSave(t *testing.T) {
	valid := "Implement {{ ticket.title|markdown_escape }}\n"
	result := ValidateHarnessContent(valid)
	if !result.Valid || len(result.Issues) != 0 {
		t.Fatalf("ValidateHarnessContent(valid) = %+v", result)
	}
	if err := validateHarnessForSave(valid); err != nil {
		t.Fatalf("validateHarnessForSave(valid) error = %v", err)
	}

	empty := ValidateHarnessContent(" \r\n ")
	if empty.Valid || len(empty.Issues) != 1 || empty.Issues[0].Line != 1 || empty.Issues[0].Column != 1 {
		t.Fatalf("ValidateHarnessContent(empty) = %+v", empty)
	}
	if err := validateHarnessForSave(" "); err == nil || !strings.Contains(err.Error(), "Harness content must not be empty") {
		t.Fatalf("validateHarnessForSave(empty) error = %v", err)
	}

	legacyFrontmatter := ValidateHarnessContent("---\nworkflow:\n  role: x\n---\nBody")
	if legacyFrontmatter.Valid || !strings.Contains(legacyFrontmatter.Issues[0].Message, "no longer supported") {
		t.Fatalf("ValidateHarnessContent(legacy frontmatter) = %+v", legacyFrontmatter)
	}

	invalidTemplate := ValidateHarnessContent("{{")
	if invalidTemplate.Valid || len(invalidTemplate.Issues) == 0 || invalidTemplate.Issues[0].Level != "error" {
		t.Fatalf("ValidateHarnessContent(invalid template) = %+v", invalidTemplate)
	}
	if err := validateHarnessForSave("{{"); err == nil || !strings.Contains(err.Error(), "line") {
		t.Fatalf("validateHarnessForSave(invalid template) error = %v", err)
	}

	if got := normalizeHarnessNewlines("a\r\nb\rc"); got != "a\nb\nc" {
		t.Fatalf("normalizeHarnessNewlines() = %q", got)
	}
	if line, col, ok := extractGonjaPosition("boom Line: 4 Col: 7"); !ok || line != 4 || col != 7 {
		t.Fatalf("extractGonjaPosition() = %d, %d, %t", line, col, ok)
	}
	if _, _, ok := extractGonjaPosition("bad"); ok {
		t.Fatal("extractGonjaPosition() expected false for bad input")
	}
	if got := normalizeGonjaError("failed to parse template 'inline': broken"); got != "broken" {
		t.Fatalf("normalizeGonjaError() = %q", got)
	}
	if hasErrorIssue([]ValidationIssue{{Level: "warning"}}) {
		t.Fatal("hasErrorIssue() expected false for warnings only")
	}
	if !hasErrorIssue([]ValidationIssue{{Level: "error"}}) {
		t.Fatal("hasErrorIssue() expected true for errors")
	}
}

func TestStatusBindingSetHelpers(t *testing.T) {
	idOne := uuid.New()
	idTwo := uuid.New()

	set, err := ParseStatusBindingSet("pickup_status_ids", []uuid.UUID{idOne, idOne, idTwo})
	if err != nil {
		t.Fatalf("ParseStatusBindingSet() error = %v", err)
	}
	if set.Len() != 2 || !set.Contains(idOne) || !set.Contains(idTwo) {
		t.Fatalf("ParseStatusBindingSet() = %+v", set)
	}
	if single, ok := set.Single(); ok || single != uuid.Nil {
		t.Fatalf("Single() = %s, %t; want nil, false", single, ok)
	}
	ids := set.IDs()
	ids[0] = uuid.Nil
	if !set.Contains(idOne) {
		t.Fatal("IDs() returned backing slice")
	}

	singleSet := MustStatusBindingSet(idOne)
	if single, ok := singleSet.Single(); !ok || single != idOne {
		t.Fatalf("MustStatusBindingSet().Single() = %s, %t", single, ok)
	}
	if _, err := ParseStatusBindingSet("pickup_status_ids", nil); err == nil {
		t.Fatal("ParseStatusBindingSet(nil) expected error")
	}
	if _, err := ParseStatusBindingSet("pickup_status_ids", []uuid.UUID{uuid.Nil}); err == nil {
		t.Fatal("ParseStatusBindingSet(zero uuid) expected error")
	}
}

func TestHarnessTemplateUtilityHelpers(t *testing.T) {
	if got := cloneResourceMap(nil); len(got) != 0 {
		t.Fatalf("cloneResourceMap(nil) = %+v", got)
	}

	source := map[string]any{"transport": "local"}
	cloned := cloneResourceMap(source)
	source["transport"] = "remote"
	if cloned["transport"] != "local" {
		t.Fatalf("cloneResourceMap() = %+v", cloned)
	}

	if got := formatOptionalTime(nil); got != "" {
		t.Fatalf("formatOptionalTime(nil) = %q", got)
	}
	zero := time.Time{}
	if got := formatOptionalTime(&zero); got != "" {
		t.Fatalf("formatOptionalTime(zero) = %q", got)
	}
	createdAt := time.Date(2026, time.March, 27, 12, 0, 0, 0, time.UTC)
	if got := formatOptionalTime(&createdAt); got != "2026-03-27T12:00:00Z" {
		t.Fatalf("formatOptionalTime(value) = %q", got)
	}

	escaped := filterMarkdownEscape(nil, exec.AsValue("`*_{}[]()#+-.!>|\\"),
		exec.NewVarArgs())
	if escaped.IsError() || escaped.String() != "\\`\\*\\_\\{\\}\\[\\]\\(\\)\\#\\+\\-\\.\\!\\>\\|\\\\" {
		t.Fatalf("filterMarkdownEscape() = %q error=%t", escaped.String(), escaped.IsError())
	}

	errorValue := exec.AsValue(fmt.Errorf("boom"))
	if got := filterMarkdownEscape(nil, errorValue, exec.NewVarArgs()); got != errorValue {
		t.Fatalf("filterMarkdownEscape(error input) should return original value")
	}

	withUnexpectedArgs := exec.NewVarArgs()
	withUnexpectedArgs.Args = []*exec.Value{exec.AsValue("unexpected")}
	if got := filterMarkdownEscape(nil, exec.AsValue("body"), withUnexpectedArgs); !got.IsError() || !strings.Contains(got.Error(), "wrong signature") {
		t.Fatalf("filterMarkdownEscape(unexpected args) error = %q", got.Error())
	}
}

func TestSkillHelpersAndFilesystem(t *testing.T) {
	if got := parseSkillTitle("intro\n# Title\nbody"); got != "Title" {
		t.Fatalf("parseSkillTitle() = %q", got)
	}
	if got := parseSkillTitle("no title"); got != "" {
		t.Fatalf("parseSkillTitle(no heading) = %q", got)
	}

	if got, err := normalizeSkillNames([]string{" skill-one ", "skill-two", "skill-one"}); err != nil || len(got) != 2 || got[0] != "skill-one" || got[1] != "skill-two" {
		t.Fatalf("normalizeSkillNames() = %+v, %v", got, err)
	}
	if _, err := normalizeSkillNames([]string{" "}); err == nil {
		t.Fatal("normalizeSkillNames(blank) expected error")
	}
	if _, err := normalizeSkillNames([]string{"Bad Name"}); err == nil {
		t.Fatal("normalizeSkillNames(pattern) expected error")
	}

	content := "Body\n"
	skills, err := ParseHarnessSkills(content)
	if err != nil || len(skills) != 0 {
		t.Fatalf("ParseHarnessSkills() = %+v, %v", skills, err)
	}
	updated, err := setHarnessSkills(content, []string{"new-skill", "old-skill", "new-skill"})
	if err != nil {
		t.Fatalf("setHarnessSkills() error = %v", err)
	}
	parsedUpdated, err := ParseHarnessSkills(updated)
	if err != nil || len(parsedUpdated) != 0 {
		t.Fatalf("ParseHarnessSkills(updated) = %+v, %v", parsedUpdated, err)
	}
	removed, err := setHarnessSkills(content, nil)
	if err != nil {
		t.Fatalf("setHarnessSkills(remove) error = %v", err)
	}
	if removed != content {
		t.Fatalf("setHarnessSkills(remove) = %q", removed)
	}
	if parsed, err := ParseHarnessSkills("bad"); err != nil || len(parsed) != 0 {
		t.Fatalf("ParseHarnessSkills(plain body) = %+v, %v", parsed, err)
	}

	workspaceRoot := t.TempDir()
	codexTarget, err := resolveSkillTarget(workspaceRoot, " codex-app-server ")
	if err != nil || !strings.HasSuffix(codexTarget.skillsDir.String(), filepath.Join(".codex", "skills")) {
		t.Fatalf("resolveSkillTarget(codex) = %+v, %v", codexTarget, err)
	}
	claudeTarget, err := resolveSkillTarget(workspaceRoot, "claude-code-cli")
	if err != nil || !strings.HasSuffix(claudeTarget.skillsDir.String(), filepath.Join(".claude", "skills")) {
		t.Fatalf("resolveSkillTarget(claude) = %+v, %v", claudeTarget, err)
	}
	geminiTarget, err := resolveSkillTarget(workspaceRoot, "gemini-cli")
	if err != nil || !strings.HasSuffix(geminiTarget.skillsDir.String(), filepath.Join(".gemini", "skills")) {
		t.Fatalf("resolveSkillTarget(gemini) = %+v, %v", geminiTarget, err)
	}
	customTarget, err := resolveSkillTarget(workspaceRoot, "custom")
	if err != nil || !strings.HasSuffix(customTarget.skillsDir.String(), filepath.Join(".agent", "skills")) {
		t.Fatalf("resolveSkillTarget(custom) = %+v, %v", customTarget, err)
	}
	if _, err := resolveSkillTarget(" ", "custom"); err == nil {
		t.Fatal("resolveSkillTarget(blank workspace) expected error")
	}
	if _, err := resolveSkillTarget(workspaceRoot, "bad-adapter"); err == nil {
		t.Fatal("resolveSkillTarget(bad adapter) expected error")
	}

	skillsRoot := t.TempDir()
	mustWriteSkill(t, filepath.Join(skillsRoot, "skill-one"), "# Skill One\nbody")
	mustWriteSkill(t, filepath.Join(skillsRoot, "skill-two"), "# Skill Two\nbody")
	if err := os.Mkdir(filepath.Join(skillsRoot, "Bad Name"), 0o750); err != nil {
		t.Fatalf("mkdir invalid skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsRoot, "plain.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write plain file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(skillsRoot, "missing-skill"), 0o750); err != nil {
		t.Fatalf("mkdir missing skill: %v", err)
	}
	names, err := listSkillNames(skillsRoot)
	if err != nil || len(names) != 2 || names[0] != "skill-one" || names[1] != "skill-two" {
		t.Fatalf("listSkillNames() = %+v, %v", names, err)
	}
	if err := validateSkillDirectory(filepath.Join(skillsRoot, "skill-one")); err != nil {
		t.Fatalf("validateSkillDirectory(valid) error = %v", err)
	}
	if err := validateSkillDirectory(filepath.Join(skillsRoot, "missing-skill")); err == nil {
		t.Fatal("validateSkillDirectory(missing SKILL.md) expected error")
	}
	invalidSkillDir := filepath.Join(skillsRoot, "invalid-skill")
	if err := os.MkdirAll(invalidSkillDir, 0o750); err != nil {
		t.Fatalf("mkdir invalid skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(invalidSkillDir, "SKILL.md"), []byte("# Invalid Skill\nbody"), 0o600); err != nil {
		t.Fatalf("write invalid skill file: %v", err)
	}
	if err := validateSkillDirectory(invalidSkillDir); err == nil || !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("validateSkillDirectory(missing frontmatter) error = %v", err)
	}

	if err := validateSkillDirectory(filepath.Join(skillsRoot, "skill-one")); err != nil {
		t.Fatalf("validateSkillDirectory(valid) error = %v", err)
	}
	if err := validateSkillDirectory(filepath.Join(skillsRoot, "missing-skill")); err == nil {
		t.Fatal("validateSkillDirectory(missing) expected error")
	}
	if got := skillContentRelativePath("skill-one"); got != ".openase/skills/skill-one/SKILL.md" {
		t.Fatalf("skillContentRelativePath() = %q", got)
	}

	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o750); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# Src\n"), 0o600); err != nil {
		t.Fatalf("write src skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "notes.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "dst")
	if err := copyDirectory(src, dst); err != nil {
		t.Fatalf("copyDirectory() error = %v", err)
	}
	if got, err := readWorkflowTestFile(filepath.Join(dst, "nested", "notes.txt")); err != nil || string(got) != "hello" {
		t.Fatalf("copied nested file = %q, %v", got, err)
	}
	firstFingerprint, err := directoryFingerprint(src)
	if err != nil {
		t.Fatalf("directoryFingerprint(src) error = %v", err)
	}
	secondFingerprint, err := directoryFingerprint(dst)
	if err != nil {
		t.Fatalf("directoryFingerprint(dst) error = %v", err)
	}
	if firstFingerprint != secondFingerprint {
		t.Fatalf("directoryFingerprint mismatch: %q != %q", firstFingerprint, secondFingerprint)
	}
	if err := os.WriteFile(filepath.Join(dst, "nested", "notes.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatalf("mutate dst file: %v", err)
	}
	changedFingerprint, err := directoryFingerprint(dst)
	if err != nil {
		t.Fatalf("directoryFingerprint(changed dst) error = %v", err)
	}
	if changedFingerprint == firstFingerprint {
		t.Fatal("directoryFingerprint() expected change after content mutation")
	}

	replaceDst := filepath.Join(t.TempDir(), "replace-dst")
	if err := os.MkdirAll(replaceDst, 0o750); err != nil {
		t.Fatalf("mkdir replace dst: %v", err)
	}
	if err := os.WriteFile(filepath.Join(replaceDst, "old.txt"), []byte("old"), 0o600); err != nil {
		t.Fatalf("write replace old file: %v", err)
	}
	if err := replaceDirectory(src, replaceDst); err != nil {
		t.Fatalf("replaceDirectory() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(replaceDst, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("replaceDirectory() left stale file, stat err = %v", err)
	}

	if err := ensureCopyTargetWithinRoot(replaceDst, filepath.Join(replaceDst, "nested", "notes.txt")); err != nil {
		t.Fatalf("ensureCopyTargetWithinRoot(valid) error = %v", err)
	}
	if err := ensureCopyTargetWithinRoot(replaceDst, filepath.Clean(filepath.Join(replaceDst, "..", "escape.txt"))); err == nil {
		t.Fatal("ensureCopyTargetWithinRoot(escape) expected error")
	}

	symlinkSrc := t.TempDir()
	mustWriteSkill(t, filepath.Join(symlinkSrc, "skill"), "# Skill\nbody")
	if err := os.Symlink(filepath.Join(symlinkSrc, "skill", "SKILL.md"), filepath.Join(symlinkSrc, "skill", "link.md")); err == nil {
		if err := copyDirectory(filepath.Join(symlinkSrc, "skill"), filepath.Join(t.TempDir(), "symlink-dst")); err == nil {
			t.Fatal("copyDirectory(symlink) expected error")
		}
	}
}

func TestWorkspaceWrapperAndWorkflowHookParserCoverage(t *testing.T) {
	workspaceRoot := t.TempDir()
	if err := writeWorkspaceOpenASEWrapper(workspaceRoot); err != nil {
		t.Fatalf("writeWorkspaceOpenASEWrapper() error = %v", err)
	}

	wrapperPath := filepath.Join(workspaceRoot, ".openase", "bin", "openase")
	wrapperContent, err := readWorkflowTestFile(wrapperPath)
	if err != nil {
		t.Fatalf("ReadFile(wrapper) error = %v", err)
	}
	if !strings.Contains(string(wrapperContent), "OPENASE_REAL_BIN") {
		t.Fatalf("wrapper content = %q", wrapperContent)
	}

	wrapperInfo, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatalf("Stat(wrapper) error = %v", err)
	}
	if wrapperInfo.Mode().Perm() != 0o700 {
		t.Fatalf("wrapper mode = %#o, want 0700", wrapperInfo.Mode().Perm())
	}

	brokenWorkspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(brokenWorkspaceRoot, ".openase"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile(broken workspace marker) error = %v", err)
	}
	if err := writeWorkspaceOpenASEWrapper(brokenWorkspaceRoot); err == nil || !strings.Contains(err.Error(), "create workspace openase wrapper directory") {
		t.Fatalf("writeWorkspaceOpenASEWrapper(broken workspace) error = %v", err)
	}

	if _, err := parseWorkflowHookList(map[string]any{
		string(workflowHookOnReload): "bad",
	}, workflowHookOnReload); err == nil || !strings.Contains(err.Error(), "must be a list") {
		t.Fatalf("parseWorkflowHookList(invalid list) error = %v", err)
	}
	if _, err := parseWorkflowHookEntries([]any{"bad"}, "hooks.workflow_hooks.on_reload"); err == nil || !strings.Contains(err.Error(), "must be an object") {
		t.Fatalf("parseWorkflowHookEntries(invalid entry) error = %v", err)
	}
	for _, testCase := range []struct {
		name string
		raw  any
		want string
	}{
		{
			name: "missing cmd",
			raw:  map[string]any{},
			want: ".cmd is required",
		},
		{
			name: "blank cmd",
			raw:  map[string]any{"cmd": "   "},
			want: ".cmd must be a non-empty string",
		},
		{
			name: "bad timeout",
			raw:  map[string]any{"cmd": "echo ok", "timeout": "bad"},
			want: ".timeout must be a whole number of seconds",
		},
		{
			name: "bad on_failure",
			raw:  map[string]any{"cmd": "echo ok", "on_failure": "explode"},
			want: ".on_failure must be one of block, warn, ignore",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := parseWorkflowHookDefinition(testCase.raw, "hooks.workflow_hooks.on_reload[0]"); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("parseWorkflowHookDefinition() error = %v, want substring %q", err, testCase.want)
			}
		})
	}
}

func readWorkflowTestFile(path string) ([]byte, error) {
	//nolint:gosec // Test paths are created under t.TempDir and validated by the test flow.
	return os.ReadFile(path)
}

func TestHarnessTemplateHelpers(t *testing.T) {
	repoID := uuid.New()
	projectID := uuid.New()
	ticketID := uuid.New()
	agentID := uuid.New()
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	repo := &ent.ProjectRepo{
		ID:            repoID,
		Name:          "backend",
		RepositoryURL: "https://example.com/backend.git",
		DefaultBranch: "main",
		Labels:        pgarray.StringArray{"go", "api"},
	}
	scope := &ent.TicketRepoScope{
		BranchName: "feature/ase-278",
		Edges:      ent.TicketRepoScopeEdges{Repo: repo},
	}
	scopedRepos, branches := mapHarnessScopedRepos("ASE-278", []*ent.TicketRepoScope{
		scope,
		{Edges: ent.TicketRepoScopeEdges{}},
	}, "/workspaces/ASE-278")
	if len(scopedRepos) != 1 || branches[repoID] != "feature/ase-278" || scopedRepos[0].Path != "/workspaces/ASE-278/backend" {
		t.Fatalf("mapHarnessScopedRepos() = %+v, %+v", scopedRepos, branches)
	}
	allRepos := mapHarnessAllRepos("ASE-278", []*ent.ProjectRepo{
		repo,
		{
			ID:               uuid.New(),
			Name:             "frontend",
			RepositoryURL:    "https://example.com/frontend.git",
			DefaultBranch:    "develop",
			WorkspaceDirname: "/repos/frontend",
		},
	}, branches, "/workspaces/ASE-278")
	if len(allRepos) != 2 || allRepos[0].Branch != "feature/ase-278" || allRepos[1].Branch != "agent/ASE-278" || allRepos[1].Path != "/repos/frontend" {
		t.Fatalf("mapHarnessAllRepos() = %+v", allRepos)
	}

	if got := joinStatusNames([]*ent.TicketStatus{{Name: "Todo"}, {Name: "Done"}}); got != "Todo, Done" {
		t.Fatalf("joinStatusNames() = %q", got)
	}
	if got := resolveRepoPath("", "/tmp/ws", "backend"); got != "/tmp/ws/backend" {
		t.Fatalf("resolveRepoPath() = %q", got)
	}
	if got := resolveRepoPath("/repos/backend", "/tmp/ws", "backend"); got != "/repos/backend" {
		t.Fatalf("resolveRepoPath(clonePath) = %q", got)
	}
	if got := resolveRepoPath("", "", "backend"); got != "" {
		t.Fatalf("resolveRepoPath(no workspace) = %q", got)
	}

	machines := mapHarnessProjectMachines(
		HarnessMachineData{Name: "local", Host: "127.0.0.1", Description: "Current", Labels: []string{"local"}},
		[]HarnessAccessibleMachineData{
			{Name: "storage", Host: "10.0.0.2", Description: "Storage", Labels: []string{"nfs"}, SSHUser: "openase"},
			{Name: "local", Host: "127.0.0.1", Description: "Duplicate", Labels: []string{"dup"}},
		},
	)
	if len(machines) != 2 || machines[0].Name != "local" || machines[1].Status != "accessible" {
		t.Fatalf("mapHarnessProjectMachines() = %+v", machines)
	}

	agent := mapHarnessAgent(&ent.Agent{
		ID:                    agentID,
		Name:                  "codex-01",
		TotalTicketsCompleted: 7,
		Edges: ent.AgentEdges{
			Provider: &ent.AgentProvider{
				Name:        "Codex",
				AdapterType: entagentprovider.AdapterTypeCodexAppServer,
				ModelName:   "gpt-5.4",
			},
		},
	})
	if agent.Provider != "Codex" || agent.AdapterType != "codex-app-server" || agent.Model != "gpt-5.4" {
		t.Fatalf("mapHarnessAgent() = %+v", agent)
	}
	if got := mapHarnessAgent(nil); got != (HarnessAgentData{}) {
		t.Fatalf("mapHarnessAgent(nil) = %+v", got)
	}

	links := mapHarnessTicketLinks([]*ent.TicketExternalLink{{
		LinkType: entticketexternallink.LinkTypeGithubIssue,
		URL:      "https://example.com/issues/1",
		Title:    "Issue",
		Status:   "open",
		Relation: entticketexternallink.RelationResolves,
	}})
	if len(links) != 1 || links[0].Type != "github_issue" || links[0].Relation != "resolves" {
		t.Fatalf("mapHarnessTicketLinks() = %+v", links)
	}

	dependencies := mapHarnessDependencies([]*ent.TicketDependency{
		{
			Type: entticketdependency.TypeSubIssue,
			Edges: ent.TicketDependencyEdges{
				TargetTicket: &ent.Ticket{
					Identifier: "ASE-12",
					Title:      "Parent",
					Edges: ent.TicketEdges{
						Status: &ent.TicketStatus{Name: "Done"},
					},
				},
			},
		},
		{},
	})
	if len(dependencies) != 1 || dependencies[0].Type != "sub_issue" || dependencies[0].Status != "Done" {
		t.Fatalf("mapHarnessDependencies() = %+v", dependencies)
	}
	if got := edgeTicketStatusName(nil); got != "" {
		t.Fatalf("edgeTicketStatusName(nil) = %q", got)
	}
	if got := edgeTicketStatusName(&ent.TicketStatus{Name: "Todo"}); got != "Todo" {
		t.Fatalf("edgeTicketStatusName() = %q", got)
	}
	if got := parentIdentifier(&ent.Ticket{Edges: ent.TicketEdges{Parent: &ent.Ticket{Identifier: "ASE-1"}}}); got != "ASE-1" {
		t.Fatalf("parentIdentifier() = %q", got)
	}
	if got := parentIdentifier(nil); got != "" {
		t.Fatalf("parentIdentifier(nil) = %q", got)
	}
	if got := normalizeDependencyType(entticketdependency.TypeSubIssue); got != "sub_issue" {
		t.Fatalf("normalizeDependencyType() = %q", got)
	}
	if got := normalizeAttemptCount(0); got != 1 {
		t.Fatalf("normalizeAttemptCount(0) = %d", got)
	}
	if got := normalizeAttemptCount(3); got != 3 {
		t.Fatalf("normalizeAttemptCount(3) = %d", got)
	}

	platform := normalizePlatformData(HarnessPlatformData{APIURL: "http://localhost:19836/api/v1"}, projectID, ticketID)
	if platform.ProjectID != projectID.String() || platform.TicketID != ticketID.String() {
		t.Fatalf("normalizePlatformData() = %+v", platform)
	}
	harnessContent := "# Backend Engineer\n\nBuild APIs safely.\n"

	machine := cloneHarnessMachine(HarnessMachineData{Name: "local", Labels: []string{"a"}})
	machine.Labels[0] = "changed"
	accessible := cloneAccessibleMachines([]HarnessAccessibleMachineData{{Name: "storage", Labels: []string{"nfs"}}})
	accessible[0].Labels[0] = "changed"

	startedAt := now.Add(-time.Hour)
	completedAt := now
	workflowTicket := mapHarnessProjectWorkflowTicket(&ent.Ticket{
		Identifier:        "ASE-278",
		Title:             "Coverage rollout",
		Priority:          "high",
		Type:              "feature",
		AttemptCount:      0,
		ConsecutiveErrors: 2,
		RetryPaused:       true,
		PauseReason:       "budget_exhausted",
		CreatedAt:         now.Add(-2 * time.Hour),
		StartedAt:         &startedAt,
		CompletedAt:       &completedAt,
		Edges:             ent.TicketEdges{Status: &ent.TicketStatus{Name: "Done"}},
	})
	if workflowTicket.AttemptCount != 1 || workflowTicket.Status != "Done" || workflowTicket.StartedAt == "" || workflowTicket.CompletedAt == "" {
		t.Fatalf("mapHarnessProjectWorkflowTicket() = %+v", workflowTicket)
	}
	if got := formatOptionalTime(nil); got != "" {
		t.Fatalf("formatOptionalTime(nil) = %q", got)
	}
	if got := formatOptionalTime(&time.Time{}); got != "" {
		t.Fatalf("formatOptionalTime(zero) = %q", got)
	}
	if got := formatOptionalTime(&now); got != now.Format(time.RFC3339) {
		t.Fatalf("formatOptionalTime(now) = %q", got)
	}
	if got := cloneResourceMap(nil); len(got) != 0 {
		t.Fatalf("cloneResourceMap(nil) = %+v", got)
	}

	data := HarnessTemplateData{
		Ticket: HarnessTicketData{
			ID:               ticketID.String(),
			Identifier:       "ASE-278",
			Title:            "Escape *this*",
			Description:      "Desc",
			Status:           "Todo",
			Priority:         "high",
			Type:             "feature",
			CreatedBy:        "user:test",
			CreatedAt:        now.Format(time.RFC3339),
			AttemptCount:     1,
			MaxAttempts:      3,
			BudgetUSD:        12.5,
			ExternalRef:      "gh-278",
			ParentIdentifier: "ASE-1",
			URL:              "https://example.com/tickets/ASE-278",
			Links:            links,
			Dependencies:     dependencies,
		},
		Project: HarnessProjectData{
			ID:          projectID.String(),
			Name:        "OpenASE",
			Slug:        "openase",
			Description: "Automation",
			Status:      "In Progress",
			Workflows: []HarnessProjectWorkflowData{{
				Name:            "Coding",
				Type:            "coding",
				RoleName:        "Backend Engineer",
				RoleDescription: "Build APIs safely.",
				PickupStatus:    "Todo",
				FinishStatus:    "Done",
				HarnessPath:     ".openase/harnesses/coding.md",
				HarnessContent:  harnessContent,
				Skills:          []string{"skill-one"},
				MaxConcurrent:   2,
				CurrentActive:   1,
				RecentTickets:   []HarnessProjectWorkflowTicketData{workflowTicket},
			}},
			Statuses: []HarnessProjectStatusData{{Name: "Todo", Color: "#000"}},
			Machines: machines,
		},
		Repos:              scopedRepos,
		AllRepos:           allRepos,
		Agent:              agent,
		Machine:            HarnessMachineData{Name: "local", Host: "127.0.0.1", Description: "Current", Labels: []string{"local"}, WorkspaceRoot: "/srv/openase/workspaces"},
		AccessibleMachines: []HarnessAccessibleMachineData{{Name: "storage", Host: "10.0.0.2", Description: "Storage", Labels: []string{"nfs"}, SSHUser: "openase"}},
		Attempt:            1,
		MaxAttempts:        3,
		Workspace:          "/workspaces/ASE-278",
		Timestamp:          now.Format(time.RFC3339),
		OpenASEVersion:     "0.1.1",
		Workflow:           HarnessWorkflowData{Name: "Coding", Type: "coding", RoleName: "Backend Engineer", PickupStatus: "Todo", FinishStatus: "Done"},
		Platform:           platform,
	}
	contextMap := data.contextMap()
	projectMap := contextMap["project"].(map[string]any)
	reposMap := contextMap["repos"].([]map[string]any)
	if _, exists := projectMap["default_branch"]; exists || reposMap[0]["name"] != "backend" {
		t.Fatalf("contextMap() = %+v", contextMap)
	}
	reposMap[0]["labels"].([]string)[0] = "changed"
	if data.Repos[0].Labels[0] != "go" {
		t.Fatal("contextMap() returned mutable repo labels")
	}

	rendered, err := RenderHarnessBody(`Title={{ ticket.title|markdown_escape }}
Role={{ workflow.role_name }}
MachineCount={{ project.machines|length }}
`, data)
	if err != nil {
		t.Fatalf("RenderHarnessBody() error = %v", err)
	}
	if !strings.Contains(rendered, `Title=Escape \*this\*`) || !strings.Contains(rendered, "Role=Backend Engineer") || !strings.Contains(rendered, "MachineCount=2") {
		t.Fatalf("RenderHarnessBody() = %q", rendered)
	}
	if got, err := RenderHarnessBody("", data); err != nil || got != "" {
		t.Fatalf("RenderHarnessBody(empty body) = %q, %v", got, err)
	}
	if _, err := RenderHarnessBody("{{", data); err == nil {
		t.Fatal("RenderHarnessBody(invalid template) expected error")
	}

	dictionary := HarnessVariableDictionary()
	if len(dictionary) == 0 || dictionary[0].Name == "" {
		t.Fatalf("HarnessVariableDictionary() = %+v", dictionary)
	}
}

func TestWorkflowHookHelpersAndExecution(t *testing.T) {
	if timeout, err := parseWorkflowHookTimeout(nil, "hooks.timeout"); err != nil || timeout != 0 {
		t.Fatalf("parseWorkflowHookTimeout(nil) = %v, %v", timeout, err)
	}
	if timeout, err := parseWorkflowHookTimeout(3, "hooks.timeout"); err != nil || timeout != 3*time.Second {
		t.Fatalf("parseWorkflowHookTimeout(3) = %v, %v", timeout, err)
	}
	if _, err := parseWorkflowHookTimeout(1.5, "hooks.timeout"); err == nil {
		t.Fatal("parseWorkflowHookTimeout(fractional) expected error")
	}
	if _, err := parseWorkflowHookTimeout(-1, "hooks.timeout"); err == nil {
		t.Fatal("parseWorkflowHookTimeout(negative) expected error")
	}

	if policy, err := parseWorkflowHookFailure(nil, "hooks.on_failure"); err != nil || policy != workflowHookFailureBlock {
		t.Fatalf("parseWorkflowHookFailure(nil) = %q, %v", policy, err)
	}
	if policy, err := parseWorkflowHookFailure(" warn ", "hooks.on_failure"); err != nil || policy != workflowHookFailureWarn {
		t.Fatalf("parseWorkflowHookFailure(warn) = %q, %v", policy, err)
	}
	if _, err := parseWorkflowHookFailure(12, "hooks.on_failure"); err == nil {
		t.Fatal("parseWorkflowHookFailure(non-string) expected error")
	}

	runtime := workflowHookRuntime{
		ProjectID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		WorkflowID:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		WorkflowName:    "Coverage Workflow",
		WorkflowVersion: 7,
	}
	command := renderWorkflowHookCommand(
		`echo {{ project.id }} {{ workflow.id }} {{ workflow.name }} {{ workflow.version }} {{ hook.name }} {{ unknown }}`,
		workflowHookOnReload,
		runtime,
	)
	if !strings.Contains(command, shellQuote(runtime.ProjectID.String())) ||
		!strings.Contains(command, shellQuote("Coverage Workflow")) ||
		!strings.Contains(command, shellQuote("7")) ||
		!strings.Contains(command, shellQuote("on_reload")) ||
		!strings.Contains(command, "{{ unknown }}") {
		t.Fatalf("renderWorkflowHookCommand() = %q", command)
	}
	if got := shellQuote("foo 'bar'\n$(baz)"); got != `'foo '"'"'bar'"'"'
$(baz)'` {
		t.Fatalf("shellQuote() = %q", got)
	}

	parsedHooks, err := parseWorkflowHooks(map[string]any{
		"workflow_hooks": map[string]any{
			"on_activate": []map[string]any{{
				"cmd":        "printf 'hello'",
				"timeout":    2,
				"on_failure": "ignore",
			}},
		},
	})
	if err != nil || len(parsedHooks.OnActivate) != 1 || parsedHooks.OnActivate[0].Timeout != 2*time.Second {
		t.Fatalf("parseWorkflowHooks() = %+v, %v", parsedHooks, err)
	}
	if _, err := parseWorkflowHooks(map[string]any{"workflow_hooks": "bad"}); err == nil {
		t.Fatal("parseWorkflowHooks(invalid) expected error")
	}

	var hookLogBuffer bytes.Buffer
	executor := newWorkflowHookExecutor(t.TempDir(), slog.New(slog.NewTextHandler(&hookLogBuffer, nil)))
	if err := executor.RunAll(context.Background(), workflowHookOnReload, []workflowHookDefinition{{
		Command:   "printf 'ok'",
		OnFailure: workflowHookFailureBlock,
	}}, runtime); err != nil {
		t.Fatalf("RunAll(success) error = %v", err)
	}
	if err := executor.RunAll(context.Background(), workflowHookOnReload, []workflowHookDefinition{{
		Command:   "printf 'ignored' >&2; exit 7",
		OnFailure: workflowHookFailureIgnore,
	}}, runtime); err != nil {
		t.Fatalf("RunAll(ignore) error = %v", err)
	}
	if err := executor.RunAll(context.Background(), workflowHookOnReload, []workflowHookDefinition{{
		Command:   "printf 'warned' >&2; exit 7",
		OnFailure: workflowHookFailureWarn,
	}}, runtime); err != nil {
		t.Fatalf("RunAll(warn) error = %v", err)
	}
	if err := executor.RunAll(context.Background(), workflowHookOnReload, []workflowHookDefinition{{
		Command:   "exit 9",
		OnFailure: workflowHookFailureBlock,
	}}, runtime); err == nil || !strings.Contains(err.Error(), ErrWorkflowHookBlocked.Error()) {
		t.Fatalf("RunAll(block) error = %v", err)
	}
	if err := executor.run(context.Background(), workflowHookOnReload, workflowHookDefinition{
		Command: "sleep 1",
		Timeout: 10 * time.Millisecond,
	}, runtime); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("run(timeout) error = %v", err)
	}

	maliciousRuntime := workflowHookRuntime{
		ProjectID:       runtime.ProjectID,
		WorkflowID:      runtime.WorkflowID,
		WorkflowName:    "foo; touch pwned && printf hacked #\nline two 'quoted' $(boom)",
		WorkflowVersion: runtime.WorkflowVersion,
	}
	safeWorkspace := t.TempDir()
	executor = newWorkflowHookExecutor(safeWorkspace, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := executor.run(context.Background(), workflowHookOnReload, workflowHookDefinition{
		Command: "printf '%s' {{ workflow.name }} > safe.txt",
	}, maliciousRuntime); err != nil {
		t.Fatalf("run(shell-safe interpolation) error = %v", err)
	}
	//nolint:gosec // test reads a file it created under t.TempDir-backed workspace.
	content, err := os.ReadFile(filepath.Join(safeWorkspace, "safe.txt"))
	if err != nil {
		t.Fatalf("ReadFile(safe.txt) error = %v", err)
	}
	if got := string(content); got != maliciousRuntime.WorkflowName {
		t.Fatalf("safe.txt = %q, want %q", got, maliciousRuntime.WorkflowName)
	}
	if _, err := os.Stat(filepath.Join(safeWorkspace, "pwned")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected pwned file to be absent, got err=%v", err)
	}

	hookLogs := hookLogBuffer.String()
	for _, want := range []string{
		"workflow hook started",
		"workflow hook completed",
		"workflow hook failed but was ignored",
		"workflow hook failed with warning policy",
		"hook_scope=workflow",
		"workflow_version=7",
	} {
		if !strings.Contains(hookLogs, want) {
			t.Fatalf("expected hook logs to contain %q, got %s", want, hookLogs)
		}
	}
}

func TestWorkflowServiceLifecycleCoverage(t *testing.T) {
	if got := mapTicketHookConfigError(nil); got != nil {
		t.Fatalf("mapTicketHookConfigError(nil) = %v", got)
	}
	mapped := mapTicketHookConfigError(infrahook.ErrConfigInvalid)
	if mapped == nil || !strings.Contains(mapped.Error(), ErrHookConfigInvalid.Error()) {
		t.Fatalf("mapTicketHookConfigError() = %v", mapped)
	}

	statuses := []*ent.TicketStatus{{Name: "Backlog"}, {Name: "Done"}}
	if got := statusNamesFromEdges(statuses); len(got) != 2 || got[0] != "Backlog" || got[1] != "Done" {
		t.Fatalf("statusNamesFromEdges() = %+v", got)
	}

	content := defaultHarnessContent("Coverage Workflow", TypeCoding, []string{"Todo"}, []string{"Done"})
	if !strings.Contains(content, "# Coverage Workflow") || !strings.Contains(content, "Pickup statuses: Todo") || !strings.Contains(content, "Finish statuses: Done") {
		t.Fatalf("defaultHarnessContent() = %q", content)
	}

	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	generated, err := service.resolveHarnessContent(
		ctx,
		"Coverage Workflow",
		TypeCoding,
		MustStatusBindingSet(fixture.statusIDs["Todo"]),
		MustStatusBindingSet(fixture.statusIDs["Done"]),
		"",
	)
	if err != nil {
		t.Fatalf("resolveHarnessContent(default) error = %v", err)
	}
	if !strings.Contains(generated, "Pickup statuses: Todo") || !strings.Contains(generated, "Finish statuses: Done") {
		t.Fatalf("resolveHarnessContent(default) = %q", generated)
	}
	if _, err := service.resolveHarnessContent(
		ctx,
		"Coverage Workflow",
		TypeCoding,
		MustStatusBindingSet(fixture.statusIDs["Todo"]),
		MustStatusBindingSet(fixture.statusIDs["Done"]),
		"{{",
	); err == nil {
		t.Fatal("resolveHarnessContent(invalid raw content) expected error")
	}
}

func TestWorkflowServiceSkillAndHookHelperCoverage(t *testing.T) {
	ctx := context.Background()
	nilService := &Service{}
	projectID := uuid.New()

	if _, err := nilService.ListSkills(ctx, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListSkills(nil service) error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := nilService.RefreshSkills(ctx, RefreshSkillsInput{ProjectID: projectID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RefreshSkills(nil service) error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := nilService.BindSkills(ctx, UpdateWorkflowSkillsInput{WorkflowID: uuid.New(), Skills: []string{"skill-one"}}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("BindSkills(nil service) error = %v, want %v", err, ErrUnavailable)
	}

	service := &Service{}
	if err := service.runWorkflowHooks(ctx, uuid.Nil, workflowHooksConfig{}, workflowHookOnReload, workflowHookRuntime{}); err != nil {
		t.Fatalf("runWorkflowHooks(nil storage) error = %v", err)
	}
	service.projectsRoot = t.TempDir()
	if err := service.runWorkflowHooks(ctx, uuid.Nil, workflowHooksConfig{}, workflowHookName("unexpected"), workflowHookRuntime{}); err != nil {
		t.Fatalf("runWorkflowHooks(default) error = %v", err)
	}

	if got := defaultHarnessPath("   "); got != ".openase/harnesses/workflow.md" {
		t.Fatalf("defaultHarnessPath(blank) = %q", got)
	}
	if got := defaultHarnessPath("Coverage Workflow"); got != ".openase/harnesses/coverage-workflow.md" {
		t.Fatalf("defaultHarnessPath(value) = %q", got)
	}

	for _, testCase := range []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: " ", want: "harness_path must not be empty"},
		{name: "absolute", path: filepath.Join(string(os.PathSeparator), "tmp", "workflow.md"), want: "harness_path must be relative to the project control root"},
		{name: "escape", path: "../workflow.md", want: "harness_path must stay within the project control root"},
		{name: "outside root", path: "workflow.md", want: "harness_path must stay under .openase/harnesses/"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := normalizeHarnessPath(testCase.path); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("normalizeHarnessPath(%q) error = %v, want substring %q", testCase.path, err, testCase.want)
			}
		})
	}

	normalizedPath, err := normalizeHarnessPath(" .openase/harnesses/nested/coverage.md ")
	if err != nil {
		t.Fatalf("normalizeHarnessPath(valid) error = %v", err)
	}
	if normalizedPath != ".openase/harnesses/nested/coverage.md" {
		t.Fatalf("normalizeHarnessPath(valid) = %q", normalizedPath)
	}
}

func TestWorkflowServiceHelperCoverage(t *testing.T) {
	repoRoot := createWorkflowTestGitRepo(t)

	service, err := NewService(nil, nil, repoRoot)
	if err != nil {
		t.Fatalf("NewService(projectsRoot) error = %v", err)
	}
	if service.logger == nil || service.projectsRoot != repoRoot {
		t.Fatalf("NewService(projectsRoot) = %+v", service)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("Service.Close() error = %v", err)
	}
	if got, err := (*Service)(nil).ProjectControlRoot(uuid.New()); err != nil || got != "" {
		t.Fatalf("(*Service)(nil).ProjectControlRoot() = %q, %v", got, err)
	}
	if err := (*Service)(nil).Close(); err != nil {
		t.Fatalf("(*Service)(nil).Close() error = %v", err)
	}

	projectID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
	projectControlRoot, err := service.ProjectControlRoot(projectID)
	if err != nil {
		t.Fatalf("ProjectControlRoot() error = %v", err)
	}
	wantProjectControlRoot, err := workspaceinfra.ProjectStatePath(repoRoot, projectID.String())
	if err != nil {
		t.Fatalf("ProjectStatePath() error = %v", err)
	}
	if projectControlRoot != wantProjectControlRoot {
		t.Fatalf("ProjectControlRoot() = %q, want %q", projectControlRoot, wantProjectControlRoot)
	}

	if _, err := NewService(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), ""); err == nil || !strings.Contains(err.Error(), "projects_root must not be empty") {
		t.Fatalf("NewService(empty projects root) error = %v", err)
	}

	if got := service.mapWorkflowWriteError("write workflow", ErrWorkflowHarnessPathConflict); !errors.Is(got, ErrWorkflowHarnessPathConflict) {
		t.Fatalf("mapWorkflowWriteError(harness path conflict) = %v", got)
	}
	if got := service.mapWorkflowWriteError("write workflow", errors.New("boom")); got == nil || got.Error() != "write workflow: boom" {
		t.Fatalf("mapWorkflowWriteError(default) = %v", got)
	}

	nilClientService := &Service{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := nilClientService.Close(); err != nil {
		t.Fatalf("Service.Close() error = %v", err)
	}
}

func mustWriteSkill(t *testing.T, dir string, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir skill dir %s: %v", dir, err)
	}
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		title := parseSkillTitle(content)
		if title == "" {
			title = filepath.Base(dir)
		}
		content = fmt.Sprintf("---\nname: %q\ndescription: %q\n---\n\n%s\n", filepath.Base(dir), title, strings.TrimSpace(content))
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("write SKILL.md in %s: %v", dir, err)
	}
}
