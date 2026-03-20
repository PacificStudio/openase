package notification

import "testing"

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
