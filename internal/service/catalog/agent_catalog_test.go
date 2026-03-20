package catalog

import (
	"context"
	"errors"
	"testing"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/google/uuid"
)

func TestCreateAgentProviderAutoDetectsCLICommand(t *testing.T) {
	repo := &stubRepository{}
	svc := New(repo, stubExecutableResolver{
		paths: map[string]string{"codex": "/usr/local/bin/codex"},
	})

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
}

func TestCreateAgentProviderRejectsMissingCustomCLICommand(t *testing.T) {
	svc := New(&stubRepository{}, stubExecutableResolver{})

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
	svc := New(&stubRepository{}, stubExecutableResolver{})

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
	createdProvider *domain.CreateAgentProvider
}

func (r *stubRepository) ListOrganizations(context.Context) ([]domain.Organization, error) {
	return nil, nil
}

func (r *stubRepository) CreateOrganization(context.Context, domain.CreateOrganization) (domain.Organization, error) {
	return domain.Organization{}, nil
}

func (r *stubRepository) GetOrganization(context.Context, uuid.UUID) (domain.Organization, error) {
	return domain.Organization{}, nil
}

func (r *stubRepository) UpdateOrganization(context.Context, domain.UpdateOrganization) (domain.Organization, error) {
	return domain.Organization{}, nil
}

func (r *stubRepository) ListProjects(context.Context, uuid.UUID) ([]domain.Project, error) {
	return nil, nil
}

func (r *stubRepository) CreateProject(context.Context, domain.CreateProject) (domain.Project, error) {
	return domain.Project{}, nil
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
	return nil, nil
}

func (r *stubRepository) CreateAgentProvider(_ context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error) {
	r.createdProvider = &input

	return domain.AgentProvider{
		ID:             uuid.New(),
		OrganizationID: input.OrganizationID,
		Name:           input.Name,
		AdapterType:    input.AdapterType,
		CliCommand:     input.CliCommand,
		ModelName:      input.ModelName,
		AuthConfig:     input.AuthConfig,
	}, nil
}

func (r *stubRepository) GetAgentProvider(context.Context, uuid.UUID) (domain.AgentProvider, error) {
	return domain.AgentProvider{}, nil
}

func (r *stubRepository) UpdateAgentProvider(context.Context, domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	return domain.AgentProvider{}, nil
}

func (r *stubRepository) ListAgents(context.Context, uuid.UUID) ([]domain.Agent, error) {
	return nil, nil
}

func (r *stubRepository) ListActivityEvents(context.Context, domain.ListActivityEvents) ([]domain.ActivityEvent, error) {
	return nil, nil
}

func (r *stubRepository) CreateAgent(context.Context, domain.CreateAgent) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (r *stubRepository) GetAgent(context.Context, uuid.UUID) (domain.Agent, error) {
	return domain.Agent{}, nil
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
