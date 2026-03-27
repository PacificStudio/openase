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
	ID                    string         `json:"id"`
	OrganizationID        string         `json:"organization_id"`
	MachineID             string         `json:"machine_id"`
	MachineName           string         `json:"machine_name"`
	MachineHost           string         `json:"machine_host"`
	MachineStatus         string         `json:"machine_status"`
	MachineSSHUser        *string        `json:"machine_ssh_user,omitempty"`
	MachineWorkspaceRoot  *string        `json:"machine_workspace_root,omitempty"`
	Name                  string         `json:"name"`
	AdapterType           string         `json:"adapter_type"`
	AvailabilityState     string         `json:"availability_state"`
	Available             bool           `json:"available"`
	AvailabilityCheckedAt *string        `json:"availability_checked_at,omitempty"`
	AvailabilityReason    *string        `json:"availability_reason,omitempty"`
	CliCommand            string         `json:"cli_command"`
	CliArgs               []string       `json:"cli_args"`
	AuthConfig            map[string]any `json:"auth_config"`
	ModelName             string         `json:"model_name"`
	ModelTemperature      float64        `json:"model_temperature"`
	ModelMaxTokens        int            `json:"model_max_tokens"`
	CostPerInputToken     float64        `json:"cost_per_input_token"`
	CostPerOutputToken    float64        `json:"cost_per_output_token"`
}

type agentResponse struct {
	ID                    string                `json:"id"`
	ProviderID            string                `json:"provider_id"`
	ProjectID             string                `json:"project_id"`
	Name                  string                `json:"name"`
	RuntimeControlState   string                `json:"runtime_control_state"`
	TotalTokensUsed       int64                 `json:"total_tokens_used"`
	TotalTicketsCompleted int                   `json:"total_tickets_completed"`
	Runtime               *agentRuntimeResponse `json:"runtime,omitempty"`
}

type agentRuntimeResponse struct {
	ActiveRunCount       int     `json:"active_run_count"`
	CurrentRunID         *string `json:"current_run_id,omitempty"`
	Status               string  `json:"status"`
	CurrentTicketID      *string `json:"current_ticket_id,omitempty"`
	SessionID            string  `json:"session_id"`
	RuntimePhase         string  `json:"runtime_phase"`
	RuntimeStartedAt     *string `json:"runtime_started_at,omitempty"`
	LastError            string  `json:"last_error"`
	LastHeartbeatAt      *string `json:"last_heartbeat_at,omitempty"`
	CurrentStepStatus    *string `json:"current_step_status,omitempty"`
	CurrentStepSummary   *string `json:"current_step_summary,omitempty"`
	CurrentStepChangedAt *string `json:"current_step_changed_at,omitempty"`
}

type agentRunResponse struct {
	ID               string  `json:"id"`
	AgentID          string  `json:"agent_id"`
	WorkflowID       string  `json:"workflow_id"`
	TicketID         string  `json:"ticket_id"`
	ProviderID       string  `json:"provider_id"`
	Status           string  `json:"status"`
	SessionID        string  `json:"session_id"`
	RuntimeStartedAt *string `json:"runtime_started_at,omitempty"`
	LastError        string  `json:"last_error"`
	LastHeartbeatAt  *string `json:"last_heartbeat_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

type agentProviderPatchRequest struct {
	MachineID          *string         `json:"machine_id"`
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
		MachineID:          current.MachineID.String(),
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
	if patch.MachineID != nil {
		request.MachineID = *patch.MachineID
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

func (s *Server) listAgentRuns(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListAgentRuns(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent_runs": mapAgentRunResponses(items),
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
	availabilityState := item.AvailabilityState
	if !availabilityState.IsValid() {
		availabilityState = domain.AgentProviderAvailabilityStateUnknown
	}

	return agentProviderResponse{
		ID:                    item.ID.String(),
		OrganizationID:        item.OrganizationID.String(),
		MachineID:             item.MachineID.String(),
		MachineName:           item.MachineName,
		MachineHost:           item.MachineHost,
		MachineStatus:         item.MachineStatus.String(),
		MachineSSHUser:        stringPointerValue(item.MachineSSHUser),
		MachineWorkspaceRoot:  stringPointerValue(item.MachineWorkspaceRoot),
		Name:                  item.Name,
		AdapterType:           item.AdapterType.String(),
		AvailabilityState:     availabilityState.String(),
		Available:             item.Available,
		AvailabilityCheckedAt: timePointerString(item.AvailabilityCheckedAt),
		AvailabilityReason:    stringPointerValue(item.AvailabilityReason),
		CliCommand:            item.CliCommand,
		CliArgs:               cloneStringSlice(item.CliArgs),
		AuthConfig:            cloneMap(item.AuthConfig),
		ModelName:             item.ModelName,
		ModelTemperature:      item.ModelTemperature,
		ModelMaxTokens:        item.ModelMaxTokens,
		CostPerInputToken:     item.CostPerInputToken,
		CostPerOutputToken:    item.CostPerOutputToken,
	}
}

func stringPointerValue(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func timePointerString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
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
		RuntimeControlState:   item.RuntimeControlState.String(),
		TotalTokensUsed:       item.TotalTokensUsed,
		TotalTicketsCompleted: item.TotalTicketsCompleted,
		Runtime:               mapAgentRuntimeResponse(item.Runtime),
	}
}

func mapAgentRuntimeResponse(item *domain.AgentRuntime) *agentRuntimeResponse {
	if item == nil {
		return nil
	}

	return &agentRuntimeResponse{
		ActiveRunCount:       item.ActiveRunCount,
		CurrentRunID:         uuidToStringPointer(item.CurrentRunID),
		Status:               item.Status.String(),
		CurrentTicketID:      uuidToStringPointer(item.CurrentTicketID),
		SessionID:            item.SessionID,
		RuntimePhase:         item.RuntimePhase.String(),
		RuntimeStartedAt:     timeToStringPointer(item.RuntimeStartedAt),
		LastError:            item.LastError,
		LastHeartbeatAt:      timeToStringPointer(item.LastHeartbeatAt),
		CurrentStepStatus:    stringPointerValue(item.CurrentStepStatus),
		CurrentStepSummary:   stringPointerValue(item.CurrentStepSummary),
		CurrentStepChangedAt: timeToStringPointer(item.CurrentStepChangedAt),
	}
}

func mapAgentRunResponses(items []domain.AgentRun) []agentRunResponse {
	response := make([]agentRunResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapAgentRunResponse(item))
	}

	return response
}

func mapAgentRunResponse(item domain.AgentRun) agentRunResponse {
	return agentRunResponse{
		ID:               item.ID.String(),
		AgentID:          item.AgentID.String(),
		WorkflowID:       item.WorkflowID.String(),
		TicketID:         item.TicketID.String(),
		ProviderID:       item.ProviderID.String(),
		Status:           item.Status.String(),
		SessionID:        item.SessionID,
		RuntimeStartedAt: timeToStringPointer(item.RuntimeStartedAt),
		LastError:        item.LastError,
		LastHeartbeatAt:  timeToStringPointer(item.LastHeartbeatAt),
		CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
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
