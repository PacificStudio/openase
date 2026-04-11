package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
)

func TestAuthBootstrapCreateLinkOutputsShortLivedAuthorizationURL(t *testing.T) {
	client, dsn := openCLIEntClient(t)
	_ = client

	t.Setenv("OPENASE_DATABASE_DSN", dsn)
	t.Setenv("OPENASE_SERVER_HOST", "127.0.0.1")
	t.Setenv("OPENASE_SERVER_PORT", "19836")

	command := newAuthBootstrapCreateLinkCommand(&rootOptions{})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{
		"--requested-by", "cli:test",
		"--ttl", "5m",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	var response localBootstrapLinkResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if response.RequestID == "" || response.Code == "" || response.Nonce == "" {
		t.Fatalf("expected request materials in response, got %+v", response)
	}
	if !strings.HasPrefix(response.URL, "http://127.0.0.1:19836/local-bootstrap?") {
		t.Fatalf("unexpected url %q", response.URL)
	}
	if !strings.Contains(response.URL, "request_id=") || !strings.Contains(response.URL, "code=") || !strings.Contains(response.URL, "nonce=") {
		t.Fatalf("expected one-time materials in url %q", response.URL)
	}
	if strings.Contains(response.URL, "OPENASE_AUTH_TOKEN") || strings.Contains(response.URL, "ase_local_") {
		t.Fatalf("url must not leak a long-lived bearer token: %q", response.URL)
	}
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		t.Fatalf("parse expires_at: %v", err)
	}
	if time.Until(expiresAt) <= 0 {
		t.Fatalf("expected future expiry, got %s", response.ExpiresAt)
	}
}

func TestAuthBootstrapCreateLinkRejectsActiveOIDC(t *testing.T) {
	client, dsn := openCLIEntClient(t)
	t.Setenv("OPENASE_DATABASE_DSN", dsn)
	t.Setenv("OPENASE_SERVER_HOST", "127.0.0.1")
	t.Setenv("OPENASE_SERVER_PORT", "19836")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configBody := strings.TrimSpace(`
database:
  dsn: "` + dsn + `"
server:
  host: 127.0.0.1
  port: 19836
auth:
  mode: oidc
  oidc:
    issuer_url: https://idp.example.com
    client_id: openase
    client_secret: secret
    redirect_url: http://127.0.0.1:19836/api/v1/auth/oidc/callback
    scopes: [openid, profile, email]
    session_ttl: 8h
    session_idle_ttl: 30m
`)
	if err := os.WriteFile(configPath, []byte(configBody+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	command := newAuthBootstrapCreateLinkCommand(&rootOptions{configFile: configPath})
	stateSvc, err := accesscontrolservice.New(accesscontrolrepo.NewEntRepository(client), dsn, configPath, "")
	if err != nil {
		t.Fatalf("new access control service: %v", err)
	}
	now := time.Now().UTC()
	_, err = stateSvc.Activate(context.Background(), iam.ActiveOIDCConfig{
		IssuerURL:        "https://idp.example.com",
		ClientID:         "openase",
		ClientSecret:     "secret",
		RedirectMode:     iam.OIDCRedirectModeFixed,
		FixedRedirectURL: "http://127.0.0.1:19836/api/v1/auth/oidc/callback",
		Scopes:           []string{"openid", "profile", "email"},
		Claims:           iam.DefaultDraftOIDCConfig().Claims,
		SessionPolicy:    iam.DefaultDraftOIDCConfig().SessionPolicy,
	}, iam.OIDCActivationMetadata{ActivatedAt: &now, Source: "test"})
	if err != nil {
		t.Fatalf("seed active oidc: %v", err)
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)

	err = command.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "local bootstrap authorization is disabled") {
		t.Fatalf("expected oidc-active rejection, got %v", err)
	}
}

func TestAuthBootstrapLoginStoresCLIHumanSessionState(t *testing.T) {
	client, dsn := openCLIEntClient(t)
	_ = client

	t.Setenv("OPENASE_DATABASE_DSN", dsn)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/auth/local-bootstrap/redeem" {
			t.Fatalf("path = %s, want /api/v1/auth/local-bootstrap/redeem", r.URL.Path)
		}
		if r.Header.Get("Origin") != server.URL {
			t.Fatalf("origin = %q, want %q", r.Header.Get("Origin"), server.URL)
		}
		if r.Header.Get("User-Agent") != openASECLIUserAgent {
			t.Fatalf("user-agent = %q, want %q", r.Header.Get("User-Agent"), openASECLIUserAgent)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll(body) error = %v", err)
		}
		var payload localBootstrapRedeemRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("Unmarshal(body) error = %v", err)
		}
		if payload.RequestID == "" || payload.Code == "" || payload.Nonce == "" {
			t.Fatalf("expected redeem materials, got %+v", payload)
		}
		http.SetCookie(w, &http.Cookie{Name: humanSessionCookieHeaderName, Value: "session-token", Path: "/", HttpOnly: true})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "auth_mode":"disabled",
  "authenticated":true,
  "current_auth_method":"local_bootstrap_link",
  "available_auth_methods":["local_bootstrap_link"],
  "csrf_token":"csrf-token",
  "roles":["instance_admin"],
  "permissions":["security_read","security_update"]
}`))
	}))
	defer server.Close()

	sessionPath := filepath.Join(t.TempDir(), "human-session.json")
	command := newAuthBootstrapLoginCommand(&rootOptions{}, authBootstrapLoginDeps{httpClient: server.Client()})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{
		"--control-plane-url", server.URL,
		"--session-file", sessionPath,
		"--format", "json",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	var output localBootstrapLoginOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("Unmarshal(stdout) error = %v", err)
	}
	if !output.Authenticated {
		t.Fatal("expected authenticated login output")
	}
	if output.APIURL != server.URL+"/api/v1" {
		t.Fatalf("api_url = %q, want %q", output.APIURL, server.URL+"/api/v1")
	}
	if output.SessionFile != sessionPath {
		t.Fatalf("session_file = %q, want %q", output.SessionFile, sessionPath)
	}

	state, err := loadHumanSessionState(sessionPath)
	if err != nil {
		t.Fatalf("loadHumanSessionState(%q) error = %v", sessionPath, err)
	}
	if state.SessionToken != "session-token" {
		t.Fatalf("session token = %q, want session-token", state.SessionToken)
	}
	if state.CSRFToken != "csrf-token" {
		t.Fatalf("csrf token = %q, want csrf-token", state.CSRFToken)
	}
	if state.CurrentAuthMethod != "local_bootstrap_link" {
		t.Fatalf("current auth method = %q, want local_bootstrap_link", state.CurrentAuthMethod)
	}
}
