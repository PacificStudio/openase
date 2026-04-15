package httpapi

import (
	"context"
	"io"
	"log/slog"
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

func TestProjectEventStreamPGNotifyCarriesRuntimeLifecycleSummaryAndDashboardRefresh(t *testing.T) {
	dsn := testPostgres.NewIsolatedDatabase(t).DSN
	bus, err := eventinfra.NewPGNotifyBus(dsn, nil)
	if err != nil {
		t.Fatalf("NewPGNotifyBus returned error: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := bus.Close(); closeErr != nil {
			t.Errorf("Close returned error: %v", closeErr)
		}
	})

	projectID := uuid.New()
	otherProjectID := uuid.New()
	ticketID := uuid.New()
	runID := uuid.New()
	agentID := uuid.New()
	providerID := uuid.New()

	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID, OrganizationID: uuid.New(), Name: "OpenASE", Slug: "openase"}
	catalog.tickets[ticketID] = fakeCatalogTicket{ID: ticketID, ProjectID: projectID}
	catalog.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: uuid.New(), Name: "Codex"}
	catalog.agents[agentID] = domain.Agent{ID: agentID, ProjectID: projectID, ProviderID: providerID, Name: "Ticket Runner"}
	catalog.agentRuns[runID] = domain.AgentRun{
		ID:         runID,
		AgentID:    agentID,
		TicketID:   ticketID,
		ProviderID: providerID,
		WorkflowID: uuid.New(),
		Status:     domain.AgentRunStatusExecuting,
		CreatedAt:  time.Date(2026, time.April, 15, 6, 0, 0, 0, time.UTC),
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		catalog,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	previousDebounce := projectDashboardRefreshDebounceInterval
	projectDashboardRefreshDebounceInterval = 10 * time.Millisecond
	defer func() {
		projectDashboardRefreshDebounceInterval = previousDebounce
	}()

	response, cancel := openSSERequest(
		t,
		testServer.URL+"/api/v1/projects/"+projectID.String()+"/events/stream",
	)
	t.Cleanup(func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close project event stream response body: %v", closeErr)
		}
	})

	publishProviderEvent(t, bus, agentStreamTopic, provider.MustParseEventType("agent.ready"), map[string]any{
		"agent": map[string]any{
			"id":                agentID.String(),
			"project_id":        projectID.String(),
			"current_ticket_id": ticketID.String(),
		},
	})
	publishProviderEvent(t, bus, activityStreamTopic, provider.MustParseEventType("agent.executing"), map[string]any{
		"event": map[string]any{
			"id":         uuid.NewString(),
			"project_id": projectID.String(),
			"ticket_id":  ticketID.String(),
			"event_type": "agent.executing",
			"message":    "Runtime executing",
			"metadata": map[string]any{
				"run_id": runID.String(),
			},
			"created_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	publishProviderEvent(t, bus, ticketRunStreamTopic, ticketRunSummaryStreamType, map[string]any{
		"project_id": projectID.String(),
		"ticket_id":  ticketID.String(),
		"run_id":     runID.String(),
		"completion_summary": map[string]any{
			"status":       "completed",
			"markdown":     "## Overview\n\nRealtime catch-up verified.",
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	publishProviderEvent(t, bus, agentStreamTopic, provider.MustParseEventType("agent.failed"), map[string]any{
		"agent": map[string]any{
			"id":                uuid.NewString(),
			"project_id":        otherProjectID.String(),
			"current_ticket_id": uuid.NewString(),
		},
	})

	body := readSSEBodyUntilContainsAll(t, response, cancel, []string{
		"event: agent.ready",
		`"topic":"agent.events"`,
		"event: ticket.run.lifecycle",
		`"message":"Runtime executing"`,
		"event: ticket.run.summary",
		"Realtime catch-up verified.",
		`"type":"project.dashboard.refresh"`,
	})

	if strings.Contains(body, otherProjectID.String()) {
		t.Fatalf("did not expect unrelated project payload, got %q", body)
	}
}

func publishProviderEvent(
	t *testing.T,
	bus provider.EventProvider,
	topic provider.Topic,
	eventType provider.EventType,
	payload any,
) {
	t.Helper()

	message, err := provider.NewJSONEvent(topic, eventType, payload, time.Now())
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
}
