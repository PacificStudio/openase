package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagentstepevent "github.com/BetterAndBetterII/openase/ent/agentstepevent"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var (
	agentLifecycleTopic    = provider.MustParseTopic("agent.events")
	activityLifecycleTopic = provider.MustParseTopic("activity.events")
	agentTraceTopic        = provider.MustParseTopic("agent.trace.events")
	agentStepTopic         = provider.MustParseTopic("agent.step.events")

	agentClaimedType     = provider.MustParseEventType("agent.claimed")
	agentLaunchingType   = provider.MustParseEventType("agent.launching")
	agentReadyType       = provider.MustParseEventType("agent.ready")
	agentExecutingType   = provider.MustParseEventType("agent.executing")
	agentHeartbeatType   = provider.MustParseEventType("agent.heartbeat")
	agentInterruptedType = provider.MustParseEventType("agent.interrupted")
	agentPausedType      = provider.MustParseEventType("agent.paused")
	agentFailedType      = provider.MustParseEventType("agent.failed")
	agentTerminatedType  = provider.MustParseEventType("agent.terminated")
	agentTraceType       = provider.MustParseEventType("agent.trace")
	agentOutputType      = provider.MustParseEventType(catalogdomain.AgentOutputEventType)
	agentStepType        = provider.MustParseEventType(catalogdomain.AgentStepEventType)
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

type agentTraceEnvelope struct {
	Entry agentTraceSnapshot `json:"entry"`
}

type agentTraceSnapshot struct {
	ID         string         `json:"id"`
	ProjectID  string         `json:"project_id"`
	TicketID   string         `json:"ticket_id"`
	AgentID    string         `json:"agent_id"`
	AgentRunID string         `json:"agent_run_id"`
	Sequence   int64          `json:"sequence"`
	Provider   string         `json:"provider"`
	Kind       string         `json:"kind"`
	Stream     string         `json:"stream"`
	Output     string         `json:"output"`
	Payload    map[string]any `json:"payload"`
	CreatedAt  string         `json:"created_at"`
}

type agentStepEnvelope struct {
	Entry agentStepSnapshot `json:"entry"`
}

type agentStepSnapshot struct {
	ID                 string  `json:"id"`
	ProjectID          string  `json:"project_id"`
	TicketID           string  `json:"ticket_id"`
	AgentID            string  `json:"agent_id"`
	AgentRunID         string  `json:"agent_run_id"`
	StepStatus         string  `json:"step_status"`
	Summary            string  `json:"summary"`
	SourceTraceEventID *string `json:"source_trace_event_id,omitempty"`
	CreatedAt          string  `json:"created_at"`
}

type agentTraceEventInput struct {
	ProjectID   uuid.UUID
	AgentID     uuid.UUID
	TicketID    uuid.UUID
	AgentRunID  uuid.UUID
	Provider    string
	Kind        string
	Stream      string
	Text        string
	Payload     map[string]any
	EventType   provider.EventType
	PublishedAt time.Time
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

	var activityType activityevent.Type
	activityAllowed := false
	if parsedType, err := activityevent.ParseRawType(eventType.String()); err == nil {
		activityType = parsedType
		activityAllowed = true
	}

	if client != nil && activityAllowed {
		input := activitysvc.RecordInput{
			ProjectID: state.agent.ProjectID,
			AgentID:   &state.agent.ID,
			EventType: activityType,
			Message:   message,
			Metadata:  cloneLifecycleMetadata(metadata),
			CreatedAt: publishedAt.UTC(),
		}
		if state.run != nil {
			input.TicketID = &state.run.TicketID
		}
		if _, err := activitysvc.NewEmitter(activitysvc.EntRecorder{Client: client}, events).Emit(ctx, input); err != nil {
			return fmt.Errorf("emit %s activity event: %w", eventType, err)
		}
	}

	if events == nil {
		return nil
	}

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

	return nil
}

func publishAgentTraceEvent(
	ctx context.Context,
	client *ent.Client,
	events provider.EventProvider,
	input agentTraceEventInput,
) (*ent.AgentTraceEvent, error) {
	if client == nil {
		return nil, fmt.Errorf("agent trace event requires a client")
	}

	nextSequence, err := nextAgentTraceSequence(ctx, client, input.AgentRunID)
	if err != nil {
		return nil, err
	}

	create := client.AgentTraceEvent.Create().
		SetProjectID(input.ProjectID).
		SetTicketID(input.TicketID).
		SetAgentID(input.AgentID).
		SetAgentRunID(input.AgentRunID).
		SetSequence(nextSequence).
		SetProvider(strings.TrimSpace(input.Provider)).
		SetKind(strings.TrimSpace(input.Kind)).
		SetStream(strings.TrimSpace(input.Stream)).
		SetPayload(cloneLifecycleMetadata(input.Payload)).
		SetCreatedAt(input.PublishedAt.UTC())
	if trimmedText := strings.TrimSpace(input.Text); trimmedText != "" {
		create.SetText(trimmedText)
	}

	traceItem, err := create.
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("persist agent trace event: %w", err)
	}

	if events == nil {
		return traceItem, nil
	}

	eventType := input.EventType
	if eventType == "" {
		eventType = agentTraceType
	}
	traceEvent, err := provider.NewJSONEvent(
		agentTraceTopic,
		eventType,
		agentTraceEnvelope{Entry: mapAgentTraceSnapshot(traceItem)},
		input.PublishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("construct agent trace stream event: %w", err)
	}
	if err := events.Publish(ctx, traceEvent); err != nil {
		return nil, fmt.Errorf("publish agent trace stream event: %w", err)
	}

	return traceItem, nil
}

func publishAgentStepEvent(
	ctx context.Context,
	client *ent.Client,
	events provider.EventProvider,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	agentRunID uuid.UUID,
	stepStatus string,
	summary string,
	sourceTraceEventID *uuid.UUID,
	publishedAt time.Time,
) error {
	if client == nil {
		return fmt.Errorf("agent step event requires a client")
	}

	normalizedStatus := normalizeAgentStepStatus(stepStatus)
	if normalizedStatus == "" {
		return nil
	}
	normalizedSummary := normalizeAgentStepSummary(summary)
	lastStepEvent, err := client.AgentStepEvent.Query().
		Where(entagentstepevent.AgentRunIDEQ(agentRunID)).
		Order(ent.Desc(entagentstepevent.FieldCreatedAt), ent.Desc(entagentstepevent.FieldID)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("load last step event for run %s: %w", agentRunID, err)
	}
	if err == nil &&
		lastStepEvent.StepStatus == normalizedStatus &&
		lastStepEvent.Summary == normalizedSummary &&
		uuidPointersEqual(lastStepEvent.SourceTraceEventID, sourceTraceEventID) {
		return nil
	}

	update := client.AgentRun.UpdateOneID(agentRunID).
		SetCurrentStepStatus(normalizedStatus).
		SetCurrentStepChangedAt(publishedAt.UTC())
	if normalizedSummary == "" {
		update.ClearCurrentStepSummary()
	} else {
		update.SetCurrentStepSummary(normalizedSummary)
	}
	if _, err := update.Save(ctx); err != nil {
		return fmt.Errorf("update run %s current step snapshot: %w", agentRunID, err)
	}

	create := client.AgentStepEvent.Create().
		SetProjectID(projectID).
		SetTicketID(ticketID).
		SetAgentID(agentID).
		SetAgentRunID(agentRunID).
		SetStepStatus(normalizedStatus).
		SetCreatedAt(publishedAt.UTC())
	if normalizedSummary != "" {
		create.SetSummary(normalizedSummary)
	}
	if sourceTraceEventID != nil {
		create.SetSourceTraceEventID(*sourceTraceEventID)
	}

	stepItem, err := create.Save(ctx)
	if err != nil {
		return fmt.Errorf("persist agent step event: %w", err)
	}

	if events == nil {
		return nil
	}

	stepEvent, err := provider.NewJSONEvent(
		agentStepTopic,
		agentStepType,
		agentStepEnvelope{Entry: mapAgentStepSnapshot(stepItem)},
		publishedAt,
	)
	if err != nil {
		return fmt.Errorf("construct agent step stream event: %w", err)
	}
	if err := events.Publish(ctx, stepEvent); err != nil {
		return fmt.Errorf("publish agent step stream event: %w", err)
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

func mapAgentTraceSnapshot(item *ent.AgentTraceEvent) agentTraceSnapshot {
	return agentTraceSnapshot{
		ID:         item.ID.String(),
		ProjectID:  item.ProjectID.String(),
		TicketID:   item.TicketID.String(),
		AgentID:    item.AgentID.String(),
		AgentRunID: item.AgentRunID.String(),
		Sequence:   item.Sequence,
		Provider:   item.Provider,
		Kind:       item.Kind,
		Stream:     item.Stream,
		Output:     item.Text,
		Payload:    cloneLifecycleMetadata(item.Payload),
		CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func mapAgentStepSnapshot(item *ent.AgentStepEvent) agentStepSnapshot {
	return agentStepSnapshot{
		ID:                 item.ID.String(),
		ProjectID:          item.ProjectID.String(),
		TicketID:           item.TicketID.String(),
		AgentID:            item.AgentID.String(),
		AgentRunID:         item.AgentRunID.String(),
		StepStatus:         item.StepStatus,
		Summary:            item.Summary,
		SourceTraceEventID: uuidPointerToString(item.SourceTraceEventID),
		CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339),
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

func uuidPointersEqual(left *uuid.UUID, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func nextAgentTraceSequence(ctx context.Context, client *ent.Client, agentRunID uuid.UUID) (int64, error) {
	lastItem, err := client.AgentTraceEvent.Query().
		Where(entagenttraceevent.AgentRunID(agentRunID)).
		Order(ent.Desc(entagenttraceevent.FieldSequence)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("load last trace sequence for run %s: %w", agentRunID, err)
	}

	return lastItem.Sequence + 1, nil
}

func normalizeAgentStepStatus(raw string) string {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

func normalizeAgentStepSummary(raw string) string {
	const maxSummaryBytes = 240
	const ellipsis = "..."

	trimmed := strings.TrimSpace(strings.ToValidUTF8(raw, ""))
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= maxSummaryBytes {
		return trimmed
	}

	limit := maxSummaryBytes - len(ellipsis)
	if limit <= 0 {
		return ellipsis
	}

	truncated := truncateUTF8Bytes(trimmed, limit)
	if truncated == "" {
		return ellipsis
	}
	return strings.TrimSpace(truncated) + ellipsis
}

func truncateUTF8Bytes(raw string, maxBytes int) string {
	if maxBytes <= 0 || raw == "" {
		return ""
	}
	if len(raw) <= maxBytes {
		return raw
	}

	lastBoundary := 0
	for index := range raw {
		if index > maxBytes {
			break
		}
		lastBoundary = index
	}
	if lastBoundary == 0 {
		_, width := utf8.DecodeRuneInString(raw)
		if width <= 0 || width > maxBytes {
			return ""
		}
		return raw[:width]
	}
	return raw[:lastBoundary]
}

func lifecycleMessage(eventType provider.EventType, agentName string) string {
	switch eventType {
	case agentClaimedType:
		return fmt.Sprintf("Agent %s claimed a ticket and is waiting for runtime launch.", agentName)
	case agentLaunchingType:
		return fmt.Sprintf("Agent %s is launching a Codex session.", agentName)
	case agentReadyType:
		return fmt.Sprintf("Agent %s launched a Codex session and is ready.", agentName)
	case agentExecutingType:
		return fmt.Sprintf("Agent %s is executing work in its Codex session.", agentName)
	case agentHeartbeatType:
		return fmt.Sprintf("Agent %s reported a runtime heartbeat.", agentName)
	case agentInterruptedType:
		return fmt.Sprintf("Agent %s interrupted its runtime session on operator request.", agentName)
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
	case entagentrun.StatusInterrupted:
		return "interrupted"
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
	case entagentrun.StatusInterrupted:
		return "none"
	default:
		return "none"
	}
}
