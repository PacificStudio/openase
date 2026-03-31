package setup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/builtin"
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
}

func (s *stubConnector) Ping(_ context.Context, dsn string) error {
	s.pingDSN = dsn
	return nil
}

func (s *stubConnector) Migrate(_ context.Context, dsn string) error {
	s.migrateDSN = dsn
	return nil
}

type stubInstaller struct {
	input InstallInput
}

func (s *stubInstaller) Initialize(_ context.Context, input InstallInput) error {
	s.input = input
	return nil
}

func TestServiceCompleteWritesFilesAndScaffold(t *testing.T) {
	homeDir := t.TempDir()
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	connector := &stubConnector{}
	installer := &stubInstaller{}
	service, err := NewService(Options{
		HomeDir:   homeDir,
		Resolver:  stubResolver{paths: map[string]string{"codex": "/usr/local/bin/codex"}},
		Connector: connector,
		Installer: installer,
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Complete(context.Background(), RawCompleteRequest{
		Mode: "personal",
		Database: RawDatabaseInput{
			Host:     "localhost",
			Port:     5432,
			Name:     "openase",
			User:     "openase",
			Password: "secret",
			SSLMode:  "disable",
		},
		Agents: []string{"codex"},
		Project: RawProjectInput{
			Name:          "Demo App",
			RepoPath:      repoRoot,
			RepoURL:       "https://github.com/acme/demo.git",
			DefaultBranch: "main",
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if connector.migrateDSN == "" {
		t.Fatal("expected database migrate to be invoked")
	}
	if installer.input.Project.Name != "Demo App" {
		t.Fatalf("expected installer to receive project input, got %+v", installer.input)
	}
	if len(result.ScaffoldedFiles) != len(projectRepoScaffold(repoRoot)) {
		t.Fatalf("expected %d scaffolded files, got %d", len(projectRepoScaffold(repoRoot)), len(result.ScaffoldedFiles))
	}

	//nolint:gosec // test reads files from a controlled temp home directory
	configContent, err := os.ReadFile(filepath.Join(homeDir, ".openase", "config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile config returned error: %v", err)
	}
	if !strings.Contains(string(configContent), "project_name: Demo App") {
		t.Fatalf("expected config to contain project name, got %q", string(configContent))
	}
	if !strings.Contains(string(configContent), "observability:") {
		t.Fatalf("expected config to contain observability defaults, got %q", string(configContent))
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".openase", "openase.yaml")); err != nil {
		t.Fatalf("expected legacy config file to exist: %v", err)
	}

	envPath := filepath.Join(homeDir, ".openase", ".env")
	//nolint:gosec // test reads files from a controlled temp home directory
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("ReadFile env returned error: %v", err)
	}
	if !strings.HasPrefix(string(envContent), "OPENASE_AUTH_TOKEN=") {
		t.Fatalf("expected env file to contain auth token, got %q", string(envContent))
	}

	envInfo, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("Stat env returned error: %v", err)
	}
	if envInfo.Mode().Perm() != 0o600 {
		t.Fatalf("expected env mode 0600, got %#o", envInfo.Mode().Perm())
	}

	for _, path := range []string{
		filepath.Join(repoRoot, ".openase", "harnesses", "coding.md"),
		filepath.Join(repoRoot, ".openase", "skills", ".gitkeep"),
		filepath.Join(repoRoot, ".openase", "skills", "openase-platform", "SKILL.md"),
		filepath.Join(repoRoot, ".openase", "bin", "openase"),
		filepath.Join(repoRoot, ".openase", "harnesses", "roles", "fullstack-developer.md"),
		filepath.Join(repoRoot, ".openase", "harnesses", "roles", "data-analyst.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", path, err)
		}
	}

	for _, skill := range builtin.Skills() {
		if _, err := os.Stat(filepath.Join(repoRoot, ".openase", "skills", skill.Name, "SKILL.md")); err != nil {
			t.Fatalf("expected built-in skill scaffold for %s: %v", skill.Name, err)
		}
	}

	//nolint:gosec // test reads files from a controlled temp repository
	skillContent, err := os.ReadFile(filepath.Join(repoRoot, ".openase", "skills", "openase-platform", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile SKILL.md returned error: %v", err)
	}
	if !strings.Contains(string(skillContent), "./.openase/bin/openase ticket create") {
		t.Fatalf("expected built-in skill usage example, got %q", string(skillContent))
	}

	wrapperPath := filepath.Join(repoRoot, ".openase", "bin", "openase")
	wrapperInfo, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatalf("Stat wrapper returned error: %v", err)
	}
	if wrapperInfo.Mode().Perm() != 0o755 {
		t.Fatalf("expected wrapper mode 0755, got %#o", wrapperInfo.Mode().Perm())
	}

	fakeBinDir := t.TempDir()
	fakeOpenasePath := filepath.Join(fakeBinDir, "openase")
	if err := os.WriteFile(fakeOpenasePath, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile fake openase returned error: %v", err)
	}
	if err := os.Chmod(fakeOpenasePath, 0o700); err != nil { //nolint:gosec // test needs the temp helper script to be executable
		t.Fatalf("Chmod fake openase returned error: %v", err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	//nolint:gosec // test executes a temporary wrapper script under a controlled temp directory
	output, err := exec.Command(wrapperPath, "ticket", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper execution returned error: %v (output=%q)", err, string(output))
	}
	if strings.TrimSpace(string(output)) != "ticket list" {
		t.Fatalf("expected wrapper to forward arguments, got %q", string(output))
	}
}

func TestServerRoutes(t *testing.T) {
	homeDir := t.TempDir()
	service, err := NewService(Options{
		HomeDir:   homeDir,
		Resolver:  stubResolver{paths: map[string]string{"codex": "/usr/local/bin/codex"}},
		Connector: &stubConnector{},
		Installer: &stubInstaller{},
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
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
	if !strings.Contains(rec.Body.String(), "OpenASE Setup Wizard") {
		t.Fatalf("expected setup UI response, got %q", rec.Body.String())
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
	if len(payload.Agents) != 3 {
		t.Fatalf("expected 3 agent options, got %d", len(payload.Agents))
	}
	if !payload.Agents[1].Available {
		t.Fatalf("expected codex to be detected, got %+v", payload.Agents[1])
	}
}
