package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var agentOutputStreamTopic = provider.MustParseTopic("agent.output.events")

type agentOutputEntryResponse struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	AgentID   string  `json:"agent_id"`
	TicketID  *string `json:"ticket_id,omitempty"`
	Stream    string  `json:"stream"`
	Output    string  `json:"output"`
	CreatedAt string  `json:"created_at"`
}

func (s *Server) listAgentOutput(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	input, err := domain.ParseListAgentOutput(projectID, agentID, domain.AgentOutputListInput{
		TicketID: c.QueryParam("ticket_id"),
		Limit:    c.QueryParam("limit"),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	items, err := s.catalog.ListAgentOutput(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"entries": mapAgentOutputResponses(items),
	})
}

func (s *Server) streamAgentOutput(c echo.Context) error {
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

	stream, err := s.sseHub.Register(c.Request().Context(), activityStreamTopic)
	if err != nil {
		return fmt.Errorf("register agent output stream: %w", err)
	}

	response := c.Response()
	header := response.Header()
	header.Set(echo.HeaderContentType, "text/event-stream")
	header.Set(echo.HeaderCacheControl, "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	if err := writeSSEComment(response, "keepalive"); err != nil {
		return err
	}

	heartbeat := time.NewTicker(sseKeepaliveInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-stream:
			if !ok {
				return nil
			}

			outputEvent, matched, err := buildAgentOutputStreamEvent(projectID, agentID, ticketID, event)
			if err != nil {
				s.logger.Warn("skip malformed agent output stream event", "error", err)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, outputEvent); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := writeSSEComment(response, "keepalive"); err != nil {
				return err
			}
		}
	}
}

func mapAgentOutputResponses(items []domain.AgentOutputEntry) []agentOutputEntryResponse {
	response := make([]agentOutputEntryResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapAgentOutputResponse(item))
	}

	return response
}

func mapAgentOutputResponse(item domain.AgentOutputEntry) agentOutputEntryResponse {
	return agentOutputEntryResponse{
		ID:        item.ID.String(),
		ProjectID: item.ProjectID.String(),
		AgentID:   item.AgentID.String(),
		TicketID:  uuidToStringPointer(item.TicketID),
		Stream:    item.Stream,
		Output:    item.Output,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func buildAgentOutputStreamEvent(
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID *uuid.UUID,
	event provider.Event,
) (provider.Event, bool, error) {
	if event.Type.String() != domain.AgentOutputEventType {
		return provider.Event{}, false, nil
	}

	var payload struct {
		Event activityEventResponse `json:"event"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode activity stream payload: %w", err)
	}
	if payload.Event.ProjectID != projectID.String() {
		return provider.Event{}, false, nil
	}
	if payload.Event.AgentID == nil || *payload.Event.AgentID != agentID.String() {
		return provider.Event{}, false, nil
	}
	if ticketID != nil {
		if payload.Event.TicketID == nil || *payload.Event.TicketID != ticketID.String() {
			return provider.Event{}, false, nil
		}
	}

	outputEvent, err := provider.NewJSONEvent(
		agentOutputStreamTopic,
		event.Type,
		map[string]any{
			"entry": agentOutputEntryResponse{
				ID:        payload.Event.ID,
				ProjectID: payload.Event.ProjectID,
				AgentID:   *payload.Event.AgentID,
				TicketID:  payload.Event.TicketID,
				Stream:    agentOutputMetadataStream(payload.Event.Metadata),
				Output:    payload.Event.Message,
				CreatedAt: payload.Event.CreatedAt,
			},
		},
		event.PublishedAt,
	)
	if err != nil {
		return provider.Event{}, false, fmt.Errorf("construct agent output stream event: %w", err)
	}

	return outputEvent, true, nil
}

func parseOptionalUUIDQueryParam(fieldName string, raw string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", fieldName)
	}

	return &parsed, nil
}

func agentOutputMetadataStream(metadata map[string]any) string {
	rawStream, ok := metadata["stream"].(string)
	if !ok {
		return "runtime"
	}

	stream := strings.TrimSpace(rawStream)
	if stream == "" {
		return "runtime"
	}

	return stream
}
