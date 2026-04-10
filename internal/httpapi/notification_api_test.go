package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	"github.com/BetterAndBetterII/openase/internal/provider"
	notificationrepo "github.com/BetterAndBetterII/openase/internal/repo/notification"
	"github.com/google/uuid"
)

func TestNotificationRoutesErrorMappingsAndInvalidPayloads(t *testing.T) {
	client := openTestEntClient(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer webhookServer.Close()

	server := NewServer(
		config.ServerConfig{Port: 40025},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithNotificationService(notificationservice.NewService(notificationrepo.NewEntRepository(client), logger, webhookServer.Client())),
	)
	unavailableServer := NewServer(
		config.ServerConfig{Port: 40026},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase-notification-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Platform").
		SetSlug("platform-notification-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	rec := performJSONRequest(t, unavailableServer, http.MethodGet, fmt.Sprintf("/api/v1/orgs/%s/channels", org.ID), "")
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "SERVICE_UNAVAILABLE") {
		t.Fatalf("list channels unavailable = %d %s", rec.Code, rec.Body.String())
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
	service := notificationservice.NewService(notificationrepo.NewEntRepository(client), logger, webhookServer.Client())
	channel, err := service.Create(ctx, channelInput)
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}

	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "create channel invalid org", method: http.MethodPost, target: "/api/v1/orgs/not-a-uuid/channels", body: `{"name":"Ops","type":"webhook","config":{"url":"https://example.com"}}`, wantStatus: http.StatusBadRequest, wantBody: "orgId must be a valid UUID"},
		{name: "create channel invalid payload", method: http.MethodPost, target: fmt.Sprintf("/api/v1/orgs/%s/channels", org.ID), body: `{"name":"","type":"webhook","config":{}}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update channel invalid id", method: http.MethodPatch, target: "/api/v1/channels/not-a-uuid", body: `{"name":"Ops"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_CHANNEL_ID"},
		{name: "delete channel missing", method: http.MethodDelete, target: fmt.Sprintf("/api/v1/channels/%s", uuid.New()), wantStatus: http.StatusNotFound, wantBody: "CHANNEL_NOT_FOUND"},
		{name: "test channel missing", method: http.MethodPost, target: fmt.Sprintf("/api/v1/channels/%s/test", uuid.New()), wantStatus: http.StatusNotFound, wantBody: "CHANNEL_NOT_FOUND"},
		{name: "list rules invalid project", method: http.MethodGet, target: "/api/v1/projects/not-a-uuid/notification-rules", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create rule invalid payload", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/notification-rules", project.ID), body: `{"name":"","event_type":"","channel_id":"` + channel.ID.String() + `"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update rule invalid id", method: http.MethodPatch, target: "/api/v1/notification-rules/not-a-uuid", body: `{"name":"Ops"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_RULE_ID"},
		{name: "delete rule missing", method: http.MethodDelete, target: fmt.Sprintf("/api/v1/notification-rules/%s", uuid.New()), wantStatus: http.StatusNotFound, wantBody: "RULE_NOT_FOUND"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, testCase.method, testCase.target, testCase.body)
			if rec.Code != testCase.wantStatus || !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("%s %s = %d %s, want %d containing %q", testCase.method, testCase.target, rec.Code, rec.Body.String(), testCase.wantStatus, testCase.wantBody)
			}
		})
	}
}

func TestNotificationRoutesLogStructuredBoundaryErrors(t *testing.T) {
	client := openTestEntClient(t)
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer webhookServer.Close()

	server := NewServer(
		config.ServerConfig{Port: 40025},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithNotificationService(notificationservice.NewService(notificationrepo.NewEntRepository(client), logger, webhookServer.Client())),
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase-notification-log-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	rec := performJSONRequest(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orgs/%s/channels", org.ID), `{"name":"","type":"webhook","config":{}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d %s", rec.Code, rec.Body.String())
	}

	logOutput := logBuffer.String()
	for _, want := range []string{
		"http api boundary error",
		"error_code=INVALID_REQUEST",
		"route=/api/v1/orgs/:orgId/channels",
		"request_id=",
		"operation=api_boundary_error",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("expected log output to contain %q, got %s", want, logOutput)
		}
	}
}

func TestNotificationChannelRoutesCRUDAndTestSend(t *testing.T) {
	client := openTestEntClient(t)

	webhookRequests := make(chan map[string]any, 4)
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
		WithNotificationService(notificationservice.NewService(notificationrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), webhookServer.Client())),
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
		WithNotificationService(notificationservice.NewService(notificationrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), webhookServer.Client())),
	)
	service := notificationservice.NewService(notificationrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), webhookServer.Client())

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
	if eventTypesResp.EventTypes[0].Group == "" || eventTypesResp.EventTypes[0].Level == "" {
		t.Fatalf("expected grouped event catalog metadata, got %+v", eventTypesResp.EventTypes[0])
	}
	gotEventTypes := make(map[string]notificationRuleEventTypeResponse, len(eventTypesResp.EventTypes))
	for _, item := range eventTypesResp.EventTypes {
		gotEventTypes[item.EventType] = item
	}
	for _, want := range []string{
		"ticket.completed",
		"ticket.cancelled",
		"ticket.retry_scheduled",
		"ticket.retry_resumed",
		"ticket.budget_exhausted",
		"machine.connected",
		"machine.reconnected",
		"machine.disconnected",
		"machine.daemon_auth_failed",
	} {
		if _, ok := gotEventTypes[want]; !ok {
			t.Fatalf("expected notification event catalog to include %s, got %+v", want, eventTypesResp.EventTypes)
		}
	}
	for _, excluded := range domain.ExplicitlyUnsupportedRuleEvents() {
		if _, ok := gotEventTypes[excluded.EventType]; ok {
			t.Fatalf("excluded notification event %s leaked into API catalog", excluded.EventType)
		}
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
	service := notificationservice.NewService(notificationrepo.NewEntRepository(client), logger, webhookServer.Client())
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

	createAgentRule, err := domain.ParseCreateRule(project.ID, domain.RuleInput{
		Name:      "Agent Failures",
		EventType: "agent.failed",
		ChannelID: goodChannelID.String(),
	})
	if err != nil {
		t.Fatalf("parse agent rule: %v", err)
	}
	if _, err := service.CreateRule(ctx, createAgentRule); err != nil {
		t.Fatalf("create agent rule: %v", err)
	}

	createHookRule, err := domain.ParseCreateRule(project.ID, domain.RuleInput{
		Name:      "Hook Failures",
		EventType: "hook.failed",
		ChannelID: goodChannelID.String(),
	})
	if err != nil {
		t.Fatalf("parse hook rule: %v", err)
	}
	if _, err := service.CreateRule(ctx, createHookRule); err != nil {
		t.Fatalf("create hook rule: %v", err)
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

	agentCurrentTicketID := uuid.NewString()
	agentEvent, err := provider.NewJSONEvent(
		provider.MustParseTopic("agent.events"),
		provider.MustParseEventType("agent.failed"),
		map[string]any{
			"agent": map[string]any{
				"id":                uuid.NewString(),
				"project_id":        project.ID.String(),
				"name":              "worker-1",
				"status":            "errored",
				"current_ticket_id": agentCurrentTicketID,
			},
		},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("build agent event: %v", err)
	}
	if err := bus.Publish(ctx, agentEvent); err != nil {
		t.Fatalf("publish agent event: %v", err)
	}

	select {
	case payload := <-webhookRequests:
		if payload["title"] != "Agent worker-1 failed ticket "+agentCurrentTicketID {
			t.Fatalf("unexpected agent notification payload: %+v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for agent notification")
	}

	hookEvent, err := provider.NewJSONEvent(
		provider.MustParseTopic("activity.events"),
		provider.MustParseEventType("hook.failed"),
		map[string]any{
			"event": map[string]any{
				"id":         uuid.NewString(),
				"project_id": project.ID.String(),
				"ticket_id":  uuid.NewString(),
				"event_type": "hook.failed",
				"message":    "ASE-68 hook on_done failed",
				"metadata": map[string]any{
					"ticket_identifier": "ASE-68",
					"hook_name":         "on_done",
					"error":             "exit status 1",
				},
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("build hook event: %v", err)
	}
	if err := bus.Publish(ctx, hookEvent); err != nil {
		t.Fatalf("publish hook event: %v", err)
	}

	select {
	case payload := <-webhookRequests:
		if payload["title"] != "ASE-68 hook on_done failed" {
			t.Fatalf("unexpected hook notification payload: %+v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for hook notification")
	}
}

func TestNotificationEngineSupportsEveryConfiguredEventContract(t *testing.T) {
	client := openTestEntClient(t)

	bus := eventinfra.NewChannelBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	webhookRequests := make(chan map[string]any, 32)
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
	service := notificationservice.NewService(notificationrepo.NewEntRepository(client), logger, webhookServer.Client())
	engine := notificationservice.NewEngine(service, bus, logger)
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("start notification engine: %v", err)
	}

	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase-notification-contract").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Platform").
		SetSlug("platform-notification-contract").
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

	for _, item := range domain.SupportedRuleEventContracts() {
		ruleInput, err := domain.ParseCreateRule(project.ID, domain.RuleInput{
			Name:      item.Label,
			EventType: item.EventType.String(),
			ChannelID: channel.ID.String(),
		})
		if err != nil {
			t.Fatalf("parse rule for %s: %v", item.EventType, err)
		}
		if _, err := service.CreateRule(ctx, ruleInput); err != nil {
			t.Fatalf("create rule for %s: %v", item.EventType, err)
		}
	}

	type notificationContractCase struct {
		topic            string
		payload          map[string]any
		wantTitle        string
		wantBodyContains []string
	}

	projectID := project.ID.String()
	ticketPayload := func(identifier string, statusName string, extra map[string]any) map[string]any {
		ticket := map[string]any{
			"id":                 uuid.NewString(),
			"identifier":         identifier,
			"title":              "Notification contract coverage",
			"status_name":        statusName,
			"priority":           "high",
			"type":               "bugfix",
			"retry_paused":       false,
			"pause_reason":       "",
			"budget_usd":         25.0,
			"cost_amount":        12.5,
			"consecutive_errors": 0,
		}
		for key, value := range extra {
			ticket[key] = value
		}
		return map[string]any{
			"project_id": projectID,
			"ticket":     ticket,
		}
	}
	activityPayload := func(eventType string, message string, metadata map[string]any) map[string]any {
		return map[string]any{
			"event": map[string]any{
				"id":         uuid.NewString(),
				"project_id": projectID,
				"ticket_id":  uuid.NewString(),
				"event_type": eventType,
				"message":    message,
				"metadata":   metadata,
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		}
	}

	cases := map[domain.RuleEventType]notificationContractCase{
		domain.RuleEventTypeTicketCreated: {
			topic:     "ticket.events",
			payload:   ticketPayload("ASE-501", "Todo", nil),
			wantTitle: "Ticket created: ASE-501",
			wantBodyContains: []string{
				"Status: Todo",
				"Priority: high",
			},
		},
		domain.RuleEventTypeTicketUpdated: {
			topic:            "ticket.events",
			payload:          ticketPayload("ASE-501", "In Progress", nil),
			wantTitle:        "Ticket updated: ASE-501",
			wantBodyContains: []string{"Status: In Progress"},
		},
		domain.RuleEventTypeTicketStatusChanged: {
			topic:            "ticket.events",
			payload:          ticketPayload("ASE-501", "In Review", nil),
			wantTitle:        "Ticket status changed: ASE-501",
			wantBodyContains: []string{"New status: In Review"},
		},
		domain.RuleEventTypeTicketCompleted: {
			topic:            "ticket.events",
			payload:          ticketPayload("ASE-501", "Done", nil),
			wantTitle:        "Ticket completed: ASE-501",
			wantBodyContains: []string{"Status: Done"},
		},
		domain.RuleEventTypeTicketCancelled: {
			topic:            "ticket.events",
			payload:          ticketPayload("ASE-501", "Cancelled", nil),
			wantTitle:        "Ticket cancelled: ASE-501",
			wantBodyContains: []string{"Status: Cancelled"},
		},
		domain.RuleEventTypeTicketRetryScheduled: {
			topic: "activity.events",
			payload: activityPayload("ticket.retry_scheduled", "Scheduled retry for ASE-501.", map[string]any{
				"ticket": map[string]any{
					"identifier":  "ASE-501",
					"title":       "Notification contract coverage",
					"status_name": "Todo",
					"priority":    "high",
				},
				"next_retry_at":      "2026-04-10T16:30:00Z",
				"consecutive_errors": 2,
			}),
			wantTitle:        "ASE-501 retry scheduled",
			wantBodyContains: []string{"Next retry: 2026-04-10T16:30:00Z"},
		},
		domain.RuleEventTypeTicketRetryResumed: {
			topic: "ticket.events",
			payload: ticketPayload("ASE-501", "Todo", map[string]any{
				"retry_paused": false,
				"pause_reason": "repeated_stalls",
			}),
			wantTitle:        "ASE-501 retry resumed",
			wantBodyContains: []string{"Pause reason: repeated_stalls"},
		},
		domain.RuleEventTypeTicketRetryPaused: {
			topic: "activity.events",
			payload: activityPayload("ticket.retry_paused", "Paused ticket retries after repeated stalls.", map[string]any{
				"ticket_identifier": "ASE-501",
				"pause_reason":      "repeated_stalls",
			}),
			wantTitle:        "Paused ticket retries after repeated stalls.",
			wantBodyContains: []string{"Pause reason: repeated_stalls"},
		},
		domain.RuleEventTypeTicketBudgetExhausted: {
			topic: "activity.events",
			payload: activityPayload("ticket.budget_exhausted", "Paused retries for ASE-501 because the ticket budget is exhausted.", map[string]any{
				"ticket": map[string]any{
					"identifier":  "ASE-501",
					"title":       "Notification contract coverage",
					"status_name": "Todo",
					"priority":    "high",
				},
				"budget_usd":   25.0,
				"cost_amount":  25.0,
				"pause_reason": "budget_exhausted",
			}),
			wantTitle:        "ASE-501 budget exhausted",
			wantBodyContains: []string{"Budget: 25", "Cost: 25"},
		},
		domain.RuleEventTypeAgentClaimed: {
			topic: "agent.events",
			payload: map[string]any{
				"agent": map[string]any{
					"id":                uuid.NewString(),
					"project_id":        projectID,
					"name":              "worker-1",
					"status":            "ready",
					"current_ticket_id": uuid.NewString(),
				},
			},
			wantTitle: "Agent worker-1 claimed ticket",
		},
		domain.RuleEventTypeAgentFailed: {
			topic: "agent.events",
			payload: map[string]any{
				"agent": map[string]any{
					"id":                uuid.NewString(),
					"project_id":        projectID,
					"name":              "worker-1",
					"status":            "errored",
					"current_ticket_id": uuid.NewString(),
				},
			},
			wantTitle:        "Agent worker-1 failed ticket",
			wantBodyContains: []string{"Status: errored"},
		},
		domain.RuleEventTypeHookFailed: {
			topic: "activity.events",
			payload: activityPayload("hook.failed", "ASE-501 hook on_done failed", map[string]any{
				"ticket_identifier": "ASE-501",
				"hook_name":         "on_done",
				"error":             "exit status 1",
			}),
			wantTitle:        "ASE-501 hook on_done failed",
			wantBodyContains: []string{"exit status 1"},
		},
		domain.RuleEventTypeHookPassed: {
			topic: "activity.events",
			payload: activityPayload("hook.passed", "ASE-501 hook on_done passed", map[string]any{
				"ticket_identifier": "ASE-501",
				"hook_name":         "on_done",
			}),
			wantTitle: "ASE-501 hook on_done passed",
		},
		domain.RuleEventTypePROpened: {
			topic: "activity.events",
			payload: activityPayload("pr.opened", "ASE-501 PR opened", map[string]any{
				"ticket_identifier": "ASE-501",
				"pull_request_url":  "https://github.com/acme/openase/pull/501",
			}),
			wantTitle:        "ASE-501 PR opened",
			wantBodyContains: []string{"https://github.com/acme/openase/pull/501"},
		},
		domain.RuleEventTypePRClosed: {
			topic: "activity.events",
			payload: activityPayload("pr.closed", "ASE-501 PR closed", map[string]any{
				"ticket_identifier": "ASE-501",
				"pull_request_url":  "https://github.com/acme/openase/pull/501",
			}),
			wantTitle:        "ASE-501 PR closed",
			wantBodyContains: []string{"https://github.com/acme/openase/pull/501"},
		},
		domain.RuleEventTypeMachineConnected: {
			topic: "activity.events",
			payload: activityPayload("machine.connected", "Machine connected over reverse websocket.", map[string]any{
				"machine_id":      uuid.NewString(),
				"session_id":      "sess-1",
				"transport_mode":  "ws_reverse",
				"connection_mode": "reverse_websocket",
			}),
			wantTitle:        "Machine connected",
			wantBodyContains: []string{"Transport: ws_reverse"},
		},
		domain.RuleEventTypeMachineReconnected: {
			topic: "activity.events",
			payload: activityPayload("machine.reconnected", "Machine reconnected over reverse websocket.", map[string]any{
				"machine_id":      uuid.NewString(),
				"session_id":      "sess-2",
				"transport_mode":  "ws_reverse",
				"connection_mode": "reverse_websocket",
			}),
			wantTitle:        "Machine reconnected",
			wantBodyContains: []string{"Transport: ws_reverse"},
		},
		domain.RuleEventTypeMachineDisconnected: {
			topic: "activity.events",
			payload: activityPayload("machine.disconnected", "Machine disconnected from reverse websocket.", map[string]any{
				"machine_id":      uuid.NewString(),
				"session_id":      "sess-3",
				"transport_mode":  "ws_reverse",
				"connection_mode": "reverse_websocket",
				"reason":          "socket closed",
			}),
			wantTitle:        "Machine disconnected",
			wantBodyContains: []string{"Reason: socket closed"},
		},
		domain.RuleEventTypeMachineDaemonAuthFail: {
			topic: "activity.events",
			payload: activityPayload("machine.daemon_auth_failed", "Machine failed reverse websocket authentication.", map[string]any{
				"machine_id":      uuid.NewString(),
				"session_id":      "sess-4",
				"transport_mode":  "ws_reverse",
				"connection_mode": "reverse_websocket",
				"failure_code":    "token_expired",
				"error":           "token expired",
			}),
			wantTitle:        "Machine daemon auth failed",
			wantBodyContains: []string{"Failure code: token_expired"},
		},
	}

	contracts := domain.SupportedRuleEventContracts()
	if len(cases) != len(contracts) {
		t.Fatalf("notification contract case count = %d, want %d", len(cases), len(contracts))
	}
	for _, item := range contracts {
		if _, ok := cases[item.EventType]; !ok {
			t.Fatalf("missing contract test coverage for %s", item.EventType)
		}
	}
	for eventType := range cases {
		found := false
		for _, item := range contracts {
			if item.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("contract test includes unsupported event %s", eventType)
		}
	}

	for _, item := range contracts {
		tc := cases[item.EventType]
		event, err := provider.NewJSONEvent(
			provider.MustParseTopic(tc.topic),
			provider.MustParseEventType(item.EventType.String()),
			tc.payload,
			time.Now(),
		)
		if err != nil {
			t.Fatalf("build event for %s: %v", item.EventType, err)
		}
		if err := bus.Publish(ctx, event); err != nil {
			t.Fatalf("publish event for %s: %v", item.EventType, err)
		}

		select {
		case payload := <-webhookRequests:
			title, _ := payload["title"].(string)
			body, _ := payload["body"].(string)
			if !strings.Contains(title, tc.wantTitle) {
				t.Fatalf("notification title for %s = %q, want containing %q (payload=%+v)", item.EventType, title, tc.wantTitle, payload)
			}
			for _, want := range tc.wantBodyContains {
				if !strings.Contains(body, want) {
					t.Fatalf("notification body for %s = %q, want containing %q", item.EventType, body, want)
				}
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for notification for %s", item.EventType)
		}
	}
}
