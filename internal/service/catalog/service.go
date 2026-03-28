package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var (
	ErrNotFound                  = domain.ErrNotFound
	ErrConflict                  = domain.ErrConflict
	ErrInvalidInput              = domain.ErrInvalidInput
	ErrMachineTestingUnavailable = errors.New("machine testing unavailable")
	ErrMachineProbeFailed        = errors.New("machine probe failed")
)

type MachineTester interface {
	TestConnection(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error)
}

type Option func(*service)

type ProjectStatusBootstrapper interface {
	BootstrapProjectStatuses(ctx context.Context, projectID uuid.UUID) error
}

type ProjectStatusBootstrapperFunc func(ctx context.Context, projectID uuid.UUID) error

func (fn ProjectStatusBootstrapperFunc) BootstrapProjectStatuses(ctx context.Context, projectID uuid.UUID) error {
	return fn(ctx, projectID)
}

func WithProjectStatusBootstrapper(bootstrapper ProjectStatusBootstrapper) Option {
	return func(s *service) {
		if bootstrapper == nil {
			return
		}
		s.projectStatusBootstrapper = bootstrapper
	}
}

type Service interface {
	ListOrganizations(ctx context.Context) ([]domain.Organization, error)
	CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error)
	ArchiveOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
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
	ListAgentRuns(ctx context.Context, projectID uuid.UUID) ([]domain.AgentRun, error)
	ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error)
	ListAgentOutput(ctx context.Context, input domain.ListAgentOutput) ([]domain.AgentOutputEntry, error)
	ListAgentSteps(ctx context.Context, input domain.ListAgentSteps) ([]domain.AgentStepEntry, error)
	CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	GetAgentRun(ctx context.Context, id uuid.UUID) (domain.AgentRun, error)
	RequestAgentPause(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	RequestAgentResume(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
}

type service struct {
	repo                      Repository
	resolver                  provider.ExecutableResolver
	machineTester             MachineTester
	projectStatusBootstrapper ProjectStatusBootstrapper
}

func New(repo Repository, resolver provider.ExecutableResolver, machineTester MachineTester, opts ...Option) Service {
	svc := &service{repo: repo, resolver: resolver, machineTester: machineTester}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc
}

func (s *service) ListOrganizations(ctx context.Context) ([]domain.Organization, error) {
	return s.repo.ListOrganizations(ctx)
}

func (s *service) CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error) {
	item, err := s.repo.CreateOrganization(ctx, input)
	if err != nil {
		return domain.Organization{}, err
	}

	if item.DefaultAgentProviderID != nil {
		return item, nil
	}

	providers, err := s.ListAgentProviders(ctx, item.ID)
	if err != nil {
		return domain.Organization{}, err
	}

	defaultProviderID := preferredAvailableProviderID(providers)
	if defaultProviderID == nil {
		return item, nil
	}

	return s.repo.UpdateOrganization(ctx, domain.UpdateOrganization{
		ID:                     item.ID,
		Name:                   item.Name,
		Slug:                   item.Slug,
		DefaultAgentProviderID: defaultProviderID,
	})
}

func (s *service) GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	return s.repo.GetOrganization(ctx, id)
}

func (s *service) UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	return s.repo.UpdateOrganization(ctx, input)
}

func (s *service) ArchiveOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	return s.repo.ArchiveOrganization(ctx, id)
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
			Resources:       mergeMachineProbeResources(machine.Resources, probe, checkedAt, err),
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
		Resources:       mergeMachineProbeResources(machine.Resources, probe, probe.CheckedAt, nil),
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
	item, err := s.repo.CreateProject(ctx, input)
	if err != nil {
		return domain.Project{}, err
	}
	if s.projectStatusBootstrapper == nil {
		return item, nil
	}
	if err := s.projectStatusBootstrapper.BootstrapProjectStatuses(ctx, item.ID); err != nil {
		return domain.Project{}, fmt.Errorf("seed default project statuses: %w", err)
	}
	return item, nil
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

func domainMachineFailureStatus(machine domain.Machine) domain.MachineStatus {
	if machine.Host == domain.LocalMachineHost {
		return domain.MachineStatusDegraded
	}
	return domain.MachineStatusOffline
}

func domainMachineSuccessStatus(machine domain.Machine) domain.MachineStatus {
	if machine.Status == domain.MachineStatusMaintenance {
		return domain.MachineStatusOnline
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

func mergeMachineProbeResources(
	base map[string]any,
	probe domain.MachineProbe,
	checkedAt time.Time,
	probeErr error,
) map[string]any {
	merged := cloneResources(base)
	if probe.Transport != "" {
		merged["transport"] = probe.Transport
	}

	connectionTest := map[string]any{
		"checked_at":   checkedAt.UTC().Format(time.RFC3339),
		"transport":    probe.Transport,
		"last_success": probeErr == nil,
	}
	if probeErr != nil {
		connectionTest["error"] = probeErr.Error()
	}
	if probe.Output != "" {
		connectionTest["output"] = probe.Output
	}
	if len(probe.Resources) > 0 {
		connectionTest["resources"] = cloneResources(probe.Resources)
	}
	merged["connection_test"] = connectionTest

	return merged
}
