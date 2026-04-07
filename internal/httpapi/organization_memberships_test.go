package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entorganizationmembership "github.com/BetterAndBetterII/openase/ent/organizationmembership"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/google/uuid"
)

func TestOrganizationInvitationAcceptFlowActivatesMembershipAndVisibility(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	ownerToken, ownerCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	setOrganizationMembershipRole(t, fixture, orgID, fixture.userIDByEmail(t, "owner@example.com"), entorganizationmembership.RoleOwner)

	inviteRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/orgs/"+orgID.String()+"/invitations",
		`{"email":"invitee@example.com","role":"member"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + ownerToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": ownerCSRF,
			"User-Agent":     "OrgInviteTest/1.0",
		},
	)
	if inviteRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", inviteRec.Code, inviteRec.Body.String())
	}

	var invitePayload struct {
		AcceptToken string `json:"accept_token"`
		Membership  struct {
			ID     string `json:"id"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"membership"`
		Invitation struct {
			Status string `json:"status"`
			Email  string `json:"email"`
		} `json:"invitation"`
	}
	decodeResponse(t, inviteRec, &invitePayload)
	if invitePayload.AcceptToken == "" {
		t.Fatal("expected accept token in invitation response")
	}
	if invitePayload.Membership.Status != string(humanauthdomain.OrganizationMembershipStatusInvited) {
		t.Fatalf("membership.status = %q, want invited", invitePayload.Membership.Status)
	}
	if invitePayload.Invitation.Status != string(humanauthdomain.OrganizationInvitationStatusPending) {
		t.Fatalf("invitation.status = %q, want pending", invitePayload.Invitation.Status)
	}

	inviteeToken, inviteeCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "invitee@example.com",
		displayName: "Invitee",
	})
	acceptRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/org-invitations/accept",
		`{"token":"`+invitePayload.AcceptToken+`"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + inviteeToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": inviteeCSRF,
			"User-Agent":     "OrgAcceptTest/1.0",
		},
	)
	if acceptRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", acceptRec.Code, acceptRec.Body.String())
	}

	var acceptPayload struct {
		Membership struct {
			Role   string  `json:"role"`
			Status string  `json:"status"`
			UserID *string `json:"user_id"`
		} `json:"membership"`
	}
	decodeResponse(t, acceptRec, &acceptPayload)
	if acceptPayload.Membership.Role != "member" {
		t.Fatalf("membership.role = %q, want member", acceptPayload.Membership.Role)
	}
	if acceptPayload.Membership.Status != string(humanauthdomain.OrganizationMembershipStatusActive) {
		t.Fatalf("membership.status = %q, want active", acceptPayload.Membership.Status)
	}
	if acceptPayload.Membership.UserID == nil || *acceptPayload.Membership.UserID == "" {
		t.Fatalf("membership.user_id = %+v, want populated user id", acceptPayload.Membership.UserID)
	}

	permissionsRec := fixture.request(
		t,
		http.MethodGet,
		"/api/v1/auth/me/permissions?org_id="+orgID.String(),
		map[string]string{
			"Cookie":     humanSessionCookieName + "=" + inviteeToken,
			"User-Agent": "OrgAcceptPermissionsTest/1.0",
		},
	)
	if permissionsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", permissionsRec.Code, permissionsRec.Body.String())
	}
	var permissionsPayload struct {
		Roles []string `json:"roles"`
	}
	decodeResponse(t, permissionsRec, &permissionsPayload)
	assertStringSet(t, permissionsPayload.Roles, "org_member")

	orgsRec := fixture.request(t, http.MethodGet, "/api/v1/orgs", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + inviteeToken,
		"User-Agent": "OrgAcceptVisibilityTest/1.0",
	})
	if orgsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", orgsRec.Code, orgsRec.Body.String())
	}
	var orgsPayload struct {
		Organizations []struct {
			ID string `json:"id"`
		} `json:"organizations"`
	}
	decodeResponse(t, orgsRec, &orgsPayload)
	if len(orgsPayload.Organizations) != 1 || orgsPayload.Organizations[0].ID != orgID.String() {
		t.Fatalf("unexpected organizations payload: %+v", orgsPayload.Organizations)
	}
}

func TestOrganizationMembershipLastOwnerGuardRejectsRemoval(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	ownerToken, ownerCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	ownerUserID := fixture.userIDByEmail(t, "owner@example.com")
	setOrganizationMembershipRole(t, fixture, orgID, ownerUserID, entorganizationmembership.RoleOwner)
	ownerMembershipID := organizationMembershipIDByUser(t, fixture, orgID, ownerUserID)

	removeRec := fixture.requestJSON(
		t,
		http.MethodPatch,
		"/api/v1/orgs/"+orgID.String()+"/members/"+ownerMembershipID.String(),
		`{"status":"removed"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + ownerToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": ownerCSRF,
			"User-Agent":     "OrgLastOwnerTest/1.0",
		},
	)
	assertAPIErrorResponse(
		t,
		removeRec,
		http.StatusConflict,
		"ORGANIZATION_MEMBERSHIP_CONFLICT",
		"cannot change the last active organization owner",
	)

	membership := organizationMembershipByUser(t, fixture, orgID, ownerUserID)
	if membership.Status != entorganizationmembership.StatusActive {
		t.Fatalf("membership.status = %q, want active", membership.Status)
	}
	if membership.Role != entorganizationmembership.RoleOwner {
		t.Fatalf("membership.role = %q, want owner", membership.Role)
	}
}

func TestOrganizationMembershipTransferOwnershipPromotesTargetAndDemotesPreviousOwner(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	ownerToken, ownerCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	ownerUserID := fixture.userIDByEmail(t, "owner@example.com")
	setOrganizationMembershipRole(t, fixture, orgID, ownerUserID, entorganizationmembership.RoleOwner)

	fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "member@example.com",
		displayName: "Member",
		orgID:       orgID,
	})
	memberUserID := fixture.userIDByEmail(t, "member@example.com")
	memberMembershipID := organizationMembershipIDByUser(t, fixture, orgID, memberUserID)

	transferRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/orgs/"+orgID.String()+"/members/"+memberMembershipID.String()+"/transfer-ownership",
		`{"previous_owner_role":"admin"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + ownerToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": ownerCSRF,
			"User-Agent":     "OrgTransferOwnerTest/1.0",
		},
	)
	if transferRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", transferRec.Code, transferRec.Body.String())
	}

	memberMembership := organizationMembershipByUser(t, fixture, orgID, memberUserID)
	if memberMembership.Role != entorganizationmembership.RoleOwner {
		t.Fatalf("member membership role = %q, want owner", memberMembership.Role)
	}
	if memberMembership.Status != entorganizationmembership.StatusActive {
		t.Fatalf("member membership status = %q, want active", memberMembership.Status)
	}

	ownerMembership := organizationMembershipByUser(t, fixture, orgID, ownerUserID)
	if ownerMembership.Role != entorganizationmembership.RoleAdmin {
		t.Fatalf("previous owner role = %q, want admin", ownerMembership.Role)
	}
	if ownerMembership.Status != entorganizationmembership.StatusActive {
		t.Fatalf("previous owner status = %q, want active", ownerMembership.Status)
	}
}

func TestOrganizationMembershipRejectsActivatingInvitationBeforeAcceptance(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	ownerToken, ownerCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	ownerUserID := fixture.userIDByEmail(t, "owner@example.com")
	setOrganizationMembershipRole(t, fixture, orgID, ownerUserID, entorganizationmembership.RoleOwner)

	inviteRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/orgs/"+orgID.String()+"/invitations",
		`{"email":"pending@example.com","role":"member"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + ownerToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": ownerCSRF,
			"User-Agent":     "OrgPrematureActivationTest/1.0",
		},
	)
	if inviteRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", inviteRec.Code, inviteRec.Body.String())
	}

	var invitePayload struct {
		Membership struct {
			ID string `json:"id"`
		} `json:"membership"`
	}
	decodeResponse(t, inviteRec, &invitePayload)

	activateRec := fixture.requestJSON(
		t,
		http.MethodPatch,
		"/api/v1/orgs/"+orgID.String()+"/members/"+invitePayload.Membership.ID,
		`{"status":"active"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + ownerToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": ownerCSRF,
			"User-Agent":     "OrgPrematureActivationTest/1.0",
		},
	)
	assertAPIErrorResponse(
		t,
		activateRec,
		http.StatusConflict,
		"ORGANIZATION_MEMBERSHIP_CONFLICT",
		"organization membership cannot become active before invitation acceptance",
	)

	membership := organizationMembershipByEmail(t, fixture, orgID, "pending@example.com")
	if membership.Status != entorganizationmembership.StatusInvited {
		t.Fatalf("membership.status = %q, want invited", membership.Status)
	}
	if membership.UserID != nil {
		t.Fatalf("membership.user_id = %v, want nil", membership.UserID)
	}
	if membership.AcceptedAt != nil {
		t.Fatalf("membership.accepted_at = %v, want nil", membership.AcceptedAt)
	}
}

func TestOrganizationMembershipTransferOwnershipRejectsUnacceptedActiveMembership(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	ownerToken, ownerCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	ownerUserID := fixture.userIDByEmail(t, "owner@example.com")
	setOrganizationMembershipRole(t, fixture, orgID, ownerUserID, entorganizationmembership.RoleOwner)

	inviteRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/orgs/"+orgID.String()+"/invitations",
		`{"email":"ghost-owner@example.com","role":"member"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + ownerToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": ownerCSRF,
			"User-Agent":     "OrgTransferPendingOwnerTest/1.0",
		},
	)
	if inviteRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", inviteRec.Code, inviteRec.Body.String())
	}

	targetMembership := organizationMembershipByEmail(t, fixture, orgID, "ghost-owner@example.com")
	if _, err := fixture.client.OrganizationMembership.UpdateOneID(targetMembership.ID).
		SetStatus(entorganizationmembership.StatusActive).
		ClearUserID().
		ClearAcceptedAt().
		Save(context.Background()); err != nil {
		t.Fatalf("force invalid active membership: %v", err)
	}

	transferRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/orgs/"+orgID.String()+"/members/"+targetMembership.ID.String()+"/transfer-ownership",
		`{"previous_owner_role":"admin"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + ownerToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": ownerCSRF,
			"User-Agent":     "OrgTransferPendingOwnerTest/1.0",
		},
	)
	assertAPIErrorResponse(
		t,
		transferRec,
		http.StatusConflict,
		"ORGANIZATION_MEMBERSHIP_CONFLICT",
		"organization membership cannot become active before invitation acceptance",
	)

	ownerMembership := organizationMembershipByUser(t, fixture, orgID, ownerUserID)
	if ownerMembership.Role != entorganizationmembership.RoleOwner {
		t.Fatalf("owner membership role = %q, want owner", ownerMembership.Role)
	}
}

func TestOrganizationAdminCannotInviteElevatedRoles(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	setOrganizationMembershipRole(t, fixture, orgID, fixture.userIDByEmail(t, "owner@example.com"), entorganizationmembership.RoleOwner)

	adminToken, adminCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "admin@example.com",
		displayName: "Admin",
		orgID:       orgID,
	})
	setOrganizationMembershipRole(t, fixture, orgID, fixture.userIDByEmail(t, "admin@example.com"), entorganizationmembership.RoleAdmin)

	inviteRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/orgs/"+orgID.String()+"/invitations",
		`{"email":"new-admin@example.com","role":"admin"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + adminToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": adminCSRF,
			"User-Agent":     "OrgAdminInviteElevatedTest/1.0",
		},
	)
	assertAPIErrorResponse(
		t,
		inviteRec,
		http.StatusForbidden,
		"ORGANIZATION_MEMBERSHIP_FORBIDDEN",
		"organization owner role is required to grant or revoke org_owner or org_admin",
	)
}

func TestOrganizationAdminCannotPromoteMemberToAdmin(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	setOrganizationMembershipRole(t, fixture, orgID, fixture.userIDByEmail(t, "owner@example.com"), entorganizationmembership.RoleOwner)

	adminToken, adminCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "admin@example.com",
		displayName: "Admin",
		orgID:       orgID,
	})
	setOrganizationMembershipRole(t, fixture, orgID, fixture.userIDByEmail(t, "admin@example.com"), entorganizationmembership.RoleAdmin)

	fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "member@example.com",
		displayName: "Member",
		orgID:       orgID,
	})
	memberUserID := fixture.userIDByEmail(t, "member@example.com")
	memberMembershipID := organizationMembershipIDByUser(t, fixture, orgID, memberUserID)

	promoteRec := fixture.requestJSON(
		t,
		http.MethodPatch,
		"/api/v1/orgs/"+orgID.String()+"/members/"+memberMembershipID.String(),
		`{"role":"admin"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + adminToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": adminCSRF,
			"User-Agent":     "OrgAdminPromoteMemberTest/1.0",
		},
	)
	assertAPIErrorResponse(
		t,
		promoteRec,
		http.StatusForbidden,
		"ORGANIZATION_MEMBERSHIP_FORBIDDEN",
		"organization owner role is required to grant or revoke org_owner or org_admin",
	)

	memberMembership := organizationMembershipByUser(t, fixture, orgID, memberUserID)
	if memberMembership.Role != entorganizationmembership.RoleMember {
		t.Fatalf("member role = %q, want member", memberMembership.Role)
	}
}

func TestOrganizationAdminCannotSuspendAdminMembership(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, _ := fixture.createOrganizationProject(t)

	fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "owner@example.com",
		displayName: "Owner",
		orgID:       orgID,
	})
	setOrganizationMembershipRole(t, fixture, orgID, fixture.userIDByEmail(t, "owner@example.com"), entorganizationmembership.RoleOwner)

	adminToken, adminCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "admin@example.com",
		displayName: "Admin",
		orgID:       orgID,
	})
	adminUserID := fixture.userIDByEmail(t, "admin@example.com")
	setOrganizationMembershipRole(t, fixture, orgID, adminUserID, entorganizationmembership.RoleAdmin)

	fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "other-admin@example.com",
		displayName: "Other Admin",
		orgID:       orgID,
	})
	otherAdminUserID := fixture.userIDByEmail(t, "other-admin@example.com")
	setOrganizationMembershipRole(t, fixture, orgID, otherAdminUserID, entorganizationmembership.RoleAdmin)
	otherAdminMembershipID := organizationMembershipIDByUser(t, fixture, orgID, otherAdminUserID)

	suspendRec := fixture.requestJSON(
		t,
		http.MethodPatch,
		"/api/v1/orgs/"+orgID.String()+"/members/"+otherAdminMembershipID.String(),
		`{"status":"suspended"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + adminToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": adminCSRF,
			"User-Agent":     "OrgAdminSuspendAdminTest/1.0",
		},
	)
	assertAPIErrorResponse(
		t,
		suspendRec,
		http.StatusForbidden,
		"ORGANIZATION_MEMBERSHIP_FORBIDDEN",
		"organization owner role is required to manage owner or admin memberships",
	)

	otherAdminMembership := organizationMembershipByUser(t, fixture, orgID, otherAdminUserID)
	if otherAdminMembership.Status != entorganizationmembership.StatusActive {
		t.Fatalf("other admin status = %q, want active", otherAdminMembership.Status)
	}
}

func organizationMembershipByUser(
	t *testing.T,
	fixture humanAuthFixture,
	orgID uuid.UUID,
	userID uuid.UUID,
) *ent.OrganizationMembership {
	t.Helper()

	item, err := fixture.client.OrganizationMembership.Query().
		Where(
			entorganizationmembership.OrganizationIDEQ(orgID),
			entorganizationmembership.UserID(userID),
		).
		Only(context.Background())
	if err != nil {
		t.Fatalf("load organization membership: %v", err)
	}
	return item
}

func organizationMembershipByEmail(
	t *testing.T,
	fixture humanAuthFixture,
	orgID uuid.UUID,
	email string,
) *ent.OrganizationMembership {
	t.Helper()

	item, err := fixture.client.OrganizationMembership.Query().
		Where(
			entorganizationmembership.OrganizationIDEQ(orgID),
			entorganizationmembership.EmailEQ(email),
		).
		Only(context.Background())
	if err != nil {
		t.Fatalf("load organization membership by email: %v", err)
	}
	return item
}

func organizationMembershipIDByUser(
	t *testing.T,
	fixture humanAuthFixture,
	orgID uuid.UUID,
	userID uuid.UUID,
) uuid.UUID {
	t.Helper()
	return organizationMembershipByUser(t, fixture, orgID, userID).ID
}

func setOrganizationMembershipRole(
	t *testing.T,
	fixture humanAuthFixture,
	orgID uuid.UUID,
	userID uuid.UUID,
	role entorganizationmembership.Role,
) {
	t.Helper()

	membership := organizationMembershipByUser(t, fixture, orgID, userID)
	if _, err := fixture.client.OrganizationMembership.UpdateOneID(membership.ID).
		SetRole(role).
		Save(context.Background()); err != nil {
		t.Fatalf("update organization membership role: %v", err)
	}
}
