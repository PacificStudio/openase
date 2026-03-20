package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

func TestGitHubWebhookRouteAcceptsValidSignedPullRequest(t *testing.T) {
	secret := "topsecret"
	payload := `{"action":"opened","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"open","merged":false,"head":{"ref":"agent/codex/ASE-42"}}}`
	rec := performGitHubWebhookRequest(
		t,
		config.GitHubConfig{WebhookSecret: secret},
		"pull_request",
		payload,
		signGitHubWebhookPayload(secret, []byte(payload)),
	)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", rec.Body.String())
	}
}

func TestGitHubWebhookRouteRejectsInvalidSignature(t *testing.T) {
	payload := `{"action":"opened","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git"},"pull_request":{"head":{"ref":"agent/codex/ASE-42"}}}`
	rec := performGitHubWebhookRequest(
		t,
		config.GitHubConfig{WebhookSecret: "topsecret"},
		"pull_request",
		payload,
		signGitHubWebhookPayload("wrongsecret", []byte(payload)),
	)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "INVALID_SIGNATURE") {
		t.Fatalf("expected invalid signature error, got %s", rec.Body.String())
	}
}

func TestGitHubWebhookRouteAllowsUnsignedRequestsWhenSecretIsUnset(t *testing.T) {
	payload := `{"action":"opened","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git"},"pull_request":{"head":{"ref":"agent/codex/ASE-42"}}}`
	rec := performGitHubWebhookRequest(t, config.GitHubConfig{}, "pull_request", payload, "")

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGitHubWebhookRouteAcceptsSignedPullRequestReview(t *testing.T) {
	secret := "topsecret"
	payload := `{"action":"submitted","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git"},"pull_request":{"head":{"ref":"agent/codex/ASE-42"}},"review":{"state":"changes_requested"}}`
	rec := performGitHubWebhookRequest(
		t,
		config.GitHubConfig{WebhookSecret: secret},
		"pull_request_review",
		payload,
		signGitHubWebhookPayload(secret, []byte(payload)),
	)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGitHubWebhookRouteIgnoresUnsupportedEventsAfterVerification(t *testing.T) {
	secret := "topsecret"
	payload := `{"zen":"keep it logically awesome"}`
	rec := performGitHubWebhookRequest(
		t,
		config.GitHubConfig{WebhookSecret: secret},
		"ping",
		payload,
		signGitHubWebhookPayload(secret, []byte(payload)),
	)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func performGitHubWebhookRequest(
	t *testing.T,
	githubCfg config.GitHubConfig,
	event string,
	payload string,
	signature string,
) *httptest.ResponseRecorder {
	t.Helper()

	server := NewServer(
		config.ServerConfig{Port: 40023},
		githubCfg,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", strings.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(gitHubWebhookEventHeader, event)
	req.Header.Set(gitHubWebhookDeliveryIDHeader, "delivery-123")
	if signature != "" {
		req.Header.Set(gitHubWebhookSignatureHeader, signature)
	}

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	return rec
}

func signGitHubWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)

	return gitHubWebhookSignaturePrefix + hex.EncodeToString(mac.Sum(nil))
}
