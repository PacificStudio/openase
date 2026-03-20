package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
)

func mustWriteString(t *testing.T, writer io.Writer, value string) {
	t.Helper()

	if _, err := io.WriteString(writer, value); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func closeBody(t *testing.T, closer io.Closer) {
	t.Helper()

	if err := closer.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
}

func TestConnectorPullIssuesMapsIssuesAndSkipsPullRequests(t *testing.T) {
	since := time.Date(2026, 3, 20, 10, 30, 0, 0, time.UTC)
	requests := make([]string, 0, 2)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("Authorization header = %q, want Bearer token-123", got)
		}

		switch {
		case r.URL.Path == "/repos/acme/backend/issues" && r.URL.Query().Get("page") == "":
			if got := r.URL.Query().Get("since"); got != since.Format(time.RFC3339) {
				t.Fatalf("since query = %q, want %q", got, since.Format(time.RFC3339))
			}
			if got := r.URL.Query().Get("state"); got != "all" {
				t.Fatalf("state query = %q, want all", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Link", `<`+server.URL+`/repos/acme/backend/issues?page=2>; rel="next"`)
			mustWriteString(t, w, `[
				{
					"number": 42,
					"html_url": "https://github.com/acme/backend/issues/42",
					"title": " Connector issue ",
					"body": " body ",
					"state": "open",
					"user": {"login": "octocat"},
					"assignees": [{"login": "codex"}],
					"labels": [{"name": "openase"}, {"name": "backend"}],
					"created_at": "2026-03-20T08:00:00Z",
					"updated_at": "2026-03-20T09:00:00Z"
				},
				{
					"number": 43,
					"title": "Pull request masquerading as issue",
					"state": "open",
					"created_at": "2026-03-20T08:00:00Z",
					"updated_at": "2026-03-20T09:00:00Z",
					"pull_request": {}
				}
			]`)
		case r.URL.Path == "/repos/acme/backend/issues" && r.URL.Query().Get("page") == "2":
			w.Header().Set("Content-Type", "application/json")
			mustWriteString(t, w, `[
				{
					"number": 44,
					"html_url": "https://github.com/acme/backend/issues/44",
					"title": "Second page",
					"body": "",
					"state": "closed",
					"user": {"login": "bot"},
					"assignees": [],
					"labels": [{"name": "openase"}],
					"created_at": "2026-03-20T09:00:00Z",
					"updated_at": "2026-03-20T10:00:00Z"
				}
			]`)
		default:
			t.Fatalf("unexpected request path %s", r.URL.String())
		}
	}))
	defer server.Close()

	connector := New(server.Client())
	issues, err := connector.PullIssues(context.Background(), domain.Config{
		Type:       domain.TypeGitHub,
		BaseURL:    server.URL,
		AuthToken:  "token-123",
		ProjectRef: "Acme/Backend",
	}, since)
	if err != nil {
		t.Fatalf("PullIssues returned error: %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("len(issues) = %d, want 2", len(issues))
	}
	if issues[0].ExternalID != "acme/backend#42" || issues[0].Title != "Connector issue" || issues[0].Author != "octocat" {
		t.Fatalf("unexpected first issue: %+v", issues[0])
	}
	if issues[1].ExternalID != "acme/backend#44" || issues[1].Status != "closed" {
		t.Fatalf("unexpected second issue: %+v", issues[1])
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %+v, want 2 calls", requests)
	}
}

func TestConnectorParseWebhookMapsIssueAndCommentEvents(t *testing.T) {
	connector := New(nil)

	event, err := connector.ParseWebhook(context.Background(), http.Header{
		"X-GitHub-Event": []string{"issues"},
	}, []byte(`{
		"action": "labeled",
		"repository": {"full_name": "Acme/Backend"},
		"issue": {
			"number": 42,
			"html_url": "https://github.com/acme/backend/issues/42",
			"title": "Connector issue",
			"body": "Details",
			"state": "open",
			"user": {"login": "octocat"},
			"assignees": [{"login": "codex"}],
			"labels": [{"name": "openase"}],
			"created_at": "2026-03-20T08:00:00Z",
			"updated_at": "2026-03-20T09:00:00Z"
		}
	}`))
	if err != nil {
		t.Fatalf("ParseWebhook issues returned error: %v", err)
	}
	if event.Action != "updated" || event.Issue.ExternalID != "acme/backend#42" {
		t.Fatalf("unexpected issues webhook event: %+v", event)
	}

	commentEvent, err := connector.ParseWebhook(context.Background(), http.Header{
		"X-GitHub-Event": []string{"issue_comment"},
	}, []byte(`{
		"action": "created",
		"repository": {"full_name": "acme/backend"},
		"issue": {
			"number": 42,
			"html_url": "https://github.com/acme/backend/issues/42",
			"title": "Connector issue",
			"body": "Details",
			"state": "open",
			"user": {"login": "octocat"},
			"assignees": [],
			"labels": [],
			"created_at": "2026-03-20T08:00:00Z",
			"updated_at": "2026-03-20T09:00:00Z"
		},
		"comment": {
			"id": 7,
			"body": "Ship it",
			"user": {"login": "reviewer"},
			"created_at": "2026-03-20T09:30:00Z",
			"updated_at": "2026-03-20T09:30:00Z"
		}
	}`))
	if err != nil {
		t.Fatalf("ParseWebhook issue_comment returned error: %v", err)
	}
	if commentEvent.Action != "commented" || commentEvent.Comment == nil || commentEvent.Comment.ExternalID != "7" {
		t.Fatalf("unexpected issue_comment webhook event: %+v", commentEvent)
	}
}

func TestConnectorSyncBackMapsStatusCommentAndLabel(t *testing.T) {
	type recordedRequest struct {
		Method string
		Path   string
		Body   map[string]any
	}

	recorded := make([]recordedRequest, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("Authorization header = %q, want Bearer token-123", got)
		}

		payload := map[string]any{}
		if r.Body != nil {
			defer closeBody(t, r.Body)
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			if len(strings.TrimSpace(string(raw))) > 0 {
				if err := json.Unmarshal(raw, &payload); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
			}
		}
		recorded = append(recorded, recordedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   payload,
		})

		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/backend/issues/42":
			mustWriteString(t, w, `{}`)
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/backend/issues/42/comments":
			w.WriteHeader(http.StatusCreated)
			mustWriteString(t, w, `{}`)
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/backend/issues/42/labels":
			mustWriteString(t, w, `[]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	connector := New(server.Client())
	cfg := domain.Config{
		Type:       domain.TypeGitHub,
		BaseURL:    server.URL,
		AuthToken:  "token-123",
		ProjectRef: "acme/backend",
		StatusMapping: map[string]string{
			"open":   "Todo",
			"closed": "Done",
		},
	}

	if err := connector.SyncBack(context.Background(), cfg, domain.SyncBackUpdate{
		ExternalID: "acme/backend#42",
		Action:     domain.SyncBackActionUpdateStatus,
		Status:     "Done",
	}); err != nil {
		t.Fatalf("SyncBack update_status returned error: %v", err)
	}
	if err := connector.SyncBack(context.Background(), cfg, domain.SyncBackUpdate{
		ExternalID: "acme/backend#42",
		Action:     domain.SyncBackActionAddComment,
		Comment:    "Agent claimed this issue",
	}); err != nil {
		t.Fatalf("SyncBack add_comment returned error: %v", err)
	}
	if err := connector.SyncBack(context.Background(), cfg, domain.SyncBackUpdate{
		ExternalID: "acme/backend#42",
		Action:     domain.SyncBackActionAddLabel,
		Label:      "openase-retrying",
	}); err != nil {
		t.Fatalf("SyncBack add_label returned error: %v", err)
	}

	if len(recorded) != 3 {
		t.Fatalf("recorded requests = %d, want 3", len(recorded))
	}
	if got := recorded[0].Body["state"]; got != "closed" {
		t.Fatalf("status sync body = %+v, want state=closed", recorded[0].Body)
	}
	if got := recorded[1].Body["body"]; got != "Agent claimed this issue" {
		t.Fatalf("comment sync body = %+v", recorded[1].Body)
	}
	labels, ok := recorded[2].Body["labels"].([]any)
	if !ok || len(labels) != 1 || labels[0] != "openase-retrying" {
		t.Fatalf("label sync body = %+v", recorded[2].Body)
	}
}

func TestConnectorHealthCheckHitsRepositoryEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/repos/acme/backend" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		mustWriteString(t, w, `{}`)
	}))
	defer server.Close()

	connector := New(server.Client())
	if err := connector.HealthCheck(context.Background(), domain.Config{
		Type:       domain.TypeGitHub,
		BaseURL:    server.URL,
		ProjectRef: "acme/backend",
	}); err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
}
