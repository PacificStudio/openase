package httpapi

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
)

type skillResponse struct {
	ID             string                         `json:"id"`
	Name           string                         `json:"name"`
	Description    string                         `json:"description"`
	Path           string                         `json:"path"`
	CurrentVersion int                            `json:"current_version"`
	IsBuiltin      bool                           `json:"is_builtin"`
	IsEnabled      bool                           `json:"is_enabled"`
	CreatedBy      string                         `json:"created_by"`
	CreatedAt      string                         `json:"created_at"`
	BoundWorkflows []skillWorkflowBindingResponse `json:"bound_workflows"`
}

type skillVersionResponse struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type skillDetailResponse struct {
	Skill   skillResponse          `json:"skill"`
	Content string                 `json:"content"`
	Files   []skillFileResponse    `json:"files,omitempty"`
	History []skillVersionResponse `json:"history"`
}

type skillHistoryResponse struct {
	History []skillVersionResponse `json:"history"`
}

type skillWorkflowBindingResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HarnessPath string `json:"harness_path"`
}

type skillSyncResponse struct {
	SkillsDir      string   `json:"skills_dir"`
	InjectedSkills []string `json:"injected_skills,omitempty"`
}

type skillFileResponse struct {
	Path          string `json:"path"`
	FileKind      string `json:"file_kind"`
	MediaType     string `json:"media_type"`
	Encoding      string `json:"encoding"`
	IsExecutable  bool   `json:"is_executable"`
	SizeBytes     int64  `json:"size_bytes"`
	SHA256        string `json:"sha256"`
	Content       string `json:"content,omitempty"`
	ContentBase64 string `json:"content_base64,omitempty"`
}

type skillFilesResponse struct {
	Files []skillFileResponse `json:"files"`
}

func (s *Server) registerSkillRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/skills", s.handleListSkills)
	api.POST("/projects/:projectId/skills", s.handleCreateSkill)
	api.POST("/projects/:projectId/skills/import", s.handleImportSkillBundle)
	api.POST("/projects/:projectId/skills/refresh", s.handleRefreshSkills)
	api.GET("/skills/:skillId", s.handleGetSkill)
	api.GET("/skills/:skillId/files", s.handleGetSkillFiles)
	api.GET("/skills/:skillId/history", s.handleGetSkillHistory)
	api.PUT("/skills/:skillId", s.handleUpdateSkill)
	api.DELETE("/skills/:skillId", s.handleDeleteSkill)
	api.POST("/skills/:skillId/enable", s.handleEnableSkill)
	api.POST("/skills/:skillId/disable", s.handleDisableSkill)
	api.POST("/skills/:skillId/bind", s.handleBindSkill)
	api.POST("/skills/:skillId/unbind", s.handleUnbindSkill)
	api.POST("/skills/:skillId/refinement-runs", s.handleStartSkillRefinement)
	api.DELETE("/skills/refinement-runs/:sessionId", s.handleDeleteSkillRefinementSession)
	api.POST("/workflows/:workflowId/skills/bind", s.handleBindWorkflowSkills)
	api.POST("/workflows/:workflowId/skills/unbind", s.handleUnbindWorkflowSkills)
}

func (s *Server) handleListSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	items, err := s.workflowService.ListSkills(c.Request().Context(), projectID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"skills": mapSkillResponses(items),
	})
}

func (s *Server) handleCreateSkill(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawCreateSkillRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	if actor := actorFromHumanPrincipal(c); strings.TrimSpace(raw.CreatedBy) == "" {
		raw.CreatedBy = actor
	}

	input, err := parseCreateSkillRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.workflowService.CreateSkill(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusCreated, mapSkillDetailResponse(item))
}

func (s *Server) handleImportSkillBundle(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawImportSkillBundleRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	if actor := actorFromHumanPrincipal(c); strings.TrimSpace(raw.CreatedBy) == "" {
		raw.CreatedBy = actor
	}

	input, err := parseImportSkillBundleRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.workflowService.CreateSkillBundle(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusCreated, mapSkillDetailResponse(item))
}

func (s *Server) handleRefreshSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawSkillSyncRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseRefreshSkillsRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	result, err := s.workflowService.RefreshSkills(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, skillSyncResponse{
		SkillsDir:      result.SkillsDir,
		InjectedSkills: result.InjectedSkills,
	})
}

func (s *Server) handleGetSkill(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	item, err := s.workflowService.GetSkill(c.Request().Context(), skillID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, mapSkillDetailResponse(item))
}

func (s *Server) handleGetSkillFiles(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	item, err := s.workflowService.GetSkill(c.Request().Context(), skillID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, skillFilesResponse{
		Files: mapSkillFileResponses(item.Files),
	})
}

func (s *Server) handleGetSkillHistory(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	items, err := s.workflowService.ListSkillVersions(c.Request().Context(), skillID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, skillHistoryResponse{
		History: mapSkillVersionResponses(items),
	})
}

func (s *Server) handleUpdateSkill(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	var raw rawUpdateSkillRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateSkillRequest(skillID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	var item workflowservice.SkillDetail
	switch {
	case input.BundleFiles != nil:
		item, err = s.workflowService.UpdateSkillBundle(c.Request().Context(), *input.BundleFiles)
	case input.SingleFile != nil:
		item, err = s.workflowService.UpdateSkill(c.Request().Context(), *input.SingleFile)
	default:
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "update payload must include content or files")
	}
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, mapSkillDetailResponse(item))
}

func (s *Server) handleDeleteSkill(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	if err := s.workflowService.DeleteSkill(c.Request().Context(), skillID); err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"deleted_skill_id": skillID.String(),
	})
}

func (s *Server) handleEnableSkill(c echo.Context) error {
	return s.handleSkillEnabledState(c, true)
}

func (s *Server) handleDisableSkill(c echo.Context) error {
	return s.handleSkillEnabledState(c, false)
}

func (s *Server) handleSkillEnabledState(c echo.Context, enabled bool) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	var item workflowservice.SkillDetail
	if enabled {
		item, err = s.workflowService.EnableSkill(c.Request().Context(), skillID)
	} else {
		item, err = s.workflowService.DisableSkill(c.Request().Context(), skillID)
	}
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, mapSkillDetailResponse(item))
}

func (s *Server) handleBindSkill(c echo.Context) error {
	return s.handleSkillBindings(c, true)
}

func (s *Server) handleUnbindSkill(c echo.Context) error {
	return s.handleSkillBindings(c, false)
}

func (s *Server) handleStartSkillRefinement(c echo.Context) error {
	if s.skillRefinementService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "skill refinement service unavailable")
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	var raw rawSkillRefinementRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseSkillRefinementRequest(skillID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	userID, err := s.currentRequestChatUserID(c)
	if err != nil {
		if errors.Is(err, humanauthservice.ErrUnauthorized) {
			return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", err.Error())
		}
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	stream, err := s.skillRefinementService.Start(streamCtx, userID, input)
	if err != nil {
		return writeSkillRefinementError(c, err)
	}

	heartbeat := time.NewTicker(s.chatStreamKeepaliveInterval())
	defer heartbeat.Stop()

	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set("Connection", "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "response writer does not support flushing")
	}
	if _, err := response.Write([]byte(": keepalive\n\n")); err != nil {
		return nil
	}
	flusher.Flush()

	for {
		select {
		case <-streamCtx.Done():
			return nil
		case <-heartbeat.C:
			if _, err := response.Write([]byte(": keepalive\n\n")); err != nil {
				return nil
			}
			flusher.Flush()
		case event, ok := <-stream.Events:
			if !ok {
				return nil
			}
			if err := writeSSEFrame(response, event.Event, event.Payload); err != nil {
				return nil
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleDeleteSkillRefinementSession(c echo.Context) error {
	if s.skillRefinementService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "skill refinement service unavailable")
	}

	sessionID, err := chatservice.ParseCloseSessionID(c.Param("sessionId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SESSION_ID", err.Error())
	}
	userID, err := s.currentRequestChatUserID(c)
	if err != nil {
		if errors.Is(err, humanauthservice.ErrUnauthorized) {
			return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", err.Error())
		}
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}
	if !s.skillRefinementService.CloseSession(userID, sessionID) {
		return writeAPIError(c, http.StatusNotFound, "SKILL_REFINEMENT_SESSION_NOT_FOUND", "skill refinement session not found")
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleSkillBindings(c echo.Context, bind bool) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	skillID, err := parseUUIDPathParamValue(c, "skillId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL_ID", err.Error())
	}

	var raw rawUpdateSkillBindingsRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateSkillBindingsRequest(skillID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	var item workflowservice.SkillDetail
	if bind {
		item, err = s.workflowService.BindSkill(c.Request().Context(), input)
	} else {
		item, err = s.workflowService.UnbindSkill(c.Request().Context(), input)
	}
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, mapSkillDetailResponse(item))
}

func (s *Server) handleBindWorkflowSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawUpdateWorkflowSkillsRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateWorkflowSkillsRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	document, err := s.workflowService.BindSkills(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"harness": mapHarnessResponse(document),
	})
}

func (s *Server) handleUnbindWorkflowSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawUpdateWorkflowSkillsRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateWorkflowSkillsRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	document, err := s.workflowService.UnbindSkills(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"harness": mapHarnessResponse(document),
	})
}

func mapSkillResponses(items []workflowservice.Skill) []skillResponse {
	response := make([]skillResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapSkillResponse(item))
	}
	return response
}

func mapSkillResponse(item workflowservice.Skill) skillResponse {
	return skillResponse{
		ID:             item.ID.String(),
		Name:           item.Name,
		Description:    item.Description,
		Path:           item.Path,
		CurrentVersion: item.CurrentVersion,
		IsBuiltin:      item.IsBuiltin,
		IsEnabled:      item.IsEnabled,
		CreatedBy:      item.CreatedBy,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		BoundWorkflows: mapSkillWorkflowBindings(item.BoundWorkflows),
	}
}

func mapSkillDetailResponse(item workflowservice.SkillDetail) skillDetailResponse {
	return skillDetailResponse{
		Skill:   mapSkillResponse(item.Skill),
		Content: item.Content,
		Files:   mapSkillFileResponses(item.Files),
		History: mapSkillVersionResponses(item.History),
	}
}

func mapSkillFileResponses(items []workflowservice.SkillBundleFile) []skillFileResponse {
	response := make([]skillFileResponse, 0, len(items))
	for _, item := range items {
		file := skillFileResponse{
			Path:          item.Path,
			FileKind:      item.FileKind,
			MediaType:     item.MediaType,
			Encoding:      item.Encoding,
			IsExecutable:  item.IsExecutable,
			SizeBytes:     item.SizeBytes,
			SHA256:        item.SHA256,
			ContentBase64: base64.StdEncoding.EncodeToString(item.Content),
		}
		if item.Encoding == "utf8" {
			file.Content = string(item.Content)
		}
		response = append(response, file)
	}
	return response
}

func mapSkillWorkflowBindings(items []workflowservice.SkillWorkflowBinding) []skillWorkflowBindingResponse {
	response := make([]skillWorkflowBindingResponse, 0, len(items))
	for _, item := range items {
		response = append(response, skillWorkflowBindingResponse{
			ID:          item.ID.String(),
			Name:        item.Name,
			HarnessPath: item.HarnessPath,
		})
	}
	return response
}

func mapSkillVersionResponses(items []workflowservice.VersionSummary) []skillVersionResponse {
	response := make([]skillVersionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, skillVersionResponse{
			ID:        item.ID.String(),
			Version:   item.Version,
			CreatedBy: item.CreatedBy,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return response
}

func writeSkillRefinementError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, chatservice.ErrSkillRefinementUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, chatservice.ErrProviderNotFound):
		return writeAPIError(c, http.StatusConflict, "CHAT_PROVIDER_NOT_CONFIGURED", err.Error())
	case errors.Is(err, chatservice.ErrProviderUnavailable):
		return writeAPIError(c, http.StatusConflict, "CHAT_PROVIDER_UNAVAILABLE", err.Error())
	case errors.Is(err, chatservice.ErrProviderUnsupported):
		return writeAPIError(c, http.StatusConflict, "CHAT_PROVIDER_UNSUPPORTED", err.Error())
	case errors.Is(err, workflowservice.ErrSkillNotFound),
		errors.Is(err, catalogservice.ErrNotFound):
		return writeAPIError(c, http.StatusNotFound, "CHAT_CONTEXT_NOT_FOUND", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
