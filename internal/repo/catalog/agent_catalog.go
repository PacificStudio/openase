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
	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
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
		SetPermissionProfile(entagentprovider.PermissionProfile(normalizeProviderPermissionProfile(input.PermissionProfile).String())).
		SetCliCommand(input.CliCommand).
		SetCliArgs(pgarray.StringArray(input.CliArgs)).
		SetAuthConfig(input.AuthConfig).
		SetModelName(input.ModelName).
		SetNillableReasoningEffort(reasoningEffortStringPointer(input.ReasoningEffort)).
		SetModelTemperature(input.ModelTemperature).
		SetModelMaxTokens(input.ModelMaxTokens).
		SetMaxParallelRuns(input.MaxParallelRuns).
		SetCostPerInputToken(input.CostPerInputToken).
		SetCostPerOutputToken(input.CostPerOutputToken).
		SetPricingConfig(input.PricingConfig.ToMap()).
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
		SetPermissionProfile(entagentprovider.PermissionProfile(normalizeProviderPermissionProfile(input.PermissionProfile).String())).
		SetCliCommand(input.CliCommand).
		SetCliArgs(pgarray.StringArray(input.CliArgs)).
		SetAuthConfig(input.AuthConfig).
		SetModelName(input.ModelName).
		SetNillableReasoningEffort(reasoningEffortStringPointer(input.ReasoningEffort)).
		SetModelTemperature(input.ModelTemperature).
		SetModelMaxTokens(input.ModelMaxTokens).
		SetMaxParallelRuns(input.MaxParallelRuns).
		SetCostPerInputToken(input.CostPerInputToken).
		SetCostPerOutputToken(input.CostPerOutputToken).
		SetPricingConfig(input.PricingConfig.ToMap()).
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

	conflict, err := r.agentDeleteConflict(ctx, id)
	if err != nil {
		return domain.Agent{}, err
	}
	if conflict != nil {
		return domain.Agent{}, conflict
	}

	if err := r.client.Agent.DeleteOneID(id).Exec(ctx); err != nil {
		return domain.Agent{}, mapWriteError("delete agent", err)
	}

	return mapAgent(item, agentCurrentRunSummary{}), nil
}

func (r *EntRepository) agentDeleteConflict(ctx context.Context, agentID uuid.UUID) (*domain.AgentDeleteConflict, error) {
	activeRuns, err := r.client.AgentRun.Query().
		Where(
			entagentrun.AgentIDEQ(agentID),
			entagentrun.TerminalAtIsNil(),
		).
		Order(ent.Desc(entagentrun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active agent runs for delete: %w", err)
	}

	historicalRuns, err := r.client.AgentRun.Query().
		Where(
			entagentrun.AgentIDEQ(agentID),
			entagentrun.TerminalAtNotNil(),
		).
		Order(ent.Desc(entagentrun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list historical agent runs for delete: %w", err)
	}

	if len(activeRuns) == 0 && len(historicalRuns) == 0 {
		return nil, nil
	}

	conflict := &domain.AgentDeleteConflict{
		AgentID:        agentID,
		ActiveRuns:     make([]domain.AgentRunReference, 0, len(activeRuns)),
		HistoricalRuns: make([]domain.AgentRunReference, 0, len(historicalRuns)),
	}
	for _, item := range activeRuns {
		conflict.ActiveRuns = append(conflict.ActiveRuns, domain.AgentRunReference{
			ID:       item.ID,
			TicketID: item.TicketID,
			Status:   item.Status.String(),
		})
	}
	for _, item := range historicalRuns {
		conflict.HistoricalRuns = append(conflict.HistoricalRuns, domain.AgentRunReference{
			ID:       item.ID,
			TicketID: item.TicketID,
			Status:   item.Status.String(),
		})
	}
	return conflict, nil
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
		ID:                    item.ID,
		OrganizationID:        item.OrganizationID,
		MachineID:             item.MachineID,
		MachineName:           machineName,
		MachineHost:           machineHost,
		MachineStatus:         machineStatus,
		MachineSSHUser:        machineSSHUser,
		MachineWorkspaceRoot:  machineWorkspaceRoot,
		MachineAgentCLIPath:   machineAgentCLIPath,
		MachineResources:      machineResources,
		Name:                  item.Name,
		AdapterType:           toDomainAgentProviderAdapterType(item.AdapterType),
		PermissionProfile:     normalizeProviderPermissionProfile(domain.AgentProviderPermissionProfile(item.PermissionProfile)),
		CliCommand:            item.CliCommand,
		CliArgs:               append([]string(nil), item.CliArgs...),
		AuthConfig:            cloneAnyMap(item.AuthConfig),
		CLIRateLimit:          cloneAnyMap(item.CliRateLimit),
		CLIRateLimitUpdatedAt: cloneTimePointer(item.CliRateLimitUpdatedAt),
		ModelName:             item.ModelName,
		ReasoningEffort:       domain.ParseStoredAgentProviderReasoningEffort(item.ReasoningEffort),
		ModelTemperature:      item.ModelTemperature,
		ModelMaxTokens:        item.ModelMaxTokens,
		MaxParallelRuns:       item.MaxParallelRuns,
		CostPerInputToken:     item.CostPerInputToken,
		CostPerOutputToken:    item.CostPerOutputToken,
		PricingConfig:         parseProviderPricingConfig(item.PricingConfig, item.CostPerInputToken, item.CostPerOutputToken),
	}
}

func parseProviderPricingConfig(
	raw map[string]any,
	costPerInputToken float64,
	costPerOutputToken float64,
) pricing.ProviderModelPricingConfig {
	config, err := pricing.ParseRawProviderModelPricingConfig(raw, costPerInputToken, costPerOutputToken)
	if err != nil {
		return pricing.CustomFlatPricingConfig(costPerInputToken, costPerOutputToken)
	}
	return config
}

func normalizeProviderPermissionProfile(
	profile domain.AgentProviderPermissionProfile,
) domain.AgentProviderPermissionProfile {
	if !profile.IsValid() {
		return domain.DefaultAgentProviderPermissionProfile
	}
	return profile
}

func reasoningEffortStringPointer(
	value *domain.AgentProviderReasoningEffort,
) *string {
	if value == nil {
		return nil
	}
	copied := value.String()
	return &copied
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
	var completionSummaryStatus *domain.AgentRunCompletionSummaryStatus
	if item.CompletionSummaryStatus != nil {
		status := domain.AgentRunCompletionSummaryStatus(*item.CompletionSummaryStatus)
		completionSummaryStatus = &status
	}

	return domain.AgentRun{
		ID:                           item.ID,
		AgentID:                      item.AgentID,
		WorkflowID:                   item.WorkflowID,
		WorkflowVersionID:            cloneUUIDPointer(item.WorkflowVersionID),
		TicketID:                     item.TicketID,
		ProviderID:                   item.ProviderID,
		SkillVersionIDs:              parseUUIDArray(item.SkillVersionIds),
		Status:                       toDomainAgentRunStatus(item.Status),
		SessionID:                    item.SessionID,
		RuntimeStartedAt:             cloneTimePointer(item.RuntimeStartedAt),
		TerminalAt:                   cloneTimePointer(item.TerminalAt),
		LastError:                    item.LastError,
		LastHeartbeatAt:              cloneTimePointer(item.LastHeartbeatAt),
		InputTokens:                  item.InputTokens,
		OutputTokens:                 item.OutputTokens,
		CachedInputTokens:            item.CachedInputTokens,
		CacheCreationInputTokens:     item.CacheCreationInputTokens,
		ReasoningTokens:              item.ReasoningTokens,
		PromptTokens:                 item.PromptTokens,
		CandidateTokens:              item.CandidateTokens,
		ToolTokens:                   item.ToolTokens,
		TotalTokens:                  item.TotalTokens,
		CurrentStepStatus:            cloneStringPointer(item.CurrentStepStatus),
		CurrentStepSummary:           cloneStringPointer(item.CurrentStepSummary),
		CurrentStepChangedAt:         cloneTimePointer(item.CurrentStepChangedAt),
		CompletionSummaryStatus:      completionSummaryStatus,
		CompletionSummaryMarkdown:    cloneStringPointer(item.CompletionSummaryMarkdown),
		CompletionSummaryJSON:        cloneAnyMap(item.CompletionSummaryJSON),
		CompletionSummaryInput:       cloneAnyMap(item.CompletionSummaryInput),
		CompletionSummaryGeneratedAt: cloneTimePointer(item.CompletionSummaryGeneratedAt),
		CompletionSummaryError:       cloneStringPointer(item.CompletionSummaryError),
		CreatedAt:                    item.CreatedAt.UTC(),
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
