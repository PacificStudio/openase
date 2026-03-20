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
	writeFile(t, filepath.Join(repoRoot, "openase.yaml"), []byte("server:\n  mode: all-in-one\ndatabase:\n  dsn: postgres://openase:secret@localhost:5432/openase?sslmode=disable\n"))
	writeFile(t, filepath.Join(repoRoot, ".openase", "harnesses", "coding.md"), []byte(`---
workflow_hooks:
  on_activate:
    - cmd: "claude --version"
ticket_hooks:
  on_complete:
    - cmd: "bash scripts/ci/run-tests.sh"
    - cmd: "./scripts/ci/lint.sh"
---
# Coding
`))
	writeFileMode(t, filepath.Join(repoRoot, "scripts", "ci", "run-tests.sh"), []byte("#!/usr/bin/env bash\n"), 0o755)
	writeFileMode(t, filepath.Join(repoRoot, "scripts", "ci", "lint.sh"), []byte("#!/usr/bin/env bash\n"), 0o755)
	writeFileMode(t, filepath.Join(homeDir, ".openase", ".env"), []byte("OPENASE_TOKEN=x\n"), 0o600)
	mkdirAll(t, filepath.Join(homeDir, ".openase", "logs"))

	report := Diagnose(context.Background(), Options{
		ConfigFile: filepath.Join(repoRoot, "openase.yaml"),
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
	assertStatus(t, report, "Harness", StatusOK)
	assertStatus(t, report, "Hook 脚本", StatusOK)

	rendered := report.Render()
	if !strings.Contains(rendered, "git version 2.44.0") {
		t.Fatalf("expected rendered report to include git version, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "已配置 2 个脚本") {
		t.Fatalf("expected rendered report to include hook script summary, got:\n%s", rendered)
	}
}

func TestDiagnoseReportsConfigAndPermissionProblems(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()

	writeFile(t, filepath.Join(repoRoot, ".git"), []byte("gitdir"))
	writeFile(t, filepath.Join(repoRoot, "openase.yaml"), []byte("server:\n  mode: invalid\n"))
	writeFile(t, filepath.Join(repoRoot, ".openase", "harnesses", "coding.md"), []byte(`---
ticket_hooks:
  on_complete:
    - cmd: "bash scripts/ci/run-tests.sh"
    - cmd: "bash scripts/ci/missing.sh"
---
`))
	writeFileMode(t, filepath.Join(repoRoot, "scripts", "ci", "run-tests.sh"), []byte("#!/usr/bin/env bash\n"), 0o644)
	writeFileMode(t, filepath.Join(homeDir, ".openase", ".env"), []byte("OPENASE_TOKEN=x\n"), 0o644)

	report := Diagnose(context.Background(), Options{
		ConfigFile: filepath.Join(repoRoot, "openase.yaml"),
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
	assertStatus(t, report, "Hook 脚本", StatusError)
	assertStatus(t, report, "~/.openase", StatusWarning)
	assertStatus(t, report, "Codex", StatusWarning)

	rendered := report.Render()
	if !strings.Contains(rendered, "chmod +x scripts/ci/run-tests.sh") {
		t.Fatalf("expected rendered report to suggest chmod fix, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "scripts/ci/missing.sh") {
		t.Fatalf("expected rendered report to include missing script, got:\n%s", rendered)
	}
}

func TestParseScriptReference(t *testing.T) {
	testCases := []struct {
		command string
		want    string
		ok      bool
	}{
		{command: "bash scripts/ci/run-tests.sh", want: "scripts/ci/run-tests.sh", ok: true},
		{command: "./scripts/ci/lint.sh", want: "scripts/ci/lint.sh", ok: true},
		{command: "git fetch origin && git checkout -b branch", ok: false},
	}

	for _, testCase := range testCases {
		got, ok := parseScriptReference(testCase.command)
		if ok != testCase.ok {
			t.Fatalf("parseScriptReference(%q) ok=%v, want %v", testCase.command, ok, testCase.ok)
		}
		if got != testCase.want {
			t.Fatalf("parseScriptReference(%q)=%q, want %q", testCase.command, got, testCase.want)
		}
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
