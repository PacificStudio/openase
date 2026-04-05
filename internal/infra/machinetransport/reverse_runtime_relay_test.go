package machinetransport

import (
	"context"
	"errors"
	"strings"
	"testing"

	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/google/uuid"
)

func TestReverseRuntimeRelayRegisterReplacesExistingClientSessions(t *testing.T) {
	t.Parallel()

	registry := NewReverseRuntimeRelayRegistry()
	machineID := uuid.New()

	registry.Register(machineID, "session-a", func(context.Context, runtimecontract.Envelope) error { return nil })
	firstClient := registry.sessions["session-a"]
	if firstClient == nil {
		t.Fatal("expected first reverse runtime client to be registered")
	}

	managedSession := newRuntimeManagedClientSession(context.Background(), firstClient, func(error) {})
	firstClient.registerSession("process-1", managedSession)

	registry.Register(machineID, "session-b", func(context.Context, runtimecontract.Envelope) error { return nil })

	if _, err := registry.client(machineID); err != nil {
		t.Fatalf("expected replacement client to stay available, got %v", err)
	}
	if err := managedSession.Wait(); err == nil || !strings.Contains(err.Error(), "reverse runtime session replaced") {
		t.Fatalf("expected replaced reverse runtime session error, got %v", err)
	}
}

func TestReverseRuntimeRelayRemoveDisconnectsManagedSessions(t *testing.T) {
	t.Parallel()

	registry := NewReverseRuntimeRelayRegistry()
	machineID := uuid.New()

	registry.Register(machineID, "session-a", func(context.Context, runtimecontract.Envelope) error { return nil })
	client := registry.sessions["session-a"]
	if client == nil {
		t.Fatal("expected reverse runtime client to be registered")
	}

	managedSession := newRuntimeManagedClientSession(context.Background(), client, func(error) {})
	client.registerSession("process-1", managedSession)

	registry.Remove("session-a")

	if _, err := registry.client(machineID); err == nil {
		t.Fatal("expected removed reverse runtime session to become unavailable")
	}
	if err := managedSession.Wait(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled after reverse runtime disconnect, got %v", err)
	}
}
