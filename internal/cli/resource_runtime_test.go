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
	if requestPath != "/api/v1/projects/"+projectID {
		t.Fatalf("project get request path = %q, want %q", requestPath, "/api/v1/projects/"+projectID)
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
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/"+projectID+"/updates":
			_, _ = w.Write([]byte(`{"threads":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/"+projectID+"/updates":
			_, _ = w.Write([]byte(`{"thread":{"id":"` + threadID + `","status":"on_track","title":"CLI parity","body":"Created"}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/projects/"+projectID+"/updates/"+threadID:
			_, _ = w.Write([]byte(`{"thread":{"id":"` + threadID + `","status":"at_risk","title":"CLI parity","body":"Updated"}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/projects/"+projectID+"/updates/"+threadID:
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
	run("project", "updates", "create", "--status", "on_track", "--title", "CLI parity", "--body", "Created")
	run("project", "updates", "update", "--thread-id", threadID, "--status", "at_risk", "--title", "CLI parity", "--body", "Updated")
	run("project", "updates", "delete", "--thread-id", threadID)

	if len(requests) != 4 {
		t.Fatalf("request count = %d, want 4 (%+v)", len(requests), requests)
	}
	if requests[0].Path != "/api/v1/projects/"+projectID+"/updates" {
		t.Fatalf("list path = %q", requests[0].Path)
	}
	if requests[1].Method != http.MethodPost || requests[1].Path != "/api/v1/projects/"+projectID+"/updates" {
		t.Fatalf("create request = %+v", requests[1])
	}
	if requests[1].Payload["status"] != "on_track" || requests[1].Payload["title"] != "CLI parity" || requests[1].Payload["body"] != "Created" {
		t.Fatalf("create payload = %+v", requests[1].Payload)
	}
	if requests[2].Method != http.MethodPatch || requests[2].Path != "/api/v1/projects/"+projectID+"/updates/"+threadID {
		t.Fatalf("update request = %+v", requests[2])
	}
	if requests[2].Payload["status"] != "at_risk" || requests[2].Payload["body"] != "Updated" {
		t.Fatalf("update payload = %+v", requests[2].Payload)
	}
	if requests[3].Method != http.MethodDelete || requests[3].Path != "/api/v1/projects/"+projectID+"/updates/"+threadID {
		t.Fatalf("delete request = %+v", requests[3])
	}
}
