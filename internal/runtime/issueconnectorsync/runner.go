package issueconnectorsync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	registrypkg "github.com/BetterAndBetterII/openase/internal/issueconnector"
	"github.com/BetterAndBetterII/openase/internal/orchestrator"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	issueconnectorservice "github.com/BetterAndBetterII/openase/internal/service/issueconnector"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

type Runner struct {
	syncer *orchestrator.ConnectorSyncer
}

func NewRunner(
	repository orchestrator.ConnectorRepository,
	registry *registrypkg.Registry,
	client *ent.Client,
	tickets *ticketservice.Service,
	statuses *ticketstatus.Service,
	logger *slog.Logger,
) *Runner {
	return &Runner{
		syncer: orchestrator.NewConnectorSyncer(
			repository,
			registry,
			newTicketSyncSink(client, tickets, statuses),
			logger,
		),
	}
}

func (r *Runner) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	if r == nil || r.syncer == nil {
		return
	}
	r.syncer.ConfigureGitHubCredentials(resolver)
}

func (r *Runner) SyncConnector(ctx context.Context, connectorID uuid.UUID) (issueconnectorservice.SyncReport, error) {
	if r == nil || r.syncer == nil {
		return issueconnectorservice.SyncReport{}, issueconnectorservice.ErrConnectorRuntimeAbsent
	}

	report, err := r.syncer.SyncConnector(ctx, connectorID)
	if err != nil {
		return issueconnectorservice.SyncReport{}, err
	}

	return issueconnectorservice.SyncReport{
		ConnectorsScanned: report.ConnectorsScanned,
		ConnectorsSynced:  report.ConnectorsSynced,
		ConnectorsFailed:  report.ConnectorsFailed,
		IssuesSynced:      report.IssuesSynced,
	}, nil
}

type ticketSyncSink struct {
	client   *ent.Client
	tickets  *ticketservice.Service
	statuses *ticketstatus.Service
}

func newTicketSyncSink(client *ent.Client, tickets *ticketservice.Service, statuses *ticketstatus.Service) *ticketSyncSink {
	return &ticketSyncSink{
		client:   client,
		tickets:  tickets,
		statuses: statuses,
	}
}

func (s *ticketSyncSink) SyncExternalIssue(ctx context.Context, connector domain.IssueConnector, issue domain.ExternalIssue) error {
	ticketID, currentTicket, err := s.findTicketByExternalID(ctx, connector.ProjectID, issue.ExternalID)
	if err != nil {
		return err
	}

	statusID, err := s.resolveStatusID(ctx, connector, issue.Status)
	if err != nil {
		return err
	}

	if ticketID == uuid.Nil {
		created, err := s.tickets.Create(ctx, ticketservice.CreateInput{
			ProjectID:   connector.ProjectID,
			Title:       strings.TrimSpace(issue.Title),
			Description: strings.TrimSpace(issue.Description),
			StatusID:    statusID,
			Priority:    mapExternalPriority(issue.Priority),
			Type:        entticket.TypeFeature,
			CreatedBy:   "connector:" + string(connector.Type),
			ExternalRef: strings.TrimSpace(issue.ExternalID),
		})
		if err != nil {
			return err
		}
		_, err = s.tickets.AddExternalLink(ctx, ticketservice.AddExternalLinkInput{
			TicketID:   created.ID,
			LinkType:   linkTypeForConnector(connector.Type),
			URL:        strings.TrimSpace(issue.ExternalURL),
			ExternalID: strings.TrimSpace(issue.ExternalID),
			Title:      strings.TrimSpace(issue.Title),
			Status:     strings.TrimSpace(issue.Status),
			Relation:   entticketexternallink.RelationRelated,
		})
		return err
	}

	update := ticketservice.UpdateInput{TicketID: ticketID}
	if title := strings.TrimSpace(issue.Title); title != "" && title != currentTicket.Title {
		update.Title = ticketservice.Some(title)
	}
	if description := strings.TrimSpace(issue.Description); description != currentTicket.Description {
		update.Description = ticketservice.Some(description)
	}
	if statusID != nil && currentTicket.StatusID != *statusID {
		update.StatusID = ticketservice.Some(*statusID)
	}
	if priority := mapExternalPriority(issue.Priority); priority != currentTicket.Priority {
		update.Priority = ticketservice.Some(priority)
	}

	if update.Title.Set || update.Description.Set || update.StatusID.Set || update.Priority.Set {
		if _, err := s.tickets.Update(ctx, update); err != nil {
			return err
		}
	}

	return s.updateExternalLinkSnapshot(ctx, connector.ProjectID, issue)
}

func (s *ticketSyncSink) ApplyWebhookEvent(ctx context.Context, connector domain.IssueConnector, event domain.WebhookEvent) error {
	return s.SyncExternalIssue(ctx, connector, event.Issue)
}

func (s *ticketSyncSink) findTicketByExternalID(
	ctx context.Context,
	projectID uuid.UUID,
	externalID string,
) (uuid.UUID, ticketservice.Ticket, error) {
	links, err := s.client.TicketExternalLink.Query().
		Where(
			entticketexternallink.ExternalIDEQ(strings.TrimSpace(externalID)),
			entticketexternallink.HasTicketWith(entticket.ProjectIDEQ(projectID)),
		).
		WithTicket().
		All(ctx)
	if err != nil {
		return uuid.Nil, ticketservice.Ticket{}, fmt.Errorf("query external link by external id: %w", err)
	}
	if len(links) == 0 {
		return uuid.Nil, ticketservice.Ticket{}, nil
	}
	if len(links) > 1 {
		return uuid.Nil, ticketservice.Ticket{}, fmt.Errorf("multiple tickets are linked to external issue %q", externalID)
	}

	item, err := s.tickets.Get(ctx, links[0].TicketID)
	if err != nil {
		return uuid.Nil, ticketservice.Ticket{}, err
	}

	return links[0].TicketID, item, nil
}

func (s *ticketSyncSink) resolveStatusID(
	ctx context.Context,
	connector domain.IssueConnector,
	externalStatus string,
) (*uuid.UUID, error) {
	mapped := strings.TrimSpace(connector.Config.MapStatus(externalStatus))
	if mapped == "" {
		return nil, nil
	}

	statusID, err := s.statuses.ResolveStatusIDByName(ctx, connector.ProjectID, mapped)
	if err != nil {
		return nil, err
	}

	return &statusID, nil
}

func (s *ticketSyncSink) updateExternalLinkSnapshot(ctx context.Context, projectID uuid.UUID, issue domain.ExternalIssue) error {
	links, err := s.client.TicketExternalLink.Query().
		Where(
			entticketexternallink.ExternalIDEQ(strings.TrimSpace(issue.ExternalID)),
			entticketexternallink.HasTicketWith(entticket.ProjectIDEQ(projectID)),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query external link snapshot: %w", err)
	}
	if len(links) == 0 {
		return nil
	}

	for _, link := range links {
		if err := s.client.TicketExternalLink.UpdateOneID(link.ID).
			SetURL(strings.TrimSpace(issue.ExternalURL)).
			SetTitle(strings.TrimSpace(issue.Title)).
			SetStatus(strings.TrimSpace(issue.Status)).
			Exec(ctx); err != nil {
			return fmt.Errorf("update external link snapshot: %w", err)
		}
	}

	return nil
}

func mapExternalPriority(raw string) entticket.Priority {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "urgent", "critical", "blocker":
		return entticket.PriorityUrgent
	case "high":
		return entticket.PriorityHigh
	case "low", "minor":
		return entticket.PriorityLow
	default:
		return entticket.PriorityMedium
	}
}

func linkTypeForConnector(connectorType domain.Type) entticketexternallink.LinkType {
	switch connectorType {
	case domain.TypeGitHub:
		return entticketexternallink.LinkTypeGithubIssue
	case domain.TypeGitLab:
		return entticketexternallink.LinkTypeGitlabIssue
	case domain.TypeJira:
		return entticketexternallink.LinkTypeJiraTicket
	default:
		return entticketexternallink.LinkTypeCustom
	}
}
