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
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
		"github.com/google/uuid"
)

func TestTicketRunRoutesExposeRunNativeTranscriptData(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetStatusID(backlogID).
		SetIdentifier("ASE-433").
		SetTitle("Render ticket runs").
		SetDescription("Build live transcript feed").
		SetPriority("high").
		SetType("feature").
		SetCreatedBy("user:api").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	catalog := newFakeCatalogService()
	catalog.organizations[org.ID] = domain.Organization{ID: org.ID, Name: org.Name, Slug: org.Slug}
	catalog.projects[project.ID] = domain.Project{ID: project.ID, OrganizationID: org.ID, Name: project.Name, Slug: project.Slug}
	catalog.tickets[ticketItem.ID] = fakeCatalogTicket{ID: ticketItem.ID, ProjectID: project.ID}

	providerID := uuid.New()
	agentID := uuid.New()
	firstRunID := uuid.New()
	secondRunID := uuid.New()
	catalog.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: org.ID, Name: "Codex"}
	catalog.agents[agentID] = domain.Agent{ID: agentID, ProjectID: project.ID, ProviderID: providerID, Name: "Ticket Runner"}

	firstCreatedAt := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	secondCreatedAt := firstCreatedAt.Add(15 * time.Minute)
	secondRuntimeStartedAt := secondCreatedAt.Add(30 * time.Second)
	secondHeartbeatAt := secondRuntimeStartedAt.Add(2 * time.Minute)
	secondStepStatus := "running_tests"
	secondStepSummary := "Running backend transcript checks."

	catalog.agentRuns[firstRunID] = domain.AgentRun{
		ID:         firstRunID,
		AgentID:    agentID,
		TicketID:   ticketItem.ID,
		ProviderID: providerID,
		WorkflowID: uuid.New(),
		Status:     domain.AgentRunStatusCompleted,
		CreatedAt:  firstCreatedAt,
	}
	catalog.agentRuns[secondRunID] = domain.AgentRun{
		ID:                 secondRunID,
		AgentID:            agentID,
		TicketID:           ticketItem.ID,
		ProviderID:         providerID,
		WorkflowID:         uuid.New(),
		Status:             domain.AgentRunStatusExecuting,
		RuntimeStartedAt:   &secondRuntimeStartedAt,
		LastHeartbeatAt:    &secondHeartbeatAt,
		CurrentStepStatus:  &secondStepStatus,
		CurrentStepSummary: &secondStepSummary,
		CreatedAt:          secondCreatedAt,
	}

	traceOneAt := secondRuntimeStartedAt.Add(10 * time.Second)
	traceTwoAt := traceOneAt.Add(5 * time.Second)
	traceThreeAt := traceTwoAt.Add(5 * time.Second)
	traceFourAt := traceThreeAt.Add(5 * time.Second)
	traceFiveAt := traceFourAt.Add(5 * time.Second)
	traceSixAt := traceFiveAt.Add(5 * time.Second)
	catalog.traceEvents = []domain.AgentTraceEntry{
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			Sequence:   1,
			Provider:   "codex",
			Kind:       domain.AgentTraceKindAssistantDelta,
			Stream:     "assistant",
			Output:     "Inspecting repo structure\n",
			Payload:    map[string]any{"item_id": "assistant-1"},
			CreatedAt:  traceOneAt,
		},
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			Sequence:   2,
			Provider:   "codex",
			Kind:       domain.AgentTraceKindToolCallStarted,
			Stream:     "tool",
			Output:     "functions.exec_command",
			Payload: map[string]any{
				"tool":      "functions.exec_command",
				"call_id":   "call-1",
				"arguments": map[string]any{"cmd": "pnpm vitest run", "workdir": "/repo"},
			},
			CreatedAt: traceTwoAt,
		},
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			Sequence:   3,
			Provider:   "codex",
			Kind:       domain.AgentTraceKindCommandDelta,
			Stream:     "command",
			Output:     "ok   ./internal/httpapi\n",
			Payload:    map[string]any{"item_id": "command-1", "command": "pnpm vitest run"},
			CreatedAt:  traceThreeAt,
		},
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			Sequence:   4,
			Provider:   "codex",
			Kind:       domain.AgentTraceKindThreadStatus,
			Stream:     "system",
			Output:     "active · waitingOnUserInput",
			Payload:    map[string]any{"status": "active", "active_flags": []string{"waitingOnUserInput"}},
			CreatedAt:  traceFourAt,
		},
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			Sequence:   5,
			Provider:   "codex",
			Kind:       domain.AgentTraceKindReasoningUpdated,
			Stream:     "reasoning",
			Output:     "Inspecting ticket detail reducer.",
			Payload:    map[string]any{"kind": "text_delta", "content_index": 1},
			CreatedAt:  traceFiveAt,
		},
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			Sequence:   6,
			Provider:   "codex",
			Kind:       domain.AgentTraceKindTurnDiffUpdated,
			Stream:     "diff",
			Output:     "diff --git a/app.ts b/app.ts\n@@ -1 +1 @@\n-old\n+new",
			Payload:    map[string]any{"turn_id": "turn-1"},
			CreatedAt:  traceSixAt,
		},
	}
	catalog.stepEvents = []domain.AgentStepEntry{
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			StepStatus: "planning",
			Summary:    "Inspecting ticket detail data paths.",
			CreatedAt:  secondRuntimeStartedAt,
		},
		{
			ID:         uuid.New(),
			ProjectID:  project.ID,
			TicketID:   &ticketItem.ID,
			AgentID:    agentID,
			AgentRunID: secondRunID,
			StepStatus: "running_tests",
			Summary:    secondStepSummary,
			CreatedAt:  traceThreeAt,
		},
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalog,
		nil,
	)

	listRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticketItem.ID.String()+"/runs",
		"",
	)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected ticket runs 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listPayload struct {
		Runs []ticketRunResponse `json:"runs"`
	}
	decodeResponse(t, listRec, &listPayload)
	if len(listPayload.Runs) != 2 {
		t.Fatalf("expected two ticket runs, got %+v", listPayload.Runs)
	}
	if listPayload.Runs[0].ID != secondRunID.String() || listPayload.Runs[0].AttemptNumber != 2 {
		t.Fatalf("expected newest run first with attempt number 2, got %+v", listPayload.Runs[0])
	}
	if listPayload.Runs[0].Status != "executing" || listPayload.Runs[0].Provider != "Codex" {
		t.Fatalf("expected mapped status/provider on latest run, got %+v", listPayload.Runs[0])
	}

	detailRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticketItem.ID.String()+"/runs/"+secondRunID.String(),
		"",
	)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected ticket run detail 200, got %d: %s", detailRec.Code, detailRec.Body.String())
	}
	var detailPayload struct {
		Run          ticketRunResponse             `json:"run"`
		TraceEntries []ticketRunTraceEntryResponse `json:"trace_entries"`
		StepEntries  []ticketRunStepEntryResponse  `json:"step_entries"`
	}
	decodeResponse(t, detailRec, &detailPayload)
	if detailPayload.Run.AttemptNumber != 2 || detailPayload.Run.CurrentStepStatus == nil || *detailPayload.Run.CurrentStepStatus != "running_tests" {
		t.Fatalf("expected latest run detail to expose attempt/current step, got %+v", detailPayload.Run)
	}
	if len(detailPayload.TraceEntries) != 6 || detailPayload.TraceEntries[1].Kind != domain.AgentTraceKindToolCallStarted {
		t.Fatalf("expected ordered raw trace entries, got %+v", detailPayload.TraceEntries)
	}
	if tool := detailPayload.TraceEntries[1].Payload["tool"]; tool != "functions.exec_command" {
		t.Fatalf("expected tool_call payload to round-trip, got %+v", detailPayload.TraceEntries[1])
	}
	if arguments := detailPayload.TraceEntries[1].Payload["arguments"]; arguments == nil {
		t.Fatalf("expected tool_call arguments to round-trip, got %+v", detailPayload.TraceEntries[1])
	}
	if command := detailPayload.TraceEntries[2].Payload["command"]; command != "pnpm vitest run" {
		t.Fatalf("expected command payload to round-trip, got %+v", detailPayload.TraceEntries[2])
	}
	if detailPayload.TraceEntries[3].Kind != domain.AgentTraceKindThreadStatus ||
		detailPayload.TraceEntries[4].Kind != domain.AgentTraceKindReasoningUpdated ||
		detailPayload.TraceEntries[5].Kind != domain.AgentTraceKindTurnDiffUpdated {
		t.Fatalf("expected thread/reasoning/diff trace entries, got %+v", detailPayload.TraceEntries)
	}
	if len(detailPayload.StepEntries) != 2 || detailPayload.StepEntries[1].StepStatus != "running_tests" {
		t.Fatalf("expected run-scoped step entries, got %+v", detailPayload.StepEntries)
	}
}

func TestTicketRunStreamFiltersTicketScopedLifecycleTraceAndStepEvents(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	bus := eventinfra.NewChannelBus()

	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetStatusID(backlogID).
		SetIdentifier("ASE-433").
		SetTitle("Stream ticket runs").
		SetDescription("Build live transcript feed").
		SetPriority("high").
		SetType("feature").
		SetCreatedBy("user:api").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	otherTicketID := uuid.New()

	catalog := newFakeCatalogService()
	catalog.organizations[org.ID] = domain.Organization{ID: org.ID, Name: org.Name, Slug: org.Slug}
	catalog.projects[project.ID] = domain.Project{ID: project.ID, OrganizationID: org.ID, Name: project.Name, Slug: project.Slug}
	catalog.tickets[ticketItem.ID] = fakeCatalogTicket{ID: ticketItem.ID, ProjectID: project.ID}
	catalog.tickets[otherTicketID] = fakeCatalogTicket{ID: otherTicketID, ProjectID: project.ID}

	providerID := uuid.New()
	agentID := uuid.New()
	runID := uuid.New()
	catalog.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: org.ID, Name: "Codex"}
	catalog.agents[agentID] = domain.Agent{ID: agentID, ProjectID: project.ID, ProviderID: providerID, Name: "Ticket Runner"}
	catalog.agentRuns[runID] = domain.AgentRun{
		ID:         runID,
		AgentID:    agentID,
		TicketID:   ticketItem.ID,
		ProviderID: providerID,
		WorkflowID: uuid.New(),
		Status:     domain.AgentRunStatusReady,
		CreatedAt:  time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalog,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	response, cancel := openSSERequest(
		t,
		testServer.URL+"/api/v1/projects/"+project.ID.String()+"/events/stream",
	)
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close ticket run stream response body: %v", err)
		}
	})

	publishTestEvent(
		t,
		bus,
		ticketRunActivityStreamTopic,
		provider.MustParseEventType(activityevent.TypeAgentReady.String()),
		map[string]any{
			"event": map[string]any{
				"project_id": project.ID.String(),
				"ticket_id":  otherTicketID.String(),
				"event_type": activityevent.TypeAgentReady.String(),
				"message":    "ignore me",
				"metadata":   map[string]any{"run_id": runID.String()},
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	)
	publishTestEvent(
		t,
		bus,
		ticketRunActivityStreamTopic,
		provider.MustParseEventType(activityevent.TypeAgentReady.String()),
		map[string]any{
			"event": map[string]any{
				"project_id": project.ID.String(),
				"ticket_id":  ticketItem.ID.String(),
				"event_type": activityevent.TypeAgentReady.String(),
				"message":    "Runtime ready",
				"metadata":   map[string]any{"run_id": runID.String()},
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	)
	publishTestEvent(
		t,
		bus,
		agentTraceStreamTopic,
		provider.MustParseEventType(domain.AgentOutputEventType),
		map[string]any{
			"entry": map[string]any{
				"id":           uuid.NewString(),
				"project_id":   project.ID.String(),
				"ticket_id":    ticketItem.ID.String(),
				"agent_run_id": runID.String(),
				"sequence":     1,
				"provider":     "codex",
				"kind":         domain.AgentTraceKindAssistantDelta,
				"stream":       "assistant",
				"output":       "Planning the fix",
				"payload":      map[string]any{"item_id": "assistant-1"},
				"created_at":   time.Now().UTC().Format(time.RFC3339),
			},
		},
	)
	publishTestEvent(
		t,
		bus,
		agentStepStreamTopic,
		provider.MustParseEventType(domain.AgentStepEventType),
		map[string]any{
			"entry": map[string]any{
				"id":                    uuid.NewString(),
				"project_id":            project.ID.String(),
				"ticket_id":             ticketItem.ID.String(),
				"agent_run_id":          runID.String(),
				"step_status":           "planning",
				"summary":               "Inspecting transcript reducer behavior.",
				"source_trace_event_id": nil,
				"created_at":            time.Now().UTC().Format(time.RFC3339),
			},
		},
	)

	body := readSSEBody(t, response, cancel)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected project event bus 200, got %d", response.StatusCode)
	}
	if !strings.Contains(body, "\"topic\":\"ticket.run.events\"") {
		t.Fatalf("expected dedicated ticket run stream topic, got %q", body)
	}
	if !strings.Contains(body, "event: ticket.run.lifecycle") || !strings.Contains(body, "\"message\":\"Runtime ready\"") {
		t.Fatalf("expected lifecycle frame, got %q", body)
	}
	if !strings.Contains(body, "event: ticket.run.trace") || !strings.Contains(body, "\"output\":\"Planning the fix\"") {
		t.Fatalf("expected trace frame, got %q", body)
	}
	if !strings.Contains(body, "event: ticket.run.step") || !strings.Contains(body, "\"step_status\":\"planning\"") {
		t.Fatalf("expected step frame, got %q", body)
	}
	if strings.Contains(body, "\"topic\":\"ticket.run.events\",\"type\":\"ticket.run.lifecycle\",\"payload\":{\"lifecycle\":{\"event_type\":\"agent.ready\",\"message\":\"ignore me\"") {
		t.Fatalf("did not expect other-ticket lifecycle event to be promoted onto the ticket run bus, got %q", body)
	}
}
