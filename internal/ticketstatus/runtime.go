package ticketstatus

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketstage "github.com/BetterAndBetterII/openase/ent/ticketstage"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	"github.com/google/uuid"
)

// StageRuntimeSnapshot describes the live active-run occupancy for a ticket stage.
type StageRuntimeSnapshot struct {
	StageID       uuid.UUID `json:"stage_id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	MaxActiveRuns *int      `json:"max_active_runs,omitempty"`
	ActiveRuns    int       `json:"active_runs"`
}

// ListProjectStageRuntimeSnapshots returns ordered runtime occupancy for all stages in a project.
func ListProjectStageRuntimeSnapshots(ctx context.Context, client *ent.Client, projectID uuid.UUID) ([]StageRuntimeSnapshot, error) {
	if client == nil {
		return nil, ErrUnavailable
	}

	stages, err := client.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project ticket stages: %w", err)
	}

	activeRunsByStageID, err := countProjectStageActiveRuns(ctx, client, projectID)
	if err != nil {
		return nil, err
	}

	return buildStageRuntimeSnapshots(stages, activeRunsByStageID), nil
}

// ListStageRuntimeSnapshots returns ordered runtime occupancy for all stages across projects.
func ListStageRuntimeSnapshots(ctx context.Context, client *ent.Client) ([]StageRuntimeSnapshot, error) {
	if client == nil {
		return nil, ErrUnavailable
	}

	stages, err := client.TicketStage.Query().
		Order(ent.Asc(entticketstage.FieldProjectID), ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket stages: %w", err)
	}

	activeRunsByStageID, err := countStageActiveRunsAcrossProjects(ctx, client)
	if err != nil {
		return nil, err
	}

	return buildStageRuntimeSnapshots(stages, activeRunsByStageID), nil
}

func buildStageRuntimeSnapshots(stages []*ent.TicketStage, activeRunsByStageID map[uuid.UUID]int) []StageRuntimeSnapshot {
	snapshots := make([]StageRuntimeSnapshot, 0, len(stages))
	for _, stage := range stages {
		snapshots = append(snapshots, StageRuntimeSnapshot{
			StageID:       stage.ID,
			ProjectID:     stage.ProjectID,
			Key:           stage.Key,
			Name:          stage.Name,
			MaxActiveRuns: cloneIntPointer(stage.MaxActiveRuns),
			ActiveRuns:    activeRunsByStageID[stage.ID],
		})
	}
	return snapshots
}

func stageActiveRunsByID(snapshots []StageRuntimeSnapshot) map[uuid.UUID]int {
	counts := make(map[uuid.UUID]int, len(snapshots))
	for _, snapshot := range snapshots {
		counts[snapshot.StageID] = snapshot.ActiveRuns
	}
	return counts
}

func countProjectStageActiveRuns(ctx context.Context, client *ent.Client, projectID uuid.UUID) (map[uuid.UUID]int, error) {
	return countStageActiveRunsFromTickets(ctx, client,
		entticket.ProjectIDEQ(projectID),
		entticket.CurrentRunIDNotNil(),
	)
}

func countStageActiveRunsAcrossProjects(ctx context.Context, client *ent.Client) (map[uuid.UUID]int, error) {
	return countStageActiveRunsFromTickets(ctx, client, entticket.CurrentRunIDNotNil())
}

func countStageActiveRuns(ctx context.Context, client *ent.Client, projectID uuid.UUID, stageID uuid.UUID) (int, error) {
	count, err := client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(projectID),
			entticket.CurrentRunIDNotNil(),
			entticket.HasStatusWith(entticketstatus.HasStageWith(entticketstage.IDEQ(stageID))),
		).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active runs in stage %s: %w", stageID, err)
	}
	return count, nil
}

func countStageActiveRunsFromTickets(ctx context.Context, client *ent.Client, predicates ...predicate.Ticket) (map[uuid.UUID]int, error) {
	tickets, err := client.Ticket.Query().
		Where(predicates...).
		WithStatus().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active tickets for stage occupancy: %w", err)
	}

	counts := make(map[uuid.UUID]int)
	for _, ticket := range tickets {
		status := ticket.Edges.Status
		if status == nil || status.StageID == nil {
			continue
		}
		counts[*status.StageID]++
	}

	return counts, nil
}
