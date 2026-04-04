package notification

import (
	"errors"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/google/uuid"
)

func TestMapChannelAndRule(t *testing.T) {
	t.Parallel()

	channelID := uuid.New()
	orgID := uuid.New()
	channel := &ent.NotificationChannel{
		ID:             channelID,
		OrganizationID: orgID,
		Name:           "Alerts",
		Type:           " webhook ",
		Config:         map[string]any{"url": "https://hooks.example.com"},
		IsEnabled:      true,
		CreatedAt:      time.Now(),
	}
	mappedChannel := mapChannel(channel)
	if mappedChannel.Type != domain.ChannelType("webhook") {
		t.Fatalf("mapChannel type = %q, want webhook", mappedChannel.Type)
	}

	invalidChannel := &ent.NotificationChannel{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Legacy",
		Type:           " legacy ",
	}
	if got := mapChannel(invalidChannel); got.Type != domain.ChannelType("legacy") {
		t.Fatalf("mapChannel fallback type = %q, want legacy", got.Type)
	}

	rule := &ent.NotificationRule{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		ChannelID: channelID,
		Name:      "Status changed",
		EventType: " ticket.status_changed ",
		Filter:    map[string]any{"status": "review"},
		Template:  "{{ ticket.identifier }}",
		IsEnabled: true,
		CreatedAt: time.Now(),
		Edges: ent.NotificationRuleEdges{
			Channel: channel,
		},
	}
	mappedRule := mapRule(rule)
	if mappedRule.EventType != domain.RuleEventTypeTicketStatusChanged || mappedRule.Channel.ID != channelID {
		t.Fatalf("mapRule = %+v", mappedRule)
	}

	invalidRule := &ent.NotificationRule{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		ChannelID: channelID,
		Name:      "Legacy event",
		EventType: " legacy.event ",
	}
	if got := mapRule(invalidRule); got.EventType != domain.RuleEventType("legacy.event") {
		t.Fatalf("mapRule fallback event type = %q, want legacy.event", got.EventType)
	}
}

func TestRepositoryErrorMappingPassThrough(t *testing.T) {
	t.Parallel()

	if got := mapChannelNotFound(errors.New("plain")); got.Error() != "plain" {
		t.Fatalf("mapChannelNotFound = %v, want original error", got)
	}
	if got := mapRuleNotFound(errors.New("plain")); got.Error() != "plain" {
		t.Fatalf("mapRuleNotFound = %v, want original error", got)
	}
	if got := mapPersistenceError("create rule", errors.New("plain")); got.Error() != "create rule: plain" {
		t.Fatalf("mapPersistenceError = %v, want wrapped error", got)
	}
}

func TestNewEntRepository(t *testing.T) {
	t.Parallel()

	client := &ent.Client{}
	repo := NewEntRepository(client)
	if repo == nil || repo.client != client {
		t.Fatalf("NewEntRepository() = %+v, want client wired", repo)
	}
}
