package humanauth

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entrolebinding "github.com/BetterAndBetterII/openase/ent/rolebinding"
	entuser "github.com/BetterAndBetterII/openase/ent/user"
	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
	"github.com/google/uuid"
)

var testPostgres *pgtest.Server

func TestMain(m *testing.M) {
	os.Exit(pgtest.RunTestMain(m, "humanauth_repo", func(server *pgtest.Server) {
		testPostgres = server
	}))
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return testPostgres.NewIsolatedEntClient(t)
}

func TestDeleteOrganizationRoleBindingStaysWithinScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)

	orgA := createTestOrganization(ctx, t, client, "acme")
	orgB := createTestOrganization(ctx, t, client, "beta")
	user := createTestUser(ctx, t, client, "alice@example.com")

	binding, err := client.RoleBinding.Create().
		SetScopeKind(entrolebinding.ScopeKindOrganization).
		SetScopeID(orgB.String()).
		SetSubjectKind(entrolebinding.SubjectKindUser).
		SetSubjectKey(user.ID.String()).
		SetRoleKey(string(domain.RoleOrgAdmin)).
		SetGrantedBy("system:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create role binding: %v", err)
	}

	err = repo.DeleteOrganizationRoleBinding(ctx, orgA, binding.ID)
	if !errors.Is(err, ErrRoleBindingNotFound) {
		t.Fatalf("DeleteOrganizationRoleBinding() err = %v, want ErrRoleBindingNotFound", err)
	}

	stillThere, err := client.RoleBinding.Get(ctx, binding.ID)
	if err != nil {
		t.Fatalf("load role binding after failed delete: %v", err)
	}
	if stillThere.ScopeID != orgB.String() {
		t.Fatalf("binding scope_id = %q, want %q", stillThere.ScopeID, orgB.String())
	}
}

func TestUpdateProjectRoleBindingStaysWithinScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)

	orgID := createTestOrganization(ctx, t, client, "gamma")
	projectA := createTestProject(ctx, t, client, orgID, "atlas")
	projectB := createTestProject(ctx, t, client, orgID, "zeus")
	user := createTestUser(ctx, t, client, "bob@example.com")
	groupSubject, err := domain.ParseGroupSubjectRef("platform-admins")
	if err != nil {
		t.Fatalf("ParseGroupSubjectRef() error = %v", err)
	}

	binding, err := client.RoleBinding.Create().
		SetScopeKind(entrolebinding.ScopeKindProject).
		SetScopeID(projectB.String()).
		SetSubjectKind(entrolebinding.SubjectKindUser).
		SetSubjectKey(user.ID.String()).
		SetRoleKey(string(domain.RoleProjectMember)).
		SetGrantedBy("system:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create role binding: %v", err)
	}

	_, err = repo.UpdateProjectRoleBinding(ctx, projectA, binding.ID, domain.UpdateProjectRoleBinding{
		UpdateRoleBindingMetadata: domain.UpdateRoleBindingMetadata{
			Subject:   groupSubject,
			GrantedBy: "system:update",
		},
		RoleKey: domain.ProjectRoleReviewer,
	})
	if !errors.Is(err, ErrRoleBindingNotFound) {
		t.Fatalf("UpdateProjectRoleBinding() err = %v, want ErrRoleBindingNotFound", err)
	}

	stillThere, err := client.RoleBinding.Get(ctx, binding.ID)
	if err != nil {
		t.Fatalf("load role binding after failed update: %v", err)
	}
	if stillThere.ScopeID != projectB.String() {
		t.Fatalf("binding scope_id = %q, want %q", stillThere.ScopeID, projectB.String())
	}
	if stillThere.RoleKey != string(domain.RoleProjectMember) {
		t.Fatalf("binding role_key = %q, want %q", stillThere.RoleKey, domain.RoleProjectMember)
	}
}

func TestUpsertUserFromOIDCUpdatesExistingIdentityWithoutDuplicates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)

	initialProfile := domain.OIDCProfile{
		Issuer:        "https://idp.example.com",
		Subject:       "subject-1",
		Email:         "alice@example.com",
		EmailVerified: true,
		DisplayName:   "Alice Control Plane",
		AvatarURL:     "https://cdn.example.com/alice-1.png",
		RawClaimsJSON: `{"sub":"subject-1","email":"alice@example.com"}`,
		Groups: []domain.Group{
			{Key: "platform-admins", Name: "Platform Admins"},
		},
	}

	user, identity, groups, err := repo.UpsertUserFromOIDC(ctx, initialProfile)
	if err != nil {
		t.Fatalf("initial UpsertUserFromOIDC() error = %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected one initial group, got %+v", groups)
	}

	updatedProfile := initialProfile
	updatedProfile.Email = "alice.renamed@example.com"
	updatedProfile.DisplayName = "Alice Renamed"
	updatedProfile.AvatarURL = "https://cdn.example.com/alice-2.png"
	updatedProfile.RawClaimsJSON = `{"sub":"subject-1","email":"alice.renamed@example.com"}`
	updatedProfile.Groups = []domain.Group{{Key: "incident-responders", Name: "Incident Responders"}}

	updatedUser, updatedIdentity, updatedGroups, err := repo.UpsertUserFromOIDC(ctx, updatedProfile)
	if err != nil {
		t.Fatalf("updated UpsertUserFromOIDC() error = %v", err)
	}

	if updatedUser.ID != user.ID {
		t.Fatalf("expected same user id, got %s then %s", user.ID, updatedUser.ID)
	}
	if updatedIdentity.ID != identity.ID {
		t.Fatalf("expected same identity id, got %s then %s", identity.ID, updatedIdentity.ID)
	}
	if updatedUser.PrimaryEmail != "alice.renamed@example.com" {
		t.Fatalf("primary_email = %q, want updated value", updatedUser.PrimaryEmail)
	}
	if updatedUser.DisplayName != "Alice Renamed" {
		t.Fatalf("display_name = %q, want updated value", updatedUser.DisplayName)
	}
	if updatedUser.AvatarURL != "https://cdn.example.com/alice-2.png" {
		t.Fatalf("avatar_url = %q, want updated value", updatedUser.AvatarURL)
	}
	if updatedIdentity.ClaimsVersion != identity.ClaimsVersion+1 {
		t.Fatalf("claims_version = %d, want %d", updatedIdentity.ClaimsVersion, identity.ClaimsVersion+1)
	}
	if len(updatedGroups) != 1 || updatedGroups[0].GroupKey != "incident-responders" {
		t.Fatalf("updatedGroups = %+v, want incident-responders", updatedGroups)
	}

	userCount, err := client.User.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("user count = %d, want 1", userCount)
	}
	identityCount, err := client.UserIdentity.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if identityCount != 1 {
		t.Fatalf("identity count = %d, want 1", identityCount)
	}
}

func TestUpsertUserFromOIDCRejectsAutomaticEmailBasedMerge(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	repo := NewEntRepository(client)

	_, _, _, err := repo.UpsertUserFromOIDC(ctx, domain.OIDCProfile{
		Issuer:        "https://idp.example.com",
		Subject:       "subject-1",
		Email:         "alice@example.com",
		EmailVerified: true,
		DisplayName:   "Alice",
		RawClaimsJSON: `{"sub":"subject-1","email":"alice@example.com"}`,
	})
	if err != nil {
		t.Fatalf("seed UpsertUserFromOIDC() error = %v", err)
	}

	_, _, _, err = repo.UpsertUserFromOIDC(ctx, domain.OIDCProfile{
		Issuer:        "https://idp.example.com",
		Subject:       "subject-2",
		Email:         "alice@example.com",
		EmailVerified: true,
		DisplayName:   "Alice Alias",
		RawClaimsJSON: `{"sub":"subject-2","email":"alice@example.com"}`,
	})
	if !errors.Is(err, ErrOIDCIdentityConflict) {
		t.Fatalf("UpsertUserFromOIDC() err = %v, want ErrOIDCIdentityConflict", err)
	}

	userCount, err := client.User.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("user count = %d, want 1", userCount)
	}
}

func createTestOrganization(ctx context.Context, t *testing.T, client *ent.Client, slug string) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName(slug).
		SetSlug(slug).
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization %q: %v", slug, err)
	}
	return org.ID
}

func createTestProject(ctx context.Context, t *testing.T, client *ent.Client, organizationID uuid.UUID, slug string) uuid.UUID {
	t.Helper()

	project, err := client.Project.Create().
		SetOrganizationID(organizationID).
		SetName(slug).
		SetSlug(slug).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project %q: %v", slug, err)
	}
	return project.ID
}

func createTestUser(ctx context.Context, t *testing.T, client *ent.Client, email string) domain.User {
	t.Helper()

	item, err := client.User.Create().
		SetStatus(entuser.StatusActive).
		SetPrimaryEmail(email).
		SetDisplayName(email).
		Save(ctx)
	if err != nil {
		t.Fatalf("create user %q: %v", email, err)
	}
	return domain.User{
		ID:           item.ID,
		Status:       domain.UserStatus(item.Status),
		PrimaryEmail: item.PrimaryEmail,
		DisplayName:  item.DisplayName,
		AvatarURL:    item.AvatarURL,
		LastLoginAt:  item.LastLoginAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}
