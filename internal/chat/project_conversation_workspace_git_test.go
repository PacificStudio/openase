package chat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestParseWorkspaceBranchName(t *testing.T) {
	t.Parallel()

	parsed, err := ParseWorkspaceBranchName(" origin/feature/workspace-git ")
	if err != nil {
		t.Fatalf("ParseWorkspaceBranchName() error = %v", err)
	}
	if parsed.String() != "origin/feature/workspace-git" {
		t.Fatalf("branch = %q", parsed)
	}
}

func TestParseWorkspaceBranchNameRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	_, err := ParseWorkspaceBranchName(" ../main ")
	if err == nil {
		t.Fatal("expected invalid branch name error")
	}
}

func TestProjectConversationWorkspaceRepoRefsDetectDetachedHEAD(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "hello\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	commitID := strings.TrimSpace(runConversationGitCommand(t, "", "git", "-C", repoPath, "rev-parse", "HEAD"))
	runConversationGitCommand(t, "", "git", "-C", repoPath, "checkout", commitID)

	refs, err := fixture.service.GetWorkspaceRepoRefs(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		WorkspaceRepoPath("backend"),
	)
	if err != nil {
		t.Fatalf("GetWorkspaceRepoRefs() error = %v", err)
	}
	if refs.CurrentRef.Kind != ProjectConversationWorkspaceCurrentRefKindDetached {
		t.Fatalf("current ref kind = %q", refs.CurrentRef.Kind)
	}
	if refs.CurrentRef.CommitID != commitID {
		t.Fatalf("current ref commit = %q, want %q", refs.CurrentRef.CommitID, commitID)
	}
	if !strings.Contains(refs.CurrentRef.DisplayName, refs.CurrentRef.ShortCommitID) {
		t.Fatalf("detached display name = %q", refs.CurrentRef.DisplayName)
	}
}

func TestProjectConversationWorkspaceGitGraphListsLabelsAndHead(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	featureCommit := createConversationBranchCommit(
		t,
		repoPath,
		"feature/workspace-git",
		map[string]string{"feature.txt": "feature\n"},
	)
	runConversationGitCommand(t, "", "git", "-C", repoPath, "checkout", "main")

	graph, err := fixture.service.GetWorkspaceGitGraph(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		WorkspaceRepoPath("backend"),
		WorkspaceGitGraphWindow{Limit: 10},
	)
	if err != nil {
		t.Fatalf("GetWorkspaceGitGraph() error = %v", err)
	}
	if len(graph.Commits) < 2 {
		t.Fatalf("expected at least two commits, got %+v", graph.Commits)
	}

	byCommit := make(map[string]ProjectConversationWorkspaceGitGraphCommit, len(graph.Commits))
	for _, commit := range graph.Commits {
		byCommit[commit.CommitID] = commit
	}
	featureNode, ok := byCommit[featureCommit.String()]
	if !ok {
		t.Fatalf("expected feature commit %s in graph", featureCommit)
	}
	if !projectConversationWorkspaceLabelExists(featureNode.Labels, "feature/workspace-git") {
		t.Fatalf("expected feature branch label on %+v", featureNode.Labels)
	}

	headCount := 0
	for _, commit := range graph.Commits {
		if !commit.Head {
			continue
		}
		headCount++
		if !projectConversationWorkspaceLabelExists(commit.Labels, "HEAD") {
			t.Fatalf("head commit labels = %+v", commit.Labels)
		}
		if !projectConversationWorkspaceLabelExists(commit.Labels, "main") {
			t.Fatalf("expected main label on head commit %+v", commit.Labels)
		}
	}
	if headCount != 1 {
		t.Fatalf("expected exactly one head commit, got %d", headCount)
	}
}

func TestProjectConversationWorkspaceCheckoutRejectsDirtyWorkspace(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	createConversationBranchCommit(
		t,
		repoPath,
		"feature/dirty-checkout",
		map[string]string{"feature.txt": "feature\n"},
	)
	runConversationGitCommand(t, "", "git", "-C", repoPath, "checkout", "main")
	writeConversationWorkspaceFile(t, filepath.Join(repoPath, "README.md"), "dirty\n")

	_, err := fixture.service.CheckoutWorkspaceBranch(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceCheckoutInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Target: WorkspaceCheckoutTarget{
				Kind:       WorkspaceCheckoutTargetKindLocalBranch,
				BranchName: WorkspaceBranchName("feature/dirty-checkout"),
			},
			ExpectedCleanWorkspace: true,
		},
	)
	var preconditionErr *ProjectConversationWorkspaceCheckoutPreconditionError
	if !errors.As(err, &preconditionErr) {
		t.Fatalf("CheckoutWorkspaceBranch() error = %v, want precondition error", err)
	}
	if preconditionErr.Reason != ProjectConversationWorkspaceCheckoutPreconditionDirtyWorkspace {
		t.Fatalf("precondition reason = %q", preconditionErr.Reason)
	}
	currentBranch := strings.TrimSpace(runConversationGitCommand(t, "", "git", "-C", repoPath, "branch", "--show-current"))
	if currentBranch != "main" {
		t.Fatalf("current branch = %q, want main", currentBranch)
	}
}

func TestProjectConversationWorkspaceCheckoutCreatesTrackingBranchAndRefreshesMetadata(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})

	remoteRepoPath := fixture.catalog.projectRepos[0].RepositoryURL
	createConversationBranchCommit(
		t,
		remoteRepoPath,
		"feature/remote-switch",
		map[string]string{"remote.txt": "remote branch\n"},
	)
	runConversationGitCommand(t, "", "git", "-C", fixture.repoPaths["backend"], "fetch", "origin")

	result, err := fixture.service.CheckoutWorkspaceBranch(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceCheckoutInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Target: WorkspaceCheckoutTarget{
				Kind:                 WorkspaceCheckoutTargetKindRemoteTrackingBranch,
				BranchName:           WorkspaceBranchName("origin/feature/remote-switch"),
				CreateTrackingBranch: true,
			},
			ExpectedCleanWorkspace: true,
		},
	)
	if err != nil {
		t.Fatalf("CheckoutWorkspaceBranch() error = %v", err)
	}
	if result.CreatedLocalBranch != "feature/remote-switch" {
		t.Fatalf("created local branch = %q", result.CreatedLocalBranch)
	}
	if result.CurrentRef.BranchName != "feature/remote-switch" {
		t.Fatalf("current branch = %q", result.CurrentRef.BranchName)
	}

	metadata, err := fixture.service.GetWorkspaceMetadata(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceMetadata() error = %v", err)
	}
	if metadata.Repos[0].Branch != "feature/remote-switch" ||
		metadata.Repos[0].CurrentRef.BranchName != "feature/remote-switch" {
		t.Fatalf("workspace metadata repo = %+v", metadata.Repos[0])
	}

	diff, err := fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() error = %v", err)
	}
	if diff.Dirty {
		t.Fatalf("expected clean diff after checkout, got %+v", diff)
	}

	refs, err := fixture.service.GetWorkspaceRepoRefs(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		WorkspaceRepoPath("backend"),
	)
	if err != nil {
		t.Fatalf("GetWorkspaceRepoRefs() error = %v", err)
	}
	if !projectConversationWorkspaceBranchRefExistsByName(refs.LocalBranches, "feature/remote-switch") {
		t.Fatalf("expected local tracking branch in refs %+v", refs.LocalBranches)
	}
}

func TestProjectConversationWorkspaceCheckoutCreatesNewLocalBranch(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]

	result, err := fixture.service.CheckoutWorkspaceBranch(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceCheckoutInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Target: WorkspaceCheckoutTarget{
				Kind:       WorkspaceCheckoutTargetKindNewLocalBranch,
				BranchName: WorkspaceBranchName("feature/new-local-branch"),
			},
			ExpectedCleanWorkspace: true,
		},
	)
	if err != nil {
		t.Fatalf("CheckoutWorkspaceBranch() error = %v", err)
	}
	if result.CreatedLocalBranch != "feature/new-local-branch" {
		t.Fatalf("created local branch = %q", result.CreatedLocalBranch)
	}
	if result.CurrentRef.BranchName != "feature/new-local-branch" {
		t.Fatalf("current branch = %q", result.CurrentRef.BranchName)
	}

	currentBranch := strings.TrimSpace(runConversationGitCommand(t, "", "git", "-C", repoPath, "branch", "--show-current"))
	if currentBranch != "feature/new-local-branch" {
		t.Fatalf("current branch = %q, want feature/new-local-branch", currentBranch)
	}

	refs, err := fixture.service.GetWorkspaceRepoRefs(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		WorkspaceRepoPath("backend"),
	)
	if err != nil {
		t.Fatalf("GetWorkspaceRepoRefs() error = %v", err)
	}
	if !projectConversationWorkspaceBranchRefExistsByName(refs.LocalBranches, "feature/new-local-branch") {
		t.Fatalf("expected created local branch in refs %+v", refs.LocalBranches)
	}
}

func TestProjectConversationWorkspaceCheckoutRemoteBranchReportsExistingLocalTrackingBranch(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	createConversationBranchCommit(
		t,
		repoPath,
		"feature/existing-local-main",
		map[string]string{"feature.txt": "feature\n"},
	)

	_, err := fixture.service.CheckoutWorkspaceBranch(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceCheckoutInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Target: WorkspaceCheckoutTarget{
				Kind:                 WorkspaceCheckoutTargetKindRemoteTrackingBranch,
				BranchName:           WorkspaceBranchName("origin/main"),
				CreateTrackingBranch: true,
			},
			ExpectedCleanWorkspace: true,
		},
	)
	var preconditionErr *ProjectConversationWorkspaceCheckoutPreconditionError
	if !errors.As(err, &preconditionErr) {
		t.Fatalf("CheckoutWorkspaceBranch() error = %v, want precondition error", err)
	}
	if preconditionErr.Reason != ProjectConversationWorkspaceCheckoutPreconditionLocalBranchExists {
		t.Fatalf("precondition reason = %q", preconditionErr.Reason)
	}
	if preconditionErr.RequestedBranch != "origin/main" {
		t.Fatalf("requested branch = %q, want origin/main", preconditionErr.RequestedBranch)
	}
	if preconditionErr.SuggestedBranch != "main" {
		t.Fatalf("suggested branch = %q, want main", preconditionErr.SuggestedBranch)
	}
	currentBranch := strings.TrimSpace(runConversationGitCommand(t, "", "git", "-C", repoPath, "branch", "--show-current"))
	if currentBranch != "feature/existing-local-main" {
		t.Fatalf("current branch = %q, want feature/existing-local-main", currentBranch)
	}
}

func TestProjectConversationWorkspaceCheckoutRejectsExistingNewLocalBranch(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})

	_, err := fixture.service.CheckoutWorkspaceBranch(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceCheckoutInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Target: WorkspaceCheckoutTarget{
				Kind:       WorkspaceCheckoutTargetKindNewLocalBranch,
				BranchName: WorkspaceBranchName("main"),
			},
			ExpectedCleanWorkspace: true,
		},
	)
	var preconditionErr *ProjectConversationWorkspaceCheckoutPreconditionError
	if !errors.As(err, &preconditionErr) {
		t.Fatalf("CheckoutWorkspaceBranch() error = %v, want precondition error", err)
	}
	if preconditionErr.Reason != ProjectConversationWorkspaceCheckoutPreconditionLocalBranchExists {
		t.Fatalf("precondition reason = %q", preconditionErr.Reason)
	}
	if preconditionErr.RequestedBranch != "main" {
		t.Fatalf("requested branch = %q, want main", preconditionErr.RequestedBranch)
	}
	if preconditionErr.SuggestedBranch != "main" {
		t.Fatalf("suggested branch = %q, want main", preconditionErr.SuggestedBranch)
	}
}

func TestProjectConversationWorkspaceStageAndCommitFile(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	writeConversationWorkspaceFile(t, filepath.Join(repoPath, "README.md"), "updated\n")

	diff, err := fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() before stage error = %v", err)
	}
	if got := diff.Repos[0].Files[0]; got.Staged || !got.Unstaged {
		t.Fatalf("expected unstaged diff before add, got %+v", got)
	}

	if _, err := fixture.service.StageWorkspaceFile(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceStageFileInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Path:     "README.md",
		},
	); err != nil {
		t.Fatalf("StageWorkspaceFile() error = %v", err)
	}

	diff, err = fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() after stage error = %v", err)
	}
	if got := diff.Repos[0].Files[0]; !got.Staged || got.Unstaged {
		t.Fatalf("expected staged-only diff after add, got %+v", got)
	}

	result, err := fixture.service.CommitWorkspace(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceCommitInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Message:  "feat: commit staged workspace file",
		},
	)
	if err != nil {
		t.Fatalf("CommitWorkspace() error = %v", err)
	}
	if !strings.Contains(result.Output, "feat: commit staged workspace file") {
		t.Fatalf("commit output = %q", result.Output)
	}

	diff, err = fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() after commit error = %v", err)
	}
	if diff.Dirty {
		t.Fatalf("expected clean diff after commit, got %+v", diff)
	}
}

func TestProjectConversationWorkspaceDiscardFileRestoresWorkspaceState(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	writeConversationWorkspaceFile(t, filepath.Join(repoPath, "README.md"), "updated\n")
	if _, err := fixture.service.StageWorkspaceFile(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceStageFileInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Path:     "README.md",
		},
	); err != nil {
		t.Fatalf("StageWorkspaceFile() error = %v", err)
	}

	if _, err := fixture.service.DiscardWorkspaceFile(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceDiscardFileInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Path:     "README.md",
		},
	); err != nil {
		t.Fatalf("DiscardWorkspaceFile() error = %v", err)
	}

	diff, err := fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() after discard error = %v", err)
	}
	if diff.Dirty {
		t.Fatalf("expected clean diff after discard, got %+v", diff)
	}
	//nolint:gosec // Test reads a fixture-controlled temp repo file.
	content, err := os.ReadFile(filepath.Join(repoPath, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "base\n" {
		t.Fatalf("README.md content = %q", string(content))
	}
}

func TestProjectConversationWorkspaceStageAllAndUnstageFile(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	writeConversationWorkspaceFile(t, filepath.Join(repoPath, "README.md"), "updated\n")
	writeConversationWorkspaceFile(t, filepath.Join(repoPath, "notes.txt"), "hello\n")

	if _, err := fixture.service.StageWorkspaceAll(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceStageAllInput{
			RepoPath: WorkspaceRepoPath("backend"),
		},
	); err != nil {
		t.Fatalf("StageWorkspaceAll() error = %v", err)
	}

	diff, err := fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() after stage all error = %v", err)
	}
	filesByPath := map[string]ProjectConversationWorkspaceFileDiff{}
	for _, file := range diff.Repos[0].Files {
		filesByPath[file.Path] = file
	}
	if got := filesByPath["README.md"]; !got.Staged || got.Unstaged {
		t.Fatalf("expected README.md staged-only after stage all, got %+v", got)
	}
	if got := filesByPath["notes.txt"]; !got.Staged || got.Unstaged {
		t.Fatalf("expected notes.txt staged-only after stage all, got %+v", got)
	}

	if _, err := fixture.service.UnstageWorkspace(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceUnstageInput{
			RepoPath: WorkspaceRepoPath("backend"),
			Path:     "README.md",
		},
	); err != nil {
		t.Fatalf("UnstageWorkspace() error = %v", err)
	}

	diff, err = fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() after unstage file error = %v", err)
	}
	filesByPath = map[string]ProjectConversationWorkspaceFileDiff{}
	for _, file := range diff.Repos[0].Files {
		filesByPath[file.Path] = file
	}
	if got := filesByPath["README.md"]; got.Staged || !got.Unstaged {
		t.Fatalf("expected README.md unstaged after unstage file, got %+v", got)
	}
	if got := filesByPath["notes.txt"]; !got.Staged || got.Unstaged {
		t.Fatalf("expected notes.txt to stay staged, got %+v", got)
	}
}

func TestProjectConversationWorkspaceUnstageAllRestoresUnstagedState(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
		{
			name: "backend",
			files: map[string]string{
				"README.md": "base\n",
			},
		},
	})
	repoPath := fixture.repoPaths["backend"]
	writeConversationWorkspaceFile(t, filepath.Join(repoPath, "README.md"), "updated\n")

	if _, err := fixture.service.StageWorkspaceAll(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceStageAllInput{
			RepoPath: WorkspaceRepoPath("backend"),
		},
	); err != nil {
		t.Fatalf("StageWorkspaceAll() error = %v", err)
	}

	if _, err := fixture.service.UnstageWorkspace(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
		ProjectConversationWorkspaceUnstageInput{
			RepoPath: WorkspaceRepoPath("backend"),
		},
	); err != nil {
		t.Fatalf("UnstageWorkspace() error = %v", err)
	}

	diff, err := fixture.service.GetWorkspaceDiff(
		fixture.ctx,
		UserID("user:conversation"),
		fixture.conversation.ID,
	)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() after unstage all error = %v", err)
	}
	if got := diff.Repos[0].Files[0]; got.Staged || !got.Unstaged {
		t.Fatalf("expected unstaged diff after unstage all, got %+v", got)
	}
}

func createConversationBranchCommit(
	t *testing.T,
	repoPath string,
	branchName string,
	files map[string]string,
) plumbing.Hash {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository %s: %v", repoPath, err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("worktree %s: %v", repoPath, err)
	}
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: true,
	})
	if err != nil && !errors.Is(err, git.ErrBranchExists) {
		t.Fatalf("checkout branch %s: %v", branchName, err)
	}
	if errors.Is(err, git.ErrBranchExists) {
		if err := worktree.Checkout(&git.CheckoutOptions{Branch: branchRef}); err != nil {
			t.Fatalf("checkout existing branch %s: %v", branchName, err)
		}
	}
	for path, content := range files {
		absolutePath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		if _, err := worktree.Add(path); err != nil {
			t.Fatalf("add %s: %v", path, err)
		}
	}
	hash, err := worktree.Commit("branch commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@example.com",
			When:  time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("commit branch %s: %v", branchName, err)
	}
	return hash
}

func projectConversationWorkspaceLabelExists(
	labels []ProjectConversationWorkspaceGitRefLabel,
	name string,
) bool {
	for _, label := range labels {
		if label.Name == name {
			return true
		}
	}
	return false
}

func projectConversationWorkspaceBranchRefExistsByName(
	refs []ProjectConversationWorkspaceBranchRef,
	name string,
) bool {
	for _, ref := range refs {
		if ref.Name == name {
			return true
		}
	}
	return false
}
