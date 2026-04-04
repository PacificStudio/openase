package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestNotificationServiceNilClientGuards(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil)
	ctx := context.Background()
	orgID := uuid.New()
	projectID := uuid.New()
	channelID := uuid.New()
	ruleID := uuid.New()

	if _, err := service.List(ctx, orgID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("List error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Get(ctx, channelID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListRules(ctx, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListRules error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Create(ctx, domain.CreateChannelInput{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.CreateRule(ctx, domain.CreateRuleInput{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("CreateRule error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Update(ctx, domain.UpdateChannelInput{ChannelID: channelID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Update error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.UpdateRule(ctx, domain.UpdateRuleInput{RuleID: ruleID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("UpdateRule error = %v, want %v", err, ErrUnavailable)
	}
	if err := service.Delete(ctx, channelID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Delete error = %v, want %v", err, ErrUnavailable)
	}
	if err := service.DeleteRule(ctx, ruleID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("DeleteRule error = %v, want %v", err, ErrUnavailable)
	}
	if err := service.Test(ctx, channelID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Test error = %v, want %v", err, ErrUnavailable)
	}
	if err := service.SendToProjectChannels(ctx, projectID, domain.Message{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("SendToProjectChannels error = %v, want %v", err, ErrUnavailable)
	}
	if err := service.SendRule(ctx, domain.Rule{}, domain.Message{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("SendRule error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.MatchingRules(ctx, projectID, domain.RuleEventTypeTicketCreated); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("MatchingRules error = %v, want %v", err, ErrUnavailable)
	}
}

func TestNotificationRegistryAndRenderHelpers(t *testing.T) {
	t.Parallel()

	registry := NewDefaultAdapterRegistry(nil)
	if registry == nil {
		t.Fatal("NewDefaultAdapterRegistry() = nil")
	}
	for _, channelType := range []domain.ChannelType{
		domain.ChannelTypeWebhook,
		domain.ChannelTypeTelegram,
		domain.ChannelTypeSlack,
		domain.ChannelTypeWeCom,
	} {
		adapter, err := registry.Get(channelType)
		if err != nil {
			t.Fatalf("registry.Get(%s) error = %v", channelType, err)
		}
		if adapter.Type() != channelType {
			t.Fatalf("adapter.Type() = %s, want %s", adapter.Type(), channelType)
		}
	}
	if _, err := (*AdapterRegistry)(nil).Get(domain.ChannelTypeWebhook); !errors.Is(err, ErrAdapterUnavailable) {
		t.Fatalf("nil registry Get error = %v, want %v", err, ErrAdapterUnavailable)
	}
	if _, err := NewAdapterRegistry().Get(domain.ChannelTypeWebhook); !errors.Is(err, ErrAdapterUnavailable) {
		t.Fatalf("empty registry Get error = %v, want %v", err, ErrAdapterUnavailable)
	}

	msg := domain.Message{
		Title: "OpenASE",
		Body:  "Coverage gate",
		Link:  "https://example.com",
	}
	wantText := "OpenASE\nCoverage gate\nLink: https://example.com"
	if got := renderPlainText(msg); got != wantText {
		t.Fatalf("renderPlainText = %q, want %q", got, wantText)
	}
	if got := renderMarkdown(msg); got != wantText {
		t.Fatalf("renderMarkdown = %q, want %q", got, wantText)
	}
	if got := webhookSignature("secret", []byte("payload")); got != "sha256=b82fcb791acec57859b989b430a826488ce2e479fdf92326bd0a2e8375a42ba4" {
		t.Fatalf("webhookSignature = %q", got)
	}
}

func TestNotificationServiceLogsInvalidChannelConfig(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	service := NewService(nil, logger, nil)
	channelID := uuid.New()

	err := service.sendChannel(context.Background(), domain.Channel{
		ID:   channelID,
		Name: "Broken Webhook",
		Type: domain.ChannelTypeWebhook,
		Config: map[string]any{
			"url": "bad",
		},
	}, domain.Message{Title: "Ship"})
	if err == nil || !errors.Is(err, ErrInvalidChannelConfig) {
		t.Fatalf("sendChannel() error = %v, want %v", err, ErrInvalidChannelConfig)
	}

	logOutput := logBuffer.String()
	for _, want := range []string{
		"notification channel config invalid",
		"channel_id=" + channelID.String(),
		"operation=parse_notification_channel_config",
		"config_keys=[url]",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("expected log output to contain %q, got %s", want, logOutput)
		}
	}
}

func TestNotificationAdapters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		adapter       ChannelAdapter
		config        domain.ChannelConfig
		wantContains  []string
		wantHeaders   map[string]string
		wantURLSubstr string
	}{
		{
			name: "webhook",
			adapter: &WebhookAdapter{client: testHTTPClient(func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				for _, part := range []string{`"title":"Ship"`, `"body":"It works"`, `"source":"openase"`} {
					if !strings.Contains(string(body), part) {
						t.Fatalf("webhook body %q missing %q", string(body), part)
					}
				}
				if got := req.Header.Get("X-Test"); got != "1" {
					t.Fatalf("webhook X-Test header = %q, want 1", got)
				}
				if got := req.Header.Get("X-OpenASE-Signature"); got == "" {
					t.Fatal("webhook signature header missing")
				}
				return okResponse(), nil
			})},
			config: domain.WebhookConfig{
				URL:     "https://hooks.example.com/openase",
				Headers: map[string]string{"X-Test": "1"},
				Secret:  "secret",
			},
		},
		{
			name: "telegram",
			adapter: &TelegramAdapter{client: testHTTPClient(func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				if !strings.Contains(req.URL.String(), "/bottoken/sendMessage") {
					t.Fatalf("telegram url = %s", req.URL.String())
				}
				if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
					t.Fatalf("telegram content-type = %q", got)
				}
				if !strings.Contains(string(body), "chat_id=chat") || !strings.Contains(string(body), "text=Ship%0AIt+works") {
					t.Fatalf("telegram body = %q", string(body))
				}
				return okResponse(), nil
			})},
			config: domain.TelegramConfig{BotToken: "token", ChatID: "chat"},
		},
		{
			name: "slack",
			adapter: &SlackAdapter{client: testHTTPClient(func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				if req.URL.String() != "https://slack.example.com/hook" {
					t.Fatalf("slack url = %s", req.URL.String())
				}
				if got := req.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("slack content-type = %q", got)
				}
				if !strings.Contains(string(body), `"text":"Ship\nIt works"`) {
					t.Fatalf("slack body = %q", string(body))
				}
				return okResponse(), nil
			})},
			config: domain.SlackConfig{WebhookURL: "https://slack.example.com/hook"},
		},
		{
			name: "wecom",
			adapter: &WeComAdapter{client: testHTTPClient(func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				if !strings.Contains(req.URL.String(), "key=abc123") {
					t.Fatalf("wecom url = %s", req.URL.String())
				}
				if !strings.Contains(string(body), `"msgtype":"markdown"`) || !strings.Contains(string(body), "Ship\\nIt works") {
					t.Fatalf("wecom body = %q", string(body))
				}
				return okResponse(), nil
			})},
			config: domain.WeComConfig{WebhookKey: "abc123"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := domain.Message{Title: "Ship", Body: "It works"}
			if err := tt.adapter.Validate(context.Background(), tt.config); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if err := tt.adapter.Send(context.Background(), tt.config, msg); err != nil {
				t.Fatalf("Send() error = %v", err)
			}
		})
	}

	if err := (&WebhookAdapter{}).Validate(context.Background(), domain.TelegramConfig{}); err == nil {
		t.Fatal("WebhookAdapter.Validate() expected type error")
	}
	if err := (&TelegramAdapter{}).Validate(context.Background(), domain.SlackConfig{}); err == nil {
		t.Fatal("TelegramAdapter.Validate() expected type error")
	}
	if err := (&SlackAdapter{}).Validate(context.Background(), domain.WeComConfig{}); err == nil {
		t.Fatal("SlackAdapter.Validate() expected type error")
	}
	if err := (&WeComAdapter{}).Validate(context.Background(), domain.WebhookConfig{}); err == nil {
		t.Fatal("WeComAdapter.Validate() expected type error")
	}
}

func TestNotificationRequestAndContextHelpers(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://hooks.example.com", bytes.NewBufferString("payload"))
	if err != nil {
		t.Fatalf("NewRequestWithContext error = %v", err)
	}
	if err := doRequest(testHTTPClient(func(*http.Request) (*http.Response, error) {
		return okResponse(), nil
	}), req); err != nil {
		t.Fatalf("doRequest success error = %v", err)
	}
	if err := doRequest(testHTTPClient(func(*http.Request) (*http.Response, error) {
		return errorResponse(http.StatusBadGateway, " upstream failed "), nil
	}), req); err == nil || !strings.Contains(err.Error(), "unexpected response 502: upstream failed") {
		t.Fatalf("doRequest failure error = %v", err)
	}
	if err := doRequest(testHTTPClient(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	}), req); err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("doRequest transport error = %v", err)
	}

	now := time.Date(2026, 3, 27, 12, 34, 56, 0, time.FixedZone("UTC+2", 2*60*60))
	projectID := uuid.New()
	event, err := provider.NewJSONEvent(
		ticketEventsTopic,
		ticketStatusEventType,
		map[string]any{
			"project_id": projectID.String(),
			"ticket": map[string]any{
				"identifier":  "ASE-278",
				"status_name": "In Review",
			},
		},
		now,
	)
	if err != nil {
		t.Fatalf("NewJSONEvent error = %v", err)
	}
	gotProjectID, contextMap, err := buildRuleContext(event)
	if err != nil {
		t.Fatalf("buildRuleContext error = %v", err)
	}
	if gotProjectID != projectID {
		t.Fatalf("buildRuleContext projectID = %s, want %s", gotProjectID, projectID)
	}
	if contextMap["event_type"] != ticketStatusEventType.String() || contextMap["new_status"] != "In Review" {
		t.Fatalf("buildRuleContext context = %+v", contextMap)
	}

	badTopicEvent := event
	badTopicEvent.Topic = provider.MustParseTopic("runtime.events")
	if _, _, err := buildRuleContext(badTopicEvent); err == nil || !strings.Contains(err.Error(), "unsupported topic") {
		t.Fatalf("buildRuleContext bad topic error = %v", err)
	}
	badJSONEvent := event
	badJSONEvent.Payload = []byte("{")
	if _, _, err := buildRuleContext(badJSONEvent); err == nil || !strings.Contains(err.Error(), "decode ticket event payload") {
		t.Fatalf("buildRuleContext bad json error = %v", err)
	}
	missingProjectEvent, err := provider.NewJSONEvent(ticketEventsTopic, ticketStatusEventType, map[string]any{}, now)
	if err != nil {
		t.Fatalf("NewJSONEvent missing project error = %v", err)
	}
	if _, _, err := buildRuleContext(missingProjectEvent); err == nil || !strings.Contains(err.Error(), "project_id is missing") {
		t.Fatalf("buildRuleContext missing project error = %v", err)
	}
}

func TestNotificationEngineLifecycleAndDisabledSendRule(t *testing.T) {
	t.Parallel()

	buffer := bytes.NewBuffer(nil)
	logger := slog.New(slog.NewTextHandler(buffer, nil))
	events := &fakeEventProvider{subscribed: make(chan struct{}, 1), stream: make(chan provider.Event)}
	service := NewService(nil, logger, nil)

	engine := NewEngine(service, events, nil)
	if engine == nil || engine.logger == nil {
		t.Fatalf("NewEngine() = %+v, want initialized engine", engine)
	}
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	select {
	case <-events.subscribed:
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not subscribe to ticket events")
	}
	close(events.stream)

	if err := NewEngine(nil, events, logger).Start(context.Background()); err != nil {
		t.Fatalf("Start() with nil service error = %v", err)
	}
	if err := NewEngine(service, nil, logger).Start(context.Background()); err != nil {
		t.Fatalf("Start() with nil events error = %v", err)
	}

	guardBypassService := &Service{repo: fakeNotificationRepository{}}
	disabledRule := domain.Rule{IsEnabled: false, Channel: domain.Channel{IsEnabled: true}}
	if err := guardBypassService.SendRule(context.Background(), disabledRule, domain.Message{}); err != nil {
		t.Fatalf("SendRule disabled error = %v", err)
	}
	channelDisabledRule := domain.Rule{IsEnabled: true, Channel: domain.Channel{IsEnabled: false}}
	if err := guardBypassService.SendRule(context.Background(), channelDisabledRule, domain.Message{}); err != nil {
		t.Fatalf("SendRule channel-disabled error = %v", err)
	}
}

type fakeEventProvider struct {
	subscribed chan struct{}
	stream     chan provider.Event
}

func (f *fakeEventProvider) Publish(_ context.Context, _ provider.Event) error {
	return nil
}

func (f *fakeEventProvider) Subscribe(_ context.Context, topics ...provider.Topic) (<-chan provider.Event, error) {
	if len(topics) != 1 || topics[0] != ticketEventsTopic {
		return nil, errors.New("unexpected topics")
	}
	select {
	case f.subscribed <- struct{}{}:
	default:
	}
	return f.stream, nil
}

func (f *fakeEventProvider) Close() error {
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func okResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
	}
}

func errorResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNotificationServiceUsesFallbackLogger(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil)
	if service.logger == nil || service.registry == nil {
		t.Fatalf("NewService() = %+v, want logger and registry", service)
	}
}

func TestNotificationWebhookPayloadIsJSON(t *testing.T) {
	t.Parallel()

	var payload map[string]any
	adapter := &WebhookAdapter{client: testHTTPClient(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("json.Unmarshal webhook payload: %v", err)
		}
		return okResponse(), nil
	})}
	if err := adapter.Send(context.Background(), domain.WebhookConfig{URL: "https://hooks.example.com"}, domain.Message{
		Title:    "Ship",
		Body:     "It works",
		Level:    "info",
		Metadata: map[string]string{"issue": "278"},
	}); err != nil {
		t.Fatalf("WebhookAdapter.Send() error = %v", err)
	}
	if payload["source"] != "openase" || payload["level"] != "info" {
		t.Fatalf("webhook payload = %+v", payload)
	}
}

type fakeNotificationRepository struct{}

func (fakeNotificationRepository) OrganizationExists(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

func (fakeNotificationRepository) Project(context.Context, uuid.UUID) (domain.ProjectRef, error) {
	return domain.ProjectRef{}, nil
}

func (fakeNotificationRepository) Channels(context.Context, uuid.UUID, bool) ([]domain.Channel, error) {
	return nil, nil
}

func (fakeNotificationRepository) Channel(context.Context, uuid.UUID) (domain.Channel, error) {
	return domain.Channel{}, nil
}

func (fakeNotificationRepository) CreateChannel(context.Context, domain.CreateChannelInput) (domain.Channel, error) {
	return domain.Channel{}, nil
}

func (fakeNotificationRepository) UpdateChannel(context.Context, domain.Channel) (domain.Channel, error) {
	return domain.Channel{}, nil
}

func (fakeNotificationRepository) DeleteChannel(context.Context, uuid.UUID) error {
	return nil
}

func (fakeNotificationRepository) Rules(context.Context, uuid.UUID) ([]domain.Rule, error) {
	return nil, nil
}

func (fakeNotificationRepository) Rule(context.Context, uuid.UUID) (domain.Rule, error) {
	return domain.Rule{}, nil
}

func (fakeNotificationRepository) CreateRule(context.Context, domain.CreateRuleInput) (domain.Rule, error) {
	return domain.Rule{}, nil
}

func (fakeNotificationRepository) UpdateRule(context.Context, domain.Rule) (domain.Rule, error) {
	return domain.Rule{}, nil
}

func (fakeNotificationRepository) DeleteRule(context.Context, uuid.UUID) error {
	return nil
}

func (fakeNotificationRepository) MatchingRules(context.Context, uuid.UUID, domain.RuleEventType) ([]domain.Rule, error) {
	return nil, nil
}
