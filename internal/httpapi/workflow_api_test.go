package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestWorkflowRoutesRoundTripExpandedPlatformAccessAllowed(t *testing.T) {
	client := openTestEntClient(t)
	serviceRepoRoot := createTestGitRepo(t)
	primaryRepoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), serviceRepoRoot)
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
		nil,
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-platform-access").
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-platform-access").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, primaryRepoRoot)
	attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, primaryRepoRoot)

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	allScopes := agentplatform.SupportedScopes()
	createResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"agent_id":                agent.ID.String(),
			"name":                    "Platform Access Workflow",
			"type":                    "coding",
			"pickup_status_ids":       []string{todoID.String()},
			"finish_status_ids":       []string{doneID.String()},
			"harness_content":         "# Platform Access\n",
			"platform_access_allowed": allScopes,
		},
		http.StatusCreated,
		&createResp,
	)
	if strings.Join(createResp.Workflow.PlatformAccessAllowed, ",") != strings.Join(allScopes, ",") {
		t.Fatalf("create platform_access_allowed = %v, want %v", createResp.Workflow.PlatformAccessAllowed, allScopes)
	}

	updatedScopes := []string{
		string(agentplatform.ScopeSkillsList),
		string(agentplatform.ScopeStatusesList),
		string(agentplatform.ScopeWorkflowsRead),
	}
	updateResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		map[string]any{
			"platform_access_allowed": updatedScopes,
		},
		http.StatusOK,
		&updateResp,
	)
	if strings.Join(updateResp.Workflow.PlatformAccessAllowed, ",") != strings.Join(updatedScopes, ",") {
		t.Fatalf("update platform_access_allowed = %v, want %v", updateResp.Workflow.PlatformAccessAllowed, updatedScopes)
	}
}

func TestWorkflowRoutesCRUDHarnessVersionsWithoutRepoSync(t *testing.T) {
	client := openTestEntClient(t)
	serviceRepoRoot := createTestGitRepo(t)
	primaryRepoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), serviceRepoRoot)
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
		nil,
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, primaryRepoRoot)

	statusSvc := newTicketStatusService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	activateMarkerPath := filepath.Join(serviceRepoRoot, "activate.marker")
	reloadMarkerPath := filepath.Join(serviceRepoRoot, "reload.marker")
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	createResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"agent_id":          agent.ID.String(),
			"name":              "Coding Workflow",
			"type":              "coding",
			"pickup_status_ids": []string{todoID.String()},
			"finish_status_ids": []string{doneID.String()},
			"harness_content":   "# Coding\n",
			"hooks": map[string]any{
				"workflow_hooks": map[string]any{
					"on_activate": []map[string]any{{
						"cmd": "printf '%s:%s' \"$OPENASE_WORKFLOW_NAME\" \"$OPENASE_WORKFLOW_VERSION\" > activate.marker",
					}},
					"on_reload": []map[string]any{{
						"cmd": "printf '%s:%s' \"$OPENASE_HOOK_NAME\" \"$OPENASE_WORKFLOW_VERSION\" > reload.marker",
					}},
				},
			},
		},
		http.StatusCreated,
		&createResp,
	)
	if createResp.Workflow.Type != "coding" {
		t.Fatalf("expected coding workflow, got %+v", createResp.Workflow)
	}
	if createResp.Workflow.AgentID == nil || *createResp.Workflow.AgentID != agent.ID.String() {
		t.Fatalf("expected bound agent %s, got %+v", agent.ID, createResp.Workflow.AgentID)
	}
	if createResp.Workflow.HarnessContent == nil || *createResp.Workflow.HarnessContent == "" {
		t.Fatalf("expected harness content in create response, got %+v", createResp.Workflow)
	}
	if createResp.Workflow.HarnessPath != ".openase/harnesses/coding-workflow.md" {
		t.Fatalf("expected default harness path under service repo root, got %q", createResp.Workflow.HarnessPath)
	}
	//nolint:gosec // test reads files from a controlled temp repository
	activateMarker, err := os.ReadFile(activateMarkerPath)
	if err != nil {
		t.Fatalf("read activate marker: %v", err)
	}
	if string(activateMarker) != "Coding Workflow:1" {
		t.Fatalf("expected activate marker to capture workflow context, got %q", string(activateMarker))
	}

	if _, err := os.Stat(filepath.Join(primaryRepoRoot, filepath.FromSlash(createResp.Workflow.HarnessPath))); !os.IsNotExist(err) {
		t.Fatalf("expected project repo checkout to stay clean, stat err=%v", err)
	}
	listResp := struct {
		Workflows []workflowResponse `json:"workflows"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(listResp.Workflows))
	}

	getResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		nil,
		http.StatusOK,
		&getResp,
	)
	if getResp.Workflow.HarnessContent == nil || *getResp.Workflow.HarnessContent != *createResp.Workflow.HarnessContent {
		t.Fatalf("expected workflow detail to include harness content, got %+v", getResp.Workflow)
	}

	patchResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		map[string]any{
			"name":           "Core Coding Workflow",
			"max_concurrent": 7,
		},
		http.StatusOK,
		&patchResp,
	)
	if patchResp.Workflow.Name != "Core Coding Workflow" || patchResp.Workflow.MaxConcurrent != 7 || !patchResp.Workflow.IsActive {
		t.Fatalf("unexpected patched workflow payload: %+v", patchResp.Workflow)
	}
	rejectRoleSlugUpdate := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		`{"role_slug":"dispatcher"}`,
	)
	if rejectRoleSlugUpdate.Code != http.StatusBadRequest {
		t.Fatalf("expected role_slug patch to fail with 400, got %d body=%s", rejectRoleSlugUpdate.Code, rejectRoleSlugUpdate.Body.String())
	}
	if !strings.Contains(rejectRoleSlugUpdate.Body.String(), "role_slug cannot be updated") {
		t.Fatalf("expected role_slug patch error message, got %s", rejectRoleSlugUpdate.Body.String())
	}

	legacyCreateRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		fmt.Sprintf(`{"agent_id":"%s","name":"Legacy Workflow","type":"coding","pickup_status_ids":["%s"],"required_machine_labels":["gpu"]}`, agent.ID, todoID),
	)
	if legacyCreateRec.Code != http.StatusBadRequest ||
		!strings.Contains(legacyCreateRec.Body.String(), "invalid JSON body") ||
		!strings.Contains(legacyCreateRec.Body.String(), "required_machine_labels") {
		t.Fatalf("expected legacy workflow machine labels to be rejected, got %d: %s", legacyCreateRec.Code, legacyCreateRec.Body.String())
	}

	legacyPatchRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		`{"required_machine_labels":["gpu"]}`,
	)
	if legacyPatchRec.Code != http.StatusBadRequest ||
		!strings.Contains(legacyPatchRec.Body.String(), "invalid JSON body") ||
		!strings.Contains(legacyPatchRec.Body.String(), "required_machine_labels") {
		t.Fatalf("expected legacy workflow patch machine labels to be rejected, got %d: %s", legacyPatchRec.Code, legacyPatchRec.Body.String())
	}

	harnessResp := struct {
		Harness harnessResponse `json:"harness"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPut,
		fmt.Sprintf("/api/v1/workflows/%s/harness", createResp.Workflow.ID),
		map[string]any{
			"content": "# Updated by API\n",
		},
		http.StatusOK,
		&harnessResp,
	)
	if harnessResp.Harness.Version != 2 {
		t.Fatalf("expected harness version 2 after API update, got %+v", harnessResp.Harness)
	}
	//nolint:gosec // test reads files from a controlled temp repository
	reloadMarker, err := os.ReadFile(reloadMarkerPath)
	if err != nil {
		t.Fatalf("read reload marker after API update: %v", err)
	}
	if string(reloadMarker) != "on_reload:2" {
		t.Fatalf("expected API reload marker to capture new version, got %q", string(reloadMarker))
	}

	invalidHarnessRec := performJSONRequest(
		t,
		server,
		http.MethodPut,
		fmt.Sprintf("/api/v1/workflows/%s/harness", createResp.Workflow.ID),
		`{"content":"{{"}`,
	)
	if invalidHarnessRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid harness update to fail with 400, got %d body=%s", invalidHarnessRec.Code, invalidHarnessRec.Body.String())
	}

	getAfterInvalidResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		nil,
		http.StatusOK,
		&getAfterInvalidResp,
	)
	if getAfterInvalidResp.Workflow.Version != 2 {
		t.Fatalf("expected invalid harness update to keep version 2, got %+v", getAfterInvalidResp.Workflow)
	}

	harnessGetResp := struct {
		Harness harnessResponse `json:"harness"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/workflows/%s/harness", createResp.Workflow.ID),
		nil,
		http.StatusOK,
		&harnessGetResp,
	)
	if harnessGetResp.Harness.Version != 2 || !strings.Contains(harnessGetResp.Harness.Content, "# Updated by API") {
		t.Fatalf("expected harness GET to read current DB version, got %+v", harnessGetResp.Harness)
	}

	historyResp := struct {
		History []workflowVersionResponse `json:"history"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/workflows/%s/harness/history", createResp.Workflow.ID),
		nil,
		http.StatusOK,
		&historyResp,
	)
	if len(historyResp.History) != 2 || historyResp.History[0].Version != 2 || historyResp.History[1].Version != 1 {
		t.Fatalf("expected workflow harness history to expose published versions, got %+v", historyResp.History)
	}

	deleteResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		nil,
		http.StatusOK,
		&deleteResp,
	)
}

func TestWorkflowRoutesPersistExplicitAuditActor(t *testing.T) {
	client := openTestEntClient(t)
	serviceRepoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), serviceRepoRoot)
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
		nil,
		ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client)),
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client)).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	auditActor := "user:browser-user via project-conversation:" + uuid.NewString()
	createResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"agent_id":          agent.ID.String(),
			"name":              "Coding Workflow",
			"type":              "coding",
			"pickup_status_ids": []string{todoID.String()},
			"finish_status_ids": []string{doneID.String()},
			"harness_content":   "# Coding\n",
			"created_by":        auditActor,
		},
		http.StatusCreated,
		&createResp,
	)

	historyResp := workflowHistoryResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/workflows/%s/harness/history", createResp.Workflow.ID),
		nil,
		http.StatusOK,
		&historyResp,
	)
	if len(historyResp.History) != 1 || historyResp.History[0].CreatedBy != auditActor {
		t.Fatalf("unexpected workflow history after create: %+v", historyResp.History)
	}

	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/workflows/%s", createResp.Workflow.ID),
		map[string]any{
			"name":      "Renamed Workflow",
			"edited_by": auditActor,
		},
		http.StatusOK,
		&struct {
			Workflow workflowResponse `json:"workflow"`
		}{},
	)

	executeJSON(
		t,
		server,
		http.MethodPut,
		fmt.Sprintf("/api/v1/workflows/%s/harness", createResp.Workflow.ID),
		map[string]any{
			"content":   "# Updated\n",
			"edited_by": auditActor,
		},
		http.StatusOK,
		&struct {
			Harness harnessResponse `json:"harness"`
		}{},
	)
	historyResp = workflowHistoryResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/workflows/%s/harness/history", createResp.Workflow.ID),
		nil,
		http.StatusOK,
		&historyResp,
	)
	if len(historyResp.History) != 2 || historyResp.History[0].CreatedBy != auditActor {
		t.Fatalf("unexpected workflow history after harness update: %+v", historyResp.History)
	}
}

func TestWorkflowRoutesAllowFinishStatusInStartedStage(t *testing.T) {
	client := openTestEntClient(t)
	serviceRepoRoot := createTestGitRepo(t)
	primaryRepoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), serviceRepoRoot)
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
		nil,
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-finish-status").
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-finish-status").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, project.ID, primaryRepoRoot)

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	inReviewID := findStatusIDByName(t, statuses, "In Review")

	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	createResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"agent_id":          agent.ID.String(),
			"name":              "Review Workflow",
			"type":              "coding",
			"pickup_status_ids": []string{todoID.String()},
			"finish_status_ids": []string{inReviewID.String()},
			"harness_content":   "# Coding\n",
		},
		http.StatusCreated,
		&createResp,
	)

	if len(createResp.Workflow.FinishStatusIDs) != 1 || createResp.Workflow.FinishStatusIDs[0] != inReviewID.String() {
		t.Fatalf("expected finish status to keep In Review binding, got %+v", createResp.Workflow.FinishStatusIDs)
	}
}

func TestWorkflowCreateDoesNotRequireProjectRepo(t *testing.T) {
	client := openTestEntClient(t)
	serviceRepoRoot := createTestGitRepo(t)
	primaryRepoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), serviceRepoRoot)
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
		nil,
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}

	projectWithoutRepo, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Repo Less").
		SetSlug("repo-less").
		Save(ctx)
	if err != nil {
		t.Fatalf("create projectWithoutRepo: %v", err)
	}
	projectReady, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Ready Project").
		SetSlug("ready-project").
		Save(ctx)
	if err != nil {
		t.Fatalf("create projectReady: %v", err)
	}
	createPrimaryProjectRepo(ctx, t, client, projectReady.ID, primaryRepoRoot)

	repoLessStatuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, projectWithoutRepo.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses for projectWithoutRepo: %v", err)
	}
	repoLessTodoID := findStatusIDByName(t, repoLessStatuses, "Todo")
	repoLessDoneID := findStatusIDByName(t, repoLessStatuses, "Done")

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, projectReady.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses for projectReady: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(projectReady.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	repoLessAgent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(projectWithoutRepo.ID).
		SetName("codex-coding-repo-less").
		Save(ctx)
	if err != nil {
		t.Fatalf("create repoLessAgent: %v", err)
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", projectWithoutRepo.ID),
		fmt.Sprintf(`{"agent_id":"%s","name":"Coding Workflow","type":"coding","pickup_status_ids":["%s"],"finish_status_ids":["%s"],"harness_content":"# Coding\n"}`, repoLessAgent.ID, repoLessTodoID, repoLessDoneID),
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create workflow without bound repo status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var createPayload struct {
		Workflow workflowResponse `json:"workflow"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode workflow create payload: %v", err)
	}
	if createPayload.Workflow.Name != "Coding Workflow" ||
		createPayload.Workflow.Version != 1 ||
		createPayload.Workflow.HarnessContent == nil ||
		*createPayload.Workflow.HarnessContent == "" {
		t.Fatalf("unexpected workflow create payload without bound repo: %+v", createPayload.Workflow)
	}
	var workflowCreate struct {
		Workflow workflowResponse `json:"workflow"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", projectReady.ID),
		map[string]any{
			"agent_id":          agent.ID.String(),
			"name":              "Coding Workflow",
			"type":              "coding",
			"pickup_status_ids": []string{todoID.String()},
			"finish_status_ids": []string{doneID.String()},
			"harness_content":   "# Coding\n",
		},
		http.StatusCreated,
		&workflowCreate,
	)
	if workflowCreate.Workflow.Name != "Coding Workflow" {
		t.Fatalf("workflow create = %+v", workflowCreate.Workflow)
	}
}

func TestValidateHarnessRoute(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	validRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/harness/validate",
		`{"content":"# Coding\n"}`,
	)
	if validRec.Code != http.StatusOK {
		t.Fatalf("expected validate success, got %d body=%s", validRec.Code, validRec.Body.String())
	}
	var validResp harnessValidationResponse
	if err := json.Unmarshal(validRec.Body.Bytes(), &validResp); err != nil {
		t.Fatalf("decode valid response: %v", err)
	}
	if !validResp.Valid || len(validResp.Issues) != 0 {
		t.Fatalf("expected valid harness response, got %+v", validResp)
	}

	invalidRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/harness/validate",
		`{"content":"---\nworkflow:\n  role: coding\n---\n"}`,
	)
	if invalidRec.Code != http.StatusOK {
		t.Fatalf("expected validate response, got %d body=%s", invalidRec.Code, invalidRec.Body.String())
	}
	var invalidResp harnessValidationResponse
	if err := json.Unmarshal(invalidRec.Body.Bytes(), &invalidResp); err != nil {
		t.Fatalf("decode invalid response: %v", err)
	}
	if invalidResp.Valid {
		t.Fatalf("expected invalid harness response, got %+v", invalidResp)
	}
	if len(invalidResp.Issues) == 0 || invalidResp.Issues[0].Level != "error" {
		t.Fatalf("expected validation issues, got %+v", invalidResp)
	}

	templateInvalidRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/harness/validate",
		`{"content":"{% if ticket.title %}\nmissing endif\n"}`,
	)
	if templateInvalidRec.Code != http.StatusOK {
		t.Fatalf("expected template validate response, got %d body=%s", templateInvalidRec.Code, templateInvalidRec.Body.String())
	}
	var templateInvalidResp harnessValidationResponse
	if err := json.Unmarshal(templateInvalidRec.Body.Bytes(), &templateInvalidResp); err != nil {
		t.Fatalf("decode template invalid response: %v", err)
	}
	if templateInvalidResp.Valid {
		t.Fatalf("expected invalid gonja template response, got %+v", templateInvalidResp)
	}
	if len(templateInvalidResp.Issues) == 0 || templateInvalidResp.Issues[0].Level != "error" {
		t.Fatalf("expected gonja validation issues, got %+v", templateInvalidResp)
	}
	if !strings.Contains(templateInvalidResp.Issues[0].Message, "endif") {
		t.Fatalf("expected gonja validation message to mention endif, got %+v", templateInvalidResp.Issues[0])
	}
}

func TestHarnessVariablesRoute(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/harness/variables",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected variables route success, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response harnessVariablesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode harness variables response: %v", err)
	}
	if len(response.Groups) == 0 {
		t.Fatalf("expected harness variable groups, got %+v", response)
	}

	foundTicket := false
	foundMarkdownEscape := false
	for _, group := range response.Groups {
		if group.Name == "ticket" {
			foundTicket = true
		}
		for _, variable := range group.Variables {
			if variable.Path == "markdown_escape" {
				foundMarkdownEscape = true
			}
		}
	}

	if !foundTicket {
		t.Fatalf("expected ticket variable group, got %+v", response.Groups)
	}
	if !foundMarkdownEscape {
		t.Fatalf("expected markdown_escape filter metadata, got %+v", response.Groups)
	}
}

func TestWorkflowRoutesErrorMappingsAndInvalidInputs(t *testing.T) {
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

	serverWithService := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		workflowSvc,
	)
	serverWithoutService := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	for _, testCase := range []struct {
		name       string
		server     *Server
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "list unavailable",
			server:     serverWithoutService,
			method:     http.MethodGet,
			target:     fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "SERVICE_UNAVAILABLE",
		},
		{
			name:       "list invalid project id",
			server:     serverWithService,
			method:     http.MethodGet,
			target:     "/api/v1/projects/not-a-uuid/workflows",
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_PROJECT_ID",
		},
		{
			name:       "list project not found",
			server:     serverWithService,
			method:     http.MethodGet,
			target:     "/api/v1/projects/00000000-0000-0000-0000-000000000000/workflows",
			wantStatus: http.StatusNotFound,
			wantBody:   "PROJECT_NOT_FOUND",
		},
		{
			name:       "get invalid workflow id",
			server:     serverWithService,
			method:     http.MethodGet,
			target:     "/api/v1/workflows/not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_WORKFLOW_ID",
		},
		{
			name:       "get workflow not found",
			server:     serverWithService,
			method:     http.MethodGet,
			target:     "/api/v1/workflows/00000000-0000-0000-0000-000000000000",
			wantStatus: http.StatusNotFound,
			wantBody:   "WORKFLOW_NOT_FOUND",
		},
		{
			name:       "delete invalid workflow id",
			server:     serverWithService,
			method:     http.MethodDelete,
			target:     "/api/v1/workflows/not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_WORKFLOW_ID",
		},
		{
			name:       "delete workflow not found",
			server:     serverWithService,
			method:     http.MethodDelete,
			target:     "/api/v1/workflows/00000000-0000-0000-0000-000000000000",
			wantStatus: http.StatusNotFound,
			wantBody:   "WORKFLOW_NOT_FOUND",
		},
		{
			name:       "get harness invalid workflow id",
			server:     serverWithService,
			method:     http.MethodGet,
			target:     "/api/v1/workflows/not-a-uuid/harness",
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_WORKFLOW_ID",
		},
		{
			name:       "get harness workflow not found",
			server:     serverWithService,
			method:     http.MethodGet,
			target:     "/api/v1/workflows/00000000-0000-0000-0000-000000000000/harness",
			wantStatus: http.StatusNotFound,
			wantBody:   "WORKFLOW_NOT_FOUND",
		},
		{
			name:       "update harness invalid workflow id",
			server:     serverWithService,
			method:     http.MethodPut,
			target:     "/api/v1/workflows/not-a-uuid/harness",
			body:       `{"content":"# Coding\n"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_WORKFLOW_ID",
		},
		{
			name:       "update harness empty content",
			server:     serverWithService,
			method:     http.MethodPut,
			target:     "/api/v1/workflows/00000000-0000-0000-0000-000000000000/harness",
			body:       `{"content":"   "}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "content must not be empty",
		},
		{
			name:       "validate harness invalid json",
			server:     serverWithService,
			method:     http.MethodPost,
			target:     "/api/v1/harness/validate",
			body:       `{"content":`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid JSON body",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, testCase.server, testCase.method, testCase.target, testCase.body)
			if rec.Code != testCase.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", testCase.wantStatus, rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("expected body %q to contain %q", rec.Body.String(), testCase.wantBody)
			}
		})
	}
}

func TestBuildHarnessTemplateDataAndRenderBody(t *testing.T) {
	ctx := context.Background()
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
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
	backendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname("backend").
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	attachProjectRepoCheckout(ctx, t, client, backendRepo.ID, localMachine.ID, repoRoot)

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	todoID := findStatusIDByName(t, statuses, "Todo")
	inReviewID := findStatusIDByName(t, statuses, "In Review")
	doneID := findStatusIDByName(t, statuses, "Done")
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Claude Code").
		SetAdapterType(entagentprovider.AdapterTypeClaudeCodeCli).
		SetCliCommand("claude").
		SetModelName("claude-sonnet-4-6").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("claude-01").
		SetTotalTicketsCompleted(47).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	templateContent := `Implement product changes end to end.

Ticket {{ ticket.identifier }} {{ ticket.title | markdown_escape }}
Status {{ ticket.status }} parent={{ ticket.parent_identifier }} attempts={{ attempt }}/{{ max_attempts }}
Links {{ ticket.links | length }} {{ ticket.links[0].type }} {{ ticket.links[0].relation }}
Deps {% for dep in ticket.dependencies %}{{ dep.identifier }}:{{ dep.type }}:{{ dep.status }}{% endfor %}
Repos {% for repo in repos %}{{ repo.name }}@{{ repo.branch }} labels={{ repo.labels | join(",") }} path={{ repo.path }}{% endfor %}
All {{ all_repos | map(attribute="name") | join(",") }}
Agent {{ agent.name }} {{ agent.provider }} {{ agent.adapter_type }} {{ agent.model }} {{ agent.total_tickets_completed }}
Machine {{ machine.name }} {{ accessible_machines[0].ssh_user }}
Workflow {{ workflow.name }} {{ workflow.type }} {{ workflow.role_name }} {{ workflow.pickup_status }} {{ workflow.finish_status }}
ProjectWorkflows {% for wf in project.workflows %}{{ wf.role_name }}:{{ wf.pickup_status }}:{{ wf.current_active }}/{{ wf.max_concurrent }}:{{ wf.role_description }}|{% endfor %}
WorkflowArtifacts {% for wf in project.workflows %}{{ wf.role_name }}={{ wf.finish_status }}:{{ wf.harness_path }}:{{ wf.skills | join(",") }}|{% endfor %}
WorkflowBindings {% for wf in project.workflows %}{{ wf.role_name }}=pickup[{% for status in wf.pickup_statuses %}{{ status.name }}:{{ status.stage }}|{% endfor %}]finish[{% for status in wf.finish_statuses %}{{ status.name }}:{{ status.stage }}|{% endfor %}];{% endfor %}
WorkflowHistory {% for wf in project.workflows %}{{ wf.role_name }}={% for recent in wf.recent_tickets %}{{ recent.identifier }}:{{ recent.status }}:{{ recent.retry_paused }}:{{ recent.consecutive_errors }}|{% endfor %};{% endfor %}
ProjectStatuses {% for status in project.statuses %}{{ status.name }}:{{ status.stage }}:{{ status.color }}|{% endfor %}
ProjectMachines {% for machine in project.machines %}{{ machine.name }}:{{ machine.status }}:{{ machine.resources.transport | default("none") }}:{{ machine.labels | join(",") }}|{% endfor %}
Platform {{ platform.api_url }} {{ platform.project_id }} {{ platform.ticket_id }}
Timestamp {{ timestamp }} Version {{ openase_version }} URL {{ ticket.url }}
{% if attempt > 1 %}retry{% endif %}
`

	createdWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		AgentID:             agent.ID,
		Name:                "Coding Workflow",
		Type:                "coding",
		RoleSlug:            "fullstack-developer",
		RoleName:            "fullstack-developer",
		RoleDescription:     "Implement product changes end to end.",
		HarnessContent:      templateContent,
		Hooks:               map[string]any{},
		MaxConcurrent:       3,
		MaxRetryAttempts:    3,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     workflowservice.MustStatusBindingSet(todoID),
		FinishStatusIDs:     workflowservice.MustStatusBindingSet(doneID),
	})
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if _, err := workflowSvc.BindSkills(ctx, workflowservice.UpdateWorkflowSkillsInput{
		WorkflowID: createdWorkflow.ID,
		Skills:     []string{"openase-platform", "commit"},
	}); err != nil {
		t.Fatalf("bind workflow skills: %v", err)
	}
	if _, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		AgentID:             agent.ID,
		Name:                "Dispatcher Workflow",
		Type:                "custom",
		RoleSlug:            "dispatcher",
		RoleName:            "dispatcher",
		RoleDescription:     "Evaluate backlog tickets and route them to the right workflow.",
		HarnessContent:      "Evaluate backlog tickets and route them to the right workflow.\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      5,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     workflowservice.MustStatusBindingSet(backlogID),
		FinishStatusIDs:     workflowservice.MustStatusBindingSet(todoID),
	}); err != nil {
		t.Fatalf("create dispatcher workflow: %v", err)
	}

	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/acme/frontend").
		SetDefaultBranch("develop").
		Save(ctx)
	if err != nil {
		t.Fatalf("create frontend repo: %v", err)
	}

	activeTicket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-40").
		SetTitle("Implement auth boundary parsing").
		SetStatusID(todoID).
		SetPriority(entticket.PriorityMedium).
		SetType(entticket.TypeFeature).
		SetCreatedBy("user:gary").
		SetWorkflowID(createdWorkflow.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active workflow ticket: %v", err)
	}
	activeRun, err := client.AgentRun.Create().
		SetAgentID(agent.ID).
		SetWorkflowID(createdWorkflow.ID).
		SetTicketID(activeTicket.ID).
		SetProviderID(provider.ID).
		SetStatus(entagentrun.StatusExecuting).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active workflow run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(activeTicket.ID).
		SetCurrentRunID(activeRun.ID).
		Save(ctx); err != nil {
		t.Fatalf("attach active workflow run: %v", err)
	}
	if _, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-41").
		SetTitle("Tighten auth retry guidance").
		SetStatusID(inReviewID).
		SetPriority(entticket.PriorityHigh).
		SetType(entticket.TypeBugfix).
		SetCreatedBy("user:gary").
		SetWorkflowID(createdWorkflow.ID).
		SetAttemptCount(3).
		SetConsecutiveErrors(2).
		SetRetryPaused(true).
		SetPauseReason("needs_human_review").
		SetStartedAt(time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC)).
		Save(ctx); err != nil {
		t.Fatalf("create paused workflow history ticket: %v", err)
	}

	parentTicket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-30").
		SetTitle("Parent ticket").
		SetStatusID(doneID).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:gary").
		Save(ctx)
	if err != nil {
		t.Fatalf("create parent ticket: %v", err)
	}
	dependencyTarget, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-31").
		SetTitle("Dependency ticket").
		SetStatusID(doneID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:gary").
		Save(ctx)
	if err != nil {
		t.Fatalf("create dependency target: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-42").
		SetTitle("Escape * markdown").
		SetDescription("Render the harness body").
		SetStatusID(todoID).
		SetPriority(entticket.PriorityHigh).
		SetType(entticket.TypeBugfix).
		SetCreatedBy("user:gary").
		SetParentTicketID(parentTicket.ID).
		SetAttemptCount(2).
		SetBudgetUsd(5.0).
		SetExternalRef("PacificStudio/openase#20").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(frontendRepo.ID).
		SetBranchName("agent/claude-01/ASE-42").
		Save(ctx); err != nil {
		t.Fatalf("create ticket repo scope: %v", err)
	}
	if _, err := client.TicketExternalLink.Create().
		SetTicketID(ticketItem.ID).
		SetLinkType("github_issue").
		SetURL("https://github.com/acme/backend/issues/42").
		SetExternalID("42").
		SetTitle("Login validation broken on Safari").
		SetStatus("open").
		SetRelation("resolves").
		Save(ctx); err != nil {
		t.Fatalf("create external link: %v", err)
	}
	if _, err := client.TicketDependency.Create().
		SetSourceTicketID(ticketItem.ID).
		SetTargetTicketID(dependencyTarget.ID).
		SetType(entticketdependency.TypeBlocks).
		Save(ctx); err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	data, err := workflowSvc.BuildHarnessTemplateData(ctx, workflowservice.BuildHarnessTemplateDataInput{
		WorkflowID:     createdWorkflow.ID,
		TicketID:       ticketItem.ID,
		AgentID:        &agent.ID,
		Workspace:      "/workspaces/ASE-42",
		Timestamp:      time.Date(2026, 3, 20, 10, 30, 0, 0, time.UTC),
		OpenASEVersion: "0.3.1",
		TicketURL:      "http://localhost:19836/tickets/ASE-42",
		Platform: workflowservice.HarnessPlatformData{
			APIURL:     "http://localhost:19836/api/v1",
			AgentToken: "ase_agent_token",
		},
		Machine: workflowservice.HarnessMachineData{
			Name:        "gpu-01",
			Host:        "10.0.1.10",
			Description: "NVIDIA A100 x4",
			Labels:      []string{"gpu", "a100"},
			Resources: map[string]any{
				"transport": "ssh",
			},
			WorkspaceRoot: "/workspaces",
		},
		AccessibleMachines: []workflowservice.HarnessAccessibleMachineData{{
			Name:        "storage",
			Host:        "10.0.1.20",
			Description: "Artifact storage",
			Labels:      []string{"storage", "nfs"},
			Resources: map[string]any{
				"transport": "ssh",
			},
			SSHUser: "openase",
		}},
	})
	if err != nil {
		t.Fatalf("build harness template data: %v", err)
	}

	rendered, err := workflowservice.RenderHarnessBody(templateContent, data)
	if err != nil {
		t.Fatalf("render harness body: %v", err)
	}

	for _, want := range []string{
		`Ticket ASE-42 Escape \* markdown`,
		"parent=ASE-30 attempts=2/3",
		"Links 1 github_issue resolves",
		"ASE-31:blocks:Done",
		"frontend@agent/claude-01/ASE-42 labels= path=/workspaces/ASE-42/frontend",
		"All backend,frontend",
		"Agent claude-01 Claude Code claude-code-cli claude-sonnet-4-6 47",
		"Machine gpu-01 openase",
		"Workflow Coding Workflow coding fullstack-developer Todo Done",
		"ProjectWorkflows fullstack-developer:Todo:1/3:Implement product changes end to end.|dispatcher:Backlog:0/1:Evaluate backlog tickets and route them to the right workflow.|",
		"WorkflowArtifacts fullstack-developer=Done:.openase/harnesses/coding-workflow.md:commit,openase-platform|dispatcher=Todo:.openase/harnesses/dispatcher-workflow.md:|",
		"WorkflowBindings fullstack-developer=pickup[Todo:unstarted|]finish[Done:completed|];dispatcher=pickup[Backlog:backlog|]finish[Todo:unstarted|];",
		"WorkflowHistory fullstack-developer=ASE-41:In Review:True:2|ASE-40:Todo:False:0|;dispatcher=;",
		"ProjectStatuses Backlog:backlog:#6B7280|Todo:unstarted:#3B82F6|In Progress:started:#F59E0B|In Review:started:#8B5CF6|Done:completed:#10B981|Cancelled:canceled:#4B5563|",
		"ProjectMachines gpu-01:current:ssh:gpu,a100|storage:accessible:ssh:storage,nfs|",
		fmt.Sprintf("Platform http://localhost:19836/api/v1 %s %s", project.ID, ticketItem.ID),
		"Timestamp 2026-03-20T10:30:00Z Version 0.3.1 URL http://localhost:19836/tickets/ASE-42",
		"retry",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered harness to contain %q, got:\n%s", want, rendered)
		}
	}
}

func TestWorkflowCreateAndUpdateRoutesRejectInvalidPayloads(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)
	primaryRepoRoot := createTestGitRepo(t)

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
		nil,
		newTicketStatusService(client),
		nil,
		nil,
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-invalid-workflows").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-invalid-workflows").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
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
	createPrimaryProjectRepoForCheckout(ctx, t, client, project.ID, localMachine.ID, primaryRepoRoot)
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	inReviewID := findStatusIDByName(t, statuses, "In Review")
	inProgressID := findStatusIDByName(t, statuses, "In Progress")
	doneID := findStatusIDByName(t, statuses, "Done")
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	created := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"agent_id":          agent.ID.String(),
			"name":              "Coding Workflow",
			"type":              "coding",
			"pickup_status_ids": []string{todoID.String()},
			"finish_status_ids": []string{doneID.String()},
			"harness_content":   "# Coding\n",
		},
		http.StatusCreated,
		&created,
	)

	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "create invalid request", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID), body: `{"agent_id":"` + agent.ID.String() + `","name":" ","type":"coding","pickup_status_ids":["` + todoID.String() + `"],"finish_status_ids":["` + doneID.String() + `"]}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "create missing agent", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID), body: `{"agent_id":"` + uuid.New().String() + `","name":"Missing Agent","type":"coding","pickup_status_ids":["` + todoID.String() + `"],"finish_status_ids":["` + doneID.String() + `"]}`, wantStatus: http.StatusBadRequest, wantBody: "AGENT_NOT_FOUND"},
		{name: "create missing status", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID), body: `{"agent_id":"` + agent.ID.String() + `","name":"Missing Status","type":"coding","pickup_status_ids":["` + uuid.New().String() + `"],"finish_status_ids":["` + doneID.String() + `"]}`, wantStatus: http.StatusBadRequest, wantBody: "STATUS_NOT_FOUND"},
		{name: "create duplicate pickup status", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID), body: `{"agent_id":"` + agent.ID.String() + `","name":"Parallel Coding","type":"coding","pickup_status_ids":["` + todoID.String() + `"],"finish_status_ids":["` + doneID.String() + `"]}`, wantStatus: http.StatusConflict, wantBody: "PICKUP_STATUS_CONFLICT"},
		{name: "create overlapping status bindings", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID), body: `{"agent_id":"` + agent.ID.String() + `","name":"Looping Workflow","type":"coding","pickup_status_ids":["` + inReviewID.String() + `"],"finish_status_ids":["` + inReviewID.String() + `"]}`, wantStatus: http.StatusConflict, wantBody: "WORKFLOW_STATUS_BINDING_OVERLAP"},
		{name: "update invalid request", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/workflows/%s", created.Workflow.ID), body: `{"max_concurrent":-1}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update missing workflow", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/workflows/%s", uuid.New()), body: `{"name":"missing"}`, wantStatus: http.StatusNotFound, wantBody: "WORKFLOW_NOT_FOUND"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, testCase.method, testCase.target, testCase.body)
			if rec.Code != testCase.wantStatus || !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("%s %s = %d %s, want %d containing %q", testCase.method, testCase.target, rec.Code, rec.Body.String(), testCase.wantStatus, testCase.wantBody)
			}
		})
	}

	secondCreated := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"agent_id":          agent.ID.String(),
			"name":              "Dispatcher Workflow",
			"type":              "coding",
			"pickup_status_ids": []string{backlogID.String()},
			"finish_status_ids": []string{doneID.String()},
			"harness_content":   "# Dispatcher\n",
		},
		http.StatusCreated,
		&secondCreated,
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/workflows/%s", secondCreated.Workflow.ID),
		`{"pickup_status_ids":["`+todoID.String()+`"]}`,
	)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "PICKUP_STATUS_CONFLICT") {
		t.Fatalf("PATCH duplicate pickup status = %d %s, want %d containing %q", rec.Code, rec.Body.String(), http.StatusConflict, "PICKUP_STATUS_CONFLICT")
	}

	overlapRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/workflows/%s", created.Workflow.ID),
		`{"finish_status_ids":["`+todoID.String()+`"]}`,
	)
	if overlapRec.Code != http.StatusConflict || !strings.Contains(overlapRec.Body.String(), "WORKFLOW_STATUS_BINDING_OVERLAP") {
		t.Fatalf("PATCH overlapping status bindings = %d %s, want %d containing %q", overlapRec.Code, overlapRec.Body.String(), http.StatusConflict, "WORKFLOW_STATUS_BINDING_OVERLAP")
	}

	duplicateNameRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		`{"agent_id":"`+agent.ID.String()+`","name":"Coding Workflow","type":"coding","pickup_status_ids":["`+backlogID.String()+`"],"finish_status_ids":["`+doneID.String()+`"]}`,
	)
	assertAPIErrorResponse(
		t,
		duplicateNameRec,
		http.StatusConflict,
		"WORKFLOW_NAME_CONFLICT",
		`workflow name "Coding Workflow" is already used in this project`,
	)

	duplicateHarnessPathRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		`{"agent_id":"`+agent.ID.String()+`","name":"Conflicting Harness Workflow","type":"coding","harness_path":"`+created.Workflow.HarnessPath+`","pickup_status_ids":["`+inProgressID.String()+`"],"finish_status_ids":["`+doneID.String()+`"]}`,
	)
	assertAPIErrorResponse(
		t,
		duplicateHarnessPathRec,
		http.StatusConflict,
		"WORKFLOW_HARNESS_PATH_CONFLICT",
		`harness_path "`+created.Workflow.HarnessPath+`" is already used by another workflow`,
	)
}

func TestListWorkflowsRouteReturnsEmptyArrayForNewProject(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("create git marker: %v", err)
	}

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
		nil,
		newTicketStatusService(client),
		nil,
		nil,
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	rec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workflow list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"workflows":[]`) {
		t.Fatalf("expected empty workflows array in payload, got %s", rec.Body.String())
	}

	var payload struct {
		Workflows []workflowResponse `json:"workflows"`
	}
	decodeResponse(t, rec, &payload)
	if payload.Workflows == nil || len(payload.Workflows) != 0 {
		t.Fatalf("expected non-nil empty workflows slice, got %+v", payload.Workflows)
	}
}
