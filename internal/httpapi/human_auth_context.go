package httpapi

import (
	"strings"

	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/labstack/echo/v4"
)

const (
	humanSessionCookieName = "openase_session"
	oidcFlowCookieName     = "openase_oidc_flow"
	oidcFlowCookiePath     = "/api/v1/auth/oidc/callback"
	csrfHeaderName         = "X-OpenASE-CSRF"
)

type humanPrincipalContextKey struct{}

func currentHumanPrincipal(c echo.Context) (humanauthdomain.AuthenticatedPrincipal, bool) {
	value := c.Get(humanPrincipalContextKey{}.String())
	principal, ok := value.(humanauthdomain.AuthenticatedPrincipal)
	return principal, ok
}

func setHumanPrincipal(c echo.Context, principal humanauthdomain.AuthenticatedPrincipal) {
	c.Set(humanPrincipalContextKey{}.String(), principal)
}

func actorFromHumanPrincipal(c echo.Context) string {
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return ""
	}
	return principal.ActorID()
}

func optionalActor(raw *string, fallback string) *string {
	if raw != nil {
		trimmed := strings.TrimSpace(*raw)
		if trimmed != "" {
			return &trimmed
		}
	}
	if strings.TrimSpace(fallback) == "" {
		return nil
	}
	value := strings.TrimSpace(fallback)
	return &value
}

func (humanPrincipalContextKey) String() string {
	return "openase.human_principal"
}
