package event

import (
	"context"
	"fmt"
	"math"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

func TestNewPGNotifyBusRequiresDSN(t *testing.T) {
	if _, err := NewPGNotifyBus("   ", nil); err == nil {
		t.Fatal("expected missing dsn error")
	}
}

func TestPGNotifyEncodeDecodeRoundTrip(t *testing.T) {
	message, err := provider.NewJSONEvent(
		provider.MustParseTopic("ticket.events"),
		provider.MustParseEventType("ticket.updated"),
		map[string]any{"ticket_id": "ASE-21", "status": "in_progress"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	encoded, err := encodePGNotifyEvent(message)
	if err != nil {
		t.Fatalf("encodePGNotifyEvent returned error: %v", err)
	}

	decoded, err := decodePGNotifyEvent(encoded)
	if err != nil {
		t.Fatalf("decodePGNotifyEvent returned error: %v", err)
	}

	if decoded.Topic != message.Topic {
		t.Fatalf("expected topic %q, got %q", message.Topic, decoded.Topic)
	}
	if decoded.Type != message.Type {
		t.Fatalf("expected type %q, got %q", message.Type, decoded.Type)
	}
	if string(decoded.Payload) != string(message.Payload) {
		t.Fatalf("expected payload %s, got %s", string(message.Payload), string(decoded.Payload))
	}
}

func TestPGNotifyEncodeRejectsOversizedPayload(t *testing.T) {
	message, err := provider.NewJSONEvent(
		provider.MustParseTopic("ticket.events"),
		provider.MustParseEventType("ticket.updated"),
		map[string]string{"blob": strings.Repeat("x", maxPGNotifyPayloadBytes)},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	if _, err := encodePGNotifyEvent(message); err == nil {
		t.Fatal("expected oversize payload error")
	}
}

func TestPGChannelNameFitsPostgresLimit(t *testing.T) {
	topic, err := provider.ParseTopic(strings.Repeat("a", 128))
	if err != nil {
		t.Fatalf("ParseTopic returned error: %v", err)
	}

	channel := pgChannelName(topic)
	if len(channel) != maxPGChannelNameBytes {
		t.Fatalf("expected channel name length %d, got %d (%q)", maxPGChannelNameBytes, len(channel), channel)
	}
	if !strings.HasPrefix(channel, pgChannelPrefix) {
		t.Fatalf("expected channel prefix %q, got %q", pgChannelPrefix, channel)
	}
	if channel != pgChannelName(topic) {
		t.Fatalf("expected deterministic channel mapping for %q", topic)
	}
}

func TestPGNotifyBusPublishesRuntimeEventsWithLengthSafeChannelNames(t *testing.T) {
	dsn := startEmbeddedPostgres(t)
	bus, err := NewPGNotifyBus(dsn, nil)
	if err != nil {
		t.Fatalf("NewPGNotifyBus returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := bus.Close(); err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := provider.MustParseTopic("runtime.events")
	subscriber, err := bus.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	message, err := provider.NewJSONEvent(
		topic,
		provider.MustParseEventType("runtime.started"),
		map[string]string{"mode": "orchestrate"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(ctx, message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case received := <-subscriber:
		if received.Topic != message.Topic {
			t.Fatalf("expected topic %q, got %q", message.Topic, received.Topic)
		}
		if received.Type != message.Type {
			t.Fatalf("expected type %q, got %q", message.Type, received.Type)
		}
		if string(received.Payload) != string(message.Payload) {
			t.Fatalf("expected payload %s, got %s", string(message.Payload), string(received.Payload))
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for pgnotify delivery: %v", ctx.Err())
	}
}

func startEmbeddedPostgres(t *testing.T) string {
	t.Helper()

	port := freePort(t)
	dataDir := t.TempDir()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Username("postgres").
			Password("postgres").
			Database("openase").
			RuntimePath(filepath.Join(dataDir, "runtime")).
			BinariesPath(filepath.Join(dataDir, "binaries")).
			DataPath(filepath.Join(dataDir, "data")),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})

	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/openase?sslmode=disable", port)
}

func freePort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP address, got %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > math.MaxUint16 {
		t.Fatalf("expected TCP port in uint16 range, got %d", tcpAddr.Port)
	}

	return uint32(tcpAddr.Port) //nolint:gosec // validated above to fit the TCP port range
}
