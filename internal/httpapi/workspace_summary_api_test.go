package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func TestWorkspaceSummaryRouteReturnsEmptyWorkspace(t *testing.T) {
	client := openTestEntClient(t)
	server := newWorkspaceSummaryTestServer(client)

	payload := struct {
		Workspace     workspaceDashboardMetricsResponse      `json:"workspace"`
		Organizations []workspaceOrganizationSummaryResponse `json:"organizations"`
	}{}
	executeJSON(t, server, http.MethodGet, "/api/v1/workspace/summary", nil, http.StatusOK, &payload)

	if payload.Workspace.OrganizationCount != 0 || payload.Workspace.ProjectCount != 0 || payload.Workspace.ProviderCount != 0 {
		t.Fatalf("expected empty workspace counts, got %+v", payload.Workspace)
	}
	if len(payload.Organizations) != 0 {
		t.Fatalf("expected no organization summaries, got %+v", payload.Organizations)
	}
}

func TestWorkspaceSummaryRouteReturnsAggregates(t *testing.T) {
	client := openTestEntClient(t)
	server := newWorkspaceSummaryTestServer(client)
	ctx := context.Background()

	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)

	orgA, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		SetStatus(entorganization.StatusActive).
		Save(ctx)
	if err != nil {
		t.Fatalf("create org A: %v", err)
	}
	orgB, err := client.Organization.Create().
		SetName("Bravo").
		SetSlug("bravo").
		SetStatus(entorganization.StatusActive).
		Save(ctx)
	if err != nil {
		t.Fatalf("create org B: %v", err)
	}
	if _, err := client.Organization.Create().
		SetName("Archive Me").
		SetSlug("archive-me").
		SetStatus(entorganization.StatusArchived).
		Save(ctx); err != nil {
		t.Fatalf("create archived org: %v", err)
	}

	machineA, err := client.Machine.Create().
		SetOrganizationID(orgA.ID).
		SetName("acme-host").
		SetHost("acme-host").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org A machine: %v", err)
	}
	machineB, err := client.Machine.Create().
		SetOrganizationID(orgB.ID).
		SetName("bravo-host").
		SetHost("bravo-host").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org B machine: %v", err)
	}

	providerA1, err := client.AgentProvider.Create().
		SetOrganizationID(orgA.ID).
		SetMachineID(machineA.ID).
		SetName("Codex A").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider A1: %v", err)
	}
	if _, err := client.AgentProvider.Create().
		SetOrganizationID(orgA.ID).
		SetMachineID(machineA.ID).
		SetName("Codex B").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx); err != nil {
		t.Fatalf("create provider A2: %v", err)
	}
	if _, err := client.AgentProvider.Create().
		SetOrganizationID(orgB.ID).
		SetMachineID(machineB.ID).
		SetName("Gemini").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("gemini").
		SetModelName("gemini-2.5-pro").
		Save(ctx); err != nil {
		t.Fatalf("create provider B1: %v", err)
	}

	projectAlpha, err := client.Project.Create().
		SetOrganizationID(orgA.ID).
		SetName("Alpha").
		SetSlug("alpha").
		SetDescription("Primary project").
		SetStatus("active").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project alpha: %v", err)
	}
	projectBeta, err := client.Project.Create().
		SetOrganizationID(orgA.ID).
		SetName("Beta").
		SetSlug("beta").
		SetDescription("Archived project").
		SetStatus("Archived").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project beta: %v", err)
	}
	if _, err := client.Project.Create().
		SetOrganizationID(orgB.ID).
		SetName("Gamma").
		SetSlug("gamma").
		SetDescription("Quiet project").
		SetStatus("active").
		Save(ctx); err != nil {
		t.Fatalf("create project gamma: %v", err)
	}

	alphaStatuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, projectAlpha.ID)
	if err != nil {
		t.Fatalf("reset alpha statuses: %v", err)
	}
	alphaTodoID := findStatusIDByName(t, alphaStatuses, "Todo")
	alphaDoneID := findStatusIDByName(t, alphaStatuses, "Done")
	alphaWorkflow, err := client.Workflow.Create().
		SetProjectID(projectAlpha.ID).
		SetName("alpha-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(alphaTodoID).
		AddFinishStatusIDs(alphaDoneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create alpha workflow: %v", err)
	}

	betaStatuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, projectBeta.ID)
	if err != nil {
		t.Fatalf("reset beta statuses: %v", err)
	}
	betaBacklogID := findStatusIDByName(t, betaStatuses, "Backlog")
	betaDoneID := findStatusIDByName(t, betaStatuses, "Done")
	betaWorkflow, err := client.Workflow.Create().
		SetProjectID(projectBeta.ID).
		SetName("beta-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(betaBacklogID).
		AddFinishStatusIDs(betaDoneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create beta workflow: %v", err)
	}

	agentA1, err := client.Agent.Create().
		SetProviderID(providerA1.ID).
		SetProjectID(projectAlpha.ID).
		SetName("alpha-runner").
		SetTotalTokensUsed(1200).
		Save(ctx)
	if err != nil {
		t.Fatalf("create alpha runner: %v", err)
	}
	if _, err := client.Agent.Create().
		SetProviderID(providerA1.ID).
		SetProjectID(projectAlpha.ID).
		SetName("alpha-idle").
		SetTotalTokensUsed(800).
		Save(ctx); err != nil {
		t.Fatalf("create alpha idle: %v", err)
	}
	if _, err := client.Agent.Create().
		SetProviderID(providerA1.ID).
		SetProjectID(projectBeta.ID).
		SetName("beta-idle").
		SetTotalTokensUsed(50).
		Save(ctx); err != nil {
		t.Fatalf("create beta idle: %v", err)
	}

	activeTicket, err := client.Ticket.Create().
		SetProjectID(projectAlpha.ID).
		SetIdentifier("ASE-101").
		SetTitle("Active ticket").
		SetStatusID(alphaTodoID).
		SetPriority(entticket.PriorityHigh).
		SetType(entticket.TypeFeature).
		SetCreatedBy("user:test").
		SetCostAmount(5.00).
		SetCreatedAt(now.Add(-30 * time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active ticket: %v", err)
	}
	completedToday, err := client.Ticket.Create().
		SetProjectID(projectAlpha.ID).
		SetIdentifier("ASE-102").
		SetTitle("Completed today").
		SetStatusID(alphaDoneID).
		SetPriority(entticket.PriorityMedium).
		SetType(entticket.TypeChore).
		SetCreatedBy("user:test").
		SetCostAmount(9.75).
		SetCreatedAt(now.Add(-1 * time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create completed ticket: %v", err)
	}
	if _, err := client.Ticket.Create().
		SetProjectID(projectBeta.ID).
		SetIdentifier("ASE-201").
		SetTitle("Legacy backlog").
		SetStatusID(betaBacklogID).
		SetPriority(entticket.PriorityLow).
		SetType(entticket.TypeChore).
		SetCreatedBy("user:test").
		SetCostAmount(2.50).
		SetCreatedAt(yesterday).
		Save(ctx); err != nil {
		t.Fatalf("create beta backlog ticket: %v", err)
	}

	runItem, err := client.AgentRun.Create().
		SetAgentID(agentA1.ID).
		SetWorkflowID(alphaWorkflow.ID).
		SetTicketID(activeTicket.ID).
		SetProviderID(providerA1.ID).
		SetStatus(entagentrun.StatusExecuting).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(activeTicket.ID).
		SetCurrentRunID(runItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind active run: %v", err)
	}

	payload := struct {
		Workspace     workspaceDashboardMetricsResponse      `json:"workspace"`
		Organizations []workspaceOrganizationSummaryResponse `json:"organizations"`
	}{}
	executeJSON(t, server, http.MethodGet, "/api/v1/workspace/summary", nil, http.StatusOK, &payload)

	if payload.Workspace.OrganizationCount != 2 || payload.Workspace.ProjectCount != 3 || payload.Workspace.ProviderCount != 3 {
		t.Fatalf("unexpected workspace inventory counts: %+v", payload.Workspace)
	}
	if payload.Workspace.RunningAgents != 1 || payload.Workspace.ActiveTickets != 2 {
		t.Fatalf("unexpected workspace activity counts: %+v", payload.Workspace)
	}
	if payload.Workspace.TodayCost != 14.75 || payload.Workspace.TotalTokens != 2050 {
		t.Fatalf("unexpected workspace usage totals: %+v", payload.Workspace)
	}
	if len(payload.Organizations) != 2 {
		t.Fatalf("expected 2 organization cards, got %+v", payload.Organizations)
	}

	acme := payload.Organizations[0]
	if acme.OrganizationID != orgA.ID.String() || acme.Name != "Acme" || acme.ProjectCount != 2 || acme.ProviderCount != 2 {
		t.Fatalf("unexpected Acme summary: %+v", acme)
	}
	if acme.RunningAgents != 1 || acme.ActiveTickets != 2 || acme.TodayCost != 14.75 || acme.TotalTokens != 2050 {
		t.Fatalf("unexpected Acme usage summary: %+v", acme)
	}

	bravo := payload.Organizations[1]
	if bravo.OrganizationID != orgB.ID.String() || bravo.ProjectCount != 1 || bravo.ProviderCount != 1 {
		t.Fatalf("unexpected Bravo summary: %+v", bravo)
	}
	if bravo.RunningAgents != 0 || bravo.ActiveTickets != 0 || bravo.TodayCost != 0 || bravo.TotalTokens != 0 {
		t.Fatalf("expected Bravo to stay empty, got %+v", bravo)
	}

	orgPayload := struct {
		Organization organizationDashboardMetricsResponse `json:"organization"`
		Projects     []organizationProjectSummaryResponse `json:"projects"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/orgs/"+orgA.ID.String()+"/summary",
		nil,
		http.StatusOK,
		&orgPayload,
	)

	if orgPayload.Organization.OrganizationID != orgA.ID.String() {
		t.Fatalf("expected org summary for %s, got %+v", orgA.ID, orgPayload.Organization)
	}
	if orgPayload.Organization.ProjectCount != 2 || orgPayload.Organization.ActiveProjectCount != 1 || orgPayload.Organization.ProviderCount != 2 {
		t.Fatalf("unexpected org inventory summary: %+v", orgPayload.Organization)
	}
	if orgPayload.Organization.RunningAgents != 1 || orgPayload.Organization.ActiveTickets != 2 || orgPayload.Organization.TodayCost != 14.75 || orgPayload.Organization.TotalTokens != 2050 {
		t.Fatalf("unexpected org usage summary: %+v", orgPayload.Organization)
	}
	if len(orgPayload.Projects) != 2 {
		t.Fatalf("expected 2 project summaries, got %+v", orgPayload.Projects)
	}
	if orgPayload.Projects[0].ProjectID != projectAlpha.ID.String() || orgPayload.Projects[0].RunningAgents != 1 || orgPayload.Projects[0].ActiveTickets != 1 || orgPayload.Projects[0].TodayCost != 14.75 || orgPayload.Projects[0].TotalTokens != 2000 || orgPayload.Projects[0].LastActivityAt == nil {
		t.Fatalf("unexpected alpha project summary: %+v", orgPayload.Projects[0])
	}
	if orgPayload.Projects[1].ProjectID != projectBeta.ID.String() || orgPayload.Projects[1].Status != "Archived" || orgPayload.Projects[1].RunningAgents != 0 || orgPayload.Projects[1].ActiveTickets != 1 || orgPayload.Projects[1].TodayCost != 0 || orgPayload.Projects[1].TotalTokens != 50 || orgPayload.Projects[1].LastActivityAt == nil {
		t.Fatalf("unexpected beta project summary: %+v", orgPayload.Projects[1])
	}

	_ = completedToday
	_ = betaWorkflow
}

func TestOrganizationSummaryRouteRejectsInvalidAndMissingIDs(t *testing.T) {
	client := openTestEntClient(t)
	server := newWorkspaceSummaryTestServer(client)

	invalidRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/not-a-uuid/summary", "")
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid org id to return 400, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}

	missingRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/"+uuid.NewString()+"/summary", "")
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("expected missing org to return 404, got %d: %s", missingRec.Code, missingRec.Body.String())
	}
}

func newWorkspaceSummaryTestServer(client *ent.Client) *Server {
	return NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)
}
