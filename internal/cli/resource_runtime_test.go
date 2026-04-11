package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTypedProjectGetNormalizesPlatformAPIBase(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	var requestPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"project":{"id":"` + projectID + `","organization_id":"8c44cdd8-02d2-4bc9-aefa-e8c5ca7dd87e"}}`))
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")

	root := NewRootCommand("dev")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"project", "get", projectID})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if requestPath != "/api/v1/platform/projects/"+projectID {
		t.Fatalf("project get request path = %q, want %q", requestPath, "/api/v1/platform/projects/"+projectID)
	}
}

func TestMachineListBridgesProjectContextToOrganization(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	orgID := "8c44cdd8-02d2-4bc9-aefa-e8c5ca7dd87e"
	requests := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/projects/" + projectID:
			_, _ = w.Write([]byte(`{"project":{"id":"` + projectID + `","organization_id":"` + orgID + `"}}`))
		case "/api/v1/orgs/" + orgID + "/machines":
			_, _ = w.Write([]byte(`{"machines":[{"id":"machine-1","organization_id":"` + orgID + `"}]}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.RequestURI())
		}
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", projectID)

	root := NewRootCommand("dev")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"machine", "list"})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2 (%v)", len(requests), requests)
	}
	if requests[0] != "/api/v1/projects/"+projectID {
		t.Fatalf("first request = %q, want %q", requests[0], "/api/v1/projects/"+projectID)
	}
	if requests[1] != "/api/v1/orgs/"+orgID+"/machines" {
		t.Fatalf("second request = %q, want %q", requests[1], "/api/v1/orgs/"+orgID+"/machines")
	}
}

func TestProjectUpdatesCRUDUsesRuntimeDefaults(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	threadID := "8c44cdd8-02d2-4bc9-aefa-e8c5ca7dd87e"

	type requestRecord struct {
		Method  string
		Path    string
		Payload map[string]any
	}

	requests := make([]requestRecord, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := requestRecord{Method: r.Method, Path: r.URL.RequestURI()}
		if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPatch) {
			if err := json.NewDecoder(r.Body).Decode(&record.Payload); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
		}
		requests = append(requests, record)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/updates":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/updates":
			_, _ = w.Write([]byte(`{"thread":{"id":"` + threadID + `","status":"on_track","title":"CLI parity","body":"Created"}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/updates/"+threadID:
			_, _ = w.Write([]byte(`{"thread":{"id":"` + threadID + `","status":"at_risk","title":"CLI parity","body":"Updated"}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/updates/"+threadID:
			_, _ = w.Write([]byte(`{"thread":{"id":"` + threadID + `"}}`))
		default:
			t.Fatalf("unexpected request %s %q", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", projectID)

	run := func(args ...string) {
		t.Helper()
		root := NewRootCommand("dev")
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("ExecuteContext(%v) returned error: %v", args, err)
		}
	}

	run("project", "updates", "list")
	run("project", "updates", "create", "--status", "on_track", "--body", "Created")
	run("project", "updates", "update", "--thread-id", threadID, "--status", "at_risk", "--body", "Updated")
	run("project", "updates", "delete", "--thread-id", threadID)

	if len(requests) != 4 {
		t.Fatalf("request count = %d, want 4 (%+v)", len(requests), requests)
	}
	if requests[0].Path != "/api/v1/platform/projects/"+projectID+"/updates" {
		t.Fatalf("list path = %q", requests[0].Path)
	}
	if requests[1].Method != http.MethodPost || requests[1].Path != "/api/v1/platform/projects/"+projectID+"/updates" {
		t.Fatalf("create request = %+v", requests[1])
	}
	if requests[1].Payload["status"] != "on_track" || requests[1].Payload["body"] != "Created" {
		t.Fatalf("create payload = %+v", requests[1].Payload)
	}
	if _, ok := requests[1].Payload["title"]; ok {
		t.Fatalf("create payload unexpectedly included title: %+v", requests[1].Payload)
	}
	if requests[2].Method != http.MethodPatch || requests[2].Path != "/api/v1/platform/projects/"+projectID+"/updates/"+threadID {
		t.Fatalf("update request = %+v", requests[2])
	}
	if requests[2].Payload["status"] != "at_risk" || requests[2].Payload["body"] != "Updated" {
		t.Fatalf("update payload = %+v", requests[2].Payload)
	}
	if _, ok := requests[2].Payload["title"]; ok {
		t.Fatalf("update payload unexpectedly included title: %+v", requests[2].Payload)
	}
	if requests[3].Method != http.MethodDelete || requests[3].Path != "/api/v1/platform/projects/"+projectID+"/updates/"+threadID {
		t.Fatalf("delete request = %+v", requests[3])
	}
}

func TestTicketDependencyMutationsUsePlatformRuntimeDefaults(t *testing.T) {
	ticketID := "550e8400-e29b-41d4-a716-446655440000"
	targetTicketID := "8c44cdd8-02d2-4bc9-aefa-e8c5ca7dd87e"
	dependencyID := "0a5e6e2a-15c4-4f7b-b8fa-7e7d145f352f"

	type requestRecord struct {
		Method  string
		Path    string
		Payload map[string]any
	}

	requests := make([]requestRecord, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := requestRecord{Method: r.Method, Path: r.URL.RequestURI()}
		if r.Body != nil && r.Method == http.MethodPost && r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&record.Payload); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
		}
		requests = append(requests, record)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/platform/tickets/"+ticketID+"/dependencies":
			_, _ = w.Write([]byte(`{"dependency":{"id":"` + dependencyID + `","type":"blocks","target":{"id":"` + targetTicketID + `"}}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/platform/tickets/"+ticketID+"/dependencies/"+dependencyID:
			_, _ = w.Write([]byte(`{"deleted_dependency_id":"` + dependencyID + `"}`))
		default:
			t.Fatalf("unexpected request %s %q", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")

	run := func(args ...string) {
		t.Helper()
		root := NewRootCommand("dev")
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("ExecuteContext(%v) returned error: %v", args, err)
		}
	}

	run("ticket", "dependency", "add", ticketID, "--type", "blocks", "--target-ticket-id", targetTicketID)
	run("ticket", "dependency", "delete", ticketID, dependencyID)

	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2 (%+v)", len(requests), requests)
	}
	if requests[0].Method != http.MethodPost || requests[0].Path != "/api/v1/platform/tickets/"+ticketID+"/dependencies" {
		t.Fatalf("add request = %+v", requests[0])
	}
	if requests[0].Payload["type"] != "blocks" || requests[0].Payload["target_ticket_id"] != targetTicketID {
		t.Fatalf("add payload = %+v", requests[0].Payload)
	}
	if requests[1].Method != http.MethodDelete || requests[1].Path != "/api/v1/platform/tickets/"+ticketID+"/dependencies/"+dependencyID {
		t.Fatalf("delete request = %+v", requests[1])
	}
}

func TestTicketResourceCommandsUsePlatformRuntimeDefaults(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	ticketID := "8c44cdd8-02d2-4bc9-aefa-e8c5ca7dd87e"
	runID := "0a5e6e2a-15c4-4f7b-b8fa-7e7d145f352f"
	externalLinkID := "cf6e6b6a-b04f-4f96-8dbe-0d741f4944b0"

	type requestRecord struct {
		Method  string
		Path    string
		Payload map[string]any
	}

	requests := make([]requestRecord, 0, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := requestRecord{Method: r.Method, Path: r.URL.RequestURI()}
		if r.Body != nil && r.Method == http.MethodPost && r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&record.Payload); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
		}
		requests = append(requests, record)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/tickets/archived":
			_, _ = w.Write([]byte(`{"tickets":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/tickets/"+ticketID:
			_, _ = w.Write([]byte(`{"ticket":{"id":"` + ticketID + `"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/tickets/"+ticketID+"/detail":
			_, _ = w.Write([]byte(`{"ticket":{"id":"` + ticketID + `"},"repo_scopes":[],"comments":[],"timeline":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/platform/tickets/"+ticketID+"/retry/resume":
			_, _ = w.Write([]byte(`{"ticket":{"id":"` + ticketID + `"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/platform/tickets/"+ticketID+"/external-links":
			_, _ = w.Write([]byte(`{"external_link":{"id":"` + externalLinkID + `","url":"https://example.com/issues/123","external_id":"123","title":"Issue 123"}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/platform/tickets/"+ticketID+"/external-links/"+externalLinkID:
			_, _ = w.Write([]byte(`{"deleted_external_link_id":"` + externalLinkID + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/tickets/"+ticketID+"/runs":
			_, _ = w.Write([]byte(`{"runs":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/projects/"+projectID+"/tickets/"+ticketID+"/runs/"+runID:
			_, _ = w.Write([]byte(`{"run":{"id":"` + runID + `"},"transcript_page":{"entries":[]}}`))
		default:
			t.Fatalf("unexpected request %s %q", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", projectID)
	t.Setenv("OPENASE_TICKET_ID", ticketID)

	run := func(args ...string) {
		t.Helper()
		root := NewRootCommand("dev")
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("ExecuteContext(%v) returned error: %v", args, err)
		}
	}

	run("ticket", "archived")
	run("ticket", "get", ticketID)
	run("ticket", "detail", projectID, ticketID)
	run("ticket", "retry-resume", ticketID)
	run("ticket", "external-link", "add", ticketID, "--url", "https://example.com/issues/123", "--external-id", "123", "--title", "Issue 123")
	run("ticket", "external-link", "delete", ticketID, externalLinkID)
	run("ticket", "run", "list", projectID, ticketID)
	run("ticket", "run", "get", projectID, ticketID, runID)

	if len(requests) != 8 {
		t.Fatalf("request count = %d, want 8 (%+v)", len(requests), requests)
	}
	if requests[4].Payload["external_id"] != "123" || requests[4].Payload["url"] != "https://example.com/issues/123" {
		t.Fatalf("external link add payload = %+v", requests[4].Payload)
	}
}

func TestProjectResourceCommandsUsePlatformRuntimeDefaults(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	orgID := "8c44cdd8-02d2-4bc9-aefa-e8c5ca7dd87e"
	createdProjectID := "0a5e6e2a-15c4-4f7b-b8fa-7e7d145f352f"

	type requestRecord struct {
		Method  string
		Path    string
		Payload map[string]any
	}

	requests := make([]requestRecord, 0, 5)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := requestRecord{Method: r.Method, Path: r.URL.RequestURI()}
		if r.Body != nil && (r.Method == http.MethodPost) {
			if err := json.NewDecoder(r.Body).Decode(&record.Payload); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
		}
		requests = append(requests, record)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/projects/"+projectID:
			_, _ = w.Write([]byte(`{"project":{"id":"` + projectID + `","organization_id":"` + orgID + `"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/platform/orgs/"+orgID+"/projects":
			_, _ = w.Write([]byte(`{"projects":[{"id":"` + projectID + `"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/platform/orgs/"+orgID+"/projects":
			_, _ = w.Write([]byte(`{"project":{"id":"` + createdProjectID + `","name":"Runtime Project","slug":"runtime-project"}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/platform/projects/"+createdProjectID:
			_, _ = w.Write([]byte(`{"project":{"id":"` + createdProjectID + `","status":"archived"}}`))
		default:
			t.Fatalf("unexpected request %s %q", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
	t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
	t.Setenv("OPENASE_PROJECT_ID", projectID)

	run := func(args ...string) {
		t.Helper()
		root := NewRootCommand("dev")
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("ExecuteContext(%v) returned error: %v", args, err)
		}
	}

	run("project", "current")
	run("project", "get", projectID)
	run("project", "list", orgID)
	run("project", "create", orgID, "--name", "Runtime Project", "--slug", "runtime-project")
	run("project", "delete", createdProjectID)

	if len(requests) != 5 {
		t.Fatalf("request count = %d, want 5 (%+v)", len(requests), requests)
	}
	if requests[0].Path != "/api/v1/platform/projects/"+projectID {
		t.Fatalf("current path = %q", requests[0].Path)
	}
	if requests[3].Payload["name"] != "Runtime Project" || requests[3].Payload["slug"] != "runtime-project" {
		t.Fatalf("project create payload = %+v", requests[3].Payload)
	}
}

func TestTypedCommandsPreservePlatformBaseForProjectAIRoutes(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	ticketID := "8c44cdd8-02d2-4bc9-aefa-e8c5ca7dd87e"
	agentID := "1eeb5f8b-6679-4694-b890-d79044f52ef1"

	for _, tc := range []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{
			name:     "activity list",
			args:     []string{"activity", "list"},
			wantPath: "/api/v1/platform/projects/" + projectID + "/activity",
			body:     `{"events":[]}`,
		},
		{
			name:     "repo list",
			args:     []string{"repo", "list"},
			wantPath: "/api/v1/platform/projects/" + projectID + "/repos",
			body:     `{"repos":[]}`,
		},
		{
			name:     "scheduled-job list",
			args:     []string{"scheduled-job", "list"},
			wantPath: "/api/v1/platform/projects/" + projectID + "/scheduled-jobs",
			body:     `{"jobs":[]}`,
		},
		{
			name:     "skill list",
			args:     []string{"skill", "list"},
			wantPath: "/api/v1/platform/projects/" + projectID + "/skills",
			body:     `{"skills":[]}`,
		},
		{
			name:     "ticket run list",
			args:     []string{"ticket", "run", "list", projectID, ticketID},
			wantPath: "/api/v1/platform/projects/" + projectID + "/tickets/" + ticketID + "/runs",
			body:     `{"runs":[]}`,
		},
		{
			name:     "agent interrupt",
			args:     []string{"agent", "interrupt", agentID},
			wantPath: "/api/v1/platform/agents/" + agentID + "/interrupt",
			body:     `{"agent":{"id":"` + agentID + `","runtime_control_state":"interrupt_requested"}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requestPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestPath = r.URL.RequestURI()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			t.Setenv("OPENASE_API_URL", server.URL+"/api/v1/platform")
			t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
			t.Setenv("OPENASE_PROJECT_ID", projectID)

			root := NewRootCommand("dev")
			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stdout)
			root.SetArgs(tc.args)

			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext returned error: %v", err)
			}
			if requestPath != tc.wantPath {
				t.Fatalf("request path = %q, want %q", requestPath, tc.wantPath)
			}
		})
	}
}
