package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
)

// RemoteManager prepares ticket workspaces on a remote machine over SSH.
type RemoteManager struct {
	pool *sshinfra.Pool
}

func NewRemoteManager(pool *sshinfra.Pool) *RemoteManager {
	return &RemoteManager{pool: pool}
}

func (m *RemoteManager) Prepare(ctx context.Context, machine domain.Machine, request SetupRequest) (Workspace, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil || m.pool == nil {
		return Workspace{}, fmt.Errorf("remote workspace manager unavailable")
	}
	if machine.Host == domain.LocalMachineHost {
		return Workspace{}, fmt.Errorf("local machine does not use remote workspace preparation")
	}

	client, err := m.pool.Get(ctx, machine)
	if err != nil {
		return Workspace{}, fmt.Errorf("get ssh client for machine %s: %w", machine.Name, err)
	}

	session, err := client.NewSession()
	if err != nil {
		return Workspace{}, fmt.Errorf("open ssh session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	command := buildPrepareWorkspaceCommand(request)
	if output, err := session.CombinedOutput(command); err != nil {
		return Workspace{}, fmt.Errorf("prepare remote workspace: %w: %s", err, strings.TrimSpace(string(output)))
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
		repoPath := RepoPath(workspacePath, repo.ClonePath, repo.Name)
		preparedRepos = append(preparedRepos, PreparedRepo{
			Name:          repo.Name,
			RepositoryURL: repo.RepositoryURL,
			DefaultBranch: repo.DefaultBranch,
			BranchName:    repo.BranchName,
			ClonePath:     repo.ClonePath,
			Path:          repoPath,
		})
	}

	return Workspace{
		Path:       workspacePath,
		BranchName: request.BranchName,
		Repos:      preparedRepos,
	}, nil
}

func buildPrepareWorkspaceCommand(request SetupRequest) string {
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
		repoPath := RepoPath(workspacePath, repo.ClonePath, repo.Name)
		lines = append(lines,
			"mkdir -p "+sshinfra.ShellQuote(filepath.Dir(repoPath)),
			"if [ -e "+sshinfra.ShellQuote(repoPath)+" ] && [ ! -d "+sshinfra.ShellQuote(filepath.Join(repoPath, ".git"))+" ]; then echo "+sshinfra.ShellQuote("repository path "+repoPath+" is not a git clone")+" >&2; exit 1; fi",
			"if [ ! -e "+sshinfra.ShellQuote(repoPath)+" ]; then git clone --branch "+sshinfra.ShellQuote(repo.DefaultBranch)+" --single-branch "+sshinfra.ShellQuote(repo.RepositoryURL)+" "+sshinfra.ShellQuote(repoPath)+"; fi",
			"actual_origin=$(git -C "+sshinfra.ShellQuote(repoPath)+" remote get-url origin)",
			"if [ \"$actual_origin\" != "+sshinfra.ShellQuote(repo.RepositoryURL)+" ]; then echo "+sshinfra.ShellQuote("origin remote URL mismatch")+" >&2; exit 1; fi",
			"git -C "+sshinfra.ShellQuote(repoPath)+" fetch origin",
			"git -C "+sshinfra.ShellQuote(repoPath)+" rev-parse --verify "+sshinfra.ShellQuote("origin/"+repo.DefaultBranch)+" >/dev/null",
			"git -C "+sshinfra.ShellQuote(repoPath)+" checkout -B "+sshinfra.ShellQuote(repo.BranchName)+" "+sshinfra.ShellQuote("origin/"+repo.DefaultBranch),
		)
	}

	return strings.Join(lines, "\n")
}
