package chat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	secretsrepo "github.com/BetterAndBetterII/openase/internal/repo/secrets"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
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
		"## OpenASE Platform Capability Contract",
		"Current principal: `project_conversation`",
		"`OPENASE_AGENT_TOKEN`",
		"`OPENASE_TICKET_ID` only when this Project AI session is ticket-focused",
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
		"## OpenASE Platform Capability Contract",
		"Current principal: `project_conversation`",
		"`OPENASE_AGENT_TOKEN`",
		"`OPENASE_TICKET_ID` only when this Project AI session is ticket-focused",
		"### Current User Focus Area",
		"- Type: workflow",
		"- Name: Backend Engineer",
		"- harness_path: .openase/harnesses/backend.md",
		"- selected_area: harness",
		"- has_dirty_draft: true",
	) {
		t.Fatalf("expected focus context in prompt, got %q", prompt)
	}
}

func TestProjectConversationPromptIncludesTicketCapsule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	orgID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	ticketID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	targetMachineID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	service := NewProjectConversationService(
		nil,
		nil,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             projectID,
					OrganizationID: orgID,
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
				repoScopes: []catalogdomain.TicketRepoScope{
					{
						RepoID:     uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"),
						BranchName: "feat/openase-470-project-ai",
						PullRequestURL: optionalString(
							"https://github.com/PacificStudio/openase/pull/999",
						),
					},
				},
				activityEvents: []catalogdomain.ActivityEvent{
					{
						CreatedAt: time.Date(2026, 4, 2, 8, 10, 0, 0, time.UTC),
						EventType: activityevent.TypeTicketRetryPaused,
						Message:   "Paused retries after repeated failures.",
					},
					{
						CreatedAt: time.Date(2026, 4, 2, 8, 15, 0, 0, time.UTC),
						EventType: activityevent.TypeHookFailed,
						Message:   "go test ./... failed in internal/chat",
					},
				},
			},
			machine: catalogdomain.Machine{
				ID:   targetMachineID,
				Name: "worker-a",
				Host: "10.0.0.15",
			},
		},
		fakeTicketReader{
			ticket: ticketservice.Ticket{
				ID:                ticketID,
				Identifier:        "ASE-470",
				Title:             "Replace Ticket AI",
				Description:       "Route the drawer entry through Project AI.",
				StatusName:        "In Review",
				Priority:          "high",
				AttemptCount:      3,
				ConsecutiveErrors: 2,
				RetryPaused:       true,
				PauseReason:       "Repeated hook failures",
				CurrentRunID:      optionalUUID(uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000")),
				TargetMachineID:   optionalUUID(targetMachineID),
				Dependencies: []ticketservice.Dependency{
					{
						Type: ticketservice.DependencyTypeBlocks,
						Target: ticketservice.TicketReference{
							Identifier: "ASE-100",
							Title:      "Primary blocker",
						},
					},
				},
			},
		},
		harnessWorkflowReader{},
		nil,
		nil,
	)

	prompt, err := service.buildProjectConversationPrompt(
		ctx,
		chatdomain.Conversation{
			ID:        uuid.MustParse("bb0e8400-e29b-41d4-a716-446655440000"),
			ProjectID: projectID,
		},
		catalogdomain.Project{
			ID:             projectID,
			OrganizationID: orgID,
			Name:           "OpenASE",
			Slug:           "openase",
			Description:    "Issue-driven automation",
		},
		&ProjectConversationFocus{
			Kind: ProjectConversationFocusTicket,
			Ticket: &ProjectConversationTicketFocus{
				ID:           ticketID,
				Identifier:   "ASE-470",
				Title:        "Replace Ticket AI",
				Status:       "In Review",
				SelectedArea: "hooks",
				AssignedAgent: &ProjectConversationTicketAssignedAgent{
					Name:                "todo-app-coding-01",
					Provider:            "codex-cloud",
					RuntimeControlState: "active",
					RuntimePhase:        "executing",
				},
				CurrentRun: &ProjectConversationTicketRun{
					ID:                 "run-1",
					AttemptNumber:      3,
					Status:             "failed",
					CurrentStepStatus:  "failed",
					CurrentStepSummary: "openase test ./internal/chat",
					LastError:          "ticket.on_complete hook failed",
				},
			},
		},
		false,
	)
	if err != nil {
		t.Fatalf("build project conversation prompt: %v", err)
	}

	if !containsAll(
		prompt,
		"## OpenASE Platform Capability Contract",
		"Current principal: `project_conversation`",
		"`OPENASE_TICKET_ID`",
		"## Ticket Capsule",
		"Ticket: ASE-470 - Replace Ticket AI",
		"Route the drawer entry through Project AI.",
		"### Dependent Tickets",
		"ASE-100",
		"### Repository Scope",
		"https://github.com/PacificStudio/openase/pull/999",
		"### Activity Log",
		"Paused retries after repeated failures.",
		"### Hook History",
		"go test ./... failed in internal/chat",
		"### Current Ticket Subarea",
		"- selected_area: hooks",
		"### Runtime Summary",
		"- assigned_agent: todo-app-coding-01",
		"- retry_paused: true",
		"- consecutive_errors: 2",
		"- current_run_status: failed",
		"- current_run_last_error: ticket.on_complete hook failed",
		"- target_machine_name: worker-a",
	) {
		t.Fatalf("expected ticket capsule prompt, got %q", prompt)
	}
}

func TestProjectConversationRuntimeEnvironmentInjectsTicketIDForTicketFocus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := chatrepo.NewEntRepository(client)
	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	conversationID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	ticketID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	platform := &capturingProjectConversationAgentPlatform{}

	service := NewProjectConversationService(
		nil,
		repo,
		fakeProjectConversationCatalog{
			fakeCatalogReader: fakeCatalogReader{
				project: catalogdomain.Project{
					ID:             projectID,
					OrganizationID: uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"),
					Name:           "OpenASE",
					Slug:           "openase",
					Description:    "Issue-driven automation",
				},
				agents: []catalogdomain.Agent{},
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		nil,
	)
	service.ConfigurePlatformEnvironment("http://127.0.0.1:19836/api/v1/platform", platform)

	conversation := chatdomain.Conversation{
		ID:         conversationID,
		ProjectID:  projectID,
		ProviderID: providerID,
	}
	project := catalogdomain.Project{
		ID:             projectID,
		OrganizationID: uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"),
		Name:           "OpenASE",
		Slug:           "openase",
		Description:    "Issue-driven automation",
	}
	providerItem := catalogdomain.AgentProvider{
		ID:             providerID,
		OrganizationID: project.OrganizationID,
		AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
		MachineHost:    catalogdomain.LocalMachineHost,
	}

	secretManager := &fakeProjectConversationSecretManager{
		resolveBoundForRuntime: func(_ context.Context, input secretsservice.ResolveBoundRuntimeInput) ([]secretsdomain.ResolvedSecret, error) {
			if input.ProjectID != projectID {
				t.Fatalf("ResolveBoundForRuntime project_id = %s, want %s", input.ProjectID, projectID)
			}
			if input.TicketID != nil && *input.TicketID == ticketID {
				return []secretsdomain.ResolvedSecret{{BindingKey: "OPENAI_API_KEY", Value: "sk-ticket-conversation"}}, nil
			}
			if input.TicketID != nil {
				t.Fatalf("ResolveBoundForRuntime ticket_id = %s, want %s", *input.TicketID, ticketID)
			}
			return []secretsdomain.ResolvedSecret{{BindingKey: "OPENAI_API_KEY", Value: "sk-project-conversation"}}, nil
		},
	}
	service.ConfigureSecretManager(secretManager)

	environment, err := service.buildConversationRuntimeEnvironment(
		ctx,
		conversation,
		project,
		providerItem,
		&ProjectConversationFocus{
			Kind: ProjectConversationFocusTicket,
			Ticket: &ProjectConversationTicketFocus{
				ID:         ticketID,
				Identifier: "ASE-470",
				Title:      "Replace Ticket AI",
				Status:     "In Review",
			},
		},
	)
	if err != nil {
		t.Fatalf("build conversation runtime environment with ticket focus: %v", err)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_TICKET_ID="+ticketID.String()) {
		t.Fatalf("expected OPENASE_TICKET_ID in environment, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_PRINCIPAL_KIND=project_conversation") {
		t.Fatalf("expected OPENASE_PRINCIPAL_KIND in environment, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, expectedProjectConversationScopesEnvPrefix()) {
		t.Fatalf("expected OPENASE_AGENT_SCOPES in environment, got %+v", environment)
	}
	if platform.lastInput.PrincipalKind != agentplatform.PrincipalKindProjectConversation {
		t.Fatalf("expected project conversation principal token, got %+v", platform.lastInput)
	}
	if platform.lastInput.PrincipalID != conversationID || platform.lastInput.ConversationID != conversationID {
		t.Fatalf("unexpected principal ids: %+v", platform.lastInput)
	}
	if platform.lastInput.AgentID != uuid.Nil || platform.lastInput.TicketID != uuid.Nil {
		t.Fatalf("project conversation token should not carry agent/ticket ids: %+v", platform.lastInput)
	}
	if value, ok := provider.LookupEnvironmentValue(environment, "OPENAI_API_KEY"); !ok || value != "sk-ticket-conversation" {
		t.Fatalf("expected ticket-bound OPENAI_API_KEY in environment, got %+v", environment)
	}

	platform.lastInput = agentplatform.IssueInput{}
	environment, err = service.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem, nil)
	if err != nil {
		t.Fatalf("build conversation runtime environment without ticket focus: %v", err)
	}
	if containsEnvironmentPrefix(environment, "OPENASE_TICKET_ID=") {
		t.Fatalf("did not expect OPENASE_TICKET_ID without ticket focus, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_PRINCIPAL_KIND=project_conversation") {
		t.Fatalf("expected OPENASE_PRINCIPAL_KIND without ticket focus, got %+v", environment)
	}
	if platform.lastInput.PrincipalKind != agentplatform.PrincipalKindProjectConversation {
		t.Fatalf("expected project conversation principal token without ticket focus, got %+v", platform.lastInput)
	}
	if platform.lastInput.PrincipalID != conversationID || platform.lastInput.ConversationID != conversationID {
		t.Fatalf("unexpected non-ticket principal ids: %+v", platform.lastInput)
	}
	if platform.lastInput.AgentID != uuid.Nil || platform.lastInput.TicketID != uuid.Nil {
		t.Fatalf("non-ticket project conversation token should not carry agent/ticket ids: %+v", platform.lastInput)
	}
	if value, ok := provider.LookupEnvironmentValue(environment, "OPENAI_API_KEY"); !ok || value != "sk-project-conversation" {
		t.Fatalf("expected project-bound OPENAI_API_KEY without ticket focus, got %+v", environment)
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

func TestProjectConversationRespondInterruptContinuesCodexTurnWhenRuntimeMissing(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()
	machineID := uuid.New()
	ticketID := uuid.New()
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
		ProviderThreadID:          optionalString("thread-live"),
		LastTurnID:                optionalString("turn-live"),
		RollingSummary:            "Thread paused for approval",
		ProviderThreadStatus:      optionalString("waitingOnApproval"),
		ProviderThreadActiveFlags: &[]string{"waitingOnApproval"},
	})
	if err != nil {
		t.Fatalf("update conversation anchors: %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Need approval")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	if _, err := repoStore.AppendEntry(ctx, conversation.ID, &turn.ID, chatdomain.EntryKindSystem, serializeProjectConversationFocus(&ProjectConversationFocus{
		Kind: ProjectConversationFocusTicket,
		Ticket: &ProjectConversationTicketFocus{
			ID:           ticketID,
			Identifier:   "ASE-470",
			Title:        "Replace Ticket AI",
			Status:       "In Review",
			SelectedArea: "runs",
			CurrentRun: &ProjectConversationTicketRun{
				ID:                 "run-1",
				AttemptNumber:      3,
				Status:             "failed",
				CurrentStepSummary: "openase test ./internal/chat",
			},
		},
	})); err != nil {
		t.Fatalf("append focus snapshot: %v", err)
	}
	interrupt, _, err := repoStore.CreatePendingInterrupt(ctx, conversation.ID, turn.ID, "req-2", chatdomain.InterruptKindCommandExecutionApproval, map[string]any{
		"provider": "codex",
	})
	if err != nil {
		t.Fatalf("create pending interrupt: %v", err)
	}

	fakeCodex := &fakeProjectConversationCodexRuntime{
		respondStream: TurnStream{Events: streamWithEvents(
			StreamEvent{Event: "done", Payload: donePayload{SessionID: conversation.ID.String()}},
		)},
		anchor: RuntimeSessionAnchor{
			ProviderThreadID:          "thread-live",
			LastTurnID:                "provider-turn-2",
			ProviderThreadStatus:      "idle",
			ProviderThreadActiveFlags: []string{},
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
				repoScopes: []catalogdomain.TicketRepoScope{
					{
						RepoID:     uuid.New(),
						BranchName: "feat/openase-470-project-ai",
					},
				},
				activityEvents: []catalogdomain.ActivityEvent{
					{
						CreatedAt: time.Date(2026, 4, 2, 8, 10, 0, 0, time.UTC),
						EventType: activityevent.TypeHookFailed,
						Message:   "go test ./... failed in internal/chat",
					},
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
		fakeTicketReader{
			ticket: ticketservice.Ticket{
				ID:           ticketID,
				Identifier:   "ASE-470",
				Title:        "Replace Ticket AI",
				Description:  "Route the drawer entry through Project AI.",
				StatusName:   "In Review",
				Priority:     "high",
				AttemptCount: 3,
			},
		},
		harnessWorkflowReader{},
		&fakeAgentCLIProcessManager{process: &fakeAgentCLIProcess{stdin: &trackingWriteCloser{}, stdout: `{"response":"OK"}`}},
		nil,
	)
	service.newCodexRuntime = func(provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		return fakeCodex, nil
	}
	service.ConfigurePlatformEnvironment(
		"http://127.0.0.1:19836/api/v1/platform",
		&capturingProjectConversationAgentPlatform{},
	)

	resolved, err := service.RespondInterrupt(ctx, UserID("user:conversation"), conversation.ID, interrupt.ID, chatdomain.InterruptResponse{
		Decision: optionalString("approve_once"),
	})
	if err != nil {
		t.Fatalf("RespondInterrupt() error = %v", err)
	}
	if resolved.Status != chatdomain.InterruptStatusResolved {
		t.Fatalf("expected resolved interrupt, got %+v", resolved)
	}
	if !strings.Contains(fakeCodex.ensureInput.SystemPrompt, "## Ticket Capsule") {
		t.Fatalf("expected interrupt recovery to restore ticket focus capsule, got %q", fakeCodex.ensureInput.SystemPrompt)
	}
	if !containsEnvironmentPrefix(fakeCodex.respondInput.Environment, "OPENASE_TICKET_ID="+ticketID.String()) {
		t.Fatalf("expected interrupt runtime env to retain OPENASE_TICKET_ID, got %+v", fakeCodex.respondInput.Environment)
	}

	var completedTurn *ent.ChatTurn
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, getErr := client.ChatTurn.Get(ctx, turn.ID)
		if getErr == nil && item.Status == string(chatdomain.TurnStatusCompleted) && item.ProviderTurnID != nil && *item.ProviderTurnID == "provider-turn-2" {
			completedTurn = item
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if completedTurn == nil {
		t.Fatal("expected codex continuation stream to complete interrupted turn")
	}

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.LastTurnID == nil || *reloadedConversation.LastTurnID != "provider-turn-2" {
		t.Fatalf("expected conversation anchor to advance, got %+v", reloadedConversation)
	}
}

func TestProjectConversationStartTurnResumesClaudeSessionOverSSHWithoutRecoveryPrompt(t *testing.T) {
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
	const persistentSessionID = "claude-session-remote-42"
	const rollingSummary = "Remote Claude should resume via durable session, not replay recovery context."
	conversation, err = repoStore.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusActive, chatdomain.ConversationAnchors{
		ProviderThreadID:     optionalString(persistentSessionID),
		RollingSummary:       rollingSummary,
		ProviderThreadStatus: optionalString("idle"),
	})
	if err != nil {
		t.Fatalf("update conversation anchors: %v", err)
	}

	prepareSession := &projectConversationSSHPrepareSession{}
	processSession := &projectConversationSSHProcessSession{
		stdin: &trackingWriteCloser{},
		stdout: strings.Join([]string{
			fmt.Sprintf(`{"type":"system","subtype":"init","data":{"session_id":"%s"}}`, persistentSessionID),
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Remote done"}]}}`,
			fmt.Sprintf(`{"type":"result","subtype":"success","session_id":"%s","num_turns":1}`, persistentSessionID),
		}, "\n"),
	}
	sshPool := newProjectConversationTestSSHPool(t, &projectConversationSSHClient{
		sessions: []sshinfra.Session{prepareSession, processSession},
	})
	workspaceRoot := "/srv/openase/workspaces"
	sshUser := "openase"
	sshKeyPath := "keys/remote-builder.pem"
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
				ID:             machineID,
				Name:           "builder-01",
				Host:           "10.0.1.15",
				Port:           22,
				ConnectionMode: catalogdomain.MachineConnectionModeSSH,
				SSHUser:        &sshUser,
				SSHKeyPath:     &sshKeyPath,
				WorkspaceRoot:  stringPointer(workspaceRoot),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		sshPool,
	)

	turn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Continue after remote restart", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if turn.ID == uuid.Nil {
		t.Fatal("expected persisted turn id")
	}

	remoteWorkspace := filepath.Join(
		workspaceRoot,
		org.ID.String(),
		"openase",
		projectConversationWorkspaceName(conversation.ID),
	)
	if !strings.Contains(prepareSession.command, remoteWorkspace) {
		t.Fatalf("prepare command = %q, want workspace %q", prepareSession.command, remoteWorkspace)
	}
	if !strings.Contains(processSession.startedCommand, "cd '"+remoteWorkspace+"' &&") {
		t.Fatalf("remote process command = %q, want workspace cd", processSession.startedCommand)
	}
	if !strings.Contains(processSession.startedCommand, "'--resume' '"+persistentSessionID+"'") {
		t.Fatalf("remote process command = %q, want durable --resume", processSession.startedCommand)
	}
	if !strings.Contains(processSession.startedCommand, "'"+claudeCodeResumeInterruptedTurnEnv+"=1'") {
		t.Fatalf("remote process command = %q, want %s=1", processSession.startedCommand, claudeCodeResumeInterruptedTurnEnv)
	}
	if strings.Contains(processSession.startedCommand, rollingSummary) {
		t.Fatalf("expected remote claude resume to avoid recovery replay, got %q", processSession.startedCommand)
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
		t.Fatal("expected remote claude turn to complete")
	}

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.ProviderThreadID == nil || *reloadedConversation.ProviderThreadID != persistentSessionID {
		t.Fatalf("expected remote claude provider session anchor to persist, got %+v", reloadedConversation)
	}
}

func TestProjectConversationRespondInterruptRestoresCodexSessionOverSSHWhenRuntimeMissing(t *testing.T) {
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
		ProviderThreadID:     optionalString("thread-remote"),
		LastTurnID:           optionalString("turn-remote"),
		RollingSummary:       "Remote Codex thread paused for approval",
		ProviderThreadStatus: optionalString("waitingOnApproval"),
	})
	if err != nil {
		t.Fatalf("update conversation anchors: %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Need remote approval")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	interrupt, _, err := repoStore.CreatePendingInterrupt(ctx, conversation.ID, turn.ID, "req-remote", chatdomain.InterruptKindCommandExecutionApproval, map[string]any{
		"provider": "codex",
	})
	if err != nil {
		t.Fatalf("create pending interrupt: %v", err)
	}

	prepareSession := &projectConversationSSHPrepareSession{}
	sshPool := newProjectConversationTestSSHPool(t, &projectConversationSSHClient{
		sessions: []sshinfra.Session{prepareSession},
	})
	workspaceRoot := "/srv/openase/workspaces"
	sshUser := "openase"
	sshKeyPath := "keys/remote-builder.pem"
	fakeCodex := &fakeProjectConversationCodexRuntime{}
	var capturedManager provider.AgentCLIProcessManager
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
				ID:             machineID,
				Name:           "builder-01",
				Host:           "10.0.1.15",
				Port:           22,
				ConnectionMode: catalogdomain.MachineConnectionModeSSH,
				SSHUser:        &sshUser,
				SSHKeyPath:     &sshKeyPath,
				WorkspaceRoot:  stringPointer(workspaceRoot),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		sshPool,
	)
	service.newCodexRuntime = func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		capturedManager = manager
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
	if _, ok := capturedManager.(*sshinfra.ProcessManager); !ok {
		t.Fatalf("expected ssh process manager, got %T", capturedManager)
	}
	if fakeCodex.ensureInput.ResumeProviderThreadID != "thread-remote" || fakeCodex.ensureInput.ResumeProviderTurnID != "turn-remote" {
		t.Fatalf("expected remote codex interrupt recovery to resume thread, got %+v", fakeCodex.ensureInput)
	}
	if fakeCodex.requestID != "req-remote" || fakeCodex.kind != "command_execution" || fakeCodex.decision != "approve_once" {
		t.Fatalf("unexpected remote interrupt response routed to codex runtime: %+v", fakeCodex)
	}
	remoteWorkspace := filepath.Join(
		workspaceRoot,
		org.ID.String(),
		"openase",
		projectConversationWorkspaceName(conversation.ID),
	)
	if fakeCodex.ensureInput.WorkingDirectory.String() != remoteWorkspace {
		t.Fatalf("ensure working directory = %q, want %q", fakeCodex.ensureInput.WorkingDirectory, remoteWorkspace)
	}
	if !strings.Contains(prepareSession.command, remoteWorkspace) {
		t.Fatalf("prepare command = %q, want workspace %q", prepareSession.command, remoteWorkspace)
	}
}

func TestProjectConversationStartTurnUsesWebsocketListenerRuntimeManager(t *testing.T) {
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

	server := httptest.NewServer(machinetransport.NewWebsocketListenerHandler(machinetransport.ListenerHandlerOptions{}))
	defer server.Close()

	workspaceRoot := t.TempDir()
	fakeOpenASEBinDir := t.TempDir()
	writeProjectConversationFakeOpenASEBinary(t, fakeOpenASEBinDir)
	fakeCodex := &fakeProjectConversationCodexRuntime{}
	var capturedManager provider.AgentCLIProcessManager

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
				ID:                 machineID,
				Name:               "listener-01",
				Host:               "listener.internal",
				WorkspaceRoot:      stringPointer(workspaceRoot),
				ConnectionMode:     catalogdomain.MachineConnectionModeWSListener,
				AdvertisedEndpoint: stringPointer(projectConversationWebsocketURL(server.URL)),
				AgentCLIPath:       stringPointer("/bin/sh"),
				EnvVars: []string{
					"PATH=" + fakeOpenASEBinDir + string(os.PathListSeparator) + os.Getenv("PATH"),
					"OPENASE_REAL_BIN=" + filepath.Join(fakeOpenASEBinDir, "openase"),
				},
			},
		},
		fakeTicketReader{},
		fakeProjectConversationWorkflowSync{},
		nil,
		nil,
	)
	service.newCodexRuntime = func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		capturedManager = manager
		return fakeCodex, nil
	}

	turn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Inspect websocket runtime", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if turn.ID == uuid.Nil {
		t.Fatal("expected persisted turn id")
	}
	if capturedManager == nil {
		t.Fatal("expected websocket runtime to resolve a process manager")
	}

	workspacePath := filepath.Join(
		workspaceRoot,
		org.ID.String(),
		"openase",
		projectConversationWorkspaceName(conversation.ID),
	)
	skillTarget, err := workflowservice.ResolveSkillTargetForRuntime(
		workspacePath,
		string(catalogdomain.AgentProviderAdapterTypeCodexAppServer),
	)
	if err != nil {
		t.Fatalf("resolve skill target: %v", err)
	}
	assertConversationFileExists(t, filepath.Join(workspacePath, ".openase", "bin", "openase"))
	assertConversationFileExists(t, filepath.Join(skillTarget.SkillsDir, "openase-platform", "SKILL.md"))

	spec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("/bin/sh"),
		[]string{"-lc", "printf ws-conversation"},
		nil,
		[]string{"PATH=" + fakeOpenASEBinDir + string(os.PathListSeparator) + os.Getenv("PATH")},
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec() error = %v", err)
	}
	process, err := capturedManager.Start(ctx, spec)
	if err != nil {
		t.Fatalf("capturedManager.Start() error = %v", err)
	}
	output, err := io.ReadAll(process.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(process.Stdout()) error = %v", err)
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("process.Wait() error = %v", err)
	}
	if strings.TrimSpace(string(output)) != "ws-conversation" {
		t.Fatalf("process stdout = %q, want ws-conversation", string(output))
	}
}

func TestProjectConversationRespondInterruptResumesClaudeSessionOverSSHAndContinuesInterruptedTurn(t *testing.T) {
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
		ProviderThreadID:     optionalString("claude-session-remote"),
		RollingSummary:       "Claude remote session paused",
		ProviderThreadStatus: optionalString("requires_action"),
	})
	if err != nil {
		t.Fatalf("update conversation anchors: %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Need remote answer")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	interrupt, _, err := repoStore.CreatePendingInterrupt(ctx, conversation.ID, turn.ID, "req-claude", chatdomain.InterruptKindUserInput, map[string]any{
		"provider": "claude",
	})
	if err != nil {
		t.Fatalf("create pending interrupt: %v", err)
	}

	prepareSession := &projectConversationSSHPrepareSession{}
	stdin := &trackingWriteCloser{}
	processSession := &projectConversationSSHProcessSession{
		stdin: stdin,
		stdout: strings.Join([]string{
			`{"type":"system","subtype":"init","data":{"session_id":"claude-session-remote"}}`,
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Continuing on the remote Claude session"}]}}`,
			`{"type":"result","subtype":"success","session_id":"claude-session-remote","num_turns":2}`,
		}, "\n"),
	}
	sshPool := newProjectConversationTestSSHPool(t, &projectConversationSSHClient{
		sessions: []sshinfra.Session{prepareSession, processSession},
	})
	workspaceRoot := "/srv/openase/workspaces"
	sshUser := "openase"
	sshKeyPath := "keys/remote-builder.pem"
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
				ID:             machineID,
				Name:           "builder-01",
				Host:           "10.0.1.15",
				Port:           22,
				ConnectionMode: catalogdomain.MachineConnectionModeSSH,
				SSHUser:        &sshUser,
				SSHKeyPath:     &sshKeyPath,
				WorkspaceRoot:  stringPointer(workspaceRoot),
			},
		},
		fakeTicketReader{},
		harnessWorkflowReader{},
		nil,
		sshPool,
	)

	resolved, err := service.RespondInterrupt(ctx, UserID("user:conversation"), conversation.ID, interrupt.ID, chatdomain.InterruptResponse{
		Answer: map[string]any{"text": "continue"},
	})
	if err != nil {
		t.Fatalf("RespondInterrupt() error = %v", err)
	}
	if resolved.Status != chatdomain.InterruptStatusResolved {
		t.Fatalf("expected resolved interrupt, got %+v", resolved)
	}
	remoteWorkspace := filepath.Join(
		workspaceRoot,
		org.ID.String(),
		"openase",
		projectConversationWorkspaceName(conversation.ID),
	)
	if !strings.Contains(prepareSession.command, remoteWorkspace) {
		t.Fatalf("prepare command = %q, want workspace %q", prepareSession.command, remoteWorkspace)
	}
	if !strings.Contains(processSession.startedCommand, "cd '"+remoteWorkspace+"' &&") {
		t.Fatalf("remote process command = %q, want workspace cd", processSession.startedCommand)
	}
	if !strings.Contains(processSession.startedCommand, "'--resume' 'claude-session-remote'") {
		t.Fatalf("remote process command = %q, want durable --resume", processSession.startedCommand)
	}
	if !strings.Contains(processSession.startedCommand, "'"+claudeCodeResumeInterruptedTurnEnv+"=1'") {
		t.Fatalf("remote process command = %q, want %s=1", processSession.startedCommand, claudeCodeResumeInterruptedTurnEnv)
	}
	if !strings.Contains(stdin.String(), `\"text\": \"continue\"`) || !strings.Contains(stdin.String(), `\"request_id\": \"req-claude\"`) {
		t.Fatalf("stdin payload = %q, want encoded claude interrupt response", stdin.String())
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
		t.Fatal("expected interrupted Claude turn to complete after remote response")
	}

	entries, err := repoStore.ListEntries(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if !containsAll(strings.Join(renderRecoveryLines(entries, len(entries)), "\n"), "assistant: Continuing on the remote Claude session") {
		t.Fatalf("expected assistant continuation entry after response, got %+v", entries)
	}

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.Status != chatdomain.ConversationStatusActive {
		t.Fatalf("expected active conversation after response, got %+v", reloadedConversation)
	}
	if reloadedConversation.ProviderThreadID == nil || *reloadedConversation.ProviderThreadID != "claude-session-remote" {
		t.Fatalf("expected claude session anchor to persist, got %+v", reloadedConversation)
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

	service.consumeTurn(ctx, conversation, turn, live, chatdomain.ProjectConversationRun{}, TurnStream{Events: events})

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

func TestProjectConversationConsumeTurnKeepsStableTitleWhenRollingSummaryUpdates(t *testing.T) {
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
	turn, _, err := repoStore.CreateTurnWithUserEntry(
		ctx,
		conversation.ID,
		"Keep this first sentence as the permanent title. Later summaries may change.",
	)
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	if _, err := repoStore.AppendEntry(ctx, conversation.ID, &turn.ID, chatdomain.EntryKindAssistantTextDelta, map[string]any{
		"role":    "assistant",
		"content": "Done. I updated the summary context.",
	}); err != nil {
		t.Fatalf("append assistant entry: %v", err)
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

	service.consumeTurn(
		ctx,
		conversation,
		turn,
		&liveProjectConversation{
			codex: &fakeProjectConversationCodexRuntime{
				anchor: RuntimeSessionAnchor{
					ProviderThreadID:          "thread-1",
					LastTurnID:                "provider-turn-1",
					ProviderThreadStatus:      "idle",
					ProviderThreadActiveFlags: []string{},
				},
			},
		},
		chatdomain.ProjectConversationRun{},
		TurnStream{Events: events},
	)

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if got, want := reloadedConversation.Title.String(), "Keep this first sentence as the permanent title."; got != want {
		t.Fatalf("conversation title = %q, want %q", got, want)
	}
	if !containsAll(
		reloadedConversation.RollingSummary,
		"user: Keep this first sentence as the permanent title. Later summaries may change.",
		"assistant: Done. I updated the summary context.",
	) {
		t.Fatalf("rolling summary = %q", reloadedConversation.RollingSummary)
	}
}

func TestProjectConversationConsumeTurnAutoReleasesCompletedRuntime(t *testing.T) {
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
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "Finish and release the runtime")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	principal, err := repoStore.EnsurePrincipal(ctx, chatdomain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      project.ID,
		ProviderID:     providerID,
		Name:           projectConversationPrincipalName(conversation.ID),
	})
	if err != nil {
		t.Fatalf("ensure principal: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	workspacePath := t.TempDir()
	run, err := repoStore.CreateRun(ctx, chatdomain.CreateRunInput{
		RunID:                uuid.New(),
		PrincipalID:          principal.ID,
		ConversationID:       conversation.ID,
		ProjectID:            project.ID,
		ProviderID:           providerID,
		TurnID:               &turn.ID,
		Status:               chatdomain.RunStatusExecuting,
		SessionID:            optionalString(conversation.ID.String()),
		WorkspacePath:        optionalString(workspacePath),
		RuntimeStartedAt:     &now,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &now,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	principal, err = repoStore.UpdatePrincipalRuntime(ctx, chatdomain.UpdatePrincipalRuntimeInput{
		PrincipalID:          principal.ID,
		RuntimeState:         chatdomain.RuntimeStateExecuting,
		CurrentSessionID:     optionalString(conversation.ID.String()),
		CurrentWorkspacePath: optionalString(workspacePath),
		CurrentRunID:         &run.ID,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &now,
	})
	if err != nil {
		t.Fatalf("seed principal runtime state: %v", err)
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
	watched, cleanup, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation() error = %v", err)
	}
	initial := requireProjectConversationStreamEvent(t, watched)
	if initial.Event != "session" {
		t.Fatalf("initial event = %q, want session", initial.Event)
	}

	closer := &fakeRuntime{closeResult: true}
	live := &liveProjectConversation{
		principal: principal,
		runtime:   closer,
		codex: &fakeProjectConversationCodexRuntime{
			anchor: RuntimeSessionAnchor{
				ProviderThreadID:          "thread-release",
				LastTurnID:                "provider-turn-release",
				ProviderThreadStatus:      "idle",
				ProviderThreadActiveFlags: []string{},
			},
		},
		workspace: provider.AbsolutePath(t.TempDir()),
	}
	service.runtimeManager.live[conversation.ID] = live

	streamEvents := streamWithEvents(
		StreamEvent{
			Event:   "message",
			Payload: textPayload{Type: chatMessageTypeText, Content: "Completed output"},
		},
		StreamEvent{
			Event:   "done",
			Payload: donePayload{SessionID: conversation.ID.String(), CostUSD: floatPointer(0.21)},
		},
	)

	service.consumeTurn(ctx, conversation, turn, live, run, TurnStream{Events: streamEvents})

	if len(closer.closeCalls) != 1 || closer.closeCalls[0] != SessionID(conversation.ID.String()) {
		t.Fatalf("runtime close calls = %+v, want %s", closer.closeCalls, conversation.ID)
	}
	if _, exists := service.runtimeManager.Get(conversation.ID); exists {
		t.Fatal("expected live runtime registry entry to be removed after completion")
	}

	reloadedPrincipal, err := repoStore.GetPrincipal(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload principal: %v", err)
	}
	if reloadedPrincipal.RuntimeState != chatdomain.RuntimeStateInactive {
		t.Fatalf("principal runtime state = %q, want inactive", reloadedPrincipal.RuntimeState)
	}
	if reloadedPrincipal.CurrentSessionID != nil || reloadedPrincipal.CurrentRunID != nil {
		t.Fatalf("expected principal runtime ownership to be cleared, got %+v", reloadedPrincipal)
	}

	reloadedRun, err := client.ProjectConversationRun.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if string(reloadedRun.Status) != string(chatdomain.RunStatusCompleted) || reloadedRun.TerminalAt == nil {
		t.Fatalf("run = %+v, want completed terminal run", reloadedRun)
	}

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.Status != chatdomain.ConversationStatusActive {
		t.Fatalf("conversation status = %q, want active", reloadedConversation.Status)
	}
	if reloadedConversation.ProviderThreadID == nil || *reloadedConversation.ProviderThreadID != "thread-release" {
		t.Fatalf("expected provider thread anchor to persist, got %+v", reloadedConversation.ProviderThreadID)
	}
	if reloadedConversation.LastTurnID == nil || *reloadedConversation.LastTurnID != "provider-turn-release" {
		t.Fatalf("expected last turn anchor to persist, got %+v", reloadedConversation.LastTurnID)
	}
	if reloadedConversation.ProviderThreadStatus == nil || *reloadedConversation.ProviderThreadStatus != "notLoaded" {
		t.Fatalf("expected provider thread status notLoaded after release, got %+v", reloadedConversation.ProviderThreadStatus)
	}
	if len(reloadedConversation.ProviderThreadActiveFlags) != 0 {
		t.Fatalf("expected provider active flags to clear after release, got %+v", reloadedConversation.ProviderThreadActiveFlags)
	}
	if !containsAll(reloadedConversation.RollingSummary, "user: Finish and release the runtime", "assistant: Completed output") {
		t.Fatalf("rolling summary = %q", reloadedConversation.RollingSummary)
	}

	cleanup()
	collected := collectStreamEvents(watched)
	if len(collected) < 3 {
		t.Fatalf("expected initial session, turn_done, and inactive session events, got %+v", collected)
	}
	if collected[len(collected)-2].Event != "turn_done" {
		t.Fatalf("penultimate event = %q, want turn_done", collected[len(collected)-2].Event)
	}
	lastPayload, ok := collected[len(collected)-1].Payload.(map[string]any)
	if collected[len(collected)-1].Event != "session" || !ok {
		t.Fatalf("last event = %+v, want inactive session payload", collected[len(collected)-1])
	}
	if lastPayload["runtime_state"] != string(chatdomain.RuntimeStateInactive) {
		t.Fatalf("inactive session payload = %#v", lastPayload)
	}
	if lastPayload["provider_thread_id"] != "thread-release" || lastPayload["last_turn_id"] != "provider-turn-release" {
		t.Fatalf("inactive session should retain anchors, got %#v", lastPayload)
	}
	if lastPayload["provider_thread_status"] != "notLoaded" {
		t.Fatalf("inactive session should report notLoaded provider status, got %#v", lastPayload)
	}
}

func TestProjectConversationStartTurnResumesCodexAfterAutoRelease(t *testing.T) {
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
	firstTurn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "First completed turn")
	if err != nil {
		t.Fatalf("create first turn: %v", err)
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

	firstLive := &liveProjectConversation{
		runtime: &fakeRuntime{closeResult: true},
		codex: &fakeProjectConversationCodexRuntime{
			anchor: RuntimeSessionAnchor{
				ProviderThreadID:          "thread-shared",
				LastTurnID:                "provider-turn-1",
				ProviderThreadStatus:      "idle",
				ProviderThreadActiveFlags: []string{},
			},
		},
	}
	service.runtimeManager.live[conversation.ID] = firstLive
	service.consumeTurn(
		ctx,
		conversation,
		firstTurn,
		firstLive,
		chatdomain.ProjectConversationRun{},
		TurnStream{Events: streamWithEvents(
			StreamEvent{
				Event:   "message",
				Payload: textPayload{Type: chatMessageTypeText, Content: "First completed answer"},
			},
			StreamEvent{
				Event:   "done",
				Payload: donePayload{SessionID: conversation.ID.String()},
			},
		)},
	)

	nextCodex := &fakeProjectConversationCodexRuntime{
		startStream: TurnStream{Events: closedStreamEvents()},
	}
	service.newCodexRuntime = func(provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		return nextCodex, nil
	}

	secondTurn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Continue after auto release", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if secondTurn.ID == uuid.Nil {
		t.Fatal("expected second turn to persist")
	}
	if nextCodex.ensureInput.ResumeProviderThreadID != "thread-shared" || nextCodex.ensureInput.ResumeProviderTurnID != "provider-turn-1" {
		t.Fatalf("expected resume from persisted Codex anchors, got %+v", nextCodex.ensureInput)
	}
	if strings.Contains(nextCodex.startInput.SystemPrompt, "First completed answer") {
		t.Fatalf("expected resumed Codex turn to avoid replaying recovery prompt, got %q", nextCodex.startInput.SystemPrompt)
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

	service.consumeTurn(ctx, conversation, turn, live, chatdomain.ProjectConversationRun{}, TurnStream{Events: events})

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

	service.runtimeManager.mu.Lock()
	delete(service.runtimeManager.live, conversation.ID)
	service.runtimeManager.mu.Unlock()

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
	events, cleanup, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation() error = %v", err)
	}
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
	events, cleanup, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation() error = %v", err)
	}
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

func TestProjectConversationWatchConversationIsolatesEventsPerConversation(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	_, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)

	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	firstEvents, cleanupFirst, err := service.WatchConversation(ctx, UserID("user:conversation"), firstConversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation(first) error = %v", err)
	}
	defer cleanupFirst()
	secondEvents, cleanupSecond, err := service.WatchConversation(ctx, UserID("user:conversation"), secondConversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation(second) error = %v", err)
	}
	defer cleanupSecond()

	if event := requireProjectConversationStreamEvent(t, firstEvents); event.Event != "session" {
		t.Fatalf("first watcher initial event = %q, want session", event.Event)
	}
	if event := requireProjectConversationStreamEvent(t, secondEvents); event.Event != "session" {
		t.Fatalf("second watcher initial event = %q, want session", event.Event)
	}

	_, err = service.AppendSystemEntry(
		ctx,
		UserID("user:conversation"),
		firstConversation.ID,
		nil,
		testTaskNotificationPayload("conversation-1"),
	)
	if err != nil {
		t.Fatalf("append action result: %v", err)
	}

	delivered := requireProjectConversationStreamEvent(t, firstEvents)
	if delivered.Event != "message" {
		t.Fatalf("first watcher event = %q, want message", delivered.Event)
	}
	payload, ok := delivered.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected message payload map, got %#v", delivered.Payload)
	}
	if payload["type"] != "task_notification" {
		t.Fatalf("first watcher payload type = %#v, want task_notification", payload["type"])
	}
	nested, ok := payload["raw"].(map[string]any)
	if !ok || nested["marker"] != "conversation-1" {
		t.Fatalf("first watcher payload = %#v, want marker conversation-1", payload["raw"])
	}

	requireNoProjectConversationStreamEvent(t, secondEvents)
}

func TestProjectConversationWatchConversationFansOutToAllWatchers(t *testing.T) {
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

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	firstWatcher, cleanupFirst, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation(first) error = %v", err)
	}
	secondWatcher, cleanupSecond, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation(second) error = %v", err)
	}
	defer cleanupSecond()
	defer func() {
		if cleanupFirst != nil {
			cleanupFirst()
		}
	}()

	if event := requireProjectConversationStreamEvent(t, firstWatcher); event.Event != "session" {
		t.Fatalf("first watcher initial event = %q, want session", event.Event)
	}
	if event := requireProjectConversationStreamEvent(t, secondWatcher); event.Event != "session" {
		t.Fatalf("second watcher initial event = %q, want session", event.Event)
	}

	_, err = service.AppendSystemEntry(
		ctx,
		UserID("user:conversation"),
		conversation.ID,
		nil,
		testTaskNotificationPayload("fanout-1"),
	)
	if err != nil {
		t.Fatalf("append first action result: %v", err)
	}

	for name, watcher := range map[string]<-chan StreamEvent{
		"first":  firstWatcher,
		"second": secondWatcher,
	} {
		delivered := requireProjectConversationStreamEvent(t, watcher)
		if delivered.Event != "message" {
			t.Fatalf("%s watcher event = %q, want message", name, delivered.Event)
		}
		payload, ok := delivered.Payload.(map[string]any)
		if !ok {
			t.Fatalf("%s watcher payload = %#v, want map", name, delivered.Payload)
		}
		nested, ok := payload["raw"].(map[string]any)
		if !ok || nested["marker"] != "fanout-1" {
			t.Fatalf("%s watcher payload = %#v, want marker fanout-1", name, payload["raw"])
		}
	}

	cleanupFirst()
	cleanupFirst = nil
	_, err = service.AppendSystemEntry(
		ctx,
		UserID("user:conversation"),
		conversation.ID,
		nil,
		testTaskNotificationPayload("fanout-2"),
	)
	if err != nil {
		t.Fatalf("append second action result: %v", err)
	}

	delivered := requireProjectConversationStreamEvent(t, secondWatcher)
	payload, ok := delivered.Payload.(map[string]any)
	if delivered.Event != "message" || !ok {
		t.Fatalf("remaining watcher event = %+v, want message payload", delivered)
	}
	nested, ok := payload["raw"].(map[string]any)
	if !ok || nested["marker"] != "fanout-2" {
		t.Fatalf("remaining watcher payload = %#v, want marker fanout-2", payload["raw"])
	}
}

func TestProjectConversationWatchConversationBlockedWatcherDoesNotBlockOthers(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	_, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)

	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	blockedWatcher, cleanupBlocked, err := service.WatchConversation(ctx, UserID("user:conversation"), firstConversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation(blocked) error = %v", err)
	}
	defer cleanupBlocked()
	activeWatcher, cleanupActive, err := service.WatchConversation(ctx, UserID("user:conversation"), firstConversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation(active) error = %v", err)
	}
	defer cleanupActive()
	otherConversationWatcher, cleanupOther, err := service.WatchConversation(ctx, UserID("user:conversation"), secondConversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation(other) error = %v", err)
	}
	defer cleanupOther()

	requireProjectConversationStreamEvent(t, blockedWatcher)
	requireProjectConversationStreamEvent(t, activeWatcher)
	requireProjectConversationStreamEvent(t, otherConversationWatcher)

	for index := range projectConversationStreamBufferSize {
		service.broadcast(firstConversation.ID, StreamEvent{
			Event:   "message",
			Payload: testTaskNotificationPayload(fmt.Sprintf("buffer-%d", index)),
		})

		delivered := requireProjectConversationStreamEvent(t, activeWatcher)
		payload, ok := delivered.Payload.(map[string]any)
		if delivered.Event != "message" || !ok {
			t.Fatalf("active watcher event = %+v, want message payload", delivered)
		}
		nested, ok := payload["raw"].(map[string]any)
		if !ok || nested["marker"] != fmt.Sprintf("buffer-%d", index) {
			t.Fatalf("active watcher payload = %#v, want marker buffer-%d", payload["raw"], index)
		}
	}

	service.broadcast(firstConversation.ID, StreamEvent{
		Event:   "message",
		Payload: testTaskNotificationPayload("after-blocked"),
	})

	delivered := requireProjectConversationStreamEvent(t, activeWatcher)
	payload, ok := delivered.Payload.(map[string]any)
	if delivered.Event != "message" || !ok {
		t.Fatalf("active watcher event after blocked = %+v, want message payload", delivered)
	}
	nested, ok := payload["raw"].(map[string]any)
	if !ok || nested["marker"] != "after-blocked" {
		t.Fatalf("active watcher payload after blocked = %#v, want marker after-blocked", payload["raw"])
	}

	service.broadcast(secondConversation.ID, StreamEvent{
		Event:   "message",
		Payload: testTaskNotificationPayload("other-conversation"),
	})

	otherDelivered := requireProjectConversationStreamEvent(t, otherConversationWatcher)
	otherPayload, ok := otherDelivered.Payload.(map[string]any)
	if otherDelivered.Event != "message" || !ok {
		t.Fatalf("other conversation watcher event = %+v, want message payload", otherDelivered)
	}
	otherNested, ok := otherPayload["raw"].(map[string]any)
	if !ok || otherNested["marker"] != "other-conversation" {
		t.Fatalf("other conversation watcher payload = %#v, want marker other-conversation", otherPayload["raw"])
	}

	overflowDelivered := false
	for range projectConversationStreamBufferSize {
		event := requireProjectConversationStreamEvent(t, blockedWatcher)
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			continue
		}
		nested, ok := payload["raw"].(map[string]any)
		if ok && nested["marker"] == "after-blocked" {
			overflowDelivered = true
		}
	}
	if overflowDelivered {
		t.Fatal("blocked watcher unexpectedly received overflow event")
	}
	requireNoProjectConversationStreamEvent(t, blockedWatcher)
}

func TestProjectConversationWatchProjectConversationsIsolatesByProjectAndUser(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, firstProject := createProjectConversationTestProject(ctx, t, client)
	secondProject, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Secondary").
		SetSlug("openase-secondary-" + uuid.NewString()).
		SetDescription("Secondary project").
		Save(ctx)
	if err != nil {
		t.Fatalf("create second project: %v", err)
	}
	repoStore := chatrepo.NewEntRepository(client)

	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  firstProject.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  firstProject.ID,
		UserID:     "user:other",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}
	thirdConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  secondProject.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create third conversation: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	events, cleanup, err := service.WatchProjectConversations(ctx, UserID("user:conversation"), firstProject.ID)
	if err != nil {
		t.Fatalf("watch project conversations: %v", err)
	}
	defer cleanup()

	initial := requireProjectConversationMuxEvent(t, events)
	if initial.Event != "session" {
		t.Fatalf("initial mux event = %q, want session", initial.Event)
	}
	if initial.ConversationID != firstConversation.ID {
		t.Fatalf("initial mux conversation = %s, want %s", initial.ConversationID, firstConversation.ID)
	}
	requireNoProjectConversationMuxEvent(t, events)

	if _, err := service.AppendSystemEntry(
		ctx,
		UserID("user:conversation"),
		firstConversation.ID,
		nil,
		testTaskNotificationPayload("first-project"),
	); err != nil {
		t.Fatalf("append first action result: %v", err)
	}
	delivered := requireProjectConversationMuxEvent(t, events)
	if delivered.Event != "message" || delivered.ConversationID != firstConversation.ID {
		t.Fatalf("delivered mux event = %+v, want first conversation message", delivered)
	}

	if _, err := service.AppendSystemEntry(
		ctx,
		UserID("user:other"),
		secondConversation.ID,
		nil,
		testTaskNotificationPayload("wrong-user"),
	); err != nil {
		t.Fatalf("append second action result: %v", err)
	}
	if _, err := service.AppendSystemEntry(
		ctx,
		UserID("user:conversation"),
		thirdConversation.ID,
		nil,
		testTaskNotificationPayload("wrong-project"),
	); err != nil {
		t.Fatalf("append third action result: %v", err)
	}
	requireNoProjectConversationMuxEvent(t, events)
}

func TestProjectConversationWatchProjectConversationsFansOutToAllWatchers(t *testing.T) {
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

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	firstWatcher, cleanupFirst, err := service.WatchProjectConversations(
		ctx,
		UserID("user:conversation"),
		project.ID,
	)
	if err != nil {
		t.Fatalf("watch first mux stream: %v", err)
	}
	defer cleanupFirst()
	secondWatcher, cleanupSecond, err := service.WatchProjectConversations(
		ctx,
		UserID("user:conversation"),
		project.ID,
	)
	if err != nil {
		t.Fatalf("watch second mux stream: %v", err)
	}
	defer cleanupSecond()

	requireProjectConversationMuxEvent(t, firstWatcher)
	requireProjectConversationMuxEvent(t, secondWatcher)

	if _, err := service.AppendSystemEntry(
		ctx,
		UserID("user:conversation"),
		conversation.ID,
		nil,
		testTaskNotificationPayload("mux-fanout"),
	); err != nil {
		t.Fatalf("append action result: %v", err)
	}

	for name, watcher := range map[string]<-chan ProjectConversationMuxEvent{
		"first":  firstWatcher,
		"second": secondWatcher,
	} {
		delivered := requireProjectConversationMuxEvent(t, watcher)
		if delivered.Event != "message" || delivered.ConversationID != conversation.ID {
			t.Fatalf("%s mux watcher event = %+v, want conversation message", name, delivered)
		}
	}
}

func TestProjectConversationWatchProjectConversationsUsesExecutingPrincipalRuntimeState(t *testing.T) {
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
	principal, err := repoStore.EnsurePrincipal(ctx, chatdomain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      project.ID,
		ProviderID:     conversation.ProviderID,
		Name:           projectConversationPrincipalName(conversation.ID),
	})
	if err != nil {
		t.Fatalf("ensure principal: %v", err)
	}
	now := time.Now().UTC()
	if _, err := repoStore.UpdatePrincipalRuntime(ctx, chatdomain.UpdatePrincipalRuntimeInput{
		PrincipalID:          principal.ID,
		RuntimeState:         chatdomain.RuntimeStateExecuting,
		CurrentSessionID:     optionalString(conversation.ID.String()),
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &now,
	}); err != nil {
		t.Fatalf("update principal runtime: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	events, cleanup, err := service.WatchProjectConversations(
		ctx,
		UserID("user:conversation"),
		project.ID,
	)
	if err != nil {
		t.Fatalf("watch project conversations: %v", err)
	}
	defer cleanup()

	initial := requireProjectConversationMuxEvent(t, events)
	if initial.Event != "session" {
		t.Fatalf("initial mux event = %q, want session", initial.Event)
	}
	payload, ok := initial.Payload.(map[string]any)
	if !ok {
		t.Fatalf("initial mux payload = %#v, want map", initial.Payload)
	}
	if payload["runtime_state"] != string(chatdomain.RuntimeStateExecuting) {
		t.Fatalf(
			"initial mux runtime_state = %#v, want %q",
			payload["runtime_state"],
			chatdomain.RuntimeStateExecuting,
		)
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
	watched, cleanup, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation() error = %v", err)
	}

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

	service.consumeTurn(ctx, conversation, turn, live, chatdomain.ProjectConversationRun{}, TurnStream{Events: streamEvents})
	cleanup()

	reloadedConversation, err := repoStore.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if reloadedConversation.ProviderThreadStatus == nil || *reloadedConversation.ProviderThreadStatus != "notLoaded" {
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
	foundExecutingSession := false
	for _, item := range collected {
		if item.Event != "session" {
			continue
		}
		payload, ok := item.Payload.(map[string]any)
		if !ok {
			continue
		}
		if payload["provider_thread_status"] == "waitingOnUserInput" &&
			payload["runtime_state"] == string(chatdomain.RuntimeStateExecuting) {
			foundExecutingSession = true
			break
		}
	}
	if !foundExecutingSession {
		t.Fatalf("expected executing session payload in collected events, got %+v", collected)
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
	events, cleanup, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation() error = %v", err)
	}

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

	service.consumeTurn(ctx, conversation, turn, &liveProjectConversation{}, chatdomain.ProjectConversationRun{}, TurnStream{Events: streamEvents})
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
	if collected[len(collected)-2].Event != "turn_done" {
		t.Fatalf("penultimate stream event = %q, want turn_done", collected[len(collected)-2].Event)
	}
	lastPayload, ok := collected[len(collected)-1].Payload.(map[string]any)
	if collected[len(collected)-1].Event != "session" || !ok {
		t.Fatalf("last stream event = %+v, want inactive session payload", collected[len(collected)-1])
	}
	if lastPayload["runtime_state"] != string(chatdomain.RuntimeStateInactive) {
		t.Fatalf("expected inactive session after completion, got %#v", lastPayload)
	}
}

func TestProjectConversationConsumeTurnRecordsUsageAndProviderRateLimit(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	org, project := createProjectConversationTestProject(ctx, t, client)
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("127.0.0.1").
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("OpenAI Codex").
		SetAdapterType("codex-app-server").
		SetCliCommand("codex").
		SetModelName("gpt-5-codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	repo := chatrepo.NewEntRepository(client)
	conversation, err := repo.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerItem.ID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	turn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "Inspect spend and provider state")
	if err != nil {
		t.Fatalf("create turn: %v", err)
	}
	principal, err := repo.EnsurePrincipal(ctx, chatdomain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      project.ID,
		ProviderID:     providerItem.ID,
		Name:           projectConversationPrincipalName(conversation.ID),
	})
	if err != nil {
		t.Fatalf("ensure principal: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	run, err := repo.CreateRun(ctx, chatdomain.CreateRunInput{
		RunID:                uuid.New(),
		PrincipalID:          principal.ID,
		ConversationID:       conversation.ID,
		ProjectID:            project.ID,
		ProviderID:           providerItem.ID,
		TurnID:               &turn.ID,
		Status:               chatdomain.RunStatusExecuting,
		SessionID:            optionalString(conversation.ID.String()),
		WorkspacePath:        optionalString(t.TempDir()),
		RuntimeStartedAt:     &now,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &now,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	service := NewProjectConversationService(nil, repo, nil, nil, nil, nil, nil)
	streamEvents := make(chan StreamEvent, 3)
	usedPercent := 42.0
	resetAt := now.Add(5 * time.Minute)
	streamEvents <- StreamEvent{
		Event: "token_usage_updated",
		Payload: runtimeTokenUsagePayload{
			TotalInputTokens:       120,
			TotalOutputTokens:      35,
			TotalCachedInputTokens: 12,
			TotalReasoningTokens:   7,
			TotalTokens:            155,
			CostUSD:                floatPointer(0.19),
		},
	}
	streamEvents <- StreamEvent{
		Event: "rate_limit_updated",
		Payload: runtimeRateLimitPayload{
			ObservedAt: now,
			RateLimit: &provider.CLIRateLimit{
				Provider: provider.CLIRateLimitProviderCodex,
				Codex: &provider.CodexRateLimit{
					LimitID:   "codex-limit",
					LimitName: "Codex Pro",
					PlanType:  "pro",
					Primary: &provider.CodexRateLimitWindow{
						UsedPercent:   &usedPercent,
						WindowMinutes: 5,
						ResetsAt:      &resetAt,
					},
				},
			},
		},
	}
	streamEvents <- StreamEvent{
		Event:   "done",
		Payload: donePayload{SessionID: conversation.ID.String(), CostUSD: floatPointer(0.19)},
	}
	close(streamEvents)

	live := &liveProjectConversation{
		principal: principal,
		provider: catalogdomain.AgentProvider{
			ID:             providerItem.ID,
			OrganizationID: org.ID,
			MachineID:      machine.ID,
			Name:           providerItem.Name,
			AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
			ModelName:      providerItem.ModelName,
		},
		workspace: provider.AbsolutePath(t.TempDir()),
	}
	service.consumeTurn(ctx, conversation, turn, live, run, TurnStream{Events: streamEvents})

	reloadedRun, err := client.ProjectConversationRun.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if reloadedRun.InputTokens != 120 || reloadedRun.OutputTokens != 35 || reloadedRun.TotalTokens != 155 {
		t.Fatalf("run usage = %+v", reloadedRun)
	}
	if reloadedRun.CostAmount != 0.19 {
		t.Fatalf("run cost_amount = %.2f, want 0.19", reloadedRun.CostAmount)
	}

	updatedProvider, err := client.AgentProvider.Get(ctx, providerItem.ID)
	if err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if updatedProvider.CliRateLimitUpdatedAt == nil || !updatedProvider.CliRateLimitUpdatedAt.UTC().Equal(now) {
		t.Fatalf("provider cli_rate_limit_updated_at = %+v, want %s", updatedProvider.CliRateLimitUpdatedAt, now.Format(time.RFC3339))
	}
	if updatedProvider.CliRateLimit["provider"] != string(provider.CLIRateLimitProviderCodex) {
		t.Fatalf("provider cli_rate_limit = %+v", updatedProvider.CliRateLimit)
	}

	activities, err := client.ActivityEvent.Query().
		Where(entactivityevent.ProjectIDEQ(project.ID)).
		Order(entactivityevent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		t.Fatalf("list activities: %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("activity count = %d, want 2", len(activities))
	}
	var costEvent *ent.ActivityEvent
	var rateLimitEvent *ent.ActivityEvent
	for _, item := range activities {
		switch item.EventType {
		case chatdomain.CostRecordedEventType:
			costEvent = item
		case activityevent.TypeProviderRateLimitUpdated.String():
			rateLimitEvent = item
		}
	}
	if costEvent == nil {
		t.Fatalf("missing project conversation cost activity in %+v", activities)
	}
	if costUSD, ok := costEvent.Metadata["cost_usd"].(float64); !ok || costUSD != 0.19 {
		t.Fatalf("project conversation cost metadata = %+v", costEvent.Metadata)
	}
	if rateLimitEvent == nil {
		t.Fatalf("missing provider rate limit activity in %+v", activities)
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
	service.runtimeManager.live[conversation.ID] = &liveProjectConversation{codex: codexRuntime}
	events, cleanup, err := service.WatchConversation(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("WatchConversation() error = %v", err)
	}

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
	service.runtimeManager.live[firstConversation.ID] = &liveProjectConversation{runtime: previousRuntime}

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

	if service.runtimeManager.live[firstConversation.ID] == nil {
		t.Fatal("expected first conversation live runtime to remain registered")
	}
	if service.runtimeManager.live[secondConversation.ID] == nil {
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
	service.runtimeManager.live[conversation.ID] = &liveProjectConversation{runtime: &fakeRuntime{}}

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

func TestProjectConversationStartTurnRecoversStaleRunningTurnAfterRestart(t *testing.T) {
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
	staleTurn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "The runtime was interrupted by restart")
	if err != nil {
		t.Fatalf("seed stale turn: %v", err)
	}
	principal, err := repo.EnsurePrincipal(ctx, chatdomain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      project.ID,
		ProviderID:     providerID,
		Name:           projectConversationPrincipalName(conversation.ID),
	})
	if err != nil {
		t.Fatalf("ensure principal: %v", err)
	}
	now := time.Now().UTC()
	run, err := repo.CreateRun(ctx, chatdomain.CreateRunInput{
		RunID:                uuid.New(),
		PrincipalID:          principal.ID,
		ConversationID:       conversation.ID,
		ProjectID:            project.ID,
		ProviderID:           providerID,
		TurnID:               &staleTurn.ID,
		Status:               chatdomain.RunStatusExecuting,
		SessionID:            optionalString(conversation.ID.String()),
		WorkspacePath:        optionalString(t.TempDir()),
		RuntimeStartedAt:     &now,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &now,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if _, err := repo.UpdatePrincipalRuntime(ctx, chatdomain.UpdatePrincipalRuntimeInput{
		PrincipalID:          principal.ID,
		RuntimeState:         chatdomain.RuntimeStateExecuting,
		CurrentSessionID:     optionalString(conversation.ID.String()),
		CurrentRunID:         &run.ID,
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &now,
	}); err != nil {
		t.Fatalf("update principal runtime: %v", err)
	}

	fakeCodex := &fakeProjectConversationCodexRuntime{
		startStream: TurnStream{Events: streamWithEvents(
			StreamEvent{Event: "done", Payload: donePayload{SessionID: conversation.ID.String()}},
		)},
		anchor: RuntimeSessionAnchor{
			ProviderThreadID:          "thread-recovered",
			LastTurnID:                "provider-turn-recovered",
			ProviderThreadStatus:      "idle",
			ProviderThreadActiveFlags: []string{},
		},
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
		&fakeAgentCLIProcessManager{
			process: &fakeAgentCLIProcess{
				stdin:  &trackingWriteCloser{},
				stdout: `{"response":"OK"}`,
			},
		},
		nil,
	)
	service.newCodexRuntime = func(provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		return fakeCodex, nil
	}

	newTurn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Continue after restart", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if newTurn.ID == staleTurn.ID {
		t.Fatalf("expected a new turn, got stale turn id %s", newTurn.ID)
	}

	reloadedStaleTurn, err := client.ChatTurn.Get(ctx, staleTurn.ID)
	if err != nil {
		t.Fatalf("reload stale turn: %v", err)
	}
	if reloadedStaleTurn.Status != string(chatdomain.TurnStatusTerminated) {
		t.Fatalf("stale turn status = %q, want %q", reloadedStaleTurn.Status, chatdomain.TurnStatusTerminated)
	}
	reloadedRun, err := client.ProjectConversationRun.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload stale run: %v", err)
	}
	if string(reloadedRun.Status) != string(chatdomain.RunStatusTerminated) {
		t.Fatalf("stale run status = %q, want %q", reloadedRun.Status, chatdomain.RunStatusTerminated)
	}
	var completedNewTurn *ent.ChatTurn
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, getErr := client.ChatTurn.Get(ctx, newTurn.ID)
		if getErr == nil && item.Status == string(chatdomain.TurnStatusCompleted) {
			completedNewTurn = item
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if completedNewTurn == nil {
		t.Fatal("expected recovered new turn to complete")
	}
}

func TestProjectConversationStartTurnRejectsPendingInterruptAfterRestart(t *testing.T) {
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
	staleTurn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "Need approval before continuing")
	if err != nil {
		t.Fatalf("seed stale turn: %v", err)
	}
	if _, _, err := repo.CreatePendingInterrupt(ctx, conversation.ID, staleTurn.ID, "req-restart", chatdomain.InterruptKindCommandExecutionApproval, map[string]any{
		"provider": "codex",
	}); err != nil {
		t.Fatalf("create pending interrupt: %v", err)
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
		&fakeAgentCLIProcessManager{
			process: &fakeAgentCLIProcess{
				stdin:  &trackingWriteCloser{},
				stdout: `{"response":"OK"}`,
			},
		},
		nil,
	)

	_, err = service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Try to continue without resolving approval", nil)
	if !errors.Is(err, ErrConversationInterruptPending) {
		t.Fatalf("expected ErrConversationInterruptPending, got %v", err)
	}
	reloadedTurn, err := client.ChatTurn.Get(ctx, staleTurn.ID)
	if err != nil {
		t.Fatalf("reload stale turn: %v", err)
	}
	if reloadedTurn.Status != string(chatdomain.TurnStatusInterrupted) {
		t.Fatalf("stale turn status = %q, want %q", reloadedTurn.Status, chatdomain.TurnStatusInterrupted)
	}
}

func TestProjectConversationStartTurnRecoversInterruptedTurnWithoutPendingInterruptAfterRestart(t *testing.T) {
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
	staleTurn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "The interrupted turn lost its runtime")
	if err != nil {
		t.Fatalf("seed stale turn: %v", err)
	}
	if err := client.ChatTurn.UpdateOneID(staleTurn.ID).
		SetStatus(string(chatdomain.TurnStatusInterrupted)).
		ClearCompletedAt().
		Exec(ctx); err != nil {
		t.Fatalf("mark turn interrupted: %v", err)
	}
	conversation, err = repo.UpdateConversationAnchors(ctx, conversation.ID, chatdomain.ConversationStatusInterrupted, chatdomain.ConversationAnchors{
		ProviderThreadID:     optionalString("thread-interrupted"),
		LastTurnID:           optionalString("provider-turn-old"),
		ProviderThreadStatus: optionalString("active"),
		RollingSummary:       "The previous turn was interrupted unexpectedly.",
	})
	if err != nil {
		t.Fatalf("mark conversation interrupted: %v", err)
	}

	fakeCodex := &fakeProjectConversationCodexRuntime{
		startStream: TurnStream{Events: streamWithEvents(
			StreamEvent{Event: "done", Payload: donePayload{SessionID: conversation.ID.String()}},
		)},
		anchor: RuntimeSessionAnchor{
			ProviderThreadID:          "thread-interrupted",
			LastTurnID:                "provider-turn-new",
			ProviderThreadStatus:      "idle",
			ProviderThreadActiveFlags: []string{},
		},
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
		&fakeAgentCLIProcessManager{
			process: &fakeAgentCLIProcess{
				stdin:  &trackingWriteCloser{},
				stdout: `{"response":"OK"}`,
			},
		},
		nil,
	)
	service.newCodexRuntime = func(provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		return fakeCodex, nil
	}

	newTurn, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Resume after interrupted restart", nil)
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if newTurn.ID == staleTurn.ID {
		t.Fatalf("expected a new turn, got stale turn id %s", newTurn.ID)
	}
	reloadedStaleTurn, err := client.ChatTurn.Get(ctx, staleTurn.ID)
	if err != nil {
		t.Fatalf("reload stale turn: %v", err)
	}
	if reloadedStaleTurn.Status != string(chatdomain.TurnStatusTerminated) {
		t.Fatalf("stale turn status = %q, want %q", reloadedStaleTurn.Status, chatdomain.TurnStatusTerminated)
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

	remoteRepoPath, _ := createConversationRemoteRepo(t, "develop", map[string]string{
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
					DefaultBranch: "develop",
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
	platform := &fakeProjectConversationAgentPlatform{}
	service.ConfigurePlatformEnvironment("http://127.0.0.1:19836/api/v1/platform", platform)
	secretSvc, err := secretsservice.New(secretsrepo.NewEntRepository(client), "project-conversation-start-turn-secret-test")
	if err != nil {
		t.Fatalf("create secret service: %v", err)
	}
	projectSecret, err := secretSvc.CreateSecret(ctx, secretsservice.CreateSecretInput{
		ProjectID: project.ID,
		Scope:     string(secretsdomain.ScopeKindProject),
		Name:      "PROJECT_START_TURN_KEY",
		Value:     "sk-start-turn",
	})
	if err != nil {
		t.Fatalf("create project secret: %v", err)
	}
	if _, err := secretsrepo.NewEntRepository(client).CreateBinding(ctx, secretsdomain.Binding{
		OrganizationID:  org.ID,
		ProjectID:       project.ID,
		SecretID:        projectSecret.ID,
		Scope:           secretsdomain.BindingScopeKindProject,
		ScopeResourceID: project.ID,
		BindingKey:      "OPENAI_API_KEY",
	}); err != nil {
		t.Fatalf("create project secret binding: %v", err)
	}
	service.ConfigureSecretManager(secretSvc)

	if _, err := service.StartTurn(ctx, UserID("user:conversation"), conversation.ID, "Inspect the project", nil); err != nil {
		t.Fatalf("start conversation turn: %v", err)
	}

	workspacePath := filepath.Join(
		workspaceRoot,
		org.ID.String(),
		"openase",
		projectConversationWorkspaceName(conversation.ID),
	)
	skillTarget, err := workflowservice.ResolveSkillTargetForRuntime(
		workspacePath,
		string(catalog.providerByID[providerID].AdapterType),
	)
	if err != nil {
		t.Fatalf("resolve skill target: %v", err)
	}
	assertConversationFileExists(t, filepath.Join(workspacePath, "backend", "README.md"))
	assertConversationFileExists(t, filepath.Join(workspacePath, ".openase", "bin", "openase"))
	assertConversationFileExists(t, filepath.Join(skillTarget.SkillsDir, "openase-platform", "SKILL.md"))

	repository, err := git.PlainOpen(filepath.Join(workspacePath, "backend"))
	if err != nil {
		t.Fatalf("open prepared repo: %v", err)
	}
	head, err := repository.Head()
	if err != nil {
		t.Fatalf("repository head: %v", err)
	}
	if head.Name().Short() != "develop" {
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
	if !containsEnvironmentPrefix(environment, "OPENASE_CONVERSATION_ID="+conversation.ID.String()) {
		t.Fatalf("expected OPENASE_CONVERSATION_ID in environment, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_PRINCIPAL_KIND=project_conversation") {
		t.Fatalf("expected OPENASE_PRINCIPAL_KIND in environment, got %+v", environment)
	}
	if !containsEnvironmentPrefix(environment, expectedProjectConversationScopesEnvPrefix()) {
		t.Fatalf("expected OPENASE_AGENT_SCOPES in environment, got %+v", environment)
	}
	if value, ok := provider.LookupEnvironmentValue(environment, "OPENAI_API_KEY"); !ok || value != "sk-start-turn" {
		t.Fatalf("expected secret-bound OPENAI_API_KEY in environment, got %+v", environment)
	}
	if platform.lastInput.PrincipalKind != agentplatform.PrincipalKindProjectConversation {
		t.Fatalf("expected project conversation principal token, got %+v", platform.lastInput)
	}
	if platform.lastInput.PrincipalID != conversation.ID || platform.lastInput.ConversationID != conversation.ID {
		t.Fatalf("unexpected principal ids: %+v", platform.lastInput)
	}
	if platform.lastInput.AgentID != uuid.Nil || platform.lastInput.TicketID != uuid.Nil {
		t.Fatalf("project conversation token should not carry agent/ticket ids: %+v", platform.lastInput)
	}
}

func expectedProjectConversationScopesEnvPrefix() string {
	scopes := append(
		agentplatform.DefaultScopesForPrincipalKind(agentplatform.PrincipalKindProjectConversation),
		agentplatform.PrivilegedScopesForPrincipalKind(agentplatform.PrincipalKindProjectConversation)...,
	)
	scopes = slices.Compact(scopes)
	slices.Sort(scopes)
	return "OPENASE_AGENT_SCOPES=" + strings.Join(scopes, ",")
}

func TestProjectConversationCreateConversationPersistsPrincipalWithoutSyntheticAgent(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()

	org, project := createProjectConversationTestProject(ctx, t, client)
	providerID := uuid.New()
	machineID := uuid.New()
	catalog := fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
				Slug:           "openase",
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
	}
	service := NewProjectConversationService(nil, chatrepo.NewEntRepository(client), catalog, fakeTicketReader{}, nil, nil, nil)

	conversation, err := service.CreateConversation(ctx, UserID("user:conversation"), project.ID, providerID)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	principal, err := client.ProjectConversationPrincipal.Get(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("load project conversation principal: %v", err)
	}
	if principal.ConversationID != conversation.ID || principal.ProjectID != project.ID || principal.Name != projectConversationPrincipalName(conversation.ID) {
		t.Fatalf("unexpected principal row: %+v", principal)
	}
	agentCount, err := client.Agent.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count agents: %v", err)
	}
	if agentCount != 0 {
		t.Fatalf("expected no synthetic agents, found %d", agentCount)
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
		if repo.Name != "backend" || repo.Branch != "main" {
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
		fixture.service.runtimeManager.live[fixture.conversation.ID] = &liveProjectConversation{
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

func TestProjectConversationWorkspaceDiffToleratesUnresolvedWorkspaceDuringInitialSession(t *testing.T) {
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

	service := NewProjectConversationService(nil, repoStore, fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{
				ID:             project.ID,
				OrganizationID: org.ID,
				Name:           "OpenASE",
				Slug:           "openase",
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
			ID:   machineID,
			Name: "remote-builder",
			Host: "10.0.0.25",
		},
	}, nil, nil, nil, nil)

	summary, err := service.GetWorkspaceDiff(ctx, UserID("user:conversation"), conversation.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceDiff() error = %v", err)
	}
	if summary.ConversationID != conversation.ID || summary.WorkspacePath != "" || summary.Dirty || summary.ReposChanged != 0 || summary.FilesChanged != 0 || len(summary.Repos) != 0 {
		t.Fatalf("unexpected summary for unresolved initial workspace: %+v", summary)
	}
}

func TestProjectConversationWorkspaceDiffStillFailsForStartedConversationWithoutResolvableWorkspace(t *testing.T) {
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
		ProviderThreadID: optionalString("thread-visible"),
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
				Slug:           "openase",
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
			ID:   machineID,
			Name: "remote-builder",
			Host: "10.0.0.25",
		},
	}, nil, nil, nil, nil)

	_, err = service.GetWorkspaceDiff(ctx, UserID("user:conversation"), conversation.ID)
	if err == nil || !strings.Contains(err.Error(), "missing workspace_root") {
		t.Fatalf("GetWorkspaceDiff() error = %v, want missing workspace_root", err)
	}
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
	sessionID     SessionID
	requestID     string
	kind          string
	decision      string
	answer        map[string]any
	ensureInput   RuntimeTurnInput
	ensureErr     error
	startInput    RuntimeTurnInput
	startStream   TurnStream
	respondInput  RuntimeInterruptResponseInput
	respondStream TurnStream
	respondErr    error
	anchor        RuntimeSessionAnchor
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
	input RuntimeInterruptResponseInput,
) (TurnStream, error) {
	r.sessionID = input.SessionID
	r.requestID = input.RequestID
	r.kind = input.Kind
	r.decision = input.Decision
	r.answer = input.Answer
	r.respondInput = input
	if r.respondErr != nil {
		return TurnStream{}, r.respondErr
	}
	if r.respondStream.Events == nil {
		return TurnStream{}, nil
	}
	return r.respondStream, nil
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

func requireProjectConversationStreamEvent(t *testing.T, events <-chan StreamEvent) StreamEvent {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("expected stream event, got closed channel")
		}
		return event
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for stream event")
		return StreamEvent{}
	}
}

func requireNoProjectConversationStreamEvent(t *testing.T, events <-chan StreamEvent) {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			return
		}
		t.Fatalf("expected no stream event, got %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

func testTaskNotificationPayload(marker string) map[string]any {
	return map[string]any{
		"type": "task_notification",
		"raw": map[string]any{
			"marker": marker,
		},
	}
}

func requireProjectConversationMuxEvent(
	t *testing.T,
	events <-chan ProjectConversationMuxEvent,
) ProjectConversationMuxEvent {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("expected mux stream event, got closed channel")
		}
		return event
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for mux stream event")
		return ProjectConversationMuxEvent{}
	}
}

func requireNoProjectConversationMuxEvent(t *testing.T, events <-chan ProjectConversationMuxEvent) {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			return
		}
		t.Fatalf("expected no mux stream event, got %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
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
	// #nosec G302 -- test wrapper must be executable in the temp workspace.
	if err := os.Chmod(wrapperPath, 0o700); err != nil {
		return workflowservice.RefreshSkillsResult{}, err
	}
	return workflowservice.RefreshSkillsResult{
		SkillsDir:      target.SkillsDir,
		InjectedSkills: []string{"openase-platform"},
	}, nil
}

type fakeProjectConversationAgentPlatform struct {
	lastInput agentplatform.IssueInput
}

func (p *fakeProjectConversationAgentPlatform) IssueToken(
	_ context.Context,
	input agentplatform.IssueInput,
) (agentplatform.IssuedToken, error) {
	p.lastInput = input
	return agentplatform.IssuedToken{Token: "project-conversation-placeholder"}, nil
}

type capturingProjectConversationAgentPlatform struct {
	lastInput agentplatform.IssueInput
}

func (p *capturingProjectConversationAgentPlatform) IssueToken(
	_ context.Context,
	input agentplatform.IssueInput,
) (agentplatform.IssuedToken, error) {
	p.lastInput = input
	return agentplatform.IssuedToken{Token: "project-conversation-placeholder"}, nil
}

type fakeProjectConversationSecretManager struct {
	resolveBoundForRuntime func(context.Context, secretsservice.ResolveBoundRuntimeInput) ([]secretsdomain.ResolvedSecret, error)
}

func (m *fakeProjectConversationSecretManager) ResolveBoundForRuntime(
	ctx context.Context,
	input secretsservice.ResolveBoundRuntimeInput,
) ([]secretsdomain.ResolvedSecret, error) {
	if m == nil || m.resolveBoundForRuntime == nil {
		return nil, nil
	}
	return m.resolveBoundForRuntime(ctx, input)
}

func newProjectConversationTestSSHPool(t *testing.T, client sshinfra.Client) *sshinfra.Pool {
	t.Helper()
	return sshinfra.NewPool(t.TempDir(), sshinfra.WithDialer(&projectConversationSSHDialer{client: client}), sshinfra.WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))
}

type projectConversationSSHDialer struct {
	client sshinfra.Client
}

func (d *projectConversationSSHDialer) DialContext(context.Context, sshinfra.DialConfig) (sshinfra.Client, error) {
	return d.client, nil
}

type projectConversationSSHClient struct {
	sessions   []sshinfra.Session
	sessionIdx int
}

func (c *projectConversationSSHClient) NewSession() (sshinfra.Session, error) {
	if c.sessionIdx >= len(c.sessions) {
		return nil, fmt.Errorf("unexpected ssh session request %d", c.sessionIdx)
	}
	session := c.sessions[c.sessionIdx]
	c.sessionIdx++
	return session, nil
}

func (c *projectConversationSSHClient) SendRequest(string, bool, []byte) (bool, []byte, error) {
	return true, nil, nil
}

func (c *projectConversationSSHClient) Close() error {
	return nil
}

type projectConversationSSHPrepareSession struct {
	command string
	output  []byte
	err     error
}

func (s *projectConversationSSHPrepareSession) CombinedOutput(cmd string) ([]byte, error) {
	s.command = cmd
	return s.output, s.err
}

func (s *projectConversationSSHPrepareSession) StdinPipe() (io.WriteCloser, error) {
	return nil, fmt.Errorf("not supported")
}

func (s *projectConversationSSHPrepareSession) StdoutPipe() (io.Reader, error) {
	return strings.NewReader(""), nil
}

func (s *projectConversationSSHPrepareSession) StderrPipe() (io.Reader, error) {
	return strings.NewReader(""), nil
}

func (s *projectConversationSSHPrepareSession) Start(string) error {
	return fmt.Errorf("not supported")
}

func (s *projectConversationSSHPrepareSession) Signal(string) error { return nil }

func (s *projectConversationSSHPrepareSession) Wait() error { return nil }

func (s *projectConversationSSHPrepareSession) Close() error { return nil }

type projectConversationSSHProcessSession struct {
	stdin          io.WriteCloser
	stdout         string
	stderr         string
	waitErr        error
	startedCommand string
}

func (s *projectConversationSSHProcessSession) CombinedOutput(string) ([]byte, error) {
	return nil, fmt.Errorf("not supported")
}

func (s *projectConversationSSHProcessSession) StdinPipe() (io.WriteCloser, error) {
	if s.stdin != nil {
		return s.stdin, nil
	}
	return nopWriteCloser{}, nil
}

func (s *projectConversationSSHProcessSession) StdoutPipe() (io.Reader, error) {
	return strings.NewReader(s.stdout), nil
}

func (s *projectConversationSSHProcessSession) StderrPipe() (io.Reader, error) {
	return strings.NewReader(s.stderr), nil
}

func (s *projectConversationSSHProcessSession) Start(cmd string) error {
	s.startedCommand = cmd
	return nil
}

func (s *projectConversationSSHProcessSession) Signal(string) error { return nil }

func (s *projectConversationSSHProcessSession) Wait() error { return s.waitErr }

func (s *projectConversationSSHProcessSession) Close() error { return nil }

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

func TestProjectConversationServiceListConversationsUsesStableLocalPrincipal(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	_, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)

	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "browser-user-a",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "browser-user-b",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	items, err := service.ListConversations(ctx, LocalProjectConversationUserID, project.ID, nil)
	if err != nil {
		t.Fatalf("ListConversations() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ListConversations() returned %d conversations, want 2", len(items))
	}
	for _, item := range items {
		if item.UserID != LocalProjectConversationUserID.String() {
			t.Fatalf("conversation %s user_id = %q, want %q", item.ID, item.UserID, LocalProjectConversationUserID)
		}
	}

	reloadedFirst, err := repoStore.GetConversation(ctx, firstConversation.ID)
	if err != nil {
		t.Fatalf("GetConversation(first) error = %v", err)
	}
	if reloadedFirst.UserID != LocalProjectConversationUserID.String() {
		t.Fatalf("first conversation user_id = %q, want %q", reloadedFirst.UserID, LocalProjectConversationUserID)
	}
	reloadedSecond, err := repoStore.GetConversation(ctx, secondConversation.ID)
	if err != nil {
		t.Fatalf("GetConversation(second) error = %v", err)
	}
	if reloadedSecond.UserID != LocalProjectConversationUserID.String() {
		t.Fatalf("second conversation user_id = %q, want %q", reloadedSecond.UserID, LocalProjectConversationUserID)
	}
}

func TestProjectConversationServiceGetConversationUsesStableLocalPrincipal(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	_, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)

	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "browser-user-a",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	got, err := service.GetConversation(ctx, LocalProjectConversationUserID, conversation.ID)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if got.UserID != LocalProjectConversationUserID.String() {
		t.Fatalf("GetConversation().UserID = %q, want %q", got.UserID, LocalProjectConversationUserID)
	}
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

func projectConversationWebsocketURL(raw string) string {
	switch {
	case strings.HasPrefix(raw, "https://"):
		return "wss://" + strings.TrimPrefix(raw, "https://")
	case strings.HasPrefix(raw, "http://"):
		return "ws://" + strings.TrimPrefix(raw, "http://")
	default:
		return raw
	}
}

func writeProjectConversationFakeOpenASEBinary(t *testing.T, binDir string) {
	t.Helper()

	fakeBinaryPath := filepath.Join(binDir, "openase")
	content := `#!/bin/sh
set -eu

if [ "${1:-}" = "version" ]; then
  exit 0
fi

exit 0
`
	if err := os.WriteFile(fakeBinaryPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write fake openase binary: %v", err)
	}
	// #nosec G302 -- test binary must be executable in the temp workspace.
	if err := os.Chmod(fakeBinaryPath, 0o700); err != nil {
		t.Fatalf("write fake openase binary: %v", err)
	}
}

func intPointer(value int) *int {
	return &value
}

func optionalUUID(value uuid.UUID) *uuid.UUID {
	return &value
}
