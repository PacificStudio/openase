package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var agentStepStreamTopic = provider.MustParseTopic("agent.step.events")

type agentStepEntryResponse struct {
	ID                 string  `json:"id"`
	ProjectID          string  `json:"project_id"`
	AgentID            string  `json:"agent_id"`
	TicketID           *string `json:"ticket_id,omitempty"`
	AgentRunID         string  `json:"agent_run_id"`
	StepStatus         string  `json:"step_status"`
	Summary            string  `json:"summary"`
	SourceTraceEventID *string `json:"source_trace_event_id,omitempty"`
	CreatedAt          string  `json:"created_at"`
}

func (s *Server) listAgentSteps(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	input, err := domain.ParseListAgentSteps(projectID, agentID, domain.AgentEventListInput{
		TicketID: c.QueryParam("ticket_id"),
		Limit:    c.QueryParam("limit"),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	items, err := s.catalog.ListAgentSteps(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"entries": mapAgentStepResponses(items),
	})
}

func (s *Server) streamAgentSteps(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}
	ticketID, err := parseOptionalUUIDQueryParam("ticket_id", c.QueryParam("ticket_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	agentItem, err := s.catalog.GetAgent(c.Request().Context(), agentID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if agentItem.ProjectID != projectID {
		return writeCatalogError(c, catalogservice.ErrNotFound)
	}

	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("disable sse write deadline: %w", err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	stream, err := s.sseHub.Register(streamCtx, agentStepStreamTopic)
	if err != nil {
		return fmt.Errorf("register agent step stream: %w", err)
	}

	response := c.Response()
	header := response.Header()
	header.Set(echo.HeaderContentType, "text/event-stream")
	header.Set(echo.HeaderCacheControl, "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	if err := writeSSEKeepaliveComment(response); err != nil {
		return err
	}

	heartbeat := time.NewTicker(sseKeepaliveInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-streamCtx.Done():
			return nil
		case event, ok := <-stream:
			if !ok {
				return nil
			}

			stepEvent, matched, err := buildAgentStepStreamEvent(projectID, agentID, ticketID, event)
			if err != nil {
				s.logger.Warn(
					"skip malformed agent step stream event",
					"operation", "build_agent_step_stream_event",
					"project_id", projectID,
					"agent_id", agentID,
					"ticket_id", ticketID,
					"topic", event.Topic.String(),
					"type", event.Type.String(),
					"payload_bytes", len(event.Payload),
					"error", err,
				)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, stepEvent); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := writeSSEKeepaliveComment(response); err != nil {
				return err
			}
		}
	}
}

func mapAgentStepResponses(items []domain.AgentStepEntry) []agentStepEntryResponse {
	response := make([]agentStepEntryResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapAgentStepResponse(item))
	}

	return response
}

func mapAgentStepResponse(item domain.AgentStepEntry) agentStepEntryResponse {
	return agentStepEntryResponse{
		ID:                 item.ID.String(),
		ProjectID:          item.ProjectID.String(),
		AgentID:            item.AgentID.String(),
		TicketID:           uuidToStringPointer(item.TicketID),
		AgentRunID:         item.AgentRunID.String(),
		StepStatus:         item.StepStatus,
		Summary:            item.Summary,
		SourceTraceEventID: uuidToStringPointer(item.SourceTraceEventID),
		CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func buildAgentStepStreamEvent(
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID *uuid.UUID,
	event provider.Event,
) (provider.Event, bool, error) {
	if event.Type.String() != domain.AgentStepEventType {
		return provider.Event{}, false, nil
	}

	var payload struct {
		Entry agentStepEntryResponse `json:"entry"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode step stream payload: %w", err)
	}
	if payload.Entry.ProjectID != projectID.String() || payload.Entry.AgentID != agentID.String() {
		return provider.Event{}, false, nil
	}
	if ticketID != nil {
		if payload.Entry.TicketID == nil || *payload.Entry.TicketID != ticketID.String() {
			return provider.Event{}, false, nil
		}
	}

	stepEvent, err := provider.NewJSONEvent(
		agentStepStreamTopic,
		event.Type,
		map[string]any{
			"entry": payload.Entry,
		},
		event.PublishedAt,
	)
	if err != nil {
		return provider.Event{}, false, fmt.Errorf("construct agent step stream event: %w", err)
	}

	return stepEvent, true, nil
}
