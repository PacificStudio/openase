package humanauth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPrincipalContextRoundTrip(t *testing.T) {
	t.Parallel()

	base := context.Background()
	if _, ok := PrincipalFromContext(base); ok {
		t.Fatal("PrincipalFromContext() unexpectedly found principal in empty context")
	}

	principal := AuthenticatedPrincipal{
		User: User{ID: uuid.New()},
	}
	ctx := WithPrincipal(base, principal)

	got, ok := PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("PrincipalFromContext() did not find principal after WithPrincipal()")
	}
	if got.User.ID != principal.User.ID {
		t.Fatalf("PrincipalFromContext() user id = %s, want %s", got.User.ID, principal.User.ID)
	}
}

func TestEffectiveVisibilityAllowsOrganization(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	otherOrgID := uuid.New()

	if !(EffectiveVisibility{Instance: true}).AllowsOrganization(otherOrgID) {
		t.Fatal("instance visibility should allow any organization")
	}
	if !(EffectiveVisibility{OrganizationIDs: []uuid.UUID{orgID}}).AllowsOrganization(orgID) {
		t.Fatal("organization visibility should allow direct organization membership")
	}
	if !(EffectiveVisibility{OrganizationScopeIDs: []uuid.UUID{orgID}}).AllowsOrganization(orgID) {
		t.Fatal("organization scope visibility should allow organization access")
	}
	if (EffectiveVisibility{}).AllowsOrganization(otherOrgID) {
		t.Fatal("empty visibility should not allow unrelated organization")
	}
}

func TestEffectiveVisibilityAllowsOrganizationScope(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	if !(EffectiveVisibility{Instance: true}).AllowsOrganizationScope(orgID) {
		t.Fatal("instance visibility should allow any organization scope")
	}
	if !(EffectiveVisibility{OrganizationScopeIDs: []uuid.UUID{orgID}}).AllowsOrganizationScope(orgID) {
		t.Fatal("organization scope visibility should allow matching organization scope")
	}
	if (EffectiveVisibility{OrganizationIDs: []uuid.UUID{orgID}}).AllowsOrganizationScope(uuid.New()) {
		t.Fatal("direct organization visibility should not allow unrelated organization scope")
	}
}

func TestEffectiveVisibilityAllowsProject(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	orgID := uuid.New()

	if !(EffectiveVisibility{Instance: true}).AllowsProject(uuid.New(), uuid.New()) {
		t.Fatal("instance visibility should allow any project")
	}
	if !(EffectiveVisibility{ProjectIDs: []uuid.UUID{projectID}}).AllowsProject(projectID, uuid.New()) {
		t.Fatal("project visibility should allow matching project")
	}
	if !(EffectiveVisibility{OrganizationScopeIDs: []uuid.UUID{orgID}}).AllowsProject(uuid.New(), orgID) {
		t.Fatal("organization scope visibility should allow descendant project")
	}
	if (EffectiveVisibility{}).AllowsProject(uuid.New(), uuid.New()) {
		t.Fatal("empty visibility should not allow unrelated project")
	}
}
