package cli

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/httpapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"
)

var intentionalCLIOpenAPIGaps = map[string]string{
	contractKey("POST", "/api/v1/agents/{agentId}/retire"):                                                         "agent retirement has no first-class CLI yet",
	contractKey("GET", "/api/v1/auth/me/permissions"):                                                              "effective permission inspection has no first-class CLI yet",
	contractKey("GET", "/api/v1/auth/oidc/callback"):                                                               "OIDC browser callback is intentionally not exposed as a CLI command",
	contractKey("GET", "/api/v1/auth/oidc/start"):                                                                  "OIDC browser start flow is intentionally not exposed as a CLI command",
	contractKey("GET", "/api/v1/admin/auth"):                                                                       "instance auth console inspection has no first-class CLI yet",
	contractKey("PUT", "/api/v1/admin/auth/oidc-draft"):                                                            "instance OIDC draft persistence has no first-class CLI yet",
	contractKey("POST", "/api/v1/admin/auth/oidc-draft/test"):                                                      "instance OIDC connectivity testing has no first-class CLI yet",
	contractKey("POST", "/api/v1/admin/auth/oidc-enable"):                                                          "instance OIDC activation is intentionally guided through the admin UI today",
	contractKey("POST", "/api/v1/admin/auth/disable"):                                                              "instance auth rollback is intentionally guided through the admin UI today",
	contractKey("GET", "/api/v1/chat/conversations/{conversationId}/workspace"):                                    "project conversation workspace inspection has no first-class CLI yet",
	contractKey("GET", "/api/v1/chat/conversations/{conversationId}/workspace/file"):                               "project conversation workspace file preview has no first-class CLI yet",
	contractKey("GET", "/api/v1/chat/conversations/{conversationId}/workspace/file-patch"):                         "project conversation workspace file diff has no first-class CLI yet",
	contractKey("GET", "/api/v1/chat/conversations/{conversationId}/workspace/tree"):                               "project conversation workspace tree browsing has no first-class CLI yet",
	contractKey("POST", "/api/v1/chat/conversations/{conversationId}/workspace/sync"):                              "project conversation workspace sync has no first-class CLI yet",
	contractKey("POST", "/api/v1/chat/conversations/{conversationId}/terminal-sessions"):                           "project conversation terminal session creation has no first-class CLI yet",
	contractKey("GET", "/api/v1/chat/conversations/{conversationId}/terminal-sessions/{terminalSessionId}/attach"): "project conversation terminal websocket attach has no first-class CLI yet",
	contractKey("GET", "/api/v1/chat/projects/{projectId}/conversations/stream"):                                   "project-scoped conversation stream has no first-class CLI yet",
	contractKey("GET", "/api/v1/instance/users"):                                                                   "instance user directory listing has no first-class CLI yet",
	contractKey("GET", "/api/v1/instance/users/{userId}"):                                                          "instance user directory detail has no first-class CLI yet",
	contractKey("DELETE", "/api/v1/instance/sessions/{id}"):                                                        "instance session governance has no first-class CLI yet",
	contractKey("POST", "/api/v1/instance/users/{userId}/status"):                                                  "instance user lifecycle transition has no first-class CLI yet",
	contractKey("GET", "/api/v1/instance/role-bindings"):                                                           "instance role binding inspection has no first-class CLI yet",
	contractKey("POST", "/api/v1/instance/role-bindings"):                                                          "instance role binding mutation has no first-class CLI yet",
	contractKey("DELETE", "/api/v1/instance/role-bindings/{bindingId}"):                                            "instance role binding deletion has no first-class CLI yet",
	contractKey("GET", "/api/v1/notification-event-types"):                                                         "notification event catalog has no first-class CLI yet",
	contractKey("POST", "/api/v1/org-invitations/accept"):                                                          "organization invitation acceptance has no first-class CLI yet",
	contractKey("GET", "/api/v1/organizations/{orgId}/role-bindings"):                                              "organization role binding inspection has no first-class CLI yet",
	contractKey("POST", "/api/v1/organizations/{orgId}/role-bindings"):                                             "organization role binding mutation has no first-class CLI yet",
	contractKey("DELETE", "/api/v1/organizations/{orgId}/role-bindings/{bindingId}"):                               "organization role binding deletion has no first-class CLI yet",
	contractKey("GET", "/api/v1/orgs"):                                                                             "organization catalog has no first-class CLI namespace yet",
	contractKey("POST", "/api/v1/orgs"):                                                                            "organization creation has no first-class CLI namespace yet",
	contractKey("DELETE", "/api/v1/orgs/{orgId}"):                                                                  "organization archive has no first-class CLI namespace yet",
	contractKey("POST", "/api/v1/orgs/{orgId}/invitations"):                                                        "organization invitation lifecycle has no first-class CLI yet",
	contractKey("POST", "/api/v1/orgs/{orgId}/invitations/{invitationId}/cancel"):                                  "organization invitation lifecycle has no first-class CLI yet",
	contractKey("POST", "/api/v1/orgs/{orgId}/invitations/{invitationId}/resend"):                                  "organization invitation lifecycle has no first-class CLI yet",
	contractKey("GET", "/api/v1/orgs/{orgId}/members"):                                                             "organization membership lifecycle has no first-class CLI yet",
	contractKey("PATCH", "/api/v1/orgs/{orgId}/members/{membershipId}"):                                            "organization membership lifecycle has no first-class CLI yet",
	contractKey("POST", "/api/v1/orgs/{orgId}/members/{membershipId}/transfer-ownership"):                          "organization ownership transfer has no first-class CLI yet",
	contractKey("GET", "/api/v1/orgs/{orgId}/security-settings/secrets"):                                           "organization scoped secret inspection has no first-class CLI yet",
	contractKey("POST", "/api/v1/orgs/{orgId}/security-settings/secrets"):                                          "organization scoped secret creation has no first-class CLI yet",
	contractKey("DELETE", "/api/v1/orgs/{orgId}/security-settings/secrets/{secretId}"):                             "organization scoped secret deletion has no first-class CLI yet",
	contractKey("POST", "/api/v1/orgs/{orgId}/security-settings/secrets/{secretId}/disable"):                       "organization scoped secret disable has no first-class CLI yet",
	contractKey("POST", "/api/v1/orgs/{orgId}/security-settings/secrets/{secretId}/rotate"):                        "organization scoped secret rotation has no first-class CLI yet",
	contractKey("PATCH", "/api/v1/orgs/{orgId}"):                                                                   "organization update has no first-class CLI namespace yet",
	contractKey("GET", "/api/v1/orgs/{orgId}/providers/stream"):                                                    "provider stream has no first-class CLI yet",
	contractKey("GET", "/api/v1/orgs/{orgId}/summary"):                                                             "organization summary has no first-class CLI yet",
	contractKey("GET", "/api/v1/orgs/{orgId}/token-usage"):                                                         "organization token usage has no first-class CLI yet",
	contractKey("GET", "/api/v1/projects/{projectId}/agent-runs"):                                                  "project agent run listing has no first-class CLI yet",
	contractKey("GET", "/api/v1/projects/{projectId}/hr-advisor"):                                                  "HR advisor inspection has no first-class CLI yet",
	contractKey("POST", "/api/v1/projects/{projectId}/hr-advisor/activate"):                                        "HR advisor activation has no first-class CLI yet",
	contractKey("GET", "/api/v1/projects/{projectId}/role-bindings"):                                               "project role binding inspection has no first-class CLI yet",
	contractKey("POST", "/api/v1/projects/{projectId}/role-bindings"):                                              "project role binding mutation has no first-class CLI yet",
	contractKey("DELETE", "/api/v1/projects/{projectId}/role-bindings/{bindingId}"):                                "project role binding deletion has no first-class CLI yet",
	contractKey("GET", "/api/v1/projects/{projectId}/security-settings"):                                           "security settings inspection has no first-class CLI yet",
	contractKey("DELETE", "/api/v1/projects/{projectId}/security-settings/github-outbound-credential"):             "GitHub outbound credential deletion has no first-class CLI yet",
	contractKey("PUT", "/api/v1/projects/{projectId}/security-settings/github-outbound-credential"):                "GitHub outbound credential upsert has no first-class CLI yet",
	contractKey("POST", "/api/v1/projects/{projectId}/security-settings/github-outbound-credential/import-gh-cli"): "GitHub CLI credential import has no first-class CLI yet",
	contractKey("POST", "/api/v1/projects/{projectId}/security-settings/github-outbound-credential/retest"):        "GitHub credential retest has no first-class CLI yet",
	contractKey("PUT", "/api/v1/projects/{projectId}/security-settings/oidc-draft"):                                "OIDC draft persistence has no first-class CLI yet",
	contractKey("POST", "/api/v1/projects/{projectId}/security-settings/oidc-draft/test"):                          "OIDC draft connectivity testing has no first-class CLI yet",
	contractKey("POST", "/api/v1/projects/{projectId}/security-settings/oidc-enable"):                              "OIDC activation is intentionally guided through the admin UI today",
	contractKey("GET", "/api/v1/projects/{projectId}/token-usage"):                                                 "project token usage has no first-class CLI yet",
	contractKey("POST", "/api/v1/projects/{projectId}/updates/{threadId}/comments"):                                "project update comments have no first-class CLI yet",
	contractKey("DELETE", "/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}"):                  "project update comment deletion has no first-class CLI yet",
	contractKey("PATCH", "/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}"):                   "project update comment update has no first-class CLI yet",
	contractKey("GET", "/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}/revisions"):           "project update comment revisions have no first-class CLI yet",
	contractKey("GET", "/api/v1/provider-model-options"):                                                           "provider model option discovery has no first-class CLI yet",
	contractKey("GET", "/api/v1/roles/builtin"):                                                                    "builtin role listing has no first-class CLI yet",
	contractKey("GET", "/api/v1/roles/builtin/{roleSlug}"):                                                         "builtin role detail has no first-class CLI yet",
	contractKey("GET", "/api/v1/skills/{skillId}/files"):                                                           "skill file inspection has no first-class CLI yet",
	contractKey("GET", "/api/v1/skills/{skillId}/history"):                                                         "skill history inspection has no first-class CLI yet",
	contractKey("GET", "/api/v1/system/dashboard"):                                                                 "system dashboard has no first-class CLI yet",
	contractKey("POST", "/api/v1/tickets/{ticketId}/workspace/reset"):                                              "ticket workspace reset has no first-class CLI yet",
	contractKey("GET", "/api/v1/tickets/{ticketId}/comments/{commentId}/revisions"):                                "ticket comment revisions are missing from the root wrapper",
	contractKey("DELETE", "/api/v1/tickets/{ticketId}/comments/{commentId}"):                                       "ticket comment deletion is missing from the root wrapper",
	contractKey("GET", "/api/v1/workflows/{workflowId}/impact"):                                                    "workflow impact inspection has no first-class CLI yet",
	contractKey("POST", "/api/v1/workflows/{workflowId}/replace-references"):                                       "workflow reference replacement has no first-class CLI yet",
	contractKey("POST", "/api/v1/workflows/{workflowId}/retire"):                                                   "workflow retirement has no first-class CLI yet",
	contractKey("GET", "/api/v1/workspace/summary"):                                                                "workspace summary has no first-class CLI yet",
}

var intentionalCLIWithoutOpenAPIOperations = map[string]string{
	contractKey("POST", "/api/v1/projects/{projectId}/skills/import"): "skill import ships as a custom CLI command before the endpoint is modeled in OpenAPI",
}

type openAPIOperationMetadata struct {
	Method      string
	Path        string
	OperationID string
}

func TestRootCLIAPICoverageMatchesOpenAPI(t *testing.T) {
	root := NewRootCommand("dev")
	coveredOperations := collectCLIAPICoverage(root)

	doc, err := httpapi.BuildOpenAPIDocument()
	if err != nil {
		t.Fatalf("BuildOpenAPIDocument() error = %v", err)
	}
	openAPIOperations := collectOpenAPIOperations(doc)

	missing := make([]string, 0)
	for key, operation := range openAPIOperations {
		if _, ok := coveredOperations[key]; ok {
			continue
		}
		if reason, ok := intentionalCLIOpenAPIGaps[key]; ok {
			_ = reason
			continue
		}
		missing = append(missing, formatOpenAPIOperation(operation))
	}

	staleAllowlist := make([]string, 0)
	for key, reason := range intentionalCLIOpenAPIGaps {
		if _, ok := openAPIOperations[key]; !ok {
			staleAllowlist = append(staleAllowlist, fmt.Sprintf("%s (%s)", key, reason))
			continue
		}
		if commandPath, ok := coveredOperations[key]; ok {
			staleAllowlist = append(staleAllowlist, fmt.Sprintf("%s is now covered by %s (%s)", key, commandPath, reason))
		}
	}

	unknownCoverage := make([]string, 0)
	for key, commandPath := range coveredOperations {
		if _, ok := intentionalCLIWithoutOpenAPIOperations[key]; ok {
			continue
		}
		if _, ok := openAPIOperations[key]; ok {
			continue
		}
		unknownCoverage = append(unknownCoverage, fmt.Sprintf("%s via %s", key, commandPath))
	}

	for key, reason := range intentionalCLIWithoutOpenAPIOperations {
		commandPath, covered := coveredOperations[key]
		_, presentInOpenAPI := openAPIOperations[key]
		switch {
		case !covered:
			staleAllowlist = append(staleAllowlist, fmt.Sprintf("%s no longer has a CLI command (%s)", key, reason))
		case presentInOpenAPI:
			staleAllowlist = append(staleAllowlist, fmt.Sprintf("%s is now modeled in OpenAPI and covered by %s (%s)", key, commandPath, reason))
		}
	}

	sort.Strings(missing)
	sort.Strings(staleAllowlist)
	sort.Strings(unknownCoverage)

	if len(missing) == 0 && len(staleAllowlist) == 0 && len(unknownCoverage) == 0 {
		return
	}

	var report strings.Builder
	report.WriteString("CLI/OpenAPI parity drift detected.\n")
	if len(missing) > 0 {
		report.WriteString("\nMissing first-class CLI coverage for:\n")
		for _, item := range missing {
			report.WriteString("  - ")
			report.WriteString(item)
			report.WriteByte('\n')
		}
	}
	if len(staleAllowlist) > 0 {
		report.WriteString("\nStale intentional gap entries:\n")
		for _, item := range staleAllowlist {
			report.WriteString("  - ")
			report.WriteString(item)
			report.WriteByte('\n')
		}
	}
	if len(unknownCoverage) > 0 {
		report.WriteString("\nCLI commands reference API operations missing from OpenAPI:\n")
		for _, item := range unknownCoverage {
			report.WriteString("  - ")
			report.WriteString(item)
			report.WriteByte('\n')
		}
	}
	t.Fatal(report.String())
}

func collectCLIAPICoverage(root *cobra.Command) map[string]string {
	covered := make(map[string]string)
	walkCLILeaves(root, []string{root.Name()}, func(path []string, command *cobra.Command) {
		key, ok := cliCommandAPICoverageKey(command)
		if !ok {
			return
		}
		covered[key] = strings.Join(path, " ")
	})
	return covered
}

func collectOpenAPIOperations(doc *openapi3.T) map[string]openAPIOperationMetadata {
	operations := make(map[string]openAPIOperationMetadata)
	if doc == nil || doc.Paths == nil {
		return operations
	}
	for path, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for method, operation := range pathItem.Operations() {
			if operation == nil {
				continue
			}
			key := contractKey(method, path)
			operations[key] = openAPIOperationMetadata{
				Method:      strings.ToUpper(strings.TrimSpace(method)),
				Path:        path,
				OperationID: strings.TrimSpace(operation.OperationID),
			}
		}
	}
	return operations
}

func formatOpenAPIOperation(operation openAPIOperationMetadata) string {
	if strings.TrimSpace(operation.OperationID) == "" {
		return contractKey(operation.Method, operation.Path)
	}
	return fmt.Sprintf("%s (%s)", contractKey(operation.Method, operation.Path), operation.OperationID)
}
