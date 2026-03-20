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
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
)

func TestWorkflowRoutesCRUDHarnessStorageAndHotReload(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
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
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		ticketstatus.NewService(client),
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

	statusSvc := ticketstatus.NewService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	activateMarkerPath := filepath.Join(repoRoot, "activate.marker")
	reloadMarkerPath := filepath.Join(repoRoot, "reload.marker")

	createResp := struct {
		Workflow workflowResponse `json:"workflow"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
		map[string]any{
			"name":             "Coding Workflow",
			"type":             "coding",
			"pickup_status_id": todoID.String(),
			"finish_status_id": doneID.String(),
			"harness_content":  "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
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
	if createResp.Workflow.HarnessContent == nil || *createResp.Workflow.HarnessContent == "" {
		t.Fatalf("expected harness content in create response, got %+v", createResp.Workflow)
	}
	activateMarker, err := os.ReadFile(activateMarkerPath)
	if err != nil {
		t.Fatalf("read activate marker: %v", err)
	}
	if string(activateMarker) != "Coding Workflow:1" {
		t.Fatalf("expected activate marker to capture workflow context, got %q", string(activateMarker))
	}

	harnessAbsPath := filepath.Join(repoRoot, filepath.FromSlash(createResp.Workflow.HarnessPath))
	fileContent, err := os.ReadFile(harnessAbsPath)
	if err != nil {
		t.Fatalf("read created harness file: %v", err)
	}
	if string(fileContent) != *createResp.Workflow.HarnessContent {
		t.Fatalf("expected harness file content to match response, got %q", string(fileContent))
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
	if err := os.WriteFile(harnessAbsPath, []byte(externalContent), 0o644); err != nil {
		t.Fatalf("write external harness change: %v", err)
	}

	waitForWorkflowVersion(t, server, createResp.Workflow.ID, 3)
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
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
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
