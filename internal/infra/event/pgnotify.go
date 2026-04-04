package event

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/jackc/pgx/v5"
)

const maxPGNotifyPayloadBytes = 7999
const maxPGChannelNameBytes = 63
const pgChannelPrefix = "openase_"

var pgNotifyBusComponent = logging.DeclareComponent("event-pgnotify-bus")

type PGNotifyBus struct {
	dsn    string
	logger *slog.Logger

	publisherMu sync.Mutex
	publisher   *pgx.Conn

	mu               sync.Mutex
	closed           bool
	nextSubscriberID int
	subscribers      map[int]pgSubscription
}

type pgSubscription struct {
	cancel context.CancelFunc
	conn   *pgx.Conn
}

func NewPGNotifyBus(dsn string, logger *slog.Logger) (*PGNotifyBus, error) {
	trimmedDSN := strings.TrimSpace(dsn)
	if trimmedDSN == "" {
		return nil, fmt.Errorf("pgnotify bus requires a database dsn")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &PGNotifyBus{
		dsn:         trimmedDSN,
		logger:      logging.WithComponent(logger, pgNotifyBusComponent),
		subscribers: make(map[int]pgSubscription),
	}, nil
}

func (b *PGNotifyBus) Publish(ctx context.Context, event provider.Event) error {
	payload, err := encodePGNotifyEvent(event)
	if err != nil {
		return err
	}

	conn, err := b.publisherConn(ctx)
	if err != nil {
		return err
	}

	if _, err := conn.Exec(ctx, "select pg_notify($1, $2)", pgChannelName(event.Topic), payload); err != nil {
		return fmt.Errorf("publish notification: %w", err)
	}

	return nil
}

func (b *PGNotifyBus) Subscribe(ctx context.Context, topics ...provider.Topic) (<-chan provider.Event, error) {
	uniqueTopics, err := dedupeTopics(topics)
	if err != nil {
		return nil, err
	}

	subscribeCtx, cancel := context.WithCancel(ctx)
	conn, err := pgx.Connect(subscribeCtx, b.dsn)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("connect pgnotify subscriber: %w", err)
	}

	for _, topic := range uniqueTopics {
		if _, err := conn.Exec(subscribeCtx, "listen "+pgChannelName(topic)); err != nil {
			cancel()
			_ = conn.Close(context.Background())
			return nil, fmt.Errorf("listen on topic %q: %w", topic, err)
		}
	}

	out := make(chan provider.Event, 16)

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		cancel()
		_ = conn.Close(context.Background())
		return nil, errors.New("pgnotify bus is closed")
	}

	subscriberID := b.nextSubscriberID
	b.nextSubscriberID++
	b.subscribers[subscriberID] = pgSubscription{
		cancel: cancel,
		conn:   conn,
	}
	b.mu.Unlock()

	go b.runSubscription(subscribeCtx, subscriberID, conn, out)

	return out, nil
}

func (b *PGNotifyBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true

	subscribers := b.subscribers
	b.subscribers = make(map[int]pgSubscription)
	b.mu.Unlock()

	var closeErr error
	for _, subscriber := range subscribers {
		subscriber.cancel()
		closeErr = errors.Join(closeErr, subscriber.conn.Close(context.Background()))
	}

	b.publisherMu.Lock()
	publisher := b.publisher
	b.publisher = nil
	b.publisherMu.Unlock()
	if publisher != nil {
		closeErr = errors.Join(closeErr, publisher.Close(context.Background()))
	}

	return closeErr
}

func (b *PGNotifyBus) publisherConn(ctx context.Context) (*pgx.Conn, error) {
	b.publisherMu.Lock()
	defer b.publisherMu.Unlock()

	if b.publisher != nil {
		return b.publisher, nil
	}

	conn, err := pgx.Connect(ctx, b.dsn)
	if err != nil {
		return nil, fmt.Errorf("connect pgnotify publisher: %w", err)
	}

	b.publisher = conn
	return b.publisher, nil
}

func (b *PGNotifyBus) runSubscription(ctx context.Context, subscriberID int, conn *pgx.Conn, out chan provider.Event) {
	defer close(out)
	defer func() {
		b.mu.Lock()
		delete(b.subscribers, subscriberID)
		b.mu.Unlock()
		_ = conn.Close(context.Background())
	}()

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			b.logger.Error("pgnotify subscription stopped", "error", err)
			return
		}

		event, err := decodePGNotifyEvent(notification.Payload)
		if err != nil {
			b.logger.Error("discarding malformed pgnotify event", "channel", notification.Channel, "error", err)
			continue
		}

		select {
		case out <- event:
		case <-ctx.Done():
			return
		}
	}
}

func encodePGNotifyEvent(event provider.Event) (string, error) {
	encoded, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal pgnotify event: %w", err)
	}
	if len(encoded) > maxPGNotifyPayloadBytes {
		return "", fmt.Errorf("event payload exceeds PostgreSQL NOTIFY limit of %d bytes", maxPGNotifyPayloadBytes)
	}

	return string(encoded), nil
}

func decodePGNotifyEvent(payload string) (provider.Event, error) {
	var raw provider.Event
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return provider.Event{}, fmt.Errorf("unmarshal pgnotify event: %w", err)
	}

	return provider.NewEvent(raw.Topic, raw.Type, raw.Payload, raw.PublishedAt)
}

func pgChannelName(topic provider.Topic) string {
	sum := sha256.Sum256([]byte(topic))
	channelName := pgChannelPrefix + hex.EncodeToString(sum[:])
	return channelName[:maxPGChannelNameBytes]
}
