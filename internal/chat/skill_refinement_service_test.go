package chat

import (
	"context"
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

type fakeSkillRefinementRuntime struct {
	attempt    int
	closeCalls []SessionID
	supportFn  func(catalogdomain.AgentProvider) bool
	startFn    func(input RuntimeTurnInput, attempt int) []StreamEvent
}

func (r *fakeSkillRefinementRuntime) Supports(provider catalogdomain.AgentProvider) bool {
	if r.supportFn != nil {
		return r.supportFn(provider)
	}
	return true
}

func (r *fakeSkillRefinementRuntime) StartTurn(_ context.Context, input RuntimeTurnInput) (TurnStream, error) {
	r.attempt++
	events := make(chan StreamEvent, 8)
	for _, event := range r.startFn(input, r.attempt) {
		events <- event
	}
	close(events)
	return TurnStream{Events: events}, nil
}

func (r *fakeSkillRefinementRuntime) CloseSession(sessionID SessionID) bool {
	r.closeCalls = append(r.closeCalls, sessionID)
	return true
}

func (r *fakeSkillRefinementRuntime) SessionAnchor(_ SessionID) RuntimeSessionAnchor {
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
