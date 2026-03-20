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

func TestShellExecutorInjectsEnvironmentAndResolvesRelativeWorkdir(t *testing.T) {
	workspace := t.TempDir()
	frontendDir := filepath.Join(workspace, "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", frontendDir, err)
	}

	executor := NewShellExecutor()
	ticketID := uuid.New()
	env := Env{
		TicketID:         ticketID,
		TicketIdentifier: "ASE-15",
		Workspace:        workspace,
		Repos: []Repo{
			{Name: "frontend", Path: frontendDir},
		},
		AgentName:    "codex-01",
		WorkflowType: "coding",
		Attempt:      2,
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
printf '%s' "$OPENASE_REPOS" > repos.json`,
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

	var repos []Repo
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

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", path, err)
	}

	if strings.TrimSpace(string(content)) != want {
		t.Fatalf("content of %q = %q, want %q", path, strings.TrimSpace(string(content)), want)
	}
}
