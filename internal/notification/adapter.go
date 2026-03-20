package notification

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
)

var (
	// ErrAdapterUnavailable reports a missing adapter implementation.
	ErrAdapterUnavailable = errors.New("notification channel adapter is not available")
)

// ChannelAdapter provides transport-specific delivery logic.
type ChannelAdapter interface {
	Type() domain.ChannelType
	Send(ctx context.Context, cfg domain.ChannelConfig, msg domain.Message) error
	Validate(ctx context.Context, cfg domain.ChannelConfig) error
}

// AdapterRegistry resolves adapters by channel type.
type AdapterRegistry struct {
	adapters map[domain.ChannelType]ChannelAdapter
}

// NewDefaultAdapterRegistry constructs the built-in adapter registry.
func NewDefaultAdapterRegistry(httpClient *http.Client) *AdapterRegistry {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return NewAdapterRegistry(
		&WebhookAdapter{client: client},
		&TelegramAdapter{client: client},
		&SlackAdapter{client: client},
		&WeComAdapter{client: client},
	)
}

// NewAdapterRegistry constructs a registry from explicit adapters.
func NewAdapterRegistry(adapters ...ChannelAdapter) *AdapterRegistry {
	items := make(map[domain.ChannelType]ChannelAdapter, len(adapters))
	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		items[adapter.Type()] = adapter
	}

	return &AdapterRegistry{adapters: items}
}

// Get resolves the adapter for the given channel type.
func (r *AdapterRegistry) Get(channelType domain.ChannelType) (ChannelAdapter, error) {
	if r == nil {
		return nil, ErrAdapterUnavailable
	}

	adapter, ok := r.adapters[channelType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrAdapterUnavailable, channelType)
	}

	return adapter, nil
}

// WebhookAdapter sends JSON payloads to arbitrary HTTP endpoints.
type WebhookAdapter struct {
	client *http.Client
}

func (a *WebhookAdapter) Type() domain.ChannelType {
	return domain.ChannelTypeWebhook
}

func (a *WebhookAdapter) Send(ctx context.Context, cfg domain.ChannelConfig, msg domain.Message) error {
	webhookConfig, ok := cfg.(domain.WebhookConfig)
	if !ok {
		return fmt.Errorf("webhook adapter requires %T config", domain.WebhookConfig{})
	}

	payload := map[string]any{
		"title":    msg.Title,
		"body":     msg.Body,
		"level":    msg.Level,
		"link":     msg.Link,
		"metadata": msg.Metadata,
		"sent_at":  time.Now().UTC().Format(time.RFC3339),
		"source":   "openase",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookConfig.URL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range webhookConfig.Headers {
		req.Header.Set(key, value)
	}
	if webhookConfig.Secret != "" {
		req.Header.Set("X-OpenASE-Signature", webhookSignature(webhookConfig.Secret, body))
	}

	return doRequest(a.client, req)
}

func (a *WebhookAdapter) Validate(_ context.Context, cfg domain.ChannelConfig) error {
	_, ok := cfg.(domain.WebhookConfig)
	if !ok {
		return fmt.Errorf("webhook adapter requires %T config", domain.WebhookConfig{})
	}
	return nil
}

// TelegramAdapter sends plain-text Bot API messages.
type TelegramAdapter struct {
	client *http.Client
}

func (a *TelegramAdapter) Type() domain.ChannelType {
	return domain.ChannelTypeTelegram
}

func (a *TelegramAdapter) Send(ctx context.Context, cfg domain.ChannelConfig, msg domain.Message) error {
	telegramConfig, ok := cfg.(domain.TelegramConfig)
	if !ok {
		return fmt.Errorf("telegram adapter requires %T config", domain.TelegramConfig{})
	}

	form := url.Values{}
	form.Set("chat_id", telegramConfig.ChatID)
	form.Set("text", renderPlainText(msg))

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramConfig.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return doRequest(a.client, req)
}

func (a *TelegramAdapter) Validate(_ context.Context, cfg domain.ChannelConfig) error {
	_, ok := cfg.(domain.TelegramConfig)
	if !ok {
		return fmt.Errorf("telegram adapter requires %T config", domain.TelegramConfig{})
	}
	return nil
}

// SlackAdapter sends simple text payloads to incoming webhooks.
type SlackAdapter struct {
	client *http.Client
}

func (a *SlackAdapter) Type() domain.ChannelType {
	return domain.ChannelTypeSlack
}

func (a *SlackAdapter) Send(ctx context.Context, cfg domain.ChannelConfig, msg domain.Message) error {
	slackConfig, ok := cfg.(domain.SlackConfig)
	if !ok {
		return fmt.Errorf("slack adapter requires %T config", domain.SlackConfig{})
	}

	payload, err := json.Marshal(map[string]string{
		"text": renderPlainText(msg),
	})
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackConfig.WebhookURL, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return doRequest(a.client, req)
}

func (a *SlackAdapter) Validate(_ context.Context, cfg domain.ChannelConfig) error {
	_, ok := cfg.(domain.SlackConfig)
	if !ok {
		return fmt.Errorf("slack adapter requires %T config", domain.SlackConfig{})
	}
	return nil
}

// WeComAdapter sends markdown robot webhook messages.
type WeComAdapter struct {
	client *http.Client
}

func (a *WeComAdapter) Type() domain.ChannelType {
	return domain.ChannelTypeWeCom
}

func (a *WeComAdapter) Send(ctx context.Context, cfg domain.ChannelConfig, msg domain.Message) error {
	wecomConfig, ok := cfg.(domain.WeComConfig)
	if !ok {
		return fmt.Errorf("wecom adapter requires %T config", domain.WeComConfig{})
	}

	payload, err := json.Marshal(map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": renderMarkdown(msg),
		},
	})
	if err != nil {
		return fmt.Errorf("marshal wecom payload: %w", err)
	}

	endpoint := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s", wecomConfig.WebhookKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("build wecom request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return doRequest(a.client, req)
}

func (a *WeComAdapter) Validate(_ context.Context, cfg domain.ChannelConfig) error {
	_, ok := cfg.(domain.WeComConfig)
	if !ok {
		return fmt.Errorf("wecom adapter requires %T config", domain.WeComConfig{})
	}
	return nil
}

func doRequest(client *http.Client, req *http.Request) error {
	//nolint:gosec // target URLs come from parsed channel config and are an intended integration surface
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("unexpected response %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func renderPlainText(msg domain.Message) string {
	lines := make([]string, 0, 4)
	if title := strings.TrimSpace(msg.Title); title != "" {
		lines = append(lines, title)
	}
	if body := strings.TrimSpace(msg.Body); body != "" {
		lines = append(lines, body)
	}
	if link := strings.TrimSpace(msg.Link); link != "" {
		lines = append(lines, "Link: "+link)
	}

	return strings.Join(lines, "\n")
}

func renderMarkdown(msg domain.Message) string {
	return renderPlainText(msg)
}

func webhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
