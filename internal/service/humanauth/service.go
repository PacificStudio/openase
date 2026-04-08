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
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var (
	ErrAuthDisabled        = errors.New("human auth is disabled")
	ErrInvalidFlowState    = errors.New("invalid oidc login flow state")
	ErrInvalidSession      = errors.New("invalid browser session")
	ErrSessionExpired      = errors.New("browser session expired")
	ErrSessionNotFound     = errors.New("browser session not found")
	ErrUserDisabled        = errors.New("user is disabled")
	ErrUserNotFound        = errors.New("user not found")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrUnauthorized        = errors.New("human session required")
	ErrRoleBindingNotFound = errors.New("role binding not found")
)

type permissionDeniedDetail struct {
	message string
}

func (e permissionDeniedDetail) Error() string { return e.message }

func (e permissionDeniedDetail) Unwrap() error { return ErrPermissionDenied }

func permissionDeniedf(message string) error {
	return permissionDeniedDetail{message: message}
}

const flowCookieTTL = 10 * time.Minute

type Service struct {
	repo          *repo.Repository
	httpClient    *http.Client
	stateResolver AccessControlStateResolver
	mu            sync.Mutex
	providerKey   string
	provider      *oidc.Provider
	verifier      *oidc.IDTokenVerifier
	oauth         *oauth2.Config
}

type AccessControlStateResolver interface {
	RuntimeState(ctx context.Context) (iam.RuntimeAccessControlState, error)
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

type OIDCProviderDiagnostics struct {
	IssuerURL             string
	AuthorizationEndpoint string
	TokenEndpoint         string
}

type CreateRoleBindingInput struct {
	SubjectKind string
	SubjectKey  string
	RoleKey     string
	GrantedBy   string
	ExpiresAt   *time.Time
}

type UpdateRoleBindingInput struct {
	SubjectKind string
	SubjectKey  string
	RoleKey     string
	GrantedBy   string
	ExpiresAt   *time.Time
}

type flowState struct {
	State        string    `json:"state"`
	Nonce        string    `json:"nonce"`
	CodeVerifier string    `json:"code_verifier"`
	ReturnTo     string    `json:"return_to"`
	IssuedAt     time.Time `json:"issued_at"`
}

func NewService(repository *repo.Repository, httpClient *http.Client, stateResolver AccessControlStateResolver) *Service {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Service{
		repo:          repository,
		httpClient:    httpClient,
		stateResolver: stateResolver,
	}
}

func InspectOIDCProvider(
	ctx context.Context,
	cfg config.AuthConfig,
	httpClient *http.Client,
) (OIDCProviderDiagnostics, error) {
	resolver, err := newStaticAccessControlResolver(cfg)
	if err != nil {
		return OIDCProviderDiagnostics{}, err
	}
	service := NewService(nil, httpClient, resolver)
	oidcConfig, err := service.currentOIDCConfig(ctx)
	if err != nil {
		return OIDCProviderDiagnostics{}, err
	}
	if _, err := service.oauthConfig(ctx, oidcConfig); err != nil {
		return OIDCProviderDiagnostics{}, err
	}
	return OIDCProviderDiagnostics{
		IssuerURL:             strings.TrimSpace(oidcConfig.IssuerURL),
		AuthorizationEndpoint: service.provider.Endpoint().AuthURL,
		TokenEndpoint:         service.provider.Endpoint().TokenURL,
	}, nil
}

func (s *Service) StartLogin(ctx context.Context, returnTo string) (LoginStart, error) {
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
	oidcConfig, err := s.currentOIDCConfig(ctx)
	if err != nil {
		return LoginStart{}, err
	}
	encodedFlow, err := s.encodeFlowState(flowState{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		ReturnTo:     returnTo,
		IssuedAt:     time.Now().UTC(),
	}, oidcConfig)
	if err != nil {
		return LoginStart{}, err
	}
	oauthConfig, err := s.oauthConfig(ctx, oidcConfig)
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
) (result CallbackResult, err error) {
	oidcConfig, err := s.currentOIDCConfig(ctx)
	if err != nil {
		return CallbackResult{}, err
	}
	var auditUserID *uuid.UUID
	defer func() {
		if err == nil {
			return
		}
		_ = s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
			UserID:    auditUserID,
			ActorID:   "",
			EventType: domain.AuthAuditLoginFailed,
			Message:   "OIDC sign-in failed.",
			Metadata: map[string]any{
				"error":      err.Error(),
				"ip_prefix":  ipPrefix(ip),
				"user_agent": sessionDeviceMetadata(userAgent),
			},
			CreatedAt: time.Now().UTC(),
		})
	}()

	flow, err := s.decodeFlowState(flowCookieValue, oidcConfig)
	if err != nil {
		return CallbackResult{}, err
	}
	if time.Since(flow.IssuedAt) > flowCookieTTL || strings.TrimSpace(flow.State) != strings.TrimSpace(state) {
		return CallbackResult{}, ErrInvalidFlowState
	}
	oauthConfig, err := s.oauthConfig(ctx, oidcConfig)
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
	verifier, err := s.idTokenVerifier(ctx, oidcConfig)
	if err != nil {
		return CallbackResult{}, err
	}
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("verify oidc id_token: %w", err)
	}
	profile, err := s.parseOIDCProfile(idToken, oidcConfig)
	if err != nil {
		return CallbackResult{}, err
	}
	if err := s.validateProfile(profile, oidcConfig); err != nil {
		return CallbackResult{}, err
	}
	user, identity, groups, err := s.repo.UpsertUserFromOIDC(ctx, profile)
	if err != nil {
		return CallbackResult{}, err
	}
	auditUserID = &user.ID
	if user.Status == domain.UserStatusDisabled {
		now := time.Now().UTC()
		_ = s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
			UserID:    &user.ID,
			ActorID:   auditActorForUser(user.ID),
			EventType: domain.AuthAuditUserDisabledAfterLogin,
			Message:   "Blocked sign-in because the user is disabled.",
			Metadata:  map[string]any{"reason": "user_disabled_before_session_created"},
			CreatedAt: now,
		})
		return CallbackResult{}, ErrUserDisabled
	}
	if s.shouldBootstrapAdmin(user, oidcConfig) {
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
	device := parseRawSessionDevice(userAgent)
	session, err := s.repo.CreateBrowserSession(ctx, repo.CreateBrowserSessionInput{
		UserID:        user.ID,
		SessionHash:   hashToken(sessionToken),
		DeviceKind:    device.Kind,
		DeviceOS:      device.OS,
		DeviceBrowser: device.Browser,
		DeviceLabel:   device.Label,
		ExpiresAt:     time.Now().Add(oidcConfig.SessionPolicy.SessionTTL),
		IdleExpiresAt: time.Now().Add(oidcConfig.SessionPolicy.SessionIdleTTL),
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
	if err := s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		UserID:    &user.ID,
		SessionID: &session.ID,
		ActorID:   auditActorForUser(user.ID),
		EventType: domain.AuthAuditLoginSucceeded,
		Message:   "Signed in via OIDC.",
		Metadata: map[string]any{
			"device":    sessionDeviceMetadata(userAgent),
			"ip_prefix": ipPrefix(ip),
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
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
	oidcConfig, err := s.currentOIDCConfig(ctx)
	if err != nil {
		return domain.AuthenticatedPrincipal{}, err
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
		_ = s.expireSession(ctx, session, now)
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
		_ = s.handleDisabledUserSession(ctx, session, now)
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
		session, err = s.repo.TouchBrowserSession(ctx, session.ID, now.Add(oidcConfig.SessionPolicy.SessionIdleTTL))
		if err != nil {
			return domain.AuthenticatedPrincipal{}, fmt.Errorf("extend session idle ttl: %w", err)
		}
	}
	return s.buildPrincipal(ctx, session, user, identity, groups)
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	if _, err := s.currentOIDCConfig(ctx); errors.Is(err, ErrAuthDisabled) {
		return nil
	} else if err != nil {
		return err
	}
	session, err := s.repo.GetBrowserSessionByHash(ctx, hashToken(sessionToken))
	if err != nil {
		return nil
	}
	if session.RevokedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	if err := s.repo.RevokeBrowserSession(ctx, session.ID, now); err != nil {
		return err
	}
	return s.recordAuditEvent(ctx, repo.CreateAuthAuditEventInput{
		UserID:    &session.UserID,
		SessionID: &session.ID,
		ActorID:   auditActorForUser(session.UserID),
		EventType: domain.AuthAuditLogout,
		Message:   "Signed out of the current browser session.",
		Metadata:  map[string]any{"reason": "user_logout"},
		CreatedAt: now,
	})
}

func (s *Service) ListRoleBindings(
	ctx context.Context,
	scope domain.ScopeKind,
	scopeID string,
) ([]domain.RoleBinding, error) {
	switch scope {
	case domain.ScopeKindInstance:
		items, err := s.repo.ListInstanceRoleBindings(ctx)
		if err != nil {
			return nil, err
		}
		return mapGenericRoleBindings(items), nil
	case domain.ScopeKindOrganization:
		parsed, err := uuid.Parse(strings.TrimSpace(scopeID))
		if err != nil {
			return nil, fmt.Errorf("organization id must be a UUID")
		}
		items, err := s.repo.ListOrganizationRoleBindings(ctx, parsed)
		if err != nil {
			return nil, err
		}
		return mapGenericRoleBindings(items), nil
	case domain.ScopeKindProject:
		parsed, err := uuid.Parse(strings.TrimSpace(scopeID))
		if err != nil {
			return nil, fmt.Errorf("project id must be a UUID")
		}
		items, err := s.repo.ListProjectRoleBindings(ctx, parsed)
		if err != nil {
			return nil, err
		}
		return mapGenericRoleBindings(items), nil
	default:
		return nil, fmt.Errorf("unsupported scope kind %q", scope)
	}
}

func (s *Service) CreateInstanceRoleBinding(ctx context.Context, input CreateRoleBindingInput) (domain.InstanceRoleBinding, error) {
	subject, err := s.resolveRoleBindingSubject(ctx, input.SubjectKind, input.SubjectKey)
	if err != nil {
		return domain.InstanceRoleBinding{}, err
	}
	roleKey, err := domain.ParseInstanceRole(input.RoleKey)
	if err != nil {
		return domain.InstanceRoleBinding{}, err
	}
	return s.repo.CreateInstanceRoleBinding(ctx, domain.InstanceRoleBinding{
		RoleBindingMetadata: domain.RoleBindingMetadata{
			Subject:   subject,
			GrantedBy: strings.TrimSpace(input.GrantedBy),
			ExpiresAt: input.ExpiresAt,
		},
		RoleKey: roleKey,
	})
}

func (s *Service) CreateOrganizationRoleBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
	input CreateRoleBindingInput,
) (domain.OrganizationRoleBinding, error) {
	if err := requireOrganizationRoleBindingPrivilege(ctx, s.repo, organizationID, actor, domain.RoleKey(strings.TrimSpace(input.RoleKey))); err != nil {
		return domain.OrganizationRoleBinding{}, err
	}
	subject, err := s.resolveRoleBindingSubject(ctx, input.SubjectKind, input.SubjectKey)
	if err != nil {
		return domain.OrganizationRoleBinding{}, err
	}
	roleKey, err := domain.ParseOrganizationRole(input.RoleKey)
	if err != nil {
		return domain.OrganizationRoleBinding{}, err
	}
	return s.repo.CreateOrganizationRoleBinding(ctx, domain.OrganizationRoleBinding{
		RoleBindingMetadata: domain.RoleBindingMetadata{
			Subject:   subject,
			GrantedBy: strings.TrimSpace(input.GrantedBy),
			ExpiresAt: input.ExpiresAt,
		},
		OrganizationID: organizationID.String(),
		RoleKey:        roleKey,
	})
}

func (s *Service) CreateProjectRoleBinding(
	ctx context.Context,
	projectID uuid.UUID,
	input CreateRoleBindingInput,
) (domain.ProjectRoleBinding, error) {
	subject, err := s.resolveRoleBindingSubject(ctx, input.SubjectKind, input.SubjectKey)
	if err != nil {
		return domain.ProjectRoleBinding{}, err
	}
	roleKey, err := domain.ParseProjectRole(input.RoleKey)
	if err != nil {
		return domain.ProjectRoleBinding{}, err
	}
	return s.repo.CreateProjectRoleBinding(ctx, domain.ProjectRoleBinding{
		RoleBindingMetadata: domain.RoleBindingMetadata{
			Subject:   subject,
			GrantedBy: strings.TrimSpace(input.GrantedBy),
			ExpiresAt: input.ExpiresAt,
		},
		ProjectID: projectID.String(),
		RoleKey:   roleKey,
	})
}

func (s *Service) UpdateInstanceRoleBinding(
	ctx context.Context,
	id uuid.UUID,
	input UpdateRoleBindingInput,
) (domain.InstanceRoleBinding, error) {
	subject, err := s.resolveRoleBindingSubject(ctx, input.SubjectKind, input.SubjectKey)
	if err != nil {
		return domain.InstanceRoleBinding{}, err
	}
	roleKey, err := domain.ParseInstanceRole(input.RoleKey)
	if err != nil {
		return domain.InstanceRoleBinding{}, err
	}
	item, err := s.repo.UpdateInstanceRoleBinding(ctx, id, domain.UpdateInstanceRoleBinding{
		UpdateRoleBindingMetadata: domain.UpdateRoleBindingMetadata{
			Subject:   subject,
			GrantedBy: strings.TrimSpace(input.GrantedBy),
			ExpiresAt: input.ExpiresAt,
		},
		RoleKey: roleKey,
	})
	if errors.Is(err, repo.ErrRoleBindingNotFound) {
		return domain.InstanceRoleBinding{}, ErrRoleBindingNotFound
	}
	return item, err
}

func (s *Service) UpdateOrganizationRoleBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	input UpdateRoleBindingInput,
) (domain.OrganizationRoleBinding, error) {
	subject, err := s.resolveRoleBindingSubject(ctx, input.SubjectKind, input.SubjectKey)
	if err != nil {
		return domain.OrganizationRoleBinding{}, err
	}
	roleKey, err := domain.ParseOrganizationRole(input.RoleKey)
	if err != nil {
		return domain.OrganizationRoleBinding{}, err
	}
	item, err := s.repo.UpdateOrganizationRoleBinding(ctx, organizationID, id, domain.UpdateOrganizationRoleBinding{
		UpdateRoleBindingMetadata: domain.UpdateRoleBindingMetadata{
			Subject:   subject,
			GrantedBy: strings.TrimSpace(input.GrantedBy),
			ExpiresAt: input.ExpiresAt,
		},
		RoleKey: roleKey,
	})
	if errors.Is(err, repo.ErrRoleBindingNotFound) {
		return domain.OrganizationRoleBinding{}, ErrRoleBindingNotFound
	}
	return item, err
}

func (s *Service) UpdateProjectRoleBinding(
	ctx context.Context,
	projectID uuid.UUID,
	id uuid.UUID,
	input UpdateRoleBindingInput,
) (domain.ProjectRoleBinding, error) {
	subject, err := s.resolveRoleBindingSubject(ctx, input.SubjectKind, input.SubjectKey)
	if err != nil {
		return domain.ProjectRoleBinding{}, err
	}
	roleKey, err := domain.ParseProjectRole(input.RoleKey)
	if err != nil {
		return domain.ProjectRoleBinding{}, err
	}
	item, err := s.repo.UpdateProjectRoleBinding(ctx, projectID, id, domain.UpdateProjectRoleBinding{
		UpdateRoleBindingMetadata: domain.UpdateRoleBindingMetadata{
			Subject:   subject,
			GrantedBy: strings.TrimSpace(input.GrantedBy),
			ExpiresAt: input.ExpiresAt,
		},
		RoleKey: roleKey,
	})
	if errors.Is(err, repo.ErrRoleBindingNotFound) {
		return domain.ProjectRoleBinding{}, ErrRoleBindingNotFound
	}
	return item, err
}

func (s *Service) DeleteInstanceRoleBinding(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteInstanceRoleBinding(ctx, id)
	if errors.Is(err, repo.ErrRoleBindingNotFound) {
		return ErrRoleBindingNotFound
	}
	return err
}

func (s *Service) DeleteOrganizationRoleBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
	id uuid.UUID,
) error {
	if err := requireOrganizationRoleBindingDeletePrivilege(ctx, s.repo, organizationID, actor, id); err != nil {
		return err
	}
	err := s.repo.DeleteOrganizationRoleBinding(ctx, organizationID, id)
	if errors.Is(err, repo.ErrRoleBindingNotFound) {
		return ErrRoleBindingNotFound
	}
	return err
}

func (s *Service) DeleteProjectRoleBinding(ctx context.Context, projectID uuid.UUID, id uuid.UUID) error {
	err := s.repo.DeleteProjectRoleBinding(ctx, projectID, id)
	if errors.Is(err, repo.ErrRoleBindingNotFound) {
		return ErrRoleBindingNotFound
	}
	return err
}

func requireOrganizationRoleBindingPrivilege(
	ctx context.Context,
	repository *repo.Repository,
	organizationID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
	role domain.RoleKey,
) error {
	if !isPrivilegedOrganizationRole(role) {
		return nil
	}
	allowed, err := canManagePrivilegedOrganizationRoles(ctx, repository, organizationID, actor)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}
	return permissionDeniedf("organization owner role is required to grant or revoke org_owner or org_admin")
}

func requireOrganizationRoleBindingDeletePrivilege(
	ctx context.Context,
	repository *repo.Repository,
	organizationID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
	bindingID uuid.UUID,
) error {
	items, err := repository.ListOrganizationRoleBindings(ctx, organizationID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == bindingID {
			return requireOrganizationRoleBindingPrivilege(ctx, repository, organizationID, actor, domain.RoleKey(item.RoleKey))
		}
	}
	return ErrRoleBindingNotFound
}

func canManagePrivilegedOrganizationRoles(
	ctx context.Context,
	repository *repo.Repository,
	organizationID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
) (bool, error) {
	if principalHasRole(actor, domain.RoleInstanceAdmin) {
		return true, nil
	}
	membership, err := repository.GetOrganizationMembershipByUser(ctx, organizationID, actor.User.ID)
	if errors.Is(err, repo.ErrOrganizationMembershipNotFound) {
		return false, permissionDeniedf("active organization membership is required")
	}
	if err != nil {
		return false, err
	}
	return membership.Status == domain.OrganizationMembershipStatusActive &&
		membership.Role == domain.OrganizationMembershipRoleOwner, nil
}

func isPrivilegedOrganizationRole(role domain.RoleKey) bool {
	return role == domain.RoleOrgOwner || role == domain.RoleOrgAdmin
}

func (s *Service) CountApprovalPolicies(ctx context.Context) (int, error) {
	if _, err := s.currentOIDCConfig(ctx); err != nil {
		return 0, err
	}
	return s.repo.CountApprovalPolicies(ctx)
}

type genericRoleBinding interface {
	Generic() domain.RoleBinding
}

func mapGenericRoleBindings[T genericRoleBinding](items []T) []domain.RoleBinding {
	result := make([]domain.RoleBinding, 0, len(items))
	for _, item := range items {
		result = append(result, item.Generic())
	}
	return result
}

func (s *Service) resolveRoleBindingSubject(
	ctx context.Context,
	subjectKind string,
	subjectKey string,
) (domain.SubjectRef, error) {
	kind, err := domain.ParseSubjectKind(subjectKind)
	if err != nil {
		return domain.SubjectRef{}, err
	}
	switch kind {
	case domain.SubjectKindUser:
		user, resolveErr := s.repo.ResolveRoleBindingUser(ctx, subjectKey)
		if resolveErr != nil {
			switch {
			case errors.Is(resolveErr, repo.ErrRoleBindingUserNotFound):
				return domain.SubjectRef{}, fmt.Errorf("user binding subject must reference an existing user id or email")
			case errors.Is(resolveErr, repo.ErrRoleBindingUserAmbiguous):
				return domain.SubjectRef{}, fmt.Errorf("user binding subject matches multiple users; use a user id")
			default:
				return domain.SubjectRef{}, resolveErr
			}
		}
		return domain.NewUserSubjectRef(user.ID), nil
	case domain.SubjectKindGroup:
		return domain.ParseGroupSubjectRef(subjectKey)
	default:
		return domain.SubjectRef{}, fmt.Errorf("unsupported subject kind %q", subjectKind)
	}
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

func (s *Service) runtimeState(ctx context.Context) (iam.RuntimeAccessControlState, error) {
	if s.stateResolver == nil {
		return iam.RuntimeAccessControlState{}, ErrAuthDisabled
	}
	return s.stateResolver.RuntimeState(ctx)
}

func (s *Service) currentOIDCConfig(ctx context.Context) (iam.ActiveOIDCConfig, error) {
	state, err := s.runtimeState(ctx)
	if err != nil {
		return iam.ActiveOIDCConfig{}, err
	}
	if !state.LoginRequired || state.ResolvedOIDCConfig == nil {
		return iam.ActiveOIDCConfig{}, ErrAuthDisabled
	}
	return *state.ResolvedOIDCConfig, nil
}

func (s *Service) oauthConfig(ctx context.Context, oidcConfig iam.ActiveOIDCConfig) (*oauth2.Config, error) {
	if err := s.ensureProvider(ctx, oidcConfig); err != nil {
		return nil, err
	}
	return s.oauth, nil
}

func (s *Service) idTokenVerifier(ctx context.Context, oidcConfig iam.ActiveOIDCConfig) (*oidc.IDTokenVerifier, error) {
	if err := s.ensureProvider(ctx, oidcConfig); err != nil {
		return nil, err
	}
	return s.verifier, nil
}

func (s *Service) ensureProvider(ctx context.Context, oidcConfig iam.ActiveOIDCConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := oidcConfigFingerprint(oidcConfig)
	if s.provider != nil && s.oauth != nil && s.verifier != nil && s.providerKey == key {
		return nil
	}
	oidcCtx := oidc.ClientContext(ctx, s.httpClient)
	provider, err := oidc.NewProvider(oidcCtx, oidcConfig.IssuerURL)
	if err != nil {
		return fmt.Errorf("initialize oidc provider: %w", err)
	}
	s.provider = provider
	s.providerKey = key
	s.verifier = provider.Verifier(&oidc.Config{ClientID: oidcConfig.ClientID})
	s.oauth = &oauth2.Config{
		ClientID:     oidcConfig.ClientID,
		ClientSecret: oidcConfig.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  oidcConfig.RedirectURL,
		Scopes:       append([]string(nil), oidcConfig.Scopes...),
	}
	return nil
}

func (s *Service) parseOIDCProfile(idToken *oidc.IDToken, oidcConfig iam.ActiveOIDCConfig) (domain.OIDCProfile, error) {
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return domain.OIDCProfile{}, fmt.Errorf("decode oidc claims: %w", err)
	}
	rawClaims, err := json.Marshal(claims)
	if err != nil {
		return domain.OIDCProfile{}, fmt.Errorf("marshal oidc claims: %w", err)
	}
	email := strings.ToLower(strings.TrimSpace(stringClaim(claims, oidcConfig.Claims.EmailClaim)))
	if email == "" {
		return domain.OIDCProfile{}, errors.New("oidc claims missing email")
	}
	groups := parseGroupsClaim(claims[oidcConfig.Claims.GroupsClaim])
	return domain.OIDCProfile{
		Issuer:        idToken.Issuer,
		Subject:       idToken.Subject,
		Email:         email,
		EmailVerified: boolClaim(claims, "email_verified"),
		DisplayName:   strings.TrimSpace(stringClaim(claims, oidcConfig.Claims.NameClaim)),
		Username:      strings.TrimSpace(stringClaim(claims, oidcConfig.Claims.UsernameClaim)),
		AvatarURL:     strings.TrimSpace(stringClaim(claims, "picture")),
		Groups:        groups,
		RawClaimsJSON: string(rawClaims),
	}, nil
}

func (s *Service) validateProfile(profile domain.OIDCProfile, oidcConfig iam.ActiveOIDCConfig) error {
	if len(oidcConfig.AllowedEmailDomains) == 0 {
		return nil
	}
	at := strings.LastIndex(profile.Email, "@")
	if at < 0 {
		return errors.New("oidc email claim must be a valid email address")
	}
	domainPart := profile.Email[at+1:]
	for _, allowed := range oidcConfig.AllowedEmailDomains {
		if strings.EqualFold(domainPart, allowed) {
			return nil
		}
	}
	return errors.New("oidc email domain is not allowed")
}

func (s *Service) shouldBootstrapAdmin(user domain.User, oidcConfig iam.ActiveOIDCConfig) bool {
	email := strings.ToLower(strings.TrimSpace(user.PrimaryEmail))
	for _, candidate := range oidcConfig.BootstrapAdminEmails {
		if email == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func (s *Service) encodeFlowState(state flowState, oidcConfig iam.ActiveOIDCConfig) (string, error) {
	body, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode oidc flow state: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(oidcConfig.ClientSecret))
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + signature, nil
}

func (s *Service) decodeFlowState(raw string, oidcConfig iam.ActiveOIDCConfig) (flowState, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return flowState{}, ErrInvalidFlowState
	}
	mac := hmac.New(sha256.New, []byte(oidcConfig.ClientSecret))
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

type staticAccessControlResolver struct {
	runtimeState iam.RuntimeAccessControlState
}

func newStaticAccessControlResolver(cfg config.AuthConfig) (staticAccessControlResolver, error) {
	var mode string
	switch strings.ToLower(strings.TrimSpace(string(cfg.Mode))) {
	case string(config.AuthModeOIDC):
		mode = iam.AccessControlStatusActive.String()
	default:
		mode = iam.AccessControlStatusAbsent.String()
	}
	state, err := iam.ParseAccessControlState(iam.AccessControlStateInput{
		Status:               mode,
		IssuerURL:            strings.TrimSpace(cfg.OIDC.IssuerURL),
		ClientID:             strings.TrimSpace(cfg.OIDC.ClientID),
		ClientSecret:         strings.TrimSpace(cfg.OIDC.ClientSecret),
		RedirectURL:          strings.TrimSpace(cfg.OIDC.RedirectURL),
		Scopes:               append([]string(nil), cfg.OIDC.Scopes...),
		EmailClaim:           strings.TrimSpace(cfg.OIDC.EmailClaim),
		NameClaim:            strings.TrimSpace(cfg.OIDC.NameClaim),
		UsernameClaim:        strings.TrimSpace(cfg.OIDC.UsernameClaim),
		GroupsClaim:          strings.TrimSpace(cfg.OIDC.GroupsClaim),
		AllowedEmailDomains:  append([]string(nil), cfg.OIDC.AllowedEmailDomains...),
		BootstrapAdminEmails: append([]string(nil), cfg.OIDC.BootstrapAdminEmails...),
		SessionTTL:           cfg.OIDC.SessionTTL.String(),
		SessionIdleTTL:       cfg.OIDC.SessionIdleTTL.String(),
	})
	if err != nil {
		return staticAccessControlResolver{}, err
	}
	return staticAccessControlResolver{runtimeState: iam.ResolveRuntimeAccessControlState(state)}, nil
}

func (r staticAccessControlResolver) RuntimeState(context.Context) (iam.RuntimeAccessControlState, error) {
	return r.runtimeState, nil
}

func oidcConfigFingerprint(oidcConfig iam.ActiveOIDCConfig) string {
	return strings.Join([]string{
		oidcConfig.IssuerURL,
		oidcConfig.ClientID,
		oidcConfig.ClientSecret,
		oidcConfig.RedirectURL,
		strings.Join(oidcConfig.Scopes, ","),
		oidcConfig.Claims.EmailClaim,
		oidcConfig.Claims.NameClaim,
		oidcConfig.Claims.UsernameClaim,
		oidcConfig.Claims.GroupsClaim,
		oidcConfig.SessionPolicy.SessionTTL.String(),
		oidcConfig.SessionPolicy.SessionIdleTTL.String(),
	}, "|")
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
