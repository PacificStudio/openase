package httpapi

import (
	"context"
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
					"on_reload": []map[string]any{{"cmd": "echo reload"}},
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
			"is_active":      false,
		},
		http.StatusOK,
		&patchResp,
	)
	if patchResp.Workflow.Name != "Core Coding Workflow" || patchResp.Workflow.MaxConcurrent != 7 || patchResp.Workflow.IsActive {
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

	externalContent := "---\nworkflow:\n  role: coding\n---\n\n# Updated on disk\n"
	if err := os.WriteFile(harnessAbsPath, []byte(externalContent), 0o644); err != nil {
		t.Fatalf("write external harness change: %v", err)
	}

	waitForWorkflowVersion(t, server, createResp.Workflow.ID, 3)

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
