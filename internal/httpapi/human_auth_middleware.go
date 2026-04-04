package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/config"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/labstack/echo/v4"
)

func (s *Server) requireHumanSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if s.auth.Mode != config.AuthModeOIDC || s.humanAuthService == nil {
			return next(c)
		}
		cookie, err := c.Cookie(humanSessionCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", humanauthservice.ErrUnauthorized.Error())
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
			switch {
			case errors.Is(err, humanauthservice.ErrUserDisabled):
				return writeAPIError(c, http.StatusUnauthorized, "HUMAN_USER_DISABLED", err.Error())
			case errors.Is(err, humanauthservice.ErrSessionExpired):
				return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_EXPIRED", err.Error())
			default:
				return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_INVALID", err.Error())
			}
		}
		setHumanPrincipal(c, principal)
		if err := s.validateMutatingHumanRequest(c, principal); err != nil {
			return err
		}
		return next(c)
	}
}

func (s *Server) validateMutatingHumanRequest(
	c echo.Context,
	principal humanauthdomain.AuthenticatedPrincipal,
) error {
	switch c.Request().Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return nil
	}
	if !sameOriginRequest(c.Request()) {
		return writeAPIError(c, http.StatusForbidden, "CSRF_ORIGIN_FORBIDDEN", "origin or referer must match this host")
	}
	if strings.TrimSpace(c.Request().Header.Get(csrfHeaderName)) != strings.TrimSpace(principal.Session.CSRFSecret) {
		return writeAPIError(c, http.StatusForbidden, "CSRF_TOKEN_INVALID", "csrf token is missing or invalid")
	}
	return nil
}

func sameOriginRequest(req *http.Request) bool {
	targetScheme := "http"
	if req.TLS != nil || strings.EqualFold(req.Header.Get("X-Forwarded-Proto"), "https") {
		targetScheme = "https"
	}
	targetHost := strings.TrimSpace(req.Host)
	origins := []string{
		strings.TrimSpace(req.Header.Get("Origin")),
		strings.TrimSpace(req.Header.Get("Referer")),
	}
	for _, candidate := range origins {
		if candidate == "" {
			continue
		}
		parsed, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		if strings.EqualFold(parsed.Scheme, targetScheme) && strings.EqualFold(parsed.Host, targetHost) {
			return true
		}
	}
	return false
}
