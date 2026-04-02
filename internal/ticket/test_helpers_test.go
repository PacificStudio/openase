package ticket

import (
	"github.com/BetterAndBetterII/openase/ent"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
)

func newTicketStatusService(client *ent.Client) *ticketstatus.Service {
	return ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
}

func newTicketService(client *ent.Client) *Service {
	return NewService(ticketrepo.NewEntRepository(client))
}
