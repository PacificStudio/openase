package httpapi

import (
	"net/http"
	"time"

	projectupdatedomain "github.com/BetterAndBetterII/openase/internal/domain/projectupdate"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	"github.com/labstack/echo/v4"
)

type projectUpdateCommentResponse struct {
	ID           string  `json:"id"`
	ThreadID     string  `json:"thread_id"`
	BodyMarkdown string  `json:"body_markdown"`
	CreatedBy    string  `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	EditedAt     *string `json:"edited_at,omitempty"`
	EditCount    int     `json:"edit_count"`
	LastEditedBy *string `json:"last_edited_by,omitempty"`
	IsDeleted    bool    `json:"is_deleted"`
	DeletedAt    *string `json:"deleted_at,omitempty"`
	DeletedBy    *string `json:"deleted_by,omitempty"`
}

type projectUpdateThreadResponse struct {
	ID             string                         `json:"id"`
	ProjectID      string                         `json:"project_id"`
	Status         string                         `json:"status"`
	Title          string                         `json:"title"`
	BodyMarkdown   string                         `json:"body_markdown"`
	CreatedBy      string                         `json:"created_by"`
	CreatedAt      string                         `json:"created_at"`
	UpdatedAt      string                         `json:"updated_at"`
	EditedAt       *string                        `json:"edited_at,omitempty"`
	EditCount      int                            `json:"edit_count"`
	LastEditedBy   *string                        `json:"last_edited_by,omitempty"`
	IsDeleted      bool                           `json:"is_deleted"`
	DeletedAt      *string                        `json:"deleted_at,omitempty"`
	DeletedBy      *string                        `json:"deleted_by,omitempty"`
	LastActivityAt string                         `json:"last_activity_at"`
	CommentCount   int                            `json:"comment_count"`
	Comments       []projectUpdateCommentResponse `json:"comments"`
}

type projectUpdateThreadRevisionResponse struct {
	ID             string  `json:"id"`
	ThreadID       string  `json:"thread_id"`
	RevisionNumber int     `json:"revision_number"`
	Status         string  `json:"status"`
	Title          string  `json:"title"`
	BodyMarkdown   string  `json:"body_markdown"`
	EditedBy       string  `json:"edited_by"`
	EditedAt       string  `json:"edited_at"`
	EditReason     *string `json:"edit_reason,omitempty"`
}

type projectUpdateCommentRevisionResponse struct {
	ID             string  `json:"id"`
	CommentID      string  `json:"comment_id"`
	RevisionNumber int     `json:"revision_number"`
	BodyMarkdown   string  `json:"body_markdown"`
	EditedBy       string  `json:"edited_by"`
	EditedAt       string  `json:"edited_at"`
	EditReason     *string `json:"edit_reason,omitempty"`
}

func (s *Server) registerProjectUpdateRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/updates", s.handleListProjectUpdates)
	api.POST("/projects/:projectId/updates", s.handleCreateProjectUpdateThread)
	api.PATCH("/projects/:projectId/updates/:threadId", s.handleUpdateProjectUpdateThread)
	api.DELETE("/projects/:projectId/updates/:threadId", s.handleDeleteProjectUpdateThread)
	api.GET("/projects/:projectId/updates/:threadId/revisions", s.handleListProjectUpdateThreadRevisions)
	api.POST("/projects/:projectId/updates/:threadId/comments", s.handleCreateProjectUpdateComment)
	api.PATCH("/projects/:projectId/updates/:threadId/comments/:commentId", s.handleUpdateProjectUpdateComment)
	api.DELETE("/projects/:projectId/updates/:threadId/comments/:commentId", s.handleDeleteProjectUpdateComment)
	api.GET("/projects/:projectId/updates/:threadId/comments/:commentId/revisions", s.handleListProjectUpdateCommentRevisions)
}

func (s *Server) handleListProjectUpdates(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	pageInput, err := parseListProjectUpdatesPageRequest(projectID, projectupdatedomain.ListThreadsPageRequest{
		Limit:  c.QueryParam("limit"),
		Before: c.QueryParam("before"),
	})
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_UPDATES_PAGE", err.Error())
	}
	page, err := s.projectUpdateService.ListThreadPage(c.Request().Context(), pageInput)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"threads":     mapProjectUpdateThreadResponses(page.Threads),
		"next_cursor": page.NextCursor,
		"has_more":    page.HasMore,
	})
}

func (s *Server) handleCreateProjectUpdateThread(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	var raw rawCreateProjectUpdateThreadRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseCreateProjectUpdateThreadRequest(projectID, actorFromWritePrincipal(c), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	item, err := s.projectUpdateService.AddThread(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"thread": mapProjectUpdateThreadResponse(item),
	})
}

func (s *Server) handleUpdateProjectUpdateThread(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	var raw rawUpdateProjectUpdateThreadRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseUpdateProjectUpdateThreadRequest(projectID, threadID, actorFromWritePrincipal(c), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	item, err := s.projectUpdateService.UpdateThread(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"thread": mapProjectUpdateThreadResponse(item),
	})
}

func (s *Server) handleDeleteProjectUpdateThread(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	result, err := s.projectUpdateService.RemoveThread(c.Request().Context(), projectID, threadID)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"deleted_thread_id": result.DeletedThreadID.String(),
	})
}

func (s *Server) handleListProjectUpdateThreadRevisions(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	items, err := s.projectUpdateService.ListThreadRevisions(c.Request().Context(), projectID, threadID)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"revisions": mapProjectUpdateThreadRevisionResponses(items),
	})
}

func (s *Server) handleCreateProjectUpdateComment(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	var raw rawCreateProjectUpdateCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseCreateProjectUpdateCommentRequest(projectID, threadID, actorFromWritePrincipal(c), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	item, err := s.projectUpdateService.AddComment(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"comment": mapProjectUpdateCommentResponse(item),
	})
}

func (s *Server) handleUpdateProjectUpdateComment(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	commentID, err := parseUUIDPathParam(c, "commentId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}
	var raw rawUpdateProjectUpdateCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseUpdateProjectUpdateCommentRequest(projectID, threadID, commentID, actorFromWritePrincipal(c), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	item, err := s.projectUpdateService.UpdateComment(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"comment": mapProjectUpdateCommentResponse(item),
	})
}

func (s *Server) handleDeleteProjectUpdateComment(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	commentID, err := parseUUIDPathParam(c, "commentId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}
	result, err := s.projectUpdateService.RemoveComment(c.Request().Context(), projectID, threadID, commentID)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"deleted_comment_id": result.DeletedCommentID.String(),
	})
}

func (s *Server) handleListProjectUpdateCommentRevisions(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	commentID, err := parseUUIDPathParam(c, "commentId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}
	items, err := s.projectUpdateService.ListCommentRevisions(c.Request().Context(), projectID, threadID, commentID)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"revisions": mapProjectUpdateCommentRevisionResponses(items),
	})
}

func writeProjectUpdateError(c echo.Context, err error) error {
	switch err {
	case projectupdateservice.ErrUnavailable:
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case projectupdateservice.ErrProjectNotFound:
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case projectupdateservice.ErrThreadNotFound:
		return writeAPIError(c, http.StatusNotFound, "UPDATE_THREAD_NOT_FOUND", err.Error())
	case projectupdateservice.ErrCommentNotFound:
		return writeAPIError(c, http.StatusNotFound, "UPDATE_COMMENT_NOT_FOUND", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func mapProjectUpdateThreadResponses(items []projectupdateservice.Thread) []projectUpdateThreadResponse {
	response := make([]projectUpdateThreadResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectUpdateThreadResponse(item))
	}
	return response
}

func mapProjectUpdateThreadResponse(item projectupdateservice.Thread) projectUpdateThreadResponse {
	return projectUpdateThreadResponse{
		ID:             item.ID.String(),
		ProjectID:      item.ProjectID.String(),
		Status:         string(item.Status),
		Title:          item.Title,
		BodyMarkdown:   item.BodyMarkdown,
		CreatedBy:      item.CreatedBy,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
		EditedAt:       projectUpdateTimePointerString(item.EditedAt),
		EditCount:      item.EditCount,
		LastEditedBy:   item.LastEditedBy,
		IsDeleted:      item.IsDeleted,
		DeletedAt:      projectUpdateTimePointerString(item.DeletedAt),
		DeletedBy:      item.DeletedBy,
		LastActivityAt: item.LastActivityAt.UTC().Format(time.RFC3339),
		CommentCount:   item.CommentCount,
		Comments:       mapProjectUpdateCommentResponses(item.Comments),
	}
}

func mapProjectUpdateCommentResponses(items []projectupdateservice.Comment) []projectUpdateCommentResponse {
	response := make([]projectUpdateCommentResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectUpdateCommentResponse(item))
	}
	return response
}

func mapProjectUpdateCommentResponse(item projectupdateservice.Comment) projectUpdateCommentResponse {
	return projectUpdateCommentResponse{
		ID:           item.ID.String(),
		ThreadID:     item.ThreadID.String(),
		BodyMarkdown: item.BodyMarkdown,
		CreatedBy:    item.CreatedBy,
		CreatedAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.UTC().Format(time.RFC3339),
		EditedAt:     projectUpdateTimePointerString(item.EditedAt),
		EditCount:    item.EditCount,
		LastEditedBy: item.LastEditedBy,
		IsDeleted:    item.IsDeleted,
		DeletedAt:    projectUpdateTimePointerString(item.DeletedAt),
		DeletedBy:    item.DeletedBy,
	}
}

func mapProjectUpdateThreadRevisionResponses(items []projectupdateservice.ThreadRevision) []projectUpdateThreadRevisionResponse {
	response := make([]projectUpdateThreadRevisionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, projectUpdateThreadRevisionResponse{
			ID:             item.ID.String(),
			ThreadID:       item.ThreadID.String(),
			RevisionNumber: item.RevisionNumber,
			Status:         string(item.Status),
			Title:          item.Title,
			BodyMarkdown:   item.BodyMarkdown,
			EditedBy:       item.EditedBy,
			EditedAt:       item.EditedAt.UTC().Format(time.RFC3339),
			EditReason:     item.EditReason,
		})
	}
	return response
}

func mapProjectUpdateCommentRevisionResponses(items []projectupdateservice.CommentRevision) []projectUpdateCommentRevisionResponse {
	response := make([]projectUpdateCommentRevisionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, projectUpdateCommentRevisionResponse{
			ID:             item.ID.String(),
			CommentID:      item.CommentID.String(),
			RevisionNumber: item.RevisionNumber,
			BodyMarkdown:   item.BodyMarkdown,
			EditedBy:       item.EditedBy,
			EditedAt:       item.EditedAt.UTC().Format(time.RFC3339),
			EditReason:     item.EditReason,
		})
	}
	return response
}

func projectUpdateTimePointerString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
