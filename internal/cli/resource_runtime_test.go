package cli

import (
	"bytes"
	"context"
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
