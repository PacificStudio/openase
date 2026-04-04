package workspace

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/logging"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	transport "github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

var _ = logging.DeclareComponent("workspace-manager")

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
	Name             string
	RepositoryURL    string
	DefaultBranch    string
	WorkspaceDirname *string
	BranchName       *string
	HTTPBasicAuth    *HTTPBasicAuthInput
}

// HTTPBasicAuthInput describes raw HTTPS credentials for one repository.
type HTTPBasicAuthInput struct {
	Username string
	Password string
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
	Name             string
	RepositoryURL    string
	DefaultBranch    string
	WorkspaceDirname string
	BranchName       string
	HeadCommit       string
	HTTPBasicAuth    *HTTPBasicAuthRequest
}

// HTTPBasicAuthRequest is the parsed HTTPS credential for one repository.
type HTTPBasicAuthRequest struct {
	Username string
	Password string
}

// Workspace describes a prepared ticket workspace on disk.
type Workspace struct {
	Path       string
	BranchName string
	Repos      []PreparedRepo
}

// PreparedRepo describes one repository that was prepared inside a workspace.
type PreparedRepo struct {
	Name             string
	RepositoryURL    string
	DefaultBranch    string
	BranchName       string
	WorkspaceDirname string
	HeadCommit       string
	Path             string
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

	ticketIdentifier, err := parseBranchSegment("ticket_identifier", input.TicketIdentifier)
	if err != nil {
		return SetupRequest{}, err
	}

	branchName := fmt.Sprintf("agent/%s", ticketIdentifier)
	// Tickets without registered project repos still need a deterministic
	// workspace root so hooks, harness rendering, and agent launches can share
	// the same ticket-scoped path convention.
	repos := make([]RepoRequest, 0, len(input.Repos))
	workspaceDirnames := make(map[string]struct{}, len(input.Repos))
	for index, rawRepo := range input.Repos {
		repo, err := parseRepoInput(index, rawRepo, branchName)
		if err != nil {
			return SetupRequest{}, err
		}
		if _, exists := workspaceDirnames[repo.WorkspaceDirname]; exists {
			return SetupRequest{}, fmt.Errorf("repos[%d].workspace_dirname duplicates %q", index, repo.WorkspaceDirname)
		}
		workspaceDirnames[repo.WorkspaceDirname] = struct{}{}
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
		repoPath := RepoPath(workspacePath, repo.WorkspaceDirname, repo.Name)
		if err := os.MkdirAll(filepath.Dir(repoPath), 0o750); err != nil {
			return Workspace{}, fmt.Errorf("create parent directory for repo %s: %w", repo.Name, err)
		}

		headCommit, err := prepareRepository(ctx, repoPath, repo)
		if err != nil {
			return Workspace{}, err
		}

		preparedRepos = append(preparedRepos, PreparedRepo{
			Name:             repo.Name,
			RepositoryURL:    repo.RepositoryURL,
			DefaultBranch:    repo.DefaultBranch,
			BranchName:       repo.BranchName,
			WorkspaceDirname: repo.WorkspaceDirname,
			HeadCommit:       headCommit,
			Path:             repoPath,
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

	repositoryURL, err := parseRepositorySource(fmt.Sprintf("repos[%d].repository_url", index), input.RepositoryURL)
	if err != nil {
		return RepoRequest{}, err
	}

	defaultBranch := strings.TrimSpace(input.DefaultBranch)
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if strings.Contains(defaultBranch, "/") {
		return RepoRequest{}, fmt.Errorf("repos[%d].default_branch must not contain '/'", index)
	}

	workspaceDirname := name
	if input.WorkspaceDirname != nil {
		workspaceDirname, err = parseWorkspaceDirname(fmt.Sprintf("repos[%d].workspace_dirname", index), *input.WorkspaceDirname)
		if err != nil {
			return RepoRequest{}, err
		}
	}

	if input.BranchName != nil {
		parsedBranchName, err := parseRepoBranchName(fmt.Sprintf("repos[%d].branch_name", index), *input.BranchName)
		if err != nil {
			return RepoRequest{}, err
		}
		branchName = parsedBranchName
	}

	httpBasicAuth, err := parseHTTPBasicAuth(fmt.Sprintf("repos[%d].http_basic_auth", index), input.HTTPBasicAuth)
	if err != nil {
		return RepoRequest{}, err
	}

	return RepoRequest{
		Name:             name,
		RepositoryURL:    repositoryURL,
		DefaultBranch:    defaultBranch,
		WorkspaceDirname: workspaceDirname,
		BranchName:       branchName,
		HTTPBasicAuth:    httpBasicAuth,
	}, nil
}

func parseHTTPBasicAuth(fieldName string, input *HTTPBasicAuthInput) (*HTTPBasicAuthRequest, error) {
	if input == nil {
		return nil, nil
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		return nil, fmt.Errorf("%s.username must not be empty", fieldName)
	}
	password := strings.TrimSpace(input.Password)
	if password == "" {
		return nil, fmt.Errorf("%s.password must not be empty", fieldName)
	}

	return &HTTPBasicAuthRequest{
		Username: username,
		Password: password,
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

func parseRepoBranchName(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}
	segments := strings.Split(trimmed, "/")
	for index, segment := range segments {
		if _, err := parseBranchSegment(fmt.Sprintf("%s[%d]", fieldName, index), segment); err != nil {
			return "", err
		}
	}

	return strings.Join(segments, "/"), nil
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

func parseWorkspaceDirname(fieldName string, raw string) (string, error) {
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

func parseAbsolutePath(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}
	if !filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("%s must be absolute", fieldName)
	}

	return filepath.Clean(trimmed), nil
}

func parseRepositorySource(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}
	return trimmed, nil
}

func prepareRepository(ctx context.Context, repoPath string, repo RepoRequest) (string, error) {
	repository, existing, err := openOrCloneRepository(ctx, repoPath, repo)
	if err != nil {
		return "", fmt.Errorf("prepare repo %s: %w", repo.Name, err)
	}

	if err := ensureOriginMatches(repository, repo.RepositoryURL); err != nil {
		return "", fmt.Errorf("prepare repo %s: %w", repo.Name, err)
	}
	if existing {
		head, err := repository.Head()
		if err != nil {
			return "", fmt.Errorf("prepare repo %s: resolve existing head: %w", repo.Name, err)
		}
		return head.Hash().String(), nil
	}

	headCommit, err := ensureFeatureBranchCheckedOut(repository, repo.DefaultBranch, repo.BranchName)
	if err != nil {
		return "", fmt.Errorf("prepare repo %s: %w", repo.Name, err)
	}

	return headCommit, nil
}

func cloneOrOpenRepository(ctx context.Context, repoPath string, repo RepoRequest) (*git.Repository, error) {
	repository, _, err := openOrCloneRepository(ctx, repoPath, repo)
	return repository, err
}

func openOrCloneRepository(ctx context.Context, repoPath string, repo RepoRequest) (*git.Repository, bool, error) {
	stat, err := os.Stat(repoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cloneOptions, cloneOptionsErr := buildCloneOptions(repo)
			if cloneOptionsErr != nil {
				return nil, false, fmt.Errorf("build clone options for %s: %w", repo.RepositoryURL, cloneOptionsErr)
			}
			repository, cloneErr := git.PlainCloneContext(ctx, repoPath, false, cloneOptions)
			if cloneErr != nil {
				return nil, false, fmt.Errorf("clone repository %s into %s: %w", repo.RepositoryURL, repoPath, cloneErr)
			}

			return repository, false, nil
		}

		return nil, false, fmt.Errorf("stat repository path %s: %w", repoPath, err)
	}
	if !stat.IsDir() {
		return nil, false, fmt.Errorf("repository path %s is not a directory", repoPath)
	}

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, false, fmt.Errorf("open repository %s: %w", repoPath, err)
	}

	return repository, true, nil
}

func fetchRepository(ctx context.Context, repository *git.Repository, repo RepoRequest) error {
	fetchOptions, err := buildFetchOptions(repo)
	if err != nil {
		return fmt.Errorf("build fetch options for %s: %w", repo.RepositoryURL, err)
	}
	err = repository.FetchContext(ctx, fetchOptions)
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

func ensureFeatureBranchCheckedOut(repository *git.Repository, defaultBranch string, featureBranch string) (string, error) {
	remoteRefName := plumbing.NewRemoteReferenceName("origin", defaultBranch)
	remoteRef, err := repository.Reference(remoteRefName, true)
	if err != nil {
		return "", fmt.Errorf("resolve remote default branch %s: %w", defaultBranch, err)
	}
	targetHash := remoteRef.Hash()

	featureRemoteRefName := plumbing.NewRemoteReferenceName("origin", featureBranch)
	featureRemoteRef, err := repository.Reference(featureRemoteRefName, true)
	switch {
	case err == nil:
		targetHash = featureRemoteRef.Hash()
	case errors.Is(err, plumbing.ErrReferenceNotFound):
	default:
		return "", fmt.Errorf("lookup remote work branch %s: %w", featureBranch, err)
	}

	featureRefName := plumbing.NewBranchReferenceName(featureBranch)
	if _, err := repository.Reference(featureRefName, true); err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", fmt.Errorf("lookup feature branch %s: %w", featureBranch, err)
		}
		if err := repository.Storer.SetReference(plumbing.NewHashReference(featureRefName, targetHash)); err != nil {
			return "", fmt.Errorf("create feature branch %s: %w", featureBranch, err)
		}
	}

	head, err := repository.Head()
	if err == nil && head.Name() == featureRefName {
		return head.Hash().String(), nil
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return "", fmt.Errorf("load worktree: %w", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: featureRefName}); err != nil {
		return "", fmt.Errorf("checkout feature branch %s: %w", featureBranch, err)
	}

	head, err = repository.Head()
	if err != nil {
		return "", fmt.Errorf("resolve feature branch head %s: %w", featureBranch, err)
	}
	return head.Hash().String(), nil
}

func buildCloneOptions(repo RepoRequest) (*git.CloneOptions, error) {
	auth, err := buildRepositoryAuthMethod(repo)
	if err != nil {
		return nil, err
	}
	return &git.CloneOptions{
		URL:  repo.RepositoryURL,
		Auth: auth,
	}, nil
}

func buildFetchOptions(repo RepoRequest) (*git.FetchOptions, error) {
	auth, err := buildRepositoryAuthMethod(repo)
	if err != nil {
		return nil, err
	}
	return &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
	}, nil
}

func buildRepositoryAuthMethod(repo RepoRequest) (transport.AuthMethod, error) {
	endpoint, err := transport.NewEndpoint(repo.RepositoryURL)
	if err != nil {
		return nil, fmt.Errorf("parse repository URL %q: %w", repo.RepositoryURL, err)
	}
	if endpoint.Protocol != "ssh" {
		if repo.HTTPBasicAuth == nil || (endpoint.Protocol != "https" && endpoint.Protocol != "http") {
			return nil, nil
		}
		return &githttp.BasicAuth{
			Username: repo.HTTPBasicAuth.Username,
			Password: repo.HTTPBasicAuth.Password,
		}, nil
	}

	keyPath, hasKey, err := defaultSSHPrivateKeyPath()
	if err != nil {
		return nil, err
	}
	if !hasKey {
		return nil, nil
	}

	user := strings.TrimSpace(endpoint.User)
	if user == "" {
		user = gitssh.DefaultUsername
	}

	auth, err := gitssh.NewPublicKeysFromFile(user, keyPath, "")
	if err != nil {
		return nil, fmt.Errorf("load ssh private key %s: %w", keyPath, err)
	}
	return auth, nil
}

func repositoryHTTPSExtraHeader(repo RepoRequest) (string, string, bool, error) {
	if repo.HTTPBasicAuth == nil {
		return "", "", false, nil
	}

	endpoint, err := transport.NewEndpoint(repo.RepositoryURL)
	if err != nil {
		return "", "", false, fmt.Errorf("parse repository URL %q: %w", repo.RepositoryURL, err)
	}
	if endpoint.Protocol != "https" && endpoint.Protocol != "http" {
		return "", "", false, nil
	}

	authToken := base64.StdEncoding.EncodeToString([]byte(repo.HTTPBasicAuth.Username + ":" + repo.HTTPBasicAuth.Password))
	return "http." + endpoint.Protocol + "://" + endpoint.Host + "/.extraheader", "AUTHORIZATION: basic " + authToken, true, nil
}

func defaultSSHPrivateKeyPath() (string, bool, error) {
	overridePath := strings.TrimSpace(os.Getenv("OPENASE_GIT_SSH_KEY_PATH"))
	if overridePath != "" {
		info, err := os.Stat(overridePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return overridePath, false, nil
			}
			return "", false, fmt.Errorf("stat OPENASE_GIT_SSH_KEY_PATH %q: %w", overridePath, err)
		}
		if info.IsDir() {
			return "", false, fmt.Errorf("OPENASE_GIT_SSH_KEY_PATH %q must be a file", overridePath)
		}
		return overridePath, true, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("resolve user home for ssh auth: %w", err)
	}
	keyPath := filepath.Join(homeDir, ".ssh", "id_ed25519")
	info, err := os.Stat(keyPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return keyPath, false, nil
		}
		return "", false, fmt.Errorf("stat ssh private key %q: %w", keyPath, err)
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("ssh private key %q must be a file", keyPath)
	}
	return keyPath, true, nil
}
