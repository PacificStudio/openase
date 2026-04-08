package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type authSessionUserResponse struct {
	ID           string `json:"id"`
	PrimaryEmail string `json:"primary_email"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
}

type authSessionResponse struct {
	AuthMode                   string                   `json:"auth_mode"`
	LoginRequired              bool                     `json:"login_required"`
	Authenticated              bool                     `json:"authenticated"`
	PrincipalKind              string                   `json:"principal_kind"`
	AuthConfigured             bool                     `json:"auth_configured"`
	SessionGovernanceAvailable bool                     `json:"session_governance_available"`
	CanManageAuth              bool                     `json:"can_manage_auth"`
	IssuerURL                  string                   `json:"issuer_url,omitempty"`
	User                       *authSessionUserResponse `json:"user,omitempty"`
	CSRFToken                  string                   `json:"csrf_token,omitempty"`
	Roles                      []string                 `json:"roles,omitempty"`
	Permissions                []string                 `json:"permissions,omitempty"`
}

type authSessionDeviceResponse struct {
	Kind    string `json:"kind"`
	OS      string `json:"os,omitempty"`
	Browser string `json:"browser,omitempty"`
	Label   string `json:"label"`
}

type authManagedSessionResponse struct {
	ID            string                    `json:"id"`
	Current       bool                      `json:"current"`
	Device        authSessionDeviceResponse `json:"device"`
	IPSummary     string                    `json:"ip_summary,omitempty"`
	CreatedAt     string                    `json:"created_at"`
	LastActiveAt  string                    `json:"last_active_at"`
	ExpiresAt     string                    `json:"expires_at"`
	IdleExpiresAt string                    `json:"idle_expires_at"`
}

type authAuditEventResponse struct {
	ID        string         `json:"id"`
	EventType string         `json:"event_type"`
	ActorID   string         `json:"actor_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}

type authStepUpCapabilityResponse struct {
	Status           string   `json:"status"`
	Summary          string   `json:"summary"`
	SupportedMethods []string `json:"supported_methods"`
}

type authSessionsResponse struct {
	AuthMode         string                       `json:"auth_mode"`
	CurrentSessionID string                       `json:"current_session_id,omitempty"`
	Sessions         []authManagedSessionResponse `json:"sessions"`
	AuditEvents      []authAuditEventResponse     `json:"audit_events"`
	StepUp           authStepUpCapabilityResponse `json:"step_up"`
}

type authRevokeSessionsResponse struct {
	RevokedCount          int    `json:"revoked_count"`
	UserID                string `json:"user_id,omitempty"`
	CurrentSessionRevoked bool   `json:"current_session_revoked,omitempty"`
}

func (s *Server) registerAuthRoutes(api *echo.Group) {
	api.GET("/auth/oidc/start", s.handleOIDCStart)
	api.GET("/auth/oidc/callback", s.handleOIDCCallback)
	api.GET("/auth/session", s.handleAuthSession)
	api.GET("/auth/me/permissions", s.handleGetMyPermissions)
	api.POST("/auth/logout", s.handleLogout)
}

func (s *Server) registerProtectedAuthRoutes(api *echo.Group) {
	api.GET("/auth/sessions", s.handleListSessions)
	api.DELETE("/auth/sessions/:id", s.handleDeleteSession)
	api.POST("/auth/sessions/revoke-all", s.handleRevokeAllSessions)
	api.POST("/auth/users/:userId/sessions/revoke", s.handleAdminRevokeUserSessions)
	api.DELETE("/instance/sessions/:id", s.handleAdminRevokeSession)
}

func (s *Server) handleOIDCStart(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if !runtimeState.LoginRequired || s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "oidc login is not enabled")
	}
	start, err := s.humanAuthService.StartLogin(c.Request().Context(), humanauthservice.NormalizeReturnTo(c.QueryParam("return_to")))
	if err != nil {
		return writeAPIError(c, http.StatusBadGateway, "OIDC_LOGIN_FAILED", err.Error())
	}
	s.setFlowCookie(c, start.FlowCookieValue)
	return c.Redirect(http.StatusFound, start.RedirectURL)
}

func (s *Server) handleOIDCCallback(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if !runtimeState.LoginRequired || s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "oidc login is not enabled")
	}
	flowCookie, err := c.Cookie(oidcFlowCookieName)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "OIDC_FLOW_MISSING", "oidc login flow cookie is missing")
	}
	result, err := s.humanAuthService.HandleCallback(
		c.Request().Context(),
		c.QueryParam("code"),
		c.QueryParam("state"),
		flowCookie.Value,
		c.Request().UserAgent(),
		c.RealIP(),
	)
	if err != nil {
		s.clearOIDCFlowCookie(c)
		return writeAPIError(c, http.StatusUnauthorized, "OIDC_CALLBACK_FAILED", err.Error())
	}
	s.setHumanSessionCookie(c, result.SessionToken)
	s.clearOIDCFlowCookie(c)
	return c.Redirect(http.StatusFound, humanauthservice.NormalizeReturnTo(result.ReturnTo))
}

func (s *Server) handleAuthSession(c echo.Context) error {
	authContext, err := s.resolveAuthRequestContext(c, invalidHumanSessionAsAnonymous)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	response := authSessionResponse{
		AuthMode:                   authContext.RuntimeState.AuthMode.String(),
		LoginRequired:              authContext.LoginRequired,
		Authenticated:              authContext.Authenticated,
		PrincipalKind:              string(authContext.PrincipalKind),
		AuthConfigured:             authContext.AuthConfigured,
		SessionGovernanceAvailable: authContext.SessionGovernanceAvailable,
		CanManageAuth:              authContext.CanManageAuth,
		IssuerURL:                  authContext.IssuerURL,
		User:                       authContext.User,
		CSRFToken:                  authContext.CSRFToken,
		Roles:                      roleKeysToStrings(authContext.Roles),
		Permissions:                permissionKeysToStrings(authContext.Permissions),
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleLogout(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if runtimeState.LoginRequired && s.humanAuthService != nil {
		if cookie, err := c.Cookie(humanSessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
			principal, authErr := s.humanAuthService.AuthenticateSession(
				c.Request().Context(),
				cookie.Value,
				c.Request().UserAgent(),
				c.RealIP(),
				false,
			)
			if authErr == nil {
				if err := s.validateMutatingHumanRequest(c, principal); err != nil {
					return err
				}
			}
			_ = s.humanAuthService.Logout(c.Request().Context(), cookie.Value)
		}
	}
	s.clearHumanSessionCookies(c)
	s.clearOIDCFlowCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleListSessions(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	response := authSessionsResponse{
		AuthMode:    runtimeState.AuthMode.String(),
		Sessions:    []authManagedSessionResponse{},
		AuditEvents: []authAuditEventResponse{},
		StepUp:      mapStepUpCapability(humanauthservice.ReservedStepUpCapability()),
	}
	if !runtimeState.SessionGovernanceEnabled || s.humanAuthService == nil {
		return c.JSON(http.StatusOK, response)
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", humanauthservice.ErrUnauthorized.Error())
	}
	snapshot, err := s.humanAuthService.ListSessionGovernance(c.Request().Context(), principal)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SESSION_GOVERNANCE_FAILED", err.Error())
	}
	response.CurrentSessionID = principal.Session.ID.String()
	response.StepUp = mapStepUpCapability(snapshot.StepUp)
	for _, session := range snapshot.Sessions {
		response.Sessions = append(response.Sessions, mapManagedSessionResponse(session, principal.Session.ID))
	}
	for _, event := range snapshot.AuditEvents {
		response.AuditEvents = append(response.AuditEvents, mapAuthAuditEventResponse(event))
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleDeleteSession(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if !runtimeState.SessionGovernanceEnabled || s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "session governance is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", humanauthservice.ErrUnauthorized.Error())
	}
	sessionID, err := parseUUIDPathParamValue(c, "id")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SESSION_ID", err.Error())
	}
	_, isCurrent, err := s.humanAuthService.RevokeSession(c.Request().Context(), principal, sessionID)
	if err != nil {
		switch {
		case errors.Is(err, humanauthservice.ErrSessionNotFound), errors.Is(err, humanauthservice.ErrPermissionDenied):
			return writeAPIError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "browser session not found")
		default:
			return writeAPIError(c, http.StatusInternalServerError, "SESSION_REVOKE_FAILED", err.Error())
		}
	}
	if isCurrent {
		s.clearHumanSessionCookies(c)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleRevokeAllSessions(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if !runtimeState.SessionGovernanceEnabled || s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "session governance is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", humanauthservice.ErrUnauthorized.Error())
	}
	sessions, err := s.humanAuthService.RevokeOtherSessions(c.Request().Context(), principal)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SESSION_REVOKE_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, authRevokeSessionsResponse{
		RevokedCount: len(sessions),
	})
}

func (s *Server) handleAdminRevokeUserSessions(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if !runtimeState.SessionGovernanceEnabled || s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "session governance is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", humanauthservice.ErrUnauthorized.Error())
	}
	userID, err := parseUUIDPathParamValue(c, "userId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_USER_ID", err.Error())
	}
	sessions, err := s.humanAuthService.ForceRevokeUserSessions(c.Request().Context(), principal, userID)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SESSION_REVOKE_FAILED", err.Error())
	}
	if userID == principal.User.ID {
		s.clearHumanSessionCookies(c)
	}
	return c.JSON(http.StatusOK, authRevokeSessionsResponse{
		RevokedCount:          len(sessions),
		UserID:                userID.String(),
		CurrentSessionRevoked: userID == principal.User.ID,
	})
}

func (s *Server) handleAdminRevokeSession(c echo.Context) error {
	runtimeState, err := s.currentRuntimeAccessControlState(c)
	if err != nil {
		return writeAuthRuntimeUnavailable(c, "AUTH_RUNTIME_STATE_FAILED", err)
	}
	if !runtimeState.SessionGovernanceEnabled || s.humanAuthService == nil {
		return writeAPIError(c, http.StatusNotFound, "AUTH_DISABLED", "session governance is only available when oidc auth is enabled")
	}
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", humanauthservice.ErrUnauthorized.Error())
	}
	sessionID, err := parseUUIDPathParamValue(c, "id")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SESSION_ID", err.Error())
	}
	session, isCurrent, err := s.humanAuthService.ForceRevokeSession(c.Request().Context(), principal, sessionID)
	if err != nil {
		switch {
		case errors.Is(err, humanauthservice.ErrSessionNotFound):
			return writeAPIError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "browser session not found")
		default:
			return writeAPIError(c, http.StatusInternalServerError, "SESSION_REVOKE_FAILED", err.Error())
		}
	}
	if isCurrent {
		s.clearHumanSessionCookies(c)
	}
	return c.JSON(http.StatusOK, authRevokeSessionsResponse{
		RevokedCount:          1,
		UserID:                session.UserID.String(),
		CurrentSessionRevoked: isCurrent,
	})
}

func (s *Server) setHumanSessionCookie(c echo.Context, sessionToken string) {
	c.SetCookie(&http.Cookie{
		Name:     humanSessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure(c),
	})
}

func (s *Server) setFlowCookie(c echo.Context, value string) {
	c.SetCookie(&http.Cookie{
		Name:     oidcFlowCookieName,
		Value:    value,
		Path:     oidcFlowCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure(c),
		MaxAge:   int((10 * time.Minute).Seconds()),
	})
}

func (s *Server) clearHumanSessionCookies(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     humanSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure(c),
		MaxAge:   -1,
	})
}

func (s *Server) clearOIDCFlowCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     oidcFlowCookieName,
		Value:    "",
		Path:     oidcFlowCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure(c),
		MaxAge:   -1,
	})
}

func cookieSecure(c echo.Context) bool {
	return c.Request().TLS != nil || strings.EqualFold(c.Request().Header.Get("X-Forwarded-Proto"), "https")
}

func mapManagedSessionResponse(session humanauthdomain.BrowserSession, currentSessionID uuid.UUID) authManagedSessionResponse {
	label := strings.TrimSpace(session.DeviceLabel)
	if label == "" {
		label = "Unknown device"
	}
	ipSummary := strings.TrimSpace(session.IPPrefix)
	if ipSummary == "" {
		ipSummary = "Unavailable"
	}
	return authManagedSessionResponse{
		ID:      session.ID.String(),
		Current: session.ID == currentSessionID,
		Device: authSessionDeviceResponse{
			Kind:    string(session.DeviceKind),
			OS:      session.DeviceOS,
			Browser: session.DeviceBrowser,
			Label:   label,
		},
		IPSummary:     ipSummary,
		CreatedAt:     session.CreatedAt.UTC().Format(time.RFC3339),
		LastActiveAt:  session.UpdatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:     session.ExpiresAt.UTC().Format(time.RFC3339),
		IdleExpiresAt: session.IdleExpiresAt.UTC().Format(time.RFC3339),
	}
}

func mapAuthAuditEventResponse(event humanauthdomain.AuthAuditEvent) authAuditEventResponse {
	response := authAuditEventResponse{
		ID:        event.ID.String(),
		EventType: string(event.EventType),
		ActorID:   event.ActorID,
		Message:   event.Message,
		Metadata:  event.Metadata,
		CreatedAt: event.CreatedAt.UTC().Format(time.RFC3339),
	}
	if event.SessionID != nil {
		response.SessionID = event.SessionID.String()
	}
	return response
}

func mapStepUpCapability(capability humanauthservice.StepUpCapability) authStepUpCapabilityResponse {
	return authStepUpCapabilityResponse{
		Status:           capability.Status,
		Summary:          capability.Summary,
		SupportedMethods: capability.SupportedMethods,
	}
}
