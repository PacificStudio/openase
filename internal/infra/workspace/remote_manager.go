package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/logging"
)

var _ = logging.DeclareComponent("workspace-remote-manager")

type PrepareFailureStage string

const (
	PrepareFailureStageTransport     PrepareFailureStage = "transport"
	PrepareFailureStageWorkspaceRoot PrepareFailureStage = "workspace_root"
	PrepareFailureStageRepoAuth      PrepareFailureStage = "repo_auth"
	PrepareFailureStageGitOperation  PrepareFailureStage = "git_operation"
)

type PrepareError struct {
	Stage   PrepareFailureStage
	Message string
	Cause   error
}

func (e *PrepareError) Error() string {
	if e == nil {
		return "prepare remote workspace failed"
	}

	stage := strings.TrimSpace(string(e.Stage))
	if stage == "" {
		stage = string(PrepareFailureStageGitOperation)
	}
	message := strings.TrimSpace(e.Message)
	switch {
	case message != "" && e.Cause != nil:
		return fmt.Sprintf("prepare remote workspace (%s): %s: %v", stage, message, e.Cause)
	case message != "":
		return fmt.Sprintf("prepare remote workspace (%s): %s", stage, message)
	case e.Cause != nil:
		return fmt.Sprintf("prepare remote workspace (%s): %v", stage, e.Cause)
	default:
		return fmt.Sprintf("prepare remote workspace (%s) failed", stage)
	}
}

func (e *PrepareError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// RemoteManager prepares ticket workspaces on a remote machine over SSH.
type RemoteManager struct {
	pool *sshinfra.Pool
}

type commandRunner interface {
	CombinedOutput(cmd string) ([]byte, error)
}

func NewRemoteManager(pool *sshinfra.Pool) *RemoteManager {
	return &RemoteManager{pool: pool}
}

func (m *RemoteManager) Prepare(ctx context.Context, machine domain.Machine, request SetupRequest) (Workspace, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil || m.pool == nil {
		return Workspace{}, wrapPrepareTransportError("", fmt.Errorf("remote workspace manager unavailable"))
	}
	if machine.Host == domain.LocalMachineHost {
		return Workspace{}, fmt.Errorf("local machine does not use remote workspace preparation")
	}

	client, err := m.pool.Get(ctx, machine)
	if err != nil {
		return Workspace{}, wrapPrepareTransportError(machine.Name, fmt.Errorf("get ssh client for machine %s: %w", machine.Name, err))
	}

	session, err := client.NewSession()
	if err != nil {
		return Workspace{}, wrapPrepareTransportError(machine.Name, fmt.Errorf("open ssh session: %w", err))
	}
	defer func() {
		_ = session.Close()
	}()

	return PrepareWithCommandRunner(session, request)
}

func PrepareWithCommandRunner(runner commandRunner, request SetupRequest) (Workspace, error) {
	if runner == nil {
		return Workspace{}, fmt.Errorf("command runner unavailable")
	}

	command, err := buildPrepareWorkspaceCommand(request)
	if err != nil {
		return Workspace{}, fmt.Errorf("build remote workspace command: %w", err)
	}
	if output, err := runner.CombinedOutput(command); err != nil {
		return Workspace{}, classifyPrepareWorkspaceFailure(request, err, string(output))
	}

	workspacePath, err := TicketWorkspacePath(
		request.WorkspaceRoot,
		request.OrganizationSlug,
		request.ProjectSlug,
		request.TicketIdentifier,
	)
	if err != nil {
		return Workspace{}, fmt.Errorf("derive remote workspace path: %w", err)
	}
	preparedRepos := make([]PreparedRepo, 0, len(request.Repos))
	for _, repo := range request.Repos {
		repoPath := RepoPath(workspacePath, repo.WorkspaceDirname, repo.Name)
		preparedRepos = append(preparedRepos, PreparedRepo{
			Name:             repo.Name,
			RepositoryURL:    repo.RepositoryURL,
			DefaultBranch:    repo.DefaultBranch,
			BranchName:       repo.BranchName,
			WorkspaceDirname: repo.WorkspaceDirname,
			Path:             repoPath,
		})
	}

	return Workspace{
		Path:       workspacePath,
		BranchName: request.BranchName,
		Repos:      preparedRepos,
	}, nil
}

func buildPrepareWorkspaceCommand(request SetupRequest) (string, error) {
	lines := make([]string, 0, 2+8*len(request.Repos))
	workspacePath, _ := TicketWorkspacePath(
		request.WorkspaceRoot,
		request.OrganizationSlug,
		request.ProjectSlug,
		request.TicketIdentifier,
	)
	lines = append(lines,
		"set -eu",
		"mkdir -p "+sshinfra.ShellQuote(workspacePath),
	)

	for _, repo := range request.Repos {
		repoPath := RepoPath(workspacePath, repo.WorkspaceDirname, repo.Name)
		cloneCommand, err := buildRemoteGitCommand(
			repo,
			"clone",
			"--branch",
			repo.DefaultBranch,
			"--single-branch",
			repo.RepositoryURL,
			repoPath,
		)
		if err != nil {
			return "", err
		}
		fetchCommand, err := buildRemoteGitCommand(
			repo,
			"-C",
			repoPath,
			"fetch",
			"origin",
		)
		if err != nil {
			return "", err
		}
		lines = append(lines,
			"mkdir -p "+sshinfra.ShellQuote(filepath.Dir(repoPath)),
			"if [ -e "+sshinfra.ShellQuote(repoPath)+" ] && [ ! -d "+sshinfra.ShellQuote(filepath.Join(repoPath, ".git"))+" ]; then echo "+sshinfra.ShellQuote("repository path "+repoPath+" is not a git clone")+" >&2; exit 1; fi",
			"if [ ! -e "+sshinfra.ShellQuote(repoPath)+" ]; then "+cloneCommand+"; "+
				"actual_origin=$(git -C "+sshinfra.ShellQuote(repoPath)+" remote get-url origin); "+
				"if [ \"$actual_origin\" != "+sshinfra.ShellQuote(repo.RepositoryURL)+" ]; then echo "+sshinfra.ShellQuote("origin remote URL mismatch")+" >&2; exit 1; fi; "+
				fetchCommand+"; "+
				"git -C "+sshinfra.ShellQuote(repoPath)+" rev-parse --verify "+sshinfra.ShellQuote("origin/"+repo.DefaultBranch)+" >/dev/null; "+
				"if git -C "+sshinfra.ShellQuote(repoPath)+" rev-parse --verify "+sshinfra.ShellQuote("origin/"+repo.BranchName)+" >/dev/null 2>&1; then git -C "+sshinfra.ShellQuote(repoPath)+" checkout -B "+sshinfra.ShellQuote(repo.BranchName)+" "+sshinfra.ShellQuote("origin/"+repo.BranchName)+"; else git -C "+sshinfra.ShellQuote(repoPath)+" checkout -B "+sshinfra.ShellQuote(repo.BranchName)+" "+sshinfra.ShellQuote("origin/"+repo.DefaultBranch)+"; fi; "+
				"fi",
			"actual_origin=$(git -C "+sshinfra.ShellQuote(repoPath)+" remote get-url origin)",
			"if [ \"$actual_origin\" != "+sshinfra.ShellQuote(repo.RepositoryURL)+" ]; then echo "+sshinfra.ShellQuote("origin remote URL mismatch")+" >&2; exit 1; fi",
		)
	}

	return strings.Join(lines, "\n"), nil
}

func buildRemoteGitCommand(repo RepoRequest, args ...string) (string, error) {
	parts := []string{"git"}
	configKey, headerValue, ok, err := repositoryHTTPSExtraHeader(repo)
	if err != nil {
		return "", err
	}
	if ok {
		parts = append(parts, "-c", configKey+"="+headerValue, "-c", "credential.helper=")
	}
	parts = append(parts, args...)

	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, sshinfra.ShellQuote(part))
	}
	return strings.Join(quoted, " "), nil
}

func wrapPrepareTransportError(machineName string, cause error) error {
	if cause == nil {
		return nil
	}
	message := "transport unavailable"
	if trimmed := strings.TrimSpace(machineName); trimmed != "" {
		message = fmt.Sprintf("transport unavailable for machine %s", trimmed)
	}
	return &PrepareError{
		Stage:   PrepareFailureStageTransport,
		Message: message,
		Cause:   cause,
	}
}

func WrapPrepareTransportError(machineName string, cause error) error {
	return wrapPrepareTransportError(machineName, cause)
}

func classifyPrepareWorkspaceFailure(request SetupRequest, cause error, output string) error {
	trimmedOutput := strings.TrimSpace(output)
	combined := strings.ToLower(strings.TrimSpace(trimmedOutput + "\n" + errorString(cause)))
	workspaceRoot := strings.ToLower(strings.TrimSpace(request.WorkspaceRoot))

	stage := PrepareFailureStageGitOperation
	switch {
	case strings.Contains(combined, "authentication failed"),
		strings.Contains(combined, "permission denied (publickey)"),
		strings.Contains(combined, "could not read from remote repository"),
		strings.Contains(combined, "could not read username"),
		strings.Contains(combined, "repository not found"):
		stage = PrepareFailureStageRepoAuth
	case workspaceRoot != "" &&
		strings.Contains(combined, workspaceRoot) &&
		(strings.Contains(combined, "permission denied") ||
			strings.Contains(combined, "read-only file system") ||
			strings.Contains(combined, "not a directory") ||
			strings.Contains(combined, "no such file or directory")):
		stage = PrepareFailureStageWorkspaceRoot
	case strings.Contains(combined, "origin remote url mismatch"),
		strings.Contains(combined, "is not a git clone"),
		strings.Contains(combined, "fatal:"):
		stage = PrepareFailureStageGitOperation
	}

	message := trimmedOutput
	if message == "" && cause != nil {
		message = strings.TrimSpace(cause.Error())
	}
	return &PrepareError{
		Stage:   stage,
		Message: message,
		Cause:   cause,
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
