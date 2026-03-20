package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	repository "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/google/uuid"
)

var (
	ErrNotFound     = repository.ErrNotFound
	ErrConflict     = repository.ErrConflict
	ErrInvalidInput = repository.ErrInvalidInput
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
	ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRepo, error)
	CreateProjectRepo(ctx context.Context, input domain.CreateProjectRepo) (domain.ProjectRepo, error)
	GetProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error)
	UpdateProjectRepo(ctx context.Context, input domain.UpdateProjectRepo) (domain.ProjectRepo, error)
	DeleteProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error)
	ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]domain.TicketRepoScope, error)
	CreateTicketRepoScope(ctx context.Context, input domain.CreateTicketRepoScope) (domain.TicketRepoScope, error)
	GetTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error)
	UpdateTicketRepoScope(ctx context.Context, input domain.UpdateTicketRepoScope) (domain.TicketRepoScope, error)
	DeleteTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error)
	CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error)
	UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error)
	ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error)
	ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error)
	CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
}

type service struct {
	repo     repository.Repository
	resolver provider.ExecutableResolver
}

func New(repo repository.Repository, resolver provider.ExecutableResolver) Service {
	return &service{repo: repo, resolver: resolver}
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

func (s *service) ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRepo, error) {
	return s.repo.ListProjectRepos(ctx, projectID)
}

func (s *service) CreateProjectRepo(ctx context.Context, input domain.CreateProjectRepo) (domain.ProjectRepo, error) {
	return s.repo.CreateProjectRepo(ctx, input)
}

func (s *service) GetProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	return s.repo.GetProjectRepo(ctx, projectID, id)
}

func (s *service) UpdateProjectRepo(ctx context.Context, input domain.UpdateProjectRepo) (domain.ProjectRepo, error) {
	return s.repo.UpdateProjectRepo(ctx, input)
}

func (s *service) DeleteProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	return s.repo.DeleteProjectRepo(ctx, projectID, id)
}

func (s *service) ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]domain.TicketRepoScope, error) {
	return s.repo.ListTicketRepoScopes(ctx, projectID, ticketID)
}

func (s *service) CreateTicketRepoScope(ctx context.Context, input domain.CreateTicketRepoScope) (domain.TicketRepoScope, error) {
	return s.repo.CreateTicketRepoScope(ctx, input)
}

func (s *service) GetTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error) {
	return s.repo.GetTicketRepoScope(ctx, projectID, ticketID, id)
}

func (s *service) UpdateTicketRepoScope(ctx context.Context, input domain.UpdateTicketRepoScope) (domain.TicketRepoScope, error) {
	return s.repo.UpdateTicketRepoScope(ctx, input)
}

func (s *service) DeleteTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error) {
	return s.repo.DeleteTicketRepoScope(ctx, projectID, ticketID, id)
}
