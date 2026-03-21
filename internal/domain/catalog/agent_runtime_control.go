package catalog

import (
	"fmt"
	"time"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	"github.com/google/uuid"
)

type AgentRuntimeControlAction string

const (
	AgentRuntimeControlPause  AgentRuntimeControlAction = "paused"
	AgentRuntimeControlResume AgentRuntimeControlAction = "resumed"
)

type AgentRuntimeControlResult struct {
	Agent       Agent
	Transition  AgentRuntimeControlAction
	RequestedAt time.Time
}

type UpdateAgentRuntimeState struct {
	ID               uuid.UUID
	Status           entagent.Status
	CurrentTicketID  *uuid.UUID
	SessionID        string
	RuntimePhase     entagent.RuntimePhase
	RuntimeStartedAt *time.Time
	LastError        string
	LastHeartbeatAt  *time.Time
}

func BuildPauseAgentRuntime(current Agent) (UpdateAgentRuntimeState, error) {
	if current.CurrentTicketID == nil {
		return UpdateAgentRuntimeState{}, fmt.Errorf("agent can only be paused while holding a ticket")
	}

	switch current.Status {
	case entagent.StatusClaimed, entagent.StatusRunning:
		return UpdateAgentRuntimeState{
			ID:              current.ID,
			Status:          entagent.StatusPaused,
			CurrentTicketID: cloneUUIDPointer(current.CurrentTicketID),
			RuntimePhase:    entagent.RuntimePhaseNone,
		}, nil
	case entagent.StatusPaused:
		return UpdateAgentRuntimeState{}, fmt.Errorf("agent is already paused")
	default:
		return UpdateAgentRuntimeState{}, fmt.Errorf("agent can only be paused from claimed or running")
	}
}

func BuildResumeAgentRuntime(current Agent) (UpdateAgentRuntimeState, error) {
	if current.Status != entagent.StatusPaused {
		return UpdateAgentRuntimeState{}, fmt.Errorf("agent can only be resumed from paused")
	}
	if current.CurrentTicketID == nil {
		return UpdateAgentRuntimeState{}, fmt.Errorf("paused agent is missing its claimed ticket")
	}

	return UpdateAgentRuntimeState{
		ID:              current.ID,
		Status:          entagent.StatusClaimed,
		CurrentTicketID: cloneUUIDPointer(current.CurrentTicketID),
		RuntimePhase:    entagent.RuntimePhaseNone,
	}, nil
}

func cloneUUIDPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
