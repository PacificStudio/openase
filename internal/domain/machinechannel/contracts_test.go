package machinechannel

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseDaemonConfig(t *testing.T) {
	machineID := uuid.New()
	config, err := ParseDaemonConfig(
		machineID.String(),
		TokenPrefix+"abc123",
		" http://127.0.0.1:19836/ ",
		15*time.Second,
		5*time.Second,
		" /usr/local/bin/openase ",
		" /usr/local/bin/codex ",
	)
	if err != nil {
		t.Fatalf("ParseDaemonConfig returned error: %v", err)
	}
	if config.MachineID != machineID || config.Token != TokenPrefix+"abc123" {
		t.Fatalf("unexpected parsed config: %+v", config)
	}
	if config.ControlPlaneURL != "http://127.0.0.1:19836" {
		t.Fatalf("expected trimmed control plane url, got %+v", config)
	}
	if config.OpenASEBinaryPath != "/usr/local/bin/openase" || config.AgentCLIPath != "/usr/local/bin/codex" {
		t.Fatalf("expected trimmed binary paths, got %+v", config)
	}

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "invalid machine id",
			fn: func() error {
				_, err := ParseDaemonConfig("bad", TokenPrefix+"abc123", "http://127.0.0.1:19836", time.Second, time.Second, "", "")
				return err
			},
		},
		{
			name: "invalid token",
			fn: func() error {
				_, err := ParseDaemonConfig(machineID.String(), "bad", "http://127.0.0.1:19836", time.Second, time.Second, "", "")
				return err
			},
		},
		{
			name: "empty url",
			fn: func() error {
				_, err := ParseDaemonConfig(machineID.String(), TokenPrefix+"abc123", " ", time.Second, time.Second, "", "")
				return err
			},
		},
		{
			name: "nonpositive heartbeat",
			fn: func() error {
				_, err := ParseDaemonConfig(machineID.String(), TokenPrefix+"abc123", "http://127.0.0.1:19836", 0, time.Second, "", "")
				return err
			},
		},
		{
			name: "nonpositive reconnect backoff",
			fn: func() error {
				_, err := ParseDaemonConfig(machineID.String(), TokenPrefix+"abc123", "http://127.0.0.1:19836", time.Second, 0, "", "")
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Fatalf("%s expected error", tt.name)
			}
		})
	}
}

func TestParseTokenAndUUID(t *testing.T) {
	token, err := ParseToken(" " + TokenPrefix + "xyz ")
	if err != nil || token != TokenPrefix+"xyz" {
		t.Fatalf("ParseToken returned %q, %v", token, err)
	}
	if _, err := ParseToken(" "); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token error, got %v", err)
	}
	if _, err := ParseToken("ase_agent_123"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token prefix error, got %v", err)
	}

	validUUID := uuid.New()
	parsedUUID, err := parseUUID(validUUID.String())
	if err != nil || parsedUUID != validUUID {
		t.Fatalf("parseUUID returned %+v, %v", parsedUUID, err)
	}
	if _, err := parseUUID(" "); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("expected empty uuid error, got %v", err)
	}
	if _, err := parseUUID("bad"); err == nil || !strings.Contains(err.Error(), "must be a valid UUID") {
		t.Fatalf("expected invalid uuid error, got %v", err)
	}
}

func TestParseEnvelopeAndDecodePayload(t *testing.T) {
	heartbeatPayload, err := json.Marshal(Heartbeat{SentAt: "2026-04-04T15:00:00Z"})
	if err != nil {
		t.Fatalf("marshal heartbeat payload: %v", err)
	}
	validEnvelope, err := json.Marshal(Envelope{
		Version:   ProtocolVersion,
		Type:      MessageTypeHeartbeat,
		SessionID: "session-1",
		Payload:   heartbeatPayload,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	envelope, err := ParseEnvelope(validEnvelope)
	if err != nil {
		t.Fatalf("ParseEnvelope returned error: %v", err)
	}
	if envelope.Type != MessageTypeHeartbeat || envelope.SessionID != "session-1" {
		t.Fatalf("unexpected parsed envelope: %+v", envelope)
	}

	decodedHeartbeat, err := DecodePayload[Heartbeat](envelope)
	if err != nil {
		t.Fatalf("DecodePayload returned error: %v", err)
	}
	if decodedHeartbeat.SentAt != "2026-04-04T15:00:00Z" {
		t.Fatalf("unexpected decoded heartbeat: %+v", decodedHeartbeat)
	}

	emptyPayload, err := DecodePayload[Goodbye](Envelope{Type: MessageTypeGoodbye})
	if err != nil {
		t.Fatalf("DecodePayload(empty) returned error: %v", err)
	}
	if emptyPayload.Reason != "" {
		t.Fatalf("expected zero-value goodbye payload, got %+v", emptyPayload)
	}

	if _, err := ParseEnvelope([]byte("not-json")); err == nil {
		t.Fatal("ParseEnvelope expected json parsing error")
	}
	unsupportedVersion, _ := json.Marshal(Envelope{Version: 2, Type: MessageTypeHello})
	if _, err := ParseEnvelope(unsupportedVersion); err == nil || !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected unsupported version error, got %v", err)
	}
	unsupportedType := []byte(`{"version":1,"type":"mystery"}`)
	if _, err := ParseEnvelope(unsupportedType); err == nil || !errors.Is(err, ErrUnexpectedMessage) {
		t.Fatalf("expected unexpected message error, got %v", err)
	}
	if _, err := DecodePayload[Heartbeat](Envelope{
		Type:    MessageTypeHeartbeat,
		Payload: json.RawMessage(`{"sent_at":`),
	}); err == nil {
		t.Fatal("DecodePayload expected payload decoding error")
	}
}
