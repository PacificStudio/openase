package issueconnector

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	repo "github.com/BetterAndBetterII/openase/internal/repo/issueconnector"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
)

var (
	ErrUnavailable            = errors.New("issue connector service unavailable")
	ErrProjectNotFound        = errors.New("project not found")
	ErrConnectorNotFound      = errors.New("issue connector not found")
	ErrConnectorConflict      = errors.New("issue connector already exists")
	ErrConnectorTypeNotFound  = errors.New("issue connector type is not registered")
	ErrConnectorRuntimeAbsent = errors.New("issue connector runtime is unavailable")
)

type Optional[T any] struct {
	Set   bool
	Value T
}

type UpdateInput struct {
	ID            uuid.UUID
	Name          Optional[string]
	Status        Optional[domain.Status]
	BaseURL       Optional[string]
	AuthToken     Optional[string]
	ProjectRef    Optional[string]
	PollInterval  Optional[time.Duration]
	SyncDirection Optional[domain.SyncDirection]
	Filters       Optional[domain.Filters]
	StatusMapping Optional[map[string]string]
	WebhookSecret Optional[string]
	AutoWorkflow  Optional[string]
}

type TestResult struct {
	Healthy   bool      `json:"healthy"`
	CheckedAt time.Time `json:"checked_at"`
	Message   string    `json:"message"`
}

type SyncReport struct {
	ConnectorsScanned int `json:"connectors_scanned"`
	ConnectorsSynced  int `json:"connectors_synced"`
	ConnectorsFailed  int `json:"connectors_failed"`
	IssuesSynced      int `json:"issues_synced"`
}

type SyncRunner interface {
	SyncConnector(ctx context.Context, connectorID uuid.UUID) (SyncReport, error)
}

type RuntimeRegistry interface {
	Get(connectorType domain.Type) (domain.Connector, error)
}

type SyncResult struct {
	Connector domain.IssueConnector `json:"connector"`
	Report    SyncReport            `json:"report"`
}

type StatsResult struct {
	ConnectorID uuid.UUID        `json:"connector_id"`
	Status      domain.Status    `json:"status"`
	LastSyncAt  *time.Time       `json:"last_sync_at,omitempty"`
	LastError   string           `json:"last_error"`
	Stats       domain.SyncStats `json:"stats"`
}

type Service struct {
	repo     repo.Repository
	registry RuntimeRegistry
	syncer   SyncRunner
	resolver githubauthservice.TokenResolver
	logger   *slog.Logger
	now      func() time.Time
}

func New(
	repository repo.Repository,
	registry RuntimeRegistry,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repo:     repository,
		registry: registry,
		logger:   logger.With("component", "issue-connector-service"),
		now:      time.Now,
	}
}

func (s *Service) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	s.resolver = resolver
}

func (s *Service) ConfigureSyncRunner(syncer SyncRunner) {
	s.syncer = syncer
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]domain.IssueConnector, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}

	items, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, s.mapRepositoryError(err)
	}

	return items, nil
}

func (s *Service) Create(ctx context.Context, input domain.CreateIssueConnector) (domain.IssueConnector, error) {
	if s.repo == nil {
		return domain.IssueConnector{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return domain.IssueConnector{}, err
	}
	if err := s.ensureConnectorType(input.Type); err != nil {
		return domain.IssueConnector{}, err
	}

	item, err := s.repo.Create(ctx, input)
	if err != nil {
		return domain.IssueConnector{}, s.mapRepositoryError(err)
	}

	return item, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (domain.IssueConnector, error) {
	if s.repo == nil {
		return domain.IssueConnector{}, ErrUnavailable
	}

	current, err := s.repo.Get(ctx, input.ID)
	if err != nil {
		return domain.IssueConnector{}, s.mapRepositoryError(err)
	}
	merged := connectorInputFromModel(current)

	if input.Name.Set {
		merged.Name = input.Name.Value
	}
	if input.Status.Set {
		merged.Status = string(input.Status.Value)
	}
	if input.BaseURL.Set {
		merged.Config.BaseURL = input.BaseURL.Value
	}
	if input.AuthToken.Set {
		merged.Config.AuthToken = input.AuthToken.Value
	}
	if input.ProjectRef.Set {
		merged.Config.ProjectRef = input.ProjectRef.Value
	}
	if input.PollInterval.Set {
		merged.Config.PollInterval = input.PollInterval.Value.String()
	}
	if input.SyncDirection.Set {
		merged.Config.SyncDirection = string(input.SyncDirection.Value)
	}
	if input.Filters.Set {
		merged.Config.Filters = filtersInputFromDomain(input.Filters.Value)
	}
	if input.StatusMapping.Set {
		merged.Config.StatusMapping = cloneStringMap(input.StatusMapping.Value)
	}
	if input.WebhookSecret.Set {
		merged.Config.WebhookSecret = input.WebhookSecret.Value
	}
	if input.AutoWorkflow.Set {
		merged.Config.AutoWorkflow = input.AutoWorkflow.Value
	}

	parsed, err := domain.ParseUpdateIssueConnector(current.ID, current.ProjectID, merged)
	if err != nil {
		return domain.IssueConnector{}, err
	}
	if err := s.ensureConnectorType(parsed.Type); err != nil {
		return domain.IssueConnector{}, err
	}

	updated, err := s.repo.Update(ctx, parsed)
	if err != nil {
		return domain.IssueConnector{}, s.mapRepositoryError(err)
	}
	updated.LastSyncAt = cloneTime(current.LastSyncAt)
	updated.LastError = current.LastError
	updated.Stats = current.Stats
	if err := s.repo.Save(ctx, updated); err != nil {
		return domain.IssueConnector{}, s.mapRepositoryError(err)
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, connectorID uuid.UUID) error {
	if s.repo == nil {
		return ErrUnavailable
	}
	if _, err := s.repo.Get(ctx, connectorID); err != nil {
		return s.mapRepositoryError(err)
	}
	if err := s.repo.Delete(ctx, connectorID); err != nil {
		return s.mapRepositoryError(err)
	}

	return nil
}

func (s *Service) Test(ctx context.Context, connectorID uuid.UUID) (TestResult, error) {
	if s.repo == nil || s.registry == nil {
		return TestResult{}, ErrConnectorRuntimeAbsent
	}

	connector, err := s.repo.Get(ctx, connectorID)
	if err != nil {
		return TestResult{}, s.mapRepositoryError(err)
	}

	impl, err := s.registry.Get(connector.Type)
	if err != nil {
		return TestResult{}, ErrConnectorTypeNotFound
	}
	cfg, err := s.resolveConnectorConfig(ctx, connector)
	if err != nil {
		return TestResult{}, err
	}
	if err := impl.HealthCheck(ctx, cfg); err != nil {
		return TestResult{
			Healthy:   false,
			CheckedAt: s.now().UTC(),
			Message:   err.Error(),
		}, nil
	}

	return TestResult{
		Healthy:   true,
		CheckedAt: s.now().UTC(),
		Message:   "connection healthy",
	}, nil
}

func (s *Service) Sync(ctx context.Context, connectorID uuid.UUID) (SyncResult, error) {
	if s.syncer == nil || s.repo == nil {
		return SyncResult{}, ErrConnectorRuntimeAbsent
	}

	report, err := s.syncer.SyncConnector(ctx, connectorID)
	if err != nil {
		return SyncResult{}, err
	}
	connector, getErr := s.repo.Get(ctx, connectorID)
	if getErr != nil {
		return SyncResult{}, s.mapRepositoryError(getErr)
	}

	return SyncResult{
		Connector: connector,
		Report:    report,
	}, nil
}

func (s *Service) Stats(ctx context.Context, connectorID uuid.UUID) (StatsResult, error) {
	if s.repo == nil {
		return StatsResult{}, ErrUnavailable
	}

	connector, err := s.repo.Get(ctx, connectorID)
	if err != nil {
		return StatsResult{}, s.mapRepositoryError(err)
	}

	return StatsResult{
		ConnectorID: connector.ID,
		Status:      connector.Status,
		LastSyncAt:  cloneTime(connector.LastSyncAt),
		LastError:   connector.LastError,
		Stats:       connector.Stats,
	}, nil
}

func (s *Service) ensureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := s.repo.ProjectExists(ctx, projectID)
	if err != nil {
		return s.mapRepositoryError(err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func (s *Service) ensureConnectorType(connectorType domain.Type) error {
	if s.registry == nil {
		return ErrConnectorRuntimeAbsent
	}
	if _, err := s.registry.Get(connectorType); err != nil {
		return ErrConnectorTypeNotFound
	}

	return nil
}

func (s *Service) resolveConnectorConfig(ctx context.Context, connector domain.IssueConnector) (domain.Config, error) {
	cfg := connector.Config
	if connector.Type != domain.TypeGitHub || strings.TrimSpace(cfg.AuthToken) != "" || s.resolver == nil {
		return cfg, nil
	}

	resolved, err := s.resolver.ResolveProjectCredential(ctx, connector.ProjectID)
	if err != nil {
		return domain.Config{}, err
	}
	cfg.AuthToken = strings.TrimSpace(resolved.Token)
	return cfg, nil
}

func (s *Service) mapRepositoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrProjectNotFound), errors.Is(err, ErrConnectorNotFound), errors.Is(err, ErrConnectorConflict):
		return err
	case errors.Is(err, repo.ErrConnectorNotFound):
		return ErrConnectorNotFound
	case errors.Is(err, repo.ErrConnectorConflict):
		return ErrConnectorConflict
	default:
		return err
	}
}

func connectorInputFromModel(connector domain.IssueConnector) domain.Input {
	return domain.Input{
		Type:   string(connector.Type),
		Name:   connector.Name,
		Status: string(connector.Status),
		Config: domain.ConfigInput{
			Type:          string(connector.Config.Type),
			BaseURL:       connector.Config.BaseURL,
			AuthToken:     connector.Config.AuthToken,
			ProjectRef:    connector.Config.ProjectRef,
			PollInterval:  connector.Config.PollInterval.String(),
			SyncDirection: string(connector.Config.SyncDirection),
			Filters:       filtersInputFromDomain(connector.Config.Filters),
			StatusMapping: cloneStringMap(connector.Config.StatusMapping),
			WebhookSecret: connector.Config.WebhookSecret,
			AutoWorkflow:  connector.Config.AutoWorkflow,
		},
	}
}

func filtersInputFromDomain(filters domain.Filters) domain.FiltersInput {
	return domain.FiltersInput{
		Labels:        append([]string(nil), filters.Labels...),
		ExcludeLabels: append([]string(nil), filters.ExcludeLabels...),
		States:        append([]string(nil), filters.States...),
		Authors:       append([]string(nil), filters.Authors...),
	}
}

func cloneStringMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}

	return cloned
}

func cloneTime(raw *time.Time) *time.Time {
	if raw == nil {
		return nil
	}
	cloned := raw.UTC()
	return &cloned
}
