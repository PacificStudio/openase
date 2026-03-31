package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	githubconnector "github.com/BetterAndBetterII/openase/internal/infra/issueconnector/github"
	issueconnectorregistry "github.com/BetterAndBetterII/openase/internal/issueconnector"
	issueconnectorrepo "github.com/BetterAndBetterII/openase/internal/repo/issueconnector"
	issueconnectorsync "github.com/BetterAndBetterII/openase/internal/runtime/issueconnectorsync"
	issueconnectorservice "github.com/BetterAndBetterII/openase/internal/service/issueconnector"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func TestIssueConnectorRoutesCRUDTestSyncAndStats(t *testing.T) {
	ctx := context.Background()
	client := testPostgres.NewIsolatedEntClient(t)
	projectID := seedIssueConnectorProject(ctx, t, client)

	statusService := ticketstatus.NewService(client)
	if _, err := statusService.ResetToDefaultTemplate(ctx, projectID); err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/backend":
			_, _ = io.WriteString(w, `{"full_name":"acme/backend"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/backend/issues":
			_, _ = io.WriteString(w, `[
				{
					"number": 42,
					"html_url": "https://github.com/acme/backend/issues/42",
					"title": "Import connector management plane",
					"body": "Backfill project-facing settings.",
					"state": "open",
					"user": {"login": "octocat"},
					"assignees": [{"login": "codex"}],
					"labels": [{"name": "openase"}],
					"created_at": "2026-03-30T10:00:00Z",
					"updated_at": "2026-03-30T12:00:00Z"
				},
				{
					"number": 43,
					"html_url": "https://github.com/acme/backend/issues/43",
					"title": "Ignore non-matching label",
					"body": "Should stay outside the connector filter.",
					"state": "open",
					"user": {"login": "octocat"},
					"labels": [{"name": "other"}],
					"created_at": "2026-03-30T10:00:00Z",
					"updated_at": "2026-03-30T12:00:00Z"
				}
			]`)
		default:
			t.Fatalf("unexpected GitHub connector request %s %s", r.Method, r.URL.String())
		}
	}))
	defer githubAPI.Close()

	registry, err := issueconnectorregistry.NewRegistry(githubconnector.New(githubAPI.Client()))
	if err != nil {
		t.Fatalf("create issue connector registry: %v", err)
	}
	connectorService := issueconnectorservice.New(
		issueconnectorrepo.NewEntRepository(client),
		registry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	connectorService.ConfigureSyncRunner(issueconnectorsync.NewRunner(
		issueconnectorrepo.NewEntRepository(client),
		registry,
		client,
		ticketservice.NewService(client),
		statusService,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	))
	server := NewServer(
		config.ServerConfig{Port: 40123},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		statusService,
		nil,
		nil,
		nil,
		WithIssueConnectorService(connectorService),
	)

	createRec := performJSONRequest(t, server, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/connectors", projectID), fmt.Sprintf(`{
		"type":"github",
		"name":"GitHub Backend",
		"status":"active",
		"config":{
			"type":"github",
			"base_url":"%s",
			"auth_token":"ghu_direct_token",
			"project_ref":"acme/backend",
			"poll_interval":"5m",
			"sync_direction":"bidirectional",
			"filters":{"labels":["openase"]},
			"status_mapping":{"open":"Todo","closed":"Done"},
			"webhook_secret":"secret-value"
		}
	}`, githubAPI.URL))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected connector create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var createResp struct {
		Connector issueConnectorResponse `json:"connector"`
	}
	decodeResponse(t, createRec, &createResp)
	if !createResp.Connector.Config.AuthTokenConfigured || !createResp.Connector.Config.WebhookSecretConfigured {
		t.Fatalf("expected secret configuration flags in create response, got %+v", createResp.Connector.Config)
	}

	listRec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/connectors", projectID), "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected connector list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Connectors []issueConnectorResponse `json:"connectors"`
	}
	decodeResponse(t, listRec, &listResp)
	if len(listResp.Connectors) != 1 || listResp.Connectors[0].Name != "GitHub Backend" {
		t.Fatalf("unexpected connector list payload: %+v", listResp.Connectors)
	}

	testRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/connectors/%s/test", createResp.Connector.ID),
		"",
	)
	if testRec.Code != http.StatusOK {
		t.Fatalf("expected connector test 200, got %d: %s", testRec.Code, testRec.Body.String())
	}
	var testResp issueConnectorTestResponse
	decodeResponse(t, testRec, &testResp)
	if !testResp.Result.Healthy {
		t.Fatalf("expected connector test to succeed, got %+v", testResp.Result)
	}

	syncRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/connectors/%s/sync", createResp.Connector.ID),
		"",
	)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("expected connector sync 200, got %d: %s", syncRec.Code, syncRec.Body.String())
	}
	var syncResp issueConnectorSyncResponse
	decodeResponse(t, syncRec, &syncResp)
	if syncResp.Report.IssuesSynced != 1 {
		t.Fatalf("expected one synced issue, got %+v", syncResp.Report)
	}

	tickets, err := ticketservice.NewService(client).List(ctx, ticketservice.ListInput{ProjectID: projectID})
	if err != nil {
		t.Fatalf("list synced tickets: %v", err)
	}
	if len(tickets) != 1 || tickets[0].ExternalRef != "acme/backend#42" {
		t.Fatalf("expected synced ticket external ref, got %+v", tickets)
	}

	statsRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/connectors/%s/stats", createResp.Connector.ID),
		"",
	)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("expected connector stats 200, got %d: %s", statsRec.Code, statsRec.Body.String())
	}
	var statsResp issueConnectorStatsEnvelope
	decodeResponse(t, statsRec, &statsResp)
	if statsResp.Stats.Stats.TotalSynced != 1 || statsResp.Stats.LastError != "" {
		t.Fatalf("unexpected connector stats payload: %+v", statsResp.Stats)
	}

	patchRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/connectors/%s", createResp.Connector.ID),
		`{
			"name":"GitHub Backend Mirror",
			"status":"paused",
			"config":{
				"auth_token":"",
				"project_ref":"acme/backend",
				"filters":{"labels":["openase","platform"]},
				"status_mapping":{"open":"Todo"}
			}
		}`,
	)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected connector patch 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}
	var patchResp struct {
		Connector issueConnectorResponse `json:"connector"`
	}
	decodeResponse(t, patchRec, &patchResp)
	if patchResp.Connector.Name != "GitHub Backend Mirror" || patchResp.Connector.Status != "paused" {
		t.Fatalf("unexpected patched connector payload: %+v", patchResp.Connector)
	}
	if patchResp.Connector.Config.AuthTokenConfigured {
		t.Fatalf("expected cleared connector auth token, got %+v", patchResp.Connector.Config)
	}

	deleteRec := performJSONRequest(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/connectors/%s", createResp.Connector.ID),
		"",
	)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected connector delete 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestIssueConnectorRoutesRejectUnknownConnectorType(t *testing.T) {
	ctx := context.Background()
	client := testPostgres.NewIsolatedEntClient(t)
	projectID := seedIssueConnectorProject(ctx, t, client)

	registry, err := issueconnectorregistry.NewRegistry(githubconnector.New(nil))
	if err != nil {
		t.Fatalf("create issue connector registry: %v", err)
	}
	connectorService := issueconnectorservice.New(
		issueconnectorrepo.NewEntRepository(client),
		registry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	server := NewServer(
		config.ServerConfig{Port: 40124},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		nil,
		nil,
		WithIssueConnectorService(connectorService),
	)

	rec := performJSONRequest(t, server, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/connectors", projectID), `{
		"type":"jira",
		"name":"Jira",
		"config":{"type":"jira","project_ref":"acme/backend"}
	}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown connector type 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !json.Valid(rec.Body.Bytes()) || !strings.Contains(rec.Body.String(), "CONNECTOR_TYPE_NOT_FOUND") {
		t.Fatalf("unexpected unknown connector type payload: %s", rec.Body.String())
	}
}

func seedIssueConnectorProject(ctx context.Context, t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Connectors").
		SetSlug("connectors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	return project.ID
}
