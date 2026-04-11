package httpapi

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/labstack/echo/v4"
)

// Shared platform resources must keep the same suffix as the human control
// plane. This guard enforces canonical resource naming without merging the two
// auth entrypoints: human-only routes can stay human-only, and explicit
// platform-only runtime helpers stay in the exclusion list below.
var platformRouteAlignmentExclusions = map[routeSignature]string{
	{
		method: http.MethodPost,
		suffix: "/tickets/:ticketId/usage",
	}: "platform-only runtime usage reporting helper; no human control-plane parity required",
}

type routeSignature struct {
	method string
	suffix string
}

func TestPlatformRoutesShareCanonicalResourceSuffixes(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	server := &Server{
		echo:          echoServer,
		agentPlatform: &agentplatform.Service{},
		catalog: catalogservice.Services{
			OrganizationService: authorizationRouteCoverageOrganizationService{},
		},
	}

	api := echoServer.Group("/api/v1")
	public := api.Group("")
	protected := api.Group("")
	registrar := routeRegistrar{server: server, api: api}
	registrar.registerPublicAPIRoutes(public)
	registrar.registerProtectedAPIRoutes(protected)

	humanRoutes := make(map[routeSignature]struct{})
	platformRoutes := make(map[routeSignature]struct{})
	usedExclusions := make(map[routeSignature]struct{})

	for _, route := range echoServer.Routes() {
		if route.Method == http.MethodHead || route.Method == http.MethodOptions {
			continue
		}
		switch route.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			continue
		}
		switch {
		case strings.HasPrefix(route.Path, "/api/v1/platform/"):
			signature := routeSignature{
				method: route.Method,
				suffix: strings.TrimPrefix(route.Path, "/api/v1/platform"),
			}
			platformRoutes[signature] = struct{}{}
		case strings.HasPrefix(route.Path, "/api/v1/"):
			signature := routeSignature{
				method: route.Method,
				suffix: strings.TrimPrefix(route.Path, "/api/v1"),
			}
			humanRoutes[signature] = struct{}{}
		}
	}

	var missing []string
	for signature := range platformRoutes {
		if _, excluded := platformRouteAlignmentExclusions[signature]; excluded {
			usedExclusions[signature] = struct{}{}
			continue
		}
		if _, ok := humanRoutes[signature]; ok {
			continue
		}
		missing = append(
			missing,
			signature.method+" /api/v1/platform"+signature.suffix+" -> expected "+signature.method+" /api/v1"+signature.suffix,
		)
	}

	var staleExclusions []string
	for signature, rationale := range platformRouteAlignmentExclusions {
		if _, ok := usedExclusions[signature]; ok {
			continue
		}
		staleExclusions = append(
			staleExclusions,
			signature.method+" /api/v1/platform"+signature.suffix+" ("+rationale+")",
		)
	}

	slices.Sort(missing)
	slices.Sort(staleExclusions)
	if len(missing) > 0 || len(staleExclusions) > 0 {
		if len(missing) > 0 {
			t.Fatalf(
				"platform routes must keep canonical human suffixes for shared resources:\n%s",
				strings.Join(missing, "\n"),
			)
		}
		t.Fatalf(
			"platform route alignment exclusions are stale:\n%s",
			strings.Join(staleExclusions, "\n"),
		)
	}
}
