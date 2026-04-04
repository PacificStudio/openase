package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/getkin/kin-openapi/openapi3"
)

func TestBuildOpenAPIDocument(t *testing.T) {
	doc, err := BuildOpenAPIDocument()
	if err != nil {
		t.Fatalf("build openapi document: %v", err)
	}

	requiredPaths := []string{
		"/api/v1/auth/oidc/start",
		"/api/v1/auth/oidc/callback",
		"/api/v1/auth/session",
		"/api/v1/auth/logout",
		"/api/v1/auth/me/permissions",
		"/api/v1/system/dashboard",
		"/api/v1/workspace/summary",
		"/api/v1/orgs",
		"/api/v1/organizations/{orgId}/role-bindings",
		"/api/v1/organizations/{orgId}/role-bindings/{bindingId}",
		"/api/v1/orgs/{orgId}/summary",
		"/api/v1/orgs/{orgId}/token-usage",
		"/api/v1/orgs/{orgId}/channels",
		"/api/v1/orgs/{orgId}/machines",
		"/api/v1/machines/{machineId}/test",
		"/api/v1/provider-model-options",
		"/api/v1/orgs/{orgId}/providers",
		"/api/v1/providers/{providerId}",
		"/api/v1/harness/variables",
		"/api/v1/projects/{projectId}/repos",
		"/api/v1/projects/{projectId}/token-usage",
		"/api/v1/projects/{projectId}/statuses/reset",
		"/api/v1/projects/{projectId}/workflows",
		"/api/v1/workflows/{workflowId}/impact",
		"/api/v1/workflows/{workflowId}/retire",
		"/api/v1/workflows/{workflowId}/replace-references",
		"/api/v1/tickets/{ticketId}/external-links",
		"/api/v1/projects/{projectId}/scheduled-jobs",
		"/api/v1/projects/{projectId}/notification-rules",
		"/api/v1/projects/{projectId}/security-settings",
		"/api/v1/projects/{projectId}/security-settings/github-outbound-credential",
		"/api/v1/projects/{projectId}/security-settings/github-outbound-credential/import-gh-cli",
		"/api/v1/projects/{projectId}/security-settings/github-outbound-credential/retest",
		"/api/v1/projects/{projectId}/updates",
		"/api/v1/projects/{projectId}/role-bindings",
		"/api/v1/projects/{projectId}/role-bindings/{bindingId}",
		"/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes",
		"/api/v1/projects/{projectId}/tickets/{ticketId}/detail",
		"/api/v1/agents/{agentId}/retire",
		"/api/v1/tickets/{ticketId}/comments",
		"/api/v1/tickets/{ticketId}/dependencies",
		"/api/v1/tickets/{ticketId}/workspace/reset",
		"/api/v1/chat",
		"/api/v1/chat/projects/{projectId}/conversations/stream",
		"/api/v1/chat/conversations",
		"/api/v1/chat/conversations/{conversationId}",
		"/api/v1/chat/conversations/{conversationId}/entries",
		"/api/v1/chat/conversations/{conversationId}/turns",
		"/api/v1/chat/conversations/{conversationId}/stream",
		"/api/v1/chat/conversations/{conversationId}/interrupts/{interruptId}/respond",
		"/api/v1/chat/conversations/{conversationId}/runtime",
		"/api/v1/skills/{skillId}/files",
		"/api/v1/projects/{projectId}/events/stream",
	}
	for _, path := range requiredPaths {
		if doc.Paths.Value(path) == nil {
			t.Fatalf("expected path %s to exist in the openapi document", path)
		}
	}
}

func TestBuildOpenAPIJSONAndRoute(t *testing.T) {
	payload, err := BuildOpenAPIJSON()
	if err != nil {
		t.Fatalf("BuildOpenAPIJSON() error = %v", err)
	}
	if !strings.HasSuffix(string(payload), "\n") {
		t.Fatalf("expected trailing newline, got %q", string(payload[len(payload)-1:]))
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal openapi json: %v", err)
	}
	if decoded["openapi"] == "" {
		t.Fatalf("expected openapi version in payload, got %+v", decoded)
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/openapi.json", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected openapi route 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected application/json content type, got %q", contentType)
	}
	if !strings.Contains(rec.Body.String(), "\"openapi\"") {
		t.Fatalf("expected openapi payload body, got %s", rec.Body.String())
	}
}

func TestBuildOpenAPIDocumentRequestFieldsHaveDescriptions(t *testing.T) {
	doc, err := BuildOpenAPIDocument()
	if err != nil {
		t.Fatalf("build openapi document: %v", err)
	}

	missing := make([]string, 0)
	for path, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for method, operation := range pathItem.Operations() {
			if operation == nil {
				continue
			}
			parameters := append(openapi3.Parameters{}, pathItem.Parameters...)
			parameters = append(parameters, operation.Parameters...)
			for _, parameter := range parameters {
				if parameter == nil || parameter.Value == nil {
					continue
				}
				if strings.TrimSpace(parameter.Value.Description) == "" {
					missing = append(missing, strings.ToUpper(method)+" "+path+" param "+parameter.Value.Name)
				}
			}
			if operation.RequestBody == nil || operation.RequestBody.Value == nil {
				continue
			}
			for mediaType, body := range operation.RequestBody.Value.Content {
				if body == nil || body.Schema == nil {
					continue
				}
				collectMissingSchemaDescriptions(body.Schema, strings.ToUpper(method)+" "+path+" body "+mediaType, &missing)
			}
		}
	}

	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("openapi request field descriptions must be non-empty:\n%s", strings.Join(missing, "\n"))
	}
}

func TestOpenAPIProjectStreamsStayOnCanonicalBus(t *testing.T) {
	doc, err := BuildOpenAPIDocument()
	if err != nil {
		t.Fatalf("build openapi document: %v", err)
	}

	allowed := map[string]bool{
		"/api/v1/projects/{projectId}/events/stream":                  true,
		"/api/v1/projects/{projectId}/agents/{agentId}/output/stream": true,
		"/api/v1/projects/{projectId}/agents/{agentId}/steps/stream":  true,
	}

	found := 0
	for path := range doc.Paths.Map() {
		if !strings.HasPrefix(path, "/api/v1/projects/{projectId}/") || !strings.HasSuffix(path, "/stream") {
			continue
		}
		found++
		if !allowed[path] {
			t.Fatalf("unexpected project stream path in openapi: %s", path)
		}
	}

	if found != len(allowed) {
		t.Fatalf("expected %d project stream paths in openapi, found %d", len(allowed), found)
	}
}

func collectMissingSchemaDescriptions(schemaRef *openapi3.SchemaRef, prefix string, missing *[]string) {
	if schemaRef == nil || schemaRef.Value == nil {
		return
	}
	schema := schemaRef.Value
	for name, property := range schema.Properties {
		if property == nil || property.Value == nil {
			continue
		}
		if strings.TrimSpace(property.Value.Description) == "" {
			*missing = append(*missing, prefix+" field "+name)
		}
		collectMissingSchemaDescriptions(property, prefix+"."+name, missing)
	}
	if schema.Items != nil {
		collectMissingSchemaDescriptions(schema.Items, prefix+"[]", missing)
	}
	for _, item := range schema.AllOf {
		collectMissingSchemaDescriptions(item, prefix, missing)
	}
	for _, item := range schema.AnyOf {
		collectMissingSchemaDescriptions(item, prefix, missing)
	}
	for _, item := range schema.OneOf {
		collectMissingSchemaDescriptions(item, prefix, missing)
	}
	if schema.Not != nil {
		collectMissingSchemaDescriptions(schema.Not, prefix, missing)
	}
	if schema.AdditionalProperties.Schema != nil {
		collectMissingSchemaDescriptions(schema.AdditionalProperties.Schema, prefix+".*", missing)
	}
}
