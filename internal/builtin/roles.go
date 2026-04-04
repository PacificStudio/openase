package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:roles
var builtinRoleFS embed.FS

// RoleTemplate describes a built-in workflow role scaffold.
type RoleTemplate struct {
	Slug                  string
	Name                  string
	WorkflowType          string
	Summary               string
	HarnessPath           string
	Content               string
	PickupStatusNames     []string
	FinishStatusNames     []string
	SkillNames            []string
	PlatformAccessAllowed []string
}

type roleMetadata struct {
	Name                  string
	WorkflowType          string
	PickupStatusNames     []string
	FinishStatusNames     []string
	SkillNames            []string
	PlatformAccessAllowed []string
}

var builtinRoleOrder = []string{
	"dispatcher",
	"harness-optimizer",
	"env-provisioner",
	"fullstack-developer",
	"frontend-engineer",
	"backend-engineer",
	"qa-engineer",
	"devops-engineer",
	"security-engineer",
	"technical-writer",
	"code-reviewer",
	"product-manager",
	"market-analyst",
	"research-ideation",
	"experiment-runner",
	"report-writer",
	"data-analyst",
}

var builtinRoleMetadata = map[string]roleMetadata{
	"backend-engineer": {
		Name:              "Backend Engineer",
		WorkflowType:      "Backend Engineer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "pull", "commit", "push"},
	},
	"code-reviewer": {
		Name:              "Code Reviewer",
		WorkflowType:      "Code Reviewer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "review-code"},
	},
	"data-analyst": {
		Name:              "Data Analyst",
		WorkflowType:      "Data Analyst",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform"},
	},
	"devops-engineer": {
		Name:              "DevOps Engineer",
		WorkflowType:      "DevOps Engineer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "pull", "push"},
	},
	"dispatcher": {
		Name:              "Dispatcher",
		WorkflowType:      "Dispatcher",
		PickupStatusNames: []string{"Backlog"},
		FinishStatusNames: []string{"Todo"},
		PlatformAccessAllowed: []string{
			"activity.read",
			"statuses.list",
			"tickets.create",
			"tickets.list",
			"tickets.update.self",
			"workflows.list",
		},
	},
	"env-provisioner": {
		Name:              "Environment Provisioner",
		WorkflowType:      "Environment Provisioner",
		PickupStatusNames: []string{"环境修复"},
		FinishStatusNames: []string{"环境就绪"},
		SkillNames:        []string{"openase-platform", "install-claude-code", "install-codex", "setup-git", "setup-gh-cli"},
	},
	"experiment-runner": {
		Name:              "Experiment Runner",
		WorkflowType:      "Experiment Runner",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "write-test"},
	},
	"frontend-engineer": {
		Name:              "Frontend Engineer",
		WorkflowType:      "Frontend Engineer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "pull", "commit", "push"},
	},
	"fullstack-developer": {
		Name:              "Fullstack Developer",
		WorkflowType:      "Fullstack Developer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "pull", "commit", "push"},
	},
	"harness-optimizer": {
		Name:                  "Harness Optimizer",
		WorkflowType:          "Harness Optimizer",
		PickupStatusNames:     []string{"Todo"},
		FinishStatusNames:     []string{"Done"},
		SkillNames:            []string{"openase-platform", "pull", "commit", "push"},
		PlatformAccessAllowed: []string{"tickets.create", "tickets.list", "tickets.update.self"},
	},
	"market-analyst": {
		Name:              "Market Analyst",
		WorkflowType:      "Market Analyst",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform"},
	},
	"product-manager": {
		Name:              "Product Manager",
		WorkflowType:      "Product Manager",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform"},
	},
	"qa-engineer": {
		Name:              "QA Engineer",
		WorkflowType:      "QA Engineer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "write-test"},
	},
	"report-writer": {
		Name:              "Report Writer",
		WorkflowType:      "Report Writer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "commit"},
	},
	"research-ideation": {
		Name:              "Research Ideation",
		WorkflowType:      "Research Ideation",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform"},
	},
	"security-engineer": {
		Name:              "Security Engineer",
		WorkflowType:      "Security Engineer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "security-scan"},
	},
	"technical-writer": {
		Name:              "Technical Writer",
		WorkflowType:      "Technical Writer",
		PickupStatusNames: []string{"Todo"},
		FinishStatusNames: []string{"Done"},
		SkillNames:        []string{"openase-platform", "commit"},
	},
}

// Roles returns the built-in role templates.
func Roles() []RoleTemplate {
	return cloneRoles(builtinRoles)
}

// RoleBySlug returns a built-in role template by slug.
func RoleBySlug(slug string) (RoleTemplate, bool) {
	for _, item := range builtinRoles {
		if item.Slug == slug {
			return item, true
		}
	}

	return RoleTemplate{}, false
}

func cloneRoles(items []RoleTemplate) []RoleTemplate {
	cloned := make([]RoleTemplate, len(items))
	for index, item := range items {
		cloned[index] = item
		cloned[index].PickupStatusNames = append([]string(nil), item.PickupStatusNames...)
		cloned[index].FinishStatusNames = append([]string(nil), item.FinishStatusNames...)
		cloned[index].SkillNames = append([]string(nil), item.SkillNames...)
		cloned[index].PlatformAccessAllowed = append([]string(nil), item.PlatformAccessAllowed...)
	}
	return cloned
}

func mustLoadBuiltinRoles() []RoleTemplate {
	templates, err := loadBuiltinRoles(builtinRoleFS)
	if err != nil {
		panic(fmt.Sprintf("load builtin roles: %v", err))
	}
	return templates
}

func loadBuiltinRoles(root fs.FS) ([]RoleTemplate, error) {
	if err := validateBuiltinRoleFiles(root); err != nil {
		return nil, err
	}

	templates := make([]RoleTemplate, 0, len(builtinRoleOrder))
	for _, slug := range builtinRoleOrder {
		template, err := loadBuiltinRole(root, slug)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

func validateBuiltinRoleFiles(root fs.FS) error {
	entries, err := fs.ReadDir(root, "roles")
	if err != nil {
		return fmt.Errorf("read builtin roles directory: %w", err)
	}

	expected := make(map[string]struct{}, len(builtinRoleOrder))
	for _, slug := range builtinRoleOrder {
		expected[slug+".md"] = struct{}{}
	}

	extras := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := expected[name]; ok {
			delete(expected, name)
			continue
		}
		extras = append(extras, name)
	}

	if len(expected) == 0 && len(extras) == 0 {
		return nil
	}

	missing := make([]string, 0, len(expected))
	for name := range expected {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	sort.Strings(extras)

	parts := make([]string, 0, 2)
	if len(missing) > 0 {
		parts = append(parts, "missing "+strings.Join(missing, ", "))
	}
	if len(extras) > 0 {
		parts = append(parts, "unexpected "+strings.Join(extras, ", "))
	}
	return fmt.Errorf("builtin roles directory mismatch: %s", strings.Join(parts, "; "))
}

func loadBuiltinRole(root fs.FS, slug string) (RoleTemplate, error) {
	metadata, ok := builtinRoleMetadata[slug]
	if !ok {
		return RoleTemplate{}, fmt.Errorf("builtin role metadata missing for %s", slug)
	}

	contentBytes, err := fs.ReadFile(root, "roles/"+slug+".md")
	if err != nil {
		return RoleTemplate{}, fmt.Errorf("read builtin role %s: %w", slug, err)
	}
	content := strings.TrimSpace(strings.ReplaceAll(string(contentBytes), "\r\n", "\n"))
	if content == "" {
		return RoleTemplate{}, fmt.Errorf("builtin role %s content must not be empty", slug)
	}
	if strings.HasPrefix(content, "---\n") || content == "---" {
		return RoleTemplate{}, fmt.Errorf("builtin role %s content must not contain YAML frontmatter", slug)
	}

	summary := parseBuiltinRoleSummary(content)
	if summary == "" {
		return RoleTemplate{}, fmt.Errorf("parse builtin role %s: missing summary paragraph after heading", slug)
	}

	return RoleTemplate{
		Slug:                  slug,
		Name:                  metadata.Name,
		WorkflowType:          metadata.WorkflowType,
		Summary:               summary,
		HarnessPath:           filepath.ToSlash(filepath.Join(".openase", "harnesses", "roles", slug+".md")),
		Content:               content + "\n",
		PickupStatusNames:     append([]string(nil), metadata.PickupStatusNames...),
		FinishStatusNames:     append([]string(nil), metadata.FinishStatusNames...),
		SkillNames:            append([]string(nil), metadata.SkillNames...),
		PlatformAccessAllowed: append([]string(nil), metadata.PlatformAccessAllowed...),
	}, nil
}

func parseBuiltinRoleSummary(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	foundHeading := false
	paragraph := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !foundHeading {
			if strings.HasPrefix(trimmed, "# ") {
				foundHeading = true
			}
			continue
		}
		if trimmed == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		paragraph = append(paragraph, trimmed)
	}

	return strings.Join(paragraph, " ")
}

var builtinRoles = mustLoadBuiltinRoles()
