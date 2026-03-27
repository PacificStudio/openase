package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestListAgentOutputRoute(t *testing.T) {
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
	agentOneID := uuid.New()
	agentTwoID := uuid.New()
	ticketOneID := uuid.New()
	ticketTwoID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.agents[agentOneID] = domain.Agent{ID: agentOneID, ProjectID: projectID, Name: "Worker 1"}
	service.agents[agentTwoID] = domain.Agent{ID: agentTwoID, ProjectID: projectID, Name: "Worker 2"}
	runID := uuid.New()
	service.traceEvents = []domain.AgentTraceEntry{
		{
			ID:         uuid.New(),
			ProjectID:  projectID,
			TicketID:   &ticketOneID,
			AgentID:    agentOneID,
			AgentRunID: runID,
			Kind:       domain.AgentTraceKindCommandDelta,
			Stream:     "stdout",
			Output:     "stdout line",
			CreatedAt:  time.Date(2026, 3, 19, 17, 1, 0, 0, time.UTC),
		},
		{
			ID:         uuid.New(),
			ProjectID:  projectID,
			TicketID:   &ticketTwoID,
			AgentID:    agentTwoID,
			AgentRunID: uuid.New(),
			Kind:       domain.AgentTraceKindCommandDelta,
			Stream:     "stderr",
			Output:     "other agent line",
			CreatedAt:  time.Date(2026, 3, 19, 17, 2, 0, 0, time.UTC),
		},
		{
			ID:         uuid.New(),
			ProjectID:  projectID,
			TicketID:   &ticketOneID,
			AgentID:    agentOneID,
			AgentRunID: runID,
			Kind:       domain.AgentTraceKindToolCallStarted,
			Stream:     "tool",
			Output:     "run_shell",
			CreatedAt:  time.Date(2026, 3, 19, 17, 3, 0, 0, time.UTC),
		},
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/agents/"+agentOneID.String()+"/output?ticket_id="+ticketOneID.String()+"&limit=1",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected agent output list 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Entries []agentOutputEntryResponse `json:"entries"`
	}
	decodeResponse(t, rec, &payload)
	if len(payload.Entries) != 1 {
		t.Fatalf("expected one filtered output entry, got %+v", payload.Entries)
	}
	if payload.Entries[0].Output != "stdout line" || payload.Entries[0].Stream != "stdout" {
		t.Fatalf("unexpected output payload: %+v", payload.Entries[0])
	}
}

func TestListAgentOutputRouteRejectsInvalidQuery(t *testing.T) {
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

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+uuid.New().String()+"/agents/"+uuid.New().String()+"/output?limit=0",
		"",
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid limit to return 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAgentOutputStreamFiltersTraceEvents(t *testing.T) {
	bus := eventinfra.NewChannelBus()
	service := newFakeCatalogService()
	orgID := uuid.New()
	projectID := uuid.New()
	agentID := uuid.New()
	otherProjectID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.projects[otherProjectID] = domain.Project{ID: otherProjectID, OrganizationID: orgID, Name: "Other", Slug: "other"}
	service.agents[agentID] = domain.Agent{ID: agentID, ProjectID: projectID, Name: "Worker 1"}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		service,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	response, cancel := openSSERequest(t, testServer.URL+"/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String()+"/output/stream")
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close agent output stream response body: %v", err)
		}
	})

	publishAgentOutputTraceEvent(t, bus, otherProjectID, agentID, "ignored line")
	publishAgentOutputTraceEvent(t, bus, projectID, agentID, "expected line")

	body := readSSEBody(t, response, cancel)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected event-stream content type, got %q", contentType)
	}
	if !strings.Contains(body, "\"topic\":\"agent.output.events\"") {
		t.Fatalf("expected dedicated agent output topic, got %q", body)
	}
	if !strings.Contains(body, "\"output\":\"expected line\"") {
		t.Fatalf("expected matching output line, got %q", body)
	}
	if strings.Contains(body, "ignored line") {
		t.Fatalf("did not expect unrelated output line, got %q", body)
	}
}

func publishAgentOutputTraceEvent(
	t *testing.T,
	bus *eventinfra.ChannelBus,
	projectID uuid.UUID,
	agentID uuid.UUID,
	output string,
) {
	t.Helper()

	message, err := provider.NewJSONEvent(
		agentTraceStreamTopic,
		provider.MustParseEventType(domain.AgentOutputEventType),
		map[string]any{
			"entry": map[string]any{
				"id":           uuid.NewString(),
				"project_id":   projectID.String(),
				"ticket_id":    "",
				"agent_id":     agentID.String(),
				"agent_run_id": uuid.NewString(),
				"stream":       "stdout",
				"output":       output,
				"created_at":   time.Now().UTC().Format(time.RFC3339),
			},
		},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
}
