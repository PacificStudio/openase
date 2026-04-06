package chatconversation

import (
	"context"
	"os"
	"testing"

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
