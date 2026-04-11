package httpapi

import (
	"net/http"
	"slices"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerProjectRoutes(api *echo.Group) {
	api.GET("/orgs/:orgId/summary", s.handleGetOrganizationSummary)
	api.GET("/orgs/:orgId/projects", s.listProjects)
	api.POST("/orgs/:orgId/projects", s.createProject)
	api.GET("/projects/:projectId", s.getProject)
	api.PATCH("/projects/:projectId", s.patchProject)
	api.DELETE("/projects/:projectId", s.archiveProject)
}

func (s *Server) listProjects(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListProjects(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"projects": mapProjectResponses(items),
	})
}

func (s *Server) createProject(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	var request domain.ProjectInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateProject(orgID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateProject(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ID,
		EventType: activityevent.TypeProjectCreated,
		Message:   "Created project " + item.Name,
		Metadata: map[string]any{
			"project_name":           item.Name,
			"status":                 item.Status.String(),
			"default_provider_id":    uuidToStringPointer(item.DefaultAgentProviderID),
			"accessible_machine_ids": uuidSliceToStrings(item.AccessibleMachineIDs),
			"max_concurrent_agents":  item.MaxConcurrentAgents,
			"changed_fields":         []string{"project"},
		},
	}); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) getProject(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) patchProject(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch projectPatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	input, err := parseProjectPatchRequest(projectID, current, patch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateProject(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	activityInputs := buildProjectPatchActivityInputs(current, item)
	if err := s.emitActivities(c.Request().Context(), activityInputs...); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func buildProjectPatchActivityInputs(current domain.Project, item domain.Project) []activitysvc.RecordInput {
	changedFields := make([]string, 0, 4)
	if current.Name != item.Name || current.Slug != item.Slug || current.Description != item.Description ||
		!slices.Equal(current.AccessibleMachineIDs, item.AccessibleMachineIDs) ||
		!slices.Equal(current.ProjectAIPlatformAccessAllowed, item.ProjectAIPlatformAccessAllowed) ||
		current.AgentRunSummaryPrompt != item.AgentRunSummaryPrompt {
		changedFields = append(changedFields, "project")
	}

	activityInputs := make([]activitysvc.RecordInput, 0, 4)
	if current.Status != item.Status {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ID,
			EventType: activityevent.TypeProjectStatusChanged,
			Message:   "Changed project status to " + item.Status.String(),
			Metadata:  projectStatusMetadata(current, item),
		})
	}
	if !uuidPointersEqual(current.DefaultAgentProviderID, item.DefaultAgentProviderID) {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ID,
			EventType: activityevent.TypeProjectProviderChanged,
			Message:   "Changed project default provider for " + item.Name,
			Metadata: map[string]any{
				"project_name":     item.Name,
				"from_provider_id": uuidToStringPointer(current.DefaultAgentProviderID),
				"to_provider_id":   uuidToStringPointer(item.DefaultAgentProviderID),
				"changed_fields":   []string{"default_agent_provider_id"},
			},
		})
	}
	if current.MaxConcurrentAgents != item.MaxConcurrentAgents {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ID,
			EventType: activityevent.TypeProjectConcurrencyChanged,
			Message:   "Changed project concurrency for " + item.Name,
			Metadata: map[string]any{
				"project_name":         item.Name,
				"from_max_concurrency": current.MaxConcurrentAgents,
				"to_max_concurrency":   item.MaxConcurrentAgents,
				"changed_fields":       []string{"max_concurrent_agents"},
			},
		})
	}
	if current.ProjectAIRetention != item.ProjectAIRetention {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ID,
			EventType: activityevent.TypeProjectAIRetentionUpdated,
			Message:   "Updated Project AI retention for " + item.Name,
			Metadata: map[string]any{
				"project_name": item.Name,
				"from": map[string]any{
					"enabled":          current.ProjectAIRetention.Enabled,
					"keep_latest_n":    current.ProjectAIRetention.KeepLatestN,
					"keep_recent_days": current.ProjectAIRetention.KeepRecentDays,
				},
				"to": map[string]any{
					"enabled":          item.ProjectAIRetention.Enabled,
					"keep_latest_n":    item.ProjectAIRetention.KeepLatestN,
					"keep_recent_days": item.ProjectAIRetention.KeepRecentDays,
				},
				"changed_fields": []string{"project_ai_retention"},
			},
		})
	}
	if len(changedFields) > 0 {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ID,
			EventType: activityevent.TypeProjectUpdated,
			Message:   "Updated project " + item.Name,
			Metadata: map[string]any{
				"project_name":           item.Name,
				"accessible_machine_ids": uuidSliceToStrings(item.AccessibleMachineIDs),
				"changed_fields":         changedFields,
			},
		})
	}

	return activityInputs
}

func (s *Server) archiveProject(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	item, err := s.catalog.ArchiveProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ID,
		EventType: activityevent.TypeProjectArchived,
		Message:   "Archived project " + item.Name,
		Metadata: map[string]any{
			"project_name":   item.Name,
			"status":         item.Status.String(),
			"changed_fields": []string{"archived"},
		},
	}); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}
