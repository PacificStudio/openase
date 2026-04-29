package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestAPICommandContextUsesStoredHumanSessionForMutations(t *testing.T) {
	t.Setenv("OPENASE_AGENT_TOKEN", "")
	t.Setenv("OPENASE_API_URL", "")
	t.Setenv(envHumanSessionToken, "")
	t.Setenv(envHumanCSRFToken, "")

	sessionPath := filepath.Join(t.TempDir(), "human-session.json")
	if err := saveHumanSessionState(sessionPath, humanSessionState{
		APIURL:       "http://127.0.0.1:19836/api/v1",
		SessionToken: "session-token",
		CSRFToken:    "csrf-token",
	}); err != nil {
		t.Fatalf("saveHumanSessionState() error = %v", err)
	}

	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != humanSessionCookieHeaderName+"=session-token" {
			t.Fatalf("cookie = %q", got)
		}
		if got := r.Header.Get("X-OpenASE-CSRF"); got != "csrf-token" {
			t.Fatalf("csrf header = %q", got)
		}
		if got := r.Header.Get("Origin"); got != serverURL {
			t.Fatalf("origin = %q, want %q", got, serverURL)
		}
		if got := r.Header.Get("User-Agent"); got != openASECLIUserAgent {
			t.Fatalf("user-agent = %q, want %q", got, openASECLIUserAgent)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	serverURL = server.URL

	ctx, err := (apiCommandOptions{
		apiURL:           server.URL + "/api/v1",
		humanSessionFile: sessionPath,
	}).resolve()
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}

	if _, err := ctx.do(context.Background(), apiCommandDeps{httpClient: server.Client()}, apiRequest{
		Method: http.MethodPost,
		Path:   "auth/logout",
	}); err != nil {
		t.Fatalf("do() error = %v", err)
	}
}

func TestAPICommandContextUsesStoredHumanSessionForReadsWithoutCSRF(t *testing.T) {
	t.Setenv("OPENASE_AGENT_TOKEN", "")
	t.Setenv("OPENASE_API_URL", "")
	t.Setenv(envHumanSessionToken, "")
	t.Setenv(envHumanCSRFToken, "")

	sessionPath := filepath.Join(t.TempDir(), "human-session.json")
	if err := saveHumanSessionState(sessionPath, humanSessionState{
		SessionToken: "session-token",
		CSRFToken:    "csrf-token",
	}); err != nil {
		t.Fatalf("saveHumanSessionState() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != humanSessionCookieHeaderName+"=session-token" {
			t.Fatalf("cookie = %q", got)
		}
		if got := r.Header.Get("X-OpenASE-CSRF"); got != "" {
			t.Fatalf("csrf header = %q, want empty", got)
		}
		if got := r.Header.Get("Origin"); got != "" {
			t.Fatalf("origin = %q, want empty", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ctx, err := (apiCommandOptions{
		apiURL:           server.URL + "/api/v1",
		humanSessionFile: sessionPath,
	}).resolve()
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}

	if _, err := ctx.do(context.Background(), apiCommandDeps{httpClient: server.Client()}, apiRequest{
		Method: http.MethodGet,
		Path:   "auth/session",
	}); err != nil {
		t.Fatalf("do() error = %v", err)
	}
}

func TestAPICommandContextPrefersBearerTokenOverStoredHumanSession(t *testing.T) {
	t.Setenv(envHumanSessionToken, "")
	t.Setenv(envHumanCSRFToken, "")

	sessionPath := filepath.Join(t.TempDir(), "human-session.json")
	if err := saveHumanSessionState(sessionPath, humanSessionState{
		SessionToken: "session-token",
		CSRFToken:    "csrf-token",
	}); err != nil {
		t.Fatalf("saveHumanSessionState() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer agent-token" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("Cookie"); got != "" {
			t.Fatalf("cookie = %q, want empty", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ctx, err := (apiCommandOptions{
		apiURL:           server.URL + "/api/v1",
		token:            "agent-token",
		humanSessionFile: sessionPath,
	}).resolve()
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}

	if _, err := ctx.do(context.Background(), apiCommandDeps{httpClient: server.Client()}, apiRequest{
		Method: http.MethodGet,
		Path:   "auth/session",
	}); err != nil {
		t.Fatalf("do() error = %v", err)
	}
}

func TestResolveForOperationPreservesPlatformBaseForAgentCapableOperation(t *testing.T) {
	ctx, err := (apiCommandOptions{
		apiURL: "http://127.0.0.1:19836/api/v1/platform",
		token:  "agent-token",
	}).resolveForOperation(http.MethodPost, "/api/v1/projects/{projectId}/skills/refresh")
	if err != nil {
		t.Fatalf("resolveForOperation() error = %v", err)
	}
	if ctx.apiURL != "http://127.0.0.1:19836/api/v1/platform" {
		t.Fatalf("apiURL = %q, want platform base", ctx.apiURL)
	}
}

func TestResolveForOperationNormalizesHumanOnlyOperationBackToHumanBase(t *testing.T) {
	ctx, err := (apiCommandOptions{
		apiURL: "http://127.0.0.1:19836/api/v1/platform",
		token:  "agent-token",
	}).resolveForOperation(http.MethodGet, "/api/v1/orgs/{orgId}/machines")
	if err != nil {
		t.Fatalf("resolveForOperation() error = %v", err)
	}
	if ctx.apiURL != "http://127.0.0.1:19836/api/v1" {
		t.Fatalf("apiURL = %q, want human base", ctx.apiURL)
	}
}

func TestBuildRequestURLPreservesPlatformPrefixForRelativePaths(t *testing.T) {
	requestURL, err := buildRequestURL("http://127.0.0.1:19836/api/v1/platform", "projects/project-123/skills")
	if err != nil {
		t.Fatalf("buildRequestURL() error = %v", err)
	}
	if requestURL != "http://127.0.0.1:19836/api/v1/platform/projects/project-123/skills" {
		t.Fatalf("requestURL = %q", requestURL)
	}
}

func TestBuildRequestURLPreservesPlatformPrefixForAbsolutePaths(t *testing.T) {
	requestURL, err := buildRequestURL("http://127.0.0.1:19836/api/v1/platform", "/api/v1/platform/projects/project-123/skills")
	if err != nil {
		t.Fatalf("buildRequestURL() error = %v", err)
	}
	if requestURL != "http://127.0.0.1:19836/api/v1/platform/projects/project-123/skills" {
		t.Fatalf("requestURL = %q", requestURL)
	}
}
