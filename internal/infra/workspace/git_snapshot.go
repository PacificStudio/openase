package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/logging"
)

// GitCommandRunner executes a git command and returns the combined output.
type GitCommandRunner func(ctx context.Context, args []string, allowExitCodeOne bool) ([]byte, error)

// ErrGitWorkspaceUnavailable marks repo paths that are missing or not git workspaces yet.
var ErrGitWorkspaceUnavailable = errors.New("git workspace unavailable")

const emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

var _ = logging.DeclareComponent("workspace-git-snapshot")

// ReadWorkspaceGitBranch resolves the current branch and falls back to symbolic-ref for unborn HEADs.
func ReadWorkspaceGitBranch(ctx context.Context, repoPath string, run GitCommandRunner) (string, error) {
	if run == nil {
		return "", fmt.Errorf("git command runner is nil")
	}

	output, err := run(ctx, []string{"git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD"}, false)
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}
	if isGitWorkspaceUnavailableOutput(output) {
		return "", wrapGitWorkspaceUnavailable(output)
	}
	if !isGitUnbornHeadOutput(output) {
		return "", err
	}

	output, err = run(ctx, []string{"git", "-C", repoPath, "symbolic-ref", "-q", "--short", "HEAD"}, false)
	if err != nil {
		if isGitWorkspaceUnavailableOutput(output) {
			return "", wrapGitWorkspaceUnavailable(output)
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ReadWorkspaceGitNumstat reads numstat output and falls back to the empty tree when HEAD is unborn.
func ReadWorkspaceGitNumstat(ctx context.Context, repoPath string, run GitCommandRunner) ([]byte, error) {
	if run == nil {
		return nil, fmt.Errorf("git command runner is nil")
	}

	output, err := run(ctx, []string{"git", "-C", repoPath, "diff", "--numstat", "-z", "-M", "HEAD", "--"}, false)
	if err == nil {
		return output, nil
	}
	if isGitWorkspaceUnavailableOutput(output) {
		return nil, wrapGitWorkspaceUnavailable(output)
	}
	if !isGitUnbornHeadOutput(output) {
		return output, err
	}

	output, err = run(ctx, []string{"git", "-C", repoPath, "diff", "--numstat", "-z", "-M", emptyTreeHash, "--"}, false)
	if err != nil {
		if isGitWorkspaceUnavailableOutput(output) {
			return nil, wrapGitWorkspaceUnavailable(output)
		}
		return output, err
	}
	return output, nil
}

func wrapGitWorkspaceUnavailable(output []byte) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return ErrGitWorkspaceUnavailable
	}
	return fmt.Errorf("%w: %s", ErrGitWorkspaceUnavailable, message)
}

func isGitWorkspaceUnavailableOutput(output []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(trimmed, "not a git repository") ||
		strings.Contains(trimmed, "cannot change to") ||
		strings.Contains(trimmed, "no such file or directory")
}

func isGitUnbornHeadOutput(output []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(trimmed, "ambiguous argument 'head'") ||
		strings.Contains(trimmed, "bad revision 'head'") ||
		strings.Contains(trimmed, "unknown revision or path not in the working tree") ||
		strings.Contains(trimmed, "does not have any commits yet")
}
