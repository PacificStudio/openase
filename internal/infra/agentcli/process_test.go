package agentcli

import (
	"context"
	"io"
	"os/exec"
	"runtime"
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

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(stdout) returned error: %v", err)
	}
	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		t.Fatalf("ReadAll(stderr) returned error: %v", err)
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}

	if string(stdout) != "stdout-line" {
		t.Fatalf("unexpected stdout: %q", string(stdout))
	}
	if string(stderr) != "stderr-line" {
		t.Fatalf("unexpected stderr: %q", string(stderr))
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

	running, ok := process.(*runningProcess)
	if !ok {
		t.Fatalf("expected *runningProcess, got %T", process)
	}

	<-running.done

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(stdout) returned error after exit: %v", err)
	}
	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		t.Fatalf("ReadAll(stderr) returned error after exit: %v", err)
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
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

func TestManagerStartCancelsProcessViaContext(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{StopGracePeriod: 50 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	spec := newShellSpec(t, command, "sleep 30")

	process, err := manager.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	cancel()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- process.Wait()
	}()

	select {
	case err := <-waitDone:
		if err == nil {
			t.Fatal("expected Wait to fail after context cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("process did not exit after context cancellation")
	}
}

func TestRunningProcessStopTerminatesRunningProcess(t *testing.T) {
	command := requirePOSIXShell(t)
	manager := NewManager(ManagerOptions{StopGracePeriod: 500 * time.Millisecond})
	spec := newShellSpec(t, command, "sleep 30")

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
