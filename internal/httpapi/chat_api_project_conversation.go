package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleProjectConversationMuxStream(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}

	projectID, err := parseUUIDString("project_id", c.Param("projectId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()
	heartbeat := time.NewTicker(s.chatStreamKeepaliveInterval())
	defer heartbeat.Stop()

	events, cleanup, err := s.projectConversationService.WatchProjectConversations(
		streamCtx,
		userID,
		projectID,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	defer cleanup()

	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("disable project conversation mux sse write deadline: %w", err)
	}

	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set("Connection", "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)
	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
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
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if err := writeSSEFrame(response, event.Event, map[string]any{
				"conversation_id": event.ConversationID.String(),
				"payload":         event.Payload,
				"sent_at":         event.SentAt.UTC().Format(time.RFC3339),
			}); err != nil {
				return nil
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleCreateProjectConversation(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}

	var raw rawCreateConversationRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseCreateProjectConversationRequest(raw)
	if err != nil {
		code := "INVALID_REQUEST"
		if strings.Contains(err.Error(), "source") {
			code = "INVALID_CHAT_SOURCE"
		}
		return writeAPIError(c, http.StatusBadRequest, code, err.Error())
	}
	if request.Source != chatdomain.SourceProjectSidebar {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_SOURCE", "project conversations only support project_sidebar")
	}
	if err := s.requireHumanPermission(c, humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindProject, ID: request.ProjectID.String()}, humanauthdomain.PermissionConversationCreate); err != nil {
		return err
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	conversation, err := s.projectConversationService.CreateConversation(
		c.Request().Context(),
		userID,
		request.ProjectID,
		request.ProviderID,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{"conversation": s.mapProjectConversationResponse(c.Request().Context(), conversation)})
}

func (s *Server) handleListProjectConversations(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}

	projectID, err := parseUUIDString("project_id", c.QueryParam("project_id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if err := s.requireHumanPermission(
		c,
		humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindProject, ID: projectID.String()},
		humanauthdomain.PermissionConversationRead,
	); err != nil {
		return err
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var providerID *uuid.UUID
	if trimmed := strings.TrimSpace(c.QueryParam("provider_id")); trimmed != "" {
		parsed, err := parseUUIDString("provider_id", trimmed)
		if err != nil {
			return writeAPIError(c, http.StatusBadRequest, "INVALID_PROVIDER_ID", err.Error())
		}
		providerID = &parsed
	}

	items, err := s.projectConversationService.ListConversations(c.Request().Context(), userID, projectID, providerID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"conversations": s.mapProjectConversationResponses(c.Request().Context(), items)})
}

func (s *Server) handleGetProjectConversation(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.GetConversation(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"conversation": s.mapProjectConversationResponse(c.Request().Context(), item)})
}

func (s *Server) handleListProjectConversationEntries(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	items, err := s.projectConversationService.ListEntries(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"entries": mapProjectConversationEntries(items)})
}

func (s *Server) handleGetProjectConversationWorkspace(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.GetWorkspaceMetadata(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"workspace": mapProjectConversationWorkspaceMetadataResponse(item)})
}

func (s *Server) handleSyncProjectConversationWorkspace(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.SyncWorkspace(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"workspace": mapProjectConversationWorkspaceMetadataResponse(item)})
}

func (s *Server) handleGetProjectConversationWorkspaceRepoRefs(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	request, err := parseProjectConversationWorkspaceRepoRefsRequest(c.QueryParam("repo_path"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.GetWorkspaceRepoRefs(
		c.Request().Context(),
		userID,
		conversationID,
		request.RepoPath,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"repo_refs": mapProjectConversationWorkspaceRepoRefsResponse(item)})
}

func (s *Server) handleGetProjectConversationWorkspaceGitGraph(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	request, err := parseProjectConversationWorkspaceGitGraphRequest(
		c.QueryParam("repo_path"),
		c.QueryParam("limit"),
	)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.GetWorkspaceGitGraph(
		c.Request().Context(),
		userID,
		conversationID,
		request.RepoPath,
		request.Window,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"git_graph": mapProjectConversationWorkspaceGitGraphResponse(item)})
}

func (s *Server) handlePostProjectConversationWorkspaceCheckout(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw rawProjectConversationWorkspaceCheckoutRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseProjectConversationWorkspaceCheckoutRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	item, err := s.projectConversationService.CheckoutWorkspaceBranch(
		c.Request().Context(),
		userID,
		conversationID,
		request,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"checkout": mapProjectConversationWorkspaceCheckoutResponse(item)})
}

func (s *Server) handlePostProjectConversationWorkspaceGitRemoteOp(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw struct {
		RepoPath string `json:"repo_path"`
		Op       string `json:"op"`
	}
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	op := chatservice.ProjectConversationWorkspaceGitRemoteOpKind(raw.Op)
	switch op {
	case chatservice.ProjectConversationWorkspaceGitRemoteOpFetch,
		chatservice.ProjectConversationWorkspaceGitRemoteOpPull,
		chatservice.ProjectConversationWorkspaceGitRemoteOpPush:
	default:
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "op must be fetch, pull, or push")
	}

	item, err := s.projectConversationService.RunWorkspaceGitRemoteOp(
		c.Request().Context(),
		userID,
		conversationID,
		chatservice.ProjectConversationWorkspaceGitRemoteOpInput{
			RepoPath: repoPath,
			Op:       op,
		},
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"conversation_id": item.ConversationID.String(),
		"repo_path":       item.RepoPath,
		"op":              string(item.Op),
		"output":          item.Output,
	})
}

func (s *Server) handlePostProjectConversationWorkspaceGitStage(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw struct {
		RepoPath string `json:"repo_path"`
		Path     string `json:"path"`
	}
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	if strings.TrimSpace(raw.Path) == "" {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
	}

	item, err := s.projectConversationService.StageWorkspaceFile(
		c.Request().Context(),
		userID,
		conversationID,
		chatservice.ProjectConversationWorkspaceStageFileInput{
			RepoPath: repoPath,
			Path:     raw.Path,
		},
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"conversation_id": item.ConversationID.String(),
		"repo_path":       item.RepoPath,
		"path":            item.Path,
	})
}

func (s *Server) handlePostProjectConversationWorkspaceGitStageAll(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw struct {
		RepoPath string `json:"repo_path"`
	}
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectConversationService.StageWorkspaceAll(
		c.Request().Context(),
		userID,
		conversationID,
		chatservice.ProjectConversationWorkspaceStageAllInput{
			RepoPath: repoPath,
		},
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"conversation_id": item.ConversationID.String(),
		"repo_path":       item.RepoPath,
	})
}

func (s *Server) handlePostProjectConversationWorkspaceGitUnstage(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw struct {
		RepoPath string `json:"repo_path"`
		Path     string `json:"path"`
	}
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectConversationService.UnstageWorkspace(
		c.Request().Context(),
		userID,
		conversationID,
		chatservice.ProjectConversationWorkspaceUnstageInput{
			RepoPath: repoPath,
			Path:     raw.Path,
		},
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"conversation_id": item.ConversationID.String(),
		"repo_path":       item.RepoPath,
		"path":            item.Path,
	})
}

func (s *Server) handlePostProjectConversationWorkspaceGitCommit(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw struct {
		RepoPath string `json:"repo_path"`
		Message  string `json:"message"`
	}
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	if strings.TrimSpace(raw.Message) == "" {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "message is required")
	}

	item, err := s.projectConversationService.CommitWorkspace(
		c.Request().Context(),
		userID,
		conversationID,
		chatservice.ProjectConversationWorkspaceCommitInput{
			RepoPath: repoPath,
			Message:  raw.Message,
		},
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"conversation_id": item.ConversationID.String(),
		"repo_path":       item.RepoPath,
		"output":          item.Output,
	})
}

func (s *Server) handlePostProjectConversationWorkspaceGitDiscard(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw struct {
		RepoPath string `json:"repo_path"`
		Path     string `json:"path"`
	}
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	if strings.TrimSpace(raw.Path) == "" {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
	}

	item, err := s.projectConversationService.DiscardWorkspaceFile(
		c.Request().Context(),
		userID,
		conversationID,
		chatservice.ProjectConversationWorkspaceDiscardFileInput{
			RepoPath: repoPath,
			Path:     raw.Path,
		},
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"conversation_id": item.ConversationID.String(),
		"repo_path":       item.RepoPath,
		"path":            item.Path,
	})
}

func (s *Server) handlePostProjectConversationWorkspaceCreateBranch(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw struct {
		RepoPath   string `json:"repo_path"`
		BranchName string `json:"branch_name"`
		StartPoint string `json:"start_point"`
	}
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	repoPath, err := chatservice.ParseWorkspaceRepoPath(raw.RepoPath)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	branchName := chatservice.WorkspaceBranchName(raw.BranchName)
	if branchName.String() == "" {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "branch_name is required")
	}

	item, err := s.projectConversationService.CreateWorkspaceBranch(
		c.Request().Context(),
		userID,
		conversationID,
		chatservice.ProjectConversationWorkspaceCreateBranchInput{
			RepoPath:   repoPath,
			BranchName: branchName,
			StartPoint: raw.StartPoint,
		},
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"conversation_id": item.ConversationID.String(),
		"repo_path":       item.RepoPath,
		"branch_name":     item.BranchName,
	})
}

func (s *Server) handleListProjectConversationWorkspaceTree(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	request, err := parseProjectConversationWorkspaceTreeRequest(c.QueryParam("repo_path"), c.QueryParam("path"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.ListWorkspaceTree(
		c.Request().Context(),
		userID,
		conversationID,
		request.RepoPath,
		request.Path,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"workspace_tree": mapProjectConversationWorkspaceTreeResponse(item)})
}

func (s *Server) handleSearchProjectConversationWorkspacePaths(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	request, err := parseProjectConversationWorkspaceSearchRequest(
		c.QueryParam("repo_path"),
		c.QueryParam("q"),
		c.QueryParam("limit"),
	)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.SearchWorkspacePaths(
		c.Request().Context(),
		userID,
		conversationID,
		request.RepoPath,
		request.Query,
		request.Limit,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"workspace_search": mapProjectConversationWorkspaceSearchResponse(item)})
}

func (s *Server) handleGetProjectConversationWorkspaceFile(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	request, err := parseProjectConversationWorkspaceFileRequest(c.QueryParam("repo_path"), c.QueryParam("path"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.ReadWorkspaceFilePreview(
		c.Request().Context(),
		userID,
		conversationID,
		request.RepoPath,
		request.Path,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"file_preview": mapProjectConversationWorkspaceFilePreviewResponse(item)})
}

func (s *Server) handlePutProjectConversationWorkspaceFile(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw rawUpdateProjectConversationWorkspaceFileRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseUpdateProjectConversationWorkspaceFileRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectConversationService.SaveWorkspaceFile(
		c.Request().Context(),
		userID,
		conversationID,
		request.File,
	)
	if err != nil {
		var conflictErr *chatservice.ProjectConversationWorkspaceFileConflictError
		if errors.As(err, &conflictErr) {
			details := map[string]any{
				"current_file": mapProjectConversationWorkspaceFilePreviewResponse(conflictErr.CurrentFile),
			}
			return writeAPIErrorWithDetails(
				c,
				http.StatusConflict,
				"PROJECT_CONVERSATION_WORKSPACE_FILE_CONFLICT",
				conflictErr.Error(),
				details,
			)
		}
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"file": mapProjectConversationWorkspaceFileSavedResponse(item)})
}

func (s *Server) handlePostProjectConversationWorkspaceFile(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw rawCreateProjectConversationWorkspaceFileRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseCreateProjectConversationWorkspaceFileRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectConversationService.CreateWorkspaceFile(
		c.Request().Context(),
		userID,
		conversationID,
		request.File,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"file": mapProjectConversationWorkspaceFileCreatedResponse(item)})
}

func (s *Server) handlePatchProjectConversationWorkspaceFile(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw rawRenameProjectConversationWorkspaceFileRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseRenameProjectConversationWorkspaceFileRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectConversationService.RenameWorkspaceFile(
		c.Request().Context(),
		userID,
		conversationID,
		request.File,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"file": mapProjectConversationWorkspaceFileRenamedResponse(item)})
}

func (s *Server) handleDeleteProjectConversationWorkspaceFile(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw rawDeleteProjectConversationWorkspaceFileRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseDeleteProjectConversationWorkspaceFileRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectConversationService.DeleteWorkspaceFile(
		c.Request().Context(),
		userID,
		conversationID,
		request.File,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"file": mapProjectConversationWorkspaceFileDeletedResponse(item)})
}

func (s *Server) handleGetProjectConversationWorkspaceFilePatch(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	request, err := parseProjectConversationWorkspaceFileRequest(c.QueryParam("repo_path"), c.QueryParam("path"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.ReadWorkspaceFilePatch(
		c.Request().Context(),
		userID,
		conversationID,
		request.RepoPath,
		request.Path,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"file_patch": mapProjectConversationWorkspaceFilePatchResponse(item)})
}

func (s *Server) handleGetProjectConversationWorkspaceDiff(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	item, err := s.projectConversationService.GetWorkspaceDiff(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"workspace_diff": mapProjectConversationWorkspaceDiffResponse(item)})
}

func (s *Server) handleCreateProjectConversationTerminalSession(c echo.Context) error {
	if s.conversationTerminalService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "conversation terminal service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	var raw rawCreateProjectConversationTerminalSessionRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseCreateProjectConversationTerminalSessionRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	session, err := s.conversationTerminalService.CreateSession(
		c.Request().Context(),
		userID,
		conversationID,
		request.Terminal,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{"terminal_session": mapProjectConversationTerminalSessionResponse(session)})
}

func (s *Server) handleAttachProjectConversationTerminalSession(c echo.Context) error {
	if s.conversationTerminalService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "conversation terminal service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	terminalSessionID, err := parseUUIDString("terminal_session_id", c.Param("terminalSessionId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TERMINAL_SESSION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	attachment, err := s.conversationTerminalService.AttachSession(
		userID,
		conversationID,
		terminalSessionID,
		c.QueryParam("attach_token"),
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	conn, err := conversationTerminalUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return fmt.Errorf("upgrade conversation terminal websocket: %w", err)
	}
	defer func() { _ = conn.Close() }()

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()
	readErr := make(chan error, 1)
	go func() {
		readErr <- s.readConversationTerminalFrames(streamCtx, conn, attachment)
	}()
	readFrames := readErr

	for {
		select {
		case <-streamCtx.Done():
			_ = attachment.Detach()
			return nil
		case err := <-readFrames:
			switch {
			case err == nil, errors.Is(err, errConversationTerminalExplicitClose):
				readFrames = nil
				continue
			case websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway), errors.Is(err, io.EOF):
				_ = attachment.Detach()
				return nil
			default:
				_ = attachment.Detach()
				return nil
			}
		case event, ok := <-attachment.Events:
			if !ok {
				return nil
			}
			if err := writeConversationTerminalFrame(conn, event); err != nil {
				_ = attachment.Detach()
				return nil
			}
			if event.Type == "exit" || event.Type == "error" {
				return nil
			}
		}
	}
}

func (s *Server) handleStartProjectConversationTurn(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw rawConversationTurnRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	request, err := parseProjectConversationTurnRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	var turn chatdomain.Turn
	if request.WorkspaceFileDraft != nil {
		turn, err = s.projectConversationService.StartTurnWithWorkspaceFileDraft(
			c.Request().Context(),
			userID,
			conversationID,
			request.Message,
			request.Focus,
			request.WorkspaceFileDraft,
		)
	} else {
		turn, err = s.projectConversationService.StartTurn(
			c.Request().Context(),
			userID,
			conversationID,
			request.Message,
			request.Focus,
		)
	}
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	response := map[string]any{
		"turn": map[string]any{
			"id":         turn.ID.String(),
			"turn_index": turn.TurnIndex,
			"status":     string(turn.Status),
		},
	}
	conversation, err := s.projectConversationService.GetConversation(
		c.Request().Context(),
		userID,
		conversationID,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	response["conversation"] = s.mapProjectConversationResponse(c.Request().Context(), conversation)
	return c.JSON(http.StatusAccepted, response)
}

func (s *Server) handleInterruptProjectConversationTurn(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	if err := s.projectConversationService.InterruptTurn(c.Request().Context(), userID, conversationID); err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.NoContent(http.StatusAccepted)
}

func (s *Server) handleProjectConversationStream(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	events, cleanup, err := s.projectConversationService.WatchConversation(
		streamCtx,
		userID,
		conversationID,
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	defer cleanup()

	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("disable project conversation sse write deadline: %w", err)
	}

	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set("Connection", "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)
	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}
	if _, err := response.Write([]byte(": keepalive\n\n")); err != nil {
		return nil
	}
	flusher.Flush()

	for {
		select {
		case <-streamCtx.Done():
			return nil
		case event, ok := <-events:
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

func (s *Server) handleRespondProjectConversationInterrupt(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	interruptID, err := parseUUIDString("interrupt_id", c.Param("interruptId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_INTERRUPT_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	var raw rawInterruptResponseRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	interrupt, err := s.projectConversationService.RespondInterrupt(
		c.Request().Context(),
		userID,
		conversationID,
		interruptID,
		parseInterruptResponseRequest(raw),
	)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"interrupt": mapPendingInterruptResponse(interrupt)})
}

func (s *Server) handleDeleteProjectConversationRuntime(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	if err := s.projectConversationService.CloseRuntime(c.Request().Context(), userID, conversationID); err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleDeleteProjectConversation(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	force, err := parseOptionalBoolQueryParam("force", c.QueryParam("force"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_FORCE", err.Error())
	}
	userID, err := s.currentProjectConversationUserID(c)
	if err != nil {
		return writeChatUserError(c, err)
	}
	if _, err := s.projectConversationService.DeleteConversation(
		c.Request().Context(),
		userID,
		conversationID,
		chatdomain.DeleteConversationInput{Force: force},
	); err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
