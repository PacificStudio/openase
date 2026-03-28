package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
)

const chatUserHeader = "X-OpenASE-Chat-User"

func (s *Server) registerChatRoutes(api *echo.Group) {
	api.POST("/chat", s.handleStartChat)
	api.DELETE("/chat/:sessionId", s.handleDeleteChat)
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
		return writeChatError(c, err)
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

	for event := range stream.Events {
		if err := writeSSEFrame(response, event.Event, event.Payload); err != nil {
			return nil
		}
		flusher.Flush()
	}

	return nil
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
