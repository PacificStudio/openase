package catalog

type OrganizationStatus string

const (
	OrganizationStatusActive   OrganizationStatus = "active"
	OrganizationStatusArchived OrganizationStatus = "archived"
)

func (s OrganizationStatus) String() string {
	return string(s)
}

func (s OrganizationStatus) IsValid() bool {
	switch s {
	case OrganizationStatusActive, OrganizationStatusArchived:
		return true
	default:
		return false
	}
}

type ProjectStatus string

const (
	DefaultProjectStatus              ProjectStatus = ProjectStatusPlanning
	DefaultProjectMaxConcurrentAgents               = 5

	ProjectStatusPlanning ProjectStatus = "planning"
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusPaused   ProjectStatus = "paused"
	ProjectStatusArchived ProjectStatus = "archived"
)

func (s ProjectStatus) String() string {
	return string(s)
}

func (s ProjectStatus) IsValid() bool {
	switch s {
	case ProjectStatusPlanning, ProjectStatusActive, ProjectStatusPaused, ProjectStatusArchived:
		return true
	default:
		return false
	}
}

type TicketRepoScopePRStatus string

const (
	DefaultTicketRepoScopePRStatus          TicketRepoScopePRStatus = TicketRepoScopePRStatusNone
	TicketRepoScopePRStatusNone             TicketRepoScopePRStatus = "none"
	TicketRepoScopePRStatusOpen             TicketRepoScopePRStatus = "open"
	TicketRepoScopePRStatusChangesRequested TicketRepoScopePRStatus = "changes_requested"
	TicketRepoScopePRStatusApproved         TicketRepoScopePRStatus = "approved"
	TicketRepoScopePRStatusMerged           TicketRepoScopePRStatus = "merged"
	TicketRepoScopePRStatusClosed           TicketRepoScopePRStatus = "closed"
)

func (s TicketRepoScopePRStatus) String() string {
	return string(s)
}

func (s TicketRepoScopePRStatus) IsValid() bool {
	switch s {
	case TicketRepoScopePRStatusNone, TicketRepoScopePRStatusOpen, TicketRepoScopePRStatusChangesRequested, TicketRepoScopePRStatusApproved, TicketRepoScopePRStatusMerged, TicketRepoScopePRStatusClosed:
		return true
	default:
		return false
	}
}

type TicketRepoScopeCIStatus string

const (
	DefaultTicketRepoScopeCIStatus TicketRepoScopeCIStatus = TicketRepoScopeCIStatusPending
	TicketRepoScopeCIStatusPending TicketRepoScopeCIStatus = "pending"
	TicketRepoScopeCIStatusPassing TicketRepoScopeCIStatus = "passing"
	TicketRepoScopeCIStatusFailing TicketRepoScopeCIStatus = "failing"
)

func (s TicketRepoScopeCIStatus) String() string {
	return string(s)
}

func (s TicketRepoScopeCIStatus) IsValid() bool {
	switch s {
	case TicketRepoScopeCIStatusPending, TicketRepoScopeCIStatusPassing, TicketRepoScopeCIStatusFailing:
		return true
	default:
		return false
	}
}

type MachineStatus string

const (
	MachineStatusOnline      MachineStatus = "online"
	MachineStatusOffline     MachineStatus = "offline"
	MachineStatusDegraded    MachineStatus = "degraded"
	MachineStatusMaintenance MachineStatus = "maintenance"
)

func (s MachineStatus) String() string {
	return string(s)
}

func (s MachineStatus) IsValid() bool {
	switch s {
	case MachineStatusOnline, MachineStatusOffline, MachineStatusDegraded, MachineStatusMaintenance:
		return true
	default:
		return false
	}
}

type AgentProviderAdapterType string

const (
	DefaultAgentProviderModelTemperature   float64 = 0
	DefaultAgentProviderModelMaxTokens             = 16384
	DefaultAgentProviderCostPerInputToken  float64 = 0
	DefaultAgentProviderCostPerOutputToken float64 = 0

	AgentProviderAdapterTypeClaudeCodeCLI  AgentProviderAdapterType = "claude-code-cli"
	AgentProviderAdapterTypeCodexAppServer AgentProviderAdapterType = "codex-app-server"
	AgentProviderAdapterTypeGeminiCLI      AgentProviderAdapterType = "gemini-cli"
	AgentProviderAdapterTypeCustom         AgentProviderAdapterType = "custom"
)

func (s AgentProviderAdapterType) String() string {
	return string(s)
}

func (s AgentProviderAdapterType) IsValid() bool {
	switch s {
	case AgentProviderAdapterTypeClaudeCodeCLI, AgentProviderAdapterTypeCodexAppServer, AgentProviderAdapterTypeGeminiCLI, AgentProviderAdapterTypeCustom:
		return true
	default:
		return false
	}
}

type AgentStatus string

const (
	DefaultAgentStatus                AgentStatus = AgentStatusIdle
	DefaultAgentTotalTokensUsed       int64       = 0
	DefaultAgentTotalTicketsCompleted             = 0

	AgentStatusIdle       AgentStatus = "idle"
	AgentStatusClaimed    AgentStatus = "claimed"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusPaused     AgentStatus = "paused"
	AgentStatusFailed     AgentStatus = "failed"
	AgentStatusTerminated AgentStatus = "terminated"
)

func (s AgentStatus) String() string {
	return string(s)
}

func (s AgentStatus) IsValid() bool {
	switch s {
	case AgentStatusIdle, AgentStatusClaimed, AgentStatusRunning, AgentStatusPaused, AgentStatusFailed, AgentStatusTerminated:
		return true
	default:
		return false
	}
}

type AgentRuntimePhase string

const (
	DefaultAgentRuntimePhase   AgentRuntimePhase = AgentRuntimePhaseNone
	AgentRuntimePhaseNone      AgentRuntimePhase = "none"
	AgentRuntimePhaseLaunching AgentRuntimePhase = "launching"
	AgentRuntimePhaseReady     AgentRuntimePhase = "ready"
	AgentRuntimePhaseExecuting AgentRuntimePhase = "executing"
	AgentRuntimePhaseFailed    AgentRuntimePhase = "failed"
)

func (s AgentRuntimePhase) String() string {
	return string(s)
}

func (s AgentRuntimePhase) IsValid() bool {
	switch s {
	case AgentRuntimePhaseNone, AgentRuntimePhaseLaunching, AgentRuntimePhaseReady, AgentRuntimePhaseExecuting, AgentRuntimePhaseFailed:
		return true
	default:
		return false
	}
}

type AgentRunStatus string

const (
	AgentRunStatusLaunching  AgentRunStatus = "launching"
	AgentRunStatusReady      AgentRunStatus = "ready"
	AgentRunStatusExecuting  AgentRunStatus = "executing"
	AgentRunStatusCompleted  AgentRunStatus = "completed"
	AgentRunStatusErrored    AgentRunStatus = "errored"
	AgentRunStatusTerminated AgentRunStatus = "terminated"
)

func (s AgentRunStatus) String() string {
	return string(s)
}

func (s AgentRunStatus) IsValid() bool {
	switch s {
	case AgentRunStatusLaunching, AgentRunStatusReady, AgentRunStatusExecuting, AgentRunStatusCompleted, AgentRunStatusErrored, AgentRunStatusTerminated:
		return true
	default:
		return false
	}
}

type AgentRuntimeControlState string

const (
	DefaultAgentRuntimeControlState        AgentRuntimeControlState = AgentRuntimeControlStateActive
	AgentRuntimeControlStateActive         AgentRuntimeControlState = "active"
	AgentRuntimeControlStatePauseRequested AgentRuntimeControlState = "pause_requested"
	AgentRuntimeControlStatePaused         AgentRuntimeControlState = "paused"
)

func (s AgentRuntimeControlState) String() string {
	return string(s)
}

func (s AgentRuntimeControlState) IsValid() bool {
	switch s {
	case AgentRuntimeControlStateActive, AgentRuntimeControlStatePauseRequested, AgentRuntimeControlStatePaused:
		return true
	default:
		return false
	}
}
