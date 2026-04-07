package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
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
	if enablePayload.Auth.ConfiguredMode != "oidc" {
		t.Fatalf("configured_mode = %q, want oidc", enablePayload.Auth.ConfiguredMode)
	}
	if enablePayload.Auth.LastValidation.Status != "ok" {
		t.Fatalf("last_validation.status = %q, want ok", enablePayload.Auth.LastValidation.Status)
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/disable", http.NoBody)
	disableRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(disableRec, disableReq)

	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", disableRec.Code, disableRec.Body.String())
	}

	var disablePayload adminAuthModeTransitionResponse
	if err := json.Unmarshal(disableRec.Body.Bytes(), &disablePayload); err != nil {
		t.Fatalf("unmarshal disable response: %v", err)
	}
	if disablePayload.Transition.Status != "disabled" {
		t.Fatalf("transition.status = %q, want disabled", disablePayload.Transition.Status)
	}
	if disablePayload.Auth.ConfiguredMode != "disabled" {
		t.Fatalf("configured_mode = %q, want disabled", disablePayload.Auth.ConfiguredMode)
	}

	// #nosec G304 -- configPath is created inside this test's TempDir.
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(written), "mode: disabled") {
		t.Fatalf("expected disabled mode in config, got %s", written)
	}
	if !strings.Contains(string(written), "last_validation:") {
		t.Fatalf("expected last_validation to be persisted, got %s", written)
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
