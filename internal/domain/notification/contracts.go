package notification

import (
	"errors"

	"github.com/google/uuid"
)

var (
	// ErrOrganizationNotFound reports a missing organization.
	ErrOrganizationNotFound = errors.New("organization not found")
	// ErrProjectNotFound reports a missing project.
	ErrProjectNotFound = errors.New("project not found")
	// ErrChannelNotFound reports a missing notification channel.
	ErrChannelNotFound = errors.New("notification channel not found")
	// ErrChannelInUse reports that a notification channel is still referenced by rules.
	ErrChannelInUse = errors.New("notification channel is still referenced by notification rules")
	// ErrDuplicateChannelName reports duplicate channel names within the same organization.
	ErrDuplicateChannelName = errors.New("notification channel name already exists in organization")
	// ErrRuleNotFound reports a missing notification rule.
	ErrRuleNotFound = errors.New("notification rule not found")
	// ErrDuplicateRuleName reports duplicate notification rule names within the same project.
	ErrDuplicateRuleName = errors.New("notification rule name already exists in project")
)

type ChannelUsageRuleReference struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	EventType string    `json:"event_type"`
	IsEnabled bool      `json:"is_enabled"`
}

type ChannelUsageConflict struct {
	ChannelID uuid.UUID                   `json:"channel_id"`
	Rules     []ChannelUsageRuleReference `json:"rules"`
}

func (e *ChannelUsageConflict) Error() string {
	if e == nil {
		return ""
	}
	return ErrChannelInUse.Error()
}

func (e *ChannelUsageConflict) Unwrap() error {
	if e == nil {
		return nil
	}
	return ErrChannelInUse
}

// ProjectRef carries the minimal project data needed by the notification service.
type ProjectRef struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
}
