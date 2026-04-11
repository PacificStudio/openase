package humanauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/google/uuid"
)

var (
	ErrLocalBootstrapDisabled    = errors.New("local bootstrap authorization is disabled")
	ErrLocalBootstrapInvalid     = errors.New("invalid local bootstrap authorization request")
	ErrLocalBootstrapExpired     = errors.New("local bootstrap authorization request expired")
	ErrLocalBootstrapAlreadyUsed = errors.New("local bootstrap authorization request was already used")
)

const (
	DefaultLocalBootstrapRequestTTL     = 10 * time.Minute
	defaultLocalBootstrapSessionTTL     = 0
	defaultLocalBootstrapSessionIdleTTL = 0
	localBootstrapPurposeBrowserSession = "browser_session"
	localBootstrapActorID               = "local_instance_admin:default"
)

type LocalBootstrapIssueInput struct {
	RequestedBy string
	Purpose     string
	TTL         time.Duration
}

type LocalBootstrapIssueResult struct {
	RequestID string    `json:"request_id"`
	Code      string    `json:"code"`
	Nonce     string    `json:"nonce"`
	Purpose   string    `json:"purpose"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LocalSessionAuthentication struct {
	Session     domain.BrowserSession
	CSRFToken   string
	Roles       []domain.RoleKey
	Permissions []domain.PermissionKey
}

func (s *Service) CreateLocalBootstrapRequest(
	ctx context.Context,
	input LocalBootstrapIssueInput,
) (LocalBootstrapIssueResult, error) {
	if err := s.ensureLocalBootstrapAllowed(ctx); err != nil {
		return LocalBootstrapIssueResult{}, err
	}

	ttl := input.TTL
	if ttl <= 0 {
		ttl = DefaultLocalBootstrapRequestTTL
	}
	purpose := strings.TrimSpace(input.Purpose)
	if purpose == "" {
		purpose = localBootstrapPurposeBrowserSession
	}

	code, err := randomToken(24)
	if err != nil {
		return LocalBootstrapIssueResult{}, fmt.Errorf("generate local bootstrap code: %w", err)
	}
	nonce, err := randomToken(24)
	if err != nil {
		return LocalBootstrapIssueResult{}, fmt.Errorf("generate local bootstrap nonce: %w", err)
	}

	request, err := s.repo.CreateLocalBootstrapAuthRequest(ctx, repo.CreateLocalBootstrapAuthRequestInput{
		CodeHash:    hashToken(code),
		NonceHash:   hashToken(nonce),
		Purpose:     purpose,
		RequestedBy: strings.TrimSpace(input.RequestedBy),
		ExpiresAt:   time.Now().UTC().Add(ttl),
	})
	if err != nil {
		return LocalBootstrapIssueResult{}, err
	}

	_ = s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		ActorID:   strings.TrimSpace(input.RequestedBy),
		EventType: domain.AuthAuditLocalBootstrapIssued,
		Message:   "Issued a local bootstrap authorization request.",
		Metadata: map[string]any{
			"request_id": request.ID.String(),
			"purpose":    request.Purpose,
			"expires_at": request.ExpiresAt.UTC().Format(time.RFC3339),
		},
		CreatedAt: time.Now().UTC(),
	})

	return LocalBootstrapIssueResult{
		RequestID: request.ID.String(),
		Code:      code,
		Nonce:     nonce,
		Purpose:   request.Purpose,
		ExpiresAt: request.ExpiresAt.UTC(),
	}, nil
}

func (s *Service) RedeemLocalBootstrapRequest(
	ctx context.Context,
	requestID string,
	code string,
	nonce string,
	userAgent string,
	ip string,
) (result CallbackResult, err error) {
	if err := s.ensureLocalBootstrapAllowed(ctx); err != nil {
		return CallbackResult{}, err
	}

	parsedRequestID, err := uuid.Parse(strings.TrimSpace(requestID))
	if err != nil {
		return CallbackResult{}, ErrLocalBootstrapInvalid
	}
	request, err := s.repo.GetLocalBootstrapAuthRequest(ctx, parsedRequestID)
	if err != nil {
		return CallbackResult{}, ErrLocalBootstrapInvalid
	}
	if request.UsedAt != nil {
		return CallbackResult{}, ErrLocalBootstrapAlreadyUsed
	}
	now := time.Now().UTC()
	if now.After(request.ExpiresAt) {
		_ = s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
			ActorID:   localBootstrapActorID,
			EventType: domain.AuthAuditLocalBootstrapFailed,
			Message:   "Rejected an expired local bootstrap authorization request.",
			Metadata:  map[string]any{"request_id": request.ID.String(), "reason": "expired"},
			CreatedAt: now,
		})
		return CallbackResult{}, ErrLocalBootstrapExpired
	}
	if hashToken(code) != request.CodeHash || hashToken(nonce) != request.NonceHash {
		_ = s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
			ActorID:   localBootstrapActorID,
			EventType: domain.AuthAuditLocalBootstrapFailed,
			Message:   "Rejected an invalid local bootstrap authorization request.",
			Metadata:  map[string]any{"request_id": request.ID.String(), "reason": "mismatch"},
			CreatedAt: now,
		})
		return CallbackResult{}, ErrLocalBootstrapInvalid
	}

	sessionToken, err := randomToken(32)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("generate local bootstrap session token: %w", err)
	}
	csrfToken, err := randomToken(24)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("generate local bootstrap csrf token: %w", err)
	}
	device := parseRawSessionDevice(userAgent)
	session, err := s.repo.CreateBrowserSession(ctx, repo.CreateBrowserSessionInput{
		UserID:        uuid.Nil,
		SessionHash:   hashToken(sessionToken),
		DeviceKind:    device.Kind,
		DeviceOS:      device.OS,
		DeviceBrowser: device.Browser,
		DeviceLabel:   device.Label,
		ExpiresAt:     sessionDeadline(now, defaultLocalBootstrapSessionTTL),
		IdleExpiresAt: sessionDeadline(now, defaultLocalBootstrapSessionIdleTTL),
		CSRFSecret:    csrfToken,
		UserAgentHash: hashValue(userAgent),
		IPPrefix:      ipPrefix(ip),
	})
	if err != nil {
		return CallbackResult{}, err
	}

	used, err := s.repo.MarkLocalBootstrapAuthRequestUsed(ctx, request.ID, session.ID, now)
	if err != nil {
		return CallbackResult{}, err
	}
	if used.UsedSessionID != nil && *used.UsedSessionID != session.ID {
		_ = s.repo.RevokeBrowserSession(ctx, session.ID, now)
		return CallbackResult{}, ErrLocalBootstrapAlreadyUsed
	}

	roles := []domain.RoleKey{domain.RoleInstanceAdmin}
	permissions := domain.PermissionsForRoles(roles)
	_ = s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		SessionID: &session.ID,
		ActorID:   localBootstrapActorID,
		EventType: domain.AuthAuditLocalBootstrapRedeemed,
		Message:   "Redeemed a local bootstrap authorization request.",
		Metadata: map[string]any{
			"request_id": request.ID.String(),
			"device":     sessionDeviceMetadata(userAgent),
			"ip_prefix":  ipPrefix(ip),
		},
		CreatedAt: now,
	})

	return CallbackResult{
		SessionToken: sessionToken,
		CSRFToken:    csrfToken,
		ReturnTo:     "/",
		Principal: domain.AuthenticatedPrincipal{
			Session:        session,
			EffectiveRoles: roles,
			Permissions:    permissions,
		},
	}, nil
}

func (s *Service) AuthenticateLocalSession(
	ctx context.Context,
	sessionToken string,
	userAgent string,
	ip string,
	touch bool,
) (LocalSessionAuthentication, error) {
	if err := s.ensureLocalBootstrapAllowed(ctx); err != nil {
		return LocalSessionAuthentication{}, err
	}

	session, err := s.repo.GetBrowserSessionByHash(ctx, hashToken(sessionToken))
	if err != nil {
		return LocalSessionAuthentication{}, ErrInvalidSession
	}
	if session.UserID != uuid.Nil {
		return LocalSessionAuthentication{}, ErrInvalidSession
	}

	now := time.Now().UTC()
	if session.RevokedAt != nil {
		return LocalSessionAuthentication{}, ErrInvalidSession
	}
	if browserSessionExpired(now, session) {
		_ = s.repo.RevokeBrowserSession(ctx, session.ID, now)
		_ = s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
			SessionID: &session.ID,
			ActorID:   localBootstrapActorID,
			EventType: domain.AuthAuditSessionExpired,
			Message:   "Local bootstrap browser session expired.",
			Metadata:  map[string]any{"reason": "session_expired"},
			CreatedAt: now,
		})
		return LocalSessionAuthentication{}, ErrSessionExpired
	}
	if session.UserAgentHash != "" && session.UserAgentHash != hashValue(userAgent) {
		return LocalSessionAuthentication{}, ErrInvalidSession
	}
	if session.IPPrefix != "" && session.IPPrefix != ipPrefix(ip) {
		return LocalSessionAuthentication{}, ErrInvalidSession
	}
	if touch {
		session, err = s.repo.TouchBrowserSession(
			ctx,
			session.ID,
			sessionRefreshAbsoluteDeadline(session.ExpiresAt, now, defaultLocalBootstrapSessionTTL),
			sessionDeadline(now, defaultLocalBootstrapSessionIdleTTL),
		)
		if err != nil {
			return LocalSessionAuthentication{}, fmt.Errorf("refresh local bootstrap session deadlines: %w", err)
		}
	}

	roles := []domain.RoleKey{domain.RoleInstanceAdmin}
	return LocalSessionAuthentication{
		Session:     session,
		CSRFToken:   session.CSRFSecret,
		Roles:       roles,
		Permissions: domain.PermissionsForRoles(roles),
	}, nil
}

func (s *Service) LogoutLocalSession(ctx context.Context, sessionToken string) error {
	if err := s.ensureLocalBootstrapAllowed(ctx); err != nil {
		return nil
	}
	session, err := s.repo.GetBrowserSessionByHash(ctx, hashToken(sessionToken))
	if err != nil {
		return nil
	}
	if session.UserID != uuid.Nil || session.RevokedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	if err := s.repo.RevokeBrowserSession(ctx, session.ID, now); err != nil {
		return err
	}
	return s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		SessionID: &session.ID,
		ActorID:   localBootstrapActorID,
		EventType: domain.AuthAuditLogout,
		Message:   "Signed out of the local bootstrap browser session.",
		Metadata:  map[string]any{"reason": "user_logout"},
		CreatedAt: now,
	})
}

func (s *Service) ensureLocalBootstrapAllowed(ctx context.Context) error {
	state, err := s.runtimeState(ctx)
	if err != nil {
		if errors.Is(err, ErrAuthDisabled) {
			return nil
		}
		return err
	}
	if state.AuthMode == iam.AuthModeOIDC || state.ResolvedOIDCConfig != nil {
		return ErrLocalBootstrapDisabled
	}
	return nil
}
