package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
		WithGitHubAuthService(stubGitHubAuthReader{
			security: githubauthservice.ProjectSecurity{
				Scope:        githubauthdomain.ScopeOrganization,
				Source:       githubauthdomain.SourceGHCLIImport,
				TokenPreview: "ghu_test...1234",
				Probe: githubauthdomain.TokenProbe{
					State:       githubauthdomain.ProbeStateValid,
					Configured:  true,
					Valid:       true,
					Permissions: []string{"read:org", "repo"},
					RepoAccess:  githubauthdomain.RepoAccessGranted,
					CheckedAt:   timePtr(time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)),
				},
			},
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
	if payload.Security.AgentTokens.TokenPrefix != agentplatform.TokenPrefix {
		t.Fatalf("expected token prefix %q, got %q", agentplatform.TokenPrefix, payload.Security.AgentTokens.TokenPrefix)
	}
	if payload.Security.GitHub.Scope != "organization" || payload.Security.GitHub.Source != "gh_cli_import" {
		t.Fatalf("expected GitHub credential metadata, got %+v", payload.Security.GitHub)
	}
	if !payload.Security.GitHub.Probe.Configured || !payload.Security.GitHub.Probe.Valid || payload.Security.GitHub.Probe.RepoAccess != "granted" {
		t.Fatalf("expected GitHub token probe details, got %+v", payload.Security.GitHub.Probe)
	}
	if !payload.Security.Webhooks.LegacyGitHubSignatureRequired {
		t.Fatal("expected legacy GitHub signature to be required when webhook secret is configured")
	}
	if !payload.Security.SecretHygiene.NotificationChannelConfigsRedacted {
		t.Fatal("expected notification channel configs to be marked redacted")
	}
	if len(payload.Security.Deferred) == 0 {
		t.Fatal("expected deferred security scope to be described")
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

type stubGitHubAuthReader struct {
	security githubauthservice.ProjectSecurity
	err      error
}

func (s stubGitHubAuthReader) ReadProjectSecurity(context.Context, uuid.UUID) (githubauthservice.ProjectSecurity, error) {
	return s.security, s.err
}

func timePtr(value time.Time) *time.Time {
	copied := value.UTC()
	return &copied
}
