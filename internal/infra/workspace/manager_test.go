package workspace

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func TestParseSetupRequestUsesScopedBranchNameWhenProvided(t *testing.T) {
	rawBranch := "feature/custom"
	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{
			{
				Name:          "backend",
				RepositoryURL: "/tmp/backend.git",
				BranchName:    &rawBranch,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected parse to accept scoped branch name: %v", err)
	}
	if request.Repos[0].BranchName != rawBranch {
		t.Fatalf("expected scoped branch %q, got %+v", rawBranch, request.Repos)
	}
}

func TestParseSetupRequestAllowsEmptyRepos(t *testing.T) {
	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
	})
	if err != nil {
		t.Fatalf("expected parse to allow empty repos: %v", err)
	}
	if len(request.Repos) != 0 {
		t.Fatalf("expected no repos, got %+v", request.Repos)
	}
	if request.BranchName != "agent/ASE-33" {
		t.Fatalf("unexpected branch name %q", request.BranchName)
	}
}

func TestManagerPrepareCreatesJointWorkspaceWithFeatureBranch(t *testing.T) {
	backendRepoPath, _ := createRemoteRepo(t, "main", map[string]string{
		"README.md": "backend",
	})
	frontendRepoPath, _ := createRemoteRepo(t, "main", map[string]string{
		"package.json": "{}",
	})

	clonePath := "services/frontend"
	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{
			{
				Name:          "backend",
				RepositoryURL: backendRepoPath,
				DefaultBranch: "main",
			},
			{
				Name:             "frontend",
				RepositoryURL:    frontendRepoPath,
				DefaultBranch:    "main",
				WorkspaceDirname: &clonePath,
			},
		},
	})
	if err != nil {
		t.Fatalf("parse setup request: %v", err)
	}

	manager := NewManager()
	workspace, err := manager.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}

	expectedWorkspacePath := filepath.Join(request.WorkspaceRoot, "acme", "payments", "ASE-33")
	if workspace.Path != expectedWorkspacePath {
		t.Fatalf("expected workspace path %s, got %s", expectedWorkspacePath, workspace.Path)
	}
	if workspace.BranchName != "agent/ASE-33" {
		t.Fatalf("expected branch name agent/ASE-33, got %s", workspace.BranchName)
	}

	backendClonePath := filepath.Join(workspace.Path, "backend")
	frontendClonePath := filepath.Join(workspace.Path, filepath.FromSlash(clonePath))

	assertFileExists(t, filepath.Join(backendClonePath, "README.md"))
	assertFileExists(t, filepath.Join(frontendClonePath, "package.json"))
	assertHeadBranch(t, backendClonePath, "agent/ASE-33")
	assertHeadBranch(t, frontendClonePath, "agent/ASE-33")
}

func TestManagerPreparePreservesExistingCloneState(t *testing.T) {
	repositoryURL, initialHash := createRemoteRepo(t, "main", map[string]string{
		"README.md": "first",
	})

	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{
			{
				Name:          "backend",
				RepositoryURL: repositoryURL,
				DefaultBranch: "main",
			},
		},
	})
	if err != nil {
		t.Fatalf("parse setup request: %v", err)
	}

	manager := NewManager()
	workspace, err := manager.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace first run: %v", err)
	}

	backendClonePath := filepath.Join(workspace.Path, "backend")
	assertRemoteBranchHash(t, backendClonePath, "main", initialHash)

	updatedHash := appendCommit(t, repositoryURL, "main", "README.md", "second")
	repository, err := git.PlainOpen(backendClonePath)
	if err != nil {
		t.Fatalf("open prepared repository: %v", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("load prepared worktree: %v", err)
	}
	featureRef := plumbing.NewBranchReferenceName("scratch")
	head, err := repository.Head()
	if err != nil {
		t.Fatalf("load prepared head: %v", err)
	}
	if err := repository.Storer.SetReference(plumbing.NewHashReference(featureRef, head.Hash())); err != nil {
		t.Fatalf("create scratch branch: %v", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: featureRef}); err != nil {
		t.Fatalf("checkout scratch branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendClonePath, "DIRTY.txt"), []byte("dirty"), 0o600); err != nil {
		t.Fatalf("write dirty marker: %v", err)
	}

	workspace, err = manager.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace second run: %v", err)
	}

	assertHeadBranch(t, backendClonePath, "scratch")
	assertRemoteBranchHash(t, backendClonePath, "main", initialHash)
	assertFileExists(t, filepath.Join(backendClonePath, "DIRTY.txt"))
	if workspace.Path == "" || updatedHash == initialHash {
		t.Fatalf("expected preserved workspace path and divergent remote update, got path=%q", workspace.Path)
	}
}

func TestManagerPrepareClonesFileRepositoryURL(t *testing.T) {
	repositoryPath, initialHash := createRemoteRepo(t, "main", map[string]string{
		"README.md": "file transport",
	})
	repositoryURL := (&url.URL{Scheme: "file", Path: repositoryPath}).String()

	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{{
			Name:          "backend",
			RepositoryURL: repositoryURL,
			DefaultBranch: "main",
		}},
	})
	if err != nil {
		t.Fatalf("parse setup request: %v", err)
	}

	workspace, err := NewManager().Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	if len(workspace.Repos) != 1 {
		t.Fatalf("expected one prepared repo, got %+v", workspace.Repos)
	}
	if workspace.Repos[0].RepositoryURL != repositoryURL {
		t.Fatalf("prepared repo repository_url = %q, want %q", workspace.Repos[0].RepositoryURL, repositoryURL)
	}

	backendClonePath := filepath.Join(workspace.Path, "backend")
	assertHeadBranch(t, backendClonePath, "agent/ASE-33")
	assertRemoteBranchHash(t, backendClonePath, "main", initialHash)
	assertOriginURL(t, backendClonePath, repositoryURL)
}

func TestManagerPrepareTracksExistingRemoteWorkBranch(t *testing.T) {
	repositoryURL, _ := createRemoteRepo(t, "main", map[string]string{
		"README.md": "base",
	})
	workBranchHash := appendCommit(t, repositoryURL, "main", "README.md", "remote work branch")

	repository, err := git.PlainOpen(repositoryURL)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	if err := repository.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewBranchReferenceName("release/ASE-33"), workBranchHash),
	); err != nil {
		t.Fatalf("create remote work branch: %v", err)
	}

	workBranch := "release/ASE-33"
	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{
			{
				Name:          "backend",
				RepositoryURL: repositoryURL,
				DefaultBranch: "main",
				BranchName:    &workBranch,
			},
		},
	})
	if err != nil {
		t.Fatalf("parse setup request: %v", err)
	}

	workspace, err := NewManager().Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}

	backendClonePath := filepath.Join(workspace.Path, "backend")
	assertHeadBranch(t, backendClonePath, "release/ASE-33")
	assertHeadHash(t, backendClonePath, workBranchHash)
}

func TestManagerPrepareLogsRepoPreparePhases(t *testing.T) {
	repositoryURL, _ := createRemoteRepo(t, "main", map[string]string{
		"README.md": "backend",
	})

	var logBuffer bytes.Buffer
	manager := NewManagerWithLogger(slog.New(slog.NewTextHandler(&logBuffer, nil)))
	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-146",
		Observability: PrepareObservability{
			MachineID: "machine-1",
			RunID:     "run-1",
			TicketID:  "ticket-1",
		},
		Repos: []RepoInput{{
			Name:          "backend",
			RepositoryURL: repositoryURL,
			DefaultBranch: "main",
			HTTPBasicAuth: &HTTPBasicAuthInput{
				Username: "x-access-token",
				Password: "ghu_test",
			},
		}},
	})
	if err != nil {
		t.Fatalf("ParseSetupRequest() error = %v", err)
	}

	workspaceItem, err := manager.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	logOutput := logBuffer.String()
	repoPath := filepath.Join(workspaceItem.Path, "backend")
	for _, needle := range []string{
		"machine_id=machine-1",
		"run_id=run-1",
		"ticket_id=ticket-1",
		"repo_name=backend",
		"repo_path=" + repoPath,
		"phase=repo_prepare_begin",
		"phase=clone_or_open",
		"phase_result=clone",
		"phase=auth_inject",
		"phase=fetch",
		"phase=checkout_reset",
		"phase=repo_prepare_done",
	} {
		if !strings.Contains(logOutput, needle) {
			t.Fatalf("expected log output to contain %q, got %q", needle, logOutput)
		}
	}
}

func TestBuildCloneOptionsUsesConfiguredSSHKeyForSSHURL(t *testing.T) {
	keyPath := writeTestPrivateKey(t)
	t.Setenv("OPENASE_GIT_SSH_KEY_PATH", keyPath)

	options, err := buildCloneOptions(RepoRequest{RepositoryURL: "git@github.com:acme/private-repo.git"})
	if err != nil {
		t.Fatalf("buildCloneOptions() error = %v", err)
	}
	if options.Auth == nil {
		t.Fatal("buildCloneOptions() expected SSH auth")
	}

	auth, ok := options.Auth.(*gitssh.PublicKeys)
	if !ok {
		t.Fatalf("buildCloneOptions() auth type = %T, want *ssh.PublicKeys", options.Auth)
	}
	if auth.User != "git" {
		t.Fatalf("buildCloneOptions() auth user = %q, want git", auth.User)
	}
}

func TestBuildCloneOptionsLeavesHTTPSAuthEmpty(t *testing.T) {
	options, err := buildCloneOptions(RepoRequest{RepositoryURL: "https://github.com/acme/public-repo.git"})
	if err != nil {
		t.Fatalf("buildCloneOptions() error = %v", err)
	}
	if options.Auth != nil {
		t.Fatalf("buildCloneOptions() auth = %T, want nil for HTTPS", options.Auth)
	}
}

func TestBuildCloneOptionsUsesHTTPBasicAuthForHTTPSURL(t *testing.T) {
	options, err := buildCloneOptions(RepoRequest{
		RepositoryURL: "https://github.com/acme/private-repo.git",
		HTTPBasicAuth: &HTTPBasicAuthRequest{
			Username: "x-access-token",
			Password: "ghu_test",
		},
	})
	if err != nil {
		t.Fatalf("buildCloneOptions() error = %v", err)
	}
	auth, ok := options.Auth.(*githttp.BasicAuth)
	if !ok {
		t.Fatalf("buildCloneOptions() auth type = %T, want *http.BasicAuth", options.Auth)
	}
	if auth.Username != "x-access-token" || auth.Password != "ghu_test" {
		t.Fatalf("buildCloneOptions() auth = %+v", auth)
	}
}

func TestTicketWorkspacePathAndPattern(t *testing.T) {
	workspacePath, err := TicketWorkspacePath("/srv/openase/workspace", "acme", "payments", "ASE-42")
	if err != nil {
		t.Fatalf("derive workspace path: %v", err)
	}
	if workspacePath != filepath.Join("/srv/openase/workspace", "acme", "payments", "ASE-42") {
		t.Fatalf("unexpected workspace path %q", workspacePath)
	}

	pattern, err := TicketWorkspacePattern(LocalWorkspacePatternRoot, "acme", "payments")
	if err != nil {
		t.Fatalf("derive workspace pattern: %v", err)
	}
	if pattern != filepath.Join(LocalWorkspacePatternRoot, "acme", "payments", ticketPlaceholder) {
		t.Fatalf("unexpected workspace pattern %q", pattern)
	}
}

func TestWorkspaceLayoutAndParserHelpers(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	root, err := LocalWorkspaceRoot()
	if err != nil {
		t.Fatalf("LocalWorkspaceRoot() error = %v", err)
	}
	if root != filepath.Join(homeDir, ".openase", "workspace") {
		t.Fatalf("LocalWorkspaceRoot() = %q", root)
	}

	projectsRoot, err := LocalProjectStateRoot()
	if err != nil {
		t.Fatalf("LocalProjectStateRoot() error = %v", err)
	}
	if projectsRoot != filepath.Join(homeDir, ".openase", "projects") {
		t.Fatalf("LocalProjectStateRoot() = %q", projectsRoot)
	}

	projectStatePath, err := ProjectStatePath(projectsRoot, "40c56726-5a02-4c1b-9cac-8e6352f340c6")
	if err != nil {
		t.Fatalf("ProjectStatePath() error = %v", err)
	}
	if projectStatePath != filepath.Join(projectsRoot, "40c56726-5a02-4c1b-9cac-8e6352f340c6") {
		t.Fatalf("ProjectStatePath() = %q", projectStatePath)
	}

	projectChatPath, err := ProjectChatPath(projectsRoot, "40c56726-5a02-4c1b-9cac-8e6352f340c6")
	if err != nil {
		t.Fatalf("ProjectChatPath() error = %v", err)
	}
	if projectChatPath != filepath.Join(projectsRoot, "40c56726-5a02-4c1b-9cac-8e6352f340c6", "chat") {
		t.Fatalf("ProjectChatPath() = %q", projectChatPath)
	}

	projectHooksPath, err := ProjectHooksPath(projectsRoot, "40c56726-5a02-4c1b-9cac-8e6352f340c6")
	if err != nil {
		t.Fatalf("ProjectHooksPath() error = %v", err)
	}
	if projectHooksPath != filepath.Join(projectsRoot, "40c56726-5a02-4c1b-9cac-8e6352f340c6", "hooks") {
		t.Fatalf("ProjectHooksPath() = %q", projectHooksPath)
	}

	if got := RepoPath("/srv/workspace/ASE-1", "", " backend "); got != filepath.Join("/srv/workspace/ASE-1", "backend") {
		t.Fatalf("RepoPath(default clone path) = %q", got)
	}
	if got := RepoPath("/srv/workspace/ASE-1", " services/api ", "backend"); got != filepath.Join("/srv/workspace/ASE-1", "services", "api") {
		t.Fatalf("RepoPath(custom clone path) = %q", got)
	}
	if got := RepoPath(" /srv/workspace/ASE-1 ", " ", " "); got != "/srv/workspace/ASE-1" {
		t.Fatalf("RepoPath(empty repo name) = %q", got)
	}

	if _, err := parseTicketWorkspaceRoot(" ", true); err == nil || !strings.Contains(err.Error(), "workspace_root must not be empty") {
		t.Fatalf("parseTicketWorkspaceRoot(blank) error = %v", err)
	}
	if _, err := parseTicketWorkspaceRoot("relative/path", true); err == nil || !strings.Contains(err.Error(), "workspace_root must be absolute") {
		t.Fatalf("parseTicketWorkspaceRoot(relative) error = %v", err)
	}
	if got, err := parseTicketWorkspaceRoot(" /srv/workspace/../workspace ", false); err != nil || got != "/srv/workspace" {
		t.Fatalf("parseTicketWorkspaceRoot(clean) = %q, %v", got, err)
	}

	if got, err := parseTicketSegment("ticket_identifier", ticketPlaceholder); err != nil || got != ticketPlaceholder {
		t.Fatalf("parseTicketSegment(placeholder) = %q, %v", got, err)
	}
	if _, err := parseAbsolutePath("repos[0].mirror_path", "relative/repo"); err == nil || !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("parseAbsolutePath(relative) error = %v", err)
	}
	if _, err := parseWorkspaceDirname("repos[0].workspace_dirname", "/abs"); err == nil || !strings.Contains(err.Error(), "must be relative") {
		t.Fatalf("parseWorkspaceDirname(abs) error = %v", err)
	}
	if _, err := parseWorkspaceDirname("repos[0].workspace_dirname", "../escape"); err == nil || !strings.Contains(err.Error(), "must stay inside the workspace") {
		t.Fatalf("parseWorkspaceDirname(parent) error = %v", err)
	}
	if _, err := parseWorkspaceDirname("repos[0].workspace_dirname", "bad path"); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("parseWorkspaceDirname(pattern) error = %v", err)
	}
	if got, err := parseWorkspaceDirname("repos[0].workspace_dirname", "./services/api"); err != nil || got != "services/api" {
		t.Fatalf("parseWorkspaceDirname(clean) = %q, %v", got, err)
	}

	if _, err := parseRepoInput(0, RepoInput{Name: "backend", RepositoryURL: "/repo.git", DefaultBranch: "feature/x"}, "agent/ASE-33"); err == nil || !strings.Contains(err.Error(), "default_branch must not contain '/'") {
		t.Fatalf("parseRepoInput(default branch) error = %v", err)
	}
	branch := "feature/x"
	request, err := parseRepoInput(0, RepoInput{Name: "backend", RepositoryURL: "/repo.git", BranchName: &branch}, "agent/ASE-33")
	if err != nil {
		t.Fatalf("parseRepoInput(branch name) error = %v", err)
	}
	if request.BranchName != branch {
		t.Fatalf("parseRepoInput(branch name) = %+v", request)
	}

	notDirPath := filepath.Join(t.TempDir(), "repo")
	if err := os.WriteFile(notDirPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(notDirPath) error = %v", err)
	}
	if _, err := cloneOrOpenRepository(context.Background(), notDirPath, RepoRequest{RepositoryURL: "/tmp/example.invalid/repo.git"}); err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("cloneOrOpenRepository(file path) error = %v", err)
	}
}

func createRemoteRepo(t *testing.T, defaultBranch string, files map[string]string) (string, plumbing.Hash) {
	t.Helper()

	repoPath := t.TempDir()
	repository, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("init repository: %v", err)
	}

	hash := commitFiles(t, repository, repoPath, files, "initial commit")
	setDefaultBranch(t, repository, defaultBranch, hash)

	return repoPath, hash
}

func appendCommit(t *testing.T, repoPath string, branch string, filePath string, content string) plumbing.Hash {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("open worktree: %v", err)
	}

	branchRef := plumbing.NewBranchReferenceName(branch)
	head, err := repository.Head()
	if err != nil {
		t.Fatalf("load head: %v", err)
	}
	if head.Name() != branchRef {
		if err := worktree.Checkout(&git.CheckoutOptions{Branch: branchRef}); err != nil {
			t.Fatalf("checkout branch %s: %v", branch, err)
		}
	}

	absoluteFilePath := filepath.Join(repoPath, filePath)
	if err := os.MkdirAll(filepath.Dir(absoluteFilePath), 0o750); err != nil {
		t.Fatalf("create directories for %s: %v", filePath, err)
	}
	if err := os.WriteFile(absoluteFilePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", filePath, err)
	}

	if _, err := worktree.Add(filePath); err != nil {
		t.Fatalf("add file %s: %v", filePath, err)
	}

	hash, err := worktree.Commit("update commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@example.com",
			When:  time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("commit update: %v", err)
	}

	return hash
}

func writeTestPrivateKey(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}

	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("WriteFile(private key) error = %v", err)
	}
	return keyPath
}

func commitFiles(t *testing.T, repository *git.Repository, repoPath string, files map[string]string, message string) plumbing.Hash {
	t.Helper()

	for relativePath, content := range files {
		absolutePath := filepath.Join(repoPath, relativePath)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
			t.Fatalf("create directory for %s: %v", relativePath, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write file %s: %v", relativePath, err)
		}
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("open worktree: %v", err)
	}
	if err := worktree.AddGlob("."); err != nil {
		t.Fatalf("add files: %v", err)
	}

	hash, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@example.com",
			When:  time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("commit files: %v", err)
	}

	return hash
}

func setDefaultBranch(t *testing.T, repository *git.Repository, branch string, hash plumbing.Hash) {
	t.Helper()

	branchRef := plumbing.NewBranchReferenceName(branch)
	if err := repository.Storer.SetReference(plumbing.NewHashReference(branchRef, hash)); err != nil {
		t.Fatalf("set branch %s: %v", branch, err)
	}
	if err := repository.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, branchRef)); err != nil {
		t.Fatalf("set HEAD to %s: %v", branch, err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("open worktree: %v", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: branchRef, Force: true}); err != nil {
		t.Fatalf("checkout branch %s: %v", branch, err)
	}
}

func assertFileExists(t *testing.T, filePath string) {
	t.Helper()

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected %s to exist: %v", filePath, err)
	}
}

func assertHeadBranch(t *testing.T, repoPath string, expectedBranch string) {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository %s: %v", repoPath, err)
	}

	head, err := repository.Head()
	if err != nil {
		t.Fatalf("load head for %s: %v", repoPath, err)
	}
	if head.Name().Short() != expectedBranch {
		t.Fatalf("expected branch %s, got %s", expectedBranch, head.Name().Short())
	}
}

func assertHeadHash(t *testing.T, repoPath string, expectedHash plumbing.Hash) {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository %s: %v", repoPath, err)
	}

	head, err := repository.Head()
	if err != nil {
		t.Fatalf("load head for %s: %v", repoPath, err)
	}
	if head.Hash() != expectedHash {
		t.Fatalf("expected head hash %s, got %s", expectedHash, head.Hash())
	}
}

func assertRemoteBranchHash(t *testing.T, repoPath string, branch string, expectedHash plumbing.Hash) {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository %s: %v", repoPath, err)
	}

	ref, err := repository.Reference(plumbing.NewRemoteReferenceName("origin", branch), true)
	if err != nil {
		t.Fatalf("load remote branch %s: %v", branch, err)
	}
	if ref.Hash() != expectedHash {
		t.Fatalf("expected remote branch hash %s, got %s", expectedHash, ref.Hash())
	}
}

func assertOriginURL(t *testing.T, repoPath string, expectedURL string) {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository %s: %v", repoPath, err)
	}

	remote, err := repository.Remote("origin")
	if err != nil {
		t.Fatalf("load origin remote for %s: %v", repoPath, err)
	}
	if len(remote.Config().URLs) == 0 {
		t.Fatalf("origin remote for %s has no URLs", repoPath)
	}
	if got := strings.TrimSpace(remote.Config().URLs[0]); got != expectedURL {
		t.Fatalf("expected origin URL %q, got %q", expectedURL, got)
	}
}
