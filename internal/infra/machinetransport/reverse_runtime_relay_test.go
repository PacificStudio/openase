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

	if _, err := registry.client(machineID); !errors.Is(err, ErrTransportUnavailable) {
		t.Fatalf("expected removed reverse runtime session to report transport unavailable, got %v", err)
	}
	if err := managedSession.Wait(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled after reverse runtime disconnect, got %v", err)
	}
}

func TestReverseRuntimeRelayRemoveStaleSessionKeepsReplacementClientConnected(t *testing.T) {
	t.Parallel()

	registry := NewReverseRuntimeRelayRegistry()
	machineID := uuid.New()

	registry.Register(machineID, "session-a", func(context.Context, runtimecontract.Envelope) error { return nil })
	firstClient := registry.sessions["session-a"]
	if firstClient == nil {
		t.Fatal("expected first reverse runtime client to be registered")
	}

	firstManagedSession := newRuntimeManagedClientSession(context.Background(), firstClient, func(error) {})
	firstClient.registerSession("process-old", firstManagedSession)

	registry.Register(machineID, "session-b", func(context.Context, runtimecontract.Envelope) error { return nil })

	replacementClient, err := registry.client(machineID)
	if err != nil {
		t.Fatalf("expected replacement reverse runtime client to stay available, got %v", err)
	}
	replacementManagedSession := newRuntimeManagedClientSession(context.Background(), replacementClient, func(error) {})
	replacementClient.registerSession("process-new", replacementManagedSession)

	registry.Remove("session-a")

	currentClient, err := registry.client(machineID)
	if err != nil {
		t.Fatalf("expected stale session removal to keep replacement client available, got %v", err)
	}
	if currentClient != replacementClient {
		t.Fatalf("expected replacement client to remain current, got current=%p replacement=%p", currentClient, replacementClient)
	}
	select {
	case <-replacementManagedSession.waitDone:
		t.Fatal("expected replacement managed session to remain active after removing stale session")
	default:
	}
	if err := firstManagedSession.Wait(); err == nil || !strings.Contains(err.Error(), "reverse runtime session replaced") {
		t.Fatalf("expected original managed session to fail with replacement error, got %v", err)
	}

	registry.Remove("session-b")
	if err := replacementManagedSession.Wait(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected replacement managed session to cancel only after removing current session, got %v", err)
	}
}
