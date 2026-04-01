package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type Repository interface {
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
	RecordMachineProbe(ctx context.Context, input domain.RecordMachineProbe) error
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
	GetWorkspaceDashboardSummary(ctx context.Context) (domain.WorkspaceDashboardSummary, error)
	GetOrganizationDashboardSummary(ctx context.Context, organizationID uuid.UUID) (domain.OrganizationDashboardSummary, error)
	GetOrganizationTokenUsage(ctx context.Context, input domain.GetOrganizationTokenUsage) (domain.OrganizationTokenUsageReport, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error)
	CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error)
	UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error)
	ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error)
	ListAgentRuns(ctx context.Context, projectID uuid.UUID) ([]domain.AgentRun, error)
	ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error)
	ListAgentOutput(ctx context.Context, input domain.ListAgentOutput) ([]domain.AgentOutputEntry, error)
	ListAgentSteps(ctx context.Context, input domain.ListAgentSteps) ([]domain.AgentStepEntry, error)
	ListAgentRunTraceEntries(ctx context.Context, input domain.ListAgentRunTraceEntries) ([]domain.AgentTraceEntry, error)
	ListAgentRunStepEntries(ctx context.Context, input domain.ListAgentRunStepEntries) ([]domain.AgentStepEntry, error)
	CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	GetAgentRun(ctx context.Context, id uuid.UUID) (domain.AgentRun, error)
	UpdateAgent(ctx context.Context, input domain.UpdateAgent) (domain.Agent, error)
	UpdateAgentRuntimeControlState(ctx context.Context, input domain.UpdateAgentRuntimeControlState) (domain.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
}
