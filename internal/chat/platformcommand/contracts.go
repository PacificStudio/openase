package platformcommand

import (
	"context"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	ticketstatusservice "github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

const ProposalType = "platform_command_proposal"

type CommandName string

const (
	CommandProjectUpdateCreate CommandName = "project_update.create"
	CommandTicketUpdate        CommandName = "ticket.update"
	CommandTicketCreate        CommandName = "ticket.create"
)

type Proposal struct {
	Summary  string
	Commands []Command
}

type Command struct {
	Name CommandName
	Args any
}

type ProjectUpdateCreateArgs struct {
	Project string
	Content string
	Title   string
	Status  string
}

type TicketUpdateArgs struct {
	Ticket      string
	Title       *string
	Description *string
	Status      *string
}

type TicketCreateArgs struct {
	Project      string
	Title        string
	Description  string
	Status       *string
	ParentTicket *string
}

type ResolvedCommand struct {
	Name CommandName
	Args any
}

type ResolvedProjectUpdateCreateArgs struct {
	ProjectID   uuid.UUID
	ProjectName string
	Content     string
	Title       string
	Status      projectupdateservice.Status
}

type ResolvedTicketUpdateArgs struct {
	TicketID         uuid.UUID
	TicketIdentifier string
	Title            *string
	Description      *string
	StatusID         *uuid.UUID
	StatusName       *string
}

type ResolvedTicketCreateArgs struct {
	ProjectID        uuid.UUID
	ProjectName      string
	Title            string
	Description      string
	StatusID         *uuid.UUID
	StatusName       *string
	ParentTicketID   *uuid.UUID
	ParentIdentifier *string
}

type ExecutionResult struct {
	CommandIndex int            `json:"command_index"`
	Command      map[string]any `json:"command"`
	Ok           bool           `json:"ok"`
	Summary      string         `json:"summary"`
	Detail       string         `json:"detail,omitempty"`
}

type CatalogResolver interface {
	GetProject(ctx context.Context, id uuid.UUID) (catalogdomain.Project, error)
	ListProjects(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.Project, error)
}

type TicketResolver interface {
	List(ctx context.Context, input ticketservice.ListInput) ([]ticketservice.Ticket, error)
}

type StatusResolver interface {
	List(ctx context.Context, projectID uuid.UUID) (ticketstatusservice.ListResult, error)
}

type TicketExecutor interface {
	Create(ctx context.Context, input ticketservice.CreateInput) (ticketservice.Ticket, error)
	Update(ctx context.Context, input ticketservice.UpdateInput) (ticketservice.Ticket, error)
}

type ProjectUpdateExecutor interface {
	AddThread(ctx context.Context, input projectupdateservice.AddThreadInput) (projectupdateservice.Thread, error)
}

func (c Command) Payload() map[string]any {
	payload := map[string]any{
		"command": string(c.Name),
	}
	switch args := c.Args.(type) {
	case ProjectUpdateCreateArgs:
		payload["args"] = map[string]any{
			"project": valueOrEmpty(args.Project),
			"content": args.Content,
			"title":   valueOrEmpty(args.Title),
			"status":  valueOrEmpty(args.Status),
		}
	case TicketUpdateArgs:
		argsPayload := map[string]any{"ticket": args.Ticket}
		if args.Title != nil {
			argsPayload["title"] = *args.Title
		}
		if args.Description != nil {
			argsPayload["description"] = *args.Description
		}
		if args.Status != nil {
			argsPayload["status"] = *args.Status
		}
		payload["args"] = argsPayload
	case TicketCreateArgs:
		argsPayload := map[string]any{
			"project":     valueOrEmpty(args.Project),
			"title":       args.Title,
			"description": args.Description,
		}
		if args.Status != nil {
			argsPayload["status"] = *args.Status
		}
		if args.ParentTicket != nil {
			argsPayload["parent_ticket"] = *args.ParentTicket
		}
		payload["args"] = argsPayload
	default:
		payload["args"] = map[string]any{}
	}
	return payload
}

func valueOrEmpty(value string) string {
	return strings.TrimSpace(value)
}
