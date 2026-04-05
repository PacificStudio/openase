package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
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
	catalog.providers[providerID] = domain.AgentProvider{
		ID:             providerID,
		OrganizationID: org.ID,
		Name:           "Codex",
		AdapterType:    domain.AgentProviderAdapterTypeCodexAppServer,
		ModelName:      "gpt-5.4",
	}
	catalog.agents[agentID] = domain.Agent{ID: agentID, ProjectID: project.ID, ProviderID: providerID, Name: "Ticket Runner"}

	firstCreatedAt := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	secondCreatedAt := firstCreatedAt.Add(15 * time.Minute)
	secondRuntimeStartedAt := secondCreatedAt.Add(30 * time.Second)
	secondHeartbeatAt := secondRuntimeStartedAt.Add(2 * time.Minute)
	secondStepStatus := "running_tests"
	secondStepSummary := "Running backend transcript checks."
	secondSummaryStatus := domain.AgentRunCompletionSummaryStatusCompleted
	secondSummaryMarkdown := "## Overview\n\nRun completed with a summary.\n\n## Outcome\n\nDone."
	secondSummaryGeneratedAt := secondHeartbeatAt.Add(30 * time.Second)
	secondSummaryError := ""

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
		ID:                           secondRunID,
		AgentID:                      agentID,
		TicketID:                     ticketItem.ID,
		ProviderID:                   providerID,
		WorkflowID:                   uuid.New(),
		Status:                       domain.AgentRunStatusExecuting,
		RuntimeStartedAt:             &secondRuntimeStartedAt,
		LastHeartbeatAt:              &secondHeartbeatAt,
		InputTokens:                  1200,
		OutputTokens:                 340,
		CachedInputTokens:            120,
		CacheCreationInputTokens:     45,
		ReasoningTokens:              80,
		PromptTokens:                 920,
		CandidateTokens:              260,
		ToolTokens:                   30,
		TotalTokens:                  1540,
		CurrentStepStatus:            &secondStepStatus,
		CurrentStepSummary:           &secondStepSummary,
		CompletionSummaryStatus:      &secondSummaryStatus,
		CompletionSummaryMarkdown:    &secondSummaryMarkdown,
		CompletionSummaryGeneratedAt: &secondSummaryGeneratedAt,
		CompletionSummaryError:       &secondSummaryError,
		CompletionSummaryJSON: map[string]any{
			"provider": "Codex",
		},
		CreatedAt: secondCreatedAt,
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
	if listPayload.Runs[0].AdapterType != "codex-app-server" || listPayload.Runs[0].ModelName != "gpt-5.4" {
		t.Fatalf("expected adapter/model metadata on latest run, got %+v", listPayload.Runs[0])
	}
	if !reflect.DeepEqual(
		listPayload.Runs[0].Usage,
		ticketRunUsageResponse{
			Total:         1540,
			Input:         1200,
			Output:        340,
			CachedInput:   120,
			CacheCreation: 45,
			Reasoning:     80,
			Prompt:        920,
			Candidate:     260,
			Tool:          30,
		},
	) {
		t.Fatalf("expected structured usage breakdown on latest run, got %+v", listPayload.Runs[0].Usage)
	}
	if listPayload.Runs[0].CompletionSummary == nil ||
		listPayload.Runs[0].CompletionSummary.Status != "completed" ||
		listPayload.Runs[0].CompletionSummary.Markdown == nil {
		t.Fatalf("expected list payload to expose completion summary, got %+v", listPayload.Runs[0])
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
	if detailPayload.Run.CompletionSummary == nil ||
		detailPayload.Run.CompletionSummary.GeneratedAt == nil ||
		detailPayload.Run.CompletionSummary.JSON["provider"] != "Codex" {
		t.Fatalf("expected run detail to expose completion summary payload, got %+v", detailPayload.Run)
	}
	if detailPayload.Run.AdapterType != "codex-app-server" ||
		detailPayload.Run.ModelName != "gpt-5.4" ||
		detailPayload.Run.Usage.CacheCreation != 45 ||
		detailPayload.Run.Usage.Prompt != 920 ||
		detailPayload.Run.Usage.Candidate != 260 ||
		detailPayload.Run.Usage.Tool != 30 {
		t.Fatalf("expected run detail usage breakdown and adapter/model metadata, got %+v", detailPayload.Run)
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
	publishTestEvent(
		t,
		bus,
		ticketRunStreamTopic,
		ticketRunSummaryStreamType,
		map[string]any{
			"project_id": project.ID.String(),
			"ticket_id":  ticketItem.ID.String(),
			"run_id":     runID.String(),
			"completion_summary": map[string]any{
				"status":       "completed",
				"markdown":     "## Overview\n\nSummary ready.",
				"generated_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	)

	body := readSSEBodyUntilContainsAll(t, response, cancel, []string{
		"event: ticket.run.lifecycle",
		"\"message\":\"Runtime ready\"",
		"event: ticket.run.trace",
		"\"output\":\"Planning the fix\"",
		"event: ticket.run.step",
		"\"step_status\":\"planning\"",
		"event: ticket.run.summary",
		"Summary ready.",
	})
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
	if !strings.Contains(body, "event: ticket.run.summary") || !strings.Contains(body, "Summary ready.") {
		t.Fatalf("expected summary frame, got %q", body)
	}
	if strings.Contains(body, "\"topic\":\"ticket.run.events\",\"type\":\"ticket.run.lifecycle\",\"payload\":{\"lifecycle\":{\"event_type\":\"agent.ready\",\"message\":\"ignore me\"") {
		t.Fatalf("did not expect other-ticket lifecycle event to be promoted onto the ticket run bus, got %q", body)
	}
}

func TestTicketRunStreamPayloadsMirrorDetailPayloadsForTheSameRun(t *testing.T) {
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
		SetIdentifier("ASE-434").
		SetTitle("Mirror ticket run payloads").
		SetDescription("Verify stream and detail payload parity").
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
	runID := uuid.New()
	runCreatedAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	runStartedAt := runCreatedAt.Add(30 * time.Second)
	runHeartbeatAt := runStartedAt.Add(2 * time.Minute)
	stepStatus := "running_tests"
	stepSummary := "Running contract parity checks."
	summaryStatus := domain.AgentRunCompletionSummaryStatusCompleted
	summaryMarkdown := "## Overview\n\nParity verified."
	summaryGeneratedAt := runHeartbeatAt.Add(30 * time.Second)
	summaryError := ""

	catalog.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: org.ID, Name: "Codex"}
	catalog.agents[agentID] = domain.Agent{ID: agentID, ProjectID: project.ID, ProviderID: providerID, Name: "Ticket Runner"}
	catalog.agentRuns[runID] = domain.AgentRun{
		ID:                           runID,
		AgentID:                      agentID,
		TicketID:                     ticketItem.ID,
		ProviderID:                   providerID,
		WorkflowID:                   uuid.New(),
		Status:                       domain.AgentRunStatusExecuting,
		RuntimeStartedAt:             &runStartedAt,
		LastHeartbeatAt:              &runHeartbeatAt,
		CurrentStepStatus:            &stepStatus,
		CurrentStepSummary:           &stepSummary,
		CompletionSummaryStatus:      &summaryStatus,
		CompletionSummaryMarkdown:    &summaryMarkdown,
		CompletionSummaryGeneratedAt: &summaryGeneratedAt,
		CompletionSummaryError:       &summaryError,
		CompletionSummaryJSON: map[string]any{
			"provider": "Codex",
		},
		CreatedAt: runCreatedAt,
	}

	traceAt := runStartedAt.Add(15 * time.Second)
	traceEntry := domain.AgentTraceEntry{
		ID:         uuid.New(),
		ProjectID:  project.ID,
		TicketID:   &ticketItem.ID,
		AgentID:    agentID,
		AgentRunID: runID,
		Sequence:   1,
		Provider:   "codex",
		Kind:       domain.AgentTraceKindAssistantDelta,
		Stream:     "assistant",
		Output:     "Planning the fix",
		Payload:    map[string]any{"item_id": "assistant-1"},
		CreatedAt:  traceAt,
	}
	stepEntry := domain.AgentStepEntry{
		ID:         uuid.New(),
		ProjectID:  project.ID,
		TicketID:   &ticketItem.ID,
		AgentID:    agentID,
		AgentRunID: runID,
		StepStatus: "planning",
		Summary:    "Inspecting transcript reducer behavior.",
		CreatedAt:  traceAt.Add(5 * time.Second),
	}
	catalog.traceEvents = []domain.AgentTraceEntry{traceEntry}
	catalog.stepEvents = []domain.AgentStepEntry{stepEntry}

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

	detailRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticketItem.ID.String()+"/runs/"+runID.String(),
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

	response, cancel := openSSERequest(
		t,
		testServer.URL+"/api/v1/projects/"+project.ID.String()+"/events/stream",
	)
	defer cancel()
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close ticket run stream response body: %v", err)
		}
	})
	reader := bufio.NewReader(response.Body)

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
				"created_at": runStartedAt.Format(time.RFC3339),
			},
		},
	)
	lifecycleFrame := readTicketRunSSEFrameByEvent(t, reader, "ticket.run.lifecycle")
	lifecycleData := decodeTicketRunSSEPayload(t, lifecycleFrame.Data)
	var lifecyclePayload struct {
		Run       ticketRunResponse               `json:"run"`
		Lifecycle ticketRunLifecycleEventResponse `json:"lifecycle"`
	}
	if err := json.Unmarshal(lifecycleData, &lifecyclePayload); err != nil {
		t.Fatalf("decode lifecycle frame: %v", err)
	}
	if !reflect.DeepEqual(lifecyclePayload.Run, detailPayload.Run) {
		t.Fatalf("lifecycle run snapshot = %+v, want %+v", lifecyclePayload.Run, detailPayload.Run)
	}

	publishTestEvent(
		t,
		bus,
		agentTraceStreamTopic,
		provider.MustParseEventType(domain.AgentOutputEventType),
		map[string]any{
			"entry": map[string]any{
				"id":           traceEntry.ID.String(),
				"project_id":   project.ID.String(),
				"ticket_id":    ticketItem.ID.String(),
				"agent_run_id": runID.String(),
				"sequence":     traceEntry.Sequence,
				"provider":     traceEntry.Provider,
				"kind":         traceEntry.Kind,
				"stream":       traceEntry.Stream,
				"output":       traceEntry.Output,
				"payload":      traceEntry.Payload,
				"created_at":   traceEntry.CreatedAt.UTC().Format(time.RFC3339),
			},
		},
	)
	traceFrame := readTicketRunSSEFrameByEvent(t, reader, "ticket.run.trace")
	traceData := decodeTicketRunSSEPayload(t, traceFrame.Data)
	var tracePayload struct {
		Entry ticketRunTraceEntryResponse `json:"entry"`
	}
	if err := json.Unmarshal(traceData, &tracePayload); err != nil {
		t.Fatalf("decode trace frame: %v", err)
	}
	if len(detailPayload.TraceEntries) != 1 {
		t.Fatalf("detail trace entry count = %d, want 1", len(detailPayload.TraceEntries))
	}
	if !reflect.DeepEqual(tracePayload.Entry, detailPayload.TraceEntries[0]) {
		t.Fatalf("trace stream entry = %+v, want %+v", tracePayload.Entry, detailPayload.TraceEntries[0])
	}

	publishTestEvent(
		t,
		bus,
		agentStepStreamTopic,
		provider.MustParseEventType(domain.AgentStepEventType),
		map[string]any{
			"entry": map[string]any{
				"id":                    stepEntry.ID.String(),
				"project_id":            project.ID.String(),
				"ticket_id":             ticketItem.ID.String(),
				"agent_run_id":          runID.String(),
				"step_status":           stepEntry.StepStatus,
				"summary":               stepEntry.Summary,
				"source_trace_event_id": nil,
				"created_at":            stepEntry.CreatedAt.UTC().Format(time.RFC3339),
			},
		},
	)
	stepFrame := readTicketRunSSEFrameByEvent(t, reader, "ticket.run.step")
	stepData := decodeTicketRunSSEPayload(t, stepFrame.Data)
	var stepPayload struct {
		Entry ticketRunStepEntryResponse `json:"entry"`
	}
	if err := json.Unmarshal(stepData, &stepPayload); err != nil {
		t.Fatalf("decode step frame: %v", err)
	}
	if len(detailPayload.StepEntries) != 1 {
		t.Fatalf("detail step entry count = %d, want 1", len(detailPayload.StepEntries))
	}
	if !reflect.DeepEqual(stepPayload.Entry, detailPayload.StepEntries[0]) {
		t.Fatalf("step stream entry = %+v, want %+v", stepPayload.Entry, detailPayload.StepEntries[0])
	}

	publishTestEvent(
		t,
		bus,
		ticketRunStreamTopic,
		ticketRunSummaryStreamType,
		map[string]any{
			"project_id": project.ID.String(),
			"ticket_id":  ticketItem.ID.String(),
			"run_id":     runID.String(),
			"completion_summary": map[string]any{
				"status":       detailPayload.Run.CompletionSummary.Status,
				"markdown":     detailPayload.Run.CompletionSummary.Markdown,
				"json":         detailPayload.Run.CompletionSummary.JSON,
				"generated_at": detailPayload.Run.CompletionSummary.GeneratedAt,
				"error":        detailPayload.Run.CompletionSummary.Error,
			},
		},
	)
	summaryFrame := readTicketRunSSEFrameByEvent(t, reader, "ticket.run.summary")
	summaryData := decodeTicketRunSSEPayload(t, summaryFrame.Data)
	var summaryPayload struct {
		ticketRunSummaryEnvelope
		Run *ticketRunResponse `json:"run"`
	}
	if err := json.Unmarshal(summaryData, &summaryPayload); err != nil {
		t.Fatalf("decode summary frame: %v", err)
	}
	if summaryPayload.RunID != detailPayload.Run.ID {
		t.Fatalf("summary run_id = %q, want %q", summaryPayload.RunID, detailPayload.Run.ID)
	}
	if summaryPayload.CompletionSummary == nil || detailPayload.Run.CompletionSummary == nil {
		t.Fatalf("expected non-nil completion summaries, got %+v %+v", summaryPayload.CompletionSummary, detailPayload.Run.CompletionSummary)
	}
	if !reflect.DeepEqual(summaryPayload.CompletionSummary, detailPayload.Run.CompletionSummary) {
		t.Fatalf("summary payload = %+v, want %+v", summaryPayload.CompletionSummary, detailPayload.Run.CompletionSummary)
	}
	if summaryPayload.Run == nil || !reflect.DeepEqual(*summaryPayload.Run, detailPayload.Run) {
		t.Fatalf("summary run payload = %+v, want %+v", summaryPayload.Run, detailPayload.Run)
	}
}

func readTicketRunSSEFrameByEvent(t *testing.T, reader *bufio.Reader, wantEvent string) projectConversationSSEFrame {
	t.Helper()

	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for SSE event %q", wantEvent)
		default:
		}
		frame := readProjectConversationSSEFrame(t, reader)
		if frame.Event == wantEvent {
			return frame
		}
	}
}

func decodeTicketRunSSEPayload(t *testing.T, raw string) json.RawMessage {
	t.Helper()

	var envelope struct {
		Topic   string          `json:"topic"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("decode ticket run SSE envelope: %v", err)
	}
	return envelope.Payload
}

func TestMapTicketRunResponseMapsTerminatedRunsToEndedWithoutCompletedAt(t *testing.T) {
	providerID := uuid.New()
	agentID := uuid.New()
	runID := uuid.New()
	terminalAt := time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC)
	completedRunID := uuid.New()

	catalog := ticketRunCatalog{
		attempts: map[uuid.UUID]int{
			runID:          1,
			completedRunID: 2,
		},
		agents: map[uuid.UUID]domain.Agent{
			agentID: {ID: agentID, Name: "Runner"},
		},
		providers: map[uuid.UUID]domain.AgentProvider{
			providerID: {
				ID:          providerID,
				Name:        "Codex",
				AdapterType: domain.AgentProviderAdapterTypeCodexAppServer,
				ModelName:   "gpt-5.4",
			},
		},
	}

	ended := mapTicketRunResponse(domain.AgentRun{
		ID:                       runID,
		AgentID:                  agentID,
		TicketID:                 uuid.New(),
		ProviderID:               providerID,
		WorkflowID:               uuid.New(),
		Status:                   domain.AgentRunStatusTerminated,
		InputTokens:              20,
		OutputTokens:             5,
		CachedInputTokens:        3,
		CacheCreationInputTokens: 2,
		ReasoningTokens:          1,
		PromptTokens:             18,
		CandidateTokens:          4,
		ToolTokens:               2,
		TotalTokens:              25,
		TerminalAt:               &terminalAt,
		CreatedAt:                terminalAt.Add(-10 * time.Minute),
	}, catalog)
	if ended.Status != "ended" {
		t.Fatalf("terminated run status = %q, want ended", ended.Status)
	}
	if ended.TerminalAt == nil || *ended.TerminalAt != terminalAt.Format(time.RFC3339) {
		t.Fatalf("terminated run terminal_at = %+v, want %s", ended.TerminalAt, terminalAt.Format(time.RFC3339))
	}
	if ended.CompletedAt != nil {
		t.Fatalf("terminated run completed_at = %+v, want nil", ended.CompletedAt)
	}
	if ended.AdapterType != "codex-app-server" || ended.ModelName != "gpt-5.4" || ended.Usage.Total != 25 {
		t.Fatalf("terminated run metadata/usage = %+v", ended)
	}

	completed := mapTicketRunResponse(domain.AgentRun{
		ID:         completedRunID,
		AgentID:    agentID,
		TicketID:   uuid.New(),
		ProviderID: providerID,
		WorkflowID: uuid.New(),
		Status:     domain.AgentRunStatusCompleted,
		TerminalAt: &terminalAt,
		CreatedAt:  terminalAt.Add(-20 * time.Minute),
	}, catalog)
	if completed.Status != "completed" {
		t.Fatalf("completed run status = %q, want completed", completed.Status)
	}
	if completed.CompletedAt == nil || *completed.CompletedAt != terminalAt.Format(time.RFC3339) {
		t.Fatalf("completed run completed_at = %+v, want %s", completed.CompletedAt, terminalAt.Format(time.RFC3339))
	}
}

func TestBuildTicketRunSummaryStreamEventFiltersTicketScope(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	ticketID := uuid.New()
	runID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()

	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{ID: projectID}
	catalog.tickets[ticketID] = fakeCatalogTicket{ID: ticketID, ProjectID: projectID}
	catalog.providers[providerID] = domain.AgentProvider{ID: providerID, Name: "Codex"}
	catalog.agents[agentID] = domain.Agent{ID: agentID, ProjectID: projectID, ProviderID: providerID, Name: "Ticket Runner"}
	catalog.agentRuns[runID] = domain.AgentRun{
		ID:         runID,
		AgentID:    agentID,
		TicketID:   ticketID,
		ProviderID: providerID,
		WorkflowID: uuid.New(),
		Status:     domain.AgentRunStatusCompleted,
		CreatedAt:  time.Now().UTC(),
	}

	server := &Server{catalog: catalogservice.SplitServices(catalog)}

	event, err := provider.NewJSONEvent(
		ticketRunStreamTopic,
		ticketRunSummaryStreamType,
		map[string]any{
			"project_id": projectID.String(),
			"ticket_id":  ticketID.String(),
			"run_id":     runID.String(),
			"completion_summary": map[string]any{
				"status":   "completed",
				"markdown": "Summary ready.",
			},
		},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	streamEvent, matched, err := server.buildTicketRunSummaryStreamEvent(ctx, projectID, ticketID, event)
	if err != nil {
		t.Fatalf("buildTicketRunSummaryStreamEvent returned error: %v", err)
	}
	if !matched {
		t.Fatal("expected summary event to match ticket scope")
	}
	if streamEvent.Topic != ticketRunStreamTopic || streamEvent.Type != ticketRunSummaryStreamType {
		t.Fatalf("unexpected summary stream event routing: topic=%s type=%s", streamEvent.Topic.String(), streamEvent.Type.String())
	}
	var streamPayload struct {
		Run               *ticketRunResponse                  `json:"run"`
		CompletionSummary *ticketRunCompletionSummaryResponse `json:"completion_summary"`
	}
	if err := json.Unmarshal(streamEvent.Payload, &streamPayload); err != nil {
		t.Fatalf("decode stream payload: %v", err)
	}
	if streamPayload.Run == nil || streamPayload.Run.ID != runID.String() {
		t.Fatalf("summary payload run = %+v, want run %s", streamPayload.Run, runID)
	}
	if streamPayload.CompletionSummary == nil || streamPayload.CompletionSummary.Status != "completed" {
		t.Fatalf("summary payload completion summary = %+v", streamPayload.CompletionSummary)
	}

	otherTicketEvent, err := provider.NewJSONEvent(
		ticketRunStreamTopic,
		ticketRunSummaryStreamType,
		map[string]any{
			"project_id": projectID.String(),
			"ticket_id":  uuid.NewString(),
			"run_id":     runID.String(),
		},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	_, matched, err = server.buildTicketRunSummaryStreamEvent(ctx, projectID, ticketID, otherTicketEvent)
	if err != nil {
		t.Fatalf("buildTicketRunSummaryStreamEvent returned error for mismatched ticket: %v", err)
	}
	if matched {
		t.Fatal("did not expect summary event for another ticket to match")
	}
}

func TestIsTicketRunLifecycleEventType(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "agent ready", raw: activityevent.TypeAgentReady.String(), want: true},
		{name: "agent executing", raw: activityevent.TypeAgentExecuting.String(), want: true},
		{name: "agent completed", raw: activityevent.TypeAgentCompleted.String(), want: true},
		{name: "agent heartbeat", raw: "agent.heartbeat", want: false},
		{name: "provider rate limit", raw: activityevent.TypeProviderRateLimitUpdated.String(), want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTicketRunLifecycleEventType(tc.raw); got != tc.want {
				t.Fatalf("isTicketRunLifecycleEventType(%q) = %t, want %t", tc.raw, got, tc.want)
			}
		})
	}
}
