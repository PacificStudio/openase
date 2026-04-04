package humanauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var (
	ErrAuthDisabled     = errors.New("human auth is disabled")
	ErrInvalidFlowState = errors.New("invalid oidc login flow state")
	ErrInvalidSession   = errors.New("invalid browser session")
	ErrSessionExpired   = errors.New("browser session expired")
	ErrUserDisabled     = errors.New("user is disabled")
	ErrPermissionDenied = errors.New("permission denied")
	ErrUnauthorized     = errors.New("human session required")
)

const flowCookieTTL = 10 * time.Minute

type Service struct {
	cfg        config.AuthConfig
	repo       *repo.Repository
	httpClient *http.Client
	mu         sync.Mutex
	provider   *oidc.Provider
	verifier   *oidc.IDTokenVerifier
	oauth      *oauth2.Config
}

type LoginStart struct {
	RedirectURL     string
	FlowCookieValue string
}

type CallbackResult struct {
	SessionToken string
	CSRFToken    string
	ReturnTo     string
	Principal    domain.AuthenticatedPrincipal
}

type flowState struct {
	State        string    `json:"state"`
	Nonce        string    `json:"nonce"`
	CodeVerifier string    `json:"code_verifier"`
	ReturnTo     string    `json:"return_to"`
	IssuedAt     time.Time `json:"issued_at"`
}

func NewService(cfg config.AuthConfig, repository *repo.Repository, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Service{
		cfg:        cfg,
		repo:       repository,
		httpClient: httpClient,
	}
}

func (s *Service) Mode() config.AuthMode {
	return s.cfg.Mode
}

func (s *Service) StartLogin(_ context.Context, returnTo string) (LoginStart, error) {
	if s.cfg.Mode != config.AuthModeOIDC {
		return LoginStart{}, ErrAuthDisabled
	}
	state, err := randomToken(24)
	if err != nil {
		return LoginStart{}, fmt.Errorf("generate state: %w", err)
	}
	nonce, err := randomToken(24)
	if err != nil {
		return LoginStart{}, fmt.Errorf("generate nonce: %w", err)
	}
	codeVerifier := oauth2.GenerateVerifier()
	if strings.TrimSpace(returnTo) == "" {
		returnTo = "/"
	}
	encodedFlow, err := s.encodeFlowState(flowState{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		ReturnTo:     returnTo,
		IssuedAt:     time.Now().UTC(),
	})
	if err != nil {
		return LoginStart{}, err
	}
	oauthConfig, err := s.oauthConfig(context.Background())
	if err != nil {
		return LoginStart{}, err
	}
	redirectURL := oauthConfig.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(codeVerifier),
	)
	return LoginStart{
		RedirectURL:     redirectURL,
		FlowCookieValue: encodedFlow,
	}, nil
}

func (s *Service) HandleCallback(
	ctx context.Context,
	code string,
	state string,
	flowCookieValue string,
	userAgent string,
	ip string,
) (CallbackResult, error) {
	if s.cfg.Mode != config.AuthModeOIDC {
		return CallbackResult{}, ErrAuthDisabled
	}
	flow, err := s.decodeFlowState(flowCookieValue)
	if err != nil {
		return CallbackResult{}, err
	}
	if time.Since(flow.IssuedAt) > flowCookieTTL || strings.TrimSpace(flow.State) != strings.TrimSpace(state) {
		return CallbackResult{}, ErrInvalidFlowState
	}
	oauthConfig, err := s.oauthConfig(ctx)
	if err != nil {
		return CallbackResult{}, err
	}
	token, err := oauthConfig.Exchange(
		ctx,
		strings.TrimSpace(code),
		oauth2.VerifierOption(flow.CodeVerifier),
	)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("exchange oidc code: %w", err)
	}
	rawIDToken, _ := token.Extra("id_token").(string)
	if strings.TrimSpace(rawIDToken) == "" {
		return CallbackResult{}, errors.New("oidc callback missing id_token")
	}
	verifier, err := s.idTokenVerifier(ctx)
	if err != nil {
		return CallbackResult{}, err
	}
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("verify oidc id_token: %w", err)
	}
	profile, err := s.parseOIDCProfile(idToken)
	if err != nil {
		return CallbackResult{}, err
	}
	if err := s.validateProfile(profile); err != nil {
		return CallbackResult{}, err
	}
	user, identity, groups, err := s.repo.UpsertUserFromOIDC(ctx, profile)
	if err != nil {
		return CallbackResult{}, err
	}
	if user.Status == domain.UserStatusDisabled {
		return CallbackResult{}, ErrUserDisabled
	}
	if s.shouldBootstrapAdmin(user) {
		if _, err := s.repo.EnsureBootstrapRoleBinding(ctx, user, "system:bootstrap-admin"); err != nil {
			return CallbackResult{}, err
		}
	}
	sessionToken, err := randomToken(32)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("generate session token: %w", err)
	}
	csrfToken, err := randomToken(24)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("generate csrf token: %w", err)
	}
	session, err := s.repo.CreateBrowserSession(ctx, repo.CreateBrowserSessionInput{
		UserID:        user.ID,
		SessionHash:   hashToken(sessionToken),
		ExpiresAt:     time.Now().Add(s.cfg.OIDC.SessionTTL),
		IdleExpiresAt: time.Now().Add(s.cfg.OIDC.SessionIdleTTL),
		CSRFSecret:    csrfToken,
		UserAgentHash: hashValue(userAgent),
		IPPrefix:      ipPrefix(ip),
	})
	if err != nil {
		return CallbackResult{}, err
	}
	principal, err := s.buildPrincipal(ctx, session, user, identity, groups)
	if err != nil {
		return CallbackResult{}, err
	}
	return CallbackResult{
		SessionToken: sessionToken,
		CSRFToken:    csrfToken,
		ReturnTo:     flow.ReturnTo,
		Principal:    principal,
	}, nil
}

func (s *Service) AuthenticateSession(
	ctx context.Context,
	sessionToken string,
	userAgent string,
	ip string,
	touch bool,
) (domain.AuthenticatedPrincipal, error) {
	if s.cfg.Mode != config.AuthModeOIDC {
		return domain.AuthenticatedPrincipal{}, ErrAuthDisabled
	}
	session, err := s.repo.GetBrowserSessionByHash(ctx, hashToken(sessionToken))
	if err != nil {
		return domain.AuthenticatedPrincipal{}, ErrInvalidSession
	}
	now := time.Now().UTC()
	if session.RevokedAt != nil {
		return domain.AuthenticatedPrincipal{}, ErrInvalidSession
	}
	if now.After(session.ExpiresAt) || now.After(session.IdleExpiresAt) {
		return domain.AuthenticatedPrincipal{}, ErrSessionExpired
	}
	if session.UserAgentHash != "" && session.UserAgentHash != hashValue(userAgent) {
		return domain.AuthenticatedPrincipal{}, ErrInvalidSession
	}
	if session.IPPrefix != "" && session.IPPrefix != ipPrefix(ip) {
		return domain.AuthenticatedPrincipal{}, ErrInvalidSession
	}
	user, err := s.repo.GetUser(ctx, session.UserID)
	if err != nil {
		return domain.AuthenticatedPrincipal{}, ErrInvalidSession
	}
	if user.Status == domain.UserStatusDisabled {
		return domain.AuthenticatedPrincipal{}, ErrUserDisabled
	}
	identity, err := s.repo.GetPrimaryIdentity(ctx, user.ID)
	if err != nil {
		return domain.AuthenticatedPrincipal{}, ErrInvalidSession
	}
	groups, err := s.repo.ListUserGroups(ctx, user.ID)
	if err != nil {
		return domain.AuthenticatedPrincipal{}, ErrInvalidSession
	}
	if touch {
		session, err = s.repo.TouchBrowserSession(ctx, session.ID, now.Add(s.cfg.OIDC.SessionIdleTTL))
		if err != nil {
			return domain.AuthenticatedPrincipal{}, fmt.Errorf("extend session idle ttl: %w", err)
		}
	}
	return s.buildPrincipal(ctx, session, user, identity, groups)
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	if s.cfg.Mode != config.AuthModeOIDC {
		return nil
	}
	session, err := s.repo.GetBrowserSessionByHash(ctx, hashToken(sessionToken))
	if err != nil {
		return nil
	}
	return s.repo.RevokeBrowserSession(ctx, session.ID, time.Now().UTC())
}

func (s *Service) ListRoleBindings(
	ctx context.Context,
	scope domain.ScopeKind,
	scopeID string,
) ([]domain.RoleBinding, error) {
	return s.repo.ListRoleBindings(ctx, repo.ListRoleBindingsFilter{
		ScopeKind: &scope,
		ScopeID:   stringPointer(strings.TrimSpace(scopeID)),
	})
}

func (s *Service) CreateRoleBinding(ctx context.Context, input domain.RoleBinding) (domain.RoleBinding, error) {
	return s.repo.CreateRoleBinding(ctx, input)
}

func (s *Service) DeleteRoleBinding(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRoleBinding(ctx, id)
}

func (s *Service) CountApprovalPolicies(ctx context.Context) (int, error) {
	if s.cfg.Mode != config.AuthModeOIDC {
		return 0, ErrAuthDisabled
	}
	return s.repo.CountApprovalPolicies(ctx)
}

func (s *Service) buildPrincipal(
	ctx context.Context,
	session domain.BrowserSession,
	user domain.User,
	identity domain.UserIdentity,
	groups []domain.UserGroupMembership,
) (domain.AuthenticatedPrincipal, error) {
	authorizer := NewAuthorizer(s.repo)
	roles, permissions, err := authorizer.Evaluate(ctx, user, identity, groups, domain.ScopeRef{Kind: domain.ScopeKindInstance, ID: ""})
	if err != nil {
		return domain.AuthenticatedPrincipal{}, err
	}
	return domain.AuthenticatedPrincipal{
		User:           user,
		Identity:       identity,
		Groups:         groups,
		Session:        session,
		EffectiveRoles: roles,
		Permissions:    permissions,
	}, nil
}

func (s *Service) oauthConfig(ctx context.Context) (*oauth2.Config, error) {
	if err := s.ensureProvider(ctx); err != nil {
		return nil, err
	}
	return s.oauth, nil
}

func (s *Service) idTokenVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	if err := s.ensureProvider(ctx); err != nil {
		return nil, err
	}
	return s.verifier, nil
}

func (s *Service) ensureProvider(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.provider != nil && s.oauth != nil && s.verifier != nil {
		return nil
	}
	oidcCtx := oidc.ClientContext(ctx, s.httpClient)
	provider, err := oidc.NewProvider(oidcCtx, s.cfg.OIDC.IssuerURL)
	if err != nil {
		return fmt.Errorf("initialize oidc provider: %w", err)
	}
	s.provider = provider
	s.verifier = provider.Verifier(&oidc.Config{ClientID: s.cfg.OIDC.ClientID})
	s.oauth = &oauth2.Config{
		ClientID:     s.cfg.OIDC.ClientID,
		ClientSecret: s.cfg.OIDC.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  s.cfg.OIDC.RedirectURL,
		Scopes:       append([]string(nil), s.cfg.OIDC.Scopes...),
	}
	return nil
}

func (s *Service) parseOIDCProfile(idToken *oidc.IDToken) (domain.OIDCProfile, error) {
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return domain.OIDCProfile{}, fmt.Errorf("decode oidc claims: %w", err)
	}
	rawClaims, err := json.Marshal(claims)
	if err != nil {
		return domain.OIDCProfile{}, fmt.Errorf("marshal oidc claims: %w", err)
	}
	email := strings.ToLower(strings.TrimSpace(stringClaim(claims, s.cfg.OIDC.EmailClaim)))
	if email == "" {
		return domain.OIDCProfile{}, errors.New("oidc claims missing email")
	}
	groups := parseGroupsClaim(claims[s.cfg.OIDC.GroupsClaim])
	return domain.OIDCProfile{
		Issuer:        idToken.Issuer,
		Subject:       idToken.Subject,
		Email:         email,
		EmailVerified: boolClaim(claims, "email_verified"),
		DisplayName:   strings.TrimSpace(stringClaim(claims, s.cfg.OIDC.NameClaim)),
		Username:      strings.TrimSpace(stringClaim(claims, s.cfg.OIDC.UsernameClaim)),
		AvatarURL:     strings.TrimSpace(stringClaim(claims, "picture")),
		Groups:        groups,
		RawClaimsJSON: string(rawClaims),
	}, nil
}

func (s *Service) validateProfile(profile domain.OIDCProfile) error {
	if len(s.cfg.OIDC.AllowedEmailDomains) == 0 {
		return nil
	}
	at := strings.LastIndex(profile.Email, "@")
	if at < 0 {
		return errors.New("oidc email claim must be a valid email address")
	}
	domainPart := profile.Email[at+1:]
	for _, allowed := range s.cfg.OIDC.AllowedEmailDomains {
		if strings.EqualFold(domainPart, allowed) {
			return nil
		}
	}
	return errors.New("oidc email domain is not allowed")
}

func (s *Service) shouldBootstrapAdmin(user domain.User) bool {
	email := strings.ToLower(strings.TrimSpace(user.PrimaryEmail))
	for _, candidate := range s.cfg.OIDC.BootstrapAdminEmails {
		if email == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func (s *Service) encodeFlowState(state flowState) (string, error) {
	body, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode oidc flow state: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(s.cfg.OIDC.ClientSecret))
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + signature, nil
}

func (s *Service) decodeFlowState(raw string) (flowState, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return flowState{}, ErrInvalidFlowState
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.OIDC.ClientSecret))
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(actual, expected) {
		return flowState{}, ErrInvalidFlowState
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return flowState{}, ErrInvalidFlowState
	}
	var state flowState
	if err := json.Unmarshal(body, &state); err != nil {
		return flowState{}, ErrInvalidFlowState
	}
	return state, nil
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func hashValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func ipPrefix(raw string) string {
	host := strings.TrimSpace(raw)
	if host == "" {
		return ""
	}
	if parsed := net.ParseIP(host); parsed == nil {
		if candidate, _, err := net.SplitHostPort(host); err == nil {
			host = candidate
		}
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return fmt.Sprintf("%d.%d.%d", ipv4[0], ipv4[1], ipv4[2])
	}
	return ip.Mask(net.CIDRMask(64, 128)).String()
}

func stringClaim(claims map[string]any, key string) string {
	value := claims[key]
	typed, _ := value.(string)
	return typed
}

func boolClaim(claims map[string]any, key string) bool {
	value := claims[key]
	typed, _ := value.(bool)
	return typed
}

func parseGroupsClaim(raw any) []domain.Group {
	switch value := raw.(type) {
	case []any:
		groups := make([]domain.Group, 0, len(value))
		for _, item := range value {
			typed, ok := item.(string)
			if !ok {
				continue
			}
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				continue
			}
			groups = append(groups, domain.Group{Key: strings.ToLower(trimmed), Name: trimmed})
		}
		return groups
	case []string:
		groups := make([]domain.Group, 0, len(value))
		for _, item := range value {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			groups = append(groups, domain.Group{Key: strings.ToLower(trimmed), Name: trimmed})
		}
		return groups
	default:
		return []domain.Group{}
	}
}

func NormalizeReturnTo(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.IsAbs() || !strings.HasPrefix(parsed.Path, "/") {
		return "/"
	}
	return parsed.RequestURI()
}

func stringPointer(value string) *string {
	return &value
}
