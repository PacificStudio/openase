package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/labstack/echo/v4"
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

func TestTicketRepoScopeGitHubWebhookRouteUsesSharedReceiver(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/ticket-repo-scope/github", strings.NewReader(`{"action":"opened"}`))
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

func TestInboundWebhookRouteErrorMappingsAndHelpers(t *testing.T) {
	t.Run("invalid route and unavailable receiver", func(t *testing.T) {
		server := newInboundWebhookTestServer()

		invalidRouteRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/webhooks/Bad!/github", `{}`)
		if invalidRouteRec.Code != http.StatusBadRequest || !strings.Contains(invalidRouteRec.Body.String(), "INVALID_WEBHOOK_ROUTE") {
			t.Fatalf("invalid route response = %d %s", invalidRouteRec.Code, invalidRouteRec.Body.String())
		}

		server.inboundWebhooks = nil
		unavailableRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/webhooks/ticket-repo-scope/github", `{}`)
		if unavailableRec.Code != http.StatusServiceUnavailable || !strings.Contains(unavailableRec.Body.String(), "SERVICE_UNAVAILABLE") {
			t.Fatalf("unavailable response = %d %s", unavailableRec.Code, unavailableRec.Body.String())
		}
	})

	t.Run("signature parse and dispatch failures", func(t *testing.T) {
		for _, testCase := range []struct {
			name          string
			endpoint      *stubInboundWebhookEndpoint
			body          string
			wantStatus    int
			wantSubstring string
			wantDispatch  int
		}{
			{
				name: "payload too large",
				endpoint: &stubInboundWebhookEndpoint{
					target:          ticketRepoScopeWebhookTarget,
					maxPayloadBytes: 4,
				},
				body:          `{"long":true}`,
				wantStatus:    http.StatusRequestEntityTooLarge,
				wantSubstring: "PAYLOAD_TOO_LARGE",
			},
			{
				name: "custom signature error",
				endpoint: &stubInboundWebhookEndpoint{
					target:    ticketRepoScopeWebhookTarget,
					verifyErr: &inboundWebhookError{StatusCode: http.StatusTeapot, Code: "CUSTOM_SIGNATURE", Message: "brew tea"},
				},
				body:          `{}`,
				wantStatus:    http.StatusTeapot,
				wantSubstring: "CUSTOM_SIGNATURE",
			},
			{
				name: "generic parse error",
				endpoint: &stubInboundWebhookEndpoint{
					target:   ticketRepoScopeWebhookTarget,
					parseErr: errors.New("bad payload"),
				},
				body:          `{}`,
				wantStatus:    http.StatusBadRequest,
				wantSubstring: "INVALID_REQUEST",
			},
			{
				name: "ignored dispatch",
				endpoint: &stubInboundWebhookEndpoint{
					target: ticketRepoScopeWebhookTarget,
					dispatch: inboundWebhookDispatch{
						Summary: inboundWebhookSummary{Event: "push"},
						Ignore:  true,
					},
				},
				body:          `{}`,
				wantStatus:    http.StatusAccepted,
				wantSubstring: "",
				wantDispatch:  0,
			},
			{
				name: "dispatch failure",
				endpoint: &stubInboundWebhookEndpoint{
					target:      ticketRepoScopeWebhookTarget,
					dispatch:    inboundWebhookDispatch{Summary: inboundWebhookSummary{Event: "push"}},
					dispatchErr: errors.New("dispatch failed"),
				},
				body:          `{}`,
				wantStatus:    http.StatusInternalServerError,
				wantSubstring: "WEBHOOK_DISPATCH_FAILED",
				wantDispatch:  1,
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				server := newInboundWebhookTestServer()
				server.inboundWebhooks = newInboundWebhookReceiver(newInboundWebhookTestLogger(), testCase.endpoint)

				rec := performJSONRequest(t, server, http.MethodPost, "/api/v1/webhooks/ticket-repo-scope/github", testCase.body)
				if rec.Code != testCase.wantStatus {
					t.Fatalf("status = %d, want %d: %s", rec.Code, testCase.wantStatus, rec.Body.String())
				}
				if testCase.wantSubstring != "" && !strings.Contains(rec.Body.String(), testCase.wantSubstring) {
					t.Fatalf("response body = %s, want substring %q", rec.Body.String(), testCase.wantSubstring)
				}
				if testCase.endpoint.dispatchCalls != testCase.wantDispatch {
					t.Fatalf("dispatchCalls = %d, want %d", testCase.endpoint.dispatchCalls, testCase.wantDispatch)
				}
			})
		}
	})
}

func TestInboundWebhookUtilityHelpers(t *testing.T) {
	t.Run("write errors", func(t *testing.T) {
		custom := invokeAPIErrorWriter(t, func(c echo.Context) error {
			return writeInboundWebhookError(c, &inboundWebhookError{
				StatusCode: http.StatusConflict,
				Code:       "CUSTOM_WEBHOOK_ERROR",
				Message:    "conflict",
			}, http.StatusBadRequest, "INVALID_REQUEST")
		})
		assertAPIErrorResponse(t, custom, http.StatusConflict, "CUSTOM_WEBHOOK_ERROR", "conflict")

		fallback := invokeAPIErrorWriter(t, func(c echo.Context) error {
			return writeInboundWebhookError(c, errors.New("boom"), http.StatusUnauthorized, "INVALID_SIGNATURE")
		})
		assertAPIErrorResponse(t, fallback, http.StatusUnauthorized, "INVALID_SIGNATURE", "boom")
	})

	t.Run("payload reader and target parsing", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"ok":true}`))
		payload, err := readInboundWebhookPayload(request, 0)
		if err != nil || string(payload) != `{"ok":true}` {
			t.Fatalf("readInboundWebhookPayload(default limit) = %q, %v", payload, err)
		}

		tooLargeRequest := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("abcdef"))
		if _, err := readInboundWebhookPayload(tooLargeRequest, 4); err == nil || !strings.Contains(err.Error(), "exceeds 4 bytes") {
			t.Fatalf("readInboundWebhookPayload(too large) error = %v", err)
		}

		readErrRequest := httptest.NewRequest(http.MethodPost, "/webhook", http.NoBody)
		readErrRequest.Body = errReadCloser{err: errors.New("broken stream")}
		if _, err := readInboundWebhookPayload(readErrRequest, 16); err == nil || !strings.Contains(err.Error(), "broken stream") {
			t.Fatalf("readInboundWebhookPayload(read error) error = %v", err)
		}

		target, err := parseInboundWebhookTarget(" issue-sync ", " GitHub ")
		if err != nil {
			t.Fatalf("parseInboundWebhookTarget() error = %v", err)
		}
		if target != (inboundWebhookTarget{Connector: "issue-sync", Provider: "github"}) {
			t.Fatalf("parseInboundWebhookTarget() = %+v", target)
		}
		if _, err := parseInboundWebhookTarget("", "github"); err == nil || !strings.Contains(err.Error(), "connector must not be empty") {
			t.Fatalf("parseInboundWebhookTarget(empty connector) error = %v", err)
		}
		if _, err := parseInboundWebhookTarget("issue-sync", "bad!"); err == nil || !strings.Contains(err.Error(), "provider must match") {
			t.Fatalf("parseInboundWebhookTarget(bad provider) error = %v", err)
		}

		if got := (&inboundWebhookError{Message: "boom"}).Error(); got != "boom" {
			t.Fatalf("inboundWebhookError.Error() = %q", got)
		}
		if got := (*inboundWebhookError)(nil).Error(); got != "" {
			t.Fatalf("(*inboundWebhookError)(nil).Error() = %q", got)
		}
		if got := (inboundWebhookTarget{Connector: "issue-sync", Provider: "github"}).logArgs(); len(got) != 4 || got[1] != "issue-sync" || got[3] != "github" {
			t.Fatalf("logArgs() = %+v", got)
		}
	})
}

func TestNewInboundWebhookReceiverRejectsDuplicateTargets(t *testing.T) {
	target := inboundWebhookTarget{
		Connector: inboundWebhookKey("issue-sync"),
		Provider:  inboundWebhookKey("gitlab"),
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic for duplicate targets")
		}
		if got := fmt.Sprint(recovered); !strings.Contains(got, "duplicate inbound webhook endpoint") {
			t.Fatalf("panic = %q", got)
		}
	}()

	newInboundWebhookReceiver(
		newInboundWebhookTestLogger(),
		&stubInboundWebhookEndpoint{target: target},
		&stubInboundWebhookEndpoint{target: target},
	)
}

type stubInboundWebhookEndpoint struct {
	target          inboundWebhookTarget
	dispatch        inboundWebhookDispatch
	verifyErr       error
	parseErr        error
	dispatchErr     error
	maxPayloadBytes int64
	lastRequest     inboundWebhookRequest
	verifyCalls     int
	parseCalls      int
	dispatchCalls   int
}

func (s *stubInboundWebhookEndpoint) Target() inboundWebhookTarget {
	return s.target
}

func (s *stubInboundWebhookEndpoint) MaxPayloadBytes() int64 {
	if s.maxPayloadBytes > 0 {
		return s.maxPayloadBytes
	}
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

type errReadCloser struct {
	err error
}

func (e errReadCloser) Read([]byte) (int, error) {
	return 0, e.err
}

func (e errReadCloser) Close() error {
	return nil
}
