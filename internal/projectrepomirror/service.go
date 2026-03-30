package projectrepomirror

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	rootent "github.com/BetterAndBetterII/openase/ent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entprojectrepomirror "github.com/BetterAndBetterII/openase/ent/projectrepomirror"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("project repo mirror not found")
	ErrInvalidInput      = errors.New("project repo mirror input is invalid")
	ErrMirrorNotReady    = errors.New("project repo mirror is not ready")
	ErrMirrorSyncFailed  = errors.New("project repo mirror sync failed")
	ErrInvalidTransition = errors.New("project repo mirror state transition is invalid")
)

type ListFilter struct {
	ProjectID     uuid.UUID
	ProjectRepoID uuid.UUID
	MachineID     *uuid.UUID
}

type RegisterExistingInput struct {
	ProjectRepoID uuid.UUID
	MachineID     uuid.UUID
	LocalPath     string
}

type PrepareInput struct {
	ProjectRepoID uuid.UUID
	MachineID     uuid.UUID
	LocalPath     string
}

type SyncInput struct {
	ProjectRepoID uuid.UUID
	MachineID     uuid.UUID
}

type VerifyInput struct {
	ProjectRepoID uuid.UUID
	MachineID     uuid.UUID
}

type DeleteInput struct {
	ProjectRepoID uuid.UUID
	MachineID     uuid.UUID
}

type Service struct {
	client *rootent.Client
	logger *slog.Logger
	now    func() time.Time
}

func NewService(client *rootent.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &Service{
		client: client,
		logger: logger.With("component", "project-repo-mirror"),
		now:    time.Now,
	}
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]domain.ProjectRepoMirror, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}
	if filter.ProjectID == uuid.Nil {
		return nil, fmt.Errorf("%w: project id must not be empty", ErrInvalidInput)
	}
	if filter.ProjectRepoID == uuid.Nil {
		return nil, fmt.Errorf("%w: project repo id must not be empty", ErrInvalidInput)
	}

	exists, err := s.client.ProjectRepo.Query().
		Where(
			entprojectrepo.ID(filter.ProjectRepoID),
			entprojectrepo.ProjectID(filter.ProjectID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check project repo before listing mirrors: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	query := s.client.ProjectRepoMirror.Query().
		Where(entprojectrepomirror.ProjectRepoID(filter.ProjectRepoID)).
		WithProjectRepo().
		Order(
			entprojectrepomirror.ByMachineID(),
			entprojectrepomirror.ByCreatedAt(),
		)
	if filter.MachineID != nil {
		query.Where(entprojectrepomirror.MachineID(*filter.MachineID))
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project repo mirrors: %w", err)
	}
	return mapProjectRepoMirrors(items), nil
}

func (s *Service) RegisterExisting(ctx context.Context, input RegisterExistingInput) (mirror domain.ProjectRepoMirror, err error) {
	if s == nil || s.client == nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}

	projectRepo, localPath, err := s.parseRegisterExistingTarget(ctx, input.ProjectRepoID, input.MachineID, input.LocalPath)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	current, err := s.upsertMirrorState(ctx, projectRepo.ID, input.MachineID, localPath, entprojectrepomirror.StateProvisioning)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	headCommit, verifyErr := inspectExistingRepository(localPath, projectRepo.RepositoryURL)
	if verifyErr != nil {
		s.recordFailure(ctx, current.ID, localPath, verifyErr)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, verifyErr)
	}

	verifiedAt := s.now().UTC()
	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateReady).
		SetLocalPath(localPath).
		SetHeadCommit(headCommit).
		SetLastSyncedAt(verifiedAt).
		SetLastVerifiedAt(verifiedAt).
		ClearLastError().
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("persist registered project repo mirror: %w", err)
	}

	current, err = s.loadMirror(ctx, current.ProjectRepoID, current.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	return mapProjectRepoMirror(current), nil
}

func (s *Service) Prepare(ctx context.Context, input PrepareInput) (mirror domain.ProjectRepoMirror, err error) {
	if s == nil || s.client == nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}

	projectRepo, localPath, err := s.parsePrepareTarget(ctx, input.ProjectRepoID, input.MachineID, input.LocalPath)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	current, err := s.upsertMirrorState(ctx, projectRepo.ID, input.MachineID, localPath, entprojectrepomirror.StateProvisioning)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	headCommit, syncErr := syncRepository(ctx, localPath, projectRepo.RepositoryURL, projectRepo.DefaultBranch)
	if syncErr != nil {
		s.recordFailure(ctx, current.ID, localPath, syncErr)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, syncErr)
	}

	syncedAt := s.now().UTC()
	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateReady).
		SetLocalPath(localPath).
		SetHeadCommit(headCommit).
		SetLastSyncedAt(syncedAt).
		SetLastVerifiedAt(syncedAt).
		ClearLastError().
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("persist prepared project repo mirror: %w", err)
	}

	current, err = s.loadMirror(ctx, current.ProjectRepoID, current.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	return mapProjectRepoMirror(current), nil
}

func (s *Service) Sync(ctx context.Context, input SyncInput) (mirror domain.ProjectRepoMirror, err error) {
	if s == nil || s.client == nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}

	current, projectRepo, err := s.loadMirrorForUpdate(ctx, input.ProjectRepoID, input.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	if err := requireState(current.State, entprojectrepomirror.StateReady, entprojectrepomirror.StateStale, entprojectrepomirror.StateError); err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateSyncing).
		ClearLastError().
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("mark project repo mirror syncing: %w", err)
	}

	headCommit, syncErr := syncRepository(ctx, current.LocalPath, projectRepo.RepositoryURL, projectRepo.DefaultBranch)
	if syncErr != nil {
		s.recordFailure(ctx, current.ID, current.LocalPath, syncErr)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, syncErr)
	}

	syncedAt := s.now().UTC()
	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateReady).
		SetHeadCommit(headCommit).
		SetLastSyncedAt(syncedAt).
		SetLastVerifiedAt(syncedAt).
		ClearLastError().
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("persist synced project repo mirror: %w", err)
	}

	current, err = s.loadMirror(ctx, current.ProjectRepoID, current.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	return mapProjectRepoMirror(current), nil
}

func (s *Service) Verify(ctx context.Context, input VerifyInput) (mirror domain.ProjectRepoMirror, err error) {
	if s == nil || s.client == nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}

	current, projectRepo, err := s.loadMirrorForUpdate(ctx, input.ProjectRepoID, input.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	if err := requireState(current.State, entprojectrepomirror.StateReady, entprojectrepomirror.StateStale, entprojectrepomirror.StateError); err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	headCommit, verifyErr := inspectExistingRepository(current.LocalPath, projectRepo.RepositoryURL)
	if verifyErr != nil {
		s.recordFailure(ctx, current.ID, current.LocalPath, verifyErr)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, verifyErr)
	}

	verifiedAt := s.now().UTC()
	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateReady).
		SetHeadCommit(headCommit).
		SetLastVerifiedAt(verifiedAt).
		ClearLastError().
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("persist verified project repo mirror: %w", err)
	}

	current, err = s.loadMirror(ctx, current.ProjectRepoID, current.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	return mapProjectRepoMirror(current), nil
}

func (s *Service) MarkStaleMirrors(ctx context.Context, staleAfter time.Duration) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}
	if staleAfter <= 0 {
		return fmt.Errorf("%w: stale_after must be greater than zero", ErrInvalidInput)
	}

	cutoff := s.now().UTC().Add(-staleAfter)
	_, err := s.client.ProjectRepoMirror.Update().
		Where(
			entprojectrepomirror.StateEQ(entprojectrepomirror.StateReady),
			entprojectrepomirror.Or(
				entprojectrepomirror.LastSyncedAtLT(cutoff),
				entprojectrepomirror.LastSyncedAtIsNil(),
			),
		).
		SetState(entprojectrepomirror.StateStale).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark stale project repo mirrors: %w", err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, input DeleteInput) (mirror domain.ProjectRepoMirror, err error) {
	if s == nil || s.client == nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}

	current, _, err := s.loadMirrorForUpdate(ctx, input.ProjectRepoID, input.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateDeleting).
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("mark project repo mirror deleting: %w", err)
	}

	if err := os.RemoveAll(current.LocalPath); err != nil {
		s.recordFailure(ctx, current.ID, current.LocalPath, err)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, err)
	}

	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateMissing).
		ClearHeadCommit().
		ClearLastSyncedAt().
		ClearLastVerifiedAt().
		ClearLastError().
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("persist deleted project repo mirror: %w", err)
	}

	current, err = s.loadMirror(ctx, current.ProjectRepoID, current.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	return mapProjectRepoMirror(current), nil
}

type mirrorTarget struct {
	projectRepo *rootent.ProjectRepo
	machine     *rootent.Machine
	orgSlug     string
	projectSlug string
}

func (s *Service) parseRegisterExistingTarget(ctx context.Context, projectRepoID uuid.UUID, machineID uuid.UUID, localPath string) (*rootent.ProjectRepo, string, error) {
	target, err := s.loadTarget(ctx, projectRepoID, machineID)
	if err != nil {
		return nil, "", err
	}
	parsedPath, err := parseAbsoluteLocalPath(localPath)
	if err != nil {
		return nil, "", err
	}
	return target.projectRepo, parsedPath, nil
}

func (s *Service) parsePrepareTarget(ctx context.Context, projectRepoID uuid.UUID, machineID uuid.UUID, localPath string) (*rootent.ProjectRepo, string, error) {
	target, err := s.loadTarget(ctx, projectRepoID, machineID)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(localPath) != "" {
		parsedPath, parseErr := parseAbsoluteLocalPath(localPath)
		if parseErr != nil {
			return nil, "", parseErr
		}
		return target.projectRepo, parsedPath, nil
	}

	defaultPath, err := deriveDefaultMirrorLocalPath(target.machine, target.orgSlug, target.projectSlug, target.projectRepo.Name)
	if err != nil {
		return nil, "", err
	}
	return target.projectRepo, defaultPath, nil
}

func (s *Service) loadTarget(ctx context.Context, projectRepoID uuid.UUID, machineID uuid.UUID) (*mirrorTarget, error) {
	if projectRepoID == uuid.Nil {
		return nil, fmt.Errorf("%w: project repo id must not be empty", ErrInvalidInput)
	}
	if machineID == uuid.Nil {
		return nil, fmt.Errorf("%w: machine id must not be empty", ErrInvalidInput)
	}

	projectRepo, err := s.client.ProjectRepo.Query().
		Where(entprojectrepo.ID(projectRepoID)).
		WithProject(func(query *rootent.ProjectQuery) {
			query.WithOrganization()
		}).
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load project repo: %w", err)
	}
	if projectRepo.Edges.Project == nil || projectRepo.Edges.Project.Edges.Organization == nil {
		return nil, fmt.Errorf("project repo project organization edge must be loaded")
	}

	machine, err := s.client.Machine.Query().
		Where(entmachine.ID(machineID)).
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load machine before mirror update: %w", err)
	}

	return &mirrorTarget{
		projectRepo: projectRepo,
		machine:     machine,
		orgSlug:     projectRepo.Edges.Project.Edges.Organization.Slug,
		projectSlug: projectRepo.Edges.Project.Slug,
	}, nil
}

func (s *Service) upsertMirrorState(ctx context.Context, projectRepoID uuid.UUID, machineID uuid.UUID, localPath string, state entprojectrepomirror.State) (*rootent.ProjectRepoMirror, error) {
	current, err := s.client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoID(projectRepoID),
			entprojectrepomirror.MachineID(machineID),
		).
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			item, createErr := s.client.ProjectRepoMirror.Create().
				SetProjectRepoID(projectRepoID).
				SetMachineID(machineID).
				SetLocalPath(localPath).
				SetState(state).
				Save(ctx)
			if createErr != nil {
				return nil, fmt.Errorf("create project repo mirror: %w", createErr)
			}
			return item, nil
		}
		return nil, fmt.Errorf("load project repo mirror before update: %w", err)
	}

	if err := requireState(current.State, entprojectrepomirror.StateMissing, entprojectrepomirror.StateReady, entprojectrepomirror.StateStale, entprojectrepomirror.StateError); err != nil {
		return nil, err
	}

	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetLocalPath(localPath).
		SetState(state).
		ClearLastError().
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("mark project repo mirror %s: %w", state, err)
	}
	return current, nil
}

func (s *Service) loadMirrorForUpdate(ctx context.Context, projectRepoID uuid.UUID, machineID uuid.UUID) (*rootent.ProjectRepoMirror, *rootent.ProjectRepo, error) {
	if projectRepoID == uuid.Nil {
		return nil, nil, fmt.Errorf("%w: project repo id must not be empty", ErrInvalidInput)
	}
	if machineID == uuid.Nil {
		return nil, nil, fmt.Errorf("%w: machine id must not be empty", ErrInvalidInput)
	}

	current, err := s.client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoID(projectRepoID),
			entprojectrepomirror.MachineID(machineID),
		).
		WithProjectRepo().
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("load project repo mirror: %w", err)
	}
	if current.Edges.ProjectRepo == nil {
		return nil, nil, fmt.Errorf("project repo mirror project repo edge must be loaded")
	}

	return current, current.Edges.ProjectRepo, nil
}

func (s *Service) loadMirror(ctx context.Context, projectRepoID uuid.UUID, machineID uuid.UUID) (*rootent.ProjectRepoMirror, error) {
	current, err := s.client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoID(projectRepoID),
			entprojectrepomirror.MachineID(machineID),
		).
		WithProjectRepo().
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reload project repo mirror: %w", err)
	}
	return current, nil
}

func (s *Service) recordFailure(ctx context.Context, mirrorID uuid.UUID, localPath string, failure error) {
	if mirrorID == uuid.Nil || failure == nil {
		return
	}

	if _, err := s.client.ProjectRepoMirror.UpdateOneID(mirrorID).
		SetState(entprojectrepomirror.StateError).
		SetLocalPath(localPath).
		SetLastError(failure.Error()).
		Save(ctx); err != nil {
		s.logger.Error("persist project repo mirror failure", "mirror_id", mirrorID, "error", err)
	}
}

func parseAbsoluteLocalPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: local_path must not be empty", ErrInvalidInput)
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("%w: local_path must be absolute", ErrInvalidInput)
	}
	return cleaned, nil
}

func requireState(current entprojectrepomirror.State, allowed ...entprojectrepomirror.State) error {
	for _, candidate := range allowed {
		if current == candidate {
			return nil
		}
	}
	return fmt.Errorf("%w: current state %q", ErrInvalidTransition, current)
}

func inspectExistingRepository(localPath string, expectedURL string) (string, error) {
	repository, err := git.PlainOpen(localPath)
	if err != nil {
		return "", fmt.Errorf("open repository %s: %w", localPath, err)
	}
	if strings.TrimSpace(expectedURL) != "" {
		if err := ensureOriginMatches(repository, expectedURL); err != nil {
			return "", err
		}
	}
	head, err := repository.Head()
	if err != nil {
		return "", fmt.Errorf("resolve repository head: %w", err)
	}
	return head.Hash().String(), nil
}

func syncRepository(ctx context.Context, localPath string, repositoryURL string, defaultBranch string) (string, error) {
	repository, err := cloneOrOpenRepository(ctx, localPath, repositoryURL)
	if err != nil {
		return "", err
	}
	if err := ensureOriginMatches(repository, repositoryURL); err != nil {
		return "", err
	}
	if err := ensureDefaultBranchCheckedOut(repository, defaultBranch); err != nil {
		return "", err
	}
	head, err := repository.Head()
	if err != nil {
		return "", fmt.Errorf("resolve repository head: %w", err)
	}
	return head.Hash().String(), nil
}

func cloneOrOpenRepository(ctx context.Context, localPath string, repositoryURL string) (*git.Repository, error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if strings.TrimSpace(repositoryURL) == "" {
				return nil, fmt.Errorf("clone repository %s: remote URL is required", localPath)
			}
			return git.PlainCloneContext(ctx, localPath, false, &git.CloneOptions{
				URL: repositoryURL,
			})
		}
		return nil, fmt.Errorf("stat repository path %s: %w", localPath, err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("repository path %s is not a directory", localPath)
	}

	repository, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("open repository %s: %w", localPath, err)
	}

	err = repository.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin"})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("fetch repository %s: %w", localPath, err)
	}
	return repository, nil
}

func ensureOriginMatches(repository *git.Repository, expectedURL string) error {
	if strings.TrimSpace(expectedURL) == "" {
		return nil
	}

	remote, err := repository.Remote("origin")
	if err != nil {
		return fmt.Errorf("load origin remote: %w", err)
	}
	if len(remote.Config().URLs) == 0 {
		return errors.New("origin remote has no configured URLs")
	}
	actualURL := strings.TrimSpace(remote.Config().URLs[0])
	if actualURL != strings.TrimSpace(expectedURL) {
		return fmt.Errorf("origin remote URL mismatch: got %q want %q", actualURL, expectedURL)
	}
	return nil
}

func ensureDefaultBranchCheckedOut(repository *git.Repository, defaultBranch string) error {
	branchName := strings.TrimSpace(defaultBranch)
	if branchName == "" {
		branchName = "main"
	}

	remoteRefName := plumbing.NewRemoteReferenceName("origin", branchName)
	remoteRef, err := repository.Reference(remoteRefName, true)
	if err != nil {
		return fmt.Errorf("resolve remote default branch %s: %w", branchName, err)
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)
	if _, err := repository.Reference(branchRefName, true); err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("lookup local default branch %s: %w", branchName, err)
		}
		if err := repository.Storer.SetReference(plumbing.NewHashReference(branchRefName, remoteRef.Hash())); err != nil {
			return fmt.Errorf("create local default branch %s: %w", branchName, err)
		}
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("load worktree: %w", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: branchRefName, Force: true}); err != nil {
		return fmt.Errorf("checkout default branch %s: %w", branchName, err)
	}
	if err := worktree.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: remoteRef.Hash()}); err != nil {
		return fmt.Errorf("reset default branch %s: %w", branchName, err)
	}
	return nil
}

func mapProjectRepoMirrors(items []*rootent.ProjectRepoMirror) []domain.ProjectRepoMirror {
	response := make([]domain.ProjectRepoMirror, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectRepoMirror(item))
	}
	return response
}

func mapProjectRepoMirror(item *rootent.ProjectRepoMirror) domain.ProjectRepoMirror {
	if item == nil {
		return domain.ProjectRepoMirror{}
	}

	var projectID uuid.UUID
	if item.Edges.ProjectRepo != nil {
		projectID = item.Edges.ProjectRepo.ProjectID
	}

	return domain.ProjectRepoMirror{
		ID:             item.ID,
		ProjectID:      projectID,
		ProjectRepoID:  item.ProjectRepoID,
		MachineID:      item.MachineID,
		LocalPath:      item.LocalPath,
		State:          toDomainState(item.State),
		HeadCommit:     optionalString(item.HeadCommit),
		LastSyncedAt:   optionalTime(item.LastSyncedAt),
		LastVerifiedAt: optionalTime(item.LastVerifiedAt),
		LastError:      optionalString(item.LastError),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func toDomainState(value entprojectrepomirror.State) domain.ProjectRepoMirrorState {
	switch value {
	case entprojectrepomirror.StateMissing:
		return domain.ProjectRepoMirrorStateMissing
	case entprojectrepomirror.StateProvisioning:
		return domain.ProjectRepoMirrorStateProvisioning
	case entprojectrepomirror.StateReady:
		return domain.ProjectRepoMirrorStateReady
	case entprojectrepomirror.StateStale:
		return domain.ProjectRepoMirrorStateStale
	case entprojectrepomirror.StateSyncing:
		return domain.ProjectRepoMirrorStateSyncing
	case entprojectrepomirror.StateError:
		return domain.ProjectRepoMirrorStateError
	case entprojectrepomirror.StateDeleting:
		return domain.ProjectRepoMirrorStateDeleting
	default:
		return domain.ProjectRepoMirrorState(value)
	}
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	result := value
	return &result
}

func optionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}
