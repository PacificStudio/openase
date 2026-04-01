package chatconversation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entchatconversation "github.com/BetterAndBetterII/openase/ent/chatconversation"
	entchatentry "github.com/BetterAndBetterII/openase/ent/chatentry"
	entchatpendinginterrupt "github.com/BetterAndBetterII/openase/ent/chatpendinginterrupt"
	entchatturn "github.com/BetterAndBetterII/openase/ent/chatturn"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/google/uuid"
)

var (
	ErrNotFound          = domain.ErrNotFound
	ErrConflict          = domain.ErrConflict
	ErrInvalidInput      = domain.ErrInvalidInput
	ErrTurnAlreadyActive = domain.ErrTurnAlreadyActive
)

type Repository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) CreateConversation(ctx context.Context, input domain.CreateConversation) (domain.Conversation, error) {
	item, err := r.client.ChatConversation.Create().
		SetProjectID(input.ProjectID).
		SetUserID(strings.TrimSpace(input.UserID)).
		SetSource(string(input.Source)).
		SetProviderID(input.ProviderID).
		SetStatus(string(domain.ConversationStatusActive)).
		Save(ctx)
	if err != nil {
		return domain.Conversation{}, mapWriteError("create chat conversation", err)
	}

	return mapConversation(item), nil
}

func (r *Repository) ListConversations(ctx context.Context, filter domain.ListConversationsFilter) ([]domain.Conversation, error) {
	query := r.client.ChatConversation.Query().
		Where(
			entchatconversation.ProjectIDEQ(filter.ProjectID),
			entchatconversation.UserIDEQ(strings.TrimSpace(filter.UserID)),
		).
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

	return mapConversations(items), nil
}

func (r *Repository) GetConversation(ctx context.Context, id uuid.UUID) (domain.Conversation, error) {
	item, err := r.client.ChatConversation.Get(ctx, id)
	if err != nil {
		return domain.Conversation{}, mapReadError("get chat conversation", err)
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

	if _, err := tx.ChatConversation.UpdateOneID(conversationID).
		SetLastActivityAt(time.Now().UTC()).
		SetStatus(string(domain.ConversationStatusActive)).
		Save(ctx); err != nil {
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
	providerThreadID *string,
	lastTurnID *string,
	rollingSummary string,
) (domain.Conversation, error) {
	builder := r.client.ChatConversation.UpdateOneID(conversationID).
		SetStatus(string(status)).
		SetRollingSummary(strings.TrimSpace(rollingSummary)).
		SetLastActivityAt(time.Now().UTC())
	if providerThreadID != nil && strings.TrimSpace(*providerThreadID) != "" {
		builder.SetProviderThreadID(strings.TrimSpace(*providerThreadID))
	}
	if lastTurnID != nil && strings.TrimSpace(*lastTurnID) != "" {
		builder.SetLastTurnID(strings.TrimSpace(*lastTurnID))
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
		ID:               item.ID,
		ProjectID:        item.ProjectID,
		UserID:           item.UserID,
		Source:           domain.Source(item.Source),
		ProviderID:       item.ProviderID,
		Status:           domain.ConversationStatus(item.Status),
		ProviderThreadID: cloneStringPointer(item.ProviderThreadID),
		LastTurnID:       cloneStringPointer(item.LastTurnID),
		RollingSummary:   item.RollingSummary,
		LastActivityAt:   item.LastActivityAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
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
