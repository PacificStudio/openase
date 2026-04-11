package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var errAPIResponseCommitted = errors.New("api response already committed")

type organizationResponse struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	Status                 string  `json:"status"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id,omitempty"`
}

type projectResponse struct {
	ID                             string                     `json:"id"`
	OrganizationID                 string                     `json:"organization_id"`
	Name                           string                     `json:"name"`
	Slug                           string                     `json:"slug"`
	Description                    string                     `json:"description"`
	Status                         string                     `json:"status"`
	DefaultAgentProviderID         *string                    `json:"default_agent_provider_id,omitempty"`
	ProjectAIPlatformAccessAllowed []string                   `json:"project_ai_platform_access_allowed"`
	AccessibleMachineIDs           []string                   `json:"accessible_machine_ids,omitempty"`
	MaxConcurrentAgents            int                        `json:"max_concurrent_agents"`
	AgentRunSummaryPrompt          *string                    `json:"agent_run_summary_prompt,omitempty"`
	EffectiveAgentRunSummaryPrompt string                     `json:"effective_agent_run_summary_prompt"`
	AgentRunSummaryPromptSource    string                     `json:"agent_run_summary_prompt_source"`
	ProjectAIRetention             projectAIRetentionResponse `json:"project_ai_retention"`
}

type projectAIRetentionResponse struct {
	Enabled        bool `json:"enabled"`
	KeepLatestN    int  `json:"keep_latest_n"`
	KeepRecentDays int  `json:"keep_recent_days"`
}

type machineResponse struct {
	ID                    string                           `json:"id"`
	OrganizationID        string                           `json:"organization_id"`
	Name                  string                           `json:"name"`
	Host                  string                           `json:"host"`
	Port                  int                              `json:"port"`
	ReachabilityMode      string                           `json:"reachability_mode"`
	ExecutionMode         string                           `json:"execution_mode"`
	ExecutionCapabilities []string                         `json:"execution_capabilities,omitempty"`
	SSHHelperEnabled      bool                             `json:"ssh_helper_enabled"`
	SSHUser               *string                          `json:"ssh_user,omitempty"`
	SSHKeyPath            *string                          `json:"ssh_key_path,omitempty"`
	AdvertisedEndpoint    *string                          `json:"advertised_endpoint,omitempty"`
	DaemonStatus          machineDaemonStatusResponse      `json:"daemon_status"`
	DetectedOS            string                           `json:"detected_os"`
	DetectedArch          string                           `json:"detected_arch"`
	DetectionStatus       string                           `json:"detection_status"`
	DetectionMessage      string                           `json:"detection_message"`
	ChannelCredential     machineChannelCredentialResponse `json:"channel_credential"`
	Description           string                           `json:"description"`
	Labels                []string                         `json:"labels"`
	Status                string                           `json:"status"`
	WorkspaceRoot         *string                          `json:"workspace_root,omitempty"`
	AgentCLIPath          *string                          `json:"agent_cli_path,omitempty"`
	EnvVars               []string                         `json:"env_vars"`
	LastHeartbeatAt       *string                          `json:"last_heartbeat_at,omitempty"`
	Resources             map[string]any                   `json:"resources"`
}

type machineDaemonStatusResponse struct {
	Registered       bool    `json:"registered"`
	LastRegisteredAt *string `json:"last_registered_at,omitempty"`
	CurrentSessionID *string `json:"current_session_id,omitempty"`
	SessionState     string  `json:"session_state"`
}

type machineChannelCredentialResponse struct {
	Kind          string  `json:"kind"`
	TokenID       *string `json:"token_id,omitempty"`
	CertificateID *string `json:"certificate_id,omitempty"`
}

type machineProbeResponse struct {
	CheckedAt        string         `json:"checked_at"`
	Transport        string         `json:"transport"`
	Output           string         `json:"output"`
	Resources        map[string]any `json:"resources"`
	DetectedOS       string         `json:"detected_os"`
	DetectedArch     string         `json:"detected_arch"`
	DetectionStatus  string         `json:"detection_status"`
	DetectionMessage string         `json:"detection_message"`
}

type projectRepoResponse struct {
	ID               string   `json:"id"`
	ProjectID        string   `json:"project_id"`
	Name             string   `json:"name"`
	RepositoryURL    string   `json:"repository_url"`
	DefaultBranch    string   `json:"default_branch"`
	WorkspaceDirname string   `json:"workspace_dirname"`
	Labels           []string `json:"labels"`
}

type ticketRepoScopeResponse struct {
	ID             string  `json:"id"`
	TicketID       string  `json:"ticket_id"`
	RepoID         string  `json:"repo_id"`
	BranchName     string  `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url,omitempty"`
}

func decodeJSON(c echo.Context, target any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		message := fmt.Sprintf("invalid JSON body: %v", err)
		logAPIBoundaryError(c, http.StatusBadRequest, "INVALID_REQUEST", message)
		if writeErr := c.JSON(http.StatusBadRequest, errorResponse(fmt.Sprintf("invalid JSON body: %v", err))); writeErr != nil {
			return writeErr
		}
		return errAPIResponseCommitted
	}
	if decoder.More() {
		message := "invalid JSON body: multiple JSON values are not allowed"
		logAPIBoundaryError(c, http.StatusBadRequest, "INVALID_REQUEST", message)
		if writeErr := c.JSON(http.StatusBadRequest, errorResponse("invalid JSON body: multiple JSON values are not allowed")); writeErr != nil {
			return writeErr
		}
		return errAPIResponseCommitted
	}

	return nil
}

func parseUUIDPathParam(c echo.Context, name string) (uuid.UUID, error) {
	parsed, err := parseUUIDPathParamValue(c, name)
	if err != nil {
		logAPIBoundaryError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		if writeErr := c.JSON(http.StatusBadRequest, errorResponse(err.Error())); writeErr != nil {
			return uuid.UUID{}, writeErr
		}
		return uuid.UUID{}, errAPIResponseCommitted
	}

	return parsed, nil
}

func parseUUIDPathParamValue(c echo.Context, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Param(name))
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s must be a valid UUID", name)
	}

	return parsed, nil
}

func writeCatalogError(c echo.Context, err error) error {
	var projectRepoConflict *domain.ProjectRepoDeleteConflict
	var ticketRepoScopeConflict *domain.TicketRepoScopeDeleteConflict
	var agentDeleteConflict *domain.AgentDeleteConflict
	switch {
	case errors.As(err, &projectRepoConflict):
		return writeAPIErrorWithDetails(
			c,
			http.StatusConflict,
			"REPOSITORY_IN_USE",
			"Repository cannot be deleted because tickets or workspaces still reference it.",
			projectRepoConflict,
		)
	case errors.As(err, &ticketRepoScopeConflict):
		return writeAPIErrorWithDetails(
			c,
			http.StatusConflict,
			"TICKET_REPO_SCOPE_IN_USE",
			"Repository scope cannot be deleted while the ticket has an active run or repo workspace activity.",
			ticketRepoScopeConflict,
		)
	case errors.As(err, &agentDeleteConflict):
		return writeAPIErrorWithDetails(
			c,
			http.StatusConflict,
			"AGENT_IN_USE",
			"Agent cannot be deleted because runs still reference it.",
			agentDeleteConflict,
		)
	}
	statusCode, code, message := catalogErrorResponse(err)
	return writeAPIError(c, statusCode, code, message)
}

func errorResponse(message string) map[string]string {
	return map[string]string{"error": message}
}

func catalogErrorMessage(err error) string {
	for _, prefix := range []string{
		catalogservice.ErrInvalidInput.Error() + ": ",
		catalogservice.ErrMachineProbeFailed.Error() + ": ",
		catalogservice.ErrMachineTestingUnavailable.Error() + ": ",
		catalogservice.ErrMachineHealthUnavailable.Error() + ": ",
	} {
		if strings.HasPrefix(err.Error(), prefix) {
			return strings.TrimPrefix(err.Error(), prefix)
		}
	}

	return err.Error()
}

func catalogConflictMessage(err error) string {
	prefix := catalogservice.ErrConflict.Error() + ": "
	if strings.HasPrefix(err.Error(), prefix) {
		return strings.TrimPrefix(err.Error(), prefix)
	}

	return err.Error()
}

func catalogErrorResponse(err error) (statusCode int, code string, message string) {
	switch {
	case errors.Is(err, catalogservice.ErrInvalidInput):
		return http.StatusBadRequest, "INVALID_REQUEST", catalogErrorMessage(err)
	case errors.Is(err, catalogservice.ErrNotFound):
		return http.StatusNotFound, "RESOURCE_NOT_FOUND", "resource not found"
	case errors.Is(err, domain.ErrOrganizationSlugConflict):
		return http.StatusConflict, "ORGANIZATION_SLUG_CONFLICT", "Organization slug already exists."
	case errors.Is(err, domain.ErrProjectSlugConflict):
		return http.StatusConflict, "PROJECT_SLUG_CONFLICT", "Project slug already exists in this organization."
	case errors.Is(err, domain.ErrMachineNameConflict):
		return http.StatusConflict, "MACHINE_NAME_CONFLICT", "Machine name already exists in this organization."
	case errors.Is(err, domain.ErrMachineInUseConflict):
		return http.StatusConflict, "MACHINE_IN_USE", "Machine cannot be deleted because agent providers still reference it."
	case errors.Is(err, domain.ErrAgentProviderNameConflict):
		return http.StatusConflict, "AGENT_PROVIDER_NAME_CONFLICT", "Agent provider name already exists in this organization."
	case errors.Is(err, domain.ErrProjectRepoNameConflict):
		return http.StatusConflict, "REPOSITORY_NAME_CONFLICT", "Repository name already exists in this project."
	case errors.Is(err, domain.ErrProjectRepoInUseConflict):
		return http.StatusConflict, "REPOSITORY_IN_USE", "Repository cannot be deleted because tickets or workspaces still reference it."
	case errors.Is(err, domain.ErrTicketRepoScopeConflict):
		return http.StatusConflict, "TICKET_REPO_SCOPE_CONFLICT", "Repository is already attached to this ticket."
	case errors.Is(err, domain.ErrTicketRepoScopeInUseConflict):
		return http.StatusConflict, "TICKET_REPO_SCOPE_IN_USE", "Repository scope cannot be deleted while the ticket has an active run or repo workspace activity."
	case errors.Is(err, domain.ErrAgentNameConflict):
		return http.StatusConflict, "AGENT_NAME_CONFLICT", "Agent name already exists in this project."
	case errors.Is(err, domain.ErrAgentInUseConflict):
		return http.StatusConflict, "AGENT_IN_USE", "Agent cannot be deleted because runs still reference it."
	case errors.Is(err, catalogservice.ErrConflict):
		return http.StatusConflict, "RESOURCE_CONFLICT", normalizeCatalogConflictMessage(err)
	case errors.Is(err, catalogservice.ErrMachineProbeFailed):
		return http.StatusBadGateway, "MACHINE_PROBE_FAILED", catalogErrorMessage(err)
	case errors.Is(err, catalogservice.ErrMachineTestingUnavailable):
		return http.StatusServiceUnavailable, "MACHINE_TESTING_UNAVAILABLE", catalogErrorMessage(err)
	case errors.Is(err, catalogservice.ErrMachineHealthUnavailable):
		return http.StatusServiceUnavailable, "MACHINE_HEALTH_UNAVAILABLE", catalogErrorMessage(err)
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	}
}

func normalizeCatalogConflictMessage(err error) string {
	message := catalogConflictMessage(err)
	if message == "" || message == catalogservice.ErrConflict.Error() {
		return "resource conflict"
	}

	return message
}

func mapOrganizationResponses(items []domain.Organization) []organizationResponse {
	response := make([]organizationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapOrganizationResponse(item))
	}

	return response
}

func mapOrganizationResponse(item domain.Organization) organizationResponse {
	return organizationResponse{
		ID:                     item.ID.String(),
		Name:                   item.Name,
		Slug:                   item.Slug,
		Status:                 string(item.Status),
		DefaultAgentProviderID: uuidToStringPointer(item.DefaultAgentProviderID),
	}
}

func mapProjectResponses(items []domain.Project) []projectResponse {
	response := make([]projectResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectResponse(item))
	}

	return response
}

func mapMachineResponses(items []domain.Machine) []machineResponse {
	response := make([]machineResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapMachineResponse(item))
	}

	return response
}

func mapProjectResponse(item domain.Project) projectResponse {
	effectivePrompt, promptSource := domain.EffectiveAgentRunSummaryPrompt(item.AgentRunSummaryPrompt)
	return projectResponse{
		ID:                             item.ID.String(),
		OrganizationID:                 item.OrganizationID.String(),
		Name:                           item.Name,
		Slug:                           item.Slug,
		Description:                    item.Description,
		Status:                         item.Status.String(),
		DefaultAgentProviderID:         uuidToStringPointer(item.DefaultAgentProviderID),
		ProjectAIPlatformAccessAllowed: cloneStringSlice(item.ProjectAIPlatformAccessAllowed),
		AccessibleMachineIDs:           uuidSliceToStrings(item.AccessibleMachineIDs),
		MaxConcurrentAgents:            item.MaxConcurrentAgents,
		AgentRunSummaryPrompt:          stringPointerOrNil(item.AgentRunSummaryPrompt),
		EffectiveAgentRunSummaryPrompt: effectivePrompt,
		AgentRunSummaryPromptSource:    promptSource.String(),
		ProjectAIRetention: projectAIRetentionResponse{
			Enabled:        item.ProjectAIRetention.Enabled,
			KeepLatestN:    item.ProjectAIRetention.KeepLatestN,
			KeepRecentDays: item.ProjectAIRetention.KeepRecentDays,
		},
	}
}

func mapMachineResponse(item domain.Machine) machineResponse {
	executionCapabilities := machineTransportCapabilityStrings(item.TransportCapabilities)
	sshHelperEnabled := item.SSHUser != nil || item.SSHKeyPath != nil
	return machineResponse{
		ID:                    item.ID.String(),
		OrganizationID:        item.OrganizationID.String(),
		Name:                  item.Name,
		Host:                  item.Host,
		Port:                  item.Port,
		ReachabilityMode:      item.ReachabilityMode.String(),
		ExecutionMode:         item.ExecutionMode.String(),
		ExecutionCapabilities: executionCapabilities,
		SSHHelperEnabled:      sshHelperEnabled,
		SSHUser:               item.SSHUser,
		SSHKeyPath:            item.SSHKeyPath,
		AdvertisedEndpoint:    item.AdvertisedEndpoint,
		DaemonStatus:          mapMachineDaemonStatusResponse(item.DaemonStatus),
		DetectedOS:            item.DetectedOS.String(),
		DetectedArch:          item.DetectedArch.String(),
		DetectionStatus:       item.DetectionStatus.String(),
		DetectionMessage:      domain.MachineDetectionMessage(item.DetectedOS, item.DetectedArch, item.DetectionStatus),
		ChannelCredential:     mapMachineChannelCredentialResponse(item.ChannelCredential),
		Description:           item.Description,
		Labels:                cloneStringSlice(item.Labels),
		Status:                item.Status.String(),
		WorkspaceRoot:         item.WorkspaceRoot,
		AgentCLIPath:          item.AgentCLIPath,
		EnvVars:               domain.MaskMachineEnvVars(item.EnvVars),
		LastHeartbeatAt:       timeToStringPointer(item.LastHeartbeatAt),
		Resources:             cloneMap(item.Resources),
	}
}

func mapMachineProbeResponse(item domain.MachineProbe) machineProbeResponse {
	return machineProbeResponse{
		CheckedAt:        item.CheckedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Transport:        item.Transport,
		Output:           item.Output,
		Resources:        cloneMap(item.Resources),
		DetectedOS:       item.DetectedOS.String(),
		DetectedArch:     item.DetectedArch.String(),
		DetectionStatus:  item.DetectionStatus.String(),
		DetectionMessage: domain.MachineDetectionMessage(item.DetectedOS, item.DetectedArch, item.DetectionStatus),
	}
}

func baseProjectRepoResponse(item domain.ProjectRepo) projectRepoResponse {
	return projectRepoResponse{
		ID:               item.ID.String(),
		ProjectID:        item.ProjectID.String(),
		Name:             item.Name,
		RepositoryURL:    item.RepositoryURL,
		DefaultBranch:    item.DefaultBranch,
		WorkspaceDirname: item.WorkspaceDirname,
		Labels:           cloneStringSlice(item.Labels),
	}
}

func mapProjectRepoResponse(item domain.ProjectRepo) projectRepoResponse {
	return baseProjectRepoResponse(item)
}

func mapProjectRepoResponses(items []domain.ProjectRepo) []projectRepoResponse {
	response := make([]projectRepoResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectRepoResponse(item))
	}
	return response
}

func mapTicketRepoScopeResponses(items []domain.TicketRepoScope) []ticketRepoScopeResponse {
	response := make([]ticketRepoScopeResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapTicketRepoScopeResponse(item))
	}

	return response
}

func mapTicketRepoScopeResponse(item domain.TicketRepoScope) ticketRepoScopeResponse {
	return ticketRepoScopeResponse{
		ID:             item.ID.String(),
		TicketID:       item.TicketID.String(),
		RepoID:         item.RepoID.String(),
		BranchName:     item.BranchName,
		PullRequestURL: item.PullRequestURL,
	}
}

func uuidToStringPointer(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	text := value.String()
	return &text
}

func timeStringPointer(value *time.Time) *string {
	if value == nil {
		return nil
	}

	text := value.UTC().Format(time.RFC3339)
	return &text
}

func cloneStringPointerValue(value *string) *string {
	if value == nil {
		return nil
	}

	text := *value
	return &text
}

func uuidSliceToStrings(values []uuid.UUID) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, value.String())
	}
	return items
}

func intPointer(value int) *int {
	copied := value
	return &copied
}

func stringPointer(value string) *string {
	copied := value
	return &copied
}

func boolPointer(value bool) *bool {
	copied := value
	return &copied
}

func mapMachineDaemonStatusResponse(status domain.MachineDaemonStatus) machineDaemonStatusResponse {
	return machineDaemonStatusResponse{
		Registered:       status.Registered,
		LastRegisteredAt: timeToStringPointer(status.LastRegisteredAt),
		CurrentSessionID: cloneStringPointerValue(status.CurrentSessionID),
		SessionState:     status.SessionState.String(),
	}
}

func mapMachineChannelCredentialResponse(credential domain.MachineChannelCredential) machineChannelCredentialResponse {
	return machineChannelCredentialResponse{
		Kind:          credential.Kind.String(),
		TokenID:       cloneStringPointerValue(credential.TokenID),
		CertificateID: cloneStringPointerValue(credential.CertificateID),
	}
}

func machineTransportCapabilityStrings(items []domain.MachineTransportCapability) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.String())
	}
	return values
}
