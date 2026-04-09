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

type authMethodCapability string

const (
	authMethodCapabilityOIDC               authMethodCapability = "oidc"
	authMethodCapabilityLocalBootstrapLink authMethodCapability = "local_bootstrap_link"
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
	AvailableAuthMethods       []authMethodCapability
	CurrentAuthMethod          authMethodCapability
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
	capabilities := resolveAuthCapabilities(runtimeState, s.humanAuthService != nil)
	ctx := resolvedAuthRequestContext{
		RuntimeState:         runtimeState,
		LoginRequired:        capabilities.RequiresAuthorization,
		PrincipalKind:        authRequestPrincipalKindAnonymous,
		AvailableAuthMethods: capabilities.AvailableAuthMethods,
		CurrentAuthMethod:    capabilities.CurrentAuthMethod,
		AuthConfigured:       runtimeState.AuthMode == iam.AuthModeOIDC,
	}
	if runtimeState.ResolvedOIDCConfig != nil {
		ctx.IssuerURL = runtimeState.ResolvedOIDCConfig.IssuerURL
	}
	if !capabilities.RequiresAuthorization {
		ctx.Authenticated = true
		ctx.PrincipalKind = authRequestPrincipalKindLocalBootstrap
		ctx.Roles = localBootstrapRoles()
		ctx.Permissions = localBootstrapPermissions()
		ctx.CanManageAuth = true
		return ctx, nil
	}
	if s.humanAuthService == nil {
		return ctx, nil
	}
	if !runtimeState.LoginRequired {
		return s.resolveLocalBootstrapRequestContext(c, ctx, invalidDisposition)
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

type authCapabilities struct {
	RequiresAuthorization bool
	AvailableAuthMethods  []authMethodCapability
	CurrentAuthMethod     authMethodCapability
}

func resolveAuthCapabilities(
	runtimeState iam.RuntimeAccessControlState,
	humanAuthAvailable bool,
) authCapabilities {
	if runtimeState.LoginRequired {
		return authCapabilities{
			RequiresAuthorization: true,
			AvailableAuthMethods:  []authMethodCapability{authMethodCapabilityOIDC},
			CurrentAuthMethod:     authMethodCapabilityOIDC,
		}
	}
	if humanAuthAvailable {
		return authCapabilities{
			RequiresAuthorization: true,
			AvailableAuthMethods:  []authMethodCapability{authMethodCapabilityLocalBootstrapLink},
			CurrentAuthMethod:     authMethodCapabilityLocalBootstrapLink,
		}
	}
	return authCapabilities{}
}

func (s *Server) resolveLocalBootstrapRequestContext(
	c echo.Context,
	ctx resolvedAuthRequestContext,
	invalidDisposition invalidHumanSessionDisposition,
) (resolvedAuthRequestContext, error) {
	cookie, err := c.Cookie(humanSessionCookieName)
	if err != nil || cookie.Value == "" {
		return ctx, nil
	}
	localSession, err := s.humanAuthService.AuthenticateLocalSession(
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
	ctx.PrincipalKind = authRequestPrincipalKindLocalBootstrap
	ctx.CanManageAuth = true
	ctx.CSRFToken = localSession.CSRFToken
	ctx.Roles = append([]humanauthdomain.RoleKey(nil), localSession.Roles...)
	ctx.Permissions = append([]humanauthdomain.PermissionKey(nil), localSession.Permissions...)
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

func authMethodCapabilitiesToStrings(methods []authMethodCapability) []string {
	if len(methods) == 0 {
		return nil
	}
	values := make([]string, 0, len(methods))
	for _, method := range methods {
		if method == "" {
			continue
		}
		values = append(values, string(method))
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
