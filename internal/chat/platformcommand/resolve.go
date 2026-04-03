package platformcommand

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

type Resolver struct {
	Catalog  CatalogResolver
	Tickets  TicketResolver
	Statuses StatusResolver
}

func (r Resolver) ResolveCommand(
	ctx context.Context,
	currentProject catalogdomain.Project,
	command Command,
) (ResolvedCommand, error) {
	switch args := command.Args.(type) {
	case ProjectUpdateCreateArgs:
		project, err := r.resolveProjectRef(ctx, currentProject, args.Project)
		if err != nil {
			return ResolvedCommand{}, err
		}
		status, err := resolveProjectUpdateStatus(args.Status)
		if err != nil {
			return ResolvedCommand{}, err
		}
		title := strings.TrimSpace(args.Title)
		if title == "" {
			title = deriveProjectUpdateTitle(args.Content)
		}
		return ResolvedCommand{
			Name: command.Name,
			Args: ResolvedProjectUpdateCreateArgs{
				ProjectID:   project.ID,
				ProjectName: project.Name,
				Content:     args.Content,
				Title:       title,
				Status:      status,
			},
		}, nil
	case TicketUpdateArgs:
		ticketItem, err := r.resolveTicketRef(ctx, currentProject.ID, args.Ticket)
		if err != nil {
			return ResolvedCommand{}, err
		}
		statusID, statusName, err := r.resolveStatusRef(ctx, currentProject.ID, args.Status)
		if err != nil {
			return ResolvedCommand{}, err
		}
		return ResolvedCommand{
			Name: command.Name,
			Args: ResolvedTicketUpdateArgs{
				TicketID:         ticketItem.ID,
				TicketIdentifier: ticketItem.Identifier,
				Title:            args.Title,
				Description:      args.Description,
				StatusID:         statusID,
				StatusName:       statusName,
			},
		}, nil
	case TicketCreateArgs:
		project, err := r.resolveProjectRef(ctx, currentProject, args.Project)
		if err != nil {
			return ResolvedCommand{}, err
		}
		statusID, statusName, err := r.resolveStatusRef(ctx, project.ID, args.Status)
		if err != nil {
			return ResolvedCommand{}, err
		}
		parentTicketID, parentIdentifier, err := r.resolveOptionalTicketRef(ctx, project.ID, args.ParentTicket)
		if err != nil {
			return ResolvedCommand{}, err
		}
		return ResolvedCommand{
			Name: command.Name,
			Args: ResolvedTicketCreateArgs{
				ProjectID:        project.ID,
				ProjectName:      project.Name,
				Title:            args.Title,
				Description:      args.Description,
				StatusID:         statusID,
				StatusName:       statusName,
				ParentTicketID:   parentTicketID,
				ParentIdentifier: parentIdentifier,
			},
		}, nil
	default:
		return ResolvedCommand{}, fmt.Errorf("unsupported command args %T", command.Args)
	}
}

func (r Resolver) resolveProjectRef(
	ctx context.Context,
	currentProject catalogdomain.Project,
	raw string,
) (catalogdomain.Project, error) {
	ref := normalizeRef(raw)
	if ref == "" || ref == "current" || ref == "this project" {
		return currentProject, nil
	}
	if projectMatches(currentProject, ref) {
		return currentProject, nil
	}
	if r.Catalog == nil {
		return catalogdomain.Project{}, fmt.Errorf("project %q could not be resolved", raw)
	}

	projects, err := r.Catalog.ListProjects(ctx, currentProject.OrganizationID)
	if err != nil {
		return catalogdomain.Project{}, fmt.Errorf("list projects: %w", err)
	}
	matches := make([]catalogdomain.Project, 0, 1)
	for _, project := range projects {
		if projectMatches(project, ref) {
			matches = append(matches, project)
		}
	}
	switch len(matches) {
	case 0:
		return catalogdomain.Project{}, fmt.Errorf("project %q could not be resolved", raw)
	case 1:
		return matches[0], nil
	default:
		return catalogdomain.Project{}, fmt.Errorf("project %q is ambiguous", raw)
	}
}

func (r Resolver) resolveTicketRef(
	ctx context.Context,
	projectID uuid.UUID,
	raw string,
) (ticketservice.Ticket, error) {
	if r.Tickets == nil {
		return ticketservice.Ticket{}, fmt.Errorf("ticket %q could not be resolved", raw)
	}
	ref := normalizeRef(raw)
	if ref == "" {
		return ticketservice.Ticket{}, fmt.Errorf("ticket must not be empty")
	}
	items, err := r.Tickets.List(ctx, ticketservice.ListInput{ProjectID: projectID})
	if err != nil {
		return ticketservice.Ticket{}, fmt.Errorf("list tickets: %w", err)
	}

	matches := make([]ticketservice.Ticket, 0, 1)
	for _, item := range items {
		if strings.EqualFold(item.ID.String(), ref) || normalizeRef(item.Identifier) == ref {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return ticketservice.Ticket{}, fmt.Errorf("ticket %q could not be resolved", raw)
	case 1:
		return matches[0], nil
	default:
		return ticketservice.Ticket{}, fmt.Errorf("ticket %q is ambiguous", raw)
	}
}

func (r Resolver) resolveOptionalTicketRef(
	ctx context.Context,
	projectID uuid.UUID,
	raw *string,
) (*uuid.UUID, *string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil, nil
	}
	item, err := r.resolveTicketRef(ctx, projectID, *raw)
	if err != nil {
		return nil, nil, err
	}
	identifier := item.Identifier
	return &item.ID, &identifier, nil
}

func (r Resolver) resolveStatusRef(
	ctx context.Context,
	projectID uuid.UUID,
	raw *string,
) (*uuid.UUID, *string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil, nil
	}
	if r.Statuses == nil {
		return nil, nil, fmt.Errorf("status %q could not be resolved", *raw)
	}
	ref := normalizeRef(*raw)
	result, err := r.Statuses.List(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("list statuses: %w", err)
	}

	type statusMatch struct {
		id   uuid.UUID
		name string
	}
	matches := make([]statusMatch, 0, 1)
	for _, item := range result.Statuses {
		if strings.EqualFold(item.ID.String(), ref) || normalizeRef(item.Name) == ref {
			matches = append(matches, statusMatch{id: item.ID, name: item.Name})
		}
	}
	switch len(matches) {
	case 0:
		return nil, nil, fmt.Errorf("status %q could not be resolved", *raw)
	case 1:
		return &matches[0].id, &matches[0].name, nil
	default:
		return nil, nil, fmt.Errorf("status %q is ambiguous", *raw)
	}
}

func resolveProjectUpdateStatus(raw string) (projectupdateservice.Status, error) {
	switch normalizeRef(raw) {
	case "", "ontrack", "on_track", "on track":
		return projectupdateservice.StatusOnTrack, nil
	case "atrisk", "at_risk", "at risk":
		return projectupdateservice.StatusAtRisk, nil
	case "offtrack", "off_track", "off track":
		return projectupdateservice.StatusOffTrack, nil
	default:
		return "", fmt.Errorf("project update status %q is unsupported", raw)
	}
}

func deriveProjectUpdateTitle(content string) string {
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if trimmed == "" {
			continue
		}
		if utf8.RuneCountInString(trimmed) <= 72 {
			return trimmed
		}
		runes := []rune(trimmed)
		return strings.TrimSpace(string(runes[:72])) + "..."
	}
	return "Project update from Project AI"
}

func projectMatches(project catalogdomain.Project, ref string) bool {
	candidates := []string{
		project.ID.String(),
		project.Name,
		project.Slug,
	}
	return slices.ContainsFunc(candidates, func(candidate string) bool {
		return normalizeRef(candidate) == ref
	})
}

func normalizeRef(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
