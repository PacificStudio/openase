package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

type ProjectConversationWorkspaceDiff struct {
	ConversationID uuid.UUID
	WorkspacePath  string
	Dirty          bool
	ReposChanged   int
	FilesChanged   int
	Added          int
	Removed        int
	Repos          []ProjectConversationWorkspaceRepoDiff
	SyncPrompt     *ProjectConversationWorkspaceSyncPrompt
}

type ProjectConversationWorkspaceRepoDiff struct {
	Name         string
	Path         string
	Branch       string
	Dirty        bool
	FilesChanged int
	Added        int
	Removed      int
	Files        []ProjectConversationWorkspaceFileDiff
}

type ProjectConversationWorkspaceFileDiff struct {
	Path    string
	Status  ProjectConversationWorkspaceFileStatus
	Added   int
	Removed int
}

type ProjectConversationWorkspaceFileStatus string

const (
	ProjectConversationWorkspaceFileStatusModified  ProjectConversationWorkspaceFileStatus = "modified"
	ProjectConversationWorkspaceFileStatusAdded     ProjectConversationWorkspaceFileStatus = "added"
	ProjectConversationWorkspaceFileStatusDeleted   ProjectConversationWorkspaceFileStatus = "deleted"
	ProjectConversationWorkspaceFileStatusRenamed   ProjectConversationWorkspaceFileStatus = "renamed"
	ProjectConversationWorkspaceFileStatusUntracked ProjectConversationWorkspaceFileStatus = "untracked"
)

var errProjectConversationWorkspaceLocationUnavailable = errors.New("project conversation workspace location unavailable")

type projectConversationWorkspaceLocation struct {
	machine       catalogdomain.Machine
	workspacePath string
	repos         []projectConversationWorkspaceRepoLocation
	syncPrompt    *ProjectConversationWorkspaceSyncPrompt
	missingRepos  map[string]ProjectConversationWorkspaceMissingRepo
}

type projectConversationWorkspaceRepoLocation struct {
	name         string
	repoPath     string
	relativePath string
}

type projectConversationGitStatusEntry struct {
	code    string
	path    string
	oldPath string
}

type projectConversationGitNumstat struct {
	path    string
	added   int
	removed int
}

func (s *ProjectConversationService) GetWorkspaceDiff(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (ProjectConversationWorkspaceDiff, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return ProjectConversationWorkspaceDiff{}, err
	}

	project, err := s.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return ProjectConversationWorkspaceDiff{}, fmt.Errorf("get project for workspace diff: %w", err)
	}
	providerItem, err := s.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return ProjectConversationWorkspaceDiff{}, fmt.Errorf("get provider for workspace diff: %w", err)
	}

	location, err := s.resolveConversationWorkspaceLocation(ctx, conversation, project, providerItem)
	if err != nil {
		if errors.Is(err, errProjectConversationWorkspaceLocationUnavailable) &&
			projectConversationWorkspaceMayNotExistYet(conversation) {
			return ProjectConversationWorkspaceDiff{ConversationID: conversationID}, nil
		}
		return ProjectConversationWorkspaceDiff{}, err
	}

	summary := ProjectConversationWorkspaceDiff{
		ConversationID: conversationID,
		WorkspacePath:  location.workspacePath,
		Repos:          make([]ProjectConversationWorkspaceRepoDiff, 0, len(location.repos)),
		SyncPrompt:     location.syncPrompt,
	}

	for _, repo := range location.repos {
		repoSummary, err := s.summarizeConversationWorkspaceRepo(ctx, location.machine, repo)
		if err != nil {
			return ProjectConversationWorkspaceDiff{}, err
		}
		if !repoSummary.Dirty {
			continue
		}
		summary.Dirty = true
		summary.ReposChanged++
		summary.FilesChanged += repoSummary.FilesChanged
		summary.Added += repoSummary.Added
		summary.Removed += repoSummary.Removed
		summary.Repos = append(summary.Repos, repoSummary)
	}

	sort.Slice(summary.Repos, func(i, j int) bool {
		return summary.Repos[i].Path < summary.Repos[j].Path
	})
	return summary, nil
}

func (s *ProjectConversationService) resolveConversationWorkspaceLocation(
	ctx context.Context,
	conversation chatdomain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
) (projectConversationWorkspaceLocation, error) {
	machine, err := s.catalog.GetMachine(ctx, providerItem.MachineID)
	if err != nil {
		return projectConversationWorkspaceLocation{}, fmt.Errorf("get chat provider machine for workspace diff: %w", err)
	}

	workspacePath, err := s.resolveConversationWorkspacePath(machine, project, conversation.ID)
	if err != nil {
		return projectConversationWorkspaceLocation{}, err
	}

	projectRepos, err := s.catalog.ListProjectRepos(ctx, project.ID)
	if err != nil {
		return projectConversationWorkspaceLocation{}, fmt.Errorf("list project repos for workspace diff: %w", err)
	}

	repos, syncPrompt, missingRepos, err := s.buildConversationWorkspaceRepoLocations(
		ctx,
		conversation,
		project,
		machine,
		workspacePath,
		projectRepos,
	)
	if err != nil {
		return projectConversationWorkspaceLocation{}, err
	}

	return projectConversationWorkspaceLocation{
		machine:       machine,
		workspacePath: workspacePath,
		repos:         repos,
		syncPrompt:    syncPrompt,
		missingRepos:  missingRepos,
	}, nil
}

func (s *ProjectConversationService) resolveConversationWorkspacePath(
	machine catalogdomain.Machine,
	project catalogdomain.Project,
	conversationID uuid.UUID,
) (string, error) {
	if workspacePath, ok := s.runtimeManager.WorkspacePath(conversationID); ok {
		return filepath.Clean(workspacePath.String()), nil
	}

	root, err := resolveProjectConversationWorkspaceRoot(machine)
	if err != nil {
		return "", err
	}

	workspacePath, err := workspaceinfra.TicketWorkspacePath(
		root,
		project.OrganizationID.String(),
		project.Slug,
		projectConversationWorkspaceName(conversationID),
	)
	if err != nil {
		return "", fmt.Errorf("derive project conversation workspace path: %w", err)
	}
	return workspacePath, nil
}

func (s *ProjectConversationService) summarizeConversationWorkspaceRepo(
	ctx context.Context,
	machine catalogdomain.Machine,
	repo projectConversationWorkspaceRepoLocation,
) (ProjectConversationWorkspaceRepoDiff, error) {
	branch, err := workspaceinfra.ReadWorkspaceGitBranch(ctx, repo.repoPath, func(
		ctx context.Context,
		args []string,
		allowExitCodeOne bool,
	) ([]byte, error) {
		return s.runProjectConversationGitCommand(ctx, machine, args, allowExitCodeOne)
	})
	if err != nil {
		if errors.Is(err, workspaceinfra.ErrGitWorkspaceUnavailable) {
			return ProjectConversationWorkspaceRepoDiff{}, nil
		}
		return ProjectConversationWorkspaceRepoDiff{}, fmt.Errorf("read workspace branch for %s: %w", repo.name, err)
	}
	statusOutput, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repo.repoPath, "status", "--porcelain=v1", "-z", "--untracked-files=all"},
		false,
	)
	if err != nil {
		return ProjectConversationWorkspaceRepoDiff{}, fmt.Errorf("read workspace git status for %s: %w", repo.name, err)
	}
	statuses, err := parseProjectConversationGitStatusEntries(statusOutput)
	if err != nil {
		return ProjectConversationWorkspaceRepoDiff{}, fmt.Errorf("parse workspace git status for %s: %w", repo.name, err)
	}
	if len(statuses) == 0 {
		return ProjectConversationWorkspaceRepoDiff{}, nil
	}

	numstatOutput, err := workspaceinfra.ReadWorkspaceGitNumstat(ctx, repo.repoPath, func(
		ctx context.Context,
		args []string,
		allowExitCodeOne bool,
	) ([]byte, error) {
		return s.runProjectConversationGitCommand(ctx, machine, args, allowExitCodeOne)
	})
	if err != nil {
		return ProjectConversationWorkspaceRepoDiff{}, fmt.Errorf("read workspace diff stats for %s: %w", repo.name, err)
	}
	numstats, err := parseProjectConversationGitNumstat(numstatOutput)
	if err != nil {
		return ProjectConversationWorkspaceRepoDiff{}, fmt.Errorf("parse workspace diff stats for %s: %w", repo.name, err)
	}
	fileStats := make(map[string]projectConversationGitNumstat, len(numstats))
	for _, item := range numstats {
		fileStats[item.path] = item
	}

	files := make([]ProjectConversationWorkspaceFileDiff, 0, len(statuses))
	repoSummary := ProjectConversationWorkspaceRepoDiff{
		Name:   repo.name,
		Path:   repo.relativePath,
		Branch: branch,
		Dirty:  true,
	}
	for _, status := range statuses {
		stat := fileStats[status.path]
		if status.code == "??" {
			stat, err = s.readProjectConversationUntrackedNumstat(ctx, machine, repo.repoPath, status.path)
			if err != nil {
				return ProjectConversationWorkspaceRepoDiff{}, fmt.Errorf("read workspace untracked stats for %s: %w", status.path, err)
			}
		}

		file := ProjectConversationWorkspaceFileDiff{
			Path:    status.path,
			Status:  mapProjectConversationWorkspaceFileStatus(status.code),
			Added:   stat.added,
			Removed: stat.removed,
		}
		files = append(files, file)
		repoSummary.Added += file.Added
		repoSummary.Removed += file.Removed
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	repoSummary.Files = files
	repoSummary.FilesChanged = len(files)
	return repoSummary, nil
}

func (s *ProjectConversationService) readProjectConversationUntrackedNumstat(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	filePath string,
) (projectConversationGitNumstat, error) {
	output, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "diff", "--no-index", "--numstat", "-z", "/dev/null", filePath},
		true,
	)
	if err != nil {
		return projectConversationGitNumstat{}, err
	}
	stats, err := parseProjectConversationGitNumstat(output)
	if err != nil {
		return projectConversationGitNumstat{}, err
	}
	if len(stats) == 0 {
		return projectConversationGitNumstat{path: filePath}, nil
	}
	return projectConversationGitNumstat{
		path:    filePath,
		added:   stats[0].added,
		removed: stats[0].removed,
	}, nil
}

func (s *ProjectConversationService) runProjectConversationGitCommand(
	ctx context.Context,
	machine catalogdomain.Machine,
	args []string,
	allowExitCodeOne bool,
) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("git command is empty")
	}
	if machine.Host == catalogdomain.LocalMachineHost {
		// #nosec G204 -- git is a fixed executable and the arguments are assembled by the service.
		command := exec.CommandContext(ctx, args[0], args[1:]...)
		output, err := command.CombinedOutput()
		if err != nil && (!allowExitCodeOne || !projectConversationCommandExitedWithCode(err, 1)) {
			return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return output, nil
	}
	if s == nil || s.sshPool == nil {
		return nil, fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
	}

	client, err := s.sshPool.Get(ctx, machine)
	if err != nil {
		return nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("open ssh session for workspace diff: %w", err)
	}
	defer func() { _ = session.Close() }()

	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, sshinfra.ShellQuote(arg))
	}
	output, err := session.CombinedOutput("sh -lc " + sshinfra.ShellQuote(strings.Join(quoted, " ")))
	if err != nil && (!allowExitCodeOne || !strings.Contains(err.Error(), "exit status 1")) {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func parseProjectConversationGitStatusEntries(raw []byte) ([]projectConversationGitStatusEntry, error) {
	parts := bytes.Split(raw, []byte{0})
	entries := make([]projectConversationGitStatusEntry, 0, len(parts))
	for index := 0; index < len(parts); index++ {
		entry := parts[index]
		if len(entry) == 0 {
			continue
		}
		if len(entry) < 4 {
			return nil, fmt.Errorf("status entry %d is truncated", index)
		}
		status := string(entry[:2])
		path := string(entry[3:])
		item := projectConversationGitStatusEntry{
			code: status,
			path: filepath.ToSlash(path),
		}
		if strings.Contains(status, "R") || strings.Contains(status, "C") {
			index++
			if index >= len(parts) || len(parts[index]) == 0 {
				return nil, fmt.Errorf("status entry %q is missing original path", status)
			}
			item.oldPath = filepath.ToSlash(string(parts[index]))
		}
		entries = append(entries, item)
	}
	return entries, nil
}

func parseProjectConversationGitNumstat(raw []byte) ([]projectConversationGitNumstat, error) {
	parts := bytes.Split(raw, []byte{0})
	stats := make([]projectConversationGitNumstat, 0, len(parts))
	for index := 0; index < len(parts); index++ {
		entry := parts[index]
		if len(entry) == 0 {
			continue
		}
		fields := strings.SplitN(string(entry), "\t", 3)
		if len(fields) != 3 {
			return nil, fmt.Errorf("numstat entry %d is malformed", index)
		}
		added, err := parseProjectConversationGitNumstatCount(fields[0])
		if err != nil {
			return nil, err
		}
		removed, err := parseProjectConversationGitNumstatCount(fields[1])
		if err != nil {
			return nil, err
		}

		path := fields[2]
		if path == "" {
			if index+2 >= len(parts) {
				return nil, fmt.Errorf("rename numstat entry %d is truncated", index)
			}
			path = string(parts[index+2])
			index += 2
		}
		stats = append(stats, projectConversationGitNumstat{
			path:    filepath.ToSlash(path),
			added:   added,
			removed: removed,
		})
	}
	return stats, nil
}

func parseProjectConversationGitNumstatCount(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "-" {
		return 0, nil
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse numstat count %q: %w", raw, err)
	}
	return count, nil
}

func mapProjectConversationWorkspaceFileStatus(code string) ProjectConversationWorkspaceFileStatus {
	switch {
	case code == "??":
		return ProjectConversationWorkspaceFileStatusUntracked
	case strings.Contains(code, "R") || strings.Contains(code, "C"):
		return ProjectConversationWorkspaceFileStatusRenamed
	case strings.Contains(code, "D"):
		return ProjectConversationWorkspaceFileStatusDeleted
	case strings.Contains(code, "A"):
		return ProjectConversationWorkspaceFileStatusAdded
	default:
		return ProjectConversationWorkspaceFileStatusModified
	}
}

func projectConversationCommandExitedWithCode(err error, code int) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == code
}

func projectConversationWorkspaceMayNotExistYet(conversation chatdomain.Conversation) bool {
	return conversation.LastTurnID == nil && conversation.ProviderThreadID == nil
}
