package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/labstack/echo/v4"
)

type authSessionUserResponse struct {
	ID           string `json:"id"`
	PrimaryEmail string `json:"primary_email"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
}

type authSessionResponse struct {
	AuthMode      string                   `json:"auth_mode"`
	Authenticated bool                     `json:"authenticated"`
	IssuerURL     string                   `json:"issuer_url,omitempty"`
	User          *authSessionUserResponse `json:"user,omitempty"`
	CSRFToken     string                   `json:"csrf_token,omitempty"`
	Roles         []string                 `json:"roles,omitempty"`
	Permissions   []string                 `json:"permissions,omitempty"`
}

func (s *Server) registerAuthRoutes(api *echo.Group) {
	api.GET("/auth/oidc/start", s.handleOIDCStart)
	api.GET("/auth/oidc/callback", s.handleOIDCCallback)
	api.GET("/auth/session", s.handleAuthSession)
	api.POST("/auth/logout", s.handleLogout)
}

func (s *Server) registerProtectedAuthRoutes(api *echo.Group) {
	api.GET("/auth/me/permissions", s.handleGetMyPermissions)
}

func (s *Server) handleOIDCStart(c echo.Context) error {
	if s.auth.Mode != config.AuthModeOIDC || s.humanAuthService == nil {
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
	if s.auth.Mode != config.AuthModeOIDC || s.humanAuthService == nil {
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
	response := authSessionResponse{
		AuthMode: string(s.auth.Mode),
	}
	if s.auth.Mode == config.AuthModeOIDC {
		response.IssuerURL = s.auth.OIDC.IssuerURL
	}
	if s.auth.Mode != config.AuthModeOIDC || s.humanAuthService == nil {
		return c.JSON(http.StatusOK, response)
	}
	cookie, err := c.Cookie(humanSessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return c.JSON(http.StatusOK, response)
	}
	principal, err := s.humanAuthService.AuthenticateSession(
		c.Request().Context(),
		cookie.Value,
		c.Request().UserAgent(),
		c.RealIP(),
		true,
	)
	if err != nil {
		s.clearHumanSessionCookies(c)
		return c.JSON(http.StatusOK, response)
	}
	response.Authenticated = true
	response.User = &authSessionUserResponse{
		ID:           principal.User.ID.String(),
		PrimaryEmail: principal.User.PrimaryEmail,
		DisplayName:  principal.User.DisplayName,
		AvatarURL:    principal.User.AvatarURL,
	}
	response.CSRFToken = principal.Session.CSRFSecret
	response.Roles = roleKeysToStrings(principal.EffectiveRoles)
	response.Permissions = permissionKeysToStrings(principal.Permissions)
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleLogout(c echo.Context) error {
	if s.auth.Mode == config.AuthModeOIDC && s.humanAuthService != nil {
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
