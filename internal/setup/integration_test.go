package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o750); err != nil {
		t.Fatalf("MkdirAll(repoRoot) error = %v", err)
	}
	repository, err := git.PlainInit(repoRoot, false)
	if err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	if err := repository.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")),
	); err != nil {
		t.Fatalf("set HEAD to main: %v", err)
	}
	if _, err := repository.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/GrandCX/openase.git"},
	}); err != nil {
		t.Fatalf("CreateRemote(origin) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("setup integration\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("Worktree() error = %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("Add(README.md) error = %v", err)
	}
	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	templates := catalog.BuiltinAgentProviderTemplates()
	input := InstallInput{
		Mode: ModeTeam,
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
		}},
		Project: ProjectConfig{
			Name:          "Setup Coverage Repo",
			RepoPath:      repoRoot,
			RepoURL:       "https://github.com/GrandCX/openase.git",
			DefaultBranch: "main",
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
	if repos[0].RepositoryURL != "https://github.com/GrandCX/openase.git" {
		t.Fatalf("repository URL = %q", repos[0].RepositoryURL)
	}

	mirrors, err := client.ProjectRepoMirror.Query().All(ctx)
	if err != nil {
		t.Fatalf("ProjectRepoMirror.Query().All() error = %v", err)
	}
	if len(mirrors) != 1 || mirrors[0].LocalPath != repoRoot || mirrors[0].State != "ready" {
		t.Fatalf("project repo mirrors = %+v", mirrors)
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
