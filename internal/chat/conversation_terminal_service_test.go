package chat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/google/uuid"
)

type fakeConversationTerminalProcess struct {
	reader *io.PipeReader
	writer *io.PipeWriter

	mu          sync.Mutex
	input       bytes.Buffer
	resizeCalls [][2]int
	closeCount  int

	waitCh       chan error
	completeOnce sync.Once
}

func newFakeConversationTerminalProcess() *fakeConversationTerminalProcess {
	reader, writer := io.Pipe()
	return &fakeConversationTerminalProcess{
		reader: reader,
		writer: writer,
		waitCh: make(chan error, 1),
	}
}

func (p *fakeConversationTerminalProcess) Read(buffer []byte) (int, error) {
	return p.reader.Read(buffer)
}

func (p *fakeConversationTerminalProcess) Write(buffer []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.input.Write(buffer)
}

func (p *fakeConversationTerminalProcess) Close() error {
	p.mu.Lock()
	p.closeCount++
	p.mu.Unlock()
	p.complete()
	_ = p.writer.Close()
	return p.reader.Close()
}

func (p *fakeConversationTerminalProcess) Resize(cols int, rows int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resizeCalls = append(p.resizeCalls, [2]int{cols, rows})
	return nil
}

func (p *fakeConversationTerminalProcess) Wait() error {
	return <-p.waitCh
}

func (p *fakeConversationTerminalProcess) emitOutput(t *testing.T, value string) {
	t.Helper()
	if _, err := p.writer.Write([]byte(value)); err != nil {
		t.Fatalf("emitOutput() error = %v", err)
	}
}

func (p *fakeConversationTerminalProcess) complete() {
	p.completeOnce.Do(func() {
		p.waitCh <- nil
		close(p.waitCh)
	})
}

func (p *fakeConversationTerminalProcess) inputString() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.input.String()
}

func (p *fakeConversationTerminalProcess) lastResize() [2]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.resizeCalls) == 0 {
		return [2]int{}
	}
	return p.resizeCalls[len(p.resizeCalls)-1]
}

func (p *fakeConversationTerminalProcess) closes() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeCount
}

func TestConversationTerminalServiceCreateSessionResolvesRepoCWD(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{{
		name: "backend",
		files: map[string]string{
			"src/main.go": "package main\n",
		},
	}})
	service := NewConversationTerminalService(nil, fixture.service)
	process := newFakeConversationTerminalProcess()
	var launchedSpec conversationTerminalLaunchSpec
	service.launch = func(_ context.Context, spec conversationTerminalLaunchSpec) (conversationTerminalProcess, error) {
		launchedSpec = spec
		return process, nil
	}

	repoPath := "backend"
	cwdPath := "src"
	input, err := chatdomain.ParseOpenTerminalSessionInput(chatdomain.OpenTerminalSessionRawInput{
		Mode:     "shell",
		RepoPath: &repoPath,
		CWDPath:  &cwdPath,
	})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}

	session, err := service.CreateSession(fixture.ctx, UserID("user:conversation"), fixture.conversation.ID, input)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	wantCWD := filepath.Join(fixture.repoPaths["backend"], "src")
	if launchedSpec.CWD != wantCWD {
		t.Fatalf("launch cwd = %q, want %q", launchedSpec.CWD, wantCWD)
	}
	if session.Mode != chatdomain.TerminalModeShell || session.CWD != wantCWD || session.ID == uuid.Nil || session.AttachToken == "" {
		t.Fatalf("unexpected session = %+v", session)
	}
	process.complete()
}

func TestConversationTerminalServiceCreateSessionRejectsPathEscape(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{{
		name:  "backend",
		files: map[string]string{"README.md": "workspace\n"},
	}})
	service := NewConversationTerminalService(nil, fixture.service)
	service.launch = func(_ context.Context, spec conversationTerminalLaunchSpec) (conversationTerminalProcess, error) {
		return newFakeConversationTerminalProcess(), nil
	}

	repoPath := "backend"
	cwdPath := "../outside"
	input, err := chatdomain.ParseOpenTerminalSessionInput(chatdomain.OpenTerminalSessionRawInput{
		Mode:     "shell",
		RepoPath: &repoPath,
		CWDPath:  &cwdPath,
	})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}

	_, err = service.CreateSession(fixture.ctx, UserID("user:conversation"), fixture.conversation.ID, input)
	if !errors.Is(err, ErrProjectConversationWorkspacePathInvalid) {
		t.Fatalf("CreateSession() error = %v, want ErrProjectConversationWorkspacePathInvalid", err)
	}
}

func TestConversationTerminalServiceAttachStreamsAndCleansUp(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{{
		name:  "backend",
		files: map[string]string{"README.md": "workspace\n"},
	}})
	service := NewConversationTerminalService(nil, fixture.service)
	process := newFakeConversationTerminalProcess()
	service.launch = func(_ context.Context, spec conversationTerminalLaunchSpec) (conversationTerminalProcess, error) {
		return process, nil
	}

	input, err := chatdomain.ParseOpenTerminalSessionInput(chatdomain.OpenTerminalSessionRawInput{Mode: "shell"})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}
	session, err := service.CreateSession(fixture.ctx, UserID("user:conversation"), fixture.conversation.ID, input)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, err := service.AttachSession(UserID("user:conversation"), fixture.conversation.ID, session.ID, "wrong-token"); !errors.Is(err, ErrConversationTerminalAttachForbidden) {
		t.Fatalf("AttachSession(wrong token) error = %v", err)
	}

	attachment, err := service.AttachSession(UserID("user:conversation"), fixture.conversation.ID, session.ID, session.AttachToken)
	if err != nil {
		t.Fatalf("AttachSession() error = %v", err)
	}
	ready := requireConversationTerminalEvent(t, attachment.Events)
	if ready.Type != "ready" {
		t.Fatalf("ready event = %+v", ready)
	}

	process.emitOutput(t, "prompt$ ")
	output := requireConversationTerminalEvent(t, attachment.Events)
	if output.Type != "output" || string(output.Data) != "prompt$ " {
		t.Fatalf("output event = %+v", output)
	}

	if err := attachment.Resize(90, 33); err != nil {
		t.Fatalf("Resize() error = %v", err)
	}
	if got := process.lastResize(); got != [2]int{90, 33} {
		t.Fatalf("last resize = %+v, want [90 33]", got)
	}
	if err := attachment.WriteInput([]byte("pwd\n")); err != nil {
		t.Fatalf("WriteInput() error = %v", err)
	}
	if got := process.inputString(); got != "pwd\n" {
		t.Fatalf("input = %q, want %q", got, "pwd\\n")
	}

	process.complete()
	exit := requireConversationTerminalEvent(t, attachment.Events)
	if exit.Type != "exit" || exit.ExitCode != 0 {
		t.Fatalf("exit event = %+v", exit)
	}
	awaitConversationTerminalCleanup(t, service, fixture.conversation.ID, session.ID)
}

func TestConversationTerminalServiceDetachAllowsReattachAndReplaysBufferedOutput(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{{
		name:  "backend",
		files: map[string]string{"README.md": "workspace\n"},
	}})
	service := NewConversationTerminalService(nil, fixture.service)
	process := newFakeConversationTerminalProcess()
	service.launch = func(_ context.Context, spec conversationTerminalLaunchSpec) (conversationTerminalProcess, error) {
		return process, nil
	}

	input, err := chatdomain.ParseOpenTerminalSessionInput(chatdomain.OpenTerminalSessionRawInput{Mode: "shell"})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}
	session, err := service.CreateSession(fixture.ctx, UserID("user:conversation"), fixture.conversation.ID, input)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	firstAttachment, err := service.AttachSession(
		UserID("user:conversation"),
		fixture.conversation.ID,
		session.ID,
		session.AttachToken,
	)
	if err != nil {
		t.Fatalf("AttachSession(first) error = %v", err)
	}
	if ready := requireConversationTerminalEvent(t, firstAttachment.Events); ready.Type != "ready" {
		t.Fatalf("first ready event = %+v", ready)
	}

	process.emitOutput(t, "before-detach\n")
	firstOutput := requireConversationTerminalEvent(t, firstAttachment.Events)
	if firstOutput.Type != "output" || string(firstOutput.Data) != "before-detach\n" {
		t.Fatalf("first output = %+v", firstOutput)
	}

	if err := firstAttachment.Detach(); err != nil {
		t.Fatalf("Detach() error = %v", err)
	}

	process.emitOutput(t, "after-detach\n")

	secondAttachment, err := service.AttachSession(
		UserID("user:conversation"),
		fixture.conversation.ID,
		session.ID,
		session.AttachToken,
	)
	if err != nil {
		t.Fatalf("AttachSession(second) error = %v", err)
	}
	if ready := requireConversationTerminalEvent(t, secondAttachment.Events); ready.Type != "ready" {
		t.Fatalf("second ready event = %+v", ready)
	}
	buffered := requireConversationTerminalEvent(t, secondAttachment.Events)
	if buffered.Type != "output" || string(buffered.Data) != "after-detach\n" {
		t.Fatalf("buffered output = %+v", buffered)
	}

	if err := secondAttachment.WriteInput([]byte("pwd\n")); err != nil {
		t.Fatalf("WriteInput() after reattach error = %v", err)
	}
	if got := process.inputString(); got != "pwd\n" {
		t.Fatalf("input after reattach = %q, want %q", got, "pwd\\n")
	}

	process.complete()
	exit := requireConversationTerminalEvent(t, secondAttachment.Events)
	if exit.Type != "exit" || exit.ExitCode != 0 {
		t.Fatalf("exit event = %+v", exit)
	}
	awaitConversationTerminalCleanup(t, service, fixture.conversation.ID, session.ID)
}

func TestConversationTerminalServiceCloseTriggersCleanup(t *testing.T) {
	t.Parallel()

	fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{{
		name:  "backend",
		files: map[string]string{"README.md": "workspace\n"},
	}})
	service := NewConversationTerminalService(nil, fixture.service)
	process := newFakeConversationTerminalProcess()
	service.launch = func(_ context.Context, spec conversationTerminalLaunchSpec) (conversationTerminalProcess, error) {
		return process, nil
	}

	input, _ := chatdomain.ParseOpenTerminalSessionInput(chatdomain.OpenTerminalSessionRawInput{Mode: "shell"})
	session, err := service.CreateSession(fixture.ctx, UserID("user:conversation"), fixture.conversation.ID, input)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	attachment, err := service.AttachSession(UserID("user:conversation"), fixture.conversation.ID, session.ID, session.AttachToken)
	if err != nil {
		t.Fatalf("AttachSession() error = %v", err)
	}
	_ = requireConversationTerminalEvent(t, attachment.Events)

	if err := attachment.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	exit := requireConversationTerminalEvent(t, attachment.Events)
	if exit.Type != "exit" {
		t.Fatalf("exit event = %+v", exit)
	}
	if process.closes() == 0 {
		t.Fatal("expected process close to be called")
	}
	awaitConversationTerminalCleanup(t, service, fixture.conversation.ID, session.ID)
}

func requireConversationTerminalEvent(t *testing.T, events <-chan ConversationTerminalEvent) ConversationTerminalEvent {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("expected terminal event, got closed channel")
		}
		return event
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for terminal event")
		return ConversationTerminalEvent{}
	}
}

func awaitConversationTerminalCleanup(
	t *testing.T,
	service *ConversationTerminalService,
	conversationID uuid.UUID,
	sessionID uuid.UUID,
) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if service.registry.get(conversationID, sessionID) == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected terminal session cleanup")
}
