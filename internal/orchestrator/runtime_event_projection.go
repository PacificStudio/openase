package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentactivityinstance "github.com/BetterAndBetterII/openase/ent/agentactivityinstance"
	entagentrawevent "github.com/BetterAndBetterII/openase/ent/agentrawevent"
	entagenttranscriptentry "github.com/BetterAndBetterII/openase/ent/agenttranscriptentry"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

const (
	maxAgentActivityLiveTextBytes  = 64 * 1024
	maxAgentActivityFinalTextBytes = 64 * 1024
	maxAgentTranscriptSummaryBytes = 512
	maxAgentTranscriptBodyBytes    = 64 * 1024
)

type runtimeEventProjectionInput struct {
	ProjectID  uuid.UUID
	AgentID    uuid.UUID
	TicketID   uuid.UUID
	RunID      uuid.UUID
	Provider   string
	ObservedAt time.Time
	Event      agentEvent
}

type activityIdentity struct {
	Kind               string
	ActivityID         string
	IDSource           string
	IdentityConfidence string
	ParentActivityID   *string
	ThreadID           *string
	TurnID             *string
	Command            *string
	ToolName           *string
	Title              *string
	Metadata           map[string]any
}

type activityUpsertInput struct {
	Identity    activityIdentity
	Status      string
	LiveText    *activityTextUpdate
	FinalText   *string
	StartedAt   *time.Time
	CompletedAt *time.Time
}

type activityTextUpdate struct {
	Text string
	Mode activityTextMode
}

type activityTextMode string

const (
	activityTextModeReplace activityTextMode = "replace"
	activityTextModeAppend  activityTextMode = "append"
	activityTextModePrefix  activityTextMode = "prefix_merge"
)

type transcriptAppendInput struct {
	EntryKey     string
	EntryKind    string
	ActivityKind *string
	ActivityID   *string
	Title        *string
	Summary      *string
	BodyText     *string
	Command      *string
	ToolName     *string
	Metadata     map[string]any
	CreatedAt    time.Time
}

func (l *RuntimeLauncher) projectRuntimeEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	if l == nil || l.client == nil {
		return nil
	}

	if raw := input.Event.Raw; raw != nil {
		if err := l.appendAgentRawEvent(ctx, input, raw); err != nil {
			return err
		}
	}

	switch input.Event.Type {
	case agentEventTypeItemStarted:
		return l.projectItemStartedEvent(ctx, input)
	case agentEventTypeOutputProduced:
		return l.projectOutputEvent(ctx, input)
	case agentEventTypeToolCallRequested:
		return l.projectToolCallEvent(ctx, input)
	case agentEventTypeApprovalRequested:
		return l.projectApprovalEvent(ctx, input)
	case agentEventTypeUserInputRequested:
		return l.projectUserInputEvent(ctx, input)
	case agentEventTypeTaskStatus:
		return l.projectTaskStatusEvent(ctx, input)
	case agentEventTypeReasoningUpdated:
		return l.projectReasoningEvent(ctx, input)
	case agentEventTypeTurnDiffUpdated:
		return l.projectTurnDiffEvent(ctx, input)
	case agentEventTypeTurnStarted, agentEventTypeTurnCompleted, agentEventTypeTurnFailed:
		return l.projectTurnEvent(ctx, input)
	default:
		return nil
	}
}

func (l *RuntimeLauncher) appendAgentRawEvent(
	ctx context.Context,
	input runtimeEventProjectionInput,
	raw *agentRawProviderEvent,
) error {
	if raw == nil {
		return nil
	}
	dedupKey := sanitizeRawEventDBText(raw.DedupKey)
	if dedupKey == "" {
		return nil
	}
	exists, err := l.client.AgentRawEvent.Query().
		Where(
			entagentrawevent.AgentRunIDEQ(input.RunID),
			entagentrawevent.DedupKeyEQ(dedupKey),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check raw event dedup key for run %s: %w", input.RunID, err)
	}
	if exists {
		return nil
	}

	create := l.client.AgentRawEvent.Create().
		SetProjectID(input.ProjectID).
		SetTicketID(input.TicketID).
		SetAgentID(input.AgentID).
		SetAgentRunID(input.RunID).
		SetDedupKey(dedupKey).
		SetProvider(sanitizeRawEventDBText(input.Provider)).
		SetProviderEventKind(sanitizeRawEventDBText(raw.ProviderEventKind)).
		SetProviderEventSubtype(sanitizeRawEventDBText(raw.ProviderEventSubtype)).
		SetOccurredAt(input.ObservedAt.UTC()).
		SetPayload(cloneProjectionMap(raw.Payload)).
		SetTextExcerpt(sanitizeRawEventDBText(raw.TextExcerpt))
	if trimmed := sanitizeRawEventDBText(raw.ProviderEventID); trimmed != "" {
		create.SetProviderEventID(trimmed)
	}
	if trimmed := sanitizeRawEventDBText(raw.ThreadID); trimmed != "" {
		create.SetThreadID(trimmed)
	}
	if trimmed := sanitizeRawEventDBText(raw.TurnID); trimmed != "" {
		create.SetTurnID(trimmed)
	}
	if trimmed := sanitizeRawEventDBText(raw.ActivityHintID); trimmed != "" {
		create.SetActivityHintID(trimmed)
	}
	if _, err := create.Save(ctx); err != nil {
		return fmt.Errorf("append raw event for run %s: %w", input.RunID, err)
	}
	return nil
}

func sanitizeRawEventDBText(raw string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ToValidUTF8(raw, ""), "\x00", ""))
}

func (l *RuntimeLauncher) projectItemStartedEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	item := input.Event.Item
	if item == nil {
		return nil
	}
	identity := codexItemActivityIdentity(item)
	if identity.ActivityID == "" {
		return nil
	}
	startedAt := input.ObservedAt.UTC()
	if err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity:  identity,
		Status:    "started",
		StartedAt: &startedAt,
	}); err != nil {
		return err
	}

	if identity.Kind == catalogdomain.AgentActivityKindCommandExecution {
		return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
			EntryKey:     "command_started:" + identity.ActivityID,
			EntryKind:    catalogdomain.AgentTranscriptEntryKindCommandStarted,
			ActivityKind: &identity.Kind,
			ActivityID:   optionalStringPointer(identity.ActivityID),
			Title:        identity.Title,
			Summary:      identity.Command,
			Command:      identity.Command,
			Metadata:     cloneProjectionMap(identity.Metadata),
			CreatedAt:    input.ObservedAt.UTC(),
		})
	}
	return nil
}

func (l *RuntimeLauncher) projectOutputEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	output := input.Event.Output
	if output == nil {
		return nil
	}
	identity := outputActivityIdentity(input, output)
	if identity.ActivityID == "" {
		return nil
	}

	status := "in_progress"
	var startedAt *time.Time
	var completedAt *time.Time
	var finalText *string
	textMode := activityTextModeAppend
	if strings.EqualFold(strings.TrimSpace(input.Event.RawProviderSubtype()), "completed") || output.Snapshot {
		textMode = activityTextModeReplace
	}
	if started := input.ObservedAt.UTC(); !output.Snapshot {
		startedAt = &started
	}
	if output.Snapshot {
		completed := input.ObservedAt.UTC()
		completedAt = &completed
		status = "completed"
		if trimmed := strings.TrimSpace(output.Text); trimmed != "" {
			finalText = &trimmed
		}
	}
	if output.Snapshot {
		textMode = activityTextModeReplace
	}

	if err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity: identity,
		Status:   status,
		LiveText: &activityTextUpdate{
			Text: output.Text,
			Mode: textMode,
		},
		FinalText:   finalText,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
	}); err != nil {
		return err
	}

	if !output.Snapshot {
		return nil
	}
	switch identity.Kind {
	case catalogdomain.AgentActivityKindAssistantMessage:
		return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
			EntryKey:     "assistant_message:" + identity.ActivityID,
			EntryKind:    catalogdomain.AgentTranscriptEntryKindAssistantMessage,
			ActivityKind: &identity.Kind,
			ActivityID:   optionalStringPointer(identity.ActivityID),
			Title:        identity.Title,
			Summary:      summarizeTranscriptText(output.Text),
			BodyText:     trimmedPointer(output.Text),
			Metadata:     cloneProjectionMap(identity.Metadata),
			CreatedAt:    input.ObservedAt.UTC(),
		})
	case catalogdomain.AgentActivityKindCommandExecution:
		entryKind := catalogdomain.AgentTranscriptEntryKindCommandCompleted
		entryKey := "command_completed:" + identity.ActivityID
		return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
			EntryKey:     entryKey,
			EntryKind:    entryKind,
			ActivityKind: &identity.Kind,
			ActivityID:   optionalStringPointer(identity.ActivityID),
			Title:        identity.Title,
			Summary:      summarizeTranscriptText(output.Text),
			BodyText:     trimmedPointer(output.Text),
			Command:      identity.Command,
			Metadata:     cloneProjectionMap(identity.Metadata),
			CreatedAt:    input.ObservedAt.UTC(),
		})
	default:
		return nil
	}
}

func (l *RuntimeLauncher) projectToolCallEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	request := input.Event.ToolCall
	if request == nil {
		return nil
	}
	identity := toolCallActivityIdentity(request)
	if identity.ActivityID == "" {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(input.Provider), "claude") {
		identity.IDSource = catalogdomain.AgentActivityIDSourceClaudeToolUseID
		identity.IdentityConfidence = catalogdomain.AgentActivityIdentityConfidenceHigh
	}
	startedAt := input.ObservedAt.UTC()
	if err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity:  identity,
		Status:    "started",
		StartedAt: &startedAt,
	}); err != nil {
		return err
	}

	entryKind := catalogdomain.AgentTranscriptEntryKindToolCallStarted
	if identity.Kind == catalogdomain.AgentActivityKindCommandExecution {
		entryKind = catalogdomain.AgentTranscriptEntryKindCommandStarted
	}
	return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
		EntryKey:     entryKind + ":" + identity.ActivityID,
		EntryKind:    entryKind,
		ActivityKind: &identity.Kind,
		ActivityID:   optionalStringPointer(identity.ActivityID),
		Title:        identity.Title,
		Summary:      identity.Title,
		Command:      identity.Command,
		ToolName:     identity.ToolName,
		Metadata:     cloneProjectionMap(identity.Metadata),
		CreatedAt:    input.ObservedAt.UTC(),
	})
}

func (l *RuntimeLauncher) projectApprovalEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	request := input.Event.Approval
	if request == nil {
		return nil
	}
	identity := approvalActivityIdentity(request)
	if identity.ActivityID == "" {
		return nil
	}
	startedAt := input.ObservedAt.UTC()
	if err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity:  identity,
		Status:    "requested",
		StartedAt: &startedAt,
	}); err != nil {
		return err
	}
	return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
		EntryKey:     "approval_requested:" + identity.ActivityID,
		EntryKind:    catalogdomain.AgentTranscriptEntryKindApprovalRequest,
		ActivityKind: &identity.Kind,
		ActivityID:   optionalStringPointer(identity.ActivityID),
		Title:        identity.Title,
		Summary:      identity.Title,
		Metadata:     cloneProjectionMap(identity.Metadata),
		CreatedAt:    input.ObservedAt.UTC(),
	})
}

func (l *RuntimeLauncher) projectUserInputEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	request := input.Event.UserInput
	if request == nil {
		return nil
	}
	identity := userInputActivityIdentity(request)
	if identity.ActivityID == "" {
		return nil
	}
	startedAt := input.ObservedAt.UTC()
	if err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity:  identity,
		Status:    "requested",
		StartedAt: &startedAt,
	}); err != nil {
		return err
	}
	return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
		EntryKey:     "user_input_requested:" + identity.ActivityID,
		EntryKind:    catalogdomain.AgentTranscriptEntryKindApprovalRequest,
		ActivityKind: &identity.Kind,
		ActivityID:   optionalStringPointer(identity.ActivityID),
		Title:        identity.Title,
		Summary:      identity.Title,
		Metadata:     cloneProjectionMap(identity.Metadata),
		CreatedAt:    input.ObservedAt.UTC(),
	})
}

func (l *RuntimeLauncher) projectTaskStatusEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	status := input.Event.TaskStatus
	if status == nil {
		return nil
	}
	identity := claudeTaskActivityIdentity(status, input.Event.Raw)
	if identity.ActivityID == "" {
		return nil
	}

	var statusValue string
	var liveText *activityTextUpdate
	var finalText *string
	var completedAt *time.Time

	switch strings.TrimSpace(status.StatusType) {
	case catalogdomain.AgentTraceKindTaskStarted:
		statusValue = "started"
	case catalogdomain.AgentTraceKindError:
		statusValue = "failed"
		completed := input.ObservedAt.UTC()
		completedAt = &completed
		finalText = trimmedPointerValue(status.Text)
	default:
		statusValue = "in_progress"
	}

	if text, mode, ok := claudeTaskStatusTextUpdate(status); ok {
		liveText = &activityTextUpdate{Text: text, Mode: mode}
	}

	if input.Event.Raw != nil && strings.EqualFold(strings.TrimSpace(input.Event.Raw.ProviderEventKind), "user") {
		if text := firstNonEmptyString(
			statusTextFromPayload(status.Payload),
			strings.TrimSpace(status.Text),
		); text != "" {
			finalText = &text
			statusValue = "completed"
			completed := input.ObservedAt.UTC()
			completedAt = &completed
		}
	}

	if err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity:    identity,
		Status:      statusValue,
		LiveText:    liveText,
		FinalText:   finalText,
		CompletedAt: completedAt,
	}); err != nil {
		return err
	}

	if finalText == nil {
		return nil
	}

	if identity.Kind == catalogdomain.AgentActivityKindCommandExecution {
		return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
			EntryKey:     "command_completed:" + identity.ActivityID,
			EntryKind:    catalogdomain.AgentTranscriptEntryKindCommandCompleted,
			ActivityKind: &identity.Kind,
			ActivityID:   optionalStringPointer(identity.ActivityID),
			Title:        identity.Title,
			Summary:      summarizeTranscriptText(*finalText),
			BodyText:     finalText,
			Command:      identity.Command,
			Metadata:     cloneProjectionMap(identity.Metadata),
			CreatedAt:    input.ObservedAt.UTC(),
		})
	}

	return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
		EntryKey:     "tool_call_finished:" + identity.ActivityID,
		EntryKind:    catalogdomain.AgentTranscriptEntryKindToolCallFinished,
		ActivityKind: &identity.Kind,
		ActivityID:   optionalStringPointer(identity.ActivityID),
		Title:        identity.Title,
		Summary:      summarizeTranscriptText(*finalText),
		BodyText:     finalText,
		ToolName:     identity.ToolName,
		Metadata:     cloneProjectionMap(identity.Metadata),
		CreatedAt:    input.ObservedAt.UTC(),
	})
}

func (l *RuntimeLauncher) projectReasoningEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	reasoning := input.Event.Reasoning
	if reasoning == nil {
		return nil
	}
	identity := activityIdentity{
		Kind:               catalogdomain.AgentActivityKindReasoning,
		ActivityID:         strings.TrimSpace(reasoning.ItemID),
		IDSource:           catalogdomain.AgentActivityIDSourceCodexItemID,
		IdentityConfidence: catalogdomain.AgentActivityIdentityConfidenceHigh,
		ThreadID:           trimmedPointer(reasoning.ThreadID),
		TurnID:             trimmedPointer(reasoning.TurnID),
		Title:              trimmedPointer("Reasoning"),
		Metadata: map[string]any{
			"kind": reasoning.Kind,
		},
	}
	if identity.ActivityID == "" {
		return nil
	}
	err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity: identity,
		Status:   "in_progress",
		LiveText: &activityTextUpdate{Text: reasoning.Delta, Mode: activityTextModeAppend},
	})
	return err
}

func (l *RuntimeLauncher) projectTurnDiffEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	diff := input.Event.Diff
	if diff == nil || strings.TrimSpace(diff.Diff) == "" {
		return nil
	}
	turnID := strings.TrimSpace(diff.TurnID)
	if turnID == "" {
		return nil
	}
	kind := catalogdomain.AgentActivityKindTurn
	return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
		EntryKey:     "turn_diff:" + turnID + ":" + strings.ReplaceAll(input.ObservedAt.UTC().Format(time.RFC3339Nano), ":", "_"),
		EntryKind:    catalogdomain.AgentTranscriptEntryKindDiff,
		ActivityKind: &kind,
		ActivityID:   &turnID,
		Title:        trimmedPointer("Diff updated"),
		Summary:      summarizeTranscriptText(diff.Diff),
		BodyText:     trimmedPointer(diff.Diff),
		Metadata: map[string]any{
			"turn_id": turnID,
		},
		CreatedAt: input.ObservedAt.UTC(),
	})
}

func (l *RuntimeLauncher) projectTurnEvent(ctx context.Context, input runtimeEventProjectionInput) error {
	turn := input.Event.Turn
	if turn == nil {
		return nil
	}
	turnID := strings.TrimSpace(turn.TurnID)
	if turnID == "" {
		return nil
	}
	identity := activityIdentity{
		Kind:               catalogdomain.AgentActivityKindTurn,
		ActivityID:         turnID,
		IDSource:           catalogdomain.AgentActivityIDSourceSynthetic,
		IdentityConfidence: catalogdomain.AgentActivityIdentityConfidenceHigh,
		ThreadID:           trimmedPointer(turn.ThreadID),
		TurnID:             trimmedPointer(turnID),
		Title:              trimmedPointer("Turn"),
		Metadata: map[string]any{
			"status": turn.Status,
		},
	}
	status := strings.TrimSpace(turn.Status)
	if status == "" {
		status = "in_progress"
	}
	var completedAt *time.Time
	if input.Event.Type == agentEventTypeTurnCompleted || input.Event.Type == agentEventTypeTurnFailed {
		completed := input.ObservedAt.UTC()
		completedAt = &completed
	}
	if err := l.upsertAgentActivity(ctx, input, activityUpsertInput{
		Identity:    identity,
		Status:      status,
		CompletedAt: completedAt,
	}); err != nil {
		return err
	}

	if input.Event.Type == agentEventTypeTurnFailed {
		message := "Turn failed"
		if turn.Error != nil && strings.TrimSpace(turn.Error.Message) != "" {
			message = strings.TrimSpace(turn.Error.Message)
		}
		if err := l.failOpenActivitiesForTurn(ctx, input, turnID, message); err != nil {
			return err
		}
		return l.appendAgentTranscriptEntry(ctx, input, transcriptAppendInput{
			EntryKey:     "turn_error:" + turnID,
			EntryKind:    catalogdomain.AgentTranscriptEntryKindError,
			ActivityKind: &identity.Kind,
			ActivityID:   &turnID,
			Title:        trimmedPointer("Turn failed"),
			Summary:      summarizeTranscriptText(message),
			BodyText:     trimmedPointer(message),
			Metadata: map[string]any{
				"turn_id": turnID,
			},
			CreatedAt: input.ObservedAt.UTC(),
		})
	}
	return nil
}

func (l *RuntimeLauncher) upsertAgentActivity(
	ctx context.Context,
	input runtimeEventProjectionInput,
	update activityUpsertInput,
) error {
	identity := update.Identity
	if strings.TrimSpace(identity.Kind) == "" || strings.TrimSpace(identity.ActivityID) == "" {
		return nil
	}

	existing, err := l.client.AgentActivityInstance.Query().
		Where(
			entagentactivityinstance.AgentRunIDEQ(input.RunID),
			entagentactivityinstance.ActivityKindEQ(identity.Kind),
			entagentactivityinstance.ActivityIDEQ(identity.ActivityID),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("query activity %s/%s for run %s: %w", identity.Kind, identity.ActivityID, input.RunID, err)
	}

	if ent.IsNotFound(err) {
		create := l.client.AgentActivityInstance.Create().
			SetProjectID(input.ProjectID).
			SetTicketID(input.TicketID).
			SetAgentID(input.AgentID).
			SetAgentRunID(input.RunID).
			SetProvider(strings.TrimSpace(input.Provider)).
			SetActivityKind(identity.Kind).
			SetActivityID(identity.ActivityID).
			SetIDSource(identity.IDSource).
			SetIdentityConfidence(identity.IdentityConfidence).
			SetStatus(defaultIfEmpty(update.Status, "in_progress")).
			SetMetadata(cloneProjectionMap(identity.Metadata)).
			SetUpdatedAt(input.ObservedAt.UTC())
		if identity.ParentActivityID != nil {
			create.SetParentActivityID(*identity.ParentActivityID)
		}
		if identity.ThreadID != nil {
			create.SetThreadID(*identity.ThreadID)
		}
		if identity.TurnID != nil {
			create.SetTurnID(*identity.TurnID)
		}
		if identity.Command != nil {
			create.SetCommand(*identity.Command)
		}
		if identity.ToolName != nil {
			create.SetToolName(*identity.ToolName)
		}
		if identity.Title != nil {
			create.SetTitle(*identity.Title)
		}
		if update.LiveText != nil {
			liveText := truncateTranscriptBody(update.LiveText.Text)
			if liveText != "" {
				create.SetLiveText(liveText)
				create.SetLiveTextBytes(len([]byte(liveText)))
			}
		}
		if update.FinalText != nil {
			finalText := truncateFinalText(*update.FinalText)
			if finalText != "" {
				create.SetFinalText(finalText)
				create.SetFinalTextBytes(len([]byte(finalText)))
			}
		}
		if update.StartedAt != nil {
			create.SetStartedAt(update.StartedAt.UTC())
		}
		if update.CompletedAt != nil {
			create.SetCompletedAt(update.CompletedAt.UTC())
		}
		if _, saveErr := create.Save(ctx); saveErr != nil {
			return fmt.Errorf("create activity %s/%s for run %s: %w", identity.Kind, identity.ActivityID, input.RunID, saveErr)
		}
		return nil
	}

	builder := existing.Update().
		SetUpdatedAt(input.ObservedAt.UTC()).
		SetStatus(defaultIfEmpty(update.Status, existing.Status)).
		SetMetadata(mergeProjectionMaps(existing.Metadata, identity.Metadata))
	if identity.ParentActivityID != nil && strings.TrimSpace(stringOrEmpty(existing.ParentActivityID)) == "" {
		builder.SetParentActivityID(*identity.ParentActivityID)
	}
	if identity.ThreadID != nil && strings.TrimSpace(stringOrEmpty(existing.ThreadID)) == "" {
		builder.SetThreadID(*identity.ThreadID)
	}
	if identity.TurnID != nil && strings.TrimSpace(stringOrEmpty(existing.TurnID)) == "" {
		builder.SetTurnID(*identity.TurnID)
	}
	if identity.Command != nil && strings.TrimSpace(stringOrEmpty(existing.Command)) == "" {
		builder.SetCommand(*identity.Command)
	}
	if identity.ToolName != nil && strings.TrimSpace(stringOrEmpty(existing.ToolName)) == "" {
		builder.SetToolName(*identity.ToolName)
	}
	if identity.Title != nil && strings.TrimSpace(stringOrEmpty(existing.Title)) == "" {
		builder.SetTitle(*identity.Title)
	}
	if update.StartedAt != nil && existing.StartedAt == nil {
		builder.SetStartedAt(update.StartedAt.UTC())
	}
	if update.CompletedAt != nil {
		builder.SetCompletedAt(update.CompletedAt.UTC())
	}
	if update.LiveText != nil {
		nextLiveText := mergeActivityText(stringOrEmpty(existing.LiveText), update.LiveText.Text, update.LiveText.Mode)
		nextLiveText = truncateLiveText(nextLiveText)
		if nextLiveText != "" {
			builder.SetLiveText(nextLiveText)
			builder.SetLiveTextBytes(len([]byte(nextLiveText)))
		}
	}
	if update.FinalText != nil {
		nextFinalText := truncateFinalText(*update.FinalText)
		if nextFinalText != "" {
			builder.SetFinalText(nextFinalText)
			builder.SetFinalTextBytes(len([]byte(nextFinalText)))
		}
	}
	if _, saveErr := builder.Save(ctx); saveErr != nil {
		return fmt.Errorf("update activity %s/%s for run %s: %w", identity.Kind, identity.ActivityID, input.RunID, saveErr)
	}
	return nil
}

func (l *RuntimeLauncher) appendAgentTranscriptEntry(
	ctx context.Context,
	input runtimeEventProjectionInput,
	entry transcriptAppendInput,
) error {
	entryKey := strings.TrimSpace(entry.EntryKey)
	entryKind := strings.TrimSpace(entry.EntryKind)
	if entryKey == "" || entryKind == "" {
		return nil
	}
	exists, err := l.client.AgentTranscriptEntry.Query().
		Where(
			entagenttranscriptentry.AgentRunIDEQ(input.RunID),
			entagenttranscriptentry.EntryKeyEQ(entryKey),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check transcript entry %s for run %s: %w", entryKey, input.RunID, err)
	}
	if exists {
		return nil
	}

	create := l.client.AgentTranscriptEntry.Create().
		SetProjectID(input.ProjectID).
		SetTicketID(input.TicketID).
		SetAgentID(input.AgentID).
		SetAgentRunID(input.RunID).
		SetProvider(strings.TrimSpace(input.Provider)).
		SetEntryKey(entryKey).
		SetEntryKind(entryKind).
		SetMetadata(cloneProjectionMap(entry.Metadata)).
		SetCreatedAt(entry.CreatedAt.UTC())
	if entry.ActivityKind != nil {
		create.SetActivityKind(*entry.ActivityKind)
	}
	if entry.ActivityID != nil {
		create.SetActivityID(*entry.ActivityID)
	}
	if entry.Title != nil {
		create.SetTitle(*entry.Title)
	}
	if entry.Summary != nil {
		create.SetSummary(truncateTranscriptSummary(*entry.Summary))
	}
	if entry.BodyText != nil {
		create.SetBodyText(truncateTranscriptBody(*entry.BodyText))
	}
	if entry.Command != nil {
		create.SetCommand(*entry.Command)
	}
	if entry.ToolName != nil {
		create.SetToolName(*entry.ToolName)
	}
	if _, err := create.Save(ctx); err != nil {
		return fmt.Errorf("append transcript entry %s for run %s: %w", entryKey, input.RunID, err)
	}
	return nil
}

func (l *RuntimeLauncher) failOpenActivitiesForTurn(
	ctx context.Context,
	input runtimeEventProjectionInput,
	turnID string,
	message string,
) error {
	items, err := l.client.AgentActivityInstance.Query().
		Where(
			entagentactivityinstance.AgentRunIDEQ(input.RunID),
			entagentactivityinstance.TurnIDEQ(turnID),
			entagentactivityinstance.CompletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list open activities for failed turn %s: %w", turnID, err)
	}
	for _, item := range items {
		builder := item.Update().
			SetStatus("failed").
			SetUpdatedAt(input.ObservedAt.UTC()).
			SetCompletedAt(input.ObservedAt.UTC())
		if strings.TrimSpace(message) != "" && strings.TrimSpace(stringOrEmpty(item.FinalText)) == "" {
			finalText := truncateFinalText(message)
			builder.SetFinalText(finalText)
			builder.SetFinalTextBytes(len([]byte(finalText)))
		}
		if _, err := builder.Save(ctx); err != nil {
			return fmt.Errorf("mark activity %s/%s failed for run %s: %w", item.ActivityKind, item.ActivityID, input.RunID, err)
		}
	}
	return nil
}

func codexItemActivityIdentity(item *agentItemStartedEvent) activityIdentity {
	if item == nil {
		return activityIdentity{}
	}
	kind := catalogdomain.AgentActivityKindToolCall
	switch strings.TrimSpace(item.ItemType) {
	case "agentMessage":
		kind = catalogdomain.AgentActivityKindAssistantMessage
	case "commandExecution":
		kind = catalogdomain.AgentActivityKindCommandExecution
	}
	title := firstNonEmptyString(strings.TrimSpace(item.Command), strings.TrimSpace(item.Text), strings.TrimSpace(item.ItemType))
	metadata := map[string]any{
		"item_type": strings.TrimSpace(item.ItemType),
	}
	if phase := strings.TrimSpace(item.Phase); phase != "" {
		metadata["phase"] = phase
	}
	return activityIdentity{
		Kind:               kind,
		ActivityID:         strings.TrimSpace(item.ItemID),
		IDSource:           catalogdomain.AgentActivityIDSourceCodexItemID,
		IdentityConfidence: catalogdomain.AgentActivityIdentityConfidenceHigh,
		ThreadID:           trimmedPointer(item.ThreadID),
		TurnID:             trimmedPointer(item.TurnID),
		Command:            trimmedPointer(item.Command),
		Title:              trimmedPointer(title),
		Metadata:           metadata,
	}
}

func outputActivityIdentity(input runtimeEventProjectionInput, output *agentOutputEvent) activityIdentity {
	if output == nil {
		return activityIdentity{}
	}
	kind := catalogdomain.AgentActivityKindAssistantMessage
	if strings.TrimSpace(output.Stream) == "command" {
		kind = catalogdomain.AgentActivityKindCommandExecution
	}
	source := catalogdomain.AgentActivityIDSourceCodexItemID
	confidence := catalogdomain.AgentActivityIdentityConfidenceHigh
	if strings.EqualFold(strings.TrimSpace(input.Provider), "claude") {
		if strings.TrimSpace(output.Stream) == "command" {
			source = catalogdomain.AgentActivityIDSourceClaudeToolUseID
		} else {
			source = catalogdomain.AgentActivityIDSourceSynthetic
			confidence = catalogdomain.AgentActivityIdentityConfidenceMedium
		}
	}
	title := firstNonEmptyString(strings.TrimSpace(output.Command), summarizeAgentStepText(output.Text))
	metadata := map[string]any{
		"stream": strings.TrimSpace(output.Stream),
	}
	if phase := strings.TrimSpace(output.Phase); phase != "" {
		metadata["phase"] = phase
	}
	if output.Snapshot {
		metadata["snapshot"] = true
	}
	return activityIdentity{
		Kind:               kind,
		ActivityID:         strings.TrimSpace(output.ItemID),
		IDSource:           source,
		IdentityConfidence: confidence,
		ThreadID:           trimmedPointer(output.ThreadID),
		TurnID:             trimmedPointer(output.TurnID),
		Command:            trimmedPointer(output.Command),
		Title:              trimmedPointer(title),
		Metadata:           metadata,
	}
}

func toolCallActivityIdentity(request *agentToolCallRequest) activityIdentity {
	if request == nil {
		return activityIdentity{}
	}
	arguments := decodeRawJSON(request.Arguments)
	kind, command := deriveToolActivityKind(strings.TrimSpace(request.Tool), arguments)
	title := firstNonEmptyString(command, strings.TrimSpace(request.Tool))
	metadata := map[string]any{
		"tool":      strings.TrimSpace(request.Tool),
		"arguments": arguments,
	}
	return activityIdentity{
		Kind:               kind,
		ActivityID:         strings.TrimSpace(request.CallID),
		IDSource:           catalogdomain.AgentActivityIDSourceCodexCallID,
		IdentityConfidence: catalogdomain.AgentActivityIdentityConfidenceHigh,
		ThreadID:           trimmedPointer(request.ThreadID),
		TurnID:             trimmedPointer(request.TurnID),
		Command:            trimmedPointer(command),
		ToolName:           trimmedPointer(strings.TrimSpace(request.Tool)),
		Title:              trimmedPointer(title),
		Metadata:           metadata,
	}
}

func approvalActivityIdentity(request *agentApprovalRequest) activityIdentity {
	if request == nil {
		return activityIdentity{}
	}
	title := firstNonEmptyString(
		readProjectionString(request.Payload, "command"),
		readProjectionString(request.Payload, "kind"),
		"Approval requested",
	)
	return activityIdentity{
		Kind:               catalogdomain.AgentActivityKindApproval,
		ActivityID:         strings.TrimSpace(request.RequestID),
		IDSource:           catalogdomain.AgentActivityIDSourceSynthetic,
		IdentityConfidence: catalogdomain.AgentActivityIdentityConfidenceHigh,
		ThreadID:           trimmedPointer(request.ThreadID),
		TurnID:             trimmedPointer(request.TurnID),
		Title:              trimmedPointer(title),
		Metadata: map[string]any{
			"kind":    strings.TrimSpace(request.Kind),
			"payload": cloneProjectionMap(request.Payload),
			"options": mapApprovalOptionsForProjection(request.Options),
		},
	}
}

func userInputActivityIdentity(request *agentUserInputRequest) activityIdentity {
	if request == nil {
		return activityIdentity{}
	}
	title := firstNonEmptyString(firstInterruptQuestion(request.Payload), "User input requested")
	return activityIdentity{
		Kind:               catalogdomain.AgentActivityKindApproval,
		ActivityID:         strings.TrimSpace(request.RequestID),
		IDSource:           catalogdomain.AgentActivityIDSourceSynthetic,
		IdentityConfidence: catalogdomain.AgentActivityIdentityConfidenceHigh,
		ThreadID:           trimmedPointer(request.ThreadID),
		TurnID:             trimmedPointer(request.TurnID),
		Title:              trimmedPointer(title),
		Metadata:           cloneProjectionMap(request.Payload),
	}
}

func claudeTaskActivityIdentity(status *agentTaskStatusEvent, raw *agentRawProviderEvent) activityIdentity {
	if status == nil {
		return activityIdentity{}
	}
	payload := cloneProjectionMap(status.Payload)
	activityID, idSource, confidence, parent := resolveClaudeActivityID(payload, raw, status)
	if activityID == "" {
		return activityIdentity{}
	}
	kind := deriveClaudeActivityKind(payload)
	command := firstNonEmptyString(
		readProjectionString(payload, "command"),
		readProjectionString(payload, "cmd"),
	)
	toolName := firstNonEmptyString(
		readProjectionString(payload, "tool"),
		readProjectionString(payload, "last_tool_name"),
	)
	title := firstNonEmptyString(
		command,
		toolName,
		readProjectionString(payload, "description"),
		readProjectionString(payload, "summary"),
		readProjectionString(payload, "message"),
		strings.TrimSpace(status.Text),
	)
	return activityIdentity{
		Kind:               kind,
		ActivityID:         activityID,
		IDSource:           idSource,
		IdentityConfidence: confidence,
		ParentActivityID:   parent,
		ThreadID:           trimmedPointer(firstNonEmptyString(status.ThreadID, readProjectionString(payload, "thread_id", "session_id"))),
		TurnID:             trimmedPointer(firstNonEmptyString(status.TurnID, readProjectionString(payload, "turn_id"))),
		Command:            trimmedPointer(command),
		ToolName:           trimmedPointer(toolName),
		Title:              trimmedPointer(title),
		Metadata:           payload,
	}
}

func resolveClaudeActivityID(payload map[string]any, raw *agentRawProviderEvent, status *agentTaskStatusEvent) (activityID string, idSource string, confidence string, parentActivityID *string) {
	if toolUseID := readProjectionString(payload, "tool_use_id"); toolUseID != "" {
		parent := trimmedPointer(readProjectionString(payload, "parent_tool_use_id"))
		return toolUseID, catalogdomain.AgentActivityIDSourceClaudeToolUseID, catalogdomain.AgentActivityIdentityConfidenceHigh, parent
	}
	if parentToolUseID := readProjectionString(payload, "parent_tool_use_id"); parentToolUseID != "" {
		return parentToolUseID, catalogdomain.AgentActivityIDSourceClaudeParentToolUseID, catalogdomain.AgentActivityIdentityConfidenceMedium, nil
	}
	if taskID := readProjectionString(payload, "task_id"); taskID != "" {
		parent := trimmedPointer(readProjectionString(payload, "parent_tool_use_id"))
		return taskID, catalogdomain.AgentActivityIDSourceClaudeTaskID, catalogdomain.AgentActivityIdentityConfidenceMedium, parent
	}
	if raw != nil && strings.TrimSpace(raw.ProviderEventID) != "" {
		return strings.TrimSpace(raw.ProviderEventID), catalogdomain.AgentActivityIDSourceEventUUID, catalogdomain.AgentActivityIdentityConfidenceLow, nil
	}
	if status != nil && strings.TrimSpace(status.ItemID) != "" {
		return strings.TrimSpace(status.ItemID), catalogdomain.AgentActivityIDSourceSynthetic, catalogdomain.AgentActivityIdentityConfidenceLow, nil
	}
	return "", "", "", nil
}

func deriveClaudeActivityKind(payload map[string]any) string {
	toolName := firstNonEmptyString(readProjectionString(payload, "tool"), readProjectionString(payload, "last_tool_name"))
	kind, _ := deriveToolActivityKind(toolName, payload)
	if kind != catalogdomain.AgentActivityKindToolCall {
		return kind
	}
	stream := readProjectionString(payload, "stream")
	if stream == "command" || readProjectionString(payload, "command") != "" {
		return catalogdomain.AgentActivityKindCommandExecution
	}
	if readProjectionString(payload, "output_file") != "" {
		return catalogdomain.AgentActivityKindFileChange
	}
	return catalogdomain.AgentActivityKindToolCall
}

func deriveToolActivityKind(toolName string, arguments any) (kind string, command string) {
	normalized := strings.ToLower(strings.TrimSpace(toolName))
	command = readToolCommand(arguments)
	switch normalized {
	case "functions.exec_command", "bash", "terminal", "shell", "shell_script_runner":
		return catalogdomain.AgentActivityKindCommandExecution, command
	case "functions.apply_patch", "apply_patch":
		return catalogdomain.AgentActivityKindFileChange, command
	default:
		return catalogdomain.AgentActivityKindToolCall, command
	}
}

func claudeTaskStatusTextUpdate(status *agentTaskStatusEvent) (string, activityTextMode, bool) {
	if status == nil {
		return "", "", false
	}
	text := statusTextFromPayload(status.Payload)
	if text == "" {
		text = strings.TrimSpace(status.Text)
	}
	if text == "" {
		return "", "", false
	}
	snapshot := readProjectionBool(status.Payload, "snapshot")
	if snapshot {
		return text, activityTextModeReplace, true
	}
	return text, activityTextModePrefix, true
}

func statusTextFromPayload(payload map[string]any) string {
	return firstNonEmptyString(
		readProjectionString(payload, "text"),
		readProjectionString(payload, "summary"),
		readProjectionString(payload, "message"),
		readProjectionString(payload, "description"),
	)
}

func mapApprovalOptionsForProjection(options []agentApprovalOption) []map[string]any {
	if len(options) == 0 {
		return nil
	}
	items := make([]map[string]any, 0, len(options))
	for _, option := range options {
		record := map[string]any{}
		if trimmed := strings.TrimSpace(option.ID); trimmed != "" {
			record["id"] = trimmed
		}
		if trimmed := strings.TrimSpace(option.Label); trimmed != "" {
			record["label"] = trimmed
		}
		if trimmed := strings.TrimSpace(option.RawDecision); trimmed != "" {
			record["raw_decision"] = trimmed
		}
		if len(record) > 0 {
			items = append(items, record)
		}
	}
	return items
}

func readToolCommand(arguments any) string {
	record := projectionMap(arguments)
	if len(record) == 0 {
		return ""
	}
	return firstNonEmptyString(
		readProjectionString(record, "cmd"),
		readProjectionString(record, "command"),
	)
}

func projectionMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneProjectionMap(typed)
	case json.RawMessage:
		if len(typed) == 0 {
			return nil
		}
		var record map[string]any
		if err := json.Unmarshal(typed, &record); err != nil {
			return nil
		}
		return record
	default:
		return nil
	}
}

func readProjectionString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := record[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func readProjectionBool(record map[string]any, key string) bool {
	value, ok := record[key]
	if !ok {
		return false
	}
	flag, ok := value.(bool)
	return ok && flag
}

func mergeActivityText(existing string, next string, mode activityTextMode) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return strings.TrimSpace(existing)
	}
	switch mode {
	case activityTextModeReplace:
		return next
	case activityTextModePrefix:
		if existingTrimmed := strings.TrimSpace(existing); existingTrimmed != "" {
			if strings.HasPrefix(next, existingTrimmed) {
				return next
			}
			if strings.HasPrefix(existingTrimmed, next) {
				return existingTrimmed
			}
			return strings.TrimSpace(existingTrimmed + "\n" + next)
		}
		return next
	default:
		return strings.TrimSpace(existing + next)
	}
}

func cloneProjectionMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func mergeProjectionMaps(left map[string]any, right map[string]any) map[string]any {
	merged := cloneProjectionMap(left)
	for key, value := range right {
		merged[key] = value
	}
	return merged
}

func truncateLiveText(text string) string {
	return truncateTailUTF8(text, maxAgentActivityLiveTextBytes)
}

func truncateFinalText(text string) string {
	return truncateTailUTF8(text, maxAgentActivityFinalTextBytes)
}

func truncateTranscriptSummary(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if len([]byte(trimmed)) <= maxAgentTranscriptSummaryBytes {
		return trimmed
	}
	return strings.TrimSpace(truncateUTF8Bytes(trimmed, maxAgentTranscriptSummaryBytes-3)) + "..."
}

func truncateTranscriptBody(text string) string {
	return truncateTailUTF8(text, maxAgentTranscriptBodyBytes)
}

func truncateTailUTF8(text string, maxBytes int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxBytes <= 0 || len([]byte(text)) <= maxBytes {
		return text
	}
	prefix := "[truncated]\n"
	remaining := maxBytes - len([]byte(prefix))
	if remaining <= 0 {
		return truncateUTF8Bytes(text, maxBytes)
	}
	runes := []rune(text)
	for start := 0; start < len(runes); start++ {
		suffix := string(runes[start:])
		if len([]byte(suffix)) <= remaining {
			return prefix + suffix
		}
	}
	return truncateUTF8Bytes(text, maxBytes)
}

func summarizeTranscriptText(text string) *string {
	summary := summarizeAgentStepText(text)
	if summary == "" {
		return nil
	}
	return &summary
}

func optionalStringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func trimmedPointer(value string) *string {
	return optionalStringPointer(value)
}

func trimmedPointerValue(value string) *string {
	return optionalStringPointer(value)
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func defaultIfEmpty(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func (e agentEvent) RawProviderSubtype() string {
	if e.Raw == nil {
		return ""
	}
	return strings.TrimSpace(e.Raw.ProviderEventSubtype)
}
