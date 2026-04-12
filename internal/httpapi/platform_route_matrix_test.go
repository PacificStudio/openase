package httpapi

import (
	"net/http"
	"testing"
)

func TestHasAgentPlatformRouteRecognizesAgentRuntimeOperations(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/projects/{projectId}/skills/refresh"},
		{method: http.MethodDelete, path: "/api/v1/skills/{skillId}"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/github/namespaces"},
		{method: http.MethodPost, path: "/api/v1/projects/{projectId}/github/repos"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/agents"},
		{method: http.MethodPatch, path: "/api/v1/agents/{agentId}"},
		{method: http.MethodPost, path: "/api/v1/agents/{agentId}/pause"},
		{method: http.MethodPost, path: "/api/v1/agents/{agentId}/resume"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/agents/{agentId}/output"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/agents/{agentId}/steps"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/notification-rules"},
		{method: http.MethodPatch, path: "/api/v1/notification-rules/{ruleId}"},
		{method: http.MethodPost, path: "/api/v1/harness/validate"},
		{method: http.MethodGet, path: "/api/v1/harness/variables"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/scheduled-jobs"},
		{method: http.MethodPatch, path: "/api/v1/scheduled-jobs/{jobId}"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/activity"},
		{method: http.MethodPost, path: "/api/v1/projects/{projectId}/tickets"},
		{method: http.MethodPatch, path: "/api/v1/tickets/{ticketId}"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/statuses"},
	} {
		if HasAgentPlatformRoute(tc.method, tc.path) {
			continue
		}
		t.Fatalf("HasAgentPlatformRoute(%s, %q) = false, want true", tc.method, tc.path)
	}
}

func TestHasAgentPlatformRouteRejectsHumanOnlyOperations(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/auth/session"},
		{method: http.MethodPost, path: "/api/v1/auth/logout"},
		{method: http.MethodGet, path: "/api/v1/orgs/{orgId}/machines"},
		{method: http.MethodGet, path: "/api/v1/providers/{providerId}"},
		{method: http.MethodPost, path: "/api/v1/channels/{channelId}/test"},
		{method: http.MethodGet, path: "/api/v1/projects/{projectId}/agents/{agentId}/output/stream"},
		{method: http.MethodGet, path: "/api/v1/events/stream"},
	} {
		if !HasAgentPlatformRoute(tc.method, tc.path) {
			continue
		}
		t.Fatalf("HasAgentPlatformRoute(%s, %q) = true, want false", tc.method, tc.path)
	}
}

func TestNormalizeAgentPlatformRoutePathCanonicalizesHumanAndPlatformForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "human path with braces",
			in:   "/api/v1/projects/{projectId}/notification-rules",
			want: "/api/v1/projects/:projectId/notification-rules",
		},
		{
			name: "platform path with braces",
			in:   "/api/v1/platform/projects/{projectId}/agents/{agentId}/steps",
			want: "/api/v1/projects/:projectId/agents/:agentId/steps",
		},
		{
			name: "echo route path",
			in:   "/api/v1/platform/tickets/:ticketId/dependencies/:dependencyId",
			want: "/api/v1/tickets/:ticketId/dependencies/:dependencyId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAgentPlatformRoutePath(tt.in); got != tt.want {
				t.Fatalf("normalizeAgentPlatformRoutePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
