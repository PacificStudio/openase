package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/google/uuid"
)

func TestListActivityEventsRoute(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)
	orgID := uuid.New()
	projectID := uuid.New()
	agentOneID := uuid.New()
	agentTwoID := uuid.New()
	ticketOneID := uuid.New()
	ticketTwoID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.activityEvents = []domain.ActivityEvent{
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			TicketID:  &ticketOneID,
			AgentID:   &agentOneID,
			EventType: activityevent.TypeAgentLaunching,
			Message:   "older line",
			Metadata:  map[string]any{"stream": "stdout"},
			CreatedAt: time.Date(2026, 3, 19, 17, 1, 0, 0, time.UTC),
		},
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			TicketID:  &ticketTwoID,
			AgentID:   &agentTwoID,
			EventType: activityevent.TypeAgentReady,
			Message:   "other agent line",
			Metadata:  map[string]any{"stream": "stdout"},
			CreatedAt: time.Date(2026, 3, 19, 17, 2, 0, 0, time.UTC),
		},
		{
			ID:        uuid.New(),
			ProjectID: projectID,
			TicketID:  &ticketOneID,
			AgentID:   &agentOneID,
			EventType: activityevent.TypeAgentFailed,
			Message:   "still running",
			Metadata:  map[string]any{"stream": "system"},
			CreatedAt: time.Date(2026, 3, 19, 17, 3, 0, 0, time.UTC),
		},
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/activity?agent_id="+agentOneID.String()+"&ticket_id="+ticketOneID.String()+"&limit=1",
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
	if payload.Events[0].EventType != activityevent.TypeAgentFailed.String() || payload.Events[0].Message != "still running" {
		t.Fatalf("unexpected activity payload: %+v", payload.Events[0])
	}
}

func TestListActivityEventsRouteRejectsInvalidQuery(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+uuid.New().String()+"/activity?limit=0", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid limit to return 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListActivityEventsRouteReturnsEmptyArrayWhenNoEventsExist(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)
	orgID := uuid.New()
	projectID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+projectID.String()+"/activity", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected activity list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"events":[]`) {
		t.Fatalf("expected empty events array in payload, got %s", rec.Body.String())
	}

	var payload struct {
		Events []activityEventResponse `json:"events"`
	}
	decodeResponse(t, rec, &payload)
	if payload.Events == nil || len(payload.Events) != 0 {
		t.Fatalf("expected non-nil empty events slice, got %+v", payload.Events)
	}
}
