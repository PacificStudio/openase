package projectrepomirror

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

const localMirrorPatternRoot = "~/.openase/mirrors"

func deriveDefaultMirrorLocalPath(machine *ent.Machine, orgSlug string, projectSlug string, repoName string) (string, error) {
	mirrorRoot, err := deriveMirrorRoot(machine)
	if err != nil {
		return "", err
	}

	orgSegment, err := parseMirrorPathSegment("org_slug", orgSlug)
	if err != nil {
		return "", err
	}
	projectSegment, err := parseMirrorPathSegment("project_slug", projectSlug)
	if err != nil {
		return "", err
	}
	repoSegment, err := parseMirrorPathSegment("repo_name", repoName)
	if err != nil {
		return "", err
	}

	return filepath.Join(mirrorRoot, orgSegment, projectSegment, repoSegment), nil
}

func deriveMirrorRoot(machine *ent.Machine) (string, error) {
	if machine == nil {
		return "", fmt.Errorf("%w: machine must not be nil", ErrInvalidInput)
	}
	if configuredRoot := strings.TrimSpace(machine.MirrorRoot); configuredRoot != "" {
		return parseAbsoluteFieldPath("mirror_root", configuredRoot)
	}
	if strings.EqualFold(strings.TrimSpace(machine.Host), domain.LocalMachineHost) {
		return localMirrorRoot()
	}

	workspaceRoot := strings.TrimSpace(machine.WorkspaceRoot)
	if workspaceRoot == "" {
		return "", fmt.Errorf("%w: remote machine mirror_root requires workspace_root or explicit mirror_root", ErrInvalidInput)
	}
	parsedWorkspaceRoot, err := parseAbsoluteFieldPath("workspace_root", workspaceRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(parsedWorkspaceRoot), "mirrors"), nil
}

func localMirrorRoot() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory for mirror_root: %w", err)
	}
	return filepath.Join(homeDir, ".openase", "mirrors"), nil
}

func parseAbsoluteFieldPath(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s must not be empty", ErrInvalidInput, fieldName)
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("%w: %s must be absolute", ErrInvalidInput, fieldName)
	}
	return cleaned, nil
}

func parseMirrorPathSegment(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s must not be empty", ErrInvalidInput, fieldName)
	}
	if trimmed == "." || trimmed == ".." {
		return "", fmt.Errorf("%w: %s must be a stable path segment", ErrInvalidInput, fieldName)
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s must not contain path separators", ErrInvalidInput, fieldName)
	}
	return trimmed, nil
}
