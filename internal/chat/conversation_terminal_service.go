package chat

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	"github.com/creack/pty"
	"github.com/google/uuid"
)

const conversationTerminalPendingOutputLimitBytes = 256 * 1024

var (
	ErrConversationTerminalUnsupported     = errors.New("project conversation terminal is unsupported")
	ErrConversationTerminalSessionNotFound = errors.New("project conversation terminal session not found")
	ErrConversationTerminalAttachForbidden = errors.New("project conversation terminal attach forbidden")
	ErrConversationTerminalAlreadyAttached = errors.New("project conversation terminal session is already attached")
)

type ConversationTerminalService struct {
	logger        *slog.Logger
	conversations *ProjectConversationService
	registry      *conversationTerminalRegistry
	now           func() time.Time
	launch        func(context.Context, conversationTerminalLaunchSpec) (conversationTerminalProcess, error)
}

type ConversationTerminalSession struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	UserID         UserID
	Mode           chatdomain.TerminalMode
	CWD            string
	WSPath         string
	AttachToken    string
	CreatedAt      time.Time
	LastAttachedAt *time.Time
}

type ConversationTerminalEvent struct {
	Type     string
	Data     []byte
	ExitCode int
	Signal   string
	Message  string
}

type AttachedConversationTerminal struct {
	Session ConversationTerminalSession
	Events  <-chan ConversationTerminalEvent
	session *conversationTerminalManagedSession
	client  *conversationTerminalAttachedClient
}

func (a AttachedConversationTerminal) WriteInput(data []byte) error {
	if a.session == nil {
		return ErrConversationTerminalSessionNotFound
	}
	return a.session.writeInput(data)
}

func (a AttachedConversationTerminal) Resize(cols int, rows int) error {
	if a.session == nil {
		return ErrConversationTerminalSessionNotFound
	}
	return a.session.resize(cols, rows)
}

func (a AttachedConversationTerminal) Close() error {
	if a.session == nil {
		return ErrConversationTerminalSessionNotFound
	}
	return a.session.close()
}

func (a AttachedConversationTerminal) Detach() error {
	if a.session == nil {
		return ErrConversationTerminalSessionNotFound
	}
	return a.session.detach(a.client)
}

func NewConversationTerminalService(
	logger *slog.Logger,
	conversations *ProjectConversationService,
) *ConversationTerminalService {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &ConversationTerminalService{
		logger:        logger.With("component", "conversation-terminal-service"),
		conversations: conversations,
		registry:      newConversationTerminalRegistry(),
		now:           func() time.Time { return time.Now().UTC() },
		launch:        startLocalConversationTerminalProcess,
	}
}

func (s *ConversationTerminalService) CreateSession(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input chatdomain.OpenTerminalSessionInput,
) (ConversationTerminalSession, error) {
	if s == nil || s.conversations == nil {
		return ConversationTerminalSession{}, fmt.Errorf("conversation terminal service unavailable")
	}
	machine, cwd, err := s.resolveConversationTerminalTarget(ctx, userID, conversationID, input)
	if err != nil {
		return ConversationTerminalSession{}, err
	}
	attachToken, err := newConversationTerminalAttachToken()
	if err != nil {
		return ConversationTerminalSession{}, err
	}
	createdAt := s.now()
	sessionCtx, cancel := context.WithCancel(context.Background())
	process, err := s.startSessionProcess(sessionCtx, machine, conversationTerminalLaunchSpec{
		CWD:         cwd,
		Cols:        input.Cols,
		Rows:        input.Rows,
		Environment: buildConversationTerminalEnvironment(),
	})
	if err != nil {
		cancel()
		return ConversationTerminalSession{}, err
	}

	session := &conversationTerminalManagedSession{
		service: s,
		meta: ConversationTerminalSession{
			ID:             uuid.New(),
			ConversationID: conversationID,
			UserID:         userID,
			Mode:           input.Mode,
			CWD:            cwd,
			WSPath:         fmt.Sprintf("/api/v1/chat/conversations/%s/terminal-sessions/%s/attach", conversationID.String(), uuid.Nil.String()),
			AttachToken:    attachToken,
			CreatedAt:      createdAt,
		},
		process: process,
		cancel:  cancel,
	}
	session.meta.WSPath = fmt.Sprintf("/api/v1/chat/conversations/%s/terminal-sessions/%s/attach", conversationID.String(), session.meta.ID.String())
	s.registry.put(session)
	session.start()
	return session.snapshot(), nil
}

func (s *ConversationTerminalService) AttachSession(
	userID UserID,
	conversationID uuid.UUID,
	sessionID uuid.UUID,
	attachToken string,
) (AttachedConversationTerminal, error) {
	session := s.registry.get(conversationID, sessionID)
	if session == nil {
		return AttachedConversationTerminal{}, ErrConversationTerminalSessionNotFound
	}
	return session.attach(userID, attachToken, s.now())
}

func (s *ConversationTerminalService) CloseSession(conversationID uuid.UUID, sessionID uuid.UUID) error {
	session := s.registry.get(conversationID, sessionID)
	if session == nil {
		return ErrConversationTerminalSessionNotFound
	}
	return session.close()
}

func (s *ConversationTerminalService) startSessionProcess(
	ctx context.Context,
	machine catalogdomain.Machine,
	spec conversationTerminalLaunchSpec,
) (conversationTerminalProcess, error) {
	if machine.Host == catalogdomain.LocalMachineHost {
		process, err := s.launch(ctx, spec)
		if err != nil {
			return nil, fmt.Errorf("start local terminal session: %w", err)
		}
		return process, nil
	}
	process, err := s.startRemoteProcess(ctx, machine, spec)
	if err != nil {
		return nil, fmt.Errorf("start remote terminal session: %w", err)
	}
	return process, nil
}

func (s *ConversationTerminalService) resolveConversationTerminalTarget(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input chatdomain.OpenTerminalSessionInput,
) (catalogdomain.Machine, string, error) {
	if s.conversations == nil {
		return catalogdomain.Machine{}, "", fmt.Errorf("project conversation service unavailable")
	}
	if input.RepoPath != nil {
		resolved, relativePath, err := s.conversations.resolveConversationWorkspaceRepoPath(
			ctx,
			userID,
			conversationID,
			*input.RepoPath,
			valueOrEmpty(input.CWDPath),
			true,
		)
		if err != nil {
			return catalogdomain.Machine{}, "", err
		}
		cwd := resolved.repo.repoPath
		if resolved.machine.Host == catalogdomain.LocalMachineHost {
			cwd, err = resolveLocalProjectConversationWorkspaceDirectory(resolved.repo.repoPath, relativePath)
			if err != nil {
				return catalogdomain.Machine{}, "", err
			}
		} else if relativePath != "" {
			cwd = filepath.Join(resolved.repo.repoPath, filepath.FromSlash(relativePath))
		}
		return resolved.machine, cwd, nil
	}

	_, location, err := s.conversations.resolveConversationWorkspace(ctx, userID, conversationID)
	if err != nil {
		return catalogdomain.Machine{}, "", err
	}
	relativePath, err := parseProjectConversationWorkspaceRelativePath(valueOrEmpty(input.CWDPath), true)
	if err != nil {
		return catalogdomain.Machine{}, "", err
	}
	cwd := location.workspacePath
	if location.machine.Host == catalogdomain.LocalMachineHost {
		cwd, err = resolveLocalProjectConversationWorkspaceDirectory(location.workspacePath, relativePath)
		if err != nil {
			return catalogdomain.Machine{}, "", err
		}
	} else if relativePath != "" {
		cwd = filepath.Join(location.workspacePath, filepath.FromSlash(relativePath))
	}
	return location.machine, cwd, nil
}

type conversationTerminalRegistry struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]*conversationTerminalManagedSession
}

func newConversationTerminalRegistry() *conversationTerminalRegistry {
	return &conversationTerminalRegistry{sessions: make(map[uuid.UUID]*conversationTerminalManagedSession)}
}

func (r *conversationTerminalRegistry) put(session *conversationTerminalManagedSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.meta.ID] = session
}

func (r *conversationTerminalRegistry) get(conversationID uuid.UUID, sessionID uuid.UUID) *conversationTerminalManagedSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	if session == nil || session.meta.ConversationID != conversationID {
		return nil
	}
	return session
}

func (r *conversationTerminalRegistry) remove(sessionID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
}

type conversationTerminalManagedSession struct {
	service *ConversationTerminalService
	meta    ConversationTerminalSession
	process conversationTerminalProcess
	cancel  context.CancelFunc

	closeOnce    sync.Once
	finalizeOnce sync.Once

	mu                 sync.Mutex
	client             *conversationTerminalAttachedClient
	clientReady        bool
	pendingOutput      [][]byte
	pendingOutputBytes int
	closing            bool
}

type conversationTerminalAttachedClient struct {
	events chan ConversationTerminalEvent
	done   chan struct{}
	once   sync.Once
}

func newConversationTerminalAttachedClient(bufferSize int) *conversationTerminalAttachedClient {
	if bufferSize < 64 {
		bufferSize = 64
	}
	return &conversationTerminalAttachedClient{
		events: make(chan ConversationTerminalEvent, bufferSize),
		done:   make(chan struct{}),
	}
}

func (c *conversationTerminalAttachedClient) detach() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		close(c.done)
	})
}

func (s *conversationTerminalManagedSession) snapshot() ConversationTerminalSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.meta
}

func (s *conversationTerminalManagedSession) start() {
	go s.readLoop()
	go s.waitLoop()
}

func (s *conversationTerminalManagedSession) attach(
	userID UserID,
	attachToken string,
	attachedAt time.Time,
) (AttachedConversationTerminal, error) {
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(attachToken)), []byte(s.meta.AttachToken)) != 1 || s.meta.UserID != userID {
		return AttachedConversationTerminal{}, ErrConversationTerminalAttachForbidden
	}

	s.mu.Lock()
	if s.client != nil {
		s.mu.Unlock()
		return AttachedConversationTerminal{}, ErrConversationTerminalAlreadyAttached
	}
	pendingOutput := append([][]byte(nil), s.pendingOutput...)
	s.pendingOutput = nil
	s.pendingOutputBytes = 0
	client := newConversationTerminalAttachedClient(len(pendingOutput) + 1)
	s.client = client
	s.clientReady = false
	s.meta.LastAttachedAt = &attachedAt
	meta := s.meta
	s.mu.Unlock()

	go s.flushPendingOutput(client, pendingOutput)
	return AttachedConversationTerminal{
		Session: meta,
		Events:  client.events,
		session: s,
		client:  client,
	}, nil
}

func (s *conversationTerminalManagedSession) writeInput(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if _, err := s.process.Write(data); err != nil {
		return fmt.Errorf("write terminal input: %w", err)
	}
	return nil
}

func (s *conversationTerminalManagedSession) resize(cols int, rows int) error {
	if cols <= 0 || rows <= 0 {
		return fmt.Errorf("terminal resize must use positive cols and rows")
	}
	if err := s.process.Resize(cols, rows); err != nil {
		return fmt.Errorf("resize terminal: %w", err)
	}
	return nil
}

func (s *conversationTerminalManagedSession) close() error {
	s.mu.Lock()
	s.closing = true
	s.mu.Unlock()
	s.closeOnce.Do(func() {
		s.cancel()
		_ = s.process.Close()
	})
	return nil
}

func (s *conversationTerminalManagedSession) detach(client *conversationTerminalAttachedClient) error {
	if client == nil {
		return nil
	}
	s.mu.Lock()
	if s.client == client {
		s.client = nil
	}
	s.mu.Unlock()
	client.detach()
	return nil
}

func (s *conversationTerminalManagedSession) readLoop() {
	buffer := make([]byte, 4096)
	for {
		count, err := s.process.Read(buffer)
		if count > 0 {
			s.emitOutput(buffer[:count])
		}
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) || s.isClosing() {
			return
		}
		s.emitError(fmt.Sprintf("read terminal output: %v", err))
		return
	}
}

func (s *conversationTerminalManagedSession) waitLoop() {
	err := s.process.Wait()
	exit := conversationTerminalExitFromError(err)
	s.finalize(ConversationTerminalEvent{Type: "exit", ExitCode: exit.Code, Signal: exit.Signal})
}

func (s *conversationTerminalManagedSession) emitOutput(chunk []byte) {
	copied := append([]byte(nil), chunk...)
	s.mu.Lock()
	client := s.client
	if client == nil || !s.clientReady {
		s.queuePendingOutputLocked(copied)
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	if s.sendEvent(client, ConversationTerminalEvent{Type: "output", Data: copied}) {
		return
	}
	s.mu.Lock()
	if s.client == nil {
		s.queuePendingOutputLocked(copied)
	}
	s.mu.Unlock()
}

func (s *conversationTerminalManagedSession) emitError(message string) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	_ = s.sendEvent(client, ConversationTerminalEvent{Type: "error", Message: message})
}

func (s *conversationTerminalManagedSession) finalize(event ConversationTerminalEvent) {
	s.finalizeOnce.Do(func() {
		_ = s.close()
		s.mu.Lock()
		client := s.client
		s.client = nil
		s.mu.Unlock()
		if client != nil {
			_ = s.sendEvent(client, event)
			client.detach()
		}
		s.service.registry.remove(s.meta.ID)
	})
}

func (s *conversationTerminalManagedSession) flushPendingOutput(
	client *conversationTerminalAttachedClient,
	pendingOutput [][]byte,
) {
	if !s.sendEvent(client, ConversationTerminalEvent{Type: "ready"}) {
		return
	}
	queued := pendingOutput
	for {
		for _, chunk := range queued {
			if !s.sendEvent(client, ConversationTerminalEvent{Type: "output", Data: append([]byte(nil), chunk...)}) {
				return
			}
		}
		s.mu.Lock()
		if s.client != client {
			s.mu.Unlock()
			return
		}
		if len(s.pendingOutput) == 0 {
			s.clientReady = true
			s.mu.Unlock()
			return
		}
		queued = append([][]byte(nil), s.pendingOutput...)
		s.pendingOutput = nil
		s.pendingOutputBytes = 0
		s.mu.Unlock()
	}
}

func (s *conversationTerminalManagedSession) sendEvent(
	client *conversationTerminalAttachedClient,
	event ConversationTerminalEvent,
) bool {
	if client == nil {
		return false
	}
	select {
	case <-client.done:
		return false
	case client.events <- event:
		return true
	}
}

func (s *conversationTerminalManagedSession) queuePendingOutputLocked(chunk []byte) {
	s.pendingOutput = append(s.pendingOutput, chunk)
	s.pendingOutputBytes += len(chunk)
	for s.pendingOutputBytes > conversationTerminalPendingOutputLimitBytes && len(s.pendingOutput) > 0 {
		s.pendingOutputBytes -= len(s.pendingOutput[0])
		s.pendingOutput = s.pendingOutput[1:]
	}
}

func (s *conversationTerminalManagedSession) isClosing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closing
}

type conversationTerminalExit struct {
	Code   int
	Signal string
}

func conversationTerminalExitFromError(err error) conversationTerminalExit {
	if err == nil {
		return conversationTerminalExit{Code: 0}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exit := conversationTerminalExit{Code: exitErr.ExitCode()}
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			exit.Signal = status.Signal().String()
		}
		return exit
	}
	var runtimeExit machinetransport.ProcessExitError
	if errors.As(err, &runtimeExit) {
		return conversationTerminalExit{Code: runtimeExit.ExitStatus()}
	}
	if errors.Is(err, context.Canceled) {
		return conversationTerminalExit{Code: 0}
	}
	return conversationTerminalExit{Code: 1}
}

type conversationTerminalLaunchSpec struct {
	CWD         string
	Cols        int
	Rows        int
	Environment []string
}

type conversationTerminalProcess interface {
	io.ReadWriteCloser
	Resize(cols int, rows int) error
	Wait() error
}

type localConversationTerminalProcess struct {
	file *os.File
	cmd  *exec.Cmd
}

func conversationTerminalPTYSize(cols int, rows int) (*pty.Winsize, error) {
	if cols <= 0 || cols > chatdomain.MaxTerminalSize {
		return nil, fmt.Errorf("cols must be between 1 and %d", chatdomain.MaxTerminalSize)
	}
	if rows <= 0 || rows > chatdomain.MaxTerminalSize {
		return nil, fmt.Errorf("rows must be between 1 and %d", chatdomain.MaxTerminalSize)
	}
	return &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}, nil
}

func startLocalConversationTerminalProcess(
	ctx context.Context,
	spec conversationTerminalLaunchSpec,
) (conversationTerminalProcess, error) {
	args, err := resolveLocalConversationTerminalShellArgs()
	if err != nil {
		return nil, err
	}
	size, err := conversationTerminalPTYSize(spec.Cols, spec.Rows)
	if err != nil {
		return nil, err
	}
	// #nosec G204 -- the shell executable is selected from a fixed local allowlist or the resolved SHELL path.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = spec.CWD
	cmd.Env = spec.Environment
	ptmx, err := pty.StartWithSize(cmd, size)
	if err != nil {
		return nil, err
	}
	return &localConversationTerminalProcess{file: ptmx, cmd: cmd}, nil
}

func (p *localConversationTerminalProcess) Read(buffer []byte) (int, error) {
	return p.file.Read(buffer)
}

func (p *localConversationTerminalProcess) Write(buffer []byte) (int, error) {
	return p.file.Write(buffer)
}

func (p *localConversationTerminalProcess) Close() error {
	return p.file.Close()
}

func (p *localConversationTerminalProcess) Resize(cols int, rows int) error {
	size, err := conversationTerminalPTYSize(cols, rows)
	if err != nil {
		return err
	}
	return pty.Setsize(p.file, size)
}

func (p *localConversationTerminalProcess) Wait() error {
	return p.cmd.Wait()
}

type remoteConversationTerminalProcess struct {
	session machinetransport.CommandSession
	stdin   io.WriteCloser
	merged  *io.PipeReader
	writer  *io.PipeWriter
	writeMu sync.Mutex
	copyWG  sync.WaitGroup
	done    chan struct{}
	waitErr error
	once    sync.Once
}

func (p *remoteConversationTerminalProcess) Read(buffer []byte) (int, error) {
	return p.merged.Read(buffer)
}

func (p *remoteConversationTerminalProcess) Write(buffer []byte) (int, error) {
	return p.stdin.Write(buffer)
}

func (p *remoteConversationTerminalProcess) Close() error {
	closeErr := error(nil)
	p.once.Do(func() {
		if p.stdin != nil {
			_ = p.stdin.Close()
		}
		if p.session != nil {
			closeErr = p.session.Close()
		}
	})
	return closeErr
}

func (p *remoteConversationTerminalProcess) Resize(cols int, rows int) error {
	return p.session.Resize(cols, rows)
}

func (p *remoteConversationTerminalProcess) Wait() error {
	if p == nil {
		return fmt.Errorf("remote terminal session unavailable")
	}
	<-p.done
	return p.waitErr
}

func (s *ConversationTerminalService) startRemoteProcess(
	ctx context.Context,
	machine catalogdomain.Machine,
	spec conversationTerminalLaunchSpec,
) (conversationTerminalProcess, error) {
	if s == nil || s.conversations == nil {
		return nil, ErrConversationTerminalUnsupported
	}
	session, err := s.conversations.openProjectConversationCommandSession(ctx, machine, "terminal")
	if err != nil {
		if errors.Is(err, machinetransport.ErrTransportUnavailable) {
			return nil, ErrConversationTerminalUnsupported
		}
		return nil, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("open remote terminal stdin: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		_ = session.Close()
		return nil, fmt.Errorf("open remote terminal stdout: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = session.Close()
		return nil, fmt.Errorf("open remote terminal stderr: %w", err)
	}

	reader, writer := io.Pipe()
	process := &remoteConversationTerminalProcess{
		session: session,
		stdin:   stdin,
		merged:  reader,
		writer:  writer,
		done:    make(chan struct{}),
	}

	if err := session.StartPTY(buildRemoteConversationTerminalCommand(spec), spec.Cols, spec.Rows); err != nil {
		_ = stdin.Close()
		_ = session.Close()
		_ = writer.CloseWithError(err)
		return nil, fmt.Errorf("start remote terminal shell: %w", err)
	}
	process.copyWG.Add(2)
	go process.copyOutput(stdout)
	go process.copyOutput(stderr)
	go process.waitLoop()
	return process, nil
}

func (p *remoteConversationTerminalProcess) copyOutput(reader io.Reader) {
	defer p.copyWG.Done()
	if reader == nil || p == nil || p.writer == nil {
		return
	}
	buffer := make([]byte, 4096)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			p.writeMu.Lock()
			_, _ = p.writer.Write(buffer[:count])
			p.writeMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (p *remoteConversationTerminalProcess) waitLoop() {
	if p == nil {
		return
	}
	p.waitErr = p.session.Wait()
	p.copyWG.Wait()
	_ = p.writer.Close()
	close(p.done)
}

func buildRemoteConversationTerminalCommand(spec conversationTerminalLaunchSpec) string {
	cwd := strings.TrimSpace(spec.CWD)
	command := []string{}
	if cwd != "" {
		command = append(command, "cd "+projectConversationShellQuote(cwd))
	}
	command = append(command,
		`if [ -n "${SHELL:-}" ] && command -v "$SHELL" >/dev/null 2>&1; then shell="$SHELL"; `+
			`elif command -v bash >/dev/null 2>&1; then shell="$(command -v bash)"; `+
			`elif command -v zsh >/dev/null 2>&1; then shell="$(command -v zsh)"; `+
			`else shell="$(command -v sh)"; fi`,
		`exec env TERM=xterm-256color "$shell" -i`,
	)
	return "sh -lc " + projectConversationShellQuote(strings.Join(command, " && "))
}

func resolveLocalConversationTerminalShellArgs() ([]string, error) {
	candidates := []string{}
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		candidates = append(candidates, shell)
	}
	candidates = append(candidates, "/bin/bash", "/bin/zsh", "/bin/sh", "bash", "zsh", "sh")
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		path, err := resolveLocalConversationTerminalShell(candidate)
		if err != nil {
			continue
		}
		return buildConversationTerminalShellArgs(path), nil
	}
	return nil, fmt.Errorf("resolve local shell for terminal session")
}

func resolveLocalConversationTerminalShell(candidate string) (string, error) {
	if filepath.IsAbs(candidate) {
		// #nosec G703 -- the shell path comes from the local runtime environment, not browser input.
		info, err := os.Stat(candidate)
		if err != nil {
			return "", err
		}
		if info.IsDir() || info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("shell %s is not executable", candidate)
		}
		return candidate, nil
	}
	return exec.LookPath(candidate)
}

func buildConversationTerminalShellArgs(shell string) []string {
	args := []string{shell}
	switch filepath.Base(shell) {
	case "bash", "zsh", "sh", "dash", "ash", "ksh", "mksh", "fish":
		args = append(args, "-i")
	}
	return args
}

func buildConversationTerminalEnvironment() []string {
	environment := os.Environ()
	filtered := environment[:0]
	for _, item := range environment {
		if strings.HasPrefix(item, "TERM=") {
			continue
		}
		filtered = append(filtered, item)
	}
	filtered = append(filtered, "TERM=xterm-256color")
	return filtered
}

func newConversationTerminalAttachToken() (string, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("generate terminal attach token: %w", err)
	}
	return hex.EncodeToString(token), nil
}

func valueOrEmpty(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}
