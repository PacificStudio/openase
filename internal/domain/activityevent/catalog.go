package activityevent

import (
	"fmt"
	"log/slog"
	"strings"
)

// Type identifies a canonical business activity event.
type Type string

// CatalogEntry describes one canonical activity event type.
type CatalogEntry struct {
	EventType Type
	Label     string
}

const (
	TypeUnknown                          Type = "unknown"
	TypeProjectCreated                   Type = "project.created"
	TypeProjectUpdated                   Type = "project.updated"
	TypeProjectArchived                  Type = "project.archived"
	TypeProjectStatusChanged             Type = "project.status_changed"
	TypeProjectProviderChanged           Type = "project.default_provider_changed"
	TypeProjectConcurrencyChanged        Type = "project.concurrency_changed"
	TypeProjectRepoCreated               Type = "project_repo.created"
	TypeProjectRepoUpdated               Type = "project_repo.updated"
	TypeProjectRepoDeleted               Type = "project_repo.deleted"
	TypeProjectUpdateThreadCreated       Type = "project_update_thread.created"
	TypeProjectUpdateThreadEdited        Type = "project_update_thread.edited"
	TypeProjectUpdateThreadDeleted       Type = "project_update_thread.deleted"
	TypeProjectUpdateThreadStatusChanged Type = "project_update_thread.status_changed"
	TypeProjectUpdateCommentCreated      Type = "project_update_comment.created"
	TypeProjectUpdateCommentEdited       Type = "project_update_comment.edited"
	TypeProjectUpdateCommentDeleted      Type = "project_update_comment.deleted"
	TypeTicketStatusCreated              Type = "ticket_status.created"
	TypeTicketStatusUpdated              Type = "ticket_status.updated"
	TypeTicketStatusReordered            Type = "ticket_status.reordered"
	TypeTicketStatusConcurrencyChanged   Type = "ticket_status.concurrency_changed"
	TypeTicketStatusDeleted              Type = "ticket_status.deleted"
	TypeTicketStatusReset                Type = "ticket_status.reset"
	TypeWorkflowCreated                  Type = "workflow.created"
	TypeWorkflowUpdated                  Type = "workflow.updated"
	TypeWorkflowActivated                Type = "workflow.activated"
	TypeWorkflowDeactivated              Type = "workflow.deactivated"
	TypeWorkflowDeleted                  Type = "workflow.deleted"
	TypeWorkflowHarnessUpdated           Type = "workflow.harness_updated"
	TypeWorkflowHooksUpdated             Type = "workflow.hooks_updated"
	TypeWorkflowAgentChanged             Type = "workflow.agent_changed"
	TypeWorkflowPickupStatusesChanged    Type = "workflow.pickup_statuses_changed"
	TypeWorkflowFinishStatusesChanged    Type = "workflow.finish_statuses_changed"
	TypeWorkflowConcurrencyChanged       Type = "workflow.concurrency_changed"
	TypeWorkflowRetryPolicyChanged       Type = "workflow.retry_policy_changed"
	TypeWorkflowTimeoutChanged           Type = "workflow.timeout_changed"
	TypeProviderCreated                  Type = "provider.created"
	TypeProviderUpdated                  Type = "provider.updated"
	TypeProviderAvailabilityChanged      Type = "provider.availability_changed"
	TypeProviderMachineBindingChanged    Type = "provider.machine_binding_changed"
	TypeProviderRateLimitUpdated         Type = "provider.rate_limit_updated"
	TypeAgentCreated                     Type = "agent.created"
	TypeAgentUpdated                     Type = "agent.updated"
	TypeAgentResumed                     Type = "agent.resumed"
	TypeAgentDeleted                     Type = "agent.deleted"
	TypeScheduledJobCreated              Type = "scheduled_job.created"
	TypeScheduledJobUpdated              Type = "scheduled_job.updated"
	TypeScheduledJobEnabled              Type = "scheduled_job.enabled"
	TypeScheduledJobDisabled             Type = "scheduled_job.disabled"
	TypeScheduledJobDeleted              Type = "scheduled_job.deleted"
	TypeScheduledJobTriggered            Type = "scheduled_job.triggered"
	TypeTicketCommentCreated             Type = "ticket_comment.created"
	TypeTicketCommentEdited              Type = "ticket_comment.edited"
	TypeTicketCommentDeleted             Type = "ticket_comment.deleted"
	TypeTicketCreated                    Type = "ticket.created"
	TypeTicketUpdated                    Type = "ticket.updated"
	TypeTicketArchived                   Type = "ticket.archived"
	TypeTicketUnarchived                 Type = "ticket.unarchived"
	TypeTicketStatusChanged              Type = "ticket.status_changed"
	TypeTicketCompleted                  Type = "ticket.completed"
	TypeTicketCancelled                  Type = "ticket.cancelled"
	TypeTicketRetryScheduled             Type = "ticket.retry_scheduled"
	TypeTicketRetryPaused                Type = "ticket.retry_paused"
	TypeTicketRetryResumed               Type = "ticket.retry_resumed"
	TypeTicketBudgetExhausted            Type = "ticket.budget_exhausted"
	TypeAgentClaimed                     Type = "agent.claimed"
	TypeAgentLaunching                   Type = "agent.launching"
	TypeAgentReady                       Type = "agent.ready"
	TypeAgentExecuting                   Type = "agent.executing"
	TypeAgentPaused                      Type = "agent.paused"
	TypeAgentFailed                      Type = "agent.failed"
	TypeAgentCompleted                   Type = "agent.completed"
	TypeAgentTerminated                  Type = "agent.terminated"
	TypeMachineConnected                 Type = "machine.connected"
	TypeMachineDisconnected              Type = "machine.disconnected"
	TypeMachineReconnected               Type = "machine.reconnected"
	TypeMachineDaemonAuthFailed          Type = "machine.daemon_auth_failed"
	TypeRuntimeFallbackToSSH             Type = "runtime.fallback_to_ssh"
	TypeHookStarted                      Type = "hook.started"
	TypeHookPassed                       Type = "hook.passed"
	TypeHookFailed                       Type = "hook.failed"
	TypePROpened                         Type = "pr.opened"
	TypePRMerged                         Type = "pr.merged"
	TypePRClosed                         Type = "pr.closed"
)

var canonicalCatalog = []CatalogEntry{
	{EventType: TypeProjectCreated, Label: "Project Created"},
	{EventType: TypeProjectUpdated, Label: "Project Updated"},
	{EventType: TypeProjectArchived, Label: "Project Archived"},
	{EventType: TypeProjectStatusChanged, Label: "Project Status Changed"},
	{EventType: TypeProjectProviderChanged, Label: "Project Default Provider Changed"},
	{EventType: TypeProjectConcurrencyChanged, Label: "Project Concurrency Changed"},
	{EventType: TypeProjectRepoCreated, Label: "Project Repo Created"},
	{EventType: TypeProjectRepoUpdated, Label: "Project Repo Updated"},
	{EventType: TypeProjectRepoDeleted, Label: "Project Repo Deleted"},
	{EventType: TypeProjectUpdateThreadCreated, Label: "Project Update Thread Created"},
	{EventType: TypeProjectUpdateThreadEdited, Label: "Project Update Thread Edited"},
	{EventType: TypeProjectUpdateThreadDeleted, Label: "Project Update Thread Deleted"},
	{EventType: TypeProjectUpdateThreadStatusChanged, Label: "Project Update Thread Status Changed"},
	{EventType: TypeProjectUpdateCommentCreated, Label: "Project Update Comment Created"},
	{EventType: TypeProjectUpdateCommentEdited, Label: "Project Update Comment Edited"},
	{EventType: TypeProjectUpdateCommentDeleted, Label: "Project Update Comment Deleted"},
	{EventType: TypeTicketStatusCreated, Label: "Ticket Status Created"},
	{EventType: TypeTicketStatusUpdated, Label: "Ticket Status Updated"},
	{EventType: TypeTicketStatusReordered, Label: "Ticket Status Reordered"},
	{EventType: TypeTicketStatusConcurrencyChanged, Label: "Ticket Status Concurrency Changed"},
	{EventType: TypeTicketStatusDeleted, Label: "Ticket Status Deleted"},
	{EventType: TypeTicketStatusReset, Label: "Ticket Status Reset"},
	{EventType: TypeWorkflowCreated, Label: "Workflow Created"},
	{EventType: TypeWorkflowUpdated, Label: "Workflow Updated"},
	{EventType: TypeWorkflowActivated, Label: "Workflow Activated"},
	{EventType: TypeWorkflowDeactivated, Label: "Workflow Deactivated"},
	{EventType: TypeWorkflowDeleted, Label: "Workflow Deleted"},
	{EventType: TypeWorkflowHarnessUpdated, Label: "Workflow Harness Updated"},
	{EventType: TypeWorkflowHooksUpdated, Label: "Workflow Hooks Updated"},
	{EventType: TypeWorkflowAgentChanged, Label: "Workflow Agent Changed"},
	{EventType: TypeWorkflowPickupStatusesChanged, Label: "Workflow Pickup Statuses Changed"},
	{EventType: TypeWorkflowFinishStatusesChanged, Label: "Workflow Finish Statuses Changed"},
	{EventType: TypeWorkflowConcurrencyChanged, Label: "Workflow Concurrency Changed"},
	{EventType: TypeWorkflowRetryPolicyChanged, Label: "Workflow Retry Policy Changed"},
	{EventType: TypeWorkflowTimeoutChanged, Label: "Workflow Timeout Changed"},
	{EventType: TypeProviderCreated, Label: "Provider Created"},
	{EventType: TypeProviderUpdated, Label: "Provider Updated"},
	{EventType: TypeProviderAvailabilityChanged, Label: "Provider Availability Changed"},
	{EventType: TypeProviderMachineBindingChanged, Label: "Provider Machine Binding Changed"},
	{EventType: TypeProviderRateLimitUpdated, Label: "Provider Rate Limit Updated"},
	{EventType: TypeAgentCreated, Label: "Agent Created"},
	{EventType: TypeAgentUpdated, Label: "Agent Updated"},
	{EventType: TypeAgentResumed, Label: "Agent Resumed"},
	{EventType: TypeAgentDeleted, Label: "Agent Deleted"},
	{EventType: TypeScheduledJobCreated, Label: "Scheduled Job Created"},
	{EventType: TypeScheduledJobUpdated, Label: "Scheduled Job Updated"},
	{EventType: TypeScheduledJobEnabled, Label: "Scheduled Job Enabled"},
	{EventType: TypeScheduledJobDisabled, Label: "Scheduled Job Disabled"},
	{EventType: TypeScheduledJobDeleted, Label: "Scheduled Job Deleted"},
	{EventType: TypeScheduledJobTriggered, Label: "Scheduled Job Triggered"},
	{EventType: TypeTicketCommentCreated, Label: "Ticket Comment Created"},
	{EventType: TypeTicketCommentEdited, Label: "Ticket Comment Edited"},
	{EventType: TypeTicketCommentDeleted, Label: "Ticket Comment Deleted"},
	{EventType: TypeTicketCreated, Label: "Ticket Created"},
	{EventType: TypeTicketUpdated, Label: "Ticket Updated"},
	{EventType: TypeTicketArchived, Label: "Ticket Archived"},
	{EventType: TypeTicketUnarchived, Label: "Ticket Unarchived"},
	{EventType: TypeTicketStatusChanged, Label: "Ticket Status Changed"},
	{EventType: TypeTicketCompleted, Label: "Ticket Completed"},
	{EventType: TypeTicketCancelled, Label: "Ticket Cancelled"},
	{EventType: TypeTicketRetryScheduled, Label: "Ticket Retry Scheduled"},
	{EventType: TypeTicketRetryPaused, Label: "Ticket Retry Paused"},
	{EventType: TypeTicketRetryResumed, Label: "Ticket Retry Resumed"},
	{EventType: TypeTicketBudgetExhausted, Label: "Ticket Budget Exhausted"},
	{EventType: TypeAgentClaimed, Label: "Agent Claimed"},
	{EventType: TypeAgentLaunching, Label: "Agent Launching"},
	{EventType: TypeAgentReady, Label: "Agent Ready"},
	{EventType: TypeAgentExecuting, Label: "Agent Executing"},
	{EventType: TypeAgentPaused, Label: "Agent Paused"},
	{EventType: TypeAgentFailed, Label: "Agent Failed"},
	{EventType: TypeAgentCompleted, Label: "Agent Completed"},
	{EventType: TypeAgentTerminated, Label: "Agent Terminated"},
	{EventType: TypeMachineConnected, Label: "Machine Connected"},
	{EventType: TypeMachineDisconnected, Label: "Machine Disconnected"},
	{EventType: TypeMachineReconnected, Label: "Machine Reconnected"},
	{EventType: TypeMachineDaemonAuthFailed, Label: "Machine Daemon Auth Failed"},
	{EventType: TypeRuntimeFallbackToSSH, Label: "Runtime Fallback To SSH"},
	{EventType: TypeHookStarted, Label: "Hook Started"},
	{EventType: TypeHookPassed, Label: "Hook Passed"},
	{EventType: TypeHookFailed, Label: "Hook Failed"},
	{EventType: TypePROpened, Label: "PR Opened"},
	{EventType: TypePRMerged, Label: "PR Merged"},
	{EventType: TypePRClosed, Label: "PR Closed"},
}

var canonicalIndex = func() map[Type]CatalogEntry {
	index := make(map[Type]CatalogEntry, len(canonicalCatalog))
	for _, item := range canonicalCatalog {
		index[item.EventType] = item
	}
	return index
}()

func (t Type) String() string {
	return string(t)
}

func (t Type) IsHook() bool {
	switch t {
	case TypeHookStarted, TypeHookPassed, TypeHookFailed:
		return true
	default:
		return false
	}
}

func (t Type) IsTicketComment() bool {
	switch t {
	case TypeTicketCommentCreated, TypeTicketCommentEdited, TypeTicketCommentDeleted:
		return true
	default:
		return false
	}
}

func Catalog() []CatalogEntry {
	items := make([]CatalogEntry, 0, len(canonicalCatalog))
	items = append(items, canonicalCatalog...)
	return items
}

func ParseRawType(raw string) (Type, error) {
	trimmed := Type(strings.TrimSpace(raw))
	if _, ok := canonicalIndex[trimmed]; !ok {
		return "", fmt.Errorf("activity event type %q is not supported", strings.TrimSpace(raw))
	}
	return trimmed, nil
}

func MustParseType(raw string) Type {
	eventType, err := ParseRawType(raw)
	if err != nil {
		panic(err)
	}
	return eventType
}

// ParseStoredType maps persisted data into the canonical catalog.
// Unknown historical values stay readable but surface as unknown with diagnostics.
func ParseStoredType(raw string, logger *slog.Logger) (Type, string) {
	eventType, err := ParseRawType(raw)
	if err == nil {
		return eventType, ""
	}

	if logger == nil {
		logger = slog.Default()
	}
	trimmed := strings.TrimSpace(raw)
	logger.Warn("unknown persisted activity event type", "event_type", trimmed, "error", err)
	return TypeUnknown, trimmed
}
