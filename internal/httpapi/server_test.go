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
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/labstack/echo/v4"
)

func TestHealthRoutes(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, slog.New(slog.NewTextHandler(io.Discard, nil)), eventinfra.NewChannelBus())

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

func TestEmbeddedUIRoutes(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, slog.New(slog.NewTextHandler(io.Discard, nil)), eventinfra.NewChannelBus())

	for target, needle := range map[string]string{
		"/":                  "OpenASE Scaffold",
		"/_app/version.json": "version",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", target, rec.Code)
		}

		if !strings.Contains(rec.Body.String(), needle) {
			t.Fatalf("expected %s response to contain %q", target, needle)
		}
	}
}

func TestEventStreamRoute(t *testing.T) {
	bus := eventinfra.NewChannelBus()
	server := NewServer(config.ServerConfig{Port: 40023}, slog.New(slog.NewTextHandler(io.Discard, nil)), bus)
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
	defer response.Body.Close()

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

func TestProjectEventStreamRoutesUseFixedTopics(t *testing.T) {
	testCases := []struct {
		name          string
		path          string
		topic         provider.Topic
		eventType     provider.EventType
		unrelated     provider.Topic
		unrelatedType provider.EventType
	}{
		{
			name:          "tickets",
			path:          "/api/v1/projects/project-123/tickets/stream",
			topic:         provider.MustParseTopic("ticket.events"),
			eventType:     provider.MustParseEventType("ticket.created"),
			unrelated:     provider.MustParseTopic("agent.events"),
			unrelatedType: provider.MustParseEventType("agent.progress"),
		},
		{
			name:          "agents",
			path:          "/api/v1/projects/project-123/agents/stream",
			topic:         provider.MustParseTopic("agent.events"),
			eventType:     provider.MustParseEventType("agent.progress"),
			unrelated:     provider.MustParseTopic("ticket.events"),
			unrelatedType: provider.MustParseEventType("ticket.created"),
		},
		{
			name:          "hooks",
			path:          "/api/v1/projects/project-123/hooks/stream",
			topic:         provider.MustParseTopic("hook.events"),
			eventType:     provider.MustParseEventType("hook.failed"),
			unrelated:     provider.MustParseTopic("ticket.events"),
			unrelatedType: provider.MustParseEventType("ticket.updated"),
		},
		{
			name:          "activity",
			path:          "/api/v1/projects/project-123/activity/stream",
			topic:         provider.MustParseTopic("activity.events"),
			eventType:     provider.MustParseEventType("activity.new"),
			unrelated:     provider.MustParseTopic("hook.events"),
			unrelatedType: provider.MustParseEventType("hook.failed"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bus := eventinfra.NewChannelBus()
			server := NewServer(config.ServerConfig{Port: 40023}, slog.New(slog.NewTextHandler(io.Discard, nil)), bus)
			testServer := httptest.NewServer(server.Handler())
			defer testServer.Close()

			response, cancel := openSSERequest(t, testServer.URL+testCase.path)
			defer response.Body.Close()

			publishTestEvent(t, bus, testCase.unrelated, testCase.unrelatedType, map[string]string{"scope": "other"})
			publishTestEvent(t, bus, testCase.topic, testCase.eventType, map[string]string{"scope": "expected"})

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
			if !strings.Contains(body, "\"topic\":\""+testCase.topic.String()+"\"") {
				t.Fatalf("expected %s topic payload, got %q", testCase.topic, body)
			}
			if strings.Contains(body, testCase.unrelatedType.String()) {
				t.Fatalf("did not expect unrelated %s frame, got %q", testCase.unrelatedType, body)
			}
		})
	}
}

func TestEventStreamRouteRejectsMissingTopic(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, slog.New(slog.NewTextHandler(io.Discard, nil)), eventinfra.NewChannelBus())

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
	payload map[string]string,
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
