package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
)

func TestAuthBreakGlassDisableOIDCDisablesActiveConfigAndPrintsRepairSteps(t *testing.T) {
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
`)
	if err := os.WriteFile(configPath, []byte(configBody+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	command := newAuthBreakGlassDisableOIDCCommand(&rootOptions{configFile: configPath})
	stateSvc, err := accesscontrolservice.New(accesscontrolrepo.NewEntRepository(client), dsn, configPath, "")
	if err != nil {
		t.Fatalf("new access control service: %v", err)
	}
	now := time.Now().UTC()
	_, err = stateSvc.Activate(context.Background(), iam.ActiveOIDCConfig{
		IssuerURL:            "https://idp.example.com",
		ClientID:             "openase",
		ClientSecret:         "secret",
		RedirectMode:         iam.OIDCRedirectModeFixed,
		FixedRedirectURL:     "http://127.0.0.1:19836/api/v1/auth/oidc/callback",
		Scopes:               []string{"openid", "profile", "email"},
		Claims:               iam.DefaultDraftOIDCConfig().Claims,
		BootstrapAdminEmails: []string{"admin@example.com"},
		SessionPolicy:        iam.DefaultDraftOIDCConfig().SessionPolicy,
	}, iam.OIDCActivationMetadata{ActivatedAt: &now, Source: "test"})
	if err != nil {
		t.Fatalf("seed active oidc: %v", err)
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Disabled active OIDC browser auth") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "openase auth bootstrap create-link --return-to /admin/auth --format text") {
		t.Fatalf("output = %q", output)
	}

	stored, err := client.InstanceAuthConfig.Query().Only(context.Background())
	if err != nil {
		t.Fatalf("query stored auth state: %v", err)
	}
	if stored.Status != iam.AccessControlStatusDraft.String() {
		t.Fatalf("stored status = %q, want draft", stored.Status)
	}
}
