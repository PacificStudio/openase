package chat

import (
	"context"
	"errors"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	"github.com/google/uuid"
)

func TestProjectConversationLifecycleGetConversationRejectsForeignUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()

	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:owner",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	service := NewProjectConversationService(nil, repoStore, fakeProjectConversationCatalog{
		fakeCatalogReader: fakeCatalogReader{
			project: catalogdomain.Project{ID: project.ID, OrganizationID: org.ID},
		},
	}, nil, nil, nil, nil)

	_, err = service.GetConversation(ctx, UserID("user:other"), conversation.ID)
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("GetConversation() error = %v, want %v", err, ErrConversationNotFound)
	}
}
