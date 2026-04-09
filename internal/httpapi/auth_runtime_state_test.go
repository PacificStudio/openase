package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
)

func TestCurrentRuntimeAccessControlStateUsesConfiguredDisabledModeWhenStoredStateIsUnreadable(t *testing.T) {
	t.Parallel()

	client := openTestEntClient(t)
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})

	ctx := context.Background()
	seedSvc, err := accesscontrolservice.New(accesscontrolrepo.NewEntRepository(client), "seed-a", "", "")
	if err != nil {
		t.Fatalf("seed access control service: %v", err)
	}
	now := time.Now().UTC()
	if _, err := seedSvc.Activate(ctx, iam.ActiveOIDCConfig{
		IssuerURL:        "https://idp.example.com",
		ClientID:         "openase",
		ClientSecret:     "super-secret",
		RedirectMode:     iam.OIDCRedirectModeFixed,
		FixedRedirectURL: "http://127.0.0.1:19836/api/v1/auth/oidc/callback",
		Scopes:           []string{"openid", "profile", "email"},
		Claims:           iam.DefaultDraftOIDCConfig().Claims,
		SessionPolicy:    iam.DefaultDraftOIDCConfig().SessionPolicy,
	}, iam.OIDCActivationMetadata{ActivatedAt: &now, Source: "test"}); err != nil {
		t.Fatalf("seed active oidc config: %v", err)
	}

	brokenSvc, err := accesscontrolservice.New(accesscontrolrepo.NewEntRepository(client), "seed-b", "", "")
	if err != nil {
		t.Fatalf("broken access control service: %v", err)
	}
	server := NewServer(
		config.ServerConfig{Port: 40023, Host: "127.0.0.1"},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeDisabled}),
		WithInstanceAuthService(brokenSvc),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	runtimeState, err := server.currentRuntimeAccessControlState(server.echo.NewContext(req, rec))
	if err != nil {
		t.Fatalf("currentRuntimeAccessControlState() error = %v", err)
	}
	if runtimeState.AuthMode != iam.AuthModeDisabled {
		t.Fatalf("runtime auth mode = %q, want disabled", runtimeState.AuthMode)
	}
	if runtimeState.LoginRequired {
		t.Fatal("disabled runtime state should not require oidc login")
	}
	if runtimeState.ResolvedOIDCConfig != nil {
		t.Fatalf("resolved oidc config = %#v, want nil", runtimeState.ResolvedOIDCConfig)
	}
}
