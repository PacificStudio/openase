package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRootCommandIncludesTypedAPICommands(t *testing.T) {
	root := NewRootCommand("dev")

	for _, path := range [][]string{
		{"api"},
		{"ticket"},
		{"project"},
		{"workflow"},
		{"scheduled-job"},
		{"machine"},
		{"provider"},
		{"agent"},
		{"skill"},
		{"watch"},
		{"stream"},
	} {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", path, err)
		}
		if command == nil {
			t.Fatalf("expected command %v", path)
		}
	}
}

func TestTicketCreateCommandUsesAgentPlatformEnvironment(t *testing.T) {
	var method string
	var authHeader string
	var path string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		authHeader = r.Header.Get("Authorization")
		path = r.URL.RequestURI()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-1","title":"Follow-up"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{"create", "--title", "Follow-up", "--description", "split out tests", "--priority", "high"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodPost {
		t.Fatalf("expected POST, got %s", method)
	}
	if authHeader != "Bearer ase_agent_test" {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	if path != "/projects/project-123/tickets" {
		t.Fatalf("expected ticket create path, got %q", path)
	}
	if payload["title"] != "Follow-up" || payload["description"] != "split out tests" || payload["priority"] != "high" {
		t.Fatalf("unexpected request payload: %+v", payload)
	}
	if !strings.Contains(stdout.String(), `"title": "Follow-up"`) {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}

func TestPlatformTicketUpdateHelpMentionsEnvFallbackAndUUIDSemantics(t *testing.T) {
	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	command.SetArgs([]string{"update", "--help"})

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"OPENASE_TICKET_ID",
		"At least one update field must be provided.",
		"ASE-2 are not accepted",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestPlatformWorkpadHelpMentionsUpsertAndBodyRules(t *testing.T) {
	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	command.SetArgs([]string{"comment", "workpad", "--help"})

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"idempotent upsert",
		"Exactly one of --body or --body-file should be used",
		"OPENASE_TICKET_ID",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestPlatformAddRepoHelpMentionsProjectFallback(t *testing.T) {
	command := newAgentPlatformProjectCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	command.SetArgs([]string{"add-repo", "--help"})

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"OPENASE_PROJECT_ID",
		"Register a repository in the current project.",
		"openase project add-repo --name backend",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestTicketUpdateCommandFallsBackToCurrentTicketEnv(t *testing.T) {
	var path string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.RequestURI()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-9","description":"updated"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"update", "--description", "updated"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if path != "/tickets/ticket-9" {
		t.Fatalf("expected env-backed ticket path, got %q", path)
	}
	if payload["description"] != "updated" {
		t.Fatalf("unexpected update payload: %+v", payload)
	}
}

func TestTicketUpdateCommandAcceptsStatusName(t *testing.T) {
	var path string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.RequestURI()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-9","status_name":"Done"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"update", "--status", "Done"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if path != "/tickets/ticket-9" {
		t.Fatalf("expected env-backed ticket path, got %q", path)
	}
	if payload["status_name"] != "Done" {
		t.Fatalf("unexpected update payload: %+v", payload)
	}
}

func TestTicketCommentWorkpadCommandCreatesCommentWhenMissing(t *testing.T) {
	requests := make([]string, 0, 2)
	var postPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.RequestURI() == "/tickets/ticket-9/comments":
			_, _ = w.Write([]byte(`{"comments":[]}`))
		case r.Method == http.MethodPost && r.URL.RequestURI() == "/tickets/ticket-9/comments":
			if err := json.NewDecoder(r.Body).Decode(&postPayload); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			_, _ = w.Write([]byte(`{"comment":{"id":"comment-1","body":"## Codex Workpad\n\nProgress\n- started"}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"comment", "workpad", "--body", "Progress\n- started"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	wantRequests := []string{
		"GET /tickets/ticket-9/comments",
		"POST /tickets/ticket-9/comments",
	}
	if strings.Join(requests, "|") != strings.Join(wantRequests, "|") {
		t.Fatalf("request sequence = %v, want %v", requests, wantRequests)
	}
	if postPayload["body"] != "## Codex Workpad\n\nProgress\n- started" {
		t.Fatalf("unexpected workpad create payload: %+v", postPayload)
	}
}

func TestTicketCommentWorkpadCommandUpdatesExistingComment(t *testing.T) {
	requests := make([]string, 0, 2)
	var patchPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.RequestURI() == "/tickets/ticket-9/comments":
			_, _ = w.Write([]byte(`{"comments":[{"id":"comment-7","body":"## Codex Workpad\n\nOld"}]}`))
		case r.Method == http.MethodPatch && r.URL.RequestURI() == "/tickets/ticket-9/comments/comment-7":
			if err := json.NewDecoder(r.Body).Decode(&patchPayload); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			_, _ = w.Write([]byte(`{"comment":{"id":"comment-7","body":"## Codex Workpad\n\nValidation\n- npm test"}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"comment", "workpad", "--body", "Validation\n- npm test"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	wantRequests := []string{
		"GET /tickets/ticket-9/comments",
		"PATCH /tickets/ticket-9/comments/comment-7",
	}
	if strings.Join(requests, "|") != strings.Join(wantRequests, "|") {
		t.Fatalf("request sequence = %v, want %v", requests, wantRequests)
	}
	if patchPayload["body"] != "## Codex Workpad\n\nValidation\n- npm test" {
		t.Fatalf("unexpected workpad patch payload: %+v", patchPayload)
	}
}

func TestTicketReportUsageCommandPostsUsagePayload(t *testing.T) {
	var method string
	var path string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.RequestURI()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-9","cost_tokens_input":120,"cost_tokens_output":45,"cost_amount":0.21},"applied":{"input_tokens":120,"output_tokens":45,"cost_usd":0.21},"budget_exceeded":true}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"report-usage", "--input-tokens", "120", "--output-tokens", "45", "--cost-usd", "0.21"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodPost {
		t.Fatalf("expected POST, got %s", method)
	}
	if path != "/tickets/ticket-9/usage" {
		t.Fatalf("expected ticket usage path, got %q", path)
	}
	if payload["input_tokens"] != float64(120) || payload["output_tokens"] != float64(45) || payload["cost_usd"] != 0.21 {
		t.Fatalf("unexpected usage payload: %+v", payload)
	}
}

func TestProjectAddRepoCommandPostsRepoPayload(t *testing.T) {
	var method string
	var path string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.RequestURI()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repo":{"id":"repo-1","name":"worker-tools"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")

	command := newAgentPlatformProjectCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"add-repo", "--name", "worker-tools", "--url", "https://github.com/acme/worker-tools.git", "--label", "go", "--label", "backend"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodPost {
		t.Fatalf("expected POST, got %s", method)
	}
	if path != "/projects/project-123/repos" {
		t.Fatalf("expected project repo path, got %q", path)
	}
	if payload["name"] != "worker-tools" || payload["repository_url"] != "https://github.com/acme/worker-tools.git" {
		t.Fatalf("unexpected repo payload: %+v", payload)
	}
	labels, ok := payload["labels"].([]any)
	if !ok || len(labels) != 2 {
		t.Fatalf("expected two labels, got %+v", payload["labels"])
	}
}

func TestTicketListCommandBuildsQueryFromFilters(t *testing.T) {
	var method string
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[{"id":"ticket-1","title":"Follow-up"}]}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{"list", "--status-name", "Todo", "--status-name", "In Review", "--priority", "high", "--priority", "medium"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodGet {
		t.Fatalf("expected GET, got %s", method)
	}
	if path != "/projects/project-123/tickets?priority=high%2Cmedium&status_name=Todo%2CIn+Review" {
		t.Fatalf("expected filtered ticket list path, got %q", path)
	}
	if !strings.Contains(stdout.String(), `"title": "Follow-up"`) {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}

func TestProjectUpdateCommandPatchesDescription(t *testing.T) {
	var method string
	var path string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.RequestURI()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"project":{"id":"project-123","description":"Coverage raised"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")

	command := newAgentPlatformProjectCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{"update", "--description", "Coverage raised"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", method)
	}
	if path != "/projects/project-123" {
		t.Fatalf("expected project patch path, got %q", path)
	}
	if payload["description"] != "Coverage raised" {
		t.Fatalf("unexpected project update payload: %+v", payload)
	}
	if !strings.Contains(stdout.String(), `"description": "Coverage raised"`) {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}
