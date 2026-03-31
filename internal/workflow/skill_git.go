package workflow

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const skillIndexRelativePath = ".openase/skills/.openase-index.json"

func (s *Service) commitSkillMutation(repoRoot string, message string, prefixes ...string) error {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil
	}

	repository, err := git.PlainOpen(repoRoot)
	if err != nil {
		return fmt.Errorf("open git repository for skill mutation: %w", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("load git worktree for skill mutation: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("inspect git status for skill mutation: %w", err)
	}

	normalizedPrefixes := normalizeGitPrefixes(prefixes)
	changedPaths := make([]string, 0, len(status))
	for rawPath := range status {
		path := filepath.ToSlash(rawPath)
		if !matchesGitPrefix(path, normalizedPrefixes) {
			continue
		}
		changedPaths = append(changedPaths, path)
	}
	sort.Strings(changedPaths)
	if len(changedPaths) == 0 {
		return nil
	}

	for _, path := range changedPaths {
		fileStatus := status[path]
		if fileStatus.Staging == git.Deleted || fileStatus.Worktree == git.Deleted {
			if _, err := worktree.Remove(path); err != nil && !errorsIsNotExist(err) {
				return fmt.Errorf("stage deleted skill path %s: %w", path, err)
			}
			continue
		}
		if _, err := worktree.Add(path); err != nil {
			return fmt.Errorf("stage skill path %s: %w", path, err)
		}
	}

	if _, err := worktree.Commit(trimmedMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OpenASE",
			Email: "openase@local",
			When:  time.Now().UTC(),
		},
	}); err != nil {
		return fmt.Errorf("commit skill mutation: %w", err)
	}

	return nil
}

func normalizeGitPrefixes(prefixes []string) []string {
	items := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		normalized := strings.TrimSpace(filepath.ToSlash(prefix))
		normalized = strings.TrimPrefix(normalized, "./")
		normalized = strings.Trim(normalized, "/")
		if normalized == "" {
			continue
		}
		items = append(items, normalized)
	}
	sort.Strings(items)
	return slicesCompact(items)
}

func matchesGitPrefix(path string, prefixes []string) bool {
	normalizedPath := strings.TrimSpace(filepath.ToSlash(path))
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")
	normalizedPath = strings.Trim(normalizedPath, "/")
	for _, prefix := range prefixes {
		if normalizedPath == prefix || strings.HasPrefix(normalizedPath, prefix+"/") {
			return true
		}
	}
	return false
}

func skillRelativePath(name string) string {
	return filepath.ToSlash(filepath.Join(".openase", "skills", name))
}

func workflowSkillCommitMessage(action string, workflowName string, skillNames []string) string {
	summary := summarizeSkillNames(skillNames)
	if strings.TrimSpace(workflowName) == "" {
		return fmt.Sprintf("feat(skills): %s %s", action, summary)
	}
	return fmt.Sprintf("feat(skills): %s %s for %s", action, summary, workflowName)
}

func skillCommitMessage(action string, skillName string) string {
	return fmt.Sprintf("feat(skills): %s %s", action, strings.TrimSpace(skillName))
}

func harvestSkillCommitMessage(harvested []string, updated []string) string {
	switch {
	case len(harvested) == 1 && len(updated) == 0:
		return fmt.Sprintf("feat(skills): auto-harvest %s", harvested[0])
	case len(harvested) > 1 && len(updated) == 0:
		return fmt.Sprintf("feat(skills): auto-harvest %d skills", len(harvested))
	case len(harvested) == 0 && len(updated) == 1:
		return fmt.Sprintf("feat(skills): refresh %s", updated[0])
	default:
		return fmt.Sprintf("feat(skills): persist %s", summarizeSkillNames(append(append([]string(nil), harvested...), updated...)))
	}
}

func summarizeSkillNames(skillNames []string) string {
	names := append([]string(nil), skillNames...)
	sort.Strings(names)
	switch len(names) {
	case 0:
		return "skills"
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return fmt.Sprintf("%d skills", len(names))
	}
}

func slicesCompact(items []string) []string {
	if len(items) == 0 {
		return items
	}
	writeIndex := 1
	for readIndex := 1; readIndex < len(items); readIndex++ {
		if items[readIndex] == items[readIndex-1] {
			continue
		}
		items[writeIndex] = items[readIndex]
		writeIndex++
	}
	return items[:writeIndex]
}
