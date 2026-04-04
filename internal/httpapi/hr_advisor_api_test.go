package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestHRAdvisorRouteReturnsRecommendationsAndActivationState(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, repoRoot)

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")

	fullstackRole, ok := builtin.RoleBySlug("fullstack-developer")
	if !ok {
		t.Fatal("expected builtin fullstack-developer role")
	}

	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-1").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	workflowResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"name":              "Fullstack Developer",
			"type":              "coding",
			"harness_path":      fullstackRole.HarnessPath,
			"harness_content":   fullstackRole.Content,
			"agent_id":          agentItem.ID.String(),
			"pickup_status_ids": []string{todoID.String()},
			"finish_status_ids": []string{doneID.String()},
		},
		http.StatusCreated,
		&workflowResp,
	)

	var activeTicketID uuid.UUID
	for index := 0; index < 4; index++ {
		ticketItem, err := client.Ticket.Create().
			SetProjectID(project.ID).
			SetIdentifier(fmt.Sprintf("ASE-%d", index+1)).
			SetTitle(fmt.Sprintf("Ticket %d", index+1)).
			SetStatusID(todoID).
			SetPriority(entticket.PriorityHigh).
			SetType(entticket.TypeFeature).
			SetWorkflowID(parseUUID(t, workflowResp.Workflow.ID)).
			SetCreatedBy("user:test").
			Save(ctx)
		if err != nil {
			t.Fatalf("create ticket %d: %v", index+1, err)
		}
		if index == 0 {
			activeTicketID = ticketItem.ID
		}
	}

	runItem, err := client.AgentRun.Create().
		SetAgentID(agentItem.ID).
		SetWorkflowID(parseUUID(t, workflowResp.Workflow.ID)).
		SetTicketID(activeTicketID).
		SetProviderID(provider.ID).
		SetStatus(entagentrun.StatusExecuting).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(activeTicketID).
		SetCurrentRunID(runItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind active run to ticket: %v", err)
	}

	resp := struct {
		ProjectID       string                            `json:"project_id"`
		Summary         hrAdvisorSummaryResponse          `json:"summary"`
		Staffing        hrAdvisorStaffingResponse         `json:"staffing"`
		Recommendations []hrAdvisorRecommendationResponse `json:"recommendations"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/hr-advisor", project.ID),
		nil,
		http.StatusOK,
		&resp,
	)

	if resp.ProjectID != project.ID.String() {
		t.Fatalf("expected project id %s, got %s", project.ID, resp.ProjectID)
	}
	if resp.Summary.OpenTickets != 4 || resp.Summary.CodingTickets != 4 || resp.Summary.ActiveAgents != 1 {
		t.Fatalf("unexpected summary: %+v", resp.Summary)
	}
	if len(resp.Recommendations) == 0 {
		t.Fatalf("expected at least one recommendation, got %+v", resp)
	}

	var qaRecommendation *hrAdvisorRecommendationResponse
	for index := range resp.Recommendations {
		recommendation := &resp.Recommendations[index]
		if recommendation.RoleSlug == "qa-engineer" {
			qaRecommendation = recommendation
		}
		if recommendation.RoleSlug == "fullstack-developer" {
			t.Fatalf("did not expect fullstack recommendation when role workflow and agent are active: %+v", resp.Recommendations)
		}
	}
	if qaRecommendation == nil {
		t.Fatalf("expected qa-engineer recommendation, got %+v", resp.Recommendations)
	}
	if qaRecommendation.Priority != "high" || !qaRecommendation.ActivationReady {
		t.Fatalf("unexpected qa recommendation payload: %+v", qaRecommendation)
	}
	if qaRecommendation.RoleName != "QA Engineer" || qaRecommendation.HarnessPath == "" || qaRecommendation.SuggestedWorkflowName == "" {
		t.Fatalf("expected builtin qa role metadata, got %+v", qaRecommendation)
	}
}

func TestHRAdvisorRouteReturnsDefaultRecommendationsForFreshProject(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("Planned").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)

	rec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/hr-advisor", project.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected hr advisor 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"recommendations":[`) {
		t.Fatalf("expected recommendations array in payload, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"active_workflow_families":[]`) {
		t.Fatalf("expected empty active_workflow_families array in payload, got %s", rec.Body.String())
	}

	resp := struct {
		Summary struct {
			ActiveWorkflowFamilies []string `json:"active_workflow_families"`
		} `json:"summary"`
		Recommendations []hrAdvisorRecommendationResponse `json:"recommendations"`
	}{}
	decodeResponse(t, rec, &resp)
	if len(resp.Summary.ActiveWorkflowFamilies) != 0 {
		t.Fatalf("expected non-nil empty active workflow families, got %+v", resp.Summary.ActiveWorkflowFamilies)
	}
	if len(resp.Recommendations) == 0 {
		t.Fatalf("expected non-nil recommendations slice, got %+v", resp.Recommendations)
	}
	if resp.Recommendations[0].Evidence == nil {
		t.Fatalf("expected recommendation evidence to be an array, got %+v", resp.Recommendations[0])
	}
}

func TestHRAdvisorRouteIncludesDocumentationDriftTrendEvidence(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)

	for index := 0; index < 4; index++ {
		if _, err := client.ActivityEvent.Create().
			SetProjectID(project.ID).
			SetEventType("pr.merged").
			SetMessage(fmt.Sprintf("Merged PR #%d without docs update", index+1)).
			Save(ctx); err != nil {
			t.Fatalf("create merged activity event %d: %v", index+1, err)
		}
	}

	resp := struct {
		Summary struct {
			RecentActivityCount int `json:"recent_activity_count"`
		} `json:"summary"`
		Recommendations []hrAdvisorRecommendationResponse `json:"recommendations"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/hr-advisor", project.ID),
		nil,
		http.StatusOK,
		&resp,
	)

	if resp.Summary.RecentActivityCount != 4 {
		t.Fatalf("expected recent activity count 4, got %+v", resp.Summary)
	}

	var writerRecommendation *hrAdvisorRecommendationResponse
	for index := range resp.Recommendations {
		recommendation := &resp.Recommendations[index]
		if recommendation.RoleSlug == "technical-writer" {
			writerRecommendation = recommendation
			break
		}
	}
	if writerRecommendation == nil {
		t.Fatalf("expected technical writer recommendation, got %+v", resp.Recommendations)
	}
	evidence := strings.Join(writerRecommendation.Evidence, " ")
	if !strings.Contains(evidence, "merge-like activity events: 4") || !strings.Contains(evidence, "documentation update events: 0") {
		t.Fatalf("expected documentation drift evidence, got %+v", writerRecommendation.Evidence)
	}
}

func TestHRAdvisorRouteIncludesDispatcherRecommendationFromBacklogPressure(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, repoRoot)

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")

	fullstackRole, ok := builtin.RoleBySlug("fullstack-developer")
	if !ok {
		t.Fatal("expected builtin fullstack-developer role")
	}

	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-1").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"name":              "Fullstack Developer",
			"type":              "coding",
			"harness_path":      fullstackRole.HarnessPath,
			"harness_content":   fullstackRole.Content,
			"agent_id":          agentItem.ID.String(),
			"pickup_status_ids": []string{todoID.String()},
			"finish_status_ids": []string{doneID.String()},
		},
		http.StatusCreated,
		nil,
	)

	for index := 0; index < 11; index++ {
		if _, err := client.Ticket.Create().
			SetProjectID(project.ID).
			SetIdentifier(fmt.Sprintf("ASE-%d", index+1)).
			SetTitle(fmt.Sprintf("Backlog %d", index+1)).
			SetStatusID(backlogID).
			SetPriority(entticket.PriorityHigh).
			SetType(entticket.TypeFeature).
			SetCreatedBy("user:test").
			Save(ctx); err != nil {
			t.Fatalf("create backlog ticket %d: %v", index+1, err)
		}
	}

	resp := struct {
		Recommendations []hrAdvisorRecommendationResponse `json:"recommendations"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/hr-advisor", project.ID),
		nil,
		http.StatusOK,
		&resp,
	)

	var dispatcherRecommendation *hrAdvisorRecommendationResponse
	for index := range resp.Recommendations {
		recommendation := &resp.Recommendations[index]
		if recommendation.RoleSlug == "dispatcher" {
			dispatcherRecommendation = recommendation
			break
		}
	}
	if dispatcherRecommendation == nil {
		t.Fatalf("expected dispatcher recommendation, got %+v", resp.Recommendations)
	}
	if dispatcherRecommendation.SuggestedWorkflowName != "Dispatcher" {
		t.Fatalf("expected dispatcher workflow suggestion, got %+v", dispatcherRecommendation)
	}
	if !strings.Contains(dispatcherRecommendation.Reason, "Backlog") {
		t.Fatalf("expected backlog-specific reason, got %+v", dispatcherRecommendation)
	}
	if !strings.Contains(strings.Join(dispatcherRecommendation.Evidence, " "), "pick up Backlog and finish into downstream non-backlog work statuses") {
		t.Fatalf("expected backlog lane evidence, got %+v", dispatcherRecommendation.Evidence)
	}
}

func TestActivateHRRecommendationRouteCreatesWorkflowAgentAndBootstrapTicket(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l4": map[string]any{
					"checked_at": time.Now().UTC().Format(time.RFC3339),
					"codex": map[string]any{
						"installed":   true,
						"auth_status": "logged_in",
						"auth_mode":   "login",
						"ready":       true,
					},
				},
			},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("OpenAI Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		SetDefaultAgentProviderID(provider.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
	attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, repoRoot)
	if _, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID); err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}

	resp := hrAdvisorActivationResponse{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/hr-advisor/activate", project.ID),
		map[string]any{
			"role_slug":               "qa-engineer",
			"create_bootstrap_ticket": true,
		},
		http.StatusCreated,
		&resp,
	)

	if resp.ProjectID != project.ID.String() || resp.RoleSlug != "qa-engineer" {
		t.Fatalf("unexpected activation response: %+v", resp)
	}
	if resp.Agent.ID == "" || resp.Workflow.ID == "" || resp.Workflow.AgentID == nil || *resp.Workflow.AgentID != resp.Agent.ID {
		t.Fatalf("expected workflow to bind created agent, got %+v", resp)
	}
	if resp.Workflow.HarnessPath != ".openase/harnesses/roles/qa-engineer.md" || !resp.Workflow.IsActive {
		t.Fatalf("unexpected workflow payload: %+v", resp.Workflow)
	}
	if resp.BootstrapTicket.Status != "created" || resp.BootstrapTicket.Ticket == nil || resp.BootstrapTicket.Ticket.WorkflowID == nil || *resp.BootstrapTicket.Ticket.WorkflowID != resp.Workflow.ID {
		t.Fatalf("unexpected bootstrap ticket payload: %+v", resp.BootstrapTicket)
	}

	agentCount, err := client.Agent.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count agents: %v", err)
	}
	if agentCount != 1 {
		t.Fatalf("expected one created agent, got %d", agentCount)
	}
	workflowCount, err := client.Workflow.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count workflows: %v", err)
	}
	if workflowCount != 1 {
		t.Fatalf("expected one created workflow, got %d", workflowCount)
	}
	ticketCount, err := client.Ticket.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if ticketCount != 1 {
		t.Fatalf("expected one created bootstrap ticket, got %d", ticketCount)
	}
}

func TestActivateHRRecommendationRouteMapsDispatcherToBacklogStageWhenNamesAreCustomized(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l4": map[string]any{
					"checked_at": time.Now().UTC().Format(time.RFC3339),
					"codex": map[string]any{
						"installed":   true,
						"auth_status": "logged_in",
						"auth_mode":   "login",
						"ready":       true,
					},
				},
			},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("OpenAI Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		SetDefaultAgentProviderID(provider.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
	attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, repoRoot)

	statusResult, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statusResult, "Backlog")
	todoID := findStatusIDByName(t, statusResult, "Todo")
	if _, err := client.TicketStatus.UpdateOneID(backlogID).
		SetName("Inbox").
		Save(ctx); err != nil {
		t.Fatalf("rename backlog status: %v", err)
	}

	resp := hrAdvisorActivationResponse{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/hr-advisor/activate", project.ID),
		map[string]any{
			"role_slug":               "dispatcher",
			"create_bootstrap_ticket": true,
		},
		http.StatusCreated,
		&resp,
	)

	if resp.RoleSlug != "dispatcher" || resp.Workflow.Name != "Dispatcher" {
		t.Fatalf("unexpected activation response: %+v", resp)
	}
	if strings.Join(resp.Workflow.PickupStatusIDs, ",") != backlogID.String() {
		t.Fatalf("expected dispatcher pickup to bind renamed backlog status %s, got %+v", backlogID, resp.Workflow)
	}
	if strings.Join(resp.Workflow.FinishStatusIDs, ",") != todoID.String() {
		t.Fatalf("expected dispatcher finish to bind downstream work status %s, got %+v", todoID, resp.Workflow)
	}
	if resp.BootstrapTicket.Ticket == nil || resp.BootstrapTicket.Ticket.StatusID != backlogID.String() || resp.BootstrapTicket.Ticket.StatusName != "Inbox" {
		t.Fatalf("expected bootstrap ticket to use renamed backlog lane, got %+v", resp.BootstrapTicket)
	}
}

func TestActivateHRRecommendationRouteReturnsConflictWhenNoProviderIsAvailable(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
	if _, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID); err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/hr-advisor/activate", project.ID),
		`{"role_slug":"qa-engineer"}`,
	)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"AGENT_PROVIDER_UNAVAILABLE"`) {
		t.Fatalf("expected provider unavailable code, got %s", rec.Body.String())
	}
}

func parseUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()

	parsed, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", raw, err)
	}
	return parsed
}
