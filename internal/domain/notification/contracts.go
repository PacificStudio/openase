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
	// ErrDuplicateChannelName reports duplicate channel names within the same organization.
	ErrDuplicateChannelName = errors.New("notification channel name already exists in organization")
	// ErrRuleNotFound reports a missing notification rule.
	ErrRuleNotFound = errors.New("notification rule not found")
	// ErrDuplicateRuleName reports duplicate notification rule names within the same project.
	ErrDuplicateRuleName = errors.New("notification rule name already exists in project")
)

// ProjectRef carries the minimal project data needed by the notification service.
type ProjectRef struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
}
