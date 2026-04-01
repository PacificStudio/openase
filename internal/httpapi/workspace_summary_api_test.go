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
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
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

	currentUTC := time.Now().UTC()
	now := time.Date(currentUTC.Year(), currentUTC.Month(), currentUTC.Day(), 12, 0, 0, 0, time.UTC)
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
		SetCreatedAt(yesterday).
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
	if _, err := client.ActivityEvent.Create().
		SetProjectID(projectAlpha.ID).
		SetTicketID(activeTicket.ID).
		SetEventType(ticketing.CostRecordedEventType).
		SetMetadata(map[string]any{
			"cost_usd":    5.00,
			"cost_source": ticketing.UsageCostSourceEstimated.String(),
		}).
		SetCreatedAt(now.Add(-20 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create active ticket cost event: %v", err)
	}
	if _, err := client.ActivityEvent.Create().
		SetProjectID(projectAlpha.ID).
		SetTicketID(completedToday.ID).
		SetEventType(ticketing.CostRecordedEventType).
		SetMetadata(map[string]any{
			"cost_usd":    9.75,
			"cost_source": ticketing.UsageCostSourceActual.String(),
		}).
		SetCreatedAt(now.Add(-10 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create completed ticket cost event: %v", err)
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

func TestOrganizationTokenUsageRouteMaterializesBackfillsAndAvoidsDoubleCount(t *testing.T) {
	client := openTestEntClient(t)
	server := newWorkspaceSummaryTestServer(client)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		SetStatus(entorganization.StatusActive).
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("acme-host").
		SetHost("acme-host").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Alpha").
		SetSlug("alpha").
		SetDescription("Analytics test").
		SetStatus("active").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("Coding").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("runner").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	makeTicket := func(identifier string) *ent.Ticket {
		t.Helper()
		item, createErr := client.Ticket.Create().
			SetProjectID(project.ID).
			SetIdentifier(identifier).
			SetTitle(identifier).
			SetStatusID(todoID).
			SetPriority(entticket.PriorityMedium).
			SetType(entticket.TypeFeature).
			SetCreatedBy("user:test").
			Save(ctx)
		if createErr != nil {
			t.Fatalf("create ticket %s: %v", identifier, createErr)
		}
		return item
	}

	dayOneEarly := time.Date(2026, 3, 29, 0, 5, 0, 0, time.UTC)
	dayOneLate := time.Date(2026, 3, 29, 23, 59, 0, 0, time.UTC)
	dayTwoEarly := time.Date(2026, 3, 30, 0, 1, 0, 0, time.UTC)
	dayThree := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	runOne, err := client.AgentRun.Create().
		SetAgentID(agentItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(makeTicket("ASE-201").ID).
		SetProviderID(providerItem.ID).
		SetStatus(entagentrun.StatusCompleted).
		SetTerminalAt(dayOneEarly).
		SetSnapshotMaterializedAt(dayOneEarly.Add(1 * time.Minute)).
		SetInputTokens(10).
		SetOutputTokens(5).
		SetCachedInputTokens(2).
		SetReasoningTokens(1).
		SetTotalTokens(15).
		Save(ctx)
	if err != nil {
		t.Fatalf("create run one: %v", err)
	}
	if _, err := client.OrganizationDailyTokenUsage.Create().
		SetOrganizationID(org.ID).
		SetUsageDate(dayOneEarly).
		SetInputTokens(runOne.InputTokens).
		SetOutputTokens(runOne.OutputTokens).
		SetCachedInputTokens(runOne.CachedInputTokens).
		SetReasoningTokens(runOne.ReasoningTokens).
		SetTotalTokens(runOne.TotalTokens).
		SetFinalizedRunCount(1).
		SetRecomputedAt(dayOneEarly.Add(2 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("seed daily usage row: %v", err)
	}

	runTwo, err := client.AgentRun.Create().
		SetAgentID(agentItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(makeTicket("ASE-202").ID).
		SetProviderID(providerItem.ID).
		SetStatus(entagentrun.StatusErrored).
		SetTerminalAt(dayOneLate).
		SetInputTokens(7).
		SetOutputTokens(3).
		SetCachedInputTokens(4).
		SetReasoningTokens(2).
		SetTotalTokens(10).
		Save(ctx)
	if err != nil {
		t.Fatalf("create run two: %v", err)
	}

	runThree, err := client.AgentRun.Create().
		SetAgentID(agentItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(makeTicket("ASE-203").ID).
		SetProviderID(providerItem.ID).
		SetStatus(entagentrun.StatusTerminated).
		SetTerminalAt(dayTwoEarly).
		SetInputTokens(4).
		SetOutputTokens(1).
		SetCachedInputTokens(1).
		SetReasoningTokens(1).
		SetTotalTokens(5).
		Save(ctx)
	if err != nil {
		t.Fatalf("create run three: %v", err)
	}

	type tokenUsagePayload struct {
		Days    []organizationTokenUsageDayResponse   `json:"days"`
		Summary organizationTokenUsageSummaryResponse `json:"summary"`
	}

	firstPayload := tokenUsagePayload{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/orgs/"+org.ID.String()+"/token-usage?from=2026-03-29&to=2026-03-31",
		nil,
		http.StatusOK,
		&firstPayload,
	)

	if len(firstPayload.Days) != 3 {
		t.Fatalf("expected 3 token-usage days, got %+v", firstPayload.Days)
	}
	if firstPayload.Days[0].Date != "2026-03-29" || firstPayload.Days[0].TotalTokens != 25 || firstPayload.Days[0].FinalizedRunCount != 2 {
		t.Fatalf("unexpected 2026-03-29 usage: %+v", firstPayload.Days[0])
	}
	if firstPayload.Days[1].Date != "2026-03-30" || firstPayload.Days[1].TotalTokens != 5 || firstPayload.Days[1].FinalizedRunCount != 1 {
		t.Fatalf("unexpected 2026-03-30 usage: %+v", firstPayload.Days[1])
	}
	if firstPayload.Days[2].Date != "2026-03-31" || firstPayload.Days[2].TotalTokens != 0 || firstPayload.Days[2].FinalizedRunCount != 0 {
		t.Fatalf("unexpected 2026-03-31 usage: %+v", firstPayload.Days[2])
	}
	if firstPayload.Summary.TotalTokens != 30 || firstPayload.Summary.AvgDailyTokens != 10 {
		t.Fatalf("unexpected usage summary: %+v", firstPayload.Summary)
	}
	if firstPayload.Summary.PeakDay == nil || firstPayload.Summary.PeakDay.Date != "2026-03-29" || firstPayload.Summary.PeakDay.TotalTokens != 25 {
		t.Fatalf("unexpected peak day: %+v", firstPayload.Summary.PeakDay)
	}

	secondPayload := tokenUsagePayload{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/orgs/"+org.ID.String()+"/token-usage?from=2026-03-29&to=2026-03-31",
		nil,
		http.StatusOK,
		&secondPayload,
	)
	if secondPayload.Days[0].TotalTokens != 25 || secondPayload.Days[0].FinalizedRunCount != 2 {
		t.Fatalf("expected repeated query to avoid double-counting, got %+v", secondPayload.Days[0])
	}

	rows, err := client.OrganizationDailyTokenUsage.Query().All(ctx)
	if err != nil {
		t.Fatalf("query daily usage rows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 persisted daily usage rows, got %+v", rows)
	}
	if row := findUsageRowByDate(t, rows, dayOneEarly); row.TotalTokens != 25 || row.FinalizedRunCount != 2 || row.SourceMode.String() != "materialized" {
		t.Fatalf("unexpected materialized day one row: %+v", row)
	}
	if row := findUsageRowByDate(t, rows, dayTwoEarly); row.TotalTokens != 5 || row.FinalizedRunCount != 1 || row.SourceMode.String() != "materialized" {
		t.Fatalf("unexpected materialized day two row: %+v", row)
	}
	if row := findUsageRowByDate(t, rows, dayThree); row.TotalTokens != 0 || row.FinalizedRunCount != 0 || row.SourceMode.String() != "lazy_backfill" {
		t.Fatalf("unexpected lazy backfill day three row: %+v", row)
	}

	runTwoAfter, err := client.AgentRun.Get(ctx, runTwo.ID)
	if err != nil {
		t.Fatalf("reload run two: %v", err)
	}
	if runTwoAfter.SnapshotMaterializedAt == nil {
		t.Fatalf("expected run two snapshot to be marked materialized, got %+v", runTwoAfter)
	}
	runThreeAfter, err := client.AgentRun.Get(ctx, runThree.ID)
	if err != nil {
		t.Fatalf("reload run three: %v", err)
	}
	if runThreeAfter.SnapshotMaterializedAt == nil {
		t.Fatalf("expected run three snapshot to be marked materialized, got %+v", runThreeAfter)
	}
}

func findUsageRowByDate(t *testing.T, rows []*ent.OrganizationDailyTokenUsage, day time.Time) *ent.OrganizationDailyTokenUsage {
	t.Helper()

	want := day.UTC().Format("2006-01-02")
	for _, row := range rows {
		if row != nil && row.UsageDate.UTC().Format("2006-01-02") == want {
			return row
		}
	}

	t.Fatalf("daily usage row for %s not found in %+v", want, rows)
	return nil
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
