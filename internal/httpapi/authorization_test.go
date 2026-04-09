package httpapi

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestProtectedRoutesRequireDefinedAuthorizationRules(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	server := &Server{
		echo: echoServer,
		catalog: catalogservice.Services{
			OrganizationService: authorizationRouteCoverageOrganizationService{},
		},
	}

	api := echoServer.Group("/api/v1")
	protected := api.Group("")
	routeRegistrar{server: server, api: api}.registerProtectedAPIRoutes(protected)

	missing := make([]string, 0)
	for _, route := range echoServer.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		if route.Method == http.MethodOptions || route.Method == http.MethodHead {
			continue
		}
		if _, ok := humanRouteAuthorizationRuleFor(route.Path, route.Method); ok {
			continue
		}
		missing = append(missing, route.Method+" "+route.Path)
	}

	slices.Sort(missing)
	if len(missing) > 0 {
		t.Fatalf("protected routes missing authorization rules:\n%s", strings.Join(missing, "\n"))
	}
}

type authorizationRouteCoverageOrganizationService struct{}

func (authorizationRouteCoverageOrganizationService) ListOrganizations(context.Context) ([]catalogdomain.Organization, error) {
	return nil, nil
}

func (authorizationRouteCoverageOrganizationService) CreateOrganization(context.Context, catalogdomain.CreateOrganization) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}

func (authorizationRouteCoverageOrganizationService) GetOrganization(context.Context, uuid.UUID) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}

func (authorizationRouteCoverageOrganizationService) UpdateOrganization(context.Context, catalogdomain.UpdateOrganization) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}

func (authorizationRouteCoverageOrganizationService) ArchiveOrganization(context.Context, uuid.UUID) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}
