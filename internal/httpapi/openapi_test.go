package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
)

func TestBuildOpenAPIDocument(t *testing.T) {
	doc, err := BuildOpenAPIDocument()
	if err != nil {
		t.Fatalf("build openapi document: %v", err)
	}

	requiredPaths := []string{
		"/api/v1/system/dashboard",
		"/api/v1/orgs",
		"/api/v1/orgs/{orgId}/channels",
		"/api/v1/orgs/{orgId}/machines",
		"/api/v1/machines/{machineId}/test",
		"/api/v1/provider-model-options",
		"/api/v1/orgs/{orgId}/providers",
		"/api/v1/harness/variables",
		"/api/v1/projects/{projectId}/repos",
		"/api/v1/projects/{projectId}/stages",
		"/api/v1/projects/{projectId}/statuses/reset",
		"/api/v1/stages/{stageId}",
		"/api/v1/projects/{projectId}/workflows",
		"/api/v1/projects/{projectId}/workflows/prerequisite",
		"/api/v1/tickets/{ticketId}/external-links",
		"/api/v1/projects/{projectId}/scheduled-jobs",
		"/api/v1/projects/{projectId}/notification-rules",
		"/api/v1/projects/{projectId}/security-settings",
		"/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes",
		"/api/v1/projects/{projectId}/tickets/{ticketId}/detail",
		"/api/v1/tickets/{ticketId}/comments",
		"/api/v1/tickets/{ticketId}/dependencies",
		"/api/v1/chat",
		"/api/v1/projects/{projectId}/tickets/stream",
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
