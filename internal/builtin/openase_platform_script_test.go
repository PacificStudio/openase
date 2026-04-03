package builtin

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenASEPlatformWorkpadScriptCreatesCommentWhenMissing(t *testing.T) {
	scriptPath := builtinWorkpadScriptPath(t)
	workspace := newFakeOpenASEWorkspace(t, `{"comments":[]}`)

	// #nosec G204 -- test executes a repo-local script under a controlled temp workspace.
	command := exec.Command("bash", scriptPath, "--body", "Progress\n- started")
	command.Dir = workspace.root
	command.Env = append(os.Environ(),
		"OPENASE_TICKET_ID=ticket-9",
		"FAKE_OPENASE_LOG="+workspace.logPath,
		"FAKE_OPENASE_LIST_FILE="+workspace.listPath,
		"FAKE_OPENASE_CREATED_BODY_FILE="+workspace.createdBodyPath,
		"FAKE_OPENASE_UPDATED_BODY_FILE="+workspace.updatedBodyPath,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("script returned error: %v\n%s", err, output)
	}

	logLines := builtinTestLogLines(t, workspace.logPath)
	if len(logLines) != 2 {
		t.Fatalf("expected 2 fake openase calls, got %v", logLines)
	}
	if logLines[0] != "ticket comment list ticket-9" {
		t.Fatalf("unexpected list invocation: %q", logLines[0])
	}
	if !strings.HasPrefix(logLines[1], "ticket comment create ticket-9 --body-file ") {
		t.Fatalf("unexpected create invocation: %q", logLines[1])
	}
	if got := mustReadBuiltinTestFile(t, workspace.createdBodyPath); got != "## Workpad\n\nProgress\n- started" {
		t.Fatalf("created workpad body = %q", got)
	}
}

func TestOpenASEPlatformWorkpadScriptUpdatesExistingCommentOnly(t *testing.T) {
	scriptPath := builtinWorkpadScriptPath(t)
	workspace := newFakeOpenASEWorkspace(t, `{"comments":[{"id":"comment-7","body_markdown":"## Workpad\n\nOld"}]}`)

	// #nosec G204 -- test executes a repo-local script under a controlled temp workspace.
	command := exec.Command("bash", scriptPath, "--body", "Validation\n- go test ./...")
	command.Dir = workspace.root
	command.Env = append(os.Environ(),
		"OPENASE_TICKET_ID=ticket-9",
		"FAKE_OPENASE_LOG="+workspace.logPath,
		"FAKE_OPENASE_LIST_FILE="+workspace.listPath,
		"FAKE_OPENASE_CREATED_BODY_FILE="+workspace.createdBodyPath,
		"FAKE_OPENASE_UPDATED_BODY_FILE="+workspace.updatedBodyPath,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("script returned error: %v\n%s", err, output)
	}

	logLines := builtinTestLogLines(t, workspace.logPath)
	if len(logLines) != 2 {
		t.Fatalf("expected 2 fake openase calls, got %v", logLines)
	}
	if logLines[0] != "ticket comment list ticket-9" {
		t.Fatalf("unexpected list invocation: %q", logLines[0])
	}
	if !strings.HasPrefix(logLines[1], "ticket comment update ticket-9 comment-7 --body-file ") {
		t.Fatalf("unexpected update invocation: %q", logLines[1])
	}
	if got := mustReadBuiltinTestFile(t, workspace.updatedBodyPath); got != "## Workpad\n\nValidation\n- go test ./..." {
		t.Fatalf("updated workpad body = %q", got)
	}
	if _, err := os.Stat(workspace.createdBodyPath); !os.IsNotExist(err) {
		t.Fatalf("expected no create payload, stat err=%v", err)
	}
}

func TestOpenASEPlatformWorkpadScriptPrependsHeadingWhenMissing(t *testing.T) {
	scriptPath := builtinWorkpadScriptPath(t)
	workspace := newFakeOpenASEWorkspace(t, `{"comments":[]}`)
	bodyFile := filepath.Join(workspace.root, "body.md")
	if err := os.WriteFile(bodyFile, []byte("Notes\n- captured"), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	// #nosec G204 -- test executes a repo-local script under a controlled temp workspace.
	command := exec.Command("bash", scriptPath, "--body-file", bodyFile)
	command.Dir = workspace.root
	command.Env = append(os.Environ(),
		"OPENASE_TICKET_ID=ticket-9",
		"FAKE_OPENASE_LOG="+workspace.logPath,
		"FAKE_OPENASE_LIST_FILE="+workspace.listPath,
		"FAKE_OPENASE_CREATED_BODY_FILE="+workspace.createdBodyPath,
		"FAKE_OPENASE_UPDATED_BODY_FILE="+workspace.updatedBodyPath,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("script returned error: %v\n%s", err, output)
	}

	if got := mustReadBuiltinTestFile(t, workspace.createdBodyPath); got != "## Workpad\n\nNotes\n- captured" {
		t.Fatalf("normalized workpad body = %q", got)
	}
}

func TestOpenASEPlatformWorkpadScriptRejectsInvalidArgumentCombinations(t *testing.T) {
	scriptPath := builtinWorkpadScriptPath(t)
	workspace := newFakeOpenASEWorkspace(t, `{"comments":[]}`)
	bodyFile := filepath.Join(workspace.root, "body.md")
	if err := os.WriteFile(bodyFile, []byte("ignored"), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	// #nosec G204 -- test executes a repo-local script under a controlled temp workspace.
	command := exec.Command("bash", scriptPath, "--body", "Progress", "--body-file", bodyFile)
	command.Dir = workspace.root
	command.Env = append(os.Environ(),
		"OPENASE_TICKET_ID=ticket-9",
		"FAKE_OPENASE_LOG="+workspace.logPath,
		"FAKE_OPENASE_LIST_FILE="+workspace.listPath,
		"FAKE_OPENASE_CREATED_BODY_FILE="+workspace.createdBodyPath,
		"FAKE_OPENASE_UPDATED_BODY_FILE="+workspace.updatedBodyPath,
	)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("expected invalid arguments to fail")
	}
	if !strings.Contains(string(output), "--body and --body-file are mutually exclusive") {
		t.Fatalf("unexpected output: %s", output)
	}
	if _, err := os.Stat(workspace.logPath); !os.IsNotExist(err) {
		t.Fatalf("expected fake openase not to be called, stat err=%v", err)
	}
}

func TestOpenASEPlatformWorkpadScriptFailsWithoutTicketContext(t *testing.T) {
	scriptPath := builtinWorkpadScriptPath(t)
	workspace := newFakeOpenASEWorkspace(t, `{"comments":[]}`)

	// #nosec G204 -- test executes a repo-local script under a controlled temp workspace.
	command := exec.Command("bash", scriptPath, "--body", "Progress")
	command.Dir = workspace.root
	command.Env = append(os.Environ(),
		"FAKE_OPENASE_LOG="+workspace.logPath,
		"FAKE_OPENASE_LIST_FILE="+workspace.listPath,
		"FAKE_OPENASE_CREATED_BODY_FILE="+workspace.createdBodyPath,
		"FAKE_OPENASE_UPDATED_BODY_FILE="+workspace.updatedBodyPath,
	)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("expected missing ticket context to fail")
	}
	if !strings.Contains(string(output), "ticket id is required via [ticket-id] or OPENASE_TICKET_ID") {
		t.Fatalf("unexpected output: %s", output)
	}
	if _, err := os.Stat(workspace.logPath); !os.IsNotExist(err) {
		t.Fatalf("expected fake openase not to be called, stat err=%v", err)
	}
}

type fakeOpenASEWorkspace struct {
	root            string
	logPath         string
	listPath        string
	createdBodyPath string
	updatedBodyPath string
}

func builtinWorkpadScriptPath(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file path")
	}
	return filepath.Join(filepath.Dir(currentFile), "skills", "openase-platform", "scripts", "upsert_workpad.sh")
}

func newFakeOpenASEWorkspace(t *testing.T, listPayload string) fakeOpenASEWorkspace {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".openase", "bin"), 0o750); err != nil {
		t.Fatalf("mkdir fake openase bin: %v", err)
	}

	workspace := fakeOpenASEWorkspace{
		root:            root,
		logPath:         filepath.Join(root, "openase.log"),
		listPath:        filepath.Join(root, "comments.json"),
		createdBodyPath: filepath.Join(root, "created.md"),
		updatedBodyPath: filepath.Join(root, "updated.md"),
	}
	if err := os.WriteFile(workspace.listPath, []byte(listPayload), 0o600); err != nil {
		t.Fatalf("write fake comments payload: %v", err)
	}

	fakeOpenASE := strings.TrimSpace(`#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >>"$FAKE_OPENASE_LOG"

copy_body_file() {
	local target=""
	local args=("$@")
	local index=0
	while [[ $index -lt ${#args[@]} ]]; do
		if [[ "${args[$index]}" == "--body-file" ]]; then
			local next_index=$((index + 1))
			target="${args[$next_index]}"
			break
		fi
		index=$((index + 1))
	done
	[[ -n "$target" ]] || exit 1
	cat "$target"
}

case "$1 $2 $3" in
	"ticket comment list")
		cat "$FAKE_OPENASE_LIST_FILE"
		;;
	"ticket comment create")
		copy_body_file "$@" >"$FAKE_OPENASE_CREATED_BODY_FILE"
		printf '{"comment":{"id":"comment-created"}}\n'
		;;
	"ticket comment update")
		copy_body_file "$@" >"$FAKE_OPENASE_UPDATED_BODY_FILE"
		printf '{"comment":{"id":"comment-updated"}}\n'
		;;
	*)
		echo "unexpected fake openase invocation: $*" >&2
		exit 1
		;;
esac
	`) + "\n"
	fakeOpenASEPath := filepath.Join(root, ".openase", "bin", "openase")
	if err := os.WriteFile(fakeOpenASEPath, []byte(fakeOpenASE), 0o600); err != nil {
		t.Fatalf("write fake openase wrapper: %v", err)
	}
	// #nosec G302 -- tests require an executable wrapper inside an isolated temp workspace.
	if err := os.Chmod(fakeOpenASEPath, 0o700); err != nil {
		t.Fatalf("chmod fake openase wrapper: %v", err)
	}
	return workspace
}

func mustReadBuiltinTestFile(t *testing.T, path string) string {
	t.Helper()

	// #nosec G304 -- tests only read files created inside isolated temp directories.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func builtinTestLogLines(t *testing.T, path string) []string {
	t.Helper()

	content := strings.TrimSpace(mustReadBuiltinTestFile(t, path))
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}
