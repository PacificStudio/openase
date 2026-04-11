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

func TestUpdateRunTerminalStateMaterializesScopedDailyUsage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)
	projectID := createConversationTestProject(ctx, t, client)

	projectItem, err := client.Project.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}

	run, err := repo.CreateRun(ctx, domain.CreateRunInput{
		RunID:          uuid.New(),
		PrincipalID:    uuid.New(),
		ConversationID: uuid.New(),
		ProjectID:      projectID,
		ProviderID:     uuid.New(),
		Status:         domain.RunStatusExecuting,
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if _, err := repo.RecordRunUsage(ctx, domain.RecordRunUsageInput{
		RunID:      run.ID,
		ProjectID:  projectID,
		ProviderID: run.ProviderID,
		RecordedAt: time.Date(2026, 4, 10, 11, 58, 0, 0, time.UTC),
		Totals: domain.RunUsageSnapshot{
			InputTokens:       11,
			OutputTokens:      7,
			CachedInputTokens: 3,
			ReasoningTokens:   2,
			TotalTokens:       18,
		},
		Delta: domain.RunUsageSnapshot{
			InputTokens:       11,
			OutputTokens:      7,
			CachedInputTokens: 3,
			ReasoningTokens:   2,
			TotalTokens:       18,
		},
	}); err != nil {
		t.Fatalf("RecordRunUsage() error = %v", err)
	}

	completedStatus := domain.RunStatusCompleted
	terminalAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	updatedRun, err := repo.UpdateRun(ctx, domain.UpdateRunInput{
		RunID:      run.ID,
		Status:     &completedStatus,
		TerminalAt: &terminalAt,
	})
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}
	if updatedRun.TerminalAt == nil || !updatedRun.TerminalAt.UTC().Equal(terminalAt) {
		t.Fatalf("updated run terminal_at = %+v, want %s", updatedRun.TerminalAt, terminalAt.Format(time.RFC3339))
	}

	reloadedRun, err := client.ProjectConversationRun.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if reloadedRun.SnapshotMaterializedAt == nil {
		t.Fatalf("expected snapshot_materialized_at to be set, got %+v", reloadedRun)
	}

	orgRows, err := client.OrganizationDailyTokenUsage.Query().All(ctx)
	if err != nil {
		t.Fatalf("query organization daily usage: %v", err)
	}
	if len(orgRows) != 1 {
		t.Fatalf("expected 1 organization usage row, got %+v", orgRows)
	}
	orgRow := orgRows[0]
	if orgRow.OrganizationID != projectItem.OrganizationID || orgRow.TotalTokens != 18 || orgRow.FinalizedRunCount != 1 || orgRow.SourceMode.String() != "materialized" {
		t.Fatalf("unexpected organization usage row: %+v", orgRow)
	}

	projectRows, err := client.ProjectDailyTokenUsage.Query().All(ctx)
	if err != nil {
		t.Fatalf("query project daily usage: %v", err)
	}
	if len(projectRows) != 1 {
		t.Fatalf("expected 1 project usage row, got %+v", projectRows)
	}
	projectRow := projectRows[0]
	if projectRow.ProjectID != projectID || projectRow.TotalTokens != 18 || projectRow.FinalizedRunCount != 1 || projectRow.SourceMode.String() != "materialized" {
		t.Fatalf("unexpected project usage row: %+v", projectRow)
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
