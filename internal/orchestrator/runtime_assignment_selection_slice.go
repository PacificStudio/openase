package orchestrator

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func (s runtimeAssignmentSelectionSlice) listAssignments(ctx context.Context, predicates ...predicate.Ticket) ([]runtimeAssignment, error) {
	l := s.launcher
	items, err := l.client.Ticket.Query().
		Where(predicates...).
		WithCurrentRun(func(query *ent.AgentRunQuery) {
			query.WithAgent()
		}).
		Order(ent.Asc(entticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	assignments := make([]runtimeAssignment, 0, len(items))
	for _, ticketItem := range items {
		if ticketItem.Edges.CurrentRun == nil || ticketItem.Edges.CurrentRun.Edges.Agent == nil {
			continue
		}
		assignments = append(assignments, runtimeAssignment{
			ticket: ticketItem,
			agent:  ticketItem.Edges.CurrentRun.Edges.Agent,
			run:    ticketItem.Edges.CurrentRun,
		})
	}
	return assignments, nil
}

func (s runtimeAssignmentSelectionSlice) loadAssignmentByRun(ctx context.Context, runID uuid.UUID) (runtimeAssignment, error) {
	assignments, err := s.listAssignments(ctx,
		entticket.CurrentRunIDEQ(runID),
	)
	if err != nil {
		return runtimeAssignment{}, err
	}
	if len(assignments) == 0 {
		return runtimeAssignment{}, nil
	}
	return assignments[0], nil
}

func (s runtimeAssignmentSelectionSlice) loadLaunchContext(ctx context.Context, agentID uuid.UUID, ticketID uuid.UUID) (runtimeLaunchContext, error) {
	l := s.launcher
	if agentID == uuid.Nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent id must not be empty")
	}
	if ticketID == uuid.Nil {
		return runtimeLaunchContext{}, fmt.Errorf("ticket id must not be empty")
	}

	loadedAgent, err := l.client.Agent.Query().
		Where(entagent.IDEQ(agentID)).
		WithProvider().
		WithProject(func(query *ent.ProjectQuery) {
			query.WithOrganization()
			query.WithRepos(func(repoQuery *ent.ProjectRepoQuery) {
				repoQuery.Order(entprojectrepo.ByName())
			})
		}).
		Only(ctx)
	if err != nil {
		return runtimeLaunchContext{}, fmt.Errorf("load runtime launch context for agent %s: %w", agentID, err)
	}
	if loadedAgent.Edges.Provider == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent provider must be loaded")
	}
	if loadedAgent.Edges.Project == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent project must be loaded")
	}
	if loadedAgent.Edges.Project.Edges.Organization == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent project organization must be loaded")
	}

	ticketItem, err := l.client.Ticket.Query().
		Where(entticket.IDEQ(ticketID)).
		WithRepoScopes(func(scopeQuery *ent.TicketRepoScopeQuery) {
			scopeQuery.Order(entticketreposcope.ByRepoID())
		}).
		Only(ctx)
	if err != nil {
		return runtimeLaunchContext{}, fmt.Errorf("load runtime launch ticket %s: %w", ticketID, err)
	}

	return runtimeLaunchContext{
		agent:        loadedAgent,
		project:      loadedAgent.Edges.Project,
		ticket:       ticketItem,
		projectRepos: loadedAgent.Edges.Project.Edges.Repos,
		ticketScopes: ticketItem.Edges.RepoScopes,
	}, nil
}

func (s runtimeAssignmentSelectionSlice) resolveLaunchMachine(ctx context.Context, launchContext runtimeLaunchContext) (catalogdomain.Machine, bool, error) {
	l := s.launcher
	machines, err := l.client.Machine.Query().
		Where(entmachine.OrganizationID(launchContext.project.OrganizationID)).
		Order(entmachine.ByName()).
		All(ctx)
	if err != nil {
		return catalogdomain.Machine{}, false, fmt.Errorf("list machines for runtime launch: %w", err)
	}

	providerItem := launchContext.agent.Edges.Provider
	if providerItem == nil {
		return catalogdomain.Machine{}, false, fmt.Errorf("agent provider must be loaded")
	}

	for _, machineItem := range machines {
		if machineItem.ID == providerItem.MachineID {
			mapped := mapRuntimeMachine(machineItem)
			return mapped, mapped.Host != catalogdomain.LocalMachineHost, nil
		}
	}

	return catalogdomain.Machine{}, false, fmt.Errorf("provider %s bound machine %s not found", providerItem.ID, providerItem.MachineID)
}
