package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	humanauthrepo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
)

func TestAdminAuthRouteReturnsDBBackedAbsentState(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	_, instanceAuthSvc := newInstanceAuthTestService(t, config.AuthConfig{Mode: config.AuthModeDisabled}, configPath)
	server := NewServer(
		config.ServerConfig{Port: 40023, Host: "127.0.0.1"},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithRuntimeConfigFile(configPath),
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeDisabled}),
		WithInstanceAuthService(instanceAuthSvc),
	)
	if _, err := instanceAuthSvc.Disable(context.Background()); err != nil {
		t.Fatalf("seed absent db-backed auth state: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/auth", http.NoBody)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload adminAuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Auth.ActiveMode != "disabled" {
		t.Fatalf("active_mode = %q, want disabled", payload.Auth.ActiveMode)
	}
	if payload.Auth.ConfiguredMode != "disabled" {
		t.Fatalf("configured_mode = %q, want disabled", payload.Auth.ConfiguredMode)
	}
	if payload.Auth.ConfigPath != "db:instance_auth_configs" {
		t.Fatalf("config_path = %q, want db-backed storage label", payload.Auth.ConfigPath)
	}
	if payload.Auth.LastValidation.Status != "not_tested" {
		t.Fatalf("last_validation.status = %q, want not_tested", payload.Auth.LastValidation.Status)
	}
	if payload.Auth.SessionPolicy.SessionTTL == "" || payload.Auth.SessionPolicy.SessionIdleTTL == "" {
		t.Fatalf("expected session policy to be populated, got %+v", payload.Auth.SessionPolicy)
	}
}

func TestAdminAuthRoutePersistsDraftActivatesAndDisablesOIDC(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	issuerServer := newTestOIDCDiscoveryServer(t)
	defer issuerServer.Close()
	client, instanceAuthSvc := newInstanceAuthTestService(t, config.AuthConfig{Mode: config.AuthModeDisabled}, configPath)
	humanRepo := humanauthrepo.NewEntRepository(client)
	humanAuthSvc := humanauthservice.NewService(humanRepo, nil, instanceAuthSvc)
	humanAuthorizer := humanauthservice.NewAuthorizer(humanRepo)
	server := NewServer(
		config.ServerConfig{Port: 40023, Host: "0.0.0.0"},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithRuntimeConfigFile(configPath),
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeDisabled}),
		WithInstanceAuthService(instanceAuthSvc),
		WithHumanAuthService(humanAuthSvc, humanAuthorizer),
	)
	fixture := humanAuthFixture{client: client, repo: humanRepo, server: server}

	draftReq := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/auth/oidc-draft",
		strings.NewReader(`{"issuer_url":"`+issuerServer.URL+`","client_id":"openase","client_secret":"secret","redirect_mode":"fixed","fixed_redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"]}`),
	)
	draftReq.Header.Set("Content-Type", "application/json")
	draftRec := httptest.NewRecorder()

	server.Handler().ServeHTTP(draftRec, draftReq)

	if draftRec.Code != http.StatusOK {
		t.Fatalf("expected draft save 200, got %d: %s", draftRec.Code, draftRec.Body.String())
	}

	var draftPayload adminAuthResponse
	if err := json.Unmarshal(draftRec.Body.Bytes(), &draftPayload); err != nil {
		t.Fatalf("unmarshal draft response: %v", err)
	}
	if draftPayload.Auth.ActiveMode != "disabled" {
		t.Fatalf("draft active_mode = %q, want disabled", draftPayload.Auth.ActiveMode)
	}
	if draftPayload.Auth.ConfiguredMode != "disabled" {
		t.Fatalf("draft configured_mode = %q, want disabled", draftPayload.Auth.ConfiguredMode)
	}
	if draftPayload.Auth.ConfigPath != "db:instance_auth_configs" {
		t.Fatalf("draft config_path = %q, want db-backed storage label", draftPayload.Auth.ConfigPath)
	}
	if draftPayload.Auth.LastValidation.Status != "not_tested" {
		t.Fatalf("draft last_validation.status = %q, want not_tested", draftPayload.Auth.LastValidation.Status)
	}
	if !draftPayload.Auth.OIDCDraft.ClientSecretConfigured {
		t.Fatal("draft should report stored client_secret_configured = true")
	}
	if draftPayload.Auth.OIDCDraft.RedirectMode != "fixed" {
		t.Fatalf("draft redirect_mode = %q, want fixed", draftPayload.Auth.OIDCDraft.RedirectMode)
	}
	if draftPayload.Auth.OIDCDraft.FixedRedirectURL != "http://127.0.0.1:19836/api/v1/auth/oidc/callback" {
		t.Fatalf("draft fixed_redirect_url = %q", draftPayload.Auth.OIDCDraft.FixedRedirectURL)
	}
	storedDraft, err := client.InstanceAuthConfig.Query().Only(draftReq.Context())
	if err != nil {
		t.Fatalf("load draft instance auth config: %v", err)
	}
	if storedDraft.Status != "draft" {
		t.Fatalf("stored draft status = %q, want draft", storedDraft.Status)
	}

	enableReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/auth/oidc-enable",
		strings.NewReader(`{"issuer_url":"`+issuerServer.URL+`","client_id":"openase","client_secret":"secret","redirect_mode":"fixed","fixed_redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"]}`),
	)
	enableReq.Header.Set("Content-Type", "application/json")
	enableRec := httptest.NewRecorder()

	server.Handler().ServeHTTP(enableRec, enableReq)

	if enableRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", enableRec.Code, enableRec.Body.String())
	}

	var enablePayload adminAuthModeTransitionResponse
	if err := json.Unmarshal(enableRec.Body.Bytes(), &enablePayload); err != nil {
		t.Fatalf("unmarshal enable response: %v", err)
	}
	if enablePayload.Transition.Status != "configured" {
		t.Fatalf("transition.status = %q, want configured", enablePayload.Transition.Status)
	}
	if enablePayload.Transition.RestartRequired {
		t.Fatalf("restart_required = true, want false")
	}
	if enablePayload.Auth.ActiveMode != "oidc" {
		t.Fatalf("active_mode = %q, want oidc", enablePayload.Auth.ActiveMode)
	}
	if enablePayload.Auth.ConfiguredMode != "oidc" {
		t.Fatalf("configured_mode = %q, want oidc", enablePayload.Auth.ConfiguredMode)
	}
	if enablePayload.Auth.ConfigPath != "db:instance_auth_configs" {
		t.Fatalf("enable config_path = %q, want db-backed storage label", enablePayload.Auth.ConfigPath)
	}
	if enablePayload.Auth.LastValidation.Status != "ok" {
		t.Fatalf("last_validation.status = %q, want ok", enablePayload.Auth.LastValidation.Status)
	}
	if enablePayload.Auth.LastValidation.RedirectURL != "http://127.0.0.1:19836/api/v1/auth/oidc/callback" {
		t.Fatalf("last_validation.redirect_url = %q", enablePayload.Auth.LastValidation.RedirectURL)
	}
	storedActive, err := client.InstanceAuthConfig.Query().Only(enableReq.Context())
	if err != nil {
		t.Fatalf("load instance auth config: %v", err)
	}
	if storedActive.Status != "active" {
		t.Fatalf("stored active status = %q, want active", storedActive.Status)
	}
	if storedActive.ClientSecretEncrypted == nil {
		t.Fatal("expected encrypted client secret to be stored")
	}
	if strings.Contains(storedActive.ClientSecretEncrypted.Ciphertext, "secret") {
		t.Fatalf("expected encrypted client secret, got %+v", storedActive.ClientSecretEncrypted)
	}

	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:       "instance-admin@example.com",
		displayName:     "Instance Admin",
		instanceRoleKey: "instance_admin",
	})
	disableRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/admin/auth/disable",
		"",
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + sessionToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": csrfToken,
			"User-Agent":     "AdminAuthRouteTest/1.0",
		},
	)

	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected disable 200, got %d: %s", disableRec.Code, disableRec.Body.String())
	}

	var disablePayload adminAuthModeTransitionResponse
	if err := json.Unmarshal(disableRec.Body.Bytes(), &disablePayload); err != nil {
		t.Fatalf("unmarshal disable response: %v", err)
	}
	if disablePayload.Transition.Status != "disabled" {
		t.Fatalf("disable transition.status = %q, want disabled", disablePayload.Transition.Status)
	}
	if disablePayload.Auth.ActiveMode != "disabled" {
		t.Fatalf("disable active_mode = %q, want disabled", disablePayload.Auth.ActiveMode)
	}
	if disablePayload.Auth.ConfiguredMode != "disabled" {
		t.Fatalf("disable configured_mode = %q, want disabled", disablePayload.Auth.ConfiguredMode)
	}
	if disablePayload.Auth.OIDCDraft.IssuerURL != issuerServer.URL {
		t.Fatalf("disable saved draft issuer = %q, want %q", disablePayload.Auth.OIDCDraft.IssuerURL, issuerServer.URL)
	}
	if !disablePayload.Auth.OIDCDraft.ClientSecretConfigured {
		t.Fatal("disable should preserve the saved draft secret marker")
	}
	storedDisabled, err := client.InstanceAuthConfig.Query().Only(context.Background())
	if err != nil {
		t.Fatalf("load disabled instance auth config: %v", err)
	}
	if storedDisabled.Status != "draft" {
		t.Fatalf("stored disabled status = %q, want draft", storedDisabled.Status)
	}
}

func TestAdminAuthRouteRequiresInstanceAdminInOIDCMode(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	orgID, projectID := fixture.createOrganizationProject(t)
	sessionToken, csrfToken := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:      "project-admin@example.com",
		displayName:    "Project Admin",
		orgID:          orgID,
		projectID:      projectID,
		projectRoleKey: "project_admin",
	})

	adminRec := fixture.request(t, http.MethodGet, "/api/v1/admin/auth", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + sessionToken,
		"User-Agent": "AdminAuthPermissionTest/1.0",
	})
	assertAPIErrorResponse(t, adminRec, http.StatusForbidden, "AUTHORIZATION_DENIED", "required permission is missing")

	legacyRec := fixture.requestJSON(
		t,
		http.MethodPut,
		"/api/v1/projects/"+projectID.String()+"/security-settings/oidc-draft",
		`{"issuer_url":"https://idp.example.com","client_id":"openase","client_secret":"secret","redirect_mode":"fixed","fixed_redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"]}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + sessionToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": csrfToken,
			"User-Agent":     "AdminAuthPermissionTest/1.0",
		},
	)
	assertAPIErrorResponse(t, legacyRec, http.StatusForbidden, "AUTHORIZATION_DENIED", "required permission is missing")
}
