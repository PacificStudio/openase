package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	repository "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/google/uuid"
)

var (
	ErrNotFound                  = repository.ErrNotFound
	ErrConflict                  = repository.ErrConflict
	ErrInvalidInput              = repository.ErrInvalidInput
	ErrMachineTestingUnavailable = errors.New("machine testing unavailable")
	ErrMachineProbeFailed        = errors.New("machine probe failed")
)

type MachineTester interface {
	TestConnection(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error)
}

type Service interface {
	ListOrganizations(ctx context.Context) ([]domain.Organization, error)
	CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error)
	ListMachines(ctx context.Context, organizationID uuid.UUID) ([]domain.Machine, error)
	CreateMachine(ctx context.Context, input domain.CreateMachine) (domain.Machine, error)
	GetMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error)
	UpdateMachine(ctx context.Context, input domain.UpdateMachine) (domain.Machine, error)
	DeleteMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error)
	TestMachineConnection(ctx context.Context, id uuid.UUID) (domain.Machine, domain.MachineProbe, error)
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
	repo          repository.Repository
	resolver      provider.ExecutableResolver
	machineTester MachineTester
}

func New(repo repository.Repository, resolver provider.ExecutableResolver, machineTester MachineTester) Service {
	return &service{repo: repo, resolver: resolver, machineTester: machineTester}
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

func (s *service) ListMachines(ctx context.Context, organizationID uuid.UUID) ([]domain.Machine, error) {
	return s.repo.ListMachines(ctx, organizationID)
}

func (s *service) CreateMachine(ctx context.Context, input domain.CreateMachine) (domain.Machine, error) {
	return s.repo.CreateMachine(ctx, input)
}

func (s *service) GetMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error) {
	return s.repo.GetMachine(ctx, id)
}

func (s *service) UpdateMachine(ctx context.Context, input domain.UpdateMachine) (domain.Machine, error) {
	return s.repo.UpdateMachine(ctx, input)
}

func (s *service) DeleteMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error) {
	return s.repo.DeleteMachine(ctx, id)
}

func (s *service) TestMachineConnection(ctx context.Context, id uuid.UUID) (domain.Machine, domain.MachineProbe, error) {
	if s.machineTester == nil {
		return domain.Machine{}, domain.MachineProbe{}, ErrMachineTestingUnavailable
	}

	machine, err := s.repo.GetMachine(ctx, id)
	if err != nil {
		return domain.Machine{}, domain.MachineProbe{}, err
	}

	probe, err := s.machineTester.TestConnection(ctx, machine)
	if err != nil {
		checkedAt := probe.CheckedAt
		if checkedAt.IsZero() {
			checkedAt = time.Now().UTC()
		}
		updateErr := s.repo.RecordMachineProbe(ctx, domain.RecordMachineProbe{
			ID:              id,
			Status:          domainMachineFailureStatus(machine),
			LastHeartbeatAt: checkedAt,
			Resources: map[string]any{
				"transport":    probe.Transport,
				"error":        err.Error(),
				"checked_at":   checkedAt.Format(time.RFC3339),
				"last_success": false,
			},
		})
		if updateErr != nil {
			return domain.Machine{}, domain.MachineProbe{}, fmt.Errorf("%w: %v (status update failed: %v)", ErrMachineProbeFailed, err, updateErr)
		}
		return domain.Machine{}, domain.MachineProbe{}, fmt.Errorf("%w: %v", ErrMachineProbeFailed, err)
	}

	if err := s.repo.RecordMachineProbe(ctx, domain.RecordMachineProbe{
		ID:              id,
		Status:          domainMachineSuccessStatus(machine),
		LastHeartbeatAt: probe.CheckedAt,
		Resources:       cloneResources(probe.Resources),
	}); err != nil {
		return domain.Machine{}, domain.MachineProbe{}, err
	}

	updated, err := s.repo.GetMachine(ctx, id)
	if err != nil {
		return domain.Machine{}, domain.MachineProbe{}, err
	}

	return updated, probe, nil
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

func domainMachineFailureStatus(machine domain.Machine) entmachine.Status {
	if machine.Host == domain.LocalMachineHost {
		return entmachine.StatusDegraded
	}
	return entmachine.StatusOffline
}

func domainMachineSuccessStatus(machine domain.Machine) entmachine.Status {
	if machine.Status == entmachine.StatusMaintenance {
		return entmachine.StatusOnline
	}
	return machine.Status
}

func cloneResources(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}

	return cloned
}
