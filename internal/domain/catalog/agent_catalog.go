package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AgentProvider struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	Name               string
	AdapterType        AgentProviderAdapterType
	Available          bool
	CliCommand         string
	CliArgs            []string
	AuthConfig         map[string]any
	ModelName          string
	ModelTemperature   float64
	ModelMaxTokens     int
	CostPerInputToken  float64
	CostPerOutputToken float64
}

type Agent struct {
	ID                    uuid.UUID
	ProviderID            uuid.UUID
	ProjectID             uuid.UUID
	Name                  string
	Status                AgentStatus
	CurrentTicketID       *uuid.UUID
	SessionID             string
	RuntimePhase          AgentRuntimePhase
	RuntimeControlState   AgentRuntimeControlState
	RuntimeStartedAt      *time.Time
	LastError             string
	WorkspacePath         string
	Capabilities          []string
	TotalTokensUsed       int64
	TotalTicketsCompleted int
	LastHeartbeatAt       *time.Time
}

type AgentProviderInput struct {
	Name               string         `json:"name"`
	AdapterType        string         `json:"adapter_type"`
	CliCommand         string         `json:"cli_command"`
	CliArgs            []string       `json:"cli_args"`
	AuthConfig         map[string]any `json:"auth_config"`
	ModelName          string         `json:"model_name"`
	ModelTemperature   *float64       `json:"model_temperature"`
	ModelMaxTokens     *int           `json:"model_max_tokens"`
	CostPerInputToken  *float64       `json:"cost_per_input_token"`
	CostPerOutputToken *float64       `json:"cost_per_output_token"`
}

type AgentInput struct {
	ProviderID    string   `json:"provider_id"`
	Name          string   `json:"name"`
	WorkspacePath string   `json:"workspace_path"`
	Capabilities  []string `json:"capabilities"`
}

type CreateAgentProvider struct {
	OrganizationID     uuid.UUID
	Name               string
	AdapterType        AgentProviderAdapterType
	CliCommand         string
	CliArgs            []string
	AuthConfig         map[string]any
	ModelName          string
	ModelTemperature   float64
	ModelMaxTokens     int
	CostPerInputToken  float64
	CostPerOutputToken float64
}

type UpdateAgentProvider struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	Name               string
	AdapterType        AgentProviderAdapterType
	CliCommand         string
	CliArgs            []string
	AuthConfig         map[string]any
	ModelName          string
	ModelTemperature   float64
	ModelMaxTokens     int
	CostPerInputToken  float64
	CostPerOutputToken float64
}

type CreateAgent struct {
	ProjectID             uuid.UUID
	ProviderID            uuid.UUID
	Name                  string
	Status                AgentStatus
	CurrentTicketID       *uuid.UUID
	SessionID             string
	RuntimePhase          AgentRuntimePhase
	RuntimeControlState   AgentRuntimeControlState
	RuntimeStartedAt      *time.Time
	LastError             string
	WorkspacePath         string
	Capabilities          []string
	TotalTokensUsed       int64
	TotalTicketsCompleted int
	LastHeartbeatAt       *time.Time
}

func ParseCreateAgentProvider(organizationID uuid.UUID, raw AgentProviderInput) (CreateAgentProvider, error) {
	name, err := parseName("name", raw.Name)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	adapterType, err := parseAgentProviderAdapterType(raw.AdapterType)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	cliArgs, err := parseStringList("cli_args", raw.CliArgs)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	modelName, err := parseName("model_name", raw.ModelName)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	modelTemperature, err := parseNonNegativeFloat("model_temperature", raw.ModelTemperature, DefaultAgentProviderModelTemperature)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	modelMaxTokens, err := parsePositiveInt("model_max_tokens", raw.ModelMaxTokens, DefaultAgentProviderModelMaxTokens)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	costPerInputToken, err := parseNonNegativeFloat("cost_per_input_token", raw.CostPerInputToken, DefaultAgentProviderCostPerInputToken)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	costPerOutputToken, err := parseNonNegativeFloat("cost_per_output_token", raw.CostPerOutputToken, DefaultAgentProviderCostPerOutputToken)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	return CreateAgentProvider{
		OrganizationID:     organizationID,
		Name:               name,
		AdapterType:        adapterType,
		CliCommand:         strings.TrimSpace(raw.CliCommand),
		CliArgs:            cliArgs,
		AuthConfig:         cloneAnyMap(raw.AuthConfig),
		ModelName:          modelName,
		ModelTemperature:   modelTemperature,
		ModelMaxTokens:     modelMaxTokens,
		CostPerInputToken:  costPerInputToken,
		CostPerOutputToken: costPerOutputToken,
	}, nil
}

func ParseUpdateAgentProvider(id uuid.UUID, organizationID uuid.UUID, raw AgentProviderInput) (UpdateAgentProvider, error) {
	input, err := ParseCreateAgentProvider(organizationID, raw)
	if err != nil {
		return UpdateAgentProvider{}, err
	}

	return UpdateAgentProvider{
		ID:                 id,
		OrganizationID:     input.OrganizationID,
		Name:               input.Name,
		AdapterType:        input.AdapterType,
		CliCommand:         input.CliCommand,
		CliArgs:            input.CliArgs,
		AuthConfig:         input.AuthConfig,
		ModelName:          input.ModelName,
		ModelTemperature:   input.ModelTemperature,
		ModelMaxTokens:     input.ModelMaxTokens,
		CostPerInputToken:  input.CostPerInputToken,
		CostPerOutputToken: input.CostPerOutputToken,
	}, nil
}

func ParseCreateAgent(projectID uuid.UUID, raw AgentInput) (CreateAgent, error) {
	providerID, err := parseRequiredUUID("provider_id", raw.ProviderID)
	if err != nil {
		return CreateAgent{}, err
	}

	name, err := parseName("name", raw.Name)
	if err != nil {
		return CreateAgent{}, err
	}

	capabilities, err := parseStringList("capabilities", raw.Capabilities)
	if err != nil {
		return CreateAgent{}, err
	}

	return CreateAgent{
		ProjectID:             projectID,
		ProviderID:            providerID,
		Name:                  name,
		Status:                DefaultAgentStatus,
		RuntimePhase:          DefaultAgentRuntimePhase,
		RuntimeControlState:   DefaultAgentRuntimeControlState,
		WorkspacePath:         strings.TrimSpace(raw.WorkspacePath),
		Capabilities:          capabilities,
		TotalTokensUsed:       DefaultAgentTotalTokensUsed,
		TotalTicketsCompleted: DefaultAgentTotalTicketsCompleted,
	}, nil
}

func parseRequiredUUID(fieldName string, raw string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return uuid.UUID{}, fmt.Errorf("%s must not be empty", fieldName)
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s must be a valid UUID", fieldName)
	}

	return parsed, nil
}

func parseAgentProviderAdapterType(raw string) (AgentProviderAdapterType, error) {
	adapterType := AgentProviderAdapterType(strings.TrimSpace(strings.ToLower(raw)))
	if !adapterType.IsValid() {
		return "", fmt.Errorf("adapter_type must be one of claude-code-cli, codex-app-server, gemini-cli, custom")
	}

	return adapterType, nil
}

func parseStringList(fieldName string, raw []string) ([]string, error) {
	if raw == nil {
		return nil, nil
	}

	parsed := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("%s must not contain empty values", fieldName)
		}
		parsed = append(parsed, trimmed)
	}

	return parsed, nil
}

func parsePositiveInt(fieldName string, raw *int, defaultValue int) (int, error) {
	if raw == nil {
		return defaultValue, nil
	}
	if *raw <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", fieldName)
	}

	return *raw, nil
}

func parseNonNegativeFloat(fieldName string, raw *float64, defaultValue float64) (float64, error) {
	if raw == nil {
		return defaultValue, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("%s must be greater than or equal to zero", fieldName)
	}

	return *raw, nil
}

func cloneAnyMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}

	return cloned
}
