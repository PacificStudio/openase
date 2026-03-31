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
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gittransport "github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
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

type EnsureOperation string

const (
	EnsureOperationRead    EnsureOperation = "read"
	EnsureOperationWrite   EnsureOperation = "write"
	EnsureOperationExecute EnsureOperation = "execute"
)

type EnsureInput struct {
	ProjectRepoID uuid.UUID
	MachineID     uuid.UUID
	Operation     EnsureOperation
}

type DeleteInput struct {
	ProjectRepoID uuid.UUID
	MachineID     uuid.UUID
}

type Service struct {
	client     *rootent.Client
	logger     *slog.Logger
	now        func() time.Time
	sshPool    *sshinfra.Pool
	githubAuth githubauthservice.TokenResolver
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

func (s *Service) ConfigureSSHPool(pool *sshinfra.Pool) {
	if s == nil {
		return
	}
	s.sshPool = pool
}

func (s *Service) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	if s == nil {
		return
	}
	s.githubAuth = resolver
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

	target, err := s.resolveTarget(ctx, input.ProjectRepoID, input.MachineID, input.LocalPath, false)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	current, err := s.upsertMirrorState(ctx, target.projectRepo.ID, input.MachineID, target.localPath, entprojectrepomirror.StateProvisioning)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	transport, err := s.resolveRepositoryTransport(ctx, target.projectRepo)
	if err != nil {
		s.recordFailure(ctx, current.ID, target.localPath, err)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, err)
	}

	headCommit, verifyErr := s.inspectRepository(ctx, target.machine, target.localPath, transport, target.projectRepo.DefaultBranch)
	if verifyErr != nil {
		s.recordFailure(ctx, current.ID, target.localPath, verifyErr)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, verifyErr)
	}

	verifiedAt := s.now().UTC()
	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateReady).
		SetLocalPath(target.localPath).
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

	target, err := s.resolveTarget(ctx, input.ProjectRepoID, input.MachineID, input.LocalPath, true)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	current, err := s.upsertMirrorState(ctx, target.projectRepo.ID, input.MachineID, target.localPath, entprojectrepomirror.StateProvisioning)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	transport, err := s.resolveRepositoryTransport(ctx, target.projectRepo)
	if err != nil {
		s.recordFailure(ctx, current.ID, target.localPath, err)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, err)
	}

	headCommit, syncErr := s.syncRepository(ctx, target.machine, target.localPath, transport, target.projectRepo.DefaultBranch)
	if syncErr != nil {
		s.recordFailure(ctx, current.ID, target.localPath, syncErr)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, syncErr)
	}

	syncedAt := s.now().UTC()
	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateReady).
		SetLocalPath(target.localPath).
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

	current, projectRepo, machine, err := s.loadMirrorForUpdate(ctx, input.ProjectRepoID, input.MachineID)
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

	transport, err := s.resolveRepositoryTransport(ctx, projectRepo)
	if err != nil {
		s.recordFailure(ctx, current.ID, current.LocalPath, err)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, err)
	}

	headCommit, syncErr := s.syncRepository(ctx, machine, current.LocalPath, transport, projectRepo.DefaultBranch)
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

	current, projectRepo, machine, err := s.loadMirrorForUpdate(ctx, input.ProjectRepoID, input.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}
	if err := requireState(current.State, entprojectrepomirror.StateReady, entprojectrepomirror.StateStale, entprojectrepomirror.StateError); err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	transport, err := s.resolveRepositoryTransport(ctx, projectRepo)
	if err != nil {
		s.recordFailure(ctx, current.ID, current.LocalPath, err)
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: %v", ErrMirrorSyncFailed, err)
	}

	headCommit, verifyErr := s.inspectRepository(ctx, machine, current.LocalPath, transport, projectRepo.DefaultBranch)
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

func (s *Service) Ensure(ctx context.Context, input EnsureInput) (domain.ProjectRepoMirror, error) {
	if s == nil || s.client == nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: service unavailable", ErrInvalidInput)
	}
	if input.ProjectRepoID == uuid.Nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: project repo id must not be empty", ErrInvalidInput)
	}
	if input.MachineID == uuid.Nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: machine id must not be empty", ErrInvalidInput)
	}

	switch input.Operation {
	case EnsureOperationRead, EnsureOperationWrite, EnsureOperationExecute:
	default:
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: operation must be read, write, or execute", ErrInvalidInput)
	}

	current, _, _, err := s.loadMirrorForUpdate(ctx, input.ProjectRepoID, input.MachineID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return s.Prepare(ctx, PrepareInput{
				ProjectRepoID: input.ProjectRepoID,
				MachineID:     input.MachineID,
			})
		}
		return domain.ProjectRepoMirror{}, err
	}

	switch current.State {
	case entprojectrepomirror.StateProvisioning, entprojectrepomirror.StateSyncing, entprojectrepomirror.StateDeleting:
		return domain.ProjectRepoMirror{}, fmt.Errorf("%w: current state %q", ErrMirrorNotReady, current.State)
	case entprojectrepomirror.StateMissing:
		return s.Prepare(ctx, PrepareInput{
			ProjectRepoID: input.ProjectRepoID,
			MachineID:     input.MachineID,
			LocalPath:     current.LocalPath,
		})
	}

	if input.Operation == EnsureOperationRead {
		return s.Verify(ctx, VerifyInput{
			ProjectRepoID: input.ProjectRepoID,
			MachineID:     input.MachineID,
		})
	}

	return s.Sync(ctx, SyncInput{
		ProjectRepoID: input.ProjectRepoID,
		MachineID:     input.MachineID,
	})
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

	current, _, machine, err := s.loadMirrorForUpdate(ctx, input.ProjectRepoID, input.MachineID)
	if err != nil {
		return domain.ProjectRepoMirror{}, err
	}

	current, err = s.client.ProjectRepoMirror.UpdateOneID(current.ID).
		SetState(entprojectrepomirror.StateDeleting).
		Save(ctx)
	if err != nil {
		return domain.ProjectRepoMirror{}, fmt.Errorf("mark project repo mirror deleting: %w", err)
	}

	if err := s.deleteRepository(ctx, machine, current.LocalPath); err != nil {
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

type resolvedTarget struct {
	projectRepo *rootent.ProjectRepo
	machine     *rootent.Machine
	localPath   string
}

func (s *Service) resolveTarget(
	ctx context.Context,
	projectRepoID uuid.UUID,
	machineID uuid.UUID,
	rawLocalPath string,
	allowDefaultPath bool,
) (resolvedTarget, error) {
	if projectRepoID == uuid.Nil {
		return resolvedTarget{}, fmt.Errorf("%w: project repo id must not be empty", ErrInvalidInput)
	}
	if machineID == uuid.Nil {
		return resolvedTarget{}, fmt.Errorf("%w: machine id must not be empty", ErrInvalidInput)
	}

	projectRepo, err := s.client.ProjectRepo.Query().
		Where(entprojectrepo.ID(projectRepoID)).
		WithProject(func(query *rootent.ProjectQuery) {
			query.WithOrganization()
		}).
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			return resolvedTarget{}, ErrNotFound
		}
		return resolvedTarget{}, fmt.Errorf("load project repo: %w", err)
	}
	if projectRepo.Edges.Project == nil || projectRepo.Edges.Project.Edges.Organization == nil {
		return resolvedTarget{}, fmt.Errorf("project repo project organization edge must be loaded")
	}

	machine, err := s.client.Machine.Query().
		Where(entmachine.ID(machineID)).
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			return resolvedTarget{}, ErrNotFound
		}
		return resolvedTarget{}, fmt.Errorf("load machine before mirror update: %w", err)
	}

	localPath := strings.TrimSpace(rawLocalPath)
	if localPath == "" {
		if !allowDefaultPath {
			return resolvedTarget{}, fmt.Errorf("%w: local_path must not be empty", ErrInvalidInput)
		}
		localPath, err = deriveDefaultMirrorLocalPath(
			machine,
			projectRepo.Edges.Project.Edges.Organization.Slug,
			projectRepo.Edges.Project.Slug,
			projectRepo.Name,
		)
		if err != nil {
			return resolvedTarget{}, err
		}
	}

	parsedPath, err := parseAbsoluteLocalPath(localPath)
	if err != nil {
		return resolvedTarget{}, err
	}

	return resolvedTarget{
		projectRepo: projectRepo,
		machine:     machine,
		localPath:   parsedPath,
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

func (s *Service) loadMirrorForUpdate(ctx context.Context, projectRepoID uuid.UUID, machineID uuid.UUID) (*rootent.ProjectRepoMirror, *rootent.ProjectRepo, *rootent.Machine, error) {
	if projectRepoID == uuid.Nil {
		return nil, nil, nil, fmt.Errorf("%w: project repo id must not be empty", ErrInvalidInput)
	}
	if machineID == uuid.Nil {
		return nil, nil, nil, fmt.Errorf("%w: machine id must not be empty", ErrInvalidInput)
	}

	current, err := s.client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoID(projectRepoID),
			entprojectrepomirror.MachineID(machineID),
		).
		WithProjectRepo().
		WithMachine().
		Only(ctx)
	if err != nil {
		if rootent.IsNotFound(err) {
			return nil, nil, nil, ErrNotFound
		}
		return nil, nil, nil, fmt.Errorf("load project repo mirror: %w", err)
	}
	if current.Edges.ProjectRepo == nil {
		return nil, nil, nil, fmt.Errorf("project repo mirror project repo edge must be loaded")
	}
	if current.Edges.Machine == nil {
		return nil, nil, nil, fmt.Errorf("project repo mirror machine edge must be loaded")
	}

	return current, current.Edges.ProjectRepo, current.Edges.Machine, nil
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

func inspectExistingRepository(localPath string, expectedURL string, defaultBranch string) (string, error) {
	repository, err := git.PlainOpen(localPath)
	if err != nil {
		return "", fmt.Errorf("open repository %s: %w", localPath, err)
	}
	if strings.TrimSpace(expectedURL) != "" {
		if err := ensureOriginMatches(repository, expectedURL); err != nil {
			return "", err
		}
	}
	if err := ensureDefaultBranchResolvable(repository, defaultBranch); err != nil {
		return "", err
	}
	head, err := repository.Head()
	if err != nil {
		return "", fmt.Errorf("resolve repository head: %w", err)
	}
	return head.Hash().String(), nil
}

type repositoryTransportConfig struct {
	transportURL string
	auth         gittransport.AuthMethod
	githubToken  string
}

func syncLocalRepository(ctx context.Context, localPath string, transport repositoryTransportConfig, defaultBranch string) (string, error) {
	repository, err := cloneOrOpenRepository(ctx, localPath, transport)
	if err != nil {
		return "", err
	}
	if err := ensureOriginMatches(repository, transport.transportURL); err != nil {
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

func (s *Service) inspectRepository(
	ctx context.Context,
	machine *rootent.Machine,
	localPath string,
	transport repositoryTransportConfig,
	defaultBranch string,
) (string, error) {
	if machineIsLocal(machine) {
		return inspectExistingRepository(localPath, transport.transportURL, defaultBranch)
	}
	output, err := s.runRemoteMirrorScript(ctx, machine, buildRemoteInspectMirrorScript(localPath, transport.transportURL, defaultBranch))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (s *Service) syncRepository(
	ctx context.Context,
	machine *rootent.Machine,
	localPath string,
	transport repositoryTransportConfig,
	defaultBranch string,
) (string, error) {
	if machineIsLocal(machine) {
		return syncLocalRepository(ctx, localPath, transport, defaultBranch)
	}
	output, err := s.runRemoteMirrorScript(ctx, machine, buildRemoteSyncMirrorScript(localPath, transport.transportURL, defaultBranch, transport.githubToken))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (s *Service) deleteRepository(ctx context.Context, machine *rootent.Machine, localPath string) error {
	if machineIsLocal(machine) {
		return os.RemoveAll(localPath)
	}
	_, err := s.runRemoteMirrorScript(ctx, machine, buildRemoteDeleteMirrorScript(localPath))
	return err
}

func (s *Service) resolveRepositoryTransport(ctx context.Context, projectRepo *rootent.ProjectRepo) (repositoryTransportConfig, error) {
	if projectRepo == nil {
		return repositoryTransportConfig{}, fmt.Errorf("%w: project repo is required", ErrInvalidInput)
	}

	repositoryURL := strings.TrimSpace(projectRepo.RepositoryURL)
	transport := repositoryTransportConfig{transportURL: repositoryURL}
	normalizedURL, ok := githubauthdomain.NormalizeGitHubRepositoryURL(repositoryURL)
	if !ok {
		return transport, nil
	}

	transport.transportURL = normalizedURL
	if s == nil || s.githubAuth == nil {
		return repositoryTransportConfig{}, fmt.Errorf("resolve GitHub outbound credential: resolver unavailable")
	}

	resolved, err := s.githubAuth.ResolveProjectCredential(ctx, projectRepo.ProjectID)
	if err != nil {
		return repositoryTransportConfig{}, fmt.Errorf("resolve GitHub outbound credential: %w", err)
	}

	token := strings.TrimSpace(resolved.Token)
	if token == "" {
		return repositoryTransportConfig{}, errors.New("platform-managed GitHub outbound credential is not configured")
	}

	transport.auth = &githttp.BasicAuth{
		Username: "x-access-token",
		Password: token,
	}
	transport.githubToken = token
	return transport, nil
}

func cloneOrOpenRepository(ctx context.Context, localPath string, transport repositoryTransportConfig) (*git.Repository, error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if strings.TrimSpace(transport.transportURL) == "" {
				return nil, fmt.Errorf("clone repository %s: remote URL is required", localPath)
			}
			return git.PlainCloneContext(ctx, localPath, false, &git.CloneOptions{
				URL:  transport.transportURL,
				Auth: transport.auth,
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

	if err := alignOriginTransport(repository, transport.transportURL); err != nil {
		return nil, err
	}

	err = repository.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		RemoteURL:  transport.transportURL,
		Auth:       transport.auth,
	})
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
	if actualURL != strings.TrimSpace(expectedURL) && !sameGitHubRepositoryURL(actualURL, expectedURL) {
		return fmt.Errorf("origin remote URL mismatch: got %q want %q", actualURL, expectedURL)
	}
	return nil
}

func alignOriginTransport(repository *git.Repository, expectedURL string) error {
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
	if actualURL == strings.TrimSpace(expectedURL) || !sameGitHubRepositoryURL(actualURL, expectedURL) {
		return nil
	}

	cfg, err := repository.Config()
	if err != nil {
		return fmt.Errorf("load repository config: %w", err)
	}
	remoteCfg, ok := cfg.Remotes["origin"]
	if !ok || remoteCfg == nil {
		return errors.New("origin remote config is missing")
	}

	updated := *remoteCfg
	updated.URLs = []string{strings.TrimSpace(expectedURL)}
	cfg.Remotes["origin"] = &updated
	if err := repository.Storer.SetConfig(cfg); err != nil {
		return fmt.Errorf("set origin remote URL: %w", err)
	}
	return nil
}

func sameGitHubRepositoryURL(left string, right string) bool {
	leftRef, leftOK := githubauthdomain.ParseGitHubRepositoryURL(left)
	rightRef, rightOK := githubauthdomain.ParseGitHubRepositoryURL(right)
	return leftOK && rightOK && leftRef == rightRef
}

func ensureDefaultBranchResolvable(repository *git.Repository, defaultBranch string) error {
	branchName := strings.TrimSpace(defaultBranch)
	if branchName == "" {
		branchName = "main"
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)
	if _, err := repository.Reference(branchRefName, true); err == nil {
		return nil
	} else if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return fmt.Errorf("lookup local default branch %s: %w", branchName, err)
	}

	remoteRefName := plumbing.NewRemoteReferenceName("origin", branchName)
	if _, err := repository.Reference(remoteRefName, true); err == nil {
		return nil
	} else if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return fmt.Errorf("lookup remote default branch %s: %w", branchName, err)
	}

	return fmt.Errorf("resolve default branch %s: reference not found", branchName)
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

func machineIsLocal(machine *rootent.Machine) bool {
	if machine == nil {
		return true
	}
	return machine.Name == domain.LocalMachineName || machine.Host == domain.LocalMachineHost
}

func (s *Service) runRemoteMirrorScript(ctx context.Context, machine *rootent.Machine, script string) (string, error) {
	if s == nil || s.sshPool == nil {
		return "", fmt.Errorf("ssh pool unavailable for remote machine %s", machine.Name)
	}

	client, err := s.sshPool.Get(ctx, mapMachine(machine))
	if err != nil {
		return "", fmt.Errorf("get ssh client for machine %s: %w", machine.Name, err)
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("open ssh session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	output, err := session.CombinedOutput("sh -lc " + sshinfra.ShellQuote(script))
	if err != nil {
		return "", fmt.Errorf("run remote mirror command: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func buildRemoteInspectMirrorScript(localPath string, expectedURL string, defaultBranch string) string {
	lines := make([]string, 0, 7)
	lines = append(lines,
		"set -eu",
		"if [ ! -d "+sshinfra.ShellQuote(filepath.Join(localPath, ".git"))+" ]; then echo "+sshinfra.ShellQuote("repository path "+localPath+" is not a git repository")+" >&2; exit 1; fi",
	)
	lines = append(lines, remoteOriginCheckLines(localPath, expectedURL, false)...)
	lines = append(lines,
		"git -C "+sshinfra.ShellQuote(localPath)+" rev-parse --verify HEAD >/dev/null",
		buildRemoteResolveDefaultBranchLine(localPath, defaultBranch),
		"git -C "+sshinfra.ShellQuote(localPath)+" rev-parse HEAD",
	)
	return strings.Join(lines, "\n")
}

func buildRemoteSyncMirrorScript(localPath string, repositoryURL string, defaultBranch string, githubToken string) string {
	branchName := normalizedDefaultBranch(defaultBranch)
	lines := make([]string, 0, 18)
	lines = append(lines,
		"set -eu",
	)
	lines = append(lines, buildRemoteGitTransportLines(githubToken)...)
	lines = append(lines,
		"mkdir -p "+sshinfra.ShellQuote(filepath.Dir(localPath)),
		"if [ -e "+sshinfra.ShellQuote(localPath)+" ] && [ ! -d "+sshinfra.ShellQuote(filepath.Join(localPath, ".git"))+" ]; then echo "+sshinfra.ShellQuote("repository path "+localPath+" is not a git repository")+" >&2; exit 1; fi",
		"if [ ! -e "+sshinfra.ShellQuote(localPath)+" ]; then git_transport clone --branch "+sshinfra.ShellQuote(branchName)+" --single-branch "+sshinfra.ShellQuote(repositoryURL)+" "+sshinfra.ShellQuote(localPath)+"; fi",
	)
	lines = append(lines, remoteOriginCheckLines(localPath, repositoryURL, true)...)
	lines = append(lines,
		"git_transport -C "+sshinfra.ShellQuote(localPath)+" fetch origin",
		"git -C "+sshinfra.ShellQuote(localPath)+" rev-parse --verify "+sshinfra.ShellQuote("origin/"+branchName)+" >/dev/null",
		"git -C "+sshinfra.ShellQuote(localPath)+" checkout -B "+sshinfra.ShellQuote(branchName)+" "+sshinfra.ShellQuote("origin/"+branchName),
		"git -C "+sshinfra.ShellQuote(localPath)+" reset --hard "+sshinfra.ShellQuote("origin/"+branchName),
		"git -C "+sshinfra.ShellQuote(localPath)+" rev-parse HEAD",
	)
	return strings.Join(lines, "\n")
}

func buildRemoteDeleteMirrorScript(localPath string) string {
	return strings.Join([]string{
		"set -eu",
		"rm -rf " + sshinfra.ShellQuote(localPath),
	}, "\n")
}

func buildRemoteGitTransportLines(githubToken string) []string {
	lines := []string{"git_transport() { git \"$@\"; }"}
	if strings.TrimSpace(githubToken) == "" {
		return lines
	}

	return []string{
		"export GH_TOKEN=" + sshinfra.ShellQuote(githubToken),
		"unset SSH_AUTH_SOCK",
		"askpass_dir=$(mktemp -d)",
		"trap 'rm -rf \"$askpass_dir\"' EXIT",
		"askpass_script=\"$askpass_dir/git-askpass.sh\"",
		`cat >"$askpass_script" <<'EOF'
#!/bin/sh
case "$1" in
  *Username*) printf '%s\n' x-access-token ;;
  *) printf '%s\n' "$GH_TOKEN" ;;
esac
EOF`,
		"chmod 700 \"$askpass_script\"",
		"export GIT_TERMINAL_PROMPT=0",
		"export GIT_ASKPASS=\"$askpass_script\"",
		"git_transport() { git -c credential.helper= \"$@\"; }",
	}
}

func remoteOriginCheckLines(localPath string, expectedURL string, alignTransport bool) []string {
	if strings.TrimSpace(expectedURL) == "" {
		return nil
	}

	lines := []string{
		buildRemoteCanonicalGitHubRepoFunction(),
		"actual_origin=$(git -C " + sshinfra.ShellQuote(localPath) + " remote get-url origin)",
		"if [ \"$actual_origin\" != " + sshinfra.ShellQuote(expectedURL) + " ]; then",
		"  expected_github=$(canonical_github_repo " + sshinfra.ShellQuote(expectedURL) + " 2>/dev/null || true)",
		"  actual_github=$(canonical_github_repo \"$actual_origin\" 2>/dev/null || true)",
		"  if [ -n \"$expected_github\" ] && [ \"$expected_github\" = \"$actual_github\" ]; then",
	}
	if alignTransport {
		lines = append(lines,
			"    git -C "+sshinfra.ShellQuote(localPath)+" remote set-url origin "+sshinfra.ShellQuote(expectedURL),
			"    actual_origin="+sshinfra.ShellQuote(expectedURL),
		)
	}
	lines = append(lines,
		"    :",
		"  else",
		"    echo "+sshinfra.ShellQuote("origin remote URL mismatch")+" >&2; exit 1",
		"  fi",
		"fi",
	)
	return lines
}

func buildRemoteCanonicalGitHubRepoFunction() string {
	return `canonical_github_repo() {
  raw="$1"
  case "$raw" in
    https://github.com/*)
      repo="${raw#https://github.com/}"
      ;;
    git@github.com:*)
      repo="${raw#git@github.com:}"
      ;;
    ssh://git@github.com/*)
      repo="${raw#ssh://git@github.com/}"
      ;;
    *)
      return 1
      ;;
  esac
  repo="${repo%.git}"
  repo="${repo#/}"
  case "$repo" in
    */*)
      owner="${repo%%/*}"
      name="${repo#*/}"
      case "$name" in
        */*)
          return 1
          ;;
      esac
      printf '%s/%s\n' "$(printf '%s' "$owner" | tr '[:upper:]' '[:lower:]')" "$(printf '%s' "$name" | tr '[:upper:]' '[:lower:]')"
      ;;
    *)
      return 1
      ;;
  esac
}`
}

func buildRemoteResolveDefaultBranchLine(localPath string, defaultBranch string) string {
	branchName := normalizedDefaultBranch(defaultBranch)
	return "if ! git -C " + sshinfra.ShellQuote(localPath) + " rev-parse --verify " + sshinfra.ShellQuote("refs/heads/"+branchName) + " >/dev/null 2>&1 && ! git -C " + sshinfra.ShellQuote(localPath) + " rev-parse --verify " + sshinfra.ShellQuote("refs/remotes/origin/"+branchName) + " >/dev/null 2>&1; then echo " + sshinfra.ShellQuote("default branch "+branchName+" is not available") + " >&2; exit 1; fi"
}

func normalizedDefaultBranch(defaultBranch string) string {
	branchName := strings.TrimSpace(defaultBranch)
	if branchName == "" {
		return "main"
	}
	return branchName
}

func mapMachine(machine *rootent.Machine) domain.Machine {
	if machine == nil {
		return domain.Machine{}
	}
	return domain.Machine{
		ID:            machine.ID,
		Name:          machine.Name,
		Host:          machine.Host,
		Port:          machine.Port,
		SSHUser:       optionalString(machine.SSHUser),
		SSHKeyPath:    optionalString(machine.SSHKeyPath),
		WorkspaceRoot: optionalString(machine.WorkspaceRoot),
	}
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
