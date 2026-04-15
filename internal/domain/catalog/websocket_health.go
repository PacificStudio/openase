package catalog

import (
	"fmt"
	"strings"
	"time"
)

type WebsocketHealthState string

const (
	WebsocketHealthStateHealthy  WebsocketHealthState = "healthy"
	WebsocketHealthStateDegraded WebsocketHealthState = "degraded"
	WebsocketHealthStateFailed   WebsocketHealthState = "failed"
	WebsocketHealthStateUnknown  WebsocketHealthState = "unknown"
)

func (s WebsocketHealthState) String() string {
	return string(s)
}

func (s WebsocketHealthState) IsValid() bool {
	switch s {
	case WebsocketHealthStateHealthy,
		WebsocketHealthStateDegraded,
		WebsocketHealthStateFailed,
		WebsocketHealthStateUnknown:
		return true
	default:
		return false
	}
}

type WebsocketHealthLayer struct {
	State      WebsocketHealthState
	Reason     string
	ObservedAt time.Time
	Details    map[string]any
}

type WebsocketMachineHealth struct {
	TransportMode MachineConnectionMode
	CheckedAt     time.Time
	L2            WebsocketHealthLayer
	L3            WebsocketHealthLayer
	L4            WebsocketHealthLayer
	L5            WebsocketHealthLayer
}

func ParseStoredWebsocketHealthState(raw string) (WebsocketHealthState, error) {
	state := WebsocketHealthState(strings.TrimSpace(raw))
	if !state.IsValid() {
		return WebsocketHealthStateUnknown, fmt.Errorf("invalid websocket health state %q", raw)
	}
	return state, nil
}

func ParseStoredWebsocketMachineHealth(resources map[string]any) (WebsocketMachineHealth, error) {
	raw, ok := websocketHealthObject(resources)
	if !ok {
		return WebsocketMachineHealth{}, fmt.Errorf("missing websocket_health")
	}
	return parseStoredWebsocketMachineHealthObject(raw)
}

func StoreWebsocketMachineHealth(health WebsocketMachineHealth) map[string]any {
	return map[string]any{
		"transport_mode": strings.TrimSpace(health.TransportMode.String()),
		"checked_at":     health.CheckedAt.UTC().Format(time.RFC3339),
		"l2":             storeWebsocketHealthLayer(health.L2),
		"l3":             storeWebsocketHealthLayer(health.L3),
		"l4":             storeWebsocketHealthLayer(health.L4),
		"l5":             storeWebsocketHealthLayer(health.L5),
	}
}

func WebsocketHealthUnknownLayer(observedAt time.Time, reason string, details map[string]any) WebsocketHealthLayer {
	return WebsocketHealthLayer{
		State:      WebsocketHealthStateUnknown,
		Reason:     strings.TrimSpace(reason),
		ObservedAt: observedAt.UTC(),
		Details:    cloneWebsocketHealthMap(details),
	}
}

func WebsocketHealthLayerAffectsMachineStatus(layer WebsocketHealthLayer) bool {
	return layer.State == WebsocketHealthStateFailed || layer.State == WebsocketHealthStateDegraded
}

func parseStoredWebsocketMachineHealthObject(raw map[string]any) (WebsocketMachineHealth, error) {
	transportModeRaw, ok := websocketHealthStringField(raw, "transport_mode")
	if !ok {
		return WebsocketMachineHealth{}, fmt.Errorf("missing websocket_health.transport_mode")
	}
	transportMode := MachineConnectionMode(strings.TrimSpace(transportModeRaw))
	if !transportMode.IsValid() {
		return WebsocketMachineHealth{}, fmt.Errorf("invalid websocket_health.transport_mode %q", transportModeRaw)
	}
	if transportMode != MachineConnectionModeWSReverse && transportMode != MachineConnectionModeWSListener {
		return WebsocketMachineHealth{}, fmt.Errorf("websocket_health.transport_mode %q is not a websocket mode", transportModeRaw)
	}

	checkedAtRaw, ok := websocketHealthStringField(raw, "checked_at")
	if !ok {
		return WebsocketMachineHealth{}, fmt.Errorf("missing websocket_health.checked_at")
	}
	checkedAt, err := time.Parse(time.RFC3339, checkedAtRaw)
	if err != nil {
		return WebsocketMachineHealth{}, fmt.Errorf("parse websocket_health.checked_at: %w", err)
	}

	l2, err := parseStoredWebsocketHealthLayer(raw, "l2")
	if err != nil {
		return WebsocketMachineHealth{}, err
	}
	l3, err := parseStoredWebsocketHealthLayer(raw, "l3")
	if err != nil {
		return WebsocketMachineHealth{}, err
	}
	l4, err := parseStoredWebsocketHealthLayer(raw, "l4")
	if err != nil {
		return WebsocketMachineHealth{}, err
	}
	l5, err := parseStoredWebsocketHealthLayer(raw, "l5")
	if err != nil {
		return WebsocketMachineHealth{}, err
	}

	return WebsocketMachineHealth{
		TransportMode: transportMode,
		CheckedAt:     checkedAt.UTC(),
		L2:            l2,
		L3:            l3,
		L4:            l4,
		L5:            l5,
	}, nil
}

func parseStoredWebsocketHealthLayer(raw map[string]any, key string) (WebsocketHealthLayer, error) {
	layerRaw, ok := websocketHealthNestedObject(raw, key)
	if !ok {
		return WebsocketHealthLayer{}, fmt.Errorf("missing websocket_health.%s", key)
	}
	stateRaw, ok := websocketHealthStringField(layerRaw, "state")
	if !ok {
		return WebsocketHealthLayer{}, fmt.Errorf("missing websocket_health.%s.state", key)
	}
	state, err := ParseStoredWebsocketHealthState(stateRaw)
	if err != nil {
		return WebsocketHealthLayer{}, fmt.Errorf("parse websocket_health.%s.state: %w", key, err)
	}
	observedAtRaw, ok := websocketHealthStringField(layerRaw, "observed_at")
	if !ok {
		return WebsocketHealthLayer{}, fmt.Errorf("missing websocket_health.%s.observed_at", key)
	}
	observedAt, err := time.Parse(time.RFC3339, observedAtRaw)
	if err != nil {
		return WebsocketHealthLayer{}, fmt.Errorf("parse websocket_health.%s.observed_at: %w", key, err)
	}

	details, _ := websocketHealthNestedObject(layerRaw, "details")
	return WebsocketHealthLayer{
		State:      state,
		Reason:     websocketHealthStringFieldOrEmpty(layerRaw, "reason"),
		ObservedAt: observedAt.UTC(),
		Details:    cloneWebsocketHealthMap(details),
	}, nil
}

func storeWebsocketHealthLayer(layer WebsocketHealthLayer) map[string]any {
	return map[string]any{
		"state":       layer.State.String(),
		"reason":      strings.TrimSpace(layer.Reason),
		"observed_at": layer.ObservedAt.UTC().Format(time.RFC3339),
		"details":     cloneWebsocketHealthMap(layer.Details),
	}
}

func websocketHealthObject(resources map[string]any) (map[string]any, bool) {
	return websocketHealthNestedObject(resources, "websocket_health")
}

func websocketHealthNestedObject(raw map[string]any, key string) (map[string]any, bool) {
	value, ok := raw[key]
	if !ok {
		return nil, false
	}
	object, ok := value.(map[string]any)
	return object, ok
}

func websocketHealthStringField(raw map[string]any, key string) (string, bool) {
	value, ok := raw[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func websocketHealthStringFieldOrEmpty(raw map[string]any, key string) string {
	value, _ := websocketHealthStringField(raw, key)
	return value
}

func cloneWebsocketHealthMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = cloneWebsocketHealthValue(value)
	}
	return cloned
}

func cloneWebsocketHealthValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneWebsocketHealthMap(typed)
	case []any:
		cloned := make([]any, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloneWebsocketHealthValue(item))
		}
		return cloned
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}
