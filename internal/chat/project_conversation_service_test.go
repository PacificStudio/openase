package chat

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
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
		nil,
		nil,
		"Prior discussion summary",
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
	}, true)
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

	if _, err := service.StartTurn(ctx, UserID("user:conversation"), secondConversation.ID, "Switch to this conversation"); err != nil {
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

	if _, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Inspect the project"); err != nil {
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
	sessionID SessionID
	requestID string
	kind      string
	decision  string
	answer    map[string]any
}

func (r *fakeProjectConversationCodexRuntime) Supports(catalogdomain.AgentProvider) bool {
	return true
}

func (r *fakeProjectConversationCodexRuntime) StartTurn(context.Context, RuntimeTurnInput) (TurnStream, error) {
	return TurnStream{}, nil
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
	return RuntimeSessionAnchor{}
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
