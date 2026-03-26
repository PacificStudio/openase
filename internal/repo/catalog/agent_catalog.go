package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	"github.com/google/uuid"
)

func (r *EntRepository) ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error) {
	exists, err := r.organizationIsActive(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("check organization before listing agent providers: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	items, err := r.client.AgentProvider.Query().
		Where(entagentprovider.OrganizationID(organizationID)).
		Order(entagentprovider.ByName()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agent providers: %w", err)
	}

	return mapAgentProviders(items), nil
}

func (r *EntRepository) CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error) {
	exists, err := r.organizationIsActive(ctx, input.OrganizationID)
	if err != nil {
		return domain.AgentProvider{}, fmt.Errorf("check organization before creating agent provider: %w", err)
	}
	if !exists {
		return domain.AgentProvider{}, ErrNotFound
	}

	item, err := r.client.AgentProvider.Create().
		SetOrganizationID(input.OrganizationID).
		SetName(input.Name).
		SetAdapterType(input.AdapterType).
		SetCliCommand(input.CliCommand).
		SetCliArgs(pgarray.StringArray(input.CliArgs)).
		SetAuthConfig(input.AuthConfig).
		SetModelName(input.ModelName).
		SetModelTemperature(input.ModelTemperature).
		SetModelMaxTokens(input.ModelMaxTokens).
		SetCostPerInputToken(input.CostPerInputToken).
		SetCostPerOutputToken(input.CostPerOutputToken).
		Save(ctx)
	if err != nil {
		return domain.AgentProvider{}, mapWriteError("create agent provider", err)
	}

	return mapAgentProvider(item), nil
}

func (r *EntRepository) GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error) {
	item, err := r.client.AgentProvider.Get(ctx, id)
	if err != nil {
		return domain.AgentProvider{}, mapReadError("get agent provider", err)
	}

	return mapAgentProvider(item), nil
}

func (r *EntRepository) UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	item, err := r.client.AgentProvider.UpdateOneID(input.ID).
		SetOrganizationID(input.OrganizationID).
		SetName(input.Name).
		SetAdapterType(input.AdapterType).
		SetCliCommand(input.CliCommand).
		SetCliArgs(pgarray.StringArray(input.CliArgs)).
		SetAuthConfig(input.AuthConfig).
		SetModelName(input.ModelName).
		SetModelTemperature(input.ModelTemperature).
		SetModelMaxTokens(input.ModelMaxTokens).
		SetCostPerInputToken(input.CostPerInputToken).
		SetCostPerOutputToken(input.CostPerOutputToken).
		Save(ctx)
	if err != nil {
		return domain.AgentProvider{}, mapWriteError("update agent provider", err)
	}

	return mapAgentProvider(item), nil
}

func (r *EntRepository) ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error) {
	exists, err := r.client.Project.Query().
		Where(entproject.ID(projectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check project before listing agents: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	items, err := r.client.Agent.Query().
		Where(entagent.ProjectID(projectID)).
		Order(entagent.ByName()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	return mapAgents(items), nil
}

func (r *EntRepository) CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error) {
	project, err := r.client.Project.Get(ctx, input.ProjectID)
	if err != nil {
		return domain.Agent{}, mapReadError("get project for agent", err)
	}

	provider, err := r.client.AgentProvider.Get(ctx, input.ProviderID)
	if err != nil {
		return domain.Agent{}, mapReadError("get agent provider for agent", err)
	}
	if provider.OrganizationID != project.OrganizationID {
		return domain.Agent{}, fmt.Errorf("%w: provider organization must match project organization", ErrInvalidInput)
	}

	builder := r.client.Agent.Create().
		SetProjectID(input.ProjectID).
		SetProviderID(input.ProviderID).
		SetName(input.Name).
		SetStatus(input.Status).
		SetSessionID(input.SessionID).
		SetRuntimePhase(input.RuntimePhase).
		SetRuntimeControlState(input.RuntimeControlState).
		SetLastError(input.LastError).
		SetWorkspacePath(input.WorkspacePath).
		SetCapabilities(pgarray.StringArray(input.Capabilities)).
		SetTotalTokensUsed(input.TotalTokensUsed).
		SetTotalTicketsCompleted(input.TotalTicketsCompleted)
	if input.CurrentTicketID != nil {
		builder.SetCurrentTicketID(*input.CurrentTicketID)
	}
	if input.RuntimeStartedAt != nil {
		builder.SetRuntimeStartedAt(*input.RuntimeStartedAt)
	}
	if input.LastHeartbeatAt != nil {
		builder.SetLastHeartbeatAt(*input.LastHeartbeatAt)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Agent{}, mapWriteError("create agent", err)
	}

	return mapAgent(item), nil
}

func (r *EntRepository) GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	item, err := r.client.Agent.Get(ctx, id)
	if err != nil {
		return domain.Agent{}, mapReadError("get agent", err)
	}

	return mapAgent(item), nil
}

func (r *EntRepository) UpdateAgentRuntimeControlState(ctx context.Context, input domain.UpdateAgentRuntimeControlState) (domain.Agent, error) {
	item, err := r.client.Agent.UpdateOneID(input.ID).
		SetRuntimeControlState(input.RuntimeControlState).
		Save(ctx)
	if err != nil {
		return domain.Agent{}, mapWriteError("update agent runtime control state", err)
	}

	return mapAgent(item), nil
}

func (r *EntRepository) DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	item, err := r.client.Agent.Get(ctx, id)
	if err != nil {
		return domain.Agent{}, mapReadError("get agent before delete", err)
	}

	if err := r.client.Agent.DeleteOneID(id).Exec(ctx); err != nil {
		return domain.Agent{}, mapWriteError("delete agent", err)
	}

	return mapAgent(item), nil
}

func mapAgentProviders(items []*ent.AgentProvider) []domain.AgentProvider {
	providers := make([]domain.AgentProvider, 0, len(items))
	for _, item := range items {
		providers = append(providers, mapAgentProvider(item))
	}

	return providers
}

func mapAgentProvider(item *ent.AgentProvider) domain.AgentProvider {
	return domain.AgentProvider{
		ID:                 item.ID,
		OrganizationID:     item.OrganizationID,
		Name:               item.Name,
		AdapterType:        item.AdapterType,
		CliCommand:         item.CliCommand,
		CliArgs:            append([]string(nil), item.CliArgs...),
		AuthConfig:         cloneAnyMap(item.AuthConfig),
		ModelName:          item.ModelName,
		ModelTemperature:   item.ModelTemperature,
		ModelMaxTokens:     item.ModelMaxTokens,
		CostPerInputToken:  item.CostPerInputToken,
		CostPerOutputToken: item.CostPerOutputToken,
	}
}

func mapAgents(items []*ent.Agent) []domain.Agent {
	agents := make([]domain.Agent, 0, len(items))
	for _, item := range items {
		agents = append(agents, mapAgent(item))
	}

	return agents
}

func mapAgent(item *ent.Agent) domain.Agent {
	return domain.Agent{
		ID:                    item.ID,
		ProviderID:            item.ProviderID,
		ProjectID:             item.ProjectID,
		Name:                  item.Name,
		Status:                item.Status,
		CurrentTicketID:       item.CurrentTicketID,
		SessionID:             item.SessionID,
		RuntimePhase:          item.RuntimePhase,
		RuntimeControlState:   item.RuntimeControlState,
		RuntimeStartedAt:      cloneTimePointer(item.RuntimeStartedAt),
		LastError:             item.LastError,
		WorkspacePath:         item.WorkspacePath,
		Capabilities:          append([]string(nil), item.Capabilities...),
		TotalTokensUsed:       item.TotalTokensUsed,
		TotalTicketsCompleted: item.TotalTicketsCompleted,
		LastHeartbeatAt:       cloneTimePointer(item.LastHeartbeatAt),
	}
}

func cloneAnyMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}

	return cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}
