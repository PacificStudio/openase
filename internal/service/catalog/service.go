package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	repository "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/google/uuid"
)

var (
	ErrNotFound = repository.ErrNotFound
	ErrConflict = repository.ErrConflict
)

type Service interface {
	ListOrganizations(ctx context.Context) ([]domain.Organization, error)
	CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error)
	ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error)
	CreateProject(ctx context.Context, input domain.CreateProject) (domain.Project, error)
	GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
	UpdateProject(ctx context.Context, input domain.UpdateProject) (domain.Project, error)
	ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

type service struct {
	repo repository.Repository
}

func New(repo repository.Repository) Service {
	return &service{repo: repo}
}

func (s *service) ListOrganizations(ctx context.Context) ([]domain.Organization, error) {
	return s.repo.ListOrganizations(ctx)
}

func (s *service) CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error) {
	return s.repo.CreateOrganization(ctx, input)
}

func (s *service) GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	return s.repo.GetOrganization(ctx, id)
}

func (s *service) UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	return s.repo.UpdateOrganization(ctx, input)
}

func (s *service) ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx, organizationID)
}

func (s *service) CreateProject(ctx context.Context, input domain.CreateProject) (domain.Project, error) {
	return s.repo.CreateProject(ctx, input)
}

func (s *service) GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	return s.repo.GetProject(ctx, id)
}

func (s *service) UpdateProject(ctx context.Context, input domain.UpdateProject) (domain.Project, error) {
	return s.repo.UpdateProject(ctx, input)
}

func (s *service) ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	return s.repo.ArchiveProject(ctx, id)
}
