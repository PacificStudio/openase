package agentcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

const defaultStopGracePeriod = 5 * time.Second

type ManagerOptions struct {
	StopGracePeriod time.Duration
}

type Manager struct {
	stopGracePeriod time.Duration
}

func NewManager(options ManagerOptions) provider.AgentCLIProcessManager {
	stopGracePeriod := options.StopGracePeriod
	if stopGracePeriod <= 0 {
		stopGracePeriod = defaultStopGracePeriod
	}

	return &Manager{
		stopGracePeriod: stopGracePeriod,
	}
}

func (m *Manager) Start(ctx context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context must not be nil")
	}
	if spec.Command == "" {
		return nil, fmt.Errorf("agent cli command must not be empty")
	}

	//nolint:gosec // command and arguments come from validated agent provider configuration
	cmd := exec.CommandContext(ctx, spec.Command.String(), spec.Args...)
	cmd.Cancel = func() error {
		return interruptProcess(cmd.Process)
	}
	cmd.WaitDelay = m.stopGracePeriod
	if spec.WorkingDirectory != nil {
		cmd.Dir = spec.WorkingDirectory.String()
	}
	if len(spec.Environment) > 0 {
		cmd.Env = append(os.Environ(), spec.Environment...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("open stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, fmt.Errorf("start agent cli process: %w", err)
	}

	process := &runningProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan struct{}),
	}

	return process, nil
}

type runningProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	done chan struct{}

	waitOnce sync.Once
	waitMu   sync.Mutex
	waitErr  error
}

func (p *runningProcess) PID() int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}

	return p.cmd.Process.Pid
}

func (p *runningProcess) Stdin() io.WriteCloser {
	return p.stdin
}

func (p *runningProcess) Stdout() io.ReadCloser {
	return p.stdout
}

func (p *runningProcess) Stderr() io.ReadCloser {
	return p.stderr
}

func (p *runningProcess) Wait() error {
	if p == nil {
		return fmt.Errorf("process must not be nil")
	}

	p.startWait()
	<-p.done

	p.waitMu.Lock()
	defer p.waitMu.Unlock()

	return p.waitErr
}

func (p *runningProcess) Stop(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("process must not be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}

	select {
	case <-p.done:
		return nil
	default:
	}

	if err := interruptProcess(p.cmd.Process); err != nil {
		return err
	}

	p.startWait()
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		if err := killProcess(p.cmd.Process); err != nil {
			return err
		}
		<-p.done
		return nil
	}
}

func (p *runningProcess) startWait() {
	p.waitOnce.Do(func() {
		go p.awaitExit()
	})
}

func (p *runningProcess) awaitExit() {
	err := p.cmd.Wait()

	p.waitMu.Lock()
	p.waitErr = err
	p.waitMu.Unlock()

	close(p.done)
}

func interruptProcess(process *os.Process) error {
	if process == nil {
		return fmt.Errorf("process not started")
	}

	if err := process.Signal(os.Interrupt); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return killProcess(process)
	}

	return nil
}

func killProcess(process *os.Process) error {
	if process == nil {
		return fmt.Errorf("process not started")
	}

	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill process: %w", err)
	}

	return nil
}
