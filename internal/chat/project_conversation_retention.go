package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityeventdomain "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

type projectConversationWorkspaceTarget struct {
	project       catalogdomain.Project
	provider      catalogdomain.AgentProvider
	machine       catalogdomain.Machine
	workspaceRoot string
	workspacePath string
}

func (s *ProjectConversationService) DeleteConversation(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input domain.DeleteConversationInput,
) (domain.DeleteConversationResult, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if errors.Is(err, ErrConversationNotFound) {
		return domain.DeleteConversationResult{
			ConversationID: conversationID,
			Trigger:        normalizeConversationDeleteTrigger(input.Trigger),
		}, nil
	}
	if err != nil {
		return domain.DeleteConversationResult{}, err
	}
	return s.deleteConversation(ctx, conversation, input)
}

func (s *ProjectConversationService) RunRetentionCleanup(
	ctx context.Context,
) ([]domain.RetentionCleanupResult, error) {
	if s == nil || s.catalog == nil || s.conversations == nil {
		return nil, nil
	}

	organizations, err := s.catalog.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations for project conversation retention cleanup: %w", err)
	}

	results := make([]domain.RetentionCleanupResult, 0)
	var cleanupErrs []error
	for _, organization := range organizations {
		projects, listErr := s.catalog.ListProjects(ctx, organization.ID)
		if listErr != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("list projects for organization %s: %w", organization.ID, listErr))
			continue
		}
		for _, project := range projects {
			if !project.ProjectAIRetention.Enabled {
				continue
			}
			result, projectErr := s.runProjectRetentionCleanup(ctx, project)
			if projectErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Errorf("run retention cleanup for project %s: %w", project.ID, projectErr))
				continue
			}
			results = append(results, result)
		}
	}
	return results, errors.Join(cleanupErrs...)
}

func (s *ProjectConversationService) runProjectRetentionCleanup(
	ctx context.Context,
	project catalogdomain.Project,
) (domain.RetentionCleanupResult, error) {
	now := time.Now().UTC()
	result := domain.RetentionCleanupResult{
		ProjectID: project.ID,
		RanAt:     now,
	}

	source := domain.SourceProjectSidebar
	conversations, err := s.conversations.ListConversations(ctx, domain.ListConversationsFilter{
		ProjectID: project.ID,
		Source:    &source,
	})
	if err != nil {
		return domain.RetentionCleanupResult{}, err
	}
	result.Scanned = len(conversations)

	groups := map[string][]domain.Conversation{}
	for _, conversation := range conversations {
		groups[conversation.UserID] = append(groups[conversation.UserID], conversation)
	}

	cutoff := time.Time{}
	if project.ProjectAIRetention.KeepRecentDays > 0 {
		cutoff = now.AddDate(0, 0, -project.ProjectAIRetention.KeepRecentDays)
	}

	for _, items := range groups {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].LastActivityAt.Equal(items[j].LastActivityAt) {
				return items[i].CreatedAt.After(items[j].CreatedAt)
			}
			return items[i].LastActivityAt.After(items[j].LastActivityAt)
		})

		retained := make(map[uuid.UUID]struct{}, len(items))
		if project.ProjectAIRetention.KeepLatestN > 0 {
			limit := min(project.ProjectAIRetention.KeepLatestN, len(items))
			for i := 0; i < limit; i++ {
				retained[items[i].ID] = struct{}{}
			}
		}
		if !cutoff.IsZero() {
			for _, conversation := range items {
				if !conversation.LastActivityAt.Before(cutoff) {
					retained[conversation.ID] = struct{}{}
				}
			}
		}

		for _, conversation := range items {
			if _, ok := retained[conversation.ID]; ok {
				continue
			}
			skip, skipErr := s.retentionSkipReason(ctx, conversation)
			if skipErr != nil {
				return domain.RetentionCleanupResult{}, skipErr
			}
			if skip != nil {
				result.Skipped = append(result.Skipped, *skip)
				s.emitProjectConversationCleanupSkipActivity(ctx, *skip)
				continue
			}

			deleted, deleteErr := s.deleteConversation(ctx, conversation, domain.DeleteConversationInput{Trigger: domain.DeleteTriggerRetention})
			if deleteErr != nil {
				return domain.RetentionCleanupResult{}, deleteErr
			}
			result.Deleted = append(result.Deleted, deleted)
		}
	}

	s.emitProjectConversationCleanupRunActivity(ctx, project, result)
	return result, nil
}

func (s *ProjectConversationService) retentionSkipReason(
	ctx context.Context,
	conversation domain.Conversation,
) (*domain.RetentionCleanupSkip, error) {
	if active, detail, err := s.conversationHasActiveRuntime(ctx, conversation); err != nil {
		return nil, err
	} else if active {
		return &domain.RetentionCleanupSkip{
			ConversationID: conversation.ID,
			ProjectID:      conversation.ProjectID,
			UserID:         conversation.UserID,
			Reason:         domain.RetentionCleanupSkipRuntimeActive,
			Detail:         detail,
		}, nil
	}

	pendingInterrupts, err := s.interrupts.ListPendingInterrupts(ctx, conversation.ID)
	if err != nil && !errors.Is(err, ErrConversationNotFound) {
		return nil, err
	}
	for _, interrupt := range pendingInterrupts {
		if interrupt.Status == domain.InterruptStatusPending {
			return &domain.RetentionCleanupSkip{
				ConversationID: conversation.ID,
				ProjectID:      conversation.ProjectID,
				UserID:         conversation.UserID,
				Reason:         domain.RetentionCleanupSkipPendingInput,
				Detail:         "Pending user input is still required before the conversation can be pruned.",
			}, nil
		}
	}

	target, err := s.resolveConversationWorkspaceTarget(ctx, conversation)
	if err != nil {
		if errors.Is(err, errProjectConversationWorkspaceLocationUnavailable) && projectConversationWorkspaceMayNotExistYet(conversation) {
			return nil, nil
		}
		return nil, err
	}
	workspaceExists, err := s.conversationWorkspaceExists(ctx, target.machine, target.workspacePath)
	if err != nil || !workspaceExists {
		return nil, err
	}
	workspaceDirty, err := s.conversationWorkspaceDirty(ctx, target, conversation)
	if err != nil {
		return nil, err
	}
	if workspaceDirty {
		return &domain.RetentionCleanupSkip{
			ConversationID: conversation.ID,
			ProjectID:      conversation.ProjectID,
			UserID:         conversation.UserID,
			Reason:         domain.RetentionCleanupSkipDirtyWorkspace,
			Detail:         "Automatic cleanup skips dirty Project AI workspaces by default.",
		}, nil
	}
	return nil, nil
}

func (s *ProjectConversationService) deleteConversation(
	ctx context.Context,
	conversation domain.Conversation,
	input domain.DeleteConversationInput,
) (domain.DeleteConversationResult, error) {
	trigger := normalizeConversationDeleteTrigger(input.Trigger)
	result := domain.DeleteConversationResult{
		ConversationID: conversation.ID,
		ProjectID:      conversation.ProjectID,
		UserID:         conversation.UserID,
		Trigger:        trigger,
	}

	if err := s.closeConversationRuntime(ctx, conversation); err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			return result, nil
		}
		return result, err
	}

	target, err := s.resolveConversationWorkspaceTarget(ctx, conversation)
	workspaceResolvable := err == nil
	if err != nil {
		if !errors.Is(err, errProjectConversationWorkspaceLocationUnavailable) ||
			!projectConversationWorkspaceMayNotExistYet(conversation) {
			return result, err
		}
	}
	if workspaceResolvable {
		result.WorkspacePath = target.workspacePath
		workspaceExists, existsErr := s.conversationWorkspaceExists(ctx, target.machine, target.workspacePath)
		if existsErr != nil {
			return result, existsErr
		}
		if workspaceExists {
			workspaceDirty, dirtyErr := s.conversationWorkspaceDirty(ctx, target, conversation)
			if dirtyErr != nil {
				return result, dirtyErr
			}
			result.WorkspaceDirty = workspaceDirty
			if workspaceDirty && !input.Force {
				return result, fmt.Errorf("%w: rerun deletion with force to remove the dirty workspace", domain.ErrWorkspaceDirty)
			}
			workspaceDeleted, deleteErr := s.deleteConversationWorkspace(ctx, target)
			if deleteErr != nil {
				return result, deleteErr
			}
			result.WorkspaceDeleted = workspaceDeleted
		}
	}

	deleted, err := s.conversations.DeleteConversation(ctx, conversation.ID)
	if errors.Is(err, ErrConversationNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.EntriesDeleted = deleted.EntriesDeleted
	result.TurnsDeleted = deleted.TurnsDeleted
	result.InterruptsDeleted = deleted.InterruptsDeleted
	result.RunsDeleted = deleted.RunsDeleted
	result.TraceEventsDeleted = deleted.TraceEventsDeleted
	result.StepEventsDeleted = deleted.StepEventsDeleted
	result.AgentTokensDeleted = deleted.AgentTokensDeleted
	result.DeletedAt = time.Now().UTC()

	s.broadcastConversationEvent(conversation, StreamEvent{
		Event: "deleted",
		Payload: map[string]any{
			"conversation_id":   conversation.ID.String(),
			"project_id":        conversation.ProjectID.String(),
			"trigger":           string(trigger),
			"workspace_deleted": result.WorkspaceDeleted,
			"workspace_dirty":   result.WorkspaceDirty,
		},
	})
	s.emitProjectConversationDeletedActivity(ctx, conversation, result, input.Force)
	return result, nil
}

func (s *ProjectConversationService) closeConversationRuntime(
	ctx context.Context,
	conversation domain.Conversation,
) error {
	live, _ := s.runtimeManager.Close(conversation.ID)
	if live != nil && live.principal.ID != uuid.Nil {
		if live.principal.CurrentRunID != nil && *live.principal.CurrentRunID != uuid.Nil {
			now := time.Now().UTC()
			terminatedStatus := domain.RunStatusTerminated
			_, _ = s.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
				RunID:                *live.principal.CurrentRunID,
				Status:               &terminatedStatus,
				TerminalAt:           &now,
				LastHeartbeatAt:      &now,
				CurrentStepStatus:    optionalString("runtime_closed"),
				CurrentStepSummary:   optionalString("Project conversation runtime closed."),
				CurrentStepChangedAt: &now,
			})
		}
		if principal, principalErr := s.runtimeStore.ClosePrincipal(ctx, domain.ClosePrincipalInput{PrincipalID: live.principal.ID}); principalErr == nil {
			live.principal = principal
		}
	}
	updatedConversation, err := s.conversations.CloseConversationRuntime(ctx, conversation.ID)
	if err == nil {
		var providerItem *catalogdomain.AgentProvider
		if live != nil {
			providerItem = &live.provider
		} else if s.catalog != nil {
			if resolved, resolveErr := s.catalog.GetAgentProvider(ctx, conversation.ProviderID); resolveErr == nil {
				providerItem = &resolved
			}
		}
		s.broadcastConversationEvent(updatedConversation, StreamEvent{
			Event:   "session",
			Payload: conversationSessionPayload(conversation.ID, "inactive", updatedConversation, providerItem),
		})
	}
	if errors.Is(err, ErrConversationNotFound) {
		return nil
	}
	return err
}

func (s *ProjectConversationService) conversationHasActiveRuntime(
	ctx context.Context,
	conversation domain.Conversation,
) (bool, string, error) {
	if live, ok := s.runtimeManager.Get(conversation.ID); ok && live != nil {
		return true, "A live Project AI runtime is still attached to this conversation.", nil
	}
	principal, err := s.runtimeStore.GetPrincipal(ctx, conversation.ID)
	if errors.Is(err, ErrConversationNotFound) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	if principal.Status == domain.PrincipalStatusActive && principal.RuntimeState != domain.RuntimeStateInactive {
		return true, fmt.Sprintf("Runtime state is still %s.", principal.RuntimeState), nil
	}
	if principal.CurrentRunID != nil && *principal.CurrentRunID != uuid.Nil {
		return true, "A Project AI run is still attached to this conversation.", nil
	}
	return false, "", nil
}

func (s *ProjectConversationService) resolveConversationWorkspaceTarget(
	ctx context.Context,
	conversation domain.Conversation,
) (projectConversationWorkspaceTarget, error) {
	project, err := s.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return projectConversationWorkspaceTarget{}, fmt.Errorf("get project for conversation delete: %w", err)
	}
	providerItem, err := s.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return projectConversationWorkspaceTarget{}, fmt.Errorf("get provider for conversation delete: %w", err)
	}
	machine, err := s.catalog.GetMachine(ctx, providerItem.MachineID)
	if err != nil {
		return projectConversationWorkspaceTarget{}, fmt.Errorf("get machine for conversation delete: %w", err)
	}
	workspaceRoot, err := resolveProjectConversationWorkspaceRoot(machine)
	if err != nil {
		return projectConversationWorkspaceTarget{}, err
	}
	workspacePath, err := s.resolveConversationWorkspacePath(machine, project, conversation.ID)
	if err != nil {
		return projectConversationWorkspaceTarget{}, err
	}
	return projectConversationWorkspaceTarget{
		project:       project,
		provider:      providerItem,
		machine:       machine,
		workspaceRoot: workspaceRoot,
		workspacePath: workspacePath,
	}, nil
}

func resolveProjectConversationWorkspaceRoot(machine catalogdomain.Machine) (string, error) {
	root := ""
	if machine.WorkspaceRoot != nil && strings.TrimSpace(*machine.WorkspaceRoot) != "" {
		root = strings.TrimSpace(*machine.WorkspaceRoot)
	} else if machine.Host == catalogdomain.LocalMachineHost {
		localRoot, err := workspaceinfra.LocalWorkspaceRoot()
		if err != nil {
			return "", err
		}
		root = localRoot
	}
	if root == "" {
		return "", fmt.Errorf("%w: chat provider machine %s is missing workspace_root", errProjectConversationWorkspaceLocationUnavailable, machine.Name)
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("%w: chat provider machine %s workspace_root must be absolute", errProjectConversationWorkspaceLocationUnavailable, machine.Name)
	}
	return filepath.Clean(root), nil
}

func (s *ProjectConversationService) conversationWorkspaceExists(
	ctx context.Context,
	machine catalogdomain.Machine,
	workspacePath string,
) (bool, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		_, err := os.Lstat(workspacePath)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("stat project conversation workspace %s: %w", workspacePath, err)
		}
		return true, nil
	}
	command := fmt.Sprintf("if [ -e %s ]; then exit 0; fi\nexit 1\n", projectConversationShellQuote(workspacePath))
	_, err := s.runProjectConversationShellCommand(ctx, machine, command, true)
	if err != nil {
		return false, fmt.Errorf("stat remote project conversation workspace %s: %w", workspacePath, err)
	}
	return true, nil
}

func (s *ProjectConversationService) conversationWorkspaceDirty(
	ctx context.Context,
	target projectConversationWorkspaceTarget,
	conversation domain.Conversation,
) (bool, error) {
	location, err := s.resolveConversationWorkspaceLocation(ctx, conversation, target.project, target.provider)
	if err != nil {
		return false, err
	}
	for _, repo := range location.repos {
		repoSummary, repoErr := s.summarizeConversationWorkspaceRepo(ctx, target.machine, repo)
		if repoErr != nil {
			if errors.Is(repoErr, ErrProjectConversationWorkspaceUnavailable) && projectConversationWorkspaceMayNotExistYet(conversation) {
				return false, nil
			}
			return false, repoErr
		}
		if repoSummary.Dirty {
			return true, nil
		}
	}
	return false, nil
}

func (s *ProjectConversationService) deleteConversationWorkspace(
	ctx context.Context,
	target projectConversationWorkspaceTarget,
) (bool, error) {
	if target.machine.Host == catalogdomain.LocalMachineHost {
		return deleteLocalProjectConversationWorkspace(target.workspaceRoot, target.workspacePath)
	}
	return s.deleteRemoteProjectConversationWorkspace(ctx, target)
}

func deleteLocalProjectConversationWorkspace(root string, target string) (bool, error) {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	if cleanTarget == cleanRoot || !projectConversationWorkspacePathWithinRoot(cleanRoot, cleanTarget) {
		return false, domain.ErrWorkspacePathConflict
	}

	rootRealPath, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("resolve project conversation workspace root %s: %w", cleanRoot, err)
	}
	if rootRealPath == "" {
		rootRealPath = cleanRoot
	}

	info, err := os.Lstat(cleanTarget)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat project conversation workspace %s: %w", cleanTarget, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, domain.ErrWorkspacePathConflict
	}

	targetRealPath, err := filepath.EvalSymlinks(cleanTarget)
	if err != nil {
		return false, fmt.Errorf("resolve project conversation workspace %s: %w", cleanTarget, err)
	}
	if targetRealPath == rootRealPath || !projectConversationWorkspacePathWithinRoot(rootRealPath, targetRealPath) {
		return false, domain.ErrWorkspacePathConflict
	}
	if err := os.RemoveAll(targetRealPath); err != nil {
		return false, fmt.Errorf("%w: %v", domain.ErrWorkspaceDeleteFailed, err)
	}
	return true, nil
}

func (s *ProjectConversationService) deleteRemoteProjectConversationWorkspace(
	ctx context.Context,
	target projectConversationWorkspaceTarget,
) (bool, error) {
	cleanRoot := filepath.Clean(target.workspaceRoot)
	cleanTarget := filepath.Clean(target.workspacePath)
	if cleanTarget == cleanRoot || !projectConversationWorkspacePathWithinRoot(cleanRoot, cleanTarget) {
		return false, domain.ErrWorkspacePathConflict
	}

	command := fmt.Sprintf(`set -eu
n_deleted=0
root=%s
target=%s
root_real=$(cd "$root" && pwd -P)
case "$target" in
  "$root"|"$root"/*) ;;
  *) echo unsafe >&2; exit 12 ;;
esac
if [ ! -e "$target" ]; then
  printf '0'
  exit 0
fi
if [ -L "$target" ]; then
  echo symlink >&2
  exit 12
fi
if [ ! -d "$target" ]; then
  echo not_directory >&2
  exit 12
fi
target_real=$(cd "$target" && pwd -P)
case "$target_real" in
  "$root_real") echo root >&2; exit 12 ;;
  "$root_real"/*) ;;
  *) echo escape >&2; exit 12 ;;
esac
rm -rf -- "$target_real"
printf '1'
`, projectConversationShellQuote(cleanRoot), projectConversationShellQuote(cleanTarget))
	output, err := s.runProjectConversationShellCommand(ctx, target.machine, command, false)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 12") {
			return false, domain.ErrWorkspacePathConflict
		}
		return false, fmt.Errorf("%w: %v", domain.ErrWorkspaceDeleteFailed, err)
	}
	return strings.TrimSpace(string(output)) == "1", nil
}

func (s *ProjectConversationService) emitProjectConversationDeletedActivity(
	ctx context.Context,
	conversation domain.Conversation,
	result domain.DeleteConversationResult,
	force bool,
) {
	s.emitProjectConversationActivity(ctx, activitysvc.RecordInput{
		ProjectID: conversation.ProjectID,
		EventType: activityeventdomain.TypeProjectConversationDeleted,
		Message:   "Deleted Project AI conversation.",
		Metadata: map[string]any{
			"conversation_id":      conversation.ID.String(),
			"user_id":              conversation.UserID,
			"provider_id":          conversation.ProviderID.String(),
			"trigger":              string(result.Trigger),
			"force":                force,
			"workspace_path":       result.WorkspacePath,
			"workspace_deleted":    result.WorkspaceDeleted,
			"workspace_dirty":      result.WorkspaceDirty,
			"entries_deleted":      result.EntriesDeleted,
			"turns_deleted":        result.TurnsDeleted,
			"interrupts_deleted":   result.InterruptsDeleted,
			"runs_deleted":         result.RunsDeleted,
			"trace_events_deleted": result.TraceEventsDeleted,
			"step_events_deleted":  result.StepEventsDeleted,
			"agent_tokens_deleted": result.AgentTokensDeleted,
		},
	})
}

func (s *ProjectConversationService) emitProjectConversationCleanupRunActivity(
	ctx context.Context,
	project catalogdomain.Project,
	result domain.RetentionCleanupResult,
) {
	s.emitProjectConversationActivity(ctx, activitysvc.RecordInput{
		ProjectID: project.ID,
		EventType: activityeventdomain.TypeProjectConversationCleanupRun,
		Message:   "Ran Project AI retention cleanup.",
		Metadata: map[string]any{
			"project_id":               project.ID.String(),
			"enabled":                  project.ProjectAIRetention.Enabled,
			"keep_latest_n":            project.ProjectAIRetention.KeepLatestN,
			"keep_recent_days":         project.ProjectAIRetention.KeepRecentDays,
			"scanned":                  result.Scanned,
			"deleted_count":            len(result.Deleted),
			"skipped_count":            len(result.Skipped),
			"deleted_ids":              mapDeleteConversationIDs(result.Deleted),
			"skipped_conversation_ids": mapRetentionCleanupSkipConversationIDs(result.Skipped),
		},
	})
}

func (s *ProjectConversationService) emitProjectConversationCleanupSkipActivity(
	ctx context.Context,
	skip domain.RetentionCleanupSkip,
) {
	s.emitProjectConversationActivity(ctx, activitysvc.RecordInput{
		ProjectID: skip.ProjectID,
		EventType: activityeventdomain.TypeProjectConversationCleanupSkip,
		Message:   "Skipped Project AI retention cleanup for a conversation.",
		Metadata: map[string]any{
			"conversation_id": skip.ConversationID.String(),
			"user_id":         skip.UserID,
			"reason":          string(skip.Reason),
			"detail":          skip.Detail,
		},
	})
}

func (s *ProjectConversationService) emitProjectConversationActivity(
	ctx context.Context,
	input activitysvc.RecordInput,
) {
	if s == nil || s.activityEmitter == nil || input.ProjectID == uuid.Nil {
		return
	}
	if _, err := s.activityEmitter.Emit(ctx, input); err != nil {
		s.logger.Warn("emit project conversation activity", "event_type", input.EventType, "project_id", input.ProjectID, "error", err)
	}
}

func mapDeleteConversationIDs(items []domain.DeleteConversationResult) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ConversationID.String())
	}
	return ids
}

func mapRetentionCleanupSkipConversationIDs(items []domain.RetentionCleanupSkip) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ConversationID.String())
	}
	return ids
}

func normalizeConversationDeleteTrigger(trigger domain.DeleteTrigger) domain.DeleteTrigger {
	if strings.TrimSpace(string(trigger)) == "" {
		return domain.DeleteTriggerManual
	}
	return trigger
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
