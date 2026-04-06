package setup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
)

type stubResolver struct {
	paths map[string]string
}

func (s stubResolver) LookPath(name string) (string, error) {
	path, ok := s.paths[name]
	if !ok {
		return "", os.ErrNotExist
	}
	return path, nil
}

type stubConnector struct {
	pingDSN    string
	migrateDSN string
	pingErr    error
	migrateErr error
}

func (s *stubConnector) Ping(_ context.Context, dsn string) error {
	s.pingDSN = dsn
	return s.pingErr
}

func (s *stubConnector) Migrate(_ context.Context, dsn string) error {
	s.migrateDSN = dsn
	return s.migrateErr
}

type stubInstaller struct {
	input InstallInput
	err   error
}

func (s *stubInstaller) Initialize(_ context.Context, input InstallInput) error {
	s.input = input
	return s.err
}

func stubVersionRunner(_ context.Context, name string, _ ...string) (string, error) {
	switch filepath.Base(name) {
	case "git":
		return "git version 2.48.1\n", nil
	case "codex":
		return "codex 1.0.0\n", nil
	case "claude":
		return "claude 2.0.0\n", nil
	default:
		return "", os.ErrNotExist
	}
}

func TestServiceCompleteWritesRunnableFilesWithoutRepoScaffold(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("PATH", strings.Join([]string{"/usr/bin", "/custom/bin", "/usr/bin"}, string(os.PathListSeparator)))
	connector := &stubConnector{}
	installer := &stubInstaller{}
	service, err := NewService(Options{
		HomeDir:    homeDir,
		Resolver:   stubResolver{paths: map[string]string{"git": "/usr/bin/git", "codex": "/usr/local/bin/codex", "claude": "/usr/local/bin/claude"}},
		RunCommand: stubVersionRunner,
		Connector:  connector,
		Installer:  installer,
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Complete(context.Background(), RawCompleteRequest{
		Database: RawDatabaseInput{
			Host:     "127.0.0.1",
			Port:     5432,
			Name:     "openase",
			User:     "openase",
			Password: "secret",
			SSLMode:  "disable",
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if connector.migrateDSN == "" {
		t.Fatal("expected database migrate to be invoked")
	}
	if installer.input.Project.Name != DefaultProjectName {
		t.Fatalf("installer project = %+v", installer.input.Project)
	}
	if installer.input.Organization.Name != DefaultOrganizationName {
		t.Fatalf("installer organization = %+v", installer.input.Organization)
	}
	if len(installer.input.Agents) != 2 {
		t.Fatalf("expected available agents to be forwarded, got %+v", installer.input.Agents)
	}
	if result.ProjectName != DefaultProjectName || result.OrganizationName != DefaultOrganizationName {
		t.Fatalf("Complete result = %+v", result)
	}

	configPath := filepath.Join(homeDir, ".openase", "config.yaml")
	//nolint:gosec // test reads a controlled temp-home file created by setup.
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config returned error: %v", err)
	}
	configText := string(configContent)
	if !strings.Contains(configText, "project_name: "+DefaultProjectName) {
		t.Fatalf("expected config to contain project name, got %q", configText)
	}
	if !strings.Contains(configText, "mode: disabled") {
		t.Fatalf("expected config to contain disabled auth mode, got %q", configText)
	}
	if strings.Contains(configText, "repo_path:") || strings.Contains(configText, "mode: personal") {
		t.Fatalf("config should not contain legacy repo/mode setup fields, got %q", configText)
	}

	envPath := filepath.Join(homeDir, ".openase", ".env")
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("ReadFile env returned error: %v", err)
	}
	if !strings.HasPrefix(string(envContent), "OPENASE_AUTH_TOKEN=") {
		t.Fatalf("expected env file to contain auth token, got %q", string(envContent))
	}
	if !strings.Contains(string(envContent), "PATH=/usr/bin:/custom/bin\n") {
		t.Fatalf("expected env file to contain normalized PATH, got %q", string(envContent))
	}

	for _, dir := range []string{
		filepath.Join(homeDir, ".openase", "logs"),
		filepath.Join(homeDir, ".openase", "workspaces"),
	} {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			t.Fatalf("expected directory %s to exist, err=%v", dir, err)
		}
	}
}

func TestServiceCompleteWritesOIDCConfigThatLoads(t *testing.T) {
	homeDir := t.TempDir()
	service, err := NewService(Options{
		HomeDir:    homeDir,
		Resolver:   stubResolver{paths: map[string]string{"git": "/usr/bin/git"}},
		RunCommand: stubVersionRunner,
		Connector:  &stubConnector{},
		Installer:  &stubInstaller{},
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = service.Complete(context.Background(), RawCompleteRequest{
		Database: RawDatabaseInput{
			Host:     "127.0.0.1",
			Port:     5432,
			Name:     "openase",
			User:     "openase",
			Password: "secret",
			SSLMode:  "disable",
		},
		Auth: RawAuthInput{
			Mode: string(AuthModeOIDC),
			OIDC: &RawOIDCInput{
				IssuerURL:            "https://example.auth0.com/",
				ClientID:             "openase",
				ClientSecret:         "super-secret",
				RedirectURL:          DefaultOIDCRedirectURL,
				Scopes:               DefaultOIDCScopes,
				BootstrapAdminEmails: "admin@example.com",
				SessionTTL:           DefaultOIDCSessionTTL,
				SessionIdleTTL:       DefaultOIDCIdleTTL,
			},
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	cfg, err := config.Load(config.LoadOptions{
		ConfigFile: filepath.Join(homeDir, ".openase", "config.yaml"),
	})
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.Auth.Mode != config.AuthModeOIDC {
		t.Fatalf("auth mode = %q", cfg.Auth.Mode)
	}
	if cfg.Auth.OIDC.ClientSecret != "super-secret" {
		t.Fatalf("client secret = %q", cfg.Auth.OIDC.ClientSecret)
	}
	if len(cfg.Auth.OIDC.BootstrapAdminEmails) != 1 || cfg.Auth.OIDC.BootstrapAdminEmails[0] != "admin@example.com" {
		t.Fatalf("bootstrap admin emails = %v", cfg.Auth.OIDC.BootstrapAdminEmails)
	}
}

func TestBootstrapAndServerRoutesReflectTerminalFirstSetup(t *testing.T) {
	homeDir := t.TempDir()
	service, err := NewService(Options{
		HomeDir:    homeDir,
		Resolver:   stubResolver{paths: map[string]string{"git": "/usr/bin/git", "codex": "/usr/local/bin/codex"}},
		RunCommand: stubVersionRunner,
		Connector:  &stubConnector{},
		Installer:  &stubInstaller{},
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	bootstrap, err := service.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if len(bootstrap.Sources) != 2 {
		t.Fatalf("Bootstrap sources = %+v", bootstrap.Sources)
	}
	if len(bootstrap.CLI) < 4 {
		t.Fatalf("Bootstrap CLI diagnostics = %+v", bootstrap.CLI)
	}
	if !bootstrap.Agents[1].Available {
		t.Fatalf("expected codex to be detected, got %+v", bootstrap.Agents[1])
	}

	server := NewServer(ServerOptions{
		Host:    "127.0.0.1",
		Port:    19836,
		Service: service,
	})

	req := httptest.NewRequest(http.MethodGet, "/setup", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /setup to return 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/bootstrap", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected bootstrap route to return 200, got %d", rec.Code)
	}

	var payload Bootstrap
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected bootstrap JSON: %v", err)
	}
	if len(payload.Sources) != 2 || payload.Defaults.DockerDatabase.Port == 0 {
		t.Fatalf("bootstrap payload = %+v", payload)
	}
}
