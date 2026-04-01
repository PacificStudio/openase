package ssh

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

const sshInterruptSignal = "INT"

var _ = logging.DeclareComponent("ssh-process-manager")

type ProcessManager struct {
	pool    *Pool
	machine domain.Machine
}

func NewProcessManager(pool *Pool, machine domain.Machine) provider.AgentCLIProcessManager {
	return &ProcessManager{pool: pool, machine: machine}
}

func (m *ProcessManager) Start(ctx context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context must not be nil")
	}
	if m == nil || m.pool == nil {
		return nil, fmt.Errorf("ssh process manager unavailable")
	}
	if spec.Command == "" {
		return nil, fmt.Errorf("agent cli command must not be empty")
	}

	client, err := m.pool.Get(ctx, m.machine)
	if err != nil {
		return nil, fmt.Errorf("get ssh client for machine %s: %w", m.machine.Name, err)
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("open ssh session: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("open ssh stdin: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		_ = session.Close()
		return nil, fmt.Errorf("open ssh stdout: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = session.Close()
		return nil, fmt.Errorf("open ssh stderr: %w", err)
	}

	command := buildRemoteShellCommand(spec)
	if err := session.Start(command); err != nil {
		_ = stdin.Close()
		_ = session.Close()
		return nil, fmt.Errorf("start ssh process: %w", err)
	}

	process := &remoteProcess{
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		done:    make(chan struct{}),
	}
	go process.waitLoop()

	return process, nil
}

type remoteProcess struct {
	session Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	done    chan struct{}

	waitOnce sync.Once
	waitErr  error
}

func (p *remoteProcess) PID() int { return 0 }

func (p *remoteProcess) Stdin() io.WriteCloser { return p.stdin }

func (p *remoteProcess) Stdout() io.ReadCloser { return io.NopCloser(p.stdout) }

func (p *remoteProcess) Stderr() io.ReadCloser { return io.NopCloser(p.stderr) }

func (p *remoteProcess) Wait() error {
	if p == nil {
		return fmt.Errorf("process must not be nil")
	}
	p.awaitExit()
	return p.waitErr
}

func (p *remoteProcess) Stop(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("process must not be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}

	select {
	case <-p.done:
		p.awaitExit()
		return p.waitErr
	default:
	}

	_ = p.stdin.Close()
	if err := p.session.Signal(sshInterruptSignal); err != nil {
		_ = p.session.Close()
	}

	select {
	case <-p.done:
		p.awaitExit()
		return p.waitErr
	case <-ctx.Done():
		closeErr := p.session.Close()
		p.awaitExit()
		if p.waitErr != nil {
			return p.waitErr
		}
		if closeErr != nil {
			return closeErr
		}
		return p.waitErr
	}
}

func (p *remoteProcess) waitLoop() {
	p.waitErr = p.session.Wait()
	_ = p.session.Close()
	close(p.done)
}

func (p *remoteProcess) awaitExit() {
	p.waitOnce.Do(func() {
		<-p.done
	})
}

func buildRemoteShellCommand(spec provider.AgentCLIProcessSpec) string {
	commandParts := make([]string, 0, 1+len(spec.Args))
	commandParts = append(commandParts, ShellQuote(spec.Command.String()))
	for _, arg := range spec.Args {
		commandParts = append(commandParts, ShellQuote(arg))
	}

	command := strings.Join(commandParts, " ")
	if len(spec.Environment) > 0 {
		envParts := make([]string, 0, len(spec.Environment))
		for _, entry := range spec.Environment {
			envParts = append(envParts, ShellQuote(entry))
		}
		command = "env " + strings.Join(envParts, " ") + " " + command
	}
	if spec.WorkingDirectory != nil {
		command = "cd " + ShellQuote(spec.WorkingDirectory.String()) + " && " + command
	}

	return command
}

// ShellQuote escapes a raw argument for POSIX shell evaluation.
func ShellQuote(raw string) string {
	if raw == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(raw, "'", `'"'"'`) + "'"
}
