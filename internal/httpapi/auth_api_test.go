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
	entrolebinding "github.com/BetterAndBetterII/openase/ent/rolebinding"
	entuser "github.com/BetterAndBetterII/openase/ent/user"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	humanauthrepo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
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
		"agent.manage",
		"agent.read",
		"job.manage",
		"job.read",
		"org.read",
		"org.update",
		"project.delete",
		"project.read",
		"project.update",
		"proposal.approve",
		"rbac.manage",
		"repo.manage",
		"repo.read",
		"security.manage",
		"security.read",
		"skill.manage",
		"skill.read",
		"ticket.comment",
		"ticket.create",
		"ticket.read",
		"ticket.update",
		"workflow.manage",
		"workflow.read",
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
		"agent.manage",
		"agent.read",
		"job.manage",
		"job.read",
		"org.read",
		"org.update",
		"project.delete",
		"project.read",
		"project.update",
		"proposal.approve",
		"rbac.manage",
		"repo.manage",
		"repo.read",
		"security.manage",
		"security.read",
		"skill.manage",
		"skill.read",
		"ticket.comment",
		"ticket.create",
		"ticket.read",
		"ticket.update",
		"workflow.manage",
		"workflow.read",
	)
	if len(payload.Groups) != 1 || payload.Groups[0].GroupKey != "platform-admins" {
		t.Fatalf("unexpected groups payload: %+v", payload.Groups)
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
	org, err := f.client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := f.client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Atlas").
		SetSlug("atlas").
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

func humanFixtureHashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
