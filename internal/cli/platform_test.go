package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestRootStatusAndWorkflowHelpMentionAgentPlatformDefaults(t *testing.T) {
	root := NewRootCommand("dev")

	for _, tc := range []struct {
		path []string
		want []string
	}{
		{
			path: []string{"status"},
			want: []string{
				"agent platform API",
				"OPENASE_PROJECT_ID",
				"OPENASE_API_URL",
			},
		},
		{
			path: []string{"workflow"},
			want: []string{
				"agent platform API",
				"OPENASE_PROJECT_ID",
				"workflow harness",
			},
		},
	} {
		command, _, err := root.Find(tc.path)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", tc.path, err)
		}
		if command == nil {
			t.Fatalf("expected command %v", tc.path)
		}

		var stdout bytes.Buffer
		command.SetOut(&stdout)
		command.SetErr(&stdout)
		if err := command.Help(); err != nil {
			t.Fatalf("Help(%v) returned error: %v", tc.path, err)
		}

		output := stdout.String()
		for _, want := range tc.want {
			if !strings.Contains(output, want) {
				t.Fatalf("expected %v help to contain %q, got %q", tc.path, want, output)
			}
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
	command.SetArgs([]string{
		"create",
		"--title", "Follow-up",
		"--description", "split out tests",
		"--status-id", "550e8400-e29b-41d4-a716-446655440000",
		"--priority", "high",
		"--type", "bugfix",
		"--workflow-id", "550e8400-e29b-41d4-a716-446655440001",
		"--parent-ticket-id", "550e8400-e29b-41d4-a716-446655440002",
		"--external-ref", "GH-42",
		"--budget-usd", "12.5",
		"--archived=true",
	})

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
	for key, want := range map[string]any{
		"title":            "Follow-up",
		"description":      "split out tests",
		"status_id":        "550e8400-e29b-41d4-a716-446655440000",
		"priority":         "high",
		"type":             "bugfix",
		"workflow_id":      "550e8400-e29b-41d4-a716-446655440001",
		"parent_ticket_id": "550e8400-e29b-41d4-a716-446655440002",
		"external_ref":     "GH-42",
		"budget_usd":       12.5,
		"archived":         true,
	} {
		if payload[key] != want {
			t.Fatalf("expected payload[%q] = %#v, got %#v in %+v", key, want, payload[key], payload)
		}
	}
	if !strings.Contains(stdout.String(), `"title": "Follow-up"`) {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}

func TestRootStatusListUsesPlatformBaseURLProjectFallbackAndAgentToken(t *testing.T) {
	var method string
	var authHeader string
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		authHeader = r.Header.Get("Authorization")
		path = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"statuses":[{"id":"status-1","name":"Todo"}]}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")

	root := NewRootCommand("dev")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"status", "list"})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodGet {
		t.Fatalf("expected GET, got %s", method)
	}
	if authHeader != "Bearer ase_agent_test" {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	if path != "/api/v1/platform/projects/project-123/statuses" {
		t.Fatalf("expected platform status path, got %q", path)
	}
	if !strings.Contains(stdout.String(), `"name": "Todo"`) {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}

func TestRootWorkflowHarnessVariablesUsesPlatformBaseURLAndAgentToken(t *testing.T) {
	var method string
	var authHeader string
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		authHeader = r.Header.Get("Authorization")
		path = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[{"name":"workflow","variables":[{"name":"ticket.id"}]}]}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")

	root := NewRootCommand("dev")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"workflow", "harness", "variables"})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodGet {
		t.Fatalf("expected GET, got %s", method)
	}
	if authHeader != "Bearer ase_agent_test" {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	if path != "/api/v1/platform/harness/variables" {
		t.Fatalf("expected platform harness variables path, got %q", path)
	}
	if !strings.Contains(stdout.String(), `"name": "workflow"`) {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}

func TestRootStatusWrapperPreservesAPIErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"status_forbidden","message":"nope"}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")

	root := NewRootCommand("dev")
	root.SetArgs([]string{"status", "delete", "status-123"})

	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var httpErr *apiHTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected apiHTTPError, got %T: %v", err, err)
	}
	if httpErr.Code != "status_forbidden" {
		t.Fatalf("expected api error code, got %+v", httpErr)
	}
	if !strings.Contains(err.Error(), "403 Forbidden") || !strings.Contains(err.Error(), "[status_forbidden]") {
		t.Fatalf("expected platform error details, got %q", err.Error())
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

func TestPlatformTicketCommentCommandExposesPrimitiveSubcommandsOnly(t *testing.T) {
	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	commentCommand, _, err := command.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("Find(comment) returned error: %v", err)
	}

	names := make([]string, 0, len(commentCommand.Commands()))
	for _, child := range commentCommand.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "create,list,update" {
		t.Fatalf("comment subcommands = %v, want [create list update]", names)
	}
}

func TestPlatformTicketCommentUpdateHelpMentionsBodyRules(t *testing.T) {
	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	command.SetArgs([]string{"comment", "update", "--help"})

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"If two positional arguments are provided, the first is treated as ticket-id and the second as comment-id.",
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
	t.Setenv("OPENASE_PROJECT_ID", "project-123")
	t.Setenv("OPENASE_AGENT_SCOPES", "tickets.update")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"update", "--description", "updated"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if path != "/projects/project-123/tickets/ticket-9" {
		t.Fatalf("expected project-scoped ticket path, got %q", path)
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
	t.Setenv("OPENASE_PROJECT_ID", "project-123")
	t.Setenv("OPENASE_AGENT_SCOPES", "tickets.update")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"update", "--status_name", "Done"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if path != "/projects/project-123/tickets/ticket-9" {
		t.Fatalf("expected project-scoped ticket path, got %q", path)
	}
	if payload["status_name"] != "Done" {
		t.Fatalf("unexpected update payload: %+v", payload)
	}
}

func TestTicketUpdateCommandSupportsExpandedPatchSurface(t *testing.T) {
	var path string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.RequestURI()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-9","priority":"high"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")
	t.Setenv("OPENASE_AGENT_SCOPES", "tickets.update")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{
		"update",
		"--priority", "high",
		"--type", "chore",
		"--workflow-id", "550e8400-e29b-41d4-a716-446655440000",
		"--parent-ticket-id", "550e8400-e29b-41d4-a716-446655440001",
		"--budget-usd", "42.5",
		"--archived=true",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if path != "/projects/project-123/tickets/ticket-9" {
		t.Fatalf("expected project-scoped ticket path, got %q", path)
	}
	for key, want := range map[string]any{
		"priority":         "high",
		"type":             "chore",
		"workflow_id":      "550e8400-e29b-41d4-a716-446655440000",
		"parent_ticket_id": "550e8400-e29b-41d4-a716-446655440001",
		"budget_usd":       42.5,
		"archived":         true,
	} {
		if payload[key] != want {
			t.Fatalf("expected payload[%q] = %#v, got %#v in %+v", key, want, payload[key], payload)
		}
	}
}

func TestTicketCommentUpdateCommandPatchesCurrentTicketComment(t *testing.T) {
	var patchPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPatch || r.URL.RequestURI() != "/tickets/ticket-9/comments/comment-7" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
		if err := json.NewDecoder(r.Body).Decode(&patchPayload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		_, _ = w.Write([]byte(`{"comment":{"id":"comment-7","body":"Updated progress"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_TICKET_ID", "ticket-9")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"comment", "update", "comment-7", "--body", "Updated progress"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if patchPayload["body"] != "Updated progress" {
		t.Fatalf("unexpected comment patch payload: %+v", patchPayload)
	}
}

func TestTicketCommentUpdateCommandAcceptsTicketIDPositionalAndBodyFileAlias(t *testing.T) {
	var patchPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPatch || r.URL.RequestURI() != "/tickets/ticket-9/comments/comment-7" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
		if err := json.NewDecoder(r.Body).Decode(&patchPayload); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		_, _ = w.Write([]byte(`{"comment":{"id":"comment-7","body":"Updated progress from file"}}`))
	}))
	defer server.Close()

	bodyPath := filepath.Join(t.TempDir(), "comment.md")
	if err := os.WriteFile(bodyPath, []byte("Updated progress from file"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"comment", "update", "ticket-9", "comment-7", "--body_file", bodyPath})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if patchPayload["body"] != "Updated progress from file" {
		t.Fatalf("unexpected comment patch payload: %+v", patchPayload)
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
	command.SetArgs([]string{"add-repo", "--name", "worker-tools", "--url", "https://github.com/acme/worker-tools.git", "--workspace-dirname", "services/worker-tools", "--label", "go", "--label", "backend"})

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
	if payload["workspace_dirname"] != "services/worker-tools" {
		t.Fatalf("unexpected workspace dirname payload: %+v", payload)
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

func TestTicketUpdateCommandUsesCurrentTicketRouteWithoutProjectScope(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-123","status_name":"In Progress"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")
	t.Setenv("OPENASE_TICKET_ID", "ticket-123")
	t.Setenv("OPENASE_AGENT_SCOPES", "tickets.create,tickets.list,tickets.update.self")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"update", "--status-name", "In Progress"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", method)
	}
	if path != "/tickets/ticket-123" {
		t.Fatalf("expected current-ticket update path, got %q", path)
	}
	if payload["status_name"] != "In Progress" {
		t.Fatalf("unexpected ticket update payload: %+v", payload)
	}
}

func TestTicketUpdateCommandUsesProjectScopedRouteWhenTicketsUpdateScopePresent(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-456","status_name":"Done"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL)
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", "project-123")
	t.Setenv("OPENASE_AGENT_SCOPES", "tickets.create,tickets.list,tickets.update")

	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: server.Client()})
	command.SetArgs([]string{"update", "ticket-456", "--status-name", "Done"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", method)
	}
	if path != "/projects/project-123/tickets/ticket-456" {
		t.Fatalf("expected project-scoped update path, got %q", path)
	}
	if payload["status_name"] != "Done" {
		t.Fatalf("unexpected ticket update payload: %+v", payload)
	}
}

func TestProjectUpdateCommandSupportsFullPatchSurface(t *testing.T) {
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
	command.SetArgs([]string{
		"update",
		"--name", "OpenASE Automation",
		"--slug", "openase-automation",
		"--description", "Coverage raised",
		"--status", "In Progress",
		"--default-agent-provider-id", "provider-123",
		"--project-ai-platform-access-allowed", "tickets.list,tickets.update",
		"--accessible-machine-ids", "machine-a,machine-b",
		"--max-concurrent-agents", "4",
		"--agent-run-summary-prompt", "Summarize blockers first.",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if method != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", method)
	}
	if path != "/projects/project-123" {
		t.Fatalf("expected project patch path, got %q", path)
	}
	if payload["name"] != "OpenASE Automation" ||
		payload["slug"] != "openase-automation" ||
		payload["description"] != "Coverage raised" ||
		payload["status"] != "In Progress" ||
		payload["default_agent_provider_id"] != "provider-123" ||
		payload["max_concurrent_agents"] != float64(4) ||
		payload["agent_run_summary_prompt"] != "Summarize blockers first." {
		t.Fatalf("unexpected project update payload: %+v", payload)
	}
	projectAIScopes, ok := payload["project_ai_platform_access_allowed"].([]any)
	if !ok || len(projectAIScopes) != 2 || projectAIScopes[0] != "tickets.list" || projectAIScopes[1] != "tickets.update" {
		t.Fatalf("unexpected project_ai_platform_access_allowed payload: %+v", payload["project_ai_platform_access_allowed"])
	}
	accessibleMachineIDs, ok := payload["accessible_machine_ids"].([]any)
	if !ok || len(accessibleMachineIDs) != 2 || accessibleMachineIDs[0] != "machine-a" || accessibleMachineIDs[1] != "machine-b" {
		t.Fatalf("unexpected accessible_machine_ids payload: %+v", payload["accessible_machine_ids"])
	}
	if !strings.Contains(stdout.String(), `"description": "Coverage raised"`) {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}
