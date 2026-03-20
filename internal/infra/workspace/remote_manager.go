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

	workspacePath := filepath.Join(request.WorkspaceRoot, request.TicketIdentifier)
	preparedRepos := make([]PreparedRepo, 0, len(request.Repos))
	for _, repo := range request.Repos {
		repoPath := filepath.Join(workspacePath, filepath.FromSlash(repo.ClonePath))
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
	lines := []string{
		"set -eu",
		"mkdir -p " + sshinfraShellQuote(filepath.Join(request.WorkspaceRoot, request.TicketIdentifier)),
	}

	workspacePath := filepath.Join(request.WorkspaceRoot, request.TicketIdentifier)
	for _, repo := range request.Repos {
		repoPath := filepath.Join(workspacePath, filepath.FromSlash(repo.ClonePath))
		lines = append(lines,
			"mkdir -p "+sshinfraShellQuote(filepath.Dir(repoPath)),
			"if [ -e "+sshinfraShellQuote(repoPath)+" ] && [ ! -d "+sshinfraShellQuote(filepath.Join(repoPath, ".git"))+" ]; then echo "+sshinfraShellQuote("repository path "+repoPath+" is not a git clone")+" >&2; exit 1; fi",
			"if [ ! -e "+sshinfraShellQuote(repoPath)+" ]; then git clone --branch "+sshinfraShellQuote(repo.DefaultBranch)+" --single-branch "+sshinfraShellQuote(repo.RepositoryURL)+" "+sshinfraShellQuote(repoPath)+"; fi",
			"actual_origin=$(git -C "+sshinfraShellQuote(repoPath)+" remote get-url origin)",
			"if [ \"$actual_origin\" != "+sshinfraShellQuote(repo.RepositoryURL)+" ]; then echo "+sshinfraShellQuote("origin remote URL mismatch")+" >&2; exit 1; fi",
			"git -C "+sshinfraShellQuote(repoPath)+" fetch origin",
			"git -C "+sshinfraShellQuote(repoPath)+" rev-parse --verify "+sshinfraShellQuote("origin/"+repo.DefaultBranch)+" >/dev/null",
			"git -C "+sshinfraShellQuote(repoPath)+" checkout -B "+sshinfraShellQuote(repo.BranchName)+" "+sshinfraShellQuote("origin/"+repo.DefaultBranch),
		)
	}

	return strings.Join(lines, "\n")
}

func sshinfraShellQuote(raw string) string {
	if raw == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(raw, "'", `'"'"'`) + "'"
}
