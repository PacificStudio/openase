package orchestrator

import (
	"context"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func (s runtimeEventPersistenceSlice) recordAgentOutput(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	output *agentOutputEvent,
) error {
	return s.launcher.recordAgentOutput(ctx, projectID, agentID, ticketID, runID, adapterType, output)
}

func (s runtimeEventPersistenceSlice) recordAgentTaskStatus(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	status *agentTaskStatusEvent,
) error {
	return s.launcher.recordAgentTaskStatus(ctx, projectID, agentID, ticketID, runID, adapterType, status)
}

func (s runtimeEventPersistenceSlice) recordAgentToolCall(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	request *agentToolCallRequest,
) error {
	return s.launcher.recordAgentToolCall(ctx, projectID, agentID, ticketID, runID, adapterType, request)
}

func (s runtimeEventPersistenceSlice) recordAgentThreadStatus(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	status *agentThreadStatusEvent,
) error {
	return s.launcher.recordAgentThreadStatus(ctx, projectID, agentID, ticketID, runID, adapterType, status)
}

func (s runtimeEventPersistenceSlice) recordAgentTurnDiff(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	diff *agentTurnDiffEvent,
) error {
	return s.launcher.recordAgentTurnDiff(ctx, projectID, agentID, ticketID, runID, adapterType, diff)
}

func (s runtimeEventPersistenceSlice) recordAgentReasoning(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	reasoning *agentReasoningEvent,
) error {
	return s.launcher.recordAgentReasoning(ctx, projectID, agentID, ticketID, runID, adapterType, reasoning)
}

func (s runtimeEventPersistenceSlice) recordAgentStep(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	stepStatus string,
	summary string,
	sourceTraceEventID *uuid.UUID,
) error {
	return s.launcher.recordAgentStep(ctx, projectID, agentID, ticketID, runID, stepStatus, summary, sourceTraceEventID)
}

func (s runtimeEventPersistenceSlice) recordAgentApprovalRequest(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	request *agentApprovalRequest,
) error {
	return s.launcher.recordAgentApprovalRequest(ctx, projectID, agentID, ticketID, runID, adapterType, request)
}

func (s runtimeEventPersistenceSlice) recordAgentUserInputRequest(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	request *agentUserInputRequest,
) error {
	return s.launcher.recordAgentUserInputRequest(ctx, projectID, agentID, ticketID, runID, adapterType, request)
}

func (s runtimeEventPersistenceSlice) persistRuntimeSessionID(ctx context.Context, runID uuid.UUID, session agentSession) error {
	return s.launcher.persistRuntimeSessionID(ctx, runID, session)
}

func (s runtimeEventPersistenceSlice) touchHeartbeat(ctx context.Context, runID uuid.UUID) error {
	return s.launcher.touchHeartbeat(ctx, runID)
}

func (s runtimeEventPersistenceSlice) projectRuntimeEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	return s.launcher.projectRuntimeEvent(ctx, input)
}

func (s runtimeEventPersistenceSlice) recordTokenUsage(
	ctx context.Context,
	agentID uuid.UUID,
	runID uuid.UUID,
	ticketID uuid.UUID,
	usage *agentTokenUsageEvent,
	highWater *tokenUsageHighWater,
) error {
	return s.launcher.recordTokenUsage(ctx, agentID, runID, ticketID, usage, highWater)
}

func (s runtimeEventPersistenceSlice) recordProviderRateLimit(
	ctx context.Context,
	providerID uuid.UUID,
	rateLimit *provider.CLIRateLimit,
	observedAt time.Time,
) error {
	return s.launcher.recordProviderRateLimit(ctx, providerID, rateLimit, observedAt)
}
