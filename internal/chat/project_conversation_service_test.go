package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
)

func TestProjectConversationPromptIncludesRecoverySummaryAndTranscript(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	org, project := createProjectConversationTestProject(ctx, t, client)
	repo := chatrepo.NewEntRepository(client)
	conversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	turn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "Continue the ticket split")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	if _, err := repo.AppendEntry(ctx, conversation.ID, &turn.ID, chatdomain.EntryKindAssistantTextDelta, map[string]any{
		"role":    "assistant",
		"content": "We should split this by repository boundary.",
	}); err != nil {
		t.Fatalf("append assistant entry: %v", err)
	}

	conversation, err = repo.UpdateConversationAnchors(
		ctx,
		conversation.ID,
		chatdomain.ConversationStatusActive,
		chatdomain.ConversationAnchors{RollingSummary: "Prior discussion summary"},
	)
	if err != nil {
		t.Fatalf("update anchors: %v", err)
	}

	service := NewProjectConversationService(
		nil,
		repo,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             project.ID,
					OrganizationID: org.ID,
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		nil,
	)

	prompt, err := service.buildProjectConversationPrompt(ctx, conversation, catalogdomain.Project{
		ID:             project.ID,
		OrganizationID: org.ID,
		Name:           "OpenASE",
		Slug:           "openase",
		Description:    "Issue-driven automation",
	}, nil, true)
	if err != nil {
		t.Fatalf("build recovery prompt: %v", err)
	}

	if !containsAll(
		prompt,
		"## Previous conversation",
		"Rolling summary:\nPrior discussion summary",
		"user: Continue the ticket split",
		"assistant: We should split this by repository boundary.",
		"Continue from this conversation state",
	) {
		t.Fatalf("expected recovery prompt context, got %q", prompt)
	}
}

func TestProjectConversationPromptIncludesCurrentFocus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewProjectConversationService(
		nil,
		nil,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
					OrganizationID: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		nil,
	)

	prompt, err := service.buildProjectConversationPrompt(
		ctx,
		chatdomain.Conversation{
			ID:        uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
			ProjectID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		catalogdomain.Project{
			ID:             uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			OrganizationID: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			Name:           "OpenASE",
			Slug:           "openase",
			Description:    "Issue-driven automation",
		},
		&ProjectConversationFocus{
			Kind: ProjectConversationFocusWorkflow,
			Workflow: &ProjectConversationWorkflowFocus{
				ID:            uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
				Name:          "Backend Engineer",
				Type:          "coding",
				HarnessPath:   ".openase/harnesses/backend.md",
				IsActive:      true,
				SelectedArea:  "harness",
				HasDirtyDraft: true,
			},
		},
		false,
	)
	if err != nil {
		t.Fatalf("build project conversation prompt: %v", err)
	}

	if !containsAll(
		prompt,
		"### 当前用户关注区域",
		"- 类型: workflow",
		"- 名称: Backend Engineer",
		"- harness_path: .openase/harnesses/backend.md",
		"- selected_area: harness",
		"- has_dirty_draft: true",
	) {
		t.Fatalf("expected focus context in prompt, got %q", prompt)
	}
}

func TestProjectConversationApplyGitHubWorkspaceAuthInjectsHTTPSCredentials(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	projectID := uuid.New()
	service := NewProjectConversationService(nil, nil, nil, nil, nil, nil, nil)
	service.ConfigureGitHubCredentials(projectConversationStubGitHubTokenResolver{
		resolved: githubauthdomain.ResolvedCredential{Token: "ghu_test"},
	})

	request := workspaceinfra.SetupRequest{
		Repos: []workspaceinfra.RepoRequest{
			{RepositoryURL: "https://github.com/acme/private-repo.git"},
			{RepositoryURL: "git@github.com:acme/private-repo.git"},
			{RepositoryURL: "https://gitlab.com/acme/private-repo.git"},
		},
	}

	updated, err := service.applyGitHubWorkspaceAuth(ctx, projectID, request)
	if err != nil {
		t.Fatalf("applyGitHubWorkspaceAuth() error = %v", err)
	}
	if updated.Repos[0].HTTPBasicAuth == nil {
		t.Fatal("expected GitHub HTTPS repo auth to be injected")
	}
	if updated.Repos[0].HTTPBasicAuth.Username != "x-access-token" || updated.Repos[0].HTTPBasicAuth.Password != "ghu_test" {
		t.Fatalf("unexpected injected auth %+v", updated.Repos[0].HTTPBasicAuth)
	}
	if updated.Repos[1].HTTPBasicAuth != nil {
		t.Fatalf("expected SSH repo to skip injected HTTP auth, got %+v", updated.Repos[1].HTTPBasicAuth)
	}
	if updated.Repos[2].HTTPBasicAuth != nil {
		t.Fatalf("expected non-GitHub repo to skip injected auth, got %+v", updated.Repos[2].HTTPBasicAuth)
	}
}

func TestProjectConversationApplyGitHubWorkspaceAuthIgnoresMissingCredential(t *testing.T) {
	t.Parallel()

	service := NewProjectConversationService(nil, nil, nil, nil, nil, nil, nil)
	service.ConfigureGitHubCredentials(projectConversationStubGitHubTokenResolver{
		err: githubauthservice.ErrCredentialNotConfigured,
	})

	request := workspaceinfra.SetupRequest{
		Repos: []workspaceinfra.RepoRequest{{RepositoryURL: "https://github.com/acme/private-repo.git"}},
	}
	updated, err := service.applyGitHubWorkspaceAuth(context.Background(), uuid.New(), request)
	if err != nil {
		t.Fatalf("applyGitHubWorkspaceAuth() error = %v", err)
	}
	if updated.Repos[0].HTTPBasicAuth != nil {
		t.Fatalf("expected missing credential to leave request unchanged, got %+v", updated.Repos[0].HTTPBasicAuth)
	}
}

func TestProjectConversationStartTurnFallsBackToRecoveryWhenResumeThreadMissing(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	conversation, err = repoStore.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusActive, chatdomain.ConversationAnchors{
		ProviderThreadID:     optionalString("thread-stale"),
		LastTurnID:           optionalString("turn-stale"),
		RollingSummary:       "Prior discussion summary",
		ProviderThreadStatus: optionalString("idle"),
	})
	if err != nil {
		t.Fatalf("update conversation anchors: %v", err)
	}

	fakeCodex := &fakeProjectConversationCodexRuntime{
		ensureErr:   &codexadapter.RPCError{Method: "thread/resume", Code: -32600, Message: "thread not found: thread-stale"},
		startStream: TurnStream{Events: closedStreamEvents()},
	}
	service := NewProjectConversationService(
		nil,
		repoStore,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             project.ID,
					OrganizationID: org.ID,
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
				providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
					providerID: {
						ID:             providerID,
						OrganizationID: org.ID,
						MachineID:      machineID,
						AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
						CliCommand:     "codex",
					},
				},
			},
			machine: catalogdomain.Machine{
				ID:            machineID,
				Name:          catalogdomain.LocalMachineName,
				Host:          catalogdomain.LocalMachineHost,
				WorkspaceRoot: stringPointer(t.TempDir()),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		&fakeAgentCLIProcessManager{process: &fakeAgentCLIProcess{stdin: &trackingWriteCloser{}, stdout: `{"response":"OK"}`}},
		nil,
	)
	service.newCodexRuntime = func(provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		return fakeCodex, nil
	}

	turn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Continue after restart", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if turn.ID == uuid.Nil {
		t.Fatal("expected persisted turn id")
	}
	if fakeCodex.ensureInput.ResumeProviderThreadID != "thread-stale" {
		t.Fatalf("expected resume attempt with stale thread, got %+v", fakeCodex.ensureInput)
	}
	if fakeCodex.startInput.ResumeProviderThreadID != "" || fakeCodex.startInput.ResumeProviderTurnID != "" {
		t.Fatalf("expected fallback start to clear stale anchors, got %+v", fakeCodex.startInput)
	}
	if !strings.Contains(fakeCodex.startInput.SystemPrompt, "Prior discussion summary") {
		t.Fatalf("expected recovery prompt to include rolling summary, got %q", fakeCodex.startInput.SystemPrompt)
	}

	reloaded, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloaded.ProviderThreadID != nil || reloaded.LastTurnID != nil {
		t.Fatalf("expected stale anchors to be cleared, got %+v", reloaded)
	}
	if reloaded.ProviderThreadStatus == nil || *reloaded.ProviderThreadStatus != "notLoaded" {
		t.Fatalf("expected provider thread status notLoaded, got %+v", reloaded.ProviderThreadStatus)
	}
}

func TestProjectConversationStartTurnResumesClaudeSessionWithoutRecoveryPrompt(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	const persistentSessionID = "claude-session-77"
	const rollingSummary = "Claude recovery summary should stay out of the resumed system prompt."
	conversation, err = repoStore.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusActive, chatdomain.ConversationAnchors{
		ProviderThreadID:     optionalString(persistentSessionID),
		RollingSummary:       rollingSummary,
		ProviderThreadStatus: optionalString("idle"),
	})
	if err != nil {
		t.Fatalf("update conversation anchors: %v", err)
	}

	manager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
			stdin: &trackingWriteCloser{},
			stdout: strings.Join([]string{
				fmt.Sprintf(`{"type":"system","subtype":"init","data":{"session_id":"%s"}}`, persistentSessionID),
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Done"}]}}`,
				fmt.Sprintf(`{"type":"result","subtype":"success","session_id":"%s","num_turns":1}`, persistentSessionID),
			}, "\n"),
		},
	}
	service := NewProjectConversationService(
		nil,
		repoStore,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             project.ID,
					OrganizationID: org.ID,
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
				providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
					providerID: {
						ID:             providerID,
						OrganizationID: org.ID,
						MachineID:      machineID,
						AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
						CliCommand:     "claude",
					},
				},
			},
			machine: catalogdomain.Machine{
				ID:            machineID,
				Name:          catalogdomain.LocalMachineName,
				Host:          catalogdomain.LocalMachineHost,
				WorkspaceRoot: stringPointer(t.TempDir()),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		manager,
		nil,
	)

	turn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Continue after restart", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if turn.ID == uuid.Nil {
		t.Fatal("expected persisted turn id")
	}

	joinedArgs := strings.Join(manager.startSpec.Args, "\n")
	if !strings.Contains(joinedArgs, "--resume\n"+persistentSessionID) {
		t.Fatalf("process args = %v, want --resume %s", manager.startSpec.Args, persistentSessionID)
	}
	if strings.Contains(joinedArgs, rollingSummary) {
		t.Fatalf("expected durable claude resume to avoid recovery prompt replay, got args %v", manager.startSpec.Args)
	}
	if !strings.Contains(strings.Join(manager.startSpec.Environment, "\n"), claudeCodeResumeInterruptedTurnEnv+"=1") {
		t.Fatalf("process environment = %v, want %s=1", manager.startSpec.Environment, claudeCodeResumeInterruptedTurnEnv)
	}

	var completedTurn *ent.ChatTurn
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, getErr := client.ChatTurn.Get(ctx, turn.ID)
		if getErr == nil && item.Status == string(chatdomain.TurnStatusCompleted) {
			completedTurn = item
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if completedTurn == nil {
		t.Fatal("expected claude turn to complete")
	}

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.ProviderThreadID == nil || *reloadedConversation.ProviderThreadID != persistentSessionID {
		t.Fatalf("expected claude provider session anchor to persist, got %+v", reloadedConversation)
	}
}

func TestProjectConversationRespondInterruptRestoresCodexSessionWhenRuntimeMissing(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	conversation, err = repoStore.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusInterrupted, chatdomain.ConversationAnchors{
		ProviderThreadID:     optionalString("thread-live"),
		LastTurnID:           optionalString("turn-live"),
		RollingSummary:       "Thread paused for approval",
		ProviderThreadStatus: optionalString("active"),
	})
	if err != nil {
		t.Fatalf("update conversation anchors: %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Need approval")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	interrupt, _, err := repoStore.CreatePendingInterrupt(ctx, conversation.ID, turn.ID, "req-1", chatdomain.InterruptKindCommandExecutionApproval, map[string]any{
		"provider": "codex",
	})
	if err != nil {
		t.Fatalf("create pending interrupt: %v", err)
	}

	fakeCodex := &fakeProjectConversationCodexRuntime{}
	service := NewProjectConversationService(
		nil,
		repoStore,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             project.ID,
					OrganizationID: org.ID,
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
				providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
					providerID: {
						ID:             providerID,
						OrganizationID: org.ID,
						MachineID:      machineID,
						AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
						CliCommand:     "codex",
					},
				},
			},
			machine: catalogdomain.Machine{
				ID:            machineID,
				Name:          catalogdomain.LocalMachineName,
				Host:          catalogdomain.LocalMachineHost,
				WorkspaceRoot: stringPointer(t.TempDir()),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		&fakeAgentCLIProcessManager{process: &fakeAgentCLIProcess{stdin: &trackingWriteCloser{}, stdout: `{"response":"OK"}`}},
		nil,
	)
	service.newCodexRuntime = func(provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		return fakeCodex, nil
	}

	resolved, err := service.RespondInterrupt(ctx, UserID("user:conversation"), conversation.ID, interrupt.ID, chatdomain.InterruptResponse{
		Decision: optionalString("approve_once"),
	})
	if err != nil {
		t.Fatalf("RespondInterrupt() error = %v", err)
	}
	if resolved.Status != chatdomain.InterruptStatusResolved {
		t.Fatalf("expected resolved interrupt, got %+v", resolved)
	}
	if fakeCodex.ensureInput.ResumeProviderThreadID != "thread-live" || fakeCodex.ensureInput.ResumeProviderTurnID != "turn-live" {
		t.Fatalf("expected interrupt recovery to resume thread, got %+v", fakeCodex.ensureInput)
	}
	if fakeCodex.requestID != "req-1" || fakeCodex.kind != "command_execution" || fakeCodex.decision != "approve_once" {
		t.Fatalf("unexpected interrupt response routed to codex runtime: %+v", fakeCodex)
	}
}

func TestProjectConversationConsumeTurnPersistsProviderTurnIDOnCompletion(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Finish the task")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
				Slug:           "openase",
			},
		},
	}, fakeTicketReader{}, harnessWorkflowReader{}, nil, nil)

	events := make(chan StreamEvent, 1)
	events <- StreamEvent{Event: "done", Payload: donePayload{SessionID: conversation.ID.String()}}
	close(events)
	live := &liveProjectConversation{
		codex: &fakeProjectConversationCodexRuntime{
			anchor: RuntimeSessionAnchor{
				ProviderThreadID:          "thread-1",
				LastTurnID:                "provider-turn-1",
				ProviderThreadStatus:      "idle",
				ProviderThreadActiveFlags: []string{},
			},
		},
	}

	service.consumeTurn(ctx, conversation.ID, turn, live, TurnStream{Events: events})

	reloadedTurn, err := client.ChatTurn.Get(ctx, turn.ID)
	if err != nil {
		t.Fatalf("reload turn: %v", err)
	}
	if reloadedTurn.ProviderTurnID == nil || *reloadedTurn.ProviderTurnID != "provider-turn-1" {
		t.Fatalf("expected provider turn id to persist, got %+v", reloadedTurn)
	}
	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.LastTurnID == nil || *reloadedConversation.LastTurnID != "provider-turn-1" {
		t.Fatalf("expected conversation last turn anchor to persist, got %+v", reloadedConversation)
	}
}

func TestProjectConversationConsumeTurnMarksInterruptedStatus(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Interrupt the task")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
				Slug:           "openase",
			},
		},
	}, fakeTicketReader{}, harnessWorkflowReader{}, nil, nil)

	events := make(chan StreamEvent, 1)
	events <- StreamEvent{Event: "interrupted", Payload: errorPayload{Message: "operator interrupted"}}
	close(events)
	live := &liveProjectConversation{
		codex: &fakeProjectConversationCodexRuntime{
			anchor: RuntimeSessionAnchor{
				ProviderThreadID:          "thread-2",
				LastTurnID:                "provider-turn-2",
				ProviderThreadStatus:      "active",
				ProviderThreadActiveFlags: []string{"waitingOnApproval"},
			},
		},
	}

	service.consumeTurn(ctx, conversation.ID, turn, live, TurnStream{Events: events})

	reloadedTurn, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedTurn.Status != chatdomain.ConversationStatusInterrupted {
		t.Fatalf("expected interrupted conversation status, got %+v", reloadedTurn)
	}
	turnItem, err := client.ChatTurn.Get(ctx, turn.ID)
	if err != nil {
		t.Fatalf("reload turn: %v", err)
	}
	if turnItem.Status != string(chatdomain.TurnStatusInterrupted) {
		t.Fatalf("expected interrupted turn status, got %+v", turnItem)
	}
}

func TestProjectConversationCodexResumeInterruptLifecycleAcrossRuntimeRestart(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	conversation, err = repoStore.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusInterrupted, chatdomain.ConversationAnchors{
		ProviderThreadID:          optionalString("thread-shared"),
		LastTurnID:                optionalString("provider-turn-1"),
		ProviderThreadStatus:      optionalString("active"),
		ProviderThreadActiveFlags: &[]string{"waitingOnApproval"},
		RollingSummary:            "Resume the same Codex conversation after restart.",
	})
	if err != nil {
		t.Fatalf("seed conversation anchors: %v", err)
	}
	firstTurn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Need approval before continuing")
	if err != nil {
		t.Fatalf("create first turn: %v", err)
	}
	interrupt, _, err := repoStore.CreatePendingInterrupt(ctx, conversation.ID, firstTurn.ID, "req-acceptance", chatdomain.InterruptKindCommandExecutionApproval, map[string]any{
		"provider": "codex",
	})
	if err != nil {
		t.Fatalf("create pending interrupt: %v", err)
	}

	runtimes := []*fakeProjectConversationCodexRuntime{
		{
			anchor: RuntimeSessionAnchor{
				ProviderThreadID:          "thread-shared",
				LastTurnID:                "provider-turn-1",
				ProviderThreadStatus:      "active",
				ProviderThreadActiveFlags: []string{"waitingOnApproval"},
			},
		},
		{
			startStream: TurnStream{Events: streamWithEvents(
				StreamEvent{Event: "done", Payload: donePayload{SessionID: conversation.ID.String()}},
			)},
			anchor: RuntimeSessionAnchor{
				ProviderThreadID:          "thread-shared",
				LastTurnID:                "provider-turn-2",
				ProviderThreadStatus:      "idle",
				ProviderThreadActiveFlags: []string{},
			},
		},
	}
	runtimeIndex := 0
	service := NewProjectConversationService(
		nil,
		repoStore,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             project.ID,
					OrganizationID: org.ID,
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
				providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
					providerID: {
						ID:             providerID,
						OrganizationID: org.ID,
						MachineID:      machineID,
						AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
						CliCommand:     "codex",
					},
				},
			},
			machine: catalogdomain.Machine{
				ID:            machineID,
				Name:          catalogdomain.LocalMachineName,
				Host:          catalogdomain.LocalMachineHost,
				WorkspaceRoot: stringPointer(t.TempDir()),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		&fakeAgentCLIProcessManager{process: &fakeAgentCLIProcess{stdin: &trackingWriteCloser{}, stdout: `{"response":"OK"}`}},
		nil,
	)
	service.newCodexRuntime = func(provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		if runtimeIndex >= len(runtimes) {
			return runtimes[len(runtimes)-1], nil
		}
		runtime := runtimes[runtimeIndex]
		runtimeIndex++
		return runtime, nil
	}

	resolved, err := service.RespondInterrupt(ctx, UserID("user:conversation"), conversation.ID, interrupt.ID, chatdomain.InterruptResponse{
		Decision: optionalString("approve_once"),
	})
	if err != nil {
		t.Fatalf("RespondInterrupt() error = %v", err)
	}
	if resolved.Status != chatdomain.InterruptStatusResolved {
		t.Fatalf("expected resolved interrupt, got %+v", resolved)
	}
	if runtimes[0].ensureInput.ResumeProviderThreadID != "thread-shared" {
		t.Fatalf("expected interrupt resume to target shared thread, got %+v", runtimes[0].ensureInput)
	}
	if _, err := repoStore.CompleteTurn(ctx, firstTurn.ID, chatdomain.TurnStatusCompleted, optionalString("provider-turn-1")); err != nil {
		t.Fatalf("complete resumed first turn: %v", err)
	}

	service.liveMu.Lock()
	delete(service.live, conversation.ID)
	service.liveMu.Unlock()

	secondTurn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Continue after approval", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if secondTurn.ID == uuid.Nil {
		t.Fatal("expected second persisted turn id")
	}
	if runtimes[1].ensureInput.ResumeProviderThreadID != "thread-shared" {
		t.Fatalf("expected resumed thread for second turn, got %+v", runtimes[1].ensureInput)
	}
	if strings.Contains(runtimes[1].startInput.SystemPrompt, "Resume the same Codex conversation after restart.") {
		t.Fatalf("expected resume path to avoid recovery prompt replay, got %q", runtimes[1].startInput.SystemPrompt)
	}

	var completedTurn *ent.ChatTurn
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, getErr := client.ChatTurn.Get(ctx, secondTurn.ID)
		if getErr == nil && item.ProviderTurnID != nil && *item.ProviderTurnID == "provider-turn-2" && item.Status == string(chatdomain.TurnStatusCompleted) {
			completedTurn = item
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if completedTurn == nil {
		t.Fatal("expected second turn to complete with provider-turn-2")
	}
	var reloadedConversation chatdomain.Conversation
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, getErr := repoStore.GetConversation(ctx, conversation.ID)
		if getErr == nil && item.LastTurnID != nil && *item.LastTurnID == "provider-turn-2" {
			reloadedConversation = item
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if reloadedConversation.ID == uuid.Nil {
		t.Fatal("expected conversation anchors to advance to provider-turn-2")
	}
	if reloadedConversation.ProviderThreadID == nil || *reloadedConversation.ProviderThreadID != "thread-shared" {
		t.Fatalf("expected conversation to stay on the same provider thread, got %+v", reloadedConversation)
	}
	if reloadedConversation.LastTurnID == nil || *reloadedConversation.LastTurnID != "provider-turn-2" {
		t.Fatalf("expected conversation anchor to advance to provider-turn-2, got %+v", reloadedConversation)
	}
}

func TestProjectConversationWatchConversationSessionIncludesProviderAnchors(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	_, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	conversation, err = repoStore.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusActive, chatdomain.ConversationAnchors{
		ProviderThreadID:          optionalString("thread-visible"),
		LastTurnID:                optionalString("turn-visible"),
		ProviderThreadStatus:      optionalString("waitingOnApproval"),
		ProviderThreadActiveFlags: &[]string{"waitingOnApproval"},
		RollingSummary:            "summary",
	})
	if err != nil {
		t.Fatalf("update anchors: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	events, cleanup := service.WatchConversation(ctx, conversation.ID)
	defer cleanup()

	first := <-events
	if first.Event != "session" {
		t.Fatalf("expected session event, got %+v", first)
	}
	payload, ok := first.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map session payload, got %#v", first.Payload)
	}
	if payload["provider_thread_id"] != "thread-visible" || payload["last_turn_id"] != "turn-visible" || payload["provider_thread_status"] != "waitingOnApproval" {
		t.Fatalf("expected provider anchors in session payload, got %#v", payload)
	}
	flags, ok := payload["provider_thread_active_flags"].([]string)
	if !ok || len(flags) != 1 || flags[0] != "waitingOnApproval" {
		t.Fatalf("expected active flags in session payload, got %#v", payload["provider_thread_active_flags"])
	}
}

func TestProjectConversationWatchConversationSessionDistinguishesClaudeSessionAnchors(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	conversation, err = repoStore.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusActive, chatdomain.ConversationAnchors{
		ProviderThreadID:          optionalString("claude-session-visible"),
		ProviderThreadStatus:      optionalString("requires_action"),
		ProviderThreadActiveFlags: &[]string{"requires_action"},
	})
	if err != nil {
		t.Fatalf("update anchors: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
			},
			providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
				providerID: {
					ID:             providerID,
					OrganizationID: org.ID,
					Name:           "Claude Code",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
					CliCommand:     "claude",
					Available:      true,
				},
			},
		},
	}, nil, nil, nil, nil)
	events, cleanup := service.WatchConversation(ctx, conversation.ID)
	defer cleanup()

	first := <-events
	if first.Event != "session" {
		t.Fatalf("expected session event, got %+v", first)
	}
	payload, ok := first.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map session payload, got %#v", first.Payload)
	}
	if payload["provider_anchor_kind"] != "session" || payload["provider_anchor_id"] != "claude-session-visible" {
		t.Fatalf("expected claude session anchor payload, got %#v", payload)
	}
	if supported, ok := payload["provider_turn_supported"].(bool); !ok || supported {
		t.Fatalf("expected provider_turn_supported=false, got %#v", payload["provider_turn_supported"])
	}
	if payload["provider_status"] != "requires_action" {
		t.Fatalf("expected provider_status requires_action, got %#v", payload["provider_status"])
	}
	flags, ok := payload["provider_active_flags"].([]string)
	if !ok || len(flags) != 1 || flags[0] != "requires_action" {
		t.Fatalf("expected provider_active_flags in session payload, got %#v", payload["provider_active_flags"])
	}
}

func TestProjectConversationConsumeTurnPersistsThreadStatusAndProtocolEvents(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Continue")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
				Slug:           "openase",
			},
		},
	}, fakeTicketReader{}, harnessWorkflowReader{}, nil, nil)
	watched, cleanup := service.WatchConversation(ctx, conversation.ID)

	streamEvents := make(chan StreamEvent, 5)
	streamEvents <- StreamEvent{
		Event: "thread_status",
		Payload: runtimeThreadStatusPayload{
			ThreadID:    "thread-77",
			Status:      "waitingOnUserInput",
			ActiveFlags: []string{"waitingOnUserInput"},
		},
	}
	streamEvents <- StreamEvent{
		Event: "plan_updated",
		Payload: runtimePlanUpdatedPayload{
			ThreadID:    "thread-77",
			TurnID:      "provider-turn-77",
			Explanation: optionalString("Need two steps"),
			Plan: []runtimePlanStepPayload{
				{Step: "Inspect", Status: "completed"},
				{Step: "Patch", Status: "in_progress"},
			},
		},
	}
	streamEvents <- StreamEvent{
		Event: "diff_updated",
		Payload: runtimeDiffUpdatedPayload{
			ThreadID: "thread-77",
			TurnID:   "provider-turn-77",
			Diff:     "diff --git a/app.go b/app.go",
		},
	}
	streamEvents <- StreamEvent{
		Event: "reasoning_updated",
		Payload: runtimeReasoningUpdatedPayload{
			ThreadID:     "thread-77",
			TurnID:       "provider-turn-77",
			ItemID:       "item-1",
			Kind:         "summary_text_delta",
			Delta:        "Reasoning text",
			SummaryIndex: intPointer(0),
		},
	}
	streamEvents <- StreamEvent{Event: "done", Payload: donePayload{SessionID: conversation.ID.String()}}
	close(streamEvents)

	live := &liveProjectConversation{
		codex: &fakeProjectConversationCodexRuntime{
			anchor: RuntimeSessionAnchor{
				ProviderThreadID:          "thread-77",
				LastTurnID:                "provider-turn-77",
				ProviderThreadStatus:      "waitingOnUserInput",
				ProviderThreadActiveFlags: []string{"waitingOnUserInput"},
			},
		},
	}

	service.consumeTurn(ctx, conversation.ID, turn, live, TurnStream{Events: streamEvents})
	cleanup()

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.ProviderThreadStatus == nil || *reloadedConversation.ProviderThreadStatus != "waitingOnUserInput" {
		t.Fatalf("expected persisted provider thread status, got %+v", reloadedConversation.ProviderThreadStatus)
	}
	entries, err := repoStore.ListEntries(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	serialized := make([]string, 0, len(entries))
	for _, entry := range entries {
		serialized = append(serialized, string(entry.Kind)+":"+stringValue(entry.Payload["type"]))
	}
	if !containsAll(strings.Join(serialized, "\n"),
		"system:thread_status",
		"system:turn_plan_updated",
		"system:turn_diff_updated",
		"system:turn_reasoning_updated",
	) {
		t.Fatalf("expected protocol event entries, got %v", serialized)
	}

	collected := collectStreamEvents(watched)
	if len(collected) < 5 {
		t.Fatalf("expected broadcast events, got %d", len(collected))
	}
	eventNames := make([]string, 0, len(collected))
	for _, item := range collected {
		eventNames = append(eventNames, item.Event)
	}
	if !containsAll(strings.Join(eventNames, "\n"),
		"session",
		"message",
		"thread_status",
		"plan_updated",
		"diff_updated",
		"reasoning_updated",
		"turn_done",
	) {
		t.Fatalf("unexpected stream sequence: %+v", collected)
	}
}

func TestProjectConversationConsumeTurnPersistsInterruptAndSummary(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	_, project := createProjectConversationTestProject(ctx, t, client)
	repo := chatrepo.NewEntRepository(client)
	conversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	turn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "Inspect the repository state")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}

	service := NewProjectConversationService(nil, repo, nil, nil, nil, nil, nil)
	events, cleanup := service.WatchConversation(ctx, conversation.ID)

	streamEvents := make(chan StreamEvent, 3)
	streamEvents <- StreamEvent{
		Event:   "message",
		Payload: textPayload{Type: chatMessageTypeText, Content: "Repository scan complete."},
	}
	streamEvents <- StreamEvent{
		Event: "interrupt_requested",
		Payload: RuntimeInterruptEvent{
			RequestID: "req-1",
			Kind:      "command_execution",
			Options: []RuntimeInterruptDecision{
				{ID: "approve_once", Label: "Approve once"},
			},
			Payload: map[string]any{"command": "git status"},
		},
	}
	streamEvents <- StreamEvent{
		Event:   "done",
		Payload: donePayload{SessionID: conversation.ID.String(), CostUSD: floatPointer(0.2)},
	}
	close(streamEvents)

	service.consumeTurn(ctx, conversation.ID, turn, &liveProjectConversation{}, TurnStream{Events: streamEvents})
	cleanup()

	pending, err := repo.ListPendingInterrupts(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("list pending interrupts: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending interrupt count = %d, want 1", len(pending))
	}
	if pending[0].ProviderRequestID != "req-1" {
		t.Fatalf("provider request id = %q, want req-1", pending[0].ProviderRequestID)
	}
	if pending[0].Kind != chatdomain.InterruptKindCommandExecutionApproval {
		t.Fatalf("interrupt kind = %q, want command execution approval", pending[0].Kind)
	}

	conversation, err = repo.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	if conversation.Status != chatdomain.ConversationStatusActive {
		t.Fatalf("conversation status = %q, want active", conversation.Status)
	}
	if !containsAll(
		conversation.RollingSummary,
		"user: Inspect the repository state",
		"assistant: Repository scan complete.",
		"system: turn paused for interrupt",
	) {
		t.Fatalf("rolling summary = %q", conversation.RollingSummary)
	}

	collected := collectStreamEvents(events)
	if len(collected) < 4 {
		t.Fatalf("stream event count = %d, want at least 4", len(collected))
	}
	if collected[0].Event != "session" {
		t.Fatalf("first stream event = %q, want session", collected[0].Event)
	}
	if collected[2].Event != "interrupt_requested" {
		t.Fatalf("third stream event = %q, want interrupt_requested", collected[2].Event)
	}
	if collected[len(collected)-1].Event != "turn_done" {
		t.Fatalf("last stream event = %q, want turn_done", collected[len(collected)-1].Event)
	}
}

func TestProjectConversationRespondInterruptRoutesExactRequestID(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	_, project := createProjectConversationTestProject(ctx, t, client)
	repo := chatrepo.NewEntRepository(client)
	conversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	turn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "Handle approvals")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	firstInterrupt, _, err := repo.CreatePendingInterrupt(
		ctx,
		conversation.ID,
		turn.ID,
		"req-first",
		chatdomain.InterruptKindCommandExecutionApproval,
		map[string]any{"command": "git status"},
	)
	if err != nil {
		t.Fatalf("create first interrupt: %v", err)
	}
	secondInterrupt, _, err := repo.CreatePendingInterrupt(
		ctx,
		conversation.ID,
		turn.ID,
		"req-second",
		chatdomain.InterruptKindFileChangeApproval,
		map[string]any{"file": "README.md"},
	)
	if err != nil {
		t.Fatalf("create second interrupt: %v", err)
	}

	service := NewProjectConversationService(nil, repo, nil, nil, nil, nil, nil)
	codexRuntime := &fakeProjectConversationCodexRuntime{}
	service.live[conversation.ID] = &liveProjectConversation{codex: codexRuntime}
	events, cleanup := service.WatchConversation(ctx, conversation.ID)

	resolved, err := service.RespondInterrupt(
		ctx,
		UserID("user:conversation"),
		conversation.ID,
		secondInterrupt.ID,
		chatdomain.InterruptResponse{
			Decision: stringPointer("approve_once"),
			Answer:   map[string]any{"reason": "looks good"},
		},
	)
	if err != nil {
		t.Fatalf("respond interrupt: %v", err)
	}
	cleanup()

	if resolved.ID != secondInterrupt.ID {
		t.Fatalf("resolved interrupt = %s, want %s", resolved.ID, secondInterrupt.ID)
	}
	if codexRuntime.requestID != "req-second" {
		t.Fatalf("runtime request id = %q, want req-second", codexRuntime.requestID)
	}
	if codexRuntime.kind != "file_change" {
		t.Fatalf("runtime interrupt kind = %q, want file_change", codexRuntime.kind)
	}
	if codexRuntime.decision != "approve_once" {
		t.Fatalf("runtime decision = %q, want approve_once", codexRuntime.decision)
	}
	if codexRuntime.answer["reason"] != "looks good" {
		t.Fatalf("runtime answer = %#v", codexRuntime.answer)
	}

	stillPending, err := repo.GetPendingInterrupt(ctx, firstInterrupt.ID)
	if err != nil {
		t.Fatalf("get first interrupt: %v", err)
	}
	if stillPending.Status != chatdomain.InterruptStatusPending {
		t.Fatalf("first interrupt status = %q, want pending", stillPending.Status)
	}

	nowResolved, err := repo.GetPendingInterrupt(ctx, secondInterrupt.ID)
	if err != nil {
		t.Fatalf("get second interrupt: %v", err)
	}
	if nowResolved.Status != chatdomain.InterruptStatusResolved {
		t.Fatalf("second interrupt status = %q, want resolved", nowResolved.Status)
	}

	collected := collectStreamEvents(events)
	if len(collected) < 2 {
		t.Fatalf("stream event count = %d, want at least 2", len(collected))
	}
	if collected[len(collected)-1].Event != "interrupt_resolved" {
		t.Fatalf("last stream event = %q, want interrupt_resolved", collected[len(collected)-1].Event)
	}
}

func TestProjectConversationStartTurnKeepsOtherLiveConversationsRunning(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	org, project := createProjectConversationTestProject(ctx, t, client)
	repo := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()

	firstConversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}

	workspaceRoot := t.TempDir()
	providerItem := catalogdomain.AgentProvider{
		ID:             providerID,
		OrganizationID: org.ID,
		MachineID:      machineID,
		AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
		CliCommand:     "gemini",
	}
	catalog := fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
				Slug:           "openase",
				Description:    "Issue-driven automation",
			},
			providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
				providerID: providerItem,
			},
		},
		machine: catalogdomain.Machine{
			ID:            machineID,
			Name:          catalogdomain.LocalMachineName,
			Host:          catalogdomain.LocalMachineHost,
			WorkspaceRoot: stringPointer(workspaceRoot),
		},
	}
	service := NewProjectConversationService(
		nil,
		repo,
		catalog,
		fakeTicketReader{},
		harnessWorkflowReader{},
		&fakeAgentCLIProcessManager{
			process: &fakeAgentCLIProcess{
				stdin:  &trackingWriteCloser{},
				stdout: `{"response":"OK"}`,
			},
		},
		nil,
	)

	previousRuntime := &fakeRuntime{closeResult: true}
	service.live[firstConversation.ID] = &liveProjectConversation{runtime: previousRuntime}

	if _, err := service.StartTurn(ctx, UserID("user:conversation"), secondConversation.ID, "Switch to this conversation", nil); err != nil {
		t.Fatalf("start second conversation turn: %v", err)
	}

	if len(previousRuntime.closeCalls) != 0 {
		t.Fatalf("previous runtime close calls = %+v, want none", previousRuntime.closeCalls)
	}

	updatedFirst, err := repo.GetConversation(ctx, firstConversation.ID)
	if err != nil {
		t.Fatalf("get first conversation: %v", err)
	}
	if updatedFirst.Status != chatdomain.ConversationStatusActive {
		t.Fatalf("first conversation status = %q, want active", updatedFirst.Status)
	}

	if service.live[firstConversation.ID] == nil {
		t.Fatal("expected first conversation live runtime to remain registered")
	}
	if service.live[secondConversation.ID] == nil {
		t.Fatal("expected second conversation live runtime to be registered")
	}
}

func TestProjectConversationStartTurnRejectsSecondActiveTurnInSameConversation(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	org, project := createProjectConversationTestProject(ctx, t, client)
	repo := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	conversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if _, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "First turn is still running"); err != nil {
		t.Fatalf("seed active turn: %v", err)
	}

	workspaceRoot := t.TempDir()
	service := NewProjectConversationService(
		nil,
		repo,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             project.ID,
					OrganizationID: org.ID,
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
				providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
					providerID: {
						ID:             providerID,
						OrganizationID: org.ID,
						MachineID:      machineID,
						AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
						CliCommand:     "gemini",
					},
				},
			},
			machine: catalogdomain.Machine{
				ID:            machineID,
				Name:          catalogdomain.LocalMachineName,
				Host:          catalogdomain.LocalMachineHost,
				WorkspaceRoot: stringPointer(workspaceRoot),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		&fakeAgentCLIProcessManager{
			process: &fakeAgentCLIProcess{
				stdin:  &trackingWriteCloser{},
				stdout: `{"response":"OK"}`,
			},
		},
		nil,
	)

	_, err = service.StartTurn(
		ctx,
		UserID("user:conversation"),
		conversation.ID,
		"Second turn should be rejected",
		nil,
	)
	if !errors.Is(err, ErrConversationTurnActive) {
		t.Fatalf("expected ErrConversationTurnActive, got %v", err)
	}
}

func TestProjectConversationStartTurnPreparesWorkspaceSkillsAndPlatformEnvironment(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	org, project := createProjectConversationTestProject(ctx, t, client)
	repo := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	conversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	remoteRepoPath, _ := createConversationRemoteRepo(t, "main", map[string]string{
		"README.md": "project ai repo",
	})
	workspaceRoot := t.TempDir()
	processManager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
			stdin:  &trackingWriteCloser{},
			stdout: `{"response":"OK"}`,
		},
	}
	workflowSync := fakeProjectConversationWorkflowSync{}
	catalog := fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
				Slug:           "openase",
				Description:    "Issue-driven automation",
			},
			projectRepos: []catalogdomain.ProjectRepo{
				{
					ID:            uuid.New(),
					ProjectID:     project.ID,
					Name:          "backend",
					RepositoryURL: remoteRepoPath,
					DefaultBranch: "main",
				},
			},
			providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
				providerID: {
					ID:             providerID,
					OrganizationID: org.ID,
					MachineID:      machineID,
					AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
					CliCommand:     "gemini",
				},
			},
		},
		machine: catalogdomain.Machine{
			ID:            machineID,
			Name:          catalogdomain.LocalMachineName,
			Host:          catalogdomain.LocalMachineHost,
			WorkspaceRoot: stringPointer(workspaceRoot),
		},
	}

	service := NewProjectConversationService(
		nil,
		repo,
		catalog,
		fakeTicketReader{},
		workflowSync,
		processManager,
		nil,
	)
	service.ConfigurePlatformEnvironment("http://127.0.0.1:19836/api/v1/platform", fakeProjectConversationAgentPlatform{})

	if _, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Inspect the project", nil); err != nil {
		t.Fatalf("start conversation turn: %v", err)
	}

	workspacePath := filepath.Join(
		workspaceRoot,
		org.ID.String(),
		"openase",
		projectConversationWorkspaceName(conversation.ID),
	)
	assertConversationFileExists(t, filepath.Join(workspacePath, "backend", "README.md"))
	assertConversationFileExists(t, filepath.Join(workspacePath, ".openase", "bin", "openase"))
	assertConversationFileExists(t, filepath.Join(workspacePath, ".agent", "skills", "openase-platform", "SKILL.md"))

	repository, err := git.PlainOpen(filepath.Join(workspacePath, "backend"))
	if err != nil {
		t.Fatalf("open prepared repo: %v", err)
	}
	head, err := repository.Head()
	if err != nil {
		t.Fatalf("repository head: %v", err)
	}
	if head.Name().Short() != "agent/"+projectConversationWorkspaceName(conversation.ID) {
		t.Fatalf("head branch = %q", head.Name().Short())
	}

	environment := processManager.startSpec.Environment
	if !containsEnvironmentPrefix(environment, "OPENASE_REAL_BIN=") {
		t.Fatalf("expected OPENASE_REAL_BIN in environment, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_API_URL=http://127.0.0.1:19836/api/v1/platform") {
		t.Fatalf("expected OPENASE_API_URL in environment, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_AGENT_TOKEN=project-conversation-placeholder") {
		t.Fatalf("expected OPENASE_AGENT_TOKEN in environment, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_PROJECT_ID="+project.ID.String()) {
		t.Fatalf("expected OPENASE_PROJECT_ID in environment, got %+v", environment)
	}
}

func TestProjectConversationWorkspaceDiffSummary(t *testing.T) {
	t.Parallel()

	t.Run("clean workspace", func(t *testing.T) {
		t.Parallel()

		fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
			{
				name:  "backend",
				files: map[string]string{"README.md": "line one\nline two\n"},
			},
		})

		summary, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("GetWorkspaceDiff() error = %v", err)
		}
		if summary.WorkspacePath != fixture.workspacePath {
			t.Fatalf("workspace path = %q, want %q", summary.WorkspacePath, fixture.workspacePath)
		}
		if summary.Dirty || summary.ReposChanged != 0 || summary.FilesChanged != 0 || len(summary.Repos) != 0 {
			t.Fatalf("expected clean workspace summary, got %+v", summary)
		}
	})

	t.Run("modified tracked file", func(t *testing.T) {
		t.Parallel()

		fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
			{
				name:  "backend",
				files: map[string]string{"README.md": "line one\nline two\n"},
			},
		})
		writeConversationWorkspaceFile(t, filepath.Join(fixture.repoPaths["backend"], "README.md"), "line one\nline two\nline three\n")

		summary, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("GetWorkspaceDiff() error = %v", err)
		}
		if !summary.Dirty || summary.ReposChanged != 1 || summary.FilesChanged != 1 || summary.Added != 1 || summary.Removed != 0 {
			t.Fatalf("unexpected modified summary: %+v", summary)
		}
		repo := summary.Repos[0]
		if repo.Name != "backend" || repo.Branch != "agent/"+projectConversationWorkspaceName(fixture.conversation.ID) {
			t.Fatalf("unexpected repo summary: %+v", repo)
		}
		if repo.Path != "backend" || repo.FilesChanged != 1 || repo.Added != 1 || repo.Removed != 0 {
			t.Fatalf("unexpected repo totals: %+v", repo)
		}
		if len(repo.Files) != 1 || repo.Files[0].Path != "README.md" || repo.Files[0].Status != ProjectConversationWorkspaceFileStatusModified || repo.Files[0].Added != 1 || repo.Files[0].Removed != 0 {
			t.Fatalf("unexpected file diff: %+v", repo.Files)
		}
	})

	t.Run("untracked file", func(t *testing.T) {
		t.Parallel()

		fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
			{
				name:  "backend",
				files: map[string]string{"README.md": "line one\n"},
			},
		})
		writeConversationWorkspaceFile(t, filepath.Join(fixture.repoPaths["backend"], "notes.txt"), "note one\nnote two\n")

		summary, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("GetWorkspaceDiff() error = %v", err)
		}
		repo := summary.Repos[0]
		if repo.Added != 2 || repo.Removed != 0 {
			t.Fatalf("unexpected untracked totals: %+v", repo)
		}
		if len(repo.Files) != 1 || repo.Files[0].Path != "notes.txt" || repo.Files[0].Status != ProjectConversationWorkspaceFileStatusUntracked || repo.Files[0].Added != 2 || repo.Files[0].Removed != 0 {
			t.Fatalf("unexpected untracked file diff: %+v", repo.Files)
		}
	})

	t.Run("deleted file", func(t *testing.T) {
		t.Parallel()

		fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
			{
				name:  "backend",
				files: map[string]string{"README.md": "line one\nline two\n"},
			},
		})
		if err := os.Remove(filepath.Join(fixture.repoPaths["backend"], "README.md")); err != nil {
			t.Fatalf("remove tracked file: %v", err)
		}

		summary, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("GetWorkspaceDiff() error = %v", err)
		}
		repo := summary.Repos[0]
		if repo.Added != 0 || repo.Removed != 2 {
			t.Fatalf("unexpected deleted totals: %+v", repo)
		}
		if len(repo.Files) != 1 || repo.Files[0].Path != "README.md" || repo.Files[0].Status != ProjectConversationWorkspaceFileStatusDeleted || repo.Files[0].Added != 0 || repo.Files[0].Removed != 2 {
			t.Fatalf("unexpected deleted file diff: %+v", repo.Files)
		}
	})

	t.Run("multi repo workspace summary", func(t *testing.T) {
		t.Parallel()

		fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
			{
				name:             "backend",
				workspaceDirname: "services/backend",
				files:            map[string]string{"README.md": "backend\n"},
			},
			{
				name:             "frontend",
				workspaceDirname: "apps/frontend",
				files:            map[string]string{"src/app.ts": "export const app = 1\n"},
			},
		})
		writeConversationWorkspaceFile(t, filepath.Join(fixture.repoPaths["backend"], "README.md"), "backend\nupdated\n")
		writeConversationWorkspaceFile(t, filepath.Join(fixture.repoPaths["frontend"], "src/new.ts"), "export const next = 2\n")

		summary, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("GetWorkspaceDiff() error = %v", err)
		}
		if !summary.Dirty || summary.ReposChanged != 2 || summary.FilesChanged != 2 || summary.Added != 2 || summary.Removed != 0 {
			t.Fatalf("unexpected multi-repo summary: %+v", summary)
		}
		paths := []string{summary.Repos[0].Path, summary.Repos[1].Path}
		if !containsAll(strings.Join(paths, ","), "apps/frontend", "services/backend") {
			t.Fatalf("unexpected repo paths: %+v", summary.Repos)
		}
	})

	t.Run("runtime closed workspace still reports dirty state", func(t *testing.T) {
		t.Parallel()

		fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
			{
				name:  "backend",
				files: map[string]string{"README.md": "line one\n"},
			},
		})
		workspacePath, err := provider.ParseAbsolutePath(fixture.workspacePath)
		if err != nil {
			t.Fatalf("parse workspace path: %v", err)
		}
		fixture.service.live[fixture.conversation.ID] = &liveProjectConversation{
			runtime:   &fakeRuntime{closeResult: true},
			workspace: workspacePath,
		}
		writeConversationWorkspaceFile(t, filepath.Join(fixture.repoPaths["backend"], "README.md"), "line one\nline two\n")
		if err := fixture.service.CloseRuntime(fixture.ctx, UserID("user:conversation"), fixture.conversation.ID); err != nil {
			t.Fatalf("CloseRuntime() error = %v", err)
		}

		summary, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("GetWorkspaceDiff() error = %v", err)
		}
		if !summary.Dirty || summary.ReposChanged != 1 || summary.Repos[0].Files[0].Added != 1 {
			t.Fatalf("unexpected closed-runtime summary: %+v", summary)
		}
	})

	t.Run("same conversation lookup is stable", func(t *testing.T) {
		t.Parallel()

		fixture := setupProjectConversationWorkspaceDiffFixture(t, []projectConversationWorkspaceRepoFixture{
			{
				name:  "backend",
				files: map[string]string{"README.md": "line one\n"},
			},
		})
		writeConversationWorkspaceFile(t, filepath.Join(fixture.repoPaths["backend"], "README.md"), "line one\nline two\n")

		first, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("first GetWorkspaceDiff() error = %v", err)
		}
		second, err := fixture.service.GetWorkspaceDiff(
			fixture.ctx,
			UserID("user:conversation"),
			fixture.conversation.ID,
		)
		if err != nil {
			t.Fatalf("second GetWorkspaceDiff() error = %v", err)
		}

		if first.WorkspacePath != second.WorkspacePath || first.Added != second.Added || first.Removed != second.Removed || len(first.Repos) != len(second.Repos) {
			t.Fatalf("expected stable workspace summaries, got first=%+v second=%+v", first, second)
		}
	})
}

type fakeProjectConversationCatalog struct {
	fakeCatalogReader
	machine    catalogdomain.Machine
	machineErr error
}

func (c fakeProjectConversationCatalog) GetMachine(context.Context, uuid.UUID) (catalogdomain.Machine, error) {
	return c.machine, c.machineErr
}

type fakeProjectConversationCodexRuntime struct {
	sessionID   SessionID
	requestID   string
	kind        string
	decision    string
	answer      map[string]any
	ensureInput RuntimeTurnInput
	ensureErr   error
	startInput  RuntimeTurnInput
	startStream TurnStream
	anchor      RuntimeSessionAnchor
}

func (r *fakeProjectConversationCodexRuntime) Supports(catalogdomain.AgentProvider) bool {
	return true
}

func (r *fakeProjectConversationCodexRuntime) StartTurn(_ context.Context, input RuntimeTurnInput) (TurnStream, error) {
	r.startInput = input
	if r.startStream.Events == nil {
		return TurnStream{Events: closedStreamEvents()}, nil
	}
	return r.startStream, nil
}

func (r *fakeProjectConversationCodexRuntime) EnsureSession(_ context.Context, input RuntimeTurnInput) error {
	r.ensureInput = input
	return r.ensureErr
}

func (r *fakeProjectConversationCodexRuntime) CloseSession(SessionID) bool {
	return true
}

func (r *fakeProjectConversationCodexRuntime) RespondInterrupt(
	_ context.Context,
	sessionID SessionID,
	requestID string,
	kind string,
	decision string,
	answer map[string]any,
) error {
	r.sessionID = sessionID
	r.requestID = requestID
	r.kind = kind
	r.decision = decision
	r.answer = answer
	return nil
}

func (r *fakeProjectConversationCodexRuntime) SessionAnchor(SessionID) RuntimeSessionAnchor {
	if r.anchor.ProviderThreadID == "" {
		return RuntimeSessionAnchor{}
	}
	return r.anchor
}

func closedStreamEvents() <-chan StreamEvent {
	events := make(chan StreamEvent)
	close(events)
	return events
}

func streamWithEvents(items ...StreamEvent) <-chan StreamEvent {
	events := make(chan StreamEvent, len(items))
	for _, item := range items {
		events <- item
	}
	close(events)
	return events
}

type fakeProjectConversationWorkflowSync struct {
	harnessWorkflowReader
}

func (fakeProjectConversationWorkflowSync) RefreshSkills(
	_ context.Context,
	input workflowservice.RefreshSkillsInput,
) (workflowservice.RefreshSkillsResult, error) {
	target, err := workflowservice.ResolveSkillTargetForRuntime(input.WorkspaceRoot, input.AdapterType)
	if err != nil {
		return workflowservice.RefreshSkillsResult{}, err
	}
	skillDir := filepath.Join(target.SkillsDir, "openase-platform")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		return workflowservice.RefreshSkillsResult{}, err
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# openase-platform\n"), 0o600); err != nil {
		return workflowservice.RefreshSkillsResult{}, err
	}
	wrapperPath := filepath.Join(input.WorkspaceRoot, ".openase", "bin", "openase")
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o750); err != nil {
		return workflowservice.RefreshSkillsResult{}, err
	}
	if err := os.WriteFile(wrapperPath, []byte("#!/bin/sh\n"), 0o600); err != nil {
		return workflowservice.RefreshSkillsResult{}, err
	}
	return workflowservice.RefreshSkillsResult{
		SkillsDir:      target.SkillsDir,
		InjectedSkills: []string{"openase-platform"},
	}, nil
}

type fakeProjectConversationAgentPlatform struct{}

func (fakeProjectConversationAgentPlatform) IssueToken(
	context.Context,
	agentplatform.IssueInput,
) (agentplatform.IssuedToken, error) {
	return agentplatform.IssuedToken{Token: "project-conversation-placeholder"}, nil
}

func createConversationRemoteRepo(
	t *testing.T,
	defaultBranch string,
	files map[string]string,
) (string, plumbing.Hash) {
	t.Helper()

	repoPath := t.TempDir()
	repository, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("init repository: %v", err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	for path, content := range files {
		absolutePath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		if _, err := worktree.Add(path); err != nil {
			t.Fatalf("add %s: %v", path, err)
		}
	}
	hash, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@example.com",
			When:  time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("commit repository: %v", err)
	}
	if err := repository.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName(defaultBranch), hash)); err != nil {
		t.Fatalf("set default branch: %v", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(defaultBranch)}); err != nil {
		t.Fatalf("checkout default branch: %v", err)
	}
	return repoPath, hash
}

func assertConversationFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func containsEnvironmentPrefix(environment []string, prefix string) bool {
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}
	return false
}

func createProjectConversationTestProject(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
) (organization, project projectConversationTestEntity) {
	t.Helper()

	orgItem, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	projectItem, err := client.Project.Create().
		SetOrganizationID(orgItem.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return projectConversationTestEntity{ID: orgItem.ID}, projectConversationTestEntity{ID: projectItem.ID}
}

type projectConversationTestEntity struct {
	ID uuid.UUID
}

type projectConversationWorkspaceRepoFixture struct {
	name             string
	workspaceDirname string
	files            map[string]string
}

type projectConversationWorkspaceDiffFixture struct {
	ctx           context.Context
	service       *ProjectConversationService
	conversation  chatdomain.Conversation
	workspacePath string
	repoPaths     map[string]string
}

type projectConversationStubGitHubTokenResolver struct {
	resolved githubauthdomain.ResolvedCredential
	err      error
}

func (s projectConversationStubGitHubTokenResolver) ResolveProjectCredential(context.Context, uuid.UUID) (githubauthdomain.ResolvedCredential, error) {
	if s.err != nil {
		return githubauthdomain.ResolvedCredential{}, s.err
	}
	return s.resolved, nil
}

func setupProjectConversationWorkspaceDiffFixture(
	t *testing.T,
	repos []projectConversationWorkspaceRepoFixture,
) projectConversationWorkspaceDiffFixture {
	t.Helper()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	projectRepos := make([]catalogdomain.ProjectRepo, 0, len(repos))
	for _, repo := range repos {
		remoteRepoPath, _ := createConversationRemoteRepo(t, "main", repo.files)
		projectRepo := catalogdomain.ProjectRepo{
			ID:            uuid.New(),
			ProjectID:     project.ID,
			Name:          repo.name,
			RepositoryURL: remoteRepoPath,
			DefaultBranch: "main",
		}
		if strings.TrimSpace(repo.workspaceDirname) != "" {
			projectRepo.WorkspaceDirname = repo.workspaceDirname
		}
		projectRepos = append(projectRepos, projectRepo)
	}

	workspaceRoot := t.TempDir()
	projectItem := catalogdomain.Project{
		ID:             project.ID,
		OrganizationID: org.ID,
		Name:           "OpenASE",
		Slug:           "openase",
		Description:    "Issue-driven automation",
	}
	providerItem := catalogdomain.AgentProvider{
		ID:             providerID,
		OrganizationID: org.ID,
		MachineID:      machineID,
		AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
		CliCommand:     "gemini",
	}
	catalog := fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project:      projectItem,
			projectRepos: projectRepos,
			providerByID: map[uuid.UUID]catalogdomain.AgentProvider{
				providerID: providerItem,
			},
		},
		machine: catalogdomain.Machine{
			ID:            machineID,
			Name:          catalogdomain.LocalMachineName,
			Host:          catalogdomain.LocalMachineHost,
			WorkspaceRoot: stringPointer(workspaceRoot),
		},
	}

	service := NewProjectConversationService(
		nil,
		repoStore,
		catalog,
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		nil,
	)
	workspace, err := service.ensureConversationWorkspace(ctx, catalog.machine, projectItem, providerItem, conversation.ID)
	if err != nil {
		t.Fatalf("ensureConversationWorkspace() error = %v", err)
	}

	repoPaths := make(map[string]string, len(projectRepos))
	for _, repo := range projectRepos {
		repoPaths[repo.Name] = workspaceinfra.RepoPath(workspace.String(), repo.WorkspaceDirname, repo.Name)
	}

	return projectConversationWorkspaceDiffFixture{
		ctx:           ctx,
		service:       service,
		conversation:  conversation,
		workspacePath: workspace.String(),
		repoPaths:     repoPaths,
	}
}

func writeConversationWorkspaceFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func intPointer(value int) *int {
	return &value
}
