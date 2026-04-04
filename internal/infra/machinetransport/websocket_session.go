package machinetransport

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/gorilla/websocket"
)

const websocketWriteTimeout = 10 * time.Second

var websocketTransportComponent = logging.DeclareComponent("machine-transport-websocket")
var websocketTransportLogger = logging.WithComponent(nil, websocketTransportComponent)

type ProcessExitError struct {
	code int
}

func (e ProcessExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func (e ProcessExitError) ExitStatus() int {
	return e.code
}

type websocketFrame struct {
	Type     string `json:"type"`
	Command  string `json:"command,omitempty"`
	Data     string `json:"data,omitempty"`
	Signal   string `json:"signal,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

type websocketCommandSession struct {
	conn *websocket.Conn

	writeMu sync.Mutex

	stdoutReader *io.PipeReader
	stdoutWriter *io.PipeWriter
	stderrReader *io.PipeReader
	stderrWriter *io.PipeWriter
	stdinWriter  *websocketSessionStdinWriter

	waitDone   chan struct{}
	waitErr    error
	waitMu     sync.Mutex
	finishOnce sync.Once

	started bool
	startMu sync.Mutex

	closeOnce sync.Once
}

type websocketSessionStdinWriter struct {
	session   *websocketCommandSession
	closeOnce sync.Once
}

func dialWebsocketCommandSession(ctx context.Context, machine domain.Machine) (CommandSession, error) {
	endpoint := strings.TrimSpace(pointerString(machine.AdvertisedEndpoint))
	if endpoint == "" {
		return nil, fmt.Errorf("listener websocket endpoint is not configured for machine %s", machine.Name)
	}

	header := http.Header{}
	switch machine.ChannelCredential.Kind {
	case domain.MachineChannelCredentialKindNone, "":
	case domain.MachineChannelCredentialKindToken:
		token := strings.TrimSpace(pointerString(machine.ChannelCredential.TokenID))
		if token == "" {
			return nil, fmt.Errorf("listener websocket token is not configured for machine %s", machine.Name)
		}
		header.Set("Authorization", "Bearer "+token)
	case domain.MachineChannelCredentialKindCertificate:
		return nil, fmt.Errorf("listener websocket certificate credentials are not supported yet for machine %s", machine.Name)
	default:
		return nil, fmt.Errorf("listener websocket credential kind %q is not supported", machine.ChannelCredential.Kind)
	}

	conn, response, err := (&websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}).DialContext(ctx, endpoint, header)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		classifiedErr := classifyWebsocketDialError(machine, endpoint, response, err)
		websocketTransportLogger.Warn("dial listener websocket failed", "machine_id", machine.ID.String(), "machine_name", machine.Name, "endpoint", endpoint, "error", classifiedErr)
		return nil, classifiedErr
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	websocketTransportLogger.Debug("dialed listener websocket", "machine_id", machine.ID.String(), "machine_name", machine.Name, "endpoint", endpoint)
	return newWebsocketCommandSession(conn), nil
}

func newWebsocketCommandSession(conn *websocket.Conn) *websocketCommandSession {
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	session := &websocketCommandSession{
		conn:         conn,
		stdoutReader: stdoutReader,
		stdoutWriter: stdoutWriter,
		stderrReader: stderrReader,
		stderrWriter: stderrWriter,
		waitDone:     make(chan struct{}),
	}
	session.stdinWriter = &websocketSessionStdinWriter{session: session}
	go session.readLoop()
	return session
}

func (s *websocketCommandSession) CombinedOutput(cmd string) ([]byte, error) {
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

func (s *websocketCommandSession) StdinPipe() (io.WriteCloser, error) {
	if s == nil {
		return nil, fmt.Errorf("websocket command session unavailable")
	}
	return s.stdinWriter, nil
}

func (s *websocketCommandSession) StdoutPipe() (io.Reader, error) {
	if s == nil {
		return nil, fmt.Errorf("websocket command session unavailable")
	}
	return s.stdoutReader, nil
}

func (s *websocketCommandSession) StderrPipe() (io.Reader, error) {
	if s == nil {
		return nil, fmt.Errorf("websocket command session unavailable")
	}
	return s.stderrReader, nil
}

func (s *websocketCommandSession) Start(cmd string) error {
	if s == nil {
		return fmt.Errorf("websocket command session unavailable")
	}

	s.startMu.Lock()
	defer s.startMu.Unlock()
	if s.started {
		return fmt.Errorf("websocket command session already started")
	}
	s.started = true

	if err := s.writeFrame(websocketFrame{Type: "start", Command: cmd}); err != nil {
		s.finish(err)
		return err
	}
	return nil
}

func (s *websocketCommandSession) Signal(signal string) error {
	if s == nil {
		return fmt.Errorf("websocket command session unavailable")
	}
	return s.writeFrame(websocketFrame{Type: "signal", Signal: strings.TrimSpace(signal)})
}

func (s *websocketCommandSession) Wait() error {
	if s == nil {
		return fmt.Errorf("websocket command session unavailable")
	}
	<-s.waitDone
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	return s.waitErr
}

func (s *websocketCommandSession) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		_ = s.writeFrame(websocketFrame{Type: "close"})
		s.finish(errors.New("websocket command session closed"))
		_ = s.conn.Close()
	})
	return nil
}

func (s *websocketCommandSession) readLoop() {
	for {
		var frame websocketFrame
		if err := s.conn.ReadJSON(&frame); err != nil {
			s.finish(fmt.Errorf("websocket command session closed: %w", err))
			return
		}

		switch frame.Type {
		case "stdout":
			data, err := decodeWebsocketFrameData(frame.Data)
			if err != nil {
				s.finish(err)
				return
			}
			if len(data) > 0 {
				if _, err := s.stdoutWriter.Write(data); err != nil {
					s.finish(err)
					return
				}
			}
		case "stderr":
			data, err := decodeWebsocketFrameData(frame.Data)
			if err != nil {
				s.finish(err)
				return
			}
			if len(data) > 0 {
				if _, err := s.stderrWriter.Write(data); err != nil {
					s.finish(err)
					return
				}
			}
		case "error":
			s.finish(errors.New(strings.TrimSpace(frame.Error)))
			return
		case "exit":
			if frame.ExitCode == 0 {
				s.finish(nil)
			} else {
				s.finish(ProcessExitError{code: frame.ExitCode})
			}
			return
		}
	}
}

func (s *websocketCommandSession) writeFrame(frame websocketFrame) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.conn == nil {
		return fmt.Errorf("websocket connection unavailable")
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
	return s.conn.WriteJSON(frame)
}

func (s *websocketCommandSession) finish(err error) {
	s.finishOnce.Do(func() {
		s.waitMu.Lock()
		s.waitErr = err
		s.waitMu.Unlock()
		_ = s.stdoutWriter.Close()
		_ = s.stderrWriter.Close()
		close(s.waitDone)
	})
}

func (w *websocketSessionStdinWriter) Write(p []byte) (int, error) {
	if w == nil || w.session == nil {
		return 0, fmt.Errorf("websocket stdin is unavailable")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if err := w.session.writeFrame(websocketFrame{
		Type: "stdin",
		Data: base64.StdEncoding.EncodeToString(p),
	}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *websocketSessionStdinWriter) Close() error {
	if w == nil || w.session == nil {
		return nil
	}
	var err error
	w.closeOnce.Do(func() {
		err = w.session.writeFrame(websocketFrame{Type: "stdin_close"})
	})
	return err
}

type ListenerHandlerOptions struct {
	BearerToken string
}

func NewWebsocketListenerHandler(options ListenerHandlerOptions) http.Handler {
	return &websocketListenerHandler{
		bearerToken: strings.TrimSpace(options.BearerToken),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

type websocketListenerHandler struct {
	bearerToken string
	upgrader    websocket.Upgrader
}

func (h *websocketListenerHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if h == nil {
		http.Error(writer, "listener handler unavailable", http.StatusInternalServerError)
		return
	}
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.bearerToken != "" {
		token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
		if token != h.bearerToken {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	conn, err := h.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	runWebsocketListenerSession(request.Context(), conn)
}

func runWebsocketListenerSession(parent context.Context, conn *websocket.Conn) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	session := &websocketListenerSession{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}
	session.run()
}

type websocketListenerSession struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *websocket.Conn

	writeMu sync.Mutex
	streams sync.WaitGroup

	command *exec.Cmd
	stdin   io.WriteCloser
}

func (s *websocketListenerSession) run() {
	for {
		var frame websocketFrame
		if err := s.conn.ReadJSON(&frame); err != nil {
			s.stopProcess()
			return
		}

		switch frame.Type {
		case "start":
			if err := s.startCommand(strings.TrimSpace(frame.Command)); err != nil {
				_ = s.writeFrame(websocketFrame{Type: "error", Error: err.Error()})
				_ = s.writeFrame(websocketFrame{Type: "exit", ExitCode: 127})
				s.stopProcess()
				return
			}
		case "stdin":
			data, err := decodeWebsocketFrameData(frame.Data)
			if err != nil {
				_ = s.writeFrame(websocketFrame{Type: "error", Error: err.Error()})
				s.stopProcess()
				return
			}
			if len(data) > 0 {
				if _, err := s.stdin.Write(data); err != nil {
					_ = s.writeFrame(websocketFrame{Type: "error", Error: fmt.Sprintf("write stdin: %v", err)})
					s.stopProcess()
					return
				}
			}
		case "stdin_close":
			if s.stdin != nil {
				_ = s.stdin.Close()
			}
		case "signal":
			if err := s.signal(frame.Signal); err != nil {
				_ = s.writeFrame(websocketFrame{Type: "error", Error: err.Error()})
				s.stopProcess()
				return
			}
		case "close":
			s.stopProcess()
			return
		}
	}
}

func (s *websocketListenerSession) startCommand(command string) error {
	if command == "" {
		return fmt.Errorf("remote command must not be empty")
	}
	if s.command != nil {
		return fmt.Errorf("remote command session already started")
	}

	shell, args := listenerShellCommand(command)
	cmd := exec.CommandContext(s.ctx, shell, args...) // #nosec G204 -- machine websocket listener intentionally runs orchestrator-provided shell commands.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open command stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open command stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open command stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start remote command: %w", err)
	}

	s.command = cmd
	s.stdin = stdin
	s.streams.Add(2)
	go s.streamPipe("stdout", stdout)
	go s.streamPipe("stderr", stderr)
	go s.waitForExit()
	return nil
}

func (s *websocketListenerSession) waitForExit() {
	exitCode := 0
	if err := s.command.Wait(); err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			exitCode = exitErr.ExitCode()
		case errors.Is(err, context.Canceled):
			exitCode = 130
		default:
			_ = s.writeFrame(websocketFrame{Type: "error", Error: fmt.Sprintf("wait remote command: %v", err)})
			exitCode = 1
		}
	}
	s.streams.Wait()
	_ = s.writeFrame(websocketFrame{Type: "exit", ExitCode: exitCode})
	s.stopProcess()
}

func (s *websocketListenerSession) streamPipe(streamType string, reader io.Reader) {
	defer s.streams.Done()
	buffer := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			encoded := base64.StdEncoding.EncodeToString(buffer[:n])
			if writeErr := s.writeFrame(websocketFrame{Type: streamType, Data: encoded}); writeErr != nil {
				s.stopProcess()
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (s *websocketListenerSession) signal(raw string) error {
	if s.command == nil || s.command.Process == nil {
		return fmt.Errorf("remote command is not running")
	}

	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "INT":
		return s.command.Process.Signal(os.Interrupt)
	case "KILL":
		return s.command.Process.Signal(os.Kill)
	case "TERM":
		return s.command.Process.Signal(syscall.SIGTERM)
	default:
		return fmt.Errorf("unsupported remote signal %q", raw)
	}
}

func (s *websocketListenerSession) stopProcess() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.command != nil && s.command.Process != nil {
		_ = s.command.Process.Kill()
	}
}

func (s *websocketListenerSession) writeFrame(frame websocketFrame) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
	return s.conn.WriteJSON(frame)
}

func decodeWebsocketFrameData(encoded string) ([]byte, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("decode websocket frame payload: %w", err)
	}
	return data, nil
}

func listenerShellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-lc", command}
}

func classifyWebsocketDialError(
	machine domain.Machine,
	endpoint string,
	response *http.Response,
	err error,
) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("listener websocket authentication failed for machine %s at %s", machine.Name, endpoint)
		default:
			return fmt.Errorf("listener websocket handshake failed for machine %s at %s: %s", machine.Name, endpoint, response.Status)
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("listener websocket DNS resolution failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return fmt.Errorf("listener websocket host verification failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var authorityErr x509.UnknownAuthorityError
	if errors.As(err, &authorityErr) {
		return fmt.Errorf("listener websocket TLS verification failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var opErr *net.OpError
	switch {
	case errors.As(err, &opErr):
		return fmt.Errorf("listener websocket endpoint unreachable for machine %s at %s: %w", machine.Name, endpoint, err)
	default:
		return fmt.Errorf("listener websocket dial failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
