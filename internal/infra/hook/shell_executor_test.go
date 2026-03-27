package hook

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseTicketHooksParsesSupportedFields(t *testing.T) {
	parsed, err := ParseTicketHooks(map[string]any{
		"ticket_hooks": map[string]any{
			"on_complete": []any{
				map[string]any{
					"cmd":        "bash scripts/ci/run-tests.sh",
					"timeout":    float64(60),
					"workdir":    "backend",
					"on_failure": "warn",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseTicketHooks returned error: %v", err)
	}

	if len(parsed.OnComplete) != 1 {
		t.Fatalf("expected one on_complete hook, got %d", len(parsed.OnComplete))
	}

	hook := parsed.OnComplete[0]
	if hook.Command != "bash scripts/ci/run-tests.sh" {
		t.Fatalf("Command=%q, want bash scripts/ci/run-tests.sh", hook.Command)
	}
	if hook.Timeout != time.Minute {
		t.Fatalf("Timeout=%s, want 1m0s", hook.Timeout)
	}
	if hook.Workdir != "backend" {
		t.Fatalf("Workdir=%q, want backend", hook.Workdir)
	}
	if hook.OnFailure != FailurePolicyWarn {
		t.Fatalf("OnFailure=%q, want %q", hook.OnFailure, FailurePolicyWarn)
	}
}

func TestParseTicketHooksRejectsInvalidWorkdirType(t *testing.T) {
	_, err := ParseTicketHooks(map[string]any{
		"ticket_hooks": map[string]any{
			"on_complete": []any{
				map[string]any{
					"cmd":     "echo ok",
					"workdir": true,
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected ParseTicketHooks to fail")
	}
	if !strings.Contains(err.Error(), "workdir") {
		t.Fatalf("expected workdir error, got %v", err)
	}
}

func TestParseTicketHooksCoversEmptyAndInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		raw     map[string]any
		wantErr string
	}{
		{
			name: "empty config",
			raw:  map[string]any{},
		},
		{
			name: "missing ticket hooks",
			raw: map[string]any{
				"other": true,
			},
		},
		{
			name: "nil ticket hooks",
			raw: map[string]any{
				"ticket_hooks": nil,
			},
		},
		{
			name: "ticket hooks must be object",
			raw: map[string]any{
				"ticket_hooks": []any{},
			},
			wantErr: "ticket_hooks must be an object",
		},
		{
			name: "hook list must be slice",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": "invalid",
				},
			},
			wantErr: "on_done must be a list",
		},
		{
			name: "hook entry must be object",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": []any{"invalid"},
				},
			},
			wantErr: "must be an object",
		},
		{
			name: "missing cmd",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": []any{map[string]any{}},
				},
			},
			wantErr: ".cmd is required",
		},
		{
			name: "blank cmd",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": []any{
						map[string]any{"cmd": "   "},
					},
				},
			},
			wantErr: ".cmd must be a non-empty string",
		},
		{
			name: "fractional timeout",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": []any{
						map[string]any{"cmd": "echo ok", "timeout": 1.25},
					},
				},
			},
			wantErr: "whole number of seconds",
		},
		{
			name: "negative timeout",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": []any{
						map[string]any{"cmd": "echo ok", "timeout": -1},
					},
				},
			},
			wantErr: "greater than or equal to zero",
		},
		{
			name: "invalid on failure type",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": []any{
						map[string]any{"cmd": "echo ok", "on_failure": true},
					},
				},
			},
			wantErr: "must be one of block, warn, ignore",
		},
		{
			name: "invalid on failure value",
			raw: map[string]any{
				"ticket_hooks": map[string]any{
					"on_done": []any{
						map[string]any{"cmd": "echo ok", "on_failure": "panic"},
					},
				},
			},
			wantErr: "must be one of block, warn, ignore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseTicketHooks(tt.raw)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ParseTicketHooks returned error: %v", err)
				}
				if len(parsed.OnClaim) != 0 || len(parsed.OnStart) != 0 || len(parsed.OnComplete) != 0 ||
					len(parsed.OnDone) != 0 || len(parsed.OnError) != 0 || len(parsed.OnCancel) != 0 {
					t.Fatalf("expected empty hooks, got %+v", parsed)
				}
				return
			}

			if err == nil {
				t.Fatal("expected ParseTicketHooks to fail")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestParseTicketHooksSupportsMapSliceAndDefaults(t *testing.T) {
	parsed, err := ParseTicketHooks(map[string]any{
		"ticket_hooks": map[string]any{
			"on_claim": []map[string]any{
				{
					"cmd":        "echo ok",
					"workdir":    " subdir ",
					"on_failure": " IGNORE ",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseTicketHooks returned error: %v", err)
	}

	if len(parsed.OnClaim) != 1 {
		t.Fatalf("expected one on_claim hook, got %d", len(parsed.OnClaim))
	}

	hook := parsed.OnClaim[0]
	if hook.Command != "echo ok" {
		t.Fatalf("Command=%q, want echo ok", hook.Command)
	}
	if hook.Workdir != "subdir" {
		t.Fatalf("Workdir=%q, want subdir", hook.Workdir)
	}
	if hook.OnFailure != FailurePolicyIgnore {
		t.Fatalf("OnFailure=%q, want %q", hook.OnFailure, FailurePolicyIgnore)
	}
	if hook.Timeout != 0 {
		t.Fatalf("Timeout=%s, want 0", hook.Timeout)
	}
}

func TestShellExecutorInjectsEnvironmentAndResolvesRelativeWorkdir(t *testing.T) {
	workspace := t.TempDir()
	frontendDir := filepath.Join(workspace, "frontend")
	if err := os.MkdirAll(frontendDir, 0o750); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", frontendDir, err)
	}

	executor := NewShellExecutor()
	projectID := uuid.New()
	ticketID := uuid.New()
	env := Env{
		TicketID:         ticketID,
		ProjectID:        projectID,
		TicketIdentifier: "ASE-15",
		Workspace:        workspace,
		Repos: []Repo{
			{Name: "frontend", Path: frontendDir},
		},
		AgentName:    "codex-01",
		WorkflowType: "coding",
		Attempt:      2,
		APIURL:       "http://localhost:19836/api/v1/platform",
		AgentToken:   "ase_agent_token",
	}

	results, err := executor.RunAll(context.Background(), TicketHookOnComplete, []Definition{
		{
			Command: `pwd > pwd.txt
printf '%s' "$OPENASE_TICKET_ID" > ticket_id.txt
printf '%s' "$OPENASE_TICKET_IDENTIFIER" > identifier.txt
printf '%s' "$OPENASE_WORKSPACE" > workspace.txt
printf '%s' "$OPENASE_AGENT_NAME" > agent.txt
printf '%s' "$OPENASE_WORKFLOW_TYPE" > workflow_type.txt
printf '%s' "$OPENASE_ATTEMPT" > attempt.txt
printf '%s' "$OPENASE_HOOK_NAME" > hook_name.txt
printf '%s' "$OPENASE_REPOS" > repos.json
printf '%s' "$OPENASE_PROJECT_ID" > project_id.txt
printf '%s' "$OPENASE_API_URL" > api_url.txt
printf '%s' "$OPENASE_AGENT_TOKEN" > agent_token.txt`,
			Workdir: "frontend",
		},
	}, env)
	if err != nil {
		t.Fatalf("RunAll returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}

	result := results[0]
	if result.Outcome != OutcomePass {
		t.Fatalf("Outcome=%q, want %q", result.Outcome, OutcomePass)
	}
	if result.WorkingDirectory != frontendDir {
		t.Fatalf("WorkingDirectory=%q, want %q", result.WorkingDirectory, frontendDir)
	}

	assertFileContent(t, filepath.Join(frontendDir, "pwd.txt"), frontendDir)
	assertFileContent(t, filepath.Join(frontendDir, "ticket_id.txt"), ticketID.String())
	assertFileContent(t, filepath.Join(frontendDir, "identifier.txt"), "ASE-15")
	assertFileContent(t, filepath.Join(frontendDir, "workspace.txt"), workspace)
	assertFileContent(t, filepath.Join(frontendDir, "agent.txt"), "codex-01")
	assertFileContent(t, filepath.Join(frontendDir, "workflow_type.txt"), "coding")
	assertFileContent(t, filepath.Join(frontendDir, "attempt.txt"), "2")
	assertFileContent(t, filepath.Join(frontendDir, "hook_name.txt"), "on_complete")
	assertFileContent(t, filepath.Join(frontendDir, "project_id.txt"), projectID.String())
	assertFileContent(t, filepath.Join(frontendDir, "api_url.txt"), "http://localhost:19836/api/v1/platform")
	assertFileContent(t, filepath.Join(frontendDir, "agent_token.txt"), "ase_agent_token")

	var repos []Repo
	//nolint:gosec // test reads a file from the controlled temporary workspace
	reposRaw, err := os.ReadFile(filepath.Join(frontendDir, "repos.json"))
	if err != nil {
		t.Fatalf("ReadFile(repos.json) returned error: %v", err)
	}
	if err := json.Unmarshal(reposRaw, &repos); err != nil {
		t.Fatalf("Unmarshal(repos.json) returned error: %v", err)
	}
	if len(repos) != 1 || repos[0].Name != "frontend" || repos[0].Path != frontendDir {
		t.Fatalf("unexpected repos payload: %+v", repos)
	}
}

func TestShellExecutorHonorsFailurePolicies(t *testing.T) {
	workspace := t.TempDir()
	executor := NewShellExecutor()

	results, err := executor.RunAll(context.Background(), TicketHookOnError, []Definition{
		{
			Command:   "printf 'warn-out' && printf 'warn-err' >&2 && exit 7",
			OnFailure: FailurePolicyWarn,
		},
		{
			Command:   "printf 'ignore-out' && printf 'ignore-err' >&2 && exit 9",
			OnFailure: FailurePolicyIgnore,
		},
		{
			Command: "printf 'ok' > continued.txt",
		},
	}, Env{TicketID: uuid.New(), Workspace: workspace})
	if err != nil {
		t.Fatalf("RunAll returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected three results, got %d", len(results))
	}
	if results[0].Outcome != OutcomeError || results[0].ExitCode == nil || *results[0].ExitCode != 7 {
		t.Fatalf("unexpected warn result: %+v", results[0])
	}
	if results[1].Outcome != OutcomeError || results[1].ExitCode == nil || *results[1].ExitCode != 9 {
		t.Fatalf("unexpected ignore result: %+v", results[1])
	}
	if results[2].Outcome != OutcomePass {
		t.Fatalf("unexpected pass result: %+v", results[2])
	}
	assertFileContent(t, filepath.Join(workspace, "continued.txt"), "ok")
}

func TestShellExecutorBlocksOnBlockPolicy(t *testing.T) {
	workspace := t.TempDir()
	executor := NewShellExecutor()

	results, err := executor.RunAll(context.Background(), TicketHookOnComplete, []Definition{
		{Command: "printf 'broken' >&2 && exit 5"},
		{Command: "printf 'should-not-run' > skipped.txt"},
	}, Env{TicketID: uuid.New(), Workspace: workspace})
	if err == nil {
		t.Fatal("expected RunAll to return error")
	}
	if !errors.Is(err, ErrExecutionBlocked) {
		t.Fatalf("expected ErrExecutionBlocked, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result before block, got %d", len(results))
	}
	if results[0].Outcome != OutcomeError || results[0].ExitCode == nil || *results[0].ExitCode != 5 {
		t.Fatalf("unexpected block result: %+v", results[0])
	}
	if _, statErr := os.Stat(filepath.Join(workspace, "skipped.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected skipped.txt to be absent, got %v", statErr)
	}
}

func TestShellExecutorMarksTimeouts(t *testing.T) {
	workspace := t.TempDir()
	executor := NewShellExecutor()

	results, err := executor.RunAll(context.Background(), TicketHookOnComplete, []Definition{
		{
			Command: "sleep 1",
			Timeout: 10 * time.Millisecond,
		},
	}, Env{TicketID: uuid.New(), Workspace: workspace})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, ErrExecutionBlocked) {
		t.Fatalf("expected ErrExecutionBlocked, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].Outcome != OutcomeTimeout || !results[0].TimedOut {
		t.Fatalf("unexpected timeout result: %+v", results[0])
	}
}

func TestShellExecutorRejectsEscapingWorkdir(t *testing.T) {
	workspace := t.TempDir()
	executor := NewShellExecutor()

	results, err := executor.RunAll(context.Background(), TicketHookOnComplete, []Definition{
		{
			Command: "echo forbidden",
			Workdir: "../outside",
		},
	}, Env{TicketID: uuid.New(), Workspace: workspace})
	if err == nil {
		t.Fatal("expected invalid workdir error")
	}
	if !errors.Is(err, ErrExecutionBlocked) {
		t.Fatalf("expected ErrExecutionBlocked, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if !strings.Contains(results[0].Error, "escapes workspace") {
		t.Fatalf("unexpected result error: %+v", results[0])
	}
}

func TestResolveWorkingDirectoryAndHelpersCoverErrorBranches(t *testing.T) {
	workspace := t.TempDir()
	nestedDir := filepath.Join(workspace, "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	notDir := filepath.Join(workspace, "file.txt")
	if err := os.WriteFile(notDir, []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	resolved, err := resolveWorkingDirectory(workspace, nestedDir)
	if err != nil {
		t.Fatalf("resolveWorkingDirectory returned error: %v", err)
	}
	if resolved != nestedDir {
		t.Fatalf("resolved=%q, want %q", resolved, nestedDir)
	}

	if _, err := resolveWorkingDirectory("   ", ""); err == nil || !strings.Contains(err.Error(), "workspace must not be empty") {
		t.Fatalf("expected empty workspace error, got %v", err)
	}
	if _, err := resolveWorkingDirectory(workspace, "missing"); err == nil || !strings.Contains(err.Error(), "stat hook working directory") {
		t.Fatalf("expected missing workdir error, got %v", err)
	}
	if _, err := resolveWorkingDirectory(workspace, "file.txt"); err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("expected not-directory error, got %v", err)
	}

	if code, ok := extractExitCode(errors.New("plain error")); ok || code != 0 {
		t.Fatalf("extractExitCode() = (%d, %t), want (0, false)", code, ok)
	}
	if got := describeRunError(errors.New("plain error"), "   "); got != "plain error" {
		t.Fatalf("describeRunError() = %q, want plain error", got)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	//nolint:gosec // test reads a file from the controlled temporary workspace
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", path, err)
	}

	if strings.TrimSpace(string(content)) != want {
		t.Fatalf("content of %q = %q, want %q", path, strings.TrimSpace(string(content)), want)
	}
}
