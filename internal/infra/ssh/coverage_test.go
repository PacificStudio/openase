package ssh

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	gossh "golang.org/x/crypto/ssh"
)

func TestMonitorCollectorCoverage(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 27, 18, 0, 0, 0, time.UTC)
	collector := NewMonitorCollector(nil)
	if collector == nil || collector.now == nil || collector.runLocal == nil {
		t.Fatalf("NewMonitorCollector() = %+v", collector)
	}
	collector.now = func() time.Time { return now }
	collector.runLocal = func(_ context.Context, script string) ([]byte, error) {
		switch {
		case strings.Contains(script, "cpu_cores="):
			return []byte(strings.Join([]string{
				"cpu_cores=8",
				"cpu_usage_percent=12.50",
				"memory_total_kb=16777216",
				"memory_available_kb=8388608",
				"disk_total_kb=20971520",
				"disk_available_kb=10485760",
			}, "\n")), nil
		case strings.Contains(script, "nvidia-smi"):
			return []byte("0,Tesla T4,16384,8192,50.0\n"), nil
		case strings.Contains(script, "gh_cli"):
			return []byte("git\ttrue\tCodex\tcodex@example.com\ngh_cli\ttrue\tlogged_in\nnetwork\ttrue\tfalse\ttrue\n"), nil
		default:
			return []byte("claude_code\tfalse\t\tunknown\tunknown\ncodex\ttrue\t1.0\tlogged_in\tlogin\ngemini\tfalse\t\tunknown\tunknown\n"), nil
		}
	}

	local := domain.Machine{Name: domain.LocalMachineName, Host: domain.LocalMachineHost}

	systemResources, err := collector.CollectSystemResources(context.Background(), local)
	if err != nil {
		t.Fatalf("CollectSystemResources() error = %v", err)
	}
	if systemResources.CPUCores != 8 || systemResources.MemoryTotalGB == 0 || systemResources.DiskTotalGB == 0 {
		t.Fatalf("CollectSystemResources() = %+v", systemResources)
	}

	gpuResources, err := collector.CollectGPUResources(context.Background(), local)
	if err != nil {
		t.Fatalf("CollectGPUResources() error = %v", err)
	}
	if !gpuResources.Available || len(gpuResources.GPUs) != 1 || gpuResources.GPUs[0].Name != "Tesla T4" {
		t.Fatalf("CollectGPUResources() = %+v", gpuResources)
	}

	environment, err := collector.CollectAgentEnvironment(context.Background(), domain.Machine{
		Name:    domain.LocalMachineName,
		Host:    domain.LocalMachineHost,
		EnvVars: []string{"OPENAI_API_KEY=sk-test"},
	})
	if err != nil {
		t.Fatalf("CollectAgentEnvironment() error = %v", err)
	}
	if !environment.Dispatchable || len(environment.CLIs) != 3 {
		t.Fatalf("CollectAgentEnvironment() = %+v", environment)
	}

	fullAudit, err := collector.CollectFullAudit(context.Background(), local)
	if err != nil {
		t.Fatalf("CollectFullAudit() error = %v", err)
	}
	if !fullAudit.Git.Installed || !fullAudit.GitHubCLI.Installed || !fullAudit.Network.GitHubReachable {
		t.Fatalf("CollectFullAudit() = %+v", fullAudit)
	}
}

func TestMonitorCollectorRunScriptFailures(t *testing.T) {
	t.Parallel()

	local := domain.Machine{Name: domain.LocalMachineName, Host: domain.LocalMachineHost}
	collector := &MonitorCollector{}
	if _, err := collector.runScript(context.Background(), local, "echo hi"); err == nil || !strings.Contains(err.Error(), "local monitor runner unavailable") {
		t.Fatalf("runScript(nil local runner) error = %v", err)
	}

	collector.runLocal = func(context.Context, string) ([]byte, error) {
		return []byte("detail"), errors.New("boom")
	}
	if _, err := collector.runScript(context.Background(), local, "echo hi"); err == nil || !strings.Contains(err.Error(), "run local monitor script: boom: detail") {
		t.Fatalf("runScript(local failure) error = %v", err)
	}

	remote := testRemoteMachine()
	collector.runLocal = nil
	if _, err := collector.runScript(context.Background(), remote, "echo hi"); err == nil || !strings.Contains(err.Error(), "ssh pool unavailable") {
		t.Fatalf("runScript(remote without pool) error = %v", err)
	}
}

func TestSSHHelperAndRemoteProcessCoverage(t *testing.T) {
	t.Parallel()

	callback := gossh.InsecureIgnoreHostKey() //nolint:gosec
	pool := NewPool("/tmp/openase", WithTimeout(3*time.Second), WithHostKeyCallback(callback))
	if pool.timeout != 3*time.Second || pool.hostKeyCallback == nil {
		t.Fatalf("NewPool options = %+v", pool)
	}
	if got := pool.resolveKeyPath("/tmp/id_ed25519"); got != "/tmp/id_ed25519" {
		t.Fatalf("resolveKeyPath(abs) = %q", got)
	}
	if got := pool.resolveKeyPath("keys/id_ed25519"); got != "/tmp/openase/keys/id_ed25519" {
		t.Fatalf("resolveKeyPath(rel) = %q", got)
	}

	if err := joinErrors(); err != nil {
		t.Fatalf("joinErrors() = %v", err)
	}
	if err := joinErrors(errors.New("one")); err == nil || err.Error() != "one" {
		t.Fatalf("joinErrors(single) = %v", err)
	}
	if err := joinErrors(errors.New("one"), nil, errors.New("two")); err == nil || err.Error() != "one; two" {
		t.Fatalf("joinErrors(multi) = %v", err)
	}

	reader, writer := io.Pipe()
	process := &remoteProcess{
		session: &fakeSession{waitCh: make(chan error, 1)},
		stdin:   writer,
		stdout:  strings.NewReader("stdout"),
		stderr:  strings.NewReader("stderr"),
		done:    make(chan struct{}),
	}
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()
	if process.PID() != 0 || process.Stdin() == nil {
		t.Fatalf("remoteProcess accessors = (%d, %v)", process.PID(), process.Stdin())
	}
	stdoutBytes, err := io.ReadAll(process.Stdout())
	if err != nil || string(stdoutBytes) != "stdout" {
		t.Fatalf("Stdout() = (%q, %v)", string(stdoutBytes), err)
	}
	stderrBytes, err := io.ReadAll(process.Stderr())
	if err != nil || string(stderrBytes) != "stderr" {
		t.Fatalf("Stderr() = (%q, %v)", string(stderrBytes), err)
	}

	var nilProcess *remoteProcess
	if err := nilProcess.Wait(); err == nil || !strings.Contains(err.Error(), "process must not be nil") {
		t.Fatalf("nil Wait() error = %v", err)
	}
	if err := nilProcess.Stop(context.Background()); err == nil || !strings.Contains(err.Error(), "process must not be nil") {
		t.Fatalf("nil Stop() error = %v", err)
	}
	var nilCtx context.Context
	if err := process.Stop(nilCtx); err == nil || !strings.Contains(err.Error(), "context must not be nil") {
		t.Fatalf("Stop(nil ctx) error = %v", err)
	}
}
