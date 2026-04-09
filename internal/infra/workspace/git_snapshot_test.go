package workspace

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
)

type gitRunnerCall struct {
	args             []string
	allowExitCodeOne bool
	output           []byte
	err              error
}

func TestReadWorkspaceGitBranchUsesHEADWhenAvailable(t *testing.T) {
	t.Parallel()

	calls := []gitRunnerCall{{
		args:   []string{"git", "-C", "/tmp/repo", "rev-parse", "--abbrev-ref", "HEAD"},
		output: []byte("feature/ase-90\n"),
	}}
	run := gitRunnerStub(t, calls)

	branch, err := ReadWorkspaceGitBranch(context.Background(), "/tmp/repo", run)
	if err != nil {
		t.Fatalf("ReadWorkspaceGitBranch() error = %v", err)
	}
	if branch != "feature/ase-90" {
		t.Fatalf("branch = %q, want feature/ase-90", branch)
	}
}

func TestReadWorkspaceGitBranchFallsBackToSymbolicRefForUnbornHEAD(t *testing.T) {
	t.Parallel()

	calls := []gitRunnerCall{
		{
			args:   []string{"git", "-C", "/tmp/repo", "rev-parse", "--abbrev-ref", "HEAD"},
			output: []byte("fatal: ambiguous argument 'HEAD': unknown revision or path not in the working tree\n"),
			err:    fmt.Errorf("exit status 128"),
		},
		{
			args:   []string{"git", "-C", "/tmp/repo", "symbolic-ref", "-q", "--short", "HEAD"},
			output: []byte("main\n"),
		},
	}
	run := gitRunnerStub(t, calls)

	branch, err := ReadWorkspaceGitBranch(context.Background(), "/tmp/repo", run)
	if err != nil {
		t.Fatalf("ReadWorkspaceGitBranch() error = %v", err)
	}
	if branch != "main" {
		t.Fatalf("branch = %q, want main", branch)
	}
}

func TestReadWorkspaceGitBranchReturnsMissingWorkspaceError(t *testing.T) {
	t.Parallel()

	calls := []gitRunnerCall{{
		args:   []string{"git", "-C", "/tmp/missing", "rev-parse", "--abbrev-ref", "HEAD"},
		output: []byte("fatal: cannot change to '/tmp/missing': No such file or directory\n"),
		err:    fmt.Errorf("exit status 128"),
	}}
	run := gitRunnerStub(t, calls)

	_, err := ReadWorkspaceGitBranch(context.Background(), "/tmp/missing", run)
	if !errors.Is(err, ErrGitWorkspaceUnavailable) {
		t.Fatalf("ReadWorkspaceGitBranch() error = %v, want ErrGitWorkspaceUnavailable", err)
	}
}

func TestReadWorkspaceGitNumstatFallsBackToEmptyTreeForUnbornHEAD(t *testing.T) {
	t.Parallel()

	calls := []gitRunnerCall{
		{
			args:   []string{"git", "-C", "/tmp/repo", "diff", "--numstat", "-z", "-M", "HEAD", "--"},
			output: []byte("fatal: bad revision 'HEAD'\n"),
			err:    fmt.Errorf("exit status 128"),
		},
		{
			args:   []string{"git", "-C", "/tmp/repo", "diff", "--numstat", "-z", "-M", emptyTreeHash, "--"},
			output: []byte("2\t0\tREADME.md\x00"),
		},
	}
	run := gitRunnerStub(t, calls)

	output, err := ReadWorkspaceGitNumstat(context.Background(), "/tmp/repo", run)
	if err != nil {
		t.Fatalf("ReadWorkspaceGitNumstat() error = %v", err)
	}
	if string(output) != "2\t0\tREADME.md\x00" {
		t.Fatalf("output = %q, want numstat payload", string(output))
	}
}

func TestReadWorkspaceGitNumstatReturnsMissingWorkspaceError(t *testing.T) {
	t.Parallel()

	calls := []gitRunnerCall{{
		args:   []string{"git", "-C", "/tmp/missing", "diff", "--numstat", "-z", "-M", "HEAD", "--"},
		output: []byte("fatal: not a git repository (or any of the parent directories): .git\n"),
		err:    fmt.Errorf("exit status 128"),
	}}
	run := gitRunnerStub(t, calls)

	_, err := ReadWorkspaceGitNumstat(context.Background(), "/tmp/missing", run)
	if !errors.Is(err, ErrGitWorkspaceUnavailable) {
		t.Fatalf("ReadWorkspaceGitNumstat() error = %v, want ErrGitWorkspaceUnavailable", err)
	}
}

func gitRunnerStub(t *testing.T, calls []gitRunnerCall) GitCommandRunner {
	t.Helper()

	index := 0
	return func(_ context.Context, args []string, allowExitCodeOne bool) ([]byte, error) {
		t.Helper()
		if index >= len(calls) {
			t.Fatalf("unexpected git command: %v", args)
		}
		call := calls[index]
		index++
		if !slices.Equal(call.args, args) {
			t.Fatalf("git args = %v, want %v", args, call.args)
		}
		if allowExitCodeOne != call.allowExitCodeOne {
			t.Fatalf("allowExitCodeOne = %t, want %t", allowExitCodeOne, call.allowExitCodeOne)
		}
		return call.output, call.err
	}
}
