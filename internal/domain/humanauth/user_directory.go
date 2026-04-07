package humanauth

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserDirectoryStatusFilter string

const (
	UserDirectoryStatusAll      UserDirectoryStatusFilter = "all"
	UserDirectoryStatusActive   UserDirectoryStatusFilter = "active"
	UserDirectoryStatusDisabled UserDirectoryStatusFilter = "disabled"
)

type UserStatusTransitionSource string

const (
	UserStatusTransitionSourceAdminManual  UserStatusTransitionSource = "admin_manual"
	UserStatusTransitionSourceOIDCUpstream UserStatusTransitionSource = "oidc_upstream_sync"
	UserStatusTransitionSourceWebhook      UserStatusTransitionSource = "webhook"
	UserStatusTransitionSourceSCIM         UserStatusTransitionSource = "scim"
	UserDirectoryDefaultPageLimit                                     = 50
	UserDirectoryMaximumPageLimit                                     = 200
)

type UserDirectoryFilter struct {
	Query  string
	Status UserDirectoryStatusFilter
	Limit  int
}

type UserDirectoryEntry struct {
	User            User
	PrimaryIdentity *UserIdentity
}

type UserStatusAudit struct {
	Status              UserStatus
	Reason              string
	Source              UserStatusTransitionSource
	ActorID             string
	ChangedAt           time.Time
	RevokedSessionCount int
}

type UserDirectoryDetail struct {
	User               User
	Identities         []UserIdentity
	Groups             []UserGroupMembership
	ActiveSessions     []BrowserSession
	RecentAuditEvents  []AuthAuditEvent
	ActiveSessionCount int
	LatestStatusAudit  *UserStatusAudit
}

type UserStatusTransitionInput struct {
	UserID         uuid.UUID
	TargetStatus   UserStatus
	Reason         string
	ActorID        string
	Source         UserStatusTransitionSource
	RevokeSessions bool
}

type UserStatusTransitionResult struct {
	User                User
	Changed             bool
	RevokedSessionCount int
	LatestStatusAudit   *UserStatusAudit
}

func ParseUserStatus(raw string) (UserStatus, error) {
	switch UserStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case UserStatusActive:
		return UserStatusActive, nil
	case UserStatusDisabled:
		return UserStatusDisabled, nil
	default:
		return "", fmt.Errorf("unsupported user status %q", raw)
	}
}

func ParseUserDirectoryStatusFilter(raw string) (UserDirectoryStatusFilter, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" || normalized == string(UserDirectoryStatusAll) {
		return UserDirectoryStatusAll, nil
	}
	switch UserDirectoryStatusFilter(normalized) {
	case UserDirectoryStatusActive:
		return UserDirectoryStatusActive, nil
	case UserDirectoryStatusDisabled:
		return UserDirectoryStatusDisabled, nil
	default:
		return "", fmt.Errorf("unsupported user status filter %q", raw)
	}
}

func ParseUserStatusTransitionSource(raw string) (UserStatusTransitionSource, error) {
	switch UserStatusTransitionSource(strings.ToLower(strings.TrimSpace(raw))) {
	case UserStatusTransitionSourceAdminManual:
		return UserStatusTransitionSourceAdminManual, nil
	case UserStatusTransitionSourceOIDCUpstream:
		return UserStatusTransitionSourceOIDCUpstream, nil
	case UserStatusTransitionSourceWebhook:
		return UserStatusTransitionSourceWebhook, nil
	case UserStatusTransitionSourceSCIM:
		return UserStatusTransitionSourceSCIM, nil
	default:
		return "", fmt.Errorf("unsupported user status transition source %q", raw)
	}
}

func NewUserDirectoryFilter(rawQuery string, rawStatus string, rawLimit int) (UserDirectoryFilter, error) {
	status, err := ParseUserDirectoryStatusFilter(rawStatus)
	if err != nil {
		return UserDirectoryFilter{}, err
	}
	limit := rawLimit
	switch {
	case limit <= 0:
		limit = UserDirectoryDefaultPageLimit
	case limit > UserDirectoryMaximumPageLimit:
		limit = UserDirectoryMaximumPageLimit
	}
	return UserDirectoryFilter{
		Query:  strings.TrimSpace(rawQuery),
		Status: status,
		Limit:  limit,
	}, nil
}

func NewUserStatusTransitionInput(
	userID uuid.UUID,
	targetStatus UserStatus,
	reason string,
	actorID string,
	source UserStatusTransitionSource,
	revokeSessions bool,
) (UserStatusTransitionInput, error) {
	if userID == uuid.Nil {
		return UserStatusTransitionInput{}, fmt.Errorf("user id must not be empty")
	}
	if _, err := ParseUserStatus(string(targetStatus)); err != nil {
		return UserStatusTransitionInput{}, err
	}
	if strings.TrimSpace(reason) == "" {
		return UserStatusTransitionInput{}, fmt.Errorf("status transition reason must not be empty")
	}
	if strings.TrimSpace(actorID) == "" {
		return UserStatusTransitionInput{}, fmt.Errorf("status transition actor must not be empty")
	}
	if _, err := ParseUserStatusTransitionSource(string(source)); err != nil {
		return UserStatusTransitionInput{}, err
	}
	return UserStatusTransitionInput{
		UserID:         userID,
		TargetStatus:   targetStatus,
		Reason:         strings.TrimSpace(reason),
		ActorID:        strings.TrimSpace(actorID),
		Source:         source,
		RevokeSessions: revokeSessions,
	}, nil
}

func ParseUserStatusAuditEvent(event AuthAuditEvent) (*UserStatusAudit, error) {
	var status UserStatus
	switch event.EventType {
	case AuthAuditUserEnabled:
		status = UserStatusActive
	case AuthAuditUserDisabled:
		status = UserStatusDisabled
	default:
		return nil, fmt.Errorf("auth audit event %q is not a user status transition", event.EventType)
	}

	audit := &UserStatusAudit{
		Status:    status,
		ActorID:   strings.TrimSpace(event.ActorID),
		ChangedAt: event.CreatedAt.UTC(),
		Source:    UserStatusTransitionSourceAdminManual,
	}
	if reason, ok := event.Metadata["reason"]; ok {
		audit.Reason = strings.TrimSpace(fmt.Sprint(reason))
	}
	if source, ok := event.Metadata["source"]; ok {
		if parsed, err := ParseUserStatusTransitionSource(fmt.Sprint(source)); err == nil {
			audit.Source = parsed
		}
	}
	if rawCount, ok := event.Metadata["revoked_session_count"]; ok {
		if count, err := parseAuditInteger(rawCount); err == nil && count >= 0 {
			audit.RevokedSessionCount = count
		}
	}
	return audit, nil
}

func parseAuditInteger(raw any) (int, error) {
	switch value := raw.(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		return int(value), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(value))
	default:
		return 0, fmt.Errorf("unsupported integer value %T", raw)
	}
}
