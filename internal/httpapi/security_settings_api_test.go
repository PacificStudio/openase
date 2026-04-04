package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
		strings.NewReader(`{"scope":"organization","token":"ghu_manual_token"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if authSvc.lastSaveInput.Scope != githubauthdomain.ScopeOrganization || authSvc.lastSaveInput.Token != "ghu_manual_token" {
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
		"/api/v1/projects/"+projectID.String()+"/security-settings/github-outbound-credential?scope=organization",
		http.NoBody,
	)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if authSvc.lastScopeInput.Scope != githubauthdomain.ScopeOrganization {
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
