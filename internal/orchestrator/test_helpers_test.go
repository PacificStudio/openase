package orchestrator

import (
	"context"

	"github.com/BetterAndBetterII/openase/ent"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func newTicketStatusService(client *ent.Client) *ticketstatus.Service {
	return ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
}

func listProjectStatusRuntimeSnapshots(ctx context.Context, client *ent.Client, projectID uuid.UUID) ([]ticketstatus.StatusRuntimeSnapshot, error) {
	return ticketstatus.ListProjectStatusRuntimeSnapshots(ctx, ticketstatusrepo.NewEntRepository(client), projectID)
}
