package httpapi

import "testing"

func TestBuildOpenAPIDocument(t *testing.T) {
	doc, err := BuildOpenAPIDocument()
	if err != nil {
		t.Fatalf("build openapi document: %v", err)
	}

	requiredPaths := []string{
		"/api/v1/system/dashboard",
		"/api/v1/orgs",
		"/api/v1/orgs/{orgId}/machines",
		"/api/v1/machines/{machineId}/test",
		"/api/v1/orgs/{orgId}/providers",
		"/api/v1/harness/variables",
		"/api/v1/projects/{projectId}/workflows",
		"/api/v1/projects/{projectId}/scheduled-jobs",
		"/api/v1/projects/{projectId}/tickets/{ticketId}/detail",
		"/api/v1/chat",
		"/api/v1/projects/{projectId}/tickets/stream",
	}
	for _, path := range requiredPaths {
		if doc.Paths.Value(path) == nil {
			t.Fatalf("expected path %s to exist in the openapi document", path)
		}
	}
}
