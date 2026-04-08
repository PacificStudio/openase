package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	aiPrincipalCookieName   = "openase_ai_principal"
	aiPrincipalCookiePrefix = "browser-session:"
	aiPrincipalCookieTTL    = 365 * 24 * time.Hour
)

func (s *Server) currentRequestAIPrincipal(c echo.Context) (chatservice.UserID, error) {
	if actor := strings.TrimSpace(actorFromHumanPrincipal(c)); actor != "" {
		return chatservice.ParseUserID(actor)
	}
	if s != nil {
		runtimeState, err := s.currentRuntimeAccessControlState(c)
		if err != nil {
			return "", err
		}
		if runtimeState.LoginRequired {
			return "", humanauthservice.ErrUnauthorized
		}
	}
	return ensureServerDefinedAIPrincipal(c)
}

func ensureServerDefinedAIPrincipal(c echo.Context) (chatservice.UserID, error) {
	if c == nil {
		return "", fmt.Errorf("ai principal context is required")
	}
	if cookie, err := c.Cookie(aiPrincipalCookieName); err == nil {
		if principal, parseErr := parseAICookiePrincipal(cookie.Value); parseErr == nil {
			return principal, nil
		}
	}

	principal, err := chatservice.ParseUserID(aiPrincipalCookiePrefix + uuid.NewString())
	if err != nil {
		return "", err
	}

	c.SetCookie(&http.Cookie{
		Name:     aiPrincipalCookieName,
		Value:    principal.String(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure(c),
		MaxAge:   int(aiPrincipalCookieTTL.Seconds()),
	})
	return principal, nil
}

func parseAICookiePrincipal(raw string) (chatservice.UserID, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, aiPrincipalCookiePrefix) {
		return "", fmt.Errorf("ai principal cookie must start with %q", aiPrincipalCookiePrefix)
	}
	if _, err := uuid.Parse(strings.TrimPrefix(trimmed, aiPrincipalCookiePrefix)); err != nil {
		return "", fmt.Errorf("ai principal cookie must contain a valid browser session id: %w", err)
	}
	return chatservice.ParseUserID(trimmed)
}
