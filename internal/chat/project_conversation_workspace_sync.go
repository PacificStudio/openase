package chat

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

type ProjectConversationWorkspaceSyncReason string

const (
	ProjectConversationWorkspaceSyncReasonRepoBindingChanged ProjectConversationWorkspaceSyncReason = "repo_binding_changed"
	ProjectConversationWorkspaceSyncReasonRepoMissing        ProjectConversationWorkspaceSyncReason = "repo_missing"
)

type ProjectConversationWorkspaceMissingRepo struct {
	Name string
	Path string
}

type ProjectConversationWorkspaceSyncPrompt struct {
	Reason       ProjectConversationWorkspaceSyncReason
	MissingRepos []ProjectConversationWorkspaceMissingRepo
}

var ErrProjectConversationWorkspaceSyncRequired = errors.New("project conversation workspace sync required")

type ProjectConversationWorkspaceSyncRequiredError struct {
	Prompt   ProjectConversationWorkspaceSyncPrompt
	RepoPath string
}

func (e *ProjectConversationWorkspaceSyncRequiredError) Error() string {
	if e == nil {
		return ErrProjectConversationWorkspaceSyncRequired.Error()
	}
	repoNames := make([]string, 0, len(e.Prompt.MissingRepos))
	for _, repo := range e.Prompt.MissingRepos {
		if name := strings.TrimSpace(repo.Name); name != "" {
			repoNames = append(repoNames, name)
		}
	}
	if len(repoNames) == 0 {
		return "project conversation workspace is out of sync with the project's repo bindings; sync the workspace before browsing or diffing"
	}
	return fmt.Sprintf(
		"project conversation workspace is out of sync with the project's repo bindings; sync the workspace before browsing or diffing missing repo(s): %s",
		strings.Join(repoNames, ", "),
	)
}

func (e *ProjectConversationWorkspaceSyncRequiredError) Is(target error) bool {
	return target == ErrProjectConversationWorkspaceSyncRequired
}

func (s *ProjectConversationService) SyncWorkspace(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (ProjectConversationWorkspaceMetadata, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return ProjectConversationWorkspaceMetadata{}, err
	}
	project, err := s.core.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return ProjectConversationWorkspaceMetadata{}, fmt.Errorf("get project for workspace sync: %w", err)
	}
	providerItem, err := s.core.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return ProjectConversationWorkspaceMetadata{}, fmt.Errorf("get provider for workspace sync: %w", err)
	}
	machine, err := s.core.catalog.GetMachine(ctx, providerItem.MachineID)
	if err != nil {
		return ProjectConversationWorkspaceMetadata{}, fmt.Errorf("get machine for workspace sync: %w", err)
	}
	if _, err := s.ensureConversationWorkspace(ctx, machine, project, providerItem, conversationID); err != nil {
		return ProjectConversationWorkspaceMetadata{}, err
	}
	return s.GetWorkspaceMetadata(ctx, userID, conversationID)
}

func (s *ProjectConversationService) buildConversationWorkspaceRepoLocations(
	ctx context.Context,
	conversation chatdomain.Conversation,
	project catalogdomain.Project,
	machine catalogdomain.Machine,
	workspacePath string,
	projectRepos []catalogdomain.ProjectRepo,
) ([]projectConversationWorkspaceRepoLocation, *ProjectConversationWorkspaceSyncPrompt, map[string]ProjectConversationWorkspaceMissingRepo, error) {
	changedRepoIDs, err := s.listConversationWorkspaceRepoBindingChanges(ctx, project.ID, conversation.CreatedAt)
	if err != nil {
		return nil, nil, nil, err
	}

	repos := make([]projectConversationWorkspaceRepoLocation, 0, len(projectRepos))
	missingByPath := make(map[string]ProjectConversationWorkspaceMissingRepo)
	missingRepos := make([]ProjectConversationWorkspaceMissingRepo, 0)
	reason := ProjectConversationWorkspaceSyncReasonRepoMissing

	for _, repo := range projectRepos {
		repoLocation, err := buildConversationWorkspaceRepoLocation(workspacePath, repo)
		if err != nil {
			return nil, nil, nil, err
		}
		available, err := s.conversationWorkspaceRepoPrepared(ctx, machine, repoLocation.repoPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("inspect workspace repo %s: %w", repo.Name, err)
		}
		if available {
			repos = append(repos, repoLocation)
			continue
		}

		missing := ProjectConversationWorkspaceMissingRepo{
			Name: repoLocation.name,
			Path: repoLocation.relativePath,
		}
		missingRepos = append(missingRepos, missing)
		missingByPath[repoLocation.relativePath] = missing
		if _, ok := changedRepoIDs[repo.ID]; ok {
			reason = ProjectConversationWorkspaceSyncReasonRepoBindingChanged
		}
	}

	sort.Slice(missingRepos, func(i, j int) bool {
		return missingRepos[i].Path < missingRepos[j].Path
	})
	if len(missingRepos) == 0 {
		return repos, nil, missingByPath, nil
	}
	return repos, &ProjectConversationWorkspaceSyncPrompt{
		Reason:       reason,
		MissingRepos: missingRepos,
	}, missingByPath, nil
}

func buildConversationWorkspaceRepoLocation(
	workspacePath string,
	repo catalogdomain.ProjectRepo,
) (projectConversationWorkspaceRepoLocation, error) {
	repoPath := workspaceinfra.RepoPath(workspacePath, repo.WorkspaceDirname, repo.Name)
	relativePath, err := filepath.Rel(workspacePath, repoPath)
	if err != nil {
		return projectConversationWorkspaceRepoLocation{}, fmt.Errorf("derive relative repo path for %s: %w", repo.Name, err)
	}
	return projectConversationWorkspaceRepoLocation{
		name:         repo.Name,
		repoPath:     repoPath,
		relativePath: filepath.ToSlash(relativePath),
	}, nil
}

func (s *ProjectConversationService) conversationWorkspaceRepoPrepared(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
) (bool, error) {
	_, err := workspaceinfra.ReadWorkspaceGitBranch(ctx, repoPath, func(
		ctx context.Context,
		args []string,
		allowExitCodeOne bool,
	) ([]byte, error) {
		return s.runProjectConversationGitCommand(ctx, machine, args, allowExitCodeOne)
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, workspaceinfra.ErrGitWorkspaceUnavailable) {
		return false, nil
	}
	return false, err
}

func (s *ProjectConversationService) listConversationWorkspaceRepoBindingChanges(
	ctx context.Context,
	projectID uuid.UUID,
	since time.Time,
) (map[uuid.UUID]struct{}, error) {
	changes := make(map[uuid.UUID]struct{})
	before := ""
	since = since.UTC()

	for {
		input, err := catalogdomain.ParseListActivityEvents(projectID, catalogdomain.ActivityEventListInput{
			Limit:  strconv.Itoa(catalogdomain.MaxActivityEventLimit),
			Before: before,
		})
		if err != nil {
			return nil, fmt.Errorf("build activity filter for workspace repo sync: %w", err)
		}
		page, err := s.core.catalog.ListActivityEvents(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list project activity for workspace repo sync: %w", err)
		}

		reachedHistoryBeforeConversation := false
		for _, event := range page.Events {
			if !event.CreatedAt.UTC().After(since) {
				reachedHistoryBeforeConversation = true
				break
			}
			repoID, ok := projectConversationWorkspaceBindingChangeRepoID(event)
			if ok {
				changes[repoID] = struct{}{}
			}
		}

		if reachedHistoryBeforeConversation || !page.HasMore || strings.TrimSpace(page.NextCursor) == "" {
			break
		}
		before = page.NextCursor
	}

	return changes, nil
}

func projectConversationWorkspaceBindingChangeRepoID(
	event catalogdomain.ActivityEvent,
) (uuid.UUID, bool) {
	switch event.EventType {
	case activityevent.TypeProjectRepoCreated:
	case activityevent.TypeProjectRepoUpdated:
		if !projectConversationWorkspaceActivityTouchesBinding(event.Metadata) {
			return uuid.UUID{}, false
		}
	default:
		return uuid.UUID{}, false
	}

	rawRepoID, ok := event.Metadata["repo_id"].(string)
	if !ok {
		return uuid.UUID{}, false
	}
	repoID, err := uuid.Parse(strings.TrimSpace(rawRepoID))
	if err != nil {
		return uuid.UUID{}, false
	}
	return repoID, true
}

func projectConversationWorkspaceActivityTouchesBinding(metadata map[string]any) bool {
	changedFields, ok := metadata["changed_fields"]
	if !ok {
		return false
	}
	for _, field := range projectConversationWorkspaceChangedFields(changedFields) {
		switch field {
		case "repo", "name", "repository_url", "default_branch", "workspace_dirname":
			return true
		}
	}
	return false
}

func projectConversationWorkspaceChangedFields(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		fields := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text != "" {
				fields = append(fields, text)
			}
		}
		return fields
	default:
		return nil
	}
}
