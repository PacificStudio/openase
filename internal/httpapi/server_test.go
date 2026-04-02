package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	otelinfra "github.com/BetterAndBetterII/openase/internal/infra/otel"
	"github.com/BetterAndBetterII/openase/internal/provider"
	runtimeobservability "github.com/BetterAndBetterII/openase/internal/runtime/observability"
	"github.com/BetterAndBetterII/openase/internal/webui"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestHealthRoutes(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, config.GitHubConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)), eventinfra.NewChannelBus(), nil, nil, nil, nil, nil)

	for _, target := range []string{"/healthz", "/api/v1/healthz"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", target, rec.Code)
		}

		var payload map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("expected JSON payload for %s: %v", target, err)
		}

		if payload["status"] != "ok" {
			t.Fatalf("expected ok status for %s, got %q", target, payload["status"])
		}
	}
}

func TestMetricsRouteDisabledByDefault(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, config.GitHubConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)), eventinfra.NewChannelBus(), nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/metrics", http.NoBody)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected disabled metrics route to return 404, got %d", rec.Code)
	}
}

func TestSystemDashboardRouteReturnsMemorySnapshot(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProcessMemoryCollector(staticProcessMemoryCollector{snapshot: runtimeobservability.ProcessMemorySnapshot{
			ObservedAt:        time.Date(2026, time.March, 20, 9, 15, 0, 0, time.UTC),
			Goroutines:        11,
			AllocBytes:        2048,
			TotalAllocBytes:   4096,
			SysBytes:          8192,
			HeapAllocBytes:    1024,
			HeapInuseBytes:    1536,
			HeapIdleBytes:     2560,
			HeapReleasedBytes: 512,
			StackInuseBytes:   768,
			NextGCBytes:       6144,
			GCCycles:          5,
		}}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/dashboard", http.NoBody)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected system dashboard route to return 200, got %d", rec.Code)
	}

	var payload OpenAPISystemDashboardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON payload: %v", err)
	}
	if payload.Memory.ObservedAt != "2026-03-20T09:15:00Z" {
		t.Fatalf("expected observed_at to round-trip, got %q", payload.Memory.ObservedAt)
	}
	if payload.Memory.HeapInuseBytes != 1536 {
		t.Fatalf("expected heap_inuse_bytes 1536, got %d", payload.Memory.HeapInuseBytes)
	}
	if payload.Memory.GCCycles != 5 {
		t.Fatalf("expected gc_cycles 5, got %d", payload.Memory.GCCycles)
	}
}

func TestMetricsRouteExportsHTTPMetrics(t *testing.T) {
	metricsProvider, err := otelinfra.NewMetricsProvider(context.Background(), otelinfra.MetricsConfig{
		ServiceName: "openase",
		Prometheus:  true,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewMetricsProvider returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := metricsProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithMetricsProvider(metricsProvider),
		WithMetricsHandler(metricsProvider.PrometheusHandler()),
	)

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	healthRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("expected health route to return 200, got %d", healthRec.Code)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/api/v1/system/metrics", http.NoBody)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("expected metrics route to return 200, got %d", metricsRec.Code)
	}

	body := metricsRec.Body.String()
	for _, expected := range []string{
		`openase_http_server_requests_total{method="GET",route="/healthz",status="200"} 1`,
		`openase_http_server_duration_seconds_count{method="GET",route="/healthz",status="200"} 1`,
		`openase_http_server_in_flight_requests{server="http"} 0`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected metrics scrape to contain %q, got %q", expected, body)
		}
	}
}

type staticProcessMemoryCollector struct {
	snapshot runtimeobservability.ProcessMemorySnapshot
}

func (c staticProcessMemoryCollector) Snapshot() runtimeobservability.ProcessMemorySnapshot {
	return c.snapshot
}

func TestHealthRouteCreatesTracingSpan(t *testing.T) {
	traceProvider := &recordingTraceProvider{
		span: &recordingSpan{
			traceID: "trace-123",
			spanID:  "span-456",
		},
	}
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithTraceProvider(traceProvider),
	)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if traceProvider.startName != "GET /healthz" {
		t.Fatalf("expected span name GET /healthz, got %q", traceProvider.startName)
	}
	if traceProvider.startKind != provider.SpanKindServer {
		t.Fatalf("expected server span kind, got %q", traceProvider.startKind)
	}
	if got := rec.Header().Get(traceIDHeader); got != "trace-123" {
		t.Fatalf("expected %s header trace-123, got %q", traceIDHeader, got)
	}
	if got := rec.Header().Get("Traceparent"); got != "injected" {
		t.Fatalf("expected Traceparent header injected, got %q", got)
	}
	if traceProvider.span.status != provider.SpanStatusOK {
		t.Fatalf("expected span status ok, got %q", traceProvider.span.status)
	}
	if !traceProvider.span.ended {
		t.Fatal("expected span to end")
	}
	if got := findRecordedAttribute(traceProvider.span.attrs, "http.status_code").Int64Value; got != http.StatusOK {
		t.Fatalf("expected http.status_code attribute %d, got %d", http.StatusOK, got)
	}
	if got := findRecordedAttribute(traceProvider.span.attrs, "http.request_id").StringValue; got == "" {
		t.Fatal("expected http.request_id attribute")
	}
	if got := findRecordedAttribute(traceProvider.startAttrs, "http.method").StringValue; got != http.MethodGet {
		t.Fatalf("expected http.method attribute %q, got %q", http.MethodGet, got)
	}
	if got := findRecordedAttribute(traceProvider.startAttrs, "http.route").StringValue; got != "/healthz" {
		t.Fatalf("expected http.route attribute /healthz, got %q", got)
	}
}

func TestEmbeddedUIRoutes(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, config.GitHubConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)), eventinfra.NewChannelBus(), nil, nil, nil, nil, nil)
	uiHandler := webui.Handler()

	for _, target := range []string{"/", "/_app/version.json"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		expected := httptest.NewRecorder()
		rec := httptest.NewRecorder()

		uiHandler.ServeHTTP(expected, req.Clone(req.Context()))

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != expected.Code {
			t.Fatalf("expected %s to return %d, got %d", target, expected.Code, rec.Code)
		}

		if rec.Body.String() != expected.Body.String() {
			t.Fatalf("expected %s response body %q, got %q", target, expected.Body.String(), rec.Body.String())
		}

		if got := rec.Header().Get(echo.HeaderContentType); got != expected.Header().Get(echo.HeaderContentType) {
			t.Fatalf("expected %s content-type %q, got %q", target, expected.Header().Get(echo.HeaderContentType), got)
		}
	}
}

func TestEventStreamRoute(t *testing.T) {
	bus := eventinfra.NewChannelBus()
	server := NewServer(config.ServerConfig{Port: 40023}, config.GitHubConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, nil, nil, nil, nil, nil)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	topic := provider.MustParseTopic("runtime.events")
	requestCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, testServer.URL+"/api/v1/events/stream?topic=runtime.events", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close event stream response body: %v", err)
		}
	})

	message, err := provider.NewJSONEvent(
		topic,
		provider.MustParseEventType("runtime.started"),
		map[string]string{"mode": "serve"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	bodyCh := make(chan string, 1)
	go func() {
		bytes, _ := io.ReadAll(response.Body)
		bodyCh <- string(bytes)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	var body string
	select {
	case body = <-bodyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE response body")
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if contentType := response.Header.Get(echo.HeaderContentType); contentType != "text/event-stream" {
		t.Fatalf("expected event-stream content type, got %q", contentType)
	}
	if !strings.Contains(body, ": keepalive\n\n") {
		t.Fatalf("expected keepalive comment, got %q", body)
	}
	if !strings.Contains(body, "event: runtime.started\n") {
		t.Fatalf("expected runtime.started frame, got %q", body)
	}
	if !strings.Contains(body, "\"topic\":\"runtime.events\"") {
		t.Fatalf("expected topic payload in body, got %q", body)
	}
}

func TestProjectEventBusFiltersAndMultiplexesTopics(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()

	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	response, cancel := openSSERequest(
		t,
		testServer.URL+"/api/v1/projects/"+projectID.String()+"/events/stream",
	)
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close project event bus response body: %v", err)
		}
	})

	publishTestEvent(
		t,
		bus,
		ticketStreamTopic,
		provider.MustParseEventType("ticket.created"),
		map[string]any{
			"project_id": otherProjectID.String(),
			"ticket":     map[string]any{"id": uuid.NewString()},
		},
	)
	publishTestEvent(
		t,
		bus,
		ticketStreamTopic,
		provider.MustParseEventType("ticket.created"),
		map[string]any{
			"project_id": projectID.String(),
			"ticket":     map[string]any{"id": uuid.NewString()},
		},
	)
	publishTestEvent(
		t,
		bus,
		agentStreamTopic,
		provider.MustParseEventType("agent.ready"),
		map[string]any{
			"agent": map[string]any{
				"id":         uuid.NewString(),
				"project_id": projectID.String(),
			},
		},
	)
	publishTestEvent(
		t,
		bus,
		hookStreamTopic,
		provider.MustParseEventType("hook.failed"),
		map[string]any{
			"project_id": projectID.String(),
			"hook": map[string]any{
				"id":         uuid.NewString(),
				"project_id": projectID.String(),
			},
		},
	)
	publishTestEvent(
		t,
		bus,
		activityStreamTopic,
		provider.MustParseEventType("ticket.updated"),
		map[string]any{
			"event": map[string]any{
				"id":         uuid.NewString(),
				"project_id": projectID.String(),
				"event_type": "ticket.updated",
				"message":    "reload board",
				"metadata":   map[string]any{},
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	)

	body := readSSEBody(t, response, cancel)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if contentType := response.Header.Get(echo.HeaderContentType); contentType != "text/event-stream" {
		t.Fatalf("expected event-stream content type, got %q", contentType)
	}
	if !strings.Contains(body, ": keepalive\n\n") {
		t.Fatalf("expected keepalive comment, got %q", body)
	}
	for _, expected := range []string{
		"event: ticket.created\n",
		"event: agent.ready\n",
		"event: hook.failed\n",
		"event: ticket.updated\n",
		"\"topic\":\"ticket.events\"",
		"\"topic\":\"agent.events\"",
		"\"topic\":\"hook.events\"",
		"\"topic\":\"activity.events\"",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected project event bus body to contain %q, got %q", expected, body)
		}
	}
	if strings.Contains(body, otherProjectID.String()) {
		t.Fatalf("did not expect unrelated project payload, got %q", body)
	}
}

func TestProjectPassiveStreamRoutesStayOnCanonicalBus(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	allowed := map[string]bool{
		"/api/v1/projects/:projectId/events/stream":                 true,
		"/api/v1/projects/:projectId/agents/:agentId/output/stream": true,
		"/api/v1/projects/:projectId/agents/:agentId/steps/stream":  true,
	}

	projectStreamRoutes := make([]string, 0)
	for _, route := range server.echo.Routes() {
		if route.Method != http.MethodGet {
			continue
		}
		if !strings.HasPrefix(route.Path, "/api/v1/projects/:projectId/") || !strings.HasSuffix(route.Path, "/stream") {
			continue
		}
		projectStreamRoutes = append(projectStreamRoutes, route.Path)
		if !allowed[route.Path] {
			t.Fatalf("unexpected project stream route registered: %s", route.Path)
		}
	}

	if len(projectStreamRoutes) != len(allowed) {
		t.Fatalf("expected %d canonical project stream routes, got %v", len(allowed), projectStreamRoutes)
	}
}

func TestOrganizationEventStreamRoutesUseFixedTopics(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()

	testCases := []struct {
		name          string
		path          string
		topic         provider.Topic
		eventType     provider.EventType
		unrelatedType provider.EventType
		payloadKey    string
	}{
		{
			name:          "machines",
			path:          "/api/v1/orgs/" + orgID.String() + "/machines/stream",
			topic:         provider.MustParseTopic("machine.events"),
			eventType:     provider.MustParseEventType("machine.online"),
			unrelatedType: provider.MustParseEventType("machine.degraded"),
			payloadKey:    "machine",
		},
		{
			name:          "providers",
			path:          "/api/v1/orgs/" + orgID.String() + "/providers/stream",
			topic:         provider.MustParseTopic("provider.events"),
			eventType:     provider.MustParseEventType("provider.available"),
			unrelatedType: provider.MustParseEventType("provider.unavailable"),
			payloadKey:    "provider",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bus := eventinfra.NewChannelBus()
			server := NewServer(config.ServerConfig{Port: 40023}, config.GitHubConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, nil, nil, nil, nil, nil)
			testServer := httptest.NewServer(server.Handler())
			defer testServer.Close()

			response, cancel := openSSERequest(t, testServer.URL+testCase.path)
			t.Cleanup(func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("close organization event stream response body: %v", err)
				}
			})

			publishTestEvent(t, bus, testCase.topic, testCase.unrelatedType, map[string]any{
				"organization_id": otherOrgID.String(),
				testCase.payloadKey: map[string]any{
					"id": otherOrgID.String(),
				},
			})
			publishTestEvent(t, bus, testCase.topic, testCase.eventType, map[string]any{
				"organization_id": orgID.String(),
				testCase.payloadKey: map[string]any{
					"id": orgID.String(),
				},
			})

			body := readSSEBody(t, response, cancel)

			if response.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", response.StatusCode)
			}
			if contentType := response.Header.Get(echo.HeaderContentType); contentType != "text/event-stream" {
				t.Fatalf("expected event-stream content type, got %q", contentType)
			}
			if !strings.Contains(body, ": keepalive\n\n") {
				t.Fatalf("expected keepalive comment, got %q", body)
			}
			if !strings.Contains(body, "event: "+testCase.eventType.String()+"\n") {
				t.Fatalf("expected %s frame, got %q", testCase.eventType, body)
			}
			if !strings.Contains(body, "\"organization_id\":\""+orgID.String()+"\"") {
				t.Fatalf("expected organization-scoped payload, got %q", body)
			}
			if strings.Contains(body, otherOrgID.String()) {
				t.Fatalf("did not expect unrelated organization payload, got %q", body)
			}
		})
	}
}

func TestEventStreamRouteRejectsMissingTopic(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, config.GitHubConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)), eventinfra.NewChannelBus(), nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "topic query parameter") {
		t.Fatalf("expected missing topic error, got %q", rec.Body.String())
	}
}

func openSSERequest(t *testing.T, url string) (*http.Response, context.CancelFunc) {
	t.Helper()

	requestCtx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		t.Fatalf("NewRequest returned error: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		cancel()
		t.Fatalf("Do returned error: %v", err)
	}

	return response, cancel
}

func publishTestEvent(
	t *testing.T,
	bus *eventinfra.ChannelBus,
	topic provider.Topic,
	eventType provider.EventType,
	payload any,
) {
	t.Helper()

	message, err := provider.NewJSONEvent(topic, eventType, payload, time.Now())
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
}

func readSSEBody(t *testing.T, response *http.Response, cancel context.CancelFunc) string {
	t.Helper()

	bodyCh := make(chan string, 1)
	go func() {
		bytes, _ := io.ReadAll(response.Body)
		bodyCh <- string(bytes)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case body := <-bodyCh:
		return body
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE response body")
		return ""
	}
}

type recordingTraceProvider struct {
	startName  string
	startKind  provider.SpanKind
	startAttrs []provider.SpanAttribute
	span       *recordingSpan
}

func (p *recordingTraceProvider) ExtractHTTPContext(ctx context.Context, _ http.Header) context.Context {
	return ctx
}

func (p *recordingTraceProvider) InjectHTTPHeaders(_ context.Context, header http.Header) {
	header.Set("Traceparent", "injected")
}

func (p *recordingTraceProvider) StartSpan(ctx context.Context, name string, opts ...provider.SpanStartOption) (context.Context, provider.Span) {
	p.startName = name
	p.startKind, p.startAttrs = provider.ResolveSpanStartOptions(opts...)
	return ctx, p.span
}

func (p *recordingTraceProvider) Shutdown(context.Context) error {
	return nil
}

type recordingSpan struct {
	traceID string
	spanID  string
	attrs   []provider.SpanAttribute
	status  provider.SpanStatusCode
	ended   bool
}

func (s *recordingSpan) End() {
	s.ended = true
}

func (s *recordingSpan) RecordError(error) {}

func (s *recordingSpan) SetAttributes(attrs ...provider.SpanAttribute) {
	s.attrs = append(s.attrs, attrs...)
}

func (s *recordingSpan) SetStatus(code provider.SpanStatusCode, _ string) {
	s.status = code
}

func (s *recordingSpan) TraceID() string {
	return s.traceID
}

func (s *recordingSpan) SpanID() string {
	return s.spanID
}

func findRecordedAttribute(attrs []provider.SpanAttribute, key string) provider.SpanAttribute {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr
		}
	}

	return provider.SpanAttribute{}
}
