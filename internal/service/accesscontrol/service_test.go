package accesscontrol

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	repo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
)

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return testPostgres.NewIsolatedEntClient(t)
}

func TestServicePersistsAbsentDraftAndActiveStates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	defer func() {
		_ = client.Close()
	}()

	service, err := New(repo.NewEntRepository(client), "test-cipher-seed", "", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	absent, err := service.Read(ctx)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if absent.State.Status != iam.AccessControlStatusAbsent {
		t.Fatalf("initial status = %q, want absent", absent.State.Status)
	}
	absentRuntime, err := service.RuntimeState(ctx)
	if err != nil {
		t.Fatalf("RuntimeState(absent) error = %v", err)
	}
	if absentRuntime.LoginRequired {
		t.Fatal("absent runtime state should not require login")
	}

	draftResult, err := service.SaveDraft(ctx, iam.DraftOIDCConfig{
		IssuerURL:            "https://issuer.example.com",
		ClientID:             "openase",
		ClientSecret:         "super-secret",
		RedirectMode:         iam.OIDCRedirectModeFixed,
		FixedRedirectURL:     "https://openase.example.com/api/v1/auth/oidc/callback",
		Scopes:               []string{"openid", "profile", "email"},
		Claims:               iam.DefaultDraftOIDCConfig().Claims,
		AllowedEmailDomains:  []string{"example.com"},
		BootstrapAdminEmails: []string{"admin@example.com"},
		SessionPolicy:        iam.DefaultDraftOIDCConfig().SessionPolicy,
	})
	if err != nil {
		t.Fatalf("SaveDraft() error = %v", err)
	}
	if draftResult.State.Status != iam.AccessControlStatusDraft {
		t.Fatalf("draft status = %q, want draft", draftResult.State.Status)
	}
	draftRuntime, err := service.RuntimeState(ctx)
	if err != nil {
		t.Fatalf("RuntimeState(draft) error = %v", err)
	}
	if draftRuntime.LoginRequired {
		t.Fatal("draft runtime state should stay in disabled mode")
	}
	storedDraft, err := client.InstanceAuthConfig.Query().Only(ctx)
	if err != nil {
		t.Fatalf("query stored draft: %v", err)
	}
	if storedDraft.Status != iam.AccessControlStatusDraft.String() {
		t.Fatalf("stored draft status = %q", storedDraft.Status)
	}
	if storedDraft.ClientSecretEncrypted == nil {
		t.Fatal("expected encrypted secret to be stored")
	}
	if storedDraft.RedirectMode != iam.OIDCRedirectModeFixed.String() {
		t.Fatalf("stored draft redirect mode = %q", storedDraft.RedirectMode)
	}
	if storedDraft.RedirectURL != "https://openase.example.com/api/v1/auth/oidc/callback" {
		t.Fatalf("stored draft fixed redirect = %q", storedDraft.RedirectURL)
	}
	if strings.Contains(storedDraft.ClientSecretEncrypted.Ciphertext, "super-secret") {
		t.Fatalf("expected encrypted ciphertext, got %+v", storedDraft.ClientSecretEncrypted)
	}

	now := time.Now().UTC()
	activeResult, err := service.Activate(ctx, iam.ActiveOIDCConfig{
		IssuerURL:            "https://issuer.example.com",
		ClientID:             "openase",
		ClientSecret:         "super-secret",
		RedirectMode:         iam.OIDCRedirectModeFixed,
		FixedRedirectURL:     "https://openase.example.com/api/v1/auth/oidc/callback",
		Scopes:               []string{"openid", "profile", "email"},
		Claims:               iam.DefaultDraftOIDCConfig().Claims,
		AllowedEmailDomains:  []string{"example.com"},
		BootstrapAdminEmails: []string{"admin@example.com"},
		SessionPolicy:        iam.DefaultDraftOIDCConfig().SessionPolicy,
	}, iam.OIDCActivationMetadata{ActivatedAt: &now, Source: "test"})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if activeResult.State.Status != iam.AccessControlStatusActive {
		t.Fatalf("active status = %q, want active", activeResult.State.Status)
	}
	activeRuntime, err := service.RuntimeState(ctx)
	if err != nil {
		t.Fatalf("RuntimeState(active) error = %v", err)
	}
	if !activeRuntime.LoginRequired || activeRuntime.ResolvedOIDCConfig == nil {
		t.Fatalf("active runtime state = %#v, want oidc login required", activeRuntime)
	}
	storedActive, err := client.InstanceAuthConfig.Query().Only(ctx)
	if err != nil {
		t.Fatalf("query stored active: %v", err)
	}
	if storedActive.Status != iam.AccessControlStatusActive.String() {
		t.Fatalf("stored active status = %q", storedActive.Status)
	}

	disabledResult, err := service.Disable(ctx)
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if disabledResult.State.Status != iam.AccessControlStatusDraft {
		t.Fatalf("disabled status = %q, want draft", disabledResult.State.Status)
	}
	disabledRuntime, err := service.RuntimeState(ctx)
	if err != nil {
		t.Fatalf("RuntimeState(disabled) error = %v", err)
	}
	if disabledRuntime.LoginRequired {
		t.Fatal("disabled runtime state should stop requiring login immediately")
	}
	storedDisabled, err := client.InstanceAuthConfig.Query().Only(ctx)
	if err != nil {
		t.Fatalf("query stored disabled draft: %v", err)
	}
	if storedDisabled.Status != iam.AccessControlStatusDraft.String() {
		t.Fatalf("stored disabled status = %q", storedDisabled.Status)
	}
}

func TestServiceImportsLegacyYAMLOIDCConfigIntoDBWhenRepositoryIsEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	defer func() {
		_ = client.Close()
	}()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configBody := strings.TrimSpace(`
auth:
  mode: oidc
  oidc:
    issuer_url: https://idp.example.com
    client_id: openase
    client_secret: legacy-secret
    redirect_url: http://127.0.0.1:19836/api/v1/auth/oidc/callback
    scopes: [openid, profile, email]
    bootstrap_admin_emails: [admin@example.com]
    session_ttl: 8h
    session_idle_ttl: 30m
  last_validation:
    status: ok
    message: Imported from legacy YAML
`)
	if err := os.WriteFile(configPath, []byte(configBody+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	service, err := New(repo.NewEntRepository(client), "test-cipher-seed", configPath, "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := service.Read(ctx)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.StorageLocation != storageLocationDB {
		t.Fatalf("storage location = %q, want %q", result.StorageLocation, storageLocationDB)
	}
	if result.State.Status != iam.AccessControlStatusActive {
		t.Fatalf("status = %q, want active", result.State.Status)
	}
	if result.State.Validation.Status != "ok" {
		t.Fatalf("validation.status = %q, want ok", result.State.Validation.Status)
	}

	stored, err := client.InstanceAuthConfig.Query().Only(ctx)
	if err != nil {
		t.Fatalf("query imported record: %v", err)
	}
	if stored.Status != iam.AccessControlStatusActive.String() {
		t.Fatalf("stored status = %q, want active", stored.Status)
	}
	if stored.ClientSecretEncrypted == nil || strings.TrimSpace(stored.ClientSecretEncrypted.Ciphertext) == "" {
		t.Fatalf("expected encrypted imported client secret, got %+v", stored.ClientSecretEncrypted)
	}
}

func TestServiceReadsStoredSecretAcrossFreshServiceInstances(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	defer func() {
		_ = client.Close()
	}()

	const seed = "shared-seed"
	first, err := New(repo.NewEntRepository(client), seed, "", "")
	if err != nil {
		t.Fatalf("first New() error = %v", err)
	}
	now := time.Now().UTC()
	if _, err := first.Activate(ctx, iam.ActiveOIDCConfig{
		IssuerURL:        "https://idp.example.com",
		ClientID:         "openase",
		ClientSecret:     "super-secret",
		RedirectMode:     iam.OIDCRedirectModeFixed,
		FixedRedirectURL: "http://127.0.0.1:19836/api/v1/auth/oidc/callback",
		Scopes:           []string{"openid", "profile", "email"},
		Claims:           iam.DefaultDraftOIDCConfig().Claims,
		SessionPolicy:    iam.DefaultDraftOIDCConfig().SessionPolicy,
	}, iam.OIDCActivationMetadata{ActivatedAt: &now, Source: "test"}); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	second, err := New(repo.NewEntRepository(client), seed, "", "")
	if err != nil {
		t.Fatalf("second New() error = %v", err)
	}
	result, err := second.Read(ctx)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.State.Status != iam.AccessControlStatusActive {
		t.Fatalf("status = %q, want active", result.State.Status)
	}
	if result.State.Active == nil || result.State.Active.ClientSecret != "super-secret" {
		t.Fatalf("active config = %#v", result.State.Active)
	}
}
