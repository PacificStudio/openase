package catalog

import (
	"fmt"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	"github.com/google/uuid"
)

type UpdateAgentRuntimeControlState struct {
	ID                  uuid.UUID
	RuntimeControlState entagent.RuntimeControlState
}

func ResolvePauseRuntimeControlState(agent Agent) (entagent.RuntimeControlState, error) {
	if agent.CurrentTicketID == nil {
		return "", fmt.Errorf("agent must have an assigned ticket before it can be paused")
	}
	if agent.Status != entagent.StatusClaimed && agent.Status != entagent.StatusRunning {
		return "", fmt.Errorf("agent must be claimed or running before it can be paused")
	}
	if agent.RuntimeControlState == entagent.RuntimeControlStatePauseRequested {
		return "", fmt.Errorf("agent pause is already in progress")
	}
	if agent.RuntimeControlState == entagent.RuntimeControlStatePaused {
		return "", fmt.Errorf("agent is already paused")
	}
	return entagent.RuntimeControlStatePauseRequested, nil
}

func ResolveResumeRuntimeControlState(agent Agent) (entagent.RuntimeControlState, error) {
	if agent.CurrentTicketID == nil {
		return "", fmt.Errorf("agent must keep its assigned ticket before it can be resumed")
	}
	if agent.RuntimeControlState == entagent.RuntimeControlStateActive {
		return "", fmt.Errorf("agent runtime is already active")
	}
	if agent.RuntimeControlState == entagent.RuntimeControlStatePauseRequested {
		return "", fmt.Errorf("agent is still pausing; wait for the runtime to reach paused before resuming")
	}
	if agent.Status != entagent.StatusClaimed && agent.Status != entagent.StatusRunning {
		return "", fmt.Errorf("paused agent must be claimed or running before it can resume")
	}

	return entagent.RuntimeControlStateActive, nil
}
