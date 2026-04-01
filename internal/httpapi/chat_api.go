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
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const chatUserHeader = "X-OpenASE-Chat-User"

var chatSSEKeepaliveInterval = 5 * time.Second

func (s *Server) registerChatRoutes(api *echo.Group) {
	api.POST("/chat", s.handleStartChat)
	api.DELETE("/chat/:sessionId", s.handleDeleteChat)
	api.POST("/chat/conversations", s.handleCreateProjectConversation)
	api.GET("/chat/conversations", s.handleListProjectConversations)
	api.GET("/chat/conversations/:conversationId", s.handleGetProjectConversation)
	api.GET("/chat/conversations/:conversationId/entries", s.handleListProjectConversationEntries)
	api.POST("/chat/conversations/:conversationId/turns", s.handleStartProjectConversationTurn)
	api.GET("/chat/conversations/:conversationId/stream", s.handleProjectConversationStream)
	api.POST("/chat/conversations/:conversationId/interrupts/:interruptId/respond", s.handleRespondProjectConversationInterrupt)
	api.POST("/chat/conversations/:conversationId/action-proposals/:entryId/execute", s.handleExecuteProjectConversationActionProposal)
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

	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}

	stream, err := s.chatService.StartTurn(c.Request().Context(), userID, input)
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
		case <-c.Request().Context().Done():
			streamLog.Warn(
				"chat stream request context ended before completion",
				"error", c.Request().Context().Err(),
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

	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}

	s.chatService.CloseSession(userID, sessionID)
	return c.NoContent(http.StatusNoContent)
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
	case errors.Is(err, chatservice.ErrProviderUnavailable),
		errors.Is(err, chatservice.ErrProviderUnsupported):
		return writeAPIError(c, http.StatusConflict, "CHAT_PROVIDER_UNAVAILABLE", err.Error())
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
	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
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
	return c.JSON(http.StatusCreated, map[string]any{"conversation": mapProjectConversationResponse(conversation)})
}

func (s *Server) handleListProjectConversations(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}

	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}
	projectID, err := parseUUIDString("project_id", c.QueryParam("project_id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
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
	return c.JSON(http.StatusOK, map[string]any{"conversations": mapProjectConversationResponses(items)})
}

func (s *Server) handleGetProjectConversation(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}
	item, err := s.projectConversationService.GetConversation(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"conversation": mapProjectConversationResponse(item)})
}

func (s *Server) handleListProjectConversationEntries(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}
	items, err := s.projectConversationService.ListEntries(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"entries": mapProjectConversationEntries(items)})
}

func (s *Server) handleStartProjectConversationTurn(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}

	var raw rawConversationTurnRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	message, err := parseProjectConversationTurnRequest(raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	turn, err := s.projectConversationService.StartTurn(c.Request().Context(), userID, conversationID, message)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusAccepted, map[string]any{
		"turn": map[string]any{
			"id":         turn.ID.String(),
			"turn_index": turn.TurnIndex,
			"status":     string(turn.Status),
		},
	})
}

func (s *Server) handleProjectConversationStream(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	events, cleanup := s.projectConversationService.WatchConversation(c.Request().Context(), conversationID)
	defer cleanup()

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
		case <-c.Request().Context().Done():
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
	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
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

func (s *Server) handleExecuteProjectConversationActionProposal(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	entryID, err := parseUUIDString("entry_id", c.Param("entryId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ENTRY_ID", err.Error())
	}
	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}
	entries, err := s.projectConversationService.ListEntries(c.Request().Context(), userID, conversationID)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	var proposalEntry *chatdomain.Entry
	for i := range entries {
		if entries[i].ID == entryID {
			proposalEntry = &entries[i]
			break
		}
	}
	if proposalEntry == nil || proposalEntry.Kind != chatdomain.EntryKindActionProposal {
		return writeAPIError(c, http.StatusNotFound, "CHAT_ACTION_PROPOSAL_NOT_FOUND", "chat action proposal entry not found")
	}

	results, err := s.executeActionProposalActions(c.Request().Context(), proposalEntry.Payload)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	resultPayload := map[string]any{
		"entry_id": entryID.String(),
		"results":  results,
	}
	entry, err := s.projectConversationService.AppendActionExecutionResult(c.Request().Context(), userID, conversationID, proposalEntry.TurnID, resultPayload)
	if err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"result_entry": mapProjectConversationEntry(entry),
		"results":      results,
	})
}

func (s *Server) handleDeleteProjectConversationRuntime(c echo.Context) error {
	if s.projectConversationService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "project conversation service unavailable")
	}
	conversationID, err := parseUUIDString("conversation_id", c.Param("conversationId"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CONVERSATION_ID", err.Error())
	}
	userID, err := chatservice.ParseRequestUserID(c.Request().Header.Get(chatUserHeader))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CHAT_USER", err.Error())
	}
	if err := s.projectConversationService.CloseRuntime(c.Request().Context(), userID, conversationID); err != nil {
		return writeProjectConversationError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func writeProjectConversationError(c echo.Context, err error) error {
	switch {
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

func mapProjectConversationResponses(items []chatdomain.Conversation) []map[string]any {
	response := make([]map[string]any, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectConversationResponse(item))
	}
	return response
}

func mapProjectConversationResponse(item chatdomain.Conversation) map[string]any {
	return map[string]any{
		"id":               item.ID.String(),
		"project_id":       item.ProjectID.String(),
		"user_id":          item.UserID,
		"source":           string(item.Source),
		"provider_id":      item.ProviderID.String(),
		"status":           string(item.Status),
		"rolling_summary":  item.RollingSummary,
		"last_activity_at": item.LastActivityAt.UTC().Format(time.RFC3339),
		"created_at":       item.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":       item.UpdatedAt.UTC().Format(time.RFC3339),
	}
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

func (s *Server) executeActionProposalActions(_ context.Context, payload map[string]any) ([]map[string]any, error) {
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
		status, responseBody, err := s.executeInternalAPIAction(method, path, body)
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
