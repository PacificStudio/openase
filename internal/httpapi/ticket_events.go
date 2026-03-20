package httpapi

import (
	"context"
	"fmt"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
)

var (
	ticketEventsTopic      = provider.MustParseTopic("ticket.events")
	ticketCreatedEventType = provider.MustParseEventType("ticket.created")
	ticketUpdatedEventType = provider.MustParseEventType("ticket.updated")
	ticketStatusEventType  = provider.MustParseEventType("ticket.status_changed")
)

func (s *Server) publishTicketEvent(ctx context.Context, eventType provider.EventType, ticket ticketservice.Ticket) error {
	if s.events == nil {
		return nil
	}

	event, err := provider.NewJSONEvent(ticketEventsTopic, eventType, map[string]any{
		"project_id": ticket.ProjectID.String(),
		"ticket":     mapTicketResponse(ticket),
	}, time.Now())
	if err != nil {
		return fmt.Errorf("build ticket event: %w", err)
	}

	if err := s.events.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish ticket event: %w", err)
	}

	return nil
}
