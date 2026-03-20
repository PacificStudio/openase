package orchestrator

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	githubconnector "github.com/BetterAndBetterII/openase/internal/infra/issueconnector/github"
	registrypkg "github.com/BetterAndBetterII/openase/internal/issueconnector"
	"github.com/google/uuid"
)

func TestConnectorSyncerUsesGitHubConnectorForPullAndSyncBack(t *testing.T) {
	var patchedState string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/backend/issues":
			if _, err := io.WriteString(w, `[
				{
					"number": 42,
					"html_url": "https://github.com/acme/backend/issues/42",
					"title": "Connector syncer integration",
					"body": "body",
					"state": "open",
					"user": {"login": "octocat"},
					"assignees": [{"login": "codex"}],
					"labels": [{"name": "openase"}],
					"created_at": "2026-03-20T08:00:00Z",
					"updated_at": "2026-03-20T09:00:00Z"
				}
			]`); err != nil {
				t.Fatalf("write issues response: %v", err)
			}
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/backend/issues/42":
			defer func() {
				if err := r.Body.Close(); err != nil {
					t.Fatalf("close patch body: %v", err)
				}
			}()
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode patch payload: %v", err)
			}
			patchedState = payload["state"]
			if _, err := io.WriteString(w, `{}`); err != nil {
				t.Fatalf("write patch response: %v", err)
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	connectorID := uuid.New()
	repo := &stubConnectorRepo{
		connectors: map[uuid.UUID]domain.IssueConnector{
			connectorID: {
				ID:        connectorID,
				ProjectID: uuid.New(),
				Type:      domain.TypeGitHub,
				Name:      "GitHub Issues",
				Status:    domain.StatusActive,
				Config: domain.Config{
					Type:          domain.TypeGitHub,
					BaseURL:       server.URL,
					ProjectRef:    "acme/backend",
					PollInterval:  5 * time.Minute,
					SyncDirection: domain.SyncDirectionBidirectional,
					StatusMapping: map[string]string{
						"open":   "Todo",
						"closed": "Done",
					},
				},
			},
		},
	}
	registry, err := registrypkg.NewRegistry(githubconnector.New(server.Client()))
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	sink := &stubConnectorSink{}
	syncer := NewConnectorSyncer(repo, registry, sink, slog.New(slog.NewTextHandler(io.Discard, nil)))

	report, err := syncer.SyncConnector(context.Background(), connectorID)
	if err != nil {
		t.Fatalf("SyncConnector returned error: %v", err)
	}
	if report.ConnectorsSynced != 1 || report.IssuesSynced != 1 {
		t.Fatalf("unexpected pull report: %+v", report)
	}
	if len(sink.syncedIssues) != 1 || sink.syncedIssues[0].ExternalID != "acme/backend#42" {
		t.Fatalf("unexpected synced issues: %+v", sink.syncedIssues)
	}

	err = syncer.SyncBack(context.Background(), SyncBackRequest{
		ConnectorID: connectorID,
		Update: domain.SyncBackUpdate{
			ExternalID: "acme/backend#42",
			Action:     domain.SyncBackActionUpdateStatus,
			Status:     "Done",
		},
	})
	if err != nil {
		t.Fatalf("SyncBack returned error: %v", err)
	}
	if patchedState != "closed" {
		t.Fatalf("patchedState = %q, want closed", patchedState)
	}
}
