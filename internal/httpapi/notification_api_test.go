package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestNotificationChannelRoutesCRUDAndTestSend(t *testing.T) {
	client := openTestEntClient(t)

	webhookRequests := make(chan map[string]any, 2)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_ = r.Body.Close()
		}()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode webhook payload: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		payload["_signature"] = r.Header.Get("X-OpenASE-Signature")
		webhookRequests <- payload
		w.WriteHeader(http.StatusAccepted)
	}))
	defer webhookServer.Close()

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithNotificationService(notificationservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), webhookServer.Client())),
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	var createResp struct {
		Channel notificationChannelResponse `json:"channel"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/orgs/%s/channels", org.ID),
		map[string]any{
			"name": "Ops Webhook",
			"type": "webhook",
			"config": map[string]any{
				"url": webhookServer.URL,
				"headers": map[string]any{
					"X-Team": "ops",
				},
				"secret": "super-secret",
			},
		},
		http.StatusCreated,
		&createResp,
	)
	if createResp.Channel.Name != "Ops Webhook" || createResp.Channel.Type != "webhook" {
		t.Fatalf("unexpected create response: %+v", createResp.Channel)
	}
	if createResp.Channel.Config["secret"] == "super-secret" {
		t.Fatalf("expected secret to be redacted, got %+v", createResp.Channel.Config)
	}

	var listResp struct {
		Channels []notificationChannelResponse `json:"channels"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/orgs/%s/channels", org.ID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(listResp.Channels))
	}

	var patchResp struct {
		Channel notificationChannelResponse `json:"channel"`
	}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/channels/%s", createResp.Channel.ID),
		map[string]any{
			"name":       "Ops Notifications",
			"is_enabled": false,
		},
		http.StatusOK,
		&patchResp,
	)
	if patchResp.Channel.Name != "Ops Notifications" || patchResp.Channel.IsEnabled {
		t.Fatalf("unexpected patch response: %+v", patchResp.Channel)
	}

	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/channels/%s/test", createResp.Channel.ID),
		nil,
		http.StatusOK,
		&map[string]string{},
	)

	select {
	case payload := <-webhookRequests:
		if payload["title"] != "OpenASE test notification" {
			t.Fatalf("unexpected webhook title: %+v", payload)
		}
		signature, _ := payload["_signature"].(string)
		if signature == "" {
			t.Fatalf("expected webhook signature header, got %+v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for webhook test notification")
	}

	var deleteResp struct {
		DeletedChannelID string `json:"deleted_channel_id"`
	}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/channels/%s", createResp.Channel.ID),
		nil,
		http.StatusOK,
		&deleteResp,
	)
	if deleteResp.DeletedChannelID != createResp.Channel.ID {
		t.Fatalf("unexpected delete response: %+v", deleteResp)
	}
}

func TestNotificationRuleRoutesCRUDAndEventCatalog(t *testing.T) {
	client := openTestEntClient(t)

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer webhookServer.Close()

	server := NewServer(
		config.ServerConfig{Port: 40024},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithNotificationService(notificationservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), webhookServer.Client())),
	)
	service := notificationservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), webhookServer.Client())

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Platform").
		SetSlug("platform").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	channelInput, err := domain.ParseCreateChannel(org.ID, domain.ChannelInput{
		Name: "Primary Webhook",
		Type: "webhook",
		Config: map[string]any{
			"url": webhookServer.URL,
		},
	})
	if err != nil {
		t.Fatalf("parse channel: %v", err)
	}
	channel, err := service.Create(ctx, channelInput)
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}

	var eventTypesResp struct {
		EventTypes []notificationRuleEventTypeResponse `json:"event_types"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/notification-event-types",
		nil,
		http.StatusOK,
		&eventTypesResp,
	)
	if len(eventTypesResp.EventTypes) == 0 {
		t.Fatal("expected non-empty notification event type catalog")
	}

	var createResp struct {
		Rule notificationRuleResponse `json:"rule"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/notification-rules", project.ID),
		map[string]any{
			"name":       "High Priority Ticket Alerts",
			"event_type": "ticket.created",
			"channel_id": channel.ID.String(),
			"filter": map[string]any{
				"priority": "high",
			},
			"template": "Ticket created: {{ ticket.identifier }}\n{{ ticket.title }}",
		},
		http.StatusCreated,
		&createResp,
	)
	if createResp.Rule.Name != "High Priority Ticket Alerts" {
		t.Fatalf("unexpected created rule: %+v", createResp.Rule)
	}
	if createResp.Rule.Channel.ID != channel.ID.String() {
		t.Fatalf("expected embedded channel, got %+v", createResp.Rule.Channel)
	}

	var listResp struct {
		Rules []notificationRuleResponse `json:"rules"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/notification-rules", project.ID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(listResp.Rules))
	}

	var patchResp struct {
		Rule notificationRuleResponse `json:"rule"`
	}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/notification-rules/%s", createResp.Rule.ID),
		map[string]any{
			"name":       "Status Alerts",
			"event_type": "ticket.status_changed",
			"filter": map[string]any{
				"new_status": "Done",
			},
			"is_enabled": false,
		},
		http.StatusOK,
		&patchResp,
	)
	if patchResp.Rule.Name != "Status Alerts" || patchResp.Rule.IsEnabled {
		t.Fatalf("unexpected patched rule: %+v", patchResp.Rule)
	}

	var deleteResp struct {
		DeletedRuleID string `json:"deleted_rule_id"`
	}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/notification-rules/%s", createResp.Rule.ID),
		nil,
		http.StatusOK,
		&deleteResp,
	)
	if deleteResp.DeletedRuleID != createResp.Rule.ID {
		t.Fatalf("unexpected delete response: %+v", deleteResp)
	}
}

func TestNotificationEngineDispatchesMatchingRulesBestEffort(t *testing.T) {
	client := openTestEntClient(t)

	bus := eventinfra.NewChannelBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	webhookRequests := make(chan map[string]any, 2)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_ = r.Body.Close()
		}()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode webhook payload: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		webhookRequests <- payload
		w.WriteHeader(http.StatusAccepted)
	}))
	defer webhookServer.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := notificationservice.NewService(client, logger, webhookServer.Client())
	engine := notificationservice.NewEngine(service, bus, logger)
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("start notification engine: %v", err)
	}

	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Platform").
		SetSlug("platform").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	goodInput, err := domain.ParseCreateChannel(org.ID, domain.ChannelInput{
		Name: "Primary Webhook",
		Type: "webhook",
		Config: map[string]any{
			"url": webhookServer.URL,
		},
	})
	if err != nil {
		t.Fatalf("parse good channel: %v", err)
	}
	if _, err := service.Create(ctx, goodInput); err != nil {
		t.Fatalf("create good channel: %v", err)
	}

	badInput, err := domain.ParseCreateChannel(org.ID, domain.ChannelInput{
		Name: "Broken Webhook",
		Type: "webhook",
		Config: map[string]any{
			"url": "http://127.0.0.1:1/unreachable",
		},
	})
	if err != nil {
		t.Fatalf("parse bad channel: %v", err)
	}
	if _, err := service.Create(ctx, badInput); err != nil {
		t.Fatalf("create bad channel: %v", err)
	}

	channels, err := service.List(ctx, org.ID)
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	var goodChannelID uuid.UUID
	var badChannelID uuid.UUID
	for _, channel := range channels {
		switch channel.Name {
		case "Primary Webhook":
			goodChannelID = channel.ID
		case "Broken Webhook":
			badChannelID = channel.ID
		}
	}
	if goodChannelID == uuid.Nil || badChannelID == uuid.Nil {
		t.Fatalf("expected both channels to exist, got good=%s bad=%s", goodChannelID, badChannelID)
	}

	createGoodRule, err := domain.ParseCreateRule(project.ID, domain.RuleInput{
		Name:      "High Priority Created",
		EventType: "ticket.created",
		ChannelID: goodChannelID.String(),
		Filter: map[string]any{
			"priority": "high",
		},
	})
	if err != nil {
		t.Fatalf("parse good rule: %v", err)
	}
	if _, err := service.CreateRule(ctx, createGoodRule); err != nil {
		t.Fatalf("create good rule: %v", err)
	}

	createBadRule, err := domain.ParseCreateRule(project.ID, domain.RuleInput{
		Name:      "Broken Delivery",
		EventType: "ticket.created",
		ChannelID: badChannelID.String(),
		Filter: map[string]any{
			"priority": "high",
		},
	})
	if err != nil {
		t.Fatalf("parse bad rule: %v", err)
	}
	if _, err := service.CreateRule(ctx, createBadRule); err != nil {
		t.Fatalf("create bad rule: %v", err)
	}

	event, err := provider.NewJSONEvent(
		provider.MustParseTopic("ticket.events"),
		provider.MustParseEventType("ticket.created"),
		map[string]any{
			"project_id": project.ID.String(),
			"ticket": map[string]any{
				"id":          uuid.NewString(),
				"identifier":  "ASE-68",
				"title":       "Notification fan-out",
				"status_name": "Backlog",
				"priority":    "high",
				"type":        "feature",
			},
		},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("build event: %v", err)
	}

	if err := bus.Publish(ctx, event); err != nil {
		t.Fatalf("publish event: %v", err)
	}

	select {
	case payload := <-webhookRequests:
		if payload["title"] != "Ticket created: ASE-68" {
			t.Fatalf("unexpected event notification payload: %+v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event-driven notification")
	}
}
