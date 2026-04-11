package httpapi

import (
	"net/http"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerExpandedAgentPlatformRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/activity", s.handleAgentListActivityEvents)
	api.POST("/agents/:agentId/interrupt", s.handleAgentInterruptProjectAgent)
	api.GET("/projects/:projectId/statuses", s.handleAgentListTicketStatuses)
	api.POST("/projects/:projectId/statuses", s.handleAgentCreateTicketStatus)
	api.POST("/projects/:projectId/statuses/reset", s.handleAgentResetTicketStatuses)
	api.PATCH("/statuses/:statusId", s.handleAgentUpdateTicketStatus)
	api.DELETE("/statuses/:statusId", s.handleAgentDeleteTicketStatus)
	api.POST("/projects/:projectId/workflows", s.handleAgentCreateWorkflow)
	api.GET("/workflows/:workflowId", s.handleAgentGetWorkflow)
	api.PATCH("/workflows/:workflowId", s.handleAgentUpdateWorkflow)
	api.DELETE("/workflows/:workflowId", s.handleAgentDeleteWorkflow)
	api.GET("/workflows/:workflowId/harness", s.handleAgentGetWorkflowHarness)
	api.GET("/workflows/:workflowId/harness/history", s.handleAgentGetWorkflowHarnessHistory)
	api.PUT("/workflows/:workflowId/harness", s.handleAgentUpdateWorkflowHarness)
	api.GET("/harness/variables", s.handleAgentListHarnessVariables)
	api.POST("/harness/validate", s.handleAgentValidateHarness)
	api.GET("/projects/:projectId/repos", s.handleAgentListProjectRepos)
	api.PATCH("/projects/:projectId/repos/:repoId", s.handleAgentPatchProjectRepo)
	api.DELETE("/projects/:projectId/repos/:repoId", s.handleAgentDeleteProjectRepo)
	api.GET("/projects/:projectId/tickets/:ticketId/repo-scopes", s.handleAgentListTicketRepoScopes)
	api.POST("/projects/:projectId/tickets/:ticketId/repo-scopes", s.handleAgentCreateTicketRepoScope)
	api.PATCH("/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId", s.handleAgentPatchTicketRepoScope)
	api.DELETE("/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId", s.handleAgentDeleteTicketRepoScope)
	api.GET("/projects/:projectId/scheduled-jobs", s.handleAgentListScheduledJobs)
	api.POST("/projects/:projectId/scheduled-jobs", s.handleAgentCreateScheduledJob)
	api.PATCH("/scheduled-jobs/:jobId", s.handleAgentUpdateScheduledJob)
	api.DELETE("/scheduled-jobs/:jobId", s.handleAgentDeleteScheduledJob)
	api.POST("/scheduled-jobs/:jobId/trigger", s.handleAgentTriggerScheduledJob)
	api.GET("/projects/:projectId/skills", s.handleAgentListSkills)
	api.POST("/projects/:projectId/skills", s.handleAgentCreateSkill)
	api.POST("/projects/:projectId/skills/import", s.handleAgentImportSkillBundle)
	api.POST("/projects/:projectId/skills/refresh", s.handleAgentRefreshSkills)
	api.GET("/skills/:skillId", s.handleAgentGetSkill)
	api.GET("/skills/:skillId/files", s.handleAgentGetSkillFiles)
	api.GET("/skills/:skillId/history", s.handleAgentGetSkillHistory)
	api.PUT("/skills/:skillId", s.handleAgentUpdateSkill)
	api.DELETE("/skills/:skillId", s.handleAgentDeleteSkill)
	api.POST("/skills/:skillId/enable", s.handleAgentEnableSkill)
	api.POST("/skills/:skillId/disable", s.handleAgentDisableSkill)
	api.POST("/skills/:skillId/bind", s.handleAgentBindSkill)
	api.POST("/skills/:skillId/unbind", s.handleAgentUnbindSkill)
	api.POST("/workflows/:workflowId/skills/bind", s.handleAgentBindWorkflowSkills)
	api.POST("/workflows/:workflowId/skills/unbind", s.handleAgentUnbindWorkflowSkills)
}

func requireAgentProjectAnyScope(c echo.Context, scopes ...agentplatform.Scope) bool {
	claims, ok := requireAgentAnyScope(c, scopes...)
	if !ok {
		return false
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
		return false
	}
	if claims.ProjectID != projectID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
		return false
	}
	return true
}

func (s *Server) requireAgentProjectAgentAnyScope(c echo.Context, scopes ...agentplatform.Scope) (domain.Agent, bool) {
	if s.catalog.AgentService == nil {
		_ = writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
		return domain.Agent{}, false
	}

	claims, ok := requireAgentAnyScope(c, scopes...)
	if !ok {
		return domain.Agent{}, false
	}

	agentID, err := parseUUIDPathParamValue(c, "agentId")
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_AGENT_ID", err.Error())
		return domain.Agent{}, false
	}
	item, err := s.catalog.GetAgent(c.Request().Context(), agentID)
	if err != nil {
		_ = writeCatalogError(c, err)
		return domain.Agent{}, false
	}
	if item.ProjectID != claims.ProjectID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
		return domain.Agent{}, false
	}
	return item, true
}

func (s *Server) handleAgentInterruptProjectAgent(c echo.Context) error {
	if _, ok := s.requireAgentProjectAgentAnyScope(c, agentplatform.ScopeAgentsInterrupt); !ok {
		return nil
	}
	return s.interruptAgent(c)
}

func (s *Server) requireAgentWorkflowAnyScope(c echo.Context, scopes ...agentplatform.Scope) bool {
	if s.workflowService == nil {
		_ = writeWorkflowError(c, workflowservice.ErrUnavailable)
		return false
	}

	claims, ok := requireAgentAnyScope(c, scopes...)
	if !ok {
		return false
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
		return false
	}
	item, err := s.workflowService.Get(c.Request().Context(), workflowID)
	if err != nil {
		_ = writeWorkflowError(c, err)
		return false
	}
	if item.ProjectID != claims.ProjectID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
		return false
	}
	return true
}

func (s *Server) requireAgentSkillAnyScope(c echo.Context, scopes ...agentplatform.Scope) (agentplatform.Claims, bool) {
	if s.workflowService == nil {
		_ = writeWorkflowError(c, workflowservice.ErrUnavailable)
		return agentplatform.Claims{}, false
	}

	claims, ok := requireAgentAnyScope(c, scopes...)
	if !ok {
		return agentplatform.Claims{}, false
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
		return agentplatform.Claims{}, false
	}
	_, err = s.workflowService.GetSkillInProject(c.Request().Context(), claims.ProjectID, skillID)
	if err != nil {
		_ = writeWorkflowError(c, err)
		return agentplatform.Claims{}, false
	}
	return claims, true
}

func (s *Server) requireAgentScheduledJobAnyScope(c echo.Context, scopes ...agentplatform.Scope) bool {
	if s.scheduledJobService == nil {
		_ = writeScheduledJobError(c, scheduledjobservice.ErrUnavailable)
		return false
	}

	claims, ok := requireAgentAnyScope(c, scopes...)
	if !ok {
		return false
	}

	jobID, err := parseUUIDPathParamValue(c, "jobId")
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_JOB_ID", err.Error())
		return false
	}
	item, err := s.scheduledJobService.Get(c.Request().Context(), jobID)
	if err != nil {
		_ = writeScheduledJobError(c, err)
		return false
	}
	if item.ProjectID != claims.ProjectID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
		return false
	}
	return true
}

func (s *Server) requireAgentStatusAnyScope(c echo.Context, scopes ...agentplatform.Scope) bool {
	if s.ticketStatusService == nil {
		_ = writeTicketStatusError(c, ticketstatus.ErrUnavailable)
		return false
	}

	claims, ok := requireAgentAnyScope(c, scopes...)
	if !ok {
		return false
	}

	statusID, err := parseStatusID(c)
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_STATUS_ID", err.Error())
		return false
	}
	item, err := s.ticketStatusService.Get(c.Request().Context(), statusID)
	if err != nil {
		_ = writeTicketStatusError(c, err)
		return false
	}
	if item.ProjectID != claims.ProjectID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
		return false
	}
	return true
}

func (s *Server) requireAgentOwnTicketInProject(c echo.Context, scope agentplatform.Scope) bool {
	claims, ok := requireAgentScope(c, scope)
	if !ok {
		return false
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
		return false
	}
	current, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		_ = writeTicketError(c, err)
		return false
	}
	if claims.IsTicketAgent() && claims.TicketID != ticketID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_TICKET_FORBIDDEN", "agent token can only access its current ticket")
		return false
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
		return false
	}
	if current.ProjectID != projectID || claims.ProjectID != projectID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
		return false
	}
	return true
}

func (s *Server) handleAgentListActivityEvents(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeActivityRead) {
		return nil
	}
	return s.listActivityEvents(c)
}

func (s *Server) handleAgentListTicketStatuses(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeStatusesList) {
		return nil
	}
	return s.handleListTicketStatuses(c)
}

func (s *Server) handleAgentCreateTicketStatus(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeStatusesCreate) {
		return nil
	}
	return s.handleCreateTicketStatus(c)
}

func (s *Server) handleAgentResetTicketStatuses(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeStatusesReset) {
		return nil
	}
	return s.handleResetTicketStatuses(c)
}

func (s *Server) handleAgentUpdateTicketStatus(c echo.Context) error {
	if !s.requireAgentStatusAnyScope(c, agentplatform.ScopeStatusesUpdate) {
		return nil
	}
	return s.handleUpdateTicketStatus(c)
}

func (s *Server) handleAgentDeleteTicketStatus(c echo.Context) error {
	if !s.requireAgentStatusAnyScope(c, agentplatform.ScopeStatusesDelete) {
		return nil
	}
	return s.handleDeleteTicketStatus(c)
}

func (s *Server) handleAgentCreateWorkflow(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeWorkflowsCreate) {
		return nil
	}
	return s.handleCreateWorkflow(c)
}

func (s *Server) handleAgentGetWorkflow(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeWorkflowsRead) {
		return nil
	}
	return s.handleGetWorkflow(c)
}

func (s *Server) handleAgentUpdateWorkflow(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeWorkflowsUpdate) {
		return nil
	}
	return s.handleUpdateWorkflow(c)
}

func (s *Server) handleAgentDeleteWorkflow(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeWorkflowsDelete) {
		return nil
	}
	return s.handleDeleteWorkflow(c)
}

func (s *Server) handleAgentGetWorkflowHarness(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeWorkflowsHarnessRead) {
		return nil
	}
	return s.handleGetWorkflowHarness(c)
}

func (s *Server) handleAgentGetWorkflowHarnessHistory(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeWorkflowsHarnessHistoryRead) {
		return nil
	}
	return s.handleGetWorkflowHarnessHistory(c)
}

func (s *Server) handleAgentUpdateWorkflowHarness(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeWorkflowsHarnessUpdate) {
		return nil
	}
	return s.handleUpdateWorkflowHarness(c)
}

func (s *Server) handleAgentListHarnessVariables(c echo.Context) error {
	if _, ok := requireAgentAnyScope(c, agentplatform.ScopeWorkflowsHarnessVariablesRead); !ok {
		return nil
	}
	return s.handleListHarnessVariables(c)
}

func (s *Server) handleAgentValidateHarness(c echo.Context) error {
	if _, ok := requireAgentAnyScope(c, agentplatform.ScopeWorkflowsHarnessValidate); !ok {
		return nil
	}
	return s.handleValidateHarness(c)
}

func (s *Server) handleAgentListProjectRepos(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeReposRead) {
		return nil
	}
	return s.listProjectRepos(c)
}

func (s *Server) handleAgentPatchProjectRepo(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeReposUpdate) {
		return nil
	}
	return s.patchProjectRepo(c)
}

func (s *Server) handleAgentDeleteProjectRepo(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeReposDelete) {
		return nil
	}
	return s.deleteProjectRepo(c)
}

func (s *Server) handleAgentListTicketRepoScopes(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !s.requireAgentOwnTicketInProject(c, agentplatform.ScopeTicketRepoScopesList) {
		return nil
	}
	return s.listTicketRepoScopes(c)
}

func (s *Server) handleAgentCreateTicketRepoScope(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !s.requireAgentOwnTicketInProject(c, agentplatform.ScopeTicketRepoScopesCreate) {
		return nil
	}
	return s.createTicketRepoScope(c)
}

func (s *Server) handleAgentPatchTicketRepoScope(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !s.requireAgentOwnTicketInProject(c, agentplatform.ScopeTicketRepoScopesUpdate) {
		return nil
	}
	return s.patchTicketRepoScope(c)
}

func (s *Server) handleAgentDeleteTicketRepoScope(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	if !s.requireAgentOwnTicketInProject(c, agentplatform.ScopeTicketRepoScopesDelete) {
		return nil
	}
	return s.deleteTicketRepoScope(c)
}

func (s *Server) handleAgentListScheduledJobs(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeScheduledJobsList) {
		return nil
	}
	return s.handleListScheduledJobs(c)
}

func (s *Server) handleAgentCreateScheduledJob(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeScheduledJobsCreate) {
		return nil
	}
	return s.handleCreateScheduledJob(c)
}

func (s *Server) handleAgentUpdateScheduledJob(c echo.Context) error {
	if !s.requireAgentScheduledJobAnyScope(c, agentplatform.ScopeScheduledJobsUpdate) {
		return nil
	}
	return s.handleUpdateScheduledJob(c)
}

func (s *Server) handleAgentDeleteScheduledJob(c echo.Context) error {
	if !s.requireAgentScheduledJobAnyScope(c, agentplatform.ScopeScheduledJobsDelete) {
		return nil
	}
	return s.handleDeleteScheduledJob(c)
}

func (s *Server) handleAgentTriggerScheduledJob(c echo.Context) error {
	if !s.requireAgentScheduledJobAnyScope(c, agentplatform.ScopeScheduledJobsTrigger) {
		return nil
	}
	return s.handleTriggerScheduledJob(c)
}

func (s *Server) handleAgentListSkills(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeSkillsList) {
		return nil
	}
	return s.handleListSkills(c)
}

func (s *Server) handleAgentCreateSkill(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeSkillsCreate) {
		return nil
	}
	return s.handleCreateSkill(c)
}

func (s *Server) handleAgentImportSkillBundle(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeSkillsImport) {
		return nil
	}
	return s.handleImportSkillBundle(c)
}

func (s *Server) handleAgentRefreshSkills(c echo.Context) error {
	if !requireAgentProjectAnyScope(c, agentplatform.ScopeSkillsRefresh) {
		return nil
	}
	return s.handleRefreshSkills(c)
}

func (s *Server) handleAgentGetSkill(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsRead); !ok {
		return nil
	}
	return s.handleGetSkill(c)
}

func (s *Server) handleAgentGetSkillFiles(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsRead); !ok {
		return nil
	}
	return s.handleGetSkillFiles(c)
}

func (s *Server) handleAgentGetSkillHistory(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsRead); !ok {
		return nil
	}
	return s.handleGetSkillHistory(c)
}

func (s *Server) handleAgentUpdateSkill(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsUpdate); !ok {
		return nil
	}
	return s.handleUpdateSkill(c)
}

func (s *Server) handleAgentDeleteSkill(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsDelete); !ok {
		return nil
	}
	return s.handleDeleteSkill(c)
}

func (s *Server) handleAgentEnableSkill(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsEnable); !ok {
		return nil
	}
	return s.handleEnableSkill(c)
}

func (s *Server) handleAgentDisableSkill(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsDisable); !ok {
		return nil
	}
	return s.handleDisableSkill(c)
}

func (s *Server) handleAgentBindSkill(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsBind); !ok {
		return nil
	}
	return s.handleBindSkill(c)
}

func (s *Server) handleAgentUnbindSkill(c echo.Context) error {
	if _, ok := s.requireAgentSkillAnyScope(c, agentplatform.ScopeSkillsBind); !ok {
		return nil
	}
	return s.handleUnbindSkill(c)
}

func (s *Server) handleAgentBindWorkflowSkills(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeSkillsBind) {
		return nil
	}
	return s.handleBindWorkflowSkills(c)
}

func (s *Server) handleAgentUnbindWorkflowSkills(c echo.Context) error {
	if !s.requireAgentWorkflowAnyScope(c, agentplatform.ScopeSkillsBind) {
		return nil
	}
	return s.handleUnbindWorkflowSkills(c)
}
