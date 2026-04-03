package workflow

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

var (
	ErrSkillInvalid  = domain.ErrSkillInvalid
	ErrSkillNotFound = domain.ErrSkillNotFound

	skillNamePattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*$`)
)

type Skill = domain.Skill

type SkillWorkflowBinding = domain.SkillWorkflowBinding

type skillDocument struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type SkillDetail = domain.SkillDetail

type CreateSkillInput = domain.CreateSkillInput

type UpdateSkillInput = domain.UpdateSkillInput

type CreateSkillBundleInput = domain.CreateSkillBundleInput

type UpdateSkillBundleInput = domain.UpdateSkillBundleInput

type UpdateSkillBindingsInput = domain.UpdateSkillBindingsInput

type RefreshSkillsInput = domain.RefreshSkillsInput

type RefreshSkillsResult = domain.RefreshSkillsResult

type UpdateWorkflowSkillsInput = domain.UpdateWorkflowSkillsInput

type resolvedSkillTarget struct {
	workspace provider.AbsolutePath
	skillsDir provider.AbsolutePath
}

type RuntimeSkillTarget struct {
	SkillsDir string
}

func (s *Service) ListSkills(ctx context.Context, projectID uuid.UUID) ([]Skill, error) {
	return s.listSkillsPersistent(ctx, projectID)
}

func parseSkillTitle(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}

	return ""
}

func (s *Service) GetSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	return s.getSkillPersistent(ctx, skillID)
}

func (s *Service) GetSkillInProject(ctx context.Context, projectID uuid.UUID, skillID uuid.UUID) (SkillDetail, error) {
	record, err := s.resolveSkillRecordInProject(ctx, projectID, skillID)
	if err != nil {
		return SkillDetail{}, err
	}
	return s.buildSkillDetail(ctx, record)
}

func (s *Service) ListSkillVersions(ctx context.Context, skillID uuid.UUID) ([]VersionSummary, error) {
	return s.listSkillVersionsPersistent(ctx, skillID)
}

func ensureSkillContent(name string, rawContent string, fallbackDescription string) (string, error) {
	trimmed := strings.TrimSpace(rawContent)
	if trimmed == "" {
		return "", fmt.Errorf("%w: content must not be empty", ErrSkillInvalid)
	}
	if strings.HasPrefix(trimmed, "---\n") || trimmed == "---" {
		document, body, err := parseSkillDocument(trimmed)
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrSkillInvalid, err)
		}
		if document.Name != name {
			return "", fmt.Errorf("%w: skill frontmatter name %q must match %q", ErrSkillInvalid, document.Name, name)
		}
		return buildSkillContent(name, document.Description, body), nil
	}

	description := strings.TrimSpace(fallbackDescription)
	if description == "" {
		description = parseSkillTitle(trimmed)
	}
	if description == "" {
		description = strings.ReplaceAll(name, "-", " ")
	}

	return buildSkillContent(name, description, trimmed), nil
}

func buildSkillContent(name string, description string, body string) string {
	return fmt.Sprintf(
		"---\nname: %q\ndescription: %q\n---\n\n%s\n",
		name,
		strings.TrimSpace(description),
		strings.TrimSpace(body),
	)
}

func (s *Service) CreateSkill(ctx context.Context, input CreateSkillInput) (SkillDetail, error) {
	bundle, err := buildSingleFileSkillBundle(input.Name, input.Content, input.Description)
	if err != nil {
		return SkillDetail{}, err
	}
	return s.CreateSkillBundle(ctx, CreateSkillBundleInput{
		ProjectID: input.ProjectID,
		Name:      bundle.Name,
		Files:     bundleFilesToInput(bundle.Files),
		CreatedBy: input.CreatedBy,
		Enabled:   input.Enabled,
	})
}

func (s *Service) CreateSkillBundle(ctx context.Context, input CreateSkillBundleInput) (SkillDetail, error) {
	return s.createSkillBundlePersistent(ctx, input)
}

func (s *Service) UpdateSkill(ctx context.Context, input UpdateSkillInput) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, input.SkillID)
	if err != nil {
		return SkillDetail{}, err
	}
	bundle, err := buildSingleFileSkillBundle(record.skill.Name, input.Content, input.Description)
	if err != nil {
		return SkillDetail{}, err
	}
	return s.UpdateSkillBundle(ctx, UpdateSkillBundleInput{
		SkillID: input.SkillID,
		Files:   bundleFilesToInput(bundle.Files),
	})
}

func bundleFilesToInput(files []SkillBundleFile) []SkillBundleFileInput {
	inputs := make([]SkillBundleFileInput, 0, len(files))
	for _, file := range files {
		inputs = append(inputs, SkillBundleFileInput{
			Path:         file.Path,
			Content:      append([]byte(nil), file.Content...),
			IsExecutable: file.IsExecutable,
			MediaType:    file.MediaType,
		})
	}
	return inputs
}

func (s *Service) UpdateSkillBundle(ctx context.Context, input UpdateSkillBundleInput) (SkillDetail, error) {
	return s.updateSkillBundlePersistent(ctx, input)
}

func (s *Service) DeleteSkill(ctx context.Context, skillID uuid.UUID) error {
	return s.deleteSkillPersistent(ctx, skillID)
}

func (s *Service) EnableSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	return s.setSkillEnabled(ctx, skillID, true)
}

func (s *Service) DisableSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	return s.setSkillEnabled(ctx, skillID, false)
}

func (s *Service) setSkillEnabled(ctx context.Context, skillID uuid.UUID, enabled bool) (SkillDetail, error) {
	return s.setSkillEnabledPersistent(ctx, skillID, enabled)
}

func (s *Service) BindSkill(ctx context.Context, input UpdateSkillBindingsInput) (SkillDetail, error) {
	record, workflowIDs, err := s.resolveSkillRecordForWorkflowBindings(ctx, input)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, workflowID := range workflowIDs {
		if _, err := s.updateWorkflowSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: workflowID,
			Skills:     []string{record.skill.Name},
		}, true); err != nil {
			return SkillDetail{}, err
		}
	}
	return s.buildSkillDetail(ctx, record)
}

func (s *Service) UnbindSkill(ctx context.Context, input UpdateSkillBindingsInput) (SkillDetail, error) {
	record, workflowIDs, err := s.resolveSkillRecordForWorkflowBindings(ctx, input)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, workflowID := range workflowIDs {
		if _, err := s.updateWorkflowSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: workflowID,
			Skills:     []string{record.skill.Name},
		}, false); err != nil {
			return SkillDetail{}, err
		}
	}
	return s.buildSkillDetail(ctx, record)
}

func normalizeWorkflowIDs(items []uuid.UUID) ([]uuid.UUID, error) {
	unique := make([]uuid.UUID, 0, len(items))
	seen := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if item == uuid.Nil {
			return nil, fmt.Errorf("%w: workflow id must not be empty", ErrSkillInvalid)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	if len(unique) == 0 {
		return nil, fmt.Errorf("%w: workflow ids must not be empty", ErrSkillInvalid)
	}
	return unique, nil
}

func (s *Service) resolveSkillRecordForWorkflowBindings(
	ctx context.Context,
	input UpdateSkillBindingsInput,
) (resolvedSkillRecord, []uuid.UUID, error) {
	return s.resolveSkillRecordForWorkflowBindingsPersistent(ctx, input)
}

func (s *Service) resolveSkillRecord(
	ctx context.Context,
	skillID uuid.UUID,
) (resolvedSkillRecord, error) {
	return s.resolveSkillRecordPersistent(ctx, skillID)
}

func (s *Service) resolveSkillRecordInProject(
	ctx context.Context,
	projectID uuid.UUID,
	skillID uuid.UUID,
) (resolvedSkillRecord, error) {
	return s.resolveSkillRecordInProjectPersistent(ctx, projectID, skillID)
}

func (s *Service) buildSkillDetail(ctx context.Context, record resolvedSkillRecord) (SkillDetail, error) {
	return s.buildSkillDetailPersistent(ctx, record)
}

func (s *Service) skillVersionFiles(ctx context.Context, versionID uuid.UUID) ([]SkillBundleFile, error) {
	return s.skillVersionFilesPersistent(ctx, versionID)
}

func (s *Service) resolveInjectedSkillNames(
	ctx context.Context,
	projectID uuid.UUID,
	workflowID *uuid.UUID,
) ([]string, error) {
	return s.resolveInjectedSkillNamesPersistent(ctx, projectID, workflowID)
}

func (s *Service) RefreshSkills(ctx context.Context, input RefreshSkillsInput) (RefreshSkillsResult, error) {
	return s.refreshSkillsPersistent(ctx, input)
}

func (s *Service) BindSkills(ctx context.Context, input UpdateWorkflowSkillsInput) (HarnessDocument, error) {
	return s.updateWorkflowSkills(ctx, input, true)
}

func (s *Service) UnbindSkills(ctx context.Context, input UpdateWorkflowSkillsInput) (HarnessDocument, error) {
	return s.updateWorkflowSkills(ctx, input, false)
}

func (s *Service) updateWorkflowSkills(
	ctx context.Context,
	input UpdateWorkflowSkillsInput,
	bind bool,
) (HarnessDocument, error) {
	return s.updateWorkflowSkillsPersistent(ctx, input, bind)
}

func writeProjectedSkill(skillsDir string, name string, content string) error {
	skillDir := filepath.Join(skillsDir, name)
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		return fmt.Errorf("create projected skill directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		return fmt.Errorf("write projected skill: %w", err)
	}
	return nil
}

func writeProjectedSkillBundle(skillsDir string, name string, files []SkillBundleFile, fallbackContent string) error {
	if len(files) == 0 {
		return writeProjectedSkill(skillsDir, name, fallbackContent)
	}

	skillDir := filepath.Join(skillsDir, name)
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("reset projected skill directory: %w", err)
	}
	for _, file := range files {
		targetPath := filepath.Join(skillDir, filepath.FromSlash(file.Path))
		if err := ensureCopyTargetWithinRoot(skillDir, targetPath); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
			return fmt.Errorf("create projected skill parent: %w", err)
		}
		mode := os.FileMode(0o600)
		if file.IsExecutable {
			mode = 0o700
		}
		if err := os.WriteFile(targetPath, file.Content, mode); err != nil {
			return fmt.Errorf("write projected skill file %s: %w", file.Path, err)
		}
	}
	return nil
}

func writeWorkspaceOpenASEWrapper(workspaceRoot string) error {
	dst := filepath.Join(workspaceRoot, ".openase", "bin", "openase")
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create workspace openase wrapper directory: %w", err)
	}
	if err := os.WriteFile(dst, []byte(workflowOpenASECLIWrapperScript()), 0o600); err != nil {
		return fmt.Errorf("write workspace openase wrapper: %w", err)
	}
	if err := os.Chmod(dst, 0o700); err != nil { //nolint:gosec // dst stays under the prepared workspace root.
		return fmt.Errorf("chmod workspace openase wrapper: %w", err)
	}
	return nil
}

func workflowOpenASECLIWrapperScript() string {
	return strings.TrimSpace(`
#!/bin/sh
set -eu

if [ -n "${OPENASE_REAL_BIN:-}" ]; then
  OPENASE_BIN="$OPENASE_REAL_BIN"
elif command -v openase >/dev/null 2>&1; then
  OPENASE_BIN="$(command -v openase)"
else
  echo "openase wrapper: could not find an installed openase binary" >&2
  echo "set OPENASE_REAL_BIN to the desired executable path" >&2
  exit 1
fi

exec "$OPENASE_BIN" "$@"
`) + "\n"
}

func ParseHarnessSkills(content string) ([]string, error) {
	if err := validateHarnessForSave(normalizeHarnessNewlines(content)); err != nil {
		return nil, err
	}
	return nil, nil
}

func setHarnessSkills(content string, skills []string) (string, error) {
	if _, err := normalizeSkillNames(skills); err != nil {
		return "", err
	}
	return sanitizeHarnessContent(content)
}

func resolveSkillTarget(workspaceRoot string, rawAdapterType string) (resolvedSkillTarget, error) {
	target, err := resolveSkillTargetPath(workspaceRoot, rawAdapterType)
	if err != nil {
		return resolvedSkillTarget{}, err
	}
	if err := os.MkdirAll(target.workspace.String(), 0o750); err != nil {
		return resolvedSkillTarget{}, fmt.Errorf("%w: create workspace root: %s", ErrSkillInvalid, err)
	}
	return target, nil
}

func resolveSkillTargetPath(workspaceRoot string, rawAdapterType string) (resolvedSkillTarget, error) {
	trimmedWorkspaceRoot := strings.TrimSpace(workspaceRoot)
	if trimmedWorkspaceRoot == "" {
		return resolvedSkillTarget{}, fmt.Errorf("%w: workspace_root must not be empty", ErrSkillInvalid)
	}
	absoluteWorkspaceRoot, err := filepath.Abs(trimmedWorkspaceRoot)
	if err != nil {
		return resolvedSkillTarget{}, fmt.Errorf("%w: resolve workspace root: %s", ErrSkillInvalid, err)
	}
	workspace, err := provider.ParseAbsolutePath(absoluteWorkspaceRoot)
	if err != nil {
		return resolvedSkillTarget{}, fmt.Errorf("%w: %s", ErrSkillInvalid, err)
	}

	adapterType := strings.ToLower(strings.TrimSpace(rawAdapterType))
	switch adapterType {
	case "claude-code-cli", "codex-app-server", "gemini-cli", "custom":
	default:
		return resolvedSkillTarget{}, fmt.Errorf("%w: adapter_type must be one of claude-code-cli, codex-app-server, gemini-cli, custom", ErrSkillInvalid)
	}

	var skillsDir string
	switch adapterType {
	case "claude-code-cli":
		skillsDir = filepath.Join(workspace.String(), ".claude", "skills")
	case "codex-app-server":
		skillsDir = filepath.Join(workspace.String(), ".codex", "skills")
	case "gemini-cli":
		skillsDir = filepath.Join(workspace.String(), ".gemini", "skills")
	default:
		skillsDir = filepath.Join(workspace.String(), ".agent", "skills")
	}

	return resolvedSkillTarget{
		workspace: workspace,
		skillsDir: provider.MustParseAbsolutePath(filepath.Clean(skillsDir)),
	}, nil
}

func ResolveSkillTargetForRuntime(workspaceRoot string, rawAdapterType string) (RuntimeSkillTarget, error) {
	target, err := resolveSkillTargetPath(workspaceRoot, rawAdapterType)
	if err != nil {
		return RuntimeSkillTarget{}, err
	}
	return RuntimeSkillTarget{SkillsDir: target.skillsDir.String()}, nil
}

func normalizeSkillNames(raw []string) ([]string, error) {
	normalized := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: skill name must not be empty", ErrSkillInvalid)
		}
		if !skillNamePattern.MatchString(trimmed) {
			return nil, fmt.Errorf("%w: skill name %q must match %s", ErrSkillInvalid, trimmed, skillNamePattern.String())
		}
		if !slicesContainsString(normalized, trimmed) {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized, nil
}

func listSkillNames(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("list skills in %s: %w", root, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, err := normalizeSkillNames([]string{name}); err != nil {
			continue
		}
		if err := validateSkillDirectory(filepath.Join(root, name)); err != nil {
			continue
		}
		names = append(names, name)
	}

	sort.Strings(names)
	return names, nil
}

func validateSkillDirectory(dir string) error {
	contentPath := filepath.Join(dir, "SKILL.md")
	//nolint:gosec // contentPath is resolved from validated skill sources
	content, err := os.ReadFile(contentPath)
	if err != nil {
		return fmt.Errorf("%w: missing SKILL.md in %s", ErrSkillInvalid, dir)
	}
	document, _, err := parseSkillDocument(string(content))
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSkillInvalid, err)
	}
	if document.Name != filepath.Base(dir) {
		return fmt.Errorf("%w: skill frontmatter name %q must match directory %q", ErrSkillInvalid, document.Name, filepath.Base(dir))
	}
	return nil
}

func parseSkillDocument(content string) (skillDocument, string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return skillDocument{}, "", fmt.Errorf("skill must begin with YAML frontmatter delimited by ---")
	}

	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "---" {
			continue
		}

		frontmatter := strings.Join(lines[1:index], "\n")
		body := strings.Join(lines[index+1:], "\n")
		if strings.TrimSpace(frontmatter) == "" {
			return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter must not be empty")
		}

		var document skillDocument
		if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
			return skillDocument{}, "", fmt.Errorf("parse skill YAML frontmatter: %w", err)
		}
		document.Name = strings.TrimSpace(document.Name)
		document.Description = strings.TrimSpace(document.Description)
		if _, err := normalizeSkillNames([]string{document.Name}); err != nil {
			return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter name is invalid: %w", err)
		}
		if document.Description == "" {
			return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter description must not be empty")
		}
		if strings.TrimSpace(body) == "" {
			return skillDocument{}, "", fmt.Errorf("skill body must not be empty")
		}
		return document, body, nil
	}

	return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter is missing the closing --- delimiter")
}

func skillContentRelativePath(name string) string {
	return filepath.ToSlash(filepath.Join(".openase", "skills", name, "SKILL.md"))
}

func replaceDirectory(src string, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("remove %s before replace: %w", dst, err)
	}
	return copyDirectory(src, dst)
}

func copyDirectory(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source directory %s: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source %s must be a directory", src)
	}

	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk source directory %s: %w", src, walkErr)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not supported in skill directories: %s", path)
		}

		relative, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("resolve relative skill path for %s: %w", path, err)
		}
		targetPath := filepath.Join(dst, relative)

		entryInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read file info for %s: %w", path, err)
		}

		if entry.IsDir() {
			return os.MkdirAll(targetPath, entryInfo.Mode().Perm())
		}

		//nolint:gosec // path comes from walking the validated source skill directory
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read skill file %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
			return fmt.Errorf("create skill file parent %s: %w", targetPath, err)
		}
		if err := ensureCopyTargetWithinRoot(dst, targetPath); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, content, entryInfo.Mode().Perm()); err != nil { //nolint:gosec // target path is validated to remain within the destination root
			return fmt.Errorf("write skill file %s: %w", targetPath, err)
		}
		return nil
	})
}

func ensureCopyTargetWithinRoot(root string, targetPath string) error {
	relative, err := filepath.Rel(root, targetPath)
	if err != nil {
		return fmt.Errorf("resolve skill copy target %s: %w", targetPath, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("skill copy target escapes root %s: %s", root, targetPath)
	}

	return nil
}

func directoryFingerprint(root string) (string, error) {
	hash := sha256.New()
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			relative = ""
		}

		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}

		hash.Write([]byte(relative))
		hash.Write([]byte{0})
		hash.Write([]byte(entryInfo.Mode().String()))
		hash.Write([]byte{0})

		if entry.IsDir() {
			return nil
		}

		//nolint:gosec // path comes from walking the validated fingerprint root
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fileHash := sha256.Sum256(content)
		hash.Write(fileHash[:])
		hash.Write([]byte{0})
		return nil
	}); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
