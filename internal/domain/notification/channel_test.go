package notification

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCloneRawConfigClonesSlices(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"headers": []any{
			"X-Test",
			map[string]any{"nested": "value"},
		},
	}

	cloned, err := cloneRawConfig(raw)
	if err != nil {
		t.Fatalf("cloneRawConfig() error = %v", err)
	}

	items, ok := cloned["headers"].([]any)
	if !ok {
		t.Fatalf("cloneRawConfig() headers type = %T, want []any", cloned["headers"])
	}
	nested, ok := items[1].(map[string]any)
	if !ok {
		t.Fatalf("cloneRawConfig() nested type = %T, want map[string]any", items[1])
	}

	nested["nested"] = "changed"

	originalItems := raw["headers"].([]any)
	originalNested := originalItems[1].(map[string]any)
	if got := originalNested["nested"]; got != "value" {
		t.Fatalf("cloneRawConfig() mutated original nested map = %v, want value", got)
	}

	if got, err := cloneRawConfig(nil); err != nil || len(got) != 0 {
		t.Fatalf("cloneRawConfig(nil) = %+v, %v; want empty map, nil", got, err)
	}
	if _, err := cloneRawConfig(map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("cloneRawConfig() expected validation error")
	}
}

func TestRequireHTTPURLAcceptsHTTPAndHTTPS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  map[string]any
	}{
		{
			name: "http",
			raw:  map[string]any{"url": "http://example.com/hook"},
		},
		{
			name: "https",
			raw:  map[string]any{"url": "https://example.com/hook"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := requireHTTPURL(tt.raw, "url")
			if err != nil {
				t.Fatalf("requireHTTPURL() error = %v", err)
			}
			if got != tt.raw["url"] {
				t.Fatalf("requireHTTPURL() = %q, want %q", got, tt.raw["url"])
			}
		})
	}
}

func TestChannelParsingAndRedactionHelpers(t *testing.T) {
	webhookConfig := map[string]any{
		"url":     "https://example.com/hook",
		"headers": map[string]any{"X-Test": " one "},
		"secret":  " super-secret ",
	}
	createInput, err := ParseCreateChannel(uuid.New(), ChannelInput{
		Name:   " Deploy Hook ",
		Type:   " WEBHOOK ",
		Config: webhookConfig,
	})
	if err != nil {
		t.Fatalf("ParseCreateChannel() error = %v", err)
	}
	if createInput.Name != "Deploy Hook" || createInput.Type != ChannelTypeWebhook || !createInput.IsEnabled {
		t.Fatalf("ParseCreateChannel() = %+v", createInput)
	}
	if createInput.Config["secret"] != "super-secret" {
		t.Fatalf("ParseCreateChannel() normalized config = %+v", createInput.Config)
	}
	disabledCreate, err := ParseCreateChannel(uuid.New(), ChannelInput{
		Name:      " Slack ",
		Type:      "slack",
		Config:    map[string]any{"webhook_url": "https://hooks.slack.com/services/T/B/C"},
		IsEnabled: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("ParseCreateChannel(disabled) error = %v", err)
	}
	if disabledCreate.IsEnabled {
		t.Fatalf("ParseCreateChannel(disabled) = %+v", disabledCreate)
	}

	name := " Ops "
	channelType := " telegram "
	rawConfig := map[string]any{"bot_token": "abc", "chat_id": "42"}
	disabled := false
	updateInput, err := ParseUpdateChannel(uuid.New(), ChannelPatchInput{
		Name:      &name,
		Type:      &channelType,
		Config:    &rawConfig,
		IsEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("ParseUpdateChannel() error = %v", err)
	}
	if !updateInput.Name.Set || updateInput.Name.Value != "Ops" || !updateInput.Type.Set || updateInput.Type.Value != ChannelTypeTelegram || !updateInput.Config.Set || !updateInput.IsEnabled.Set || updateInput.IsEnabled.Value {
		t.Fatalf("ParseUpdateChannel() = %+v", updateInput)
	}
	if _, err := ParseUpdateChannel(uuid.New(), ChannelPatchInput{}); err == nil {
		t.Fatal("ParseUpdateChannel() expected patch validation error")
	}
	blankName := " "
	if _, err := ParseUpdateChannel(uuid.New(), ChannelPatchInput{Name: &blankName}); err == nil {
		t.Fatal("ParseUpdateChannel() expected blank name validation error")
	}
	badType := "unknown"
	if _, err := ParseUpdateChannel(uuid.New(), ChannelPatchInput{Type: &badType}); err == nil {
		t.Fatal("ParseUpdateChannel() expected type validation error")
	}
	badConfig := map[string]any{"bad": make(chan int)}
	if _, err := ParseUpdateChannel(uuid.New(), ChannelPatchInput{Config: &badConfig}); err == nil {
		t.Fatal("ParseUpdateChannel() expected config clone validation error")
	}

	channelConfigs := []struct {
		channelType ChannelType
		raw         map[string]any
	}{
		{ChannelTypeWebhook, webhookConfig},
		{ChannelTypeTelegram, map[string]any{"bot_token": "abc", "chat_id": "42"}},
		{ChannelTypeSlack, map[string]any{"webhook_url": "https://hooks.slack.com/services/T/B/C"}},
		{ChannelTypeWeCom, map[string]any{"webhook_key": "key"}},
	}
	for _, testCase := range channelConfigs {
		parsed, err := ParseChannelConfig(testCase.channelType, testCase.raw)
		if err != nil {
			t.Fatalf("ParseChannelConfig(%q) error = %v", testCase.channelType, err)
		}
		normalized, err := NormalizeChannelConfig(testCase.channelType, testCase.raw)
		if err != nil {
			t.Fatalf("NormalizeChannelConfig(%q) error = %v", testCase.channelType, err)
		}
		if parsed.channelType() != testCase.channelType || len(normalized) == 0 {
			t.Fatalf("channel config mismatch for %q", testCase.channelType)
		}
	}
	if _, err := ParseChannelConfig(ChannelType("bad"), map[string]any{}); err == nil {
		t.Fatal("ParseChannelConfig() expected unsupported type error")
	}
	if _, err := NormalizeChannelConfig(ChannelType("bad"), map[string]any{}); err == nil {
		t.Fatal("NormalizeChannelConfig() expected unsupported type error")
	}
	if _, err := NormalizeChannelConfig(ChannelTypeWebhook, map[string]any{"url": "bad"}); err == nil {
		t.Fatal("NormalizeChannelConfig() expected parse error from invalid config")
	}
	if _, err := NormalizeChannelConfig(ChannelTypeTelegram, map[string]any{"chat_id": "42"}); err == nil {
		t.Fatal("NormalizeChannelConfig(telegram) expected parse error")
	}
	if _, err := NormalizeChannelConfig(ChannelTypeSlack, map[string]any{}); err == nil {
		t.Fatal("NormalizeChannelConfig(slack) expected parse error")
	}
	if _, err := NormalizeChannelConfig(ChannelTypeWeCom, map[string]any{}); err == nil {
		t.Fatal("NormalizeChannelConfig(wecom) expected parse error")
	}
	if _, err := ParseChannelConfig(ChannelTypeWebhook, map[string]any{"url": "https://example.com", "headers": []any{}}); err == nil {
		t.Fatal("ParseChannelConfig(webhook) expected headers validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeWebhook, map[string]any{}); err == nil {
		t.Fatal("ParseChannelConfig(webhook) expected url validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeWebhook, map[string]any{"url": "https://example.com", "secret": true}); err == nil {
		t.Fatal("ParseChannelConfig(webhook) expected secret validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeTelegram, map[string]any{"chat_id": "42"}); err == nil {
		t.Fatal("ParseChannelConfig(telegram) expected bot_token validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeTelegram, map[string]any{"bot_token": "abc"}); err == nil {
		t.Fatal("ParseChannelConfig(telegram) expected chat_id validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeSlack, map[string]any{}); err == nil {
		t.Fatal("ParseChannelConfig(slack) expected webhook_url validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeSlack, map[string]any{"webhook_url": "ftp://example.com"}); err == nil {
		t.Fatal("ParseChannelConfig(slack) expected webhook_url validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeWeCom, map[string]any{}); err == nil {
		t.Fatal("ParseChannelConfig(wecom) expected webhook_key validation error")
	}
	if _, err := ParseChannelConfig(ChannelTypeWeCom, map[string]any{"webhook_key": ""}); err == nil {
		t.Fatal("ParseChannelConfig(wecom) expected webhook_key validation error")
	}
	if _, err := ParseCreateChannel(uuid.New(), ChannelInput{Name: " ", Type: "webhook", Config: map[string]any{}}); err == nil {
		t.Fatal("ParseCreateChannel() expected blank name validation error")
	}
	if _, err := ParseCreateChannel(uuid.New(), ChannelInput{Name: "bad", Type: "unknown", Config: map[string]any{}}); err == nil {
		t.Fatal("ParseCreateChannel() expected type validation error")
	}
	if _, err := ParseCreateChannel(uuid.New(), ChannelInput{Name: "bad", Type: "webhook", Config: map[string]any{"url": "bad"}}); err == nil {
		t.Fatal("ParseCreateChannel() expected config validation error")
	}

	redacted := RedactedConfig(ChannelTypeWebhook, map[string]any{
		"url":     "https://hooks.example.com/path",
		"secret":  "abcd1234",
		"headers": map[string]any{"X-Test": "ok"},
	})
	if redacted["url"] != "https://hooks.example.com/***" || redacted["secret"] != "****1234" {
		t.Fatalf("RedactedConfig(webhook) = %+v", redacted)
	}
	if got := RedactedConfig(ChannelTypeTelegram, map[string]any{"bot_token": "1234"}); got["bot_token"] != "****" {
		t.Fatalf("RedactedConfig(telegram) = %+v", got)
	}
	if got := RedactedConfig(ChannelTypeSlack, map[string]any{"webhook_url": "bad"}); got["webhook_url"] != "redacted" {
		t.Fatalf("RedactedConfig(slack) = %+v", got)
	}
	if got := RedactedConfig(ChannelTypeWeCom, map[string]any{"webhook_key": "secret"}); got["webhook_key"] != "**cret" {
		t.Fatalf("RedactedConfig(wecom) = %+v", got)
	}
	if got := RedactedConfig(ChannelTypeWebhook, map[string]any{"bad": make(chan int)}); len(got) != 0 {
		t.Fatalf("RedactedConfig(invalid) = %+v, want empty map", got)
	}

	if got := HMACPreview("secret"); got == "" || len(got) != 16 {
		t.Fatalf("HMACPreview() = %q, want 16 hex chars", got)
	}
	if _, err := cloneRawConfigValue("bad", make(chan int)); err == nil {
		t.Fatal("cloneRawConfigValue() expected validation error")
	}
	if cloned, err := cloneRawConfigValue("nil", nil); err != nil || cloned != nil {
		t.Fatalf("cloneRawConfigValue(nil) = %v, %v; want nil, nil", cloned, err)
	}
	if cloned, err := cloneRawConfigValue("bool", true); err != nil || cloned != true {
		t.Fatalf("cloneRawConfigValue(bool) = %v, %v; want true, nil", cloned, err)
	}
	if cloned, err := cloneRawConfigValue("float", 1.5); err != nil || cloned != 1.5 {
		t.Fatalf("cloneRawConfigValue(float) = %v, %v; want 1.5, nil", cloned, err)
	}
	if cloned, err := cloneRawConfigValue("int", 7); err != nil || cloned != 7 {
		t.Fatalf("cloneRawConfigValue(int) = %v, %v; want 7, nil", cloned, err)
	}
	if cloned, err := cloneRawConfigValue("string", "value"); err != nil || cloned != "value" {
		t.Fatalf("cloneRawConfigValue(string) = %v, %v; want value, nil", cloned, err)
	}
	if cloned, err := cloneRawConfigValue("map", map[string]any{"nested": "value"}); err != nil {
		t.Fatalf("cloneRawConfigValue(map) error = %v", err)
	} else if cloned.(map[string]any)["nested"] != "value" {
		t.Fatalf("cloneRawConfigValue(map) = %+v", cloned)
	}
	if cloned, err := cloneRawConfigValue("slice", []any{"value"}); err != nil {
		t.Fatalf("cloneRawConfigValue(slice) error = %v", err)
	} else if cloned.([]any)[0] != "value" {
		t.Fatalf("cloneRawConfigValue(slice) = %+v", cloned)
	}
	if _, err := cloneRawConfigValue("slice", []any{make(chan int)}); err == nil {
		t.Fatal("cloneRawConfigValue(slice) expected nested validation error")
	}
	if _, err := requireString(map[string]any{}, "token"); err == nil {
		t.Fatal("requireString() expected missing field validation error")
	}
	if _, err := requireString(map[string]any{"token": 1}, "token"); err == nil {
		t.Fatal("requireString() expected type validation error")
	}
	if _, err := requireString(map[string]any{"token": " "}, "token"); err == nil {
		t.Fatal("requireString() expected blank value validation error")
	}
	if _, err := optionalString(map[string]any{"token": 1}, "token"); err == nil {
		t.Fatal("optionalString() expected type validation error")
	}
	if got, err := optionalString(map[string]any{}, "token"); err != nil || got != "" {
		t.Fatalf("optionalString(missing) = %q, %v; want empty, nil", got, err)
	}
	if _, err := requireHTTPURL(map[string]any{"url": "ftp://example.com"}, "url"); err == nil {
		t.Fatal("requireHTTPURL() expected scheme validation error")
	}
	if _, err := requireHTTPURL(map[string]any{}, "url"); err == nil {
		t.Fatal("requireHTTPURL() expected required field validation error")
	}
	if _, err := requireHTTPURL(map[string]any{"url": "http://[::1"}, "url"); err == nil {
		t.Fatal("requireHTTPURL() expected parse validation error")
	}
	if _, err := requireHTTPURL(map[string]any{"url": "https:///path"}, "url"); err == nil {
		t.Fatal("requireHTTPURL() expected host validation error")
	}
	if got, err := optionalHeadersMap(map[string]any{}); err != nil || got != nil {
		t.Fatalf("optionalHeadersMap(missing) = %+v, %v; want nil, nil", got, err)
	}
	if _, err := optionalHeadersMap(map[string]any{"headers": []any{}}); err == nil {
		t.Fatal("optionalHeadersMap() expected object validation error")
	}
	if _, err := optionalHeadersMap(map[string]any{"headers": map[string]any{"X-Test": 1}}); err == nil {
		t.Fatal("optionalHeadersMap() expected string value validation error")
	}
	if got := copyStringMap(nil); got != nil {
		t.Fatalf("copyStringMap(nil) = %+v, want nil", got)
	}
	if got := redactSecret(" "); got != "" {
		t.Fatalf("redactSecret(blank) = %q, want empty", got)
	}
	if got := redactURL(" "); got != "redacted" {
		t.Fatalf("redactURL(blank) = %q, want redacted", got)
	}
	if got := redactURL("http://[::1"); got != "redacted" {
		t.Fatalf("redactURL(invalid) = %q, want redacted", got)
	}
	if got, err := ParseChannelType(" slack "); err != nil || got.String() != "slack" {
		t.Fatalf("ParseChannelType() = %q, %v; want slack, nil", got, err)
	}
	if _, err := ParseChannelType("unknown"); !errors.Is(err, ErrChannelTypeUnsupported) {
		t.Fatalf("ParseChannelType() error = %v, want ErrChannelTypeUnsupported", err)
	}
	if got := Some("value"); !got.Set || got.Value != "value" {
		t.Fatalf("Some() = %+v", got)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
