package httpapi

import (
	"context"
	"fmt"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
)

var (
	ticketEventsTopic      = provider.MustParseTopic("ticket.events")
	ticketCreatedEventType = activityevent.TypeTicketCreated
	ticketUpdatedEventType = activityevent.TypeTicketUpdated
	ticketStatusEventType  = activityevent.TypeTicketStatusChanged
	ticketRetryResumedType = activityevent.TypeTicketRetryResumed
)

func (s *Server) publishTicketEvent(
	ctx context.Context,
	eventType activityevent.Type,
	ticket ticketservice.Ticket,
) error {
	if _, err := activityevent.ParseRawType(eventType.String()); err != nil {
		return fmt.Errorf("parse ticket activity event type: %w", err)
	}

	var activityItem *activityEventResponse
	if s.ticketService != nil {
		recorded, err := s.ticketService.RecordActivityEvent(ctx, ticketservice.RecordActivityEventInput{
			ProjectID: ticket.ProjectID,
			TicketID:  &ticket.ID,
			EventType: eventType,
			Message:   buildTicketActivityMessage(eventType, ticket),
			Metadata:  buildTicketActivityMetadata(eventType, ticket),
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("record ticket activity event: %w", err)
		}
		mapped := mapActivityEventResponse(recorded)
		activityItem = &mapped
	}

	providerEventType, err := provider.ParseEventType(eventType.String())
	if err != nil {
		return fmt.Errorf("parse provider event type: %w", err)
	}

	if s.events == nil {
		return nil
	}

	event, err := provider.NewJSONEvent(ticketEventsTopic, providerEventType, map[string]any{
		"project_id": ticket.ProjectID.String(),
		"ticket":     mapTicketResponse(ticket),
	}, time.Now())
	if err != nil {
		return fmt.Errorf("build ticket event: %w", err)
	}

	if err := s.events.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish ticket event: %w", err)
	}
	if activityItem == nil {
		return nil
	}

	activityEvent, err := provider.NewJSONEvent(
		activityStreamTopic,
		providerEventType,
		map[string]any{"event": activityItem},
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("build activity event: %w", err)
	}
	if err := s.events.Publish(ctx, activityEvent); err != nil {
		return fmt.Errorf("publish activity event: %w", err)
	}

	return nil
}

func buildTicketActivityMessage(eventType activityevent.Type, ticket ticketservice.Ticket) string {
	switch eventType {
	case activityevent.TypeTicketCreated:
		return fmt.Sprintf("Created ticket %s", ticket.Identifier)
	case activityevent.TypeTicketStatusChanged:
		return fmt.Sprintf("Updated %s status to %s", ticket.Identifier, ticket.StatusName)
	case activityevent.TypeTicketRetryResumed:
		return fmt.Sprintf("Resumed retry for %s after repeated stalls", ticket.Identifier)
	default:
		return fmt.Sprintf("Updated ticket %s", ticket.Identifier)
	}
}

func buildTicketActivityMetadata(
	eventType activityevent.Type,
	ticket ticketservice.Ticket,
) map[string]any {
	metadata := map[string]any{
		"identifier": ticket.Identifier,
		"title":      ticket.Title,
	}

	switch eventType {
	case activityevent.TypeTicketCreated:
		metadata["created_by"] = ticket.CreatedBy
	case activityevent.TypeTicketStatusChanged:
		metadata["to_status_id"] = ticket.StatusID.String()
		metadata["to_status_name"] = ticket.StatusName
	case activityevent.TypeTicketRetryResumed:
		metadata["retry_paused"] = ticket.RetryPaused
		metadata["pause_reason"] = ticket.PauseReason
		metadata["changed_fields"] = []string{"retry"}
	default:
		metadata["changed_fields"] = []string{"ticket"}
	}

	return metadata
}
