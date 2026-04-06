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
	DefaultProjectStatus              ProjectStatus = ProjectStatusPlanned
	DefaultProjectMaxConcurrentAgents               = 0

	ProjectStatusBacklog    ProjectStatus = "Backlog"
	ProjectStatusPlanned    ProjectStatus = "Planned"
	ProjectStatusInProgress ProjectStatus = "In Progress"
	ProjectStatusCompleted  ProjectStatus = "Completed"
	ProjectStatusCanceled   ProjectStatus = "Canceled"
	ProjectStatusArchived   ProjectStatus = "Archived"
)

func (s ProjectStatus) String() string {
	return string(s)
}

func (s ProjectStatus) IsValid() bool {
	switch s {
	case ProjectStatusBacklog,
		ProjectStatusPlanned,
		ProjectStatusInProgress,
		ProjectStatusCompleted,
		ProjectStatusCanceled,
		ProjectStatusArchived:
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

type AgentProviderAvailabilityState string

const (
	AgentProviderAvailabilityStateUnknown     AgentProviderAvailabilityState = "unknown"
	AgentProviderAvailabilityStateAvailable   AgentProviderAvailabilityState = "available"
	AgentProviderAvailabilityStateUnavailable AgentProviderAvailabilityState = "unavailable"
	AgentProviderAvailabilityStateStale       AgentProviderAvailabilityState = "stale"
)

func (s AgentProviderAvailabilityState) String() string {
	return string(s)
}

func (s AgentProviderAvailabilityState) IsValid() bool {
	switch s {
	case AgentProviderAvailabilityStateUnknown,
		AgentProviderAvailabilityStateAvailable,
		AgentProviderAvailabilityStateUnavailable,
		AgentProviderAvailabilityStateStale:
		return true
	default:
		return false
	}
}

type AgentProviderCapabilityState string

const (
	AgentProviderCapabilityStateAvailable   AgentProviderCapabilityState = "available"
	AgentProviderCapabilityStateUnavailable AgentProviderCapabilityState = "unavailable"
	AgentProviderCapabilityStateUnsupported AgentProviderCapabilityState = "unsupported"
)

func (s AgentProviderCapabilityState) String() string {
	return string(s)
}

func (s AgentProviderCapabilityState) IsValid() bool {
	switch s {
	case AgentProviderCapabilityStateAvailable,
		AgentProviderCapabilityStateUnavailable,
		AgentProviderCapabilityStateUnsupported:
		return true
	default:
		return false
	}
}

type AgentProviderAdapterType string

const (
	DefaultAgentProviderModelTemperature   float64 = 0
	DefaultAgentProviderModelMaxTokens             = 16384
	DefaultAgentProviderMaxParallelRuns            = 0
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

type AgentProviderPermissionProfile string

const (
	DefaultAgentProviderPermissionProfile      = AgentProviderPermissionProfileUnrestricted
	AgentProviderPermissionProfileStandard     = "standard"
	AgentProviderPermissionProfileUnrestricted = "unrestricted"
)

func (s AgentProviderPermissionProfile) String() string {
	return string(s)
}

func (s AgentProviderPermissionProfile) IsValid() bool {
	switch s {
	case AgentProviderPermissionProfileStandard, AgentProviderPermissionProfileUnrestricted:
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

	AgentStatusIdle        AgentStatus = "idle"
	AgentStatusClaimed     AgentStatus = "claimed"
	AgentStatusRunning     AgentStatus = "running"
	AgentStatusPaused      AgentStatus = "paused"
	AgentStatusFailed      AgentStatus = "failed"
	AgentStatusInterrupted AgentStatus = "interrupted"
	AgentStatusTerminated  AgentStatus = "terminated"
)

func (s AgentStatus) String() string {
	return string(s)
}

type AgentRunCompletionSummaryStatus string

const (
	AgentRunCompletionSummaryStatusPending   AgentRunCompletionSummaryStatus = "pending"
	AgentRunCompletionSummaryStatusCompleted AgentRunCompletionSummaryStatus = "completed"
	AgentRunCompletionSummaryStatusFailed    AgentRunCompletionSummaryStatus = "failed"
)

func (s AgentRunCompletionSummaryStatus) String() string {
	return string(s)
}

func (s AgentRunCompletionSummaryStatus) IsValid() bool {
	switch s {
	case AgentRunCompletionSummaryStatusPending,
		AgentRunCompletionSummaryStatusCompleted,
		AgentRunCompletionSummaryStatusFailed:
		return true
	default:
		return false
	}
}

func (s AgentStatus) IsValid() bool {
	switch s {
	case AgentStatusIdle, AgentStatusClaimed, AgentStatusRunning, AgentStatusPaused, AgentStatusFailed, AgentStatusInterrupted, AgentStatusTerminated:
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
	AgentRunStatusLaunching   AgentRunStatus = "launching"
	AgentRunStatusReady       AgentRunStatus = "ready"
	AgentRunStatusExecuting   AgentRunStatus = "executing"
	AgentRunStatusCompleted   AgentRunStatus = "completed"
	AgentRunStatusErrored     AgentRunStatus = "errored"
	AgentRunStatusInterrupted AgentRunStatus = "interrupted"
	AgentRunStatusTerminated  AgentRunStatus = "terminated"
)

func (s AgentRunStatus) String() string {
	return string(s)
}

func (s AgentRunStatus) IsValid() bool {
	switch s {
	case AgentRunStatusLaunching, AgentRunStatusReady, AgentRunStatusExecuting, AgentRunStatusCompleted, AgentRunStatusErrored, AgentRunStatusInterrupted, AgentRunStatusTerminated:
		return true
	default:
		return false
	}
}

type AgentRuntimeControlState string

const (
	DefaultAgentRuntimeControlState            AgentRuntimeControlState = AgentRuntimeControlStateActive
	AgentRuntimeControlStateActive             AgentRuntimeControlState = "active"
	AgentRuntimeControlStateInterruptRequested AgentRuntimeControlState = "interrupt_requested"
	AgentRuntimeControlStatePauseRequested     AgentRuntimeControlState = "pause_requested"
	AgentRuntimeControlStatePaused             AgentRuntimeControlState = "paused"
	AgentRuntimeControlStateRetired            AgentRuntimeControlState = "retired"
)

func (s AgentRuntimeControlState) String() string {
	return string(s)
}

func (s AgentRuntimeControlState) IsValid() bool {
	switch s {
	case AgentRuntimeControlStateActive, AgentRuntimeControlStateInterruptRequested, AgentRuntimeControlStatePauseRequested, AgentRuntimeControlStatePaused, AgentRuntimeControlStateRetired:
		return true
	default:
		return false
	}
}
