package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

const (
	projectConversationWorkspacePreviewLimitBytes = 256 * 1024
	projectConversationWorkspaceDiffLimitBytes    = 256 * 1024
)

type ProjectConversationWorkspaceMetadata struct {
	ConversationID uuid.UUID
	Available      bool
	WorkspacePath  string
	Repos          []ProjectConversationWorkspaceRepoMetadata
	SyncPrompt     *ProjectConversationWorkspaceSyncPrompt
}

type ProjectConversationWorkspaceRepoMetadata struct {
	Name         string
	Path         string
	Branch       string
	HeadCommit   string
	HeadSummary  string
	Dirty        bool
	FilesChanged int
	Added        int
	Removed      int
}

type ProjectConversationWorkspaceTree struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
	Entries        []ProjectConversationWorkspaceTreeEntry
}

type ProjectConversationWorkspaceSearch struct {
	ConversationID uuid.UUID
	RepoPath       string
	Query          string
	Truncated      bool
	Results        []ProjectConversationWorkspaceSearchResult
}

type ProjectConversationWorkspaceSearchResult struct {
	Path string
	Name string
}

type ProjectConversationWorkspaceTreeEntry struct {
	Path      string
	Name      string
	Kind      ProjectConversationWorkspaceTreeEntryKind
	SizeBytes int64
}

type ProjectConversationWorkspaceTreeEntryKind string

const (
	ProjectConversationWorkspaceTreeEntryKindDirectory ProjectConversationWorkspaceTreeEntryKind = "directory"
	ProjectConversationWorkspaceTreeEntryKindFile      ProjectConversationWorkspaceTreeEntryKind = "file"
)

const (
	projectConversationWorkspaceSearchDefaultLimit = 20
	projectConversationWorkspaceSearchMaxLimit     = 100
)

type ProjectConversationWorkspaceFilePreview struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
	SizeBytes      int64
	MediaType      string
	PreviewKind    ProjectConversationWorkspacePreviewKind
	Truncated      bool
	Content        string
	Revision       string
	Writable       bool
	ReadOnlyReason string
	Encoding       string
	LineEnding     string
}

type ProjectConversationWorkspacePreviewKind string

const (
	ProjectConversationWorkspacePreviewKindText   ProjectConversationWorkspacePreviewKind = "text"
	ProjectConversationWorkspacePreviewKindBinary ProjectConversationWorkspacePreviewKind = "binary"
)

type ProjectConversationWorkspaceFilePatch struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
	Status         ProjectConversationWorkspaceFileStatus
	DiffKind       ProjectConversationWorkspaceDiffKind
	Truncated      bool
	Diff           string
}

type ProjectConversationWorkspaceDiffKind string

const (
	ProjectConversationWorkspaceDiffKindNone   ProjectConversationWorkspaceDiffKind = "none"
	ProjectConversationWorkspaceDiffKindText   ProjectConversationWorkspaceDiffKind = "text"
	ProjectConversationWorkspaceDiffKindBinary ProjectConversationWorkspaceDiffKind = "binary"
)

var (
	ErrProjectConversationWorkspaceUnavailable   = errors.New("project conversation workspace unavailable")
	ErrProjectConversationWorkspaceRepoNotFound  = errors.New("project conversation workspace repo not found")
	ErrProjectConversationWorkspacePathInvalid   = errors.New("project conversation workspace path is invalid")
	ErrProjectConversationWorkspaceEntryNotFound = errors.New("project conversation workspace entry not found")
	ErrProjectConversationWorkspaceEntryExists   = errors.New("project conversation workspace entry already exists")
)

type projectConversationWorkspaceResolvedRepo struct {
	conversationID uuid.UUID
	machine        catalogdomain.Machine
	workspacePath  string
	repo           projectConversationWorkspaceRepoLocation
}

func (s *ProjectConversationService) GetWorkspaceMetadata(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (ProjectConversationWorkspaceMetadata, error) {
	conversation, location, err := s.resolveConversationWorkspace(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, errProjectConversationWorkspaceLocationUnavailable) &&
			projectConversationWorkspaceMayNotExistYet(conversation) {
			return ProjectConversationWorkspaceMetadata{
				ConversationID: conversationID,
				Available:      false,
			}, nil
		}
		return ProjectConversationWorkspaceMetadata{}, err
	}

	metadata := ProjectConversationWorkspaceMetadata{
		ConversationID: conversationID,
		Available:      true,
		WorkspacePath:  location.workspacePath,
		Repos:          make([]ProjectConversationWorkspaceRepoMetadata, 0, len(location.repos)),
		SyncPrompt:     location.syncPrompt,
	}
	for _, repo := range location.repos {
		item, err := s.readConversationWorkspaceRepoMetadata(ctx, location.machine, repo)
		if err != nil {
			return ProjectConversationWorkspaceMetadata{}, err
		}
		metadata.Repos = append(metadata.Repos, item)
	}
	sort.Slice(metadata.Repos, func(i, j int) bool {
		return metadata.Repos[i].Path < metadata.Repos[j].Path
	})
	return metadata, nil
}

func (s *ProjectConversationService) ListWorkspaceTree(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	repoPath string,
	treePath string,
) (ProjectConversationWorkspaceTree, error) {
	resolved, relativePath, err := s.resolveConversationWorkspaceRepoPath(ctx, userID, conversationID, repoPath, treePath, true)
	if err != nil {
		return ProjectConversationWorkspaceTree{}, err
	}

	entries, err := s.readConversationWorkspaceTreeEntries(ctx, resolved.machine, resolved.repo.repoPath, relativePath)
	if err != nil {
		return ProjectConversationWorkspaceTree{}, err
	}
	return ProjectConversationWorkspaceTree{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Path:           relativePath,
		Entries:        entries,
	}, nil
}

func (s *ProjectConversationService) SearchWorkspacePaths(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	repoPath string,
	query string,
	limit int,
) (ProjectConversationWorkspaceSearch, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		repoPath,
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceSearch{}, err
	}

	searchQuery := strings.TrimSpace(query)
	if searchQuery == "" {
		return ProjectConversationWorkspaceSearch{}, ErrProjectConversationWorkspacePathInvalid
	}

	results, truncated, err := s.searchConversationWorkspacePaths(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
		searchQuery,
		normalizeProjectConversationWorkspaceSearchLimit(limit),
	)
	if err != nil {
		return ProjectConversationWorkspaceSearch{}, err
	}

	return ProjectConversationWorkspaceSearch{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Query:          searchQuery,
		Truncated:      truncated,
		Results:        results,
	}, nil
}

func (s *ProjectConversationService) ReadWorkspaceFilePreview(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	repoPath string,
	filePath string,
) (ProjectConversationWorkspaceFilePreview, error) {
	resolved, relativePath, err := s.resolveConversationWorkspaceRepoPath(ctx, userID, conversationID, repoPath, filePath, false)
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, err
	}

	preview, err := s.readConversationWorkspaceFilePreview(ctx, resolved.machine, resolved.repo.repoPath, relativePath)
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, err
	}
	preview.ConversationID = resolved.conversationID
	preview.RepoPath = resolved.repo.relativePath
	preview.Path = relativePath
	return preview, nil
}

func (s *ProjectConversationService) ReadWorkspaceFilePatch(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	repoPath string,
	filePath string,
) (ProjectConversationWorkspaceFilePatch, error) {
	resolved, relativePath, err := s.resolveConversationWorkspaceRepoPath(ctx, userID, conversationID, repoPath, filePath, false)
	if err != nil {
		return ProjectConversationWorkspaceFilePatch{}, err
	}

	patch, err := s.readConversationWorkspaceFilePatch(ctx, resolved.machine, resolved.repo.repoPath, relativePath)
	if err != nil {
		return ProjectConversationWorkspaceFilePatch{}, err
	}
	patch.ConversationID = resolved.conversationID
	patch.RepoPath = resolved.repo.relativePath
	patch.Path = relativePath
	return patch, nil
}

func (s *ProjectConversationService) resolveConversationWorkspace(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (chatdomain.Conversation, projectConversationWorkspaceLocation, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return chatdomain.Conversation{}, projectConversationWorkspaceLocation{}, err
	}
	project, err := s.core.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return chatdomain.Conversation{}, projectConversationWorkspaceLocation{}, fmt.Errorf("get project for workspace browser: %w", err)
	}
	providerItem, err := s.core.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return chatdomain.Conversation{}, projectConversationWorkspaceLocation{}, fmt.Errorf("get provider for workspace browser: %w", err)
	}
	location, err := s.resolveConversationWorkspaceLocation(ctx, conversation, project, providerItem)
	if err != nil {
		return conversation, projectConversationWorkspaceLocation{}, err
	}
	return conversation, location, nil
}

func (s *ProjectConversationService) resolveConversationWorkspaceRepoPath(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	repoPath string,
	targetPath string,
	allowRoot bool,
) (projectConversationWorkspaceResolvedRepo, string, error) {
	conversation, location, err := s.resolveConversationWorkspace(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, errProjectConversationWorkspaceLocationUnavailable) &&
			projectConversationWorkspaceMayNotExistYet(conversation) {
			return projectConversationWorkspaceResolvedRepo{}, "", ErrProjectConversationWorkspaceUnavailable
		}
		return projectConversationWorkspaceResolvedRepo{}, "", err
	}

	relativeRepoPath := strings.TrimSpace(filepath.ToSlash(repoPath))
	for _, repo := range location.repos {
		if repo.relativePath != relativeRepoPath {
			continue
		}
		relativePath, err := parseProjectConversationWorkspaceRelativePath(targetPath, allowRoot)
		if err != nil {
			return projectConversationWorkspaceResolvedRepo{}, "", err
		}
		return projectConversationWorkspaceResolvedRepo{
			conversationID: conversationID,
			machine:        location.machine,
			workspacePath:  location.workspacePath,
			repo:           repo,
		}, relativePath, nil
	}
	if location.syncPrompt != nil {
		if _, ok := location.missingRepos[relativeRepoPath]; ok {
			return projectConversationWorkspaceResolvedRepo{}, "", &ProjectConversationWorkspaceSyncRequiredError{
				Prompt:   *location.syncPrompt,
				RepoPath: relativeRepoPath,
			}
		}
	}
	return projectConversationWorkspaceResolvedRepo{}, "", ErrProjectConversationWorkspaceRepoNotFound
}

func (s *ProjectConversationService) readConversationWorkspaceRepoMetadata(
	ctx context.Context,
	machine catalogdomain.Machine,
	repo projectConversationWorkspaceRepoLocation,
) (ProjectConversationWorkspaceRepoMetadata, error) {
	branch, err := workspaceinfra.ReadWorkspaceGitBranch(ctx, repo.repoPath, func(
		ctx context.Context,
		args []string,
		allowExitCodeOne bool,
	) ([]byte, error) {
		return s.runProjectConversationGitCommand(ctx, machine, args, allowExitCodeOne)
	})
	if err != nil {
		return ProjectConversationWorkspaceRepoMetadata{}, fmt.Errorf("read workspace branch for %s: %w", repo.name, err)
	}
	commitOutput, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repo.repoPath, "log", "-1", "--format=%H%x00%s"},
		false,
	)
	if err != nil {
		return ProjectConversationWorkspaceRepoMetadata{}, fmt.Errorf("read workspace head for %s: %w", repo.name, err)
	}
	commitParts := bytes.SplitN(commitOutput, []byte{0}, 2)
	headCommit := strings.TrimSpace(string(commitParts[0]))
	headSummary := ""
	if len(commitParts) == 2 {
		headSummary = strings.TrimSpace(string(commitParts[1]))
	}

	summary, err := s.summarizeConversationWorkspaceRepo(ctx, machine, repo)
	if err != nil {
		return ProjectConversationWorkspaceRepoMetadata{}, err
	}
	return ProjectConversationWorkspaceRepoMetadata{
		Name:         repo.name,
		Path:         repo.relativePath,
		Branch:       branch,
		HeadCommit:   shortenProjectConversationGitCommit(headCommit),
		HeadSummary:  headSummary,
		Dirty:        summary.Dirty,
		FilesChanged: summary.FilesChanged,
		Added:        summary.Added,
		Removed:      summary.Removed,
	}, nil
}

func shortenProjectConversationGitCommit(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:12]
}

func parseProjectConversationWorkspaceRelativePath(raw string, allowRoot bool) (string, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if trimmed == "" {
		if allowRoot {
			return "", nil
		}
		return "", ErrProjectConversationWorkspacePathInvalid
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", ErrProjectConversationWorkspacePathInvalid
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." {
		if allowRoot {
			return "", nil
		}
		return "", ErrProjectConversationWorkspacePathInvalid
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrProjectConversationWorkspacePathInvalid
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".git" {
			return "", ErrProjectConversationWorkspacePathInvalid
		}
	}
	return cleaned, nil
}

func (s *ProjectConversationService) readConversationWorkspaceTreeEntries(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) ([]ProjectConversationWorkspaceTreeEntry, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		return readLocalProjectConversationWorkspaceTreeEntries(repoRoot, relativePath)
	}
	return s.readRemoteProjectConversationWorkspaceTreeEntries(ctx, machine, repoRoot, relativePath)
}

func readLocalProjectConversationWorkspaceTreeEntries(
	repoRoot string,
	relativePath string,
) ([]ProjectConversationWorkspaceTreeEntry, error) {
	targetPath, err := resolveLocalProjectConversationWorkspaceDirectory(repoRoot, relativePath)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrProjectConversationWorkspaceEntryNotFound
		}
		return nil, fmt.Errorf("read workspace directory %s: %w", targetPath, err)
	}
	items := make([]ProjectConversationWorkspaceTreeEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat workspace entry %s: %w", entry.Name(), err)
		}
		item := ProjectConversationWorkspaceTreeEntry{
			Name: entry.Name(),
			Path: joinProjectConversationWorkspacePath(relativePath, entry.Name()),
			Kind: ProjectConversationWorkspaceTreeEntryKindFile,
		}
		if info.IsDir() {
			item.Kind = ProjectConversationWorkspaceTreeEntryKindDirectory
		} else {
			item.SizeBytes = info.Size()
		}
		items = append(items, item)
	}
	sortProjectConversationWorkspaceTreeEntries(items)
	return items, nil
}

func resolveLocalProjectConversationWorkspaceDirectory(repoRoot string, relativePath string) (string, error) {
	repoRealPath, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve workspace repo root: %w", err)
	}
	targetPath := repoRoot
	if relativePath != "" {
		targetPath = filepath.Join(repoRoot, filepath.FromSlash(relativePath))
	}
	targetRealPath, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrProjectConversationWorkspaceEntryNotFound
		}
		return "", fmt.Errorf("resolve workspace directory %s: %w", targetPath, err)
	}
	if !projectConversationWorkspacePathWithinRoot(repoRealPath, targetRealPath) {
		return "", ErrProjectConversationWorkspacePathInvalid
	}
	info, err := os.Stat(targetRealPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrProjectConversationWorkspaceEntryNotFound
		}
		return "", fmt.Errorf("stat workspace directory %s: %w", targetRealPath, err)
	}
	if !info.IsDir() {
		return "", ErrProjectConversationWorkspaceEntryNotFound
	}
	return targetRealPath, nil
}

func projectConversationWorkspacePathWithinRoot(root string, target string) bool {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	return cleanTarget == cleanRoot || strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator))
}

func sortProjectConversationWorkspaceTreeEntries(entries []ProjectConversationWorkspaceTreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == ProjectConversationWorkspaceTreeEntryKindDirectory
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

func joinProjectConversationWorkspacePath(parent string, name string) string {
	if strings.TrimSpace(parent) == "" {
		return filepath.ToSlash(name)
	}
	return filepath.ToSlash(filepath.Join(parent, name))
}

func normalizeProjectConversationWorkspaceSearchLimit(limit int) int {
	switch {
	case limit <= 0:
		return projectConversationWorkspaceSearchDefaultLimit
	case limit > projectConversationWorkspaceSearchMaxLimit:
		return projectConversationWorkspaceSearchMaxLimit
	default:
		return limit
	}
}

func (s *ProjectConversationService) searchConversationWorkspacePaths(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	query string,
	limit int,
) ([]ProjectConversationWorkspaceSearchResult, bool, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		return searchLocalProjectConversationWorkspacePaths(repoRoot, query, limit)
	}
	return s.searchRemoteProjectConversationWorkspacePaths(ctx, machine, repoRoot, query, limit)
}

var errProjectConversationWorkspaceSearchLimitReached = errors.New(
	"project conversation workspace search limit reached",
)

func searchLocalProjectConversationWorkspacePaths(
	repoRoot string,
	query string,
	limit int,
) ([]ProjectConversationWorkspaceSearchResult, bool, error) {
	repoRealPath, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return nil, false, fmt.Errorf("resolve workspace repo root: %w", err)
	}

	needle := strings.ToLower(strings.TrimSpace(query))
	results := make([]ProjectConversationWorkspaceSearchResult, 0, min(limit, 16))
	truncated := false

	walkErr := filepath.WalkDir(repoRealPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk workspace path %s: %w", path, walkErr)
		}
		if path == repoRealPath {
			return nil
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(repoRealPath, path)
		if err != nil {
			return fmt.Errorf("rel workspace path %s: %w", path, err)
		}
		relativePath = filepath.ToSlash(relativePath)
		if !strings.Contains(strings.ToLower(relativePath), needle) {
			return nil
		}
		if len(results) >= limit {
			truncated = true
			return errProjectConversationWorkspaceSearchLimitReached
		}
		results = append(results, ProjectConversationWorkspaceSearchResult{
			Path: relativePath,
			Name: filepath.Base(relativePath),
		})
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errProjectConversationWorkspaceSearchLimitReached) {
		return nil, false, walkErr
	}
	return results, truncated, nil
}

func (s *ProjectConversationService) readRemoteProjectConversationWorkspaceTreeEntries(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) ([]ProjectConversationWorkspaceTreeEntry, error) {
	command := fmt.Sprintf(`set -eu
repo=%s
relative=%s
repo_real=$(cd "$repo" && pwd -P)
target="$repo"
if [ -n "$relative" ]; then
  target="$repo/$relative"
fi
target_real=$(cd "$target" 2>/dev/null && pwd -P) || { echo missing >&2; exit 11; }
case "$target_real" in
  "$repo_real"|"$repo_real"/*) ;;
  *) echo escape >&2; exit 12 ;;
esac
if [ ! -d "$target_real" ]; then
  echo missing >&2
  exit 11
fi
find -P "$target_real" -mindepth 1 -maxdepth 1 \( -name .git -prune \) -o -printf '%%y\t%%f\t%%s\0'
`, projectConversationShellQuote(repoRoot), projectConversationShellQuote(relativePath))
	output, err := s.runProjectConversationShellCommand(ctx, machine, command, false)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 11") {
			return nil, ErrProjectConversationWorkspaceEntryNotFound
		}
		if strings.Contains(err.Error(), "exit status 12") {
			return nil, ErrProjectConversationWorkspacePathInvalid
		}
		return nil, fmt.Errorf("read remote workspace tree %s: %w", relativePath, err)
	}
	parts := bytes.Split(output, []byte{0})
	items := make([]ProjectConversationWorkspaceTreeEntry, 0, len(parts))
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		fields := strings.SplitN(string(part), "\t", 3)
		if len(fields) != 3 {
			return nil, fmt.Errorf("parse remote workspace tree entry %q", string(part))
		}
		item := ProjectConversationWorkspaceTreeEntry{
			Name: fields[1],
			Path: joinProjectConversationWorkspacePath(relativePath, fields[1]),
			Kind: ProjectConversationWorkspaceTreeEntryKindFile,
		}
		if fields[0] == "d" {
			item.Kind = ProjectConversationWorkspaceTreeEntryKindDirectory
		} else if fields[2] != "" {
			var size int64
			_, _ = fmt.Sscanf(fields[2], "%d", &size)
			item.SizeBytes = size
		}
		items = append(items, item)
	}
	sortProjectConversationWorkspaceTreeEntries(items)
	return items, nil
}

func (s *ProjectConversationService) searchRemoteProjectConversationWorkspacePaths(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	query string,
	limit int,
) ([]ProjectConversationWorkspaceSearchResult, bool, error) {
	command := fmt.Sprintf(`set -eu
repo=%s
repo_real=$(cd "$repo" && pwd -P)
find -P "$repo_real" \( -path "$repo_real/.git" -o -path "$repo_real/.git/*" \) -prune -o -type f -printf '%%P\0'
`, projectConversationShellQuote(repoRoot))
	output, err := s.runProjectConversationShellCommand(ctx, machine, command, false)
	if err != nil {
		return nil, false, fmt.Errorf("search remote workspace paths: %w", err)
	}

	needle := strings.ToLower(strings.TrimSpace(query))
	parts := bytes.Split(output, []byte{0})
	results := make([]ProjectConversationWorkspaceSearchResult, 0, min(limit, 16))
	truncated := false
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		relativePath := filepath.ToSlash(string(part))
		if !strings.Contains(strings.ToLower(relativePath), needle) {
			continue
		}
		if len(results) >= limit {
			truncated = true
			break
		}
		results = append(results, ProjectConversationWorkspaceSearchResult{
			Path: relativePath,
			Name: filepath.Base(relativePath),
		})
	}
	return results, truncated, nil
}

func projectConversationShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (s *ProjectConversationService) runProjectConversationShellCommand(
	ctx context.Context,
	machine catalogdomain.Machine,
	script string,
	allowExitCodeOne bool,
) ([]byte, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		command := exec.CommandContext(ctx, "sh")
		command.Stdin = strings.NewReader(script)
		output, err := command.CombinedOutput()
		if err != nil && (!allowExitCodeOne || !projectConversationCommandExitedWithCode(err, 1)) {
			return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return output, nil
	}
	if s == nil || s.core.sshPool == nil {
		return nil, fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
	}
	client, err := s.core.sshPool.Get(ctx, machine)
	if err != nil {
		return nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("open ssh session for workspace browser: %w", err)
	}
	defer func() { _ = session.Close() }()

	output, err := session.CombinedOutput("sh -lc " + sshinfra.ShellQuote(script))
	if err != nil && (!allowExitCodeOne || !strings.Contains(err.Error(), "exit status 1")) {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (s *ProjectConversationService) readConversationWorkspaceFilePreview(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) (ProjectConversationWorkspaceFilePreview, error) {
	if machine.Host != catalogdomain.LocalMachineHost {
		return s.readRemoteProjectConversationWorkspaceFilePreview(ctx, machine, repoRoot, relativePath)
	}
	return readLocalProjectConversationWorkspaceFilePreview(repoRoot, relativePath)
}

func (s *ProjectConversationService) readRemoteProjectConversationWorkspaceFilePreview(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) (ProjectConversationWorkspaceFilePreview, error) {
	resolvedFile, err := s.resolveRemoteProjectConversationWorkspaceFile(ctx, machine, repoRoot, relativePath)
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, err
	}
	command := fmt.Sprintf(
		"head -c %d %s",
		projectConversationWorkspacePreviewLimitBytes+1,
		projectConversationShellQuote(resolvedFile.path),
	)
	snippet, err := s.runProjectConversationShellCommand(ctx, machine, command, false)
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, fmt.Errorf("read remote workspace file %s: %w", relativePath, err)
	}
	revisionCommand := fmt.Sprintf(
		"sha256sum %s | awk '{print $1}'",
		projectConversationShellQuote(resolvedFile.path),
	)
	revisionOutput, err := s.runProjectConversationShellCommand(ctx, machine, revisionCommand, false)
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, fmt.Errorf("read remote workspace file revision %s: %w", relativePath, err)
	}
	preview := buildWorkspacePreviewFromContent(relativePath, snippet, resolvedFile.sizeBytes)
	preview.Revision = strings.TrimSpace(string(revisionOutput))
	return preview, nil
}

func readLocalProjectConversationWorkspaceFilePreview(
	repoRoot string,
	relativePath string,
) (ProjectConversationWorkspaceFilePreview, error) {
	resolvedFile, err := resolveLocalProjectConversationWorkspaceFile(repoRoot, relativePath)
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, err
	}
	content, _, err := readLocalWorkspaceFileContent(resolvedFile.path)
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, err
	}
	return buildWorkspacePreviewFromContent(relativePath, content, resolvedFile.sizeBytes), nil
}

type resolvedProjectConversationWorkspaceFile struct {
	path      string
	sizeBytes int64
}

func (s *ProjectConversationService) resolveRemoteProjectConversationWorkspaceFile(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) (resolvedProjectConversationWorkspaceFile, error) {
	command := fmt.Sprintf(`set -eu
repo=%s
relative=%s
repo_real=$(cd "$repo" && pwd -P)
base="$relative"
parent=""
if [ "$relative" != "$base" ]; then
  parent="${relative%%/*}"
fi
target_dir="$repo"
if [ -n "$parent" ]; then
  target_dir="$repo/$parent"
fi
dir_real=$(cd "$target_dir" 2>/dev/null && pwd -P) || { echo missing >&2; exit 11; }
case "$dir_real" in
  "$repo_real"|"$repo_real"/*) ;;
  *) echo escape >&2; exit 12 ;;
esac
target="$dir_real/$base"
if [ -L "$target" ]; then
  echo escape >&2
  exit 12
fi
if [ ! -f "$target" ]; then
  echo missing >&2
  exit 11
fi
size=$(wc -c <"$target" | tr -d '[:space:]')
printf '%%s\t%%s' "$size" "$target"
`, projectConversationShellQuote(repoRoot), projectConversationShellQuote(relativePath))
	output, err := s.runProjectConversationShellCommand(ctx, machine, command, false)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 11") {
			return resolvedProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspaceEntryNotFound
		}
		if strings.Contains(err.Error(), "exit status 12") {
			return resolvedProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspacePathInvalid
		}
		return resolvedProjectConversationWorkspaceFile{}, fmt.Errorf("resolve remote workspace file %s: %w", relativePath, err)
	}
	fields := strings.SplitN(strings.TrimSpace(string(output)), "\t", 2)
	if len(fields) != 2 {
		return resolvedProjectConversationWorkspaceFile{}, fmt.Errorf("parse remote workspace file descriptor %q", string(output))
	}
	var size int64
	if _, err := fmt.Sscanf(fields[0], "%d", &size); err != nil {
		return resolvedProjectConversationWorkspaceFile{}, fmt.Errorf("parse remote workspace file size %q: %w", fields[0], err)
	}
	return resolvedProjectConversationWorkspaceFile{path: fields[1], sizeBytes: size}, nil
}

func resolveLocalProjectConversationWorkspaceFile(repoRoot string, relativePath string) (resolvedProjectConversationWorkspaceFile, error) {
	repoRealPath, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return resolvedProjectConversationWorkspaceFile{}, fmt.Errorf("resolve workspace repo root: %w", err)
	}
	parent := filepath.Dir(relativePath)
	base := filepath.Base(relativePath)
	parentPath := repoRoot
	if parent != "." && parent != "" {
		parentPath = filepath.Join(repoRoot, filepath.FromSlash(parent))
	}
	parentRealPath, err := filepath.EvalSymlinks(parentPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return resolvedProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspaceEntryNotFound
		}
		return resolvedProjectConversationWorkspaceFile{}, fmt.Errorf("resolve workspace file parent %s: %w", parentPath, err)
	}
	if !projectConversationWorkspacePathWithinRoot(repoRealPath, parentRealPath) {
		return resolvedProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspacePathInvalid
	}
	targetPath := filepath.Join(parentRealPath, base)
	info, err := os.Lstat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return resolvedProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspaceEntryNotFound
		}
		return resolvedProjectConversationWorkspaceFile{}, fmt.Errorf("stat workspace file %s: %w", targetPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return resolvedProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspacePathInvalid
	}
	if !info.Mode().IsRegular() {
		return resolvedProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspaceEntryNotFound
	}
	return resolvedProjectConversationWorkspaceFile{path: targetPath, sizeBytes: info.Size()}, nil
}

func projectConversationWorkspaceMediaType(path string) string {
	mediaType := mime.TypeByExtension(filepath.Ext(path))
	if strings.TrimSpace(mediaType) == "" {
		return "text/plain"
	}
	return mediaType
}

func projectConversationWorkspaceLooksBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return true
	}
	return !utf8.Valid(content)
}

func (s *ProjectConversationService) readConversationWorkspaceFilePatch(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) (ProjectConversationWorkspaceFilePatch, error) {
	status, ok, err := s.readConversationWorkspaceFileStatus(ctx, machine, repoRoot, relativePath)
	if err != nil {
		return ProjectConversationWorkspaceFilePatch{}, err
	}
	if !ok {
		return ProjectConversationWorkspaceFilePatch{Status: ProjectConversationWorkspaceFileStatusModified, DiffKind: ProjectConversationWorkspaceDiffKindNone}, nil
	}

	var output []byte
	if status == ProjectConversationWorkspaceFileStatusUntracked {
		resolvedFile, err := s.resolveProjectConversationWorkspaceFile(ctx, machine, repoRoot, relativePath)
		if err != nil {
			return ProjectConversationWorkspaceFilePatch{}, err
		}
		output, err = s.runProjectConversationGitCommand(
			ctx,
			machine,
			[]string{"git", "-C", repoRoot, "diff", "--no-index", "--no-ext-diff", "--no-color", "--", "/dev/null", resolvedFile.path},
			true,
		)
		if err != nil {
			return ProjectConversationWorkspaceFilePatch{}, fmt.Errorf("read workspace untracked diff for %s: %w", relativePath, err)
		}
	} else {
		output, err = readProjectConversationWorkspaceGitPatch(ctx, repoRoot, relativePath, func(
			ctx context.Context,
			args []string,
			allowExitCodeOne bool,
		) ([]byte, error) {
			return s.runProjectConversationGitCommand(ctx, machine, args, allowExitCodeOne)
		})
		if err != nil {
			return ProjectConversationWorkspaceFilePatch{}, fmt.Errorf("read workspace patch for %s: %w", relativePath, err)
		}
	}

	patch := ProjectConversationWorkspaceFilePatch{Status: status}
	if len(output) == 0 {
		patch.DiffKind = ProjectConversationWorkspaceDiffKindNone
		return patch, nil
	}
	truncated := len(output) > projectConversationWorkspaceDiffLimitBytes
	if truncated {
		output = output[:projectConversationWorkspaceDiffLimitBytes]
	}
	if projectConversationWorkspaceLooksBinary(output) || bytes.Contains(output, []byte("Binary files")) {
		patch.DiffKind = ProjectConversationWorkspaceDiffKindBinary
		patch.Truncated = truncated
		return patch, nil
	}
	patch.DiffKind = ProjectConversationWorkspaceDiffKindText
	patch.Truncated = truncated
	patch.Diff = string(output)
	return patch, nil
}

func readProjectConversationWorkspaceGitPatch(
	ctx context.Context,
	repoPath string,
	relativePath string,
	run workspaceinfra.GitCommandRunner,
) ([]byte, error) {
	if run == nil {
		return nil, fmt.Errorf("git command runner is nil")
	}
	output, err := run(ctx, []string{"git", "-C", repoPath, "diff", "--no-ext-diff", "--no-color", "-M", "HEAD", "--", relativePath}, false)
	if err == nil {
		return output, nil
	}
	if projectConversationGitWorkspaceUnavailableOutput(output) {
		return nil, projectConversationWrapGitWorkspaceUnavailable(output)
	}
	if !projectConversationGitUnbornHeadOutput(output) {
		return output, err
	}
	return run(ctx, []string{"git", "-C", repoPath, "diff", "--no-ext-diff", "--no-color", "-M", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "--", relativePath}, false)
}

func (s *ProjectConversationService) resolveProjectConversationWorkspaceFile(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) (resolvedProjectConversationWorkspaceFile, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		return resolveLocalProjectConversationWorkspaceFile(repoRoot, relativePath)
	}
	return s.resolveRemoteProjectConversationWorkspaceFile(ctx, machine, repoRoot, relativePath)
}

func projectConversationWrapGitWorkspaceUnavailable(output []byte) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return workspaceinfra.ErrGitWorkspaceUnavailable
	}
	return fmt.Errorf("%w: %s", workspaceinfra.ErrGitWorkspaceUnavailable, message)
}

func projectConversationGitWorkspaceUnavailableOutput(output []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(trimmed, "not a git repository") ||
		strings.Contains(trimmed, "cannot change to") ||
		strings.Contains(trimmed, "no such file or directory")
}

func projectConversationGitUnbornHeadOutput(output []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(trimmed, "ambiguous argument 'head'") ||
		strings.Contains(trimmed, "bad revision 'head'") ||
		strings.Contains(trimmed, "unknown revision or path not in the working tree") ||
		strings.Contains(trimmed, "does not have any commits yet")
}

func (s *ProjectConversationService) readConversationWorkspaceFileStatus(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
) (ProjectConversationWorkspaceFileStatus, bool, error) {
	output, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoRoot, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", relativePath},
		false,
	)
	if err != nil {
		return "", false, fmt.Errorf("read workspace status for %s: %w", relativePath, err)
	}
	entries, err := parseProjectConversationGitStatusEntries(output)
	if err != nil {
		return "", false, fmt.Errorf("parse workspace status for %s: %w", relativePath, err)
	}
	if len(entries) == 0 {
		return "", false, nil
	}
	for _, entry := range entries {
		if entry.path == relativePath || entry.oldPath == relativePath {
			return mapProjectConversationWorkspaceFileStatus(entry.code), true, nil
		}
	}
	return mapProjectConversationWorkspaceFileStatus(entries[0].code), true, nil
}
