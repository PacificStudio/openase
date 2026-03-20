package orchestrator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	registrypkg "github.com/BetterAndBetterII/openase/internal/issueconnector"
	"github.com/google/uuid"
)

func TestConnectorSyncerSyncAllPullsDueActiveConnectors(t *testing.T) {
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	repo := &stubConnectorRepo{
		connectors: map[uuid.UUID]domain.IssueConnector{},
	}
	activeID := uuid.New()
	pausedID := uuid.New()
	repo.connectors[activeID] = domain.IssueConnector{
		ID:        activeID,
		ProjectID: uuid.New(),
		Type:      domain.TypeGitHub,
		Name:      "GitHub Issues",
		Status:    domain.StatusActive,
		Config: domain.Config{
			Type:          domain.TypeGitHub,
			PollInterval:  5 * time.Minute,
			SyncDirection: domain.SyncDirectionBidirectional,
			Filters:       domain.ParseFilters(domain.FiltersInput{Labels: []string{"openase"}}),
		},
	}
	repo.connectors[pausedID] = domain.IssueConnector{
		ID:        pausedID,
		ProjectID: uuid.New(),
		Type:      domain.TypeGitHub,
		Name:      "Paused",
		Status:    domain.StatusPaused,
		Config: domain.Config{
			Type:          domain.TypeGitHub,
			PollInterval:  5 * time.Minute,
			SyncDirection: domain.SyncDirectionBidirectional,
		},
	}

	registry, err := registrypkg.NewRegistry(stubIssueConnector{
		id:   "github",
		name: "GitHub",
		pullIssues: []domain.ExternalIssue{
			{ExternalID: "1", Title: "match", Labels: []string{"openase"}, Status: "open"},
			{ExternalID: "2", Title: "skip", Labels: []string{"ignore"}, Status: "open"},
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	sink := &stubConnectorSink{}
	syncer := NewConnectorSyncer(repo, registry, sink, slog.New(slog.NewTextHandler(io.Discard, nil)))
	syncer.now = func() time.Time { return now }

	report, err := syncer.SyncAll(context.Background())
	if err != nil {
		t.Fatalf("SyncAll returned error: %v", err)
	}

	if report.ConnectorsScanned != 2 || report.ConnectorsSynced != 1 || report.ConnectorsFailed != 0 || report.IssuesSynced != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(sink.syncedIssues) != 1 || sink.syncedIssues[0].ExternalID != "1" {
		t.Fatalf("unexpected synced issues: %+v", sink.syncedIssues)
	}

	saved := repo.connectors[activeID]
	if saved.Status != domain.StatusActive || saved.LastSyncAt == nil || saved.LastError != "" {
		t.Fatalf("unexpected connector state after sync: %+v", saved)
	}
}

func TestConnectorSyncerHandleWebhookParsesAndAppliesEvent(t *testing.T) {
	connectorID := uuid.New()
	repo := &stubConnectorRepo{
		connectors: map[uuid.UUID]domain.IssueConnector{
			connectorID: {
				ID:        connectorID,
				ProjectID: uuid.New(),
				Type:      domain.TypeGitHub,
				Name:      "GitHub Issues",
				Status:    domain.StatusActive,
				Config: domain.Config{
					Type:          domain.TypeGitHub,
					PollInterval:  5 * time.Minute,
					SyncDirection: domain.SyncDirectionBidirectional,
				},
			},
		},
	}
	registry, err := registrypkg.NewRegistry(stubIssueConnector{
		id:   "github",
		name: "GitHub",
		webhookEvent: &domain.WebhookEvent{
			Action: "created",
			Issue: domain.ExternalIssue{
				ExternalID: "GH-42",
				Title:      "connector event",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	sink := &stubConnectorSink{}
	syncer := NewConnectorSyncer(repo, registry, sink, slog.New(slog.NewTextHandler(io.Discard, nil)))
	result, err := syncer.HandleWebhook(context.Background(), connectorID, http.Header{"X-Test": []string{"1"}}, []byte(`{}`))
	if err != nil {
		t.Fatalf("HandleWebhook returned error: %v", err)
	}

	if !result.Applied || result.Action != "created" || result.ExternalID != "GH-42" {
		t.Fatalf("unexpected webhook result: %+v", result)
	}
	if len(sink.events) != 1 || sink.events[0].Issue.ExternalID != "GH-42" {
		t.Fatalf("unexpected applied events: %+v", sink.events)
	}
}

func TestConnectorSyncerSyncBackRequiresBidirectionalConnector(t *testing.T) {
	connectorID := uuid.New()
	repo := &stubConnectorRepo{
		connectors: map[uuid.UUID]domain.IssueConnector{
			connectorID: {
				ID:        connectorID,
				ProjectID: uuid.New(),
				Type:      domain.TypeGitHub,
				Name:      "GitHub Issues",
				Status:    domain.StatusActive,
				Config: domain.Config{
					Type:          domain.TypeGitHub,
					PollInterval:  5 * time.Minute,
					SyncDirection: domain.SyncDirectionPullOnly,
				},
			},
		},
	}
	registry, err := registrypkg.NewRegistry(stubIssueConnector{id: "github", name: "GitHub"})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	syncer := NewConnectorSyncer(repo, registry, &stubConnectorSink{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	err = syncer.SyncBack(context.Background(), SyncBackRequest{
		ConnectorID: connectorID,
		Update: domain.SyncBackUpdate{
			ExternalID: "GH-42",
			Action:     domain.SyncBackActionAddComment,
			Comment:    "hello",
		},
	})
	if err == nil || err.Error() != "connector "+connectorID.String()+" does not allow sync back" {
		t.Fatalf("expected sync back direction error, got %v", err)
	}
}

func TestConnectorSyncerRecordsFailureWhenPullFails(t *testing.T) {
	connectorID := uuid.New()
	repo := &stubConnectorRepo{
		connectors: map[uuid.UUID]domain.IssueConnector{
			connectorID: {
				ID:        connectorID,
				ProjectID: uuid.New(),
				Type:      domain.TypeGitHub,
				Name:      "GitHub Issues",
				Status:    domain.StatusActive,
				Config: domain.Config{
					Type:          domain.TypeGitHub,
					PollInterval:  5 * time.Minute,
					SyncDirection: domain.SyncDirectionBidirectional,
				},
			},
		},
	}
	registry, err := registrypkg.NewRegistry(stubIssueConnector{
		id:        "github",
		name:      "GitHub",
		pullError: errors.New("upstream unavailable"),
	})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	syncer := NewConnectorSyncer(repo, registry, &stubConnectorSink{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	report, err := syncer.SyncConnector(context.Background(), connectorID)
	if err == nil || err.Error() != "upstream unavailable" {
		t.Fatalf("expected pull error, got report=%+v err=%v", report, err)
	}
	if report.ConnectorsFailed != 1 {
		t.Fatalf("ConnectorsFailed = %d, want 1", report.ConnectorsFailed)
	}

	saved := repo.connectors[connectorID]
	if saved.Status != domain.StatusError || saved.LastError != "upstream unavailable" || saved.Stats.FailedCount != 1 {
		t.Fatalf("unexpected connector failure state: %+v", saved)
	}
}

type stubConnectorRepo struct {
	connectors map[uuid.UUID]domain.IssueConnector
}

func (s *stubConnectorRepo) List(context.Context) ([]domain.IssueConnector, error) {
	items := make([]domain.IssueConnector, 0, len(s.connectors))
	for _, connector := range s.connectors {
		items = append(items, connector)
	}
	return items, nil
}

func (s *stubConnectorRepo) Get(_ context.Context, connectorID uuid.UUID) (domain.IssueConnector, error) {
	connector, ok := s.connectors[connectorID]
	if !ok {
		return domain.IssueConnector{}, errors.New("not found")
	}
	return connector, nil
}

func (s *stubConnectorRepo) Save(_ context.Context, connector domain.IssueConnector) error {
	s.connectors[connector.ID] = connector
	return nil
}

type stubConnectorSink struct {
	syncedIssues []domain.ExternalIssue
	events       []domain.WebhookEvent
}

func (s *stubConnectorSink) SyncExternalIssue(_ context.Context, _ domain.IssueConnector, issue domain.ExternalIssue) error {
	s.syncedIssues = append(s.syncedIssues, issue)
	return nil
}

func (s *stubConnectorSink) ApplyWebhookEvent(_ context.Context, _ domain.IssueConnector, event domain.WebhookEvent) error {
	s.events = append(s.events, event)
	return nil
}

type stubIssueConnector struct {
	id           string
	name         string
	pullIssues   []domain.ExternalIssue
	pullError    error
	webhookEvent *domain.WebhookEvent
	webhookError error
	syncBackErr  error
}

func (s stubIssueConnector) ID() string { return s.id }

func (s stubIssueConnector) Name() string { return s.name }

func (s stubIssueConnector) PullIssues(context.Context, domain.Config, time.Time) ([]domain.ExternalIssue, error) {
	if s.pullError != nil {
		return nil, s.pullError
	}
	return s.pullIssues, nil
}

func (s stubIssueConnector) ParseWebhook(context.Context, http.Header, []byte) (*domain.WebhookEvent, error) {
	if s.webhookError != nil {
		return nil, s.webhookError
	}
	return s.webhookEvent, nil
}

func (s stubIssueConnector) SyncBack(context.Context, domain.Config, domain.SyncBackUpdate) error {
	return s.syncBackErr
}

func (s stubIssueConnector) HealthCheck(context.Context, domain.Config) error {
	return nil
}
