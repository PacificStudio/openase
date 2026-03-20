package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	registrypkg "github.com/BetterAndBetterII/openase/internal/issueconnector"
	"github.com/google/uuid"
)

type ConnectorRepository interface {
	List(ctx context.Context) ([]domain.IssueConnector, error)
	Get(ctx context.Context, connectorID uuid.UUID) (domain.IssueConnector, error)
	Save(ctx context.Context, connector domain.IssueConnector) error
}

type ConnectorSyncSink interface {
	SyncExternalIssue(ctx context.Context, connector domain.IssueConnector, issue domain.ExternalIssue) error
	ApplyWebhookEvent(ctx context.Context, connector domain.IssueConnector, event domain.WebhookEvent) error
}

type ConnectorSyncReport struct {
	ConnectorsScanned int `json:"connectors_scanned"`
	ConnectorsSynced  int `json:"connectors_synced"`
	ConnectorsFailed  int `json:"connectors_failed"`
	IssuesSynced      int `json:"issues_synced"`
}

type WebhookResult struct {
	Applied    bool   `json:"applied"`
	Action     string `json:"action"`
	ExternalID string `json:"external_id"`
}

type SyncBackRequest struct {
	ConnectorID uuid.UUID
	Update      domain.SyncBackUpdate
}

type ConnectorSyncer struct {
	repo     ConnectorRepository
	registry *registrypkg.Registry
	sink     ConnectorSyncSink
	logger   *slog.Logger
	now      func() time.Time
}

func NewConnectorSyncer(
	repo ConnectorRepository,
	registry *registrypkg.Registry,
	sink ConnectorSyncSink,
	logger *slog.Logger,
) *ConnectorSyncer {
	if logger == nil {
		logger = slog.Default()
	}

	return &ConnectorSyncer{
		repo:     repo,
		registry: registry,
		sink:     sink,
		logger:   logger.With("component", "connector-syncer"),
		now:      time.Now,
	}
}

func (s *ConnectorSyncer) SyncAll(ctx context.Context) (ConnectorSyncReport, error) {
	report := ConnectorSyncReport{}
	if err := s.ensureAvailable(); err != nil {
		return report, err
	}

	connectors, err := s.repo.List(ctx)
	if err != nil {
		return report, fmt.Errorf("list issue connectors: %w", err)
	}
	report.ConnectorsScanned = len(connectors)

	now := s.now().UTC()
	for _, connector := range connectors {
		if !connector.CanPullAt(now) {
			continue
		}

		synced, syncErr := s.syncPull(ctx, connector, now)
		if syncErr != nil {
			report.ConnectorsFailed++
			s.logger.Warn("connector pull sync failed", "connector_id", connector.ID, "type", connector.Type, "error", syncErr)
			continue
		}
		report.ConnectorsSynced++
		report.IssuesSynced += synced
	}

	return report, nil
}

func (s *ConnectorSyncer) SyncConnector(ctx context.Context, connectorID uuid.UUID) (ConnectorSyncReport, error) {
	report := ConnectorSyncReport{}
	if err := s.ensureAvailable(); err != nil {
		return report, err
	}

	connector, err := s.repo.Get(ctx, connectorID)
	if err != nil {
		return report, fmt.Errorf("get issue connector %s: %w", connectorID, err)
	}
	report.ConnectorsScanned = 1

	now := s.now().UTC()
	synced, err := s.syncPull(ctx, connector, now)
	if err != nil {
		report.ConnectorsFailed = 1
		return report, err
	}

	report.ConnectorsSynced = 1
	report.IssuesSynced = synced
	return report, nil
}

func (s *ConnectorSyncer) HandleWebhook(
	ctx context.Context,
	connectorID uuid.UUID,
	headers http.Header,
	body []byte,
) (WebhookResult, error) {
	result := WebhookResult{}
	if err := s.ensureAvailable(); err != nil {
		return result, err
	}

	connector, err := s.repo.Get(ctx, connectorID)
	if err != nil {
		return result, fmt.Errorf("get issue connector %s: %w", connectorID, err)
	}
	if !connector.CanReceiveWebhook() {
		return result, fmt.Errorf("connector %s does not accept inbound webhook sync", connectorID)
	}

	impl, err := s.registry.Get(connector.Type)
	if err != nil {
		connector.RecordFailure(err)
		if saveErr := s.repo.Save(ctx, connector); saveErr != nil {
			return result, fmt.Errorf("save connector failure state: %w", saveErr)
		}
		return result, err
	}

	event, err := impl.ParseWebhook(ctx, headers, body)
	if err != nil {
		connector.RecordFailure(err)
		if saveErr := s.repo.Save(ctx, connector); saveErr != nil {
			return result, fmt.Errorf("save connector failure state: %w", saveErr)
		}
		return result, err
	}
	if event == nil {
		return result, fmt.Errorf("connector %s returned nil webhook event", connectorID)
	}

	result.Action = event.Action
	result.ExternalID = event.Issue.ExternalID
	if !connector.Config.Filters.Matches(event.Issue) {
		return result, nil
	}
	if err := s.sink.ApplyWebhookEvent(ctx, connector, *event); err != nil {
		connector.RecordFailure(err)
		if saveErr := s.repo.Save(ctx, connector); saveErr != nil {
			return result, fmt.Errorf("save connector failure state: %w", saveErr)
		}
		return result, err
	}

	connector.RecordSync(s.now().UTC(), 1)
	if err := s.repo.Save(ctx, connector); err != nil {
		return result, fmt.Errorf("save connector webhook sync state: %w", err)
	}

	result.Applied = true
	return result, nil
}

func (s *ConnectorSyncer) SyncBack(ctx context.Context, request SyncBackRequest) error {
	if err := s.ensureAvailable(); err != nil {
		return err
	}

	connector, err := s.repo.Get(ctx, request.ConnectorID)
	if err != nil {
		return fmt.Errorf("get issue connector %s: %w", request.ConnectorID, err)
	}
	if !connector.CanSyncBack() {
		return fmt.Errorf("connector %s does not allow sync back", request.ConnectorID)
	}

	impl, err := s.registry.Get(connector.Type)
	if err != nil {
		return err
	}
	if err := impl.SyncBack(ctx, connector.Config, request.Update); err != nil {
		connector.RecordFailure(err)
		if saveErr := s.repo.Save(ctx, connector); saveErr != nil {
			return fmt.Errorf("save connector failure state: %w", saveErr)
		}
		return err
	}

	connector.Status = domain.StatusActive
	connector.LastError = ""
	return s.repo.Save(ctx, connector)
}

func (s *ConnectorSyncer) ensureAvailable() error {
	if s == nil || s.repo == nil || s.registry == nil || s.sink == nil {
		return fmt.Errorf("connector syncer unavailable")
	}

	return nil
}

func (s *ConnectorSyncer) syncPull(
	ctx context.Context,
	connector domain.IssueConnector,
	now time.Time,
) (int, error) {
	if !connector.Config.SyncDirection.AllowsPull() {
		return 0, fmt.Errorf("connector %s does not allow pull sync", connector.ID)
	}
	if connector.Status != domain.StatusActive {
		return 0, fmt.Errorf("connector %s is not active", connector.ID)
	}

	impl, err := s.registry.Get(connector.Type)
	if err != nil {
		connector.RecordFailure(err)
		if saveErr := s.repo.Save(ctx, connector); saveErr != nil {
			return 0, fmt.Errorf("save connector failure state: %w", saveErr)
		}
		return 0, err
	}

	issues, err := impl.PullIssues(ctx, connector.Config, connector.LastSyncCursor())
	if err != nil {
		connector.RecordFailure(err)
		if saveErr := s.repo.Save(ctx, connector); saveErr != nil {
			return 0, fmt.Errorf("save connector failure state: %w", saveErr)
		}
		return 0, err
	}

	synced := 0
	for _, issue := range issues {
		if !connector.Config.Filters.Matches(issue) {
			continue
		}
		if err := s.sink.SyncExternalIssue(ctx, connector, issue); err != nil {
			connector.RecordFailure(err)
			if saveErr := s.repo.Save(ctx, connector); saveErr != nil {
				return synced, fmt.Errorf("save connector failure state: %w", saveErr)
			}
			return synced, err
		}
		synced++
	}

	connector.RecordSync(now, synced)
	if err := s.repo.Save(ctx, connector); err != nil {
		return synced, fmt.Errorf("save connector sync state: %w", err)
	}

	return synced, nil
}
