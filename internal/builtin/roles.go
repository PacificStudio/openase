package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

//go:embed all:roles
var builtinRoleFS embed.FS

// RoleTemplate describes a built-in workflow role scaffold.
type RoleTemplate struct {
	Slug         string
	Name         string
	WorkflowType string
	Summary      string
	HarnessPath  string
	Content      string
}

type roleFrontmatter struct {
	Workflow struct {
		Name string `yaml:"name"`
		Type string `yaml:"type"`
		Role string `yaml:"role"`
	} `yaml:"workflow"`
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
	copy(cloned, items)
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
	contentBytes, err := fs.ReadFile(root, "roles/"+slug+".md")
	if err != nil {
		return RoleTemplate{}, fmt.Errorf("read builtin role %s: %w", slug, err)
	}
	content := string(contentBytes)
	document, body, err := parseBuiltinRoleContent(content)
	if err != nil {
		return RoleTemplate{}, fmt.Errorf("parse builtin role %s: %w", slug, err)
	}

	name := strings.TrimSpace(document.Workflow.Name)
	if name == "" {
		return RoleTemplate{}, fmt.Errorf("parse builtin role %s: missing workflow.name", slug)
	}
	workflowType := strings.TrimSpace(document.Workflow.Type)
	if workflowType == "" {
		return RoleTemplate{}, fmt.Errorf("parse builtin role %s: missing workflow.type", slug)
	}
	roleSlug := strings.TrimSpace(document.Workflow.Role)
	if roleSlug == "" {
		return RoleTemplate{}, fmt.Errorf("parse builtin role %s: missing workflow.role", slug)
	}
	if roleSlug != slug {
		return RoleTemplate{}, fmt.Errorf("parse builtin role %s: workflow.role %q does not match file slug", slug, roleSlug)
	}
	summary := parseBuiltinRoleSummary(body)
	if summary == "" {
		return RoleTemplate{}, fmt.Errorf("parse builtin role %s: missing summary paragraph after heading", slug)
	}

	return RoleTemplate{
		Slug:         slug,
		Name:         name,
		WorkflowType: workflowType,
		Summary:      summary,
		HarnessPath:  filepath.ToSlash(filepath.Join(".openase", "harnesses", "roles", slug+".md")),
		Content:      content,
	}, nil
}

func parseBuiltinRoleContent(content string) (roleFrontmatter, string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return roleFrontmatter{}, "", fmt.Errorf("missing YAML frontmatter prefix")
	}

	lines := strings.Split(normalized, "\n")
	end := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			end = index
			break
		}
	}
	if end == -1 {
		return roleFrontmatter{}, "", fmt.Errorf("missing YAML frontmatter closing delimiter")
	}

	var document roleFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &document); err != nil {
		return roleFrontmatter{}, "", fmt.Errorf("unmarshal frontmatter: %w", err)
	}

	body := strings.Join(lines[end+1:], "\n")
	return document, body, nil
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
