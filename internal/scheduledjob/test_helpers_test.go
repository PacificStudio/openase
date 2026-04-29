package scheduledjob

import (
	"github.com/BetterAndBetterII/openase/ent"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
)

func newTicketStatusService(client *ent.Client) *ticketstatus.Service {
	return ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
}

func newTicketService(client *ent.Client) *ticketservice.Service {
	return ticketservice.NewService(ticketservice.Dependencies{
		Activity: ticketrepo.NewActivityRepository(client),
		Query:    ticketrepo.NewQueryRepository(client),
		Command:  ticketrepo.NewCommandRepository(client),
		Link:     ticketrepo.NewLinkRepository(client),
		Comment:  ticketrepo.NewCommentRepository(client),
		Usage:    ticketrepo.NewUsageRepository(client),
		Runtime:  ticketrepo.NewRuntimeRepository(client),
	})
}
