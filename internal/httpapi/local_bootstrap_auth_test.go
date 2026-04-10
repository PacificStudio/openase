package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	humanauthrepo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
)

type localBootstrapFixture struct {
	server          *Server
	humanAuth       *humanauthservice.Service
	instanceAuthSvc *accesscontrolservice.Service
}

func newLocalBootstrapFixture(t *testing.T, cfg config.AuthConfig) localBootstrapFixture {
	t.Helper()

	client := openTestEntClient(t)
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})

	repository := humanauthrepo.NewEntRepository(client)
	instanceAuthSvc, err := accesscontrolservice.New(
		accesscontrolrepo.NewEntRepository(client),
		t.Name(),
		"",
		"",
	)
	if err != nil {
		t.Fatalf("new instance auth service: %v", err)
	}
	if cfg.Mode == config.AuthModeOIDC {
		now := time.Now().UTC()
		if _, err := instanceAuthSvc.Activate(context.Background(), testActiveOIDCConfig(cfg), iam.OIDCActivationMetadata{
			ActivatedAt: &now,
			Source:      "test-local-bootstrap",
		}); err != nil {
			t.Fatalf("seed active instance auth state: %v", err)
		}
	}
	humanAuthSvc := humanauthservice.NewService(repository, nil, instanceAuthSvc)
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
		WithHumanAuthConfig(cfg),
		WithInstanceAuthService(instanceAuthSvc),
		WithHumanAuthService(humanAuthSvc, humanauthservice.NewAuthorizer(repository)),
	)
	return localBootstrapFixture{
		server:          server,
		humanAuth:       humanAuthSvc,
		instanceAuthSvc: instanceAuthSvc,
	}
}

func TestLocalBootstrapRedeemCreatesLocalBrowserSession(t *testing.T) {
	t.Parallel()

	fixture := newLocalBootstrapFixture(t, config.AuthConfig{Mode: config.AuthModeDisabled})
	issued, err := fixture.humanAuth.CreateLocalBootstrapRequest(context.Background(), humanauthservice.LocalBootstrapIssueInput{
		RequestedBy: "cli:test",
		Purpose:     "browser_session",
		TTL:         5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateLocalBootstrapRequest() error = %v", err)
	}

	rec := performJSONRequest(
		t,
		fixture.server,
		http.MethodPost,
		"/api/v1/auth/local-bootstrap/redeem",
		`{"request_id":"`+issued.RequestID+`","code":"`+issued.Code+`","nonce":"`+issued.Nonce+`"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload authSessionResponse
	decodeResponse(t, rec, &payload)
	if payload.AuthMode != "disabled" {
		t.Fatalf("auth_mode = %q, want disabled", payload.AuthMode)
	}
	if !payload.LoginRequired {
		t.Fatal("login_required = false, want true")
	}
	if !payload.Authenticated {
		t.Fatal("expected authenticated=true")
	}
	if payload.CurrentAuthMethod != "local_bootstrap_link" {
		t.Fatalf("current_auth_method = %q, want local_bootstrap_link", payload.CurrentAuthMethod)
	}
	assertStringSet(t, payload.AvailableAuthMethods, "local_bootstrap_link")
	assertStringSet(t, payload.Roles, "instance_admin")
	if payload.CSRFToken == "" {
		t.Fatal("expected csrf token")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != humanSessionCookieName || cookies[0].Value == "" {
		t.Fatalf("expected session cookie, got %#v", cookies)
	}
	if cookies[0].Expires.Year() != 9999 {
		t.Fatalf("session cookie expiry = %s, want year 9999", cookies[0].Expires)
	}

	sessionRec := performJSONRequestWithHeaders(t, fixture.server, http.MethodGet, "/api/v1/auth/session", "", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + cookies[0].Value,
		"User-Agent": "LocalBootstrapTest/1.0",
	})
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("expected auth session 200, got %d: %s", sessionRec.Code, sessionRec.Body.String())
	}
	var sessionPayload authSessionResponse
	decodeResponse(t, sessionRec, &sessionPayload)
	if !sessionPayload.LoginRequired {
		t.Fatal("expected local bootstrap auth gate to remain required")
	}
	if !sessionPayload.Authenticated {
		t.Fatal("expected authenticated local auth session")
	}
	if sessionPayload.PrincipalKind != "local_bootstrap" {
		t.Fatalf("principal_kind = %q, want local_bootstrap", sessionPayload.PrincipalKind)
	}
	if sessionPayload.CurrentAuthMethod != "local_bootstrap_link" {
		t.Fatalf("current_auth_method = %q, want local_bootstrap_link", sessionPayload.CurrentAuthMethod)
	}
	assertStringSet(t, sessionPayload.AvailableAuthMethods, "local_bootstrap_link")
	assertStringSet(t, sessionPayload.Roles, "instance_admin")
}

func TestLocalBootstrapProtectedRoutesRequireAuthorizedBrowserSession(t *testing.T) {
	t.Parallel()

	fixture := newLocalBootstrapFixture(t, config.AuthConfig{Mode: config.AuthModeDisabled})

	unauthorized := performJSONRequest(t, fixture.server, http.MethodGet, "/api/v1/auth/sessions", "")
	assertAPIErrorResponse(
		t,
		unauthorized,
		http.StatusUnauthorized,
		"HUMAN_SESSION_REQUIRED",
		humanauthservice.ErrUnauthorized.Error(),
	)

	issued, err := fixture.humanAuth.CreateLocalBootstrapRequest(context.Background(), humanauthservice.LocalBootstrapIssueInput{
		RequestedBy: "cli:test",
		Purpose:     "browser_session",
		TTL:         5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateLocalBootstrapRequest() error = %v", err)
	}

	redeem := performJSONRequest(
		t,
		fixture.server,
		http.MethodPost,
		"/api/v1/auth/local-bootstrap/redeem",
		`{"request_id":"`+issued.RequestID+`","code":"`+issued.Code+`","nonce":"`+issued.Nonce+`"}`,
	)
	if redeem.Code != http.StatusOK {
		t.Fatalf("expected redeem 200, got %d: %s", redeem.Code, redeem.Body.String())
	}
	cookies := redeem.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value == "" {
		t.Fatalf("expected redeemed session cookie, got %#v", cookies)
	}

	authorized := performJSONRequestWithHeaders(t, fixture.server, http.MethodGet, "/api/v1/auth/sessions", "", map[string]string{
		"Cookie":     humanSessionCookieName + "=" + cookies[0].Value,
		"User-Agent": "LocalBootstrapProtectedTest/1.0",
	})
	if authorized.Code != http.StatusOK {
		t.Fatalf("expected authorized protected route 200, got %d: %s", authorized.Code, authorized.Body.String())
	}
}

func TestLocalBootstrapRedeemRejectsExpiredRequest(t *testing.T) {
	t.Parallel()

	fixture := newLocalBootstrapFixture(t, config.AuthConfig{Mode: config.AuthModeDisabled})
	issued, err := fixture.humanAuth.CreateLocalBootstrapRequest(context.Background(), humanauthservice.LocalBootstrapIssueInput{
		RequestedBy: "cli:test",
		Purpose:     "browser_session",
		TTL:         time.Nanosecond,
	})
	if err != nil {
		t.Fatalf("CreateLocalBootstrapRequest() error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)

	rec := performJSONRequest(
		t,
		fixture.server,
		http.MethodPost,
		"/api/v1/auth/local-bootstrap/redeem",
		`{"request_id":"`+issued.RequestID+`","code":"`+issued.Code+`","nonce":"`+issued.Nonce+`"}`,
	)
	assertAPIErrorResponse(t, rec, http.StatusGone, "LOCAL_BOOTSTRAP_EXPIRED", "local bootstrap authorization request expired")
}

func TestLocalBootstrapRedeemRejectsReusedRequest(t *testing.T) {
	t.Parallel()

	fixture := newLocalBootstrapFixture(t, config.AuthConfig{Mode: config.AuthModeDisabled})
	issued, err := fixture.humanAuth.CreateLocalBootstrapRequest(context.Background(), humanauthservice.LocalBootstrapIssueInput{
		RequestedBy: "cli:test",
		Purpose:     "browser_session",
		TTL:         5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateLocalBootstrapRequest() error = %v", err)
	}

	first := performJSONRequest(
		t,
		fixture.server,
		http.MethodPost,
		"/api/v1/auth/local-bootstrap/redeem",
		`{"request_id":"`+issued.RequestID+`","code":"`+issued.Code+`","nonce":"`+issued.Nonce+`"}`,
	)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first redeem 200, got %d: %s", first.Code, first.Body.String())
	}

	second := performJSONRequest(
		t,
		fixture.server,
		http.MethodPost,
		"/api/v1/auth/local-bootstrap/redeem",
		`{"request_id":"`+issued.RequestID+`","code":"`+issued.Code+`","nonce":"`+issued.Nonce+`"}`,
	)
	assertAPIErrorResponse(t, second, http.StatusConflict, "LOCAL_BOOTSTRAP_ALREADY_USED", "local bootstrap authorization request was already used")
}

func TestLocalBootstrapRedeemRejectedWhenOIDCIsActive(t *testing.T) {
	t.Parallel()

	fixture := newLocalBootstrapFixture(t, config.AuthConfig{
		Mode: config.AuthModeOIDC,
		OIDC: config.OIDCConfig{
			IssuerURL:      "https://idp.example.com",
			ClientID:       "openase",
			ClientSecret:   "secret",
			RedirectURL:    "http://127.0.0.1:19836/api/v1/auth/oidc/callback",
			Scopes:         []string{"openid", "profile", "email"},
			SessionTTL:     8 * time.Hour,
			SessionIdleTTL: 30 * time.Minute,
		},
	})

	rec := performJSONRequest(
		t,
		fixture.server,
		http.MethodPost,
		"/api/v1/auth/local-bootstrap/redeem",
		`{"request_id":"`+"550e8400-e29b-41d4-a716-446655440000"+`","code":"code","nonce":"nonce"}`,
	)
	assertAPIErrorResponse(t, rec, http.StatusForbidden, "LOCAL_BOOTSTRAP_DISABLED", "local bootstrap authorization is disabled when OIDC is active")
}
