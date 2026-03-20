package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestHRAdvisorRouteReturnsRecommendationsAndActivationState(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("create git marker: %v", err)
	}

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver()),
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
		SetStatus("active").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")

	fullstackRole, ok := builtin.RoleBySlug("fullstack-developer")
	if !ok {
		t.Fatal("expected builtin fullstack-developer role")
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
			"name":             "Fullstack Developer",
			"type":             "coding",
			"harness_path":     fullstackRole.HarnessPath,
			"harness_content":  fullstackRole.Content,
			"pickup_status_id": todoID.String(),
			"finish_status_id": doneID.String(),
		},
		http.StatusCreated,
		&workflowResp,
	)

	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}
	if _, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-1").
		SetStatus(entagent.StatusRunning).
		Save(ctx); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	for index := 0; index < 4; index++ {
		if _, err := client.Ticket.Create().
			SetProjectID(project.ID).
			SetIdentifier(fmt.Sprintf("ASE-%d", index+1)).
			SetTitle(fmt.Sprintf("Ticket %d", index+1)).
			SetStatusID(todoID).
			SetPriority(entticket.PriorityHigh).
			SetType(entticket.TypeFeature).
			SetWorkflowID(parseUUID(t, workflowResp.Workflow.ID)).
			SetCreatedBy("user:test").
			Save(ctx); err != nil {
			t.Fatalf("create ticket %d: %v", index+1, err)
		}
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
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("create git marker: %v", err)
	}

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver()),
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
		SetStatus("planning").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	rec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/hr-advisor", project.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected hr advisor 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"recommendations":[`) {
		t.Fatalf("expected recommendations array in payload, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"active_workflow_types":[]`) {
		t.Fatalf("expected empty active_workflow_types array in payload, got %s", rec.Body.String())
	}

	resp := struct {
		Summary struct {
			ActiveWorkflowTypes []string `json:"active_workflow_types"`
		} `json:"summary"`
		Recommendations []hrAdvisorRecommendationResponse `json:"recommendations"`
	}{}
	decodeResponse(t, rec, &resp)
	if len(resp.Summary.ActiveWorkflowTypes) != 0 {
		t.Fatalf("expected non-nil empty active workflow types, got %+v", resp.Summary.ActiveWorkflowTypes)
	}
	if len(resp.Recommendations) == 0 {
		t.Fatalf("expected non-nil recommendations slice, got %+v", resp.Recommendations)
	}
	if resp.Recommendations[0].Evidence == nil {
		t.Fatalf("expected recommendation evidence to be an array, got %+v", resp.Recommendations[0])
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
