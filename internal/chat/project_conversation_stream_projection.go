package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func (s *ProjectConversationService) WatchConversation(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
) (<-chan StreamEvent, func(), error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, nil, err
	}
	live, hasLive := s.runtimeManager.Get(conversationID)

	state := s.watchConversationRuntimeState(ctx, conversationID, live, hasLive)
	sessionPayload := map[string]any{
		"conversation_id": conversationID.String(),
		"runtime_state":   state,
	}
	var sessionProvider *catalogdomain.AgentProvider
	if live != nil {
		sessionProvider = &live.provider
	}
	if sessionProvider == nil && s.core.catalog != nil {
		if providerItem, providerErr := s.core.catalog.GetAgentProvider(ctx, conversation.ProviderID); providerErr == nil {
			sessionProvider = &providerItem
		}
	}
	mergeConversationSessionPayload(sessionPayload, conversation, sessionProvider)
	if hasLive && live != nil {
		anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
		mergeConversationSessionPayload(sessionPayload, domain.Conversation{
			ID:                        conversationID,
			ProviderThreadID:          optionalString(anchor.ProviderThreadID),
			LastTurnID:                optionalString(anchor.LastTurnID),
			ProviderThreadStatus:      optionalString(anchor.ProviderThreadStatus),
			ProviderThreadActiveFlags: append([]string(nil), anchor.ProviderThreadActiveFlags...),
		}, sessionProvider)
	}
	events, cleanup := s.core.streamBroker.Watch(
		conversationID,
		StreamEvent{Event: "session", Payload: sessionPayload},
	)
	return events, cleanup, nil
}

func (s *ProjectConversationService) WatchProjectConversations(
	ctx context.Context,
	userID UserID,
	projectID uuid.UUID,
) (<-chan ProjectConversationMuxEvent, func(), error) {
	conversations, err := s.ListConversations(ctx, userID, projectID, nil)
	if err != nil {
		return nil, nil, err
	}

	providersByID := map[uuid.UUID]catalogdomain.AgentProvider{}
	initial := make([]ProjectConversationMuxEvent, 0, len(conversations))
	for _, conversation := range conversations {
		live, hasLive := s.runtimeManager.Get(conversation.ID)
		state := s.watchConversationRuntimeState(ctx, conversation.ID, live, hasLive)
		var providerItem *catalogdomain.AgentProvider
		if hasLive && live != nil {
			providerItem = &live.provider
			anchor := liveRuntimeSessionAnchor(live, SessionID(conversation.ID.String()))
			conversation.ProviderThreadID = optionalString(anchor.ProviderThreadID)
			conversation.LastTurnID = optionalString(anchor.LastTurnID)
			conversation.ProviderThreadStatus = optionalString(anchor.ProviderThreadStatus)
			conversation.ProviderThreadActiveFlags = append(
				[]string(nil),
				anchor.ProviderThreadActiveFlags...,
			)
		}
		if providerItem == nil && s.core.catalog != nil && conversation.ProviderID != uuid.Nil {
			if cached, ok := providersByID[conversation.ProviderID]; ok {
				providerItem = &cached
			} else if resolved, resolveErr := s.core.catalog.GetAgentProvider(ctx, conversation.ProviderID); resolveErr == nil {
				providersByID[conversation.ProviderID] = resolved
				providerItem = &resolved
			}
		}

		initial = append(initial, newProjectConversationMuxEvent(conversation, StreamEvent{
			Event:   "session",
			Payload: conversationSessionPayload(conversation.ID, state, conversation, providerItem),
		}))
	}

	key := projectConversationMuxWatchKey{ProjectID: projectID, UserID: userID}
	events, cleanup := s.core.muxBroker.Watch(key, initial)
	return events, cleanup, nil
}

func (s *ProjectConversationService) consumeTurn(
	ctx context.Context,
	conversation domain.Conversation,
	turn domain.Turn,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	stream TurnStream,
) {
	conversationID := conversation.ID
	usageHighWater := projectConversationUsageHighWater{
		inputTokens:         run.InputTokens,
		outputTokens:        run.OutputTokens,
		cachedInputTokens:   run.CachedInputTokens,
		cacheCreationTokens: run.CacheCreationTokens,
		reasoningTokens:     run.ReasoningTokens,
		promptTokens:        run.PromptTokens,
		candidateTokens:     run.CandidateTokens,
		toolTokens:          run.ToolTokens,
		totalTokens:         run.TotalTokens,
		costAmount:          run.CostAmount,
		hasCostAmount:       run.CostAmount > 0,
	}
	for event := range stream.Events {
		switch event.Event {
		case "message":
			if normalized, ok := s.handleConversationMessage(ctx, conversationID, turn.ID, event.Payload); ok {
				s.recordConversationTrace(ctx, live, run, "message", normalized.Payload, "assistant")
				s.broadcastConversationEvent(conversation, normalized)
				continue
			}
			s.recordConversationTrace(ctx, live, run, "message", event.Payload, "assistant")
			s.broadcastConversationEvent(conversation, event)
		case "interrupt_requested":
			payload, ok := event.Payload.(RuntimeInterruptEvent)
			if !ok {
				continue
			}
			interruptKind := mapDomainInterruptKind(payload.Kind)
			interruptProvider := providerInterruptProviderName(live.provider)
			interruptPayload := map[string]any{
				"provider": interruptProvider,
				"kind":     string(interruptKind),
				"payload":  cloneMapAny(payload.Payload),
			}
			if len(payload.Options) > 0 {
				options := make([]map[string]any, 0, len(payload.Options))
				for _, option := range payload.Options {
					options = append(options, map[string]any{
						"id":    option.ID,
						"label": option.Label,
					})
				}
				interruptPayload["options"] = options
			}
			pending, _, err := s.core.interrupts.CreatePendingInterrupt(ctx, conversationID, turn.ID, payload.RequestID, interruptKind, interruptPayload)
			if err != nil {
				s.logger.Error("persist chat interrupt", "conversation_id", conversationID, "error", err)
				continue
			}
			s.recordConversationTrace(ctx, live, run, "interrupt_requested", interruptPayload, "runtime")
			anchor := RuntimeSessionAnchor{}
			if live.codex != nil {
				anchor = live.codex.SessionAnchor(SessionID(conversationID.String()))
			}
			_, _ = s.core.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusInterrupted,
				conversationAnchorsFromRuntimeAnchor(anchor, ""),
			)
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateInterrupted, domain.RunStatusInterrupted, "interrupt_requested", "Project conversation is waiting for user interrupt resolution.", nil, nil, "")
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "interrupt_requested",
				Payload: map[string]any{
					"interrupt_id": pending.ID.String(),
					"provider":     interruptProvider,
					"kind":         string(interruptKind),
					"options":      interruptPayload["options"],
					"payload":      interruptPayload["payload"],
				},
			})
		case "done":
			done, ok := event.Payload.(donePayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "done", map[string]any{
				"cost_usd": done.CostUSD,
			}, "runtime")
			anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
			_, _ = s.core.entries.CompleteTurn(ctx, turn.ID, domain.TurnStatusCompleted, optionalNonEmptyString(anchor.LastTurnID))
			entries, _ := s.core.entries.ListEntries(ctx, conversationID)
			summary := buildRollingSummary(entries)
			updatedConversation := conversation
			if storedConversation, err := s.core.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				conversationAnchorsFromRuntimeAnchor(anchor, summary),
			); err == nil {
				updatedConversation = storedConversation
			}
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateReady, domain.RunStatusCompleted, "turn_completed", "Project conversation turn completed.", optionalNonEmptyString(anchor.ProviderThreadID), done.CostUSD, "")
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "turn_done",
				Payload: map[string]any{
					"conversation_id": conversationID.String(),
					"turn_id":         turn.ID.String(),
					"cost_usd":        done.CostUSD,
				},
			})
			s.autoReleaseCompletedRuntime(ctx, updatedConversation, live)
		case "token_usage_updated":
			payload, ok := event.Payload.(runtimeTokenUsagePayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "token_usage_updated", map[string]any{
				"input_tokens":          payload.TotalInputTokens,
				"output_tokens":         payload.TotalOutputTokens,
				"cached_input_tokens":   payload.TotalCachedInputTokens,
				"cache_creation_tokens": payload.TotalCacheCreationTokens,
				"reasoning_tokens":      payload.TotalReasoningTokens,
				"prompt_tokens":         payload.TotalPromptTokens,
				"candidate_tokens":      payload.TotalCandidateTokens,
				"tool_tokens":           payload.TotalToolTokens,
				"total_tokens":          payload.TotalTokens,
				"cost_usd":              payload.CostUSD,
				"model_context_window":  payload.ModelContextWindow,
			}, "runtime")
			run = s.recordConversationUsage(ctx, live, run, payload, &usageHighWater)
		case "rate_limit_updated":
			payload, ok := event.Payload.(runtimeRateLimitPayload)
			if !ok || payload.RateLimit == nil {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "rate_limit_updated", map[string]any{
				"observed_at": payload.ObservedAt.UTC().Format(time.RFC3339),
			}, "runtime")
			s.recordConversationProviderRateLimit(ctx, live, payload)
		case "thread_status":
			payload, ok := event.Payload.(runtimeThreadStatusPayload)
			if !ok {
				continue
			}
			activeFlags := append([]string(nil), payload.ActiveFlags...)
			s.recordConversationTrace(ctx, live, run, "thread_status", map[string]any{
				"thread_id":    payload.ThreadID,
				"status":       payload.Status,
				"active_flags": activeFlags,
			}, "runtime")
			updated, _ := s.core.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				domain.ConversationAnchors{
					ProviderThreadStatus:      optionalString(payload.Status),
					ProviderThreadActiveFlags: &activeFlags,
				},
			)
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateExecuting, domain.RunStatusExecuting, payload.Status, "Conversation provider thread status updated.", optionalNonEmptyString(payload.ThreadID), nil, "")
			entry, _ := s.core.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":         "thread_status",
				"anchor_kind":  "thread",
				"thread_id":    payload.ThreadID,
				"status":       payload.Status,
				"active_flags": append([]string(nil), payload.ActiveFlags...),
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "message",
				Payload: map[string]any{
					"type": "thread_status",
					"raw": map[string]any{
						"anchor_kind":  "thread",
						"thread_id":    payload.ThreadID,
						"status":       payload.Status,
						"active_flags": append([]string(nil), payload.ActiveFlags...),
						"entry_id":     entry.ID.String(),
					},
				},
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "thread_status",
				Payload: map[string]any{
					"thread_id":    payload.ThreadID,
					"status":       payload.Status,
					"active_flags": append([]string(nil), payload.ActiveFlags...),
					"entry_id":     entry.ID.String(),
				},
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateExecuting), updated, &live.provider),
			})
		case "session_anchor":
			anchor, ok := event.Payload.(RuntimeSessionAnchor)
			if !ok || strings.TrimSpace(anchor.ProviderThreadID) == "" {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "session_anchor", map[string]any{
				"provider_thread_id":     anchor.ProviderThreadID,
				"last_turn_id":           anchor.LastTurnID,
				"provider_thread_status": anchor.ProviderThreadStatus,
				"active_flags":           append([]string(nil), anchor.ProviderThreadActiveFlags...),
			}, "runtime")
			updated, _ := s.core.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				conversationAnchorsFromRuntimeAnchor(anchor, ""),
			)
			s.broadcastConversationEvent(updated, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateExecuting), updated, &live.provider),
			})
		case "session_state":
			payload, ok := event.Payload.(runtimeSessionStatePayload)
			if !ok {
				continue
			}
			activeFlags := append([]string(nil), payload.ActiveFlags...)
			s.recordConversationTrace(ctx, live, run, "session_state", map[string]any{
				"status":       payload.Status,
				"active_flags": activeFlags,
				"detail":       payload.Detail,
				"raw":          cloneAnyMap(payload.Raw),
			}, "runtime")
			updated, _ := s.core.conversations.UpdateConversationAnchors(
				ctx,
				conversationID,
				domain.ConversationStatusActive,
				domain.ConversationAnchors{
					ProviderThreadStatus:      optionalString(payload.Status),
					ProviderThreadActiveFlags: &activeFlags,
				},
			)
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateExecuting, domain.RunStatusExecuting, payload.Status, payload.Detail, nil, nil, "")
			entry, _ := s.core.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":         "session_state",
				"anchor_kind":  "session",
				"status":       payload.Status,
				"active_flags": append([]string(nil), payload.ActiveFlags...),
				"detail":       payload.Detail,
				"raw":          cloneAnyMap(payload.Raw),
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "message",
				Payload: map[string]any{
					"type": "session_state",
					"raw": map[string]any{
						"anchor_kind":  "session",
						"status":       payload.Status,
						"active_flags": append([]string(nil), payload.ActiveFlags...),
						"detail":       payload.Detail,
						"entry_id":     entry.ID.String(),
					},
				},
			})
			s.broadcastConversationEvent(updated, StreamEvent{
				Event:   "session",
				Payload: conversationSessionPayload(conversationID, string(domain.RuntimeStateExecuting), updated, &live.provider),
			})
		case "thread_compacted":
			payload, ok := event.Payload.(runtimeThreadCompactedPayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "thread_compacted", map[string]any{
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
			}, "runtime")
			entry, _ := s.core.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":      "thread_compacted",
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "thread_compacted",
				Payload: map[string]any{
					"thread_id": payload.ThreadID,
					"turn_id":   payload.TurnID,
					"entry_id":  entry.ID.String(),
				},
			})
		case "plan_updated":
			payload, ok := event.Payload.(runtimePlanUpdatedPayload)
			if !ok {
				continue
			}
			rawPlan := make([]map[string]any, 0, len(payload.Plan))
			for _, item := range payload.Plan {
				rawPlan = append(rawPlan, map[string]any{
					"step":   item.Step,
					"status": item.Status,
				})
			}
			s.recordConversationTrace(ctx, live, run, "plan_updated", map[string]any{
				"thread_id":   payload.ThreadID,
				"turn_id":     payload.TurnID,
				"explanation": payload.Explanation,
				"plan":        rawPlan,
			}, "runtime")
			s.recordConversationStep(ctx, live, run, domain.RuntimeStateExecuting, domain.RunStatusExecuting, "plan_updated", stringPointerValue(payload.Explanation), nil, nil, "")
			entry, _ := s.core.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":        "turn_plan_updated",
				"thread_id":   payload.ThreadID,
				"turn_id":     payload.TurnID,
				"explanation": payload.Explanation,
				"plan":        rawPlan,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "plan_updated",
				Payload: map[string]any{
					"thread_id":   payload.ThreadID,
					"turn_id":     payload.TurnID,
					"explanation": payload.Explanation,
					"plan":        rawPlan,
					"entry_id":    entry.ID.String(),
				},
			})
		case "diff_updated":
			payload, ok := event.Payload.(runtimeDiffUpdatedPayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "diff_updated", map[string]any{
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
				"diff":      payload.Diff,
			}, "runtime")
			entry, _ := s.core.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":      "turn_diff_updated",
				"thread_id": payload.ThreadID,
				"turn_id":   payload.TurnID,
				"diff":      payload.Diff,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "diff_updated",
				Payload: map[string]any{
					"thread_id": payload.ThreadID,
					"turn_id":   payload.TurnID,
					"diff":      payload.Diff,
					"entry_id":  entry.ID.String(),
				},
			})
		case "reasoning_updated":
			payload, ok := event.Payload.(runtimeReasoningUpdatedPayload)
			if !ok {
				continue
			}
			s.recordConversationTrace(ctx, live, run, "reasoning_updated", map[string]any{
				"thread_id":     payload.ThreadID,
				"turn_id":       payload.TurnID,
				"item_id":       payload.ItemID,
				"kind":          payload.Kind,
				"delta":         payload.Delta,
				"summary_index": payload.SummaryIndex,
				"content_index": payload.ContentIndex,
			}, "runtime")
			entry, _ := s.core.entries.AppendEntry(ctx, conversationID, &turn.ID, domain.EntryKindSystem, map[string]any{
				"type":          "turn_reasoning_updated",
				"thread_id":     payload.ThreadID,
				"turn_id":       payload.TurnID,
				"item_id":       payload.ItemID,
				"kind":          payload.Kind,
				"delta":         payload.Delta,
				"summary_index": payload.SummaryIndex,
				"content_index": payload.ContentIndex,
			})
			s.broadcastConversationEvent(conversation, StreamEvent{
				Event: "reasoning_updated",
				Payload: map[string]any{
					"thread_id":     payload.ThreadID,
					"turn_id":       payload.TurnID,
					"item_id":       payload.ItemID,
					"kind":          payload.Kind,
					"delta":         payload.Delta,
					"summary_index": payload.SummaryIndex,
					"content_index": payload.ContentIndex,
					"entry_id":      entry.ID.String(),
				},
			})
		case "error":
			payload, ok := event.Payload.(errorPayload)
			if ok {
				s.recordConversationTrace(ctx, live, run, "error", map[string]any{"message": payload.Message}, "runtime")
				anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
				_, _ = s.core.entries.CompleteTurn(ctx, turn.ID, domain.TurnStatusFailed, optionalNonEmptyString(anchor.LastTurnID))
				_, _ = s.core.conversations.UpdateConversationAnchors(
					ctx,
					conversationID,
					domain.ConversationStatusActive,
					conversationAnchorsFromRuntimeAnchor(anchor, ""),
				)
				s.recordConversationStep(ctx, live, run, domain.RuntimeStateReady, domain.RunStatusFailed, "turn_failed", payload.Message, optionalNonEmptyString(anchor.ProviderThreadID), nil, payload.Message)
				s.broadcastConversationEvent(conversation, StreamEvent{
					Event: "error",
					Payload: map[string]any{
						"message": payload.Message,
					},
				})
			}
		case "interrupted":
			payload, ok := event.Payload.(errorPayload)
			if ok {
				s.recordConversationTrace(ctx, live, run, "interrupted", map[string]any{"message": payload.Message}, "runtime")
				anchor := liveRuntimeSessionAnchor(live, SessionID(conversationID.String()))
				_, _ = s.core.entries.CompleteTurn(ctx, turn.ID, domain.TurnStatusInterrupted, optionalNonEmptyString(anchor.LastTurnID))
				_, _ = s.core.conversations.UpdateConversationAnchors(
					ctx,
					conversationID,
					domain.ConversationStatusInterrupted,
					conversationAnchorsFromRuntimeAnchor(anchor, ""),
				)
				s.recordConversationStep(ctx, live, run, domain.RuntimeStateInterrupted, domain.RunStatusInterrupted, "turn_interrupted", payload.Message, optionalNonEmptyString(anchor.ProviderThreadID), nil, "")
				s.broadcastConversationEvent(conversation, StreamEvent{
					Event: "interrupted",
					Payload: map[string]any{
						"message": payload.Message,
					},
				})
			}
		}
	}
}

func (s *ProjectConversationService) recordConversationTrace(
	ctx context.Context,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	kind string,
	payload any,
	stream string,
) {
	if s == nil || s.core.runtimeStore == nil || live == nil || live.principal.ID == uuid.Nil || run.ID == uuid.Nil {
		return
	}
	now := time.Now().UTC()
	tracePayload := mapConversationTracePayload(payload)
	trace, err := s.core.runtimeStore.AppendTraceEvent(ctx, domain.AppendTraceEventInput{
		RunID:          run.ID,
		PrincipalID:    live.principal.ID,
		ConversationID: live.principal.ConversationID,
		ProjectID:      live.principal.ProjectID,
		Provider:       providerInterruptProviderName(live.provider),
		Kind:           kind,
		Stream:         stream,
		Text:           optionalNonEmptyString(extractConversationTraceText(tracePayload)),
		Payload:        tracePayload,
	})
	if err != nil {
		s.logger.Warn("append project conversation trace event failed", "conversation_id", live.principal.ConversationID, "run_id", run.ID, "error", err)
		return
	}
	_ = trace
	_, _ = s.core.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
		RunID:                run.ID,
		LastHeartbeatAt:      &now,
		CurrentStepChangedAt: &now,
	})
}

func (s *ProjectConversationService) recordConversationStep(
	ctx context.Context,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	runtimeState domain.RuntimeState,
	runStatus domain.RunStatus,
	stepStatus string,
	summary string,
	providerThreadID *string,
	costUSD *float64,
	lastError string,
) {
	if s == nil || s.core.runtimeStore == nil || live == nil || live.principal.ID == uuid.Nil || run.ID == uuid.Nil {
		return
	}
	now := time.Now().UTC()
	summaryPtr := optionalNonEmptyString(summary)
	step, err := s.core.runtimeStore.AppendStepEvent(ctx, domain.AppendStepEventInput{
		RunID:          run.ID,
		PrincipalID:    live.principal.ID,
		ConversationID: live.principal.ConversationID,
		ProjectID:      live.principal.ProjectID,
		StepStatus:     stepStatus,
		Summary:        summaryPtr,
	})
	if err != nil {
		s.logger.Warn("append project conversation step event failed", "conversation_id", live.principal.ConversationID, "run_id", run.ID, "error", err)
	}
	_ = step
	runStatusCopy := runStatus
	updateInput := domain.UpdateRunInput{
		RunID:                run.ID,
		Status:               &runStatusCopy,
		ProviderThreadID:     providerThreadID,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString(stepStatus),
		CurrentStepSummary:   summaryPtr,
		CurrentStepChangedAt: &now,
	}
	if costUSD != nil {
		updateInput.CostAmount = costUSD
	}
	if strings.TrimSpace(lastError) != "" {
		updateInput.LastError = optionalString(lastError)
	}
	if runStatus == domain.RunStatusCompleted || runStatus == domain.RunStatusFailed || runStatus == domain.RunStatusTerminated {
		updateInput.TerminalAt = &now
	}
	updatedRun, err := s.core.runtimeStore.UpdateRun(ctx, updateInput)
	if err == nil {
		run = updatedRun
	}
	currentRunID := run.ID
	updatedPrincipal, err := s.core.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
		PrincipalID:          live.principal.ID,
		RuntimeState:         runtimeState,
		CurrentSessionID:     optionalString(live.principal.ConversationID.String()),
		CurrentWorkspacePath: optionalString(live.workspace.String()),
		CurrentRunID:         &currentRunID,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString(stepStatus),
		CurrentStepSummary:   summaryPtr,
		CurrentStepChangedAt: &now,
	})
	if err == nil {
		live.principal = updatedPrincipal
	}
}

func (s *ProjectConversationService) recordConversationUsage(
	ctx context.Context,
	live *liveProjectConversation,
	run domain.ProjectConversationRun,
	payload runtimeTokenUsagePayload,
	highWater *projectConversationUsageHighWater,
) domain.ProjectConversationRun {
	if s == nil || s.core.runtimeStore == nil || live == nil || run.ID == uuid.Nil || highWater == nil {
		return run
	}

	delta := domain.RunUsageSnapshot{
		InputTokens:         clampUsageDelta(payload.TotalInputTokens - highWater.inputTokens),
		OutputTokens:        clampUsageDelta(payload.TotalOutputTokens - highWater.outputTokens),
		CachedInputTokens:   clampUsageDelta(payload.TotalCachedInputTokens - highWater.cachedInputTokens),
		CacheCreationTokens: clampUsageDelta(payload.TotalCacheCreationTokens - highWater.cacheCreationTokens),
		ReasoningTokens:     clampUsageDelta(payload.TotalReasoningTokens - highWater.reasoningTokens),
		PromptTokens:        clampUsageDelta(payload.TotalPromptTokens - highWater.promptTokens),
		CandidateTokens:     clampUsageDelta(payload.TotalCandidateTokens - highWater.candidateTokens),
		ToolTokens:          clampUsageDelta(payload.TotalToolTokens - highWater.toolTokens),
		TotalTokens:         clampUsageDelta(payload.TotalTokens - highWater.totalTokens),
	}
	if payload.CostUSD != nil {
		costDelta := *payload.CostUSD
		if highWater.hasCostAmount {
			costDelta -= highWater.costAmount
		}
		if costDelta > 0 {
			delta.CostAmount = cloneCostUSD(&costDelta)
		}
	}

	updatedRun, err := s.core.runtimeStore.RecordRunUsage(ctx, domain.RecordRunUsageInput{
		RunID:      run.ID,
		ProjectID:  live.principal.ProjectID,
		ProviderID: live.principal.ProviderID,
		RecordedAt: time.Now().UTC(),
		Totals: domain.RunUsageSnapshot{
			InputTokens:         maxRunUsageTotal(highWater.inputTokens, payload.TotalInputTokens),
			OutputTokens:        maxRunUsageTotal(highWater.outputTokens, payload.TotalOutputTokens),
			CachedInputTokens:   maxRunUsageTotal(highWater.cachedInputTokens, payload.TotalCachedInputTokens),
			CacheCreationTokens: maxRunUsageTotal(highWater.cacheCreationTokens, payload.TotalCacheCreationTokens),
			ReasoningTokens:     maxRunUsageTotal(highWater.reasoningTokens, payload.TotalReasoningTokens),
			PromptTokens:        maxRunUsageTotal(highWater.promptTokens, payload.TotalPromptTokens),
			CandidateTokens:     maxRunUsageTotal(highWater.candidateTokens, payload.TotalCandidateTokens),
			ToolTokens:          maxRunUsageTotal(highWater.toolTokens, payload.TotalToolTokens),
			TotalTokens:         maxRunUsageTotal(highWater.totalTokens, payload.TotalTokens),
			CostAmount:          cloneCostUSD(payload.CostUSD),
			ModelContextWindow:  cloneInt64Pointer(payload.ModelContextWindow),
		},
		Delta: delta,
	})
	if err != nil {
		s.logger.Warn("record project conversation usage failed", "conversation_id", live.principal.ConversationID, "run_id", run.ID, "error", err)
		return run
	}

	highWater.inputTokens = maxRunUsageTotal(highWater.inputTokens, payload.TotalInputTokens)
	highWater.outputTokens = maxRunUsageTotal(highWater.outputTokens, payload.TotalOutputTokens)
	highWater.cachedInputTokens = maxRunUsageTotal(highWater.cachedInputTokens, payload.TotalCachedInputTokens)
	highWater.cacheCreationTokens = maxRunUsageTotal(highWater.cacheCreationTokens, payload.TotalCacheCreationTokens)
	highWater.reasoningTokens = maxRunUsageTotal(highWater.reasoningTokens, payload.TotalReasoningTokens)
	highWater.promptTokens = maxRunUsageTotal(highWater.promptTokens, payload.TotalPromptTokens)
	highWater.candidateTokens = maxRunUsageTotal(highWater.candidateTokens, payload.TotalCandidateTokens)
	highWater.toolTokens = maxRunUsageTotal(highWater.toolTokens, payload.TotalToolTokens)
	highWater.totalTokens = maxRunUsageTotal(highWater.totalTokens, payload.TotalTokens)
	if payload.CostUSD != nil && (!highWater.hasCostAmount || *payload.CostUSD > highWater.costAmount) {
		highWater.costAmount = *payload.CostUSD
		highWater.hasCostAmount = true
	}

	return updatedRun
}

func (s *ProjectConversationService) recordConversationProviderRateLimit(
	ctx context.Context,
	live *liveProjectConversation,
	payload runtimeRateLimitPayload,
) {
	if s == nil || s.core.runtimeStore == nil || live == nil || live.provider.ID == uuid.Nil || payload.RateLimit == nil {
		return
	}

	rateLimitPayload, err := marshalProjectConversationRateLimitPayload(payload.RateLimit)
	if err != nil {
		s.logger.Warn("marshal project conversation provider rate limit failed", "conversation_id", live.principal.ConversationID, "provider_id", live.provider.ID, "error", err)
		return
	}
	if err := s.core.runtimeStore.UpdateProviderRateLimit(ctx, domain.UpdateProviderRateLimitInput{
		ProjectID:        live.principal.ProjectID,
		ProviderID:       live.provider.ID,
		ObservedAt:       payload.ObservedAt.UTC(),
		RateLimitPayload: rateLimitPayload,
	}); err != nil {
		s.logger.Warn("update project conversation provider rate limit failed", "conversation_id", live.principal.ConversationID, "provider_id", live.provider.ID, "error", err)
	}
}

func marshalProjectConversationRateLimitPayload(rateLimit *provider.CLIRateLimit) (map[string]any, error) {
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

func clampUsageDelta(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func maxRunUsageTotal(left int64, right int64) int64 {
	if right > left {
		return right
	}
	return left
}

func mapConversationTracePayload(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMapAny(typed)
	case nil:
		return map[string]any{}
	default:
		return map[string]any{"value": fmt.Sprintf("%v", typed)}
	}
}

func extractConversationTraceText(payload map[string]any) string {
	for _, key := range []string{"content", "message", "summary", "detail"} {
		if trimmed := strings.TrimSpace(stringValue(payload[key])); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s *ProjectConversationService) autoReleaseCompletedRuntime(
	ctx context.Context,
	conversation domain.Conversation,
	live *liveProjectConversation,
) {
	if s == nil || s.runtimeManager == nil {
		return
	}

	closedLive, _ := s.runtimeManager.Close(conversation.ID)
	if closedLive == nil {
		closedLive = live
	}

	if s.core.runtimeStore != nil {
		principalID := uuid.Nil
		if closedLive != nil && closedLive.principal.ID != uuid.Nil {
			principalID = closedLive.principal.ID
		} else if principal, err := s.core.runtimeStore.GetPrincipal(ctx, conversation.ID); err == nil {
			principalID = principal.ID
		}
		if principalID != uuid.Nil {
			if _, err := s.core.runtimeStore.ClosePrincipal(ctx, domain.ClosePrincipalInput{PrincipalID: principalID}); err != nil {
				s.logger.Warn("close completed project conversation principal failed", "conversation_id", conversation.ID, "principal_id", principalID, "error", err)
			}
		}
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
		s.logger.Warn("persist completed project conversation auto-release failed", "conversation_id", conversation.ID, "error", err)
		return
	}

	var providerItem *catalogdomain.AgentProvider
	switch {
	case closedLive != nil:
		providerItem = &closedLive.provider
	case live != nil:
		providerItem = &live.provider
	case s.core.catalog != nil:
		if resolved, resolveErr := s.core.catalog.GetAgentProvider(ctx, updatedConversation.ProviderID); resolveErr == nil {
			providerItem = &resolved
		}
	}

	s.broadcastConversationEvent(updatedConversation, StreamEvent{
		Event:   "session",
		Payload: conversationSessionPayload(conversation.ID, string(domain.RuntimeStateInactive), updatedConversation, providerItem),
	})
}

func (s *ProjectConversationService) watchConversationRuntimeState(
	ctx context.Context,
	conversationID uuid.UUID,
	live *liveProjectConversation,
	hasLive bool,
) string {
	if live != nil {
		if state := strings.TrimSpace(string(live.principal.RuntimeState)); state != "" {
			return state
		}
	}
	if s != nil && s.core.runtimeStore != nil {
		if principal, err := s.core.runtimeStore.GetPrincipal(ctx, conversationID); err == nil {
			if state := strings.TrimSpace(string(principal.RuntimeState)); state != "" {
				return state
			}
		}
	}
	if hasLive {
		return string(domain.RuntimeStateReady)
	}
	return string(domain.RuntimeStateInactive)
}

func conversationSessionPayload(
	conversationID uuid.UUID,
	runtimeState string,
	conversation domain.Conversation,
	providerItem *catalogdomain.AgentProvider,
) map[string]any {
	payload := map[string]any{
		"conversation_id": conversationID.String(),
		"runtime_state":   strings.TrimSpace(runtimeState),
	}
	mergeConversationSessionPayload(payload, conversation, providerItem)
	return payload
}

func mergeConversationSessionPayload(
	payload map[string]any,
	conversation domain.Conversation,
	providerItem *catalogdomain.AgentProvider,
) {
	if payload == nil {
		return
	}
	if providerItem != nil {
		switch providerItem.AdapterType {
		case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
			payload["provider_anchor_kind"] = "thread"
			payload["provider_turn_supported"] = true
		case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
			payload["provider_anchor_kind"] = "session"
			payload["provider_turn_supported"] = false
		}
	}
	if title := conversation.Title.String(); title != "" {
		payload["title"] = title
	}
	if summary := strings.TrimSpace(conversation.RollingSummary); summary != "" {
		payload["rolling_summary"] = summary
	}
	if conversation.ProviderThreadID != nil && strings.TrimSpace(*conversation.ProviderThreadID) != "" {
		anchorID := strings.TrimSpace(*conversation.ProviderThreadID)
		payload["provider_thread_id"] = anchorID
		payload["provider_anchor_id"] = anchorID
	}
	if conversation.LastTurnID != nil && strings.TrimSpace(*conversation.LastTurnID) != "" {
		turnID := strings.TrimSpace(*conversation.LastTurnID)
		payload["last_turn_id"] = turnID
		payload["provider_turn_id"] = turnID
	}
	if conversation.ProviderThreadStatus != nil && strings.TrimSpace(*conversation.ProviderThreadStatus) != "" {
		status := strings.TrimSpace(*conversation.ProviderThreadStatus)
		payload["provider_thread_status"] = status
		payload["provider_status"] = status
	}
	if len(conversation.ProviderThreadActiveFlags) > 0 {
		flags := append([]string(nil), conversation.ProviderThreadActiveFlags...)
		payload["provider_thread_active_flags"] = flags
		payload["provider_active_flags"] = append([]string(nil), flags...)
	}
}

func (s *ProjectConversationService) handleConversationMessage(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID uuid.UUID,
	payload any,
) (StreamEvent, bool) {
	switch typed := payload.(type) {
	case textPayload:
		_, _ = s.core.entries.AppendEntry(ctx, conversationID, &turnID, domain.EntryKindAssistantTextDelta, map[string]any{
			"role":    "assistant",
			"content": typed.Content,
		})
		return StreamEvent{
			Event: "message",
			Payload: map[string]any{
				"type":    chatMessageTypeText,
				"content": typed.Content,
			},
		}, true
	case map[string]any:
		kind := domain.EntryKindSystem
		switch typed["type"] {
		case chatMessageTypeDiff:
			kind = domain.EntryKindDiff
		case chatMessageTypeTaskStarted, chatMessageTypeTaskNotification, chatMessageTypeTaskProgress:
			kind = domain.EntryKindSystem
		}
		entry, _ := s.core.entries.AppendEntry(ctx, conversationID, &turnID, kind, cloneMapAny(typed))
		normalized := cloneMapAny(typed)
		if kind == domain.EntryKindDiff {
			normalized["entry_id"] = entry.ID.String()
		}
		return StreamEvent{Event: "message", Payload: normalized}, true
	}
	return StreamEvent{}, false
}

func (s *ProjectConversationService) broadcastConversationEvent(
	conversation domain.Conversation,
	event StreamEvent,
) {
	s.core.streamBroker.Broadcast(conversation.ID, event)
	if conversation.ID == uuid.Nil || conversation.ProjectID == uuid.Nil || strings.TrimSpace(conversation.UserID) == "" {
		return
	}
	s.core.muxBroker.Broadcast(
		projectConversationMuxWatchKey{
			ProjectID: conversation.ProjectID,
			UserID:    UserID(conversation.UserID),
		},
		newProjectConversationMuxEvent(conversation, event),
	)
}

func (s *ProjectConversationService) broadcast(conversationID uuid.UUID, event StreamEvent) {
	if s == nil || s.core.conversations == nil {
		return
	}
	conversation, err := s.core.conversations.GetConversation(context.Background(), conversationID)
	if err != nil {
		return
	}
	s.broadcastConversationEvent(conversation, event)
}

func newProjectConversationMuxEvent(
	conversation domain.Conversation,
	event StreamEvent,
) ProjectConversationMuxEvent {
	return ProjectConversationMuxEvent{
		Event:          event.Event,
		ConversationID: conversation.ID,
		Payload:        event.Payload,
		SentAt:         time.Now().UTC(),
	}
}
