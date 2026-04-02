package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestSkillRefinementServiceRetriesThenReturnsVerifiedResultAndCleansUp(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	skillID := uuid.New()
	providerItem := catalogdomain.AgentProvider{
		ID:                providerID,
		OrganizationID:    orgID,
		Name:              "Codex",
		AdapterType:       catalogdomain.AgentProviderAdapterTypeCodexAppServer,
		AvailabilityState: catalogdomain.AgentProviderAvailabilityStateAvailable,
		Available:         true,
		ModelName:         "gpt-5.4",
	}

	runtime := &fakeSkillRefinementRuntime{
		supportFn: func(provider catalogdomain.AgentProvider) bool {
			return provider.AdapterType == catalogdomain.AgentProviderAdapterTypeCodexAppServer
		},
		startFn: func(input RuntimeTurnInput, attempt int) []StreamEvent {
			skillDir := mustResolveProjectedSkillDir(input.WorkingDirectory.String(), "deploy-openase")
			switch attempt {
			case 1:
				mustStat(filepath.Join(skillDir, "SKILL.md"))
				mustMkdirAll(filepath.Join(skillDir, "references"))
				if err := os.WriteFile(
					filepath.Join(skillDir, "references", "attempt-one.md"),
					[]byte("# attempt one\n"),
					0o600,
				); err != nil {
					panic(err)
				}
				return []StreamEvent{
					newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
						"text": "bash -n scripts/check.sh\nscripts/check.sh: syntax error",
					}),
					newTextMessageEvent(`{"type":"skill_refinement_result","status":"blocked","summary":"First verification failed","failure_reason":"shell script still has a syntax error"}`),
				}
			default:
				mustStat(filepath.Join(skillDir, "references", "attempt-one.md"))
				if err := os.WriteFile(
					filepath.Join(skillDir, "scripts", "check.sh"),
					[]byte("#!/usr/bin/env bash\necho verified\n"),
					0o600,
				); err != nil {
					panic(err)
				}
				// #nosec G302 -- test fixture intentionally models an executable skill script.
				if err := os.Chmod(filepath.Join(skillDir, "scripts", "check.sh"), 0o700); err != nil {
					panic(err)
				}
				return []StreamEvent{
					newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
						"text": "bash -n scripts/check.sh\n./scripts/check.sh\nverified",
					}),
					newTextMessageEvent(`{"type":"skill_refinement_result","status":"verified","summary":"Bundle verified after fixing the script","verification_notes":"bash -n passed and the script executed successfully"}`),
				}
			}
		},
	}

	service := NewSkillRefinementService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:                     projectID,
				OrganizationID:         orgID,
				Name:                   "OpenASE",
				DefaultAgentProviderID: &providerID,
			},
			providers: []catalogdomain.AgentProvider{providerItem},
		},
		harnessWorkflowReader{
			skillDetail: workflowservice.SkillDetail{
				Skill: workflowservice.Skill{
					ID:             skillID,
					Name:           "deploy-openase",
					CurrentVersion: 1,
					IsEnabled:      true,
				},
			},
		},
	)

	stream, err := service.Start(context.Background(), UserID("user:skill"), SkillRefinementInput{
		ProjectID: projectID,
		SkillID:   skillID,
		Message:   "Fix the draft script and verify it.",
		DraftFiles: []workflowservice.SkillBundleFileInput{
			{
				Path:    "SKILL.md",
				Content: []byte("---\nname: deploy-openase\ndescription: Safely redeploy OpenASE\n---\n\n# Deploy\n\nRun the check script.\n"),
			},
			{
				Path:         "scripts/check.sh",
				Content:      []byte("#!/usr/bin/env bash\necho broken\n"),
				IsExecutable: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	events := collectStreamEvents(stream.Events)
	if len(events) == 0 {
		t.Fatal("expected skill refinement events")
	}

	sessionPayload, ok := events[0].Payload.(SkillRefinementSessionPayload)
	if !ok {
		t.Fatalf("first payload type = %T, want SkillRefinementSessionPayload", events[0].Payload)
	}
	if !strings.Contains(sessionPayload.WorkspacePath, filepath.Join(".openase", "skill-tests")) {
		t.Fatalf("workspace path = %q", sessionPayload.WorkspacePath)
	}
	if _, err := os.Stat(sessionPayload.WorkspacePath); err != nil {
		t.Fatalf("expected workspace to remain for inspection: %v", err)
	}

	var resultPayload SkillRefinementResultPayload
	for _, event := range events {
		if event.Event != "result" {
			continue
		}
		resultPayload = event.Payload.(SkillRefinementResultPayload)
	}
	if resultPayload.Status != "verified" {
		t.Fatalf("result status = %q, want verified", resultPayload.Status)
	}
	if resultPayload.Attempts != 2 {
		t.Fatalf("result attempts = %d, want 2", resultPayload.Attempts)
	}
	if resultPayload.ProviderThreadID != "thread-2" || resultPayload.ProviderTurnID != "turn-2" {
		t.Fatalf("result anchors = %+v", resultPayload)
	}
	if len(resultPayload.CandidateFiles) != 3 {
		t.Fatalf("candidate files = %d, want 3", len(resultPayload.CandidateFiles))
	}

	sessionID := SessionID(resultPayload.SessionID)
	if !service.CloseSession(UserID("user:skill"), sessionID) {
		t.Fatal("CloseSession() = false, want true")
	}
	if _, err := os.Stat(sessionPayload.WorkspacePath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace cleanup after close, stat err = %v", err)
	}
	if len(runtime.closeCalls) == 0 || runtime.closeCalls[len(runtime.closeCalls)-1] != sessionID {
		t.Fatalf("runtime close calls = %+v, want session %s", runtime.closeCalls, sessionID)
	}
}

func TestSkillRefinementServiceReturnsBlockedResultAfterBoundedAttempts(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	skillID := uuid.New()
	providerItem := catalogdomain.AgentProvider{
		ID:                providerID,
		OrganizationID:    orgID,
		Name:              "Codex",
		AdapterType:       catalogdomain.AgentProviderAdapterTypeCodexAppServer,
		AvailabilityState: catalogdomain.AgentProviderAvailabilityStateAvailable,
		Available:         true,
		ModelName:         "gpt-5.4",
	}

	runtime := &fakeSkillRefinementRuntime{
		supportFn: func(provider catalogdomain.AgentProvider) bool { return true },
		anchorFn: func(_ SessionID, _ int) RuntimeSessionAnchor {
			return RuntimeSessionAnchor{
				ProviderThreadID:      "thread-rich",
				LastTurnID:            "turn-rich",
				ProviderAnchorID:      "thread-rich",
				ProviderAnchorKind:    "thread",
				ProviderTurnSupported: true,
			}
		},
		startFn: func(input RuntimeTurnInput, attempt int) []StreamEvent {
			return []StreamEvent{
				newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
					"text": "bash -n scripts/check.sh\nscripts/check.sh: syntax error",
				}),
				newTextMessageEvent(`{"type":"skill_refinement_result","status":"blocked","summary":"Still blocked","failure_reason":"syntax error remains"}`),
			}
		},
	}

	service := NewSkillRefinementService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:                     projectID,
				OrganizationID:         orgID,
				Name:                   "OpenASE",
				DefaultAgentProviderID: &providerID,
			},
			providers: []catalogdomain.AgentProvider{providerItem},
		},
		harnessWorkflowReader{
			skillDetail: workflowservice.SkillDetail{
				Skill: workflowservice.Skill{
					ID:             skillID,
					Name:           "deploy-openase",
					CurrentVersion: 1,
					IsEnabled:      true,
				},
			},
		},
	)
	service.maxAttempts = 2

	stream, err := service.Start(context.Background(), UserID("user:block"), SkillRefinementInput{
		ProjectID: projectID,
		SkillID:   skillID,
		Message:   "Try to fix it.",
		DraftFiles: []workflowservice.SkillBundleFileInput{
			{
				Path:    "SKILL.md",
				Content: []byte("---\nname: deploy-openase\ndescription: Safely redeploy OpenASE\n---\n\n# Deploy\n\nRun the check script.\n"),
			},
			{
				Path:         "scripts/check.sh",
				Content:      []byte("#!/usr/bin/env bash\necho broken\n"),
				IsExecutable: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	var resultPayload SkillRefinementResultPayload
	for _, event := range collectStreamEvents(stream.Events) {
		if event.Event == "result" {
			resultPayload = event.Payload.(SkillRefinementResultPayload)
		}
	}
	if resultPayload.Status != "blocked" {
		t.Fatalf("result status = %q, want blocked", resultPayload.Status)
	}
	if resultPayload.Attempts != 2 {
		t.Fatalf("result attempts = %d, want 2", resultPayload.Attempts)
	}
	if !strings.Contains(resultPayload.FailureReason, "syntax error remains") {
		t.Fatalf("failure reason = %q", resultPayload.FailureReason)
	}
}

func TestSkillRefinementServiceProviderMatrix(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	orgID := uuid.New()
	codexLocalID := uuid.New()
	claudeLocalID := uuid.New()
	geminiLocalID := uuid.New()
	codexRemoteID := uuid.New()
	project := catalogdomain.Project{
		ID:                     projectID,
		OrganizationID:         orgID,
		Name:                   "OpenASE",
		DefaultAgentProviderID: &claudeLocalID,
	}
	providers := []catalogdomain.AgentProvider{
		{
			ID:             codexLocalID,
			OrganizationID: orgID,
			Name:           "Codex local",
			AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
			MachineHost:    catalogdomain.LocalMachineHost,
			Available:      true,
		},
		{
			ID:             claudeLocalID,
			OrganizationID: orgID,
			Name:           "Claude local",
			AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
			MachineHost:    catalogdomain.LocalMachineHost,
			Available:      true,
		},
		{
			ID:             geminiLocalID,
			OrganizationID: orgID,
			Name:           "Gemini local",
			AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
			MachineHost:    catalogdomain.LocalMachineHost,
			Available:      true,
		},
		{
			ID:             codexRemoteID,
			OrganizationID: orgID,
			Name:           "Codex remote",
			AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
			MachineHost:    "10.0.0.40",
			Available:      true,
		},
	}
	service := NewSkillRefinementService(
		nil,
		&fakeSkillRefinementRuntime{
			supportFn: func(provider catalogdomain.AgentProvider) bool {
				return provider.AdapterType == catalogdomain.AgentProviderAdapterTypeCodexAppServer
			},
		},
		fakeCatalogReader{
			project:   project,
			providers: providers,
		},
		harnessWorkflowReader{},
	)

	t.Run("falls back from unsupported default to local codex", func(t *testing.T) {
		resolved, err := service.resolveProvider(context.Background(), project, nil)
		if err != nil {
			t.Fatalf("resolveProvider() error = %v", err)
		}
		if resolved.ID != codexLocalID {
			t.Fatalf("provider id = %s, want %s", resolved.ID, codexLocalID)
		}
	})

	for _, tc := range []struct {
		name       string
		providerID uuid.UUID
		wantErr    error
		wantReason string
	}{
		{name: "accepts local codex", providerID: codexLocalID},
		{
			name:       "rejects local claude",
			providerID: claudeLocalID,
			wantErr:    ErrProviderUnsupported,
			wantReason: "reason=skill_ai_requires_codex",
		},
		{
			name:       "rejects local gemini",
			providerID: geminiLocalID,
			wantErr:    ErrProviderUnsupported,
			wantReason: "reason=skill_ai_requires_codex",
		},
		{
			name:       "rejects remote codex",
			providerID: codexRemoteID,
			wantErr:    ErrProviderUnsupported,
			wantReason: "reason=remote_machine_not_supported",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolved, err := service.resolveProvider(context.Background(), project, &tc.providerID)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("resolveProvider() error = %v, want %v", err, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantReason) {
					t.Fatalf("expected reason %q in %v", tc.wantReason, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveProvider() error = %v", err)
			}
			if resolved.ID != tc.providerID {
				t.Fatalf("provider id = %s, want %s", resolved.ID, tc.providerID)
			}
		})
	}
}

func TestParseSkillRefinementRuntimeEventCapturesAssistantAndVerificationSignals(t *testing.T) {
	t.Parallel()

	textEvent := newTextMessageEvent(`{"type":"skill_refinement_result","status":"verified","summary":"Verified"}`)
	textParsed := parseSkillRefinementRuntimeEvent(textEvent, false)
	if textParsed.AssistantText == "" {
		t.Fatalf("expected assistant text capture, got %+v", textParsed)
	}
	if len(textParsed.ForwardEvents) != 1 || textParsed.ForwardEvents[0].Event != "message" {
		t.Fatalf("expected text event passthrough, got %+v", textParsed.ForwardEvents)
	}

	taskEvent := newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
		"text": "bash -n scripts/check.sh\n./scripts/check.sh",
	})
	taskParsed := parseSkillRefinementRuntimeEvent(taskEvent, false)
	if taskParsed.CommandOutput != "bash -n scripts/check.sh\n./scripts/check.sh" {
		t.Fatalf("command output = %q", taskParsed.CommandOutput)
	}
	if !taskParsed.EmitTesting {
		t.Fatalf("expected first verification command to trigger testing phase, got %+v", taskParsed)
	}
	if len(taskParsed.ForwardEvents) != 1 || taskParsed.ForwardEvents[0].Event != "message" {
		t.Fatalf("expected task progress passthrough, got %+v", taskParsed.ForwardEvents)
	}

	suppressedTesting := parseSkillRefinementRuntimeEvent(taskEvent, true)
	if suppressedTesting.EmitTesting {
		t.Fatalf("expected later verification command to skip testing phase, got %+v", suppressedTesting)
	}

	runtimeError := parseSkillRefinementRuntimeEvent(StreamEvent{
		Event:   "error",
		Payload: errorPayload{Message: "workspace verification failed"},
	}, false)
	if runtimeError.TurnErr == nil || runtimeError.TurnErr.Error() != "workspace verification failed" {
		t.Fatalf("expected terminal runtime error capture, got %+v", runtimeError)
	}
}

func TestParseSkillRefinementRuntimeEventPassthroughsRichProtocolEvents(t *testing.T) {
	t.Parallel()

	for _, event := range []StreamEvent{
		{
			Event: "session_anchor",
			Payload: RuntimeSessionAnchor{
				ProviderThreadID:      "thread-rich",
				ProviderAnchorID:      "thread-rich",
				ProviderAnchorKind:    "thread",
				ProviderTurnSupported: true,
			},
		},
		{
			Event: "thread_status",
			Payload: runtimeThreadStatusPayload{
				ThreadID:    "thread-rich",
				Status:      "active",
				ActiveFlags: []string{"running"},
			},
		},
		{
			Event: "interrupt_requested",
			Payload: RuntimeInterruptEvent{
				RequestID: "req-1",
				Kind:      "command_execution",
				Payload:   map[string]any{"command": "git status"},
			},
		},
		{
			Event: "session_state",
			Payload: runtimeSessionStatePayload{
				Status:      "requires_action",
				ActiveFlags: []string{"requires_action"},
				Detail:      "Approval required",
				Raw:         map[string]any{"kind": "approval"},
			},
		},
		{
			Event: "plan_updated",
			Payload: runtimePlanUpdatedPayload{
				ThreadID: "thread-rich",
				TurnID:   "turn-rich",
				Plan: []runtimePlanStepPayload{
					{Step: "Inspect", Status: "completed"},
				},
			},
		},
		{
			Event: "diff_updated",
			Payload: runtimeDiffUpdatedPayload{
				ThreadID: "thread-rich",
				TurnID:   "turn-rich",
				Diff:     "diff --git a/SKILL.md b/SKILL.md",
			},
		},
		{
			Event: "reasoning_updated",
			Payload: runtimeReasoningUpdatedPayload{
				ThreadID: "thread-rich",
				TurnID:   "turn-rich",
				ItemID:   "item-1",
				Kind:     "summary_text_delta",
				Delta:    "Reasoning",
			},
		},
	} {
		parsed := parseSkillRefinementRuntimeEvent(event, false)
		if len(parsed.ForwardEvents) != 1 {
			t.Fatalf("expected passthrough for %s, got %+v", event.Event, parsed)
		}
		if parsed.ForwardEvents[0].Event != event.Event {
			t.Fatalf("passthrough event kind = %q, want %q", parsed.ForwardEvents[0].Event, event.Event)
		}
	}
}

func TestSkillRefinementServiceForwardsRuntimeProtocolEvents(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	skillID := uuid.New()
	providerItem := catalogdomain.AgentProvider{
		ID:                providerID,
		OrganizationID:    orgID,
		Name:              "Codex",
		AdapterType:       catalogdomain.AgentProviderAdapterTypeCodexAppServer,
		AvailabilityState: catalogdomain.AgentProviderAvailabilityStateAvailable,
		Available:         true,
		ModelName:         "gpt-5.4",
	}

	runtime := &fakeSkillRefinementRuntime{
		supportFn: func(provider catalogdomain.AgentProvider) bool { return true },
		anchorFn: func(_ SessionID, _ int) RuntimeSessionAnchor {
			return RuntimeSessionAnchor{
				ProviderThreadID:      "thread-rich",
				LastTurnID:            "turn-rich",
				ProviderAnchorID:      "thread-rich",
				ProviderAnchorKind:    "thread",
				ProviderTurnSupported: true,
			}
		},
		startFn: func(input RuntimeTurnInput, attempt int) []StreamEvent {
			skillDir := mustResolveProjectedSkillDir(input.WorkingDirectory.String(), "deploy-openase")
			if err := os.WriteFile(
				filepath.Join(skillDir, "scripts", "check.sh"),
				[]byte("#!/usr/bin/env bash\necho verified\n"),
				0o600,
			); err != nil {
				panic(err)
			}
			return []StreamEvent{
				{
					Event: "session_anchor",
					Payload: RuntimeSessionAnchor{
						ProviderThreadID:      "thread-rich",
						ProviderAnchorID:      "thread-rich",
						ProviderAnchorKind:    "thread",
						ProviderTurnSupported: true,
					},
				},
				{
					Event: "thread_status",
					Payload: runtimeThreadStatusPayload{
						ThreadID:    "thread-rich",
						Status:      "active",
						ActiveFlags: []string{"running"},
					},
				},
				newTaskMessageEvent(chatMessageTypeTaskStarted, map[string]any{
					"thread_id": "thread-rich",
					"turn_id":   "turn-rich",
					"status":    "in_progress",
				}),
				newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
					"text": "bash -n scripts/check.sh\n./scripts/check.sh\nverified",
				}),
				newTaskMessageEvent(chatMessageTypeTaskNotification, map[string]any{
					"tool": "shell",
				}),
				{
					Event: "interrupt_requested",
					Payload: RuntimeInterruptEvent{
						RequestID: "req-rich",
						Kind:      "command_execution",
						Payload:   map[string]any{"command": "git status"},
					},
				},
				{
					Event: "session_state",
					Payload: runtimeSessionStatePayload{
						Status:      "active",
						ActiveFlags: []string{"running"},
						Detail:      "Verification running",
						Raw:         map[string]any{"status": "active"},
					},
				},
				{
					Event: "plan_updated",
					Payload: runtimePlanUpdatedPayload{
						ThreadID: "thread-rich",
						TurnID:   "turn-rich",
						Plan: []runtimePlanStepPayload{
							{Step: "Inspect", Status: "completed"},
							{Step: "Verify", Status: "completed"},
						},
					},
				},
				{
					Event: "diff_updated",
					Payload: runtimeDiffUpdatedPayload{
						ThreadID: "thread-rich",
						TurnID:   "turn-rich",
						Diff:     "diff --git a/SKILL.md b/SKILL.md",
					},
				},
				{
					Event: "reasoning_updated",
					Payload: runtimeReasoningUpdatedPayload{
						ThreadID: "thread-rich",
						TurnID:   "turn-rich",
						ItemID:   "item-1",
						Kind:     "summary_text_delta",
						Delta:    "Reasoning",
					},
				},
				newTextMessageEvent(`{"type":"skill_refinement_result","status":"verified","summary":"Bundle verified after forwarding runtime events","verification_notes":"bash -n passed and the script executed successfully"}`),
			}
		},
	}

	service := NewSkillRefinementService(
		nil,
		runtime,
		fakeCatalogReader{
			project: catalogdomain.Project{
				ID:                     projectID,
				OrganizationID:         orgID,
				Name:                   "OpenASE",
				DefaultAgentProviderID: &providerID,
			},
			providers: []catalogdomain.AgentProvider{providerItem},
		},
		harnessWorkflowReader{
			skillDetail: workflowservice.SkillDetail{
				Skill: workflowservice.Skill{
					ID:             skillID,
					Name:           "deploy-openase",
					CurrentVersion: 1,
					IsEnabled:      true,
				},
			},
		},
	)

	stream, err := service.Start(context.Background(), UserID("user:rich"), SkillRefinementInput{
		ProjectID: projectID,
		SkillID:   skillID,
		Message:   "Tighten the skill and keep the transcript detailed.",
		DraftFiles: []workflowservice.SkillBundleFileInput{
			{
				Path:    "SKILL.md",
				Content: []byte("---\nname: deploy-openase\ndescription: Safely redeploy OpenASE\n---\n\n# Deploy\n\nRun the check script.\n"),
			},
			{
				Path:         "scripts/check.sh",
				Content:      []byte("#!/usr/bin/env bash\necho broken\n"),
				IsExecutable: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	collected := collectStreamEvents(stream.Events)
	eventNames := make([]string, 0, len(collected))
	messageTypes := make([]string, 0, len(collected))
	var resultPayload SkillRefinementResultPayload
	for _, event := range collected {
		eventNames = append(eventNames, event.Event)
		if event.Event == "result" {
			resultPayload = event.Payload.(SkillRefinementResultPayload)
			continue
		}
		if event.Event != "message" {
			continue
		}
		switch payload := event.Payload.(type) {
		case map[string]any:
			messageTypes = append(messageTypes, stringValue(payload["type"]))
		case textPayload:
			messageTypes = append(messageTypes, payload.Type)
		}
	}

	if !containsAll(strings.Join(eventNames, "\n"),
		"session",
		"status",
		"session_anchor",
		"thread_status",
		"message",
		"interrupt_requested",
		"session_state",
		"plan_updated",
		"diff_updated",
		"reasoning_updated",
		"result",
	) {
		t.Fatalf("unexpected stream events: %+v", collected)
	}
	if !containsAll(strings.Join(messageTypes, "\n"),
		chatMessageTypeTaskStarted,
		chatMessageTypeTaskProgress,
		chatMessageTypeTaskNotification,
		chatMessageTypeText,
	) {
		t.Fatalf("unexpected forwarded message types: %+v", messageTypes)
	}
	if resultPayload.Status != "verified" {
		t.Fatalf("result status = %q, want verified", resultPayload.Status)
	}
	if resultPayload.ProviderThreadID != "thread-rich" || resultPayload.ProviderTurnID != "turn-rich" {
		t.Fatalf("result anchors = %+v", resultPayload)
	}
}

type fakeSkillRefinementRuntime struct {
	attempt    int
	closeCalls []SessionID
	supportFn  func(catalogdomain.AgentProvider) bool
	startFn    func(input RuntimeTurnInput, attempt int) []StreamEvent
	anchorFn   func(sessionID SessionID, attempt int) RuntimeSessionAnchor
}

func (r *fakeSkillRefinementRuntime) Supports(provider catalogdomain.AgentProvider) bool {
	if r.supportFn != nil {
		return r.supportFn(provider)
	}
	return true
}

func (r *fakeSkillRefinementRuntime) StartTurn(_ context.Context, input RuntimeTurnInput) (TurnStream, error) {
	r.attempt++
	streamEvents := r.startFn(input, r.attempt)
	events := make(chan StreamEvent, max(1, len(streamEvents)))
	for _, event := range streamEvents {
		events <- event
	}
	close(events)
	return TurnStream{Events: events}, nil
}

func (r *fakeSkillRefinementRuntime) CloseSession(sessionID SessionID) bool {
	r.closeCalls = append(r.closeCalls, sessionID)
	return true
}

func (r *fakeSkillRefinementRuntime) SessionAnchor(sessionID SessionID) RuntimeSessionAnchor {
	if r.anchorFn != nil {
		return r.anchorFn(sessionID, r.attempt)
	}
	return RuntimeSessionAnchor{
		ProviderThreadID: fmt.Sprintf("thread-%d", r.attempt),
		LastTurnID:       fmt.Sprintf("turn-%d", r.attempt),
	}
}

func mustResolveProjectedSkillDir(workspaceRoot string, skillName string) string {
	skillDir := filepath.Join(workspaceRoot, ".codex", "skills", skillName)
	mustStat(skillDir)
	return skillDir
}

func mustStat(path string) {
	if _, err := os.Stat(path); err != nil {
		panic(err)
	}
}

func mustMkdirAll(path string) {
	if err := os.MkdirAll(path, 0o750); err != nil {
		panic(err)
	}
}
