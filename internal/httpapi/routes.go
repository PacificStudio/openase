package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/BetterAndBetterII/openase/internal/webui"
	"github.com/labstack/echo/v4"
)

type routeRegistrar struct {
	server *Server
	api    *echo.Group
}

func registerServerRoutes(server *Server) {
	api := server.echo.Group("/api/v1")
	registrar := routeRegistrar{
		server: server,
		api:    api,
	}
	registrar.registerHealthRoutes()
	registrar.registerAPIRoutes()
	registrar.registerUIRoutes()
}

func (r routeRegistrar) registerHealthRoutes() {
	healthHandler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"service": "openase",
			"status":  "ok",
			"time":    time.Now().UTC().Format(time.RFC3339),
			"port":    strconv.Itoa(r.server.cfg.Port),
		})
	}

	r.server.echo.GET("/healthz", healthHandler)
	r.api.GET("/healthz", healthHandler)
}

func (r routeRegistrar) registerAPIRoutes() {
	r.api.GET("/openapi.json", r.server.handleOpenAPI)
	r.api.GET("/system/dashboard", r.server.handleSystemDashboard)
	r.api.GET("/system/metrics", r.server.handleMetrics)
	r.api.GET("/events/stream", r.server.handleEventStream)
	r.api.GET("/orgs/:orgId/machines/stream", r.server.handleMachineStream)
	r.api.GET("/orgs/:orgId/providers/stream", r.server.handleProviderStream)
	r.api.GET("/projects/:projectId/events/stream", r.server.handleProjectEventStream)
	r.api.GET("/projects/:projectId/agents/:agentId/output/stream", r.server.streamAgentOutput)
	r.api.GET("/projects/:projectId/agents/:agentId/steps/stream", r.server.streamAgentSteps)

	if r.server.agentPlatform != nil {
		r.server.registerAgentPlatformRoutes(r.api.Group("/platform", r.server.authenticateAgentToken))
	}
	if !r.server.catalog.Empty() {
		r.server.registerOrganizationRoutes(r.api)
		r.server.registerProjectRoutes(r.api)
		r.server.registerProjectUpdateRoutes(r.api)
		r.server.registerMachineRoutes(r.api)
		r.server.registerProjectRepoRoutes(r.api)
		r.server.registerCatalogAgentRoutes(r.api)
		r.server.registerCatalogActivityRoutes(r.api)
		r.server.registerAppContextRoutes(r.api)
		r.server.registerWorkspaceSummaryRoutes(r.api)
	}
	r.server.registerTicketRoutes(r.api)
	r.server.registerChatRoutes(r.api)
	r.server.registerWorkflowRoutes(r.api)
	r.server.registerScheduledJobRoutes(r.api)
	r.server.registerNotificationRoutes(r.api)
	r.server.registerSecuritySettingsRoutes(r.api)
	r.server.registerGitHubRepoRoutes(r.api)
	r.server.registerSkillRoutes(r.api)
	r.server.registerRoleLibraryRoutes(r.api)
	r.server.registerHRAdvisorRoutes(r.api)
	r.server.registerTicketStatusRoutes(r.api)
}

func (r routeRegistrar) registerUIRoutes() {
	uiHandler := echo.WrapHandler(webui.Handler())
	r.server.echo.GET("/", uiHandler)
	r.server.echo.GET("/*", uiHandler)
}
