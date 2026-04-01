package httpapi

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type agentProviderPatchRequest struct {
	MachineID          *string         `json:"machine_id"`
	Name               *string         `json:"name"`
	AdapterType        *string         `json:"adapter_type"`
	PermissionProfile  *string         `json:"permission_profile"`
	CliCommand         *string         `json:"cli_command"`
	CliArgs            *[]string       `json:"cli_args"`
	AuthConfig         *map[string]any `json:"auth_config"`
	ModelName          *string         `json:"model_name"`
	ModelTemperature   *float64        `json:"model_temperature"`
	ModelMaxTokens     *int            `json:"model_max_tokens"`
	MaxParallelRuns    *int            `json:"max_parallel_runs"`
	CostPerInputToken  *float64        `json:"cost_per_input_token"`
	CostPerOutputToken *float64        `json:"cost_per_output_token"`
	PricingConfig      *map[string]any `json:"pricing_config"`
}

type agentPatchRequest struct {
	ProviderID *string `json:"provider_id"`
	Name       *string `json:"name"`
}

func parseAgentProviderPatchRequest(
	providerID uuid.UUID,
	current domain.AgentProvider,
	patch agentProviderPatchRequest,
) (domain.UpdateAgentProvider, error) {
	request := domain.AgentProviderInput{
		MachineID:          current.MachineID.String(),
		Name:               current.Name,
		AdapterType:        current.AdapterType.String(),
		PermissionProfile:  mapAgentProviderResponse(current).PermissionProfile,
		CliCommand:         current.CliCommand,
		CliArgs:            append([]string(nil), current.CliArgs...),
		AuthConfig:         cloneMap(current.AuthConfig),
		ModelName:          current.ModelName,
		ModelTemperature:   floatPointer(current.ModelTemperature),
		ModelMaxTokens:     intPointer(current.ModelMaxTokens),
		MaxParallelRuns:    intPointer(current.MaxParallelRuns),
		CostPerInputToken:  floatPointer(current.CostPerInputToken),
		CostPerOutputToken: floatPointer(current.CostPerOutputToken),
		PricingConfig:      current.PricingConfig.ToMap(),
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
	if patch.PermissionProfile != nil {
		request.PermissionProfile = *patch.PermissionProfile
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
	if patch.MaxParallelRuns != nil {
		request.MaxParallelRuns = patch.MaxParallelRuns
	}
	if patch.CostPerInputToken != nil {
		request.CostPerInputToken = patch.CostPerInputToken
	}
	if patch.CostPerOutputToken != nil {
		request.CostPerOutputToken = patch.CostPerOutputToken
	}
	if patch.PricingConfig != nil {
		request.PricingConfig = cloneMap(*patch.PricingConfig)
	}

	return domain.ParseUpdateAgentProvider(providerID, current.OrganizationID, request)
}

func parseAgentPatchRequest(
	agentID uuid.UUID,
	current domain.Agent,
	patch agentPatchRequest,
) (domain.UpdateAgent, error) {
	request := domain.AgentInput{
		ProviderID: current.ProviderID.String(),
		Name:       current.Name,
	}
	if patch.ProviderID != nil {
		request.ProviderID = *patch.ProviderID
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}

	return domain.ParseUpdateAgent(agentID, current.ProjectID, request)
}
