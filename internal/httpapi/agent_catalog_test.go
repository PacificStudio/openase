package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
)

func TestAgentProviderAndAgentRoutes(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme Platform","slug":"acme-platform"}`)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &orgPayload)

	providerRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers",
		`{"machine_id":"`+findLocalMachineID(t, service, orgPayload.Organization.ID)+`","name":"Codex","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.3-codex","model_temperature":0.1,"model_max_tokens":32000,"cost_per_input_token":0.001,"cost_per_output_token":0.002}`,
	)
	if providerRec.Code != http.StatusCreated {
		t.Fatalf("expected provider create 201, got %d: %s", providerRec.Code, providerRec.Body.String())
	}

	var providerPayload struct {
		Provider agentProviderResponse `json:"provider"`
	}
	decodeResponse(t, providerRec, &providerPayload)
	if providerPayload.Provider.CliCommand != "codex" {
		t.Fatalf("expected provider cli_command to round-trip, got %+v", providerPayload.Provider)
	}
	if providerPayload.Provider.PermissionProfile != string(domain.AgentProviderPermissionProfileUnrestricted) {
		t.Fatalf("expected provider permission_profile to default to unrestricted, got %+v", providerPayload.Provider)
	}
	if providerPayload.Provider.MaxParallelRuns != domain.DefaultAgentProviderMaxParallelRuns {
		t.Fatalf("expected provider max_parallel_runs default to round-trip, got %+v", providerPayload.Provider)
	}
	if providerPayload.Provider.MachineName != domain.LocalMachineName {
		t.Fatalf("expected provider machine metadata to round-trip, got %+v", providerPayload.Provider)
	}
	if providerPayload.Provider.AvailabilityState == "" {
		t.Fatalf("expected provider availability_state to be populated, got %+v", providerPayload.Provider)
	}

	secondaryProviderRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers",
		`{"machine_id":"`+findLocalMachineID(t, service, orgPayload.Organization.ID)+`","name":"Codex Backup","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.4","model_temperature":0.1,"model_max_tokens":32000,"cost_per_input_token":0.001,"cost_per_output_token":0.002}`,
	)
	if secondaryProviderRec.Code != http.StatusCreated {
		t.Fatalf("expected secondary provider create 201, got %d: %s", secondaryProviderRec.Code, secondaryProviderRec.Body.String())
	}

	var secondaryProviderPayload struct {
		Provider agentProviderResponse `json:"provider"`
	}
	decodeResponse(t, secondaryProviderRec, &secondaryProviderPayload)

	listProviderRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/"+orgPayload.Organization.ID+"/providers", "")
	if listProviderRec.Code != http.StatusOK {
		t.Fatalf("expected provider list 200, got %d: %s", listProviderRec.Code, listProviderRec.Body.String())
	}

	getProviderRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/providers/"+providerPayload.Provider.ID, "")
	if getProviderRec.Code != http.StatusOK {
		t.Fatalf("expected provider get 200, got %d: %s", getProviderRec.Code, getProviderRec.Body.String())
	}

	patchProviderRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/providers/"+providerPayload.Provider.ID,
		`{"name":"Codex Primary","cli_args":["app-server","--listen","ws://127.0.0.1:7777"]}`,
	)
	if patchProviderRec.Code != http.StatusOK {
		t.Fatalf("expected provider patch 200, got %d: %s", patchProviderRec.Code, patchProviderRec.Body.String())
	}

	projectRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/projects",
		`{"name":"OpenASE","slug":"openase","description":"Main control plane","status":"In Progress","max_concurrent_agents":8}`,
	)
	if projectRec.Code != http.StatusCreated {
		t.Fatalf("expected project create 201, got %d: %s", projectRec.Code, projectRec.Body.String())
	}

	var projectPayload struct {
		Project projectResponse `json:"project"`
	}
	decodeResponse(t, projectRec, &projectPayload)

	agentRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+projectPayload.Project.ID+"/agents",
		`{"provider_id":"`+providerPayload.Provider.ID+`","name":"worker-1"}`,
	)
	if agentRec.Code != http.StatusCreated {
		t.Fatalf("expected agent create 201, got %d: %s", agentRec.Code, agentRec.Body.String())
	}

	var agentPayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, agentRec, &agentPayload)
	if agentPayload.Agent.Runtime != nil || agentPayload.Agent.RuntimeControlState != "active" || agentPayload.Agent.TotalTokensUsed != 0 {
		t.Fatalf("unexpected created agent payload: %+v", agentPayload.Agent)
	}

	listAgentRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+projectPayload.Project.ID+"/agents", "")
	if listAgentRec.Code != http.StatusOK {
		t.Fatalf("expected agent list 200, got %d: %s", listAgentRec.Code, listAgentRec.Body.String())
	}

	getAgentRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/agents/"+agentPayload.Agent.ID, "")
	if getAgentRec.Code != http.StatusOK {
		t.Fatalf("expected agent get 200, got %d: %s", getAgentRec.Code, getAgentRec.Body.String())
	}

	patchAgentRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/agents/"+agentPayload.Agent.ID,
		`{"name":"worker-1-renamed","provider_id":"`+secondaryProviderPayload.Provider.ID+`"}`,
	)
	if patchAgentRec.Code != http.StatusOK {
		t.Fatalf("expected agent patch 200, got %d: %s", patchAgentRec.Code, patchAgentRec.Body.String())
	}

	var patchAgentPayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, patchAgentRec, &patchAgentPayload)
	if patchAgentPayload.Agent.Name != "worker-1-renamed" || patchAgentPayload.Agent.ProviderID != secondaryProviderPayload.Provider.ID {
		t.Fatalf("unexpected patched agent payload: %+v", patchAgentPayload.Agent)
	}

	deleteAgentRec := performJSONRequest(t, server, http.MethodDelete, "/api/v1/agents/"+agentPayload.Agent.ID, "")
	if deleteAgentRec.Code != http.StatusOK {
		t.Fatalf("expected agent delete 200, got %d: %s", deleteAgentRec.Code, deleteAgentRec.Body.String())
	}
}

func TestCreateAgentProviderPublishesLifecycleEventToOrganizationStream(t *testing.T) {
	service := newFakeCatalogService()
	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		service,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme Platform","slug":"acme-platform"}`)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &orgPayload)

	response, cancel := openSSERequest(
		t,
		testServer.URL+"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers/stream",
	)
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close provider stream response body: %v", err)
		}
	})

	providerRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers",
		`{"machine_id":"`+findLocalMachineID(t, service, orgPayload.Organization.ID)+`","name":"Codex","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.3-codex"}`,
	)
	if providerRec.Code != http.StatusCreated {
		t.Fatalf("expected provider create 201, got %d: %s", providerRec.Code, providerRec.Body.String())
	}

	var providerPayload struct {
		Provider agentProviderResponse `json:"provider"`
	}
	decodeResponse(t, providerRec, &providerPayload)

	body := readSSEBodyUntilContainsAll(t, response, cancel, []string{
		"event: provider.created\n",
		`"organization_id":"` + orgPayload.Organization.ID + `"`,
		`"provider":{"id":"` + providerPayload.Provider.ID + `"`,
		`"cli_command":"codex"`,
		`"model_name":"gpt-5.3-codex"`,
	})

	if !strings.Contains(body, `"type":"provider.created"`) {
		t.Fatalf("expected provider.created envelope, got %q", body)
	}
}

func TestPatchAgentProviderPublishesLifecycleEventToOrganizationStream(t *testing.T) {
	service := newFakeCatalogService()
	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		service,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme Platform","slug":"acme-platform"}`)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &orgPayload)

	providerRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers",
		`{"machine_id":"`+findLocalMachineID(t, service, orgPayload.Organization.ID)+`","name":"Codex","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.3-codex"}`,
	)
	if providerRec.Code != http.StatusCreated {
		t.Fatalf("expected provider create 201, got %d: %s", providerRec.Code, providerRec.Body.String())
	}

	var providerPayload struct {
		Provider agentProviderResponse `json:"provider"`
	}
	decodeResponse(t, providerRec, &providerPayload)

	response, cancel := openSSERequest(
		t,
		testServer.URL+"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers/stream",
	)
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close provider stream response body: %v", err)
		}
	})

	patchRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/providers/"+providerPayload.Provider.ID,
		`{"model_name":"gpt-5.4","cli_command":"codex-beta","cli_args":["serve","--stdio"]}`,
	)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected provider patch 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}

	body := readSSEBodyUntilContainsAll(t, response, cancel, []string{
		"event: provider.updated\n",
		`"organization_id":"` + orgPayload.Organization.ID + `"`,
		`"provider":{"id":"` + providerPayload.Provider.ID + `"`,
		`"cli_command":"codex-beta"`,
		`"cli_args":["serve","--stdio"]`,
		`"model_name":"gpt-5.4"`,
	})

	if !strings.Contains(body, `"type":"provider.updated"`) {
		t.Fatalf("expected provider.updated envelope, got %q", body)
	}
}

func TestAgentProviderAndAgentRoutesWithEntRepository(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme Platform","slug":"acme-platform"}`)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &orgPayload)

	providerRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers",
		`{"machine_id":"`+loadEntLocalMachineID(t, client, orgPayload.Organization.ID)+`","name":"Codex","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.4"}`,
	)
	if providerRec.Code != http.StatusCreated {
		t.Fatalf("expected provider create 201, got %d: %s", providerRec.Code, providerRec.Body.String())
	}

	var providerPayload struct {
		Provider agentProviderResponse `json:"provider"`
	}
	decodeResponse(t, providerRec, &providerPayload)
	if want := []string{"app-server", "--listen", "stdio://"}; !slices.Equal(providerPayload.Provider.CliArgs, want) {
		t.Fatalf("expected provider cli_args %v, got %+v", want, providerPayload.Provider)
	}

	secondaryProviderRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers",
		`{"machine_id":"`+loadEntLocalMachineID(t, client, orgPayload.Organization.ID)+`","name":"Codex Backup","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.4"}`,
	)
	if secondaryProviderRec.Code != http.StatusCreated {
		t.Fatalf("expected secondary provider create 201, got %d: %s", secondaryProviderRec.Code, secondaryProviderRec.Body.String())
	}

	var secondaryProviderPayload struct {
		Provider agentProviderResponse `json:"provider"`
	}
	decodeResponse(t, secondaryProviderRec, &secondaryProviderPayload)

	projectRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/projects",
		`{"name":"OpenASE","slug":"openase","description":"Main control plane","status":"In Progress","max_concurrent_agents":8}`,
	)
	if projectRec.Code != http.StatusCreated {
		t.Fatalf("expected project create 201, got %d: %s", projectRec.Code, projectRec.Body.String())
	}

	var projectPayload struct {
		Project projectResponse `json:"project"`
	}
	decodeResponse(t, projectRec, &projectPayload)

	repoRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+projectPayload.Project.ID+"/repos",
		`{"name":"backend","repository_url":"https://github.com/acme/backend.git","labels":["go","api"]}`,
	)
	if repoRec.Code != http.StatusCreated {
		t.Fatalf("expected repo create 201, got %d: %s", repoRec.Code, repoRec.Body.String())
	}

	var repoPayload struct {
		Repo projectRepoResponse `json:"repo"`
	}
	decodeResponse(t, repoRec, &repoPayload)
	if want := []string{"go", "api"}; !slices.Equal(repoPayload.Repo.Labels, want) {
		t.Fatalf("expected repo labels %v, got %+v", want, repoPayload.Repo)
	}

	agentRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+projectPayload.Project.ID+"/agents",
		`{"provider_id":"`+providerPayload.Provider.ID+`","name":"worker-1"}`,
	)
	if agentRec.Code != http.StatusCreated {
		t.Fatalf("expected agent create 201, got %d: %s", agentRec.Code, agentRec.Body.String())
	}

	var agentPayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, agentRec, &agentPayload)
	if agentPayload.Agent.Runtime != nil || agentPayload.Agent.RuntimeControlState != "active" {
		t.Fatalf("expected runtime-owned fields to default, got %+v", agentPayload.Agent)
	}

	patchAgentRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/agents/"+agentPayload.Agent.ID,
		`{"name":"worker-1-renamed","provider_id":"`+secondaryProviderPayload.Provider.ID+`"}`,
	)
	if patchAgentRec.Code != http.StatusOK {
		t.Fatalf("expected agent patch 200, got %d: %s", patchAgentRec.Code, patchAgentRec.Body.String())
	}

	var patchAgentPayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, patchAgentRec, &patchAgentPayload)
	if patchAgentPayload.Agent.Name != "worker-1-renamed" || patchAgentPayload.Agent.ProviderID != secondaryProviderPayload.Provider.ID {
		t.Fatalf("unexpected patched agent payload: %+v", patchAgentPayload.Agent)
	}
}

func TestAgentProviderRoutesRejectInvalidInput(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme Platform","slug":"acme-platform"}`)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &orgPayload)

	providerRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/providers",
		`{"machine_id":"`+findLocalMachineID(t, service, orgPayload.Organization.ID)+`","name":"Custom","adapter_type":"custom","model_name":"manual"}`,
	)
	if providerRec.Code != http.StatusBadRequest {
		t.Fatalf("expected provider create 400, got %d: %s", providerRec.Code, providerRec.Body.String())
	}
	if !strings.Contains(providerRec.Body.String(), "cli_command") {
		t.Fatalf("expected cli_command validation error, got %s", providerRec.Body.String())
	}
}

func TestListAgentProvidersIncludesBuiltinCatalogAvailability(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme Platform","slug":"acme-platform"}`)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &orgPayload)

	listProviderRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/"+orgPayload.Organization.ID+"/providers", "")
	if listProviderRec.Code != http.StatusOK {
		t.Fatalf("expected provider list 200, got %d: %s", listProviderRec.Code, listProviderRec.Body.String())
	}

	var payload struct {
		Providers []agentProviderResponse `json:"providers"`
	}
	decodeResponse(t, listProviderRec, &payload)
	if len(payload.Providers) != len(domain.BuiltinAgentProviderTemplates()) {
		t.Fatalf("expected builtin provider count %d, got %+v", len(domain.BuiltinAgentProviderTemplates()), payload.Providers)
	}
	if payload.Providers[0].CliCommand == "" {
		t.Fatalf("expected seeded provider cli command, got %+v", payload.Providers[0])
	}
	if payload.Providers[0].AvailabilityState != domain.AgentProviderAvailabilityStateUnknown.String() {
		t.Fatalf("expected seeded provider availability_state=unknown, got %+v", payload.Providers[0])
	}
}

func TestListAgentsRouteOmitsCapabilitiesField(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)
	orgID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: orgID, MachineID: uuid.New(), Name: "Codex", MaxParallelRuns: domain.DefaultAgentProviderMaxParallelRuns}
	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "worker-1",
		RuntimeControlState: domain.AgentRuntimeControlStateActive,
		Runtime: &domain.AgentRuntime{
			ActiveRunCount: 2,
			Status:         domain.AgentStatusRunning,
			RuntimePhase:   domain.AgentRuntimePhaseExecuting,
		},
	}

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+projectID.String()+"/agents", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected agent list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"capabilities"`) {
		t.Fatalf("expected capabilities field to be omitted from payload, got %s", rec.Body.String())
	}

	var payload struct {
		Agents []agentResponse `json:"agents"`
	}
	decodeResponse(t, rec, &payload)
	if len(payload.Agents) != 1 {
		t.Fatalf("expected one agent, got %+v", payload.Agents)
	}
	if payload.Agents[0].RuntimeControlState != "active" {
		t.Fatalf("expected runtime control state active, got %+v", payload.Agents[0])
	}
	if payload.Agents[0].Runtime == nil || payload.Agents[0].Runtime.ActiveRunCount != 2 {
		t.Fatalf("expected aggregate active_run_count in payload, got %+v", payload.Agents[0])
	}
}

func TestListAgentRunsRouteExposesConcurrentRuns(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)
	orgID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	workflowID := uuid.New()
	workflowVersionID := uuid.New()
	skillVersionOneID := uuid.New()
	skillVersionTwoID := uuid.New()
	ticketOneID := uuid.New()
	ticketTwoID := uuid.New()
	runOneID := uuid.New()
	runTwoID := uuid.New()
	runOneCreatedAt := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	runTwoCreatedAt := runOneCreatedAt.Add(2 * time.Minute)

	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: orgID, MachineID: uuid.New(), Name: "Codex", MaxParallelRuns: domain.DefaultAgentProviderMaxParallelRuns}
	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "worker-1",
		RuntimeControlState: domain.AgentRuntimeControlStateActive,
	}
	service.agentRuns[runOneID] = domain.AgentRun{
		ID:                runOneID,
		AgentID:           agentID,
		WorkflowID:        workflowID,
		WorkflowVersionID: &workflowVersionID,
		TicketID:          ticketOneID,
		ProviderID:        providerID,
		SkillVersionIDs:   []uuid.UUID{skillVersionOneID, skillVersionTwoID},
		Status:            domain.AgentRunStatusExecuting,
		CreatedAt:         runOneCreatedAt,
	}
	service.agentRuns[runTwoID] = domain.AgentRun{
		ID:                runTwoID,
		AgentID:           agentID,
		WorkflowID:        workflowID,
		WorkflowVersionID: &workflowVersionID,
		TicketID:          ticketTwoID,
		ProviderID:        providerID,
		SkillVersionIDs:   []uuid.UUID{skillVersionTwoID},
		Status:            domain.AgentRunStatusReady,
		CreatedAt:         runTwoCreatedAt,
	}

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+projectID.String()+"/agent-runs", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected agent run list 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		AgentRuns []agentRunResponse `json:"agent_runs"`
	}
	decodeResponse(t, rec, &payload)
	if len(payload.AgentRuns) != 2 {
		t.Fatalf("expected two agent runs, got %+v", payload.AgentRuns)
	}
	if payload.AgentRuns[0].ID != runTwoID.String() || payload.AgentRuns[1].ID != runOneID.String() {
		t.Fatalf("expected newest run first, got %+v", payload.AgentRuns)
	}
	if payload.AgentRuns[0].TicketID != ticketTwoID.String() || payload.AgentRuns[1].TicketID != ticketOneID.String() {
		t.Fatalf("expected run ticket IDs to round-trip, got %+v", payload.AgentRuns)
	}
	if payload.AgentRuns[1].WorkflowVersionID == nil || *payload.AgentRuns[1].WorkflowVersionID != workflowVersionID.String() {
		t.Fatalf("expected workflow version usage to round-trip, got %+v", payload.AgentRuns[1])
	}
	if len(payload.AgentRuns[1].SkillVersionIDs) != 2 || payload.AgentRuns[1].SkillVersionIDs[0] != skillVersionOneID.String() || payload.AgentRuns[1].SkillVersionIDs[1] != skillVersionTwoID.String() {
		t.Fatalf("expected skill version usage to round-trip, got %+v", payload.AgentRuns[1])
	}
}

func TestPauseAndResumeAgentRoutes(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)
	orgID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: orgID, MachineID: uuid.New(), Name: "Codex", MaxParallelRuns: domain.DefaultAgentProviderMaxParallelRuns}
	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "worker-1",
		RuntimeControlState: domain.AgentRuntimeControlStateActive,
		Runtime: &domain.AgentRuntime{
			CurrentRunID:    &runID,
			Status:          domain.AgentStatusRunning,
			CurrentTicketID: &ticketID,
			RuntimePhase:    domain.AgentRuntimePhaseReady,
		},
	}

	pauseRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/agents/"+agentID.String()+"/pause", "")
	if pauseRec.Code != http.StatusOK {
		t.Fatalf("expected pause route 200, got %d: %s", pauseRec.Code, pauseRec.Body.String())
	}

	var pausePayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, pauseRec, &pausePayload)
	if pausePayload.Agent.RuntimeControlState != "pause_requested" {
		t.Fatalf("expected pause_requested control state, got %+v", pausePayload.Agent)
	}

	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "worker-1",
		RuntimeControlState: domain.AgentRuntimeControlStatePaused,
		Runtime: &domain.AgentRuntime{
			CurrentRunID:    &runID,
			Status:          domain.AgentStatusClaimed,
			CurrentTicketID: &ticketID,
			RuntimePhase:    domain.AgentRuntimePhaseNone,
		},
	}

	resumeRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/agents/"+agentID.String()+"/resume", "")
	if resumeRec.Code != http.StatusOK {
		t.Fatalf("expected resume route 200, got %d: %s", resumeRec.Code, resumeRec.Body.String())
	}

	var resumePayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, resumeRec, &resumePayload)
	if resumePayload.Agent.RuntimeControlState != "active" {
		t.Fatalf("expected active control state after resume, got %+v", resumePayload.Agent)
	}

	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "worker-1",
		RuntimeControlState: domain.AgentRuntimeControlStateActive,
	}

	retireRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/agents/"+agentID.String()+"/retire", "")
	if retireRec.Code != http.StatusOK {
		t.Fatalf("expected retire route 200, got %d: %s", retireRec.Code, retireRec.Body.String())
	}

	var retirePayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, retireRec, &retirePayload)
	if retirePayload.Agent.RuntimeControlState != "retired" {
		t.Fatalf("expected retired control state after retire, got %+v", retirePayload.Agent)
	}
}

func TestPauseAndResumeAgentRouteErrors(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)
	orgID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	pausedAgentID := uuid.New()
	activeAgentID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: orgID, MachineID: uuid.New(), Name: "Codex", MaxParallelRuns: domain.DefaultAgentProviderMaxParallelRuns}
	service.agents[pausedAgentID] = domain.Agent{
		ID:                  pausedAgentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "paused-agent",
		RuntimeControlState: domain.AgentRuntimeControlStatePaused,
		Runtime: &domain.AgentRuntime{
			CurrentRunID:    &runID,
			Status:          domain.AgentStatusRunning,
			CurrentTicketID: &ticketID,
			RuntimePhase:    domain.AgentRuntimePhaseReady,
		},
	}
	service.agents[activeAgentID] = domain.Agent{
		ID:                  activeAgentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "active-agent",
		RuntimeControlState: domain.AgentRuntimeControlStateActive,
		Runtime: &domain.AgentRuntime{
			CurrentRunID:    &runID,
			Status:          domain.AgentStatusRunning,
			CurrentTicketID: &ticketID,
			RuntimePhase:    domain.AgentRuntimePhaseReady,
		},
	}

	for _, testCase := range []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "pause invalid id", method: http.MethodPost, path: "/api/v1/agents/not-a-uuid/pause", wantStatus: http.StatusBadRequest, wantBody: "agentId must be a valid UUID"},
		{name: "pause missing agent", method: http.MethodPost, path: "/api/v1/agents/" + uuid.NewString() + "/pause", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "pause conflict", method: http.MethodPost, path: "/api/v1/agents/" + pausedAgentID.String() + "/pause", wantStatus: http.StatusConflict, wantBody: "AGENT_RUNTIME_CONTROL_CONFLICT"},
		{name: "resume invalid id", method: http.MethodPost, path: "/api/v1/agents/not-a-uuid/resume", wantStatus: http.StatusBadRequest, wantBody: "agentId must be a valid UUID"},
		{name: "resume missing agent", method: http.MethodPost, path: "/api/v1/agents/" + uuid.NewString() + "/resume", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "resume conflict", method: http.MethodPost, path: "/api/v1/agents/" + activeAgentID.String() + "/resume", wantStatus: http.StatusConflict, wantBody: "AGENT_RUNTIME_CONTROL_CONFLICT"},
		{name: "retire invalid id", method: http.MethodPost, path: "/api/v1/agents/not-a-uuid/retire", wantStatus: http.StatusBadRequest, wantBody: "agentId must be a valid UUID"},
		{name: "retire missing agent", method: http.MethodPost, path: "/api/v1/agents/" + uuid.NewString() + "/retire", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "retire conflict", method: http.MethodPost, path: "/api/v1/agents/" + activeAgentID.String() + "/retire", wantStatus: http.StatusConflict, wantBody: "AGENT_RUNTIME_CONTROL_CONFLICT"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, testCase.method, testCase.path, "")
			if rec.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, testCase.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("body %q does not contain %q", rec.Body.String(), testCase.wantBody)
			}
		})
	}
}

func TestDeleteAgentReturnsStructuredConflict(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	agentID := uuid.New()
	ticketID := uuid.New()
	runID := uuid.New()
	service.agents[agentID] = domain.Agent{
		ID:        agentID,
		ProjectID: uuid.New(),
		Name:      "worker-1",
	}
	service.agentDeleteConflicts[agentID] = &domain.AgentDeleteConflict{
		AgentID: agentID,
		ActiveRuns: []domain.AgentRunReference{{
			ID:       runID,
			TicketID: ticketID,
			Status:   "running",
		}},
	}

	rec := performJSONRequest(t, server, http.MethodDelete, "/api/v1/agents/"+agentID.String(), "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected delete conflict 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"code\":\"AGENT_IN_USE\"") {
		t.Fatalf("expected AGENT_IN_USE code, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"active_runs\"") {
		t.Fatalf("expected structured conflict details, got %s", rec.Body.String())
	}
}

func TestAgentCatalogRouteErrorMappingsAndHelpers(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)
	orgOneID := uuid.New()
	orgTwoID := uuid.New()
	projectOneID := uuid.New()
	projectTwoID := uuid.New()
	machineOneID := uuid.New()
	machineTwoID := uuid.New()
	providerOneID := uuid.New()
	agentID := uuid.New()

	sshUser := "openase"
	workspaceRoot := "/srv/openase/workspaces"
	service.organizations[orgOneID] = domain.Organization{ID: orgOneID, Name: "Org One", Slug: "org-one"}
	service.organizations[orgTwoID] = domain.Organization{ID: orgTwoID, Name: "Org Two", Slug: "org-two"}
	service.projects[projectOneID] = domain.Project{ID: projectOneID, OrganizationID: orgOneID, Name: "Project One", Slug: "project-one"}
	service.projects[projectTwoID] = domain.Project{ID: projectTwoID, OrganizationID: orgTwoID, Name: "Project Two", Slug: "project-two"}
	service.machines[machineOneID] = domain.Machine{ID: machineOneID, OrganizationID: orgOneID, Name: domain.LocalMachineName, Host: domain.LocalMachineHost, Status: domain.MachineStatusOnline, SSHUser: &sshUser, WorkspaceRoot: &workspaceRoot}
	service.machines[machineTwoID] = domain.Machine{ID: machineTwoID, OrganizationID: orgTwoID, Name: domain.LocalMachineName, Host: domain.LocalMachineHost, Status: domain.MachineStatusOnline}
	service.providers[providerOneID] = domain.AgentProvider{
		ID:                   providerOneID,
		OrganizationID:       orgOneID,
		MachineID:            machineOneID,
		MachineName:          domain.LocalMachineName,
		MachineHost:          domain.LocalMachineHost,
		MachineStatus:        domain.MachineStatusOnline,
		MachineSSHUser:       &sshUser,
		MachineWorkspaceRoot: &workspaceRoot,
		Name:                 "Codex",
		AdapterType:          domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:           "codex",
		MaxParallelRuns:      domain.DefaultAgentProviderMaxParallelRuns,
	}
	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerOneID,
		ProjectID:           projectOneID,
		Name:                "worker-1",
		RuntimeControlState: domain.AgentRuntimeControlStateActive,
	}

	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "list providers invalid org", method: http.MethodGet, target: "/api/v1/orgs/not-a-uuid/providers", wantStatus: http.StatusBadRequest, wantBody: "orgId must be a valid UUID"},
		{name: "list providers missing org", method: http.MethodGet, target: "/api/v1/orgs/" + uuid.NewString() + "/providers", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "get provider invalid id", method: http.MethodGet, target: "/api/v1/providers/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "providerId must be a valid UUID"},
		{name: "get provider missing", method: http.MethodGet, target: "/api/v1/providers/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "patch provider invalid id", method: http.MethodPatch, target: "/api/v1/providers/not-a-uuid", body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "providerId must be a valid UUID"},
		{name: "patch provider missing", method: http.MethodPatch, target: "/api/v1/providers/" + uuid.NewString(), body: `{}`, wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "patch provider invalid payload", method: http.MethodPatch, target: "/api/v1/providers/" + providerOneID.String(), body: `{"cli_command":" "}`, wantStatus: http.StatusBadRequest, wantBody: "model_name must not be empty"},
		{name: "list agents invalid project", method: http.MethodGet, target: "/api/v1/projects/not-a-uuid/agents", wantStatus: http.StatusBadRequest, wantBody: "projectId must be a valid UUID"},
		{name: "list agents missing project", method: http.MethodGet, target: "/api/v1/projects/" + uuid.NewString() + "/agents", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "create agent invalid payload", method: http.MethodPost, target: "/api/v1/projects/" + projectOneID.String() + "/agents", body: `{"provider_id":"bad","name":"worker"}`, wantStatus: http.StatusBadRequest, wantBody: "provider_id must be a valid UUID"},
		{name: "create agent missing provider", method: http.MethodPost, target: "/api/v1/projects/" + projectOneID.String() + "/agents", body: `{"provider_id":"` + uuid.NewString() + `","name":"worker"}`, wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "create agent project provider mismatch", method: http.MethodPost, target: "/api/v1/projects/" + projectTwoID.String() + "/agents", body: `{"provider_id":"` + providerOneID.String() + `","name":"worker"}`, wantStatus: http.StatusBadRequest, wantBody: "catalog invalid input"},
		{name: "get agent invalid id", method: http.MethodGet, target: "/api/v1/agents/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "agentId must be a valid UUID"},
		{name: "get agent missing", method: http.MethodGet, target: "/api/v1/agents/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "patch agent invalid id", method: http.MethodPatch, target: "/api/v1/agents/not-a-uuid", body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "agentId must be a valid UUID"},
		{name: "patch agent missing", method: http.MethodPatch, target: "/api/v1/agents/" + uuid.NewString(), body: `{}`, wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "patch agent invalid payload", method: http.MethodPatch, target: "/api/v1/agents/" + agentID.String(), body: `{"name":" "}`, wantStatus: http.StatusBadRequest, wantBody: "name must not be empty"},
		{name: "delete agent invalid id", method: http.MethodDelete, target: "/api/v1/agents/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "agentId must be a valid UUID"},
		{name: "delete agent missing", method: http.MethodDelete, target: "/api/v1/agents/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, testCase.method, testCase.target, testCase.body)
			if rec.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, testCase.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("body %q does not contain %q", rec.Body.String(), testCase.wantBody)
			}
		})
	}

	modelOptionsRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/provider-model-options", "")
	if modelOptionsRec.Code != http.StatusOK {
		t.Fatalf("provider model options status = %d, body=%s", modelOptionsRec.Code, modelOptionsRec.Body.String())
	}
	if !strings.Contains(modelOptionsRec.Body.String(), `"adapter_type":"codex-app-server"`) {
		t.Fatalf("provider model options body missing codex adapter: %s", modelOptionsRec.Body.String())
	}
	if !strings.Contains(modelOptionsRec.Body.String(), `"id":"gpt-5.4"`) {
		t.Fatalf("provider model options body missing codex model: %s", modelOptionsRec.Body.String())
	}

	mappedProvider := mapAgentProviderResponse(service.providers[providerOneID])
	if mappedProvider.MachineSSHUser == nil || *mappedProvider.MachineSSHUser != sshUser || mappedProvider.MachineWorkspaceRoot == nil || *mappedProvider.MachineWorkspaceRoot != workspaceRoot {
		t.Fatalf("mapAgentProviderResponse() = %+v", mappedProvider)
	}
	if mappedProvider.Capabilities.EphemeralChat.State != domain.AgentProviderCapabilityStateUnavailable.String() {
		t.Fatalf("expected mapped provider ephemeral chat state unavailable, got %+v", mappedProvider.Capabilities)
	}
	if mappedProvider.Capabilities.EphemeralChat.Reason == nil || *mappedProvider.Capabilities.EphemeralChat.Reason != "not_ready" {
		t.Fatalf("expected mapped provider ephemeral chat reason not_ready, got %+v", mappedProvider.Capabilities)
	}
	*mappedProvider.MachineSSHUser = "changed"
	if *service.providers[providerOneID].MachineSSHUser != sshUser {
		t.Fatalf("mapAgentProviderResponse() should clone machine pointers, got %+v", service.providers[providerOneID])
	}
	if stringPointerValue(nil) != nil {
		t.Fatal("stringPointerValue(nil) expected nil")
	}
}

func (f *fakeCatalogService) ListAgentProviders(_ context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error) {
	item, ok := f.organizations[organizationID]
	if !ok || item.Status == "archived" {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.AgentProvider, 0)
	for _, item := range f.providers {
		if item.OrganizationID == organizationID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (f *fakeCatalogService) CreateAgentProvider(_ context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error) {
	org, ok := f.organizations[input.OrganizationID]
	if !ok || org.Status == "archived" {
		return domain.AgentProvider{}, catalogservice.ErrNotFound
	}

	if input.CliCommand == "" {
		return domain.AgentProvider{}, fmt.Errorf("%w: cli_command must not be empty", catalogservice.ErrInvalidInput)
	}
	if _, ok := f.machines[input.MachineID]; !ok {
		return domain.AgentProvider{}, fmt.Errorf("%w: machine_id must reference an existing machine", catalogservice.ErrInvalidInput)
	}

	machine := f.machines[input.MachineID]

	provider := domain.AgentProvider{
		ID:                   uuid.New(),
		OrganizationID:       input.OrganizationID,
		MachineID:            input.MachineID,
		MachineName:          machine.Name,
		MachineHost:          machine.Host,
		MachineStatus:        machine.Status,
		MachineSSHUser:       cloneStringPointer(machine.SSHUser),
		MachineWorkspaceRoot: cloneStringPointer(machine.WorkspaceRoot),
		MachineAgentCLIPath:  cloneStringPointer(machine.AgentCLIPath),
		MachineResources:     cloneMap(machine.Resources),
		Name:                 input.Name,
		AdapterType:          input.AdapterType,
		PermissionProfile:    input.PermissionProfile,
		CliCommand:           input.CliCommand,
		CliArgs:              append([]string(nil), input.CliArgs...),
		AuthConfig:           cloneMap(input.AuthConfig),
		ModelName:            input.ModelName,
		ModelTemperature:     input.ModelTemperature,
		ModelMaxTokens:       input.ModelMaxTokens,
		MaxParallelRuns:      input.MaxParallelRuns,
		CostPerInputToken:    input.CostPerInputToken,
		CostPerOutputToken:   input.CostPerOutputToken,
	}
	f.providers[provider.ID] = provider

	return provider, nil
}

func (f *fakeCatalogService) GetAgentProvider(_ context.Context, id uuid.UUID) (domain.AgentProvider, error) {
	item, ok := f.providers[id]
	if !ok {
		return domain.AgentProvider{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateAgentProvider(_ context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	if _, ok := f.providers[input.ID]; !ok {
		return domain.AgentProvider{}, catalogservice.ErrNotFound
	}
	if input.CliCommand == "" {
		return domain.AgentProvider{}, fmt.Errorf("%w: cli_command must not be empty", catalogservice.ErrInvalidInput)
	}
	if _, ok := f.machines[input.MachineID]; !ok {
		return domain.AgentProvider{}, fmt.Errorf("%w: machine_id must reference an existing machine", catalogservice.ErrInvalidInput)
	}

	machine := f.machines[input.MachineID]

	item := domain.AgentProvider{
		ID:                   input.ID,
		OrganizationID:       input.OrganizationID,
		MachineID:            input.MachineID,
		MachineName:          machine.Name,
		MachineHost:          machine.Host,
		MachineStatus:        machine.Status,
		MachineSSHUser:       cloneStringPointer(machine.SSHUser),
		MachineWorkspaceRoot: cloneStringPointer(machine.WorkspaceRoot),
		MachineAgentCLIPath:  cloneStringPointer(machine.AgentCLIPath),
		MachineResources:     cloneMap(machine.Resources),
		Name:                 input.Name,
		AdapterType:          input.AdapterType,
		PermissionProfile:    input.PermissionProfile,
		CliCommand:           input.CliCommand,
		CliArgs:              append([]string(nil), input.CliArgs...),
		AuthConfig:           cloneMap(input.AuthConfig),
		ModelName:            input.ModelName,
		ModelTemperature:     input.ModelTemperature,
		ModelMaxTokens:       input.ModelMaxTokens,
		MaxParallelRuns:      input.MaxParallelRuns,
		CostPerInputToken:    input.CostPerInputToken,
		CostPerOutputToken:   input.CostPerOutputToken,
	}
	f.providers[input.ID] = item

	return item, nil
}

func (f *fakeCatalogService) ListAgents(_ context.Context, projectID uuid.UUID) ([]domain.Agent, error) {
	if _, ok := f.projects[projectID]; !ok {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.Agent, 0)
	for _, item := range f.agents {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (f *fakeCatalogService) ListAgentRuns(_ context.Context, projectID uuid.UUID) ([]domain.AgentRun, error) {
	if _, ok := f.projects[projectID]; !ok {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.AgentRun, 0)
	for _, item := range f.agentRuns {
		agentItem, ok := f.agents[item.AgentID]
		if ok && agentItem.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID.String() > items[j].ID.String()
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	return items, nil
}

func (f *fakeCatalogService) ListActivityEvents(_ context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error) {
	if _, ok := f.projects[input.ProjectID]; !ok {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.ActivityEvent, 0)
	for _, item := range f.activityEvents {
		if item.ProjectID != input.ProjectID {
			continue
		}
		if item.EventType == domain.AgentOutputEventType {
			continue
		}
		if input.AgentID != nil {
			if item.AgentID == nil || *item.AgentID != *input.AgentID {
				continue
			}
		}
		if input.TicketID != nil {
			if item.TicketID == nil || *item.TicketID != *input.TicketID {
				continue
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID.String() > items[j].ID.String()
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > input.Limit {
		items = items[:input.Limit]
	}

	return items, nil
}

func (f *fakeCatalogService) ListAgentOutput(_ context.Context, input domain.ListAgentOutput) ([]domain.AgentOutputEntry, error) {
	agent, ok := f.agents[input.AgentID]
	if !ok || agent.ProjectID != input.ProjectID {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.AgentOutputEntry, 0)
	for _, item := range f.traceEvents {
		if item.ProjectID != input.ProjectID || item.AgentID != input.AgentID {
			continue
		}
		if item.Kind != domain.AgentTraceKindAssistantDelta &&
			item.Kind != domain.AgentTraceKindAssistantSnapshot &&
			item.Kind != domain.AgentTraceKindCommandDelta &&
			item.Kind != domain.AgentTraceKindCommandSnapshot {
			continue
		}
		if input.TicketID != nil {
			if item.TicketID == nil || *item.TicketID != *input.TicketID {
				continue
			}
		}
		items = append(items, domain.AgentOutputEntry{
			ID:         item.ID,
			ProjectID:  item.ProjectID,
			AgentID:    item.AgentID,
			TicketID:   item.TicketID,
			AgentRunID: item.AgentRunID,
			Stream:     item.Stream,
			Output:     item.Output,
			CreatedAt:  item.CreatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID.String() > items[j].ID.String()
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > input.Limit {
		items = items[:input.Limit]
	}

	return items, nil
}

func (f *fakeCatalogService) ListAgentSteps(_ context.Context, input domain.ListAgentSteps) ([]domain.AgentStepEntry, error) {
	agent, ok := f.agents[input.AgentID]
	if !ok || agent.ProjectID != input.ProjectID {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.AgentStepEntry, 0)
	for _, item := range f.stepEvents {
		if item.ProjectID != input.ProjectID || item.AgentID != input.AgentID {
			continue
		}
		if input.TicketID != nil {
			if item.TicketID == nil || *item.TicketID != *input.TicketID {
				continue
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID.String() > items[j].ID.String()
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > input.Limit {
		items = items[:input.Limit]
	}

	return items, nil
}

func (f *fakeCatalogService) ListAgentRunTraceEntries(_ context.Context, input domain.ListAgentRunTraceEntries) ([]domain.AgentTraceEntry, error) {
	run, ok := f.agentRuns[input.AgentRunID]
	if !ok {
		return nil, catalogservice.ErrNotFound
	}
	ticket, ok := f.tickets[run.TicketID]
	if !ok || ticket.ProjectID != input.ProjectID {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.AgentTraceEntry, 0)
	for _, item := range f.traceEvents {
		if item.ProjectID != input.ProjectID || item.AgentRunID != input.AgentRunID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sequence == items[j].Sequence {
			return items[i].ID.String() < items[j].ID.String()
		}
		return items[i].Sequence < items[j].Sequence
	})
	if input.Limit > 0 && len(items) > input.Limit {
		items = items[:input.Limit]
	}

	return items, nil
}

func (f *fakeCatalogService) ListAgentRunStepEntries(_ context.Context, input domain.ListAgentRunStepEntries) ([]domain.AgentStepEntry, error) {
	run, ok := f.agentRuns[input.AgentRunID]
	if !ok {
		return nil, catalogservice.ErrNotFound
	}
	ticket, ok := f.tickets[run.TicketID]
	if !ok || ticket.ProjectID != input.ProjectID {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.AgentStepEntry, 0)
	for _, item := range f.stepEvents {
		if item.ProjectID != input.ProjectID || item.AgentRunID != input.AgentRunID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID.String() < items[j].ID.String()
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	if input.Limit > 0 && len(items) > input.Limit {
		items = items[:input.Limit]
	}

	return items, nil
}

func (f *fakeCatalogService) CreateAgent(_ context.Context, input domain.CreateAgent) (domain.Agent, error) {
	project, ok := f.projects[input.ProjectID]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}

	provider, ok := f.providers[input.ProviderID]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}
	if project.OrganizationID != provider.OrganizationID {
		return domain.Agent{}, catalogservice.ErrInvalidInput
	}

	item := domain.Agent{
		ID:                    uuid.New(),
		ProviderID:            input.ProviderID,
		ProjectID:             input.ProjectID,
		Name:                  input.Name,
		RuntimeControlState:   input.RuntimeControlState,
		TotalTokensUsed:       input.TotalTokensUsed,
		TotalTicketsCompleted: input.TotalTicketsCompleted,
	}
	f.agents[item.ID] = item

	return item, nil
}

func (f *fakeCatalogService) GetAgent(_ context.Context, id uuid.UUID) (domain.Agent, error) {
	item, ok := f.agents[id]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateAgent(_ context.Context, input domain.UpdateAgent) (domain.Agent, error) {
	item, ok := f.agents[input.ID]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}
	if item.ProjectID != input.ProjectID {
		return domain.Agent{}, catalogservice.ErrInvalidInput
	}

	project, ok := f.projects[input.ProjectID]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}

	provider, ok := f.providers[input.ProviderID]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}
	if provider.OrganizationID != project.OrganizationID {
		return domain.Agent{}, catalogservice.ErrInvalidInput
	}

	item.ProviderID = input.ProviderID
	item.Name = input.Name
	f.agents[input.ID] = item
	return item, nil
}

func (f *fakeCatalogService) GetAgentRun(_ context.Context, id uuid.UUID) (domain.AgentRun, error) {
	item, ok := f.agentRuns[id]
	if !ok {
		return domain.AgentRun{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) RequestAgentPause(_ context.Context, id uuid.UUID) (domain.Agent, error) {
	item, ok := f.agents[id]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}

	nextState, err := domain.ResolvePauseRuntimeControlState(item)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("%w: %v", catalogservice.ErrConflict, err)
	}

	item.RuntimeControlState = nextState
	f.agents[id] = item
	return item, nil
}

func (f *fakeCatalogService) RequestAgentResume(_ context.Context, id uuid.UUID) (domain.Agent, error) {
	item, ok := f.agents[id]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}

	nextState, err := domain.ResolveResumeRuntimeControlState(item)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("%w: %v", catalogservice.ErrConflict, err)
	}

	item.RuntimeControlState = nextState
	f.agents[id] = item
	return item, nil
}

func (f *fakeCatalogService) RetireAgent(_ context.Context, id uuid.UUID) (domain.Agent, error) {
	item, ok := f.agents[id]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}

	nextState, err := domain.ResolveRetireRuntimeControlState(item)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("%w: %v", catalogservice.ErrConflict, err)
	}

	item.RuntimeControlState = nextState
	f.agents[id] = item
	return item, nil
}

func (f *fakeCatalogService) DeleteAgent(_ context.Context, id uuid.UUID) (domain.Agent, error) {
	item, ok := f.agents[id]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}
	if conflict, ok := f.agentDeleteConflicts[id]; ok && conflict != nil {
		return domain.Agent{}, conflict
	}

	delete(f.agents, id)
	return item, nil
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func findLocalMachineID(t *testing.T, service *fakeCatalogService, organizationID string) string {
	t.Helper()
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		t.Fatalf("parse organization id: %v", err)
	}
	for _, machine := range service.machines {
		if machine.OrganizationID == orgID && machine.Name == domain.LocalMachineName {
			return machine.ID.String()
		}
	}
	t.Fatalf("local machine not found for organization %s", organizationID)
	return ""
}

func loadEntLocalMachineID(t *testing.T, client *ent.Client, organizationID string) string {
	t.Helper()
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		t.Fatalf("parse organization id: %v", err)
	}
	machine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(orgID),
			entmachine.NameEQ(domain.LocalMachineName),
		).
		Only(context.Background())
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}
	return machine.ID.String()
}
