package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
)

func TestInboundWebhookRouteDispatchesConnectorProviderEndpoint(t *testing.T) {
	endpoint := &stubInboundWebhookEndpoint{
		target: inboundWebhookTarget{
			Connector: inboundWebhookKey("issue-sync"),
			Provider:  inboundWebhookKey("gitlab"),
		},
		dispatch: inboundWebhookDispatch{
			Summary: inboundWebhookSummary{
				Event:      "issue",
				DeliveryID: "delivery-123",
				Action:     "opened",
				LogArgs: []any{
					"event", "issue",
					"delivery_id", "delivery-123",
					"action", "opened",
				},
			},
			Payload: "parsed",
		},
	}

	server := newInboundWebhookTestServer()
	server.inboundWebhooks = newInboundWebhookReceiver(newInboundWebhookTestLogger(), endpoint)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/issue-sync/gitlab", strings.NewReader(`{"object_kind":"issue"}`))
	req.Header.Set(echoHeaderContentTypeJSON, echoMIMEApplicationJSON)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if endpoint.verifyCalls != 1 || endpoint.parseCalls != 1 || endpoint.dispatchCalls != 1 {
		t.Fatalf("unexpected call counts: %+v", endpoint)
	}
	if got := string(endpoint.lastRequest.Target.Connector); got != "issue-sync" {
		t.Fatalf("connector = %q, want %q", got, "issue-sync")
	}
	if got := string(endpoint.lastRequest.Target.Provider); got != "gitlab" {
		t.Fatalf("provider = %q, want %q", got, "gitlab")
	}
	if got := string(endpoint.lastRequest.Payload); got != `{"object_kind":"issue"}` {
		t.Fatalf("payload = %q", got)
	}
}

func TestLegacyGitHubWebhookRouteUsesSharedReceiver(t *testing.T) {
	endpoint := &stubInboundWebhookEndpoint{
		target: ticketRepoScopeWebhookTarget,
		dispatch: inboundWebhookDispatch{
			Summary: inboundWebhookSummary{
				Event:      "pull_request",
				DeliveryID: "delivery-123",
				Action:     "opened",
				LogArgs: []any{
					"event", "pull_request",
					"delivery_id", "delivery-123",
					"action", "opened",
				},
			},
			Payload: "parsed",
		},
	}

	server := newInboundWebhookTestServer()
	server.inboundWebhooks = newInboundWebhookReceiver(newInboundWebhookTestLogger(), endpoint)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", strings.NewReader(`{"action":"opened"}`))
	req.Header.Set(echoHeaderContentTypeJSON, echoMIMEApplicationJSON)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if endpoint.dispatchCalls != 1 {
		t.Fatalf("dispatchCalls = %d, want 1", endpoint.dispatchCalls)
	}
	if endpoint.lastRequest.Target != ticketRepoScopeWebhookTarget {
		t.Fatalf("target = %+v, want %+v", endpoint.lastRequest.Target, ticketRepoScopeWebhookTarget)
	}
}

func TestInboundWebhookRouteRejectsUnknownTarget(t *testing.T) {
	server := newInboundWebhookTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/issue-sync/unknown", strings.NewReader(`{}`))
	req.Header.Set(echoHeaderContentTypeJSON, echoMIMEApplicationJSON)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "WEBHOOK_ROUTE_NOT_FOUND") {
		t.Fatalf("expected route not found error, got %s", rec.Body.String())
	}
}

type stubInboundWebhookEndpoint struct {
	target        inboundWebhookTarget
	dispatch      inboundWebhookDispatch
	verifyErr     error
	parseErr      error
	dispatchErr   error
	lastRequest   inboundWebhookRequest
	verifyCalls   int
	parseCalls    int
	dispatchCalls int
}

func (s *stubInboundWebhookEndpoint) Target() inboundWebhookTarget {
	return s.target
}

func (s *stubInboundWebhookEndpoint) MaxPayloadBytes() int64 {
	return 1024
}

func (s *stubInboundWebhookEndpoint) VerifySignature(request inboundWebhookRequest) error {
	s.verifyCalls++
	s.lastRequest = request
	return s.verifyErr
}

func (s *stubInboundWebhookEndpoint) ParseEvent(request inboundWebhookRequest) (inboundWebhookDispatch, error) {
	s.parseCalls++
	s.lastRequest = request
	if s.parseErr != nil {
		return inboundWebhookDispatch{}, s.parseErr
	}

	return s.dispatch, nil
}

func (s *stubInboundWebhookEndpoint) Dispatch(context.Context, inboundWebhookDispatch) error {
	s.dispatchCalls++
	return s.dispatchErr
}

func newInboundWebhookTestServer() *Server {
	return NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		newInboundWebhookTestLogger(),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func newInboundWebhookTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const (
	echoHeaderContentTypeJSON = "Content-Type"
	echoMIMEApplicationJSON   = "application/json"
)
