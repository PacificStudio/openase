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
	TypeUnknown               Type = "unknown"
	TypeTicketCreated         Type = "ticket.created"
	TypeTicketUpdated         Type = "ticket.updated"
	TypeTicketStatusChanged   Type = "ticket.status_changed"
	TypeTicketCompleted       Type = "ticket.completed"
	TypeTicketCancelled       Type = "ticket.cancelled"
	TypeTicketRetryScheduled  Type = "ticket.retry_scheduled"
	TypeTicketRetryPaused     Type = "ticket.retry_paused"
	TypeTicketBudgetExhausted Type = "ticket.budget_exhausted"
	TypeAgentClaimed          Type = "agent.claimed"
	TypeAgentLaunching        Type = "agent.launching"
	TypeAgentReady            Type = "agent.ready"
	TypeAgentPaused           Type = "agent.paused"
	TypeAgentFailed           Type = "agent.failed"
	TypeAgentCompleted        Type = "agent.completed"
	TypeAgentTerminated       Type = "agent.terminated"
	TypeHookStarted           Type = "hook.started"
	TypeHookPassed            Type = "hook.passed"
	TypeHookFailed            Type = "hook.failed"
	TypePROpened              Type = "pr.opened"
	TypePRMerged              Type = "pr.merged"
	TypePRClosed              Type = "pr.closed"
)

var canonicalCatalog = []CatalogEntry{
	{EventType: TypeTicketCreated, Label: "Ticket Created"},
	{EventType: TypeTicketUpdated, Label: "Ticket Updated"},
	{EventType: TypeTicketStatusChanged, Label: "Ticket Status Changed"},
	{EventType: TypeTicketCompleted, Label: "Ticket Completed"},
	{EventType: TypeTicketCancelled, Label: "Ticket Cancelled"},
	{EventType: TypeTicketRetryScheduled, Label: "Ticket Retry Scheduled"},
	{EventType: TypeTicketRetryPaused, Label: "Ticket Retry Paused"},
	{EventType: TypeTicketBudgetExhausted, Label: "Ticket Budget Exhausted"},
	{EventType: TypeAgentClaimed, Label: "Agent Claimed"},
	{EventType: TypeAgentLaunching, Label: "Agent Launching"},
	{EventType: TypeAgentReady, Label: "Agent Ready"},
	{EventType: TypeAgentPaused, Label: "Agent Paused"},
	{EventType: TypeAgentFailed, Label: "Agent Failed"},
	{EventType: TypeAgentCompleted, Label: "Agent Completed"},
	{EventType: TypeAgentTerminated, Label: "Agent Terminated"},
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
