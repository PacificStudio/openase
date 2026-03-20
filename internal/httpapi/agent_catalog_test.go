package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
)

func TestAgentProviderAndAgentRoutes(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
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
		`{"name":"Codex","adapter_type":"codex-app-server","cli_command":"codex","cli_args":["serve","--stdio"],"auth_config":{"token":"secret"},"model_name":"gpt-5.3-codex","model_temperature":0.1,"model_max_tokens":32000,"cost_per_input_token":0.001,"cost_per_output_token":0.002}`,
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
		`{"name":"Codex Primary","cli_args":["serve","--tcp"]}`,
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
		`{"provider_id":"`+providerPayload.Provider.ID+`","name":"worker-1","status":"running","session_id":"sess-1","workspace_path":"/tmp/openase","capabilities":["go","backend"],"total_tokens_used":42,"total_tickets_completed":3,"last_heartbeat_at":"2026-03-19T17:00:00Z"}`,
	)
	if agentRec.Code != http.StatusCreated {
		t.Fatalf("expected agent create 201, got %d: %s", agentRec.Code, agentRec.Body.String())
	}

	var agentPayload struct {
		Agent agentResponse `json:"agent"`
	}
	decodeResponse(t, agentRec, &agentPayload)
	if agentPayload.Agent.Status != "running" || agentPayload.Agent.TotalTokensUsed != 42 {
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

func TestAgentProviderRoutesRejectInvalidInput(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
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

func (f *fakeCatalogService) ListAgentProviders(_ context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error) {
	if _, ok := f.organizations[organizationID]; !ok {
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
	if _, ok := f.organizations[input.OrganizationID]; !ok {
		return domain.AgentProvider{}, catalogservice.ErrNotFound
	}

	if input.CliCommand == "" {
		return domain.AgentProvider{}, fmt.Errorf("%w: cli_command must not be empty", catalogservice.ErrInvalidInput)
	}

	item := domain.AgentProvider{
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
	f.providers[item.ID] = item

	return item, nil
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
		WorkspacePath:         input.WorkspacePath,
		Capabilities:          append([]string(nil), input.Capabilities...),
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
