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
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
)

func TestSecuritySettingsRouteReturnsCurrentBoundary(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{WebhookSecret: "top-secret"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubAuthService(&stubGitHubAuthService{
			security: sampleProjectSecurity(),
		}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/security-settings",
		http.NoBody,
	)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Security securitySettingsResponse `json:"security"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if payload.Security.ProjectID != projectID.String() {
		t.Fatalf("expected project id %q, got %q", projectID, payload.Security.ProjectID)
	}
	if payload.Security.GitHub.Effective.Scope != "organization" || payload.Security.GitHub.Effective.Source != "gh_cli_import" {
		t.Fatalf("expected effective GitHub credential metadata, got %+v", payload.Security.GitHub.Effective)
	}
	if payload.Security.GitHub.Effective.Probe.Login != "octocat" {
		t.Fatalf("expected effective GitHub login octocat, got %+v", payload.Security.GitHub.Effective.Probe)
	}
	if !payload.Security.GitHub.Organization.Configured || payload.Security.GitHub.ProjectOverride.Configured {
		t.Fatalf("expected scoped GitHub slots, got %+v", payload.Security.GitHub)
	}
	if payload.Security.Webhooks.ConnectorEndpoint != "Not supported in current version" {
		t.Fatalf("expected webhook sync to be disabled, got %+v", payload.Security.Webhooks)
	}
	if payload.Security.ApprovalPolicies.Status != "reserved" || payload.Security.ApprovalPolicies.RulesCount != 0 {
		t.Fatalf("expected reserved approval policy diagnostics, got %+v", payload.Security.ApprovalPolicies)
	}
	if len(payload.Security.AgentTokens.SupportedScopeGroups) == 0 {
		t.Fatal("expected supported scope groups to be returned")
	}
	wantGroups := agentplatform.SupportedScopeGroups()
	if len(payload.Security.AgentTokens.SupportedScopeGroups) != len(wantGroups) {
		t.Fatalf("expected %d scope groups, got %+v", len(wantGroups), payload.Security.AgentTokens.SupportedScopeGroups)
	}
	for index, want := range wantGroups {
		got := payload.Security.AgentTokens.SupportedScopeGroups[index]
		if got.Category != want.Category {
			t.Fatalf("scope group %d category = %q, want %q", index, got.Category, want.Category)
		}
		if strings.Join(got.Scopes, ",") != strings.Join(want.Scopes, ",") {
			t.Fatalf("scope group %s scopes = %v, want %v", got.Category, got.Scopes, want.Scopes)
		}
	}
	if len(payload.Security.Deferred) == 0 {
		t.Fatal("expected deferred security scope to be described")
	}
}

func TestSecuritySettingsRouteIncludesSecretMigrationDiagnostics(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	machineID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID}
	catalog.machines[machineID] = domain.Machine{
		ID:             machineID,
		OrganizationID: orgID,
		Name:           "builder",
		Host:           "builder.internal",
		Status:         domain.MachineStatusOnline,
		EnvVars:        []string{"OPENAI_API_KEY=sk-live-1234", "CUDA_VISIBLE_DEVICES=0"},
	}
	catalog.providers[uuid.New()] = domain.AgentProvider{
		ID:             uuid.New(),
		OrganizationID: orgID,
		MachineID:      machineID,
		MachineName:    "builder",
		MachineHost:    "builder.internal",
		MachineStatus:  domain.MachineStatusOnline,
		Name:           "Codex",
		AdapterType:    domain.AgentProviderAdapterTypeCodexAppServer,
		AuthConfig: map[string]any{
			"base_url":       "http://localhost:4318",
			"openai_api_key": "legacy-inline-secret",
		},
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubAuthService(&stubGitHubAuthService{
			security: sampleProjectSecurity(),
		}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/security-settings",
		http.NoBody,
	)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Security securitySettingsResponse `json:"security"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Security.SecretHygiene.LegacyProvidersRequiringMigration != 1 ||
		payload.Security.SecretHygiene.LegacyProviderInlineSecretBindings != 1 {
		t.Fatalf("unexpected provider secret hygiene: %+v", payload.Security.SecretHygiene)
	}
	if payload.Security.SecretHygiene.LegacyMachinesRequiringMigration != 1 ||
		payload.Security.SecretHygiene.LegacyMachineSecretEnvVars != 1 {
		t.Fatalf("unexpected machine secret hygiene: %+v", payload.Security.SecretHygiene)
	}
	if len(payload.Security.SecretHygiene.RolloutChecklist) == 0 {
		t.Fatalf("expected rollout checklist entries, got %+v", payload.Security.SecretHygiene)
	}
}

func TestSecuritySettingsRouteSavesManualCredential(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
	authSvc := &stubGitHubAuthService{security: sampleProjectSecurity()}
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubAuthService(authSvc),
	)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/projects/"+projectID.String()+"/security-settings/github-outbound-credential",
		strings.NewReader(`{"token":"ghu_manual_token"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if authSvc.lastSaveInput.Scope != githubauthdomain.ScopeProject || authSvc.lastSaveInput.Token != "ghu_manual_token" {
		t.Fatalf("SaveManualCredential() input = %+v", authSvc.lastSaveInput)
	}
}

func TestSecuritySettingsRouteImportsCredentialFromGHCLI(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
	authSvc := &stubGitHubAuthService{security: sampleProjectSecurity()}
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubAuthService(authSvc),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/security-settings/github-outbound-credential/import-gh-cli",
		strings.NewReader(`{"scope":"project"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if authSvc.lastScopeInput.Scope != githubauthdomain.ScopeProject {
		t.Fatalf("ImportGHCLICredential() input = %+v", authSvc.lastScopeInput)
	}
}

func TestSecuritySettingsRouteRetestMapsMissingCredential(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubAuthService(&stubGitHubAuthService{
			retestErr: githubauthservice.ErrCredentialNotConfigured,
		}),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/security-settings/github-outbound-credential/retest",
		strings.NewReader(`{"scope":"project"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSecuritySettingsRouteDeletesCredential(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
	authSvc := &stubGitHubAuthService{security: sampleProjectSecurity()}
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubAuthService(authSvc),
	)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/projects/"+projectID.String()+"/security-settings/github-outbound-credential",
		http.NoBody,
	)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if authSvc.lastScopeInput.Scope != githubauthdomain.ScopeProject {
		t.Fatalf("DeleteCredential() input = %+v", authSvc.lastScopeInput)
	}
}

func TestSecuritySettingsRouteRejectsUnknownProject(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
		nil,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+uuid.New().String()+"/security-settings",
		http.NoBody,
	)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSecuritySettingsRouteRejectsInvalidProjectID(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
		nil,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/not-a-uuid/security-settings",
		http.NoBody,
	)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSecuritySettingsRouteSavesOIDCDraftWithoutChangingMode(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	client, instanceAuthSvc := newInstanceAuthTestService(t, config.AuthConfig{Mode: config.AuthModeDisabled}, configPath)
	server := NewServer(
		config.ServerConfig{Port: 40023, Host: "127.0.0.1"},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithRuntimeConfigFile(configPath),
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeDisabled}),
		WithInstanceAuthService(instanceAuthSvc),
	)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/projects/"+projectID.String()+"/security-settings/oidc-draft",
		strings.NewReader(`{"issuer_url":"https://idp.example.com","client_id":"openase","client_secret":"secret","redirect_mode":"fixed","fixed_redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"],"session_ttl":"8h","session_idle_ttl":"30m"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Security securitySettingsResponse `json:"security"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Security.Auth.ActiveMode != "disabled" {
		t.Fatalf("active mode = %q, want disabled", payload.Security.Auth.ActiveMode)
	}
	if payload.Security.Auth.ConfiguredMode != "disabled" {
		t.Fatalf("configured mode = %q, want disabled", payload.Security.Auth.ConfiguredMode)
	}
	if payload.Security.Auth.OIDCDraft.IssuerURL != "https://idp.example.com" {
		t.Fatalf("issuer_url = %q", payload.Security.Auth.OIDCDraft.IssuerURL)
	}
	if payload.Security.Auth.OIDCDraft.RedirectMode != "fixed" {
		t.Fatalf("redirect_mode = %q, want fixed", payload.Security.Auth.OIDCDraft.RedirectMode)
	}
	if !payload.Security.Auth.OIDCDraft.ClientSecretConfigured {
		t.Fatal("expected saved oidc client secret to be marked as configured")
	}
	stored, err := client.InstanceAuthConfig.Query().Only(req.Context())
	if err != nil {
		t.Fatalf("load instance auth config: %v", err)
	}
	if stored.Status != "draft" {
		t.Fatalf("stored status = %q, want draft", stored.Status)
	}
	if stored.ClientSecretEncrypted == nil {
		t.Fatal("expected encrypted client secret to be persisted")
	}
	if stored.SessionTTL != "8h0m0s" {
		t.Fatalf("stored session_ttl = %q, want 8h0m0s", stored.SessionTTL)
	}
	if stored.SessionIdleTTL != "30m0s" {
		t.Fatalf("stored session_idle_ttl = %q, want 30m0s", stored.SessionIdleTTL)
	}
}

func TestSecuritySettingsRouteTestsOIDCDraft(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	issuerServer := newTestOIDCDiscoveryServer(t)
	defer issuerServer.Close()
	_, instanceAuthSvc := newInstanceAuthTestService(t, config.AuthConfig{Mode: config.AuthModeDisabled}, configPath)
	server := NewServer(
		config.ServerConfig{Port: 40023, Host: "127.0.0.1"},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithRuntimeConfigFile(configPath),
		WithInstanceAuthService(instanceAuthSvc),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/security-settings/oidc-draft/test",
		strings.NewReader(`{"issuer_url":"`+issuerServer.URL+`","client_id":"openase","client_secret":"secret","redirect_mode":"auto","fixed_redirect_url":"","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"],"session_ttl":"8h","session_idle_ttl":"30m"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "desktop.example.com")
	req.Header.Set("X-Forwarded-Port", "43123")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload securityOIDCTestResultResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("status = %q, want ok", payload.Status)
	}
	if payload.AuthorizationEndpoint == "" || payload.TokenEndpoint == "" {
		t.Fatalf("expected discovery endpoints, got %+v", payload)
	}
	if payload.RedirectURL != "https://desktop.example.com:43123/api/v1/auth/oidc/callback" {
		t.Fatalf("redirect_url = %q", payload.RedirectURL)
	}
}

func TestSecuritySettingsRouteEnablesOIDCInConfig(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
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
		catalog,
		nil,
		WithRuntimeConfigFile(configPath),
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeDisabled}),
		WithInstanceAuthService(instanceAuthSvc),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/security-settings/oidc-enable",
		strings.NewReader(`{"issuer_url":"`+issuerServer.URL+`","client_id":"openase","client_secret":"secret","redirect_mode":"fixed","fixed_redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"],"session_ttl":"8h","session_idle_ttl":"30m"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload securityOIDCEnableResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Activation.Status != "configured" || payload.Activation.RestartRequired {
		t.Fatalf("unexpected activation payload: %+v", payload.Activation)
	}
	if payload.Security.Auth.ActiveMode != "oidc" {
		t.Fatalf("active mode = %q, want oidc", payload.Security.Auth.ActiveMode)
	}
	if payload.Security.Auth.ConfiguredMode != "oidc" {
		t.Fatalf("configured mode = %q, want oidc", payload.Security.Auth.ConfiguredMode)
	}
	stored, err := client.InstanceAuthConfig.Query().Only(req.Context())
	if err != nil {
		t.Fatalf("load instance auth config: %v", err)
	}
	if stored.Status != "active" {
		t.Fatalf("stored status = %q, want active", stored.Status)
	}
	if stored.SessionTTL != "8h0m0s" {
		t.Fatalf("stored session_ttl = %q, want 8h0m0s", stored.SessionTTL)
	}
	if stored.SessionIdleTTL != "30m0s" {
		t.Fatalf("stored session_idle_ttl = %q, want 30m0s", stored.SessionIdleTTL)
	}
}

func TestSecuritySettingsRouteRejectsInvalidOIDCSessionPolicy(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New()}
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
		catalog,
		nil,
		WithRuntimeConfigFile(configPath),
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeDisabled}),
		WithInstanceAuthService(instanceAuthSvc),
	)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/projects/"+projectID.String()+"/security-settings/oidc-draft",
		strings.NewReader(`{"issuer_url":"https://idp.example.com","client_id":"openase","client_secret":"secret","redirect_mode":"fixed","fixed_redirect_url":"http://127.0.0.1:19836/api/v1/auth/oidc/callback","scopes":["openid","profile","email"],"allowed_email_domains":["example.com"],"bootstrap_admin_emails":["admin@example.com"],"session_ttl":"1h","session_idle_ttl":"2h"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	assertAPIErrorResponse(
		t,
		rec,
		http.StatusBadRequest,
		"INVALID_REQUEST",
		"session_idle_ttl must not exceed session_ttl",
	)
}

func newTestOIDCDiscoveryServer(t *testing.T) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"issuer":"`+server.URL+`","authorization_endpoint":"`+server.URL+`/authorize","token_endpoint":"`+server.URL+`/token","jwks_uri":"`+server.URL+`/jwks","response_types_supported":["code"],"subject_types_supported":["public"],"id_token_signing_alg_values_supported":["RS256"]}`)
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"keys":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	return server
}

type stubGitHubAuthService struct {
	security       githubauthservice.ProjectSecurity
	lastSaveInput  githubauthservice.SaveCredentialInput
	lastScopeInput githubauthservice.ScopeInput
	readErr        error
	saveErr        error
	importErr      error
	retestErr      error
	deleteErr      error
}

func (s *stubGitHubAuthService) ReadProjectSecurity(context.Context, uuid.UUID) (githubauthservice.ProjectSecurity, error) {
	return s.security, s.readErr
}

func (s *stubGitHubAuthService) SaveManualCredential(_ context.Context, input githubauthservice.SaveCredentialInput) (githubauthservice.ProjectSecurity, error) {
	s.lastSaveInput = input
	return s.security, s.saveErr
}

func (s *stubGitHubAuthService) ImportGHCLICredential(_ context.Context, input githubauthservice.ScopeInput) (githubauthservice.ProjectSecurity, error) {
	s.lastScopeInput = input
	return s.security, s.importErr
}

func (s *stubGitHubAuthService) RetestCredential(_ context.Context, input githubauthservice.ScopeInput) (githubauthservice.ProjectSecurity, error) {
	s.lastScopeInput = input
	return s.security, s.retestErr
}

func (s *stubGitHubAuthService) DeleteCredential(_ context.Context, input githubauthservice.ScopeInput) (githubauthservice.ProjectSecurity, error) {
	s.lastScopeInput = input
	return s.security, s.deleteErr
}

func sampleProjectSecurity() githubauthservice.ProjectSecurity {
	checkedAt := timePtr(time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC))
	return githubauthservice.ProjectSecurity{
		Effective: githubauthservice.ScopedSecurity{
			Scope:        githubauthdomain.ScopeOrganization,
			Configured:   true,
			Source:       githubauthdomain.SourceGHCLIImport,
			TokenPreview: "ghu_test...1234",
			Probe: githubauthdomain.TokenProbe{
				State:       githubauthdomain.ProbeStateValid,
				Configured:  true,
				Valid:       true,
				Login:       "octocat",
				Permissions: []string{"read:org", "repo"},
				RepoAccess:  githubauthdomain.RepoAccessGranted,
				CheckedAt:   checkedAt,
			},
		},
		Organization: githubauthservice.ScopedSecurity{
			Scope:        githubauthdomain.ScopeOrganization,
			Configured:   true,
			Source:       githubauthdomain.SourceGHCLIImport,
			TokenPreview: "ghu_test...1234",
			Probe: githubauthdomain.TokenProbe{
				State:       githubauthdomain.ProbeStateValid,
				Configured:  true,
				Valid:       true,
				Login:       "octocat",
				Permissions: []string{"read:org", "repo"},
				RepoAccess:  githubauthdomain.RepoAccessGranted,
				CheckedAt:   checkedAt,
			},
		},
		ProjectOverride: githubauthservice.ScopedSecurity{
			Scope:      githubauthdomain.ScopeProject,
			Configured: false,
			Probe:      githubauthdomain.MissingProbe(),
		},
	}
}

func timePtr(value time.Time) *time.Time {
	copied := value.UTC()
	return &copied
}
