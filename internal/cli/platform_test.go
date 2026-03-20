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

func TestNewRootCommandIncludesPlatformCommands(t *testing.T) {
	root := NewRootCommand("dev")

	for _, path := range [][]string{{"ticket"}, {"project"}} {
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

	command := newTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
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

	command := newTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
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

	command := newTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
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

	command := newProjectCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
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
