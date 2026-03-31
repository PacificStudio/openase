package httpapi

import (
	"net/http"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/labstack/echo/v4"
)

type activityEventResponse struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	TicketID  *string        `json:"ticket_id,omitempty"`
	AgentID   *string        `json:"agent_id,omitempty"`
	EventType string         `json:"event_type"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
}

func (s *Server) listActivityEvents(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	input, err := domain.ParseListActivityEvents(projectID, domain.ActivityEventListInput{
		AgentID:  c.QueryParam("agent_id"),
		TicketID: c.QueryParam("ticket_id"),
		Limit:    c.QueryParam("limit"),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	items, err := s.catalog.ListActivityEvents(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"events": mapActivityEventResponses(items),
	})
}

func mapActivityEventResponses(items []domain.ActivityEvent) []activityEventResponse {
	response := make([]activityEventResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapActivityEventResponse(item))
	}

	return response
}

func mapActivityEventResponse(item domain.ActivityEvent) activityEventResponse {
	metadata := cloneMap(item.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	if item.UnknownEventTypeRaw != "" {
		metadata["unknown_event_type_raw"] = item.UnknownEventTypeRaw
	}

	return activityEventResponse{
		ID:        item.ID.String(),
		ProjectID: item.ProjectID.String(),
		TicketID:  uuidToStringPointer(item.TicketID),
		AgentID:   uuidToStringPointer(item.AgentID),
		EventType: item.EventType.String(),
		Message:   item.Message,
		Metadata:  metadata,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
	}
}
