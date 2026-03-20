package notification

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

// RuleEventType identifies a supported notification subscription event.
type RuleEventType string

const (
	RuleEventTypeTicketCreated       RuleEventType = "ticket.created"
	RuleEventTypeTicketUpdated       RuleEventType = "ticket.updated"
	RuleEventTypeTicketStatusChanged RuleEventType = "ticket.status_changed"
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
	filter, err := cloneRawConfig(raw)
	if err != nil {
		return nil, err
	}
	if filter == nil {
		return map[string]any{}, nil
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
