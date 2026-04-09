package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
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
		cfg,
	)
	if err != nil {
		t.Fatalf("new instance auth service: %v", err)
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
	if !payload.Authenticated {
		t.Fatal("expected authenticated=true")
	}
	assertStringSet(t, payload.Roles, "instance_admin")
	if payload.CSRFToken == "" {
		t.Fatal("expected csrf token")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != humanSessionCookieName || cookies[0].Value == "" {
		t.Fatalf("expected session cookie, got %#v", cookies)
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
	if !sessionPayload.Authenticated {
		t.Fatal("expected authenticated local auth session")
	}
	assertStringSet(t, sessionPayload.Roles, "instance_admin")
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
