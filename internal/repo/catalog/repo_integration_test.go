package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func TestEntRepositoryOrganizationProjectRepoAndScopeLifecycle(t *testing.T) {
	client := openRepoCatalogTestEntClient(t)
	ctx := context.Background()
	repo := NewEntRepository(client)

	org, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "Better And Better",
		Slug: "better-and-better",
	})
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	if org.Status != domain.OrganizationStatusActive {
		t.Fatalf("CreateOrganization() = %+v", org)
	}

	orgs, err := repo.ListOrganizations(ctx)
	if err != nil {
		t.Fatalf("ListOrganizations() error = %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != org.ID {
		t.Fatalf("ListOrganizations() = %+v", orgs)
	}

	machines, err := repo.ListMachines(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListMachines() error = %v", err)
	}
	if len(machines) != 1 || machines[0].Name != domain.LocalMachineName || machines[0].Host != domain.LocalMachineHost {
		t.Fatalf("ListMachines() = %+v", machines)
	}

	providers, err := repo.ListAgentProviders(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListAgentProviders() error = %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("CreateOrganization() should seed builtin providers")
	}
	defaultProviderID := providers[0].ID

	updatedOrg, err := repo.UpdateOrganization(ctx, domain.UpdateOrganization{
		ID:                     org.ID,
		Name:                   "Better & Better",
		Slug:                   "better-better",
		DefaultAgentProviderID: &defaultProviderID,
	})
	if err != nil {
		t.Fatalf("UpdateOrganization() error = %v", err)
	}
	if updatedOrg.Name != "Better & Better" || updatedOrg.DefaultAgentProviderID == nil || *updatedOrg.DefaultAgentProviderID != defaultProviderID {
		t.Fatalf("UpdateOrganization() = %+v", updatedOrg)
	}

	gotOrg, err := repo.GetOrganization(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetOrganization() error = %v", err)
	}
	if gotOrg.Slug != "better-better" {
		t.Fatalf("GetOrganization() = %+v", gotOrg)
	}

	project, err := repo.CreateProject(ctx, domain.CreateProject{
		OrganizationID:         org.ID,
		Name:                   "OpenASE",
		Slug:                   "openase",
		Description:            "Issue-driven automation",
		Status:                 domain.ProjectStatusInProgress,
		DefaultAgentProviderID: &defaultProviderID,
		AccessibleMachineIDs:   []uuid.UUID{machines[0].ID},
		MaxConcurrentAgents:    4,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if project.Status != domain.ProjectStatusInProgress || project.DefaultAgentProviderID == nil || *project.DefaultAgentProviderID != defaultProviderID {
		t.Fatalf("CreateProject() = %+v", project)
	}

	projects, err := repo.ListProjects(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("ListProjects() = %+v", projects)
	}

	gotProject, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if gotProject.Name != "OpenASE" {
		t.Fatalf("GetProject() = %+v", gotProject)
	}

	updatedProject, err := repo.UpdateProject(ctx, domain.UpdateProject{
		ID:                     project.ID,
		OrganizationID:         org.ID,
		Name:                   "OpenASE Core",
		Slug:                   "openase-core",
		Description:            "Backend coverage rollout",
		Status:                 domain.ProjectStatusCanceled,
		DefaultAgentProviderID: nil,
		AccessibleMachineIDs:   []uuid.UUID{machines[0].ID},
		MaxConcurrentAgents:    6,
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updatedProject.Name != "OpenASE Core" || updatedProject.Status != domain.ProjectStatusCanceled || updatedProject.DefaultAgentProviderID != nil {
		t.Fatalf("UpdateProject() = %+v", updatedProject)
	}

	repoOne, err := repo.CreateProjectRepo(ctx, domain.CreateProjectRepo{
		ProjectID:        project.ID,
		Name:             "openase-main",
		RepositoryURL:    "https://github.com/PacificStudio/openase.git",
		DefaultBranch:    "main",
		WorkspaceDirname: "openase-main",
		Labels:           []string{"backend", "automation"},
	})
	if err != nil {
		t.Fatalf("CreateProjectRepo() repoOne error = %v", err)
	}
	if repoOne.WorkspaceDirname != "openase-main" {
		t.Fatalf("CreateProjectRepo() repoOne = %+v", repoOne)
	}

	repoTwo, err := repo.CreateProjectRepo(ctx, domain.CreateProjectRepo{
		ProjectID:     project.ID,
		Name:          "worker-tools",
		RepositoryURL: "https://github.com/GrandCX/worker-tools.git",
		DefaultBranch: "develop",
	})
	if err != nil {
		t.Fatalf("CreateProjectRepo() repoTwo error = %v", err)
	}
	projectRepos, err := repo.ListProjectRepos(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectRepos() error = %v", err)
	}
	if len(projectRepos) != 2 || projectRepos[0].ID != repoOne.ID {
		t.Fatalf("ListProjectRepos() = %+v", projectRepos)
	}

	gotRepoOne, err := repo.GetProjectRepo(ctx, project.ID, repoOne.ID)
	if err != nil {
		t.Fatalf("GetProjectRepo() error = %v", err)
	}
	if gotRepoOne.Name != "openase-main" {
		t.Fatalf("GetProjectRepo() = %+v", gotRepoOne)
	}

	repoTwoWorkspaceDirname := "worker-tools/release"
	updatedRepoTwo, err := repo.UpdateProjectRepo(ctx, domain.UpdateProjectRepo{
		ID:               repoTwo.ID,
		ProjectID:        project.ID,
		Name:             "worker-tools",
		RepositoryURL:    repoTwo.RepositoryURL,
		DefaultBranch:    "release",
		WorkspaceDirname: repoTwoWorkspaceDirname,
		Labels:           []string{"worker", "ops"},
	})
	if err != nil {
		t.Fatalf("UpdateProjectRepo() repoTwo error = %v", err)
	}
	if updatedRepoTwo.WorkspaceDirname != repoTwoWorkspaceDirname || len(updatedRepoTwo.Labels) != 2 {
		t.Fatalf("UpdateProjectRepo() repoTwo = %+v", updatedRepoTwo)
	}

	updatedRepoOne, err := repo.UpdateProjectRepo(ctx, domain.UpdateProjectRepo{
		ID:               repoOne.ID,
		ProjectID:        project.ID,
		Name:             "openase-main",
		RepositoryURL:    repoOne.RepositoryURL,
		DefaultBranch:    "main",
		WorkspaceDirname: "openase-main",
		Labels:           nil,
	})
	if err != nil {
		t.Fatalf("UpdateProjectRepo() error = %v", err)
	}
	if updatedRepoOne.WorkspaceDirname != "openase-main" || len(updatedRepoOne.Labels) != 0 {
		t.Fatalf("UpdateProjectRepo() = %+v", updatedRepoOne)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("ResetToDefaultTemplate() error = %v", err)
	}
	todoID := findRepoCatalogStatusIDByName(t, statuses, "Todo")
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-278").
		SetTitle("Finish backend coverage rollout").
		SetStatusID(todoID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	scopeOne, err := repo.CreateTicketRepoScope(ctx, domain.CreateTicketRepoScope{
		ProjectID: project.ID,
		TicketID:  ticketItem.ID,
		RepoID:    repoOne.ID,
	})
	if err != nil {
		t.Fatalf("CreateTicketRepoScope() scopeOne error = %v", err)
	}
	if scopeOne.BranchName != "" {
		t.Fatalf("CreateTicketRepoScope() scopeOne = %+v", scopeOne)
	}

	scopeTwo, err := repo.CreateTicketRepoScope(ctx, domain.CreateTicketRepoScope{
		ProjectID:      project.ID,
		TicketID:       ticketItem.ID,
		RepoID:         repoTwo.ID,
		BranchName:     strPtr("fix/openase-278-coverage"),
		PullRequestURL: strPtr("https://github.com/PacificStudio/openase/pull/278"),
	})
	if err != nil {
		t.Fatalf("CreateTicketRepoScope() scopeTwo error = %v", err)
	}
	if scopeTwo.BranchName != "fix/openase-278-coverage" {
		t.Fatalf("CreateTicketRepoScope() scopeTwo = %+v", scopeTwo)
	}

	scopes, err := repo.ListTicketRepoScopes(ctx, project.ID, ticketItem.ID)
	if err != nil {
		t.Fatalf("ListTicketRepoScopes() error = %v", err)
	}
	if len(scopes) != 2 || scopes[0].ID != scopeOne.ID {
		t.Fatalf("ListTicketRepoScopes() = %+v", scopes)
	}

	gotScopeTwo, err := repo.GetTicketRepoScope(ctx, project.ID, ticketItem.ID, scopeTwo.ID)
	if err != nil {
		t.Fatalf("GetTicketRepoScope() error = %v", err)
	}
	if gotScopeTwo.PullRequestURL == nil || *gotScopeTwo.PullRequestURL == "" {
		t.Fatalf("GetTicketRepoScope() = %+v", gotScopeTwo)
	}

	updatedScopeOne, err := repo.UpdateTicketRepoScope(ctx, domain.UpdateTicketRepoScope{
		ID:             scopeOne.ID,
		ProjectID:      project.ID,
		TicketID:       ticketItem.ID,
		RepoID:         repoOne.ID,
		BranchName:     strPtr("fix/openase-278-core"),
		BranchNameSet:  true,
		PullRequestURL: strPtr("https://github.com/PacificStudio/openase/pull/279"),
		PullRequestSet: true,
	})
	if err != nil {
		t.Fatalf("UpdateTicketRepoScope() scopeOne error = %v", err)
	}
	if updatedScopeOne.PullRequestURL == nil || *updatedScopeOne.PullRequestURL == "" || updatedScopeOne.BranchName != "fix/openase-278-core" {
		t.Fatalf("UpdateTicketRepoScope() scopeOne = %+v", updatedScopeOne)
	}

	updatedScopeTwo, err := repo.UpdateTicketRepoScope(ctx, domain.UpdateTicketRepoScope{
		ID:             scopeTwo.ID,
		ProjectID:      project.ID,
		TicketID:       ticketItem.ID,
		RepoID:         repoTwo.ID,
		BranchName:     nil,
		BranchNameSet:  true,
		PullRequestURL: nil,
		PullRequestSet: true,
	})
	if err != nil {
		t.Fatalf("UpdateTicketRepoScope() error = %v", err)
	}
	if updatedScopeTwo.PullRequestURL != nil || updatedScopeTwo.BranchName != "" {
		t.Fatalf("UpdateTicketRepoScope() = %+v", updatedScopeTwo)
	}

	deletedScopeOne, err := repo.DeleteTicketRepoScope(ctx, project.ID, ticketItem.ID, scopeOne.ID)
	if err != nil {
		t.Fatalf("DeleteTicketRepoScope() error = %v", err)
	}
	if deletedScopeOne.ID != scopeOne.ID {
		t.Fatalf("DeleteTicketRepoScope() = %+v", deletedScopeOne)
	}
	deletedScopeTwo, err := repo.DeleteTicketRepoScope(ctx, project.ID, ticketItem.ID, scopeTwo.ID)
	if err != nil {
		t.Fatalf("DeleteTicketRepoScope() scopeTwo error = %v", err)
	}
	if deletedScopeTwo.ID != scopeTwo.ID {
		t.Fatalf("DeleteTicketRepoScope() scopeTwo = %+v", deletedScopeTwo)
	}

	deletedRepoTwo, err := repo.DeleteProjectRepo(ctx, project.ID, repoTwo.ID)
	if err != nil {
		t.Fatalf("DeleteProjectRepo() error = %v", err)
	}
	if deletedRepoTwo.ID != repoTwo.ID {
		t.Fatalf("DeleteProjectRepo() = %+v", deletedRepoTwo)
	}
	archivedProject, err := repo.ArchiveProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("ArchiveProject() error = %v", err)
	}
	if archivedProject.Status != domain.ProjectStatusArchived {
		t.Fatalf("ArchiveProject() = %+v", archivedProject)
	}

	archivedOrg, err := repo.ArchiveOrganization(ctx, org.ID)
	if err != nil {
		t.Fatalf("ArchiveOrganization() error = %v", err)
	}
	if archivedOrg.Status != domain.OrganizationStatusArchived {
		t.Fatalf("ArchiveOrganization() = %+v", archivedOrg)
	}
}

func TestEntRepositoryEnsurePrimaryFallbackPromotesExcludedOnlyRecord(t *testing.T) {
	client := openRepoCatalogTestEntClient(t)
	ctx := context.Background()
	repo := NewEntRepository(client)

	org, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "Better And Better",
		Slug: "better-and-better",
	})
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	project, err := repo.CreateProject(ctx, domain.CreateProject{
		OrganizationID:       org.ID,
		Name:                 "OpenASE",
		Slug:                 "openase",
		Description:          "Issue-driven automation",
		Status:               domain.ProjectStatusInProgress,
		MaxConcurrentAgents:  2,
		AccessibleMachineIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	projectRepo, err := repo.CreateProjectRepo(ctx, domain.CreateProjectRepo{
		ProjectID:     project.ID,
		Name:          "openase-main",
		RepositoryURL: "https://github.com/PacificStudio/openase.git",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreateProjectRepo() error = %v", err)
	}
	projectRepo, err = repo.GetProjectRepo(ctx, project.ID, projectRepo.ID)
	if err != nil {
		t.Fatalf("GetProjectRepo() error = %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("ResetToDefaultTemplate() error = %v", err)
	}
	todoID := findRepoCatalogStatusIDByName(t, statuses, "Todo")
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-279").
		SetTitle("Exercise repo scope persistence").
		SetStatusID(todoID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	scope, err := repo.CreateTicketRepoScope(ctx, domain.CreateTicketRepoScope{
		ProjectID: project.ID,
		TicketID:  ticketItem.ID,
		RepoID:    projectRepo.ID,
	})
	if err != nil {
		t.Fatalf("CreateTicketRepoScope() error = %v", err)
	}

	scope, err = repo.GetTicketRepoScope(ctx, project.ID, ticketItem.ID, scope.ID)
	if err != nil {
		t.Fatalf("GetTicketRepoScope() error = %v", err)
	}
	if scope.RepoID != projectRepo.ID || scope.BranchName != "" {
		t.Fatalf("expected repo scope to persist, got %+v", scope)
	}
}

func TestEntRepositoryMachineProviderValidationAndOrganizationFiltering(t *testing.T) {
	client := openRepoCatalogTestEntClient(t)
	ctx := context.Background()
	repo := NewEntRepository(client)

	orgA, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "Better And Better",
		Slug: "better-and-better",
	})
	if err != nil {
		t.Fatalf("CreateOrganization() orgA error = %v", err)
	}
	orgB, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "GrandCX",
		Slug: "grandcx",
	})
	if err != nil {
		t.Fatalf("CreateOrganization() orgB error = %v", err)
	}

	machinesA, err := repo.ListMachines(ctx, orgA.ID)
	if err != nil {
		t.Fatalf("ListMachines() orgA error = %v", err)
	}
	machinesB, err := repo.ListMachines(ctx, orgB.ID)
	if err != nil {
		t.Fatalf("ListMachines() orgB error = %v", err)
	}
	localMachine := machinesA[0]

	remoteMachine, err := repo.CreateMachine(ctx, domain.CreateMachine{
		OrganizationID: orgA.ID,
		Name:           "builder-a",
		Host:           "10.0.0.25",
		Port:           22,
		SSHUser:        strPtr("ubuntu"),
		SSHKeyPath:     strPtr("/tmp/id_builder_a"),
		Description:    "Build worker",
		Status:         domain.MachineStatusOnline,
	})
	if err != nil {
		t.Fatalf("CreateMachine() remote error = %v", err)
	}
	if _, err := repo.CreateMachine(ctx, domain.CreateMachine{
		OrganizationID: orgA.ID,
		Name:           "builder-a",
		Host:           "10.0.0.26",
		Port:           22,
		SSHUser:        strPtr("ubuntu"),
		SSHKeyPath:     strPtr("/tmp/id_builder_a_dup"),
		Description:    "Duplicate machine name",
		Status:         domain.MachineStatusOnline,
	}); !errors.Is(err, domain.ErrMachineNameConflict) {
		t.Fatalf("CreateMachine() duplicate name error = %v, want %v", err, domain.ErrMachineNameConflict)
	}

	if _, err := repo.CreateMachine(ctx, domain.CreateMachine{
		OrganizationID: uuid.New(),
		Name:           "builder-missing",
		Host:           "10.0.0.30",
		Port:           22,
		SSHUser:        strPtr("ubuntu"),
		SSHKeyPath:     strPtr("/tmp/id_builder_missing"),
		Description:    "Missing organization",
		Status:         domain.MachineStatusOnline,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CreateMachine() missing organization error = %v, want %v", err, ErrNotFound)
	}

	if _, err := repo.UpdateMachine(ctx, domain.UpdateMachine{
		ID:             remoteMachine.ID,
		OrganizationID: orgB.ID,
		Name:           remoteMachine.Name,
		Host:           remoteMachine.Host,
		Port:           remoteMachine.Port,
		Description:    remoteMachine.Description,
		Status:         remoteMachine.Status,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateMachine() organization mismatch error = %v, want %v", err, ErrInvalidInput)
	}

	if _, err := repo.UpdateMachine(ctx, domain.UpdateMachine{
		ID:             localMachine.ID,
		OrganizationID: orgA.ID,
		Name:           "renamed-local",
		Host:           localMachine.Host,
		Port:           localMachine.Port,
		Description:    localMachine.Description,
		Status:         localMachine.Status,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateMachine() local machine mutation error = %v, want %v", err, ErrInvalidInput)
	}

	if _, err := repo.CreateAgentProvider(ctx, domain.CreateAgentProvider{
		OrganizationID:     orgA.ID,
		MachineID:          machinesB[0].ID,
		Name:               "Cross Org Provider",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		AuthConfig:         map[string]any{},
		ModelName:          "gpt-5.4",
		ModelTemperature:   0,
		ModelMaxTokens:     0,
		CostPerInputToken:  0,
		CostPerOutputToken: 0,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateAgentProvider() organization mismatch error = %v, want %v", err, ErrInvalidInput)
	}
	if _, err := repo.CreateAgentProvider(ctx, domain.CreateAgentProvider{
		OrganizationID:     orgA.ID,
		MachineID:          remoteMachine.ID,
		Name:               "Builder Codex",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		AuthConfig:         map[string]any{},
		ModelName:          "gpt-5.4",
		ModelTemperature:   0,
		ModelMaxTokens:     0,
		CostPerInputToken:  0,
		CostPerOutputToken: 0,
	}); err != nil {
		t.Fatalf("CreateAgentProvider() machine-bound provider error = %v", err)
	}
	if _, err := repo.CreateAgentProvider(ctx, domain.CreateAgentProvider{
		OrganizationID:     orgA.ID,
		MachineID:          localMachine.ID,
		Name:               "OpenAI Codex",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		AuthConfig:         map[string]any{},
		ModelName:          "gpt-5.4",
		ModelTemperature:   0,
		ModelMaxTokens:     0,
		CostPerInputToken:  0,
		CostPerOutputToken: 0,
	}); !errors.Is(err, domain.ErrAgentProviderNameConflict) {
		t.Fatalf("CreateAgentProvider() duplicate name error = %v, want %v", err, domain.ErrAgentProviderNameConflict)
	}
	if _, err := repo.DeleteMachine(ctx, remoteMachine.ID); !errors.Is(err, domain.ErrMachineInUseConflict) {
		t.Fatalf("DeleteMachine() in-use error = %v, want %v", err, domain.ErrMachineInUseConflict)
	}

	if _, err := repo.ArchiveOrganization(ctx, orgB.ID); err != nil {
		t.Fatalf("ArchiveOrganization() orgB error = %v", err)
	}
	orgs, err := repo.ListOrganizations(ctx)
	if err != nil {
		t.Fatalf("ListOrganizations() after archive error = %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != orgA.ID {
		t.Fatalf("ListOrganizations() after archive = %+v", orgs)
	}
}

func TestEntRepositoryConflictAndNotFoundPaths(t *testing.T) {
	client := openRepoCatalogTestEntClient(t)
	ctx := context.Background()
	repo := NewEntRepository(client)

	org, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "Better And Better",
		Slug: "better-and-better",
	})
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	if _, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "Duplicate Org",
		Slug: "better-and-better",
	}); !errors.Is(err, domain.ErrOrganizationSlugConflict) {
		t.Fatalf("CreateOrganization(duplicate slug) error = %v, want %v", err, domain.ErrOrganizationSlugConflict)
	}

	if _, err := repo.GetOrganization(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetOrganization(missing) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.ListProjects(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListProjects(missing org) error = %v, want %v", err, ErrNotFound)
	}

	project, err := repo.CreateProject(ctx, domain.CreateProject{
		OrganizationID:      org.ID,
		Name:                "OpenASE",
		Slug:                "openase",
		Description:         "Coverage rollout",
		Status:              domain.ProjectStatusInProgress,
		MaxConcurrentAgents: 2,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := repo.CreateProject(ctx, domain.CreateProject{
		OrganizationID:      org.ID,
		Name:                "OpenASE Duplicate",
		Slug:                "openase",
		Description:         "Duplicate slug",
		Status:              domain.ProjectStatusInProgress,
		MaxConcurrentAgents: 2,
	}); !errors.Is(err, domain.ErrProjectSlugConflict) {
		t.Fatalf("CreateProject(duplicate slug) error = %v, want %v", err, domain.ErrProjectSlugConflict)
	}

	projectRepo, err := repo.CreateProjectRepo(ctx, domain.CreateProjectRepo{
		ProjectID:     project.ID,
		Name:          "openase-main",
		RepositoryURL: "https://github.com/PacificStudio/openase.git",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreateProjectRepo() error = %v", err)
	}
	if _, err := repo.CreateProjectRepo(ctx, domain.CreateProjectRepo{
		ProjectID:     project.ID,
		Name:          "openase-main",
		RepositoryURL: "https://github.com/PacificStudio/openase-other.git",
		DefaultBranch: "main",
	}); !errors.Is(err, domain.ErrProjectRepoNameConflict) {
		t.Fatalf("CreateProjectRepo(duplicate name) error = %v, want %v", err, domain.ErrProjectRepoNameConflict)
	}
	if _, err := repo.CreateProjectRepo(ctx, domain.CreateProjectRepo{
		ProjectID:     uuid.New(),
		Name:          "missing-project",
		RepositoryURL: "https://github.com/GrandCX/missing.git",
		DefaultBranch: "main",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CreateProjectRepo(missing project) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.ListProjectRepos(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListProjectRepos(missing project) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.GetProjectRepo(ctx, project.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProjectRepo(missing) error = %v, want %v", err, ErrNotFound)
	}

	secondaryRepo, err := repo.CreateProjectRepo(ctx, domain.CreateProjectRepo{
		ProjectID:     project.ID,
		Name:          "worker-tools",
		RepositoryURL: "https://github.com/GrandCX/worker-tools.git",
		DefaultBranch: "develop",
	})
	if err != nil {
		t.Fatalf("CreateProjectRepo() secondary error = %v", err)
	}
	if _, err := repo.UpdateProjectRepo(ctx, domain.UpdateProjectRepo{
		ID:            secondaryRepo.ID,
		ProjectID:     project.ID,
		Name:          projectRepo.Name,
		RepositoryURL: secondaryRepo.RepositoryURL,
		DefaultBranch: secondaryRepo.DefaultBranch,
	}); !errors.Is(err, domain.ErrProjectRepoNameConflict) {
		t.Fatalf("UpdateProjectRepo(duplicate name) error = %v, want %v", err, domain.ErrProjectRepoNameConflict)
	}
	if _, err := repo.UpdateProjectRepo(ctx, domain.UpdateProjectRepo{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		Name:          "missing",
		RepositoryURL: "https://github.com/GrandCX/missing.git",
		DefaultBranch: "main",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateProjectRepo(missing) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.DeleteProjectRepo(ctx, project.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteProjectRepo(missing) error = %v, want %v", err, ErrNotFound)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("ResetToDefaultTemplate() error = %v", err)
	}
	todoID := findRepoCatalogStatusIDByName(t, statuses, "Todo")
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-300").
		SetTitle("Exercise repo scope conflicts").
		SetStatusID(todoID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := repo.CreateTicketRepoScope(ctx, domain.CreateTicketRepoScope{
		ProjectID: project.ID,
		TicketID:  ticketItem.ID,
		RepoID:    projectRepo.ID,
	}); err != nil {
		t.Fatalf("CreateTicketRepoScope() error = %v", err)
	}
	if _, err := repo.CreateTicketRepoScope(ctx, domain.CreateTicketRepoScope{
		ProjectID: project.ID,
		TicketID:  ticketItem.ID,
		RepoID:    projectRepo.ID,
	}); !errors.Is(err, domain.ErrTicketRepoScopeConflict) {
		t.Fatalf("CreateTicketRepoScope(duplicate repo) error = %v, want %v", err, domain.ErrTicketRepoScopeConflict)
	}
	if _, err := repo.GetTicketRepoScope(ctx, project.ID, ticketItem.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetTicketRepoScope(missing) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.DeleteProjectRepo(ctx, project.ID, projectRepo.ID); !errors.Is(err, domain.ErrProjectRepoInUseConflict) {
		t.Fatalf("DeleteProjectRepo(in use) error = %v, want %v", err, domain.ErrProjectRepoInUseConflict)
	}
	if _, err := repo.ListTicketRepoScopes(ctx, project.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListTicketRepoScopes(missing ticket) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.CreateTicketRepoScope(ctx, domain.CreateTicketRepoScope{
		ProjectID: project.ID,
		TicketID:  ticketItem.ID,
		RepoID:    uuid.New(),
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CreateTicketRepoScope(missing repo) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.UpdateTicketRepoScope(ctx, domain.UpdateTicketRepoScope{
		ID:        uuid.New(),
		ProjectID: project.ID,
		TicketID:  ticketItem.ID,
		RepoID:    projectRepo.ID,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateTicketRepoScope(missing) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.DeleteTicketRepoScope(ctx, project.ID, ticketItem.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteTicketRepoScope(missing) error = %v, want %v", err, ErrNotFound)
	}

	if _, err := repo.ArchiveOrganization(ctx, org.ID); err != nil {
		t.Fatalf("ArchiveOrganization() error = %v", err)
	}
	if _, err := repo.GetOrganization(ctx, org.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetOrganization(archived) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.UpdateOrganization(ctx, domain.UpdateOrganization{
		ID:   org.ID,
		Name: "Archived Org",
		Slug: "archived-org",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateOrganization(archived) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.ListProjects(ctx, org.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListProjects(archived org) error = %v, want %v", err, ErrNotFound)
	}
}

func openRepoCatalogTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func findRepoCatalogStatusIDByName(t *testing.T, items []ticketstatus.Status, want string) uuid.UUID {
	t.Helper()

	for _, item := range items {
		if item.Name == want {
			return item.ID
		}
	}

	t.Fatalf("missing status %q in %+v", want, items)
	return uuid.Nil
}

func strPtr(value string) *string {
	return &value
}
