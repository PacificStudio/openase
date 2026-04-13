package chat

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

type WorkspaceBranchName string

func (n WorkspaceBranchName) String() string { return string(n) }

type WorkspaceCommitID string

func (id WorkspaceCommitID) String() string { return string(id) }

type WorkspaceCheckoutTargetKind string

const (
	WorkspaceCheckoutTargetKindLocalBranch          WorkspaceCheckoutTargetKind = "local_branch"
	WorkspaceCheckoutTargetKindRemoteTrackingBranch WorkspaceCheckoutTargetKind = "remote_tracking_branch"
	WorkspaceCheckoutTargetKindNewLocalBranch       WorkspaceCheckoutTargetKind = "new_local_branch"
)

type WorkspaceCheckoutTarget struct {
	Kind                 WorkspaceCheckoutTargetKind
	BranchName           WorkspaceBranchName
	CreateTrackingBranch bool
	LocalBranchName      *WorkspaceBranchName
}

type WorkspaceGitGraphWindow struct {
	Limit int
}

const (
	projectConversationWorkspaceGitGraphDefaultLimit = 40
	projectConversationWorkspaceGitGraphMaxLimit     = 120
)

type ProjectConversationWorkspaceCurrentRefKind string

const (
	ProjectConversationWorkspaceCurrentRefKindBranch   ProjectConversationWorkspaceCurrentRefKind = "branch"
	ProjectConversationWorkspaceCurrentRefKindDetached ProjectConversationWorkspaceCurrentRefKind = "detached"
)

type ProjectConversationWorkspaceCurrentRef struct {
	Kind           ProjectConversationWorkspaceCurrentRefKind
	DisplayName    string
	CacheKey       string
	BranchName     string
	BranchFullName string
	CommitID       string
	ShortCommitID  string
	Subject        string
}

type ProjectConversationWorkspaceBranchScope string

const (
	ProjectConversationWorkspaceBranchScopeLocal          ProjectConversationWorkspaceBranchScope = "local_branch"
	ProjectConversationWorkspaceBranchScopeRemoteTracking ProjectConversationWorkspaceBranchScope = "remote_tracking_branch"
)

type ProjectConversationWorkspaceBranchRef struct {
	Name                     string
	FullName                 string
	Scope                    ProjectConversationWorkspaceBranchScope
	Current                  bool
	CommitID                 string
	ShortCommitID            string
	Subject                  string
	UpstreamName             string
	Ahead                    int
	Behind                   int
	SuggestedLocalBranchName string
}

type ProjectConversationWorkspaceRepoRefs struct {
	ConversationID uuid.UUID
	RepoPath       string
	CurrentRef     ProjectConversationWorkspaceCurrentRef
	LocalBranches  []ProjectConversationWorkspaceBranchRef
	RemoteBranches []ProjectConversationWorkspaceBranchRef
}

type ProjectConversationWorkspaceGitRefLabelScope string

const (
	ProjectConversationWorkspaceGitRefLabelScopeHead           ProjectConversationWorkspaceGitRefLabelScope = "head"
	ProjectConversationWorkspaceGitRefLabelScopeLocalBranch    ProjectConversationWorkspaceGitRefLabelScope = "local_branch"
	ProjectConversationWorkspaceGitRefLabelScopeRemoteTracking ProjectConversationWorkspaceGitRefLabelScope = "remote_tracking_branch"
)

type ProjectConversationWorkspaceGitRefLabel struct {
	Name     string
	FullName string
	Scope    ProjectConversationWorkspaceGitRefLabelScope
	Current  bool
}

type ProjectConversationWorkspaceGitGraphCommit struct {
	CommitID      string
	ShortCommitID string
	ParentIDs     []string
	Subject       string
	AuthorName    string
	AuthoredAt    time.Time
	Labels        []ProjectConversationWorkspaceGitRefLabel
	Head          bool
}

type ProjectConversationWorkspaceGitGraph struct {
	ConversationID uuid.UUID
	RepoPath       string
	Window         WorkspaceGitGraphWindow
	Commits        []ProjectConversationWorkspaceGitGraphCommit
}

type ProjectConversationWorkspaceCheckoutInput struct {
	RepoPath               WorkspaceRepoPath
	Target                 WorkspaceCheckoutTarget
	ExpectedCleanWorkspace bool
}

type ProjectConversationWorkspaceCheckoutResult struct {
	ConversationID     uuid.UUID
	RepoPath           string
	CurrentRef         ProjectConversationWorkspaceCurrentRef
	CreatedLocalBranch string
}

type ProjectConversationWorkspaceCheckoutPreconditionReason string

const (
	ProjectConversationWorkspaceCheckoutPreconditionDirtyWorkspace    ProjectConversationWorkspaceCheckoutPreconditionReason = "dirty_workspace"
	ProjectConversationWorkspaceCheckoutPreconditionLocalBranchExists ProjectConversationWorkspaceCheckoutPreconditionReason = "local_branch_exists"
)

type ProjectConversationWorkspaceCheckoutPreconditionError struct {
	Reason          ProjectConversationWorkspaceCheckoutPreconditionReason
	RequestedBranch string
	SuggestedBranch string
}

func (e *ProjectConversationWorkspaceCheckoutPreconditionError) Error() string {
	if e == nil {
		return "project conversation workspace checkout precondition failed"
	}
	switch e.Reason {
	case ProjectConversationWorkspaceCheckoutPreconditionDirtyWorkspace:
		return "project conversation workspace checkout requires a clean repo"
	case ProjectConversationWorkspaceCheckoutPreconditionLocalBranchExists:
		if strings.TrimSpace(e.SuggestedBranch) != "" {
			return fmt.Sprintf("local branch %s already exists; switch to that branch instead", e.SuggestedBranch)
		}
		return "local tracking branch already exists"
	default:
		return "project conversation workspace checkout precondition failed"
	}
}

type projectConversationWorkspaceGitBranchRef struct {
	Name         WorkspaceBranchName
	FullName     string
	Scope        ProjectConversationWorkspaceBranchScope
	CommitID     WorkspaceCommitID
	Subject      string
	UpstreamName string
	Ahead        int
	Behind       int
}

type projectConversationWorkspaceGitGraphCommitRecord struct {
	CommitID   WorkspaceCommitID
	ParentIDs  []WorkspaceCommitID
	Subject    string
	AuthorName string
	AuthoredAt time.Time
}

func ParseWorkspaceBranchName(raw string) (WorkspaceBranchName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("branch name must not be empty")
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasSuffix(trimmed, "/") {
		return "", fmt.Errorf("branch name must not start or end with /")
	}
	if strings.Contains(trimmed, "..") ||
		strings.Contains(trimmed, "@{") ||
		strings.ContainsAny(trimmed, " \t\n\r~^:?*[\\") {
		return "", fmt.Errorf("branch name is invalid")
	}
	for _, part := range strings.Split(trimmed, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".lock") || strings.HasPrefix(part, ".") {
			return "", fmt.Errorf("branch name is invalid")
		}
	}
	return WorkspaceBranchName(trimmed), nil
}

func ParseWorkspaceCommitID(raw string) (WorkspaceCommitID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("commit id must not be empty")
	}
	if len(trimmed) < 7 || len(trimmed) > 40 {
		return "", fmt.Errorf("commit id must be 7 to 40 hex characters")
	}
	for _, ch := range trimmed {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
			return "", fmt.Errorf("commit id must be hexadecimal")
		}
	}
	return WorkspaceCommitID(strings.ToLower(trimmed)), nil
}

func ParseWorkspaceGitGraphWindow(raw string) (WorkspaceGitGraphWindow, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return WorkspaceGitGraphWindow{Limit: projectConversationWorkspaceGitGraphDefaultLimit}, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return WorkspaceGitGraphWindow{}, fmt.Errorf("limit must be a positive integer")
	}
	if value <= 0 {
		return WorkspaceGitGraphWindow{}, fmt.Errorf("limit must be positive")
	}
	if value > projectConversationWorkspaceGitGraphMaxLimit {
		return WorkspaceGitGraphWindow{}, fmt.Errorf(
			"limit must not exceed %d",
			projectConversationWorkspaceGitGraphMaxLimit,
		)
	}
	return WorkspaceGitGraphWindow{Limit: value}, nil
}

func ParseWorkspaceCheckoutTarget(
	targetKind string,
	targetName string,
	createTrackingBranch bool,
	localBranchName string,
) (WorkspaceCheckoutTarget, error) {
	branchName, err := ParseWorkspaceBranchName(targetName)
	if err != nil {
		return WorkspaceCheckoutTarget{}, err
	}

	var parsedLocalBranch *WorkspaceBranchName
	if strings.TrimSpace(localBranchName) != "" {
		parsed, err := ParseWorkspaceBranchName(localBranchName)
		if err != nil {
			return WorkspaceCheckoutTarget{}, err
		}
		parsedLocalBranch = &parsed
	}

	switch strings.TrimSpace(targetKind) {
	case string(WorkspaceCheckoutTargetKindLocalBranch):
		if createTrackingBranch {
			return WorkspaceCheckoutTarget{}, fmt.Errorf("create_tracking_branch is only supported for remote branches")
		}
		if parsedLocalBranch != nil {
			return WorkspaceCheckoutTarget{}, fmt.Errorf("local_branch_name is only supported for remote branches")
		}
		return WorkspaceCheckoutTarget{
			Kind:       WorkspaceCheckoutTargetKindLocalBranch,
			BranchName: branchName,
		}, nil
	case string(WorkspaceCheckoutTargetKindRemoteTrackingBranch):
		if !createTrackingBranch {
			return WorkspaceCheckoutTarget{}, fmt.Errorf("create_tracking_branch must be true for remote branches")
		}
		return WorkspaceCheckoutTarget{
			Kind:                 WorkspaceCheckoutTargetKindRemoteTrackingBranch,
			BranchName:           branchName,
			CreateTrackingBranch: true,
			LocalBranchName:      parsedLocalBranch,
		}, nil
	case string(WorkspaceCheckoutTargetKindNewLocalBranch):
		if createTrackingBranch {
			return WorkspaceCheckoutTarget{}, fmt.Errorf("create_tracking_branch is only supported for remote branches")
		}
		if parsedLocalBranch != nil {
			return WorkspaceCheckoutTarget{}, fmt.Errorf("local_branch_name is only supported for remote branches")
		}
		return WorkspaceCheckoutTarget{
			Kind:       WorkspaceCheckoutTargetKindNewLocalBranch,
			BranchName: branchName,
		}, nil
	default:
		return WorkspaceCheckoutTarget{}, fmt.Errorf(
			"target_kind must be %s, %s, or %s",
			WorkspaceCheckoutTargetKindLocalBranch,
			WorkspaceCheckoutTargetKindRemoteTrackingBranch,
			WorkspaceCheckoutTargetKindNewLocalBranch,
		)
	}
}

func (s *ProjectConversationService) GetWorkspaceRepoRefs(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	repoPath WorkspaceRepoPath,
) (ProjectConversationWorkspaceRepoRefs, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		repoPath.String(),
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceRepoRefs{}, err
	}

	currentRef, branchRefs, err := s.readConversationWorkspaceGitRefState(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
	)
	if err != nil {
		return ProjectConversationWorkspaceRepoRefs{}, err
	}

	localBranches := make([]ProjectConversationWorkspaceBranchRef, 0, len(branchRefs))
	remoteBranches := make([]ProjectConversationWorkspaceBranchRef, 0, len(branchRefs))
	for _, ref := range branchRefs {
		item := mapProjectConversationWorkspaceBranchRef(ref, currentRef)
		switch ref.Scope {
		case ProjectConversationWorkspaceBranchScopeLocal:
			localBranches = append(localBranches, item)
		case ProjectConversationWorkspaceBranchScopeRemoteTracking:
			remoteBranches = append(remoteBranches, item)
		}
	}
	sort.Slice(localBranches, func(i, j int) bool { return localBranches[i].Name < localBranches[j].Name })
	sort.Slice(remoteBranches, func(i, j int) bool { return remoteBranches[i].Name < remoteBranches[j].Name })

	return ProjectConversationWorkspaceRepoRefs{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		CurrentRef:     currentRef,
		LocalBranches:  localBranches,
		RemoteBranches: remoteBranches,
	}, nil
}

func (s *ProjectConversationService) GetWorkspaceGitGraph(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	repoPath WorkspaceRepoPath,
	window WorkspaceGitGraphWindow,
) (ProjectConversationWorkspaceGitGraph, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		repoPath.String(),
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceGitGraph{}, err
	}

	currentRef, branchRefs, err := s.readConversationWorkspaceGitRefState(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
	)
	if err != nil {
		return ProjectConversationWorkspaceGitGraph{}, err
	}
	records, err := s.readConversationWorkspaceGitGraphRecords(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
		window,
	)
	if err != nil {
		return ProjectConversationWorkspaceGitGraph{}, err
	}

	labelsByCommit := buildProjectConversationWorkspaceLabelsByCommit(branchRefs, currentRef)
	commits := make([]ProjectConversationWorkspaceGitGraphCommit, 0, len(records))
	for _, record := range records {
		parentIDs := make([]string, 0, len(record.ParentIDs))
		for _, parentID := range record.ParentIDs {
			parentIDs = append(parentIDs, parentID.String())
		}
		labels := append([]ProjectConversationWorkspaceGitRefLabel(nil), labelsByCommit[record.CommitID.String()]...)
		sortProjectConversationWorkspaceGitLabels(labels)
		commits = append(commits, ProjectConversationWorkspaceGitGraphCommit{
			CommitID:      record.CommitID.String(),
			ShortCommitID: shortenProjectConversationGitCommit(record.CommitID.String()),
			ParentIDs:     parentIDs,
			Subject:       record.Subject,
			AuthorName:    record.AuthorName,
			AuthoredAt:    record.AuthoredAt.UTC(),
			Labels:        labels,
			Head:          currentRef.CommitID != "" && currentRef.CommitID == record.CommitID.String(),
		})
	}

	return ProjectConversationWorkspaceGitGraph{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Window:         window,
		Commits:        commits,
	}, nil
}

func (s *ProjectConversationService) CheckoutWorkspaceBranch(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceCheckoutInput,
) (ProjectConversationWorkspaceCheckoutResult, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceCheckoutResult{}, err
	}

	if input.ExpectedCleanWorkspace {
		summary, err := s.summarizeConversationWorkspaceRepo(ctx, resolved.machine, resolved.repo)
		if err != nil {
			return ProjectConversationWorkspaceCheckoutResult{}, err
		}
		if summary.Dirty {
			return ProjectConversationWorkspaceCheckoutResult{}, &ProjectConversationWorkspaceCheckoutPreconditionError{
				Reason: ProjectConversationWorkspaceCheckoutPreconditionDirtyWorkspace,
			}
		}
	}

	currentRef, branchRefs, err := s.readConversationWorkspaceGitRefState(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
	)
	if err != nil {
		return ProjectConversationWorkspaceCheckoutResult{}, err
	}

	createdLocalBranch := ""
	switch input.Target.Kind {
	case WorkspaceCheckoutTargetKindLocalBranch:
		if !projectConversationWorkspaceGitBranchRefExists(
			branchRefs,
			ProjectConversationWorkspaceBranchScopeLocal,
			input.Target.BranchName,
		) {
			return ProjectConversationWorkspaceCheckoutResult{}, ErrProjectConversationWorkspaceRepoNotFound
		}
		if currentRef.Kind == ProjectConversationWorkspaceCurrentRefKindBranch &&
			currentRef.BranchName == input.Target.BranchName.String() {
			return ProjectConversationWorkspaceCheckoutResult{
				ConversationID: resolved.conversationID,
				RepoPath:       resolved.repo.relativePath,
				CurrentRef:     currentRef,
			}, nil
		}
		if err := s.switchConversationWorkspaceLocalBranch(
			ctx,
			resolved.machine,
			resolved.repo.repoPath,
			input.Target.BranchName,
		); err != nil {
			return ProjectConversationWorkspaceCheckoutResult{}, err
		}
	case WorkspaceCheckoutTargetKindRemoteTrackingBranch:
		if !projectConversationWorkspaceGitBranchRefExists(
			branchRefs,
			ProjectConversationWorkspaceBranchScopeRemoteTracking,
			input.Target.BranchName,
		) {
			return ProjectConversationWorkspaceCheckoutResult{}, ErrProjectConversationWorkspaceRepoNotFound
		}
		localBranchName := deriveProjectConversationTrackingBranchName(
			input.Target.BranchName,
			input.Target.LocalBranchName,
		)
		if projectConversationWorkspaceGitBranchRefExists(
			branchRefs,
			ProjectConversationWorkspaceBranchScopeLocal,
			localBranchName,
		) {
			return ProjectConversationWorkspaceCheckoutResult{}, &ProjectConversationWorkspaceCheckoutPreconditionError{
				Reason:          ProjectConversationWorkspaceCheckoutPreconditionLocalBranchExists,
				RequestedBranch: input.Target.BranchName.String(),
				SuggestedBranch: localBranchName.String(),
			}
		}
		if err := s.createConversationWorkspaceTrackingBranch(
			ctx,
			resolved.machine,
			resolved.repo.repoPath,
			input.Target.BranchName,
			localBranchName,
		); err != nil {
			return ProjectConversationWorkspaceCheckoutResult{}, err
		}
		createdLocalBranch = localBranchName.String()
	case WorkspaceCheckoutTargetKindNewLocalBranch:
		if projectConversationWorkspaceGitBranchRefExists(
			branchRefs,
			ProjectConversationWorkspaceBranchScopeLocal,
			input.Target.BranchName,
		) {
			return ProjectConversationWorkspaceCheckoutResult{}, &ProjectConversationWorkspaceCheckoutPreconditionError{
				Reason:          ProjectConversationWorkspaceCheckoutPreconditionLocalBranchExists,
				RequestedBranch: input.Target.BranchName.String(),
				SuggestedBranch: input.Target.BranchName.String(),
			}
		}
		if err := s.createConversationWorkspaceLocalBranch(
			ctx,
			resolved.machine,
			resolved.repo.repoPath,
			input.Target.BranchName,
		); err != nil {
			return ProjectConversationWorkspaceCheckoutResult{}, err
		}
		createdLocalBranch = input.Target.BranchName.String()
	default:
		return ProjectConversationWorkspaceCheckoutResult{}, fmt.Errorf("unsupported checkout target %s", input.Target.Kind)
	}

	nextCurrentRef, _, err := s.readConversationWorkspaceGitRefState(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
	)
	if err != nil {
		return ProjectConversationWorkspaceCheckoutResult{}, err
	}
	return ProjectConversationWorkspaceCheckoutResult{
		ConversationID:     resolved.conversationID,
		RepoPath:           resolved.repo.relativePath,
		CurrentRef:         nextCurrentRef,
		CreatedLocalBranch: createdLocalBranch,
	}, nil
}

func (s *ProjectConversationService) readConversationWorkspaceGitRefState(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
) (ProjectConversationWorkspaceCurrentRef, []projectConversationWorkspaceGitBranchRef, error) {
	currentRef, err := s.readConversationWorkspaceCurrentRef(ctx, machine, repoPath)
	if err != nil {
		return ProjectConversationWorkspaceCurrentRef{}, nil, err
	}
	branchRefs, err := s.listConversationWorkspaceGitBranchRefs(ctx, machine, repoPath)
	if err != nil {
		return ProjectConversationWorkspaceCurrentRef{}, nil, err
	}
	return currentRef, branchRefs, nil
}

func (s *ProjectConversationService) readConversationWorkspaceCurrentRef(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
) (ProjectConversationWorkspaceCurrentRef, error) {
	branchOutput, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "symbolic-ref", "-q", "--short", "HEAD"},
		true,
	)
	if isProjectConversationGitWorkspaceUnavailableOutput(branchOutput) {
		return ProjectConversationWorkspaceCurrentRef{}, wrapProjectConversationWorkspaceUnavailable(branchOutput)
	}
	if err != nil && !projectConversationCommandExitedWithCode(err, 1) {
		return ProjectConversationWorkspaceCurrentRef{}, err
	}
	fullBranchOutput, fullBranchErr := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "symbolic-ref", "-q", "HEAD"},
		true,
	)
	if isProjectConversationGitWorkspaceUnavailableOutput(fullBranchOutput) {
		return ProjectConversationWorkspaceCurrentRef{}, wrapProjectConversationWorkspaceUnavailable(fullBranchOutput)
	}
	if fullBranchErr != nil && !projectConversationCommandExitedWithCode(fullBranchErr, 1) {
		return ProjectConversationWorkspaceCurrentRef{}, fullBranchErr
	}

	commitID := ""
	subject := ""
	commitOutput, commitErr := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "log", "-1", "--format=%H%x00%s", "HEAD"},
		true,
	)
	if isProjectConversationGitWorkspaceUnavailableOutput(commitOutput) {
		return ProjectConversationWorkspaceCurrentRef{}, wrapProjectConversationWorkspaceUnavailable(commitOutput)
	}
	if commitErr == nil {
		parts := strings.SplitN(string(commitOutput), "\x00", 2)
		commitID = strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			subject = strings.TrimSpace(parts[1])
		}
	} else if !isProjectConversationGitUnbornHeadOutput(commitOutput) && !projectConversationCommandExitedWithCode(commitErr, 1) {
		return ProjectConversationWorkspaceCurrentRef{}, commitErr
	}

	branchName := strings.TrimSpace(string(branchOutput))
	branchFullName := strings.TrimSpace(string(fullBranchOutput))
	if branchName != "" {
		return ProjectConversationWorkspaceCurrentRef{
			Kind:           ProjectConversationWorkspaceCurrentRefKindBranch,
			DisplayName:    branchName,
			CacheKey:       "branch:" + branchFullName,
			BranchName:     branchName,
			BranchFullName: branchFullName,
			CommitID:       commitID,
			ShortCommitID:  shortenProjectConversationGitCommit(commitID),
			Subject:        subject,
		}, nil
	}

	displayName := "detached HEAD"
	if commitID != "" {
		displayName = "detached@" + shortenProjectConversationGitCommit(commitID)
	}
	cacheKey := "detached"
	if commitID != "" {
		cacheKey = "detached:" + commitID
	}
	return ProjectConversationWorkspaceCurrentRef{
		Kind:          ProjectConversationWorkspaceCurrentRefKindDetached,
		DisplayName:   displayName,
		CacheKey:      cacheKey,
		CommitID:      commitID,
		ShortCommitID: shortenProjectConversationGitCommit(commitID),
		Subject:       subject,
	}, nil
}

func (s *ProjectConversationService) listConversationWorkspaceGitBranchRefs(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
) ([]projectConversationWorkspaceGitBranchRef, error) {
	output, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{
			"git",
			"-C",
			repoPath,
			"for-each-ref",
			"--format=%(refname)%00%(objectname)%00%(upstream:short)%00%(subject)",
			"refs/heads",
			"refs/remotes",
		},
		false,
	)
	if err != nil {
		if isProjectConversationGitWorkspaceUnavailableOutput(output) {
			return nil, wrapProjectConversationWorkspaceUnavailable(output)
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	refs := make([]projectConversationWorkspaceGitBranchRef, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\x00")
		if len(fields) != 4 {
			return nil, fmt.Errorf("git ref output is malformed")
		}
		fullName := strings.TrimSpace(fields[0])
		if strings.HasSuffix(fullName, "/HEAD") {
			continue
		}

		scope, name, err := parseProjectConversationWorkspaceBranchRefName(fullName)
		if err != nil {
			return nil, err
		}

		ref := projectConversationWorkspaceGitBranchRef{
			Name:         name,
			FullName:     fullName,
			Scope:        scope,
			CommitID:     WorkspaceCommitID(strings.TrimSpace(fields[1])),
			Subject:      strings.TrimSpace(fields[3]),
			UpstreamName: strings.TrimSpace(fields[2]),
		}
		if scope == ProjectConversationWorkspaceBranchScopeLocal && ref.UpstreamName != "" {
			ref.Ahead, ref.Behind = s.readConversationWorkspaceAheadBehind(
				ctx,
				machine,
				repoPath,
				name,
				WorkspaceBranchName(ref.UpstreamName),
			)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (s *ProjectConversationService) readConversationWorkspaceAheadBehind(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	localBranch WorkspaceBranchName,
	upstreamBranch WorkspaceBranchName,
) (ahead int, behind int) {
	output, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{
			"git",
			"-C",
			repoPath,
			"rev-list",
			"--left-right",
			"--count",
			localBranch.String() + "..." + upstreamBranch.String(),
		},
		false,
	)
	if err != nil {
		return 0, 0
	}
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) != 2 {
		return 0, 0
	}
	ahead, _ = strconv.Atoi(fields[0])
	behind, _ = strconv.Atoi(fields[1])
	return ahead, behind
}

func (s *ProjectConversationService) readConversationWorkspaceGitGraphRecords(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	window WorkspaceGitGraphWindow,
) ([]projectConversationWorkspaceGitGraphCommitRecord, error) {
	output, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{
			"git",
			"-C",
			repoPath,
			"log",
			"--topo-order",
			"--date=iso-strict",
			fmt.Sprintf("--max-count=%d", window.Limit),
			"--format=%H%x00%P%x00%an%x00%aI%x00%s",
			"--all",
		},
		true,
	)
	if err != nil {
		if isProjectConversationGitWorkspaceUnavailableOutput(output) {
			return nil, wrapProjectConversationWorkspaceUnavailable(output)
		}
		if isProjectConversationGitUnbornHeadOutput(output) || projectConversationCommandExitedWithCode(err, 1) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	records := make([]projectConversationWorkspaceGitGraphCommitRecord, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\x00")
		if len(fields) != 5 {
			return nil, fmt.Errorf("git graph output is malformed")
		}
		authoredAt, err := time.Parse(time.RFC3339, strings.TrimSpace(fields[3]))
		if err != nil {
			return nil, fmt.Errorf("parse git authored_at: %w", err)
		}
		parentIDs := make([]WorkspaceCommitID, 0)
		for _, parentID := range strings.Fields(strings.TrimSpace(fields[1])) {
			parentIDs = append(parentIDs, WorkspaceCommitID(parentID))
		}
		records = append(records, projectConversationWorkspaceGitGraphCommitRecord{
			CommitID:   WorkspaceCommitID(strings.TrimSpace(fields[0])),
			ParentIDs:  parentIDs,
			Subject:    strings.TrimSpace(fields[4]),
			AuthorName: strings.TrimSpace(fields[2]),
			AuthoredAt: authoredAt,
		})
	}
	return records, nil
}

func (s *ProjectConversationService) switchConversationWorkspaceLocalBranch(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	branchName WorkspaceBranchName,
) error {
	_, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "switch", "--quiet", branchName.String()},
		false,
	)
	return err
}

func (s *ProjectConversationService) createConversationWorkspaceTrackingBranch(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	remoteBranch WorkspaceBranchName,
	localBranch WorkspaceBranchName,
) error {
	_, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{
			"git",
			"-C",
			repoPath,
			"switch",
			"--quiet",
			"--track",
			"-c",
			localBranch.String(),
			remoteBranch.String(),
		},
		false,
	)
	return err
}

func (s *ProjectConversationService) createConversationWorkspaceLocalBranch(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	branchName WorkspaceBranchName,
) error {
	_, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "switch", "--quiet", "-c", branchName.String()},
		false,
	)
	return err
}

type ProjectConversationWorkspaceGitRemoteOpKind string

const (
	ProjectConversationWorkspaceGitRemoteOpFetch ProjectConversationWorkspaceGitRemoteOpKind = "fetch"
	ProjectConversationWorkspaceGitRemoteOpPull  ProjectConversationWorkspaceGitRemoteOpKind = "pull"
	ProjectConversationWorkspaceGitRemoteOpPush  ProjectConversationWorkspaceGitRemoteOpKind = "push"
)

type ProjectConversationWorkspaceGitRemoteOpInput struct {
	RepoPath WorkspaceRepoPath
	Op       ProjectConversationWorkspaceGitRemoteOpKind
}

type ProjectConversationWorkspaceGitRemoteOpResult struct {
	ConversationID uuid.UUID
	RepoPath       string
	Op             ProjectConversationWorkspaceGitRemoteOpKind
	Output         string
}

type ProjectConversationWorkspaceStageFileInput struct {
	RepoPath WorkspaceRepoPath
	Path     string
}

type ProjectConversationWorkspaceStageFileResult struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
}

type ProjectConversationWorkspaceStageAllInput struct {
	RepoPath WorkspaceRepoPath
}

type ProjectConversationWorkspaceStageAllResult struct {
	ConversationID uuid.UUID
	RepoPath       string
}

type ProjectConversationWorkspaceCommitInput struct {
	RepoPath WorkspaceRepoPath
	Message  string
}

type ProjectConversationWorkspaceCommitResult struct {
	ConversationID uuid.UUID
	RepoPath       string
	Output         string
}

type ProjectConversationWorkspaceDiscardFileInput struct {
	RepoPath WorkspaceRepoPath
	Path     string
}

type ProjectConversationWorkspaceDiscardFileResult struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
}

type ProjectConversationWorkspaceUnstageInput struct {
	RepoPath WorkspaceRepoPath
	Path     string
}

type ProjectConversationWorkspaceUnstageResult struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
}

func (s *ProjectConversationService) RunWorkspaceGitRemoteOp(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceGitRemoteOpInput,
) (ProjectConversationWorkspaceGitRemoteOpResult, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceGitRemoteOpResult{}, err
	}

	var args []string
	switch input.Op {
	case ProjectConversationWorkspaceGitRemoteOpFetch:
		args = []string{"git", "-C", resolved.repo.repoPath, "fetch", "--all", "--prune"}
	case ProjectConversationWorkspaceGitRemoteOpPull:
		args = []string{"git", "-C", resolved.repo.repoPath, "pull", "--ff-only"}
	case ProjectConversationWorkspaceGitRemoteOpPush:
		args = []string{"git", "-C", resolved.repo.repoPath, "push"}
	default:
		return ProjectConversationWorkspaceGitRemoteOpResult{}, fmt.Errorf("unknown git remote operation: %s", input.Op)
	}

	output, err := s.runProjectConversationGitCommand(ctx, resolved.machine, args, true)
	if err != nil {
		return ProjectConversationWorkspaceGitRemoteOpResult{}, err
	}

	return ProjectConversationWorkspaceGitRemoteOpResult{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Op:             input.Op,
		Output:         strings.TrimSpace(string(output)),
	}, nil
}

func (s *ProjectConversationService) StageWorkspaceFile(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceStageFileInput,
) (ProjectConversationWorkspaceStageFileResult, error) {
	resolved, relativePath, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		input.Path,
		false,
	)
	if err != nil {
		return ProjectConversationWorkspaceStageFileResult{}, err
	}
	if _, err := s.runProjectConversationGitCommand(
		ctx,
		resolved.machine,
		[]string{"git", "-C", resolved.repo.repoPath, "add", "--", relativePath},
		false,
	); err != nil {
		return ProjectConversationWorkspaceStageFileResult{}, err
	}
	return ProjectConversationWorkspaceStageFileResult{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Path:           relativePath,
	}, nil
}

func (s *ProjectConversationService) StageWorkspaceAll(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceStageAllInput,
) (ProjectConversationWorkspaceStageAllResult, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceStageAllResult{}, err
	}
	if _, err := s.runProjectConversationGitCommand(
		ctx,
		resolved.machine,
		[]string{"git", "-C", resolved.repo.repoPath, "add", "--all", "--", "."},
		false,
	); err != nil {
		return ProjectConversationWorkspaceStageAllResult{}, err
	}
	return ProjectConversationWorkspaceStageAllResult{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
	}, nil
}

func (s *ProjectConversationService) CommitWorkspace(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceCommitInput,
) (ProjectConversationWorkspaceCommitResult, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceCommitResult{}, err
	}
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return ProjectConversationWorkspaceCommitResult{}, fmt.Errorf("commit message must not be empty")
	}

	statusOutput, err := s.runProjectConversationGitCommand(
		ctx,
		resolved.machine,
		[]string{"git", "-C", resolved.repo.repoPath, "status", "--porcelain=v1", "-z"},
		false,
	)
	if err != nil {
		return ProjectConversationWorkspaceCommitResult{}, err
	}
	statuses, err := parseProjectConversationGitStatusEntries(statusOutput)
	if err != nil {
		return ProjectConversationWorkspaceCommitResult{}, err
	}
	hasStaged := false
	for _, status := range statuses {
		if projectConversationGitStatusHasStaged(status.code) {
			hasStaged = true
			break
		}
	}
	if !hasStaged {
		return ProjectConversationWorkspaceCommitResult{}, fmt.Errorf("there are no staged changes to commit")
	}

	output, err := s.runProjectConversationGitCommand(
		ctx,
		resolved.machine,
		[]string{
			"git",
			"-C",
			resolved.repo.repoPath,
			"-c",
			"user.name=OpenASE Workspace",
			"-c",
			"user.email=openase-workspace@local",
			"commit",
			"-m",
			message,
		},
		false,
	)
	if err != nil {
		return ProjectConversationWorkspaceCommitResult{}, err
	}
	return ProjectConversationWorkspaceCommitResult{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Output:         strings.TrimSpace(string(output)),
	}, nil
}

func (s *ProjectConversationService) DiscardWorkspaceFile(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceDiscardFileInput,
) (ProjectConversationWorkspaceDiscardFileResult, error) {
	resolved, relativePath, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		input.Path,
		false,
	)
	if err != nil {
		return ProjectConversationWorkspaceDiscardFileResult{}, err
	}
	status, ok, err := s.readConversationWorkspaceGitStatusEntry(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
		relativePath,
	)
	if err != nil {
		return ProjectConversationWorkspaceDiscardFileResult{}, err
	}
	if ok {
		if err := s.discardConversationWorkspaceGitStatusEntry(
			ctx,
			resolved.machine,
			resolved.repo.repoPath,
			status,
		); err != nil {
			return ProjectConversationWorkspaceDiscardFileResult{}, err
		}
	}
	return ProjectConversationWorkspaceDiscardFileResult{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Path:           relativePath,
	}, nil
}

func (s *ProjectConversationService) UnstageWorkspace(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceUnstageInput,
) (ProjectConversationWorkspaceUnstageResult, error) {
	resolved, relativePath, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		input.Path,
		strings.TrimSpace(input.Path) == "",
	)
	if err != nil {
		return ProjectConversationWorkspaceUnstageResult{}, err
	}

	statuses, err := s.readConversationWorkspaceGitStatusEntries(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
	)
	if err != nil {
		return ProjectConversationWorkspaceUnstageResult{}, err
	}

	targetPaths := make([]string, 0, len(statuses))
	headPaths := make([]string, 0, len(statuses)*2)
	for _, status := range statuses {
		if !projectConversationGitStatusHasStaged(status.code) {
			continue
		}
		if relativePath != "" && status.path != relativePath && status.oldPath != relativePath {
			continue
		}
		targetPaths = append(targetPaths, status.path)
		headPaths = append(headPaths, status.path)
		if status.oldPath != "" {
			headPaths = append(headPaths, status.oldPath)
		}
	}
	if len(targetPaths) == 0 {
		return ProjectConversationWorkspaceUnstageResult{
			ConversationID: resolved.conversationID,
			RepoPath:       resolved.repo.relativePath,
			Path:           relativePath,
		}, nil
	}

	headExists := true
	if _, err := s.runProjectConversationGitCommand(
		ctx,
		resolved.machine,
		[]string{"git", "-C", resolved.repo.repoPath, "rev-parse", "--verify", "HEAD"},
		false,
	); err != nil {
		if !projectConversationCommandExitedWithCode(err, 128) {
			return ProjectConversationWorkspaceUnstageResult{}, err
		}
		headExists = false
	}

	if headExists {
		args := make([]string, 0, 7+len(headPaths))
		args = append(args, "git", "-C", resolved.repo.repoPath, "reset", "--quiet", "HEAD", "--")
		args = append(args, headPaths...)
		if _, err := s.runProjectConversationGitCommand(ctx, resolved.machine, args, false); err != nil {
			return ProjectConversationWorkspaceUnstageResult{}, err
		}
	} else {
		for _, path := range targetPaths {
			if _, err := s.runProjectConversationGitCommand(
				ctx,
				resolved.machine,
				[]string{"git", "-C", resolved.repo.repoPath, "rm", "--quiet", "--cached", "--force", "--", path},
				true,
			); err != nil && !projectConversationCommandExitedWithCode(err, 128) {
				return ProjectConversationWorkspaceUnstageResult{}, err
			}
		}
	}

	return ProjectConversationWorkspaceUnstageResult{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Path:           relativePath,
	}, nil
}

type ProjectConversationWorkspaceCreateBranchInput struct {
	RepoPath   WorkspaceRepoPath
	BranchName WorkspaceBranchName
	StartPoint string
}

type ProjectConversationWorkspaceCreateBranchResult struct {
	ConversationID uuid.UUID
	RepoPath       string
	BranchName     string
}

func (s *ProjectConversationService) CreateWorkspaceBranch(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceCreateBranchInput,
) (ProjectConversationWorkspaceCreateBranchResult, error) {
	resolved, _, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		"",
		true,
	)
	if err != nil {
		return ProjectConversationWorkspaceCreateBranchResult{}, err
	}

	args := []string{"git", "-C", resolved.repo.repoPath, "branch", input.BranchName.String()}
	if input.StartPoint != "" {
		args = append(args, input.StartPoint)
	}

	_, err = s.runProjectConversationGitCommand(ctx, resolved.machine, args, false)
	if err != nil {
		return ProjectConversationWorkspaceCreateBranchResult{}, err
	}

	return ProjectConversationWorkspaceCreateBranchResult{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		BranchName:     input.BranchName.String(),
	}, nil
}

func (s *ProjectConversationService) discardConversationWorkspaceGitStatusEntry(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoPath string,
	status projectConversationGitStatusEntry,
) error {
	if status.oldPath != "" {
		if _, err := s.runProjectConversationGitCommand(
			ctx,
			machine,
			[]string{"git", "-C", repoPath, "restore", "--source=HEAD", "--staged", "--worktree", "--", status.oldPath},
			false,
		); err != nil {
			return err
		}
		if _, err := s.runProjectConversationGitCommand(
			ctx,
			machine,
			[]string{"git", "-C", repoPath, "rm", "--quiet", "--force", "--cached", "--", status.path},
			true,
		); err != nil && !projectConversationCommandExitedWithCode(err, 128) {
			return err
		}
		if _, err := s.runProjectConversationGitCommand(
			ctx,
			machine,
			[]string{"git", "-C", repoPath, "clean", "-f", "--", status.path},
			true,
		); err != nil {
			return err
		}
		return nil
	}

	if status.code == "??" || mapProjectConversationWorkspaceFileStatus(status.code) == ProjectConversationWorkspaceFileStatusAdded {
		if _, err := s.runProjectConversationGitCommand(
			ctx,
			machine,
			[]string{"git", "-C", repoPath, "rm", "--quiet", "--force", "--cached", "--", status.path},
			true,
		); err != nil && !projectConversationCommandExitedWithCode(err, 128) {
			return err
		}
		if _, err := s.runProjectConversationGitCommand(
			ctx,
			machine,
			[]string{"git", "-C", repoPath, "clean", "-f", "--", status.path},
			true,
		); err != nil {
			return err
		}
		return nil
	}

	_, err := s.runProjectConversationGitCommand(
		ctx,
		machine,
		[]string{"git", "-C", repoPath, "restore", "--source=HEAD", "--staged", "--worktree", "--", status.path},
		false,
	)
	return err
}

func mapProjectConversationWorkspaceBranchRef(
	ref projectConversationWorkspaceGitBranchRef,
	currentRef ProjectConversationWorkspaceCurrentRef,
) ProjectConversationWorkspaceBranchRef {
	item := ProjectConversationWorkspaceBranchRef{
		Name:          ref.Name.String(),
		FullName:      ref.FullName,
		Scope:         ref.Scope,
		CommitID:      ref.CommitID.String(),
		ShortCommitID: shortenProjectConversationGitCommit(ref.CommitID.String()),
		Subject:       ref.Subject,
		UpstreamName:  ref.UpstreamName,
		Ahead:         ref.Ahead,
		Behind:        ref.Behind,
	}
	if ref.Scope == ProjectConversationWorkspaceBranchScopeLocal {
		item.Current = currentRef.Kind == ProjectConversationWorkspaceCurrentRefKindBranch &&
			currentRef.BranchName == ref.Name.String()
	}
	if ref.Scope == ProjectConversationWorkspaceBranchScopeRemoteTracking {
		item.SuggestedLocalBranchName = deriveProjectConversationTrackingBranchName(ref.Name, nil).String()
	}
	return item
}

func buildProjectConversationWorkspaceLabelsByCommit(
	branchRefs []projectConversationWorkspaceGitBranchRef,
	currentRef ProjectConversationWorkspaceCurrentRef,
) map[string][]ProjectConversationWorkspaceGitRefLabel {
	labelsByCommit := make(map[string][]ProjectConversationWorkspaceGitRefLabel)
	for _, ref := range branchRefs {
		label := ProjectConversationWorkspaceGitRefLabel{
			Name:     ref.Name.String(),
			FullName: ref.FullName,
			Current:  false,
		}
		switch ref.Scope {
		case ProjectConversationWorkspaceBranchScopeLocal:
			label.Scope = ProjectConversationWorkspaceGitRefLabelScopeLocalBranch
			label.Current = currentRef.Kind == ProjectConversationWorkspaceCurrentRefKindBranch &&
				currentRef.BranchName == ref.Name.String()
		case ProjectConversationWorkspaceBranchScopeRemoteTracking:
			label.Scope = ProjectConversationWorkspaceGitRefLabelScopeRemoteTracking
		}
		labelsByCommit[ref.CommitID.String()] = append(labelsByCommit[ref.CommitID.String()], label)
	}
	if currentRef.CommitID != "" {
		labelsByCommit[currentRef.CommitID] = append(labelsByCommit[currentRef.CommitID], ProjectConversationWorkspaceGitRefLabel{
			Name:     "HEAD",
			FullName: "HEAD",
			Scope:    ProjectConversationWorkspaceGitRefLabelScopeHead,
			Current:  true,
		})
	}
	return labelsByCommit
}

func sortProjectConversationWorkspaceGitLabels(labels []ProjectConversationWorkspaceGitRefLabel) {
	sort.Slice(labels, func(i, j int) bool {
		left := labels[i]
		right := labels[j]
		if left.Scope != right.Scope {
			return left.Scope < right.Scope
		}
		if left.Current != right.Current {
			return left.Current
		}
		return left.Name < right.Name
	})
}

func parseProjectConversationWorkspaceBranchRefName(
	fullName string,
) (ProjectConversationWorkspaceBranchScope, WorkspaceBranchName, error) {
	refName := strings.TrimSpace(fullName)
	switch {
	case strings.HasPrefix(refName, "refs/heads/"):
		name, err := ParseWorkspaceBranchName(strings.TrimPrefix(refName, "refs/heads/"))
		return ProjectConversationWorkspaceBranchScopeLocal, name, err
	case strings.HasPrefix(refName, "refs/remotes/"):
		name, err := ParseWorkspaceBranchName(strings.TrimPrefix(refName, "refs/remotes/"))
		return ProjectConversationWorkspaceBranchScopeRemoteTracking, name, err
	default:
		return "", "", fmt.Errorf("unsupported git ref %s", refName)
	}
}

func deriveProjectConversationTrackingBranchName(
	remoteBranch WorkspaceBranchName,
	localBranchName *WorkspaceBranchName,
) WorkspaceBranchName {
	if localBranchName != nil && strings.TrimSpace(localBranchName.String()) != "" {
		return *localBranchName
	}
	parts := strings.SplitN(remoteBranch.String(), "/", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return WorkspaceBranchName(parts[1])
	}
	return remoteBranch
}

func projectConversationWorkspaceGitBranchRefExists(
	refs []projectConversationWorkspaceGitBranchRef,
	scope ProjectConversationWorkspaceBranchScope,
	name WorkspaceBranchName,
) bool {
	for _, ref := range refs {
		if ref.Scope == scope && ref.Name == name {
			return true
		}
	}
	return false
}

func wrapProjectConversationWorkspaceUnavailable(output []byte) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return workspaceinfra.ErrGitWorkspaceUnavailable
	}
	return fmt.Errorf("%w: %s", workspaceinfra.ErrGitWorkspaceUnavailable, message)
}

func isProjectConversationGitWorkspaceUnavailableOutput(output []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(trimmed, "not a git repository") ||
		strings.Contains(trimmed, "cannot change to") ||
		strings.Contains(trimmed, "no such file or directory")
}

func isProjectConversationGitUnbornHeadOutput(output []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(trimmed, "ambiguous argument 'head'") ||
		strings.Contains(trimmed, "bad revision 'head'") ||
		strings.Contains(trimmed, "unknown revision or path not in the working tree") ||
		strings.Contains(trimmed, "does not have any commits yet")
}

func projectConversationWorkspaceBranchDisplayName(
	currentRef ProjectConversationWorkspaceCurrentRef,
) string {
	if strings.TrimSpace(currentRef.DisplayName) != "" {
		return currentRef.DisplayName
	}
	return strings.TrimSpace(currentRef.BranchName)
}
