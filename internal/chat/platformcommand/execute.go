package platformcommand

import (
	"context"
	"fmt"

	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

type Executor struct {
	Tickets        TicketExecutor
	ProjectUpdates ProjectUpdateExecutor
}

func (e Executor) Execute(
	ctx context.Context,
	index int,
	command Command,
	resolved ResolvedCommand,
	executedBy string,
) ExecutionResult {
	result := ExecutionResult{
		CommandIndex: index,
		Command:      command.Payload(),
	}

	switch args := resolved.Args.(type) {
	case ResolvedProjectUpdateCreateArgs:
		if e.ProjectUpdates == nil {
			result.Ok = false
			result.Summary = fmt.Sprintf("%s failed.", command.Name)
			result.Detail = "project update service unavailable"
			return result
		}
		thread, err := e.ProjectUpdates.AddThread(ctx, projectUpdateInput(args, executedBy))
		if err != nil {
			result.Ok = false
			result.Summary = fmt.Sprintf("%s failed.", command.Name)
			result.Detail = err.Error()
			return result
		}
		result.Ok = true
		result.Summary = fmt.Sprintf("Created project update in %s.", args.ProjectName)
		result.Detail = thread.Title
		return result
	case ResolvedTicketUpdateArgs:
		if e.Tickets == nil {
			result.Ok = false
			result.Summary = fmt.Sprintf("%s failed.", command.Name)
			result.Detail = "ticket service unavailable"
			return result
		}
		ticketItem, err := e.Tickets.Update(ctx, ticketUpdateInput(args, executedBy))
		if err != nil {
			result.Ok = false
			result.Summary = fmt.Sprintf("%s failed.", command.Name)
			result.Detail = err.Error()
			return result
		}
		result.Ok = true
		result.Summary = fmt.Sprintf("Updated ticket %s.", ticketItem.Identifier)
		if args.StatusName != nil {
			result.Detail = fmt.Sprintf("Status: %s", *args.StatusName)
		}
		return result
	case ResolvedTicketCreateArgs:
		if e.Tickets == nil {
			result.Ok = false
			result.Summary = fmt.Sprintf("%s failed.", command.Name)
			result.Detail = "ticket service unavailable"
			return result
		}
		ticketItem, err := e.Tickets.Create(ctx, ticketCreateInput(args, executedBy))
		if err != nil {
			result.Ok = false
			result.Summary = fmt.Sprintf("%s failed.", command.Name)
			result.Detail = err.Error()
			return result
		}
		result.Ok = true
		result.Summary = fmt.Sprintf("Created ticket %s.", ticketItem.Identifier)
		result.Detail = ticketItem.Title
		return result
	default:
		result.Ok = false
		result.Summary = fmt.Sprintf("%s failed.", command.Name)
		result.Detail = fmt.Sprintf("unsupported resolved command args %T", resolved.Args)
		return result
	}
}

func projectUpdateInput(
	args ResolvedProjectUpdateCreateArgs,
	executedBy string,
) projectupdateservice.AddThreadInput {
	return projectupdateservice.AddThreadInput{
		ProjectID: args.ProjectID,
		Status:    args.Status,
		Title:     args.Title,
		Body:      args.Content,
		CreatedBy: executedBy,
	}
}

func ticketUpdateInput(args ResolvedTicketUpdateArgs, executedBy string) ticketservice.UpdateInput {
	input := ticketservice.UpdateInput{
		TicketID: args.TicketID,
		CreatedBy: ticketservice.Optional[string]{
			Set:   true,
			Value: executedBy,
		},
	}
	if args.Title != nil {
		input.Title = ticketservice.Optional[string]{Set: true, Value: *args.Title}
	}
	if args.Description != nil {
		input.Description = ticketservice.Optional[string]{Set: true, Value: *args.Description}
	}
	if args.StatusID != nil {
		input.StatusID = ticketservice.Optional[uuid.UUID]{Set: true, Value: *args.StatusID}
	}
	return input
}

func ticketCreateInput(args ResolvedTicketCreateArgs, executedBy string) ticketservice.CreateInput {
	return ticketservice.CreateInput{
		ProjectID:      args.ProjectID,
		Title:          args.Title,
		Description:    args.Description,
		StatusID:       args.StatusID,
		Type:           ticketservice.TypeFeature,
		CreatedBy:      executedBy,
		ParentTicketID: args.ParentTicketID,
	}
}
