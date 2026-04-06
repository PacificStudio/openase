package chatconversation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entchatconversation "github.com/BetterAndBetterII/openase/ent/chatconversation"
	entchatentry "github.com/BetterAndBetterII/openase/ent/chatentry"
	entchatpendinginterrupt "github.com/BetterAndBetterII/openase/ent/chatpendinginterrupt"
	entchatturn "github.com/BetterAndBetterII/openase/ent/chatturn"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entprojectconversationprincipal "github.com/BetterAndBetterII/openase/ent/projectconversationprincipal"
	entprojectconversationrun "github.com/BetterAndBetterII/openase/ent/projectconversationrun"
	entprojectconversationtraceevent "github.com/BetterAndBetterII/openase/ent/projectconversationtraceevent"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/google/uuid"
)

var (
	ErrNotFound          = domain.ErrNotFound
	ErrConflict          = domain.ErrConflict
	ErrInvalidInput      = domain.ErrInvalidInput
	ErrTurnAlreadyActive = domain.ErrTurnAlreadyActive
	errRepositoryNil     = fmt.Errorf("chat conversation repository unavailable")
)

type Repository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) CreateConversation(ctx context.Context, input domain.CreateConversation) (domain.Conversation, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("start create chat conversation transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	item, err := tx.ChatConversation.Create().
		SetProjectID(input.ProjectID).
		SetUserID(strings.TrimSpace(input.UserID)).
		SetSource(string(input.Source)).
		SetProviderID(input.ProviderID).
		SetStatus(string(domain.ConversationStatusActive)).
		Save(ctx)
	if err != nil {
		return domain.Conversation{}, mapWriteError("create chat conversation", err)
	}
	if _, err := tx.ProjectConversationPrincipal.Create().
		SetID(item.ID).
		SetConversationID(item.ID).
		SetProjectID(item.ProjectID).
		SetProviderID(item.ProviderID).
		SetName(defaultProjectConversationPrincipalName(item.ID)).
		SetStatus(entprojectconversationprincipal.StatusActive).
		SetRuntimeState(entprojectconversationprincipal.RuntimeStateInactive).
		Save(ctx); err != nil {
		return domain.Conversation{}, mapWriteError("create project conversation principal", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Conversation{}, fmt.Errorf("commit create chat conversation: %w", err)
	}

	return mapConversation(item), nil
}

func (r *Repository) EnsurePrincipal(ctx context.Context, input domain.EnsurePrincipalInput) (domain.ProjectConversationPrincipal, error) {
	item, err := r.client.ProjectConversationPrincipal.Query().
		Where(entprojectconversationprincipal.ConversationIDEQ(input.ConversationID)).
		Only(ctx)
	switch {
	case err == nil:
		return mapPrincipal(item), nil
	case !ent.IsNotFound(err):
		return domain.ProjectConversationPrincipal{}, fmt.Errorf("get project conversation principal: %w", err)
	}

	item, err = r.client.ProjectConversationPrincipal.Create().
		SetID(input.ConversationID).
		SetConversationID(input.ConversationID).
		SetProjectID(input.ProjectID).
		SetProviderID(input.ProviderID).
		SetName(strings.TrimSpace(input.Name)).
		SetStatus(entprojectconversationprincipal.StatusActive).
		SetRuntimeState(entprojectconversationprincipal.RuntimeStateInactive).
		Save(ctx)
	if err != nil {
		return domain.ProjectConversationPrincipal{}, mapWriteError("create project conversation principal", err)
	}
	return mapPrincipal(item), nil
}

func (r *Repository) GetPrincipal(ctx context.Context, conversationID uuid.UUID) (domain.ProjectConversationPrincipal, error) {
	item, err := r.client.ProjectConversationPrincipal.Query().
		Where(entprojectconversationprincipal.ConversationIDEQ(conversationID)).
		Only(ctx)
	if err != nil {
		return domain.ProjectConversationPrincipal{}, mapReadError("get project conversation principal", err)
	}
	return mapPrincipal(item), nil
}

func (r *Repository) UpdatePrincipalRuntime(ctx context.Context, input domain.UpdatePrincipalRuntimeInput) (domain.ProjectConversationPrincipal, error) {
	builder := r.client.ProjectConversationPrincipal.UpdateOneID(input.PrincipalID).
		SetRuntimeState(entprojectconversationprincipal.RuntimeState(input.RuntimeState))
	if input.CurrentSessionID != nil {
		if trimmed := strings.TrimSpace(*input.CurrentSessionID); trimmed != "" {
			builder.SetCurrentSessionID(trimmed)
		} else {
			builder.ClearCurrentSessionID()
		}
	}
	if input.CurrentWorkspacePath != nil {
		if trimmed := strings.TrimSpace(*input.CurrentWorkspacePath); trimmed != "" {
			builder.SetCurrentWorkspacePath(trimmed)
		} else {
			builder.ClearCurrentWorkspacePath()
		}
	}
	if input.CurrentRunID != nil {
		if *input.CurrentRunID != uuid.Nil {
			builder.SetCurrentRunID(*input.CurrentRunID)
		} else {
			builder.ClearCurrentRunID()
		}
	}
	if input.LastHeartbeatAt != nil {
		builder.SetLastHeartbeatAt(*input.LastHeartbeatAt)
	}
	if input.CurrentStepStatus != nil {
		if trimmed := strings.TrimSpace(*input.CurrentStepStatus); trimmed != "" {
			builder.SetCurrentStepStatus(trimmed)
		} else {
			builder.ClearCurrentStepStatus()
		}
	}
	if input.CurrentStepSummary != nil {
		if trimmed := strings.TrimSpace(*input.CurrentStepSummary); trimmed != "" {
			builder.SetCurrentStepSummary(trimmed)
		} else {
			builder.ClearCurrentStepSummary()
		}
	}
	if input.CurrentStepChangedAt != nil {
		builder.SetCurrentStepChangedAt(*input.CurrentStepChangedAt)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectConversationPrincipal{}, mapWriteError("update project conversation principal runtime", err)
	}
	return mapPrincipal(item), nil
}

func (r *Repository) ClosePrincipal(ctx context.Context, input domain.ClosePrincipalInput) (domain.ProjectConversationPrincipal, error) {
	item, err := r.client.ProjectConversationPrincipal.UpdateOneID(input.PrincipalID).
		SetRuntimeState(entprojectconversationprincipal.RuntimeStateInactive).
		ClearCurrentSessionID().
		ClearCurrentWorkspacePath().
		ClearCurrentRunID().
		ClearCurrentStepStatus().
		ClearCurrentStepSummary().
		ClearCurrentStepChangedAt().
		Save(ctx)
	if err != nil {
		return domain.ProjectConversationPrincipal{}, mapWriteError("close project conversation principal", err)
	}
	return mapPrincipal(item), nil
}

func (r *Repository) CreateRun(ctx context.Context, input domain.CreateRunInput) (domain.ProjectConversationRun, error) {
	builder := r.client.ProjectConversationRun.Create().
		SetID(input.RunID).
		SetPrincipalID(input.PrincipalID).
		SetConversationID(input.ConversationID).
		SetProjectID(input.ProjectID).
		SetProviderID(input.ProviderID).
		SetStatus(entprojectconversationrun.Status(input.Status))
	if input.TurnID != nil {
		builder.SetTurnID(*input.TurnID)
	}
	if input.SessionID != nil {
		builder.SetSessionID(strings.TrimSpace(*input.SessionID))
	}
	if input.WorkspacePath != nil {
		builder.SetWorkspacePath(strings.TrimSpace(*input.WorkspacePath))
	}
	if input.ProviderThreadID != nil {
		builder.SetProviderThreadID(strings.TrimSpace(*input.ProviderThreadID))
	}
	if input.ProviderTurnID != nil {
		builder.SetProviderTurnID(strings.TrimSpace(*input.ProviderTurnID))
	}
	if input.RuntimeStartedAt != nil {
		builder.SetRuntimeStartedAt(*input.RuntimeStartedAt)
	}
	if input.LastHeartbeatAt != nil {
		builder.SetLastHeartbeatAt(*input.LastHeartbeatAt)
	}
	if input.CurrentStepStatus != nil {
		builder.SetCurrentStepStatus(strings.TrimSpace(*input.CurrentStepStatus))
	}
	if input.CurrentStepSummary != nil {
		builder.SetCurrentStepSummary(strings.TrimSpace(*input.CurrentStepSummary))
	}
	if input.CurrentStepChangedAt != nil {
		builder.SetCurrentStepChangedAt(*input.CurrentStepChangedAt)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectConversationRun{}, mapWriteError("create project conversation run", err)
	}
	return mapRun(item), nil
}

func (r *Repository) GetRunByTurnID(ctx context.Context, turnID uuid.UUID) (domain.ProjectConversationRun, error) {
	item, err := r.client.ProjectConversationRun.Query().
		Where(entprojectconversationrun.TurnIDEQ(turnID)).
		Only(ctx)
	if err != nil {
		return domain.ProjectConversationRun{}, mapReadError("get project conversation run by turn", err)
	}
	return mapRun(item), nil
}

func (r *Repository) UpdateRun(ctx context.Context, input domain.UpdateRunInput) (domain.ProjectConversationRun, error) {
	builder := r.client.ProjectConversationRun.UpdateOneID(input.RunID)
	if input.Status != nil {
		builder.SetStatus(entprojectconversationrun.Status(*input.Status))
	}
	if input.ProviderThreadID != nil {
		if trimmed := strings.TrimSpace(*input.ProviderThreadID); trimmed != "" {
			builder.SetProviderThreadID(trimmed)
		} else {
			builder.ClearProviderThreadID()
		}
	}
	if input.ProviderTurnID != nil {
		if trimmed := strings.TrimSpace(*input.ProviderTurnID); trimmed != "" {
			builder.SetProviderTurnID(trimmed)
		} else {
			builder.ClearProviderTurnID()
		}
	}
	if input.TerminalAt != nil {
		builder.SetTerminalAt(*input.TerminalAt)
	}
	if input.LastError != nil {
		if trimmed := strings.TrimSpace(*input.LastError); trimmed != "" {
			builder.SetLastError(trimmed)
		} else {
			builder.ClearLastError()
		}
	}
	if input.LastHeartbeatAt != nil {
		builder.SetLastHeartbeatAt(*input.LastHeartbeatAt)
	}
	if input.CostAmount != nil {
		builder.SetCostAmount(*input.CostAmount)
	}
	if input.CurrentStepStatus != nil {
		if trimmed := strings.TrimSpace(*input.CurrentStepStatus); trimmed != "" {
			builder.SetCurrentStepStatus(trimmed)
		} else {
			builder.ClearCurrentStepStatus()
		}
	}
	if input.CurrentStepSummary != nil {
		if trimmed := strings.TrimSpace(*input.CurrentStepSummary); trimmed != "" {
			builder.SetCurrentStepSummary(trimmed)
		} else {
			builder.ClearCurrentStepSummary()
		}
	}
	if input.CurrentStepChangedAt != nil {
		builder.SetCurrentStepChangedAt(*input.CurrentStepChangedAt)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectConversationRun{}, mapWriteError("update project conversation run", err)
	}
	return mapRun(item), nil
}

func (r *Repository) RecordRunUsage(ctx context.Context, input domain.RecordRunUsageInput) (domain.ProjectConversationRun, error) {
	if r.client == nil {
		return domain.ProjectConversationRun{}, errRepositoryNil
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.ProjectConversationRun{}, fmt.Errorf("start project conversation usage tx: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	runItem, err := tx.ProjectConversationRun.Get(ctx, input.RunID)
	if err != nil {
		return domain.ProjectConversationRun{}, mapReadError("get project conversation run for usage", err)
	}
	if input.ProjectID != uuid.Nil && runItem.ProjectID != input.ProjectID {
		return domain.ProjectConversationRun{}, fmt.Errorf("project conversation run %s does not belong to project %s", runItem.ID, input.ProjectID)
	}
	if input.ProviderID != uuid.Nil && runItem.ProviderID != input.ProviderID {
		return domain.ProjectConversationRun{}, fmt.Errorf("project conversation run %s does not belong to provider %s", runItem.ID, input.ProviderID)
	}

	update := tx.ProjectConversationRun.UpdateOneID(runItem.ID).
		SetInputTokens(input.Totals.InputTokens).
		SetOutputTokens(input.Totals.OutputTokens).
		SetCachedInputTokens(input.Totals.CachedInputTokens).
		SetCacheCreationTokens(input.Totals.CacheCreationTokens).
		SetReasoningTokens(input.Totals.ReasoningTokens).
		SetPromptTokens(input.Totals.PromptTokens).
		SetCandidateTokens(input.Totals.CandidateTokens).
		SetToolTokens(input.Totals.ToolTokens).
		SetTotalTokens(input.Totals.TotalTokens)
	if input.Totals.CostAmount != nil {
		update.SetCostAmount(*input.Totals.CostAmount)
	}
	item, err := update.Save(ctx)
	if err != nil {
		return domain.ProjectConversationRun{}, mapWriteError("update project conversation run usage", err)
	}

	if hasRunUsageDelta(input.Delta) {
		metadata := map[string]any{
			"run_id":                  runItem.ID.String(),
			"provider_id":             runItem.ProviderID.String(),
			"input_tokens":            input.Delta.InputTokens,
			"output_tokens":           input.Delta.OutputTokens,
			"cached_input_tokens":     input.Delta.CachedInputTokens,
			"cache_creation_tokens":   input.Delta.CacheCreationTokens,
			"reasoning_tokens":        input.Delta.ReasoningTokens,
			"prompt_tokens":           input.Delta.PromptTokens,
			"candidate_tokens":        input.Delta.CandidateTokens,
			"tool_tokens":             input.Delta.ToolTokens,
			"total_tokens":            input.Delta.TotalTokens,
			"totals_input_tokens":     input.Totals.InputTokens,
			"totals_output_tokens":    input.Totals.OutputTokens,
			"totals_cached_input":     input.Totals.CachedInputTokens,
			"totals_cache_creation":   input.Totals.CacheCreationTokens,
			"totals_reasoning_tokens": input.Totals.ReasoningTokens,
			"totals_prompt_tokens":    input.Totals.PromptTokens,
			"totals_candidate_tokens": input.Totals.CandidateTokens,
			"totals_tool_tokens":      input.Totals.ToolTokens,
			"totals_total_tokens":     input.Totals.TotalTokens,
		}
		if input.Delta.CostAmount != nil {
			metadata["cost_usd"] = *input.Delta.CostAmount
		}
		if input.Totals.CostAmount != nil {
			metadata["totals_cost_usd"] = *input.Totals.CostAmount
		}
		if input.Totals.ModelContextWindow != nil {
			metadata["model_context_window"] = *input.Totals.ModelContextWindow
		}

		if _, err := tx.ActivityEvent.Create().
			SetProjectID(runItem.ProjectID).
			SetEventType(domain.CostRecordedEventType).
			SetMessage("").
			SetMetadata(metadata).
			SetCreatedAt(input.RecordedAt.UTC()).
			Save(ctx); err != nil {
			return domain.ProjectConversationRun{}, fmt.Errorf("create project conversation cost event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.ProjectConversationRun{}, fmt.Errorf("commit project conversation usage tx: %w", err)
	}

	return mapRun(item), nil
}

func (r *Repository) UpdateProviderRateLimit(ctx context.Context, input domain.UpdateProviderRateLimitInput) error {
	if r.client == nil || input.ProviderID == uuid.Nil {
		return errRepositoryNil
	}

	currentProvider, err := r.client.AgentProvider.Get(ctx, input.ProviderID)
	if err != nil {
		return fmt.Errorf("get provider %s for project conversation rate limit: %w", input.ProviderID, err)
	}
	payload := cloneMap(input.RateLimitPayload)
	if len(payload) == 0 {
		return nil
	}
	snapshotChanged := !reflect.DeepEqual(currentProvider.CliRateLimit, payload)

	updatedProvider, err := r.client.AgentProvider.UpdateOneID(input.ProviderID).
		SetCliRateLimit(payload).
		SetCliRateLimitUpdatedAt(input.ObservedAt.UTC()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("update provider %s project conversation rate limit: %w", input.ProviderID, err)
	}
	if !snapshotChanged || input.ProjectID == uuid.Nil {
		return nil
	}

	if _, err := r.client.ActivityEvent.Create().
		SetProjectID(input.ProjectID).
		SetEventType(activityevent.TypeProviderRateLimitUpdated.String()).
		SetMessage(fmt.Sprintf("Updated provider rate limit snapshot for %s", updatedProvider.Name)).
		SetMetadata(map[string]any{
			"provider_id":           updatedProvider.ID.String(),
			"provider_name":         updatedProvider.Name,
			"machine_id":            updatedProvider.MachineID.String(),
			"rate_limit":            cloneMap(payload),
			"rate_limit_updated_at": input.ObservedAt.UTC().Format(time.RFC3339),
			"changed_fields":        []string{"cli_rate_limit"},
		}).
		SetCreatedAt(input.ObservedAt.UTC()).
		Save(ctx); err != nil {
		return fmt.Errorf("create provider rate limit activity for project conversation: %w", err)
	}

	return nil
}

func (r *Repository) AppendTraceEvent(ctx context.Context, input domain.AppendTraceEventInput) (domain.ProjectConversationTraceEvent, error) {
	sequenceCount, err := r.client.ProjectConversationTraceEvent.Query().
		Where(entprojectconversationtraceevent.RunIDEQ(input.RunID)).
		Count(ctx)
	if err != nil {
		return domain.ProjectConversationTraceEvent{}, fmt.Errorf("count project conversation trace events: %w", err)
	}
	builder := r.client.ProjectConversationTraceEvent.Create().
		SetPrincipalID(input.PrincipalID).
		SetRunID(input.RunID).
		SetConversationID(input.ConversationID).
		SetProjectID(input.ProjectID).
		SetSequence(int64(sequenceCount)).
		SetProvider(strings.TrimSpace(input.Provider)).
		SetKind(strings.TrimSpace(input.Kind)).
		SetStream(strings.TrimSpace(input.Stream)).
		SetPayload(cloneMap(input.Payload))
	if input.Text != nil {
		builder.SetText(strings.TrimSpace(*input.Text))
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectConversationTraceEvent{}, mapWriteError("append project conversation trace event", err)
	}
	return mapTraceEvent(item), nil
}

func (r *Repository) AppendStepEvent(ctx context.Context, input domain.AppendStepEventInput) (domain.ProjectConversationStepEvent, error) {
	builder := r.client.ProjectConversationStepEvent.Create().
		SetPrincipalID(input.PrincipalID).
		SetRunID(input.RunID).
		SetConversationID(input.ConversationID).
		SetProjectID(input.ProjectID).
		SetStepStatus(strings.TrimSpace(input.StepStatus))
	if input.Summary != nil {
		builder.SetSummary(strings.TrimSpace(*input.Summary))
	}
	if input.SourceTraceEventID != nil {
		builder.SetSourceTraceEventID(*input.SourceTraceEventID)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectConversationStepEvent{}, mapWriteError("append project conversation step event", err)
	}
	return mapStepEvent(item), nil
}

func (r *Repository) ListConversations(ctx context.Context, filter domain.ListConversationsFilter) ([]domain.Conversation, error) {
	predicates := []predicate.ChatConversation{
		entchatconversation.ProjectIDEQ(filter.ProjectID),
	}
	if trimmedUserID := strings.TrimSpace(filter.UserID); trimmedUserID != "" {
		predicates = append(predicates, entchatconversation.UserIDEQ(trimmedUserID))
	}
	query := r.client.ChatConversation.Query().
		Where(predicates...).
		Order(ent.Desc(entchatconversation.FieldLastActivityAt))
	if filter.Source != nil {
		query.Where(entchatconversation.SourceEQ(string(*filter.Source)))
	}
	if filter.ProviderID != nil {
		query.Where(entchatconversation.ProviderIDEQ(*filter.ProviderID))
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chat conversations: %w", err)
	}
	items, err = r.ensureConversationTitles(ctx, items)
	if err != nil {
		return nil, err
	}

	return mapConversations(items), nil
}

func (r *Repository) UpdateConversationUser(
	ctx context.Context,
	conversationID uuid.UUID,
	userID string,
) (domain.Conversation, error) {
	item, err := r.client.ChatConversation.UpdateOneID(conversationID).
		SetUserID(strings.TrimSpace(userID)).
		Save(ctx)
	if err != nil {
		return domain.Conversation{}, mapWriteError("update chat conversation owner", err)
	}
	return mapConversation(item), nil
}

func (r *Repository) GetConversation(ctx context.Context, id uuid.UUID) (domain.Conversation, error) {
	item, err := r.client.ChatConversation.Get(ctx, id)
	if err != nil {
		return domain.Conversation{}, mapReadError("get chat conversation", err)
	}
	item, err = r.ensureConversationTitle(ctx, item)
	if err != nil {
		return domain.Conversation{}, err
	}
	return mapConversation(item), nil
}

func (r *Repository) CreateTurnWithUserEntry(
	ctx context.Context,
	conversationID uuid.UUID,
	message string,
) (domain.Turn, domain.Entry, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Turn{}, domain.Entry{}, fmt.Errorf("start create chat turn transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	hasActiveTurn, err := tx.ChatTurn.Query().
		Where(
			entchatturn.ConversationIDEQ(conversationID),
			entchatturn.StatusIn(
				string(domain.TurnStatusPending),
				string(domain.TurnStatusRunning),
				string(domain.TurnStatusInterrupted),
			),
		).
		Exist(ctx)
	if err != nil {
		return domain.Turn{}, domain.Entry{}, fmt.Errorf("query active chat turns: %w", err)
	}
	if hasActiveTurn {
		return domain.Turn{}, domain.Entry{}, fmt.Errorf(
			"%w: conversation %s",
			domain.ErrTurnAlreadyActive,
			conversationID,
		)
	}

	conversationItem, err := tx.ChatConversation.Get(ctx, conversationID)
	if err != nil {
		return domain.Turn{}, domain.Entry{}, mapReadError("get chat conversation for turn creation", err)
	}

	turnCount, err := tx.ChatTurn.Query().Where(entchatturn.ConversationIDEQ(conversationID)).Count(ctx)
	if err != nil {
		return domain.Turn{}, domain.Entry{}, fmt.Errorf("count chat turns: %w", err)
	}
	turnItem, err := tx.ChatTurn.Create().
		SetConversationID(conversationID).
		SetTurnIndex(turnCount + 1).
		SetStatus(string(domain.TurnStatusRunning)).
		Save(ctx)
	if err != nil {
		return domain.Turn{}, domain.Entry{}, mapWriteError("create chat turn", err)
	}

	entryItem, err := createEntryTx(ctx, tx, conversationID, &turnItem.ID, domain.EntryKindUserMessage, map[string]any{
		"role":    "user",
		"content": message,
	})
	if err != nil {
		return domain.Turn{}, domain.Entry{}, err
	}

	conversationUpdate := tx.ChatConversation.UpdateOneID(conversationID).
		SetLastActivityAt(time.Now().UTC()).
		SetStatus(string(domain.ConversationStatusActive))
	if strings.TrimSpace(conversationItem.Title) == "" {
		title, parseErr := domain.ParseConversationTitleFromFirstUserMessage(message)
		if parseErr != nil {
			return domain.Turn{}, domain.Entry{}, parseErr
		}
		conversationUpdate.SetTitle(title.String())
	}
	if _, err := conversationUpdate.Save(ctx); err != nil {
		return domain.Turn{}, domain.Entry{}, mapWriteError("touch chat conversation", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Turn{}, domain.Entry{}, fmt.Errorf("commit create chat turn: %w", err)
	}

	return mapTurn(turnItem), mapEntry(entryItem), nil
}

func (r *Repository) AppendEntry(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID *uuid.UUID,
	kind domain.EntryKind,
	payload map[string]any,
) (domain.Entry, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Entry{}, fmt.Errorf("start append chat entry transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	item, err := createEntryTx(ctx, tx, conversationID, turnID, kind, payload)
	if err != nil {
		return domain.Entry{}, err
	}

	if _, err := tx.ChatConversation.UpdateOneID(conversationID).
		SetLastActivityAt(time.Now().UTC()).
		Save(ctx); err != nil {
		return domain.Entry{}, mapWriteError("touch chat conversation", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Entry{}, fmt.Errorf("commit append chat entry: %w", err)
	}

	return mapEntry(item), nil
}

func (r *Repository) ListEntries(ctx context.Context, conversationID uuid.UUID) ([]domain.Entry, error) {
	items, err := r.client.ChatEntry.Query().
		Where(entchatentry.ConversationIDEQ(conversationID)).
		Order(ent.Asc(entchatentry.FieldSeq)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chat entries: %w", err)
	}
	return mapEntries(items), nil
}

func (r *Repository) GetActiveTurn(ctx context.Context, conversationID uuid.UUID) (domain.Turn, error) {
	item, err := r.client.ChatTurn.Query().
		Where(
			entchatturn.ConversationIDEQ(conversationID),
			entchatturn.StatusIn(
				string(domain.TurnStatusPending),
				string(domain.TurnStatusRunning),
				string(domain.TurnStatusInterrupted),
			),
		).
		Order(ent.Desc(entchatturn.FieldTurnIndex)).
		First(ctx)
	if err != nil {
		return domain.Turn{}, mapReadError("get active chat turn", err)
	}
	return mapTurn(item), nil
}

func (r *Repository) CreatePendingInterrupt(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID uuid.UUID,
	providerRequestID string,
	kind domain.InterruptKind,
	payload map[string]any,
) (domain.PendingInterrupt, domain.Entry, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, fmt.Errorf("start create chat interrupt transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	item, err := tx.ChatPendingInterrupt.Create().
		SetConversationID(conversationID).
		SetTurnID(turnID).
		SetProviderRequestID(strings.TrimSpace(providerRequestID)).
		SetKind(string(kind)).
		SetPayloadJSON(cloneMap(payload)).
		SetStatus(string(domain.InterruptStatusPending)).
		Save(ctx)
	if err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, mapWriteError("create chat pending interrupt", err)
	}

	entryItem, err := createEntryTx(ctx, tx, conversationID, &turnID, domain.EntryKindInterrupt, map[string]any{
		"interrupt_id":        item.ID.String(),
		"provider_request_id": item.ProviderRequestID,
		"kind":                string(kind),
		"status":              string(domain.InterruptStatusPending),
		"payload":             cloneMap(payload),
	})
	if err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, err
	}

	if _, err := tx.ChatTurn.UpdateOneID(turnID).SetStatus(string(domain.TurnStatusInterrupted)).Save(ctx); err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, mapWriteError("mark chat turn interrupted", err)
	}
	if _, err := tx.ChatConversation.UpdateOneID(conversationID).
		SetStatus(string(domain.ConversationStatusInterrupted)).
		SetLastActivityAt(time.Now().UTC()).
		Save(ctx); err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, mapWriteError("mark chat conversation interrupted", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, fmt.Errorf("commit create chat interrupt: %w", err)
	}

	return mapPendingInterrupt(item), mapEntry(entryItem), nil
}

func (r *Repository) GetPendingInterrupt(ctx context.Context, interruptID uuid.UUID) (domain.PendingInterrupt, error) {
	item, err := r.client.ChatPendingInterrupt.Get(ctx, interruptID)
	if err != nil {
		return domain.PendingInterrupt{}, mapReadError("get chat pending interrupt", err)
	}
	return mapPendingInterrupt(item), nil
}

func (r *Repository) ListPendingInterrupts(ctx context.Context, conversationID uuid.UUID) ([]domain.PendingInterrupt, error) {
	items, err := r.client.ChatPendingInterrupt.Query().
		Where(entchatpendinginterrupt.ConversationIDEQ(conversationID)).
		Order(ent.Asc(entchatpendinginterrupt.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chat pending interrupts: %w", err)
	}
	return mapPendingInterrupts(items), nil
}

func (r *Repository) ResolvePendingInterrupt(
	ctx context.Context,
	interruptID uuid.UUID,
	response domain.InterruptResponse,
) (domain.PendingInterrupt, domain.Entry, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, fmt.Errorf("start resolve chat interrupt transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	if _, err := tx.ChatPendingInterrupt.Get(ctx, interruptID); err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, mapReadError("get chat pending interrupt for resolve", err)
	}

	now := time.Now().UTC()
	builder := tx.ChatPendingInterrupt.UpdateOneID(interruptID).
		SetStatus(string(domain.InterruptStatusResolved)).
		SetResolvedAt(now)
	if response.Decision != nil {
		builder.SetDecision(strings.TrimSpace(*response.Decision))
	}
	if response.Answer != nil {
		builder.SetResponseJSON(cloneMap(response.Answer))
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, mapWriteError("resolve chat pending interrupt", err)
	}

	entryPayload := map[string]any{
		"interrupt_id": item.ID.String(),
		"status":       string(domain.InterruptStatusResolved),
	}
	if item.Decision != nil {
		entryPayload["decision"] = *item.Decision
	}
	if len(item.ResponseJSON) > 0 {
		entryPayload["response"] = cloneMap(item.ResponseJSON)
	}
	entryItem, err := createEntryTx(ctx, tx, item.ConversationID, &item.TurnID, domain.EntryKindInterruptResolution, entryPayload)
	if err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, err
	}

	if _, err := tx.ChatTurn.UpdateOneID(item.TurnID).SetStatus(string(domain.TurnStatusRunning)).Save(ctx); err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, mapWriteError("mark chat turn running", err)
	}
	if _, err := tx.ChatConversation.UpdateOneID(item.ConversationID).
		SetStatus(string(domain.ConversationStatusActive)).
		SetLastActivityAt(now).
		Save(ctx); err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, mapWriteError("mark chat conversation active", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.PendingInterrupt{}, domain.Entry{}, fmt.Errorf("commit resolve chat interrupt: %w", err)
	}

	return mapPendingInterrupt(item), mapEntry(entryItem), nil
}

func (r *Repository) CompleteTurn(
	ctx context.Context,
	turnID uuid.UUID,
	status domain.TurnStatus,
	providerTurnID *string,
) (domain.Turn, error) {
	now := time.Now().UTC()
	builder := r.client.ChatTurn.UpdateOneID(turnID).
		SetStatus(string(status)).
		SetCompletedAt(now)
	if providerTurnID != nil && strings.TrimSpace(*providerTurnID) != "" {
		builder.SetProviderTurnID(strings.TrimSpace(*providerTurnID))
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Turn{}, mapWriteError("complete chat turn", err)
	}
	return mapTurn(item), nil
}

func (r *Repository) UpdateConversationAnchors(
	ctx context.Context,
	conversationID uuid.UUID,
	status domain.ConversationStatus,
	anchors domain.ConversationAnchors,
) (domain.Conversation, error) {
	builder := r.client.ChatConversation.UpdateOneID(conversationID).
		SetStatus(string(status)).
		SetLastActivityAt(time.Now().UTC())
	if trimmed := strings.TrimSpace(anchors.RollingSummary); trimmed != "" {
		builder.SetRollingSummary(trimmed)
	}
	if anchors.ProviderThreadID != nil {
		if trimmed := strings.TrimSpace(*anchors.ProviderThreadID); trimmed != "" {
			builder.SetProviderThreadID(trimmed)
		} else {
			builder.ClearProviderThreadID()
		}
	}
	if anchors.LastTurnID != nil {
		if trimmed := strings.TrimSpace(*anchors.LastTurnID); trimmed != "" {
			builder.SetLastTurnID(trimmed)
		} else {
			builder.ClearLastTurnID()
		}
	}
	if anchors.ProviderThreadStatus != nil {
		if trimmed := strings.TrimSpace(*anchors.ProviderThreadStatus); trimmed != "" {
			builder.SetProviderThreadStatus(trimmed)
		} else {
			builder.ClearProviderThreadStatus()
		}
	}
	if anchors.ProviderThreadActiveFlags != nil {
		builder.SetProviderThreadActiveFlags(append([]string(nil), (*anchors.ProviderThreadActiveFlags)...))
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Conversation{}, mapWriteError("update chat conversation anchors", err)
	}
	return mapConversation(item), nil
}

func (r *Repository) CloseConversationRuntime(ctx context.Context, conversationID uuid.UUID) (domain.Conversation, error) {
	item, err := r.client.ChatConversation.UpdateOneID(conversationID).
		SetStatus(string(domain.ConversationStatusClosed)).
		SetProviderThreadStatus("notLoaded").
		SetProviderThreadActiveFlags([]string{}).
		SetLastActivityAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return domain.Conversation{}, mapWriteError("close chat conversation runtime", err)
	}
	return mapConversation(item), nil
}

func createEntryTx(
	ctx context.Context,
	tx *ent.Tx,
	conversationID uuid.UUID,
	turnID *uuid.UUID,
	kind domain.EntryKind,
	payload map[string]any,
) (*ent.ChatEntry, error) {
	seqCount, err := tx.ChatEntry.Query().Where(entchatentry.ConversationIDEQ(conversationID)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count chat entries: %w", err)
	}

	builder := tx.ChatEntry.Create().
		SetConversationID(conversationID).
		SetSeq(seqCount).
		SetKind(string(kind)).
		SetPayloadJSON(cloneMap(payload))
	if turnID != nil {
		builder.SetTurnID(*turnID)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return nil, mapWriteError("create chat entry", err)
	}
	return item, nil
}

func mapReadError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func mapWriteError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrNotFound
	case ent.IsConstraintError(err):
		return ErrConflict
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func rollbackOnError(_ context.Context, tx *ent.Tx, errp *error) {
	if *errp == nil {
		return
	}
	_ = tx.Rollback()
}

func mapConversations(items []*ent.ChatConversation) []domain.Conversation {
	conversations := make([]domain.Conversation, 0, len(items))
	for _, item := range items {
		conversations = append(conversations, mapConversation(item))
	}
	return conversations
}

func mapConversation(item *ent.ChatConversation) domain.Conversation {
	return domain.Conversation{
		ID:                        item.ID,
		ProjectID:                 item.ProjectID,
		UserID:                    item.UserID,
		Source:                    domain.Source(item.Source),
		ProviderID:                item.ProviderID,
		Status:                    domain.ConversationStatus(item.Status),
		Title:                     domain.ConversationTitle(strings.TrimSpace(item.Title)),
		ProviderThreadID:          cloneStringPointer(item.ProviderThreadID),
		LastTurnID:                cloneStringPointer(item.LastTurnID),
		ProviderThreadStatus:      cloneStringPointer(item.ProviderThreadStatus),
		ProviderThreadActiveFlags: append([]string(nil), item.ProviderThreadActiveFlags...),
		RollingSummary:            item.RollingSummary,
		LastActivityAt:            item.LastActivityAt,
		CreatedAt:                 item.CreatedAt,
		UpdatedAt:                 item.UpdatedAt,
	}
}

func (r *Repository) ensureConversationTitles(
	ctx context.Context,
	items []*ent.ChatConversation,
) ([]*ent.ChatConversation, error) {
	if len(items) == 0 {
		return items, nil
	}

	updated := make([]*ent.ChatConversation, 0, len(items))
	for _, item := range items {
		ensured, err := r.ensureConversationTitle(ctx, item)
		if err != nil {
			return nil, err
		}
		updated = append(updated, ensured)
	}
	return updated, nil
}

func (r *Repository) ensureConversationTitle(
	ctx context.Context,
	item *ent.ChatConversation,
) (*ent.ChatConversation, error) {
	if item == nil || strings.TrimSpace(item.Title) != "" {
		return item, nil
	}

	entry, err := r.client.ChatEntry.Query().
		Where(
			entchatentry.ConversationIDEQ(item.ID),
			entchatentry.KindEQ(string(domain.EntryKindUserMessage)),
		).
		Order(ent.Asc(entchatentry.FieldSeq)).
		First(ctx)
	switch {
	case ent.IsNotFound(err):
		return item, nil
	case err != nil:
		return nil, fmt.Errorf("query earliest user chat entry: %w", err)
	}

	title, err := domain.ParseConversationTitleFromFirstUserMessage(conversationEntryContent(entry.PayloadJSON))
	if err != nil {
		return item, nil
	}

	updated, err := r.client.ChatConversation.UpdateOneID(item.ID).
		SetTitle(title.String()).
		Save(ctx)
	if err != nil {
		return nil, mapWriteError("backfill chat conversation title", err)
	}
	return updated, nil
}

func conversationEntryContent(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	value, ok := payload["content"].(string)
	if !ok {
		return ""
	}
	return value
}

func mapTurn(item *ent.ChatTurn) domain.Turn {
	return domain.Turn{
		ID:             item.ID,
		ConversationID: item.ConversationID,
		TurnIndex:      item.TurnIndex,
		ProviderTurnID: cloneStringPointer(item.ProviderTurnID),
		Status:         domain.TurnStatus(item.Status),
		StartedAt:      item.StartedAt,
		CompletedAt:    cloneTimePointer(item.CompletedAt),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func mapEntries(items []*ent.ChatEntry) []domain.Entry {
	entries := make([]domain.Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapEntry(item))
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Seq < entries[j].Seq })
	return entries
}

func mapEntry(item *ent.ChatEntry) domain.Entry {
	return domain.Entry{
		ID:             item.ID,
		ConversationID: item.ConversationID,
		TurnID:         cloneUUIDPointer(item.TurnID),
		Seq:            item.Seq,
		Kind:           domain.EntryKind(item.Kind),
		Payload:        cloneMap(item.PayloadJSON),
		CreatedAt:      item.CreatedAt,
	}
}

func mapPendingInterrupts(items []*ent.ChatPendingInterrupt) []domain.PendingInterrupt {
	interrupts := make([]domain.PendingInterrupt, 0, len(items))
	for _, item := range items {
		interrupts = append(interrupts, mapPendingInterrupt(item))
	}
	return interrupts
}

func mapPendingInterrupt(item *ent.ChatPendingInterrupt) domain.PendingInterrupt {
	return domain.PendingInterrupt{
		ID:                item.ID,
		ConversationID:    item.ConversationID,
		TurnID:            item.TurnID,
		ProviderRequestID: item.ProviderRequestID,
		Kind:              domain.InterruptKind(item.Kind),
		Payload:           cloneMap(item.PayloadJSON),
		Status:            domain.InterruptStatus(item.Status),
		Decision:          cloneStringPointer(item.Decision),
		Response:          cloneMap(item.ResponseJSON),
		ResolvedAt:        cloneTimePointer(item.ResolvedAt),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func mapPrincipal(item *ent.ProjectConversationPrincipal) domain.ProjectConversationPrincipal {
	return domain.ProjectConversationPrincipal{
		ID:                   item.ID,
		ConversationID:       item.ConversationID,
		ProjectID:            item.ProjectID,
		ProviderID:           item.ProviderID,
		Name:                 item.Name,
		Status:               domain.PrincipalStatus(item.Status),
		RuntimeState:         domain.RuntimeState(item.RuntimeState),
		CurrentSessionID:     cloneStringPointer(item.CurrentSessionID),
		CurrentWorkspacePath: cloneStringPointer(item.CurrentWorkspacePath),
		CurrentRunID:         cloneUUIDPointer(item.CurrentRunID),
		LastHeartbeatAt:      cloneTimePointer(item.LastHeartbeatAt),
		CurrentStepStatus:    cloneStringPointer(item.CurrentStepStatus),
		CurrentStepSummary:   cloneStringPointer(item.CurrentStepSummary),
		CurrentStepChangedAt: cloneTimePointer(item.CurrentStepChangedAt),
		ClosedAt:             cloneTimePointer(item.ClosedAt),
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func mapRun(item *ent.ProjectConversationRun) domain.ProjectConversationRun {
	return domain.ProjectConversationRun{
		ID:                   item.ID,
		PrincipalID:          item.PrincipalID,
		ConversationID:       item.ConversationID,
		ProjectID:            item.ProjectID,
		ProviderID:           item.ProviderID,
		TurnID:               cloneUUIDPointer(item.TurnID),
		Status:               domain.RunStatus(item.Status),
		SessionID:            cloneStringPointer(item.SessionID),
		WorkspacePath:        cloneStringPointer(item.WorkspacePath),
		ProviderThreadID:     cloneStringPointer(item.ProviderThreadID),
		ProviderTurnID:       cloneStringPointer(item.ProviderTurnID),
		RuntimeStartedAt:     cloneTimePointer(item.RuntimeStartedAt),
		TerminalAt:           cloneTimePointer(item.TerminalAt),
		LastError:            cloneStringPointer(item.LastError),
		LastHeartbeatAt:      cloneTimePointer(item.LastHeartbeatAt),
		CostAmount:           item.CostAmount,
		InputTokens:          item.InputTokens,
		OutputTokens:         item.OutputTokens,
		CachedInputTokens:    item.CachedInputTokens,
		CacheCreationTokens:  item.CacheCreationTokens,
		ReasoningTokens:      item.ReasoningTokens,
		PromptTokens:         item.PromptTokens,
		CandidateTokens:      item.CandidateTokens,
		ToolTokens:           item.ToolTokens,
		TotalTokens:          item.TotalTokens,
		CurrentStepStatus:    cloneStringPointer(item.CurrentStepStatus),
		CurrentStepSummary:   cloneStringPointer(item.CurrentStepSummary),
		CurrentStepChangedAt: cloneTimePointer(item.CurrentStepChangedAt),
		CreatedAt:            item.CreatedAt,
	}
}

func mapTraceEvent(item *ent.ProjectConversationTraceEvent) domain.ProjectConversationTraceEvent {
	return domain.ProjectConversationTraceEvent{
		ID:             item.ID,
		PrincipalID:    item.PrincipalID,
		RunID:          item.RunID,
		ConversationID: item.ConversationID,
		ProjectID:      item.ProjectID,
		Sequence:       item.Sequence,
		Provider:       item.Provider,
		Kind:           item.Kind,
		Stream:         item.Stream,
		Text:           cloneStringPointer(item.Text),
		Payload:        cloneMap(item.Payload),
		CreatedAt:      item.CreatedAt,
	}
}

func mapStepEvent(item *ent.ProjectConversationStepEvent) domain.ProjectConversationStepEvent {
	return domain.ProjectConversationStepEvent{
		ID:                 item.ID,
		PrincipalID:        item.PrincipalID,
		RunID:              item.RunID,
		ConversationID:     item.ConversationID,
		ProjectID:          item.ProjectID,
		StepStatus:         item.StepStatus,
		Summary:            cloneStringPointer(item.Summary),
		SourceTraceEventID: cloneUUIDPointer(item.SourceTraceEventID),
		CreatedAt:          item.CreatedAt,
	}
}

func defaultProjectConversationPrincipalName(conversationID uuid.UUID) string {
	return "project-conversation:" + strings.TrimSpace(conversationID.String())
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copied := strings.TrimSpace(*value)
	return &copied
}

func cloneUUIDPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

func hasRunUsageDelta(value domain.RunUsageSnapshot) bool {
	return value.InputTokens > 0 ||
		value.OutputTokens > 0 ||
		value.CachedInputTokens > 0 ||
		value.CacheCreationTokens > 0 ||
		value.ReasoningTokens > 0 ||
		value.PromptTokens > 0 ||
		value.CandidateTokens > 0 ||
		value.ToolTokens > 0 ||
		value.TotalTokens > 0 ||
		(value.CostAmount != nil && *value.CostAmount > 0)
}

func marshalRateLimitPayload(rateLimit any) (map[string]any, error) {
	if rateLimit == nil {
		return nil, nil
	}

	payload, err := json.Marshal(rateLimit)
	if err != nil {
		return nil, err
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	return decoded, nil
}
