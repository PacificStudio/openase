package catalog

import (
	"context"
	"errors"
	"testing"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func TestCreateAgentProviderAutoDetectsCLICommand(t *testing.T) {
	repo := &stubRepository{}
	svc := New(repo, stubExecutableResolver{
		paths: map[string]string{"codex": "/usr/local/bin/codex"},
	}, nil)

	item, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
		OrganizationID: uuid.New(),
		Name:           "Codex",
		AdapterType:    entagentprovider.AdapterTypeCodexAppServer,
		ModelName:      "gpt-5.3-codex",
		AuthConfig:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateAgentProvider returned error: %v", err)
	}
	if item.CliCommand != "/usr/local/bin/codex" {
		t.Fatalf("expected auto-detected cli command, got %+v", item)
	}
	if repo.createdProvider == nil || repo.createdProvider.CliCommand != "/usr/local/bin/codex" {
		t.Fatalf("expected repo to receive resolved cli command, got %+v", repo.createdProvider)
	}
	if want := []string{"app-server", "--listen", "stdio://"}; !equalStrings(item.CliArgs, want) {
		t.Fatalf("expected default codex cli args %v, got %v", want, item.CliArgs)
	}
}

func TestCreateAgentProviderRejectsMissingCustomCLICommand(t *testing.T) {
	svc := New(&stubRepository{}, stubExecutableResolver{}, nil)

	_, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
		OrganizationID: uuid.New(),
		Name:           "Custom",
		AdapterType:    entagentprovider.AdapterTypeCustom,
		ModelName:      "manual",
		AuthConfig:     map[string]any{},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestCreateAgentProviderRejectsMissingExecutable(t *testing.T) {
	svc := New(&stubRepository{}, stubExecutableResolver{}, nil)

	_, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
		OrganizationID: uuid.New(),
		Name:           "Gemini",
		AdapterType:    entagentprovider.AdapterTypeGeminiCli,
		ModelName:      "gemini-2.5-pro",
		AuthConfig:     map[string]any{},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestUpdateAgentProviderDefaultsCodexCLIArgs(t *testing.T) {
	repo := &stubRepository{
		provider: domain.AgentProvider{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Name:           "Codex",
			AdapterType:    entagentprovider.AdapterTypeCodexAppServer,
			CliCommand:     "/usr/local/bin/codex",
			ModelName:      "gpt-5.3-codex",
			AuthConfig:     map[string]any{},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.UpdateAgentProvider(context.Background(), domain.UpdateAgentProvider{
		ID:             repo.provider.ID,
		OrganizationID: repo.provider.OrganizationID,
		Name:           repo.provider.Name,
		AdapterType:    repo.provider.AdapterType,
		CliCommand:     repo.provider.CliCommand,
		ModelName:      repo.provider.ModelName,
		AuthConfig:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("UpdateAgentProvider returned error: %v", err)
	}
	if want := []string{"app-server", "--listen", "stdio://"}; !equalStrings(item.CliArgs, want) {
		t.Fatalf("expected default codex cli args %v, got %v", want, item.CliArgs)
	}
	if repo.updatedProvider == nil || !equalStrings(repo.updatedProvider.CliArgs, []string{"app-server", "--listen", "stdio://"}) {
		t.Fatalf("expected repo update to receive default codex args, got %+v", repo.updatedProvider)
	}
}

func TestListAgentProvidersAnnotatesAvailability(t *testing.T) {
	orgID := uuid.New()
	repo := &stubRepository{
		listedProviders: []domain.AgentProvider{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				Name:           "Claude Code",
				AdapterType:    entagentprovider.AdapterTypeClaudeCodeCli,
				CliCommand:     "claude",
				ModelName:      "claude-sonnet-4-5",
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				Name:           "OpenAI Codex",
				AdapterType:    entagentprovider.AdapterTypeCodexAppServer,
				CliCommand:     "codex",
				ModelName:      "gpt-5.3-codex",
			},
		},
	}
	svc := New(repo, stubExecutableResolver{
		paths: map[string]string{"codex": "/usr/local/bin/codex"},
	}, nil)

	items, err := svc.ListAgentProviders(context.Background(), orgID)
	if err != nil {
		t.Fatalf("ListAgentProviders returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 providers, got %+v", items)
	}
	if items[0].Available {
		t.Fatalf("expected claude provider to be unavailable, got %+v", items[0])
	}
	if !items[1].Available {
		t.Fatalf("expected codex provider to be available, got %+v", items[1])
	}
}

func TestCreateOrganizationSetsDefaultProviderToPreferredAvailableBuiltin(t *testing.T) {
	orgID := uuid.New()
	repo := &stubRepository{
		createdOrganization: domain.Organization{
			ID:   orgID,
			Name: "Acme",
			Slug: "acme",
		},
		listedProviders: []domain.AgentProvider{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				Name:           "Claude Code",
				AdapterType:    entagentprovider.AdapterTypeClaudeCodeCli,
				CliCommand:     "claude",
				ModelName:      "claude-sonnet-4-5",
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				Name:           "OpenAI Codex",
				AdapterType:    entagentprovider.AdapterTypeCodexAppServer,
				CliCommand:     "codex",
				ModelName:      "gpt-5.3-codex",
			},
		},
	}
	svc := New(repo, stubExecutableResolver{
		paths: map[string]string{"codex": "/usr/local/bin/codex"},
	}, nil)

	item, err := svc.CreateOrganization(context.Background(), domain.CreateOrganization{
		Name: "Acme",
		Slug: "acme",
	})
	if err != nil {
		t.Fatalf("CreateOrganization returned error: %v", err)
	}
	if item.DefaultAgentProviderID == nil {
		t.Fatalf("expected default provider to be set, got %+v", item)
	}
	if repo.updatedOrganization == nil || repo.updatedOrganization.DefaultAgentProviderID == nil {
		t.Fatalf("expected organization update with default provider, got %+v", repo.updatedOrganization)
	}
	if *item.DefaultAgentProviderID != *repo.updatedOrganization.DefaultAgentProviderID {
		t.Fatalf("expected returned org default %s to match repo update %s", item.DefaultAgentProviderID, repo.updatedOrganization.DefaultAgentProviderID)
	}
}

func TestCreateProjectSeedsDefaultStatuses(t *testing.T) {
	projectID := uuid.New()
	repo := &stubRepository{
		createdProject: domain.Project{
			ID:             projectID,
			OrganizationID: uuid.New(),
			Name:           "OpenASE",
			Slug:           "openase",
			Status:         "active",
		},
	}
	resetter := &stubProjectStatusResetter{}
	svc := New(repo, stubExecutableResolver{}, nil, WithProjectStatusResetter(resetter))

	item, err := svc.CreateProject(context.Background(), domain.CreateProject{
		OrganizationID: repo.createdProject.OrganizationID,
		Name:           "OpenASE",
		Slug:           "openase",
		Status:         "active",
	})
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	if item.ID != projectID {
		t.Fatalf("expected created project %s, got %+v", projectID, item)
	}
	if resetter.projectID != projectID {
		t.Fatalf("expected default status bootstrap for project %s, got %s", projectID, resetter.projectID)
	}
}

func TestRequestAgentPausePersistsPauseRequestedState(t *testing.T) {
	agentID := uuid.New()
	ticketID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			Status:              entagent.StatusRunning,
			CurrentTicketID:     &ticketID,
			RuntimePhase:        entagent.RuntimePhaseReady,
			RuntimeControlState: entagent.RuntimeControlStateActive,
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.RequestAgentPause(context.Background(), agentID)
	if err != nil {
		t.Fatalf("RequestAgentPause returned error: %v", err)
	}
	if item.RuntimeControlState != entagent.RuntimeControlStatePauseRequested {
		t.Fatalf("expected pause_requested state, got %+v", item)
	}
	if repo.updatedRuntimeControl == nil || repo.updatedRuntimeControl.RuntimeControlState != entagent.RuntimeControlStatePauseRequested {
		t.Fatalf("expected repo runtime control update, got %+v", repo.updatedRuntimeControl)
	}
}

func TestRequestAgentResumeRejectsPauseRequestedState(t *testing.T) {
	agentID := uuid.New()
	ticketID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			Status:              entagent.StatusClaimed,
			CurrentTicketID:     &ticketID,
			RuntimePhase:        entagent.RuntimePhaseNone,
			RuntimeControlState: entagent.RuntimeControlStatePauseRequested,
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	_, err := svc.RequestAgentResume(context.Background(), agentID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected runtime control conflict, got %v", err)
	}
}

type stubExecutableResolver struct {
	paths map[string]string
}

func (r stubExecutableResolver) LookPath(name string) (string, error) {
	if value, ok := r.paths[name]; ok {
		return value, nil
	}

	return "", errors.New("not found")
}

type stubRepository struct {
	createdProvider       *domain.CreateAgentProvider
	updatedProvider       *domain.UpdateAgentProvider
	updatedRuntimeControl *domain.UpdateAgentRuntimeControlState
	updatedOrganization   *domain.UpdateOrganization
	createdOrganization   domain.Organization
	createdProject        domain.Project
	listedProviders       []domain.AgentProvider
	provider              domain.AgentProvider
	agent                 domain.Agent
}

func (r *stubRepository) ListOrganizations(context.Context) ([]domain.Organization, error) {
	return nil, nil
}

func (r *stubRepository) CreateOrganization(context.Context, domain.CreateOrganization) (domain.Organization, error) {
	return r.createdOrganization, nil
}

func (r *stubRepository) GetOrganization(context.Context, uuid.UUID) (domain.Organization, error) {
	return domain.Organization{}, nil
}

func (r *stubRepository) UpdateOrganization(_ context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	r.updatedOrganization = &input
	r.createdOrganization = domain.Organization(input)
	return r.createdOrganization, nil
}

func (r *stubRepository) CountActiveProjects(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (r *stubRepository) DeleteOrganization(context.Context, uuid.UUID) (domain.Organization, error) {
	return r.createdOrganization, nil
}

func (r *stubRepository) ListProjects(context.Context, uuid.UUID) ([]domain.Project, error) {
	return nil, nil
}

func (r *stubRepository) ListMachines(context.Context, uuid.UUID) ([]domain.Machine, error) {
	return nil, nil
}

func (r *stubRepository) CreateMachine(context.Context, domain.CreateMachine) (domain.Machine, error) {
	return domain.Machine{}, nil
}

func (r *stubRepository) GetMachine(context.Context, uuid.UUID) (domain.Machine, error) {
	return domain.Machine{}, nil
}

func (r *stubRepository) UpdateMachine(context.Context, domain.UpdateMachine) (domain.Machine, error) {
	return domain.Machine{}, nil
}

func (r *stubRepository) DeleteMachine(context.Context, uuid.UUID) (domain.Machine, error) {
	return domain.Machine{}, nil
}

func (r *stubRepository) RecordMachineProbe(context.Context, domain.RecordMachineProbe) error {
	return nil
}

func (r *stubRepository) CreateProject(context.Context, domain.CreateProject) (domain.Project, error) {
	return r.createdProject, nil
}

func (r *stubRepository) GetProject(context.Context, uuid.UUID) (domain.Project, error) {
	return domain.Project{}, nil
}

func (r *stubRepository) UpdateProject(context.Context, domain.UpdateProject) (domain.Project, error) {
	return domain.Project{}, nil
}

func (r *stubRepository) ArchiveProject(context.Context, uuid.UUID) (domain.Project, error) {
	return domain.Project{}, nil
}

func (r *stubRepository) ListAgentProviders(context.Context, uuid.UUID) ([]domain.AgentProvider, error) {
	return append([]domain.AgentProvider(nil), r.listedProviders...), nil
}

func (r *stubRepository) CreateAgentProvider(_ context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error) {
	r.createdProvider = &input

	return domain.AgentProvider{
		ID:             uuid.New(),
		OrganizationID: input.OrganizationID,
		Name:           input.Name,
		AdapterType:    input.AdapterType,
		CliCommand:     input.CliCommand,
		CliArgs:        append([]string(nil), input.CliArgs...),
		ModelName:      input.ModelName,
		AuthConfig:     input.AuthConfig,
	}, nil
}

func (r *stubRepository) GetAgentProvider(context.Context, uuid.UUID) (domain.AgentProvider, error) {
	return r.provider, nil
}

func (r *stubRepository) UpdateAgentProvider(_ context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	r.updatedProvider = &input

	return domain.AgentProvider{
		ID:             input.ID,
		OrganizationID: input.OrganizationID,
		Name:           input.Name,
		AdapterType:    input.AdapterType,
		CliCommand:     input.CliCommand,
		CliArgs:        append([]string(nil), input.CliArgs...),
		ModelName:      input.ModelName,
		AuthConfig:     input.AuthConfig,
	}, nil
}

func (r *stubRepository) ListAgents(context.Context, uuid.UUID) ([]domain.Agent, error) {
	return nil, nil
}

func (r *stubRepository) ListActivityEvents(context.Context, domain.ListActivityEvents) ([]domain.ActivityEvent, error) {
	return nil, nil
}

func (r *stubRepository) ListAgentOutput(context.Context, domain.ListAgentOutput) ([]domain.AgentOutputEntry, error) {
	return nil, nil
}

func (r *stubRepository) CreateAgent(context.Context, domain.CreateAgent) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (r *stubRepository) GetAgent(context.Context, uuid.UUID) (domain.Agent, error) {
	return r.agent, nil
}

func (r *stubRepository) UpdateAgentRuntimeControlState(_ context.Context, input domain.UpdateAgentRuntimeControlState) (domain.Agent, error) {
	r.updatedRuntimeControl = &input
	r.agent.RuntimeControlState = input.RuntimeControlState
	return r.agent, nil
}

func (r *stubRepository) DeleteAgent(context.Context, uuid.UUID) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (r *stubRepository) ListProjectRepos(context.Context, uuid.UUID) ([]domain.ProjectRepo, error) {
	return nil, nil
}

func (r *stubRepository) CreateProjectRepo(context.Context, domain.CreateProjectRepo) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) GetProjectRepo(context.Context, uuid.UUID, uuid.UUID) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) UpdateProjectRepo(context.Context, domain.UpdateProjectRepo) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) DeleteProjectRepo(context.Context, uuid.UUID, uuid.UUID) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) ListTicketRepoScopes(context.Context, uuid.UUID, uuid.UUID) ([]domain.TicketRepoScope, error) {
	return nil, nil
}

func (r *stubRepository) CreateTicketRepoScope(context.Context, domain.CreateTicketRepoScope) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

func (r *stubRepository) GetTicketRepoScope(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

func (r *stubRepository) UpdateTicketRepoScope(context.Context, domain.UpdateTicketRepoScope) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

func (r *stubRepository) DeleteTicketRepoScope(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

var _ catalogrepo.Repository = (*stubRepository)(nil)

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

type stubProjectStatusResetter struct {
	projectID uuid.UUID
}

func (s *stubProjectStatusResetter) ResetToDefaultTemplate(_ context.Context, projectID uuid.UUID) ([]ticketstatus.Status, error) {
	s.projectID = projectID
	return nil, nil
}
