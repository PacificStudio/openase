package notification

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	"github.com/google/uuid"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

// RuleEventType identifies a supported notification subscription event.
type RuleEventType string

const (
	RuleEventTypeTicketCreated         RuleEventType = RuleEventType(activityevent.TypeTicketCreated)
	RuleEventTypeTicketUpdated         RuleEventType = RuleEventType(activityevent.TypeTicketUpdated)
	RuleEventTypeTicketStatusChanged   RuleEventType = RuleEventType(activityevent.TypeTicketStatusChanged)
	RuleEventTypeTicketCompleted       RuleEventType = RuleEventType(activityevent.TypeTicketCompleted)
	RuleEventTypeTicketCancelled       RuleEventType = RuleEventType(activityevent.TypeTicketCancelled)
	RuleEventTypeTicketRetryScheduled  RuleEventType = RuleEventType(activityevent.TypeTicketRetryScheduled)
	RuleEventTypeTicketRetryPaused     RuleEventType = RuleEventType(activityevent.TypeTicketRetryPaused)
	RuleEventTypeTicketBudgetExhausted RuleEventType = RuleEventType(activityevent.TypeTicketBudgetExhausted)
	RuleEventTypeAgentClaimed          RuleEventType = RuleEventType(activityevent.TypeAgentClaimed)
	RuleEventTypeAgentFailed           RuleEventType = RuleEventType(activityevent.TypeAgentFailed)
	RuleEventTypeHookFailed            RuleEventType = RuleEventType(activityevent.TypeHookFailed)
	RuleEventTypeHookPassed            RuleEventType = RuleEventType(activityevent.TypeHookPassed)
	RuleEventTypePROpened              RuleEventType = RuleEventType(activityevent.TypePROpened)
	RuleEventTypePRMerged              RuleEventType = RuleEventType(activityevent.TypePRMerged)
	RuleEventTypePRClosed              RuleEventType = RuleEventType(activityevent.TypePRClosed)
	RuleEventTypeTicketStalled         RuleEventType = "ticket.stalled"
	RuleEventTypeTicketErrorRateHigh   RuleEventType = "ticket.error_rate_high"
	RuleEventTypeMachineOffline        RuleEventType = "machine.offline"
	RuleEventTypeMachineOnline         RuleEventType = "machine.online"
	RuleEventTypeMachineDegraded       RuleEventType = "machine.degraded"
	RuleEventTypeConnectorSyncError    RuleEventType = "connector.sync_error"
	RuleEventTypeBudgetThreshold       RuleEventType = "budget.threshold"
)

// RuleEventCatalogEntry describes a selectable event type for UI/API consumers.
type RuleEventCatalogEntry struct {
	EventType       RuleEventType
	Label           string
	DefaultTemplate string
}

var supportedRuleEvents = []RuleEventCatalogEntry{
	{
		EventType:       RuleEventTypeTicketCreated,
		Label:           "Ticket Created",
		DefaultTemplate: "Ticket created: {{ ticket.identifier }}\n{{ ticket.title }}\nStatus: {{ ticket.status_name }}\nPriority: {{ ticket.priority }}",
	},
	{
		EventType:       RuleEventTypeTicketUpdated,
		Label:           "Ticket Updated",
		DefaultTemplate: "Ticket updated: {{ ticket.identifier }}\n{{ ticket.title }}\nStatus: {{ ticket.status_name }}",
	},
	{
		EventType:       RuleEventTypeTicketStatusChanged,
		Label:           "Ticket Status Changed",
		DefaultTemplate: "Ticket status changed: {{ ticket.identifier }}\n{{ ticket.title }}\nNew status: {{ new_status }}",
	},
	{
		EventType:       RuleEventTypeTicketCompleted,
		Label:           "Ticket Completed",
		DefaultTemplate: "{{ ticket.identifier }} completed",
	},
	{
		EventType:       RuleEventTypeTicketCancelled,
		Label:           "Ticket Cancelled",
		DefaultTemplate: "{{ ticket.identifier }} cancelled",
	},
	{
		EventType:       RuleEventTypeTicketRetryScheduled,
		Label:           "Ticket Retry Scheduled",
		DefaultTemplate: "{{ ticket.identifier }} will retry at {{ next_retry_at }}",
	},
	{
		EventType:       RuleEventTypeTicketRetryPaused,
		Label:           "Ticket Retry Paused",
		DefaultTemplate: "{{ ticket.identifier }} retry paused: {{ pause_reason }}",
	},
	{
		EventType:       RuleEventTypeTicketBudgetExhausted,
		Label:           "Ticket Budget Exhausted",
		DefaultTemplate: "{{ ticket.identifier }} budget exhausted (${{ cost_amount }}/${{ budget_usd }})",
	},
	{
		EventType:       RuleEventTypeAgentClaimed,
		Label:           "Agent Claimed",
		DefaultTemplate: "{{ agent.name }} claimed {{ ticket.identifier }}",
	},
	{
		EventType:       RuleEventTypeAgentFailed,
		Label:           "Agent Failed",
		DefaultTemplate: "Agent {{ agent.name }} failed {{ ticket.identifier }}: {{ error }}",
	},
	{
		EventType:       RuleEventTypeHookFailed,
		Label:           "Hook Failed",
		DefaultTemplate: "{{ ticket.identifier }} hook {{ hook_name }} failed: {{ error }}",
	},
	{
		EventType:       RuleEventTypeHookPassed,
		Label:           "Hook Passed",
		DefaultTemplate: "{{ ticket.identifier }} hook {{ hook_name }} passed",
	},
	{
		EventType:       RuleEventTypePROpened,
		Label:           "PR Opened",
		DefaultTemplate: "{{ ticket.identifier }} PR opened: {{ pull_request_url }}",
	},
	{
		EventType:       RuleEventTypePRMerged,
		Label:           "PR Merged",
		DefaultTemplate: "{{ ticket.identifier }} PR merged: {{ pull_request_url }}",
	},
	{
		EventType:       RuleEventTypePRClosed,
		Label:           "PR Closed",
		DefaultTemplate: "{{ ticket.identifier }} PR closed: {{ pull_request_url }}",
	},
	{
		EventType:       RuleEventTypeTicketStalled,
		Label:           "Ticket Stalled",
		DefaultTemplate: "{{ ticket.identifier }} agent is stalled",
	},
	{
		EventType:       RuleEventTypeTicketErrorRateHigh,
		Label:           "Ticket Error Rate High",
		DefaultTemplate: "{{ ticket.identifier }} failed {{ consecutive_errors }} times",
	},
	{
		EventType:       RuleEventTypeMachineOffline,
		Label:           "Machine Offline",
		DefaultTemplate: "Machine {{ machine.name }} is offline",
	},
	{
		EventType:       RuleEventTypeMachineOnline,
		Label:           "Machine Online",
		DefaultTemplate: "Machine {{ machine.name }} is online",
	},
	{
		EventType:       RuleEventTypeMachineDegraded,
		Label:           "Machine Degraded",
		DefaultTemplate: "Machine {{ machine.name }} degraded: {{ disk_free_gb }}GB free",
	},
	{
		EventType:       RuleEventTypeConnectorSyncError,
		Label:           "Connector Sync Error",
		DefaultTemplate: "Connector {{ connector.name }} sync failed: {{ error }}",
	},
	{
		EventType:       RuleEventTypeBudgetThreshold,
		Label:           "Budget Threshold",
		DefaultTemplate: "Project {{ project.name }} spent ${{ cost_usd }}",
	},
}

var supportedRuleEventIndex = func() map[RuleEventType]RuleEventCatalogEntry {
	index := make(map[RuleEventType]RuleEventCatalogEntry, len(supportedRuleEvents))
	for _, item := range supportedRuleEvents {
		index[item.EventType] = item
	}
	return index
}()

// Rule stores a persisted notification subscription.
type Rule struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	ChannelID uuid.UUID
	Name      string
	EventType RuleEventType
	Filter    map[string]any
	Template  string
	IsEnabled bool
	CreatedAt time.Time
	Channel   Channel
}

// RuleInput carries raw rule create values before parsing.
type RuleInput struct {
	Name      string         `json:"name"`
	EventType string         `json:"event_type"`
	Filter    map[string]any `json:"filter"`
	ChannelID string         `json:"channel_id"`
	Template  string         `json:"template"`
	IsEnabled *bool          `json:"is_enabled"`
}

// RulePatchInput carries raw rule patch values before parsing.
type RulePatchInput struct {
	Name      *string         `json:"name"`
	EventType *string         `json:"event_type"`
	Filter    *map[string]any `json:"filter"`
	ChannelID *string         `json:"channel_id"`
	Template  *string         `json:"template"`
	IsEnabled *bool           `json:"is_enabled"`
}

// CreateRuleInput is the parsed create command.
type CreateRuleInput struct {
	ProjectID uuid.UUID
	Name      string
	EventType RuleEventType
	Filter    map[string]any
	ChannelID uuid.UUID
	Template  string
	IsEnabled bool
}

// UpdateRuleInput is the parsed patch command.
type UpdateRuleInput struct {
	RuleID    uuid.UUID
	Name      Optional[string]
	EventType Optional[RuleEventType]
	Filter    Optional[map[string]any]
	ChannelID Optional[uuid.UUID]
	Template  Optional[string]
	IsEnabled Optional[bool]
}

// SupportedRuleEvents returns the event catalog that UI/API clients can use.
func SupportedRuleEvents() []RuleEventCatalogEntry {
	items := make([]RuleEventCatalogEntry, 0, len(supportedRuleEvents))
	items = append(items, supportedRuleEvents...)
	return items
}

// ParseRuleEventType validates a raw event type string against the supported catalog.
func ParseRuleEventType(raw string) (RuleEventType, error) {
	eventType := RuleEventType(strings.TrimSpace(raw))
	if _, ok := supportedRuleEventIndex[eventType]; !ok {
		return "", fmt.Errorf("event_type %q is not supported", strings.TrimSpace(raw))
	}

	return eventType, nil
}

func (t RuleEventType) String() string {
	return string(t)
}

// DefaultTemplate returns the built-in template for the event type.
func (t RuleEventType) DefaultTemplate() string {
	if item, ok := supportedRuleEventIndex[t]; ok {
		return item.DefaultTemplate
	}
	return ""
}

// ParseCreateRule validates an incoming rule create request.
func ParseCreateRule(projectID uuid.UUID, raw RuleInput) (CreateRuleInput, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return CreateRuleInput{}, fmt.Errorf("name must not be empty")
	}

	eventType, err := ParseRuleEventType(raw.EventType)
	if err != nil {
		return CreateRuleInput{}, err
	}

	channelID, err := parseRuleChannelID(raw.ChannelID)
	if err != nil {
		return CreateRuleInput{}, err
	}

	filter, err := normalizeRuleFilter(raw.Filter)
	if err != nil {
		return CreateRuleInput{}, err
	}

	isEnabled := true
	if raw.IsEnabled != nil {
		isEnabled = *raw.IsEnabled
	}

	return CreateRuleInput{
		ProjectID: projectID,
		Name:      name,
		EventType: eventType,
		Filter:    filter,
		ChannelID: channelID,
		Template:  normalizeTemplate(raw.Template),
		IsEnabled: isEnabled,
	}, nil
}

// ParseUpdateRule validates a raw rule patch request.
func ParseUpdateRule(ruleID uuid.UUID, raw RulePatchInput) (UpdateRuleInput, error) {
	input := UpdateRuleInput{RuleID: ruleID}

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return UpdateRuleInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = Some(name)
	}

	if raw.EventType != nil {
		eventType, err := ParseRuleEventType(*raw.EventType)
		if err != nil {
			return UpdateRuleInput{}, err
		}
		input.EventType = Some(eventType)
	}

	if raw.Filter != nil {
		filter, err := normalizeRuleFilter(*raw.Filter)
		if err != nil {
			return UpdateRuleInput{}, err
		}
		input.Filter = Some(filter)
	}

	if raw.ChannelID != nil {
		channelID, err := parseRuleChannelID(*raw.ChannelID)
		if err != nil {
			return UpdateRuleInput{}, err
		}
		input.ChannelID = Some(channelID)
	}

	if raw.Template != nil {
		input.Template = Some(normalizeTemplate(*raw.Template))
	}

	if raw.IsEnabled != nil {
		input.IsEnabled = Some(*raw.IsEnabled)
	}

	if !input.Name.Set && !input.EventType.Set && !input.Filter.Set && !input.ChannelID.Set && !input.Template.Set && !input.IsEnabled.Set {
		return UpdateRuleInput{}, fmt.Errorf("patch request must update at least one field")
	}

	return input, nil
}

// Matches reports whether the rule filter matches the event context.
func (r Rule) Matches(context map[string]any) bool {
	return matchRuleFilter(r.Filter, context)
}

// RenderMessage renders the rule template or falls back to the event default.
func (r Rule) RenderMessage(context map[string]any) (Message, error) {
	templateText := strings.TrimSpace(r.Template)
	if templateText == "" {
		templateText = r.EventType.DefaultTemplate()
	}

	rendered, err := renderRuleTemplate(templateText, context)
	if err != nil {
		return Message{}, err
	}

	return messageFromRenderedText(rendered), nil
}

func normalizeRuleFilter(raw map[string]any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}

	filter, err := cloneRawConfig(raw)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

func normalizeTemplate(raw string) string {
	return strings.TrimSpace(raw)
}

func parseRuleChannelID(raw string) (uuid.UUID, error) {
	channelID, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("channel_id must be a valid UUID")
	}
	return channelID, nil
}

func matchRuleFilter(filter map[string]any, context map[string]any) bool {
	if len(filter) == 0 {
		return true
	}
	for key, want := range filter {
		actual, ok := lookupFilterValue(context, key)
		if !ok || !matchRuleValue(want, actual) {
			return false
		}
	}
	return true
}

func lookupFilterValue(context map[string]any, key string) (any, bool) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return nil, false
	}
	if value, ok := context[trimmed]; ok {
		return value, true
	}
	if !strings.Contains(trimmed, ".") {
		return nil, false
	}

	current := any(context)
	for _, part := range strings.Split(trimmed, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := object[part]
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func matchRuleValue(want any, actual any) bool {
	switch typedWant := want.(type) {
	case map[string]any:
		typedActual, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for key, nestedWant := range typedWant {
			nestedActual, ok := typedActual[key]
			if !ok || !matchRuleValue(nestedWant, nestedActual) {
				return false
			}
		}
		return true
	case []any:
		typedActual, ok := actual.([]any)
		if !ok || len(typedWant) != len(typedActual) {
			return false
		}
		for idx := range typedWant {
			if !matchRuleValue(typedWant[idx], typedActual[idx]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(want, actual)
	}
}

func renderRuleTemplate(templateText string, context map[string]any) (string, error) {
	if strings.TrimSpace(templateText) == "" {
		return "", nil
	}

	template, err := gonja.FromString(templateText)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	rendered, err := template.ExecuteToString(exec.NewContext(context))
	if err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}

	return strings.TrimSpace(rendered), nil
}

func messageFromRenderedText(rendered string) Message {
	trimmed := strings.TrimSpace(rendered)
	if trimmed == "" {
		return Message{}
	}

	lines := strings.Split(trimmed, "\n")
	title := strings.TrimSpace(lines[0])
	body := ""
	if len(lines) > 1 {
		body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}

	return Message{
		Title: title,
		Body:  body,
		Level: "info",
	}
}
