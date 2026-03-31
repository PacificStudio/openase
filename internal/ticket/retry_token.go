package ticket

import (
	"context"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/google/uuid"
)

// NewRetryToken returns a new retry-generation token for ticket retry state.
func NewRetryToken() string {
	return uuid.NewString()
}

// InstallRetryTokenHooks keeps retry token semantics consistent for direct ent mutations.
func InstallRetryTokenHooks(client *ent.Client) {
	if client == nil {
		return
	}

	client.Ticket.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			ticketMutation, ok := mutation.(*ent.TicketMutation)
			if !ok {
				return next.Mutate(ctx, mutation)
			}

			ensureTicketCreateRetryToken(ticketMutation)
			normalizeTicketStatusTransition(ticketMutation)

			return next.Mutate(ctx, mutation)
		})
	})
}

// ScheduleRetryOne rotates the retry token and records a delayed retry intent.
func ScheduleRetryOne(update *ent.TicketUpdateOne, nextRetryAt time.Time, pauseReason string) *ent.TicketUpdateOne {
	if update == nil {
		return nil
	}

	update.SetRetryToken(NewRetryToken()).
		SetNextRetryAt(nextRetryAt).
		SetRetryPaused(pauseReason != "")
	if pauseReason == "" {
		return update.ClearPauseReason()
	}

	return update.SetPauseReason(pauseReason)
}

// ScheduleRetry rotates the retry token and records a delayed retry intent.
func ScheduleRetry(update *ent.TicketUpdate, nextRetryAt time.Time, pauseReason string) *ent.TicketUpdate {
	if update == nil {
		return nil
	}

	update.
		SetRetryToken(NewRetryToken()).
		SetNextRetryAt(nextRetryAt).
		SetRetryPaused(pauseReason != "")
	if pauseReason == "" {
		return update.ClearPauseReason()
	}

	return update.SetPauseReason(pauseReason)
}

func ensureTicketCreateRetryToken(mutation *ent.TicketMutation) {
	if mutation == nil || !mutation.Op().Is(ent.OpCreate) {
		return
	}
	if _, ok := mutation.RetryToken(); ok {
		return
	}

	mutation.SetRetryToken(NewRetryToken())
}

func normalizeTicketStatusTransition(mutation *ent.TicketMutation) {
	if mutation == nil || !mutation.Op().Is(ent.OpUpdate|ent.OpUpdateOne) {
		return
	}
	if _, ok := mutation.StatusID(); !ok {
		return
	}
	if _, ok := mutation.RetryToken(); !ok {
		mutation.SetRetryToken(NewRetryToken())
	}

	mutation.SetConsecutiveErrors(0)
	mutation.ClearNextRetryAt()
	mutation.SetRetryPaused(false)
	mutation.ClearPauseReason()
}
