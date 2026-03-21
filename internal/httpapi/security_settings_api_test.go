package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/google/uuid"
)

func TestGetProjectSecuritySettings(t *testing.T) {
	catalog := newFakeCatalogService()
	projectID := uuid.New()
	catalog.projects[projectID] = domain.Project{
		ID:             projectID,
		OrganizationID: uuid.New(),
		Name:           "OpenASE",
		Slug:           "openase",
		Description:    "Main control plane",
		Status:         "active",
	}

	server := NewServer(
		config.ServerConfig{Mode: config.ServerModeAllInOne, Port: 40023},
		config.GitHubConfig{WebhookSecret: "topsecret"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+projectID.String()+"/security", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected security settings 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Security projectSecuritySettingsResponse `json:"security"`
	}
	decodeResponse(t, rec, &payload)

	if payload.Security.ProjectID != projectID.String() {
		t.Fatalf("ProjectID=%q, want %q", payload.Security.ProjectID, projectID)
	}
	if payload.Security.RuntimeMode != string(config.ServerModeAllInOne) {
		t.Fatalf("RuntimeMode=%q, want %q", payload.Security.RuntimeMode, config.ServerModeAllInOne)
	}
	if len(payload.Security.Surfaces) != 2 {
		t.Fatalf("expected 2 security surfaces, got %+v", payload.Security.Surfaces)
	}
	if !payload.Security.Surfaces[0].Exposed || !payload.Security.Surfaces[0].Configured {
		t.Fatalf("unexpected webhook surface: %+v", payload.Security.Surfaces[0])
	}
	if payload.Security.AgentPlatform.Exposed {
		t.Fatalf("expected agent platform to be hidden without service, got %+v", payload.Security.AgentPlatform)
	}
	if payload.Security.AgentPlatform.ActiveTokenCount != 0 || payload.Security.AgentPlatform.ExpiredTokenCount != 0 {
		t.Fatalf("unexpected token counts: %+v", payload.Security.AgentPlatform)
	}
	if len(payload.Security.AgentPlatform.DefaultScopes) == 0 {
		t.Fatalf("expected default scopes, got %+v", payload.Security.AgentPlatform)
	}
	if payload.Security.AgentPlatform.DefaultScopes[0] != agentplatform.DefaultScopes()[0] {
		t.Fatalf("DefaultScopes=%v, want prefix %v", payload.Security.AgentPlatform.DefaultScopes, agentplatform.DefaultScopes())
	}
}
