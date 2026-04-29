package chat

import (
	"context"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	"github.com/google/uuid"
)

func TestNormalizeProjectConversationOwnerForCreate(t *testing.T) {
	t.Parallel()

	humanID := uuid.MustParse("8db7261e-e16d-458e-8926-cd01550686a5")
	browserSessionID := uuid.MustParse("d2a5f692-24e8-421f-94cb-f4670e1f91c1")

	tests := []struct {
		name string
		raw  UserID
		want UserID
	}{
		{
			name: "stable human owner stays stable",
			raw:  UserID("user:" + humanID.String()),
			want: UserID("user:" + humanID.String()),
		},
		{
			name: "bare uuid converges to stable human owner",
			raw:  UserID(humanID.String()),
			want: UserID("user:" + humanID.String()),
		},
		{
			name: "legacy local owner converges to instance admin",
			raw:  LegacyLocalProjectConversationUserID,
			want: InstanceAdminProjectConversationUserID,
		},
		{
			name: "anonymous converges to instance admin",
			raw:  AnonymousUserID,
			want: InstanceAdminProjectConversationUserID,
		},
		{
			name: "browser session converges to instance admin",
			raw:  UserID(browserSessionProjectConversationUserIDPrefix + browserSessionID.String()),
			want: InstanceAdminProjectConversationUserID,
		},
		{
			name: "unknown owner converges to instance admin",
			raw:  UserID("browser-user-a"),
			want: InstanceAdminProjectConversationUserID,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeProjectConversationOwnerForCreate(tc.raw); got != tc.want {
				t.Fatalf("normalizeProjectConversationOwnerForCreate(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestResolveProjectConversationOwnerAccess(t *testing.T) {
	t.Parallel()

	humanID := uuid.MustParse("8db7261e-e16d-458e-8926-cd01550686a5")
	humanUser := UserID("user:" + humanID.String())
	legacyBrowser := UserID(browserSessionProjectConversationUserIDPrefix + uuid.MustParse("d2a5f692-24e8-421f-94cb-f4670e1f91c1").String())
	otherHuman := "user:" + uuid.MustParse("bf356dd8-aed8-4582-9278-69e72d68df79").String()

	tests := []struct {
		name   string
		hint   ProjectConversationAccessHint
		want   projectConversationOwnerAccess
		stored string
	}{
		{
			name:   "bare uuid adopts into stable human owner",
			stored: humanID.String(),
			want: projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: humanUser,
				needsMigration: true,
			},
		},
		{
			name:   "matching legacy browser owner adopts into stable human owner",
			stored: legacyBrowser.String(),
			hint: ProjectConversationAccessHint{
				LegacyBrowserOwner: legacyBrowser,
			},
			want: projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: humanUser,
				needsMigration: true,
			},
		},
		{
			name:   "instance admin legacy owner can be adopted by human principal",
			stored: LegacyLocalProjectConversationUserID.String(),
			hint: ProjectConversationAccessHint{
				AllowInstanceAdminLegacyAdoption: true,
			},
			want: projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: humanUser,
				needsMigration: true,
			},
		},
		{
			name:   "foreign stable human owner stays hidden",
			stored: otherHuman,
			want:   projectConversationOwnerAccess{},
		},
		{
			name:   "auth disabled access converges browser owner to instance admin",
			stored: legacyBrowser.String(),
			want: projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: InstanceAdminProjectConversationUserID,
				needsMigration: true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			requested := humanUser
			if tc.name == "auth disabled access converges browser owner to instance admin" {
				requested = InstanceAdminProjectConversationUserID
			}

			if got := resolveProjectConversationOwnerAccess(requested, tc.stored, tc.hint); got != tc.want {
				t.Fatalf("resolveProjectConversationOwnerAccess(%q, %q, %+v) = %+v, want %+v", requested, tc.stored, tc.hint, got, tc.want)
			}
		})
	}
}

func TestProjectConversationServiceListConversationsAdoptsLegacyOwnersForHumanPrincipal(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	_, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	service := NewProjectConversationService(nil, repoStore, nil, nil, nil, nil, nil)

	humanID := uuid.MustParse("8db7261e-e16d-458e-8926-cd01550686a5")
	stableHumanUser := UserID("user:" + humanID.String())
	legacyBrowserOwner := UserID(browserSessionProjectConversationUserIDPrefix + uuid.MustParse("d2a5f692-24e8-421f-94cb-f4670e1f91c1").String())
	foreignStableOwner := "user:" + uuid.MustParse("bf356dd8-aed8-4582-9278-69e72d68df79").String()

	createConversation := func(t *testing.T, userID string) chatdomain.Conversation {
		t.Helper()

		conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
			ProjectID:  project.ID,
			UserID:     userID,
			Source:     chatdomain.SourceProjectSidebar,
			ProviderID: uuid.New(),
		})
		if err != nil {
			t.Fatalf("create conversation for %q: %v", userID, err)
		}
		return conversation
	}

	bareUUIDConversation := createConversation(t, humanID.String())
	browserConversation := createConversation(t, legacyBrowserOwner.String())
	legacyLocalConversation := createConversation(t, LegacyLocalProjectConversationUserID.String())
	anonymousConversation := createConversation(t, AnonymousUserID.String())
	unknownConversation := createConversation(t, "browser-user-a")
	foreignConversation := createConversation(t, foreignStableOwner)

	hintCtx := WithProjectConversationAccessHint(ctx, ProjectConversationAccessHint{
		LegacyBrowserOwner:               legacyBrowserOwner,
		AllowInstanceAdminLegacyAdoption: true,
	})

	items, err := service.ListConversations(hintCtx, stableHumanUser, project.ID, nil)
	if err != nil {
		t.Fatalf("ListConversations() error = %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("ListConversations() returned %d conversations, want 5", len(items))
	}
	for _, item := range items {
		if item.UserID != stableHumanUser.String() {
			t.Fatalf("conversation %s user_id = %q, want %q", item.ID, item.UserID, stableHumanUser)
		}
	}

	for _, conversationID := range []uuid.UUID{
		bareUUIDConversation.ID,
		browserConversation.ID,
		legacyLocalConversation.ID,
		anonymousConversation.ID,
		unknownConversation.ID,
	} {
		reloaded, err := repoStore.GetConversation(ctx, conversationID)
		if err != nil {
			t.Fatalf("GetConversation(%s) error = %v", conversationID, err)
		}
		if reloaded.UserID != stableHumanUser.String() {
			t.Fatalf("conversation %s stored user_id = %q, want %q", conversationID, reloaded.UserID, stableHumanUser)
		}
	}

	reloadedForeign, err := repoStore.GetConversation(ctx, foreignConversation.ID)
	if err != nil {
		t.Fatalf("GetConversation(foreign) error = %v", err)
	}
	if reloadedForeign.UserID != foreignStableOwner {
		t.Fatalf("foreign conversation user_id = %q, want %q", reloadedForeign.UserID, foreignStableOwner)
	}
}

func TestProjectConversationServiceCreateConversationNormalizesOwnerOnWrite(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	ctx := context.Background()
	org, project := createProjectConversationTestProject(ctx, t, client)
	repoStore := chatrepo.NewEntRepository(client)
	providerID := uuid.New()

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
					MachineID:      uuid.New(),
					AdapterType:    catalogdomain.AgentProviderAdapterTypeGeminiCLI,
					CliCommand:     "gemini",
				},
			},
		},
	}
	service := NewProjectConversationService(nil, repoStore, catalog, fakeTicketReader{}, nil, nil, nil)

	humanID := uuid.MustParse("8db7261e-e16d-458e-8926-cd01550686a5")

	tests := []struct {
		name     string
		userID   UserID
		wantUser string
	}{
		{
			name:     "bare uuid is written as stable human owner",
			userID:   UserID(humanID.String()),
			wantUser: "user:" + humanID.String(),
		},
		{
			name:     "legacy local owner is written as instance admin",
			userID:   LegacyLocalProjectConversationUserID,
			wantUser: InstanceAdminProjectConversationUserID.String(),
		},
		{
			name:     "browser session owner is written as instance admin",
			userID:   UserID(browserSessionProjectConversationUserIDPrefix + uuid.NewString()),
			wantUser: InstanceAdminProjectConversationUserID.String(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conversation, err := service.CreateConversation(ctx, tc.userID, project.ID, providerID)
			if err != nil {
				t.Fatalf("CreateConversation() error = %v", err)
			}
			if conversation.UserID != tc.wantUser {
				t.Fatalf("CreateConversation().UserID = %q, want %q", conversation.UserID, tc.wantUser)
			}

			reloaded, err := repoStore.GetConversation(ctx, conversation.ID)
			if err != nil {
				t.Fatalf("GetConversation() error = %v", err)
			}
			if reloaded.UserID != tc.wantUser {
				t.Fatalf("stored conversation user_id = %q, want %q", reloaded.UserID, tc.wantUser)
			}
		})
	}
}
