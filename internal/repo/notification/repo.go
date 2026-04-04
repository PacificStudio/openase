package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entnotificationchannel "github.com/BetterAndBetterII/openase/ent/notificationchannel"
	entnotificationrule "github.com/BetterAndBetterII/openase/ent/notificationrule"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/google/uuid"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) OrganizationExists(ctx context.Context, organizationID uuid.UUID) (bool, error) {
	exists, err := r.client.Organization.Query().
		Where(entorganization.IDEQ(organizationID)).
		Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("check organization exists: %w", err)
	}

	return exists, nil
}

func (r *EntRepository) Project(ctx context.Context, projectID uuid.UUID) (domain.ProjectRef, error) {
	projectItem, err := r.client.Project.Query().
		Where(entproject.ID(projectID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ProjectRef{}, domain.ErrProjectNotFound
		}
		return domain.ProjectRef{}, fmt.Errorf("load project for notifications: %w", err)
	}

	return domain.ProjectRef{
		ID:             projectItem.ID,
		OrganizationID: projectItem.OrganizationID,
	}, nil
}

func (r *EntRepository) Channels(ctx context.Context, organizationID uuid.UUID, enabledOnly bool) ([]domain.Channel, error) {
	query := r.client.NotificationChannel.Query().
		Where(entnotificationchannel.OrganizationIDEQ(organizationID))
	if enabledOnly {
		query = query.Where(entnotificationchannel.IsEnabled(true))
	}

	items, err := query.
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

func (r *EntRepository) Channel(ctx context.Context, channelID uuid.UUID) (domain.Channel, error) {
	item, err := r.client.NotificationChannel.Get(ctx, channelID)
	if err != nil {
		return domain.Channel{}, mapChannelNotFound(err)
	}

	return mapChannel(item), nil
}

func (r *EntRepository) CreateChannel(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error) {
	created, err := r.client.NotificationChannel.Create().
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

func (r *EntRepository) UpdateChannel(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	updated, err := r.client.NotificationChannel.UpdateOneID(channel.ID).
		SetName(channel.Name).
		SetType(channel.Type.String()).
		SetConfig(channel.Config).
		SetIsEnabled(channel.IsEnabled).
		Save(ctx)
	if err != nil {
		return domain.Channel{}, mapPersistenceError("update notification channel", err)
	}

	return mapChannel(updated), nil
}

func (r *EntRepository) DeleteChannel(ctx context.Context, channelID uuid.UUID) error {
	rules, err := r.client.NotificationRule.Query().
		Where(entnotificationrule.ChannelIDEQ(channelID)).
		Order(ent.Asc(entnotificationrule.FieldName), ent.Asc(entnotificationrule.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list notification channel usage: %w", err)
	}
	if len(rules) > 0 {
		conflict := &domain.ChannelUsageConflict{
			ChannelID: channelID,
			Rules:     make([]domain.ChannelUsageRuleReference, 0, len(rules)),
		}
		for _, rule := range rules {
			conflict.Rules = append(conflict.Rules, domain.ChannelUsageRuleReference{
				ID:        rule.ID,
				ProjectID: rule.ProjectID,
				Name:      rule.Name,
				EventType: rule.EventType,
				IsEnabled: rule.IsEnabled,
			})
		}
		return conflict
	}

	if err := r.client.NotificationChannel.DeleteOneID(channelID).Exec(ctx); err != nil {
		return mapChannelNotFound(err)
	}

	return nil
}

func (r *EntRepository) Rules(ctx context.Context, projectID uuid.UUID) ([]domain.Rule, error) {
	items, err := r.client.NotificationRule.Query().
		Where(entnotificationrule.ProjectIDEQ(projectID)).
		WithChannel().
		Order(ent.Asc(entnotificationrule.FieldName), ent.Asc(entnotificationrule.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification rules: %w", err)
	}

	response := make([]domain.Rule, 0, len(items))
	for _, item := range items {
		response = append(response, mapRule(item))
	}

	return response, nil
}

func (r *EntRepository) Rule(ctx context.Context, ruleID uuid.UUID) (domain.Rule, error) {
	item, err := r.client.NotificationRule.Query().
		Where(entnotificationrule.IDEQ(ruleID)).
		WithChannel().
		Only(ctx)
	if err != nil {
		return domain.Rule{}, mapRuleNotFound(err)
	}

	return mapRule(item), nil
}

func (r *EntRepository) CreateRule(ctx context.Context, input domain.CreateRuleInput) (domain.Rule, error) {
	created, err := r.client.NotificationRule.Create().
		SetProjectID(input.ProjectID).
		SetChannelID(input.ChannelID).
		SetName(input.Name).
		SetEventType(input.EventType.String()).
		SetFilter(input.Filter).
		SetTemplate(input.Template).
		SetIsEnabled(input.IsEnabled).
		Save(ctx)
	if err != nil {
		return domain.Rule{}, mapPersistenceError("create notification rule", err)
	}

	return r.Rule(ctx, created.ID)
}

func (r *EntRepository) UpdateRule(ctx context.Context, rule domain.Rule) (domain.Rule, error) {
	updated, err := r.client.NotificationRule.UpdateOneID(rule.ID).
		SetName(rule.Name).
		SetEventType(rule.EventType.String()).
		SetFilter(rule.Filter).
		SetChannelID(rule.ChannelID).
		SetTemplate(rule.Template).
		SetIsEnabled(rule.IsEnabled).
		Save(ctx)
	if err != nil {
		return domain.Rule{}, mapPersistenceError("update notification rule", err)
	}

	return r.Rule(ctx, updated.ID)
}

func (r *EntRepository) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	if err := r.client.NotificationRule.DeleteOneID(ruleID).Exec(ctx); err != nil {
		return mapRuleNotFound(err)
	}

	return nil
}

func (r *EntRepository) MatchingRules(ctx context.Context, projectID uuid.UUID, eventType domain.RuleEventType) ([]domain.Rule, error) {
	items, err := r.client.NotificationRule.Query().
		Where(
			entnotificationrule.ProjectIDEQ(projectID),
			entnotificationrule.EventTypeEQ(eventType.String()),
			entnotificationrule.IsEnabled(true),
		).
		WithChannel().
		Order(ent.Asc(entnotificationrule.FieldName), ent.Asc(entnotificationrule.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list matching notification rules: %w", err)
	}

	rules := make([]domain.Rule, 0, len(items))
	for _, item := range items {
		rule := mapRule(item)
		if !rule.Channel.IsEnabled {
			continue
		}
		rules = append(rules, rule)
	}

	return rules, nil
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

func mapRule(item *ent.NotificationRule) domain.Rule {
	eventType, err := domain.ParseRuleEventType(item.EventType)
	if err != nil {
		eventType = domain.RuleEventType(strings.TrimSpace(item.EventType))
	}

	rule := domain.Rule{
		ID:        item.ID,
		ProjectID: item.ProjectID,
		ChannelID: item.ChannelID,
		Name:      item.Name,
		EventType: eventType,
		Filter:    item.Filter,
		Template:  item.Template,
		IsEnabled: item.IsEnabled,
		CreatedAt: item.CreatedAt,
	}
	if item.Edges.Channel != nil {
		rule.Channel = mapChannel(item.Edges.Channel)
	}

	return rule
}

func mapChannelNotFound(err error) error {
	if ent.IsNotFound(err) {
		return domain.ErrChannelNotFound
	}

	return err
}

func mapRuleNotFound(err error) error {
	if ent.IsNotFound(err) {
		return domain.ErrRuleNotFound
	}

	return err
}

func mapPersistenceError(action string, err error) error {
	if ent.IsConstraintError(err) && strings.Contains(strings.ToLower(err.Error()), "notificationchannel_organization_id_name") {
		return domain.ErrDuplicateChannelName
	}
	if ent.IsConstraintError(err) && strings.Contains(strings.ToLower(err.Error()), "notificationrule_project_id_name") {
		return domain.ErrDuplicateRuleName
	}

	return fmt.Errorf("%s: %w", action, err)
}
