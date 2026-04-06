package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var chatSSEKeepaliveInterval = 5 * time.Second

func (s *Server) registerChatRoutes(api *echo.Group) {
	api.POST("/chat", s.handleStartChat)
	api.DELETE("/chat/:sessionId", s.handleDeleteChat)
	api.GET("/chat/projects/:projectId/conversations/stream", s.handleProjectConversationMuxStream)
	api.POST("/chat/conversations", s.handleCreateProjectConversation)
	api.GET("/chat/conversations", s.handleListProjectConversations)
	api.GET("/chat/conversations/:conversationId", s.handleGetProjectConversation)
	api.GET("/chat/conversations/:conversationId/entries", s.handleListProjectConversationEntries)
	api.GET("/chat/conversations/:conversationId/workspace-diff", s.handleGetProjectConversationWorkspaceDiff)
	api.POST("/chat/conversations/:conversationId/turns", s.handleStartProjectConversationTurn)
	api.GET("/chat/conversations/:conversationId/stream", s.handleProjectConversationStream)
	api.POST("/chat/conversations/:conversationId/interrupts/:interruptId/respond", s.handleRespondProjectConversationInterrupt)
	api.DELETE("/chat/conversations/:conversationId/runtime", s.handleDeleteProjectConversationRuntime)
}

func (s *Server) handleStartChat(c echo.Context) error {
	if s.chatService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "chat service unavailable")
	}

	var raw chatservice.RawStartInput
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := chatservice.ParseStartInput(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	if err := s.requireHumanPermission(c, humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindProject, ID: input.Context.ProjectID.String()}, humanauthdomain.PermissionProjectRead); err != nil {
		return err
	}

	userID, err := s.currentRequestAIPrincipal(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	stream, err := s.chatService.StartTurn(streamCtx, userID, input)
	if err != nil {
		s.logChatStartFailure(c, raw, input, userID, err)
		return writeChatError(c, err)
	}

	streamLog := s.chatStreamLogger(c, input, userID)
	streamStartedAt := time.Now()
	heartbeat := time.NewTicker(s.chatStreamKeepaliveInterval())
	defer heartbeat.Stop()
	eventsSent := 0
	lastEvent := "keepalive"
	terminalEventSeen := false

	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		streamLog.Error("disable chat sse write deadline failed", "error", err)
		return fmt.Errorf("disable chat sse write deadline: %w", err)
	}

	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set("Connection", "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		streamLog.Error("chat stream missing http flusher")
		return fmt.Errorf("response writer does not support flushing")
	}
	if _, err := response.Write([]byte(": keepalive\n\n")); err != nil {
		streamLog.Warn("chat stream initial keepalive write failed", "error", err)
		return nil
	}
	flusher.Flush()

	for {
		select {
		case <-streamCtx.Done():
			streamLog.Warn(
				"chat stream request context ended before completion",
				"error", streamCtx.Err(),
				"duration", time.Since(streamStartedAt).String(),
				"events_sent", eventsSent,
				"last_event", lastEvent,
				"terminal_event_seen", terminalEventSeen,
			)
			return nil
		case <-heartbeat.C:
			if _, err := response.Write([]byte(": keepalive\n\n")); err != nil {
				streamLog.Warn(
					"chat stream keepalive write failed",
					"error", err,
					"duration", time.Since(streamStartedAt).String(),
					"events_sent", eventsSent,
					"last_event", lastEvent,
					"terminal_event_seen", terminalEventSeen,
				)
				return nil
			}
			lastEvent = "keepalive"
			flusher.Flush()
		case event, ok := <-stream.Events:
			if !ok {
				if !terminalEventSeen {
					streamLog.Warn(
						"chat stream terminated before completion",
						"duration", time.Since(streamStartedAt).String(),
						"events_sent", eventsSent,
						"last_event", lastEvent,
						"terminal_event_seen", terminalEventSeen,
					)
				}
				return nil
			}
			if err := writeSSEFrame(response, event.Event, event.Payload); err != nil {
				streamLog.Warn(
					"chat stream event write failed",
					"error", err,
					"duration", time.Since(streamStartedAt).String(),
					"events_sent", eventsSent,
					"last_event", lastEvent,
					"event", event.Event,
					"terminal_event_seen", terminalEventSeen,
				)
				return nil
			}
			eventsSent++
			lastEvent = event.Event
			if event.Event == "done" || event.Event == "error" {
				terminalEventSeen = true
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleDeleteChat(c echo.Context) error {
	if s.chatService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "chat service unavailable")
	}

	sessionID, err := chatservice.ParseCloseSessionID(c.Param("sessionId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SESSION_ID", err.Error())
	}

	userID, err := s.currentRequestAIPrincipal(c)
	if err != nil {
		return writeChatUserError(c, err)
	}

	s.chatService.CloseSession(userID, sessionID)
	return c.NoContent(http.StatusNoContent)
}

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

func (s *Server) logChatStartFailure(
	c echo.Context,
	raw chatservice.RawStartInput,
	input chatservice.StartInput,
	userID chatservice.UserID,
	err error,
) {
	if s == nil || s.logger == nil || err == nil {
		return
	}

	log := s.logger.With(
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
		"chat_source", strings.TrimSpace(raw.Source),
		"chat_project_id", input.Context.ProjectID.String(),
		"chat_workflow_id", optionalChatUUIDString(input.Context.WorkflowID),
		"chat_ticket_id", optionalChatUUIDString(input.Context.TicketID),
		"chat_provider_id", optionalChatUUIDString(input.ProviderID),
		"chat_session_id", optionalChatSessionIDString(input.SessionID),
		"chat_user_id", string(userID),
	)
	log.Error("chat start failed", "error", err)
}

func (s *Server) chatStreamLogger(
	c echo.Context,
	input chatservice.StartInput,
	userID chatservice.UserID,
) *slog.Logger {
	if s == nil || s.logger == nil {
		return nil
	}

	return s.logger.With(
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
		"chat_source", string(input.Source),
		"chat_project_id", input.Context.ProjectID.String(),
		"chat_workflow_id", optionalChatUUIDString(input.Context.WorkflowID),
		"chat_ticket_id", optionalChatUUIDString(input.Context.TicketID),
		"chat_provider_id", optionalChatUUIDString(input.ProviderID),
		"chat_session_id", optionalChatSessionIDString(input.SessionID),
		"chat_user_id", string(userID),
	)
}

func (s *Server) chatStreamKeepaliveInterval() time.Duration {
	interval := chatSSEKeepaliveInterval
	if interval <= 0 {
		interval = time.Second
	}
	if s == nil || s.cfg.WriteTimeout <= 0 {
		return interval
	}

	maxInterval := s.cfg.WriteTimeout / 2
	if maxInterval <= 0 {
		return interval
	}
	if interval > maxInterval {
		return maxInterval
	}
	return interval
}

func optionalChatUUIDString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func optionalChatSessionIDString(value *chatservice.SessionID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func (s *Server) currentProjectConversationUserID(c echo.Context) (chatservice.UserID, error) {
	if actor := strings.TrimSpace(actorFromHumanPrincipal(c)); actor != "" {
		return chatservice.ParseUserID(actor)
	}
	if s != nil && s.auth.Mode == config.AuthModeOIDC {
		return "", humanauthservice.ErrUnauthorized
	}
	// Auth-disabled mode is a local-instance fallback, so persistent project
	// conversations use one stable server-defined principal instead of a
	// browser-local random identifier.
	return chatservice.LocalProjectConversationUserID, nil
}

func writeChatUserError(c echo.Context, err error) error {
	if errors.Is(err, humanauthservice.ErrUnauthorized) {
		return writeAPIError(c, http.StatusUnauthorized, "HUMAN_SESSION_REQUIRED", err.Error())
	}
	return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
}

func writeChatError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, chatservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, chatservice.ErrSourceUnsupported):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_SOURCE", err.Error())
	case errors.Is(err, chatservice.ErrSessionProviderMismatch):
		return writeAPIError(c, http.StatusConflict, "CHAT_SESSION_PROVIDER_MISMATCH", err.Error())
	case errors.Is(err, chatservice.ErrSessionTurnLimitReached),
		errors.Is(err, chatservice.ErrSessionBudgetExceeded):
		return writeAPIError(c, http.StatusConflict, "CHAT_SESSION_LIMIT_REACHED", err.Error())
	case errors.Is(err, chatservice.ErrProviderNotFound):
		return writeAPIError(c, http.StatusConflict, "CHAT_PROVIDER_NOT_CONFIGURED", err.Error())
	case errors.Is(err, chatservice.ErrProviderUnavailable):
		return writeAPIError(c, http.StatusConflict, "CHAT_PROVIDER_UNAVAILABLE", err.Error())
	case errors.Is(err, chatservice.ErrProviderUnsupported):
		return writeAPIError(c, http.StatusConflict, "CHAT_PROVIDER_UNSUPPORTED", err.Error())
	case errors.Is(err, chatservice.ErrSessionNotFound):
		return writeAPIError(c, http.StatusNotFound, "CHAT_SESSION_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrTicketNotFound),
		errors.Is(err, workflowservice.ErrWorkflowNotFound),
		errors.Is(err, catalogservice.ErrNotFound):
		return writeAPIError(c, http.StatusNotFound, "CHAT_CONTEXT_NOT_FOUND", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func writeSSEFrame(response *echo.Response, event string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(response, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(response, "data: %s\n\n", body); err != nil {
		return err
	}
	return nil
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
	if err := s.requireHumanPermission(c, humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindProject, ID: request.ProjectID.String()}, humanauthdomain.PermissionTicketComment); err != nil {
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
		humanauthdomain.PermissionTicketRead,
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

	turn, err := s.projectConversationService.StartTurn(
		c.Request().Context(),
		userID,
		conversationID,
		request.Message,
		request.Focus,
	)
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

func writeProjectConversationError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, chatservice.ErrConversationTurnActive):
		return writeAPIError(c, http.StatusConflict, "PROJECT_CONVERSATION_TURN_ALREADY_ACTIVE", err.Error())
	case errors.Is(err, chatservice.ErrConversationInterruptPending):
		return writeAPIError(c, http.StatusConflict, "PROJECT_CONVERSATION_INTERRUPT_PENDING", err.Error())
	case errors.Is(err, chatservice.ErrConversationConflict):
		return writeAPIError(c, http.StatusConflict, "CHAT_CONVERSATION_CONFLICT", err.Error())
	case errors.Is(err, chatservice.ErrConversationNotFound), errors.Is(err, chatservice.ErrPendingInterruptNotFound):
		return writeAPIError(c, http.StatusNotFound, "CHAT_CONVERSATION_NOT_FOUND", err.Error())
	case errors.Is(err, chatservice.ErrConversationRuntimeAbsent):
		return writeAPIError(c, http.StatusConflict, "CHAT_CONVERSATION_RUNTIME_UNAVAILABLE", err.Error())
	case errors.Is(err, catalogservice.ErrNotFound), errors.Is(err, ticketservice.ErrTicketNotFound), errors.Is(err, workflowservice.ErrWorkflowNotFound):
		return writeAPIError(c, http.StatusNotFound, "CHAT_CONTEXT_NOT_FOUND", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func (s *Server) mapProjectConversationResponses(ctx context.Context, items []chatdomain.Conversation) []map[string]any {
	response := make([]map[string]any, 0, len(items))
	for _, item := range items {
		response = append(response, s.mapProjectConversationResponse(ctx, item))
	}
	return response
}

func (s *Server) mapProjectConversationResponse(ctx context.Context, item chatdomain.Conversation) map[string]any {
	response := map[string]any{
		"id":               item.ID.String(),
		"project_id":       item.ProjectID.String(),
		"user_id":          item.UserID,
		"source":           string(item.Source),
		"provider_id":      item.ProviderID.String(),
		"status":           string(item.Status),
		"title":            item.Title.String(),
		"rolling_summary":  item.RollingSummary,
		"last_activity_at": item.LastActivityAt.UTC().Format(time.RFC3339),
		"created_at":       item.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":       item.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if s != nil && !s.catalog.Empty() {
		if providerItem, err := s.catalog.GetAgentProvider(ctx, item.ProviderID); err == nil {
			switch providerItem.AdapterType {
			case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
				response["provider_anchor_kind"] = "thread"
				response["provider_turn_supported"] = true
			case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
				response["provider_anchor_kind"] = "session"
				response["provider_turn_supported"] = false
			}
		}
	}
	if item.ProviderThreadID != nil {
		response["provider_thread_id"] = *item.ProviderThreadID
		response["provider_anchor_id"] = *item.ProviderThreadID
	}
	if item.LastTurnID != nil {
		response["last_turn_id"] = *item.LastTurnID
		response["provider_turn_id"] = *item.LastTurnID
	}
	if item.ProviderThreadStatus != nil {
		response["provider_thread_status"] = *item.ProviderThreadStatus
		response["provider_status"] = *item.ProviderThreadStatus
	}
	if len(item.ProviderThreadActiveFlags) > 0 {
		response["provider_thread_active_flags"] = append([]string(nil), item.ProviderThreadActiveFlags...)
		response["provider_active_flags"] = append([]string(nil), item.ProviderThreadActiveFlags...)
	}
	if s != nil && s.projectConversationService != nil {
		if principal, err := s.projectConversationService.GetPrincipal(ctx, chatservice.UserID(item.UserID), item.ID); err == nil {
			response["runtime_principal"] = map[string]any{
				"id":                      principal.ID.String(),
				"name":                    principal.Name,
				"status":                  string(principal.Status),
				"runtime_state":           string(principal.RuntimeState),
				"current_session_id":      optionalConversationString(principal.CurrentSessionID),
				"current_workspace_path":  optionalConversationString(principal.CurrentWorkspacePath),
				"current_run_id":          optionalConversationUUID(principal.CurrentRunID),
				"last_heartbeat_at":       optionalConversationTime(principal.LastHeartbeatAt),
				"current_step_status":     optionalConversationString(principal.CurrentStepStatus),
				"current_step_summary":    optionalConversationString(principal.CurrentStepSummary),
				"current_step_changed_at": optionalConversationTime(principal.CurrentStepChangedAt),
			}
		}
	}
	return response
}

func mapProjectConversationEntries(items []chatdomain.Entry) []map[string]any {
	response := make([]map[string]any, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectConversationEntry(item))
	}
	return response
}

func mapProjectConversationEntry(item chatdomain.Entry) map[string]any {
	payload := map[string]any{
		"id":              item.ID.String(),
		"conversation_id": item.ConversationID.String(),
		"seq":             item.Seq,
		"kind":            string(item.Kind),
		"payload":         item.Payload,
		"created_at":      item.CreatedAt.UTC().Format(time.RFC3339),
	}
	if item.TurnID != nil {
		payload["turn_id"] = item.TurnID.String()
	}
	return payload
}

func mapProjectConversationWorkspaceDiffResponse(
	item chatservice.ProjectConversationWorkspaceDiff,
) map[string]any {
	repos := make([]map[string]any, 0, len(item.Repos))
	for _, repo := range item.Repos {
		files := make([]map[string]any, 0, len(repo.Files))
		for _, file := range repo.Files {
			files = append(files, map[string]any{
				"path":    file.Path,
				"status":  string(file.Status),
				"added":   file.Added,
				"removed": file.Removed,
			})
		}
		repos = append(repos, map[string]any{
			"name":          repo.Name,
			"path":          repo.Path,
			"branch":        repo.Branch,
			"dirty":         repo.Dirty,
			"files_changed": repo.FilesChanged,
			"added":         repo.Added,
			"removed":       repo.Removed,
			"files":         files,
		})
	}

	return map[string]any{
		"conversation_id": item.ConversationID.String(),
		"workspace_path":  item.WorkspacePath,
		"dirty":           item.Dirty,
		"repos_changed":   item.ReposChanged,
		"files_changed":   item.FilesChanged,
		"added":           item.Added,
		"removed":         item.Removed,
		"repos":           repos,
	}
}

func mapPendingInterruptResponse(item chatdomain.PendingInterrupt) map[string]any {
	payload := map[string]any{
		"id":                  item.ID.String(),
		"conversation_id":     item.ConversationID.String(),
		"turn_id":             item.TurnID.String(),
		"provider_request_id": item.ProviderRequestID,
		"kind":                string(item.Kind),
		"payload":             item.Payload,
		"status":              string(item.Status),
	}
	if item.Decision != nil {
		payload["decision"] = *item.Decision
	}
	if item.ResolvedAt != nil {
		payload["resolved_at"] = item.ResolvedAt.UTC().Format(time.RFC3339)
	}
	return payload
}

func optionalConversationString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalConversationUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return value.String()
}

func optionalConversationTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *Server) executeActionProposalActions(_ context.Context, payload map[string]any, executedBy string) ([]map[string]any, error) {
	rawActions, ok := payload["actions"].([]any)
	if !ok {
		return nil, fmt.Errorf("action proposal actions must be an array")
	}
	results := make([]map[string]any, 0, len(rawActions))
	for index, rawAction := range rawActions {
		action, ok := rawAction.(map[string]any)
		if !ok {
			results = append(results, map[string]any{
				"action_index": index,
				"ok":           false,
				"summary":      "action payload is invalid",
			})
			continue
		}
		method := strings.ToUpper(strings.TrimSpace(httpStringValue(action["method"])))
		path := strings.TrimSpace(httpStringValue(action["path"]))
		body, _ := action["body"].(map[string]any)
		result := map[string]any{
			"action_index": index,
			"action":       action,
		}
		preparedBody, err := prepareProjectConversationActionBody(method, path, body, executedBy)
		if err != nil {
			result["ok"] = false
			result["summary"] = err.Error()
			results = append(results, result)
			continue
		}
		status, responseBody, err := s.executeInternalAPIAction(method, path, preparedBody)
		result["status_code"] = status
		if responseBody != "" {
			result["detail"] = responseBody
		}
		if err != nil || status < 200 || status >= 300 {
			result["ok"] = false
			result["summary"] = fmt.Sprintf("%s %s failed.", method, path)
		} else {
			result["ok"] = true
			result["summary"] = fmt.Sprintf("%s %s succeeded.", method, path)
		}
		results = append(results, result)
	}
	return results, nil
}

func httpStringValue(value any) string {
	typed, _ := value.(string)
	return typed
}

func (s *Server) executeInternalAPIAction(method string, path string, body map[string]any) (int, string, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, "", err
		}
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String()), nil
}

func projectConversationConfirmedActionActor(userID chatservice.UserID, conversationID uuid.UUID) string {
	return fmt.Sprintf("user:%s via project-conversation:%s", strings.TrimSpace(string(userID)), conversationID.String())
}

func prepareProjectConversationActionBody(method string, path string, body map[string]any, executedBy string) (map[string]any, error) {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	normalizedPath := strings.TrimSpace(path)
	if normalizedMethod == "" || normalizedPath == "" {
		return nil, fmt.Errorf("action proposal requires method and path")
	}
	prepared := cloneHTTPActionBody(body)
	switch {
	case normalizedMethod == http.MethodPost && projectConversationTicketCreatePath(normalizedPath):
		prepared["created_by"] = executedBy
	case normalizedMethod == http.MethodPatch && projectConversationTicketPatchPath(normalizedPath):
		prepared["created_by"] = executedBy
	case normalizedMethod == http.MethodPost && projectConversationTicketCommentCreatePath(normalizedPath):
		prepared["created_by"] = executedBy
	case normalizedMethod == http.MethodPatch && projectConversationTicketCommentPatchPath(normalizedPath):
		prepared["edited_by"] = executedBy
	case normalizedMethod == http.MethodPost && projectConversationUpdateThreadCreatePath(normalizedPath):
		prepared["created_by"] = executedBy
	case normalizedMethod == http.MethodPatch && projectConversationUpdateThreadPatchPath(normalizedPath):
		prepared["edited_by"] = executedBy
	case normalizedMethod == http.MethodPost && projectConversationUpdateCommentCreatePath(normalizedPath):
		prepared["created_by"] = executedBy
	case normalizedMethod == http.MethodPatch && projectConversationUpdateCommentPatchPath(normalizedPath):
		prepared["edited_by"] = executedBy
	case normalizedMethod == http.MethodPost && projectConversationWorkflowCreatePath(normalizedPath):
		prepared["created_by"] = executedBy
	case normalizedMethod == http.MethodPatch && projectConversationWorkflowPatchPath(normalizedPath):
		prepared["edited_by"] = executedBy
	case normalizedMethod == http.MethodPut && projectConversationWorkflowHarnessUpdatePath(normalizedPath):
		prepared["edited_by"] = executedBy
	case normalizedMethod == http.MethodGet:
		return prepared, nil
	default:
		return nil, fmt.Errorf("%s %s is not allowed because its audit actor is not explicit for project conversation confirmation", normalizedMethod, normalizedPath)
	}
	return prepared, nil
}

func cloneHTTPActionBody(body map[string]any) map[string]any {
	if len(body) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(body))
	for key, value := range body {
		cloned[key] = value
	}
	return cloned
}

func projectConversationTicketCreatePath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/tickets")
}

func projectConversationTicketPatchPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/tickets/") && !strings.Contains(path, "/comments")
}

func projectConversationTicketCommentCreatePath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/tickets/") && strings.HasSuffix(path, "/comments")
}

func projectConversationTicketCommentPatchPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/tickets/") && strings.Contains(path, "/comments/")
}

func projectConversationWorkflowCreatePath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/workflows")
}

func projectConversationWorkflowPatchPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/workflows/") && !strings.HasSuffix(path, "/harness")
}

func projectConversationWorkflowHarnessUpdatePath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/workflows/") && strings.HasSuffix(path, "/harness")
}

func projectConversationUpdateThreadCreatePath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/updates")
}

func projectConversationUpdateThreadPatchPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/updates/") && !strings.Contains(path, "/comments/")
}

func projectConversationUpdateCommentCreatePath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/updates/") && strings.HasSuffix(path, "/comments")
}

func projectConversationUpdateCommentPatchPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/updates/") && strings.Contains(path, "/comments/")
}
