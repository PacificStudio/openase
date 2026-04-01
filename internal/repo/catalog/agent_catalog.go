package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
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
		WithMachine().
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

	machine, err := r.client.Machine.Get(ctx, input.MachineID)
	if err != nil {
		return domain.AgentProvider{}, mapReadError("get machine for agent provider", err)
	}
	if machine.OrganizationID != input.OrganizationID {
		return domain.AgentProvider{}, fmt.Errorf("%w: machine organization must match provider organization", ErrInvalidInput)
	}

	item, err := r.client.AgentProvider.Create().
		SetOrganizationID(input.OrganizationID).
		SetMachineID(input.MachineID).
		SetName(input.Name).
		SetAdapterType(toEntAgentProviderAdapterType(input.AdapterType)).
		SetCliCommand(input.CliCommand).
		SetCliArgs(pgarray.StringArray(input.CliArgs)).
		SetAuthConfig(input.AuthConfig).
		SetModelName(input.ModelName).
		SetModelTemperature(input.ModelTemperature).
		SetModelMaxTokens(input.ModelMaxTokens).
		SetMaxParallelRuns(input.MaxParallelRuns).
		SetCostPerInputToken(input.CostPerInputToken).
		SetCostPerOutputToken(input.CostPerOutputToken).
		Save(ctx)
	if err != nil {
		return domain.AgentProvider{}, mapWriteError("create agent provider", err)
	}

	return r.GetAgentProvider(ctx, item.ID)
}

func (r *EntRepository) GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error) {
	item, err := r.client.AgentProvider.Query().
		Where(entagentprovider.ID(id)).
		WithMachine().
		Only(ctx)
	if err != nil {
		return domain.AgentProvider{}, mapReadError("get agent provider", err)
	}

	return mapAgentProvider(item), nil
}

func (r *EntRepository) UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	machine, err := r.client.Machine.Get(ctx, input.MachineID)
	if err != nil {
		return domain.AgentProvider{}, mapReadError("get machine for agent provider update", err)
	}
	if machine.OrganizationID != input.OrganizationID {
		return domain.AgentProvider{}, fmt.Errorf("%w: machine organization must match provider organization", ErrInvalidInput)
	}

	item, err := r.client.AgentProvider.UpdateOneID(input.ID).
		SetOrganizationID(input.OrganizationID).
		SetMachineID(input.MachineID).
		SetName(input.Name).
		SetAdapterType(toEntAgentProviderAdapterType(input.AdapterType)).
		SetCliCommand(input.CliCommand).
		SetCliArgs(pgarray.StringArray(input.CliArgs)).
		SetAuthConfig(input.AuthConfig).
		SetModelName(input.ModelName).
		SetModelTemperature(input.ModelTemperature).
		SetModelMaxTokens(input.ModelMaxTokens).
		SetMaxParallelRuns(input.MaxParallelRuns).
		SetCostPerInputToken(input.CostPerInputToken).
		SetCostPerOutputToken(input.CostPerOutputToken).
		Save(ctx)
	if err != nil {
		return domain.AgentProvider{}, mapWriteError("update agent provider", err)
	}

	return r.GetAgentProvider(ctx, item.ID)
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
		WithProvider(func(query *ent.AgentProviderQuery) {
			query.WithMachine()
		}).
		WithProject(func(query *ent.ProjectQuery) {
			query.WithOrganization()
		}).
		Order(entagent.ByName()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	currentRuns, err := r.loadCurrentRunSummaries(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return mapAgents(items, currentRuns), nil
}

func (r *EntRepository) ListAgentRuns(ctx context.Context, projectID uuid.UUID) ([]domain.AgentRun, error) {
	exists, err := r.client.Project.Query().
		Where(entproject.ID(projectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check project before listing agent runs: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	items, err := r.client.AgentRun.Query().
		Where(entagentrun.HasTicketWith(entticket.ProjectIDEQ(projectID))).
		Order(ent.Desc(entagentrun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agent runs: %w", err)
	}

	return mapAgentRuns(items), nil
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
		SetRuntimeControlState(toEntAgentRuntimeControlState(input.RuntimeControlState)).
		SetTotalTokensUsed(input.TotalTokensUsed).
		SetTotalTicketsCompleted(input.TotalTicketsCompleted)

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Agent{}, mapWriteError("create agent", err)
	}

	return mapAgent(item, agentCurrentRunSummary{}), nil
}

func (r *EntRepository) GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	item, err := r.client.Agent.Query().
		Where(entagent.ID(id)).
		WithProvider(func(query *ent.AgentProviderQuery) {
			query.WithMachine()
		}).
		WithProject(func(query *ent.ProjectQuery) {
			query.WithOrganization()
		}).
		Only(ctx)
	if err != nil {
		return domain.Agent{}, mapReadError("get agent", err)
	}

	currentRun, err := r.loadCurrentRunSummaryForAgent(ctx, item.ProjectID, item.ID)
	if err != nil {
		return domain.Agent{}, err
	}

	return mapAgent(item, currentRun), nil
}

func (r *EntRepository) GetAgentRun(ctx context.Context, id uuid.UUID) (domain.AgentRun, error) {
	item, err := r.client.AgentRun.Get(ctx, id)
	if err != nil {
		return domain.AgentRun{}, mapReadError("get agent run", err)
	}

	return mapAgentRun(item), nil
}

func (r *EntRepository) UpdateAgent(ctx context.Context, input domain.UpdateAgent) (domain.Agent, error) {
	current, err := r.client.Agent.Get(ctx, input.ID)
	if err != nil {
		return domain.Agent{}, mapReadError("get agent for update", err)
	}
	if current.ProjectID != input.ProjectID {
		return domain.Agent{}, fmt.Errorf("%w: agent project must match update project", ErrInvalidInput)
	}

	project, err := r.client.Project.Get(ctx, input.ProjectID)
	if err != nil {
		return domain.Agent{}, mapReadError("get project for agent update", err)
	}

	provider, err := r.client.AgentProvider.Get(ctx, input.ProviderID)
	if err != nil {
		return domain.Agent{}, mapReadError("get agent provider for agent update", err)
	}
	if provider.OrganizationID != project.OrganizationID {
		return domain.Agent{}, fmt.Errorf("%w: provider organization must match project organization", ErrInvalidInput)
	}

	item, err := r.client.Agent.UpdateOneID(input.ID).
		SetProviderID(input.ProviderID).
		SetName(input.Name).
		Save(ctx)
	if err != nil {
		return domain.Agent{}, mapWriteError("update agent", err)
	}

	currentRun, err := r.loadCurrentRunSummaryForAgent(ctx, item.ProjectID, item.ID)
	if err != nil {
		return domain.Agent{}, err
	}

	return mapAgent(item, currentRun), nil
}

func (r *EntRepository) UpdateAgentRuntimeControlState(ctx context.Context, input domain.UpdateAgentRuntimeControlState) (domain.Agent, error) {
	item, err := r.client.Agent.UpdateOneID(input.ID).
		SetRuntimeControlState(toEntAgentRuntimeControlState(input.RuntimeControlState)).
		Save(ctx)
	if err != nil {
		return domain.Agent{}, mapWriteError("update agent runtime control state", err)
	}

	currentRun, err := r.loadCurrentRunSummaryForAgent(ctx, item.ProjectID, item.ID)
	if err != nil {
		return domain.Agent{}, err
	}

	return mapAgent(item, currentRun), nil
}

func (r *EntRepository) DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error) {
	item, err := r.client.Agent.Get(ctx, id)
	if err != nil {
		return domain.Agent{}, mapReadError("get agent before delete", err)
	}

	if err := r.client.Agent.DeleteOneID(id).Exec(ctx); err != nil {
		return domain.Agent{}, mapWriteError("delete agent", err)
	}

	return mapAgent(item, agentCurrentRunSummary{}), nil
}

func mapAgentProviders(items []*ent.AgentProvider) []domain.AgentProvider {
	providers := make([]domain.AgentProvider, 0, len(items))
	for _, item := range items {
		providers = append(providers, mapAgentProvider(item))
	}

	return providers
}

func mapAgentProvider(item *ent.AgentProvider) domain.AgentProvider {
	machineName := ""
	machineHost := ""
	machineStatus := domain.MachineStatus("")
	var machineSSHUser *string
	var machineWorkspaceRoot *string
	var machineAgentCLIPath *string
	machineResources := map[string]any{}
	if item.Edges.Machine != nil {
		machineName = item.Edges.Machine.Name
		machineHost = item.Edges.Machine.Host
		machineStatus = toDomainMachineStatus(item.Edges.Machine.Status)
		machineSSHUser = optionalString(item.Edges.Machine.SSHUser)
		machineWorkspaceRoot = optionalString(item.Edges.Machine.WorkspaceRoot)
		machineAgentCLIPath = optionalString(item.Edges.Machine.AgentCliPath)
		machineResources = cloneAnyMap(item.Edges.Machine.Resources)
	}

	return domain.AgentProvider{
		ID:                   item.ID,
		OrganizationID:       item.OrganizationID,
		MachineID:            item.MachineID,
		MachineName:          machineName,
		MachineHost:          machineHost,
		MachineStatus:        machineStatus,
		MachineSSHUser:       machineSSHUser,
		MachineWorkspaceRoot: machineWorkspaceRoot,
		MachineAgentCLIPath:  machineAgentCLIPath,
		MachineResources:     machineResources,
		Name:                 item.Name,
		AdapterType:          toDomainAgentProviderAdapterType(item.AdapterType),
		CliCommand:           item.CliCommand,
		CliArgs:              append([]string(nil), item.CliArgs...),
		AuthConfig:           cloneAnyMap(item.AuthConfig),
		ModelName:            item.ModelName,
		ModelTemperature:     item.ModelTemperature,
		ModelMaxTokens:       item.ModelMaxTokens,
		MaxParallelRuns:      item.MaxParallelRuns,
		CostPerInputToken:    item.CostPerInputToken,
		CostPerOutputToken:   item.CostPerOutputToken,
	}
}

type agentCurrentRunSummary struct {
	runs []*ent.AgentRun
}

func (r *EntRepository) loadCurrentRunSummaries(ctx context.Context, projectID uuid.UUID) (map[uuid.UUID]agentCurrentRunSummary, error) {
	tickets, err := r.client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(projectID),
			entticket.CurrentRunIDNotNil(),
		).
		WithCurrentRun().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load current agent runs for project %s: %w", projectID, err)
	}

	summaries := make(map[uuid.UUID]agentCurrentRunSummary, len(tickets))
	for _, ticketItem := range tickets {
		runItem := ticketItem.Edges.CurrentRun
		if runItem == nil {
			continue
		}
		current := summaries[runItem.AgentID]
		current.runs = append(current.runs, runItem)
		summaries[runItem.AgentID] = current
	}

	return summaries, nil
}

func (r *EntRepository) loadCurrentRunSummaryForAgent(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) (agentCurrentRunSummary, error) {
	summaries, err := r.loadCurrentRunSummaries(ctx, projectID)
	if err != nil {
		return agentCurrentRunSummary{}, err
	}

	return summaries[agentID], nil
}

func mapAgents(items []*ent.Agent, currentRuns map[uuid.UUID]agentCurrentRunSummary) []domain.Agent {
	agents := make([]domain.Agent, 0, len(items))
	for _, item := range items {
		agents = append(agents, mapAgent(item, currentRuns[item.ID]))
	}

	return agents
}

func mapAgent(item *ent.Agent, currentRun agentCurrentRunSummary) domain.Agent {
	return domain.Agent{
		ID:                    item.ID,
		ProviderID:            item.ProviderID,
		ProjectID:             item.ProjectID,
		Name:                  item.Name,
		RuntimeControlState:   toDomainAgentRuntimeControlState(item.RuntimeControlState),
		TotalTokensUsed:       item.TotalTokensUsed,
		TotalTicketsCompleted: item.TotalTicketsCompleted,
		Runtime:               domain.BuildAgentRuntimeSummary(mapAgentRunList(currentRun.runs), toDomainAgentRuntimeControlState(item.RuntimeControlState)),
	}
}

func mapAgentRuns(items []*ent.AgentRun) []domain.AgentRun {
	runs := make([]domain.AgentRun, 0, len(items))
	for _, item := range items {
		runs = append(runs, mapAgentRun(item))
	}

	return runs
}

func mapAgentRunList(items []*ent.AgentRun) []domain.AgentRun {
	if len(items) == 0 {
		return nil
	}

	runs := make([]domain.AgentRun, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		runs = append(runs, mapAgentRun(item))
	}

	return runs
}

func mapAgentRun(item *ent.AgentRun) domain.AgentRun {
	return domain.AgentRun{
		ID:                   item.ID,
		AgentID:              item.AgentID,
		WorkflowID:           item.WorkflowID,
		WorkflowVersionID:    cloneUUIDPointer(item.WorkflowVersionID),
		TicketID:             item.TicketID,
		ProviderID:           item.ProviderID,
		SkillVersionIDs:      parseUUIDArray(item.SkillVersionIds),
		Status:               toDomainAgentRunStatus(item.Status),
		SessionID:            item.SessionID,
		RuntimeStartedAt:     cloneTimePointer(item.RuntimeStartedAt),
		LastError:            item.LastError,
		LastHeartbeatAt:      cloneTimePointer(item.LastHeartbeatAt),
		CurrentStepStatus:    cloneStringPointer(item.CurrentStepStatus),
		CurrentStepSummary:   cloneStringPointer(item.CurrentStepSummary),
		CurrentStepChangedAt: cloneTimePointer(item.CurrentStepChangedAt),
		CreatedAt:            item.CreatedAt.UTC(),
	}
}

func cloneUUIDPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func parseUUIDArray(raw []string) []uuid.UUID {
	if len(raw) == 0 {
		return nil
	}

	parsed := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		id, err := uuid.Parse(strings.TrimSpace(item))
		if err != nil {
			continue
		}
		parsed = append(parsed, id)
	}
	return parsed
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
