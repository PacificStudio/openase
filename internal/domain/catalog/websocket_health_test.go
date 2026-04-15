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

func mustParseHealthTime(t *testing.T, raw string) time.Time {
	t.Helper()
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", raw, err)
	}
	return value.UTC()
}
