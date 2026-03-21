package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
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
)

type agentLifecycleEnvelope struct {
	Agent agentLifecycleSnapshot `json:"agent"`
}

type agentLifecycleSnapshot struct {
	ID                    string   `json:"id"`
	ProviderID            string   `json:"provider_id"`
	ProjectID             string   `json:"project_id"`
	Name                  string   `json:"name"`
	Status                string   `json:"status"`
	CurrentTicketID       *string  `json:"current_ticket_id,omitempty"`
	SessionID             string   `json:"session_id"`
	RuntimePhase          string   `json:"runtime_phase"`
	RuntimeControlState   string   `json:"runtime_control_state"`
	RuntimeStartedAt      *string  `json:"runtime_started_at,omitempty"`
	LastError             string   `json:"last_error"`
	WorkspacePath         string   `json:"workspace_path"`
	Capabilities          []string `json:"capabilities"`
	TotalTokensUsed       int64    `json:"total_tokens_used"`
	TotalTicketsCompleted int      `json:"total_tickets_completed"`
	LastHeartbeatAt       *string  `json:"last_heartbeat_at,omitempty"`
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
	agentItem *ent.Agent,
	message string,
	metadata map[string]any,
	publishedAt time.Time,
) error {
	if agentItem == nil {
		return fmt.Errorf("agent lifecycle event requires an agent")
	}

	if events != nil {
		event, err := provider.NewJSONEvent(
			agentLifecycleTopic,
			eventType,
			agentLifecycleEnvelope{Agent: mapAgentLifecycleSnapshot(agentItem)},
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
		SetProjectID(agentItem.ProjectID).
		SetAgentID(agentItem.ID).
		SetEventType(eventType.String()).
		SetMessage(message).
		SetMetadata(cloneLifecycleMetadata(metadata)).
		SetCreatedAt(publishedAt.UTC())
	if agentItem.CurrentTicketID != nil {
		activityCreate.SetTicketID(*agentItem.CurrentTicketID)
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

func mapAgentLifecycleSnapshot(item *ent.Agent) agentLifecycleSnapshot {
	return agentLifecycleSnapshot{
		ID:                    item.ID.String(),
		ProviderID:            item.ProviderID.String(),
		ProjectID:             item.ProjectID.String(),
		Name:                  item.Name,
		Status:                item.Status.String(),
		CurrentTicketID:       uuidPointerToString(item.CurrentTicketID),
		SessionID:             item.SessionID,
		RuntimePhase:          item.RuntimePhase.String(),
		RuntimeControlState:   item.RuntimeControlState.String(),
		RuntimeStartedAt:      timePointerToRFC3339(item.RuntimeStartedAt),
		LastError:             item.LastError,
		WorkspacePath:         item.WorkspacePath,
		Capabilities:          append([]string(nil), item.Capabilities...),
		TotalTokensUsed:       item.TotalTokensUsed,
		TotalTicketsCompleted: item.TotalTicketsCompleted,
		LastHeartbeatAt:       timePointerToRFC3339(item.LastHeartbeatAt),
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

func clearRuntimeState(update *ent.AgentUpdate) *ent.AgentUpdate {
	return update.
		ClearSessionID().
		SetRuntimePhase(entagent.RuntimePhaseNone).
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
	metadata := map[string]any{
		"status":                agentItem.Status.String(),
		"runtime_phase":         agentItem.RuntimePhase.String(),
		"runtime_control_state": agentItem.RuntimeControlState.String(),
	}
	if agentItem.CurrentTicketID != nil {
		metadata["ticket_id"] = agentItem.CurrentTicketID.String()
	}
	if agentItem.SessionID != "" {
		metadata["session_id"] = agentItem.SessionID
	}
	if agentItem.LastError != "" {
		metadata["last_error"] = agentItem.LastError
	}
	return metadata
}

func loadAgentLifecycleState(ctx context.Context, client *ent.Client, agentID uuid.UUID) (*ent.Agent, error) {
	item, err := client.Agent.Query().
		Where(entagent.IDEQ(agentID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("load agent lifecycle state %s: %w", agentID, err)
	}

	return item, nil
}
