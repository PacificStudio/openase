package httpapi

import (
	"net/http"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/labstack/echo/v4"
)

type agentOutputEntryResponse struct {
	ID        string         `json:"id"`
	TicketID  *string        `json:"ticket_id,omitempty"`
	EventType string         `json:"event_type"`
	Stream    string         `json:"stream"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
}

func (s *Server) getAgentOutput(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	input, err := domain.ParseGetAgentOutput(agentID, domain.AgentOutputListInput{
		Limit: c.QueryParam("limit"),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	output, err := s.catalog.GetAgentOutput(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent":   mapAgentResponse(output.Agent),
		"entries": mapAgentOutputEntryResponses(output.Entries),
	})
}

func mapAgentOutputEntryResponses(items []domain.AgentOutputEntry) []agentOutputEntryResponse {
	response := make([]agentOutputEntryResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapAgentOutputEntryResponse(item))
	}

	return response
}

func mapAgentOutputEntryResponse(item domain.AgentOutputEntry) agentOutputEntryResponse {
	return agentOutputEntryResponse{
		ID:        item.ID.String(),
		TicketID:  uuidToStringPointer(item.TicketID),
		EventType: item.EventType,
		Stream:    item.Stream,
		Message:   item.Message,
		Metadata:  cloneMap(item.Metadata),
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
	}
}
