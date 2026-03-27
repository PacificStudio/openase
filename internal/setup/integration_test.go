package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

func TestRuntimeDatabaseConnectorAndDefaultInstallerIntegration(t *testing.T) {
	t.Parallel()

	dsn := openSetupTestDSN(t)
	ctx := context.Background()

	connector := runtimeDatabaseConnector{}
	if err := connector.Ping(ctx, dsn); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if err := connector.Migrate(ctx, dsn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}

	templates := catalog.BuiltinAgentProviderTemplates()
	input := InstallInput{
		Mode: ModeTeam,
		Database: DatabaseConfig{
			Host:     "127.0.0.1",
			Port:     setupTestPortFromDSN(t, dsn),
			Name:     "openase",
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
		}},
		Project: ProjectConfig{
			Name:            "Setup Coverage Repo",
			PrimaryRepoPath: repoRoot,
			DefaultBranch:   "main",
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
	if orgs[0].Slug != "team-setup-coverage-repo" {
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
	if projects[0].DefaultAgentProviderID == nil {
		t.Fatalf("project default agent provider = nil")
	}

	repos, err := client.ProjectRepo.Query().All(ctx)
	if err != nil {
		t.Fatalf("ProjectRepo.Query().All() error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("project repo count = %d, want 1", len(repos))
	}
	if repos[0].RepositoryURL != repoRoot {
		t.Fatalf("repository URL = %q, want repo path fallback %q", repos[0].RepositoryURL, repoRoot)
	}

	statuses, err := client.TicketStatus.Query().All(ctx)
	if err != nil {
		t.Fatalf("TicketStatus.Query().All() error = %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("expected seeded default ticket statuses")
	}
}

func openSetupTestDSN(t *testing.T) string {
	t.Helper()

	port := freeSetupPort(t)
	dataDir := t.TempDir()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(uint32(port)).
			Username("postgres").
			Password("postgres").
			Database("openase").
			RuntimePath(filepath.Join(dataDir, "runtime")).
			BinariesPath(filepath.Join(dataDir, "binaries")).
			DataPath(filepath.Join(dataDir, "data")),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})

	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/openase?sslmode=disable", port)
}

func setupTestPortFromDSN(t *testing.T, dsn string) int {
	t.Helper()

	var port int
	if _, err := fmt.Sscanf(dsn, "postgres://postgres:postgres@127.0.0.1:%d/openase?sslmode=disable", &port); err != nil {
		t.Fatalf("parse test dsn %q: %v", dsn, err)
	}
	return port
}
