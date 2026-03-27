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

func TestListAgentStepsRoute(t *testing.T) {
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
	runID := uuid.New()
	ticketID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.agents[agentID] = domain.Agent{ID: agentID, ProjectID: projectID, Name: "Worker 1"}
	service.stepEvents = []domain.AgentStepEntry{
		{
			ID:         uuid.New(),
			ProjectID:  projectID,
			AgentID:    agentID,
			TicketID:   &ticketID,
			AgentRunID: runID,
			StepStatus: "planning",
			Summary:    "Inspecting repository layout.",
			CreatedAt:  time.Date(2026, 3, 23, 18, 1, 0, 0, time.UTC),
		},
		{
			ID:         uuid.New(),
			ProjectID:  projectID,
			AgentID:    agentID,
			TicketID:   &ticketID,
			AgentRunID: runID,
			StepStatus: "running_command",
			Summary:    "Running go test ./...",
			CreatedAt:  time.Date(2026, 3, 23, 18, 2, 0, 0, time.UTC),
		},
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String()+"/steps?ticket_id="+ticketID.String()+"&limit=1",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected agent steps list 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Entries []agentStepEntryResponse `json:"entries"`
	}
	decodeResponse(t, rec, &payload)
	if len(payload.Entries) != 1 {
		t.Fatalf("expected one filtered step entry, got %+v", payload.Entries)
	}
	if payload.Entries[0].StepStatus != "running_command" {
		t.Fatalf("unexpected step payload: %+v", payload.Entries[0])
	}
}

func TestAgentStepStreamFiltersStepEvents(t *testing.T) {
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

	response, cancel := openSSERequest(t, testServer.URL+"/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String()+"/steps/stream")
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close agent step stream response body: %v", err)
		}
	})

	publishAgentStepEventFrame(t, bus, otherProjectID, agentID, "ignored")
	publishAgentStepEventFrame(t, bus, projectID, agentID, "expected")

	body := readSSEBody(t, response, cancel)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if !strings.Contains(body, "\"topic\":\"agent.step.events\"") {
		t.Fatalf("expected dedicated agent step topic, got %q", body)
	}
	if !strings.Contains(body, "\"step_status\":\"expected\"") {
		t.Fatalf("expected matching step event, got %q", body)
	}
	if strings.Contains(body, "\"step_status\":\"ignored\"") {
		t.Fatalf("did not expect unrelated step event, got %q", body)
	}
}

func publishAgentStepEventFrame(
	t *testing.T,
	bus *eventinfra.ChannelBus,
	projectID uuid.UUID,
	agentID uuid.UUID,
	stepStatus string,
) {
	t.Helper()

	message, err := provider.NewJSONEvent(
		agentStepStreamTopic,
		provider.MustParseEventType(domain.AgentStepEventType),
		map[string]any{
			"entry": map[string]any{
				"id":                    uuid.NewString(),
				"project_id":            projectID.String(),
				"ticket_id":             "",
				"agent_id":              agentID.String(),
				"agent_run_id":          uuid.NewString(),
				"step_status":           stepStatus,
				"summary":               "summary",
				"source_trace_event_id": nil,
				"created_at":            time.Now().UTC().Format(time.RFC3339),
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
