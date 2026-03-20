package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
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

func TestGitHubWebhookRouteSyncsPullRequestStatusToTicketRepoScope(t *testing.T) {
	testCases := []struct {
		name            string
		payload         string
		expectedStatus  ticketreposcope.PrStatus
		expectedPRURL   string
		storedRepoURL   string
		event           string
		initialPRStatus ticketreposcope.PrStatus
	}{
		{
			name:           "opened pull request sets open",
			event:          "pull_request",
			storedRepoURL:  "git@github.com:acme/backend.git",
			expectedStatus: ticketreposcope.PrStatusOpen,
			expectedPRURL:  "https://github.com/acme/backend/pull/42",
			payload:        `{"action":"opened","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"open","merged":false,"head":{"ref":"agent/codex/ASE-42"}}}`,
		},
		{
			name:            "closed merged pull request sets merged",
			event:           "pull_request",
			storedRepoURL:   "https://github.com/acme/backend.git",
			initialPRStatus: ticketreposcope.PrStatusOpen,
			expectedStatus:  ticketreposcope.PrStatusMerged,
			expectedPRURL:   "https://github.com/acme/backend/pull/42",
			payload:         `{"action":"closed","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"closed","merged":true,"head":{"ref":"agent/codex/ASE-42"}}}`,
		},
		{
			name:            "closed unmerged pull request sets closed",
			event:           "pull_request",
			storedRepoURL:   "https://github.com/acme/backend.git",
			initialPRStatus: ticketreposcope.PrStatusOpen,
			expectedStatus:  ticketreposcope.PrStatusClosed,
			expectedPRURL:   "https://github.com/acme/backend/pull/42",
			payload:         `{"action":"closed","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"closed","merged":false,"head":{"ref":"agent/codex/ASE-42"}}}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			client := openTestEntClient(t)
			server, scopeID := newGitHubWebhookSyncTestServer(t, client, testCase.storedRepoURL, testCase.initialPRStatus)

			rec := performGitHubWebhookRequestWithServer(
				t,
				server,
				testCase.event,
				testCase.payload,
				"",
			)

			if rec.Code != http.StatusAccepted {
				t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
			}

			scope, err := client.TicketRepoScope.Get(ctx, scopeID)
			if err != nil {
				t.Fatalf("reload ticket repo scope: %v", err)
			}
			if scope.PrStatus != testCase.expectedStatus {
				t.Fatalf("expected pr_status %q, got %q", testCase.expectedStatus, scope.PrStatus)
			}
			if scope.PullRequestURL != testCase.expectedPRURL {
				t.Fatalf("expected pull_request_url %q, got %q", testCase.expectedPRURL, scope.PullRequestURL)
			}
		})
	}
}

func TestGitHubWebhookRouteSyncsChangesRequestedReviewToTicketRepoScope(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	server, scopeID := newGitHubWebhookSyncTestServer(
		t,
		client,
		"https://github.com/acme/backend.git",
		ticketreposcope.PrStatusOpen,
	)

	payload := `{"action":"submitted","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"open","merged":false,"head":{"ref":"agent/codex/ASE-42"}},"review":{"state":"changes_requested"}}`
	rec := performGitHubWebhookRequestWithServer(t, server, "pull_request_review", payload, "")

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	scope, err := client.TicketRepoScope.Get(ctx, scopeID)
	if err != nil {
		t.Fatalf("reload ticket repo scope: %v", err)
	}
	if scope.PrStatus != ticketreposcope.PrStatusChangesRequested {
		t.Fatalf("expected pr_status %q, got %q", ticketreposcope.PrStatusChangesRequested, scope.PrStatus)
	}
	if scope.PullRequestURL != "https://github.com/acme/backend/pull/42" {
		t.Fatalf("expected pull_request_url to be updated, got %q", scope.PullRequestURL)
	}
}

func TestGitHubWebhookRouteLeavesUnmatchedTicketRepoScopeUnchanged(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	server, scopeID := newGitHubWebhookSyncTestServer(
		t,
		client,
		"https://github.com/acme/backend.git",
		ticketreposcope.PrStatusNone,
	)

	payload := `{"action":"opened","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"open","merged":false,"head":{"ref":"agent/codex/ASE-404"}}}`
	rec := performGitHubWebhookRequestWithServer(t, server, "pull_request", payload, "")

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	scope, err := client.TicketRepoScope.Get(ctx, scopeID)
	if err != nil {
		t.Fatalf("reload ticket repo scope: %v", err)
	}
	if scope.PrStatus != ticketreposcope.PrStatusNone {
		t.Fatalf("expected unmatched scope status to remain %q, got %q", ticketreposcope.PrStatusNone, scope.PrStatus)
	}
	if scope.PullRequestURL != "" {
		t.Fatalf("expected unmatched scope pull_request_url to remain empty, got %q", scope.PullRequestURL)
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

	return performGitHubWebhookRequestWithServer(t, server, event, payload, signature)
}

func performGitHubWebhookRequestWithServer(
	t *testing.T,
	server *Server,
	event string,
	payload string,
	signature string,
) *httptest.ResponseRecorder {
	t.Helper()

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

func newGitHubWebhookSyncTestServer(
	t *testing.T,
	client *ent.Client,
	repositoryURL string,
	initialPRStatus ticketreposcope.PrStatus,
) (*Server, uuid.UUID) {
	t.Helper()

	if initialPRStatus == "" {
		initialPRStatus = ticketreposcope.PrStatusNone
	}

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")

	repo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL(repositoryURL).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-42").
		SetTitle("Sync PR status").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	scope, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(repo.ID).
		SetBranchName("agent/codex/ASE-42").
		SetPrStatus(initialPRStatus).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket repo scope: %v", err)
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		nil,
		nil,
		nil,
	)

	return server, scope.ID
}

func signGitHubWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)

	return gitHubWebhookSignaturePrefix + hex.EncodeToString(mac.Sum(nil))
}
