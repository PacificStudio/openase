package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type agentProviderResponse struct {
	ID                    string                             `json:"id"`
	OrganizationID        string                             `json:"organization_id"`
	MachineID             string                             `json:"machine_id"`
	MachineName           string                             `json:"machine_name"`
	MachineHost           string                             `json:"machine_host"`
	MachineStatus         string                             `json:"machine_status"`
	MachineSSHUser        *string                            `json:"machine_ssh_user,omitempty"`
	MachineWorkspaceRoot  *string                            `json:"machine_workspace_root,omitempty"`
	Name                  string                             `json:"name"`
	AdapterType           string                             `json:"adapter_type"`
	PermissionProfile     string                             `json:"permission_profile"`
	AvailabilityState     string                             `json:"availability_state"`
	Available             bool                               `json:"available"`
	AvailabilityCheckedAt *string                            `json:"availability_checked_at,omitempty"`
	AvailabilityReason    *string                            `json:"availability_reason,omitempty"`
	Capabilities          agentProviderCapabilitiesResponse  `json:"capabilities"`
	CliCommand            string                             `json:"cli_command"`
	CliArgs               []string                           `json:"cli_args"`
	AuthConfig            map[string]any                     `json:"auth_config"`
	CLIRateLimit          *agentProviderCLIRateLimitResponse `json:"cli_rate_limit,omitempty"`
	CLIRateLimitUpdatedAt *string                            `json:"cli_rate_limit_updated_at,omitempty"`
	ModelName             string                             `json:"model_name"`
	ModelTemperature      float64                            `json:"model_temperature"`
	ModelMaxTokens        int                                `json:"model_max_tokens"`
	MaxParallelRuns       int                                `json:"max_parallel_runs"`
	CostPerInputToken     float64                            `json:"cost_per_input_token"`
	CostPerOutputToken    float64                            `json:"cost_per_output_token"`
	PricingConfig         pricing.ProviderModelPricingConfig `json:"pricing_config"`
}

type agentProviderCapabilitiesResponse struct {
	EphemeralChat agentProviderCapabilityResponse `json:"ephemeral_chat"`
}

type agentProviderCapabilityResponse struct {
	State  string  `json:"state"`
	Reason *string `json:"reason,omitempty"`
}

type agentProviderCLIRateLimitResponse struct {
	Provider   string                                    `json:"provider"`
	ClaudeCode *agentProviderClaudeCodeRateLimitResponse `json:"claude_code,omitempty"`
	Codex      *agentProviderCodexRateLimitResponse      `json:"codex,omitempty"`
	Gemini     *agentProviderGeminiRateLimitResponse     `json:"gemini,omitempty"`
	Raw        map[string]any                            `json:"raw,omitempty"`
}

type agentProviderClaudeCodeRateLimitResponse struct {
	Status                string   `json:"status,omitempty"`
	RateLimitType         string   `json:"rate_limit_type,omitempty"`
	ResetsAt              *string  `json:"resets_at,omitempty"`
	Utilization           *float64 `json:"utilization,omitempty"`
	SurpassedThreshold    *float64 `json:"surpassed_threshold,omitempty"`
	OverageStatus         string   `json:"overage_status,omitempty"`
	OverageDisabledReason string   `json:"overage_disabled_reason,omitempty"`
	IsUsingOverage        *bool    `json:"is_using_overage,omitempty"`
}

type agentProviderCodexRateLimitResponse struct {
	LimitID   string                                     `json:"limit_id,omitempty"`
	LimitName string                                     `json:"limit_name,omitempty"`
	Primary   *agentProviderCodexRateLimitWindowResponse `json:"primary,omitempty"`
	Secondary *agentProviderCodexRateLimitWindowResponse `json:"secondary,omitempty"`
	PlanType  string                                     `json:"plan_type,omitempty"`
}

type agentProviderCodexRateLimitWindowResponse struct {
	UsedPercent   *float64 `json:"used_percent,omitempty"`
	WindowMinutes int64    `json:"window_minutes,omitempty"`
	ResetsAt      *string  `json:"resets_at,omitempty"`
}

type agentProviderGeminiRateLimitResponse struct {
	AuthType  string                                       `json:"auth_type,omitempty"`
	Remaining *int64                                       `json:"remaining,omitempty"`
	Limit     *int64                                       `json:"limit,omitempty"`
	ResetTime *string                                      `json:"reset_time,omitempty"`
	Buckets   []agentProviderGeminiRateLimitBucketResponse `json:"buckets,omitempty"`
}

type agentProviderGeminiRateLimitBucketResponse struct {
	ModelID           string   `json:"model_id,omitempty"`
	TokenType         string   `json:"token_type,omitempty"`
	RemainingAmount   string   `json:"remaining_amount,omitempty"`
	RemainingFraction *float64 `json:"remaining_fraction,omitempty"`
	ResetTime         *string  `json:"reset_time,omitempty"`
}

type agentProviderModelOptionResponse struct {
	ID            string                              `json:"id"`
	Label         string                              `json:"label"`
	Description   string                              `json:"description"`
	Recommended   bool                                `json:"recommended"`
	Preview       bool                                `json:"preview"`
	PricingConfig *pricing.ProviderModelPricingConfig `json:"pricing_config,omitempty"`
}

type agentProviderModelCatalogEntryResponse struct {
	AdapterType string                             `json:"adapter_type"`
	Options     []agentProviderModelOptionResponse `json:"options"`
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
	ID                string   `json:"id"`
	AgentID           string   `json:"agent_id"`
	WorkflowID        string   `json:"workflow_id"`
	WorkflowVersionID *string  `json:"workflow_version_id,omitempty"`
	TicketID          string   `json:"ticket_id"`
	ProviderID        string   `json:"provider_id"`
	SkillVersionIDs   []string `json:"skill_version_ids"`
	Status            string   `json:"status"`
	SessionID         string   `json:"session_id"`
	RuntimeStartedAt  *string  `json:"runtime_started_at,omitempty"`
	TerminalAt        *string  `json:"terminal_at,omitempty"`
	LastError         string   `json:"last_error"`
	LastHeartbeatAt   *string  `json:"last_heartbeat_at,omitempty"`
	CreatedAt         string   `json:"created_at"`
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

func (s *Server) listProviderModelOptions(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"adapter_model_options": mapAgentProviderModelCatalogResponses(),
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
	if err := s.publishProviderLifecycleEvent(c.Request().Context(), providerCreatedEventType, item); err != nil {
		return err
	}
	if err := s.emitProviderActivityForAffectedProjects(
		c.Request().Context(),
		item.OrganizationID,
		item.ID,
		func(projectID uuid.UUID) activitysvc.RecordInput {
			return activitysvc.RecordInput{
				ProjectID: projectID,
				EventType: activityevent.TypeProviderCreated,
				Message:   "Created provider " + item.Name,
				Metadata: map[string]any{
					"provider_id":    item.ID.String(),
					"provider_name":  item.Name,
					"machine_id":     item.MachineID.String(),
					"availability":   item.AvailabilityState.String(),
					"changed_fields": []string{"provider"},
				},
			}
		},
	); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"provider": mapAgentProviderResponse(item),
	})
}

func (s *Server) getAgentProvider(c echo.Context) error {
	providerID, err := parseUUIDPathParam(c, "providerId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetAgentProvider(c.Request().Context(), providerID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
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

	input, err := parseAgentProviderPatchRequest(providerID, current, patch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateAgentProvider(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.publishProviderLifecycleEvent(c.Request().Context(), providerUpdatedEventType, item); err != nil {
		return err
	}
	if current.MachineID != item.MachineID {
		if err := s.emitProviderActivityForAffectedProjects(
			c.Request().Context(),
			item.OrganizationID,
			item.ID,
			func(projectID uuid.UUID) activitysvc.RecordInput {
				return activitysvc.RecordInput{
					ProjectID: projectID,
					EventType: activityevent.TypeProviderMachineBindingChanged,
					Message:   "Changed provider machine binding for " + item.Name,
					Metadata: map[string]any{
						"provider_id":     item.ID.String(),
						"provider_name":   item.Name,
						"from_machine_id": current.MachineID.String(),
						"to_machine_id":   item.MachineID.String(),
						"changed_fields":  []string{"machine_id"},
					},
				}
			},
		); err != nil {
			return writeCatalogError(c, err)
		}
	}
	changedFields := providerChangedFields(current, item)
	if len(changedFields) > 0 {
		if err := s.emitProviderActivityForAffectedProjects(
			c.Request().Context(),
			item.OrganizationID,
			item.ID,
			func(projectID uuid.UUID) activitysvc.RecordInput {
				return activitysvc.RecordInput{
					ProjectID: projectID,
					EventType: activityevent.TypeProviderUpdated,
					Message:   "Updated provider " + item.Name,
					Metadata: map[string]any{
						"provider_id":    item.ID.String(),
						"provider_name":  item.Name,
						"availability":   item.AvailabilityState.String(),
						"changed_fields": changedFields,
					},
				}
			},
		); err != nil {
			return writeCatalogError(c, err)
		}
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
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   &item.ID,
		EventType: activityevent.TypeAgentCreated,
		Message:   "Created agent " + item.Name,
		Metadata: map[string]any{
			"agent_id":       item.ID.String(),
			"agent_name":     item.Name,
			"provider_id":    item.ProviderID.String(),
			"changed_fields": []string{"agent"},
		},
	}); err != nil {
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

func (s *Server) patchAgent(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetAgent(c.Request().Context(), agentID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch agentPatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	input, err := parseAgentPatchRequest(agentID, current, patch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateAgent(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   &item.ID,
		EventType: activityevent.TypeAgentUpdated,
		Message:   "Updated agent " + item.Name,
		Metadata: map[string]any{
			"agent_id":       item.ID.String(),
			"agent_name":     item.Name,
			"provider_id":    item.ProviderID.String(),
			"changed_fields": agentChangedFields(current, item),
		},
	}); err != nil {
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
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   &item.ID,
		EventType: activityevent.TypeAgentPaused,
		Message:   "Paused agent " + item.Name,
		Metadata: map[string]any{
			"agent_id":       item.ID.String(),
			"agent_name":     item.Name,
			"runtime_state":  item.RuntimeControlState.String(),
			"changed_fields": []string{"runtime_control_state"},
		},
	}); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func (s *Server) interruptAgent(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	item, err := s.catalog.RequestAgentInterrupt(c.Request().Context(), agentID)
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
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   &item.ID,
		EventType: activityevent.TypeAgentInterruptRequested,
		Message:   "Requested interrupt for agent " + item.Name,
		Metadata: map[string]any{
			"agent_id":       item.ID.String(),
			"agent_name":     item.Name,
			"runtime_state":  item.RuntimeControlState.String(),
			"changed_fields": []string{"runtime_control_state"},
		},
	}); err != nil {
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
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   &item.ID,
		EventType: activityevent.TypeAgentResumed,
		Message:   "Resumed agent " + item.Name,
		Metadata: map[string]any{
			"agent_id":       item.ID.String(),
			"agent_name":     item.Name,
			"runtime_state":  item.RuntimeControlState.String(),
			"changed_fields": []string{"runtime_control_state"},
		},
	}); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func (s *Server) retireAgent(c echo.Context) error {
	agentID, err := parseUUIDPathParam(c, "agentId")
	if err != nil {
		return err
	}

	item, err := s.catalog.RetireAgent(c.Request().Context(), agentID)
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
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   &item.ID,
		EventType: activityevent.TypeAgentUpdated,
		Message:   "Retired agent " + item.Name,
		Metadata: map[string]any{
			"agent_id":       item.ID.String(),
			"agent_name":     item.Name,
			"runtime_state":  item.RuntimeControlState.String(),
			"changed_fields": []string{"runtime_control_state"},
		},
	}); err != nil {
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
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   &item.ID,
		EventType: activityevent.TypeAgentDeleted,
		Message:   "Deleted agent " + item.Name,
		Metadata: map[string]any{
			"agent_id":       item.ID.String(),
			"agent_name":     item.Name,
			"provider_id":    item.ProviderID.String(),
			"changed_fields": []string{"agent"},
		},
	}); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"agent": mapAgentResponse(item),
	})
}

func providerChangedFields(current domain.AgentProvider, item domain.AgentProvider) []string {
	fields := make([]string, 0, 8)
	if current.Name != item.Name {
		fields = append(fields, "name")
	}
	if current.AdapterType != item.AdapterType {
		fields = append(fields, "adapter_type")
	}
	if current.PermissionProfile != item.PermissionProfile {
		fields = append(fields, "permission_profile")
	}
	if current.CliCommand != item.CliCommand || !slices.Equal(current.CliArgs, item.CliArgs) {
		fields = append(fields, "cli")
	}
	if !mapsEqual(current.AuthConfig, item.AuthConfig) {
		fields = append(fields, "auth_config")
	}
	if current.ModelName != item.ModelName || current.ModelTemperature != item.ModelTemperature || current.ModelMaxTokens != item.ModelMaxTokens {
		fields = append(fields, "model")
	}
	if current.MaxParallelRuns != item.MaxParallelRuns {
		fields = append(fields, "max_parallel_runs")
	}
	if current.CostPerInputToken != item.CostPerInputToken || current.CostPerOutputToken != item.CostPerOutputToken {
		fields = append(fields, "cost")
	}
	return fields
}

func agentChangedFields(current domain.Agent, item domain.Agent) []string {
	fields := make([]string, 0, 2)
	if current.Name != item.Name {
		fields = append(fields, "name")
	}
	if current.ProviderID != item.ProviderID {
		fields = append(fields, "provider_id")
	}
	return fields
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
	permissionProfile := item.PermissionProfile
	if !permissionProfile.IsValid() {
		permissionProfile = domain.DefaultAgentProviderPermissionProfile
	}
	capabilities := domain.DeriveAgentProviderCapabilities(item).Capabilities
	ephemeralChatState := capabilities.EphemeralChat.State
	if !ephemeralChatState.IsValid() {
		ephemeralChatState = domain.AgentProviderCapabilityStateUnsupported
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
		PermissionProfile:     permissionProfile.String(),
		AvailabilityState:     availabilityState.String(),
		Available:             item.Available,
		AvailabilityCheckedAt: timePointerString(item.AvailabilityCheckedAt),
		AvailabilityReason:    stringPointerValue(item.AvailabilityReason),
		Capabilities: agentProviderCapabilitiesResponse{
			EphemeralChat: agentProviderCapabilityResponse{
				State:  ephemeralChatState.String(),
				Reason: stringPointerValue(capabilities.EphemeralChat.Reason),
			},
		},
		CliCommand:            item.CliCommand,
		CliArgs:               cloneStringSlice(item.CliArgs),
		AuthConfig:            cloneMap(item.AuthConfig),
		CLIRateLimit:          mapAgentProviderCLIRateLimitResponse(item.CLIRateLimit),
		CLIRateLimitUpdatedAt: timePointerString(item.CLIRateLimitUpdatedAt),
		ModelName:             item.ModelName,
		ModelTemperature:      item.ModelTemperature,
		ModelMaxTokens:        item.ModelMaxTokens,
		MaxParallelRuns:       item.MaxParallelRuns,
		CostPerInputToken:     item.CostPerInputToken,
		CostPerOutputToken:    item.CostPerOutputToken,
		PricingConfig:         item.PricingConfig.Clone(),
	}
}

func mapAgentProviderCLIRateLimitResponse(raw map[string]any) *agentProviderCLIRateLimitResponse {
	if len(raw) == 0 {
		return nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var response agentProviderCLIRateLimitResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil
	}
	response.Raw = cloneMap(raw)
	if response.Provider == "" && len(response.Raw) == 0 {
		return nil
	}

	return &response
}

func mapAgentProviderModelCatalogResponses() []agentProviderModelCatalogEntryResponse {
	adapterTypes := domain.BuiltinAgentProviderAdaptersWithModelOptions()
	responses := make([]agentProviderModelCatalogEntryResponse, 0, len(adapterTypes))
	for _, adapterType := range adapterTypes {
		responses = append(responses, agentProviderModelCatalogEntryResponse{
			AdapterType: adapterType.String(),
			Options:     mapAgentProviderModelOptionResponses(domain.BuiltinAgentProviderModelOptions(adapterType)),
		})
	}

	return responses
}

func mapAgentProviderModelOptionResponses(
	items []domain.AgentProviderModelOption,
) []agentProviderModelOptionResponse {
	responses := make([]agentProviderModelOptionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, agentProviderModelOptionResponse{
			ID:            item.ID,
			Label:         item.Label,
			Description:   item.Description,
			Recommended:   item.Recommended,
			Preview:       item.Preview,
			PricingConfig: clonePricingConfigPointer(item.PricingConfig),
		})
	}

	return responses
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

func clonePricingConfigPointer(
	value *pricing.ProviderModelPricingConfig,
) *pricing.ProviderModelPricingConfig {
	if value == nil {
		return nil
	}
	cloned := value.Clone()
	return &cloned
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
		ID:                item.ID.String(),
		AgentID:           item.AgentID.String(),
		WorkflowID:        item.WorkflowID.String(),
		WorkflowVersionID: uuidToStringPointer(item.WorkflowVersionID),
		TicketID:          item.TicketID.String(),
		ProviderID:        item.ProviderID.String(),
		SkillVersionIDs:   uuidSliceToStrings(item.SkillVersionIDs),
		Status:            item.Status.String(),
		SessionID:         item.SessionID,
		RuntimeStartedAt:  timeToStringPointer(item.RuntimeStartedAt),
		TerminalAt:        timeToStringPointer(item.TerminalAt),
		LastError:         item.LastError,
		LastHeartbeatAt:   timeToStringPointer(item.LastHeartbeatAt),
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
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
