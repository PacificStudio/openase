package machinetransport

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"

	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var _ = logging.DeclareComponent("machine-transport-runtime-client-sessions")

type runtimeManagedClientSession struct {
	ctx            context.Context
	client         *runtimeProtocolClient
	transportClose func(error)

	sessionMu sync.Mutex
	sessionID string

	stdoutReader *io.PipeReader
	stdoutWriter *io.PipeWriter
	stderrReader *io.PipeReader
	stderrWriter *io.PipeWriter
	stdinWriter  *runtimeSessionInputWriter

	waitDone   chan struct{}
	waitErr    error
	waitMu     sync.Mutex
	finishOnce sync.Once

	closeOnce sync.Once
}

func newRuntimeManagedClientSession(
	ctx context.Context,
	client *runtimeProtocolClient,
	transportClose func(error),
) *runtimeManagedClientSession {
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	session := &runtimeManagedClientSession{
		ctx:            ctx,
		client:         client,
		transportClose: transportClose,
		stdoutReader:   stdoutReader,
		stdoutWriter:   stdoutWriter,
		stderrReader:   stderrReader,
		stderrWriter:   stderrWriter,
		waitDone:       make(chan struct{}),
	}
	session.stdinWriter = &runtimeSessionInputWriter{session: session}
	return session
}

func (s *runtimeManagedClientSession) setSessionID(sessionID string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.sessionID = strings.TrimSpace(sessionID)
}

func (s *runtimeManagedClientSession) currentSessionID() string {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	return s.sessionID
}

func (s *runtimeManagedClientSession) StdoutPipe() (io.Reader, error) {
	if s == nil {
		return nil, fmt.Errorf("runtime session unavailable")
	}
	return s.stdoutReader, nil
}

func (s *runtimeManagedClientSession) StderrPipe() (io.Reader, error) {
	if s == nil {
		return nil, fmt.Errorf("runtime session unavailable")
	}
	return s.stderrReader, nil
}

func (s *runtimeManagedClientSession) StdinPipe() (io.WriteCloser, error) {
	if s == nil {
		return nil, fmt.Errorf("runtime session unavailable")
	}
	return s.stdinWriter, nil
}

func (s *runtimeManagedClientSession) Wait() error {
	if s == nil {
		return fmt.Errorf("runtime session unavailable")
	}
	<-s.waitDone
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	return s.waitErr
}

func (s *runtimeManagedClientSession) Signal(signal string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("runtime session unavailable")
	}
	_, err := s.client.request(s.ctx, runtimecontract.OperationSessionSignal, runtimecontract.SessionSignalRequest{
		SessionID: s.currentSessionID(),
		Signal:    signal,
	})
	return err
}

func (s *runtimeManagedClientSession) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if s.client != nil && s.currentSessionID() != "" {
			_, _ = s.client.request(s.ctx, runtimecontract.OperationSessionClose, runtimecontract.SessionCloseRequest{
				SessionID: s.currentSessionID(),
			})
		}
		s.finish(io.EOF)
	})
	return nil
}

func (s *runtimeManagedClientSession) handleOutput(payload runtimecontract.SessionOutputEvent) error {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.DataBase64))
	if err != nil {
		s.finish(err)
		return err
	}
	switch payload.Stream {
	case "stdout":
		if len(data) > 0 {
			_, err = s.stdoutWriter.Write(data)
		}
	case "stderr":
		if len(data) > 0 {
			_, err = s.stderrWriter.Write(data)
		}
	default:
		err = fmt.Errorf("runtime output stream %q is not supported", payload.Stream)
	}
	if err != nil {
		s.finish(err)
		return err
	}
	return nil
}

func (s *runtimeManagedClientSession) handleExit(exitCode int) {
	if exitCode == 0 {
		s.finish(nil)
		return
	}
	s.finish(ProcessExitError{code: exitCode})
}

func (s *runtimeManagedClientSession) finish(err error) {
	s.finishOnce.Do(func() {
		s.waitMu.Lock()
		s.waitErr = err
		s.waitMu.Unlock()
		_ = s.stdoutWriter.Close()
		_ = s.stderrWriter.Close()
		close(s.waitDone)
		if s.transportClose != nil {
			s.transportClose(err)
		}
	})
}

type runtimeSessionInputWriter struct {
	session   *runtimeManagedClientSession
	closeOnce sync.Once
}

func (w *runtimeSessionInputWriter) Write(p []byte) (int, error) {
	if w == nil || w.session == nil || w.session.client == nil {
		return 0, fmt.Errorf("runtime stdin is unavailable")
	}
	if len(p) == 0 {
		return 0, nil
	}
	_, err := w.session.client.request(w.session.ctx, runtimecontract.OperationSessionInput, runtimecontract.SessionInputRequest{
		SessionID:  w.session.currentSessionID(),
		DataBase64: base64.StdEncoding.EncodeToString(p),
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *runtimeSessionInputWriter) Close() error {
	if w == nil || w.session == nil || w.session.client == nil {
		return nil
	}
	var err error
	w.closeOnce.Do(func() {
		_, err = w.session.client.request(w.session.ctx, runtimecontract.OperationSessionInput, runtimecontract.SessionInputRequest{
			SessionID:  w.session.currentSessionID(),
			CloseStdin: true,
		})
	})
	return err
}

type runtimeCommandSession struct {
	remote *runtimeManagedClientSession
}

func newRuntimeCommandSession(ctx context.Context, client *runtimeProtocolClient, transportClose func(error)) *runtimeCommandSession {
	return &runtimeCommandSession{
		remote: newRuntimeManagedClientSession(ctx, client, transportClose),
	}
}

func (s *runtimeCommandSession) CombinedOutput(cmd string) ([]byte, error) {
	stdout, err := s.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := s.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := s.Start(cmd); err != nil {
		return nil, err
	}

	var combinedMu sync.Mutex
	combined := make([]byte, 0, 1024)
	readInto := func(reader io.Reader, done chan<- struct{}) {
		defer close(done)
		buffer := make([]byte, 32*1024)
		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				combinedMu.Lock()
				combined = append(combined, buffer[:n]...)
				combinedMu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}

	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	go readInto(stdout, stdoutDone)
	go readInto(stderr, stderrDone)

	waitErr := s.Wait()
	<-stdoutDone
	<-stderrDone

	combinedMu.Lock()
	output := append([]byte(nil), combined...)
	combinedMu.Unlock()
	return output, waitErr
}

func (s *runtimeCommandSession) StdinPipe() (io.WriteCloser, error) { return s.remote.StdinPipe() }
func (s *runtimeCommandSession) StdoutPipe() (io.Reader, error)     { return s.remote.StdoutPipe() }
func (s *runtimeCommandSession) StderrPipe() (io.Reader, error)     { return s.remote.StderrPipe() }

func (s *runtimeCommandSession) Start(cmd string) error {
	if s == nil || s.remote == nil || s.remote.client == nil {
		return fmt.Errorf("runtime command session unavailable")
	}
	envelope, err := s.remote.client.request(s.remote.ctx, runtimecontract.OperationCommandOpen, runtimecontract.CommandOpenRequest{
		Command: cmd,
	})
	if err != nil {
		return err
	}
	payload, err := runtimecontract.DecodePayload[runtimecontract.SessionResponse](envelope)
	if err != nil {
		return err
	}
	s.remote.setSessionID(payload.SessionID)
	s.remote.client.registerSession(payload.SessionID, s.remote)
	return nil
}

func (s *runtimeCommandSession) Signal(signal string) error { return s.remote.Signal(signal) }
func (s *runtimeCommandSession) Wait() error                { return s.remote.Wait() }
func (s *runtimeCommandSession) Close() error               { return s.remote.Close() }

func startRuntimeRemoteProcess(
	ctx context.Context,
	client *runtimeProtocolClient,
	spec provider.AgentCLIProcessSpec,
	transportClose func(error),
) (provider.AgentCLIProcess, error) {
	session := newRuntimeManagedClientSession(ctx, client, transportClose)
	envelope, err := client.request(ctx, runtimecontract.OperationProcessStart, runtimeProcessRequestFromSpec(spec))
	if err != nil {
		session.finish(err)
		return nil, err
	}
	payload, err := runtimecontract.DecodePayload[runtimecontract.SessionResponse](envelope)
	if err != nil {
		session.finish(err)
		return nil, err
	}
	session.setSessionID(payload.SessionID)
	client.registerSession(payload.SessionID, session)

	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, err
	}
	return &runtimeManagedProcess{
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

type runtimeManagedProcess struct {
	session *runtimeManagedClientSession
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
}

func (p *runtimeManagedProcess) PID() int              { return 0 }
func (p *runtimeManagedProcess) Stdin() io.WriteCloser { return p.stdin }
func (p *runtimeManagedProcess) Stdout() io.ReadCloser { return io.NopCloser(p.stdout) }
func (p *runtimeManagedProcess) Stderr() io.ReadCloser { return io.NopCloser(p.stderr) }
func (p *runtimeManagedProcess) Wait() error           { return p.session.Wait() }

func (p *runtimeManagedProcess) Stop(ctx context.Context) error {
	if p == nil || p.session == nil {
		return fmt.Errorf("process must not be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}
	_ = p.stdin.Close()
	if err := p.session.Signal("INT"); err != nil {
		_ = p.session.Close()
	}
	select {
	case <-p.session.waitDone:
		return p.session.Wait()
	case <-ctx.Done():
		_ = p.session.Close()
		return p.session.Wait()
	}
}
