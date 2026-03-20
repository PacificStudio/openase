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
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
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

func TestGitHubWebhookRouteFinishesTicketWhenAllRepoScopesMerge(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := newGitHubWebhookLifecycleFixture(t, client, ticketreposcope.PrStatusOpen, ticketreposcope.PrStatusMerged)
	before := time.Now().UTC()

	payload := `{"action":"closed","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"closed","merged":true,"head":{"ref":"agent/codex/ASE-42"}}}`
	rec := performGitHubWebhookRequestWithServer(t, fixture.server, "pull_request", payload, "")
	after := time.Now().UTC()

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	ticketAfter, err := client.Ticket.Get(ctx, fixture.ticketID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StatusID != fixture.doneID {
		t.Fatalf("expected ticket to move to Done %s, got %s", fixture.doneID, ticketAfter.StatusID)
	}
	if ticketAfter.CompletedAt == nil || ticketAfter.CompletedAt.Before(before) || ticketAfter.CompletedAt.After(after) {
		t.Fatalf("expected completed_at between %s and %s, got %+v", before, after, ticketAfter.CompletedAt)
	}
	if ticketAfter.AssignedAgentID != nil {
		t.Fatalf("expected assigned agent to be cleared, got %+v", ticketAfter.AssignedAgentID)
	}
	if ticketAfter.NextRetryAt != nil || ticketAfter.RetryPaused || ticketAfter.PauseReason != "" {
		t.Fatalf("expected finish to clear retry scheduling, got %+v", ticketAfter)
	}

	scopeAfter, err := client.TicketRepoScope.Get(ctx, fixture.primaryScopeID)
	if err != nil {
		t.Fatalf("reload primary scope: %v", err)
	}
	if scopeAfter.PrStatus != ticketreposcope.PrStatusMerged {
		t.Fatalf("expected primary scope to be merged, got %q", scopeAfter.PrStatus)
	}

	agentAfter, err := client.Agent.Get(ctx, fixture.agentID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.Status != entagent.StatusIdle || agentAfter.CurrentTicketID != nil {
		t.Fatalf("expected agent release after finish, got %+v", agentAfter)
	}
}

func TestGitHubWebhookRouteSchedulesRetryWhenPullRequestClosesWithoutMerge(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := newGitHubWebhookLifecycleFixture(t, client, ticketreposcope.PrStatusOpen, ticketreposcope.PrStatusMerged)
	if _, err := client.Ticket.UpdateOneID(fixture.ticketID).
		SetAttemptCount(1).
		SetConsecutiveErrors(2).
		Save(ctx); err != nil {
		t.Fatalf("seed ticket retry counters: %v", err)
	}
	before := time.Now().UTC()

	payload := `{"action":"closed","number":42,"repository":{"clone_url":"https://github.com/acme/backend.git","full_name":"acme/backend"},"pull_request":{"html_url":"https://github.com/acme/backend/pull/42","state":"closed","merged":false,"head":{"ref":"agent/codex/ASE-42"}}}`
	rec := performGitHubWebhookRequestWithServer(t, fixture.server, "pull_request", payload, "")
	after := time.Now().UTC()

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	ticketAfter, err := client.Ticket.Get(ctx, fixture.ticketID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StatusID != fixture.todoID {
		t.Fatalf("expected ticket to remain in Todo %s, got %s", fixture.todoID, ticketAfter.StatusID)
	}
	if ticketAfter.AttemptCount != 2 || ticketAfter.ConsecutiveErrors != 3 {
		t.Fatalf("unexpected retry counters after closed PR: %+v", ticketAfter)
	}
	wantMinRetryAt := before.Add(20 * time.Second)
	wantMaxRetryAt := after.Add(20 * time.Second)
	if ticketAfter.NextRetryAt == nil || ticketAfter.NextRetryAt.Before(wantMinRetryAt) || ticketAfter.NextRetryAt.After(wantMaxRetryAt) {
		t.Fatalf("expected next_retry_at between %s and %s, got %+v", wantMinRetryAt, wantMaxRetryAt, ticketAfter.NextRetryAt)
	}
	if ticketAfter.AssignedAgentID != nil {
		t.Fatalf("expected assigned agent to be cleared, got %+v", ticketAfter.AssignedAgentID)
	}
	if ticketAfter.CompletedAt != nil {
		t.Fatalf("expected ticket to remain incomplete, got %+v", ticketAfter.CompletedAt)
	}

	scopeAfter, err := client.TicketRepoScope.Get(ctx, fixture.primaryScopeID)
	if err != nil {
		t.Fatalf("reload primary scope: %v", err)
	}
	if scopeAfter.PrStatus != ticketreposcope.PrStatusClosed {
		t.Fatalf("expected primary scope to be closed, got %q", scopeAfter.PrStatus)
	}

	agentAfter, err := client.Agent.Get(ctx, fixture.agentID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.Status != entagent.StatusIdle || agentAfter.CurrentTicketID != nil {
		t.Fatalf("expected agent release after retry scheduling, got %+v", agentAfter)
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

type gitHubWebhookLifecycleFixture struct {
	server         *Server
	ticketID       uuid.UUID
	todoID         uuid.UUID
	doneID         uuid.UUID
	agentID        uuid.UUID
	primaryScopeID uuid.UUID
}

func newGitHubWebhookLifecycleFixture(
	t *testing.T,
	client *ent.Client,
	primaryStatus ticketreposcope.PrStatus,
	secondaryStatus ticketreposcope.PrStatus,
) gitHubWebhookLifecycleFixture {
	t.Helper()

	if primaryStatus == "" {
		primaryStatus = ticketreposcope.PrStatusNone
	}
	if secondaryStatus == "" {
		secondaryStatus = ticketreposcope.PrStatusNone
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
	doneID := findStatusIDByName(t, statuses, "Done")

	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetName("codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("codex-01").
		SetStatus(entagent.StatusRunning).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetPickupStatusID(todoID).
		SetFinishStatusID(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-42").
		SetTitle("Sync PR status lifecycle").
		SetStatusID(todoID).
		SetWorkflowID(workflowItem.ID).
		SetAssignedAgentID(agentItem.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetCurrentTicketID(ticketItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("attach agent to ticket: %v", err)
	}

	backendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/acme/frontend.git").
		Save(ctx)
	if err != nil {
		t.Fatalf("create frontend repo: %v", err)
	}
	primaryScope, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("agent/codex/ASE-42").
		SetPrStatus(primaryStatus).
		Save(ctx)
	if err != nil {
		t.Fatalf("create primary ticket repo scope: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(frontendRepo.ID).
		SetBranchName("agent/codex/ASE-42").
		SetPrStatus(secondaryStatus).
		Save(ctx); err != nil {
		t.Fatalf("create secondary ticket repo scope: %v", err)
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
		nil,
	)

	return gitHubWebhookLifecycleFixture{
		server:         server,
		ticketID:       ticketItem.ID,
		todoID:         todoID,
		doneID:         doneID,
		agentID:        agentItem.ID,
		primaryScopeID: primaryScope.ID,
	}
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
	doneID := findStatusIDByName(t, statuses, "Done")
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetPickupStatusID(todoID).
		SetFinishStatusID(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

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
		SetWorkflowID(workflowItem.ID).
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
		nil,
	)

	return server, scope.ID
}

func signGitHubWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)

	return gitHubWebhookSignaturePrefix + hex.EncodeToString(mac.Sum(nil))
}
