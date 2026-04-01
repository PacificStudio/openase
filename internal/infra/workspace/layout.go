package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/logging"
)

const ticketPlaceholder = "{ticket}"

var _ = logging.DeclareComponent("workspace-layout")

// LocalWorkspacePatternRoot is the display form for the local ticket workspace root.
const LocalWorkspacePatternRoot = "~/.openase/workspace"

// LocalWorkspaceRoot resolves the absolute local ticket workspace root.
func LocalWorkspaceRoot() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".openase", "workspace"), nil
}

// TicketWorkspacePath derives the absolute ticket workspace path.
func TicketWorkspacePath(workspaceRoot string, orgSlug string, projectSlug string, ticketIdentifier string) (string, error) {
	return deriveTicketWorkspace(workspaceRoot, orgSlug, projectSlug, ticketIdentifier, true)
}

// TicketWorkspacePattern derives the read-only ticket workspace convention with a ticket placeholder.
func TicketWorkspacePattern(workspaceRoot string, orgSlug string, projectSlug string) (string, error) {
	return deriveTicketWorkspace(workspaceRoot, orgSlug, projectSlug, ticketPlaceholder, false)
}

// RepoPath derives the repository path under a ticket workspace.
func RepoPath(workspacePath string, workspaceDirname string, repoName string) string {
	resolvedWorkspaceDirname := strings.TrimSpace(workspaceDirname)
	if resolvedWorkspaceDirname == "" {
		resolvedWorkspaceDirname = strings.TrimSpace(repoName)
	}
	if resolvedWorkspaceDirname == "" {
		return strings.TrimSpace(workspacePath)
	}

	return filepath.Join(strings.TrimSpace(workspacePath), filepath.FromSlash(resolvedWorkspaceDirname))
}

func deriveTicketWorkspace(workspaceRoot string, orgSlug string, projectSlug string, ticketIdentifier string, requireAbsoluteRoot bool) (string, error) {
	root, err := parseTicketWorkspaceRoot(workspaceRoot, requireAbsoluteRoot)
	if err != nil {
		return "", err
	}

	orgSegment, err := parsePathSegment("org_slug", orgSlug)
	if err != nil {
		return "", err
	}
	projectSegment, err := parsePathSegment("project_slug", projectSlug)
	if err != nil {
		return "", err
	}
	ticketSegment, err := parseTicketSegment("ticket_identifier", ticketIdentifier)
	if err != nil {
		return "", err
	}

	return filepath.Join(root, orgSegment, projectSegment, ticketSegment), nil
}

func parseTicketWorkspaceRoot(raw string, requireAbsolute bool) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("workspace_root must not be empty")
	}
	if requireAbsolute && !filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("workspace_root must be absolute")
	}

	return filepath.Clean(trimmed), nil
}

func parseTicketSegment(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == ticketPlaceholder {
		return trimmed, nil
	}

	return parsePathSegment(fieldName, trimmed)
}
