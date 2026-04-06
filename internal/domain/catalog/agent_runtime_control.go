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
	if agent.RuntimeControlState == AgentRuntimeControlStateRetired {
		return "", fmt.Errorf("retired agent cannot be paused")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStateInterruptRequested {
		return "", fmt.Errorf("agent interrupt is already in progress")
	}
	if agent.Runtime == nil || agent.Runtime.CurrentRunID == nil || agent.Runtime.CurrentTicketID == nil {
		return "", fmt.Errorf("agent must have an active run before it can be paused")
	}
	if agent.Runtime.Status != AgentStatusClaimed && agent.Runtime.Status != AgentStatusRunning {
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

func ResolveInterruptRuntimeControlState(agent Agent) (AgentRuntimeControlState, error) {
	if agent.RuntimeControlState == AgentRuntimeControlStateRetired {
		return "", fmt.Errorf("retired agent cannot be interrupted")
	}
	if agent.Runtime == nil || agent.Runtime.CurrentRunID == nil || agent.Runtime.CurrentTicketID == nil {
		return "", fmt.Errorf("agent must have an active run before it can be interrupted")
	}
	if agent.Runtime.Status != AgentStatusClaimed && agent.Runtime.Status != AgentStatusRunning {
		return "", fmt.Errorf("agent must be claimed or running before it can be interrupted")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStateInterruptRequested {
		return "", fmt.Errorf("agent interrupt is already in progress")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStatePauseRequested {
		return "", fmt.Errorf("agent pause is already in progress")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStatePaused {
		return "", fmt.Errorf("agent is already paused")
	}

	return AgentRuntimeControlStateInterruptRequested, nil
}

func ResolveResumeRuntimeControlState(agent Agent) (AgentRuntimeControlState, error) {
	if agent.RuntimeControlState == AgentRuntimeControlStateRetired {
		return "", fmt.Errorf("retired agent cannot be resumed")
	}
	if agent.Runtime == nil || agent.Runtime.CurrentRunID == nil || agent.Runtime.CurrentTicketID == nil {
		return "", fmt.Errorf("agent must keep its active run before it can be resumed")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStateActive {
		return "", fmt.Errorf("agent runtime is already active")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStatePauseRequested {
		return "", fmt.Errorf("agent is still pausing; wait for the runtime to reach paused before resuming")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStateInterruptRequested {
		return "", fmt.Errorf("agent is still interrupting; wait for the runtime to settle before resuming")
	}
	if agent.Runtime.Status != AgentStatusClaimed && agent.Runtime.Status != AgentStatusRunning {
		return "", fmt.Errorf("paused agent must be claimed or running before it can resume")
	}

	return AgentRuntimeControlStateActive, nil
}

func ResolveRetireRuntimeControlState(agent Agent) (AgentRuntimeControlState, error) {
	if agent.Runtime != nil && agent.Runtime.CurrentRunID != nil {
		return "", fmt.Errorf("agent must not have an active run before it can be retired")
	}
	if agent.RuntimeControlState == AgentRuntimeControlStateRetired {
		return "", fmt.Errorf("agent is already retired")
	}
	return AgentRuntimeControlStateRetired, nil
}
