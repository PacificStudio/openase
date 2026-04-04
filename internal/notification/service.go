package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/google/uuid"
)

var (
	// ErrUnavailable reports a missing notification service dependency.
	ErrUnavailable = errors.New("notification service unavailable")
	// ErrOrganizationNotFound reports a missing organization.
	ErrOrganizationNotFound = domain.ErrOrganizationNotFound
	// ErrProjectNotFound reports a missing project.
	ErrProjectNotFound = domain.ErrProjectNotFound
	// ErrChannelNotFound reports a missing notification channel.
	ErrChannelNotFound = domain.ErrChannelNotFound
	// ErrChannelInUse reports a notification channel still referenced by rules.
	ErrChannelInUse = domain.ErrChannelInUse
	// ErrDuplicateChannelName reports duplicate channel names within the same organization.
	ErrDuplicateChannelName = domain.ErrDuplicateChannelName
	// ErrRuleNotFound reports a missing notification rule.
	ErrRuleNotFound = domain.ErrRuleNotFound
	// ErrDuplicateRuleName reports duplicate notification rule names within the same project.
	ErrDuplicateRuleName = domain.ErrDuplicateRuleName
	// ErrChannelProjectMismatch reports that the selected channel belongs to a different organization.
	ErrChannelProjectMismatch = errors.New("notification channel does not belong to the rule project organization")
	// ErrInvalidChannelConfig reports invalid persisted or patched channel config.
	ErrInvalidChannelConfig = errors.New("notification channel config is invalid")
)

// Service provides notification channel CRUD plus adapter-backed delivery.
type Service struct {
	repo     Repository
	logger   *slog.Logger
	registry *AdapterRegistry
}

// NewService constructs a notification service.
func NewService(repo Repository, logger *slog.Logger, httpClient *http.Client) *Service {
	resolvedLogger := logger
	if resolvedLogger == nil {
		resolvedLogger = slog.Default()
	}

	return &Service{
		repo:     repo,
		logger:   resolvedLogger.With("component", "notification-service"),
		registry: NewDefaultAdapterRegistry(httpClient),
	}
}

// List returns all channels configured for the organization.
func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Channel, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureOrganizationExists(ctx, organizationID); err != nil {
		return nil, err
	}

	return s.repo.Channels(ctx, organizationID, false)
}

// Get returns a single configured channel.
func (s *Service) Get(ctx context.Context, channelID uuid.UUID) (domain.Channel, error) {
	if s.repo == nil {
		return domain.Channel{}, ErrUnavailable
	}

	return s.repo.Channel(ctx, channelID)
}

// ListRules returns all configured notification rules for the project.
func (s *Service) ListRules(ctx context.Context, projectID uuid.UUID) ([]domain.Rule, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}
	if _, err := s.getProject(ctx, projectID); err != nil {
		return nil, err
	}

	return s.repo.Rules(ctx, projectID)
}

// Create validates and persists a new notification channel.
func (s *Service) Create(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error) {
	if s.repo == nil {
		return domain.Channel{}, ErrUnavailable
	}
	if err := s.ensureOrganizationExists(ctx, input.OrganizationID); err != nil {
		return domain.Channel{}, err
	}

	return s.repo.CreateChannel(ctx, input)
}

// CreateRule validates and persists a new notification rule.
func (s *Service) CreateRule(ctx context.Context, input domain.CreateRuleInput) (domain.Rule, error) {
	if s.repo == nil {
		return domain.Rule{}, ErrUnavailable
	}

	projectItem, err := s.getProject(ctx, input.ProjectID)
	if err != nil {
		return domain.Rule{}, err
	}
	if err := s.ensureChannelMatchesProject(ctx, projectItem.OrganizationID, input.ChannelID); err != nil {
		return domain.Rule{}, err
	}

	return s.repo.CreateRule(ctx, input)
}

// Update applies a partial update to a notification channel.
func (s *Service) Update(ctx context.Context, input domain.UpdateChannelInput) (domain.Channel, error) {
	if s.repo == nil {
		return domain.Channel{}, ErrUnavailable
	}

	current, err := s.repo.Channel(ctx, input.ChannelID)
	if err != nil {
		return domain.Channel{}, err
	}

	nextName := current.Name
	if input.Name.Set {
		nextName = input.Name.Value
	}

	nextType := current.Type
	if input.Type.Set {
		nextType = input.Type.Value
	}

	nextConfig := current.Config
	if input.Config.Set {
		nextConfig = input.Config.Value
	}
	normalizedConfig, err := domain.NormalizeChannelConfig(nextType, nextConfig)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("%w: %v", ErrInvalidChannelConfig, err)
	}

	isEnabled := current.IsEnabled
	if input.IsEnabled.Set {
		isEnabled = input.IsEnabled.Value
	}

	return s.repo.UpdateChannel(ctx, domain.Channel{
		ID:             current.ID,
		OrganizationID: current.OrganizationID,
		Name:           nextName,
		Type:           nextType,
		Config:         normalizedConfig,
		IsEnabled:      isEnabled,
		CreatedAt:      current.CreatedAt,
	})
}

// UpdateRule applies a partial update to a notification rule.
func (s *Service) UpdateRule(ctx context.Context, input domain.UpdateRuleInput) (domain.Rule, error) {
	if s.repo == nil {
		return domain.Rule{}, ErrUnavailable
	}

	current, err := s.repo.Rule(ctx, input.RuleID)
	if err != nil {
		return domain.Rule{}, err
	}

	projectItem, err := s.getProject(ctx, current.ProjectID)
	if err != nil {
		return domain.Rule{}, err
	}

	nextName := current.Name
	if input.Name.Set {
		nextName = input.Name.Value
	}

	nextEventType := current.EventType
	if input.EventType.Set {
		nextEventType = input.EventType.Value
	}

	nextFilter := current.Filter
	if input.Filter.Set {
		nextFilter = input.Filter.Value
	}

	nextChannelID := current.ChannelID
	if input.ChannelID.Set {
		nextChannelID = input.ChannelID.Value
	}
	if err := s.ensureChannelMatchesProject(ctx, projectItem.OrganizationID, nextChannelID); err != nil {
		return domain.Rule{}, err
	}

	nextTemplate := current.Template
	if input.Template.Set {
		nextTemplate = input.Template.Value
	}

	isEnabled := current.IsEnabled
	if input.IsEnabled.Set {
		isEnabled = input.IsEnabled.Value
	}

	return s.repo.UpdateRule(ctx, domain.Rule{
		ID:        current.ID,
		ProjectID: current.ProjectID,
		ChannelID: nextChannelID,
		Name:      nextName,
		EventType: nextEventType,
		Filter:    nextFilter,
		Template:  nextTemplate,
		IsEnabled: isEnabled,
		CreatedAt: current.CreatedAt,
	})
}

// Delete removes a persisted notification channel.
func (s *Service) Delete(ctx context.Context, channelID uuid.UUID) error {
	if s.repo == nil {
		return ErrUnavailable
	}

	return s.repo.DeleteChannel(ctx, channelID)
}

// DeleteRule removes a persisted notification rule.
func (s *Service) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	if s.repo == nil {
		return ErrUnavailable
	}

	return s.repo.DeleteRule(ctx, ruleID)
}

// Test sends a synthetic message through the configured channel adapter.
func (s *Service) Test(ctx context.Context, channelID uuid.UUID) error {
	if s.repo == nil {
		return ErrUnavailable
	}

	channel, err := s.Get(ctx, channelID)
	if err != nil {
		return err
	}

	return s.sendChannel(ctx, channel, domain.Message{
		Title: "OpenASE test notification",
		Body:  "Notification channel connectivity test from OpenASE.",
		Level: "info",
		Metadata: map[string]string{
			"kind": "test",
		},
	})
}

// SendToProjectChannels fans a message out to all enabled channels in the project's organization.
func (s *Service) SendToProjectChannels(ctx context.Context, projectID uuid.UUID, message domain.Message) error {
	if s.repo == nil {
		return ErrUnavailable
	}

	projectItem, err := s.getProject(ctx, projectID)
	if err != nil {
		return err
	}

	channels, err := s.repo.Channels(ctx, projectItem.OrganizationID, true)
	if err != nil {
		return err
	}

	for _, channel := range channels {
		if err := s.sendChannel(ctx, channel, message); err != nil {
			s.logger.Warn(
				"notification send failed",
				"channel_id", channel.ID.String(),
				"channel_name", channel.Name,
				"channel_type", channel.Type.String(),
				"error", err,
			)
		}
	}

	return nil
}

// SendRule delivers a message through the rule's configured channel.
func (s *Service) SendRule(ctx context.Context, rule domain.Rule, message domain.Message) error {
	if s.repo == nil {
		return ErrUnavailable
	}
	if !rule.IsEnabled || !rule.Channel.IsEnabled {
		return nil
	}

	return s.sendChannel(ctx, rule.Channel, message)
}

// MatchingRules resolves enabled notification rules for a project and event type.
func (s *Service) MatchingRules(ctx context.Context, projectID uuid.UUID, eventType domain.RuleEventType) ([]domain.Rule, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}

	return s.repo.MatchingRules(ctx, projectID, eventType)
}

func (s *Service) sendChannel(ctx context.Context, channel domain.Channel, message domain.Message) error {
	adapter, err := s.registry.Get(channel.Type)
	if err != nil {
		s.logger.Error(
			"notification adapter unavailable",
			"operation", "resolve_notification_adapter",
			"channel_id", channel.ID.String(),
			"channel_name", channel.Name,
			"channel_type", channel.Type.String(),
			"error", err,
		)
		return err
	}

	config, err := domain.ParseChannelConfig(channel.Type, channel.Config)
	if err != nil {
		s.logger.Warn(
			"notification channel config invalid",
			"operation", "parse_notification_channel_config",
			"channel_id", channel.ID.String(),
			"channel_name", channel.Name,
			"channel_type", channel.Type.String(),
			"config_keys", notificationConfigKeys(channel.Config),
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrInvalidChannelConfig, err)
	}

	return adapter.Send(ctx, config, message)
}

func notificationConfigKeys(config map[string]any) []string {
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (s *Service) ensureOrganizationExists(ctx context.Context, organizationID uuid.UUID) error {
	exists, err := s.repo.OrganizationExists(ctx, organizationID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrOrganizationNotFound
	}

	return nil
}

func (s *Service) getProject(ctx context.Context, projectID uuid.UUID) (domain.ProjectRef, error) {
	return s.repo.Project(ctx, projectID)
}

func (s *Service) ensureChannelMatchesProject(ctx context.Context, organizationID uuid.UUID, channelID uuid.UUID) error {
	channel, err := s.Get(ctx, channelID)
	if err != nil {
		return err
	}
	if channel.OrganizationID != organizationID {
		return ErrChannelProjectMismatch
	}

	return nil
}
