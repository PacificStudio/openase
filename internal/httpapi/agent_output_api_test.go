package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/google/uuid"
)

func TestGetAgentOutputRoute(t *testing.T) {
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

	service := server.catalog.(*fakeCatalogService)
	orgID := uuid.New()
	projectID := uuid.New()
	agentID := uuid.New()
	ticketID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.agents[agentID] = domain.Agent{
		ID:           agentID,
		ProviderID:   uuid.New(),
		ProjectID:    projectID,
		Name:         "Codex Worker",
		Status:       "running",
		RuntimePhase: "ready",
	}
	service.activityEvents = []domain.ActivityEvent{
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			TicketID:  &ticketID,
			AgentID:   &agentID,
			EventType: "agent.output",
			Message:   "running tests",
			Metadata:  map[string]any{"stream": "stdout"},
			CreatedAt: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			AgentID:   &agentID,
			EventType: "agent.failed",
			Message:   "Codex session exited unexpectedly",
			Metadata:  map[string]any{"stream": "system"},
			CreatedAt: time.Date(2026, 3, 21, 10, 1, 0, 0, time.UTC),
		},
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/agents/"+agentID.String()+"/output?limit=1",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected agent output 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Agent   agentResponse              `json:"agent"`
		Entries []agentOutputEntryResponse `json:"entries"`
	}
	decodeResponse(t, rec, &payload)
	if payload.Agent.ID != agentID.String() {
		t.Fatalf("expected agent id %s, got %+v", agentID, payload.Agent)
	}
	if len(payload.Entries) != 1 {
		t.Fatalf("expected one output entry, got %+v", payload.Entries)
	}
	if payload.Entries[0].EventType != "agent.failed" || payload.Entries[0].Stream != domain.AgentOutputStreamSystem {
		t.Fatalf("unexpected output entry: %+v", payload.Entries[0])
	}
}

func TestGetAgentOutputRouteRejectsInvalidLimit(t *testing.T) {
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

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/agents/"+uuid.New().String()+"/output?limit=0", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid limit to return 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
