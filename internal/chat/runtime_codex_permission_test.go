package chat

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestCodexRuntimeStartTurnUsesProviderPermissionProfileForPersistentConversation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name               string
		profile            catalogdomain.AgentProviderPermissionProfile
		wantThreadApproval string
		wantThreadSandbox  string
		wantTurnApproval   string
		wantSandboxType    string
		wantNetworkAccess  bool
	}{
		{
			name:               "standard profile requests approval in project ai",
			profile:            catalogdomain.AgentProviderPermissionProfileStandard,
			wantThreadApproval: "on-request",
			wantThreadSandbox:  "workspace-write",
			wantTurnApproval:   "on-request",
			wantSandboxType:    "workspaceWrite",
			wantNetworkAccess:  false,
		},
		{
			name:               "unrestricted profile bypasses approval in project ai",
			profile:            catalogdomain.AgentProviderPermissionProfileUnrestricted,
			wantThreadApproval: "never",
			wantThreadSandbox:  "danger-full-access",
			wantTurnApproval:   "never",
			wantSandboxType:    "dangerFullAccess",
			wantNetworkAccess:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			process := newCodexRuntimePermissionTestProcess()
			manager := &fakeAgentCLIProcessManager{process: process}
			adapter, err := codexadapter.NewAdapter(codexadapter.AdapterOptions{ProcessManager: manager})
			if err != nil {
				t.Fatalf("NewAdapter() error = %v", err)
			}
			runtime := NewCodexRuntime(adapter)

			var capturedThread codexRuntimePermissionThreadStartParams
			var capturedTurn codexRuntimePermissionTurnStartParams
			serverDone := make(chan error, 1)
			go func() {
				serverDone <- runCodexRuntimePermissionServer(
					t,
					process,
					&capturedThread,
					&capturedTurn,
				)
			}()

			stream, err := runtime.StartTurn(context.Background(), RuntimeTurnInput{
				SessionID:              SessionID("conversation-1"),
				Provider:               codexRuntimePermissionProvider(tc.profile),
				Message:                "Inspect the repository state.",
				SystemPrompt:           "You are OpenASE Project AI.",
				WorkingDirectory:       provider.MustParseAbsolutePath("/tmp/openase"),
				PersistentConversation: true,
			})
			if err != nil {
				t.Fatalf("StartTurn() error = %v", err)
			}

			events := collectStreamEvents(stream.Events)
			if len(events) != 1 || events[0].Event != "done" {
				t.Fatalf("stream events = %+v, want a single done event", events)
			}

			if !equalStringSlices(manager.startSpec.Args, []string{"app-server", "--listen", "stdio://", "--trace"}) {
				t.Fatalf("process args = %v, want normalized codex CLI args without permission flags", manager.startSpec.Args)
			}
			if capturedThread.ApprovalPolicy != tc.wantThreadApproval {
				t.Fatalf("thread approval policy = %#v, want %q", capturedThread.ApprovalPolicy, tc.wantThreadApproval)
			}
			if capturedThread.ReasoningEffort == nil || *capturedThread.ReasoningEffort != "high" {
				t.Fatalf("thread reasoning effort = %+v, want high", capturedThread.ReasoningEffort)
			}
			if capturedThread.Sandbox != tc.wantThreadSandbox {
				t.Fatalf("thread sandbox = %q, want %q", capturedThread.Sandbox, tc.wantThreadSandbox)
			}
			if capturedTurn.ApprovalPolicy != tc.wantTurnApproval {
				t.Fatalf("turn approval policy = %#v, want %q", capturedTurn.ApprovalPolicy, tc.wantTurnApproval)
			}
			if got := capturedTurn.SandboxPolicy["type"]; got != tc.wantSandboxType {
				t.Fatalf("turn sandbox policy = %+v, want type %q", capturedTurn.SandboxPolicy, tc.wantSandboxType)
			}

			_, hasNetworkAccess := capturedTurn.SandboxPolicy["networkAccess"]
			if tc.wantNetworkAccess != hasNetworkAccess {
				t.Fatalf("turn sandbox policy networkAccess presence = %v, want %v in %+v", hasNetworkAccess, tc.wantNetworkAccess, capturedTurn.SandboxPolicy)
			}
			if tc.wantNetworkAccess {
				if value, ok := capturedTurn.SandboxPolicy["networkAccess"].(bool); !ok || !value {
					t.Fatalf("turn sandbox policy networkAccess = %#v, want true", capturedTurn.SandboxPolicy["networkAccess"])
				}
			}

			if !runtime.CloseSession(SessionID("conversation-1")) {
				t.Fatal("CloseSession() = false, want true")
			}
			if err := <-serverDone; err != nil {
				t.Fatalf("protocol server error = %v", err)
			}
		})
	}
}

type codexRuntimePermissionThreadStartParams struct {
	CWD              *string `json:"cwd,omitempty"`
	ApprovalPolicy   any     `json:"approvalPolicy,omitempty"`
	Sandbox          string  `json:"sandbox,omitempty"`
	ReasoningEffort  *string `json:"reasoningEffort,omitempty"`
	PersistedHistory bool    `json:"persistExtendedHistory,omitempty"`
}

type codexRuntimePermissionTurnStartParams struct {
	ThreadID       string         `json:"threadId"`
	ApprovalPolicy any            `json:"approvalPolicy,omitempty"`
	SandboxPolicy  map[string]any `json:"sandboxPolicy,omitempty"`
}

type codexRuntimePermissionJSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type codexRuntimePermissionTestProcess struct {
	stdinRead  *io.PipeReader
	stdinWrite *io.PipeWriter

	stdoutRead  *io.PipeReader
	stdoutWrite *io.PipeWriter

	stderrRead  *io.PipeReader
	stderrWrite *io.PipeWriter

	done     chan struct{}
	stopOnce sync.Once
}

func newCodexRuntimePermissionTestProcess() *codexRuntimePermissionTestProcess {
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()
	return &codexRuntimePermissionTestProcess{
		stdinRead:   stdinRead,
		stdinWrite:  stdinWrite,
		stdoutRead:  stdoutRead,
		stdoutWrite: stdoutWrite,
		stderrRead:  stderrRead,
		stderrWrite: stderrWrite,
		done:        make(chan struct{}),
	}
}

func (p *codexRuntimePermissionTestProcess) PID() int { return 4242 }

func (p *codexRuntimePermissionTestProcess) Stdin() io.WriteCloser { return p.stdinWrite }

func (p *codexRuntimePermissionTestProcess) Stdout() io.ReadCloser { return p.stdoutRead }

func (p *codexRuntimePermissionTestProcess) Stderr() io.ReadCloser { return p.stderrRead }

func (p *codexRuntimePermissionTestProcess) Wait() error {
	<-p.done
	return nil
}

func (p *codexRuntimePermissionTestProcess) Stop(context.Context) error {
	p.stopOnce.Do(func() {
		_ = p.stdinRead.Close()
		_ = p.stdinWrite.Close()
		_ = p.stdoutRead.Close()
		_ = p.stdoutWrite.Close()
		_ = p.stderrRead.Close()
		_ = p.stderrWrite.Close()
		close(p.done)
	})
	return nil
}

func runCodexRuntimePermissionServer(
	t *testing.T,
	process *codexRuntimePermissionTestProcess,
	threadParams *codexRuntimePermissionThreadStartParams,
	turnParams *codexRuntimePermissionTurnStartParams,
) error {
	t.Helper()

	decoder := json.NewDecoder(process.stdinRead)
	encoder := json.NewEncoder(process.stdoutWrite)

	var message codexRuntimePermissionJSONRPCMessage
	if err := decoder.Decode(&message); err != nil {
		return err
	}
	if message.Method != "initialize" {
		t.Fatalf("first method = %q, want initialize", message.Method)
	}
	if err := encoder.Encode(codexRuntimePermissionJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result: mustMarshalJSON(t, map[string]any{
			"userAgent":      "codex-cli/1.0.0",
			"platformFamily": "unix",
			"platformOs":     "linux",
		}),
	}); err != nil {
		return err
	}

	if err := decoder.Decode(&message); err != nil {
		return err
	}
	if message.Method != "initialized" {
		t.Fatalf("second method = %q, want initialized", message.Method)
	}

	if err := decoder.Decode(&message); err != nil {
		return err
	}
	if message.Method != "thread/start" {
		t.Fatalf("third method = %q, want thread/start", message.Method)
	}
	if err := json.Unmarshal(message.Params, threadParams); err != nil {
		return err
	}
	if err := encoder.Encode(codexRuntimePermissionJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result: mustMarshalJSON(t, map[string]any{
			"thread": map[string]any{
				"id": "thread-1",
			},
		}),
	}); err != nil {
		return err
	}

	if err := decoder.Decode(&message); err != nil {
		return err
	}
	if message.Method != "turn/start" {
		t.Fatalf("fourth method = %q, want turn/start", message.Method)
	}
	if err := json.Unmarshal(message.Params, turnParams); err != nil {
		return err
	}
	if err := encoder.Encode(codexRuntimePermissionJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result: mustMarshalJSON(t, map[string]any{
			"turn": map[string]any{
				"id":     "turn-1",
				"status": "inProgress",
			},
		}),
	}); err != nil {
		return err
	}

	return encoder.Encode(codexRuntimePermissionJSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "turn/completed",
		Params: mustMarshalJSON(t, map[string]any{
			"threadId": "thread-1",
			"turn": map[string]any{
				"id":     "turn-1",
				"status": "completed",
			},
		}),
	})
}

func codexRuntimePermissionProvider(
	profile catalogdomain.AgentProviderPermissionProfile,
) catalogdomain.AgentProvider {
	reasoning := catalogdomain.AgentProviderReasoningEffortHigh
	return catalogdomain.AgentProvider{
		Name:              "OpenAI Codex",
		AdapterType:       catalogdomain.AgentProviderAdapterTypeCodexAppServer,
		PermissionProfile: profile,
		CliCommand:        "codex",
		CliArgs:           []string{"app-server", "--listen", "stdio://", "--trace"},
		ModelName:         "gpt-5.4",
		ReasoningEffort:   &reasoning,
	}
}

func equalStringSlices(left []string, right []string) bool {
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
