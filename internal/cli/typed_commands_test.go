package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPICommandSupportsFieldsQueriesHeadersAndJSONSelection(t *testing.T) {
	var method string
	var path string
	var authHeader string
	var customHeader string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.RequestURI()
		authHeader = r.Header.Get("Authorization")
		customHeader = r.Header.Get("X-Test")
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"id":"ticket-1","title":"CLI contract"}}`))
	}))
	defer server.Close()

	command := newAPICommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{
		"--api-url", server.URL + "/api/v1/platform",
		"--token", "ase_agent_test",
		"--header", "X-Test: yes",
		"--query", "status_name=Todo",
		"--query", "priority=high",
		"-f", "title=CLI contract",
		"-f", "attempt=2",
		"POST",
		"/api/v1/tickets/ticket-1/comments",
		"--json", "ticket.id,ticket.title",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	if method != http.MethodPost {
		t.Fatalf("method = %s, want POST", method)
	}
	if path != "/api/v1/tickets/ticket-1/comments?status_name=Todo&priority=high" {
		t.Fatalf("path = %q", path)
	}
	if authHeader != "Bearer ase_agent_test" || customHeader != "yes" {
		t.Fatalf("headers = auth:%q custom:%q", authHeader, customHeader)
	}
	if payload["title"] != "CLI contract" || payload["attempt"] != float64(2) {
		t.Fatalf("payload = %+v", payload)
	}
	if !strings.Contains(stdout.String(), `"ticket.id": "ticket-1"`) || !strings.Contains(stdout.String(), `"ticket.title": "CLI contract"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTicketCommentWorkpadCreatesAndUpdatesSingleComment(t *testing.T) {
	requests := make([]string, 0, 4)
	bodies := make([]map[string]any, 0, 2)
	listResponse := `{"comments":[]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listResponse))
		case r.Method == http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode(create) error = %v", err)
			}
			bodies = append(bodies, payload)
			listResponse = `{"comments":[{"id":"comment-1","body_markdown":"## Codex Workpad\n\nfirst pass","is_deleted":false}]}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"comment":{"id":"comment-1","body_markdown":"## Codex Workpad\n\nfirst pass"}}`))
		case r.Method == http.MethodPatch:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode(update) error = %v", err)
			}
			bodies = append(bodies, payload)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"comment":{"id":"comment-1","body_markdown":"## Codex Workpad\n\nsecond pass"}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	command := newTicketCommentWorkpadCommand()
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "--body", "first pass", "ticket-1"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("create workpad error = %v", err)
	}

	command = newTicketCommentWorkpadCommand()
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "--body", "second pass", "ticket-1"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("update workpad error = %v", err)
	}

	if got, want := strings.Join(requests, " | "), "GET /api/v1/tickets/ticket-1/comments | POST /api/v1/tickets/ticket-1/comments | GET /api/v1/tickets/ticket-1/comments | PATCH /api/v1/tickets/ticket-1/comments/comment-1"; got != want {
		t.Fatalf("requests = %q, want %q", got, want)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected two write payloads, got %d", len(bodies))
	}
	if bodies[0]["body"] != "## Codex Workpad\n\nfirst pass" {
		t.Fatalf("create payload = %+v", bodies[0])
	}
	if bodies[1]["body"] != "## Codex Workpad\n\nsecond pass" {
		t.Fatalf("update payload = %+v", bodies[1])
	}
}

func TestCLIContractSnapshotMatchesCommittedArtifact(t *testing.T) {
	snapshot, err := commandContractSnapshot()
	if err != nil {
		t.Fatalf("commandContractSnapshot() error = %v", err)
	}
	expected, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(snapshot) error = %v", err)
	}
	expected = append(expected, '\n')

	body, err := os.ReadFile(filepath.Join("testdata", "openapi_cli_contract.json"))
	if err != nil {
		t.Fatalf("ReadFile(snapshot) error = %v", err)
	}
	if !bytes.Equal(body, expected) {
		t.Fatalf("CLI contract snapshot drifted; regenerate with `openase openapi cli-contract --output internal/cli/testdata/openapi_cli_contract.json`")
	}
}
