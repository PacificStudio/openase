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
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
)

func TestWorkflowRoutesCRUDHarnessStorageAndHotReload(t *testing.T) {
	client := openTestEntClient(t)
	serviceRepoRoot := createTestGitRepo(t)
	primaryRepoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), serviceRepoRoot)
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
		ticketstatus.NewService(client),
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

	statusSvc := ticketstatus.NewService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	activateMarkerPath := filepath.Join(primaryRepoRoot, "activate.marker")
	reloadMarkerPath := filepath.Join(primaryRepoRoot, "reload.marker")
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
			"harness_content":   "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
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
		t.Fatalf("expected default harness path under primary repo root, got %q", createResp.Workflow.HarnessPath)
	}
	//nolint:gosec // test reads files from a controlled temp repository
	activateMarker, err := os.ReadFile(activateMarkerPath)
	if err != nil {
		t.Fatalf("read activate marker: %v", err)
	}
	if string(activateMarker) != "Coding Workflow:1" {
		t.Fatalf("expected activate marker to capture workflow context, got %q", string(activateMarker))
	}

	harnessAbsPath := filepath.Join(primaryRepoRoot, filepath.FromSlash(createResp.Workflow.HarnessPath))
	//nolint:gosec // test reads files from a controlled temp repository
	fileContent, err := os.ReadFile(harnessAbsPath)
	if err != nil {
		t.Fatalf("read created harness file: %v", err)
	}
	if string(fileContent) != *createResp.Workflow.HarnessContent {
		t.Fatalf("expected harness file content to match response, got %q", string(fileContent))
	}
	if _, err := os.Stat(filepath.Join(serviceRepoRoot, filepath.FromSlash(createResp.Workflow.HarnessPath))); !os.IsNotExist(err) {
		t.Fatalf("expected service checkout to stay clean, stat err=%v", err)
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
			"content": "---\nworkflow:\n  role: coding\n---\n\n# Updated by API\n",
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
		`{"content":"---\nworkflow:\n  name: broken\nstatus:\n  pickup: [Todo\n---\n"}`,
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

	externalContent := "---\nworkflow:\n  role: coding\n---\n\n# Updated on disk\n"
	if err := os.WriteFile(harnessAbsPath, []byte(externalContent), 0o600); err != nil {
		t.Fatalf("write external harness change: %v", err)
	}

	waitForWorkflowVersion(t, server, createResp.Workflow.ID, 3)
	//nolint:gosec // test reads files from a controlled temp repository
	reloadMarker, err = os.ReadFile(reloadMarkerPath)
	if err != nil {
		t.Fatalf("read reload marker after external update: %v", err)
	}
	if string(reloadMarker) != "on_reload:3" {
		t.Fatalf("expected external reload marker to capture new version, got %q", string(reloadMarker))
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
	if harnessGetResp.Harness.Content != externalContent {
		t.Fatalf("expected harness GET to see external reload, got %q", harnessGetResp.Harness.Content)
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
	if _, err := os.Stat(harnessAbsPath); !os.IsNotExist(err) {
		t.Fatalf("expected harness file to be removed, stat err=%v", err)
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
		`{"content":"---\nworkflow:\n  name: coding\nstatus:\n  pickup: Todo\n---\n\n# Coding\n"}`,
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
		`{"content":"---\nworkflow:\n  name: broken\nstatus:\n  pickup: [Todo\n---\n"}`,
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
		`{"content":"---\nworkflow:\n  name: coding\nstatus:\n  pickup: Todo\n---\n\n{% if ticket.title %}\nmissing endif\n"}`,
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

func TestBuildHarnessTemplateDataAndRenderBody(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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
		SetStatus("active").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL(repoRoot).
		SetDefaultBranch("main").
		SetIsPrimary(true).
		Save(ctx); err != nil {
		t.Fatalf("create backend repo: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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

	templateContent := `---
workflow:
  role_name: "fullstack-developer"
status:
  pickup: "Todo"
  finish: "Done"
skills:
  - openase-platform
  - commit
---
Implement product changes end to end.

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
WorkflowHistory {% for wf in project.workflows %}{{ wf.role_name }}={% for recent in wf.recent_tickets %}{{ recent.identifier }}:{{ recent.status }}:{{ recent.retry_paused }}:{{ recent.consecutive_errors }}|{% endfor %};{% endfor %}
ProjectStatuses {{ project.statuses | map(attribute="name") | join(",") }} first={{ project.statuses[0].color }}
ProjectMachines {% for machine in project.machines %}{{ machine.name }}:{{ machine.status }}:{{ machine.labels | join(",") }}|{% endfor %}
Platform {{ platform.api_url }} {{ platform.project_id }} {{ platform.ticket_id }}
Timestamp {{ timestamp }} Version {{ openase_version }} URL {{ ticket.url }}
{% if attempt > 1 %}retry{% endif %}
`

	createdWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		AgentID:             agent.ID,
		Name:                "Coding Workflow",
		Type:                "coding",
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
	if _, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		AgentID:             agent.ID,
		Name:                "Dispatcher Workflow",
		Type:                "custom",
		HarnessContent:      "---\nworkflow:\n  role: dispatcher\nstatus:\n  pickup: \"Backlog\"\n  finish: \"Backlog\"\n---\n\nEvaluate backlog tickets and route them to the right workflow.\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      5,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     workflowservice.MustStatusBindingSet(backlogID),
		FinishStatusIDs:     workflowservice.MustStatusBindingSet(backlogID),
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
		SetExternalRef("BetterAndBetterII/openase#20").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(frontendRepo.ID).
		SetBranchName("agent/claude-01/ASE-42").
		SetIsPrimaryScope(true).
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
			Name:          "gpu-01",
			Host:          "10.0.1.10",
			Description:   "NVIDIA A100 x4",
			Labels:        []string{"gpu", "a100"},
			WorkspaceRoot: "/workspaces",
		},
		AccessibleMachines: []workflowservice.HarnessAccessibleMachineData{{
			Name:        "storage",
			Host:        "10.0.1.20",
			Description: "Artifact storage",
			Labels:      []string{"storage", "nfs"},
			SSHUser:     "openase",
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
		"WorkflowArtifacts fullstack-developer=Done:.openase/harnesses/coding-workflow.md:openase-platform,commit|dispatcher=Backlog:.openase/harnesses/dispatcher-workflow.md:|",
		"WorkflowHistory fullstack-developer=ASE-41:In Review:True:2|ASE-40:Todo:False:0|;dispatcher=;",
		"ProjectStatuses Backlog,Todo,In Progress,In Review,Done,Cancelled first=#6B7280",
		"ProjectMachines gpu-01:current:gpu,a100|storage:accessible:storage,nfs|",
		fmt.Sprintf("Platform http://localhost:19836/api/v1 %s %s", project.ID, ticketItem.ID),
		"Timestamp 2026-03-20T10:30:00Z Version 0.3.1 URL http://localhost:19836/tickets/ASE-42",
		"retry",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered harness to contain %q, got:\n%s", want, rendered)
		}
	}
}

func TestListWorkflowsRouteReturnsEmptyArrayForNewProject(t *testing.T) {
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
		nil,
		ticketstatus.NewService(client),
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

func waitForWorkflowVersion(t *testing.T, server *Server, workflowID string, wantVersion int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		getResp := struct {
			Workflow workflowResponse `json:"workflow"`
		}{}
		executeJSON(
			t,
			server,
			http.MethodGet,
			fmt.Sprintf("/api/v1/workflows/%s", workflowID),
			nil,
			http.StatusOK,
			&getResp,
		)
		if getResp.Workflow.Version == wantVersion {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for workflow %s to reach version %d", workflowID, wantVersion)
}
