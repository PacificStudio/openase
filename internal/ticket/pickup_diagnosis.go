package ticket

import domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"

const (
	PickupDiagnosisStateRunnable    = domain.PickupDiagnosisStateRunnable
	PickupDiagnosisStateWaiting     = domain.PickupDiagnosisStateWaiting
	PickupDiagnosisStateBlocked     = domain.PickupDiagnosisStateBlocked
	PickupDiagnosisStateRunning     = domain.PickupDiagnosisStateRunning
	PickupDiagnosisStateCompleted   = domain.PickupDiagnosisStateCompleted
	PickupDiagnosisStateUnavailable = domain.PickupDiagnosisStateUnavailable
)

const (
	PickupDiagnosisReasonReadyForPickup            = domain.PickupDiagnosisReasonReadyForPickup
	PickupDiagnosisReasonCompleted                 = domain.PickupDiagnosisReasonCompleted
	PickupDiagnosisReasonRunningCurrentRun         = domain.PickupDiagnosisReasonRunningCurrentRun
	PickupDiagnosisReasonRetryBackoff              = domain.PickupDiagnosisReasonRetryBackoff
	PickupDiagnosisReasonRetryPausedRepeatedStalls = domain.PickupDiagnosisReasonRetryPausedRepeatedStalls
	PickupDiagnosisReasonRetryPausedBudget         = domain.PickupDiagnosisReasonRetryPausedBudget
	PickupDiagnosisReasonRetryPausedInterrupted    = domain.PickupDiagnosisReasonRetryPausedInterrupted
	PickupDiagnosisReasonRetryPausedUser           = domain.PickupDiagnosisReasonRetryPausedUser
	PickupDiagnosisReasonBlockedDependency         = domain.PickupDiagnosisReasonBlockedDependency
	PickupDiagnosisReasonNoMatchingActiveWorkflow  = domain.PickupDiagnosisReasonNoMatchingActiveWorkflow
	PickupDiagnosisReasonWorkflowInactive          = domain.PickupDiagnosisReasonWorkflowInactive
	PickupDiagnosisReasonWorkflowMissingAgent      = domain.PickupDiagnosisReasonWorkflowMissingAgent
	PickupDiagnosisReasonAgentMissing              = domain.PickupDiagnosisReasonAgentMissing
	PickupDiagnosisReasonAgentInterruptRequested   = domain.PickupDiagnosisReasonAgentInterruptRequested
	PickupDiagnosisReasonAgentPaused               = domain.PickupDiagnosisReasonAgentPaused
	PickupDiagnosisReasonAgentPauseRequested       = domain.PickupDiagnosisReasonAgentPauseRequested
	PickupDiagnosisReasonProviderMissing           = domain.PickupDiagnosisReasonProviderMissing
	PickupDiagnosisReasonMachineMissing            = domain.PickupDiagnosisReasonMachineMissing
	PickupDiagnosisReasonMachineOffline            = domain.PickupDiagnosisReasonMachineOffline
	PickupDiagnosisReasonProviderUnknown           = domain.PickupDiagnosisReasonProviderUnknown
	PickupDiagnosisReasonProviderStale             = domain.PickupDiagnosisReasonProviderStale
	PickupDiagnosisReasonProviderUnavailable       = domain.PickupDiagnosisReasonProviderUnavailable
	PickupDiagnosisReasonWorkflowConcurrencyFull   = domain.PickupDiagnosisReasonWorkflowConcurrencyFull
	PickupDiagnosisReasonProjectConcurrencyFull    = domain.PickupDiagnosisReasonProjectConcurrencyFull
	PickupDiagnosisReasonProviderConcurrencyFull   = domain.PickupDiagnosisReasonProviderConcurrencyFull
	PickupDiagnosisReasonStatusCapacityFull        = domain.PickupDiagnosisReasonStatusCapacityFull
	PickupDiagnosisReasonSchedulerUnavailable      = domain.PickupDiagnosisReasonSchedulerUnavailable
)

const (
	PickupDiagnosisReasonSeverityInfo    = domain.PickupDiagnosisReasonSeverityInfo
	PickupDiagnosisReasonSeverityWarning = domain.PickupDiagnosisReasonSeverityWarning
	PickupDiagnosisReasonSeverityError   = domain.PickupDiagnosisReasonSeverityError
)
