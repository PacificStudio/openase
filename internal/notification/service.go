package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entnotificationchannel "github.com/BetterAndBetterII/openase/ent/notificationchannel"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/google/uuid"
)

var (
	// ErrUnavailable reports a missing notification service dependency.
	ErrUnavailable = errors.New("notification service unavailable")
	// ErrOrganizationNotFound reports a missing organization.
	ErrOrganizationNotFound = errors.New("organization not found")
	// ErrProjectNotFound reports a missing project.
	ErrProjectNotFound = errors.New("project not found")
	// ErrChannelNotFound reports a missing notification channel.
	ErrChannelNotFound = errors.New("notification channel not found")
	// ErrDuplicateChannelName reports duplicate channel names within the same organization.
	ErrDuplicateChannelName = errors.New("notification channel name already exists in organization")
	// ErrInvalidChannelConfig reports invalid persisted or patched channel config.
	ErrInvalidChannelConfig = errors.New("notification channel config is invalid")
)

// Service provides notification channel CRUD plus adapter-backed delivery.
type Service struct {
	client   *ent.Client
	logger   *slog.Logger
	registry *AdapterRegistry
}

// NewService constructs a notification service.
func NewService(client *ent.Client, logger *slog.Logger, httpClient *http.Client) *Service {
	resolvedLogger := logger
	if resolvedLogger == nil {
		resolvedLogger = slog.Default()
	}

	return &Service{
		client:   client,
		logger:   resolvedLogger.With("component", "notification-service"),
		registry: NewDefaultAdapterRegistry(httpClient),
	}
}

// List returns all channels configured for the organization.
func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Channel, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureOrganizationExists(ctx, organizationID); err != nil {
		return nil, err
	}

	items, err := s.client.NotificationChannel.Query().
		Where(entnotificationchannel.OrganizationIDEQ(organizationID)).
		Order(ent.Asc(entnotificationchannel.FieldName), ent.Asc(entnotificationchannel.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}

	response := make([]domain.Channel, 0, len(items))
	for _, item := range items {
		response = append(response, mapChannel(item))
	}

	return response, nil
}

// Get returns a single configured channel.
func (s *Service) Get(ctx context.Context, channelID uuid.UUID) (domain.Channel, error) {
	if s.client == nil {
		return domain.Channel{}, ErrUnavailable
	}

	item, err := s.client.NotificationChannel.Get(ctx, channelID)
	if err != nil {
		return domain.Channel{}, mapChannelNotFound(err)
	}

	return mapChannel(item), nil
}

// Create validates and persists a new notification channel.
func (s *Service) Create(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error) {
	if s.client == nil {
		return domain.Channel{}, ErrUnavailable
	}
	if err := s.ensureOrganizationExists(ctx, input.OrganizationID); err != nil {
		return domain.Channel{}, err
	}

	created, err := s.client.NotificationChannel.Create().
		SetOrganizationID(input.OrganizationID).
		SetName(input.Name).
		SetType(input.Type.String()).
		SetConfig(input.Config).
		SetIsEnabled(input.IsEnabled).
		Save(ctx)
	if err != nil {
		return domain.Channel{}, mapPersistenceError("create notification channel", err)
	}

	return mapChannel(created), nil
}

// Update applies a partial update to a notification channel.
func (s *Service) Update(ctx context.Context, input domain.UpdateChannelInput) (domain.Channel, error) {
	if s.client == nil {
		return domain.Channel{}, ErrUnavailable
	}

	current, err := s.client.NotificationChannel.Get(ctx, input.ChannelID)
	if err != nil {
		return domain.Channel{}, mapChannelNotFound(err)
	}

	nextName := current.Name
	if input.Name.Set {
		nextName = input.Name.Value
	}

	nextType, err := domain.ParseChannelType(current.Type)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("load channel type: %w", err)
	}
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

	updated, err := s.client.NotificationChannel.UpdateOneID(input.ChannelID).
		SetName(nextName).
		SetType(nextType.String()).
		SetConfig(normalizedConfig).
		SetIsEnabled(isEnabled).
		Save(ctx)
	if err != nil {
		return domain.Channel{}, mapPersistenceError("update notification channel", err)
	}

	return mapChannel(updated), nil
}

// Delete removes a persisted notification channel.
func (s *Service) Delete(ctx context.Context, channelID uuid.UUID) error {
	if s.client == nil {
		return ErrUnavailable
	}
	if err := s.client.NotificationChannel.DeleteOneID(channelID).Exec(ctx); err != nil {
		return mapChannelNotFound(err)
	}

	return nil
}

// Test sends a synthetic message through the configured channel adapter.
func (s *Service) Test(ctx context.Context, channelID uuid.UUID) error {
	if s.client == nil {
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
	if s.client == nil {
		return ErrUnavailable
	}

	projectItem, err := s.client.Project.Query().
		Where(entproject.ID(projectID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("load project for notifications: %w", err)
	}

	channels, err := s.client.NotificationChannel.Query().
		Where(
			entnotificationchannel.OrganizationIDEQ(projectItem.OrganizationID),
			entnotificationchannel.IsEnabled(true),
		).
		Order(ent.Asc(entnotificationchannel.FieldName), ent.Asc(entnotificationchannel.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list enabled notification channels: %w", err)
	}

	for _, item := range channels {
		channel := mapChannel(item)
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

func (s *Service) sendChannel(ctx context.Context, channel domain.Channel, message domain.Message) error {
	adapter, err := s.registry.Get(channel.Type)
	if err != nil {
		return err
	}

	config, err := domain.ParseChannelConfig(channel.Type, channel.Config)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidChannelConfig, err)
	}

	return adapter.Send(ctx, config, message)
}

func (s *Service) ensureOrganizationExists(ctx context.Context, organizationID uuid.UUID) error {
	exists, err := s.client.Organization.Query().
		Where(entorganization.IDEQ(organizationID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check organization exists: %w", err)
	}
	if !exists {
		return ErrOrganizationNotFound
	}

	return nil
}

func mapChannel(item *ent.NotificationChannel) domain.Channel {
	channelType, err := domain.ParseChannelType(item.Type)
	if err != nil {
		channelType = domain.ChannelType(strings.TrimSpace(item.Type))
	}

	return domain.Channel{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		Name:           item.Name,
		Type:           channelType,
		Config:         item.Config,
		IsEnabled:      item.IsEnabled,
		CreatedAt:      item.CreatedAt,
	}
}

func mapChannelNotFound(err error) error {
	if ent.IsNotFound(err) {
		return ErrChannelNotFound
	}

	return err
}

func mapPersistenceError(action string, err error) error {
	if ent.IsConstraintError(err) && strings.Contains(strings.ToLower(err.Error()), "notificationchannel_organization_id_name") {
		return ErrDuplicateChannelName
	}

	return fmt.Errorf("%s: %w", action, err)
}
