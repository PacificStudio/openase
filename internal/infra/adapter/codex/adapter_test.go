package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestAdapterStartSendPromptAndRespondToolCall(t *testing.T) {
	process := newFakeProcess()
	manager := &fakeProcessManager{process: process}
	adapter, err := NewAdapter(AdapterOptions{ProcessManager: manager})
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runProtocolServer(process, func(decoder *json.Decoder, encoder *json.Encoder) error {
			initialize, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if initialize.Method != methodInitialize {
				return errors.New("expected initialize request")
			}
			var initializeParams wireInitializeParams
			if err := decodeParams(initialize.Params, &initializeParams); err != nil {
				return err
			}
			if initializeParams.ClientInfo.Name != "openase" || initializeParams.ClientInfo.Version != "0.1.0" {
				return errors.New("unexpected initialize client info")
			}
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      initialize.ID,
				Result: mustMarshalJSON(wireInitializeResponse{
					UserAgent:      "codex-cli/1.0.0",
					PlatformFamily: "unix",
					PlatformOS:     "linux",
				}),
			}); err != nil {
				return err
			}

			initialized, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if initialized.Method != methodInitialized {
				return errors.New("expected initialized notification")
			}

			threadStart, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if threadStart.Method != methodThreadStart {
				return errors.New("expected thread/start request")
			}
			var threadParams wireThreadStartParams
			if err := decodeParams(threadStart.Params, &threadParams); err != nil {
				return err
			}
			if threadParams.CWD == nil || *threadParams.CWD != "/tmp/openase" {
				return errors.New("expected thread/start cwd")
			}
			if !threadParams.PersistExtendedHistory {
				return errors.New("expected persistExtendedHistory true")
			}
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      threadStart.ID,
				Result: mustMarshalJSON(wireThreadStartResponse{
					Thread: wireThread{ID: "thread-1"},
				}),
			}); err != nil {
				return err
			}

			turnStart, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if turnStart.Method != methodTurnStart {
				return errors.New("expected turn/start request")
			}
			var turnParams wireTurnStartParams
			if err := decodeParams(turnStart.Params, &turnParams); err != nil {
				return err
			}
			if turnParams.ThreadID != "thread-1" {
				return errors.New("expected turn/start thread id")
			}
			if len(turnParams.Input) != 1 || turnParams.Input[0].Text != "Implement the adapter." {
				return errors.New("expected turn/start prompt")
			}
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      turnStart.ID,
				Result: mustMarshalJSON(wireTurnStartResponse{
					Turn: wireTurn{ID: "turn-1", Status: "inProgress"},
				}),
			}); err != nil {
				return err
			}

			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      mustMarshalJSON("tool-request-1"),
				Method:  methodToolCall,
				Params: mustMarshalJSON(wireToolCallRequestParams{
					ThreadID:  "thread-1",
					TurnID:    "turn-1",
					CallID:    "call-1",
					Tool:      "read_file",
					Arguments: json.RawMessage(`{"path":"README.md"}`),
				}),
			}); err != nil {
				return err
			}

			toolCallResult, err := readMessage(decoder)
			if err != nil {
				return err
			}
			requestID, err := parseRequestID(toolCallResult.ID)
			if err != nil {
				return err
			}
			if requestID.String() != `"tool-request-1"` {
				return errors.New("expected tool call result request id")
			}
			var response wireToolCallResponse
			if err := json.Unmarshal(toolCallResult.Result, &response); err != nil {
				return err
			}
			if !response.Success || len(response.ContentItems) != 1 || response.ContentItems[0].Text != "file contents" {
				return errors.New("unexpected tool call response payload")
			}

			return encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				Method:  methodTurnCompleted,
				Params: mustMarshalJSON(wireTurnNotification{
					ThreadID: "thread-1",
					Turn: wireTurn{
						ID:     "turn-1",
						Status: "completed",
					},
				}),
			})
		})
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("codex"),
		[]string{"app-server", "--listen", "stdio://"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := adapter.Start(context.Background(), StartRequest{
		Process: processSpec,
		Initialize: InitializeParams{
			ClientName:    "openase",
			ClientVersion: "0.1.0",
		},
		Thread: ThreadStartParams{
			WorkingDirectory:       "/tmp/openase",
			PersistExtendedHistory: true,
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if session.ThreadID() != "thread-1" {
		t.Fatalf("expected thread id thread-1, got %q", session.ThreadID())
	}
	if !equalStrings(manager.startSpec.Args, []string{"app-server", "--listen", "stdio://"}) {
		t.Fatalf("expected process args to round-trip, got %v", manager.startSpec.Args)
	}

	turn, err := session.SendPrompt(context.Background(), "Implement the adapter.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-1" {
		t.Fatalf("expected turn id turn-1, got %q", turn.TurnID)
	}

	toolEvent := requireEvent(t, session.Events())
	if toolEvent.Type != EventTypeToolCallRequested || toolEvent.ToolCall == nil {
		t.Fatalf("expected tool call event, got %+v", toolEvent)
	}
	if string(toolEvent.ToolCall.Arguments) != `{"path":"README.md"}` {
		t.Fatalf("unexpected tool call arguments: %s", toolEvent.ToolCall.Arguments)
	}

	if err := session.RespondToolCall(context.Background(), *toolEvent.ToolCall, ToolCallResult{
		Success: true,
		ContentItems: []ToolCallContentItem{
			{
				Type: ToolCallContentTypeText,
				Text: "file contents",
			},
		},
	}); err != nil {
		t.Fatalf("RespondToolCall returned error: %v", err)
	}

	completedEvent := requireEvent(t, session.Events())
	if completedEvent.Type != EventTypeTurnCompleted || completedEvent.Turn == nil || completedEvent.Turn.Status != "completed" {
		t.Fatalf("expected completed turn event, got %+v", completedEvent)
	}

	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestAdapterRespondsMethodNotFoundForUnsupportedServerRequest(t *testing.T) {
	process := newFakeProcess()
	adapter, err := NewAdapter(AdapterOptions{ProcessManager: &fakeProcessManager{process: process}})
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runProtocolServer(process, func(decoder *json.Decoder, encoder *json.Encoder) error {
			if err := completeHandshake(decoder, encoder); err != nil {
				return err
			}
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      mustMarshalJSON(99),
				Method:  "item/permissions/requestApproval",
				Params:  mustMarshalJSON(map[string]any{"threadId": "thread-1"}),
			}); err != nil {
				return err
			}

			response, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if response.Error == nil || response.Error.Code != jsonRPCMethodNotFound {
				return errors.New("expected method not found error response")
			}

			return nil
		})
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("codex"),
		[]string{"app-server", "--listen", "stdio://"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := adapter.Start(context.Background(), StartRequest{
		Process:    processSpec,
		Initialize: InitializeParams{},
		Thread: ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

type fakeProcessManager struct {
	process   *fakeProcess
	startSpec provider.AgentCLIProcessSpec
}

func (m *fakeProcessManager) Start(_ context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.startSpec = spec
	return m.process, nil
}

type fakeProcess struct {
	pid int

	stdinRead  *io.PipeReader
	stdinWrite *io.PipeWriter

	stdoutRead  *io.PipeReader
	stdoutWrite *io.PipeWriter

	stderrRead  *io.PipeReader
	stderrWrite *io.PipeWriter

	done     chan error
	stopOnce syncOnce
}

type syncOnce struct {
	ch chan struct{}
}

func newFakeProcess() *fakeProcess {
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()

	return &fakeProcess{
		pid:         4242,
		stdinRead:   stdinRead,
		stdinWrite:  stdinWrite,
		stdoutRead:  stdoutRead,
		stdoutWrite: stdoutWrite,
		stderrRead:  stderrRead,
		stderrWrite: stderrWrite,
		done:        make(chan error, 1),
		stopOnce:    syncOnce{ch: make(chan struct{}, 1)},
	}
}

func (p *fakeProcess) PID() int                   { return p.pid }
func (p *fakeProcess) Stdin() io.WriteCloser      { return p.stdinWrite }
func (p *fakeProcess) Stdout() io.ReadCloser      { return p.stdoutRead }
func (p *fakeProcess) Stderr() io.ReadCloser      { return p.stderrRead }
func (p *fakeProcess) Wait() error                { return <-p.done }
func (p *fakeProcess) Stop(context.Context) error { p.finish(nil); return nil }

func (p *fakeProcess) finish(err error) {
	select {
	case <-p.stopOnce.ch:
		return
	default:
		close(p.stopOnce.ch)
		_ = p.stdinRead.Close()
		_ = p.stdinWrite.Close()
		_ = p.stdoutRead.Close()
		_ = p.stdoutWrite.Close()
		_ = p.stderrRead.Close()
		_ = p.stderrWrite.Close()
		p.done <- err
		close(p.done)
	}
}

func runProtocolServer(process *fakeProcess, handler func(decoder *json.Decoder, encoder *json.Encoder) error) error {
	defer process.finish(nil)

	decoder := json.NewDecoder(process.stdinRead)
	encoder := json.NewEncoder(process.stdoutWrite)

	return handler(decoder, encoder)
}

func completeHandshake(decoder *json.Decoder, encoder *json.Encoder) error {
	initialize, err := readMessage(decoder)
	if err != nil {
		return err
	}
	if initialize.Method != methodInitialize {
		return errors.New("expected initialize request")
	}
	if err := encoder.Encode(jsonRPCMessage{
		JSONRPC: jsonRPCVersion,
		ID:      initialize.ID,
		Result: mustMarshalJSON(wireInitializeResponse{
			UserAgent:      "codex-cli/1.0.0",
			PlatformFamily: "unix",
			PlatformOS:     "linux",
		}),
	}); err != nil {
		return err
	}

	initialized, err := readMessage(decoder)
	if err != nil {
		return err
	}
	if initialized.Method != methodInitialized {
		return errors.New("expected initialized notification")
	}

	threadStart, err := readMessage(decoder)
	if err != nil {
		return err
	}
	if threadStart.Method != methodThreadStart {
		return errors.New("expected thread/start request")
	}

	return encoder.Encode(jsonRPCMessage{
		JSONRPC: jsonRPCVersion,
		ID:      threadStart.ID,
		Result: mustMarshalJSON(wireThreadStartResponse{
			Thread: wireThread{ID: "thread-1"},
		}),
	})
}

func readMessage(decoder *json.Decoder) (jsonRPCMessage, error) {
	var message jsonRPCMessage
	if err := decoder.Decode(&message); err != nil {
		return jsonRPCMessage{}, err
	}

	return message, message.validate()
}

func requireEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("expected event, got closed channel")
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for codex event")
		return Event{}
	}
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}
