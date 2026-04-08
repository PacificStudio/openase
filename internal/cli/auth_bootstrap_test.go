package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	_, dsn := openCLIEntClient(t)

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configBody := strings.TrimSpace(`
database:
  dsn: ` + dsn + `
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
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)

	err := command.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "local bootstrap authorization is disabled") {
		t.Fatalf("expected oidc-active rejection, got %v", err)
	}
}
