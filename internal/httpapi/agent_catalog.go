package httpapi

import (
	"errors"
	"net/http"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/labstack/echo/v4"
)

type agentProviderResponse struct {
	ID                 string         `json:"id"`
	OrganizationID     string         `json:"organization_id"`
	Name               string         `json:"name"`
	AdapterType        string         `json:"adapter_type"`
	Available          bool           `json:"available"`
	CliCommand         string         `json:"cli_command"`
	CliArgs            []string       `json:"cli_args"`
	AuthConfig         map[string]any `json:"auth_config"`
	ModelName          string         `json:"model_name"`
	ModelTemperature   float64        `json:"model_temperature"`
	ModelMaxTokens     int            `json:"model_max_tokens"`
	CostPerInputToken  float64        `json:"cost_per_input_token"`
	CostPerOutputToken float64        `json:"cost_per_output_token"`
}

type agentResponse struct {
	ID                    string  `json:"id"`
	ProviderID            string  `json:"provider_id"`
	ProjectID             string  `json:"project_id"`
	Name                  string  `json:"name"`
	Status                string  `json:"status"`
	CurrentTicketID       *string `json:"current_ticket_id,omitempty"`
	SessionID             string  `json:"session_id"`
	RuntimePhase          string  `json:"runtime_phase"`
	RuntimeControlState   string  `json:"runtime_control_state"`
	RuntimeStartedAt      *string `json:"runtime_started_at,omitempty"`
	LastError             string  `json:"last_error"`
	WorkspacePath         string  `json:"workspace_path"`
	TotalTokensUsed       int64   `json:"total_tokens_used"`
	TotalTicketsCompleted int     `json:"total_tickets_completed"`
	LastHeartbeatAt       *string `json:"last_heartbeat_at,omitempty"`
}

type agentProviderPatchRequest struct {
	Name               *string         `json:"name"`
	AdapterType        *string         `json:"adapter_type"`
	CliCommand         *string         `json:"cli_command"`
	CliArgs            *[]string       `json:"cli_args"`
	AuthConfig         *map[string]any `json:"auth_config"`
	ModelName          *string         `json:"model_name"`
	ModelTemperature   *float64        `json:"model_temperature"`
	ModelMaxTokens     *int            `json:"model_max_tokens"`
	CostPerInputToken  *float64        `json:"cost_per_input_token"`
	CostPerOutputToken *float64        `json:"cost_per_output_token"`
}

func (s *Server) listAgentProviders(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListAgentProviders(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"providers": mapAgentProviderResponses(items),
	})
}

func (s *Server) createAgentProvider(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	var request domain.AgentProviderInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateAgentProvider(orgID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateAgentProvider(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"provider": mapAgentProviderResponse(item),
	})
}

func (s *Server) patchAgentProvider(c echo.Context) error {
	providerID, err := parseUUIDPathParam(c, "providerId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetAgentProvider(c.Request().Context(), providerID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch agentProviderPatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	request := domain.AgentProviderInput{
		Name:               current.Name,
		AdapterType:        current.AdapterType.String(),
		CliCommand:         current.CliCommand,
		CliArgs:            append([]string(nil), current.CliArgs...),
		AuthConfig:         cloneMap(current.AuthConfig),
		ModelName:          current.ModelName,
		ModelTemperature:   floatPointer(current.ModelTemperature),
		ModelMaxTokens:     intPointer(current.ModelMaxTokens),
		CostPerInputToken:  floatPointer(current.CostPerInputToken),
		CostPerOutputToken: floatPointer(current.CostPerOutputToken),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.AdapterType != nil {
		request.AdapterType = *patch.AdapterType
	}
	if patch.CliCommand != nil {
		request.CliCommand = *patch.CliCommand
	}
	if patch.CliArgs != nil {
		request.CliArgs = append([]string(nil), (*patch.CliArgs)...)
	}
	if patch.AuthConfig != nil {
		request.AuthConfig = cloneMap(*patch.AuthConfig)
	}
	if patch.ModelName != nil {
		request.ModelName = *patch.ModelName
	}
	if patch.ModelTemperature != nil {
		request.ModelTemperature = patch.ModelTemperature
	}
	if patch.ModelMaxTokens != nil {
		request.ModelMaxTokens = patch.ModelMaxTokens
	}
	if patch.CostPerInputToken != nil {
		request.CostPerInputToken = patch.CostPerInputToken
	}
	if patch.CostPerOutputToken != nil {
		request.CostPerOutputToken = patch.CostPerOutputToken
	}

	input, err := domain.ParseUpdateAgentProvider(providerID, current.OrganizationID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateAgentProvider(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"provider": mapAgentProviderResponse(item),
	})
}

func (s *Server) listAgents(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListAgents(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agents": mapAgentResponses(items),
	})
}

func (s *Server) createAgent(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	var request domain.AgentInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateAgent(projectID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateAgent(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func (s *Server) getAgent(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetAgent(c.Request().Context(), agentID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func (s *Server) pauseAgent(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	item, err := s.catalog.RequestAgentPause(c.Request().Context(), agentID)
	if err != nil {
		if errors.Is(err, catalogservice.ErrNotFound) {
			return writeCatalogError(c, err)
		}
		if errors.Is(err, catalogservice.ErrConflict) {
			return writeAPIError(
				c,
				http.StatusConflict,
				"AGENT_RUNTIME_CONTROL_CONFLICT",
				catalogConflictMessage(err),
			)
		}
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func (s *Server) resumeAgent(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	item, err := s.catalog.RequestAgentResume(c.Request().Context(), agentID)
	if err != nil {
		if errors.Is(err, catalogservice.ErrNotFound) {
			return writeCatalogError(c, err)
		}
		if errors.Is(err, catalogservice.ErrConflict) {
			return writeAPIError(
				c,
				http.StatusConflict,
				"AGENT_RUNTIME_CONTROL_CONFLICT",
				catalogConflictMessage(err),
			)
		}
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func (s *Server) deleteAgent(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	item, err := s.catalog.DeleteAgent(c.Request().Context(), agentID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func mapAgentProviderResponses(items []domain.AgentProvider) []agentProviderResponse {
	response := make([]agentProviderResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapAgentProviderResponse(item))
	}

	return response
}

func mapAgentProviderResponse(item domain.AgentProvider) agentProviderResponse {
	return agentProviderResponse{
		ID:                 item.ID.String(),
		OrganizationID:     item.OrganizationID.String(),
		Name:               item.Name,
		AdapterType:        item.AdapterType.String(),
		Available:          item.Available,
		CliCommand:         item.CliCommand,
		CliArgs:            cloneStringSlice(item.CliArgs),
		AuthConfig:         cloneMap(item.AuthConfig),
		ModelName:          item.ModelName,
		ModelTemperature:   item.ModelTemperature,
		ModelMaxTokens:     item.ModelMaxTokens,
		CostPerInputToken:  item.CostPerInputToken,
		CostPerOutputToken: item.CostPerOutputToken,
	}
}

func mapAgentResponses(items []domain.Agent) []agentResponse {
	response := make([]agentResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapAgentResponse(item))
	}

	return response
}

func mapAgentResponse(item domain.Agent) agentResponse {
	return agentResponse{
		ID:                    item.ID.String(),
		ProviderID:            item.ProviderID.String(),
		ProjectID:             item.ProjectID.String(),
		Name:                  item.Name,
		Status:                item.Status.String(),
		CurrentTicketID:       uuidToStringPointer(item.CurrentTicketID),
		SessionID:             item.SessionID,
		RuntimePhase:          item.RuntimePhase.String(),
		RuntimeControlState:   item.RuntimeControlState.String(),
		RuntimeStartedAt:      timeToStringPointer(item.RuntimeStartedAt),
		LastError:             item.LastError,
		WorkspacePath:         item.WorkspacePath,
		TotalTokensUsed:       item.TotalTokensUsed,
		TotalTicketsCompleted: item.TotalTicketsCompleted,
		LastHeartbeatAt:       timeToStringPointer(item.LastHeartbeatAt),
	}
}

func timeToStringPointer(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func floatPointer(value float64) *float64 {
	copied := value
	return &copied
}

func cloneMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}

	return cloned
}

func cloneStringSlice(raw []string) []string {
	if len(raw) == 0 {
		return []string{}
	}

	return append([]string{}, raw...)
}
