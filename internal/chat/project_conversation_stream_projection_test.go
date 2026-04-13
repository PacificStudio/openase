package chat

import (
	"context"
	"testing"

	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	"github.com/google/uuid"
)

func TestProjectConversationHandleConversationMessageAddsDiffEntryID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repoStore := chatrepo.NewEntRepository(client)
	_, project := createProjectConversationTestProject(ctx, t, client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:projection",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	turn, _, err := repoStore.CreateTurnWithUserEntry(ctx, conversation.ID, "show diff")
	if err != nil {
		t.Fatalf("CreateTurnWithUserEntry() error = %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)
	payload := map[string]any{
		"type": chatMessageTypeDiff,
		"diff": "@@ -1 +1 @@\n-old\n+new\n",
	}

	event, ok := service.handleConversationMessage(ctx, conversation.ID, turn.ID, payload)
	if !ok {
		t.Fatal("expected diff payload to be projected")
	}
	if event.Event != "message" {
		t.Fatalf("event = %q, want message", event.Event)
	}
	eventPayload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload = %#v, want map payload", event.Payload)
	}
	if _, ok := eventPayload["entry_id"].(string); !ok {
		t.Fatalf("payload = %#v, want entry_id", event.Payload)
	}

	entries, err := repoStore.ListEntries(ctx, conversation.ID)
	if err != nil {
		t.Fatalf("ListEntries() error = %v", err)
	}
	if len(entries) < 2 || entries[len(entries)-1].Kind != chatdomain.EntryKindDiff {
		t.Fatalf("entries = %#v, want trailing diff entry", entries)
	}
}
