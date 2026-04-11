package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerProjectRepoRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/repos", s.listProjectRepos)
	api.POST("/projects/:projectId/repos", s.createProjectRepo)
	api.PATCH("/projects/:projectId/repos/:repoId", s.patchProjectRepo)
	api.DELETE("/projects/:projectId/repos/:repoId", s.deleteProjectRepo)
	api.GET("/projects/:projectId/tickets/:ticketId/repo-scopes", s.listTicketRepoScopes)
	api.POST("/projects/:projectId/tickets/:ticketId/repo-scopes", s.createTicketRepoScope)
	api.PATCH("/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId", s.patchTicketRepoScope)
	api.DELETE("/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId", s.deleteTicketRepoScope)
}

func (s *Server) listProjectRepos(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListProjectRepos(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repos": mapProjectRepoResponses(items),
	})
}

func (s *Server) createProjectRepo(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	var request domain.ProjectRepoInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateProjectRepo(projectID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateProjectRepo(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		EventType: activityevent.TypeProjectRepoCreated,
		Message:   "Added project repo " + item.Name,
		Metadata: map[string]any{
			"repo_id":           item.ID.String(),
			"repo_name":         item.Name,
			"repository_url":    item.RepositoryURL,
			"default_branch":    item.DefaultBranch,
			"workspace_dirname": item.WorkspaceDirname,
			"labels":            append([]string(nil), item.Labels...),
			"changed_fields":    []string{"repo"},
		},
	}); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"repo": mapProjectRepoResponse(item),
	})
}

func (s *Server) patchProjectRepo(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	repoID, err := parseUUIDPathParam(c, "repoId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetProjectRepo(c.Request().Context(), projectID, repoID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch projectRepoPatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	input, err := parseProjectRepoPatchRequest(projectID, repoID, current, patch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateProjectRepo(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	changedFields := make([]string, 0, 5)
	if current.Name != item.Name {
		changedFields = append(changedFields, "name")
	}
	if current.RepositoryURL != item.RepositoryURL {
		changedFields = append(changedFields, "repository_url")
	}
	if current.DefaultBranch != item.DefaultBranch {
		changedFields = append(changedFields, "default_branch")
	}
	if current.WorkspaceDirname != item.WorkspaceDirname {
		changedFields = append(changedFields, "workspace_dirname")
	}
	if !slices.Equal(current.Labels, item.Labels) {
		changedFields = append(changedFields, "labels")
	}
	if len(changedFields) > 0 {
		if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			EventType: activityevent.TypeProjectRepoUpdated,
			Message:   "Updated project repo " + item.Name,
			Metadata: map[string]any{
				"repo_id":           item.ID.String(),
				"repo_name":         item.Name,
				"repository_url":    item.RepositoryURL,
				"default_branch":    item.DefaultBranch,
				"workspace_dirname": item.WorkspaceDirname,
				"labels":            append([]string(nil), item.Labels...),
				"changed_fields":    changedFields,
			},
		}); err != nil {
			return writeCatalogError(c, err)
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo": mapProjectRepoResponse(item),
	})
}

func (s *Server) deleteProjectRepo(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	repoID, err := parseUUIDPathParam(c, "repoId")
	if err != nil {
		return err
	}

	item, err := s.catalog.DeleteProjectRepo(c.Request().Context(), projectID, repoID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		EventType: activityevent.TypeProjectRepoDeleted,
		Message:   "Removed project repo " + item.Name,
		Metadata: map[string]any{
			"repo_id":           item.ID.String(),
			"repo_name":         item.Name,
			"repository_url":    item.RepositoryURL,
			"default_branch":    item.DefaultBranch,
			"workspace_dirname": item.WorkspaceDirname,
			"labels":            append([]string(nil), item.Labels...),
			"changed_fields":    []string{"repo"},
		},
	}); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo": mapProjectRepoResponse(item),
	})
}

func (s *Server) listTicketRepoScopes(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListTicketRepoScopes(c.Request().Context(), projectID, ticketID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo_scopes": mapTicketRepoScopeResponses(items),
	})
}

func (s *Server) createTicketRepoScope(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}

	var request domain.TicketRepoScopeInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateTicketRepoScope(projectID, ticketID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateTicketRepoScope(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"repo_scope": mapTicketRepoScopeResponse(item),
	})
}

func (s *Server) patchTicketRepoScope(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}
	scopeID, err := parseUUIDPathParam(c, "scopeId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetTicketRepoScope(c.Request().Context(), projectID, ticketID, scopeID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch ticketRepoScopePatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	input, err := parseTicketRepoScopePatchRequest(scopeID, projectID, ticketID, current, patch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateTicketRepoScope(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitPullRequestActivity(c.Request().Context(), projectID, ticketID, current, item); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo_scope": mapTicketRepoScopeResponse(item),
	})
}

func (s *Server) deleteTicketRepoScope(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}
	scopeID, err := parseUUIDPathParam(c, "scopeId")
	if err != nil {
		return err
	}

	item, err := s.catalog.DeleteTicketRepoScope(c.Request().Context(), projectID, ticketID, scopeID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitPullRequestActivity(
		c.Request().Context(),
		projectID,
		ticketID,
		item,
		domain.TicketRepoScope{ID: item.ID, TicketID: item.TicketID, RepoID: item.RepoID, BranchName: item.BranchName},
	); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo_scope": mapTicketRepoScopeResponse(item),
	})
}

func (s *Server) emitPullRequestActivity(
	ctx context.Context,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	current domain.TicketRepoScope,
	updated domain.TicketRepoScope,
) error {
	if s.ticketService == nil {
		return nil
	}

	beforeURL := strings.TrimSpace(optionalStringValue(current.PullRequestURL))
	afterURL := strings.TrimSpace(optionalStringValue(updated.PullRequestURL))
	if beforeURL == afterURL {
		return nil
	}

	ticketItem, err := s.ticketService.Get(ctx, ticketID)
	if err != nil {
		return err
	}

	var eventType activityevent.Type
	var message string
	switch {
	case beforeURL == "" && afterURL != "":
		eventType = activityevent.TypePROpened
		message = fmt.Sprintf("%s PR opened", ticketItem.Identifier)
	case beforeURL != "" && afterURL == "":
		eventType = activityevent.TypePRClosed
		message = fmt.Sprintf("%s PR closed", ticketItem.Identifier)
	default:
		return nil
	}

	return s.emitActivity(ctx, activitysvc.RecordInput{
		ProjectID: projectID,
		TicketID:  &ticketID,
		EventType: eventType,
		Message:   message,
		Metadata: map[string]any{
			"ticket_identifier": ticketItem.Identifier,
			"repo_scope_id":     updated.ID.String(),
			"repo_id":           updated.RepoID.String(),
			"branch_name":       updated.BranchName,
			"pull_request_url":  afterURL,
			"previous_pr_url":   beforeURL,
			"changed_fields":    []string{"pull_request_url"},
		},
	})
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
