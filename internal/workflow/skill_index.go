package workflow

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/google/uuid"
)

const skillIndexFilename = ".openase-index.json"

type skillIndex struct {
	Version int               `json:"version"`
	Skills  []skillIndexEntry `json:"skills"`
}

type skillIndexEntry struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func loadSkillIndex(skillRoot string) (map[string]skillIndexEntry, error) {
	indexPath := filepath.Join(filepath.Clean(skillRoot), skillIndexFilename)
	//nolint:gosec // indexPath always targets the service-owned skill root plus a fixed filename.
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if errorsIsNotExist(err) {
			return map[string]skillIndexEntry{}, nil
		}
		return nil, fmt.Errorf("read skill index: %w", err)
	}

	var document skillIndex
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse skill index: %w", err)
	}

	items := make(map[string]skillIndexEntry, len(document.Skills))
	for _, item := range document.Skills {
		items[item.Name] = item
	}

	return items, nil
}

func saveSkillIndex(skillRoot string, items map[string]skillIndexEntry) error {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)

	document := skillIndex{
		Version: 1,
		Skills:  make([]skillIndexEntry, 0, len(names)),
	}
	for _, name := range names {
		document.Skills = append(document.Skills, items[name])
	}

	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal skill index: %w", err)
	}
	payload = append(payload, '\n')

	indexPath := filepath.Join(skillRoot, skillIndexFilename)
	if err := os.WriteFile(indexPath, payload, 0o600); err != nil {
		return fmt.Errorf("write skill index: %w", err)
	}

	return nil
}

func (s *Service) syncSkillIndex(skillRoot string, now time.Time) (map[string]skillIndexEntry, error) {
	items, err := loadSkillIndex(skillRoot)
	if err != nil {
		return nil, err
	}

	names, err := listSkillNames(skillRoot)
	if err != nil {
		return nil, err
	}
	validNames := make(map[string]struct{}, len(names))
	changed := false
	for _, name := range names {
		validNames[name] = struct{}{}
		if _, ok := items[name]; ok {
			continue
		}
		changed = true
		items[name] = skillIndexEntry{
			ID:        uuid.New(),
			Name:      name,
			IsEnabled: true,
			CreatedBy: defaultSkillCreator(name),
			CreatedAt: now.UTC(),
		}
	}

	for name := range items {
		if _, ok := validNames[name]; ok {
			continue
		}
		changed = true
		delete(items, name)
	}

	if changed {
		if err := saveSkillIndex(skillRoot, items); err != nil {
			return nil, err
		}
	}

	return items, nil
}

func defaultSkillCreator(name string) string {
	if builtin.IsBuiltinSkill(name) {
		return "builtin:openase"
	}
	return "user:unknown"
}

func errorsIsNotExist(err error) bool {
	return err != nil && (os.IsNotExist(err) || strings.Contains(err.Error(), fs.ErrNotExist.Error()))
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

func resolveWorkspaceSkillRoot(storage *projectStorage, workspaceRoot string) string {
	trimmed := strings.TrimSpace(workspaceRoot)
	if trimmed != "" {
		candidate := filepath.Join(trimmed, ".openase", "skills")
		indexPath := filepath.Join(candidate, skillIndexFilename)
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			if indexStat, indexErr := os.Stat(indexPath); indexErr == nil && !indexStat.IsDir() {
				return candidate
			}
		}
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() && workspaceOwnsSkillLibrary(trimmed) {
			return candidate
		}
	}

	return storage.skillRoot
}

func workspaceOwnsSkillLibrary(workspaceRoot string) bool {
	markerPath := filepath.Join(workspaceRoot, ".openase", "skills", ".workspace-owned")
	stat, err := os.Stat(markerPath)
	return err == nil && !stat.IsDir()
}
