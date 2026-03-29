package agentcli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestManagerStartCapturesStdoutAndStderr(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{})
	spec := newShellSpec(t, command, "printf 'stdout-line'; printf 'stderr-line' >&2")

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	stdoutDone := readAllAsync(process.Stdout())
	stderrDone := readAllAsync(process.Stderr())

	stdout := <-stdoutDone
	if stdout.err != nil {
		t.Fatalf("ReadAll(stdout) returned error: %v", stdout.err)
	}
	stderr := <-stderrDone
	if stderr.err != nil {
		t.Fatalf("ReadAll(stderr) returned error: %v", stderr.err)
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}

	if string(stdout.data) != "stdout-line" {
		t.Fatalf("unexpected stdout: %q", string(stdout.data))
	}
	if string(stderr.data) != "stderr-line" {
		t.Fatalf("unexpected stderr: %q", string(stderr.data))
	}
	if process.PID() <= 0 {
		t.Fatalf("expected a positive pid, got %d", process.PID())
	}
}

func TestManagerStartKeepsCapturedOutputReadableAfterProcessExit(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{})
	spec := newShellSpec(t, command, "printf 'stdout-line'; printf 'stderr-line' >&2")

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if err := process.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(stdout) returned error after exit: %v", err)
	}
	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		t.Fatalf("ReadAll(stderr) returned error after exit: %v", err)
	}

	if string(stdout) != "stdout-line" {
		t.Fatalf("unexpected stdout: %q", string(stdout))
	}
	if string(stderr) != "stderr-line" {
		t.Fatalf("unexpected stderr: %q", string(stderr))
	}
}

func TestManagerStartHonorsWorkingDirectoryAndEnvironment(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{})
	workdir := provider.MustParseAbsolutePath(t.TempDir())
	spec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand(command),
		[]string{"-c", "printf '%s|%s' \"$PWD\" \"$OPENASE_TEST_VALUE\""},
		&workdir,
		[]string{"OPENASE_TEST_VALUE=agent-cli"},
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(stdout) returned error: %v", err)
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}

	expected := workdir.String() + "|agent-cli"
	if string(stdout) != expected {
		t.Fatalf("expected %q, got %q", expected, string(stdout))
	}
}

func TestManagerStartRejectsCanceledContext(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{StopGracePeriod: 50 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := manager.Start(ctx, newShellSpec(t, command, "sleep 30")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start(canceled context) error = %v, want context.Canceled", err)
	}
}

func TestRunningProcessStopTerminatesRunningProcess(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{StopGracePeriod: 500 * time.Millisecond})
	spec := newShellSpec(t, command, "trap '' INT; exec sleep 30")

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = process.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if waitErr := process.Wait(); waitErr == nil {
		t.Fatal("expected Wait to report a terminated process")
	}
}

func TestManagerStartDoesNotTieProcessLifetimeToStartContext(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{StopGracePeriod: 200 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	spec := newShellSpec(t, command, "read line; printf '%s' \"$line\"; trap '' INT; sleep 30")

	process, err := manager.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	cancel()

	if _, err := io.WriteString(process.Stdin(), "still-running\n"); err != nil {
		t.Fatalf("WriteString(stdin) returned error after start context cancellation: %v", err)
	}
	if err := process.Stdin().Close(); err != nil {
		t.Fatalf("Close(stdin) returned error: %v", err)
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(stdout) returned error: %v", err)
	}
	if string(stdout) != "still-running" {
		t.Fatalf("unexpected stdout after start context cancellation: %q", string(stdout))
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := process.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	_ = process.Wait()
}

func TestRunningProcessStopKillsChildProcesses(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{StopGracePeriod: 200 * time.Millisecond})
	childPIDPath := t.TempDir() + "/child.pid"
	spec := newShellSpec(t, command, "sh -c 'trap \"\" INT; sleep 30' & echo $! > '"+childPIDPath+"'; trap '' INT; while :; do sleep 1; done")

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	var childPID int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		//nolint:gosec // Test reads a temp file path it created in the same function.
		raw, readErr := os.ReadFile(childPIDPath)
		if readErr == nil {
			childPID, err = strconv.Atoi(strings.TrimSpace(string(raw)))
			if err == nil && childPID > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if childPID <= 0 {
		t.Fatalf("expected child pid in %s, got %d", childPIDPath, childPID)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := process.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	_ = process.Wait()

	waitForProcessExit(t, childPID)
}

func TestManagerStartUsesStdinAndRejectsInvalidInputs(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{})
	var nilCtx context.Context

	if _, err := manager.Start(nilCtx, newShellSpec(t, command, "cat")); err == nil || err.Error() != "context must not be nil" {
		t.Fatalf("Start(nil context) error = %v", err)
	}

	emptySpec := provider.AgentCLIProcessSpec{}
	if _, err := manager.Start(context.Background(), emptySpec); err == nil || err.Error() != "agent cli command must not be empty" {
		t.Fatalf("Start(empty command) error = %v", err)
	}

	process, err := manager.Start(context.Background(), newShellSpec(t, command, "read line; printf '%s' \"$line\""))
	if err != nil {
		t.Fatalf("Start(stdin echo) error = %v", err)
	}
	if _, err := io.WriteString(process.Stdin(), "stdin-line\n"); err != nil {
		t.Fatalf("WriteString(stdin) error = %v", err)
	}
	if err := process.Stdin().Close(); err != nil {
		t.Fatalf("stdin.Close() error = %v", err)
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(stdout) error = %v", err)
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("Wait(stdin echo) error = %v", err)
	}
	if string(stdout) != "stdin-line" {
		t.Fatalf("stdout = %q, want stdin-line", string(stdout))
	}
}

func TestRunningProcessNilAndContextGuards(t *testing.T) {
	var nilProcess *runningProcess
	if got := nilProcess.PID(); got != 0 {
		t.Fatalf("PID(nil) = %d, want 0", got)
	}
	if err := nilProcess.Wait(); err == nil || err.Error() != "process must not be nil" {
		t.Fatalf("Wait(nil) error = %v", err)
	}
	if err := nilProcess.Stop(context.Background()); err == nil || err.Error() != "process must not be nil" {
		t.Fatalf("Stop(nil) error = %v", err)
	}

	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{})
	process, err := manager.Start(context.Background(), newShellSpec(t, command, "sleep 1"))
	if err != nil {
		t.Fatalf("Start(sleep) error = %v", err)
	}
	var nilCtx context.Context
	if err := process.Stop(nilCtx); err == nil || err.Error() != "context must not be nil" {
		t.Fatalf("Stop(nil context) error = %v", err)
	}
	_ = process.Stop(context.Background())
	_ = process.Wait()
}

func TestProcessOutputBufferCloseAndReadPaths(t *testing.T) {
	buffer := newProcessOutputBuffer()
	if _, err := buffer.Write([]byte("hello")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got := make([]byte, 5)
	n, err := buffer.Read(got)
	if err != nil || string(got[:n]) != "hello" {
		t.Fatalf("Read() = %d, %v, %q", n, err, string(got[:n]))
	}

	if err := buffer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	n, err = buffer.Read(got)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("Read() after Close = %d, %v", n, err)
	}
	if _, err := buffer.Write([]byte("x")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Write() after Close error = %v, want %v", err, io.ErrClosedPipe)
	}

	buffer = newProcessOutputBuffer()
	if err := buffer.closeWithError(io.ErrUnexpectedEOF); err != nil {
		t.Fatalf("closeWithError() error = %v", err)
	}
	n, err = buffer.Read(got)
	if n != 0 || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Read() after closeWithError = %d, %v", n, err)
	}
}

func TestInterruptKillAndPipeHelpers(t *testing.T) {
	command := requirePOSIXShell(t)

	if err := interruptProcess(nil); err == nil || err.Error() != "process not started" {
		t.Fatalf("interruptProcess(nil) error = %v", err)
	}
	if err := killProcess(nil); err == nil || err.Error() != "process not started" {
		t.Fatalf("killProcess(nil) error = %v", err)
	}

	cmd := newTestShellCommand(t, command, "sleep 30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start(interrupt) error = %v", err)
	}
	if err := interruptProcess(cmd.Process); err != nil {
		t.Fatalf("interruptProcess() error = %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("Wait(interrupt) expected signal exit error")
	}

	cmd = newTestShellCommand(t, command, "trap '' INT; sleep 30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start(kill) error = %v", err)
	}
	if err := killProcess(cmd.Process); err != nil {
		t.Fatalf("killProcess() error = %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("Wait(kill) expected signal exit error")
	}

	if !isProcessPipeClosedError(os.ErrClosed) {
		t.Fatal("isProcessPipeClosedError(os.ErrClosed) = false, want true")
	}
	if !isProcessPipeClosedError(errors.New("file already closed")) {
		t.Fatal("isProcessPipeClosedError(file already closed) = false, want true")
	}
	if isProcessPipeClosedError(errors.New("boom")) {
		t.Fatal("isProcessPipeClosedError(boom) = true, want false")
	}
}

type readResult struct {
	data []byte
	err  error
}

func readAllAsync(reader io.Reader) <-chan readResult {
	done := make(chan readResult, 1)
	go func() {
		data, err := io.ReadAll(reader)
		done <- readResult{data: data, err: err}
	}()
	return done
}

func requirePOSIXShell(t *testing.T) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell tests are not supported on windows")
	}

	command, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not available: %v", err)
	}

	return command
}

func newTestShellCommand(t *testing.T, command string, script string) *exec.Cmd {
	t.Helper()

	//nolint:gosec // Tests intentionally execute the discovered local POSIX shell.
	cmd := exec.Command(command, "-c", script)
	configureProcessGroup(cmd)
	return cmd
}

func newShellSpec(t *testing.T, command string, script string) provider.AgentCLIProcessSpec {
	t.Helper()

	spec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand(command),
		[]string{"-c", script},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	return spec
}

func waitForProcessExit(t *testing.T, pid int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, syscall.Signal(0))
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("process %d still exists after waiting for exit", pid)
}
