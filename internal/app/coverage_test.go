package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestAppNewProvidesTelemetryDefaults(t *testing.T) {
	t.Parallel()

	events := &appEventProvider{stream: make(chan provider.Event)}
	app := New(
		config.Config{Server: config.ServerConfig{Mode: config.ServerModeAllInOne}},
		slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
		events,
		nil,
		nil,
		http.NotFoundHandler(),
	)
	if app == nil || app.trace == nil || app.metrics == nil || app.metricsHandler == nil {
		t.Fatalf("New() = %+v, want initialized defaults", app)
	}
}

func TestAppRunModesRejectEmptyDSN(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	events := &appEventProvider{stream: make(chan provider.Event)}
	ctx := context.Background()

	serveApp := New(
		config.Config{
			Server: config.ServerConfig{Mode: config.ServerModeServe},
			Event:  config.EventConfig{Driver: config.EventDriverChannel},
		},
		logger,
		events,
		nil,
		nil,
		http.NotFoundHandler(),
	)
	if err := serveApp.RunServe(ctx); err == nil || !strings.Contains(err.Error(), "database.dsn is required") {
		t.Fatalf("RunServe() error = %v", err)
	}

	orchestrateApp := New(
		config.Config{
			Server:       config.ServerConfig{Mode: config.ServerModeOrchestrate},
			Orchestrator: config.OrchestratorConfig{TickInterval: time.Millisecond},
			Event:        config.EventConfig{Driver: config.EventDriverChannel},
		},
		logger,
		events,
		nil,
		nil,
		nil,
	)
	if err := orchestrateApp.RunOrchestrate(ctx); err == nil || !strings.Contains(err.Error(), "database.dsn is required") {
		t.Fatalf("RunOrchestrate() error = %v", err)
	}
}

func TestAppHelpersAndRuntimeEvents(t *testing.T) {
	t.Parallel()

	if got := sumSkipCounts(map[string]int{"blocked": 2, "done": 3}); got != 5 {
		t.Fatalf("sumSkipCounts() = %d, want 5", got)
	}

	joined := joinOrchestratorTickErrors(
		errors.New("health"),
		nil,
		errors.New("scheduler"),
		errors.New("launcher"),
		nil,
	)
	if joined == nil {
		t.Fatal("joinOrchestratorTickErrors() = nil, want joined error")
	}
	for _, part := range []string{"health check: health", "scheduler: scheduler", "runtime launcher: launcher"} {
		if !strings.Contains(joined.Error(), part) {
			t.Fatalf("joined error %q missing %q", joined.Error(), part)
		}
	}
	if err := joinOrchestratorTickErrors(nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("joinOrchestratorTickErrors(nil...) = %v, want nil", err)
	}

	logBuffer := bytes.NewBuffer(nil)
	events := &appEventProvider{stream: make(chan provider.Event, 1)}
	app := New(
		config.Config{Server: config.ServerConfig{Mode: config.ServerModeAllInOne}},
		slog.New(slog.NewTextHandler(logBuffer, nil)),
		events,
		nil,
		nil,
		http.NotFoundHandler(),
	)

	if err := app.publishRuntimeEvent(context.Background(), runtimeTickType, map[string]string{"mode": "all-in-one"}); err != nil {
		t.Fatalf("publishRuntimeEvent() error = %v", err)
	}
	if events.lastPublished.Topic != runtimeEventsTopic || events.lastPublished.Type != runtimeTickType {
		t.Fatalf("published event = %+v", events.lastPublished)
	}
	var payload map[string]string
	if err := json.Unmarshal(events.lastPublished.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal published payload: %v", err)
	}
	if payload["mode"] != "all-in-one" {
		t.Fatalf("published payload = %+v", payload)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := app.startRuntimeEventLogging(ctx); err != nil {
		t.Fatalf("startRuntimeEventLogging() error = %v", err)
	}
	events.stream <- provider.Event{
		Topic:       runtimeEventsTopic,
		Type:        runtimeStartedType,
		Payload:     []byte(`{"status":"ready"}`),
		PublishedAt: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC),
	}
	for i := 0; i < 50 && !strings.Contains(logBuffer.String(), "runtime event received"); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(logBuffer.String(), "runtime event received") {
		t.Fatalf("runtime event log missing: %s", logBuffer.String())
	}
	if len(events.subscribedTopics) != 1 || events.subscribedTopics[0] != runtimeEventsTopic {
		t.Fatalf("subscribed topics = %v, want [%s]", events.subscribedTopics, runtimeEventsTopic)
	}
	close(events.stream)
}

func TestAppRunAllInOneAndRuntimeEventFailurePaths(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	ctx := context.Background()

	allInOneApp := New(
		config.Config{
			Server:       config.ServerConfig{Mode: config.ServerModeAllInOne},
			Orchestrator: config.OrchestratorConfig{TickInterval: time.Millisecond},
			Event:        config.EventConfig{Driver: config.EventDriverChannel},
		},
		logger,
		&appEventProvider{stream: make(chan provider.Event)},
		nil,
		nil,
		http.NotFoundHandler(),
	)
	if err := allInOneApp.RunAllInOne(ctx); err == nil || (!strings.Contains(err.Error(), "serve runtime: database.dsn is required") && !strings.Contains(err.Error(), "orchestrate runtime: database.dsn is required")) {
		t.Fatalf("RunAllInOne() error = %v", err)
	}

	failingEvents := &appEventProvider{
		stream:       make(chan provider.Event),
		publishErr:   errors.New("publish down"),
		subscribeErr: errors.New("subscribe down"),
	}
	failingApp := New(
		config.Config{Server: config.ServerConfig{Mode: config.ServerModeAllInOne}},
		logger,
		failingEvents,
		nil,
		nil,
		http.NotFoundHandler(),
	)
	if err := failingApp.publishRuntimeEvent(ctx, runtimeTickType, func() {}); err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("publishRuntimeEvent() invalid payload error = %v", err)
	}
	if err := failingApp.publishRuntimeEvent(ctx, runtimeTickType, map[string]string{"mode": "all-in-one"}); err == nil || !strings.Contains(err.Error(), "publish down") {
		t.Fatalf("publishRuntimeEvent() publish error = %v", err)
	}
	if err := failingApp.startRuntimeEventLogging(ctx); err == nil || !strings.Contains(err.Error(), "subscribe runtime events: subscribe down") {
		t.Fatalf("startRuntimeEventLogging() subscribe error = %v", err)
	}
}

func TestAppRunServeWithRealDatabaseAndShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app := New(
		config.Config{
			Server: config.ServerConfig{
				Mode:            config.ServerModeServe,
				Host:            "127.0.0.1",
				Port:            int(freeAppPort(t)),
				ReadTimeout:     time.Second,
				WriteTimeout:    time.Second,
				ShutdownTimeout: 2 * time.Second,
			},
			Database: config.DatabaseConfig{DSN: openAppTestDSN(t)},
			Event:    config.EventConfig{Driver: config.EventDriverChannel},
		},
		slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
		&managedAppEventProvider{},
		nil,
		nil,
		http.NotFoundHandler(),
	)

	if err := app.RunServe(ctx); err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("RunServe() error = %v, want context canceled", err)
	}
}

func TestAppRunServeBuildsRuntimeBeforeListenerFailure(t *testing.T) {
	t.Parallel()

	app := New(
		config.Config{
			Server: config.ServerConfig{
				Mode:            config.ServerModeServe,
				Host:            "300.300.300.300",
				Port:            int(freeAppPort(t)),
				ReadTimeout:     time.Second,
				WriteTimeout:    time.Second,
				ShutdownTimeout: 2 * time.Second,
			},
			Database: config.DatabaseConfig{DSN: openAppTestDSN(t)},
			Event:    config.EventConfig{Driver: config.EventDriverChannel},
		},
		slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
		&managedAppEventProvider{},
		nil,
		nil,
		http.NotFoundHandler(),
	)

	err := app.RunServe(context.Background())
	if err == nil {
		t.Fatal("RunServe() expected listener error for invalid host")
	}
	if !strings.Contains(err.Error(), "listen tcp") && !strings.Contains(err.Error(), "missing port in address") && !strings.Contains(err.Error(), "no suitable address found") {
		t.Fatalf("RunServe() error = %v", err)
	}
}

func TestAppRunOrchestrateWithRealDatabaseAndTick(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := &managedAppEventProvider{
		onPublish: func(event provider.Event) {
			if event.Type == runtimeTickType {
				cancel()
			}
		},
	}
	app := New(
		config.Config{
			Server:       config.ServerConfig{Mode: config.ServerModeOrchestrate},
			Database:     config.DatabaseConfig{DSN: openAppTestDSN(t)},
			Orchestrator: config.OrchestratorConfig{TickInterval: 10 * time.Millisecond},
			Event:        config.EventConfig{Driver: config.EventDriverChannel},
		},
		slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
		events,
		nil,
		nil,
		nil,
	)

	if err := app.RunOrchestrate(ctx); err != nil {
		t.Fatalf("RunOrchestrate() error = %v", err)
	}
	if err := events.Close(); err != nil {
		t.Fatalf("event provider Close() error = %v", err)
	}

	if !events.publishedEventType(runtimeStartedType) {
		t.Fatalf("expected runtime started event, got %+v", events.published)
	}
	if !events.publishedEventType(runtimeTickType) {
		t.Fatalf("expected runtime tick event, got %+v", events.published)
	}
}

func TestAppRunServeAndOrchestrateConfigurationFailurePaths(t *testing.T) {
	t.Run("serve subscribe failure after db open", func(t *testing.T) {
		t.Parallel()

		app := New(
			config.Config{
				Server: config.ServerConfig{
					Mode:            config.ServerModeServe,
					Host:            "127.0.0.1",
					Port:            int(freeAppPort(t)),
					ReadTimeout:     time.Second,
					WriteTimeout:    time.Second,
					ShutdownTimeout: 2 * time.Second,
				},
				Database: config.DatabaseConfig{DSN: openAppTestDSN(t)},
				Event:    config.EventConfig{Driver: config.EventDriverAuto},
			},
			slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
			&appEventProvider{stream: make(chan provider.Event), subscribeErr: errors.New("subscribe down")},
			nil,
			nil,
			http.NotFoundHandler(),
		)

		if err := app.RunServe(context.Background()); err == nil || !strings.Contains(err.Error(), "subscribe runtime events: subscribe down") {
			t.Fatalf("RunServe() subscribe failure error = %v", err)
		}
	})

	t.Run("serve invalid resolved event driver", func(t *testing.T) {
		t.Parallel()

		app := New(
			config.Config{
				Server: config.ServerConfig{
					Mode:            config.ServerModeServe,
					Host:            "127.0.0.1",
					Port:            int(freeAppPort(t)),
					ReadTimeout:     time.Second,
					WriteTimeout:    time.Second,
					ShutdownTimeout: 2 * time.Second,
				},
				Database: config.DatabaseConfig{DSN: openAppTestDSN(t)},
				Event:    config.EventConfig{Driver: config.EventDriver("bogus")},
			},
			slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
			&managedAppEventProvider{},
			nil,
			nil,
			http.NotFoundHandler(),
		)

		if err := app.RunServe(context.Background()); err == nil || !strings.Contains(err.Error(), "unsupported event driver") {
			t.Fatalf("RunServe() invalid driver error = %v", err)
		}
	})

	t.Run("serve notification engine subscribe failure", func(t *testing.T) {
		t.Parallel()

		app := New(
			config.Config{
				Server: config.ServerConfig{
					Mode:            config.ServerModeServe,
					Host:            "127.0.0.1",
					Port:            int(freeAppPort(t)),
					ReadTimeout:     time.Second,
					WriteTimeout:    time.Second,
					ShutdownTimeout: 2 * time.Second,
				},
				Database: config.DatabaseConfig{DSN: openAppTestDSN(t)},
				Event:    config.EventConfig{Driver: config.EventDriverChannel},
			},
			slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
			&sequenceAppEventProvider{
				stream:              make(chan provider.Event),
				subscribeErrors:     []error{nil, errors.New("notification subscribe down")},
				defaultSubscribeErr: errors.New("unexpected extra subscribe"),
			},
			nil,
			nil,
			http.NotFoundHandler(),
		)

		if err := app.RunServe(context.Background()); err == nil || !strings.Contains(err.Error(), "subscribe notification engine: notification subscribe down") {
			t.Fatalf("RunServe() notification subscribe error = %v", err)
		}
	})

	t.Run("orchestrate publish started failure", func(t *testing.T) {
		t.Parallel()

		app := New(
			config.Config{
				Server:       config.ServerConfig{Mode: config.ServerModeOrchestrate},
				Database:     config.DatabaseConfig{DSN: openAppTestDSN(t)},
				Orchestrator: config.OrchestratorConfig{TickInterval: 10 * time.Millisecond},
				Event:        config.EventConfig{Driver: config.EventDriverAuto},
			},
			slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
			&appEventProvider{stream: make(chan provider.Event), publishErr: errors.New("publish down")},
			nil,
			nil,
			nil,
		)

		if err := app.RunOrchestrate(context.Background()); err == nil || !strings.Contains(err.Error(), "publish runtime started event: publish down") {
			t.Fatalf("RunOrchestrate() publish failure error = %v", err)
		}
	})

	t.Run("orchestrate invalid resolved event driver", func(t *testing.T) {
		t.Parallel()

		app := New(
			config.Config{
				Server:       config.ServerConfig{Mode: config.ServerModeOrchestrate},
				Database:     config.DatabaseConfig{DSN: openAppTestDSN(t)},
				Orchestrator: config.OrchestratorConfig{TickInterval: 10 * time.Millisecond},
				Event:        config.EventConfig{Driver: config.EventDriver("bogus")},
			},
			slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
			&managedAppEventProvider{},
			nil,
			nil,
			nil,
		)

		if err := app.RunOrchestrate(context.Background()); err == nil || !strings.Contains(err.Error(), "unsupported event driver") {
			t.Fatalf("RunOrchestrate() invalid driver error = %v", err)
		}
	})

	t.Run("orchestrate tick publish failure", func(t *testing.T) {
		t.Parallel()

		app := New(
			config.Config{
				Server:       config.ServerConfig{Mode: config.ServerModeOrchestrate},
				Database:     config.DatabaseConfig{DSN: openAppTestDSN(t)},
				Orchestrator: config.OrchestratorConfig{TickInterval: 10 * time.Millisecond},
				Event:        config.EventConfig{Driver: config.EventDriverChannel},
			},
			slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
			&sequenceAppEventProvider{
				publishErrors:     []error{nil, errors.New("tick publish down")},
				defaultPublishErr: errors.New("tick publish down"),
			},
			nil,
			nil,
			nil,
		)

		if err := app.RunOrchestrate(context.Background()); err == nil || !strings.Contains(err.Error(), "publish scheduler tick: tick publish down") {
			t.Fatalf("RunOrchestrate() tick publish error = %v", err)
		}
	})
}

type appEventProvider struct {
	subscribedTopics []provider.Topic
	lastPublished    provider.Event
	stream           chan provider.Event
	publishErr       error
	subscribeErr     error
}

func (p *appEventProvider) Publish(_ context.Context, event provider.Event) error {
	if p.publishErr != nil {
		return p.publishErr
	}
	p.lastPublished = event
	return nil
}

func (p *appEventProvider) Subscribe(_ context.Context, topics ...provider.Topic) (<-chan provider.Event, error) {
	if p.subscribeErr != nil {
		return nil, p.subscribeErr
	}
	p.subscribedTopics = append([]provider.Topic(nil), topics...)
	return p.stream, nil
}

func (p *appEventProvider) Close() error {
	return nil
}

type managedAppEventProvider struct {
	mu               sync.Mutex
	subscribedTopics [][]provider.Topic
	published        []provider.Event
	streams          []chan provider.Event
	onPublish        func(provider.Event)
	onSubscribe      func([]provider.Topic, int)
}

func (p *managedAppEventProvider) Publish(_ context.Context, event provider.Event) error {
	p.mu.Lock()
	p.published = append(p.published, event)
	onPublish := p.onPublish
	p.mu.Unlock()

	if onPublish != nil {
		onPublish(event)
	}
	return nil
}

func (p *managedAppEventProvider) Subscribe(_ context.Context, topics ...provider.Topic) (<-chan provider.Event, error) {
	p.mu.Lock()
	stream := make(chan provider.Event, 1)
	p.subscribedTopics = append(p.subscribedTopics, append([]provider.Topic(nil), topics...))
	p.streams = append(p.streams, stream)
	onSubscribe := p.onSubscribe
	subscribeCount := len(p.subscribedTopics)
	p.mu.Unlock()

	if onSubscribe != nil {
		onSubscribe(topics, subscribeCount)
	}

	return stream, nil
}

func (p *managedAppEventProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, stream := range p.streams {
		close(stream)
	}
	p.streams = nil
	return nil
}

func (p *managedAppEventProvider) publishedEventType(want provider.EventType) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, event := range p.published {
		if event.Type == want {
			return true
		}
	}

	return false
}

type sequenceAppEventProvider struct {
	mu                  sync.Mutex
	stream              chan provider.Event
	publishErrors       []error
	subscribeErrors     []error
	defaultPublishErr   error
	defaultSubscribeErr error
	publishCount        int
	subscribeCount      int
}

func (p *sequenceAppEventProvider) Publish(_ context.Context, event provider.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := p.publishCount
	p.publishCount++
	if idx < len(p.publishErrors) {
		return p.publishErrors[idx]
	}
	return p.defaultPublishErr
}

func (p *sequenceAppEventProvider) Subscribe(_ context.Context, topics ...provider.Topic) (<-chan provider.Event, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := p.subscribeCount
	p.subscribeCount++
	if idx < len(p.subscribeErrors) && p.subscribeErrors[idx] != nil {
		return nil, p.subscribeErrors[idx]
	}
	if idx < len(p.subscribeErrors) {
		return p.stream, nil
	}
	if p.defaultSubscribeErr != nil {
		return nil, p.defaultSubscribeErr
	}
	return p.stream, nil
}

func (p *sequenceAppEventProvider) Close() error {
	if p.stream != nil {
		close(p.stream)
	}
	return nil
}

func openAppTestDSN(t *testing.T) string {
	t.Helper()

	return testPostgres.NewIsolatedDatabase(t).DSN
}

func freeAppPort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	parsed, err := strconv.ParseUint(strconv.Itoa(port), 10, 32)
	if err != nil {
		t.Fatalf("parse free port %d: %v", port, err)
	}

	return uint32(parsed)
}
