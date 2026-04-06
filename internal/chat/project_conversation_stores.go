package chat

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/google/uuid"
)

type projectConversationConversationStore interface {
	CreateConversation(ctx context.Context, input domain.CreateConversation) (domain.Conversation, error)
	ListConversations(ctx context.Context, filter domain.ListConversationsFilter) ([]domain.Conversation, error)
	GetConversation(ctx context.Context, id uuid.UUID) (domain.Conversation, error)
	UpdateConversationUser(ctx context.Context, conversationID uuid.UUID, userID string) (domain.Conversation, error)
	UpdateConversationAnchors(ctx context.Context, conversationID uuid.UUID, status domain.ConversationStatus, anchors domain.ConversationAnchors) (domain.Conversation, error)
	CloseConversationRuntime(ctx context.Context, conversationID uuid.UUID) (domain.Conversation, error)
}

type projectConversationEntryStore interface {
	CreateTurnWithUserEntry(ctx context.Context, conversationID uuid.UUID, message string) (domain.Turn, domain.Entry, error)
	AppendEntry(ctx context.Context, conversationID uuid.UUID, turnID *uuid.UUID, kind domain.EntryKind, payload map[string]any) (domain.Entry, error)
	ListEntries(ctx context.Context, conversationID uuid.UUID) ([]domain.Entry, error)
	GetActiveTurn(ctx context.Context, conversationID uuid.UUID) (domain.Turn, error)
	CompleteTurn(ctx context.Context, turnID uuid.UUID, status domain.TurnStatus, providerTurnID *string) (domain.Turn, error)
}

type projectConversationInterruptStore interface {
	CreatePendingInterrupt(ctx context.Context, conversationID uuid.UUID, turnID uuid.UUID, providerRequestID string, kind domain.InterruptKind, payload map[string]any) (domain.PendingInterrupt, domain.Entry, error)
	GetPendingInterrupt(ctx context.Context, interruptID uuid.UUID) (domain.PendingInterrupt, error)
	ListPendingInterrupts(ctx context.Context, conversationID uuid.UUID) ([]domain.PendingInterrupt, error)
	ResolvePendingInterrupt(ctx context.Context, interruptID uuid.UUID, response domain.InterruptResponse) (domain.PendingInterrupt, domain.Entry, error)
}

type projectConversationRuntimeStore interface {
	EnsurePrincipal(ctx context.Context, input domain.EnsurePrincipalInput) (domain.ProjectConversationPrincipal, error)
	GetPrincipal(ctx context.Context, conversationID uuid.UUID) (domain.ProjectConversationPrincipal, error)
	UpdatePrincipalRuntime(ctx context.Context, input domain.UpdatePrincipalRuntimeInput) (domain.ProjectConversationPrincipal, error)
	ClosePrincipal(ctx context.Context, input domain.ClosePrincipalInput) (domain.ProjectConversationPrincipal, error)
	CreateRun(ctx context.Context, input domain.CreateRunInput) (domain.ProjectConversationRun, error)
	GetRunByTurnID(ctx context.Context, turnID uuid.UUID) (domain.ProjectConversationRun, error)
	UpdateRun(ctx context.Context, input domain.UpdateRunInput) (domain.ProjectConversationRun, error)
	RecordRunUsage(ctx context.Context, input domain.RecordRunUsageInput) (domain.ProjectConversationRun, error)
	UpdateProviderRateLimit(ctx context.Context, input domain.UpdateProviderRateLimitInput) error
	AppendTraceEvent(ctx context.Context, input domain.AppendTraceEventInput) (domain.ProjectConversationTraceEvent, error)
	AppendStepEvent(ctx context.Context, input domain.AppendStepEventInput) (domain.ProjectConversationStepEvent, error)
}

type projectConversationStoreSource interface {
	projectConversationConversationStore
	projectConversationEntryStore
	projectConversationInterruptStore
	projectConversationRuntimeStore
}
