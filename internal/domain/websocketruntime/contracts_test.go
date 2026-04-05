package websocketruntime

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestMessageTypeIsValid(t *testing.T) {
	valid := []MessageType{
		MessageTypeHello,
		MessageTypeHelloAck,
		MessageTypeRequest,
		MessageTypeResponse,
		MessageTypeEvent,
	}
	for _, item := range valid {
		if !item.IsValid() {
			t.Fatalf("MessageType(%q).IsValid() = false, want true", item)
		}
	}
	if MessageType("bad").IsValid() {
		t.Fatal("MessageType(bad).IsValid() = true, want false")
	}
}

func TestOperationIsValid(t *testing.T) {
	valid := []Operation{
		OperationProbe,
		OperationPreflight,
		OperationWorkspacePrepare,
		OperationWorkspaceReset,
		OperationArtifactSync,
		OperationCommandOpen,
		OperationSessionInput,
		OperationSessionSignal,
		OperationSessionClose,
		OperationProcessStart,
		OperationProcessStatus,
		OperationSessionOutput,
		OperationSessionExit,
	}
	for _, item := range valid {
		if !item.IsValid() {
			t.Fatalf("Operation(%q).IsValid() = false, want true", item)
		}
	}
	if Operation("bad").IsValid() {
		t.Fatal("Operation(bad).IsValid() = true, want false")
	}
}

func TestErrorClassIsValid(t *testing.T) {
	valid := []ErrorClass{
		ErrorClassAuth,
		ErrorClassMisconfiguration,
		ErrorClassTransient,
		ErrorClassUnsupported,
		ErrorClassInternal,
	}
	for _, item := range valid {
		if !item.IsValid() {
			t.Fatalf("ErrorClass(%q).IsValid() = false, want true", item)
		}
	}
	if ErrorClass("bad").IsValid() {
		t.Fatal("ErrorClass(bad).IsValid() = true, want false")
	}
}

func TestParseEnvelope(t *testing.T) {
	t.Run("parses and trims", func(t *testing.T) {
		raw := []byte(`{
			"version": 1,
			"type": "request",
			"request_id": " req-1 ",
			"operation": "probe",
			"payload": {"checked_at":"now"},
			"error": {"class":"transient","message":"  retry later  "}
		}`)

		envelope, err := ParseEnvelope(raw)
		if err != nil {
			t.Fatalf("ParseEnvelope() error = %v", err)
		}
		if envelope.RequestID != "req-1" {
			t.Fatalf("ParseEnvelope().RequestID = %q, want req-1", envelope.RequestID)
		}
		if envelope.Error == nil || envelope.Error.Message != "retry later" {
			t.Fatalf("ParseEnvelope().Error = %+v, want trimmed message", envelope.Error)
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		if _, err := ParseEnvelope([]byte("{")); err == nil {
			t.Fatal("ParseEnvelope(invalid json) error = nil, want error")
		}
	})

	t.Run("rejects invalid version", func(t *testing.T) {
		_, err := ParseEnvelope([]byte(`{"version":2,"type":"request"}`))
		if !errors.Is(err, ErrUnsupportedVersion) {
			t.Fatalf("ParseEnvelope(invalid version) error = %v, want ErrUnsupportedVersion", err)
		}
	})

	t.Run("rejects invalid message type", func(t *testing.T) {
		_, err := ParseEnvelope([]byte(`{"version":1,"type":"bad"}`))
		if !errors.Is(err, ErrUnexpectedMessage) {
			t.Fatalf("ParseEnvelope(invalid type) error = %v, want ErrUnexpectedMessage", err)
		}
	})

	t.Run("rejects invalid operation", func(t *testing.T) {
		_, err := ParseEnvelope([]byte(`{"version":1,"type":"request","operation":"bad"}`))
		if !errors.Is(err, ErrUnexpectedMessage) {
			t.Fatalf("ParseEnvelope(invalid operation) error = %v, want ErrUnexpectedMessage", err)
		}
	})

	t.Run("rejects invalid error class", func(t *testing.T) {
		_, err := ParseEnvelope([]byte(`{
			"version":1,
			"type":"response",
			"error":{"class":"bad","message":"x"}
		}`))
		if !errors.Is(err, ErrUnexpectedMessage) {
			t.Fatalf("ParseEnvelope(invalid error class) error = %v, want ErrUnexpectedMessage", err)
		}
	})
}

func TestDecodePayload(t *testing.T) {
	t.Run("returns zero for empty payload", func(t *testing.T) {
		payload, err := DecodePayload[Hello](Envelope{})
		if err != nil {
			t.Fatalf("DecodePayload(empty) error = %v", err)
		}
		if len(payload.SupportedVersions) != 0 || len(payload.Capabilities) != 0 {
			t.Fatalf("DecodePayload(empty) = %+v, want zero value", payload)
		}
	})

	t.Run("decodes payload", func(t *testing.T) {
		raw, err := json.Marshal(Hello{SupportedVersions: []int{ProtocolVersion}})
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		payload, err := DecodePayload[Hello](Envelope{
			Operation: OperationProbe,
			Payload:   raw,
		})
		if err != nil {
			t.Fatalf("DecodePayload(valid) error = %v", err)
		}
		if len(payload.SupportedVersions) != 1 || payload.SupportedVersions[0] != ProtocolVersion {
			t.Fatalf("DecodePayload(valid) = %+v", payload)
		}
	})

	t.Run("rejects invalid payload", func(t *testing.T) {
		_, err := DecodePayload[Hello](Envelope{
			Operation: OperationProbe,
			Payload:   json.RawMessage(`{`),
		})
		if err == nil {
			t.Fatal("DecodePayload(invalid) error = nil, want error")
		}
	})
}
