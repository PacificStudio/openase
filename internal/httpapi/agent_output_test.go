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

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestListAgentOutputRoute(t *testing.T) {
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

func TestAgentOutputRouteErrorAndHelperCoverage(t *testing.T) {
	t.Run("stream route rejects invalid ticket and project mismatch", func(t *testing.T) {
		bus := eventinfra.NewChannelBus()
		service := newFakeCatalogService()
		orgID := uuid.New()
		projectID := uuid.New()
		otherProjectID := uuid.New()
		agentID := uuid.New()
		service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
		service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
		service.projects[otherProjectID] = domain.Project{ID: otherProjectID, OrganizationID: orgID, Name: "Other", Slug: "other"}
		service.agents[agentID] = domain.Agent{ID: agentID, ProjectID: otherProjectID, Name: "Worker"}

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

		invalidTicketRec := performJSONRequest(
			t,
			server,
			http.MethodGet,
			"/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String()+"/output/stream?ticket_id=bad",
			"",
		)
		if invalidTicketRec.Code != http.StatusBadRequest || !strings.Contains(invalidTicketRec.Body.String(), "ticket_id must be a valid UUID") {
			t.Fatalf("invalid ticket response = %d %s", invalidTicketRec.Code, invalidTicketRec.Body.String())
		}

		projectMismatchRec := performJSONRequest(
			t,
			server,
			http.MethodGet,
			"/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String()+"/output/stream",
			"",
		)
		if projectMismatchRec.Code != http.StatusNotFound || !strings.Contains(projectMismatchRec.Body.String(), "resource not found") {
			t.Fatalf("project mismatch response = %d %s", projectMismatchRec.Code, projectMismatchRec.Body.String())
		}
	})

	t.Run("helper filters and formats events", func(t *testing.T) {
		if parsed, err := parseOptionalUUIDQueryParam("ticket_id", " "); err != nil || parsed != nil {
			t.Fatalf("parseOptionalUUIDQueryParam(blank) = %v, %v", parsed, err)
		}
		if _, err := parseOptionalUUIDQueryParam("ticket_id", "bad"); err == nil || !strings.Contains(err.Error(), "ticket_id must be a valid UUID") {
			t.Fatalf("parseOptionalUUIDQueryParam(bad) error = %v", err)
		}

		projectID := uuid.New()
		agentID := uuid.New()
		ticketID := uuid.New()
		parsedTicketID, err := parseOptionalUUIDQueryParam("ticket_id", " "+ticketID.String()+" ")
		if err != nil || parsedTicketID == nil || *parsedTicketID != ticketID {
			t.Fatalf("parseOptionalUUIDQueryParam(valid) = %v, %v", parsedTicketID, err)
		}

		wrongTypeEvent, err := provider.NewJSONEvent(
			agentTraceStreamTopic,
			provider.MustParseEventType("agent.heartbeat"),
			map[string]any{"entry": map[string]any{}},
			time.Now(),
		)
		if err != nil {
			t.Fatalf("NewJSONEvent(wrong type) error = %v", err)
		}
		if output, matched, err := buildAgentOutputStreamEvent(projectID, agentID, nil, wrongTypeEvent); err != nil || matched || output.Topic != "" || output.Type != "" || len(output.Payload) != 0 {
			t.Fatalf("buildAgentOutputStreamEvent(wrong type) = %+v, %t, %v", output, matched, err)
		}

		malformed := provider.Event{
			Topic:       agentTraceStreamTopic,
			Type:        provider.MustParseEventType(domain.AgentOutputEventType),
			Payload:     []byte("{"),
			PublishedAt: time.Now(),
		}
		if _, _, err := buildAgentOutputStreamEvent(projectID, agentID, nil, malformed); err == nil || !strings.Contains(err.Error(), "decode trace stream payload") {
			t.Fatalf("buildAgentOutputStreamEvent(malformed) error = %v", err)
		}

		makeEvent := func(payload map[string]any) provider.Event {
			t.Helper()

			event, err := provider.NewJSONEvent(
				agentTraceStreamTopic,
				provider.MustParseEventType(domain.AgentOutputEventType),
				map[string]any{"entry": payload},
				time.Now(),
			)
			if err != nil {
				t.Fatalf("NewJSONEvent() error = %v", err)
			}
			return event
		}

		basePayload := map[string]any{
			"id":           uuid.NewString(),
			"project_id":   projectID.String(),
			"agent_id":     agentID.String(),
			"ticket_id":    ticketID.String(),
			"agent_run_id": uuid.NewString(),
			"stream":       "stdout",
			"output":       "stdout line",
			"created_at":   time.Date(2026, time.March, 27, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		}

		for _, testCase := range []struct {
			name    string
			payload map[string]any
			filter  *uuid.UUID
		}{
			{name: "project mismatch", payload: cloneMapWith(basePayload, "project_id", uuid.NewString())},
			{name: "missing agent", payload: removeMapKey(basePayload, "agent_id")},
			{name: "agent mismatch", payload: cloneMapWith(basePayload, "agent_id", uuid.NewString())},
			{name: "ticket mismatch", payload: cloneMapWith(basePayload, "ticket_id", uuid.NewString()), filter: &ticketID},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				if output, matched, err := buildAgentOutputStreamEvent(projectID, agentID, testCase.filter, makeEvent(testCase.payload)); err != nil || matched || output.Topic != "" || output.Type != "" || len(output.Payload) != 0 {
					t.Fatalf("buildAgentOutputStreamEvent(%s) = %+v, %t, %v", testCase.name, output, matched, err)
				}
			})
		}

		matchedEvent, matched, err := buildAgentOutputStreamEvent(projectID, agentID, &ticketID, makeEvent(basePayload))
		if err != nil || !matched {
			t.Fatalf("buildAgentOutputStreamEvent(match) = %+v, %t, %v", matchedEvent, matched, err)
		}
		if matchedEvent.Topic != agentOutputStreamTopic || matchedEvent.Type != provider.MustParseEventType(domain.AgentOutputEventType) {
			t.Fatalf("matched event envelope = %+v", matchedEvent)
		}

		var payload struct {
			Entry agentOutputEntryResponse `json:"entry"`
		}
		if err := json.Unmarshal(matchedEvent.Payload, &payload); err != nil {
			t.Fatalf("decode matched payload: %v", err)
		}
		if payload.Entry.AgentID != agentID.String() || payload.Entry.TicketID == nil || *payload.Entry.TicketID != ticketID.String() || payload.Entry.Stream != "stdout" || payload.Entry.Output != "stdout line" {
			t.Fatalf("matched entry = %+v", payload.Entry)
		}
	})
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

func cloneMapWith(source map[string]any, key string, value any) map[string]any {
	cloned := make(map[string]any, len(source))
	for k, v := range source {
		cloned[k] = v
	}
	cloned[key] = value
	return cloned
}

func removeMapKey(source map[string]any, key string) map[string]any {
	cloned := make(map[string]any, len(source))
	for k, v := range source {
		if k == key {
			continue
		}
		cloned[k] = v
	}
	return cloned
}
