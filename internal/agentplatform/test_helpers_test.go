package agentplatform

import (
	"github.com/BetterAndBetterII/openase/ent"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
)

func newTicketStatusService(client *ent.Client) *ticketstatus.Service {
	return ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
}
