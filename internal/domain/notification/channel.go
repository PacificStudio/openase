package notification

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrChannelTypeUnsupported reports unsupported notification channel types.
	ErrChannelTypeUnsupported = errors.New("notification channel type is not supported")
)

// ChannelType identifies a notification delivery integration.
type ChannelType string

const (
	ChannelTypeWebhook  ChannelType = "webhook"
	ChannelTypeTelegram ChannelType = "telegram"
	ChannelTypeSlack    ChannelType = "slack"
	ChannelTypeWeCom    ChannelType = "wecom"
)

// ParseChannelType validates a raw channel type string.
func ParseChannelType(raw string) (ChannelType, error) {
	channelType := ChannelType(strings.ToLower(strings.TrimSpace(raw)))
	switch channelType {
	case ChannelTypeWebhook, ChannelTypeTelegram, ChannelTypeSlack, ChannelTypeWeCom:
		return channelType, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrChannelTypeUnsupported, strings.TrimSpace(raw))
	}
}

func (t ChannelType) String() string {
	return string(t)
}

// Channel stores the persisted notification channel aggregate.
type Channel struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Type           ChannelType
	Config         map[string]any
	IsEnabled      bool
	CreatedAt      time.Time
}

// Message is the adapter-facing notification payload.
type Message struct {
	Title    string
	Body     string
	Level    string
	Link     string
	Metadata map[string]string
}

// Optional captures whether a field was set in a patch request.
type Optional[T any] struct {
	Set   bool
	Value T
}

// Some marks a value as explicitly set.
func Some[T any](value T) Optional[T] {
	return Optional[T]{
		Set:   true,
		Value: value,
	}
}

// ChannelInput carries raw create/update request values before parsing.
type ChannelInput struct {
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Config    map[string]any `json:"config"`
	IsEnabled *bool          `json:"is_enabled"`
}

// ChannelPatchInput carries raw patch values before parsing.
type ChannelPatchInput struct {
	Name      *string         `json:"name"`
	Type      *string         `json:"type"`
	Config    *map[string]any `json:"config"`
	IsEnabled *bool           `json:"is_enabled"`
}

// CreateChannelInput is the validated create command input.
type CreateChannelInput struct {
	OrganizationID uuid.UUID
	Name           string
	Type           ChannelType
	Config         map[string]any
	IsEnabled      bool
}

// UpdateChannelInput is the validated patch command input.
type UpdateChannelInput struct {
	ChannelID uuid.UUID
	Name      Optional[string]
	Type      Optional[ChannelType]
	Config    Optional[map[string]any]
	IsEnabled Optional[bool]
}

// ChannelConfig is the marker interface for validated adapter configuration.
type ChannelConfig interface {
	channelType() ChannelType
}

// WebhookConfig holds generic webhook delivery settings.
type WebhookConfig struct {
	URL     string
	Headers map[string]string
	Secret  string
}

func (WebhookConfig) channelType() ChannelType { return ChannelTypeWebhook }

// TelegramConfig holds Bot API credentials.
type TelegramConfig struct {
	BotToken string
	ChatID   string
}

func (TelegramConfig) channelType() ChannelType { return ChannelTypeTelegram }

// SlackConfig holds Slack incoming webhook settings.
type SlackConfig struct {
	WebhookURL string
}

func (SlackConfig) channelType() ChannelType { return ChannelTypeSlack }

// WeComConfig holds WeCom robot webhook settings.
type WeComConfig struct {
	WebhookKey string
}

func (WeComConfig) channelType() ChannelType { return ChannelTypeWeCom }

// ParseCreateChannel validates an incoming channel create request.
func ParseCreateChannel(organizationID uuid.UUID, raw ChannelInput) (CreateChannelInput, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return CreateChannelInput{}, fmt.Errorf("name must not be empty")
	}

	channelType, err := ParseChannelType(raw.Type)
	if err != nil {
		return CreateChannelInput{}, err
	}

	config, err := NormalizeChannelConfig(channelType, raw.Config)
	if err != nil {
		return CreateChannelInput{}, err
	}

	isEnabled := true
	if raw.IsEnabled != nil {
		isEnabled = *raw.IsEnabled
	}

	return CreateChannelInput{
		OrganizationID: organizationID,
		Name:           name,
		Type:           channelType,
		Config:         config,
		IsEnabled:      isEnabled,
	}, nil
}

// ParseUpdateChannel validates a raw patch request into a typed update command.
func ParseUpdateChannel(channelID uuid.UUID, raw ChannelPatchInput) (UpdateChannelInput, error) {
	input := UpdateChannelInput{ChannelID: channelID}

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return UpdateChannelInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = Some(name)
	}

	if raw.Type != nil {
		channelType, err := ParseChannelType(*raw.Type)
		if err != nil {
			return UpdateChannelInput{}, err
		}
		input.Type = Some(channelType)
	}

	if raw.Config != nil {
		copied, err := cloneRawConfig(*raw.Config)
		if err != nil {
			return UpdateChannelInput{}, err
		}
		input.Config = Some(copied)
	}

	if raw.IsEnabled != nil {
		input.IsEnabled = Some(*raw.IsEnabled)
	}

	if !input.Name.Set && !input.Type.Set && !input.Config.Set && !input.IsEnabled.Set {
		return UpdateChannelInput{}, fmt.Errorf("patch request must update at least one field")
	}

	return input, nil
}

// ParseChannelConfig turns raw persisted config into a validated typed config.
func ParseChannelConfig(channelType ChannelType, raw map[string]any) (ChannelConfig, error) {
	switch channelType {
	case ChannelTypeWebhook:
		urlValue, err := requireHTTPURL(raw, "url")
		if err != nil {
			return nil, err
		}
		headers, err := optionalHeadersMap(raw)
		if err != nil {
			return nil, err
		}
		secret, err := optionalString(raw, "secret")
		if err != nil {
			return nil, err
		}
		return WebhookConfig{
			URL:     urlValue,
			Headers: headers,
			Secret:  secret,
		}, nil
	case ChannelTypeTelegram:
		botToken, err := requireString(raw, "bot_token")
		if err != nil {
			return nil, err
		}
		chatID, err := requireString(raw, "chat_id")
		if err != nil {
			return nil, err
		}
		return TelegramConfig{
			BotToken: botToken,
			ChatID:   chatID,
		}, nil
	case ChannelTypeSlack:
		webhookURL, err := requireHTTPURL(raw, "webhook_url")
		if err != nil {
			return nil, err
		}
		return SlackConfig{WebhookURL: webhookURL}, nil
	case ChannelTypeWeCom:
		webhookKey, err := requireString(raw, "webhook_key")
		if err != nil {
			return nil, err
		}
		return WeComConfig{WebhookKey: webhookKey}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrChannelTypeUnsupported, channelType)
	}
}

// NormalizeChannelConfig validates and normalizes raw config before persistence.
func NormalizeChannelConfig(channelType ChannelType, raw map[string]any) (map[string]any, error) {
	switch channelType {
	case ChannelTypeWebhook:
		config, err := ParseChannelConfig(channelType, raw)
		if err != nil {
			return nil, err
		}
		typed := config.(WebhookConfig)
		normalized := map[string]any{
			"url": typed.URL,
		}
		if len(typed.Headers) > 0 {
			normalized["headers"] = copyStringMap(typed.Headers)
		}
		if typed.Secret != "" {
			normalized["secret"] = typed.Secret
		}
		return normalized, nil
	case ChannelTypeTelegram:
		config, err := ParseChannelConfig(channelType, raw)
		if err != nil {
			return nil, err
		}
		typed := config.(TelegramConfig)
		return map[string]any{
			"bot_token": typed.BotToken,
			"chat_id":   typed.ChatID,
		}, nil
	case ChannelTypeSlack:
		config, err := ParseChannelConfig(channelType, raw)
		if err != nil {
			return nil, err
		}
		typed := config.(SlackConfig)
		return map[string]any{
			"webhook_url": typed.WebhookURL,
		}, nil
	case ChannelTypeWeCom:
		config, err := ParseChannelConfig(channelType, raw)
		if err != nil {
			return nil, err
		}
		typed := config.(WeComConfig)
		return map[string]any{
			"webhook_key": typed.WebhookKey,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrChannelTypeUnsupported, channelType)
	}
}

// RedactedConfig returns a response-safe config snapshot.
func RedactedConfig(channelType ChannelType, raw map[string]any) map[string]any {
	copied, err := cloneRawConfig(raw)
	if err != nil {
		return map[string]any{}
	}

	switch channelType {
	case ChannelTypeWebhook:
		if secret, ok := copied["secret"].(string); ok && strings.TrimSpace(secret) != "" {
			copied["secret"] = redactSecret(secret)
		}
		if rawURL, ok := copied["url"].(string); ok {
			copied["url"] = redactURL(rawURL)
		}
	case ChannelTypeTelegram:
		if token, ok := copied["bot_token"].(string); ok {
			copied["bot_token"] = redactSecret(token)
		}
	case ChannelTypeSlack:
		if webhookURL, ok := copied["webhook_url"].(string); ok {
			copied["webhook_url"] = redactURL(webhookURL)
		}
	case ChannelTypeWeCom:
		if webhookKey, ok := copied["webhook_key"].(string); ok {
			copied["webhook_key"] = redactSecret(webhookKey)
		}
	}

	return copied
}

// HMACPreview returns a deterministic header-safe signature preview for masked responses.
func HMACPreview(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:8])
}

func cloneRawConfig(raw map[string]any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}

	copied := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned, err := cloneRawConfigValue(key, value)
		if err != nil {
			return nil, err
		}
		copied[key] = cloned
	}

	return copied, nil
}

func cloneRawConfigValue(path string, value any) (any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		return typed, nil
	case bool:
		return typed, nil
	case float64:
		return typed, nil
	case int:
		return typed, nil
	case map[string]any:
		return cloneRawConfig(typed)
	case []any:
		items := make([]any, 0, len(typed))
		for idx, item := range typed {
			cloned, err := cloneRawConfigValue(fmt.Sprintf("%s[%d]", path, idx), item)
			if err != nil {
				return nil, err
			}
			items = append(items, cloned)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("config.%s must contain JSON scalar, array, or object values", path)
	}
}

func requireString(raw map[string]any, field string) (string, error) {
	value, ok := raw[field]
	if !ok {
		return "", fmt.Errorf("config.%s is required", field)
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("config.%s must be a string", field)
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", fmt.Errorf("config.%s must not be empty", field)
	}

	return trimmed, nil
}

func optionalString(raw map[string]any, field string) (string, error) {
	value, ok := raw[field]
	if !ok {
		return "", nil
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("config.%s must be a string", field)
	}

	return strings.TrimSpace(text), nil
}

func requireHTTPURL(raw map[string]any, field string) (string, error) {
	value, err := requireString(raw, field)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("config.%s must be a valid URL", field)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("config.%s must use http or https", field)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("config.%s must include a host", field)
	}

	return value, nil
}

func optionalHeadersMap(raw map[string]any) (map[string]string, error) {
	const field = "headers"

	value, ok := raw[field]
	if !ok {
		return nil, nil
	}

	items, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config.%s must be an object", field)
	}

	result := make(map[string]string, len(items))
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		text, ok := items[key].(string)
		if !ok {
			return nil, fmt.Errorf("config.%s.%s must be a string", field, key)
		}
		result[key] = strings.TrimSpace(text)
	}

	return result, nil
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	copied := make(map[string]string, len(input))
	for key, value := range input {
		copied[key] = value
	}

	return copied
}

func redactSecret(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return strings.Repeat("*", len(trimmed))
	}

	return strings.Repeat("*", len(trimmed)-4) + trimmed[len(trimmed)-4:]
}

func redactURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "redacted"
	}
	if parsed.Host == "" {
		return "redacted"
	}

	return parsed.Scheme + "://" + parsed.Host + "/***"
}
