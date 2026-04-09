package setup

import (
	"context"
	"errors"
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
	if strings.Contains(configText, "\nauth:\n") || strings.Contains(configText, "auth:\n") {
		t.Fatalf("expected config to omit legacy auth block, got %q", configText)
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

func TestServiceCompleteWritesConfigThatLoadsWithoutLegacyAuthSection(t *testing.T) {
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
	// #nosec G304 -- Test reads the config it just wrote under a temp home directory.
	configBody, err := os.ReadFile(filepath.Join(homeDir, ".openase", "config.yaml"))
	if err != nil {
		t.Fatalf("read config body: %v", err)
	}
	if strings.Contains(string(configBody), "\nauth:\n") || strings.Contains(string(configBody), "auth:\n") {
		t.Fatalf("expected config to omit auth section, got %q", string(configBody))
	}
	if cfg.Database.DSN == "" {
		t.Fatalf("expected config.Load() to preserve the database config, got %+v", cfg)
	}
}

func TestBootstrapReflectsTerminalFirstSetup(t *testing.T) {
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

	if len(bootstrap.Sources) != 2 || bootstrap.Defaults.DockerDatabase.Port == 0 {
		t.Fatalf("bootstrap payload = %+v", bootstrap)
	}
}

func TestDesktopPreflightReportsMissingConfig(t *testing.T) {
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

	result, err := service.DesktopPreflight(context.Background())
	if err != nil {
		t.Fatalf("DesktopPreflight() error = %v", err)
	}
	if result.Ready {
		t.Fatalf("expected preflight to block startup, got %+v", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].Code != DesktopIssueConfigMissing {
		t.Fatalf("unexpected issues: %+v", result.Issues)
	}
}

func TestDesktopPreflightClassifiesAuthenticationFailures(t *testing.T) {
	homeDir := t.TempDir()
	configPath := filepath.Join(homeDir, "custom", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(configPath, []byte("database:\n  dsn: postgres://openase:bad@127.0.0.1:5432/openase?sslmode=disable\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	service, err := NewService(Options{
		HomeDir:    homeDir,
		ConfigPath: configPath,
		Resolver:   stubResolver{paths: map[string]string{"git": "/usr/bin/git"}},
		RunCommand: stubVersionRunner,
		Connector: &stubConnector{
			pingErr: errors.New("pq: password authentication failed for user \"openase\""),
		},
		Installer: &stubInstaller{},
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.DesktopPreflight(context.Background())
	if err != nil {
		t.Fatalf("DesktopPreflight() error = %v", err)
	}
	if result.Ready {
		t.Fatalf("expected preflight to fail, got %+v", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].Code != DesktopIssueDatabaseAuthFailed {
		t.Fatalf("unexpected issues: %+v", result.Issues)
	}
}

func TestDesktopApplyWritesConfiguredOverridePath(t *testing.T) {
	homeDir := t.TempDir()
	configPath := filepath.Join(homeDir, "desktop-configs", "openase-desktop.yaml")
	connector := &stubConnector{}
	service, err := NewService(Options{
		HomeDir:    homeDir,
		ConfigPath: configPath,
		Resolver:   stubResolver{paths: map[string]string{"git": "/usr/bin/git", "codex": "/usr/local/bin/codex"}},
		RunCommand: stubVersionRunner,
		Connector:  connector,
		Installer:  &stubInstaller{},
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.DesktopApply(context.Background(), RawDesktopApplyRequest{
		Database: RawDatabaseSourceInput{
			Type: "manual",
			Manual: &RawDatabaseInput{
				Host:     "127.0.0.1",
				Port:     5432,
				Name:     "openase",
				User:     "openase",
				Password: "secret",
				SSLMode:  "disable",
			},
		},
		AllowOverwrite: true,
	})
	if err != nil {
		t.Fatalf("DesktopApply() error = %v", err)
	}
	if !result.Ready {
		t.Fatalf("expected setup success, got %+v", result)
	}
	if result.ConfigPath != configPath {
		t.Fatalf("config path = %q", result.ConfigPath)
	}
	if connector.migrateDSN == "" {
		t.Fatal("expected migrations to run during DesktopApply")
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file at override path, err=%v", err)
	}
}
