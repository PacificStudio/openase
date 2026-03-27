package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
)

func TestConnectorIdentityAndHelperCoverage(t *testing.T) {
	t.Parallel()

	connector := New(nil)
	if connector.ID() != string(domain.TypeGitHub) {
		t.Fatalf("ID() = %q", connector.ID())
	}
	if connector.Name() != "GitHub Issues" {
		t.Fatalf("Name() = %q", connector.Name())
	}

	repoRef, err := parseRepositoryRef(" Acme/Backend ")
	if err != nil {
		t.Fatalf("parseRepositoryRef() error = %v", err)
	}
	if repoRef.String() != "acme/backend" {
		t.Fatalf("repositoryRef.String() = %q", repoRef.String())
	}
	if !repoRef.matches(repositoryRef{Owner: "ACME", Name: "BACKEND"}) {
		t.Fatal("repositoryRef.matches() = false, want true")
	}
	if _, err := parseRepositoryRef("acme"); err == nil || !strings.Contains(err.Error(), "owner/repo") {
		t.Fatalf("parseRepositoryRef(invalid) error = %v", err)
	}

	externalRef, err := parseIssueExternalRef("acme/backend#42")
	if err != nil {
		t.Fatalf("parseIssueExternalRef() error = %v", err)
	}
	if externalRef.Number != 42 || externalRef.Repository.String() != "acme/backend" {
		t.Fatalf("parseIssueExternalRef() = %+v", externalRef)
	}
	if _, err := parseIssueExternalRef("acme/backend#0"); err == nil || !strings.Contains(err.Error(), "positive issue number") {
		t.Fatalf("parseIssueExternalRef(invalid) error = %v", err)
	}

	if got, err := mapIssueWebhookAction("opened"); err != nil || got != "created" {
		t.Fatalf("mapIssueWebhookAction(opened) = (%q, %v)", got, err)
	}
	if got, err := mapIssueWebhookAction("transferred"); err != nil || got != "updated" {
		t.Fatalf("mapIssueWebhookAction(transferred) = (%q, %v)", got, err)
	}
	if _, err := mapIssueWebhookAction("pinned"); err == nil || !strings.Contains(err.Error(), "unsupported GitHub issues action") {
		t.Fatalf("mapIssueWebhookAction(invalid) error = %v", err)
	}

	if got, err := mapSyncBackStatus(nil, "open"); err != nil || got != "open" {
		t.Fatalf("mapSyncBackStatus(open) = (%q, %v)", got, err)
	}
	if got, err := mapSyncBackStatus(map[string]string{"closed": "Done"}, "Done"); err != nil || got != "closed" {
		t.Fatalf("mapSyncBackStatus(mapped) = (%q, %v)", got, err)
	}
	if got, err := mapSyncBackStatus(nil, "Completed"); err != nil || got != "closed" {
		t.Fatalf("mapSyncBackStatus(fallback closed) = (%q, %v)", got, err)
	}
	if got, err := mapSyncBackStatus(nil, "In Progress"); err != nil || got != "open" {
		t.Fatalf("mapSyncBackStatus(fallback open) = (%q, %v)", got, err)
	}
	if _, err := mapSyncBackStatus(map[string]string{"open": "Done", "closed": "Done"}, "Done"); err == nil || !strings.Contains(err.Error(), "multiple GitHub states") {
		t.Fatalf("mapSyncBackStatus(ambiguous) error = %v", err)
	}
	if _, err := mapSyncBackStatus(map[string]string{"draft": "Todo"}, "Todo"); err == nil || !strings.Contains(err.Error(), "only supports open/closed") {
		t.Fatalf("mapSyncBackStatus(unsupported external) error = %v", err)
	}
	if _, err := mapSyncBackStatus(nil, "   "); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("mapSyncBackStatus(empty) error = %v", err)
	}

	if got, err := buildIssuesURL("", repoRef, time.Time{}); err != nil || !strings.HasPrefix(got, defaultBaseURL+"/repos/acme/backend/issues?") {
		t.Fatalf("buildIssuesURL(default base) = (%q, %v)", got, err)
	}
	if got, err := buildRepositoryEndpoint("https://api.github.com/", repoRef, "issues", "42", "labels"); err != nil || got != "https://api.github.com/repos/acme/backend/issues/42/labels" {
		t.Fatalf("buildRepositoryEndpoint() = (%q, %v)", got, err)
	}
	if got, err := normalizeBaseURL(" https://github.example/api/ "); err != nil || got != "https://github.example/api" {
		t.Fatalf("normalizeBaseURL() = (%q, %v)", got, err)
	}
	if _, err := normalizeBaseURL("relative/path"); err == nil || !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("normalizeBaseURL(relative) error = %v", err)
	}

	if got := parseNextLink(`<https://example.com/2>; rel="next", <https://example.com/1>; rel="prev"`); got != "https://example.com/2" {
		t.Fatalf("parseNextLink() = %q", got)
	}
	if got := parseNextLink(`<https://example.com/1>; rel="prev"`); got != "" {
		t.Fatalf("parseNextLink(prev only) = %q", got)
	}
	headers := http.Header{"x-github-event": []string{"issues"}}
	if got := headerValue(headers, webhookEventHeader); got != "issues" {
		t.Fatalf("headerValue() = %q", got)
	}

	if got := collectLabels([]githubLabel{{Name: " bug "}, {Name: "   "}}); len(got) != 1 || got[0] != "bug" {
		t.Fatalf("collectLabels() = %#v", got)
	}
	if got := collectAssignees([]githubUser{{Login: " codex "}, {Login: " "}}); len(got) != 1 || got[0] != "codex" {
		t.Fatalf("collectAssignees() = %#v", got)
	}
	if got := loginOf(nil); got != "" {
		t.Fatalf("loginOf(nil) = %q", got)
	}
}

func TestConnectorWebhookAndHTTPErrorCoverage(t *testing.T) {
	t.Parallel()

	connector := New(http.DefaultClient)
	if _, err := connector.ParseWebhook(context.Background(), http.Header{}, []byte(`{}`)); err == nil || !strings.Contains(err.Error(), "unsupported GitHub webhook event") {
		t.Fatalf("ParseWebhook(unsupported) error = %v", err)
	}
	if _, err := parseIssueCommentWebhook([]byte(`{"action":"created","repository":{"full_name":"acme/backend"},"issue":{"number":1}}`)); err == nil || !strings.Contains(err.Error(), "comment is required") {
		t.Fatalf("parseIssueCommentWebhook(missing comment) error = %v", err)
	}
	event, err := parseIssueCommentWebhook([]byte(`{
		"action":"deleted",
		"repository":{"full_name":"acme/backend"},
		"issue":{
			"number":1,
			"html_url":"https://github.com/acme/backend/issues/1",
			"title":"Ticket",
			"body":"body",
			"state":"open",
			"created_at":"2026-03-20T08:00:00Z",
			"updated_at":"2026-03-20T09:00:00Z"
		}
	}`))
	if err != nil {
		t.Fatalf("parseIssueCommentWebhook(deleted) error = %v", err)
	}
	if event.Action != "updated" || event.Comment != nil {
		t.Fatalf("parseIssueCommentWebhook(deleted) = %+v", event)
	}

	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer errorServer.Close()
	if _, err := connector.doJSON(context.Background(), http.MethodGet, errorServer.URL, "", nil, http.StatusOK, nil); err == nil || !strings.Contains(err.Error(), "unexpected status 502") {
		t.Fatalf("doJSON(unexpected status) error = %v", err)
	}

	decodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{"))
	}))
	defer decodeServer.Close()
	var payload map[string]any
	if _, err := connector.doJSON(context.Background(), http.MethodGet, decodeServer.URL, "", nil, http.StatusOK, &payload); err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("doJSON(decode) error = %v", err)
	}

	if _, err := newJSONRequest(context.Background(), http.MethodPost, "://bad-url", "", nil); err == nil || !strings.Contains(err.Error(), "build request") {
		t.Fatalf("newJSONRequest(invalid url) error = %v", err)
	}
	if _, err := newJSONRequest(context.Background(), http.MethodPost, "https://example.com", "", map[string]any{"bad": func() {}}); err == nil || !strings.Contains(err.Error(), "marshal request payload") {
		t.Fatalf("newJSONRequest(unmarshalable payload) error = %v", err)
	}
}

type roundTripErrorClient struct{}

func TestConnectorDoJSONTransportError(t *testing.T) {
	t.Parallel()

	connector := New(&http.Client{Transport: roundTripErrorClient{}})
	if _, err := connector.doJSON(context.Background(), http.MethodGet, "https://example.com", "", nil, http.StatusOK, nil); err == nil || !strings.Contains(err.Error(), "synthetic transport failure") {
		t.Fatalf("doJSON(transport) error = %v", err)
	}
}

func (roundTripErrorClient) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("synthetic transport failure")
}
