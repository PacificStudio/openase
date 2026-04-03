package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

//go:embed all:skills
var builtinSkillFS embed.FS

// SkillTemplate describes a built-in skill scaffold.
type SkillTemplate struct {
	Name        string
	Title       string
	Description string
	Content     string
	Files       []SkillTemplateFile
}

// SkillTemplateFile describes a projected file that belongs to a built-in skill.
type SkillTemplateFile struct {
	Path         string
	Content      []byte
	IsExecutable bool
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Skills returns the built-in skill templates.
func Skills() []SkillTemplate {
	return cloneSkills(builtinSkills)
}

// SkillByName returns a built-in skill template by name.
func SkillByName(name string) (SkillTemplate, bool) {
	for _, item := range builtinSkills {
		if item.Name == name {
			return item, true
		}
	}

	return SkillTemplate{}, false
}

// IsBuiltinSkill reports whether a skill name belongs to the built-in set.
func IsBuiltinSkill(name string) bool {
	_, ok := SkillByName(name)
	return ok
}

func cloneSkills(items []SkillTemplate) []SkillTemplate {
	cloned := make([]SkillTemplate, len(items))
	for index, item := range items {
		cloned[index] = SkillTemplate{
			Name:        item.Name,
			Title:       item.Title,
			Description: item.Description,
			Content:     item.Content,
			Files:       make([]SkillTemplateFile, len(item.Files)),
		}
		for fileIndex, file := range item.Files {
			cloned[index].Files[fileIndex] = SkillTemplateFile{
				Path:         file.Path,
				Content:      append([]byte(nil), file.Content...),
				IsExecutable: file.IsExecutable,
			}
		}
	}
	return cloned
}

func mustLoadBuiltinSkills() []SkillTemplate {
	templates, err := loadBuiltinSkills(builtinSkillFS)
	if err != nil {
		panic(fmt.Sprintf("load builtin skills: %v", err))
	}
	return templates
}

func loadBuiltinSkills(root fs.FS) ([]SkillTemplate, error) {
	entries, err := fs.ReadDir(root, "skills")
	if err != nil {
		return nil, fmt.Errorf("read builtin skills directory: %w", err)
	}

	templates := make([]SkillTemplate, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		template, err := loadBuiltinSkill(root, entry.Name())
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	sort.Slice(templates, func(i int, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	return templates, nil
}

func loadBuiltinSkill(root fs.FS, skillDir string) (SkillTemplate, error) {
	contentBytes, err := fs.ReadFile(root, "skills/"+skillDir+"/SKILL.md")
	if err != nil {
		return SkillTemplate{}, fmt.Errorf("read builtin skill %s: %w", skillDir, err)
	}
	content := string(contentBytes)
	document, body, err := parseBuiltinSkillContent(content)
	if err != nil {
		return SkillTemplate{}, fmt.Errorf("parse builtin skill %s: %w", skillDir, err)
	}
	if strings.TrimSpace(document.Name) == "" {
		return SkillTemplate{}, fmt.Errorf("parse builtin skill %s: missing frontmatter name", skillDir)
	}
	if document.Name != skillDir {
		return SkillTemplate{}, fmt.Errorf(
			"parse builtin skill %s: frontmatter name %q does not match directory",
			skillDir,
			document.Name,
		)
	}
	if strings.TrimSpace(document.Description) == "" {
		return SkillTemplate{}, fmt.Errorf("parse builtin skill %s: missing frontmatter description", skillDir)
	}
	files, err := loadBuiltinSkillFiles(root, skillDir)
	if err != nil {
		return SkillTemplate{}, err
	}

	return SkillTemplate{
		Name:        document.Name,
		Title:       parseBuiltinSkillTitle(body),
		Description: document.Description,
		Content:     content,
		Files:       files,
	}, nil
}

func loadBuiltinSkillFiles(root fs.FS, skillDir string) ([]SkillTemplateFile, error) {
	baseDir := "skills/" + skillDir
	files := make([]SkillTemplateFile, 0)
	if err := fs.WalkDir(root, baseDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk builtin skill %s: %w", skillDir, walkErr)
		}
		if entry.IsDir() {
			return nil
		}

		relativePath := strings.TrimPrefix(path, baseDir+"/")
		content, err := fs.ReadFile(root, path)
		if err != nil {
			return fmt.Errorf("read builtin skill file %s/%s: %w", skillDir, relativePath, err)
		}
		info, err := fs.Stat(root, path)
		if err != nil {
			return fmt.Errorf("stat builtin skill file %s/%s: %w", skillDir, relativePath, err)
		}
		files = append(files, SkillTemplateFile{
			Path:         relativePath,
			Content:      content,
			IsExecutable: strings.HasPrefix(relativePath, "scripts/") || info.Mode().Perm()&0o111 != 0,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func parseBuiltinSkillContent(content string) (skillFrontmatter, string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return skillFrontmatter{}, "", fmt.Errorf("missing YAML frontmatter prefix")
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
		return skillFrontmatter{}, "", fmt.Errorf("missing YAML frontmatter closing delimiter")
	}

	var document skillFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &document); err != nil {
		return skillFrontmatter{}, "", fmt.Errorf("unmarshal frontmatter: %w", err)
	}
	body := strings.Join(lines[end+1:], "\n")
	return document, body, nil
}

func parseBuiltinSkillTitle(body string) string {
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

var builtinSkills = mustLoadBuiltinSkills()
