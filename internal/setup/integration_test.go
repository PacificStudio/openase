package setup

import (
	"context"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
)

func TestRuntimeDatabaseConnectorAndDefaultInstallerIntegration(t *testing.T) {
	t.Parallel()

	databaseInfo := openSetupTestDatabase(t)
	dsn := databaseInfo.DSN
	ctx := context.Background()

	connector := runtimeDatabaseConnector{}
	if err := connector.Ping(ctx, dsn); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if err := connector.Migrate(ctx, dsn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	templates := catalog.BuiltinAgentProviderTemplates()
	input := InstallInput{
		Database: DatabaseConfig{
			Host:     "127.0.0.1",
			Port:     int(databaseInfo.Port),
			Name:     databaseInfo.Name,
			User:     "postgres",
			Password: "postgres",
			SSLMode:  "disable",
		},
		Agents: []AgentOption{{
			ID:          templates[1].ID,
			Name:        templates[1].Name,
			Command:     templates[1].Command,
			AdapterType: templates[1].AdapterType,
			ModelName:   templates[1].ModelName,
			Available:   true,
		}},
		Organization: OrganizationConfig{
			Name: DefaultOrganizationName,
			Slug: DefaultOrganizationSlug,
		},
		Project: ProjectConfig{
			Name: DefaultProjectName,
			Slug: DefaultProjectSlug,
		},
	}

	if err := (defaultInstaller{}).Initialize(ctx, input); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	client, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("client.Close() error = %v", closeErr)
		}
	}()

	orgs, err := client.Organization.Query().All(ctx)
	if err != nil {
		t.Fatalf("Organization.Query().All() error = %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("organization count = %d, want 1", len(orgs))
	}
	if orgs[0].Slug != DefaultOrganizationSlug {
		t.Fatalf("organization slug = %q", orgs[0].Slug)
	}
	if orgs[0].DefaultAgentProviderID == nil {
		t.Fatalf("organization default agent provider = nil")
	}

	projects, err := client.Project.Query().All(ctx)
	if err != nil {
		t.Fatalf("Project.Query().All() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("project count = %d, want 1", len(projects))
	}
	if projects[0].Slug != DefaultProjectSlug {
		t.Fatalf("project slug = %q", projects[0].Slug)
	}
	if projects[0].DefaultAgentProviderID == nil {
		t.Fatalf("project default agent provider = nil")
	}

	repos, err := client.ProjectRepo.Query().All(ctx)
	if err != nil {
		t.Fatalf("ProjectRepo.Query().All() error = %v", err)
	}
	if len(repos) != 0 {
		t.Fatalf("project repo count = %d, want 0", len(repos))
	}

	statuses, err := client.TicketStatus.Query().All(ctx)
	if err != nil {
		t.Fatalf("TicketStatus.Query().All() error = %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("expected seeded default ticket statuses")
	}
}

func openSetupTestDatabase(t *testing.T) pgtest.Database {
	t.Helper()

	return testPostgres.NewIsolatedDatabase(t)
}
