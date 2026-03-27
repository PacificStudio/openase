package workflow

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	"github.com/google/uuid"
)

type projectStorage struct {
	projectID    uuid.UUID
	repoRoot     string
	harnessRoot  string
	skillRoot    string
	registry     *harnessRegistry
	hookExecutor *workflowHookExecutor
}

func newProjectStorage(projectID uuid.UUID, repoRoot string, service *Service) (*projectStorage, error) {
	harnessRoot := filepath.Join(repoRoot, ".openase", "harnesses")
	skillRoot := filepath.Join(repoRoot, ".openase", "skills")
	if err := os.MkdirAll(skillRoot, 0o750); err != nil {
		return nil, fmt.Errorf("create skill root: %w", err)
	}

	storage := &projectStorage{
		projectID:    projectID,
		repoRoot:     repoRoot,
		harnessRoot:  harnessRoot,
		skillRoot:    skillRoot,
		hookExecutor: newWorkflowHookExecutor(repoRoot, service.logger),
	}

	registry, err := newHarnessRegistry(harnessRoot, service.logger, func(event harnessReloadEvent) {
		event.ProjectID = projectID
		service.handleHarnessReload(event)
	})
	if err != nil {
		return nil, err
	}
	storage.registry = registry

	return storage, nil
}

func (s *projectStorage) Close() error {
	if s == nil || s.registry == nil {
		return nil
	}

	return s.registry.Close()
}

func (s *Service) storageForProject(ctx context.Context, projectID uuid.UUID) (*projectStorage, error) {
	repoRoot, err := s.resolvePrimaryRepoRoot(ctx, projectID)
	if err != nil {
		return nil, err
	}

	s.storageMu.Lock()
	defer s.storageMu.Unlock()

	if existing, ok := s.storages[projectID]; ok && existing.repoRoot == repoRoot {
		return existing, nil
	}

	storage, err := newProjectStorage(projectID, repoRoot, s)
	if err != nil {
		return nil, err
	}

	if existing := s.storages[projectID]; existing != nil {
		if closeErr := existing.Close(); closeErr != nil {
			s.logger.Error("close replaced project storage", "error", closeErr, "project_id", projectID)
		}
	}
	s.storages[projectID] = storage

	return storage, nil
}

func (s *Service) resolvePrimaryRepoRoot(ctx context.Context, projectID uuid.UUID) (string, error) {
	repoItem, err := s.client.ProjectRepo.Query().
		Where(
			entprojectrepo.ProjectID(projectID),
			entprojectrepo.IsPrimary(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", ErrPrimaryRepoUnavailable
		}
		return "", fmt.Errorf("get primary project repo: %w", err)
	}

	for _, candidate := range []string{repoItem.ClonePath, repoItem.RepositoryURL} {
		repoRoot, ok, candidateErr := resolveLocalProjectRepoRoot(candidate)
		if candidateErr != nil {
			return "", fmt.Errorf("%w: %s", ErrPrimaryRepoUnavailable, candidateErr)
		}
		if ok {
			return repoRoot, nil
		}
	}

	return "", fmt.Errorf(
		"%w: primary repo %q must expose a local repository path via clone_path or repository_url",
		ErrPrimaryRepoUnavailable,
		repoItem.Name,
	)
}

func resolveLocalProjectRepoRoot(raw string) (string, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false, nil
	}

	if filepath.IsAbs(trimmed) {
		repoRoot, err := DetectRepoRoot(filepath.Clean(trimmed))
		if err != nil {
			return "", false, err
		}
		return repoRoot, true, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false, fmt.Errorf("parse project repo location %q: %w", trimmed, err)
	}
	if parsed.Scheme != "file" {
		return "", false, nil
	}

	repoPath, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", false, fmt.Errorf("decode project repo file URI %q: %w", trimmed, err)
	}
	if repoPath == "" {
		return "", false, fmt.Errorf("project repo file URI %q must include a path", trimmed)
	}

	repoRoot, err := DetectRepoRoot(filepath.Clean(filepath.FromSlash(repoPath)))
	if err != nil {
		return "", false, err
	}
	return repoRoot, true, nil
}
