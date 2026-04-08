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
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
)

func TestAdminAuthRouteReturnsDisabledModeState(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
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
	)

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
	if payload.Auth.LastValidation.Status != "not_tested" {
		t.Fatalf("last_validation.status = %q, want not_tested", payload.Auth.LastValidation.Status)
	}
	if payload.Auth.SessionPolicy.SessionTTL == "" || payload.Auth.SessionPolicy.SessionIdleTTL == "" {
		t.Fatalf("expected session policy to be populated, got %+v", payload.Auth.SessionPolicy)
	}
}

func TestAdminAuthRouteEnablesAndDisablesOIDC(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	issuerServer := newTestOIDCDiscoveryServer(t)
	defer issuerServer.Close()
	client, instanceAuthSvc := newInstanceAuthTestService(t, config.AuthConfig{Mode: config.AuthModeDisabled}, configPath)
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
	)

	enableReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/auth/oidc-enable",
		strings.NewReader(`{"issuer_url":"`+issuerServer.URL+`","client_id":"openase","client_secret":"secret","redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"]}`),
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
	if enablePayload.Auth.LastValidation.Status != "ok" {
		t.Fatalf("last_validation.status = %q, want ok", enablePayload.Auth.LastValidation.Status)
	}

	storedDisabled, err := instanceAuthSvc.Disable(context.Background())
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	disableAuth := buildSecurityAuthSettingsResponseFromAccessControl(iam.ResolveRuntimeAccessControlState(storedDisabled.State), storedDisabled.State, storedDisabled.StorageLocation, "0.0.0.0")
	if disableAuth.ActiveMode != "disabled" {
		t.Fatalf("active_mode = %q, want disabled", disableAuth.ActiveMode)
	}
	if disableAuth.ConfiguredMode != "disabled" {
		t.Fatalf("configured_mode = %q, want disabled", disableAuth.ConfiguredMode)
	}

	stored, err := client.InstanceAuthConfig.Query().Only(enableReq.Context())
	if err != nil {
		t.Fatalf("load instance auth config: %v", err)
	}
	if stored.Status != "draft" {
		t.Fatalf("stored status = %q, want draft", stored.Status)
	}
	if stored.ClientSecretEncrypted == nil {
		t.Fatal("expected encrypted client secret to be stored")
	}
	if strings.Contains(stored.ClientSecretEncrypted.Ciphertext, "secret") {
		t.Fatalf("expected encrypted client secret, got %+v", stored.ClientSecretEncrypted)
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
		`{"issuer_url":"https://idp.example.com","client_id":"openase","client_secret":"secret","redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"]}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + sessionToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": csrfToken,
			"User-Agent":     "AdminAuthPermissionTest/1.0",
		},
	)
	assertAPIErrorResponse(t, legacyRec, http.StatusForbidden, "AUTHORIZATION_DENIED", "required permission is missing")
}
