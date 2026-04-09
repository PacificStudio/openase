package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"
)

const projectConversationWorkspacePreviewMaxBytes int64 = 64 * 1024

const (
	projectConversationWorkspaceCommandInvalidPath = 41
	projectConversationWorkspaceCommandNotFound    = 42
	projectConversationWorkspaceCommandNotDir      = 43
	projectConversationWorkspaceCommandNotFile     = 44
)

var (
	ErrProjectConversationWorkspaceUnavailable   = errors.New("project conversation workspace unavailable")
	ErrProjectConversationWorkspacePathNotFound  = errors.New("project conversation workspace path not found")
	ErrProjectConversationWorkspaceInvalidTarget = errors.New("project conversation workspace target is invalid")
)

type ProjectConversationWorkspaceMetadata struct {
	ConversationID uuid.UUID
	Available      bool
	WorkspacePath  string
	MachineName    string
	MachineHost    string
	Repos          []ProjectConversationWorkspaceRepoMetadata
}

type ProjectConversationWorkspaceRepoMetadata struct {
	Name      string
	Path      string
	Available bool
	Branch    string
}

type ProjectConversationWorkspaceTree struct {
	ConversationID uuid.UUID
	RepoName       string
	Path           string
	Entries        []ProjectConversationWorkspaceTreeEntry
}

type ProjectConversationWorkspaceTreeEntry struct {
	Name      string
	Path      string
	Kind      ProjectConversationWorkspaceTreeEntryKind
	SizeBytes int64
}

type ProjectConversationWorkspaceTreeEntryKind string

const (
	ProjectConversationWorkspaceTreeEntryKindDirectory ProjectConversationWorkspaceTreeEntryKind = "directory"
	ProjectConversationWorkspaceTreeEntryKindFile      ProjectConversationWorkspaceTreeEntryKind = "file"
)

type ProjectConversationWorkspaceFile struct {
	ConversationID    uuid.UUID
	RepoName          string
	Path              string
	SizeBytes         int64
	ContentType       string
	PreviewStatus     ProjectConversationWorkspaceFilePreviewStatus
	PreviewEncoding   string
	PreviewContent    string
	UnavailableReason ProjectConversationWorkspaceFileUnavailableReason
}

type ProjectConversationWorkspaceFilePreviewStatus string

const (
	ProjectConversationWorkspaceFilePreviewStatusAvailable   ProjectConversationWorkspaceFilePreviewStatus = "available"
	ProjectConversationWorkspaceFilePreviewStatusUnavailable ProjectConversationWorkspaceFilePreviewStatus = "unavailable"
)

type ProjectConversationWorkspaceFileUnavailableReason string

const (
	ProjectConversationWorkspaceFileUnavailableReasonNonText  ProjectConversationWorkspaceFileUnavailableReason = "non_text"
	ProjectConversationWorkspaceFileUnavailableReasonTooLarge ProjectConversationWorkspaceFileUnavailableReason = "too_large"
)

type projectConversationWorkspaceInventory struct {
	conversation  chatdomain.Conversation
	machine       catalogdomain.Machine
	workspacePath string
	available     bool
	repos         []projectConversationWorkspaceRepoLocation
}

func (s *ProjectConversationService) GetWorkspaceMetadata(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (ProjectConversationWorkspaceMetadata, error) {
	inventory, err := s.loadConversationWorkspaceInventory(ctx, userID, conversationID)
	if err != nil {
		return ProjectConversationWorkspaceMetadata{}, err
	}

	metadata := ProjectConversationWorkspaceMetadata{
		ConversationID: conversationID,
		Available:      inventory.available,
		WorkspacePath:  inventory.workspacePath,
		MachineName:    inventory.machine.Name,
		MachineHost:    inventory.machine.Host,
		Repos:          make([]ProjectConversationWorkspaceRepoMetadata, 0, len(inventory.repos)),
	}
	for _, repo := range inventory.repos {
		item := ProjectConversationWorkspaceRepoMetadata{
			Name: repo.name,
			Path: repo.relativePath,
		}
		if inventory.available {
			branch, err := workspaceinfra.ReadWorkspaceGitBranch(ctx, repo.repoPath, func(
				ctx context.Context,
				args []string,
				allowExitCodeOne bool,
			) ([]byte, error) {
				return s.runProjectConversationGitCommand(ctx, inventory.machine, args, allowExitCodeOne)
			})
			switch {
			case err == nil:
				item.Available = true
				item.Branch = branch
			case errors.Is(err, workspaceinfra.ErrGitWorkspaceUnavailable):
				// Keep the repo listed so callers can still render the configured
				// workspace layout while the clone or repo metadata is unavailable.
			default:
				return ProjectConversationWorkspaceMetadata{}, fmt.Errorf("read workspace branch for %s: %w", repo.name, err)
			}
		}
		metadata.Repos = append(metadata.Repos, item)
	}
	return metadata, nil
}

func (s *ProjectConversationService) ListWorkspaceTree(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input chatdomain.WorkspaceTreeInput,
) (ProjectConversationWorkspaceTree, error) {
	inventory, err := s.loadConversationWorkspaceInventory(ctx, userID, conversationID)
	if err != nil {
		return ProjectConversationWorkspaceTree{}, err
	}
	if !inventory.available {
		return ProjectConversationWorkspaceTree{}, ErrProjectConversationWorkspaceUnavailable
	}

	repo, err := inventory.repo(input.RepoName)
	if err != nil {
		return ProjectConversationWorkspaceTree{}, err
	}

	var entries []ProjectConversationWorkspaceTreeEntry
	if inventory.machine.Host == catalogdomain.LocalMachineHost {
		entries, err = listLocalConversationWorkspaceTree(repo.repoPath, input.Path)
	} else {
		entries, err = s.listRemoteConversationWorkspaceTree(ctx, inventory.machine, repo.repoPath, input.Path)
	}
	if err != nil {
		return ProjectConversationWorkspaceTree{}, err
	}

	return ProjectConversationWorkspaceTree{
		ConversationID: conversationID,
		RepoName:       repo.name,
		Path:           input.Path.String(),
		Entries:        entries,
	}, nil
}

func (s *ProjectConversationService) ReadWorkspaceFile(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input chatdomain.WorkspaceFileInput,
) (ProjectConversationWorkspaceFile, error) {
	inventory, err := s.loadConversationWorkspaceInventory(ctx, userID, conversationID)
	if err != nil {
		return ProjectConversationWorkspaceFile{}, err
	}
	if !inventory.available {
		return ProjectConversationWorkspaceFile{}, ErrProjectConversationWorkspaceUnavailable
	}

	repo, err := inventory.repo(input.RepoName)
	if err != nil {
		return ProjectConversationWorkspaceFile{}, err
	}

	var file ProjectConversationWorkspaceFile
	if inventory.machine.Host == catalogdomain.LocalMachineHost {
		file, err = readLocalConversationWorkspaceFile(repo.repoPath, input.Path)
	} else {
		file, err = s.readRemoteConversationWorkspaceFile(ctx, inventory.machine, repo.repoPath, input.Path)
	}
	if err != nil {
		return ProjectConversationWorkspaceFile{}, err
	}

	file.ConversationID = conversationID
	file.RepoName = repo.name
	file.Path = input.Path.String()
	return file, nil
}

func (s *ProjectConversationService) loadConversationWorkspaceInventory(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (projectConversationWorkspaceInventory, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return projectConversationWorkspaceInventory{}, err
	}

	project, err := s.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return projectConversationWorkspaceInventory{}, fmt.Errorf("get project for workspace browser: %w", err)
	}
	providerItem, err := s.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return projectConversationWorkspaceInventory{}, fmt.Errorf("get provider for workspace browser: %w", err)
	}
	machine, err := s.catalog.GetMachine(ctx, providerItem.MachineID)
	if err != nil {
		return projectConversationWorkspaceInventory{}, fmt.Errorf("get machine for workspace browser: %w", err)
	}
	projectRepos, err := s.catalog.ListProjectRepos(ctx, project.ID)
	if err != nil {
		return projectConversationWorkspaceInventory{}, fmt.Errorf("list project repos for workspace browser: %w", err)
	}

	workspacePath, err := s.resolveConversationWorkspacePath(machine, project, conversationID)
	available := true
	switch {
	case err == nil:
	case errors.Is(err, errProjectConversationWorkspaceLocationUnavailable) &&
		projectConversationWorkspaceMayNotExistYet(conversation):
		available = false
	default:
		return projectConversationWorkspaceInventory{}, err
	}

	repos := buildProjectConversationWorkspaceRepoLocations(workspacePath, projectRepos)
	return projectConversationWorkspaceInventory{
		conversation:  conversation,
		machine:       machine,
		workspacePath: workspacePath,
		available:     available,
		repos:         repos,
	}, nil
}

func buildProjectConversationWorkspaceRepoLocations(
	workspacePath string,
	projectRepos []catalogdomain.ProjectRepo,
) []projectConversationWorkspaceRepoLocation {
	repos := make([]projectConversationWorkspaceRepoLocation, 0, len(projectRepos))
	for _, repo := range projectRepos {
		relativePath := repo.Name
		if strings.TrimSpace(repo.WorkspaceDirname) != "" {
			relativePath = path.Join(repo.WorkspaceDirname, repo.Name)
		}

		repoPath := ""
		if strings.TrimSpace(workspacePath) != "" {
			repoPath = workspaceinfra.RepoPath(workspacePath, repo.WorkspaceDirname, repo.Name)
		}
		repos = append(repos, projectConversationWorkspaceRepoLocation{
			name:         repo.Name,
			repoPath:     repoPath,
			relativePath: relativePath,
		})
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].relativePath < repos[j].relativePath
	})
	return repos
}

func (i projectConversationWorkspaceInventory) repo(
	name chatdomain.WorkspaceRepoName,
) (projectConversationWorkspaceRepoLocation, error) {
	for _, repo := range i.repos {
		if repo.name == name.String() {
			return repo, nil
		}
	}
	return projectConversationWorkspaceRepoLocation{}, fmt.Errorf(
		"%w: repo %q is not part of the conversation workspace",
		chatdomain.ErrInvalidInput,
		name.String(),
	)
}

func listLocalConversationWorkspaceTree(
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) ([]ProjectConversationWorkspaceTreeEntry, error) {
	root, err := os.OpenRoot(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: repo %q is unavailable", ErrProjectConversationWorkspacePathNotFound, repoPath)
		}
		return nil, fmt.Errorf("open workspace repo %s: %w", repoPath, err)
	}
	defer func() { _ = root.Close() }()

	targetName, err := resolveLocalWorkspaceTarget(root, repoRelativePath)
	if err != nil {
		return nil, err
	}
	info, err := root.Stat(targetName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: path %q does not exist", ErrProjectConversationWorkspacePathNotFound, repoRelativePath.String())
		}
		return nil, fmt.Errorf("stat workspace path %q: %w", repoRelativePath.String(), err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: path %q is not a directory", ErrProjectConversationWorkspaceInvalidTarget, repoRelativePath.String())
	}

	readPath := "."
	if !repoRelativePath.IsRoot() {
		readPath = repoRelativePath.String()
	}
	entries, err := fs.ReadDir(root.FS(), readPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: path %q does not exist", ErrProjectConversationWorkspacePathNotFound, repoRelativePath.String())
		}
		return nil, fmt.Errorf("read workspace directory %q: %w", repoRelativePath.String(), err)
	}

	items := make([]ProjectConversationWorkspaceTreeEntry, 0, len(entries))
	for _, entry := range entries {
		childPath := entry.Name()
		if !repoRelativePath.IsRoot() {
			childPath = path.Join(repoRelativePath.String(), entry.Name())
		}
		info, err := root.Lstat(childPath)
		if err != nil {
			return nil, fmt.Errorf("stat workspace entry %q: %w", childPath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || entry.Name() == ".git" {
			continue
		}

		item := ProjectConversationWorkspaceTreeEntry{
			Name: entry.Name(),
			Path: childPath,
		}
		switch {
		case info.IsDir():
			item.Kind = ProjectConversationWorkspaceTreeEntryKindDirectory
		case info.Mode().IsRegular():
			item.Kind = ProjectConversationWorkspaceTreeEntryKindFile
			item.SizeBytes = info.Size()
		default:
			continue
		}
		items = append(items, item)
	}
	sortWorkspaceTreeEntries(items)
	return items, nil
}

func readLocalConversationWorkspaceFile(
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) (ProjectConversationWorkspaceFile, error) {
	root, err := os.OpenRoot(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectConversationWorkspaceFile{}, fmt.Errorf("%w: repo %q is unavailable", ErrProjectConversationWorkspacePathNotFound, repoPath)
		}
		return ProjectConversationWorkspaceFile{}, fmt.Errorf("open workspace repo %s: %w", repoPath, err)
	}
	defer func() { _ = root.Close() }()

	targetName, err := resolveLocalWorkspaceTarget(root, repoRelativePath)
	if err != nil {
		return ProjectConversationWorkspaceFile{}, err
	}
	info, err := root.Stat(targetName)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectConversationWorkspaceFile{}, fmt.Errorf("%w: path %q does not exist", ErrProjectConversationWorkspacePathNotFound, repoRelativePath.String())
		}
		return ProjectConversationWorkspaceFile{}, fmt.Errorf("stat workspace file %q: %w", repoRelativePath.String(), err)
	}
	if !info.Mode().IsRegular() {
		return ProjectConversationWorkspaceFile{}, fmt.Errorf("%w: path %q is not a file", ErrProjectConversationWorkspaceInvalidTarget, repoRelativePath.String())
	}

	file := ProjectConversationWorkspaceFile{SizeBytes: info.Size()}
	if info.Size() > projectConversationWorkspacePreviewMaxBytes {
		file.PreviewStatus = ProjectConversationWorkspaceFilePreviewStatusUnavailable
		file.UnavailableReason = ProjectConversationWorkspaceFileUnavailableReasonTooLarge
		return file, nil
	}

	content, err := root.ReadFile(targetName)
	if err != nil {
		return ProjectConversationWorkspaceFile{}, fmt.Errorf("read workspace file %q: %w", repoRelativePath.String(), err)
	}
	file.ContentType = detectWorkspaceFileContentType(content)
	if !workspaceFilePreviewIsText(content) {
		file.PreviewStatus = ProjectConversationWorkspaceFilePreviewStatusUnavailable
		file.UnavailableReason = ProjectConversationWorkspaceFileUnavailableReasonNonText
		return file, nil
	}
	file.PreviewStatus = ProjectConversationWorkspaceFilePreviewStatusAvailable
	file.PreviewEncoding = "utf8"
	file.PreviewContent = string(content)
	return file, nil
}

func resolveLocalWorkspaceTarget(
	root *os.Root,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) (string, error) {
	current := "."
	segments := repoRelativePath.Segments()
	for index, segment := range segments {
		next := path.Join(current, segment)
		info, err := root.Lstat(next)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("%w: path %q does not exist", ErrProjectConversationWorkspacePathNotFound, repoRelativePath.String())
			}
			return "", fmt.Errorf("stat workspace path %q: %w", repoRelativePath.String(), err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: symlinks are not supported for workspace browsing", chatdomain.ErrInvalidInput)
		}
		if index < len(segments)-1 && !info.IsDir() {
			return "", fmt.Errorf("%w: path %q is not a directory", ErrProjectConversationWorkspaceInvalidTarget, repoRelativePath.String())
		}
		current = next
	}
	return current, nil
}

func (s *ProjectConversationService) listRemoteConversationWorkspaceTree(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) ([]ProjectConversationWorkspaceTreeEntry, error) {
	command := buildRemoteWorkspaceTreeCommand(repoPath, repoRelativePath)
	output, exitCode, err := s.runProjectConversationShellCommand(ctx, machine, command)
	if err != nil {
		return nil, mapProjectConversationWorkspaceRemoteError(repoRelativePath.String(), exitCode, err)
	}
	entries, err := parseRemoteWorkspaceTreeEntries(output, repoRelativePath)
	if err != nil {
		return nil, fmt.Errorf("parse remote workspace tree: %w", err)
	}
	sortWorkspaceTreeEntries(entries)
	return entries, nil
}

func (s *ProjectConversationService) readRemoteConversationWorkspaceFile(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) (ProjectConversationWorkspaceFile, error) {
	command := buildRemoteWorkspaceFileCommand(repoPath, repoRelativePath)
	output, exitCode, err := s.runProjectConversationShellCommand(ctx, machine, command)
	if err != nil {
		return ProjectConversationWorkspaceFile{}, mapProjectConversationWorkspaceRemoteError(repoRelativePath.String(), exitCode, err)
	}

	newline := bytes.IndexByte(output, '\n')
	if newline < 0 {
		return ProjectConversationWorkspaceFile{}, fmt.Errorf("remote workspace file response is malformed")
	}
	sizeBytes, err := strconv.ParseInt(strings.TrimSpace(string(output[:newline])), 10, 64)
	if err != nil {
		return ProjectConversationWorkspaceFile{}, fmt.Errorf("parse remote workspace file size: %w", err)
	}
	content := output[newline+1:]

	file := ProjectConversationWorkspaceFile{SizeBytes: sizeBytes}
	if sizeBytes > projectConversationWorkspacePreviewMaxBytes {
		file.PreviewStatus = ProjectConversationWorkspaceFilePreviewStatusUnavailable
		file.UnavailableReason = ProjectConversationWorkspaceFileUnavailableReasonTooLarge
		return file, nil
	}

	file.ContentType = detectWorkspaceFileContentType(content)
	if !workspaceFilePreviewIsText(content) {
		file.PreviewStatus = ProjectConversationWorkspaceFilePreviewStatusUnavailable
		file.UnavailableReason = ProjectConversationWorkspaceFileUnavailableReasonNonText
		return file, nil
	}
	file.PreviewStatus = ProjectConversationWorkspaceFilePreviewStatusAvailable
	file.PreviewEncoding = "utf8"
	file.PreviewContent = string(content)
	return file, nil
}

func buildRemoteWorkspaceTreeCommand(
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) string {
	targetPath := projectConversationWorkspaceTargetPath(repoPath, repoRelativePath)
	validationLines := buildRemoteWorkspacePathValidationLines(repoPath, repoRelativePath)
	lines := make([]string, 0, len(validationLines)+4)
	lines = append(lines, "set -eu")
	lines = append(lines, validationLines...)
	lines = append(lines,
		"if [ ! -e "+sshinfra.ShellQuote(targetPath)+" ]; then exit "+strconv.Itoa(projectConversationWorkspaceCommandNotFound)+"; fi",
		"if [ ! -d "+sshinfra.ShellQuote(targetPath)+" ]; then exit "+strconv.Itoa(projectConversationWorkspaceCommandNotDir)+"; fi",
		"find "+sshinfra.ShellQuote(targetPath)+" -mindepth 1 -maxdepth 1 -exec sh -c "+sshinfra.ShellQuote(remoteWorkspaceTreeEntryScript())+" sh {} +",
	)
	return strings.Join(lines, "\n")
}

func remoteWorkspaceTreeEntryScript() string {
	return `for p do
  name=${p##*/}
  if [ "$name" = ".git" ] || [ -L "$p" ]; then
    continue
  fi
  if [ -d "$p" ]; then
    printf 'directory\0%s\0%s\0' "$name" 0
  elif [ -f "$p" ]; then
    size=$(wc -c < "$p" | tr -d '[:space:]')
    printf 'file\0%s\0%s\0' "$name" "$size"
  fi
done`
}

func buildRemoteWorkspaceFileCommand(
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) string {
	targetPath := projectConversationWorkspaceTargetPath(repoPath, repoRelativePath)
	validationLines := buildRemoteWorkspacePathValidationLines(repoPath, repoRelativePath)
	lines := make([]string, 0, len(validationLines)+6)
	lines = append(lines, "set -eu")
	lines = append(lines, validationLines...)
	lines = append(lines,
		"if [ ! -e "+sshinfra.ShellQuote(targetPath)+" ]; then exit "+strconv.Itoa(projectConversationWorkspaceCommandNotFound)+"; fi",
		"if [ ! -f "+sshinfra.ShellQuote(targetPath)+" ]; then exit "+strconv.Itoa(projectConversationWorkspaceCommandNotFile)+"; fi",
		"size=$(wc -c < "+sshinfra.ShellQuote(targetPath)+" | tr -d '[:space:]')",
		"printf '%s\\n' \"$size\"",
		"if [ \"$size\" -le "+strconv.FormatInt(projectConversationWorkspacePreviewMaxBytes, 10)+" ]; then cat "+sshinfra.ShellQuote(targetPath)+"; fi",
	)
	return strings.Join(lines, "\n")
}

func buildRemoteWorkspacePathValidationLines(
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) []string {
	lines := make([]string, 0, len(repoRelativePath.Segments())+1)
	lines = append(lines, "if [ -L "+sshinfra.ShellQuote(repoPath)+" ]; then exit "+strconv.Itoa(projectConversationWorkspaceCommandInvalidPath)+"; fi")
	current := repoPath
	for _, segment := range repoRelativePath.Segments() {
		current = path.Join(current, segment)
		lines = append(lines, "if [ -L "+sshinfra.ShellQuote(current)+" ]; then exit "+strconv.Itoa(projectConversationWorkspaceCommandInvalidPath)+"; fi")
	}
	return lines
}

func projectConversationWorkspaceTargetPath(
	repoPath string,
	repoRelativePath chatdomain.WorkspaceRelativePath,
) string {
	if repoRelativePath.IsRoot() {
		return repoPath
	}
	return path.Join(repoPath, repoRelativePath.String())
}

func parseRemoteWorkspaceTreeEntries(
	raw []byte,
	basePath chatdomain.WorkspaceRelativePath,
) ([]ProjectConversationWorkspaceTreeEntry, error) {
	parts := bytes.Split(raw, []byte{0})
	entries := make([]ProjectConversationWorkspaceTreeEntry, 0, len(parts)/3)
	for index := 0; index < len(parts); {
		if len(parts[index]) == 0 {
			index++
			continue
		}
		if index+2 >= len(parts) {
			return nil, fmt.Errorf("tree entry %d is truncated", index)
		}

		kindRaw := string(parts[index])
		name := string(parts[index+1])
		sizeRaw := string(parts[index+2])
		index += 3

		sizeBytes, err := strconv.ParseInt(sizeRaw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse size for %q: %w", name, err)
		}

		entry := ProjectConversationWorkspaceTreeEntry{
			Name:      name,
			Path:      name,
			SizeBytes: sizeBytes,
		}
		if !basePath.IsRoot() {
			entry.Path = path.Join(basePath.String(), name)
		}
		switch kindRaw {
		case string(ProjectConversationWorkspaceTreeEntryKindDirectory):
			entry.Kind = ProjectConversationWorkspaceTreeEntryKindDirectory
			entry.SizeBytes = 0
		case string(ProjectConversationWorkspaceTreeEntryKindFile):
			entry.Kind = ProjectConversationWorkspaceTreeEntryKindFile
		default:
			return nil, fmt.Errorf("unsupported workspace tree entry kind %q", kindRaw)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func sortWorkspaceTreeEntries(entries []ProjectConversationWorkspaceTreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Path < entries[j].Path
	})
}

func detectWorkspaceFileContentType(content []byte) string {
	if len(content) == 0 {
		return "text/plain; charset=utf-8"
	}
	return http.DetectContentType(content)
}

func workspaceFilePreviewIsText(content []byte) bool {
	if bytes.IndexByte(content, 0) >= 0 {
		return false
	}
	return utf8.Valid(content)
}

func (s *ProjectConversationService) runProjectConversationShellCommand(
	ctx context.Context,
	machine catalogdomain.Machine,
	script string,
) ([]byte, int, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		command := exec.CommandContext(ctx, "sh", "-lc", script) // #nosec G204 -- script is generated by this package from validated repo-relative inputs and shell-quoted paths.
		output, err := command.CombinedOutput()
		if err == nil {
			return output, 0, nil
		}
		return output, projectConversationLocalExitCode(err), fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	if s == nil || s.sshPool == nil {
		return nil, 0, fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
	}

	client, err := s.sshPool.Get(ctx, machine)
	if err != nil {
		return nil, 0, err
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, 0, fmt.Errorf("open ssh session for workspace browser: %w", err)
	}
	defer func() { _ = session.Close() }()

	output, err := session.CombinedOutput("sh -lc " + sshinfra.ShellQuote(script))
	if err == nil {
		return output, 0, nil
	}
	return output, projectConversationRemoteExitCode(err), fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
}

func projectConversationLocalExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 0
}

func projectConversationRemoteExitCode(err error) int {
	var exitErr *gossh.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitStatus()
	}
	return 0
}

func mapProjectConversationWorkspaceRemoteError(path string, exitCode int, err error) error {
	switch exitCode {
	case projectConversationWorkspaceCommandInvalidPath:
		return fmt.Errorf("%w: symlinks are not supported for workspace browsing", chatdomain.ErrInvalidInput)
	case projectConversationWorkspaceCommandNotFound:
		return fmt.Errorf("%w: path %q does not exist", ErrProjectConversationWorkspacePathNotFound, path)
	case projectConversationWorkspaceCommandNotDir:
		return fmt.Errorf("%w: path %q is not a directory", ErrProjectConversationWorkspaceInvalidTarget, path)
	case projectConversationWorkspaceCommandNotFile:
		return fmt.Errorf("%w: path %q is not a file", ErrProjectConversationWorkspaceInvalidTarget, path)
	default:
		return err
	}
}
