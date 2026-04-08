package httpapi

import (
	"errors"

	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/labstack/echo/v4"
)

type authRequestPrincipalKind string

const (
	authRequestPrincipalKindAnonymous      authRequestPrincipalKind = "anonymous"
	authRequestPrincipalKindLocalBootstrap authRequestPrincipalKind = "local_bootstrap"
	authRequestPrincipalKindHumanSession   authRequestPrincipalKind = "human_session"
)

type invalidHumanSessionDisposition uint8

const (
	invalidHumanSessionAsAnonymous invalidHumanSessionDisposition = iota
	invalidHumanSessionAsError
)

type resolvedAuthRequestContext struct {
	RuntimeState               iam.RuntimeAccessControlState
	LoginRequired              bool
	Authenticated              bool
	PrincipalKind              authRequestPrincipalKind
	AuthConfigured             bool
	SessionGovernanceAvailable bool
	CanManageAuth              bool
	IssuerURL                  string
	User                       *authSessionUserResponse
	CSRFToken                  string
	Roles                      []humanauthdomain.RoleKey
	Permissions                []humanauthdomain.PermissionKey
	Groups                     []humanauthdomain.UserGroupMembership
	HumanPrincipal             *humanauthdomain.AuthenticatedPrincipal
}

func (s *Server) resolveAuthRequestContext(
	c echo.Context,
	invalidDisposition invalidHumanSessionDisposition,
) (resolvedAuthRequestContext, error) {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return resolvedAuthRequestContext{}, err
	}
	ctx := resolvedAuthRequestContext{
		RuntimeState:   runtimeState,
		LoginRequired:  runtimeState.LoginRequired,
		AuthConfigured: runtimeState.AuthMode == iam.AuthModeOIDC,
		PrincipalKind:  authRequestPrincipalKindAnonymous,
	}
	if runtimeState.ResolvedOIDCConfig != nil {
		ctx.IssuerURL = runtimeState.ResolvedOIDCConfig.IssuerURL
	}
	if !runtimeState.LoginRequired || s.humanAuthService == nil {
		ctx.Authenticated = true
		ctx.PrincipalKind = authRequestPrincipalKindLocalBootstrap
		ctx.Roles = localBootstrapRoles()
		ctx.Permissions = localBootstrapPermissions()
		ctx.CanManageAuth = true
		return ctx, nil
	}
	cookie, err := c.Cookie(humanSessionCookieName)
	if err != nil || cookie.Value == "" {
		return ctx, nil
	}
	principal, err := s.humanAuthService.AuthenticateSession(
		c.Request().Context(),
		cookie.Value,
		c.Request().UserAgent(),
		c.RealIP(),
		true,
	)
	if err != nil {
		if invalidDisposition == invalidHumanSessionAsAnonymous {
			s.clearHumanSessionCookies(c)
			return ctx, nil
		}
		return resolvedAuthRequestContext{}, err
	}
	ctx.Authenticated = true
	ctx.PrincipalKind = authRequestPrincipalKindHumanSession
	ctx.SessionGovernanceAvailable = runtimeState.SessionGovernanceEnabled
	ctx.CanManageAuth = authPermissionsIncludeManageAuth(principal.Permissions)
	ctx.User = &authSessionUserResponse{
		ID:           principal.User.ID.String(),
		PrimaryEmail: principal.User.PrimaryEmail,
		DisplayName:  principal.User.DisplayName,
		AvatarURL:    principal.User.AvatarURL,
	}
	ctx.CSRFToken = principal.Session.CSRFSecret
	ctx.Roles = append([]humanauthdomain.RoleKey(nil), principal.EffectiveRoles...)
	ctx.Permissions = append([]humanauthdomain.PermissionKey(nil), principal.Permissions...)
	ctx.Groups = append([]humanauthdomain.UserGroupMembership(nil), principal.Groups...)
	ctx.HumanPrincipal = &principal
	return ctx, nil
}

func writeHumanSessionAuthError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, humanauthservice.ErrUserDisabled):
		return writeAPIError(c, 401, "HUMAN_USER_DISABLED", err.Error())
	case errors.Is(err, humanauthservice.ErrSessionExpired):
		return writeAPIError(c, 401, "HUMAN_SESSION_EXPIRED", err.Error())
	case errors.Is(err, humanauthservice.ErrInvalidSession),
		errors.Is(err, humanauthservice.ErrSessionNotFound),
		errors.Is(err, humanauthservice.ErrUnauthorized):
		return writeAPIError(c, 401, "HUMAN_SESSION_INVALID", err.Error())
	default:
		return writeAPIError(c, 500, "AUTHENTICATION_FAILED", err.Error())
	}
}

func localBootstrapRoles() []humanauthdomain.RoleKey {
	return []humanauthdomain.RoleKey{humanauthdomain.RoleInstanceAdmin}
}

func localBootstrapPermissions() []humanauthdomain.PermissionKey {
	return humanauthdomain.PermissionsForRoles(localBootstrapRoles())
}

func authPermissionsIncludeManageAuth(permissions []humanauthdomain.PermissionKey) bool {
	for _, permission := range permissions {
		if permission == humanauthdomain.PermissionSecurityRead || permission == humanauthdomain.PermissionSecurityUpdate {
			return true
		}
	}
	return false
}
