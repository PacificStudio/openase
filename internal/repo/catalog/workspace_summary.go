package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatconversationdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

func (r *EntRepository) GetWorkspaceDashboardSummary(ctx context.Context) (domain.WorkspaceDashboardSummary, error) {
	organizations, err := r.client.Organization.Query().
		Where(entorganization.StatusEQ(entorganization.StatusActive)).
		Order(entorganization.ByName()).
		All(ctx)
	if err != nil {
		return domain.WorkspaceDashboardSummary{}, fmt.Errorf("list organizations for workspace summary: %w", err)
	}
	if len(organizations) == 0 {
		return domain.WorkspaceDashboardSummary{Organizations: []domain.WorkspaceOrganizationSummary{}}, nil
	}

	orgIDs := organizationIDsFromEnt(organizations)
	projects, err := r.listProjectsForOrganizations(ctx, orgIDs)
	if err != nil {
		return domain.WorkspaceDashboardSummary{}, err
	}
	providers, err := r.listProvidersForOrganizations(ctx, orgIDs)
	if err != nil {
		return domain.WorkspaceDashboardSummary{}, err
	}
	agents, err := r.listAgentsForProjects(ctx, projectIDsFromEnt(projects))
	if err != nil {
		return domain.WorkspaceDashboardSummary{}, err
	}
	runningAgentIDs, err := r.loadRunningAgentIDs(ctx, projectIDsFromEnt(projects))
	if err != nil {
		return domain.WorkspaceDashboardSummary{}, err
	}
	tickets, err := r.listTicketsForProjects(ctx, projectIDsFromEnt(projects))
	if err != nil {
		return domain.WorkspaceDashboardSummary{}, err
	}
	todayStart := startOfUTCDay(time.Now().UTC())
	todayCostByProject, err := r.sumTodayProjectCostByProject(ctx, projectIDsFromEnt(projects), todayStart)
	if err != nil {
		return domain.WorkspaceDashboardSummary{}, err
	}

	summary := domain.WorkspaceDashboardSummary{
		OrganizationCount: len(organizations),
		Organizations:     make([]domain.WorkspaceOrganizationSummary, 0, len(organizations)),
	}
	orgSummaries := make(map[uuid.UUID]*domain.WorkspaceOrganizationSummary, len(organizations))
	for _, organization := range organizations {
		item := domain.WorkspaceOrganizationSummary{
			OrganizationID: organization.ID,
			Name:           organization.Name,
			Slug:           organization.Slug,
		}
		summary.Organizations = append(summary.Organizations, item)
		orgSummaries[organization.ID] = &summary.Organizations[len(summary.Organizations)-1]
	}

	projectOrgIDs := make(map[uuid.UUID]uuid.UUID, len(projects))
	for _, project := range projects {
		projectOrgIDs[project.ID] = project.OrganizationID
		summary.ProjectCount++
		if orgSummary := orgSummaries[project.OrganizationID]; orgSummary != nil {
			orgSummary.ProjectCount++
		}
	}

	for _, provider := range providers {
		summary.ProviderCount++
		if orgSummary := orgSummaries[provider.OrganizationID]; orgSummary != nil {
			orgSummary.ProviderCount++
		}
	}

	for _, agent := range agents {
		orgID, ok := projectOrgIDs[agent.ProjectID]
		if !ok {
			continue
		}
		summary.TotalTokens += agent.TotalTokensUsed
		if orgSummary := orgSummaries[orgID]; orgSummary != nil {
			orgSummary.TotalTokens += agent.TotalTokensUsed
		}
		if _, running := runningAgentIDs[agent.ID]; !running {
			continue
		}
		summary.RunningAgents++
		if orgSummary := orgSummaries[orgID]; orgSummary != nil {
			orgSummary.RunningAgents++
		}
	}

	for projectID, costUSD := range todayCostByProject {
		orgID, ok := projectOrgIDs[projectID]
		if !ok {
			continue
		}
		summary.TodayCost += costUSD
		if orgSummary := orgSummaries[orgID]; orgSummary != nil {
			orgSummary.TodayCost += costUSD
		}
	}

	for _, ticket := range tickets {
		orgID, ok := projectOrgIDs[ticket.ProjectID]
		if !ok {
			continue
		}
		statusStage := ""
		if ticket.Edges.Status != nil {
			statusStage = string(ticket.Edges.Status.Stage)
		}
		if !domain.IsTerminalTicketStatusStage(statusStage) {
			summary.ActiveTickets++
			if orgSummary := orgSummaries[orgID]; orgSummary != nil {
				orgSummary.ActiveTickets++
			}
		}
	}

	return summary, nil
}

func (r *EntRepository) GetOrganizationDashboardSummary(ctx context.Context, organizationID uuid.UUID) (domain.OrganizationDashboardSummary, error) {
	organization, err := r.getActiveOrganization(ctx, organizationID)
	if err != nil {
		return domain.OrganizationDashboardSummary{}, err
	}

	projects, err := r.listProjectsForOrganizations(ctx, []uuid.UUID{organization.ID})
	if err != nil {
		return domain.OrganizationDashboardSummary{}, err
	}
	providers, err := r.listProvidersForOrganizations(ctx, []uuid.UUID{organization.ID})
	if err != nil {
		return domain.OrganizationDashboardSummary{}, err
	}
	projectIDs := projectIDsFromEnt(projects)
	agents, err := r.listAgentsForProjects(ctx, projectIDs)
	if err != nil {
		return domain.OrganizationDashboardSummary{}, err
	}
	runningAgentIDs, err := r.loadRunningAgentIDs(ctx, projectIDs)
	if err != nil {
		return domain.OrganizationDashboardSummary{}, err
	}
	tickets, err := r.listTicketsForProjects(ctx, projectIDs)
	if err != nil {
		return domain.OrganizationDashboardSummary{}, err
	}
	todayStart := startOfUTCDay(time.Now().UTC())
	todayCostByProject, err := r.sumTodayProjectCostByProject(ctx, projectIDs, todayStart)
	if err != nil {
		return domain.OrganizationDashboardSummary{}, err
	}

	summary := domain.OrganizationDashboardSummary{
		OrganizationID: organization.ID,
		ProjectCount:   len(projects),
		ProviderCount:  len(providers),
		Projects:       make([]domain.OrganizationProjectSummary, 0, len(projects)),
	}
	projectSummaries := make(map[uuid.UUID]*domain.OrganizationProjectSummary, len(projects))
	for _, project := range projects {
		item := domain.OrganizationProjectSummary{
			ProjectID:   project.ID,
			Name:        project.Name,
			Description: project.Description,
			Status:      project.Status,
		}
		if domain.IsActiveProjectStatus(project.Status) {
			summary.ActiveProjectCount++
		}
		summary.Projects = append(summary.Projects, item)
		projectSummaries[project.ID] = &summary.Projects[len(summary.Projects)-1]
	}

	for _, agent := range agents {
		projectSummary := projectSummaries[agent.ProjectID]
		if projectSummary == nil {
			continue
		}
		projectSummary.TotalTokens += agent.TotalTokensUsed
		summary.TotalTokens += agent.TotalTokensUsed
		if _, running := runningAgentIDs[agent.ID]; !running {
			continue
		}
		projectSummary.RunningAgents++
		summary.RunningAgents++
	}

	for projectID, costUSD := range todayCostByProject {
		projectSummary := projectSummaries[projectID]
		if projectSummary == nil {
			continue
		}
		projectSummary.TodayCost += costUSD
		summary.TodayCost += costUSD
	}

	for _, ticket := range tickets {
		projectSummary := projectSummaries[ticket.ProjectID]
		if projectSummary == nil {
			continue
		}
		statusStage := ""
		if ticket.Edges.Status != nil {
			statusStage = string(ticket.Edges.Status.Stage)
		}
		if !domain.IsTerminalTicketStatusStage(statusStage) {
			projectSummary.ActiveTickets++
			summary.ActiveTickets++
		}
		updateLatestActivity(&projectSummary.LastActivityAt, ticket.CreatedAt.UTC())
	}

	return summary, nil
}

func (r *EntRepository) listProjectsForOrganizations(ctx context.Context, organizationIDs []uuid.UUID) ([]*ent.Project, error) {
	if len(organizationIDs) == 0 {
		return nil, nil
	}

	items, err := r.client.Project.Query().
		Where(entproject.OrganizationIDIn(organizationIDs...)).
		Order(entproject.ByName()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list projects for summary: %w", err)
	}

	return items, nil
}

func (r *EntRepository) sumTodayProjectCostByProject(ctx context.Context, projectIDs []uuid.UUID, since time.Time) (map[uuid.UUID]float64, error) {
	sums := make(map[uuid.UUID]float64, len(projectIDs))
	if len(projectIDs) == 0 {
		return sums, nil
	}

	items, err := r.client.ActivityEvent.Query().
		Where(
			entactivityevent.ProjectIDIn(projectIDs...),
			entactivityevent.EventTypeIn(ticketing.CostRecordedEventType, chatconversationdomain.CostRecordedEventType),
			entactivityevent.CreatedAtGTE(since.UTC()),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project cost events for summary: %w", err)
	}

	for _, item := range items {
		costUSD, ok := activityEventCostUSD(item.Metadata)
		if !ok {
			continue
		}
		sums[item.ProjectID] += costUSD
	}

	return sums, nil
}

func (r *EntRepository) listProvidersForOrganizations(ctx context.Context, organizationIDs []uuid.UUID) ([]*ent.AgentProvider, error) {
	if len(organizationIDs) == 0 {
		return nil, nil
	}

	items, err := r.client.AgentProvider.Query().
		Where(entagentprovider.OrganizationIDIn(organizationIDs...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list providers for summary: %w", err)
	}

	return items, nil
}

func (r *EntRepository) listAgentsForProjects(ctx context.Context, projectIDs []uuid.UUID) ([]*ent.Agent, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}

	items, err := r.client.Agent.Query().
		Where(entagent.ProjectIDIn(projectIDs...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents for summary: %w", err)
	}

	return items, nil
}

func (r *EntRepository) listTicketsForProjects(ctx context.Context, projectIDs []uuid.UUID) ([]*ent.Ticket, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}

	items, err := r.client.Ticket.Query().
		Where(entticket.ProjectIDIn(projectIDs...)).
		WithStatus().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets for summary: %w", err)
	}

	return items, nil
}

func (r *EntRepository) loadRunningAgentIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	runningAgentIDs := make(map[uuid.UUID]struct{})
	if len(projectIDs) == 0 {
		return runningAgentIDs, nil
	}

	tickets, err := r.client.Ticket.Query().
		Where(
			entticket.ProjectIDIn(projectIDs...),
			entticket.CurrentRunIDNotNil(),
		).
		WithCurrentRun().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load running agents for summary: %w", err)
	}

	runsByAgent := make(map[uuid.UUID][]*ent.AgentRun, len(tickets))
	for _, ticket := range tickets {
		run := ticket.Edges.CurrentRun
		if run == nil {
			continue
		}
		runsByAgent[run.AgentID] = append(runsByAgent[run.AgentID], run)
	}

	for agentID, runs := range runsByAgent {
		runtime := domain.BuildAgentRuntimeSummary(mapAgentRunList(runs), domain.AgentRuntimeControlStateActive)
		if runtime != nil && runtime.Status == domain.AgentStatusRunning {
			runningAgentIDs[agentID] = struct{}{}
		}
	}

	return runningAgentIDs, nil
}

func organizationIDsFromEnt(items []*ent.Organization) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		ids = append(ids, item.ID)
	}

	return ids
}

func projectIDsFromEnt(items []*ent.Project) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		ids = append(ids, item.ID)
	}

	return ids
}

func startOfUTCDay(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func updateLatestActivity(target **time.Time, candidate time.Time) {
	if *target == nil || candidate.After(**target) {
		value := candidate.UTC()
		*target = &value
	}
}

func activityEventCostUSD(metadata map[string]any) (float64, bool) {
	if metadata == nil {
		return 0, false
	}

	switch value := metadata["cost_usd"].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	default:
		return 0, false
	}
}
