package httpapi

import (
	"context"
	"fmt"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
)

var (
	ticketEventsTopic      = provider.MustParseTopic("ticket.events")
	ticketCreatedEventType = activityevent.TypeTicketCreated
	ticketUpdatedEventType = activityevent.TypeTicketUpdated
	ticketArchivedType     = activityevent.TypeTicketArchived
	ticketUnarchivedType   = activityevent.TypeTicketUnarchived
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
	createdAt := time.Now().UTC()
	if s.activityEmitter != nil {
		if _, err := s.activityEmitter.Emit(ctx, activitysvc.RecordInput{
			ProjectID: ticket.ProjectID,
			TicketID:  &ticket.ID,
			EventType: eventType,
			Message:   buildTicketActivityMessage(eventType, ticket),
			Metadata:  buildTicketActivityMetadata(eventType, ticket),
			CreatedAt: createdAt,
		}); err != nil {
			return fmt.Errorf("record ticket activity event: %w", err)
		}
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
	}, createdAt)
	if err != nil {
		return fmt.Errorf("build ticket event: %w", err)
	}

	if err := s.events.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish ticket event: %w", err)
	}

	return nil
}

func buildTicketActivityMessage(eventType activityevent.Type, ticket ticketservice.Ticket) string {
	switch eventType {
	case activityevent.TypeTicketCreated:
		return fmt.Sprintf("Created ticket %s", ticket.Identifier)
	case activityevent.TypeTicketArchived:
		return fmt.Sprintf("Archived ticket %s", ticket.Identifier)
	case activityevent.TypeTicketUnarchived:
		return fmt.Sprintf("Unarchived ticket %s", ticket.Identifier)
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
	case activityevent.TypeTicketArchived, activityevent.TypeTicketUnarchived:
		metadata["archived"] = ticket.Archived
		metadata["changed_fields"] = []string{"archived"}
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

func ticketMutationEventType(input ticketservice.UpdateInput) activityevent.Type {
	if input.Archived.Set {
		if input.Archived.Value {
			return ticketArchivedType
		}
		return ticketUnarchivedType
	}
	if input.StatusID.Set {
		return ticketStatusEventType
	}
	return ticketUpdatedEventType
}
