package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	transport "github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

var safeSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// SetupInput is the raw boundary input for workspace preparation.
type SetupInput struct {
	WorkspaceRoot    string
	OrganizationSlug string
	ProjectSlug      string
	AgentName        string
	TicketIdentifier string
	Repos            []RepoInput
}

// RepoInput describes one repository to materialize in a workspace.
type RepoInput struct {
	Name          string
	RepositoryURL string
	DefaultBranch string
	ClonePath     *string
	BranchName    *string
	GitHubToken   *string
}

// SetupRequest is the parsed workspace preparation request.
type SetupRequest struct {
	WorkspaceRoot    string
	OrganizationSlug string
	ProjectSlug      string
	TicketIdentifier string
	BranchName       string
	Repos            []RepoRequest
}

// RepoRequest is the parsed repository setup request.
type RepoRequest struct {
	Name          string
	RepositoryURL string
	DefaultBranch string
	ClonePath     string
	BranchName    string
	GitHubToken   string
}

// Workspace describes a prepared ticket workspace on disk.
type Workspace struct {
	Path       string
	BranchName string
	Repos      []PreparedRepo
}

// PreparedRepo describes one repository that was prepared inside a workspace.
type PreparedRepo struct {
	Name          string
	RepositoryURL string
	DefaultBranch string
	BranchName    string
	ClonePath     string
	Path          string
}

// Manager prepares ticket workspaces and repository clones.
type Manager struct{}

// NewManager constructs a workspace preparation manager.
func NewManager() *Manager {
	return &Manager{}
}

// ParseSetupRequest validates raw workspace setup input into a parsed request.
func ParseSetupRequest(input SetupInput) (SetupRequest, error) {
	workspaceRoot, err := parseTicketWorkspaceRoot(input.WorkspaceRoot, true)
	if err != nil {
		return SetupRequest{}, err
	}

	organizationSlug, err := parsePathSegment("organization_slug", input.OrganizationSlug)
	if err != nil {
		return SetupRequest{}, err
	}

	projectSlug, err := parsePathSegment("project_slug", input.ProjectSlug)
	if err != nil {
		return SetupRequest{}, err
	}

	agentName, err := parseBranchSegment("agent_name", input.AgentName)
	if err != nil {
		return SetupRequest{}, err
	}

	ticketIdentifier, err := parseBranchSegment("ticket_identifier", input.TicketIdentifier)
	if err != nil {
		return SetupRequest{}, err
	}

	branchName := fmt.Sprintf("agent/%s/%s", agentName, ticketIdentifier)
	// Tickets without registered project repos still need a deterministic
	// workspace root so hooks, harness rendering, and agent launches can share
	// the same ticket-scoped path convention.
	repos := make([]RepoRequest, 0, len(input.Repos))
	clonePaths := make(map[string]struct{}, len(input.Repos))
	for index, rawRepo := range input.Repos {
		repo, err := parseRepoInput(index, rawRepo, branchName)
		if err != nil {
			return SetupRequest{}, err
		}
		if _, exists := clonePaths[repo.ClonePath]; exists {
			return SetupRequest{}, fmt.Errorf("repos[%d].clone_path duplicates %q", index, repo.ClonePath)
		}
		clonePaths[repo.ClonePath] = struct{}{}
		repos = append(repos, repo)
	}

	return SetupRequest{
		WorkspaceRoot:    workspaceRoot,
		OrganizationSlug: organizationSlug,
		ProjectSlug:      projectSlug,
		TicketIdentifier: ticketIdentifier,
		BranchName:       branchName,
		Repos:            repos,
	}, nil
}

// Prepare creates the workspace root and materializes each configured repository.
func (m *Manager) Prepare(ctx context.Context, request SetupRequest) (Workspace, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	workspacePath, err := TicketWorkspacePath(
		request.WorkspaceRoot,
		request.OrganizationSlug,
		request.ProjectSlug,
		request.TicketIdentifier,
	)
	if err != nil {
		return Workspace{}, fmt.Errorf("derive workspace path: %w", err)
	}
	if err := os.MkdirAll(workspacePath, 0o750); err != nil {
		return Workspace{}, fmt.Errorf("create workspace root %s: %w", workspacePath, err)
	}

	preparedRepos := make([]PreparedRepo, 0, len(request.Repos))
	for _, repo := range request.Repos {
		repoPath := RepoPath(workspacePath, repo.ClonePath, repo.Name)
		if err := os.MkdirAll(filepath.Dir(repoPath), 0o750); err != nil {
			return Workspace{}, fmt.Errorf("create parent directory for repo %s: %w", repo.Name, err)
		}

		if err := prepareRepository(ctx, repoPath, repo); err != nil {
			return Workspace{}, err
		}

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

func parseRepoInput(index int, input RepoInput, branchName string) (RepoRequest, error) {
	name, err := parsePathSegment(fmt.Sprintf("repos[%d].name", index), input.Name)
	if err != nil {
		return RepoRequest{}, err
	}

	repositoryURL := strings.TrimSpace(input.RepositoryURL)
	if repositoryURL == "" {
		return RepoRequest{}, fmt.Errorf("repos[%d].repository_url must not be empty", index)
	}

	defaultBranch := strings.TrimSpace(input.DefaultBranch)
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if strings.Contains(defaultBranch, "/") {
		return RepoRequest{}, fmt.Errorf("repos[%d].default_branch must not contain '/'", index)
	}

	clonePath := name
	if input.ClonePath != nil {
		clonePath, err = parseRelativeWorkspacePath(fmt.Sprintf("repos[%d].clone_path", index), *input.ClonePath)
		if err != nil {
			return RepoRequest{}, err
		}
	}

	if input.BranchName != nil {
		parsedBranchName := strings.TrimSpace(*input.BranchName)
		if parsedBranchName != branchName {
			return RepoRequest{}, fmt.Errorf("repos[%d].branch_name must equal %q", index, branchName)
		}
	}

	return RepoRequest{
		Name:          name,
		RepositoryURL: repositoryURL,
		DefaultBranch: defaultBranch,
		ClonePath:     clonePath,
		BranchName:    branchName,
		GitHubToken:   strings.TrimSpace(optionalStringValue(input.GitHubToken)),
	}, nil
}

func parseBranchSegment(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}
	if !safeSegmentPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must match %s", fieldName, safeSegmentPattern.String())
	}

	return trimmed, nil
}

func parsePathSegment(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}
	if !safeSegmentPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must match %s", fieldName, safeSegmentPattern.String())
	}

	return trimmed, nil
}

func parseRelativeWorkspacePath(fieldName string, raw string) (string, error) {
	trimmed := path.Clean(strings.TrimSpace(filepath.ToSlash(raw)))
	if trimmed == "." || trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}
	if strings.HasPrefix(trimmed, "/") || filepath.IsAbs(raw) {
		return "", fmt.Errorf("%s must be relative", fieldName)
	}
	if trimmed == ".." || strings.HasPrefix(trimmed, "../") {
		return "", fmt.Errorf("%s must stay inside the workspace", fieldName)
	}

	segments := strings.Split(trimmed, "/")
	for _, segment := range segments {
		if !safeSegmentPattern.MatchString(segment) {
			return "", fmt.Errorf("%s segment %q must match %s", fieldName, segment, safeSegmentPattern.String())
		}
	}

	return trimmed, nil
}

func prepareRepository(ctx context.Context, repoPath string, repo RepoRequest) error {
	repository, err := cloneOrOpenRepository(ctx, repoPath, repo.RepositoryURL, repo.GitHubToken)
	if err != nil {
		return fmt.Errorf("prepare repo %s: %w", repo.Name, err)
	}

	if err := ensureOriginMatches(repository, repo.RepositoryURL); err != nil {
		return fmt.Errorf("prepare repo %s: %w", repo.Name, err)
	}

	if err := ensureFeatureBranchCheckedOut(repository, repo.DefaultBranch, repo.BranchName); err != nil {
		return fmt.Errorf("prepare repo %s: %w", repo.Name, err)
	}

	return nil
}

func cloneOrOpenRepository(ctx context.Context, repoPath string, repositoryURL string, githubToken string) (*git.Repository, error) {
	stat, err := os.Stat(repoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			repository, cloneErr := git.PlainCloneContext(ctx, repoPath, false, buildCloneOptions(repositoryURL, githubToken))
			if cloneErr != nil {
				return nil, fmt.Errorf("clone repository %s into %s: %w", repositoryURL, repoPath, cloneErr)
			}

			return repository, nil
		}

		return nil, fmt.Errorf("stat repository path %s: %w", repoPath, err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("repository path %s is not a directory", repoPath)
	}

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repository %s: %w", repoPath, err)
	}

	if err := fetchRepository(ctx, repository, repositoryURL, githubToken); err != nil {
		return nil, fmt.Errorf("fetch repository %s: %w", repoPath, err)
	}

	return repository, nil
}

func fetchRepository(ctx context.Context, repository *git.Repository, repositoryURL string, githubToken string) error {
	err := repository.FetchContext(ctx, buildFetchOptions(repositoryURL, githubToken))
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	return nil
}

func ensureOriginMatches(repository *git.Repository, expectedURL string) error {
	remote, err := repository.Remote("origin")
	if err != nil {
		return fmt.Errorf("load origin remote: %w", err)
	}
	if len(remote.Config().URLs) == 0 {
		return errors.New("origin remote has no configured URLs")
	}

	actualURL := strings.TrimSpace(remote.Config().URLs[0])
	if actualURL != expectedURL {
		return fmt.Errorf("origin remote URL mismatch: got %q want %q", actualURL, expectedURL)
	}

	return nil
}

func ensureFeatureBranchCheckedOut(repository *git.Repository, defaultBranch string, featureBranch string) error {
	remoteRefName := plumbing.NewRemoteReferenceName("origin", defaultBranch)
	remoteRef, err := repository.Reference(remoteRefName, true)
	if err != nil {
		return fmt.Errorf("resolve remote default branch %s: %w", defaultBranch, err)
	}

	featureRefName := plumbing.NewBranchReferenceName(featureBranch)
	if _, err := repository.Reference(featureRefName, true); err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("lookup feature branch %s: %w", featureBranch, err)
		}
		if err := repository.Storer.SetReference(plumbing.NewHashReference(featureRefName, remoteRef.Hash())); err != nil {
			return fmt.Errorf("create feature branch %s: %w", featureBranch, err)
		}
	}

	head, err := repository.Head()
	if err == nil && head.Name() == featureRefName {
		return nil
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("load worktree: %w", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: featureRefName}); err != nil {
		return fmt.Errorf("checkout feature branch %s: %w", featureBranch, err)
	}

	return nil
}

func buildCloneOptions(repositoryURL string, githubToken string) *git.CloneOptions {
	return &git.CloneOptions{
		URL:  repositoryURL,
		Auth: gitAuthMethod(repositoryURL, githubToken),
	}
}

func buildFetchOptions(repositoryURL string, githubToken string) *git.FetchOptions {
	return &git.FetchOptions{
		RemoteName: "origin",
		Auth:       gitAuthMethod(repositoryURL, githubToken),
	}
}

func gitAuthMethod(repositoryURL string, githubToken string) transport.AuthMethod {
	if strings.TrimSpace(githubToken) == "" {
		return nil
	}
	if _, ok := githubauthdomain.ParseGitHubRepositoryURL(repositoryURL); !ok {
		return nil
	}
	return &githttp.BasicAuth{
		Username: "x-access-token",
		Password: strings.TrimSpace(githubToken),
	}
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
