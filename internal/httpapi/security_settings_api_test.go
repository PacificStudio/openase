package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/google/uuid"
)

func TestSecuritySettingsRouteReturnsCurrentBoundary(t *testing.T) {
	projectID := uuid.New()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{WebhookSecret: "top-secret"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
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

func TestSecuritySettingsRouteRejectsInvalidProjectID(t *testing.T) {
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
