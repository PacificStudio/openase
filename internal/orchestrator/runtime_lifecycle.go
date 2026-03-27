package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var (
	agentLifecycleTopic    = provider.MustParseTopic("agent.events")
	activityLifecycleTopic = provider.MustParseTopic("activity.events")

	agentClaimedType    = provider.MustParseEventType("agent.claimed")
	agentLaunchingType  = provider.MustParseEventType("agent.launching")
	agentReadyType      = provider.MustParseEventType("agent.ready")
	agentHeartbeatType  = provider.MustParseEventType("agent.heartbeat")
	agentPausedType     = provider.MustParseEventType("agent.paused")
	agentFailedType     = provider.MustParseEventType("agent.failed")
	agentTerminatedType = provider.MustParseEventType("agent.terminated")
	agentOutputType     = provider.MustParseEventType(catalogdomain.AgentOutputEventType)
)

type agentLifecycleEnvelope struct {
	Agent agentLifecycleSnapshot `json:"agent"`
}

type agentLifecycleState struct {
	agent        *ent.Agent
	run          *ent.AgentRun
	runIsCurrent bool
}

type agentLifecycleSnapshot struct {
	ID                    string  `json:"id"`
	ProviderID            string  `json:"provider_id"`
	ProjectID             string  `json:"project_id"`
	Name                  string  `json:"name"`
	CurrentRunID          *string `json:"current_run_id,omitempty"`
	Status                string  `json:"status"`
	CurrentTicketID       *string `json:"current_ticket_id,omitempty"`
	SessionID             string  `json:"session_id"`
	RuntimePhase          string  `json:"runtime_phase"`
	RuntimeControlState   string  `json:"runtime_control_state"`
	RuntimeStartedAt      *string `json:"runtime_started_at,omitempty"`
	LastError             string  `json:"last_error"`
	TotalTokensUsed       int64   `json:"total_tokens_used"`
	TotalTicketsCompleted int     `json:"total_tickets_completed"`
	LastHeartbeatAt       *string `json:"last_heartbeat_at,omitempty"`
}

type activityLifecycleEnvelope struct {
	Event activityLifecycleSnapshot `json:"event"`
}

type activityLifecycleSnapshot struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	TicketID  *string        `json:"ticket_id,omitempty"`
	AgentID   *string        `json:"agent_id,omitempty"`
	EventType string         `json:"event_type"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
}

func publishAgentLifecycleEvent(
	ctx context.Context,
	client *ent.Client,
	events provider.EventProvider,
	eventType provider.EventType,
	state agentLifecycleState,
	message string,
	metadata map[string]any,
	publishedAt time.Time,
) error {
	if state.agent == nil {
		return fmt.Errorf("agent lifecycle event requires an agent")
	}

	if events != nil {
		event, err := provider.NewJSONEvent(
			agentLifecycleTopic,
			eventType,
			agentLifecycleEnvelope{Agent: mapAgentLifecycleSnapshot(state)},
			publishedAt,
		)
		if err != nil {
			return fmt.Errorf("construct %s event: %w", eventType, err)
		}
		if err := events.Publish(ctx, event); err != nil {
			return fmt.Errorf("publish %s event: %w", eventType, err)
		}
	}

	if client == nil {
		return nil
	}

	activityCreate := client.ActivityEvent.Create().
		SetProjectID(state.agent.ProjectID).
		SetAgentID(state.agent.ID).
		SetEventType(eventType.String()).
		SetMessage(message).
		SetMetadata(cloneLifecycleMetadata(metadata)).
		SetCreatedAt(publishedAt.UTC())
	if state.run != nil {
		activityCreate.SetTicketID(state.run.TicketID)
	}

	activityItem, err := activityCreate.
		Save(ctx)
	if err != nil {
		return fmt.Errorf("persist %s activity event: %w", eventType, err)
	}

	if events == nil {
		return nil
	}

	activityEvent, err := provider.NewJSONEvent(
		activityLifecycleTopic,
		eventType,
		activityLifecycleEnvelope{Event: mapActivityLifecycleSnapshot(activityItem)},
		publishedAt,
	)
	if err != nil {
		return fmt.Errorf("construct %s activity stream event: %w", eventType, err)
	}
	if err := events.Publish(ctx, activityEvent); err != nil {
		return fmt.Errorf("publish %s activity stream event: %w", eventType, err)
	}

	return nil
}

func publishAgentOutputEvent(
	ctx context.Context,
	client *ent.Client,
	events provider.EventProvider,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	message string,
	metadata map[string]any,
	publishedAt time.Time,
) error {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil
	}
	if client == nil {
		return fmt.Errorf("agent output event requires a client")
	}

	activityItem, err := client.ActivityEvent.Create().
		SetProjectID(projectID).
		SetAgentID(agentID).
		SetTicketID(ticketID).
		SetEventType(catalogdomain.AgentOutputEventType).
		SetMessage(trimmedMessage).
		SetMetadata(cloneLifecycleMetadata(metadata)).
		SetCreatedAt(publishedAt.UTC()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("persist %s activity event: %w", catalogdomain.AgentOutputEventType, err)
	}

	if events == nil {
		return nil
	}

	activityEvent, err := provider.NewJSONEvent(
		activityLifecycleTopic,
		agentOutputType,
		activityLifecycleEnvelope{Event: mapActivityLifecycleSnapshot(activityItem)},
		publishedAt,
	)
	if err != nil {
		return fmt.Errorf("construct %s activity stream event: %w", catalogdomain.AgentOutputEventType, err)
	}
	if err := events.Publish(ctx, activityEvent); err != nil {
		return fmt.Errorf("publish %s activity stream event: %w", catalogdomain.AgentOutputEventType, err)
	}

	return nil
}

func mapAgentLifecycleSnapshot(state agentLifecycleState) agentLifecycleSnapshot {
	status, runtimePhase := lifecycleAgentStatus(state), lifecycleAgentRuntimePhase(state)
	return agentLifecycleSnapshot{
		ID:                    state.agent.ID.String(),
		ProviderID:            state.agent.ProviderID.String(),
		ProjectID:             state.agent.ProjectID.String(),
		Name:                  state.agent.Name,
		CurrentRunID:          lifecycleCurrentRunID(state),
		Status:                status,
		CurrentTicketID:       lifecycleCurrentTicketID(state),
		SessionID:             lifecycleSessionID(state),
		RuntimePhase:          runtimePhase,
		RuntimeControlState:   state.agent.RuntimeControlState.String(),
		RuntimeStartedAt:      lifecycleRuntimeStartedAt(state),
		LastError:             lifecycleLastError(state),
		TotalTokensUsed:       state.agent.TotalTokensUsed,
		TotalTicketsCompleted: state.agent.TotalTicketsCompleted,
		LastHeartbeatAt:       lifecycleLastHeartbeatAt(state),
	}
}

func mapActivityLifecycleSnapshot(item *ent.ActivityEvent) activityLifecycleSnapshot {
	return activityLifecycleSnapshot{
		ID:        item.ID.String(),
		ProjectID: item.ProjectID.String(),
		TicketID:  uuidPointerToString(item.TicketID),
		AgentID:   uuidPointerToString(item.AgentID),
		EventType: item.EventType,
		Message:   item.Message,
		Metadata:  cloneLifecycleMetadata(item.Metadata),
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func clearRuntimeState(update *ent.AgentRunUpdate) *ent.AgentRunUpdate {
	return update.
		ClearSessionID().
		ClearRuntimeStartedAt().
		SetLastError("").
		ClearLastHeartbeatAt()
}

func clearRuntimeStateOne(update *ent.AgentRunUpdateOne) *ent.AgentRunUpdateOne {
	return update.
		ClearSessionID().
		ClearRuntimeStartedAt().
		SetLastError("").
		ClearLastHeartbeatAt()
}

func timePointerToRFC3339(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func uuidPointerToString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	formatted := value.String()
	return &formatted
}

func cloneLifecycleMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}

func lifecycleMessage(eventType provider.EventType, agentName string) string {
	switch eventType {
	case agentClaimedType:
		return fmt.Sprintf("Agent %s claimed a ticket and is waiting for runtime launch.", agentName)
	case agentLaunchingType:
		return fmt.Sprintf("Agent %s is launching a Codex session.", agentName)
	case agentReadyType:
		return fmt.Sprintf("Agent %s launched a Codex session and is ready.", agentName)
	case agentHeartbeatType:
		return fmt.Sprintf("Agent %s reported a runtime heartbeat.", agentName)
	case agentPausedType:
		return fmt.Sprintf("Agent %s paused its runtime session.", agentName)
	case agentFailedType:
		return fmt.Sprintf("Agent %s failed to launch or maintain its Codex session.", agentName)
	case agentTerminatedType:
		return fmt.Sprintf("Agent %s terminated its runtime session.", agentName)
	default:
		return fmt.Sprintf("Agent %s changed runtime state.", agentName)
	}
}

func runtimeEventMetadata(agentItem *ent.Agent) map[string]any {
	return runtimeEventMetadataForState(agentLifecycleState{agent: agentItem})
}

func runtimeEventMetadataForState(state agentLifecycleState) map[string]any {
	metadata := map[string]any{
		"status":                lifecycleAgentStatus(state),
		"runtime_phase":         lifecycleAgentRuntimePhase(state),
		"runtime_control_state": state.agent.RuntimeControlState.String(),
	}
	if runID := lifecycleRunID(state); runID != nil {
		metadata["run_id"] = *runID
	}
	if runID := lifecycleCurrentRunID(state); runID != nil {
		metadata["current_run_id"] = *runID
	}
	if ticketID := lifecycleCurrentTicketID(state); ticketID != nil {
		metadata["ticket_id"] = *ticketID
	}
	if sessionID := lifecycleSessionID(state); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	if lastError := lifecycleLastError(state); lastError != "" {
		metadata["last_error"] = lastError
	}
	return metadata
}

func loadAgentLifecycleState(ctx context.Context, client *ent.Client, agentID uuid.UUID, preferredRunID *uuid.UUID) (agentLifecycleState, error) {
	item, err := client.Agent.Query().
		Where(entagent.IDEQ(agentID)).
		Only(ctx)
	if err != nil {
		return agentLifecycleState{}, fmt.Errorf("load agent lifecycle state %s: %w", agentID, err)
	}

	state := agentLifecycleState{agent: item}
	currentRuns, err := client.AgentRun.Query().
		Where(
			entagentrun.AgentIDEQ(agentID),
			entagentrun.HasCurrentForTicket(),
		).
		Order(ent.Desc(entagentrun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return agentLifecycleState{}, fmt.Errorf("load current runs for agent %s: %w", agentID, err)
	}

	if preferredRunID != nil {
		for _, runItem := range currentRuns {
			if runItem.ID == *preferredRunID {
				state.run = runItem
				state.runIsCurrent = true
				return state, nil
			}
		}

		runItem, err := client.AgentRun.Query().
			Where(
				entagentrun.IDEQ(*preferredRunID),
				entagentrun.AgentIDEQ(agentID),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return agentLifecycleState{}, fmt.Errorf("load run %s for agent %s: %w", *preferredRunID, agentID, err)
		}
		if err == nil {
			state.run = runItem
			return state, nil
		}
	}

	if len(currentRuns) > 0 {
		state.run = currentRuns[0]
		state.runIsCurrent = true
	}

	return state, nil
}

func lifecycleRunID(state agentLifecycleState) *string {
	if state.run == nil {
		return nil
	}
	value := state.run.ID.String()
	return &value
}

func lifecycleCurrentRunID(state agentLifecycleState) *string {
	if state.run == nil || !state.runIsCurrent {
		return nil
	}
	value := state.run.ID.String()
	return &value
}

func lifecycleCurrentTicketID(state agentLifecycleState) *string {
	if state.run == nil || !state.runIsCurrent {
		return nil
	}
	value := state.run.TicketID.String()
	return &value
}

func lifecycleSessionID(state agentLifecycleState) string {
	if state.run == nil {
		return ""
	}
	return state.run.SessionID
}

func lifecycleRuntimeStartedAt(state agentLifecycleState) *string {
	if state.run == nil {
		return nil
	}
	return timePointerToRFC3339(state.run.RuntimeStartedAt)
}

func lifecycleLastError(state agentLifecycleState) string {
	if state.run == nil {
		return ""
	}
	return state.run.LastError
}

func lifecycleLastHeartbeatAt(state agentLifecycleState) *string {
	if state.run == nil {
		return nil
	}
	return timePointerToRFC3339(state.run.LastHeartbeatAt)
}

func lifecycleAgentStatus(state agentLifecycleState) string {
	switch {
	case state.run == nil:
		return "idle"
	case state.agent.RuntimeControlState == entagent.RuntimeControlStatePaused:
		return "paused"
	}

	switch state.run.Status {
	case entagentrun.StatusLaunching:
		return "claimed"
	case entagentrun.StatusReady, entagentrun.StatusExecuting:
		return "running"
	case entagentrun.StatusErrored:
		return "failed"
	case entagentrun.StatusTerminated:
		return "terminated"
	default:
		return "idle"
	}
}

func lifecycleAgentRuntimePhase(state agentLifecycleState) string {
	if state.run == nil {
		return "none"
	}

	switch state.run.Status {
	case entagentrun.StatusLaunching:
		return "launching"
	case entagentrun.StatusReady:
		return "ready"
	case entagentrun.StatusExecuting:
		return "executing"
	case entagentrun.StatusErrored:
		return "failed"
	default:
		return "none"
	}
}
