package catalog

import (
	"fmt"

	"github.com/google/uuid"
)

type UpdateAgentRuntimeControlState struct {
	ID                  uuid.UUID
	RuntimeControlState AgentRuntimeControlState
}

func ResolvePauseRuntimeControlState(agent Agent) (AgentRuntimeControlState, error) {
	if agent.CurrentTicketID == nil {
		return "", fmt.Errorf("agent must have an assigned ticket before it can be paused")
	}
	if agent.Status != AgentStatusClaimed && agent.Status != AgentStatusRunning {
		return "", fmt.Errorf("agent must be claimed or running before it can be paused")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStatePauseRequested {
		return "", fmt.Errorf("agent pause is already in progress")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStatePaused {
		return "", fmt.Errorf("agent is already paused")
	}
	return AgentRuntimeControlStatePauseRequested, nil
}

func ResolveResumeRuntimeControlState(agent Agent) (AgentRuntimeControlState, error) {
	if agent.CurrentTicketID == nil {
		return "", fmt.Errorf("agent must keep its assigned ticket before it can be resumed")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStateActive {
		return "", fmt.Errorf("agent runtime is already active")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStatePauseRequested {
		return "", fmt.Errorf("agent is still pausing; wait for the runtime to reach paused before resuming")
	}
	if agent.Status != AgentStatusClaimed && agent.Status != AgentStatusRunning {
		return "", fmt.Errorf("paused agent must be claimed or running before it can resume")
	}

	return AgentRuntimeControlStateActive, nil
}
