package httpapi

import (
	"context"
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
type writeActorContextKey struct{}

func currentHumanPrincipal(c echo.Context) (humanauthdomain.AuthenticatedPrincipal, bool) {
	value := c.Get(humanPrincipalContextKey{}.String())
	principal, ok := value.(humanauthdomain.AuthenticatedPrincipal)
	return principal, ok
}

func setHumanPrincipal(c echo.Context, principal humanauthdomain.AuthenticatedPrincipal) {
	c.Set(humanPrincipalContextKey{}.String(), principal)
	c.SetRequest(c.Request().WithContext(humanauthdomain.WithPrincipal(c.Request().Context(), principal)))
}

func actorFromHumanPrincipal(c echo.Context) string {
	principal, ok := currentHumanPrincipal(c)
	if !ok {
		return ""
	}
	return principal.ActorID()
}

func actorFromWritePrincipal(c echo.Context) string {
	if actor, ok := c.Request().Context().Value(writeActorContextKey{}).(string); ok {
		return strings.TrimSpace(actor)
	}
	return strings.TrimSpace(actorFromHumanPrincipal(c))
}

func withWriteActor(ctx context.Context, actor string) context.Context {
	trimmed := strings.TrimSpace(actor)
	if trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, writeActorContextKey{}, trimmed)
}

func (humanPrincipalContextKey) String() string {
	return "openase.human_principal"
}
