package notification

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	notificationrepo "github.com/BetterAndBetterII/openase/internal/repo/notification"
	"github.com/google/uuid"
)

func TestNotificationServiceChannelRuleCRUDAndDelivery(t *testing.T) {
	client := openNotificationTestEntClient(t)
	ctx := context.Background()

	orgID := seedNotificationOrganization(ctx, t, client, "Better And Better", "better-and-better")
	projectID := seedNotificationProject(ctx, t, client, orgID, "OpenASE", "openase")
	otherOrgID := seedNotificationOrganization(ctx, t, client, "Other Org", "other-org")
	otherProjectID := seedNotificationProject(ctx, t, client, otherOrgID, "Other Project", "other-project")

	adapter := &recordingAdapter{}
	service := NewService(notificationrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	service.registry = NewAdapterRegistry(adapter)

	if _, err := service.List(ctx, uuid.New()); !errors.Is(err, ErrOrganizationNotFound) {
		t.Fatalf("List() missing org error = %v, want %v", err, ErrOrganizationNotFound)
	}
	if _, err := service.ListRules(ctx, uuid.New()); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("ListRules() missing project error = %v, want %v", err, ErrProjectNotFound)
	}

	primaryChannel, err := service.Create(ctx, domain.CreateChannelInput{
		OrganizationID: orgID,
		Name:           "Primary",
		Type:           domain.ChannelTypeWebhook,
		Config: map[string]any{
			"url":     "https://hooks.example.com/primary",
			"headers": map[string]any{"X-Test": "1"},
			"secret":  "top-secret",
		},
		IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() primary channel error = %v", err)
	}
	if primaryChannel.Name != "Primary" || primaryChannel.Type != domain.ChannelTypeWebhook {
		t.Fatalf("Create() primary channel = %+v", primaryChannel)
	}

	secondaryChannel, err := service.Create(ctx, domain.CreateChannelInput{
		OrganizationID: orgID,
		Name:           "Secondary",
		Type:           domain.ChannelTypeWebhook,
		Config:         map[string]any{"url": "https://hooks.example.com/secondary"},
		IsEnabled:      true,
	})
	if err != nil {
		t.Fatalf("Create() secondary channel error = %v", err)
	}
	failingChannel, err := service.Create(ctx, domain.CreateChannelInput{
		OrganizationID: orgID,
		Name:           "Third",
		Type:           domain.ChannelTypeWebhook,
		Config:         map[string]any{"url": "https://hooks.example.com/fail"},
		IsEnabled:      true,
	})
	if err != nil {
		t.Fatalf("Create() failing channel error = %v", err)
	}
	disabledChannel, err := service.Create(ctx, domain.CreateChannelInput{
		OrganizationID: orgID,
		Name:           "Zzz Disabled",
		Type:           domain.ChannelTypeWebhook,
		Config:         map[string]any{"url": "https://hooks.example.com/disabled"},
		IsEnabled:      false,
	})
	if err != nil {
		t.Fatalf("Create() disabled channel error = %v", err)
	}
	foreignChannel, err := service.Create(ctx, domain.CreateChannelInput{
		OrganizationID: otherOrgID,
		Name:           "Foreign",
		Type:           domain.ChannelTypeWebhook,
		Config:         map[string]any{"url": "https://hooks.example.com/foreign"},
		IsEnabled:      true,
	})
	if err != nil {
		t.Fatalf("Create() foreign channel error = %v", err)
	}

	if _, err := service.Create(ctx, domain.CreateChannelInput{
		OrganizationID: orgID,
		Name:           "Primary",
		Type:           domain.ChannelTypeWebhook,
		Config:         map[string]any{"url": "https://hooks.example.com/duplicate"},
		IsEnabled:      true,
	}); !errors.Is(err, ErrDuplicateChannelName) {
		t.Fatalf("Create() duplicate channel error = %v, want %v", err, ErrDuplicateChannelName)
	}

	channels, err := service.List(ctx, orgID)
	if err != nil {
		t.Fatalf("List() channels error = %v", err)
	}
	if len(channels) != 4 {
		t.Fatalf("List() channels len = %d, want 4", len(channels))
	}
	if channels[0].Name != "Primary" || channels[1].Name != "Secondary" || channels[2].Name != "Third" || channels[3].Name != "Zzz Disabled" {
		t.Fatalf("List() channel order = %+v", channels)
	}

	gotPrimary, err := service.Get(ctx, primaryChannel.ID)
	if err != nil {
		t.Fatalf("Get() primary channel error = %v", err)
	}
	if gotPrimary.ID != primaryChannel.ID || gotPrimary.Config["url"] != "https://hooks.example.com/primary" {
		t.Fatalf("Get() primary channel = %+v", gotPrimary)
	}
	if _, err := service.Get(ctx, uuid.New()); !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("Get() missing channel error = %v, want %v", err, ErrChannelNotFound)
	}

	if _, err := service.Update(ctx, domain.UpdateChannelInput{
		ChannelID: primaryChannel.ID,
		Config:    domain.Some(map[string]any{"url": 123}),
	}); !errors.Is(err, ErrInvalidChannelConfig) {
		t.Fatalf("Update() invalid config error = %v, want %v", err, ErrInvalidChannelConfig)
	}

	updatedPrimary, err := service.Update(ctx, domain.UpdateChannelInput{
		ChannelID: primaryChannel.ID,
		Name:      domain.Some("Primary Alerts"),
		Config: domain.Some(map[string]any{
			"url":     "https://hooks.example.com/primary-updated",
			"headers": map[string]any{"X-Updated": "yes"},
		}),
		IsEnabled: domain.Some(true),
	})
	if err != nil {
		t.Fatalf("Update() primary channel error = %v", err)
	}
	if updatedPrimary.Name != "Primary Alerts" || updatedPrimary.Config["url"] != "https://hooks.example.com/primary-updated" {
		t.Fatalf("Update() primary channel = %+v", updatedPrimary)
	}

	if _, err := service.CreateRule(ctx, domain.CreateRuleInput{
		ProjectID: otherProjectID,
		Name:      "Bad rule",
		EventType: domain.RuleEventTypeTicketCreated,
		ChannelID: primaryChannel.ID,
		Template:  "bad",
		IsEnabled: true,
	}); !errors.Is(err, ErrChannelProjectMismatch) {
		t.Fatalf("CreateRule() mismatch error = %v, want %v", err, ErrChannelProjectMismatch)
	}

	primaryRule, err := service.CreateRule(ctx, domain.CreateRuleInput{
		ProjectID: projectID,
		Name:      "Ticket Created",
		EventType: domain.RuleEventTypeTicketCreated,
		Filter:    map[string]any{"ticket.status_name": "Todo"},
		ChannelID: primaryChannel.ID,
		Template:  "Created\n{{ ticket.identifier }}",
		IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("CreateRule() primary rule error = %v", err)
	}
	disabledRule, err := service.CreateRule(ctx, domain.CreateRuleInput{
		ProjectID: projectID,
		Name:      "Ticket Updated Disabled",
		EventType: domain.RuleEventTypeTicketUpdated,
		ChannelID: disabledChannel.ID,
		Template:  "Ignored",
		IsEnabled: false,
	})
	if err != nil {
		t.Fatalf("CreateRule() disabled rule error = %v", err)
	}
	if _, err := service.CreateRule(ctx, domain.CreateRuleInput{
		ProjectID: projectID,
		Name:      "Ticket Created",
		EventType: domain.RuleEventTypeTicketCreated,
		ChannelID: secondaryChannel.ID,
		Template:  "duplicate",
		IsEnabled: true,
	}); !errors.Is(err, ErrDuplicateRuleName) {
		t.Fatalf("CreateRule() duplicate rule error = %v, want %v", err, ErrDuplicateRuleName)
	}

	rules, err := service.ListRules(ctx, projectID)
	if err != nil {
		t.Fatalf("ListRules() error = %v", err)
	}
	if len(rules) != 2 || rules[0].Name != "Ticket Created" || rules[1].Name != "Ticket Updated Disabled" {
		t.Fatalf("ListRules() = %+v", rules)
	}

	matchingCreated, err := service.MatchingRules(ctx, projectID, domain.RuleEventTypeTicketCreated)
	if err != nil {
		t.Fatalf("MatchingRules() created error = %v", err)
	}
	if len(matchingCreated) != 1 || matchingCreated[0].ID != primaryRule.ID || matchingCreated[0].Channel.ID != primaryChannel.ID {
		t.Fatalf("MatchingRules() created = %+v", matchingCreated)
	}
	matchingUpdated, err := service.MatchingRules(ctx, projectID, domain.RuleEventTypeTicketUpdated)
	if err != nil {
		t.Fatalf("MatchingRules() updated error = %v", err)
	}
	if len(matchingUpdated) != 0 {
		t.Fatalf("MatchingRules() updated = %+v, want none because rule/channel disabled", matchingUpdated)
	}

	if _, err := service.UpdateRule(ctx, domain.UpdateRuleInput{
		RuleID:    primaryRule.ID,
		ChannelID: domain.Some(foreignChannel.ID),
	}); !errors.Is(err, ErrChannelProjectMismatch) {
		t.Fatalf("UpdateRule() mismatch error = %v, want %v", err, ErrChannelProjectMismatch)
	}

	updatedRule, err := service.UpdateRule(ctx, domain.UpdateRuleInput{
		RuleID:    primaryRule.ID,
		Name:      domain.Some("Ticket Updated"),
		EventType: domain.Some(domain.RuleEventTypeTicketUpdated),
		Filter:    domain.Some(map[string]any{"ticket.identifier": "ASE-278"}),
		ChannelID: domain.Some(secondaryChannel.ID),
		Template:  domain.Some("Updated\n{{ ticket.identifier }}\n{{ new_status }}"),
		IsEnabled: domain.Some(true),
	})
	if err != nil {
		t.Fatalf("UpdateRule() primary rule error = %v", err)
	}
	if updatedRule.Name != "Ticket Updated" || updatedRule.ChannelID != secondaryChannel.ID || updatedRule.EventType != domain.RuleEventTypeTicketUpdated {
		t.Fatalf("UpdateRule() = %+v", updatedRule)
	}

	adapter.reset()
	if err := service.Test(ctx, secondaryChannel.ID); err != nil {
		t.Fatalf("Test() error = %v", err)
	}
	if got := adapter.callCount(); got != 1 {
		t.Fatalf("Test() send count = %d, want 1", got)
	}
	testMessage := adapter.lastCall().message
	if testMessage.Title != "OpenASE test notification" || testMessage.Metadata["kind"] != "test" {
		t.Fatalf("Test() message = %+v", testMessage)
	}

	adapter.reset()
	if err := service.SendRule(ctx, updatedRule, domain.Message{Title: "Rule", Body: "Direct"}); err != nil {
		t.Fatalf("SendRule() error = %v", err)
	}
	if got := adapter.callCount(); got != 1 {
		t.Fatalf("SendRule() send count = %d, want 1", got)
	}
	if adapter.lastCall().channelType != domain.ChannelTypeWebhook {
		t.Fatalf("SendRule() channel type = %s, want webhook", adapter.lastCall().channelType)
	}

	adapter.reset()
	if err := service.SendToProjectChannels(ctx, projectID, domain.Message{Title: "Project", Body: "Fanout"}); err != nil {
		t.Fatalf("SendToProjectChannels() error = %v", err)
	}
	if got := adapter.callCount(); got != 3 {
		t.Fatalf("SendToProjectChannels() send count = %d, want 3 enabled channels", got)
	}
	for _, call := range adapter.callsSnapshot() {
		if strings.Contains(call.url(), "/disabled") {
			t.Fatalf("SendToProjectChannels() should skip disabled channel, got %+v", call)
		}
	}

	if err := service.DeleteRule(ctx, disabledRule.ID); err != nil {
		t.Fatalf("DeleteRule() error = %v", err)
	}
	if err := service.DeleteRule(ctx, disabledRule.ID); !errors.Is(err, ErrRuleNotFound) {
		t.Fatalf("DeleteRule() missing rule error = %v, want %v", err, ErrRuleNotFound)
	}

	if err := service.Delete(ctx, failingChannel.ID); err != nil {
		t.Fatalf("Delete() failing channel error = %v", err)
	}
	if err := service.Delete(ctx, failingChannel.ID); !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("Delete() missing channel error = %v, want %v", err, ErrChannelNotFound)
	}
}

type recordingAdapter struct {
	mu    sync.Mutex
	calls []recordingCall
}

type recordingCall struct {
	channelType domain.ChannelType
	config      domain.ChannelConfig
	message     domain.Message
}

func (a *recordingAdapter) Type() domain.ChannelType {
	return domain.ChannelTypeWebhook
}

func (a *recordingAdapter) Validate(_ context.Context, cfg domain.ChannelConfig) error {
	_, ok := cfg.(domain.WebhookConfig)
	if !ok {
		return fmt.Errorf("webhook adapter requires %T config", domain.WebhookConfig{})
	}
	return nil
}

func (a *recordingAdapter) Send(_ context.Context, cfg domain.ChannelConfig, msg domain.Message) error {
	webhookConfig, ok := cfg.(domain.WebhookConfig)
	if !ok {
		return fmt.Errorf("webhook adapter requires %T config", domain.WebhookConfig{})
	}

	a.mu.Lock()
	a.calls = append(a.calls, recordingCall{
		channelType: domain.ChannelTypeWebhook,
		config:      webhookConfig,
		message:     msg,
	})
	a.mu.Unlock()

	if strings.Contains(webhookConfig.URL, "/fail") {
		return errors.New("synthetic delivery failure")
	}

	return nil
}

func (a *recordingAdapter) reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = nil
}

func (a *recordingAdapter) callCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.calls)
}

func (a *recordingAdapter) lastCall() recordingCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls[len(a.calls)-1]
}

func (a *recordingAdapter) callsSnapshot() []recordingCall {
	a.mu.Lock()
	defer a.mu.Unlock()

	copied := make([]recordingCall, 0, len(a.calls))
	copied = append(copied, a.calls...)
	return copied
}

func (c recordingCall) url() string {
	webhookConfig, ok := c.config.(domain.WebhookConfig)
	if !ok {
		return ""
	}
	return webhookConfig.URL
}

func openNotificationTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func seedNotificationOrganization(ctx context.Context, t *testing.T, client *ent.Client, name, slug string) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName(name).
		SetSlug(slug).
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization %q: %v", slug, err)
	}

	return org.ID
}

func seedNotificationProject(ctx context.Context, t *testing.T, client *ent.Client, organizationID uuid.UUID, name, slug string) uuid.UUID {
	t.Helper()

	project, err := client.Project.Create().
		SetOrganizationID(organizationID).
		SetName(name).
		SetSlug(slug).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project %q: %v", slug, err)
	}

	return project.ID
}
