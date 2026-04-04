package agentcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

const defaultStopGracePeriod = 5 * time.Second

var agentCLIProcessComponent = logging.DeclareComponent("agentcli-process")

type ManagerOptions struct {
	StopGracePeriod time.Duration
}

type Manager struct {
	stopGracePeriod time.Duration
	logger          *slog.Logger
}

func NewManager(options ManagerOptions) provider.AgentCLIProcessManager {
	stopGracePeriod := options.StopGracePeriod
	if stopGracePeriod <= 0 {
		stopGracePeriod = defaultStopGracePeriod
	}

	return &Manager{
		stopGracePeriod: stopGracePeriod,
		logger:          logging.WithComponent(nil, agentCLIProcessComponent),
	}
}

func (m *Manager) Start(ctx context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context must not be nil")
	}
	if spec.Command == "" {
		return nil, fmt.Errorf("agent cli command must not be empty")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	//nolint:gosec // command and arguments come from validated agent provider configuration
	cmd := exec.Command(spec.Command.String(), spec.Args...)
	configureProcessGroup(cmd)
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
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		return nil, fmt.Errorf("open stderr pipe: %w", err)
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	stdoutBuffer := newProcessOutputBuffer()
	stderrBuffer := newProcessOutputBuffer()
	stdoutReady := startOutputPump(stdoutReader, stdoutBuffer)
	stderrReady := startOutputPump(stderrReader, stderrBuffer)
	<-stdoutReady
	<-stderrReady

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		_ = stderrReader.Close()
		_ = stderrWriter.Close()
		_ = stdoutBuffer.Close()
		_ = stderrBuffer.Close()
		m.logger.Error("start local agent cli process failed", "command", spec.Command.String(), "args", spec.Args, "working_directory", safeWorkingDirectory(spec.WorkingDirectory), "error", err)
		return nil, fmt.Errorf("start agent cli process: %w", err)
	}
	if err := ctx.Err(); err != nil {
		_ = interruptProcess(cmd.Process)
		_ = cmd.Wait()
		_ = stdin.Close()
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		_ = stderrReader.Close()
		_ = stderrWriter.Close()
		_ = stdoutBuffer.Close()
		_ = stderrBuffer.Close()
		return nil, err
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()

	process := &runningProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdoutBuffer,
		stderr: stderrBuffer,
		done:   make(chan struct{}),
	}
	m.logger.Info("started local agent cli process", "command", spec.Command.String(), "args", spec.Args, "working_directory", safeWorkingDirectory(spec.WorkingDirectory), "pid", cmd.Process.Pid)

	return process, nil
}

type runningProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	done     chan struct{}
	waitOnce sync.Once

	waitMu  sync.Mutex
	waitErr error
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

	p.startWait()
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

func safeWorkingDirectory(path *provider.AbsolutePath) string {
	if path == nil {
		return ""
	}
	return path.String()
}

type processOutputBuffer struct {
	mu       sync.Mutex
	ready    *sync.Cond
	buffer   bytes.Buffer
	closed   bool
	closeErr error
}

func newProcessOutputBuffer() *processOutputBuffer {
	output := &processOutputBuffer{}
	output.ready = sync.NewCond(&output.mu)
	return output
}

func (b *processOutputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, io.ErrClosedPipe
	}

	n, err := b.buffer.Write(p)
	b.ready.Broadcast()
	return n, err
}

func (b *processOutputBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for b.buffer.Len() == 0 && !b.closed {
		b.ready.Wait()
	}
	if b.buffer.Len() == 0 && b.closed {
		if b.closeErr != nil {
			return 0, b.closeErr
		}
		return 0, io.EOF
	}

	return b.buffer.Read(p)
}

// Close marks the buffered output stream as complete for downstream readers.
func (b *processOutputBuffer) Close() error {
	return b.closeWithError(nil)
}

func (b *processOutputBuffer) closeWithError(err error) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	b.closeErr = err
	b.ready.Broadcast()
	return nil
}

func startOutputPump(source io.ReadCloser, target *processOutputBuffer) <-chan struct{} {
	ready := make(chan struct{})

	go func() {
		var buffer [4096]byte
		signaledReady := false

		for {
			if !signaledReady {
				close(ready)
				signaledReady = true
			}

			count, err := source.Read(buffer[:])
			if count > 0 {
				if _, writeErr := target.Write(buffer[:count]); writeErr != nil {
					err = writeErr
				}
			}
			if err == nil {
				continue
			}

			_ = source.Close()
			if isProcessPipeClosedError(err) || errors.Is(err, io.EOF) {
				err = nil
			}
			_ = target.closeWithError(err)
			return
		}
	}()

	return ready
}

func isProcessPipeClosedError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrClosed) {
		return true
	}

	return strings.Contains(err.Error(), "file already closed")
}
