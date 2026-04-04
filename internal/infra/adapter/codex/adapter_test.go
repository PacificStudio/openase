package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
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
			if initializeParams.Capabilities == nil || !initializeParams.Capabilities.ExperimentalAPI {
				return errors.New("expected initialize experimentalApi capability")
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

func TestAdapterStartCanResumeExistingThreadBeforeTurnStart(t *testing.T) {
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

			threadResume, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if threadResume.Method != methodThreadResume {
				return errors.New("expected thread/resume request")
			}

			var resumeParams wireThreadResumeParams
			if err := decodeParams(threadResume.Params, &resumeParams); err != nil {
				return err
			}
			if resumeParams.ThreadID != "thread-existing" {
				return errors.New("expected thread/resume id")
			}
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      threadResume.ID,
				Result: mustMarshalJSON(wireThreadResumeResponse{
					Thread: wireThread{
						ID: "thread-existing",
						Status: &wireThreadStatus{
							Type:        "active",
							ActiveFlags: []string{"waitingOnApproval"},
						},
					},
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
			if turnParams.ThreadID != "thread-existing" {
				return errors.New("expected attached thread id")
			}

			return encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      turnStart.ID,
				Result: mustMarshalJSON(wireTurnStartResponse{
					Turn: wireTurn{ID: "turn-existing", Status: "inProgress"},
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
		Thread: ThreadStartParams{
			ResumeThreadID: "thread-existing",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if session.ThreadID() != "thread-existing" {
		t.Fatalf("expected attached thread id, got %q", session.ThreadID())
	}
	threadStatus := session.ThreadStatus()
	if threadStatus == nil || threadStatus.Status != "active" || len(threadStatus.ActiveFlags) != 1 || threadStatus.ActiveFlags[0] != "waitingOnApproval" {
		t.Fatalf("expected resumed thread status, got %+v", threadStatus)
	}

	turn, err := session.SendPrompt(context.Background(), "Continue this thread.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-existing" {
		t.Fatalf("expected turn-existing, got %q", turn.TurnID)
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

func TestAdapterMapsThreadStatusAndTurnCancelledNotifications(t *testing.T) {
	process := newFakeProcess()
	manager := &fakeProcessManager{process: process}
	adapter, err := NewAdapter(AdapterOptions{ProcessManager: manager})
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
				Method:  methodThreadStatusChanged,
				Params: mustMarshalJSON(wireThreadStatusChangedNotification{
					ThreadID: "thread-1",
					Status: wireThreadStatus{
						Type:        "active",
						ActiveFlags: []string{"waitingOnUserInput"},
					},
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
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      turnStart.ID,
				Result: mustMarshalJSON(wireTurnStartResponse{
					Turn: wireTurn{ID: "turn-2", Status: "inProgress"},
				}),
			}); err != nil {
				return err
			}
			return encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				Method:  methodTurnCancelled,
				Params: mustMarshalJSON(wireErrorNotification{
					ThreadID: "thread-1",
					TurnID:   "turn-2",
					Error: wireTurnError{
						Message: "operator interrupted",
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
		Thread:  ThreadStartParams{WorkingDirectory: "/tmp/openase"},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	threadStatusEvent := requireEvent(t, session.Events())
	if threadStatusEvent.Type != EventTypeThreadStatus || threadStatusEvent.ThreadStatus == nil {
		t.Fatalf("expected thread status event, got %+v", threadStatusEvent)
	}
	if threadStatusEvent.ThreadStatus.Status != "active" || len(threadStatusEvent.ThreadStatus.ActiveFlags) != 1 || threadStatusEvent.ThreadStatus.ActiveFlags[0] != "waitingOnUserInput" {
		t.Fatalf("unexpected thread status payload %+v", threadStatusEvent.ThreadStatus)
	}

	turn, err := session.SendPrompt(context.Background(), "Need more input")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-2" {
		t.Fatalf("expected turn-2, got %q", turn.TurnID)
	}

	turnEvent := requireEvent(t, session.Events())
	if turnEvent.Type != EventTypeTurnInterrupted || turnEvent.Turn == nil || turnEvent.Turn.Status != "interrupted" {
		t.Fatalf("expected interrupted turn event, got %+v", turnEvent)
	}

	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestAdapterMapsThreadLifecyclePlanDiffAndReasoningNotifications(t *testing.T) {
	process := newFakeProcess()
	manager := &fakeProcessManager{process: process}
	adapter, err := NewAdapter(AdapterOptions{ProcessManager: manager})
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
				Method:  methodThreadStarted,
				Params: mustMarshalJSON(wireThreadStartedNotification{
					Thread: wireThread{
						ID: "thread-9",
						Status: &wireThreadStatus{
							Type:        "idle",
							ActiveFlags: []string{"booting"},
						},
					},
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
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      turnStart.ID,
				Result: mustMarshalJSON(wireTurnStartResponse{
					Turn: wireTurn{ID: "turn-9", Status: "inProgress"},
				}),
			}); err != nil {
				return err
			}

			notifications := []jsonRPCMessage{
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodTurnPlanUpdated,
					Params: mustMarshalJSON(wireTurnPlanUpdatedNotification{
						ThreadID:    "thread-9",
						TurnID:      "turn-9",
						Explanation: stringPointer("Need two steps"),
						Plan: []wireTurnPlanStep{
							{Step: "Inspect repo", Status: "completed"},
							{Step: "Patch tests", Status: "in_progress"},
						},
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodTurnDiffUpdated,
					Params: mustMarshalJSON(wireTurnDiffUpdatedNotification{
						ThreadID: "thread-9",
						TurnID:   "turn-9",
						Diff:     "diff --git a/app.go b/app.go",
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodReasoningSummaryPart,
					Params: mustMarshalJSON(wireReasoningSummaryPartAddedNotification{
						ThreadID:     "thread-9",
						TurnID:       "turn-9",
						ItemID:       "item-1",
						SummaryIndex: 0,
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodReasoningSummaryText,
					Params: mustMarshalJSON(wireReasoningSummaryTextDeltaNotification{
						ThreadID:     "thread-9",
						TurnID:       "turn-9",
						ItemID:       "item-1",
						Delta:        "Summarized reasoning",
						SummaryIndex: 0,
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodReasoningText,
					Params: mustMarshalJSON(wireReasoningTextDeltaNotification{
						ThreadID:     "thread-9",
						TurnID:       "turn-9",
						ItemID:       "item-1",
						Delta:        "Detailed reasoning",
						ContentIndex: 1,
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodThreadCompacted,
					Params: mustMarshalJSON(wireContextCompactedNotification{
						ThreadID: "thread-9",
						TurnID:   "turn-9",
					}),
				},
			}
			for _, notification := range notifications {
				if err := encoder.Encode(notification); err != nil {
					return err
				}
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
		Process: processSpec,
		Thread:  ThreadStartParams{WorkingDirectory: "/tmp/openase"},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	threadStarted := requireEvent(t, session.Events())
	if threadStarted.Type != EventTypeThreadStarted || threadStarted.Thread == nil || threadStarted.Thread.ThreadID != "thread-9" {
		t.Fatalf("expected thread started event, got %+v", threadStarted)
	}

	turn, err := session.SendPrompt(context.Background(), "Continue")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-9" {
		t.Fatalf("expected turn-9, got %q", turn.TurnID)
	}

	planEvent := requireEvent(t, session.Events())
	if planEvent.Type != EventTypeTurnPlanUpdated || planEvent.Plan == nil || len(planEvent.Plan.Plan) != 2 {
		t.Fatalf("expected plan update event, got %+v", planEvent)
	}
	diffEvent := requireEvent(t, session.Events())
	if diffEvent.Type != EventTypeTurnDiffUpdated || diffEvent.Diff == nil || !strings.Contains(diffEvent.Diff.Diff, "app.go") {
		t.Fatalf("expected diff update event, got %+v", diffEvent)
	}
	reasoningPart := requireEvent(t, session.Events())
	if reasoningPart.Type != EventTypeReasoningUpdated || reasoningPart.Reasoning == nil || reasoningPart.Reasoning.Kind != ReasoningKindSummaryPart {
		t.Fatalf("expected reasoning summary part event, got %+v", reasoningPart)
	}
	reasoningSummary := requireEvent(t, session.Events())
	if reasoningSummary.Reasoning == nil || reasoningSummary.Reasoning.Kind != ReasoningKindSummaryText || reasoningSummary.Reasoning.Delta != "Summarized reasoning" {
		t.Fatalf("expected reasoning summary text event, got %+v", reasoningSummary)
	}
	reasoningText := requireEvent(t, session.Events())
	if reasoningText.Reasoning == nil || reasoningText.Reasoning.Kind != ReasoningKindText || reasoningText.Reasoning.Delta != "Detailed reasoning" {
		t.Fatalf("expected reasoning text event, got %+v", reasoningText)
	}
	compacted := requireEvent(t, session.Events())
	if compacted.Type != EventTypeThreadCompacted || compacted.Compaction == nil || compacted.Compaction.ThreadID != "thread-9" {
		t.Fatalf("expected thread compacted event, got %+v", compacted)
	}

	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestAdapterStartAcceptsResponsesWithoutJSONRPCVersion(t *testing.T) {
	process := newFakeProcess()
	adapter, err := NewAdapter(AdapterOptions{ProcessManager: &fakeProcessManager{process: process}})
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
			if err := encoder.Encode(jsonRPCMessage{
				ID: initialize.ID,
				Result: mustMarshalJSON(wireInitializeResponse{
					UserAgent:      "codex-cli/0.115.0",
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
				ID: threadStart.ID,
				Result: mustMarshalJSON(wireThreadStartResponse{
					Thread: wireThread{ID: "thread-no-jsonrpc"},
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
		Thread: ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if session.ThreadID() != "thread-no-jsonrpc" {
		t.Fatalf("expected thread id thread-no-jsonrpc, got %q", session.ThreadID())
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestAdapterSendPromptUsesTurnDefaultsAndAutoApprovesRequests(t *testing.T) {
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
			if turnParams.CWD == nil || *turnParams.CWD != "/tmp/openase" {
				return errors.New("expected turn cwd")
			}
			if turnParams.Title == nil || *turnParams.Title != "ASE-1: Implement the adapter" {
				return errors.New("expected turn title")
			}
			if decision, ok := turnParams.ApprovalPolicy.(string); !ok || decision != "never" {
				return errors.New("expected turn approval policy")
			}
			sandboxPolicy, ok := turnParams.SandboxPolicy.(map[string]any)
			if !ok || sandboxPolicy["type"] != "dangerFullAccess" || sandboxPolicy["networkAccess"] != true {
				return errors.New("expected turn sandbox policy")
			}

			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      turnStart.ID,
				Result: mustMarshalJSON(wireTurnStartResponse{
					Turn: wireTurn{ID: "turn-approval", Status: "inProgress"},
				}),
			}); err != nil {
				return err
			}

			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      mustMarshalJSON("approval-1"),
				Method:  methodCommandApproval,
				Params:  mustMarshalJSON(map[string]any{"threadId": "thread-1"}),
			}); err != nil {
				return err
			}

			approvalResponse, err := readMessage(decoder)
			if err != nil {
				return err
			}
			var approvalResult map[string]string
			if err := json.Unmarshal(approvalResponse.Result, &approvalResult); err != nil {
				return err
			}
			if approvalResult["decision"] != "acceptForSession" {
				return errors.New("expected command approval auto response")
			}

			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				Method:  methodTokenUsageUpdated,
				Params: mustMarshalJSON(wireThreadTokenUsageUpdatedNotification{
					ThreadID: "thread-1",
					TurnID:   "turn-approval",
					TokenUsage: wireThreadTokenUsage{
						Total: wireTokenUsageBreakdown{
							InputTokens:  120,
							OutputTokens: 35,
							TotalTokens:  155,
						},
						Last: wireTokenUsageBreakdown{
							InputTokens:  20,
							OutputTokens: 5,
							TotalTokens:  25,
						},
					},
				}),
			}); err != nil {
				return err
			}

			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				Method:  methodAccountRateLimitsUpdated,
				Params: mustMarshalJSON(wireAccountRateLimitsUpdatedNotification{
					RateLimits: json.RawMessage(`{
						"limitId":"codex",
						"primary":{"usedPercent":15,"windowDurationMins":300,"resetsAt":1775050232},
						"secondary":{"usedPercent":4,"windowDurationMins":10080,"resetsAt":1775637032},
						"planType":"pro"
					}`),
				}),
			}); err != nil {
				return err
			}

			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      mustMarshalJSON("input-1"),
				Method:  methodRequestUserInput,
				Params: mustMarshalJSON(map[string]any{
					"questions": []map[string]any{
						{
							"id": "approval",
							"options": []map[string]any{
								{"label": "Approve this Session"},
								{"label": "Deny"},
							},
						},
					},
				}),
			}); err != nil {
				return err
			}

			inputResponse, err := readMessage(decoder)
			if err != nil {
				return err
			}
			var inputResult struct {
				Answers map[string]struct {
					Answers []string `json:"answers"`
				} `json:"answers"`
			}
			if err := json.Unmarshal(inputResponse.Result, &inputResult); err != nil {
				return err
			}
			if got := inputResult.Answers["approval"].Answers; len(got) != 1 || got[0] != "Approve this Session" {
				return errors.New("expected requestUserInput auto answer")
			}

			return encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				Method:  methodTurnCompleted,
				Params: mustMarshalJSON(wireTurnNotification{
					ThreadID: "thread-1",
					Turn: wireTurn{
						ID:     "turn-approval",
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
		Thread: ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
			ApprovalPolicy:   "never",
		},
		Turn: TurnConfig{
			WorkingDirectory: "/tmp/openase",
			Title:            "ASE-1: Implement the adapter",
			ApprovalPolicy:   "never",
			SandboxPolicy: map[string]any{
				"type":          "dangerFullAccess",
				"networkAccess": true,
			},
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	turn, err := session.SendPrompt(context.Background(), "Continue working.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-approval" {
		t.Fatalf("expected turn id turn-approval, got %q", turn.TurnID)
	}

	approvalEvent := requireEvent(t, session.Events())
	if approvalEvent.Type != EventTypeApprovalRequested || approvalEvent.Approval == nil || approvalEvent.Approval.Kind != ApprovalRequestKindCommandExecution {
		t.Fatalf("expected approval request event, got %+v", approvalEvent)
	}

	tokenEvent := requireEvent(t, session.Events())
	if tokenEvent.Type != EventTypeTokenUsageUpdated || tokenEvent.TokenUsage == nil {
		t.Fatalf("expected token usage event, got %+v", tokenEvent)
	}
	if tokenEvent.TokenUsage.TotalTokens != 155 || tokenEvent.TokenUsage.LastTokens != 25 {
		t.Fatalf("unexpected token usage event: %+v", tokenEvent.TokenUsage)
	}

	rateLimitEvent := requireEvent(t, session.Events())
	if rateLimitEvent.Type != EventTypeRateLimitUpdated || rateLimitEvent.RateLimit == nil || rateLimitEvent.RateLimit.Codex == nil {
		t.Fatalf("expected rate limit event, got %+v", rateLimitEvent)
	}
	if rateLimitEvent.RateLimit.Codex.Primary == nil ||
		rateLimitEvent.RateLimit.Codex.Primary.UsedPercent == nil ||
		*rateLimitEvent.RateLimit.Codex.Primary.UsedPercent != 15 {
		t.Fatalf("unexpected rate limit payload: %+v", rateLimitEvent.RateLimit.Codex)
	}

	userInputEvent := requireEvent(t, session.Events())
	if userInputEvent.Type != EventTypeUserInputRequested || userInputEvent.UserInput == nil || userInputEvent.UserInput.RequestID.String() != `"input-1"` {
		t.Fatalf("expected user input request event, got %+v", userInputEvent)
	}

	completedEvent := requireEvent(t, session.Events())
	if completedEvent.Type != EventTypeTurnCompleted || completedEvent.Turn == nil || completedEvent.Turn.TurnID != "turn-approval" {
		t.Fatalf("expected completed turn event, got %+v", completedEvent)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestAdapterEmitsOutputEventsFromRuntimeNotifications(t *testing.T) {
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

			turnStart, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if turnStart.Method != methodTurnStart {
				return errors.New("expected turn/start request")
			}
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      turnStart.ID,
				Result: mustMarshalJSON(wireTurnStartResponse{
					Turn: wireTurn{ID: "turn-output", Status: "inProgress"},
				}),
			}); err != nil {
				return err
			}

			notifications := []jsonRPCMessage{
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodAgentMessageDelta,
					Params: mustMarshalJSON(wireAgentMessageDeltaNotification{
						ThreadID: "thread-1",
						TurnID:   "turn-output",
						ItemID:   "msg-1",
						Delta:    "Thinking through the fix.",
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodCommandOutput,
					Params: mustMarshalJSON(wireCommandExecutionOutputDeltaNotification{
						ThreadID: "thread-1",
						TurnID:   "turn-output",
						ItemID:   "cmd-1",
						Command:  "go test ./...",
						Delta:    "go test ./...\n",
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodItemCompleted,
					Params: mustMarshalJSON(wireItemCompletedNotification{
						ThreadID: "thread-1",
						TurnID:   "turn-output",
						Item: wireThreadItem{
							ID:    "msg-2",
							Type:  "agentMessage",
							Text:  "Applied the patch.",
							Phase: "commentary",
						},
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodItemCompleted,
					Params: mustMarshalJSON(wireItemCompletedNotification{
						ThreadID: "thread-1",
						TurnID:   "turn-output",
						Item: wireThreadItem{
							ID:               "cmd-2",
							Type:             "commandExecution",
							Command:          stringPointer("go test ./..."),
							AggregatedOutput: stringPointer("PASS\n"),
						},
					}),
				},
				{
					JSONRPC: jsonRPCVersion,
					Method:  methodTurnCompleted,
					Params: mustMarshalJSON(wireTurnNotification{
						ThreadID: "thread-1",
						Turn: wireTurn{
							ID:     "turn-output",
							Status: "completed",
						},
					}),
				},
			}
			for _, notification := range notifications {
				if err := encoder.Encode(notification); err != nil {
					return err
				}
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
		Process: processSpec,
		Thread: ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	turn, err := session.SendPrompt(context.Background(), "Keep working.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-output" {
		t.Fatalf("expected turn id turn-output, got %q", turn.TurnID)
	}

	assistantDelta := requireEvent(t, session.Events())
	if assistantDelta.Type != EventTypeOutputProduced || assistantDelta.Output == nil {
		t.Fatalf("expected assistant output event, got %+v", assistantDelta)
	}
	if assistantDelta.Output.Stream != "assistant" || assistantDelta.Output.Text != "Thinking through the fix." || assistantDelta.Output.Snapshot {
		t.Fatalf("unexpected assistant delta event: %+v", assistantDelta.Output)
	}

	commandDelta := requireEvent(t, session.Events())
	if commandDelta.Type != EventTypeOutputProduced || commandDelta.Output == nil {
		t.Fatalf("expected command output event, got %+v", commandDelta)
	}
	if commandDelta.Output.Stream != "command" || commandDelta.Output.Command != "go test ./..." || commandDelta.Output.Text != "go test ./...\n" || commandDelta.Output.Snapshot {
		t.Fatalf("unexpected command delta event: %+v", commandDelta.Output)
	}

	assistantSnapshot := requireEvent(t, session.Events())
	if assistantSnapshot.Type != EventTypeOutputProduced || assistantSnapshot.Output == nil {
		t.Fatalf("expected assistant snapshot event, got %+v", assistantSnapshot)
	}
	if !assistantSnapshot.Output.Snapshot || assistantSnapshot.Output.Phase != "commentary" || assistantSnapshot.Output.Text != "Applied the patch." {
		t.Fatalf("unexpected assistant snapshot event: %+v", assistantSnapshot.Output)
	}

	commandSnapshot := requireEvent(t, session.Events())
	if commandSnapshot.Type != EventTypeOutputProduced || commandSnapshot.Output == nil {
		t.Fatalf("expected command snapshot event, got %+v", commandSnapshot)
	}
	if !commandSnapshot.Output.Snapshot || commandSnapshot.Output.Command != "go test ./..." || commandSnapshot.Output.Text != "PASS\n" {
		t.Fatalf("unexpected command snapshot event: %+v", commandSnapshot.Output)
	}

	completedEvent := requireEvent(t, session.Events())
	if completedEvent.Type != EventTypeTurnCompleted || completedEvent.Turn == nil || completedEvent.Turn.TurnID != "turn-output" {
		t.Fatalf("expected completed turn event, got %+v", completedEvent)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestSessionStopAddsDefaultTimeoutWhenDeadlineIsMissing(t *testing.T) {
	process := newFakeProcess()
	timeoutCh := make(chan time.Duration, 1)
	process.stopFn = func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("expected stop context deadline")
		}
		timeoutCh <- time.Until(deadline)
		process.finish(nil)
		return nil
	}

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
			<-process.stopOnce.ch
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
		Process: processSpec,
		Initialize: InitializeParams{
			ClientName:    "openase",
			ClientVersion: "0.1.0",
		},
		Thread: ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	select {
	case observed := <-timeoutCh:
		if observed < defaultShutdownTimeout-time.Second || observed > defaultShutdownTimeout+time.Second {
			t.Fatalf("expected stop timeout near %s, got %s", defaultShutdownTimeout, observed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stop timeout observation")
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestSessionStopDropsLateNotificationsAfterShutdown(t *testing.T) {
	process := newFakeProcess()
	process.stopFn = func(context.Context) error {
		process.signalStopped(nil)
		return nil
	}

	adapter, err := NewAdapter(AdapterOptions{ProcessManager: &fakeProcessManager{process: process}})
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	serverReady := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		decoder := json.NewDecoder(process.stdinRead)
		encoder := json.NewEncoder(process.stdoutWrite)

		if err := completeHandshake(decoder, encoder); err != nil {
			serverDone <- err
			return
		}

		close(serverReady)
		<-process.stopOnce.ch
		serverDone <- nil
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
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	select {
	case <-serverReady:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for fake server handshake")
	}

	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	encoder := json.NewEncoder(process.stdoutWrite)
	if err := encoder.Encode(jsonRPCMessage{
		JSONRPC: jsonRPCVersion,
		Method:  methodAgentMessageDelta,
		Params: mustMarshalJSON(wireAgentMessageDeltaNotification{
			ThreadID: "thread-1",
			TurnID:   "turn-late",
			ItemID:   "item-late",
			Delta:    "late output after shutdown",
		}),
	}); err != nil {
		t.Fatalf("encode late notification: %v", err)
	}
	if err := process.stdoutWrite.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}

	select {
	case event, ok := <-session.Events():
		if ok {
			t.Fatalf("expected closed events channel after shutdown, got %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for events channel to close")
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestAdapterPreservesExitErrorWhenStdoutClosesBeforeTurnCompleted(t *testing.T) {
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

			turnStart, err := readMessage(decoder)
			if err != nil {
				return err
			}
			if turnStart.Method != methodTurnStart {
				return errors.New("expected turn/start request")
			}
			if err := encoder.Encode(jsonRPCMessage{
				JSONRPC: jsonRPCVersion,
				ID:      turnStart.ID,
				Result: mustMarshalJSON(wireTurnStartResponse{
					Turn: wireTurn{ID: "turn-eof", Status: "inProgress"},
				}),
			}); err != nil {
				return err
			}

			if _, err := io.WriteString(process.stderrWrite, "fatal: app-server crashed"); err != nil {
				return err
			}
			process.finish(errors.New("exit status 2"))
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
		Process: processSpec,
		Thread: ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	turn, err := session.SendPrompt(context.Background(), "Keep working.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-eof" {
		t.Fatalf("expected turn id turn-eof, got %q", turn.TurnID)
	}

	select {
	case event, ok := <-session.Events():
		if ok {
			t.Fatalf("expected closed events channel, got %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for events channel to close")
	}

	if err := session.Err(); err == nil || err.Error() != "codex app server exited: exit status 2: fatal: app-server crashed" {
		t.Fatalf("session.Err() = %v", err)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
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

	done     chan struct{}
	stopOnce syncOnce
	stopFn   func(context.Context) error

	waitMu  sync.Mutex
	waitErr error
}

type syncOnce struct {
	once sync.Once
	ch   chan struct{}
}

func (s *syncOnce) Do(fn func()) {
	s.once.Do(func() {
		close(s.ch)
		fn()
	})
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
		done:        make(chan struct{}),
		stopOnce:    syncOnce{ch: make(chan struct{}, 1)},
	}
}

func (p *fakeProcess) PID() int              { return p.pid }
func (p *fakeProcess) Stdin() io.WriteCloser { return p.stdinWrite }
func (p *fakeProcess) Stdout() io.ReadCloser { return p.stdoutRead }
func (p *fakeProcess) Stderr() io.ReadCloser { return p.stderrRead }
func (p *fakeProcess) Wait() error {
	<-p.done
	p.waitMu.Lock()
	defer p.waitMu.Unlock()
	return p.waitErr
}
func (p *fakeProcess) Stop(ctx context.Context) error {
	if p.stopFn != nil {
		return p.stopFn(ctx)
	}
	p.finish(nil)
	return nil
}

func (p *fakeProcess) signalStopped(err error) {
	p.stopOnce.Do(func() {
		p.waitMu.Lock()
		p.waitErr = err
		p.waitMu.Unlock()
		close(p.done)
	})
}

func (p *fakeProcess) finish(err error) {
	p.stopOnce.Do(func() {
		_ = p.stdinWrite.Close()
		_ = p.stdoutWrite.Close()
		_ = p.stderrWrite.Close()
		p.waitMu.Lock()
		p.waitErr = err
		p.waitMu.Unlock()
		close(p.done)
	})
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

func stringPointer(value string) *string {
	return &value
}
