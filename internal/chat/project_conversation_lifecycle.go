package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/google/uuid"
)

func (s *ProjectConversationService) normalizeConversationUser(
	ctx context.Context,
	conversation domain.Conversation,
	userID UserID,
) (domain.Conversation, error) {
	if !isStableLocalProjectConversationUser(userID) || conversation.UserID == userID.String() {
		return conversation, nil
	}
	return s.core.conversations.UpdateConversationUser(ctx, conversation.ID, userID.String())
}

func (s *ProjectConversationService) CreateConversation(
	ctx context.Context,
	userID UserID,
	projectID uuid.UUID,
	providerID uuid.UUID,
) (domain.Conversation, error) {
	project, err := s.core.catalog.GetProject(ctx, projectID)
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("get project for chat conversation: %w", err)
	}
	providerItem, err := s.core.catalog.GetAgentProvider(ctx, providerID)
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("get provider for chat conversation: %w", err)
	}
	if providerItem.OrganizationID != project.OrganizationID {
		return domain.Conversation{}, fmt.Errorf("%w: provider is outside the project organization", ErrConversationConflict)
	}

	return s.core.conversations.CreateConversation(ctx, domain.CreateConversation{
		ProjectID:  projectID,
		UserID:     userID.String(),
		Source:     domain.SourceProjectSidebar,
		ProviderID: providerID,
	})
}

func (s *ProjectConversationService) ListConversations(
	ctx context.Context,
	userID UserID,
	projectID uuid.UUID,
	providerID *uuid.UUID,
) ([]domain.Conversation, error) {
	source := domain.SourceProjectSidebar
	filter := domain.ListConversationsFilter{
		ProjectID:  projectID,
		UserID:     userID.String(),
		Source:     &source,
		ProviderID: providerID,
	}
	if isStableLocalProjectConversationUser(userID) {
		filter.UserID = ""
	}
	conversations, err := s.core.conversations.ListConversations(ctx, filter)
	if err != nil {
		return nil, err
	}
	if !isStableLocalProjectConversationUser(userID) {
		return conversations, nil
	}
	for index, conversation := range conversations {
		normalized, normalizeErr := s.normalizeConversationUser(ctx, conversation, userID)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		conversations[index] = normalized
	}
	return conversations, nil
}

func (s *ProjectConversationService) GetConversation(ctx context.Context, userID UserID, conversationID uuid.UUID) (domain.Conversation, error) {
	conversation, err := s.core.conversations.GetConversation(ctx, conversationID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if conversation.UserID != userID.String() && !isStableLocalProjectConversationUser(userID) {
		return domain.Conversation{}, ErrConversationNotFound
	}
	return s.normalizeConversationUser(ctx, conversation, userID)
}

func (s *ProjectConversationService) GetPrincipal(ctx context.Context, userID UserID, conversationID uuid.UUID) (domain.ProjectConversationPrincipal, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.ProjectConversationPrincipal{}, err
	}
	return s.core.runtimeStore.EnsurePrincipal(ctx, domain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      conversation.ProjectID,
		ProviderID:     conversation.ProviderID,
		Name:           projectConversationPrincipalName(conversation.ID),
	})
}

func (s *ProjectConversationService) ListEntries(ctx context.Context, userID UserID, conversationID uuid.UUID) ([]domain.Entry, error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return s.core.entries.ListEntries(ctx, conversationID)
}

func (s *ProjectConversationService) AppendSystemEntry(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	turnID *uuid.UUID,
	payload map[string]any,
) (domain.Entry, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.Entry{}, err
	}
	entryPayload := cloneMapAny(payload)
	entry, err := s.core.entries.AppendEntry(ctx, conversationID, turnID, domain.EntryKindSystem, entryPayload)
	if err != nil {
		return domain.Entry{}, err
	}
	s.broadcastConversationEvent(conversation, StreamEvent{Event: "message", Payload: cloneMapAny(payload)})
	return entry, nil
}

func (s *ProjectConversationService) CloseRuntime(ctx context.Context, userID UserID, conversationID uuid.UUID) error {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	return s.closeConversationRuntime(ctx, conversation)
}

func (s *ProjectConversationService) InterruptTurn(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) error {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}

	unlock := s.core.turnLocks.Lock(projectConversationTurnLockKey(conversation))
	defer unlock()

	activeTurn, err := s.core.entries.GetActiveTurn(ctx, conversationID)
	switch {
	case err == nil:
	case errors.Is(err, ErrConversationNotFound):
		return ErrConversationTurnNotActive
	default:
		return err
	}

	if activeTurn.Status != domain.TurnStatusRunning && activeTurn.Status != domain.TurnStatusPending {
		return ErrConversationTurnNotActive
	}

	pendingInterrupts, err := s.core.interrupts.ListPendingInterrupts(ctx, conversationID)
	if err != nil {
		return err
	}
	for _, interrupt := range pendingInterrupts {
		if interrupt.TurnID == activeTurn.ID && interrupt.Status == domain.InterruptStatusPending {
			return ErrConversationInterruptPending
		}
	}

	live, ok := s.runtimeManager.Get(conversationID)
	if !ok || live == nil || live.turnStop == nil {
		return ErrConversationRuntimeAbsent
	}

	anchor, err := live.turnStop.InterruptTurn(ctx, SessionID(conversationID.String()))
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	statusMessage := "Turn stopped by user."
	reason := "stopped_by_user"

	anchorThreadID := ""
	anchorTurnID := ""
	if live.provider.AdapterType != catalogdomain.AgentProviderAdapterTypeCodexAppServer {
		anchorThreadID = firstNonEmptyTrimmed(
			anchor.ProviderThreadID,
			stringPointerValue(conversation.ProviderThreadID),
		)
		anchorTurnID = firstNonEmptyTrimmed(
			anchor.LastTurnID,
			stringPointerValue(conversation.LastTurnID),
		)
	}
	if _, err := s.core.entries.CompleteTurn(ctx, activeTurn.ID, domain.TurnStatusInterrupted, optionalNonEmptyString(anchorTurnID)); err != nil {
		return err
	}
	if _, err := s.appendConversationEntryWithConflictRetry(ctx, conversationID, &activeTurn.ID, domain.EntryKindSystem, map[string]any{
		"type":    "turn_interrupted",
		"reason":  reason,
		"message": statusMessage,
	}); err != nil {
		return err
	}
	entries, err := s.core.entries.ListEntries(ctx, conversationID)
	if err != nil {
		return err
	}
	summary := buildRollingSummary(entries)
	emptyFlags := []string{}
	updatedConversation, err := s.core.conversations.UpdateConversationAnchors(
		ctx,
		conversationID,
		domain.ConversationStatusActive,
		domain.ConversationAnchors{
			ProviderThreadID:          optionalNonEmptyString(anchorThreadID),
			LastTurnID:                optionalNonEmptyString(anchorTurnID),
			ProviderThreadStatus:      optionalString("notLoaded"),
			ProviderThreadActiveFlags: &emptyFlags,
			RollingSummary:            summary,
		},
	)
	if err != nil {
		return err
	}

	if live.principal.ID != uuid.Nil {
		clearRunID := uuid.Nil
		if principal, principalErr := s.core.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
			PrincipalID:          live.principal.ID,
			RuntimeState:         domain.RuntimeStateReady,
			CurrentSessionID:     optionalString(conversation.ID.String()),
			CurrentWorkspacePath: optionalString(live.workspace.String()),
			CurrentRunID:         &clearRunID,
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_interrupted"),
			CurrentStepSummary:   optionalString(statusMessage),
			CurrentStepChangedAt: &now,
		}); principalErr == nil {
			live.principal = principal
		}
	}

	var run domain.ProjectConversationRun
	if storedRun, runErr := s.core.runtimeStore.GetRunByTurnID(ctx, activeTurn.ID); runErr == nil {
		run = storedRun
		interruptedStatus := domain.RunStatusInterrupted
		_, _ = s.core.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
			RunID:                run.ID,
			Status:               &interruptedStatus,
			ProviderThreadID:     optionalNonEmptyString(anchorThreadID),
			ProviderTurnID:       optionalNonEmptyString(anchorTurnID),
			TerminalAt:           &now,
			LastError:            optionalString(""),
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_interrupted"),
			CurrentStepSummary:   optionalString(statusMessage),
			CurrentStepChangedAt: &now,
		})
	}

	if run.ID != uuid.Nil {
		s.recordConversationTrace(ctx, live, run, "interrupted", map[string]any{
			"message": statusMessage,
			"reason":  reason,
		}, "runtime")
	}

	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event: "message",
		Payload: map[string]any{
			"type": "turn_interrupted",
			"raw": map[string]any{
				"message": statusMessage,
				"reason":  reason,
			},
		},
	})
	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event: "interrupted",
		Payload: map[string]any{
			"conversation_id": conversationID.String(),
			"turn_id":         activeTurn.ID.String(),
			"message":         statusMessage,
			"reason":          reason,
		},
	})
	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event:   "session",
		Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateReady), updatedConversation, &live.provider),
	})
	return nil
}

func (s *ProjectConversationService) appendConversationEntryWithConflictRetry(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID *uuid.UUID,
	kind domain.EntryKind,
	payload map[string]any,
) (domain.Entry, error) {
	const maxAttempts = 3

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		entry, err := s.core.entries.AppendEntry(ctx, conversationID, turnID, kind, payload)
		if err == nil {
			return entry, nil
		}
		if !errors.Is(err, ErrConversationConflict) {
			return domain.Entry{}, err
		}
		lastErr = err
		if ctx.Err() != nil {
			return domain.Entry{}, ctx.Err()
		}
		if attempt == maxAttempts-1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return domain.Entry{}, lastErr
}

func (s *ProjectConversationService) recoverStaleActiveTurnBeforeStart(
	ctx context.Context,
	conversation domain.Conversation,
	hadLive bool,
) error {
	activeTurn, err := s.core.entries.GetActiveTurn(ctx, conversation.ID)
	switch {
	case err == nil:
	case errors.Is(err, ErrConversationNotFound):
		return nil
	default:
		return err
	}

	pendingInterrupts, err := s.core.interrupts.ListPendingInterrupts(ctx, conversation.ID)
	if err != nil {
		return err
	}
	hasPendingInterrupt := false
	for _, interrupt := range pendingInterrupts {
		if interrupt.TurnID == activeTurn.ID && interrupt.Status == domain.InterruptStatusPending {
			hasPendingInterrupt = true
			break
		}
	}

	if activeTurn.Status == domain.TurnStatusInterrupted && hasPendingInterrupt {
		return ErrConversationInterruptPending
	}
	if hadLive {
		return ErrConversationTurnActive
	}
	return s.terminateStaleActiveTurn(ctx, conversation, activeTurn)
}

func (s *ProjectConversationService) terminateStaleActiveTurn(
	ctx context.Context,
	conversation domain.Conversation,
	activeTurn domain.Turn,
) error {
	now := time.Now().UTC()
	statusSummary := fmt.Sprintf(
		"Recovered stale %s turn after runtime became unavailable.",
		strings.TrimSpace(string(activeTurn.Status)),
	)
	if _, err := s.core.entries.CompleteTurn(ctx, activeTurn.ID, domain.TurnStatusTerminated, activeTurn.ProviderTurnID); err != nil {
		return err
	}
	if _, err := s.core.entries.AppendEntry(ctx, conversation.ID, &activeTurn.ID, domain.EntryKindSystem, map[string]any{
		"type":                 "turn_recovered_after_runtime_unavailable",
		"previous_turn_id":     activeTurn.ID.String(),
		"previous_turn_status": string(activeTurn.Status),
		"recovery_reason":      "runtime_unavailable",
	}); err != nil {
		return err
	}

	if run, err := s.core.runtimeStore.GetRunByTurnID(ctx, activeTurn.ID); err == nil {
		terminatedStatus := domain.RunStatusTerminated
		_, _ = s.core.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
			RunID:                run.ID,
			Status:               &terminatedStatus,
			TerminalAt:           &now,
			LastError:            optionalString("project conversation runtime became unavailable before the turn completed"),
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_recovered"),
			CurrentStepSummary:   optionalString(statusSummary),
			CurrentStepChangedAt: &now,
		})
	}
	if principal, err := s.core.runtimeStore.GetPrincipal(ctx, conversation.ID); err == nil {
		clearRunID := uuid.Nil
		_, _ = s.core.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
			PrincipalID:          principal.ID,
			RuntimeState:         domain.RuntimeStateInactive,
			CurrentSessionID:     optionalString(""),
			CurrentRunID:         &clearRunID,
			LastHeartbeatAt:      &now,
			CurrentStepStatus:    optionalString("turn_recovered"),
			CurrentStepSummary:   optionalString(statusSummary),
			CurrentStepChangedAt: &now,
		})
	}

	emptyFlags := []string{}
	updatedConversation, err := s.core.conversations.UpdateConversationAnchors(
		ctx,
		conversation.ID,
		domain.ConversationStatusActive,
		domain.ConversationAnchors{
			ProviderThreadID:          conversation.ProviderThreadID,
			LastTurnID:                conversation.LastTurnID,
			ProviderThreadStatus:      optionalString("notLoaded"),
			ProviderThreadActiveFlags: &emptyFlags,
			RollingSummary:            conversation.RollingSummary,
		},
	)
	if err != nil {
		return err
	}
	if s.core.catalog != nil {
		if providerItem, providerErr := s.core.catalog.GetAgentProvider(ctx, conversation.ProviderID); providerErr == nil {
			s.broadcastConversationEvent(updatedConversation, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversation.ID, "inactive", updatedConversation, &providerItem),
			})
		}
	}
	s.broadcastConversationEvent(conversation, StreamEvent{
		Event: "turn_recovered",
		Payload: map[string]any{
			"turn_id":         activeTurn.ID.String(),
			"previous_status": string(activeTurn.Status),
			"recovery_reason": "runtime_unavailable",
		},
	})
	return nil
}
