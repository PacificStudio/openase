package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestGitHubWebhookHelperCoverage(t *testing.T) {
	t.Run("parse envelope validation and summary", func(t *testing.T) {
		validReviewPayload := []byte(`{
			"action":"submitted",
			"repository":{"clone_url":"https://github.com/acme/openase.git","full_name":"acme/openase"},
			"pull_request":{"number":42,"html_url":"https://github.com/acme/openase/pull/42","state":"open","merged":false,"head":{"ref":"feat/openase-278-coverage"}},
			"review":{"state":"changes_requested"}
		}`)

		for _, testCase := range []struct {
			name    string
			event   gitHubWebhookEvent
			payload []byte
			wantErr string
		}{
			{name: "invalid json", event: gitHubWebhookEventPullRequest, payload: []byte("{"), wantErr: "invalid GitHub webhook payload"},
			{name: "missing action", event: gitHubWebhookEventPullRequest, payload: []byte(`{"repository":{"clone_url":"https://github.com/acme/openase.git"},"pull_request":{"number":42,"head":{"ref":"feat"}}}`), wantErr: "payload action is required"},
			{name: "missing clone url", event: gitHubWebhookEventPullRequest, payload: []byte(`{"action":"opened","repository":{},"pull_request":{"number":42,"head":{"ref":"feat"}}}`), wantErr: "repository.clone_url is required"},
			{name: "missing pull request", event: gitHubWebhookEventPullRequest, payload: []byte(`{"action":"opened","repository":{"clone_url":"https://github.com/acme/openase.git"}}`), wantErr: "pull_request is required"},
			{name: "missing branch", event: gitHubWebhookEventPullRequest, payload: []byte(`{"action":"opened","repository":{"clone_url":"https://github.com/acme/openase.git"},"pull_request":{"number":42,"head":{}}}`), wantErr: "pull_request.head.ref is required"},
			{name: "missing number", event: gitHubWebhookEventPullRequest, payload: []byte(`{"action":"opened","repository":{"clone_url":"https://github.com/acme/openase.git"},"pull_request":{"head":{"ref":"feat"}}}`), wantErr: "payload number is required"},
			{name: "missing review", event: gitHubWebhookEventPullRequestReview, payload: []byte(`{"action":"submitted","repository":{"clone_url":"https://github.com/acme/openase.git"},"pull_request":{"number":42,"head":{"ref":"feat"}}}`), wantErr: "payload review is required"},
			{name: "missing review state", event: gitHubWebhookEventPullRequestReview, payload: []byte(`{"action":"submitted","repository":{"clone_url":"https://github.com/acme/openase.git"},"pull_request":{"number":42,"head":{"ref":"feat"}},"review":{}}`), wantErr: "review.state is required"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				_, err := parseGitHubWebhookEnvelope(testCase.event, "delivery-1", testCase.payload)
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("parseGitHubWebhookEnvelope() error = %v, want substring %q", err, testCase.wantErr)
				}
			})
		}

		delivery, err := parseGitHubWebhookEnvelope(gitHubWebhookEventPullRequestReview, " delivery-2 ", validReviewPayload)
		if err != nil {
			t.Fatalf("parseGitHubWebhookEnvelope(valid review) error = %v", err)
		}
		if delivery.PullRequest.Number != 42 || delivery.Review == nil || delivery.Review.State != "changes_requested" {
			t.Fatalf("unexpected parsed delivery: %+v", delivery)
		}
		summary := delivery.summary()
		if summary.DeliveryID != "delivery-2" || summary.Action != "submitted" {
			t.Fatalf("unexpected summary: %+v", summary)
		}
		if !containsArgPair(summary.LogArgs, "review_state", "changes_requested") {
			t.Fatalf("expected review_state in log args, got %+v", summary.LogArgs)
		}
	})

	t.Run("repo scope sync mapping", func(t *testing.T) {
		pullRequestInput, ok := mapGitHubWebhookRepoScopeSyncInput(gitHubWebhookEnvelope{
			Event:  gitHubWebhookEventPullRequest,
			Action: "ready_for_review",
			Repository: gitHubWebhookRepository{
				CloneURL: "https://github.com/acme/openase.git",
				FullName: "acme/openase",
			},
			PullRequest: gitHubWebhookPullRequest{
				URL:    "https://github.com/acme/openase/pull/42",
				Branch: "feat/openase-278-coverage",
			},
		})
		if !ok || pullRequestInput.PRStatus == "" || pullRequestInput.BranchName == "" {
			t.Fatalf("mapGitHubWebhookRepoScopeSyncInput(pull_request) = %+v, %t", pullRequestInput, ok)
		}

		reviewInput, ok := mapGitHubWebhookRepoScopeSyncInput(gitHubWebhookEnvelope{
			Event: gitHubWebhookEventPullRequestReview,
			Repository: gitHubWebhookRepository{
				CloneURL: "https://github.com/acme/openase.git",
				FullName: "acme/openase",
			},
			PullRequest: gitHubWebhookPullRequest{
				URL:    "https://github.com/acme/openase/pull/42",
				Branch: "feat/openase-278-coverage",
			},
			Review: &gitHubWebhookReview{State: "changes_requested"},
		})
		if !ok || reviewInput.PRStatus == "" {
			t.Fatalf("mapGitHubWebhookRepoScopeSyncInput(review) = %+v, %t", reviewInput, ok)
		}

		if _, ok := mapGitHubWebhookRepoScopeSyncInput(gitHubWebhookEnvelope{
			Event:  gitHubWebhookEventPullRequestReview,
			Review: &gitHubWebhookReview{State: "approved"},
		}); ok {
			t.Fatal("expected approved review not to map to repo scope status")
		}
		if _, ok := mapGitHubWebhookRepoScopeSyncInput(gitHubWebhookEnvelope{
			Event:  gitHubWebhookEventPullRequest,
			Action: "synchronize",
		}); ok {
			t.Fatal("expected unsupported pull request action not to map")
		}
	})
}

func TestHTTPStreamAndTicketAssignedAgentHelpers(t *testing.T) {
	t.Run("sse helpers", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := writeSSEKeepaliveComment(c.Response()); err != nil {
			t.Fatalf("writeSSEKeepaliveComment() error = %v", err)
		}

		event := provider.Event{
			Topic:       agentStepStreamTopic,
			Type:        provider.MustParseEventType(domain.AgentStepEventType),
			Payload:     []byte("{\"entry\":{\n\"step_status\":\"planning\"}}"),
			PublishedAt: time.Date(2026, 3, 27, 22, 45, 0, 0, time.UTC),
		}
		if err := writeSSEEvent(c.Response(), event); err != nil {
			t.Fatalf("writeSSEEvent() error = %v", err)
		}
		if err := writeSSEFrame(c.Response(), "message", map[string]any{"ok": true}); err != nil {
			t.Fatalf("writeSSEFrame() error = %v", err)
		}

		body := rec.Body.String()
		if !strings.Contains(body, ": keepalive\n\n") {
			t.Fatalf("expected keepalive comment, got %q", body)
		}
		if !strings.Contains(body, "event: "+domain.AgentStepEventType+"\n") {
			t.Fatalf("expected agent step event frame, got %q", body)
		}
		if !strings.Contains(body, "data: {\"topic\":\"agent.step.events\"") {
			t.Fatalf("expected SSE envelope payload, got %q", body)
		}
		if !strings.Contains(body, "event: message\n") || !strings.Contains(body, "data: {\"ok\":true}\n\n") {
			t.Fatalf("expected chat SSE frame, got %q", body)
		}
	})

	t.Run("assigned agent helper", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		service := newFakeCatalogService()
		server := NewServer(
			config.ServerConfig{Port: 40023},
			config.GitHubConfig{},
			logger,
			eventinfra.NewChannelBus(),
			nil,
			nil,
			nil,
			service,
			nil,
		)

		if item, err := server.loadTicketAssignedAgent(context.Background(), ticketservice.Ticket{}); err != nil || item != nil {
			t.Fatalf("loadTicketAssignedAgent(nil run) = %+v, %v", item, err)
		}

		missingRunID := uuid.New()
		if _, err := server.loadTicketAssignedAgent(context.Background(), ticketservice.Ticket{CurrentRunID: &missingRunID}); err == nil {
			t.Fatal("expected missing run error")
		}

		projectID := uuid.New()
		providerID := uuid.New()
		agentID := uuid.New()
		runID := uuid.New()
		service.agentRuns[runID] = domain.AgentRun{ID: runID, AgentID: agentID}
		service.agents[agentID] = domain.Agent{
			ID:                  agentID,
			ProjectID:           projectID,
			ProviderID:          providerID,
			Name:                "Worker 1",
			RuntimeControlState: domain.AgentRuntimeControlStateActive,
			Runtime: &domain.AgentRuntime{
				RuntimePhase: domain.AgentRuntimePhaseExecuting,
			},
		}
		service.providers[providerID] = domain.AgentProvider{
			ID:   providerID,
			Name: "Codex",
		}

		item, err := server.loadTicketAssignedAgent(context.Background(), ticketservice.Ticket{CurrentRunID: &runID})
		if err != nil {
			t.Fatalf("loadTicketAssignedAgent(valid) error = %v", err)
		}
		if item == nil || item.ID != agentID.String() || item.Provider != "Codex" {
			t.Fatalf("unexpected assigned agent response: %+v", item)
		}
		if item.RuntimePhase == nil || *item.RuntimePhase != domain.AgentRuntimePhaseExecuting.String() {
			t.Fatalf("expected runtime phase in response, got %+v", item)
		}

		delete(service.providers, providerID)
		if _, err := server.loadTicketAssignedAgent(context.Background(), ticketservice.Ticket{CurrentRunID: &runID}); err == nil {
			t.Fatal("expected missing provider error")
		}
	})
}

func containsArgPair(values []any, key string, want any) bool {
	for i := 0; i+1 < len(values); i += 2 {
		if values[i] == key && values[i+1] == want {
			return true
		}
	}
	return false
}
