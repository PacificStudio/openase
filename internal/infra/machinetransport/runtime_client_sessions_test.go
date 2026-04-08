package machinetransport

import (
	"context"
	"encoding/base64"
	"io"
	"testing"
	"time"

	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
)

func TestRuntimeProtocolClientRegisterSessionDoesNotBlockOnPendingOutput(t *testing.T) {
	t.Parallel()

	client := newRuntimeProtocolClient(func(context.Context, runtimecontract.Envelope) error { return nil })
	session := newRuntimeManagedClientSession(context.Background(), client, func(error) {})
	sessionID := "session-fast-exit"

	output := []byte("pending-output")
	if err := client.handleSessionEvent(sessionID, runtimecontract.Envelope{
		Type:      runtimecontract.MessageTypeEvent,
		Operation: runtimecontract.OperationSessionOutput,
		Payload: mustRuntimePayload(runtimecontract.SessionOutputEvent{
			SessionID:  sessionID,
			Stream:     "stdout",
			DataBase64: base64.StdEncoding.EncodeToString(output),
		}),
	}); err != nil {
		t.Fatalf("handleSessionEvent(output) error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		client.registerSession(sessionID, session)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("registerSession blocked on pending output before any reader attached")
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	got := make([]byte, len(output))
	if _, err := io.ReadFull(stdout, got); err != nil {
		t.Fatalf("ReadFull(stdout) error = %v", err)
	}
	if string(got) != string(output) {
		t.Fatalf("stdout = %q, want %q", string(got), string(output))
	}

	if err := client.handleSessionEvent(sessionID, runtimecontract.Envelope{
		Type:      runtimecontract.MessageTypeEvent,
		Operation: runtimecontract.OperationSessionExit,
		Payload: mustRuntimePayload(runtimecontract.SessionExitEvent{
			SessionID: sessionID,
			ExitCode:  0,
		}),
	}); err != nil {
		t.Fatalf("handleSessionEvent(exit) error = %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}
