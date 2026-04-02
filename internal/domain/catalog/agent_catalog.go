package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
	"github.com/google/uuid"
)

type AgentProvider struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	MachineID             uuid.UUID
	MachineName           string
	MachineHost           string
	MachineStatus         MachineStatus
	MachineSSHUser        *string
	MachineWorkspaceRoot  *string
	MachineAgentCLIPath   *string
	MachineResources      map[string]any
	Name                  string
	AdapterType           AgentProviderAdapterType
	PermissionProfile     AgentProviderPermissionProfile
	AvailabilityState     AgentProviderAvailabilityState
	Available             bool
	AvailabilityCheckedAt *time.Time
	AvailabilityReason    *string
	Capabilities          AgentProviderCapabilities
	CliCommand            string
	CliArgs               []string
	AuthConfig            map[string]any
	CLIRateLimit          map[string]any
	CLIRateLimitUpdatedAt *time.Time
	ModelName             string
	ModelTemperature      float64
	ModelMaxTokens        int
	MaxParallelRuns       int
	CostPerInputToken     float64
	CostPerOutputToken    float64
	PricingConfig         pricing.ProviderModelPricingConfig
}

type Agent struct {
	ID                    uuid.UUID
	ProviderID            uuid.UUID
	ProjectID             uuid.UUID
	Name                  string
	RuntimeControlState   AgentRuntimeControlState
	TotalTokensUsed       int64
	TotalTicketsCompleted int
	Runtime               *AgentRuntime
}

type AgentRuntime struct {
	ActiveRunCount       int
	CurrentRunID         *uuid.UUID
	Status               AgentStatus
	CurrentTicketID      *uuid.UUID
	SessionID            string
	RuntimePhase         AgentRuntimePhase
	RuntimeStartedAt     *time.Time
	LastError            string
	LastHeartbeatAt      *time.Time
	CurrentStepStatus    *string
	CurrentStepSummary   *string
	CurrentStepChangedAt *time.Time
}

type AgentRun struct {
	ID                           uuid.UUID
	AgentID                      uuid.UUID
	WorkflowID                   uuid.UUID
	WorkflowVersionID            *uuid.UUID
	TicketID                     uuid.UUID
	ProviderID                   uuid.UUID
	SkillVersionIDs              []uuid.UUID
	Status                       AgentRunStatus
	SessionID                    string
	RuntimeStartedAt             *time.Time
	TerminalAt                   *time.Time
	LastError                    string
	LastHeartbeatAt              *time.Time
	InputTokens                  int64
	OutputTokens                 int64
	CachedInputTokens            int64
	ReasoningTokens              int64
	TotalTokens                  int64
	CurrentStepStatus            *string
	CurrentStepSummary           *string
	CurrentStepChangedAt         *time.Time
	CompletionSummaryStatus      *AgentRunCompletionSummaryStatus
	CompletionSummaryMarkdown    *string
	CompletionSummaryJSON        map[string]any
	CompletionSummaryInput       map[string]any
	CompletionSummaryGeneratedAt *time.Time
	CompletionSummaryError       *string
	CreatedAt                    time.Time
}

type AgentProviderInput struct {
	MachineID          string         `json:"machine_id"`
	Name               string         `json:"name"`
	AdapterType        string         `json:"adapter_type"`
	PermissionProfile  string         `json:"permission_profile"`
	CliCommand         string         `json:"cli_command"`
	CliArgs            []string       `json:"cli_args"`
	AuthConfig         map[string]any `json:"auth_config"`
	ModelName          string         `json:"model_name"`
	ModelTemperature   *float64       `json:"model_temperature"`
	ModelMaxTokens     *int           `json:"model_max_tokens"`
	MaxParallelRuns    *int           `json:"max_parallel_runs"`
	CostPerInputToken  *float64       `json:"cost_per_input_token"`
	CostPerOutputToken *float64       `json:"cost_per_output_token"`
	PricingConfig      map[string]any `json:"pricing_config"`
}

type AgentInput struct {
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
}

type CreateAgentProvider struct {
	OrganizationID     uuid.UUID
	MachineID          uuid.UUID
	Name               string
	AdapterType        AgentProviderAdapterType
	PermissionProfile  AgentProviderPermissionProfile
	CliCommand         string
	CliArgs            []string
	AuthConfig         map[string]any
	ModelName          string
	ModelTemperature   float64
	ModelMaxTokens     int
	MaxParallelRuns    int
	CostPerInputToken  float64
	CostPerOutputToken float64
	PricingConfig      pricing.ProviderModelPricingConfig
}

type UpdateAgentProvider struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	MachineID          uuid.UUID
	Name               string
	AdapterType        AgentProviderAdapterType
	PermissionProfile  AgentProviderPermissionProfile
	CliCommand         string
	CliArgs            []string
	AuthConfig         map[string]any
	ModelName          string
	ModelTemperature   float64
	ModelMaxTokens     int
	MaxParallelRuns    int
	CostPerInputToken  float64
	CostPerOutputToken float64
	PricingConfig      pricing.ProviderModelPricingConfig
}

type CreateAgent struct {
	ProjectID             uuid.UUID
	ProviderID            uuid.UUID
	Name                  string
	RuntimeControlState   AgentRuntimeControlState
	TotalTokensUsed       int64
	TotalTicketsCompleted int
}

type UpdateAgent struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	ProviderID uuid.UUID
	Name       string
}

func ParseCreateAgentProvider(organizationID uuid.UUID, raw AgentProviderInput) (CreateAgentProvider, error) {
	machineID, err := parseRequiredUUID("machine_id", raw.MachineID)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	name, err := parseName("name", raw.Name)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	adapterType, err := parseAgentProviderAdapterType(raw.AdapterType)
	if err != nil {
		return CreateAgentProvider{}, err
	}

	permissionProfile, err := parseAgentProviderPermissionProfile(raw.PermissionProfile)
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

	maxParallelRuns, err := parseConcurrencyLimit("max_parallel_runs", raw.MaxParallelRuns)
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

	pricingConfig, err := pricing.ParseRawProviderModelPricingConfig(raw.PricingConfig, costPerInputToken, costPerOutputToken)
	if err != nil {
		return CreateAgentProvider{}, err
	}
	costPerInputToken = pricingConfig.SummaryInputPerToken()
	costPerOutputToken = pricingConfig.SummaryOutputPerToken()

	return CreateAgentProvider{
		OrganizationID:     organizationID,
		MachineID:          machineID,
		Name:               name,
		AdapterType:        adapterType,
		PermissionProfile:  permissionProfile,
		CliCommand:         strings.TrimSpace(raw.CliCommand),
		CliArgs:            cliArgs,
		AuthConfig:         cloneAnyMap(raw.AuthConfig),
		ModelName:          modelName,
		ModelTemperature:   modelTemperature,
		ModelMaxTokens:     modelMaxTokens,
		MaxParallelRuns:    maxParallelRuns,
		CostPerInputToken:  costPerInputToken,
		CostPerOutputToken: costPerOutputToken,
		PricingConfig:      pricingConfig,
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
		MachineID:          input.MachineID,
		Name:               input.Name,
		AdapterType:        input.AdapterType,
		PermissionProfile:  input.PermissionProfile,
		CliCommand:         input.CliCommand,
		CliArgs:            input.CliArgs,
		AuthConfig:         input.AuthConfig,
		ModelName:          input.ModelName,
		ModelTemperature:   input.ModelTemperature,
		ModelMaxTokens:     input.ModelMaxTokens,
		MaxParallelRuns:    input.MaxParallelRuns,
		CostPerInputToken:  input.CostPerInputToken,
		CostPerOutputToken: input.CostPerOutputToken,
		PricingConfig:      input.PricingConfig,
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

	return CreateAgent{
		ProjectID:             projectID,
		ProviderID:            providerID,
		Name:                  name,
		RuntimeControlState:   DefaultAgentRuntimeControlState,
		TotalTokensUsed:       DefaultAgentTotalTokensUsed,
		TotalTicketsCompleted: DefaultAgentTotalTicketsCompleted,
	}, nil
}

func ParseUpdateAgent(id uuid.UUID, projectID uuid.UUID, raw AgentInput) (UpdateAgent, error) {
	input, err := ParseCreateAgent(projectID, raw)
	if err != nil {
		return UpdateAgent{}, err
	}

	return UpdateAgent{
		ID:         id,
		ProjectID:  input.ProjectID,
		ProviderID: input.ProviderID,
		Name:       input.Name,
	}, nil
}

func BuildAgentRuntimeSummary(currentRuns []AgentRun, controlState AgentRuntimeControlState) *AgentRuntime {
	if len(currentRuns) == 0 {
		return nil
	}

	representative := currentRuns[0]
	for _, run := range currentRuns[1:] {
		if preferAgentRuntimeRepresentative(run, representative) {
			representative = run
		}
	}

	runtime := &AgentRuntime{
		ActiveRunCount:       len(currentRuns),
		CurrentRunID:         &representative.ID,
		Status:               DefaultAgentStatus,
		CurrentTicketID:      &representative.TicketID,
		SessionID:            representative.SessionID,
		RuntimePhase:         DefaultAgentRuntimePhase,
		RuntimeStartedAt:     cloneTimePointer(representative.RuntimeStartedAt),
		LastError:            representative.LastError,
		LastHeartbeatAt:      cloneTimePointer(representative.LastHeartbeatAt),
		CurrentStepStatus:    cloneStringPointer(representative.CurrentStepStatus),
		CurrentStepSummary:   cloneStringPointer(representative.CurrentStepSummary),
		CurrentStepChangedAt: cloneTimePointer(representative.CurrentStepChangedAt),
	}

	switch representative.Status {
	case AgentRunStatusLaunching:
		runtime.Status = AgentStatusClaimed
		runtime.RuntimePhase = AgentRuntimePhaseLaunching
	case AgentRunStatusReady:
		runtime.Status = AgentStatusRunning
		runtime.RuntimePhase = AgentRuntimePhaseReady
	case AgentRunStatusExecuting:
		runtime.Status = AgentStatusRunning
		runtime.RuntimePhase = AgentRuntimePhaseExecuting
	case AgentRunStatusErrored:
		runtime.Status = AgentStatusFailed
		runtime.RuntimePhase = AgentRuntimePhaseFailed
	case AgentRunStatusTerminated:
		runtime.Status = AgentStatusTerminated
		if controlState == AgentRuntimeControlStatePaused {
			runtime.Status = AgentStatusPaused
		}
	case AgentRunStatusCompleted:
		runtime.Status = DefaultAgentStatus
	}

	for _, run := range currentRuns[1:] {
		if moreRecentTime(run.LastHeartbeatAt, runtime.LastHeartbeatAt) {
			runtime.LastHeartbeatAt = cloneTimePointer(run.LastHeartbeatAt)
		}
		if moreRecentTime(run.RuntimeStartedAt, runtime.RuntimeStartedAt) {
			runtime.RuntimeStartedAt = cloneTimePointer(run.RuntimeStartedAt)
		}
		if preferAgentRuntimeError(run, runtime.LastError, runtime.LastHeartbeatAt) {
			runtime.LastError = run.LastError
		}
	}

	if len(currentRuns) > 1 {
		runtime.CurrentRunID = nil
		runtime.CurrentTicketID = nil
		runtime.SessionID = ""
		runtime.CurrentStepStatus = nil
		runtime.CurrentStepSummary = nil
		runtime.CurrentStepChangedAt = nil
	}

	return runtime
}

func preferAgentRuntimeRepresentative(candidate AgentRun, current AgentRun) bool {
	candidatePriority := agentRuntimeRepresentativePriority(candidate.Status)
	currentPriority := agentRuntimeRepresentativePriority(current.Status)
	if candidatePriority != currentPriority {
		return candidatePriority < currentPriority
	}

	if candidate.CreatedAt.Equal(current.CreatedAt) {
		return false
	}

	return candidate.CreatedAt.After(current.CreatedAt)
}

func agentRuntimeRepresentativePriority(status AgentRunStatus) int {
	switch status {
	case AgentRunStatusExecuting:
		return 0
	case AgentRunStatusReady:
		return 1
	case AgentRunStatusLaunching:
		return 2
	case AgentRunStatusErrored:
		return 3
	case AgentRunStatusTerminated:
		return 4
	default:
		return 5
	}
}

func preferAgentRuntimeError(candidate AgentRun, currentError string, currentHeartbeat *time.Time) bool {
	if strings.TrimSpace(candidate.LastError) == "" {
		return false
	}
	if strings.TrimSpace(currentError) == "" {
		return true
	}

	return moreRecentTime(candidate.LastHeartbeatAt, currentHeartbeat)
}

func moreRecentTime(candidate *time.Time, current *time.Time) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}

	return candidate.After(*current)
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := strings.TrimSpace(*value)
	return &cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	copied := value.UTC()
	return &copied
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

func parseAgentProviderPermissionProfile(raw string) (AgentProviderPermissionProfile, error) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return DefaultAgentProviderPermissionProfile, nil
	}

	profile := AgentProviderPermissionProfile(trimmed)
	if !profile.IsValid() {
		return "", fmt.Errorf("permission_profile must be one of standard, unrestricted")
	}

	return profile, nil
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

func parseConcurrencyLimit(fieldName string, raw *int) (int, error) {
	if raw == nil {
		return 0, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("%s must be greater than or equal to zero", fieldName)
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
