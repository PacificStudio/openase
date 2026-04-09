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
	public := r.api.Group("")
	protected := r.api.Group("", r.server.requireHumanSession, r.server.authorizeHumanAPI)

	r.registerPublicAPIRoutes(public)
	r.registerProtectedAPIRoutes(protected)
}

func (r routeRegistrar) registerPublicAPIRoutes(public *echo.Group) {
	public.GET("/openapi.json", r.server.handleOpenAPI)
	public.GET("/system/dashboard", r.server.handleSystemDashboard)
	public.GET("/system/metrics", r.server.handleMetrics)
	public.GET("/events/stream", r.server.handleEventStream)
	if r.server.machineChannel != nil && r.server.machineSessions != nil {
		public.GET("/machines/connect", r.server.handleMachineConnect)
	}
	r.server.registerAuthRoutes(public)

	if r.server.agentPlatform != nil {
		r.server.registerAgentPlatformRoutes(public.Group("/platform", r.server.authenticateAgentToken))
	}
}

func (r routeRegistrar) registerProtectedAPIRoutes(protected *echo.Group) {
	if !r.server.catalog.Empty() {
		r.server.registerOrganizationRoutes(protected)
		r.server.registerOrganizationMembershipRoutes(protected)
		r.server.registerProjectRoutes(protected)
		r.server.registerProjectUpdateRoutes(protected)
		r.server.registerMachineRoutes(protected)
		r.server.registerProjectRepoRoutes(protected)
		r.server.registerCatalogAgentRoutes(protected)
		r.server.registerCatalogActivityRoutes(protected)
		r.server.registerAppContextRoutes(protected)
		r.server.registerWorkspaceSummaryRoutes(protected)
	}
	protected.GET("/orgs/:orgId/machines/stream", r.server.handleMachineStream)
	protected.GET("/orgs/:orgId/providers/stream", r.server.handleProviderStream)
	protected.GET("/projects/:projectId/events/stream", r.server.handleProjectEventStream)
	protected.GET("/projects/:projectId/agents/:agentId/output/stream", r.server.streamAgentOutput)
	protected.GET("/projects/:projectId/agents/:agentId/steps/stream", r.server.streamAgentSteps)
	r.server.registerProtectedAuthRoutes(protected)
	r.server.registerTicketRoutes(protected)
	r.server.registerChatRoutes(protected)
	r.server.registerWorkflowRoutes(protected)
	r.server.registerScheduledJobRoutes(protected)
	r.server.registerNotificationRoutes(protected)
	r.server.registerAdminAuthRoutes(protected)
	r.server.registerSecuritySettingsRoutes(protected)
	r.server.registerOrgSecurityRoutes(protected)
	r.server.registerGitHubRepoRoutes(protected)
	r.server.registerSkillRoutes(protected)
	r.server.registerRoleLibraryRoutes(protected)
	r.server.registerHRAdvisorRoutes(protected)
	r.server.registerTicketStatusRoutes(protected)
	r.server.registerRoleBindingRoutes(protected)
	r.server.registerUserDirectoryRoutes(protected)
}

func (r routeRegistrar) registerUIRoutes() {
	uiHandler := echo.WrapHandler(webui.Handler())
	r.server.echo.GET("/", uiHandler)
	r.server.echo.GET("/*", uiHandler)
}
