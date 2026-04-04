package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnoseHealthyEnvironment(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()

	writeFile(t, filepath.Join(repoRoot, ".git"), []byte("gitdir"))
	writeFile(t, filepath.Join(repoRoot, "config.yaml"), []byte("server:\n  mode: all-in-one\ndatabase:\n  dsn: postgres://openase:secret@localhost:5432/openase?sslmode=disable\n"))
	writeFileMode(t, filepath.Join(homeDir, ".openase", ".env"), []byte("OPENASE_TOKEN=x\n"), 0o600)
	mkdirAll(t, filepath.Join(homeDir, ".openase", "logs"))
	mkdirAll(t, filepath.Join(homeDir, ".openase", "workspaces"))

	report := Diagnose(context.Background(), Options{
		ConfigFile: filepath.Join(repoRoot, "config.yaml"),
		RepoRoot:   repoRoot,
		HomeDir:    homeDir,
		LookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		RunCommand: func(_ context.Context, name string, _ ...string) (string, error) {
			base := filepath.Base(name)
			switch base {
			case "git":
				return "git version 2.44.0\n", nil
			case "claude":
				return "claude 1.2.3\n", nil
			case "codex":
				return "codex 0.9.0\n", nil
			case "gemini":
				return "gemini 3.1.0\n", nil
			default:
				return base + " 0.0.0\n", nil
			}
		},
		PingDatabase: func(_ context.Context, _ string) error {
			return nil
		},
	})

	if report.ErrorCount() != 0 {
		t.Fatalf("expected no errors, got %d", report.ErrorCount())
	}
	if report.WarningCount() != 0 {
		t.Fatalf("expected no warnings, got %d", report.WarningCount())
	}

	assertStatus(t, report, "Git", StatusOK)
	assertStatus(t, report, "PostgreSQL", StatusOK)
	assertStatus(t, report, "Gemini CLI", StatusOK)

	rendered := report.Render()
	if !strings.Contains(rendered, "git version 2.44.0") {
		t.Fatalf("expected rendered report to include git version, got:\n%s", rendered)
	}
}

func TestDiagnoseReportsConfigAndPermissionProblems(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()

	writeFile(t, filepath.Join(repoRoot, ".git"), []byte("gitdir"))
	writeFile(t, filepath.Join(repoRoot, "config.yaml"), []byte("server: [\n"))
	writeFileMode(t, filepath.Join(homeDir, ".openase", ".env"), []byte("OPENASE_TOKEN=x\n"), 0o644)

	report := Diagnose(context.Background(), Options{
		ConfigFile: filepath.Join(repoRoot, "config.yaml"),
		RepoRoot:   repoRoot,
		HomeDir:    homeDir,
		LookPath: func(name string) (string, error) {
			if name == "git" {
				return "/usr/bin/git", nil
			}
			return "", os.ErrNotExist
		},
		RunCommand: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "git version 2.44.0\n", nil
		},
	})

	if report.ErrorCount() == 0 {
		t.Fatal("expected at least one error")
	}

	assertStatus(t, report, "配置", StatusError)
	assertStatus(t, report, "~/.openase", StatusWarning)
	assertStatus(t, report, "OpenAI Codex", StatusWarning)
	assertStatus(t, report, "Gemini CLI", StatusWarning)
}

func TestDoctorHelperFunctions(t *testing.T) {
	report := Report{Results: []Result{{Name: "warn", Status: StatusWarning}, {Name: "err", Status: StatusError}}}
	if !report.HasErrors() {
		t.Fatal("HasErrors() expected true")
	}

	candidates := configCandidates("/repo", "/home/codex")
	if len(candidates) != 8 || candidates[0] != filepath.Join("/repo", "config.yaml") || candidates[4] != filepath.Join("/home/codex", ".openase", "config.yaml") {
		t.Fatalf("configCandidates() = %+v", candidates)
	}

	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, ".git"), []byte("gitdir"))
	configPath := filepath.Join(repoRoot, "config.yaml")
	writeFile(t, configPath, []byte("server:\n  mode: all-in-one\n"))

	resolvedRepoRoot, err := resolveRepoRoot(repoRoot)
	if err != nil || resolvedRepoRoot != repoRoot {
		t.Fatalf("resolveRepoRoot(explicit) = %q, %v", resolvedRepoRoot, err)
	}

	resolvedConfigPath, err := resolveConfigPath(configPath, "", "")
	if err != nil || resolvedConfigPath != configPath {
		t.Fatalf("resolveConfigPath(explicit file) = %q, %v", resolvedConfigPath, err)
	}

	if _, err := resolveConfigPath(repoRoot, "", ""); err == nil || !strings.Contains(err.Error(), "must be a file") {
		t.Fatalf("resolveConfigPath(directory) error = %v", err)
	}

	resolvedCandidate, err := resolveConfigPath("", repoRoot, "")
	if err != nil || resolvedCandidate != configPath {
		t.Fatalf("resolveConfigPath(candidate) = %q, %v", resolvedCandidate, err)
	}
}

func TestDoctorExecAndPostgresHelpers(t *testing.T) {
	output, err := runExecCommand(context.Background(), "sh", "-c", "printf ok")
	if err != nil || output != "ok" {
		t.Fatalf("runExecCommand(success) = %q, %v", output, err)
	}

	if _, err := runExecCommand(context.Background(), "sh", "-c", "echo nope >&2; exit 7"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("runExecCommand(failure) error = %v", err)
	}

	dsn := startDoctorPostgres(t)
	if err := pingPostgres(context.Background(), dsn); err != nil {
		t.Fatalf("pingPostgres() error = %v", err)
	}
	if err := pingPostgres(context.Background(), "postgres://postgres:postgres@127.0.0.1:1/openase?sslmode=disable"); err == nil {
		t.Fatal("pingPostgres(invalid dsn) expected error")
	}
}

func assertStatus(t *testing.T, report Report, name string, want Status) {
	t.Helper()

	for _, result := range report.Results {
		if result.Name == name {
			if result.Status != want {
				t.Fatalf("result %q status=%q, want %q", name, result.Status, want)
			}
			return
		}
	}

	t.Fatalf("result %q not found", name)
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	writeFileMode(t, path, content, 0o600)
}

func writeFileMode(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatalf("WriteFile(%q) returned error: %v", path, err)
	}
}

func startDoctorPostgres(t *testing.T) string {
	t.Helper()

	return testPostgres.NewIsolatedDatabase(t).DSN
}
