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

type RuleEventGroup string

const (
	RuleEventGroupTicketLifecycle   RuleEventGroup = "Ticket lifecycle"
	RuleEventGroupTicketReliability RuleEventGroup = "Ticket reliability"
	RuleEventGroupAgentExecution    RuleEventGroup = "Agent / execution"
	RuleEventGroupHookIntegration   RuleEventGroup = "Hook / integration"
	RuleEventGroupPRReview          RuleEventGroup = "PR / review"
	RuleEventGroupInfrastructure    RuleEventGroup = "Infrastructure"
)

type RuleEventLevel string

const (
	RuleEventLevelInfo     RuleEventLevel = "info"
	RuleEventLevelWarning  RuleEventLevel = "warning"
	RuleEventLevelCritical RuleEventLevel = "critical"
)

func (g RuleEventGroup) String() string { return string(g) }

func (l RuleEventLevel) String() string { return string(l) }

const (
	RuleEventTypeTicketCreated         RuleEventType = RuleEventType(activityevent.TypeTicketCreated)
	RuleEventTypeTicketUpdated         RuleEventType = RuleEventType(activityevent.TypeTicketUpdated)
	RuleEventTypeTicketStatusChanged   RuleEventType = RuleEventType(activityevent.TypeTicketStatusChanged)
	RuleEventTypeTicketCompleted       RuleEventType = RuleEventType(activityevent.TypeTicketCompleted)
	RuleEventTypeTicketCancelled       RuleEventType = RuleEventType(activityevent.TypeTicketCancelled)
	RuleEventTypeTicketRetryScheduled  RuleEventType = RuleEventType(activityevent.TypeTicketRetryScheduled)
	RuleEventTypeTicketRetryResumed    RuleEventType = RuleEventType(activityevent.TypeTicketRetryResumed)
	RuleEventTypeTicketRetryPaused     RuleEventType = RuleEventType(activityevent.TypeTicketRetryPaused)
	RuleEventTypeTicketBudgetExhausted RuleEventType = RuleEventType(activityevent.TypeTicketBudgetExhausted)
	RuleEventTypeAgentClaimed          RuleEventType = RuleEventType(activityevent.TypeAgentClaimed)
	RuleEventTypeAgentFailed           RuleEventType = RuleEventType(activityevent.TypeAgentFailed)
	RuleEventTypeHookFailed            RuleEventType = RuleEventType(activityevent.TypeHookFailed)
	RuleEventTypeHookPassed            RuleEventType = RuleEventType(activityevent.TypeHookPassed)
	RuleEventTypePROpened              RuleEventType = RuleEventType(activityevent.TypePROpened)
	RuleEventTypePRClosed              RuleEventType = RuleEventType(activityevent.TypePRClosed)
	RuleEventTypeMachineConnected      RuleEventType = RuleEventType(activityevent.TypeMachineConnected)
	RuleEventTypeMachineDisconnected   RuleEventType = RuleEventType(activityevent.TypeMachineDisconnected)
	RuleEventTypeMachineReconnected    RuleEventType = RuleEventType(activityevent.TypeMachineReconnected)
	RuleEventTypeMachineDaemonAuthFail RuleEventType = RuleEventType(activityevent.TypeMachineDaemonAuthFailed)
)

// RuleEventCatalogEntry describes a selectable event type for UI/API consumers.
type RuleEventCatalogEntry struct {
	EventType       RuleEventType
	Label           string
	Group           RuleEventGroup
	Level           RuleEventLevel
	DefaultTemplate string
}

// RuleEventContract is the shared notification contract used by the API, engine, and tests.
type RuleEventContract struct {
	EventType       RuleEventType
	Label           string
	Group           RuleEventGroup
	Level           RuleEventLevel
	DefaultTemplate string
	Topic           string
}

type UnsupportedRuleEvent struct {
	EventType string
	Reason    string
}

var supportedRuleEventContracts = []RuleEventContract{
	{
		EventType:       RuleEventTypeTicketCreated,
		Label:           "Ticket Created",
		Group:           RuleEventGroupTicketLifecycle,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "Ticket created: {{ ticket.identifier }}\n{{ ticket.title }}\nStatus: {{ ticket.status_name }}\nPriority: {{ ticket.priority }}",
		Topic:           "ticket.events",
	},
	{
		EventType:       RuleEventTypeTicketUpdated,
		Label:           "Ticket Updated",
		Group:           RuleEventGroupTicketLifecycle,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "Ticket updated: {{ ticket.identifier }}\n{{ ticket.title }}\nStatus: {{ ticket.status_name }}",
		Topic:           "ticket.events",
	},
	{
		EventType:       RuleEventTypeTicketStatusChanged,
		Label:           "Ticket Status Changed",
		Group:           RuleEventGroupTicketLifecycle,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "Ticket status changed: {{ ticket.identifier }}\n{{ ticket.title }}\nNew status: {{ new_status }}",
		Topic:           "ticket.events",
	},
	{
		EventType:       RuleEventTypeTicketCompleted,
		Label:           "Ticket Completed",
		Group:           RuleEventGroupTicketLifecycle,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "Ticket completed: {{ ticket.identifier }}\n{{ ticket.title }}\nStatus: {{ ticket.status_name }}",
		Topic:           "ticket.events",
	},
	{
		EventType:       RuleEventTypeTicketCancelled,
		Label:           "Ticket Cancelled",
		Group:           RuleEventGroupTicketLifecycle,
		Level:           RuleEventLevelWarning,
		DefaultTemplate: "Ticket cancelled: {{ ticket.identifier }}\n{{ ticket.title }}\nStatus: {{ ticket.status_name }}",
		Topic:           "ticket.events",
	},
	{
		EventType:       RuleEventTypeTicketRetryScheduled,
		Label:           "Ticket Retry Scheduled",
		Group:           RuleEventGroupTicketReliability,
		Level:           RuleEventLevelWarning,
		DefaultTemplate: "{{ ticket.identifier }} retry scheduled\nNext retry: {{ next_retry_at }}\nConsecutive errors: {{ consecutive_errors }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeTicketRetryResumed,
		Label:           "Ticket Retry Resumed",
		Group:           RuleEventGroupTicketReliability,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "{{ ticket.identifier }} retry resumed\n{{ ticket.title }}\nPause reason: {{ pause_reason }}",
		Topic:           "ticket.events",
	},
	{
		EventType:       RuleEventTypeTicketRetryPaused,
		Label:           "Ticket Retry Paused",
		Group:           RuleEventGroupTicketReliability,
		Level:           RuleEventLevelWarning,
		DefaultTemplate: "{{ message }}\nPause reason: {{ pause_reason }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeTicketBudgetExhausted,
		Label:           "Ticket Budget Exhausted",
		Group:           RuleEventGroupTicketReliability,
		Level:           RuleEventLevelCritical,
		DefaultTemplate: "{{ ticket.identifier }} budget exhausted\n{{ ticket.title }}\nBudget: {{ budget_usd }}\nCost: {{ cost_amount }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeAgentClaimed,
		Label:           "Agent Claimed",
		Group:           RuleEventGroupAgentExecution,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "Agent {{ agent.name }} claimed ticket {{ current_ticket_id }}",
		Topic:           "agent.events",
	},
	{
		EventType:       RuleEventTypeAgentFailed,
		Label:           "Agent Failed",
		Group:           RuleEventGroupAgentExecution,
		Level:           RuleEventLevelCritical,
		DefaultTemplate: "Agent {{ agent.name }} failed ticket {{ current_ticket_id }}\nStatus: {{ status }}",
		Topic:           "agent.events",
	},
	{
		EventType:       RuleEventTypeHookFailed,
		Label:           "Hook Failed",
		Group:           RuleEventGroupHookIntegration,
		Level:           RuleEventLevelCritical,
		DefaultTemplate: "{{ ticket_identifier }} hook {{ hook_name }} failed\n{{ error }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeHookPassed,
		Label:           "Hook Passed",
		Group:           RuleEventGroupHookIntegration,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "{{ ticket_identifier }} hook {{ hook_name }} passed",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypePROpened,
		Label:           "PR Opened",
		Group:           RuleEventGroupPRReview,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "{{ ticket_identifier }} PR opened\n{{ pull_request_url }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypePRClosed,
		Label:           "PR Closed",
		Group:           RuleEventGroupPRReview,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "{{ ticket_identifier }} PR closed\n{{ pull_request_url }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeMachineConnected,
		Label:           "Machine Connected",
		Group:           RuleEventGroupInfrastructure,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "Machine connected\n{{ machine_id }}\nTransport: {{ transport_mode }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeMachineReconnected,
		Label:           "Machine Reconnected",
		Group:           RuleEventGroupInfrastructure,
		Level:           RuleEventLevelInfo,
		DefaultTemplate: "Machine reconnected\n{{ machine_id }}\nTransport: {{ transport_mode }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeMachineDisconnected,
		Label:           "Machine Disconnected",
		Group:           RuleEventGroupInfrastructure,
		Level:           RuleEventLevelWarning,
		DefaultTemplate: "Machine disconnected\n{{ machine_id }}\nReason: {{ reason }}",
		Topic:           "activity.events",
	},
	{
		EventType:       RuleEventTypeMachineDaemonAuthFail,
		Label:           "Machine Daemon Auth Failed",
		Group:           RuleEventGroupInfrastructure,
		Level:           RuleEventLevelCritical,
		DefaultTemplate: "Machine daemon auth failed\n{{ machine_id }}\nFailure code: {{ failure_code }}",
		Topic:           "activity.events",
	},
}

var unsupportedRuleEvents = []UnsupportedRuleEvent{
	{EventType: activityevent.TypePRMerged.String(), Reason: "pr.merged has no stable product emitter yet; keep it out of notification rules until a reliable merge source exists"},
	{EventType: "ticket.stalled", Reason: "legacy draft notification event; no canonical emitter or template contract is shipped"},
	{EventType: "ticket.error_rate_high", Reason: "legacy draft notification event; no canonical emitter or template contract is shipped"},
	{EventType: "machine.offline", Reason: "legacy machine monitor provider event; reverse websocket notifications use canonical machine.connected/disconnected/reconnected/auth_failed activities instead"},
	{EventType: "machine.online", Reason: "legacy machine monitor provider event; reverse websocket notifications use canonical machine.connected/disconnected/reconnected/auth_failed activities instead"},
	{EventType: "machine.degraded", Reason: "legacy machine monitor provider event; no notification contract is shipped"},
	{EventType: "connector.sync_error", Reason: "legacy draft notification event; no canonical emitter or template contract is shipped"},
	{EventType: "budget.threshold", Reason: "legacy draft notification event; no canonical emitter or template contract is shipped"},
}

var supportedRuleEventIndex = func() map[RuleEventType]RuleEventContract {
	index := make(map[RuleEventType]RuleEventContract, len(supportedRuleEventContracts))
	for _, item := range supportedRuleEventContracts {
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
	items := make([]RuleEventCatalogEntry, 0, len(supportedRuleEventContracts))
	for _, item := range supportedRuleEventContracts {
		items = append(items, RuleEventCatalogEntry{
			EventType:       item.EventType,
			Label:           item.Label,
			Group:           item.Group,
			Level:           item.Level,
			DefaultTemplate: item.DefaultTemplate,
		})
	}
	return items
}

// SupportedRuleEventContracts returns the shared runtime contract for supported notification events.
func SupportedRuleEventContracts() []RuleEventContract {
	items := make([]RuleEventContract, 0, len(supportedRuleEventContracts))
	items = append(items, supportedRuleEventContracts...)
	return items
}

// ExplicitlyUnsupportedRuleEvents returns product-level exclusions that must stay out of notification rules.
func ExplicitlyUnsupportedRuleEvents() []UnsupportedRuleEvent {
	items := make([]UnsupportedRuleEvent, 0, len(unsupportedRuleEvents))
	items = append(items, unsupportedRuleEvents...)
	return items
}

// RuleEventTopic returns the only supported runtime topic for the event type.
func RuleEventTopic(eventType RuleEventType) (string, bool) {
	item, ok := supportedRuleEventIndex[eventType]
	if !ok {
		return "", false
	}
	return item.Topic, true
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

func (t RuleEventType) Group() RuleEventGroup {
	if item, ok := supportedRuleEventIndex[t]; ok {
		return item.Group
	}
	return ""
}

func (t RuleEventType) Level() RuleEventLevel {
	if item, ok := supportedRuleEventIndex[t]; ok {
		return item.Level
	}
	return RuleEventLevelInfo
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

	return messageFromRenderedText(rendered, r.EventType.Level()), nil
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

func messageFromRenderedText(rendered string, level RuleEventLevel) Message {
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
		Level: level.String(),
	}
}
