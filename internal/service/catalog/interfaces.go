package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type OrganizationService interface {
	ListOrganizations(ctx context.Context) ([]domain.Organization, error)
	CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error)
	ArchiveOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
}

type MachineService interface {
	ListMachines(ctx context.Context, organizationID uuid.UUID) ([]domain.Machine, error)
	CreateMachine(ctx context.Context, input domain.CreateMachine) (domain.Machine, error)
	GetMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error)
	UpdateMachine(ctx context.Context, input domain.UpdateMachine) (domain.Machine, error)
	DeleteMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error)
	TestMachineConnection(ctx context.Context, id uuid.UUID) (domain.Machine, domain.MachineProbe, error)
	RefreshMachineHealth(ctx context.Context, id uuid.UUID) (domain.Machine, error)
}

type ProjectService interface {
	ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error)
	CreateProject(ctx context.Context, input domain.CreateProject) (domain.Project, error)
	GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
	UpdateProject(ctx context.Context, input domain.UpdateProject) (domain.Project, error)
	ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

type ProjectRepoService interface {
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
}

type DashboardQueryService interface {
	GetWorkspaceDashboardSummary(ctx context.Context) (domain.WorkspaceDashboardSummary, error)
	GetOrganizationDashboardSummary(ctx context.Context, organizationID uuid.UUID) (domain.OrganizationDashboardSummary, error)
}

type UsageQueryService interface {
	GetOrganizationTokenUsage(ctx context.Context, input domain.GetOrganizationTokenUsage) (domain.OrganizationTokenUsageReport, error)
	GetProjectTokenUsage(ctx context.Context, input domain.GetProjectTokenUsage) (domain.ProjectTokenUsageReport, error)
}

type AgentProviderService interface {
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error)
	CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error)
	UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error)
}

type AgentService interface {
	ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error)
	CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	UpdateAgent(ctx context.Context, input domain.UpdateAgent) (domain.Agent, error)
	RequestAgentPause(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	RequestAgentResume(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	RetireAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
}

type AgentRunQueryService interface {
	ListAgentRuns(ctx context.Context, projectID uuid.UUID) ([]domain.AgentRun, error)
	GetAgentRun(ctx context.Context, id uuid.UUID) (domain.AgentRun, error)
	ListAgentOutput(ctx context.Context, input domain.ListAgentOutput) ([]domain.AgentOutputEntry, error)
	ListAgentSteps(ctx context.Context, input domain.ListAgentSteps) ([]domain.AgentStepEntry, error)
	ListAgentRunTraceEntries(ctx context.Context, input domain.ListAgentRunTraceEntries) ([]domain.AgentTraceEntry, error)
	ListAgentRunStepEntries(ctx context.Context, input domain.ListAgentRunStepEntries) ([]domain.AgentStepEntry, error)
}

type ActivityQueryService interface {
	ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error)
}

// Service is a transitional compatibility facade over the narrower catalog services.
// New production consumers should depend on the smallest subdomain interface they need.
type Service interface {
	OrganizationService
	MachineService
	ProjectService
	ProjectRepoService
	DashboardQueryService
	UsageQueryService
	AgentProviderService
	AgentService
	AgentRunQueryService
	ActivityQueryService
}

// Services exposes the narrow catalog capabilities as an embeddable aggregate for composition roots.
type Services struct {
	OrganizationService
	MachineService
	ProjectService
	ProjectRepoService
	DashboardQueryService
	UsageQueryService
	AgentProviderService
	AgentService
	AgentRunQueryService
	ActivityQueryService
}

func SplitServices(s Service) Services {
	if s == nil {
		return Services{}
	}
	return Services{
		OrganizationService:   s,
		MachineService:        s,
		ProjectService:        s,
		ProjectRepoService:    s,
		DashboardQueryService: s,
		UsageQueryService:     s,
		AgentProviderService:  s,
		AgentService:          s,
		AgentRunQueryService:  s,
		ActivityQueryService:  s,
	}
}

func (s Services) Empty() bool {
	return s.OrganizationService == nil &&
		s.MachineService == nil &&
		s.ProjectService == nil &&
		s.ProjectRepoService == nil &&
		s.DashboardQueryService == nil &&
		s.UsageQueryService == nil &&
		s.AgentProviderService == nil &&
		s.AgentService == nil &&
		s.AgentRunQueryService == nil &&
		s.ActivityQueryService == nil
}

type OrganizationRepository interface {
	ListOrganizations(ctx context.Context) ([]domain.Organization, error)
	CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error)
	ArchiveOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
}

type MachineRepository interface {
	ListMachines(ctx context.Context, organizationID uuid.UUID) ([]domain.Machine, error)
	CreateMachine(ctx context.Context, input domain.CreateMachine) (domain.Machine, error)
	GetMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error)
	UpdateMachine(ctx context.Context, input domain.UpdateMachine) (domain.Machine, error)
	DeleteMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error)
	RecordMachineProbe(ctx context.Context, input domain.RecordMachineProbe) error
}

type ProjectRepository interface {
	ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error)
	CreateProject(ctx context.Context, input domain.CreateProject) (domain.Project, error)
	GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
	UpdateProject(ctx context.Context, input domain.UpdateProject) (domain.Project, error)
	ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

type ProjectRepoRepository interface {
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
}

type DashboardQueryRepository interface {
	GetWorkspaceDashboardSummary(ctx context.Context) (domain.WorkspaceDashboardSummary, error)
	GetOrganizationDashboardSummary(ctx context.Context, organizationID uuid.UUID) (domain.OrganizationDashboardSummary, error)
}

type UsageQueryRepository interface {
	GetOrganizationTokenUsage(ctx context.Context, input domain.GetOrganizationTokenUsage) (domain.OrganizationTokenUsageReport, error)
	GetProjectTokenUsage(ctx context.Context, input domain.GetProjectTokenUsage) (domain.ProjectTokenUsageReport, error)
}

type AgentProviderRepository interface {
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error)
	CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error)
	UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error)
}

type AgentRepository interface {
	ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error)
	CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	UpdateAgent(ctx context.Context, input domain.UpdateAgent) (domain.Agent, error)
	UpdateAgentRuntimeControlState(ctx context.Context, input domain.UpdateAgentRuntimeControlState) (domain.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
}

type AgentRunQueryRepository interface {
	ListAgentRuns(ctx context.Context, projectID uuid.UUID) ([]domain.AgentRun, error)
	GetAgentRun(ctx context.Context, id uuid.UUID) (domain.AgentRun, error)
	ListAgentOutput(ctx context.Context, input domain.ListAgentOutput) ([]domain.AgentOutputEntry, error)
	ListAgentSteps(ctx context.Context, input domain.ListAgentSteps) ([]domain.AgentStepEntry, error)
	ListAgentRunTraceEntries(ctx context.Context, input domain.ListAgentRunTraceEntries) ([]domain.AgentTraceEntry, error)
	ListAgentRunStepEntries(ctx context.Context, input domain.ListAgentRunStepEntries) ([]domain.AgentStepEntry, error)
}

type ActivityQueryRepository interface {
	ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error)
}

// Repository is a transitional compatibility facade over the narrower catalog repositories.
type Repository interface {
	OrganizationRepository
	MachineRepository
	ProjectRepository
	ProjectRepoRepository
	DashboardQueryRepository
	UsageQueryRepository
	AgentProviderRepository
	AgentRepository
	AgentRunQueryRepository
	ActivityQueryRepository
}
