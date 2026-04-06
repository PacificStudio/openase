package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entauthauditevent "github.com/BetterAndBetterII/openase/ent/authauditevent"
	entbrowsersession "github.com/BetterAndBetterII/openase/ent/browsersession"
	entrolebinding "github.com/BetterAndBetterII/openase/ent/rolebinding"
	entuser "github.com/BetterAndBetterII/openase/ent/user"
	"github.com/BetterAndBetterII/openase/internal/config"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	humanauthrepo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestOIDCFlowCookiePathMatchesAPICallback(t *testing.T) {
	t.Parallel()

	server := &Server{}
	echoServer := echo.New()

	startReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/start", nil)
	startRec := httptest.NewRecorder()
	startCtx := echoServer.NewContext(startReq, startRec)
	server.setFlowCookie(startCtx, "flow-token")

	startCookies := startRec.Result().Cookies()
	if len(startCookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(startCookies))
	}
	if startCookies[0].Name != oidcFlowCookieName {
		t.Fatalf("cookie name = %q, want %q", startCookies[0].Name, oidcFlowCookieName)
	}
	if startCookies[0].Path != oidcFlowCookiePath {
		t.Fatalf("cookie path = %q, want %q", startCookies[0].Path, oidcFlowCookiePath)
	}

	clearReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback", nil)
	clearRec := httptest.NewRecorder()
	clearCtx := echoServer.NewContext(clearReq, clearRec)
	server.clearOIDCFlowCookie(clearCtx)

	clearCookies := clearRec.Result().Cookies()
	if len(clearCookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(clearCookies))
	}
	if clearCookies[0].Path != oidcFlowCookiePath {
		t.Fatalf("cleared cookie path = %q, want %q", clearCookies[0].Path, oidcFlowCookiePath)
	}
	if clearCookies[0].MaxAge != -1 {
		t.Fatalf("cleared cookie max age = %d, want -1", clearCookies[0].MaxAge)
	}
}

func TestAuthSessionReturnsAuthenticatedPrincipal(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "alice@example.com",
		displayName:     "Alice Control Plane",
		instanceRoleKey: "instance_admin",
	})

	rec := fixture.request(t, http.MethodGet, "/api/v1/auth/session", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "AuthSessionTest/1.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		AuthMode      string   `json:"auth_mode"`
		Authenticated bool     `json:"authenticated"`
		CSRFToken     string   `json:"csrf_token"`
		Roles         []string `json:"roles"`
		Permissions   []string `json:"permissions"`
		User          struct {
			PrimaryEmail string `json:"primary_email"`
			DisplayName  string `json:"display_name"`
		} `json:"user"`
	}
	decodeResponse(t, rec, &payload)

	if payload.AuthMode != "oidc" {
		t.Fatalf("auth_mode = %q, want oidc", payload.AuthMode)
	}
	if !payload.Authenticated {
		t.Fatal("expected authenticated=true")
	}
	if payload.CSRFToken != csrfToken {
		t.Fatalf("csrf_token = %q, want %q", payload.CSRFToken, csrfToken)
	}
	assertStringSet(t, payload.Roles, "instance_admin")
	assertStringSet(
		t,
		payload.Permissions,
		"agent.control",
		"agent.create",
		"agent.delete",
		"agent.read",
		"agent.update",
		"conversation.create",
		"conversation.delete",
		"conversation.read",
		"conversation.update",
		"harness.read",
		"harness.update",
		"machine.create",
		"machine.delete",
		"machine.read",
		"machine.update",
		"notification.create",
		"notification.delete",
		"notification.read",
		"notification.update",
		"org.create",
		"org.delete",
		"org.read",
		"org.update",
		"project.create",
		"project.delete",
		"project.read",
		"project.update",
		"project_update.create",
		"project_update.read",
		"project_update.update",
		"proposal.approve",
		"provider.create",
		"provider.delete",
		"provider.read",
		"provider.update",
		"rbac.manage",
		"repo.create",
		"repo.delete",
		"repo.read",
		"repo.update",
		"scheduled_job.create",
		"scheduled_job.delete",
		"scheduled_job.read",
		"scheduled_job.trigger",
		"scheduled_job.update",
		"security_setting.read",
		"security_setting.update",
		"skill.create",
		"skill.delete",
		"skill.read",
		"skill.update",
		"status.create",
		"status.delete",
		"status.read",
		"status.update",
		"ticket.create",
		"ticket.read",
		"ticket.update",
		"ticket_comment.create",
		"ticket_comment.read",
		"ticket_comment.update",
		"workflow.create",
		"workflow.delete",
		"workflow.read",
		"workflow.update",
	)
	if payload.User.PrimaryEmail != "alice@example.com" {
		t.Fatalf("primary_email = %q, want alice@example.com", payload.User.PrimaryEmail)
	}
	if payload.User.DisplayName != "Alice Control Plane" {
		t.Fatalf("display_name = %q, want Alice Control Plane", payload.User.DisplayName)
	}
}

func TestAuthPermissionsIncludeOrgInheritanceAndGroupUnion(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, projectID := fixture.createOrganizationProject(t)
	sessionToken, _ := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "reviewer@example.com",
		displayName:     "Reviewer",
		groupKey:        "platform-admins",
		groupName:       "Platform Admins",
		projectID:       projectID,
		projectRoleKey:  "project_viewer",
		orgID:           orgID,
		orgGroupRoleKey: "org_admin",
	})

	rec := fixture.request(t, http.MethodGet, "/api/v1/auth/me/permissions?project_id="+projectID.String(), map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "PermissionsTest/1.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Scope struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		} `json:"scope"`
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
		Groups      []struct {
			GroupKey string `json:"group_key"`
		} `json:"groups"`
	}
	decodeResponse(t, rec, &payload)

	if payload.Scope.Kind != "project" || payload.Scope.ID != projectID.String() {
		t.Fatalf("unexpected scope: %+v", payload.Scope)
	}
	assertStringSet(t, payload.Roles, "org_admin", "project_viewer")
	assertStringSet(t, payload.Permissions,
		"agent.control",
		"agent.create",
		"agent.delete",
		"agent.read",
		"agent.update",
		"conversation.create",
		"conversation.delete",
		"conversation.read",
		"conversation.update",
		"harness.read",
		"harness.update",
		"machine.create",
		"machine.delete",
		"machine.read",
		"machine.update",
		"notification.create",
		"notification.delete",
		"notification.read",
		"notification.update",
		"org.read",
		"org.update",
		"project.create",
		"project.delete",
		"project.read",
		"project.update",
		"project_update.create",
		"project_update.read",
		"project_update.update",
		"proposal.approve",
		"provider.create",
		"provider.delete",
		"provider.read",
		"provider.update",
		"rbac.manage",
		"repo.create",
		"repo.delete",
		"repo.read",
		"repo.update",
		"scheduled_job.create",
		"scheduled_job.delete",
		"scheduled_job.read",
		"scheduled_job.trigger",
		"scheduled_job.update",
		"security_setting.read",
		"security_setting.update",
		"skill.create",
		"skill.delete",
		"skill.read",
		"skill.update",
		"status.create",
		"status.delete",
		"status.read",
		"status.update",
		"ticket.create",
		"ticket.read",
		"ticket.update",
		"ticket_comment.create",
		"ticket_comment.read",
		"ticket_comment.update",
		"workflow.create",
		"workflow.delete",
		"workflow.read",
		"workflow.update",
	)
	if len(payload.Groups) != 1 || payload.Groups[0].GroupKey != "platform-admins" {
		t.Fatalf("unexpected groups payload: %+v", payload.Groups)
	}
}

func TestHumanVisibilityFiltersOrganizationAndProjectLists(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgA, projectA := fixture.createOrganizationProject(t)
	orgB, _ := fixture.createOrganizationProject(t)
	projectA2, err := fixture.client.Project.Create().
		SetOrganizationID(orgA).
		SetName("Atlas Extra").
		SetSlug("atlas-extra-" + uuid.NewString()[:8]).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create second project in org: %v", err)
	}
	sessionToken, _ := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:      "viewer@example.com",
		displayName:    "Visible Viewer",
		projectID:      projectA,
		projectRoleKey: "project_viewer",
	})

	orgRec := fixture.request(t, http.MethodGet, "/api/v1/orgs", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "VisibilityListTest/1.0",
	})
	if orgRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", orgRec.Code, orgRec.Body.String())
	}
	var orgPayload struct {
		Organizations []struct {
			ID string `json:"id"`
		} `json:"organizations"`
	}
	decodeResponse(t, orgRec, &orgPayload)
	if len(orgPayload.Organizations) != 1 || orgPayload.Organizations[0].ID != orgA.String() {
		t.Fatalf("unexpected organizations payload: %+v", orgPayload.Organizations)
	}

	projectRec := fixture.request(t, http.MethodGet, "/api/v1/orgs/"+orgA.String()+"/projects", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "VisibilityProjectListTest/1.0",
	})
	if projectRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", projectRec.Code, projectRec.Body.String())
	}
	var projectPayload struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	decodeResponse(t, projectRec, &projectPayload)
	if len(projectPayload.Projects) != 1 || projectPayload.Projects[0].ID != projectA.String() {
		t.Fatalf("unexpected projects payload: %+v", projectPayload.Projects)
	}

	hiddenOrgRec := fixture.request(t, http.MethodGet, "/api/v1/orgs/"+orgB.String()+"/projects", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "HiddenVisibilityProjectListTest/1.0",
	})
	if hiddenOrgRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for hidden org list filtering, got %d: %s", hiddenOrgRec.Code, hiddenOrgRec.Body.String())
	}
	var hiddenProjectPayload struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	decodeResponse(t, hiddenOrgRec, &hiddenProjectPayload)
	if len(hiddenProjectPayload.Projects) != 0 {
		t.Fatalf("expected hidden org project list to be empty, got %+v", hiddenProjectPayload.Projects)
	}

	if projectA2.ID == uuid.Nil {
		t.Fatal("second project id must be non-nil")
	}
}

func TestLogoutRequiresCSRFForAuthenticatedSession(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, _ := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "alice@example.com",
		displayName: "Alice Control Plane",
	})

	rec := fixture.request(t, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "LogoutTest/1.0",
	})
	assertAPIErrorResponse(t, rec, http.StatusForbidden, "CSRF_ORIGIN_FORBIDDEN", "origin or referer must match this host")
}

func TestLogoutRevokesBrowserSession(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "alice@example.com",
		displayName: "Alice Control Plane",
	})

	rec := fixture.request(t, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"Cookie":         humanSessionCookieName + "=" + sessionToken,
		"Origin":         "http://example.com",
		"X-OpenASE-CSRF": csrfToken,
		"User-Agent":     "LogoutTest/1.0",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}

	session, err := fixture.repo.GetBrowserSessionByHash(context.Background(), humanFixtureHashToken(sessionToken))
	if err != nil {
		t.Fatalf("load session after logout: %v", err)
	}
	if session.RevokedAt == nil {
		t.Fatal("expected revoked_at to be set after logout")
	}
}

func TestListSessionsReturnsInventoryAndAuditTrail(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, _ := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "alice@example.com",
		displayName:     "Alice Control Plane",
		deviceKind:      "desktop",
		deviceOS:        "Linux",
		deviceBrowser:   "Firefox",
		deviceLabel:     "Firefox on Linux",
		instanceRoleKey: "instance_admin",
	})
	userID := fixture.userIDByEmail(t, "alice@example.com")
	otherToken := fixture.createAdditionalSession(t, userID, "alice-laptop", "Chrome on macOS")
	otherSession, err := fixture.repo.GetBrowserSessionByHash(context.Background(), humanFixtureHashToken(otherToken))
	if err != nil {
		t.Fatalf("load additional session: %v", err)
	}
	if _, err := fixture.repo.CreateAuthAuditEvent(context.Background(), humanauthrepo.CreateAuthAuditEventInput{
		UserID:    &userID,
		SessionID: &otherSession.ID,
		ActorID:   "user:" + userID.String(),
		EventType: humanauthdomain.AuthAuditSessionRevoked,
		Message:   "Seeded audit event.",
		Metadata:  map[string]any{"reason": "seeded"},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create auth audit event: %v", err)
	}

	rec := fixture.request(t, http.MethodGet, "/api/v1/auth/sessions", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "InventoryTest/1.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		AuthMode         string `json:"auth_mode"`
		CurrentSessionID string `json:"current_session_id"`
		Sessions         []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
			Device  struct {
				Label string `json:"label"`
			} `json:"device"`
		} `json:"sessions"`
		AuditEvents []struct {
			EventType string `json:"event_type"`
		} `json:"audit_events"`
		StepUp struct {
			Status string `json:"status"`
		} `json:"step_up"`
	}
	decodeResponse(t, rec, &payload)

	if payload.AuthMode != "oidc" {
		t.Fatalf("auth_mode = %q, want oidc", payload.AuthMode)
	}
	if payload.StepUp.Status != "reserved" {
		t.Fatalf("step_up.status = %q, want reserved", payload.StepUp.Status)
	}
	if len(payload.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %+v", payload.Sessions)
	}
	currentCount := 0
	deviceLabels := map[string]bool{}
	for _, session := range payload.Sessions {
		if session.Current {
			currentCount++
			if payload.CurrentSessionID != session.ID {
				t.Fatalf("current session id mismatch: current_session_id=%q row=%q", payload.CurrentSessionID, session.ID)
			}
		}
		deviceLabels[session.Device.Label] = true
	}
	if currentCount != 1 {
		t.Fatalf("expected exactly 1 current session, got %d", currentCount)
	}
	if !deviceLabels["Firefox on Linux"] || !deviceLabels["Chrome on macOS"] {
		t.Fatalf("unexpected device labels: %+v", deviceLabels)
	}
	if len(payload.AuditEvents) != 1 || payload.AuditEvents[0].EventType != "session.revoked" {
		t.Fatalf("unexpected audit events: %+v", payload.AuditEvents)
	}
}

func TestDeleteSessionRevokesTargetSessionAndBlocksFutureRequests(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "alice@example.com",
		displayName:     "Alice Control Plane",
		instanceRoleKey: "instance_admin",
	})
	userID := fixture.userIDByEmail(t, "alice@example.com")
	otherToken := fixture.createAdditionalSession(t, userID, "alice-phone", "Safari on iPhone")
	otherSession, err := fixture.repo.GetBrowserSessionByHash(context.Background(), humanFixtureHashToken(otherToken))
	if err != nil {
		t.Fatalf("load additional session: %v", err)
	}

	rec := fixture.request(t, http.MethodDelete, "/api/v1/auth/sessions/"+otherSession.ID.String(), map[string]string{
		"Cookie":         humanSessionCookieName + "=" + sessionToken,
		"Origin":         "http://example.com",
		"X-OpenASE-CSRF": csrfToken,
		"User-Agent":     "RevokeSessionTest/1.0",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}

	revokedSession, err := fixture.repo.GetBrowserSessionByHash(context.Background(), humanFixtureHashToken(otherToken))
	if err != nil {
		t.Fatalf("reload revoked session: %v", err)
	}
	if revokedSession.RevokedAt == nil {
		t.Fatal("expected revoked session to have revoked_at set")
	}

	denied := fixture.request(t, http.MethodGet, "/api/v1/auth/me/permissions", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + otherToken,
		"User-Agent": "RevokeSessionTest/1.0",
	})
	assertAPIErrorResponse(t, denied, http.StatusUnauthorized, "HUMAN_SESSION_INVALID", "invalid browser session")
}

func TestRevokeAllSessionsKeepsCurrentSession(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "alice@example.com",
		displayName:     "Alice Control Plane",
		instanceRoleKey: "instance_admin",
	})
	userID := fixture.userIDByEmail(t, "alice@example.com")
	otherToken := fixture.createAdditionalSession(t, userID, "alice-tablet", "Chrome on Android")

	rec := fixture.request(t, http.MethodPost, "/api/v1/auth/sessions/revoke-all", map[string]string{
		"Cookie":         humanSessionCookieName + "=" + sessionToken,
		"Origin":         "http://example.com",
		"X-OpenASE-CSRF": csrfToken,
		"User-Agent":     "RevokeAllTest/1.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		RevokedCount int `json:"revoked_count"`
	}
	decodeResponse(t, rec, &payload)
	if payload.RevokedCount != 1 {
		t.Fatalf("revoked_count = %d, want 1", payload.RevokedCount)
	}

	currentSession, err := fixture.repo.GetBrowserSessionByHash(context.Background(), humanFixtureHashToken(sessionToken))
	if err != nil {
		t.Fatalf("reload current session: %v", err)
	}
	if currentSession.RevokedAt != nil {
		t.Fatal("expected current session to remain active")
	}
	otherSession, err := fixture.repo.GetBrowserSessionByHash(context.Background(), humanFixtureHashToken(otherToken))
	if err != nil {
		t.Fatalf("reload other session: %v", err)
	}
	if otherSession.RevokedAt == nil {
		t.Fatal("expected other session to be revoked")
	}
}

func TestAdminCanForceRevokeUserSessions(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	adminToken, adminCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "admin@example.com",
		displayName:     "Admin",
		instanceRoleKey: "instance_admin",
	})
	fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "member@example.com",
		displayName: "Member",
	})
	memberUserID := fixture.userIDByEmail(t, "member@example.com")
	memberToken := fixture.createAdditionalSession(t, memberUserID, "member-laptop", "Chrome on Windows")

	rec := fixture.request(t, http.MethodPost, "/api/v1/auth/users/"+memberUserID.String()+"/sessions/revoke", map[string]string{
		"Cookie":         humanSessionCookieName + "=" + adminToken,
		"Origin":         "http://example.com",
		"X-OpenASE-CSRF": adminCSRF,
		"User-Agent":     "AdminRevokeTest/1.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		RevokedCount int    `json:"revoked_count"`
		UserID       string `json:"user_id"`
	}
	decodeResponse(t, rec, &payload)
	if payload.RevokedCount != 2 {
		t.Fatalf("revoked_count = %d, want 2", payload.RevokedCount)
	}
	if payload.UserID != memberUserID.String() {
		t.Fatalf("user_id = %q, want %q", payload.UserID, memberUserID.String())
	}

	memberSession, err := fixture.repo.GetBrowserSessionByHash(context.Background(), humanFixtureHashToken(memberToken))
	if err != nil {
		t.Fatalf("reload member session: %v", err)
	}
	if memberSession.RevokedAt == nil {
		t.Fatal("expected member session to be revoked")
	}
}

func TestListSessionsReturnsLightweightDisabledModeResponse(t *testing.T) {
	t.Parallel()

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/auth/sessions", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		AuthMode    string `json:"auth_mode"`
		Sessions    []any  `json:"sessions"`
		AuditEvents []any  `json:"audit_events"`
		StepUp      struct {
			Status string `json:"status"`
		} `json:"step_up"`
	}
	decodeResponse(t, rec, &payload)
	if payload.AuthMode != "disabled" {
		t.Fatalf("auth_mode = %q, want disabled", payload.AuthMode)
	}
	if len(payload.Sessions) != 0 || len(payload.AuditEvents) != 0 {
		t.Fatalf("expected no session governance payload in disabled mode, got %+v", payload)
	}
	if payload.StepUp.Status != "reserved" {
		t.Fatalf("step_up.status = %q, want reserved", payload.StepUp.Status)
	}
}

func TestExpiredSessionRecordsAuditEvent(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, _ := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "alice@example.com",
		displayName: "Alice Control Plane",
	})
	session, err := fixture.client.BrowserSession.Query().
		Where(entbrowsersession.SessionHashEQ(humanFixtureHashToken(sessionToken))).
		Only(context.Background())
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := fixture.client.BrowserSession.UpdateOneID(session.ID).
		SetExpiresAt(time.Now().UTC().Add(-1 * time.Minute)).
		Save(context.Background()); err != nil {
		t.Fatalf("expire session: %v", err)
	}

	rec := fixture.request(t, http.MethodGet, "/api/v1/auth/session", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "ExpiredSessionTest/1.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	events, err := fixture.client.AuthAuditEvent.Query().
		Where(entauthauditevent.EventTypeEQ(string(humanauthdomain.AuthAuditSessionExpired))).
		All(context.Background())
	if err != nil {
		t.Fatalf("query auth audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 session.expired event, got %+v", events)
	}
}

func TestOIDCCallbackFailureRecordsAuditEvent(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	rec := fixture.request(t, http.MethodGet, "/api/v1/auth/oidc/callback?code=bad&state=bad", map[string]string{
		"Cookie": "openase_oidc_flow=invalid",
	})
	assertAPIErrorResponse(t, rec, http.StatusUnauthorized, "OIDC_CALLBACK_FAILED", "invalid oidc login flow state")

	events, err := fixture.client.AuthAuditEvent.Query().
		Where(entauthauditevent.EventTypeEQ(string(humanauthdomain.AuthAuditLoginFailed))).
		All(context.Background())
	if err != nil {
		t.Fatalf("query auth audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 login.failed event, got %+v", events)
	}
}

func TestDisabledUserAfterLoginRecordsAuditEvent(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, _ := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "alice@example.com",
		displayName: "Alice Control Plane",
	})
	userID := fixture.userIDByEmail(t, "alice@example.com")
	if _, err := fixture.client.User.UpdateOneID(userID).
		SetStatus(entuser.StatusDisabled).
		Save(context.Background()); err != nil {
		t.Fatalf("disable user: %v", err)
	}

	rec := fixture.request(t, http.MethodGet, "/api/v1/auth/me/permissions", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "DisabledSessionTest/1.0",
	})
	assertAPIErrorResponse(t, rec, http.StatusUnauthorized, "HUMAN_USER_DISABLED", "user is disabled")

	events, err := fixture.client.AuthAuditEvent.Query().
		Where(entauthauditevent.EventTypeEQ(string(humanauthdomain.AuthAuditUserDisabledAfterLogin))).
		All(context.Background())
	if err != nil {
		t.Fatalf("query auth audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 user.disabled_after_login event, got %+v", events)
	}
}

func TestCreateProjectRoleBindingRejectsInvalidScopeRoleCombination(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, projectID := fixture.createOrganizationProject(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "admin@example.com",
		displayName:     "Admin",
		instanceRoleKey: "instance_admin",
	})
	targetUserID := fixture.createUser(t, "target@example.com", "Target User")

	rec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/role-bindings",
		`{"subject_kind":"user","subject_key":"target@example.com","role_key":"instance_admin"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + sessionToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": csrfToken,
			"User-Agent":     "ProjectRoleBindingTest/1.0",
		},
	)
	assertAPIErrorResponse(t, rec, http.StatusBadRequest, "ROLE_BINDING_CREATE_FAILED", `unsupported project role "instance_admin"`)

	items, err := fixture.repo.ListRoleBindings(context.Background(), humanauthrepo.ListRoleBindingsFilter{
		ScopeKind: scopeKindPointer(humanauthdomain.ScopeKindProject),
		ScopeID:   stringPtr(projectID.String()),
	})
	if err != nil {
		t.Fatalf("ListRoleBindings() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no project bindings after rejected create, got %+v", items)
	}
	if targetUserID == uuid.Nil || orgID == uuid.Nil {
		t.Fatal("fixture ids must be non-nil")
	}
}

func TestDeleteOrganizationRoleBindingStaysWithinScope(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgA, _ := fixture.createOrganizationProject(t)
	orgB, _ := fixture.createOrganizationProject(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "admin@example.com",
		displayName:     "Admin",
		instanceRoleKey: "instance_admin",
	})
	targetUserID := fixture.createUser(t, "scope-test@example.com", "Scope Test")
	ctx := context.Background()

	binding, err := fixture.client.RoleBinding.Create().
		SetScopeKind(entrolebinding.ScopeKindOrganization).
		SetScopeID(orgB.String()).
		SetSubjectKind(entrolebinding.SubjectKindUser).
		SetSubjectKey(targetUserID.String()).
		SetRoleKey(string(humanauthdomain.RoleOrgAdmin)).
		SetGrantedBy("system:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create role binding: %v", err)
	}

	rec := fixture.requestJSON(
		t,
		http.MethodDelete,
		"/api/v1/organizations/"+orgA.String()+"/role-bindings/"+binding.ID.String(),
		"",
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + sessionToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": csrfToken,
			"User-Agent":     "DeleteScopeTest/1.0",
		},
	)
	assertAPIErrorResponse(t, rec, http.StatusNotFound, "ROLE_BINDING_NOT_FOUND", "role binding not found")

	if _, err := fixture.client.RoleBinding.Get(ctx, binding.ID); err != nil {
		t.Fatalf("binding should still exist after scoped delete rejection: %v", err)
	}
}

func TestInstanceRoleBindingRoutesCanonicalizeDirectUserSubject(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "admin@example.com",
		displayName:     "Admin",
		instanceRoleKey: "instance_admin",
	})
	targetUserID := fixture.createUser(t, "canonical@example.com", "Canonical User")

	createRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/instance/role-bindings",
		`{"subject_kind":"user","subject_key":"canonical@example.com","role_key":"instance_admin"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + sessionToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": csrfToken,
			"User-Agent":     "InstanceRoleBindingTest/1.0",
		},
	)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var createPayload struct {
		RoleBinding struct {
			ID         string `json:"id"`
			ScopeKind  string `json:"scope_kind"`
			SubjectKey string `json:"subject_key"`
			RoleKey    string `json:"role_key"`
		} `json:"role_binding"`
	}
	decodeResponse(t, createRec, &createPayload)
	if createPayload.RoleBinding.ScopeKind != "instance" {
		t.Fatalf("scope_kind = %q, want instance", createPayload.RoleBinding.ScopeKind)
	}
	if createPayload.RoleBinding.SubjectKey != targetUserID.String() {
		t.Fatalf("subject_key = %q, want canonical user id %q", createPayload.RoleBinding.SubjectKey, targetUserID.String())
	}
	if createPayload.RoleBinding.RoleKey != "instance_admin" {
		t.Fatalf("role_key = %q, want instance_admin", createPayload.RoleBinding.RoleKey)
	}

	listRec := fixture.request(t, http.MethodGet, "/api/v1/instance/role-bindings", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "InstanceRoleBindingTest/1.0",
	})
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listPayload struct {
		RoleBindings []struct {
			ID         string `json:"id"`
			SubjectKey string `json:"subject_key"`
		} `json:"role_bindings"`
	}
	decodeResponse(t, listRec, &listPayload)
	if len(listPayload.RoleBindings) != 2 {
		t.Fatalf("expected bootstrap admin + new binding, got %+v", listPayload.RoleBindings)
	}

	deleteRec := fixture.requestJSON(
		t,
		http.MethodDelete,
		"/api/v1/instance/role-bindings/"+createPayload.RoleBinding.ID,
		"",
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + sessionToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": csrfToken,
			"User-Agent":     "InstanceRoleBindingTest/1.0",
		},
	)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

type humanAuthFixture struct {
	client *ent.Client
	repo   *humanauthrepo.Repository
	server *Server
}

type humanFixtureSessionInput struct {
	userEmail       string
	displayName     string
	groupKey        string
	groupName       string
	deviceKind      string
	deviceOS        string
	deviceBrowser   string
	deviceLabel     string
	instanceRoleKey string
	orgID           uuid.UUID
	projectID       uuid.UUID
	projectRoleKey  string
	orgGroupRoleKey string
}

func newHumanAuthFixture(t *testing.T) humanAuthFixture {
	t.Helper()

	client := openTestEntClient(t)
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})

	cfg := config.AuthConfig{
		Mode: config.AuthModeOIDC,
		OIDC: config.OIDCConfig{
			ClientSecret:   "test-client-secret",
			SessionTTL:     8 * time.Hour,
			SessionIdleTTL: 30 * time.Minute,
		},
	}
	repository := humanauthrepo.NewEntRepository(client)
	service := humanauthservice.NewService(cfg, repository, nil)
	authorizer := humanauthservice.NewAuthorizer(repository)
	catalogSvc := catalogservice.New(
		catalogrepo.NewEntRepository(client),
		nil,
		nil,
		catalogservice.WithHumanVisibilityResolver(humanauthservice.NewVisibilityResolver(repository)),
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalogSvc,
		nil,
		WithHumanAuthConfig(cfg),
		WithHumanAuthService(service, authorizer),
	)

	return humanAuthFixture{
		client: client,
		repo:   repository,
		server: server,
	}
}

func (f humanAuthFixture) createOrganizationProject(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()

	ctx := context.Background()
	suffix := uuid.NewString()[:8]
	org, err := f.client.Organization.Create().
		SetName("Acme " + suffix).
		SetSlug("acme-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := f.client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Atlas " + suffix).
		SetSlug("atlas-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return org.ID, project.ID
}

func (f humanAuthFixture) createSession(t *testing.T, input humanFixtureSessionInput) (string, string) {
	t.Helper()

	ctx := context.Background()
	now := time.Now().UTC()
	user, err := f.client.User.Create().
		SetStatus(entuser.StatusActive).
		SetPrimaryEmail(input.userEmail).
		SetDisplayName(input.displayName).
		SetLastLoginAt(now).
		Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := f.client.UserIdentity.Create().
		SetUserID(user.ID).
		SetIssuer("https://idp.example.com").
		SetSubject("subject-" + user.ID.String()).
		SetEmail(input.userEmail).
		SetEmailVerified(true).
		SetClaimsVersion(1).
		SetRawClaimsJSON(`{"sub":"` + user.ID.String() + `"}`).
		SetLastSyncedAt(now).
		Save(ctx); err != nil {
		t.Fatalf("create user identity: %v", err)
	}
	if input.groupKey != "" {
		if _, err := f.client.UserGroupMembership.Create().
			SetUserID(user.ID).
			SetIssuer("https://idp.example.com").
			SetGroupKey(input.groupKey).
			SetGroupName(input.groupName).
			SetLastSyncedAt(now).
			Save(ctx); err != nil {
			t.Fatalf("create user group membership: %v", err)
		}
	}
	if input.instanceRoleKey != "" {
		if _, err := f.client.RoleBinding.Create().
			SetScopeKind(entrolebinding.ScopeKindInstance).
			SetScopeID("").
			SetSubjectKind(entrolebinding.SubjectKindUser).
			SetSubjectKey(user.ID.String()).
			SetRoleKey(input.instanceRoleKey).
			SetGrantedBy("system:test").
			Save(ctx); err != nil {
			t.Fatalf("create instance role binding: %v", err)
		}
	}
	if input.projectID != uuid.Nil && input.projectRoleKey != "" {
		if _, err := f.client.RoleBinding.Create().
			SetScopeKind(entrolebinding.ScopeKindProject).
			SetScopeID(input.projectID.String()).
			SetSubjectKind(entrolebinding.SubjectKindUser).
			SetSubjectKey(user.ID.String()).
			SetRoleKey(input.projectRoleKey).
			SetGrantedBy("system:test").
			Save(ctx); err != nil {
			t.Fatalf("create project role binding: %v", err)
		}
	}
	if input.orgID != uuid.Nil && input.orgGroupRoleKey != "" {
		if _, err := f.client.RoleBinding.Create().
			SetScopeKind(entrolebinding.ScopeKindOrganization).
			SetScopeID(input.orgID.String()).
			SetSubjectKind(entrolebinding.SubjectKindGroup).
			SetSubjectKey(input.groupKey).
			SetRoleKey(input.orgGroupRoleKey).
			SetGrantedBy("system:test").
			Save(ctx); err != nil {
			t.Fatalf("create org group role binding: %v", err)
		}
	}

	sessionToken := "session-" + user.ID.String()
	csrfToken := "csrf-" + user.ID.String()
	if _, err := f.repo.CreateBrowserSession(ctx, humanauthrepo.CreateBrowserSessionInput{
		UserID:        user.ID,
		SessionHash:   humanFixtureHashToken(sessionToken),
		DeviceKind:    humanauthdomain.SessionDeviceKind(input.deviceKind),
		DeviceOS:      input.deviceOS,
		DeviceBrowser: input.deviceBrowser,
		DeviceLabel:   input.deviceLabel,
		ExpiresAt:     now.Add(2 * time.Hour),
		IdleExpiresAt: now.Add(30 * time.Minute),
		CSRFSecret:    csrfToken,
	}); err != nil {
		t.Fatalf("create browser session: %v", err)
	}
	return sessionToken, csrfToken
}

func (f humanAuthFixture) request(
	t *testing.T,
	method string,
	target string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	if headers == nil {
		headers = map[string]string{}
	}
	headers["Host"] = "example.com"
	return performJSONRequestWithHeaders(t, f.server, method, target, "", headers)
}

func (f humanAuthFixture) userIDByEmail(t *testing.T, email string) uuid.UUID {
	t.Helper()

	item, err := f.client.User.Query().
		Where(entuser.PrimaryEmailEQ(email)).
		Only(context.Background())
	if err != nil {
		t.Fatalf("lookup user by email: %v", err)
	}
	return item.ID
}

func (f humanAuthFixture) createAdditionalSession(
	t *testing.T,
	userID uuid.UUID,
	tokenSuffix string,
	deviceLabel string,
) string {
	t.Helper()

	now := time.Now().UTC()
	sessionToken := "session-" + tokenSuffix
	if _, err := f.repo.CreateBrowserSession(context.Background(), humanauthrepo.CreateBrowserSessionInput{
		UserID:        userID,
		SessionHash:   humanFixtureHashToken(sessionToken),
		DeviceKind:    humanauthdomain.SessionDeviceKindDesktop,
		DeviceOS:      "macOS",
		DeviceBrowser: "Chrome",
		DeviceLabel:   deviceLabel,
		ExpiresAt:     now.Add(2 * time.Hour),
		IdleExpiresAt: now.Add(30 * time.Minute),
		CSRFSecret:    "csrf-" + tokenSuffix,
	}); err != nil {
		t.Fatalf("create additional browser session: %v", err)
	}
	return sessionToken
}

func (f humanAuthFixture) requestJSON(
	t *testing.T,
	method string,
	target string,
	body string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	if headers == nil {
		headers = map[string]string{}
	}
	headers["Host"] = "example.com"
	return performJSONRequestWithHeaders(t, f.server, method, target, body, headers)
}

func (f humanAuthFixture) createUser(t *testing.T, email string, displayName string) uuid.UUID {
	t.Helper()

	now := time.Now().UTC()
	user, err := f.client.User.Create().
		SetStatus(entuser.StatusActive).
		SetPrimaryEmail(email).
		SetDisplayName(displayName).
		SetLastLoginAt(now).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := f.client.UserIdentity.Create().
		SetUserID(user.ID).
		SetIssuer("https://idp.example.com").
		SetSubject("subject-" + user.ID.String()).
		SetEmail(email).
		SetEmailVerified(true).
		SetClaimsVersion(1).
		SetRawClaimsJSON(`{"sub":"` + user.ID.String() + `"}`).
		SetLastSyncedAt(now).
		Save(context.Background()); err != nil {
		t.Fatalf("create user identity: %v", err)
	}
	return user.ID
}

func scopeKindPointer(value humanauthdomain.ScopeKind) *humanauthdomain.ScopeKind {
	return &value
}

func humanFixtureHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
