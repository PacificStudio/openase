package humanauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/google/uuid"
)

func (s *Service) ListUserDirectory(
	ctx context.Context,
	filter domain.UserDirectoryFilter,
) ([]domain.UserDirectoryEntry, error) {
	users, err := s.repo.ListUsers(ctx, filter)
	if err != nil {
		return nil, err
	}

	entries := make([]domain.UserDirectoryEntry, 0, len(users))
	for _, user := range users {
		entry := domain.UserDirectoryEntry{User: user}
		identity, identityErr := s.repo.GetPrimaryIdentity(ctx, user.ID)
		if identityErr == nil {
			entry.PrimaryIdentity = &identity
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *Service) GetUserDirectoryDetail(
	ctx context.Context,
	userID uuid.UUID,
) (domain.UserDirectoryDetail, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return domain.UserDirectoryDetail{}, ErrUserNotFound
		}
		return domain.UserDirectoryDetail{}, err
	}

	identities, err := s.repo.ListUserIdentities(ctx, user.ID)
	if err != nil {
		return domain.UserDirectoryDetail{}, err
	}
	groups, err := s.repo.ListUserGroups(ctx, user.ID)
	if err != nil {
		return domain.UserDirectoryDetail{}, err
	}
	events, err := s.repo.ListAuthAuditEventsByUser(ctx, user.ID, 25)
	if err != nil {
		return domain.UserDirectoryDetail{}, err
	}
	sessions, err := s.repo.ListBrowserSessionsByUser(ctx, user.ID)
	if err != nil {
		return domain.UserDirectoryDetail{}, err
	}
	now := time.Now().UTC()

	detail := domain.UserDirectoryDetail{
		User:               user,
		Identities:         identities,
		Groups:             groups,
		ActiveSessions:     activeSessions(sessions, now),
		RecentAuditEvents:  events,
		ActiveSessionCount: countActiveSessions(sessions, now),
		LatestStatusAudit:  latestUserStatusAudit(events),
	}
	return detail, nil
}

func (s *Service) TransitionUserStatus(
	ctx context.Context,
	input domain.UserStatusTransitionInput,
) (domain.UserStatusTransitionResult, error) {
	user, err := s.repo.GetUser(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return domain.UserStatusTransitionResult{}, ErrUserNotFound
		}
		return domain.UserStatusTransitionResult{}, err
	}

	now := time.Now().UTC()
	result := domain.UserStatusTransitionResult{User: user}
	changed := user.Status != input.TargetStatus

	if changed {
		user, err = s.repo.UpdateUserStatus(ctx, input.UserID, input.TargetStatus)
		if err != nil {
			if errors.Is(err, repo.ErrUserNotFound) {
				return domain.UserStatusTransitionResult{}, ErrUserNotFound
			}
			return domain.UserStatusTransitionResult{}, err
		}
		result.User = user
		result.Changed = true
	}

	if input.TargetStatus == domain.UserStatusDisabled && input.RevokeSessions {
		revokedSessions, revokeErr := s.repo.RevokeBrowserSessionsByUser(ctx, input.UserID, nil, now)
		if revokeErr != nil {
			return domain.UserStatusTransitionResult{}, revokeErr
		}
		result.RevokedSessionCount = len(revokedSessions)
	}

	if result.Changed {
		audit, auditErr := s.recordUserStatusAudit(ctx, result.User, input, now, user.Status, result.RevokedSessionCount)
		if auditErr != nil {
			return domain.UserStatusTransitionResult{}, auditErr
		}
		result.LatestStatusAudit = audit
	}

	return result, nil
}

func (s *Service) recordUserStatusAudit(
	ctx context.Context,
	user domain.User,
	input domain.UserStatusTransitionInput,
	now time.Time,
	previousStatus domain.UserStatus,
	revokedSessionCount int,
) (*domain.UserStatusAudit, error) {
	eventType := domain.AuthAuditUserDisabled
	message := "Disabled a cached user."
	if input.TargetStatus == domain.UserStatusActive {
		eventType = domain.AuthAuditUserEnabled
		message = "Re-enabled a cached user."
	}

	event, err := s.repo.CreateAuthAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		UserID:    &user.ID,
		ActorID:   strings.TrimSpace(input.ActorID),
		EventType: eventType,
		Message:   message,
		Metadata: map[string]any{
			"reason":                input.Reason,
			"source":                string(input.Source),
			"previous_status":       string(previousStatus),
			"current_status":        string(input.TargetStatus),
			"revoked_session_count": revokedSessionCount,
		},
		CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	audit, err := domain.ParseUserStatusAuditEvent(event)
	if err != nil {
		return nil, fmt.Errorf("parse user status audit event: %w", err)
	}
	return audit, nil
}

func latestUserStatusAudit(events []domain.AuthAuditEvent) *domain.UserStatusAudit {
	for _, event := range events {
		audit, err := domain.ParseUserStatusAuditEvent(event)
		if err == nil {
			return audit
		}
	}
	return nil
}

func countActiveSessions(sessions []domain.BrowserSession, now time.Time) int {
	return len(activeSessions(sessions, now))
}

func activeSessions(sessions []domain.BrowserSession, now time.Time) []domain.BrowserSession {
	active := make([]domain.BrowserSession, 0, len(sessions))
	for _, session := range sessions {
		if session.RevokedAt != nil {
			continue
		}
		if browserSessionExpired(now, session) {
			continue
		}
		active = append(active, session)
	}
	return active
}
