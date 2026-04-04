package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestServiceDelegatesRepositoryWrappers(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{
		provider: domain.AgentProvider{ID: uuid.New()},
		agent:    domain.Agent{ID: uuid.New()},
	}
	svc := New(repo, nil, nil)
	ctx := context.Background()
	orgID := uuid.New()
	projectID := uuid.New()
	repoID := uuid.New()
	ticketID := uuid.New()
	agentID := uuid.New()

	if _, err := svc.ListOrganizations(ctx); err != nil {
		t.Fatalf("ListOrganizations error = %v", err)
	}
	if _, err := svc.GetOrganization(ctx, orgID); err != nil {
		t.Fatalf("GetOrganization error = %v", err)
	}
	if _, err := svc.UpdateOrganization(ctx, domain.UpdateOrganization{ID: orgID}); err != nil {
		t.Fatalf("UpdateOrganization error = %v", err)
	}
	if _, err := svc.ListMachines(ctx, orgID); err != nil {
		t.Fatalf("ListMachines error = %v", err)
	}
	if _, err := svc.CreateMachine(ctx, domain.CreateMachine{OrganizationID: orgID}); err != nil {
		t.Fatalf("CreateMachine error = %v", err)
	}
	if _, err := svc.GetMachine(ctx, uuid.New()); err != nil {
		t.Fatalf("GetMachine error = %v", err)
	}
	if _, err := svc.UpdateMachine(ctx, domain.UpdateMachine{ID: uuid.New(), OrganizationID: orgID}); err != nil {
		t.Fatalf("UpdateMachine error = %v", err)
	}
	if _, err := svc.DeleteMachine(ctx, uuid.New()); err != nil {
		t.Fatalf("DeleteMachine error = %v", err)
	}
	if _, err := svc.ListProjects(ctx, orgID); err != nil {
		t.Fatalf("ListProjects error = %v", err)
	}
	if _, err := svc.GetProject(ctx, projectID); err != nil {
		t.Fatalf("GetProject error = %v", err)
	}
	if _, err := svc.UpdateProject(ctx, domain.UpdateProject{ID: projectID, OrganizationID: orgID}); err != nil {
		t.Fatalf("UpdateProject error = %v", err)
	}
	if _, err := svc.ArchiveProject(ctx, projectID); err != nil {
		t.Fatalf("ArchiveProject error = %v", err)
	}
	if _, err := svc.ListProjectRepos(ctx, projectID); err != nil {
		t.Fatalf("ListProjectRepos error = %v", err)
	}
	if _, err := svc.CreateProjectRepo(ctx, domain.CreateProjectRepo{ProjectID: projectID}); err != nil {
		t.Fatalf("CreateProjectRepo error = %v", err)
	}
	if _, err := svc.GetProjectRepo(ctx, projectID, repoID); err != nil {
		t.Fatalf("GetProjectRepo error = %v", err)
	}
	if _, err := svc.UpdateProjectRepo(ctx, domain.UpdateProjectRepo{ID: repoID, ProjectID: projectID}); err != nil {
		t.Fatalf("UpdateProjectRepo error = %v", err)
	}
	if _, err := svc.DeleteProjectRepo(ctx, projectID, repoID); err != nil {
		t.Fatalf("DeleteProjectRepo error = %v", err)
	}
	if _, err := svc.ListTicketRepoScopes(ctx, projectID, ticketID); err != nil {
		t.Fatalf("ListTicketRepoScopes error = %v", err)
	}
	if _, err := svc.CreateTicketRepoScope(ctx, domain.CreateTicketRepoScope{ProjectID: projectID, TicketID: ticketID}); err != nil {
		t.Fatalf("CreateTicketRepoScope error = %v", err)
	}
	if _, err := svc.GetTicketRepoScope(ctx, projectID, ticketID, uuid.New()); err != nil {
		t.Fatalf("GetTicketRepoScope error = %v", err)
	}
	if _, err := svc.UpdateTicketRepoScope(ctx, domain.UpdateTicketRepoScope{ID: uuid.New(), ProjectID: projectID, TicketID: ticketID}); err != nil {
		t.Fatalf("UpdateTicketRepoScope error = %v", err)
	}
	if _, err := svc.DeleteTicketRepoScope(ctx, projectID, ticketID, uuid.New()); err != nil {
		t.Fatalf("DeleteTicketRepoScope error = %v", err)
	}
	if _, err := svc.ListActivityEvents(ctx, domain.ListActivityEvents{ProjectID: projectID}); err != nil {
		t.Fatalf("ListActivityEvents error = %v", err)
	}
	if _, err := svc.ListAgentOutput(ctx, domain.ListAgentOutput{ProjectID: projectID, AgentID: agentID}); err != nil {
		t.Fatalf("ListAgentOutput error = %v", err)
	}
	if _, err := svc.GetAgentProvider(ctx, repo.provider.ID); err != nil {
		t.Fatalf("GetAgentProvider error = %v", err)
	}
	if _, err := svc.ListAgents(ctx, projectID); err != nil {
		t.Fatalf("ListAgents error = %v", err)
	}
	if _, err := svc.ListAgentRuns(ctx, projectID); err != nil {
		t.Fatalf("ListAgentRuns error = %v", err)
	}
	if _, err := svc.CreateAgent(ctx, domain.CreateAgent{ProjectID: projectID}); err != nil {
		t.Fatalf("CreateAgent error = %v", err)
	}
	if _, err := svc.GetAgent(ctx, repo.agent.ID); err != nil {
		t.Fatalf("GetAgent error = %v", err)
	}
	if _, err := svc.GetAgentRun(ctx, uuid.New()); err != nil {
		t.Fatalf("GetAgentRun error = %v", err)
	}
	if _, err := svc.DeleteAgent(ctx, uuid.New()); err != nil {
		t.Fatalf("DeleteAgent error = %v", err)
	}
}

func TestServiceMachineProbePathsAndHelpers(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	svc := New(repo, nil, nil)
	if _, _, err := svc.TestMachineConnection(context.Background(), uuid.New()); !errors.Is(err, ErrMachineTestingUnavailable) {
		t.Fatalf("TestMachineConnection unavailable error = %v", err)
	}

	ctx := context.Background()
	machineID := uuid.New()
	successSvc := New(repo, nil, stubMachineTester{
		probe: domain.MachineProbe{
			CheckedAt: time.Date(2026, 3, 27, 15, 0, 0, 0, time.UTC),
			Transport: "ssh",
			Resources: map[string]any{"cpu": "8"},
		},
	})
	if _, probe, err := successSvc.TestMachineConnection(ctx, machineID); err != nil || probe.Transport != "ssh" {
		t.Fatalf("TestMachineConnection success = %+v, %v", probe, err)
	}

	failureSvc := New(repo, nil, stubMachineTester{
		probe: domain.MachineProbe{
			CheckedAt: time.Time{},
			Transport: "ssh",
		},
		err: errors.New("dial failed"),
	})
	if _, _, err := failureSvc.TestMachineConnection(ctx, machineID); err == nil || !errors.Is(err, ErrMachineProbeFailed) {
		t.Fatalf("TestMachineConnection failure error = %v", err)
	}

	if got := domainMachineFailureStatus(domain.Machine{Host: domain.LocalMachineHost}); got != domain.MachineStatusDegraded {
		t.Fatalf("domainMachineFailureStatus(local) = %q", got)
	}
	if got := domainMachineFailureStatus(domain.Machine{Host: "remote"}); got != domain.MachineStatusOffline {
		t.Fatalf("domainMachineFailureStatus(remote) = %q", got)
	}
	if got := domainMachineSuccessStatus(domain.Machine{Status: domain.MachineStatusMaintenance}); got != domain.MachineStatusOnline {
		t.Fatalf("domainMachineSuccessStatus(maintenance) = %q", got)
	}
	if got := domainMachineSuccessStatus(domain.Machine{Status: domain.MachineStatusDegraded}); got != domain.MachineStatusDegraded {
		t.Fatalf("domainMachineSuccessStatus(passthrough) = %q", got)
	}
	if got := cloneResources(map[string]any{"cpu": "8"}); got["cpu"] != "8" {
		t.Fatalf("cloneResources() = %+v", got)
	}
	if got := cloneResources(nil); len(got) != 0 {
		t.Fatalf("cloneResources(nil) = %+v", got)
	}

	projectID := uuid.New()
	bootstrapper := &stubProjectStatusBootstrapper{}
	svcWithBootstrapper := New(repo, nil, nil, WithProjectStatusBootstrapper(bootstrapper))
	if _, err := svcWithBootstrapper.CreateProject(ctx, domain.CreateProject{OrganizationID: uuid.New()}); err != nil {
		t.Fatalf("CreateProject with bootstrapper error = %v", err)
	}
	if bootstrapper.projectID != repo.createdProject.ID {
		t.Fatalf("bootstrapper projectID = %s, want %s", bootstrapper.projectID, repo.createdProject.ID)
	}
	if err := ProjectStatusBootstrapperFunc(func(context.Context, uuid.UUID) error { return nil }).BootstrapProjectStatuses(ctx, projectID); err != nil {
		t.Fatalf("ProjectStatusBootstrapperFunc error = %v", err)
	}
}

func TestServiceCreateOrganizationAndProjectFallbackPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgID := uuid.New()
	repo := &stubRepository{
		createdOrganization: domain.Organization{
			ID:   orgID,
			Name: "Acme",
			Slug: "acme",
		},
		listedProviders: []domain.AgentProvider{
			{
				ID:          uuid.New(),
				Name:        "Manual Provider",
				AdapterType: domain.AgentProviderAdapterTypeCustom,
				CliCommand:  "",
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil, WithProjectStatusBootstrapper(nil))

	item, err := svc.CreateOrganization(ctx, domain.CreateOrganization{Name: "Acme", Slug: "acme"})
	if err != nil {
		t.Fatalf("CreateOrganization(no available provider) error = %v", err)
	}
	if item.DefaultAgentProviderID != nil {
		t.Fatalf("CreateOrganization(no available provider) = %+v", item)
	}
	if repo.updatedOrganization != nil {
		t.Fatalf("CreateOrganization(no available provider) should not update organization: %+v", repo.updatedOrganization)
	}

	repo.createdOrganization.DefaultAgentProviderID = ptrUUID(uuid.New())
	item, err = svc.CreateOrganization(ctx, domain.CreateOrganization{Name: "Acme", Slug: "acme"})
	if err != nil {
		t.Fatalf("CreateOrganization(existing default) error = %v", err)
	}
	if item.DefaultAgentProviderID == nil {
		t.Fatalf("CreateOrganization(existing default) = %+v", item)
	}

	projectRepo := &stubRepository{
		createdProject: domain.Project{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Name:           "OpenASE",
			Slug:           "openase",
			Status:         "In Progress",
		},
	}
	bootstrapErr := errors.New("seed failed")
	svc = New(projectRepo, nil, nil, WithProjectStatusBootstrapper(ProjectStatusBootstrapperFunc(func(context.Context, uuid.UUID) error {
		return bootstrapErr
	})))
	if _, err := svc.CreateProject(ctx, domain.CreateProject{
		OrganizationID: projectRepo.createdProject.OrganizationID,
		Name:           "OpenASE",
		Slug:           "openase",
		Status:         "In Progress",
	}); err == nil || !errors.Is(err, bootstrapErr) {
		t.Fatalf("CreateProject(bootstrap failure) error = %v", err)
	}
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}
