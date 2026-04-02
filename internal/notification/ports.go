package notification

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/google/uuid"
)

// Repository owns notification persistence behind the service boundary.
type Repository interface {
	OrganizationExists(ctx context.Context, organizationID uuid.UUID) (bool, error)
	Project(ctx context.Context, projectID uuid.UUID) (domain.ProjectRef, error)
	Channels(ctx context.Context, organizationID uuid.UUID, enabledOnly bool) ([]domain.Channel, error)
	Channel(ctx context.Context, channelID uuid.UUID) (domain.Channel, error)
	CreateChannel(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error)
	UpdateChannel(ctx context.Context, channel domain.Channel) (domain.Channel, error)
	DeleteChannel(ctx context.Context, channelID uuid.UUID) error
	Rules(ctx context.Context, projectID uuid.UUID) ([]domain.Rule, error)
	Rule(ctx context.Context, ruleID uuid.UUID) (domain.Rule, error)
	CreateRule(ctx context.Context, input domain.CreateRuleInput) (domain.Rule, error)
	UpdateRule(ctx context.Context, rule domain.Rule) (domain.Rule, error)
	DeleteRule(ctx context.Context, ruleID uuid.UUID) error
	MatchingRules(ctx context.Context, projectID uuid.UUID, eventType domain.RuleEventType) ([]domain.Rule, error)
}
