package chatconversation

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
	"github.com/google/uuid"
)

var testPostgres *pgtest.Server

func TestMain(m *testing.M) {
	os.Exit(pgtest.RunTestMain(m, "chatconversation_repo", func(server *pgtest.Server) {
		testPostgres = server
	}))
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return testPostgres.NewIsolatedEntClient(t)
}

func createConversationTestProject(ctx context.Context, t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-repo").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-repo").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project.ID
}

func TestCreateTurnWithUserEntryPersistsStableTitleOnlyOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)
	projectID := createConversationTestProject(ctx, t, client)

	conversation, err := repo.CreateConversation(ctx, domain.CreateConversation{
		ProjectID:  projectID,
		UserID:     "user:conversation",
		Source:     domain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	if _, _, err := repo.CreateTurnWithUserEntry(
		ctx,
		conversation.ID,
		"Fix the title drift now. Later messages must not rename the tab.",
	); err != nil {
		t.Fatalf("CreateTurnWithUserEntry(first) error = %v", err)
	}

	afterFirstTurn, err := repo.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("GetConversation() after first turn error = %v", err)
	}
	if got, want := afterFirstTurn.Title.String(), "Fix the title drift now."; got != want {
		t.Fatalf("conversation title after first turn = %q, want %q", got, want)
	}

	if _, err := repo.CompleteTurn(ctx, mustActiveTurn(ctx, t, repo, conversation.ID).ID, domain.TurnStatusCompleted, nil); err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if _, _, err := repo.CreateTurnWithUserEntry(
		ctx,
		conversation.ID,
		"Rename everything from the latest message instead",
	); err != nil {
		t.Fatalf("CreateTurnWithUserEntry(second) error = %v", err)
	}

	afterSecondTurn, err := repo.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("GetConversation() after second turn error = %v", err)
	}
	if got, want := afterSecondTurn.Title.String(), "Fix the title drift now."; got != want {
		t.Fatalf("conversation title after second turn = %q, want %q", got, want)
	}
}

func TestCreateTurnWithUserEntryAllowsNewTurnAfterCompletedInterruptedTurn(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)
	projectID := createConversationTestProject(ctx, t, client)

	conversation, err := repo.CreateConversation(ctx, domain.CreateConversation{
		ProjectID:  projectID,
		UserID:     "user:conversation",
		Source:     domain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	firstTurn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "Stop after partial output")
	if err != nil {
		t.Fatalf("CreateTurnWithUserEntry(first) error = %v", err)
	}
	if _, err := repo.CompleteTurn(ctx, firstTurn.ID, domain.TurnStatusInterrupted, nil); err != nil {
		t.Fatalf("CompleteTurn(interrupted) error = %v", err)
	}

	if _, err := repo.GetActiveTurn(ctx, conversation.ID); err == nil {
		t.Fatal("GetActiveTurn() unexpectedly found a completed interrupted turn")
	}

	secondTurn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "Continue after stop")
	if err != nil {
		t.Fatalf("CreateTurnWithUserEntry(second) error = %v", err)
	}
	if secondTurn.TurnIndex != 2 {
		t.Fatalf("second turn index = %d, want 2", secondTurn.TurnIndex)
	}
}

func TestGetConversationBackfillsLegacyTitleFromEarliestUserEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)
	projectID := createConversationTestProject(ctx, t, client)

	conversation, err := client.ChatConversation.Create().
		SetProjectID(projectID).
		SetUserID("user:conversation").
		SetSource(string(domain.SourceProjectSidebar)).
		SetProviderID(uuid.New()).
		SetStatus(string(domain.ConversationStatusActive)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create legacy conversation: %v", err)
	}
	turn, err := client.ChatTurn.Create().
		SetConversationID(conversation.ID).
		SetTurnIndex(1).
		SetStatus(string(domain.TurnStatusCompleted)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create legacy turn: %v", err)
	}
	if _, err := client.ChatEntry.Create().
		SetConversationID(conversation.ID).
		SetTurnID(turn.ID).
		SetSeq(0).
		SetKind(string(domain.EntryKindUserMessage)).
		SetPayloadJSON(map[string]any{
			"role":    "user",
			"content": "\n  稳定会话标题，不要再跟着 summary 漂移。  \n第二行只是补充说明",
		}).
		Save(ctx); err != nil {
		t.Fatalf("create legacy user entry: %v", err)
	}

	got, err := repo.GetConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if want := "稳定会话标题，不要再跟着 summary 漂移。"; got.Title.String() != want {
		t.Fatalf("backfilled conversation title = %q, want %q", got.Title.String(), want)
	}

	reloaded, err := client.ChatConversation.Get(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("reload legacy conversation: %v", err)
	}
	if want := "稳定会话标题，不要再跟着 summary 漂移。"; reloaded.Title != want {
		t.Fatalf("persisted backfilled title = %q, want %q", reloaded.Title, want)
	}
}

func TestDeleteConversationCascadesConversationScopedRecords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)
	projectID := createConversationTestProject(ctx, t, client)
	providerID := uuid.New()

	conversation, err := repo.CreateConversation(ctx, domain.CreateConversation{
		ProjectID:  projectID,
		UserID:     "user:conversation",
		Source:     domain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	turn, _, err := repo.CreateTurnWithUserEntry(ctx, conversation.ID, "delete this conversation")
	if err != nil {
		t.Fatalf("CreateTurnWithUserEntry() error = %v", err)
	}
	if _, _, err := repo.CreatePendingInterrupt(
		ctx,
		conversation.ID,
		turn.ID,
		"req-delete",
		domain.InterruptKindCommandExecutionApproval,
		map[string]any{"reason": "review"},
	); err != nil {
		t.Fatalf("CreatePendingInterrupt() error = %v", err)
	}

	principal, err := repo.EnsurePrincipal(ctx, domain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      projectID,
		ProviderID:     providerID,
		Name:           "project-conversation-delete",
	})
	if err != nil {
		t.Fatalf("EnsurePrincipal() error = %v", err)
	}
	run, err := repo.CreateRun(ctx, domain.CreateRunInput{
		RunID:          uuid.New(),
		PrincipalID:    principal.ID,
		ConversationID: conversation.ID,
		ProjectID:      projectID,
		ProviderID:     providerID,
		TurnID:         &turn.ID,
		Status:         domain.RunStatusExecuting,
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	trace, err := repo.AppendTraceEvent(ctx, domain.AppendTraceEventInput{
		RunID:          run.ID,
		PrincipalID:    principal.ID,
		ConversationID: conversation.ID,
		ProjectID:      projectID,
		Provider:       "codex",
		Kind:           "output_text.delta",
		Stream:         "stdout",
		Payload:        map[string]any{"text": "hello"},
	})
	if err != nil {
		t.Fatalf("AppendTraceEvent() error = %v", err)
	}
	stepSummary := "step"
	if _, err := repo.AppendStepEvent(ctx, domain.AppendStepEventInput{
		RunID:              run.ID,
		PrincipalID:        principal.ID,
		ConversationID:     conversation.ID,
		ProjectID:          projectID,
		StepStatus:         "running",
		Summary:            &stepSummary,
		SourceTraceEventID: &trace.ID,
	}); err != nil {
		t.Fatalf("AppendStepEvent() error = %v", err)
	}
	if _, err := client.AgentToken.Create().
		SetProjectID(projectID).
		SetConversationID(conversation.ID).
		SetPrincipalKind("project_conversation").
		SetPrincipalID(principal.ID).
		SetPrincipalName(principal.Name).
		SetTokenHash("hash-delete-conversation").
		SetScopes([]string{}).
		SetExpiresAt(time.Now().UTC().Add(time.Hour)).
		Save(ctx); err != nil {
		t.Fatalf("create agent token: %v", err)
	}

	result, err := repo.DeleteConversation(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("DeleteConversation() error = %v", err)
	}

	if result.ConversationID != conversation.ID || result.ProjectID != projectID || result.UserID != "user:conversation" {
		t.Fatalf("DeleteConversation() identity = %+v", result)
	}
	if result.EntriesDeleted != 2 || result.TurnsDeleted != 1 || result.InterruptsDeleted != 1 || result.RunsDeleted != 1 || result.TraceEventsDeleted != 1 || result.StepEventsDeleted != 1 || result.AgentTokensDeleted != 1 {
		t.Fatalf("DeleteConversation() counts = %+v", result)
	}

	if _, err := repo.GetConversation(ctx, conversation.ID); err == nil {
		t.Fatal("GetConversation() expected not found after deletion")
	}
	if count, err := client.ChatEntry.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("chat entry count = %d, %v", count, err)
	}
	if count, err := client.ChatTurn.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("chat turn count = %d, %v", count, err)
	}
	if count, err := client.ChatPendingInterrupt.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("pending interrupt count = %d, %v", count, err)
	}
	if count, err := client.ProjectConversationRun.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("run count = %d, %v", count, err)
	}
	if count, err := client.ProjectConversationTraceEvent.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("trace event count = %d, %v", count, err)
	}
	if count, err := client.ProjectConversationStepEvent.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("step event count = %d, %v", count, err)
	}
	if count, err := client.ProjectConversationPrincipal.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("principal count = %d, %v", count, err)
	}
	if count, err := client.AgentToken.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("agent token count = %d, %v", count, err)
	}
}

func mustActiveTurn(
	ctx context.Context,
	t *testing.T,
	repo *Repository,
	conversationID uuid.UUID,
) domain.Turn {
	t.Helper()

	turn, err := repo.GetActiveTurn(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetActiveTurn() error = %v", err)
	}
	return turn
}
