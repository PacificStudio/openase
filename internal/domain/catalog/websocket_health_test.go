package catalog

import (
	"testing"
	"time"
)

func TestParseStoredWebsocketMachineHealth(t *testing.T) {
	raw := map[string]any{
		"websocket_health": map[string]any{
			"transport_mode": "ws_reverse",
			"checked_at":     "2026-04-15T10:00:00Z",
			"l2": map[string]any{
				"state":       "healthy",
				"reason":      "",
				"observed_at": "2026-04-15T10:00:00Z",
				"details": map[string]any{
					"interfaces": []any{"eth0"},
				},
			},
			"l3": map[string]any{
				"state":       "failed",
				"reason":      "control plane route missing",
				"observed_at": "2026-04-15T10:00:01Z",
				"details": map[string]any{
					"target_host": "control-plane.internal",
				},
			},
			"l4": map[string]any{
				"state":       "healthy",
				"reason":      "",
				"observed_at": "2026-04-15T10:00:02Z",
				"details": map[string]any{
					"session_id": "session-1",
				},
			},
			"l5": map[string]any{
				"state":       "degraded",
				"reason":      "runtime providers are not dispatchable",
				"observed_at": "2026-04-15T10:00:03Z",
				"details": map[string]any{
					"agent_dispatchable": false,
				},
			},
		},
	}

	health, err := ParseStoredWebsocketMachineHealth(raw)
	if err != nil {
		t.Fatalf("ParseStoredWebsocketMachineHealth() error = %v", err)
	}
	if health.TransportMode != MachineConnectionModeWSReverse {
		t.Fatalf("TransportMode = %q", health.TransportMode)
	}
	if health.L3.State != WebsocketHealthStateFailed || health.L3.Reason != "control plane route missing" {
		t.Fatalf("L3 = %+v", health.L3)
	}
	if got := health.L4.Details["session_id"]; got != "session-1" {
		t.Fatalf("L4 session_id = %#v", got)
	}
	if health.L5.State != WebsocketHealthStateDegraded {
		t.Fatalf("L5 = %+v", health.L5)
	}
}

func TestStoreWebsocketMachineHealthRoundTrips(t *testing.T) {
	original := WebsocketMachineHealth{
		TransportMode: MachineConnectionModeWSListener,
		CheckedAt:     mustParseHealthTime(t, "2026-04-15T10:15:00Z"),
		L2: WebsocketHealthLayer{
			State:      WebsocketHealthStateHealthy,
			ObservedAt: mustParseHealthTime(t, "2026-04-15T10:15:00Z"),
			Details:    map[string]any{"interfaces": []string{"en0"}},
		},
		L3: WebsocketHealthLayer{
			State:      WebsocketHealthStateUnknown,
			Reason:     "listener mode cannot confirm reverse path",
			ObservedAt: mustParseHealthTime(t, "2026-04-15T10:15:01Z"),
		},
		L4: WebsocketHealthLayer{
			State:      WebsocketHealthStateHealthy,
			ObservedAt: mustParseHealthTime(t, "2026-04-15T10:15:02Z"),
			Details:    map[string]any{"session_state": "connected"},
		},
		L5: WebsocketHealthLayer{
			State:      WebsocketHealthStateHealthy,
			ObservedAt: mustParseHealthTime(t, "2026-04-15T10:15:03Z"),
			Details:    map[string]any{"runtime_probe": "ok"},
		},
	}

	parsed, err := ParseStoredWebsocketMachineHealth(map[string]any{
		"websocket_health": StoreWebsocketMachineHealth(original),
	})
	if err != nil {
		t.Fatalf("ParseStoredWebsocketMachineHealth(Store(...)) error = %v", err)
	}
	if parsed.TransportMode != original.TransportMode || parsed.L3.Reason != original.L3.Reason {
		t.Fatalf("round trip mismatch: %+v", parsed)
	}
	if parsed.L2.State != WebsocketHealthStateHealthy || parsed.L5.Details["runtime_probe"] != "ok" {
		t.Fatalf("round trip details mismatch: %+v", parsed)
	}
}

func TestParseStoredWebsocketMachineHealthRejectsInvalidState(t *testing.T) {
	_, err := ParseStoredWebsocketMachineHealth(map[string]any{
		"websocket_health": map[string]any{
			"transport_mode": "ws_reverse",
			"checked_at":     "2026-04-15T10:00:00Z",
			"l2": map[string]any{
				"state":       "bogus",
				"observed_at": "2026-04-15T10:00:00Z",
				"details":     map[string]any{},
			},
			"l3": map[string]any{
				"state":       "healthy",
				"observed_at": "2026-04-15T10:00:00Z",
				"details":     map[string]any{},
			},
			"l4": map[string]any{
				"state":       "healthy",
				"observed_at": "2026-04-15T10:00:00Z",
				"details":     map[string]any{},
			},
			"l5": map[string]any{
				"state":       "healthy",
				"observed_at": "2026-04-15T10:00:00Z",
				"details":     map[string]any{},
			},
		},
	})
	if err == nil || err.Error() != `parse websocket_health.l2.state: invalid websocket health state "bogus"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebsocketHealthLayerAffectsMachineStatus(t *testing.T) {
	if WebsocketHealthLayerAffectsMachineStatus(WebsocketHealthLayer{State: WebsocketHealthStateHealthy}) {
		t.Fatal("healthy layer should not affect machine status")
	}
	if WebsocketHealthLayerAffectsMachineStatus(WebsocketHealthLayer{State: WebsocketHealthStateUnknown}) {
		t.Fatal("unknown layer should not affect machine status by default")
	}
	if !WebsocketHealthLayerAffectsMachineStatus(WebsocketHealthLayer{State: WebsocketHealthStateFailed}) {
		t.Fatal("failed layer must affect machine status")
	}
	if !WebsocketHealthLayerAffectsMachineStatus(WebsocketHealthLayer{State: WebsocketHealthStateDegraded}) {
		t.Fatal("degraded layer must affect machine status")
	}
}

func TestParseStoredWebsocketMachineHealthErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(raw map[string]any)
		wantErr string
	}{
		{
			name: "missing websocket health",
			mutate: func(raw map[string]any) {
				delete(raw, "websocket_health")
			},
			wantErr: "missing websocket_health",
		},
		{
			name: "missing transport mode",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any), "transport_mode")
			},
			wantErr: "missing websocket_health.transport_mode",
		},
		{
			name: "invalid transport mode",
			mutate: func(raw map[string]any) {
				raw["websocket_health"].(map[string]any)["transport_mode"] = "bogus"
			},
			wantErr: `invalid websocket_health.transport_mode "bogus"`,
		},
		{
			name: "non websocket transport mode",
			mutate: func(raw map[string]any) {
				raw["websocket_health"].(map[string]any)["transport_mode"] = MachineConnectionModeSSH.String()
			},
			wantErr: `websocket_health.transport_mode "ssh" is not a websocket mode`,
		},
		{
			name: "missing checked at",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any), "checked_at")
			},
			wantErr: "missing websocket_health.checked_at",
		},
		{
			name: "invalid checked at",
			mutate: func(raw map[string]any) {
				raw["websocket_health"].(map[string]any)["checked_at"] = "not-a-time"
			},
			wantErr: `parse websocket_health.checked_at: parsing time "not-a-time" as "2006-01-02T15:04:05Z07:00": cannot parse "not-a-time" as "2006"`,
		},
		{
			name: "missing layer",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any), "l2")
			},
			wantErr: "missing websocket_health.l2",
		},
		{
			name: "missing l3 layer",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any), "l3")
			},
			wantErr: "missing websocket_health.l3",
		},
		{
			name: "missing l4 layer",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any), "l4")
			},
			wantErr: "missing websocket_health.l4",
		},
		{
			name: "missing l5 layer",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any), "l5")
			},
			wantErr: "missing websocket_health.l5",
		},
		{
			name: "missing layer state",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any)["l2"].(map[string]any), "state")
			},
			wantErr: "missing websocket_health.l2.state",
		},
		{
			name: "missing observed at",
			mutate: func(raw map[string]any) {
				delete(raw["websocket_health"].(map[string]any)["l2"].(map[string]any), "observed_at")
			},
			wantErr: "missing websocket_health.l2.observed_at",
		},
		{
			name: "invalid observed at",
			mutate: func(raw map[string]any) {
				raw["websocket_health"].(map[string]any)["l2"].(map[string]any)["observed_at"] = "not-a-time"
			},
			wantErr: `parse websocket_health.l2.observed_at: parsing time "not-a-time" as "2006-01-02T15:04:05Z07:00": cannot parse "not-a-time" as "2006"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := validStoredWebsocketMachineHealth()
			tt.mutate(raw)

			_, err := ParseStoredWebsocketMachineHealth(raw)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("ParseStoredWebsocketMachineHealth() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestWebsocketHealthUnknownLayer(t *testing.T) {
	observedAt := mustParseHealthTime(t, "2026-04-15T11:00:00Z")
	details := map[string]any{
		"nested": map[string]any{"value": "ok"},
		"items":  []any{"one", map[string]any{"two": true}},
	}

	layer := WebsocketHealthUnknownLayer(observedAt, "  waiting for probe  ", details)
	if layer.State != WebsocketHealthStateUnknown {
		t.Fatalf("State = %q", layer.State)
	}
	if layer.Reason != "waiting for probe" {
		t.Fatalf("Reason = %q", layer.Reason)
	}
	if !layer.ObservedAt.Equal(observedAt.UTC()) {
		t.Fatalf("ObservedAt = %s", layer.ObservedAt)
	}

	details["nested"].(map[string]any)["value"] = "mutated"
	details["items"].([]any)[1].(map[string]any)["two"] = false
	if got := layer.Details["nested"].(map[string]any)["value"]; got != "ok" {
		t.Fatalf("nested value = %#v", got)
	}
	if got := layer.Details["items"].([]any)[1].(map[string]any)["two"]; got != true {
		t.Fatalf("items[1].two = %#v", got)
	}
}

func TestWebsocketHealthHelpers(t *testing.T) {
	t.Run("websocketHealthNestedObject", func(t *testing.T) {
		if _, ok := websocketHealthNestedObject(map[string]any{}, "missing"); ok {
			t.Fatal("missing key should not resolve")
		}
		if _, ok := websocketHealthNestedObject(map[string]any{"bad": "value"}, "bad"); ok {
			t.Fatal("non-object value should not resolve")
		}
		if nested, ok := websocketHealthNestedObject(map[string]any{"good": map[string]any{"x": 1}}, "good"); !ok || nested["x"] != 1 {
			t.Fatalf("nested object = %#v, %v", nested, ok)
		}
	})

	t.Run("websocketHealthStringField", func(t *testing.T) {
		if _, ok := websocketHealthStringField(map[string]any{}, "missing"); ok {
			t.Fatal("missing string field should not resolve")
		}
		if _, ok := websocketHealthStringField(map[string]any{"bad": 1}, "bad"); ok {
			t.Fatal("non-string field should not resolve")
		}
		if _, ok := websocketHealthStringField(map[string]any{"blank": "   "}, "blank"); ok {
			t.Fatal("blank string field should not resolve")
		}
		if got, ok := websocketHealthStringField(map[string]any{"good": " ok "}, "good"); !ok || got != "ok" {
			t.Fatalf("string field = %q, %v", got, ok)
		}
		if got := websocketHealthStringFieldOrEmpty(map[string]any{"blank": "   "}, "blank"); got != "" {
			t.Fatalf("blank fallback = %q", got)
		}
	})

	t.Run("cloneWebsocketHealthMap", func(t *testing.T) {
		if got := cloneWebsocketHealthMap(nil); len(got) != 0 {
			t.Fatalf("cloneWebsocketHealthMap(nil) = %#v", got)
		}
		original := map[string]any{
			"map":   map[string]any{"x": "y"},
			"list":  []any{map[string]any{"k": "v"}},
			"names": []string{"a", "b"},
			"flag":  true,
		}

		cloned := cloneWebsocketHealthMap(original)
		original["map"].(map[string]any)["x"] = "changed"
		original["list"].([]any)[0].(map[string]any)["k"] = "changed"
		original["names"].([]string)[0] = "changed"
		original["flag"] = false

		if got := cloned["map"].(map[string]any)["x"]; got != "y" {
			t.Fatalf("cloned map value = %#v", got)
		}
		if got := cloned["list"].([]any)[0].(map[string]any)["k"]; got != "v" {
			t.Fatalf("cloned list value = %#v", got)
		}
		if got := cloned["names"].([]string)[0]; got != "a" {
			t.Fatalf("cloned names value = %#v", got)
		}
		if got := cloned["flag"]; got != true {
			t.Fatalf("cloned flag = %#v", got)
		}
	})
}

func validStoredWebsocketMachineHealth() map[string]any {
	return map[string]any{
		"websocket_health": map[string]any{
			"transport_mode": "ws_reverse",
			"checked_at":     "2026-04-15T10:00:00Z",
			"l2": map[string]any{
				"state":       "healthy",
				"reason":      "",
				"observed_at": "2026-04-15T10:00:00Z",
				"details":     map[string]any{},
			},
			"l3": map[string]any{
				"state":       "healthy",
				"reason":      "",
				"observed_at": "2026-04-15T10:00:01Z",
				"details":     map[string]any{},
			},
			"l4": map[string]any{
				"state":       "healthy",
				"reason":      "",
				"observed_at": "2026-04-15T10:00:02Z",
				"details":     map[string]any{},
			},
			"l5": map[string]any{
				"state":       "healthy",
				"reason":      "",
				"observed_at": "2026-04-15T10:00:03Z",
				"details":     map[string]any{},
			},
		},
	}
}

func mustParseHealthTime(t *testing.T, raw string) time.Time {
	t.Helper()
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", raw, err)
	}
	return value.UTC()
}
