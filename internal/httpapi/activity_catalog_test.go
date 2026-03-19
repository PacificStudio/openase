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

func TestListActivityEventsRoute(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newFakeCatalogService(),
		nil,
	)

	service := server.catalog.(*fakeCatalogService)
	orgID := uuid.New()
	projectID := uuid.New()
	agentOneID := uuid.New()
	agentTwoID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.activityEvents = []domain.ActivityEvent{
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			AgentID:   &agentOneID,
			EventType: "agent.output",
			Message:   "older line",
			Metadata:  map[string]any{"stream": "stdout"},
			CreatedAt: time.Date(2026, 3, 19, 17, 1, 0, 0, time.UTC),
		},
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			AgentID:   &agentTwoID,
			EventType: "agent.output",
			Message:   "other agent line",
			Metadata:  map[string]any{"stream": "stdout"},
			CreatedAt: time.Date(2026, 3, 19, 17, 2, 0, 0, time.UTC),
		},
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			AgentID:   &agentOneID,
			EventType: "agent.heartbeat",
			Message:   "still running",
			Metadata:  map[string]any{"stream": "system"},
			CreatedAt: time.Date(2026, 3, 19, 17, 3, 0, 0, time.UTC),
		},
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/activity?agent_id="+agentOneID.String()+"&limit=1",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected activity list 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Events []activityEventResponse `json:"events"`
	}
	decodeResponse(t, rec, &payload)
	if len(payload.Events) != 1 {
		t.Fatalf("expected one filtered event, got %+v", payload.Events)
	}
	if payload.Events[0].EventType != "agent.heartbeat" || payload.Events[0].Message != "still running" {
		t.Fatalf("unexpected activity payload: %+v", payload.Events[0])
	}
}

func TestListActivityEventsRouteRejectsInvalidQuery(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newFakeCatalogService(),
		nil,
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+uuid.New().String()+"/activity?limit=0", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid limit to return 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
