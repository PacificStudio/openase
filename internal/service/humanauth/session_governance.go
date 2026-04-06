package humanauth

import (
	"context"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/google/uuid"
)

type StepUpCapability struct {
	Status           string
	Summary          string
	SupportedMethods []string
}

type SessionGovernanceSnapshot struct {
	Sessions    []domain.BrowserSession
	AuditEvents []domain.AuthAuditEvent
	StepUp      StepUpCapability
}

type parsedSessionDevice struct {
	Kind    domain.SessionDeviceKind
	OS      string
	Browser string
	Label   string
}

func ReservedStepUpCapability() StepUpCapability {
	return StepUpCapability{
		Status:           "reserved",
		Summary:          "Re-auth and step-up hooks are reserved for future high-risk actions. Current session governance uses the same browser session and CSRF boundary.",
		SupportedMethods: []string{},
	}
}

func (s *Service) ListSessionGovernance(
	ctx context.Context,
	principal domain.AuthenticatedPrincipal,
) (SessionGovernanceSnapshot, error) {
	snapshot := SessionGovernanceSnapshot{
		Sessions:    []domain.BrowserSession{},
		AuditEvents: []domain.AuthAuditEvent{},
		StepUp:      ReservedStepUpCapability(),
	}
	if s.cfg.Mode != config.AuthModeOIDC {
		return snapshot, nil
	}

	sessions, err := s.repo.ListBrowserSessionsByUser(ctx, principal.User.ID)
	if err != nil {
		return SessionGovernanceSnapshot{}, err
	}
	now := time.Now().UTC()
	active := make([]domain.BrowserSession, 0, len(sessions))
	for _, session := range sessions {
		if session.RevokedAt != nil {
			continue
		}
		if now.After(session.ExpiresAt) || now.After(session.IdleExpiresAt) {
			_ = s.expireSession(ctx, session, now)
			continue
		}
		active = append(active, session)
	}
	snapshot.Sessions = active

	auditEvents, err := s.repo.ListAuthAuditEventsByUser(ctx, principal.User.ID, 20)
	if err != nil {
		return SessionGovernanceSnapshot{}, err
	}
	snapshot.AuditEvents = auditEvents
	return snapshot, nil
}

func (s *Service) RevokeSession(
	ctx context.Context,
	principal domain.AuthenticatedPrincipal,
	sessionID uuid.UUID,
) (domain.BrowserSession, bool, error) {
	session, err := s.repo.GetBrowserSession(ctx, sessionID)
	if err != nil {
		return domain.BrowserSession{}, false, ErrSessionNotFound
	}
	if session.UserID != principal.User.ID {
		return domain.BrowserSession{}, false, ErrPermissionDenied
	}
	isCurrent := session.ID == principal.Session.ID
	if session.RevokedAt != nil {
		return session, isCurrent, nil
	}

	now := time.Now().UTC()
	if now.After(session.ExpiresAt) || now.After(session.IdleExpiresAt) {
		_ = s.expireSession(ctx, session, now)
		session.RevokedAt = &now
		return session, isCurrent, nil
	}

	if err := s.repo.RevokeBrowserSession(ctx, session.ID, now); err != nil {
		return domain.BrowserSession{}, false, err
	}
	session.RevokedAt = &now
	if err := s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		UserID:    &session.UserID,
		SessionID: &session.ID,
		ActorID:   principal.ActorID(),
		EventType: domain.AuthAuditSessionRevoked,
		Message:   "Revoked a browser session.",
		Metadata: map[string]any{
			"reason":  "user_revoke_session",
			"current": isCurrent,
		},
		CreatedAt: now,
	}); err != nil {
		return domain.BrowserSession{}, false, err
	}
	return session, isCurrent, nil
}

func (s *Service) RevokeOtherSessions(
	ctx context.Context,
	principal domain.AuthenticatedPrincipal,
) ([]domain.BrowserSession, error) {
	now := time.Now().UTC()
	sessions, err := s.repo.RevokeBrowserSessionsByUser(ctx, principal.User.ID, &principal.Session.ID, now)
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		if err := s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
			UserID:    &session.UserID,
			SessionID: &session.ID,
			ActorID:   principal.ActorID(),
			EventType: domain.AuthAuditSessionRevoked,
			Message:   "Revoked another browser session.",
			Metadata:  map[string]any{"reason": "user_revoke_other_sessions"},
			CreatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	return sessions, nil
}

func (s *Service) ForceRevokeUserSessions(
	ctx context.Context,
	actor domain.AuthenticatedPrincipal,
	userID uuid.UUID,
) ([]domain.BrowserSession, error) {
	now := time.Now().UTC()
	sessions, err := s.repo.RevokeBrowserSessionsByUser(ctx, userID, nil, now)
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		if err := s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
			UserID:    &session.UserID,
			SessionID: &session.ID,
			ActorID:   actor.ActorID(),
			EventType: domain.AuthAuditSessionRevoked,
			Message:   "An administrator revoked this browser session.",
			Metadata:  map[string]any{"reason": "admin_force_revoke_all"},
			CreatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	return sessions, nil
}

func (s *Service) handleDisabledUserSession(ctx context.Context, session domain.BrowserSession, now time.Time) error {
	sessions, err := s.repo.RevokeBrowserSessionsByUser(ctx, session.UserID, nil, now)
	if err != nil {
		return err
	}

	var sessionID *uuid.UUID
	if session.ID != uuid.Nil {
		sessionID = &session.ID
	}
	metadata := map[string]any{
		"reason":                "user_disabled_after_login",
		"revoked_session_count": len(sessions),
	}
	return s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		UserID:    &session.UserID,
		SessionID: sessionID,
		ActorID:   "system:auth",
		EventType: domain.AuthAuditUserDisabledAfterLogin,
		Message:   "Blocked a browser session because the user is disabled.",
		Metadata:  metadata,
		CreatedAt: now,
	})
}

func (s *Service) expireSession(ctx context.Context, session domain.BrowserSession, now time.Time) error {
	if session.RevokedAt != nil {
		return nil
	}
	if err := s.repo.RevokeBrowserSession(ctx, session.ID, now); err != nil {
		return err
	}
	return s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		UserID:    &session.UserID,
		SessionID: &session.ID,
		ActorID:   auditActorForUser(session.UserID),
		EventType: domain.AuthAuditSessionExpired,
		Message:   "Browser session expired.",
		Metadata: map[string]any{
			"reason": expirationReason(session, now),
		},
		CreatedAt: now,
	})
}

func (s *Service) recordAuditEvent(ctx context.Context, input repo.CreateAuthAuditEventInput) error {
	metadata := map[string]any{
		"auth_mode": string(s.cfg.Mode),
	}
	for key, value := range input.Metadata {
		metadata[key] = value
	}
	input.Metadata = metadata
	_, err := s.repo.CreateAuthAuditEvent(ctx, input)
	return err
}

func auditActorForUser(userID uuid.UUID) string {
	if userID == uuid.Nil {
		return ""
	}
	return "user:" + userID.String()
}

func expirationReason(session domain.BrowserSession, now time.Time) string {
	if now.After(session.IdleExpiresAt) {
		return "idle_timeout"
	}
	return "absolute_timeout"
}

func sessionDeviceMetadata(userAgent string) map[string]any {
	device := parseRawSessionDevice(userAgent)
	return map[string]any{
		"kind":    string(device.Kind),
		"os":      device.OS,
		"browser": device.Browser,
		"label":   device.Label,
	}
}

func parseRawSessionDevice(userAgent string) parsedSessionDevice {
	normalized := strings.ToLower(strings.TrimSpace(userAgent))
	device := parsedSessionDevice{
		Kind: domain.SessionDeviceKindUnknown,
	}

	switch {
	case normalized == "":
		device.Kind = domain.SessionDeviceKindUnknown
	case strings.Contains(normalized, "ipad") || strings.Contains(normalized, "tablet"):
		device.Kind = domain.SessionDeviceKindTablet
	case strings.Contains(normalized, "mobile"), strings.Contains(normalized, "iphone"), strings.Contains(normalized, "android"):
		device.Kind = domain.SessionDeviceKindMobile
	case strings.Contains(normalized, "bot"), strings.Contains(normalized, "spider"), strings.Contains(normalized, "crawler"):
		device.Kind = domain.SessionDeviceKindBot
	case strings.Contains(normalized, "windows"), strings.Contains(normalized, "macintosh"), strings.Contains(normalized, "linux"), strings.Contains(normalized, "x11"):
		device.Kind = domain.SessionDeviceKindDesktop
	}

	switch {
	case strings.Contains(normalized, "windows"):
		device.OS = "Windows"
	case strings.Contains(normalized, "mac os x"), strings.Contains(normalized, "macintosh"):
		device.OS = "macOS"
	case strings.Contains(normalized, "iphone"), strings.Contains(normalized, "ipad"), strings.Contains(normalized, "cpu os"):
		device.OS = "iOS"
	case strings.Contains(normalized, "android"):
		device.OS = "Android"
	case strings.Contains(normalized, "cros"):
		device.OS = "ChromeOS"
	case strings.Contains(normalized, "linux"), strings.Contains(normalized, "x11"):
		device.OS = "Linux"
	}

	switch {
	case strings.Contains(normalized, "edg/"):
		device.Browser = "Edge"
	case strings.Contains(normalized, "opr/"), strings.Contains(normalized, "opera"):
		device.Browser = "Opera"
	case strings.Contains(normalized, "firefox/"):
		device.Browser = "Firefox"
	case strings.Contains(normalized, "chrome/"):
		device.Browser = "Chrome"
	case strings.Contains(normalized, "safari/"):
		device.Browser = "Safari"
	}

	parts := make([]string, 0, 2)
	if device.Browser != "" {
		parts = append(parts, device.Browser)
	}
	if device.OS != "" {
		parts = append(parts, "on "+device.OS)
	}
	device.Label = strings.TrimSpace(strings.Join(parts, " "))
	if device.Label == "" {
		switch device.Kind {
		case domain.SessionDeviceKindDesktop:
			device.Label = "Desktop browser"
		case domain.SessionDeviceKindMobile:
			device.Label = "Mobile browser"
		case domain.SessionDeviceKindTablet:
			device.Label = "Tablet browser"
		case domain.SessionDeviceKindBot:
			device.Label = "Automated client"
		default:
			device.Label = "Unknown device"
		}
	}
	return device
}
