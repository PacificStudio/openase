package notification

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestRuleMatchesSupportsFlatAndNestedFields(t *testing.T) {
	t.Parallel()

	rule := Rule{
		Filter: map[string]any{
			"priority":           "high",
			"ticket.status_name": "In Progress",
		},
	}

	context := map[string]any{
		"priority": "high",
		"ticket": map[string]any{
			"status_name": "In Progress",
		},
	}

	if !rule.Matches(context) {
		t.Fatalf("expected rule filter to match context")
	}
}

func TestRuleRenderMessageUsesDefaultTemplate(t *testing.T) {
	t.Parallel()

	rule := Rule{
		EventType: RuleEventTypeTicketCreated,
	}

	message, err := rule.RenderMessage(map[string]any{
		"ticket": map[string]any{
			"identifier":  "ASE-69",
			"title":       "Build notification rules",
			"status_name": "Todo",
			"priority":    "high",
		},
		"new_status": "Todo",
	})
	if err != nil {
		t.Fatalf("RenderMessage() error = %v", err)
	}

	if message.Title != "Ticket created: ASE-69" {
		t.Fatalf("RenderMessage() title = %q, want %q", message.Title, "Ticket created: ASE-69")
	}
	if message.Body == "" {
		t.Fatalf("RenderMessage() expected non-empty body")
	}
}

func TestParseCreateRuleRejectsInvalidChannelID(t *testing.T) {
	t.Parallel()

	_, err := ParseCreateRule(uuid.New(), RuleInput{
		Name:      "Ticket Created",
		EventType: RuleEventTypeTicketCreated.String(),
		ChannelID: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected ParseCreateRule() to reject invalid channel id")
	}
}

func TestSupportedRuleEventContractsExposeSingleSourceTopics(t *testing.T) {
	t.Parallel()

	contracts := SupportedRuleEventContracts()
	if len(contracts) != len(SupportedRuleEvents()) {
		t.Fatalf("SupportedRuleEventContracts() length = %d, want %d", len(contracts), len(SupportedRuleEvents()))
	}

	seen := map[RuleEventType]struct{}{}
	for _, item := range contracts {
		if item.Topic == "" {
			t.Fatalf("contract for %s missing topic", item.EventType)
		}
		if _, ok := seen[item.EventType]; ok {
			t.Fatalf("duplicate supported contract for %s", item.EventType)
		}
		seen[item.EventType] = struct{}{}
		if topic, ok := RuleEventTopic(item.EventType); !ok || topic != item.Topic {
			t.Fatalf("RuleEventTopic(%s) = %q, %t; want %q, true", item.EventType, topic, ok, item.Topic)
		}
	}

	for _, item := range ExplicitlyUnsupportedRuleEvents() {
		if _, ok := seen[RuleEventType(item.EventType)]; ok {
			t.Fatalf("excluded event %s leaked into supported contract", item.EventType)
		}
	}
}

func TestRuleParsingAndTemplateHelpers(t *testing.T) {
	events := SupportedRuleEvents()
	if len(events) != 19 {
		t.Fatalf("SupportedRuleEvents() length = %d, want only the wired catalog", len(events))
	}
	if events[0].Group == "" || events[0].Level == "" {
		t.Fatalf("SupportedRuleEvents()[0] missing grouped metadata: %+v", events[0])
	}
	events[0].Label = "changed"
	if SupportedRuleEvents()[0].Label == "changed" {
		t.Fatal("SupportedRuleEvents() returned mutable backing slice")
	}

	eventType, err := ParseRuleEventType("ticket.updated")
	if err != nil {
		t.Fatalf("ParseRuleEventType() error = %v", err)
	}
	if eventType.String() != "ticket.updated" || eventType.DefaultTemplate() == "" {
		t.Fatalf("ParseRuleEventType() = %q with template %q", eventType, eventType.DefaultTemplate())
	}
	if eventType.Group().String() != RuleEventGroupTicketLifecycle.String() {
		t.Fatalf("Group() = %q, want %q", eventType.Group(), RuleEventGroupTicketLifecycle)
	}
	if eventType.Level().String() != RuleEventLevelInfo.String() {
		t.Fatalf("Level() = %q, want %q", eventType.Level(), RuleEventLevelInfo)
	}
	if RuleEventType("unknown").DefaultTemplate() != "" {
		t.Fatal("DefaultTemplate() for unknown event expected empty string")
	}
	if RuleEventType("unknown").Group() != "" {
		t.Fatal("Group() for unknown event expected empty string")
	}
	if RuleEventType("unknown").Level() != RuleEventLevelInfo {
		t.Fatal("Level() for unknown event expected info fallback")
	}
	if _, err := ParseRuleEventType("bad"); err == nil {
		t.Fatal("ParseRuleEventType() expected validation error")
	}
	for _, item := range ExplicitlyUnsupportedRuleEvents() {
		if _, err := ParseRuleEventType(item.EventType); err == nil {
			t.Fatalf("ParseRuleEventType(%q) unexpectedly accepted excluded event", item.EventType)
		}
	}

	channelID := uuid.New()
	disabled := false
	createInput, err := ParseCreateRule(uuid.New(), RuleInput{
		Name:      " Ticket Updated ",
		EventType: RuleEventTypeTicketUpdated.String(),
		Filter: map[string]any{
			"ticket.status_name": "In Progress",
			"labels":             []any{"backend", "coverage"},
		},
		ChannelID: channelID.String(),
		Template:  " custom ",
		IsEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("ParseCreateRule() error = %v", err)
	}
	if createInput.Name != "Ticket Updated" || createInput.EventType != RuleEventTypeTicketUpdated || createInput.ChannelID != channelID || createInput.Template != "custom" || createInput.IsEnabled {
		t.Fatalf("ParseCreateRule() = %+v", createInput)
	}

	name := " Ticket Status "
	eventTypeText := RuleEventTypeTicketStatusChanged.String()
	filter := map[string]any{"ticket.priority": "high"}
	template := " {{ ticket.identifier }} "
	updateInput, err := ParseUpdateRule(uuid.New(), RulePatchInput{
		Name:      &name,
		EventType: &eventTypeText,
		Filter:    &filter,
		ChannelID: stringPtr(channelID.String()),
		Template:  &template,
		IsEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("ParseUpdateRule() error = %v", err)
	}
	if !updateInput.Name.Set || updateInput.Name.Value != "Ticket Status" || !updateInput.EventType.Set || updateInput.EventType.Value != RuleEventTypeTicketStatusChanged || !updateInput.Filter.Set || !updateInput.ChannelID.Set || !updateInput.Template.Set || updateInput.Template.Value != "{{ ticket.identifier }}" || !updateInput.IsEnabled.Set || updateInput.IsEnabled.Value {
		t.Fatalf("ParseUpdateRule() = %+v", updateInput)
	}
	if _, err := ParseUpdateRule(uuid.New(), RulePatchInput{}); err == nil {
		t.Fatal("ParseUpdateRule() expected patch validation error")
	}
	blankName := " "
	if _, err := ParseUpdateRule(uuid.New(), RulePatchInput{Name: &blankName}); err == nil {
		t.Fatal("ParseUpdateRule() expected blank name validation error")
	}
	if _, err := ParseCreateRule(uuid.New(), RuleInput{Name: " ", EventType: RuleEventTypeTicketUpdated.String(), ChannelID: channelID.String()}); err == nil {
		t.Fatal("ParseCreateRule() expected blank name validation error")
	}
	if _, err := ParseCreateRule(uuid.New(), RuleInput{Name: "ok", EventType: "bad", ChannelID: channelID.String()}); err == nil {
		t.Fatal("ParseCreateRule() expected event type validation error")
	}
	if _, err := ParseCreateRule(uuid.New(), RuleInput{Name: "ok", EventType: RuleEventTypeTicketUpdated.String(), Filter: map[string]any{"bad": make(chan int)}, ChannelID: channelID.String()}); err == nil {
		t.Fatal("ParseCreateRule() expected filter validation error")
	}
	if _, err := ParseUpdateRule(uuid.New(), RulePatchInput{EventType: stringPtr("bad")}); err == nil {
		t.Fatal("ParseUpdateRule() expected event type validation error")
	}
	if _, err := ParseUpdateRule(uuid.New(), RulePatchInput{Filter: &map[string]any{"bad": make(chan int)}}); err == nil {
		t.Fatal("ParseUpdateRule() expected filter validation error")
	}
	if _, err := ParseUpdateRule(uuid.New(), RulePatchInput{ChannelID: stringPtr("bad")}); err == nil {
		t.Fatal("ParseUpdateRule() expected channel_id validation error")
	}

	originalFilter := map[string]any{"ticket": map[string]any{"priority": "high"}}
	clonedFilter, err := normalizeRuleFilter(originalFilter)
	if err != nil {
		t.Fatalf("normalizeRuleFilter() error = %v", err)
	}
	clonedFilter["ticket"].(map[string]any)["priority"] = "low"
	if reflect.DeepEqual(originalFilter, clonedFilter) {
		t.Fatal("normalizeRuleFilter() expected cloned map")
	}
	if got, err := normalizeRuleFilter(nil); err != nil || len(got) != 0 {
		t.Fatalf("normalizeRuleFilter(nil) = %+v, %v; want empty map, nil", got, err)
	}
	if _, err := normalizeRuleFilter(map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("normalizeRuleFilter() expected clone validation error")
	}
	if normalizeTemplate("  hi  ") != "hi" {
		t.Fatal("normalizeTemplate() did not trim input")
	}
	if _, err := parseRuleChannelID("bad"); err == nil {
		t.Fatal("parseRuleChannelID() expected validation error")
	}

	context := map[string]any{
		"ticket": map[string]any{
			"priority": "high",
			"labels":   []any{"backend", "coverage"},
		},
	}
	if !matchRuleFilter(map[string]any{"ticket.priority": "high"}, context) {
		t.Fatal("matchRuleFilter() expected nested match")
	}
	if !matchRuleFilter(nil, context) {
		t.Fatal("matchRuleFilter(nil) expected true")
	}
	if matchRuleFilter(map[string]any{"ticket.priority": "low"}, context) {
		t.Fatal("matchRuleFilter() expected mismatch")
	}
	if matchRuleFilter(map[string]any{"missing": "value"}, context) {
		t.Fatal("matchRuleFilter() expected missing key mismatch")
	}
	if got, ok := lookupFilterValue(map[string]any{"priority": "high"}, "priority"); !ok || got != "high" {
		t.Fatalf("lookupFilterValue(direct) = %v, %t; want high, true", got, ok)
	}
	if got, ok := lookupFilterValue(context, "ticket.priority"); !ok || got != "high" {
		t.Fatalf("lookupFilterValue() = %v, %t; want high, true", got, ok)
	}
	if _, ok := lookupFilterValue(context, ""); ok {
		t.Fatal("lookupFilterValue() blank key expected false")
	}
	if _, ok := lookupFilterValue(context, "priority"); ok {
		t.Fatal("lookupFilterValue() expected false for missing direct key")
	}
	if _, ok := lookupFilterValue(context, "ticket.missing"); ok {
		t.Fatal("lookupFilterValue() expected false for missing nested key")
	}
	if _, ok := lookupFilterValue(map[string]any{"ticket": "oops"}, "ticket.priority"); ok {
		t.Fatal("lookupFilterValue() expected false for non-object path")
	}
	if !matchRuleValue(map[string]any{"labels": []any{"backend", "coverage"}}, map[string]any{"labels": []any{"backend", "coverage"}}) {
		t.Fatal("matchRuleValue() expected nested collection match")
	}
	if !matchRuleValue("high", "high") {
		t.Fatal("matchRuleValue() expected primitive equality")
	}
	if matchRuleValue(map[string]any{"priority": "high"}, "high") {
		t.Fatal("matchRuleValue() expected map type mismatch")
	}
	if matchRuleValue([]any{"backend"}, "backend") {
		t.Fatal("matchRuleValue() expected slice type mismatch")
	}
	if matchRuleValue([]any{"backend"}, []any{"backend", "coverage"}) {
		t.Fatal("matchRuleValue() expected slice length mismatch")
	}
	if matchRuleValue(map[string]any{"priority": "high"}, map[string]any{}) {
		t.Fatal("matchRuleValue() expected missing nested key mismatch")
	}
	if matchRuleValue([]any{"backend"}, []any{"coverage"}) {
		t.Fatal("matchRuleValue() expected mismatch")
	}

	if got, err := renderRuleTemplate("", context); err != nil || got != "" {
		t.Fatalf("renderRuleTemplate(empty) = %q, %v; want empty, nil", got, err)
	}
	if _, err := renderRuleTemplate("{{", context); err == nil {
		t.Fatal("renderRuleTemplate() expected parse error")
	}
	if _, err := renderRuleTemplate("{{ missing.value }}", context); err == nil {
		t.Fatal("renderRuleTemplate() expected render error")
	}
	if _, err := (Rule{EventType: RuleEventTypeTicketUpdated, Template: "{{"}).RenderMessage(context); err == nil {
		t.Fatal("RenderMessage() expected template parse error")
	}

	message := messageFromRenderedText(" Title \n Body line 1 \n Body line 2 ", RuleEventLevelInfo)
	if message.Title != "Title" || message.Level != "info" || !strings.Contains(message.Body, "Body line 1") || !strings.Contains(message.Body, "Body line 2") {
		t.Fatalf("messageFromRenderedText() = %+v", message)
	}
	if got := messageFromRenderedText(" ", RuleEventLevelInfo); got.Title != "" || got.Body != "" || got.Level != "" || got.Link != "" || got.Metadata != nil {
		t.Fatalf("messageFromRenderedText(blank) = %+v, want zero value", got)
	}
}

func stringPtr(value string) *string {
	return &value
}
