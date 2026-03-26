package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
)

func TestAgentProviderAndAgentRoutes(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
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
		`{"name":"Codex","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.3-codex","model_temperature":0.1,"model_max_tokens":32000,"cost_per_input_token":0.001,"cost_per_output_token":0.002}`,
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

	listProviderRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/"+orgPayload.Organization.ID+"/providers", "")
	if listProviderRec.Code != http.StatusOK {
		t.Fatalf("expected provider list 200, got %d: %s", listProviderRec.Code, listProviderRec.Body.String())
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
		`{"name":"OpenASE","slug":"openase","description":"Main control plane","status":"active","max_concurrent_agents":8}`,
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
		`{"provider_id":"`+providerPayload.Provider.ID+`","name":"worker-1","workspace_path":"/tmp/openase"}`,
	)
	if agentRec.Code != http.StatusCreated {
		t.Fatalf("expected agent create 201, got %d: %s", agentRec.Code, agentRec.Body.String())
	}

	var agentPayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, agentRec, &agentPayload)
	if agentPayload.Agent.Status != "idle" || agentPayload.Agent.RuntimePhase != "none" || agentPayload.Agent.RuntimeControlState != "active" || agentPayload.Agent.TotalTokensUsed != 0 {
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

	deleteAgentRec := performJSONRequest(t, server, http.MethodDelete, "/api/v1/agents/"+agentPayload.Agent.ID, "")
	if deleteAgentRec.Code != http.StatusOK {
		t.Fatalf("expected agent delete 200, got %d: %s", deleteAgentRec.Code, deleteAgentRec.Body.String())
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
		`{"name":"Codex","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["app-server","--listen","stdio://"],"auth_config":{"token":"secret"},"model_name":"gpt-5.4"}`,
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

	projectRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/projects",
		`{"name":"OpenASE","slug":"openase","description":"Main control plane","status":"active","max_concurrent_agents":8}`,
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
		`{"provider_id":"`+providerPayload.Provider.ID+`","name":"worker-1","workspace_path":"/tmp/openase"}`,
	)
	if agentRec.Code != http.StatusCreated {
		t.Fatalf("expected agent create 201, got %d: %s", agentRec.Code, agentRec.Body.String())
	}

	var agentPayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, agentRec, &agentPayload)
	if agentPayload.Agent.Status != "idle" || agentPayload.Agent.RuntimePhase != "none" || agentPayload.Agent.RuntimeControlState != "active" {
		t.Fatalf("expected runtime-owned fields to default, got %+v", agentPayload.Agent)
	}
}

func TestAgentProviderRoutesRejectInvalidInput(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
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
		`{"name":"Custom","adapter_type":"custom","model_name":"manual"}`,
	)
	if providerRec.Code != http.StatusBadRequest {
		t.Fatalf("expected provider create 400, got %d: %s", providerRec.Code, providerRec.Body.String())
	}
	if !strings.Contains(providerRec.Body.String(), "cli_command") {
		t.Fatalf("expected cli_command validation error, got %s", providerRec.Body.String())
	}
}

func TestListAgentProvidersIncludesBuiltinCatalogAvailability(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
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
}

func TestListAgentsRouteOmitsCapabilitiesField(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
		nil,
	)

	service := server.catalog.(*fakeCatalogService)
	orgID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: orgID, Name: "Codex"}
	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "worker-1",
		Status:              entagent.StatusIdle,
		RuntimeControlState: entagent.RuntimeControlStateActive,
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
}

func TestPauseAndResumeAgentRoutes(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
		nil,
	)

	service := server.catalog.(*fakeCatalogService)
	orgID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	ticketID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase"}
	service.providers[providerID] = domain.AgentProvider{ID: providerID, OrganizationID: orgID, Name: "Codex"}
	service.agents[agentID] = domain.Agent{
		ID:                  agentID,
		ProviderID:          providerID,
		ProjectID:           projectID,
		Name:                "worker-1",
		Status:              entagent.StatusRunning,
		CurrentTicketID:     &ticketID,
		RuntimePhase:        entagent.RuntimePhaseReady,
		RuntimeControlState: entagent.RuntimeControlStateActive,
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
		Status:              entagent.StatusClaimed,
		CurrentTicketID:     &ticketID,
		RuntimePhase:        entagent.RuntimePhaseNone,
		RuntimeControlState: entagent.RuntimeControlStatePaused,
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

	provider := domain.AgentProvider{
		ID:                 uuid.New(),
		OrganizationID:     input.OrganizationID,
		Name:               input.Name,
		AdapterType:        input.AdapterType,
		CliCommand:         input.CliCommand,
		CliArgs:            append([]string(nil), input.CliArgs...),
		AuthConfig:         cloneMap(input.AuthConfig),
		ModelName:          input.ModelName,
		ModelTemperature:   input.ModelTemperature,
		ModelMaxTokens:     input.ModelMaxTokens,
		CostPerInputToken:  input.CostPerInputToken,
		CostPerOutputToken: input.CostPerOutputToken,
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

	item := domain.AgentProvider{
		ID:                 input.ID,
		OrganizationID:     input.OrganizationID,
		Name:               input.Name,
		AdapterType:        input.AdapterType,
		CliCommand:         input.CliCommand,
		CliArgs:            append([]string(nil), input.CliArgs...),
		AuthConfig:         cloneMap(input.AuthConfig),
		ModelName:          input.ModelName,
		ModelTemperature:   input.ModelTemperature,
		ModelMaxTokens:     input.ModelMaxTokens,
		CostPerInputToken:  input.CostPerInputToken,
		CostPerOutputToken: input.CostPerOutputToken,
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

func (f *fakeCatalogService) ListActivityEvents(_ context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error) {
	if _, ok := f.projects[input.ProjectID]; !ok {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.ActivityEvent, 0)
	for _, item := range f.activityEvents {
		if item.ProjectID != input.ProjectID {
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
	for _, item := range f.activityEvents {
		if item.ProjectID != input.ProjectID || item.EventType != domain.AgentOutputEventType {
			continue
		}
		if item.AgentID == nil || *item.AgentID != input.AgentID {
			continue
		}
		if input.TicketID != nil {
			if item.TicketID == nil || *item.TicketID != *input.TicketID {
				continue
			}
		}
		items = append(items, domain.AgentOutputEntry{
			ID:        item.ID,
			ProjectID: item.ProjectID,
			AgentID:   *item.AgentID,
			TicketID:  item.TicketID,
			Stream:    domain.AgentOutputMetadataStream(item.Metadata),
			Output:    item.Message,
			CreatedAt: item.CreatedAt,
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
		Status:                input.Status,
		CurrentTicketID:       input.CurrentTicketID,
		SessionID:             input.SessionID,
		RuntimePhase:          input.RuntimePhase,
		RuntimeControlState:   input.RuntimeControlState,
		RuntimeStartedAt:      cloneTimePointer(input.RuntimeStartedAt),
		LastError:             input.LastError,
		WorkspacePath:         input.WorkspacePath,
		TotalTokensUsed:       input.TotalTokensUsed,
		TotalTicketsCompleted: input.TotalTicketsCompleted,
		LastHeartbeatAt:       cloneTimePointer(input.LastHeartbeatAt),
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

func (f *fakeCatalogService) DeleteAgent(_ context.Context, id uuid.UUID) (domain.Agent, error) {
	item, ok := f.agents[id]
	if !ok {
		return domain.Agent{}, catalogservice.ErrNotFound
	}

	delete(f.agents, id)
	return item, nil
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}
