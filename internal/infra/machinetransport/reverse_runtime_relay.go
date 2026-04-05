package machinetransport

import (
	"context"
	"fmt"
	"strings"
	"sync"

	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/google/uuid"
)

var _ = logging.DeclareComponent("machine-transport-reverse-runtime-relay")

type ReverseRuntimeRelayRegistry struct {
	mu       sync.Mutex
	machines map[uuid.UUID]string
	sessions map[string]*runtimeProtocolClient
}

func NewReverseRuntimeRelayRegistry() *ReverseRuntimeRelayRegistry {
	return &ReverseRuntimeRelayRegistry{
		machines: map[uuid.UUID]string{},
		sessions: map[string]*runtimeProtocolClient{},
	}
}

func (r *ReverseRuntimeRelayRegistry) Register(
	machineID uuid.UUID,
	sessionID string,
	send runtimeEnvelopeSender,
) {
	if r == nil {
		return
	}
	client := newRuntimeProtocolClient(send)
	trimmedSessionID := strings.TrimSpace(sessionID)

	r.mu.Lock()
	if existingID, ok := r.machines[machineID]; ok {
		if existing := r.sessions[existingID]; existing != nil {
			existing.Close(fmt.Errorf("reverse runtime session replaced"))
		}
		delete(r.sessions, existingID)
	}
	r.machines[machineID] = trimmedSessionID
	r.sessions[trimmedSessionID] = client
	r.mu.Unlock()
}

func (r *ReverseRuntimeRelayRegistry) Remove(sessionID string) {
	if r == nil {
		return
	}
	trimmed := strings.TrimSpace(sessionID)
	r.mu.Lock()
	client := r.sessions[trimmed]
	delete(r.sessions, trimmed)
	for machineID, current := range r.machines {
		if current == trimmed {
			delete(r.machines, machineID)
		}
	}
	r.mu.Unlock()
	if client != nil {
		client.Close(context.Canceled)
	}
}

func (r *ReverseRuntimeRelayRegistry) Deliver(sessionID string, envelope runtimecontract.Envelope) error {
	if r == nil {
		return fmt.Errorf("reverse runtime relay registry unavailable")
	}
	r.mu.Lock()
	client := r.sessions[strings.TrimSpace(sessionID)]
	r.mu.Unlock()
	if client == nil {
		return nil
	}
	return client.HandleEnvelope(envelope)
}

func (r *ReverseRuntimeRelayRegistry) client(machineID uuid.UUID) (*runtimeProtocolClient, error) {
	if r == nil {
		return nil, fmt.Errorf("reverse runtime relay registry unavailable")
	}
	r.mu.Lock()
	sessionID, ok := r.machines[machineID]
	client := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok || client == nil {
		return nil, fmt.Errorf("%w: reverse websocket runtime session is not connected for machine %s", ErrTransportUnavailable, machineID)
	}
	return client, nil
}
