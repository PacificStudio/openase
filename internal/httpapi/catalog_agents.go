package httpapi

import "github.com/labstack/echo/v4"

func (s *Server) registerCatalogAgentRoutes(api *echo.Group) {
	api.GET("/provider-model-options", s.listProviderModelOptions)
	api.GET("/orgs/:orgId/providers", s.listAgentProviders)
	api.POST("/orgs/:orgId/providers", s.createAgentProvider)
	api.GET("/providers/:providerId", s.getAgentProvider)
	api.PATCH("/providers/:providerId", s.patchAgentProvider)
	api.GET("/projects/:projectId/agents", s.listAgents)
	api.GET("/projects/:projectId/agent-runs", s.listAgentRuns)
	api.POST("/projects/:projectId/agents", s.createAgent)
	api.GET("/projects/:projectId/agents/:agentId/output", s.listAgentOutput)
	api.GET("/projects/:projectId/agents/:agentId/steps", s.listAgentSteps)
	api.GET("/agents/:agentId", s.getAgent)
	api.PATCH("/agents/:agentId", s.patchAgent)
	api.POST("/agents/:agentId/pause", s.pauseAgent)
	api.POST("/agents/:agentId/resume", s.resumeAgent)
	api.POST("/agents/:agentId/retire", s.retireAgent)
	api.DELETE("/agents/:agentId", s.deleteAgent)
}
