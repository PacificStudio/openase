package iam

import (
	"strings"
	"testing"
	"time"
)

func TestParseAccessControlStateSupportsAbsentDraftAndActive(t *testing.T) {
	t.Parallel()

	absent, err := ParseAccessControlState(AccessControlStateInput{Status: "absent"})
	if err != nil {
		t.Fatalf("ParseAccessControlState(absent) error = %v", err)
	}
	if absent.Status != AccessControlStatusAbsent {
		t.Fatalf("absent status = %q", absent.Status)
	}
	if absent.ConfiguredAuthMode() != AuthModeDisabled {
		t.Fatalf("absent configured auth mode = %q", absent.ConfiguredAuthMode())
	}
	if absent.Draft != nil || absent.Active != nil {
		t.Fatalf("absent state should not contain oidc configs: %+v", absent)
	}

	draft, err := ParseAccessControlState(AccessControlStateInput{
		Status:               "draft",
		IssuerURL:            " https://issuer.example.com ",
		ClientID:             " openase ",
		AllowedEmailDomains:  []string{" Example.com ", "example.com"},
		BootstrapAdminEmails: []string{" Admin@example.com ", "admin@example.com"},
		SessionTTL:           "12h",
		SessionIdleTTL:       "45m",
	})
	if err != nil {
		t.Fatalf("ParseAccessControlState(draft) error = %v", err)
	}
	if draft.Status != AccessControlStatusDraft {
		t.Fatalf("draft status = %q", draft.Status)
	}
	if draft.Draft == nil {
		t.Fatal("draft config = nil")
	}
	if draft.Draft.IssuerURL != "https://issuer.example.com" {
		t.Fatalf("draft issuer = %q", draft.Draft.IssuerURL)
	}
	if draft.Draft.RedirectMode != OIDCRedirectModeAuto {
		t.Fatalf("draft redirect mode = %q", draft.Draft.RedirectMode)
	}
	if draft.Draft.FixedRedirectURL != "" {
		t.Fatalf("draft fixed redirect = %q", draft.Draft.FixedRedirectURL)
	}
	redirectURL, err := draft.Draft.EffectiveRedirectURL("https://desktop.example.com:43123")
	if err != nil {
		t.Fatalf("draft EffectiveRedirectURL() error = %v", err)
	}
	if redirectURL != "https://desktop.example.com:43123/api/v1/auth/oidc/callback" {
		t.Fatalf("draft effective redirect = %q", redirectURL)
	}
	if len(draft.Draft.AllowedEmailDomains) != 1 || draft.Draft.AllowedEmailDomains[0] != "example.com" {
		t.Fatalf("draft allowed domains = %#v", draft.Draft.AllowedEmailDomains)
	}
	if len(draft.Draft.BootstrapAdminEmails) != 1 || draft.Draft.BootstrapAdminEmails[0] != "admin@example.com" {
		t.Fatalf("draft bootstrap admins = %#v", draft.Draft.BootstrapAdminEmails)
	}
	if draft.Draft.SessionPolicy.SessionTTL != 12*time.Hour {
		t.Fatalf("draft session ttl = %s", draft.Draft.SessionPolicy.SessionTTL)
	}
	if draft.Draft.SessionPolicy.SessionIdleTTL != 45*time.Minute {
		t.Fatalf("draft idle ttl = %s", draft.Draft.SessionPolicy.SessionIdleTTL)
	}
	if draft.ConfiguredAuthMode() != AuthModeDisabled {
		t.Fatalf("draft configured auth mode = %q", draft.ConfiguredAuthMode())
	}

	active, err := ParseAccessControlState(AccessControlStateInput{
		Status:               "active",
		IssuerURL:            "https://issuer.example.com",
		ClientID:             "openase",
		ClientSecret:         "super-secret",
		RedirectMode:         "fixed",
		FixedRedirectURL:     "https://openase.example.com/api/v1/auth/oidc/callback",
		Scopes:               []string{"openid", "profile", "email"},
		AllowedEmailDomains:  []string{"example.com"},
		BootstrapAdminEmails: []string{"admin@example.com"},
	})
	if err != nil {
		t.Fatalf("ParseAccessControlState(active) error = %v", err)
	}
	if active.Status != AccessControlStatusActive {
		t.Fatalf("active status = %q", active.Status)
	}
	if active.Active == nil {
		t.Fatal("active config = nil")
	}
	if active.Active.ClientSecret != "super-secret" {
		t.Fatalf("active client secret = %q", active.Active.ClientSecret)
	}
	if active.Active.RedirectMode != OIDCRedirectModeFixed {
		t.Fatalf("active redirect mode = %q", active.Active.RedirectMode)
	}
	if active.Active.FixedRedirectURL != "https://openase.example.com/api/v1/auth/oidc/callback" {
		t.Fatalf("active fixed redirect = %q", active.Active.FixedRedirectURL)
	}
	if active.ConfiguredAuthMode() != AuthModeOIDC {
		t.Fatalf("active configured auth mode = %q", active.ConfiguredAuthMode())
	}

	absentRuntime := ResolveRuntimeAccessControlState(absent)
	if absentRuntime.AuthMode != AuthModeDisabled {
		t.Fatalf("absent runtime auth mode = %q", absentRuntime.AuthMode)
	}
	if absentRuntime.LoginRequired {
		t.Fatal("absent runtime should not require login")
	}
	if absentRuntime.PrincipalKind != RuntimePrincipalKindLocal {
		t.Fatalf("absent runtime principal kind = %q", absentRuntime.PrincipalKind)
	}
	if absentRuntime.ResolvedOIDCConfig != nil {
		t.Fatalf("absent runtime resolved oidc config = %#v, want nil", absentRuntime.ResolvedOIDCConfig)
	}

	activeRuntime := ResolveRuntimeAccessControlState(active)
	if activeRuntime.AuthMode != AuthModeOIDC {
		t.Fatalf("active runtime auth mode = %q", activeRuntime.AuthMode)
	}
	if !activeRuntime.LoginRequired {
		t.Fatal("active runtime should require login")
	}
	if !activeRuntime.SessionGovernanceEnabled {
		t.Fatal("active runtime should enable session governance")
	}
	if activeRuntime.PrincipalKind != RuntimePrincipalKindHuman {
		t.Fatalf("active runtime principal kind = %q", activeRuntime.PrincipalKind)
	}
	if activeRuntime.ResolvedOIDCConfig == nil || activeRuntime.ResolvedOIDCConfig.ClientID != "openase" {
		t.Fatalf("active runtime oidc config = %#v", activeRuntime.ResolvedOIDCConfig)
	}
}

func TestParseAccessControlStateRejectsInvalidActiveConfig(t *testing.T) {
	t.Parallel()

	_, err := ParseAccessControlState(AccessControlStateInput{
		Status:           "active",
		IssuerURL:        "https://issuer.example.com",
		ClientID:         "openase",
		RedirectMode:     "fixed",
		FixedRedirectURL: "https://openase.example.com/api/v1/auth/oidc/callback",
		SessionTTL:       "30m",
		SessionIdleTTL:   "2h",
	})
	if err == nil {
		t.Fatal("expected ParseAccessControlState(active) to fail")
	}

	_, err = ParseAccessControlState(AccessControlStateInput{
		Status:           "active",
		ClientID:         "openase",
		ClientSecret:     "secret",
		RedirectMode:     "fixed",
		FixedRedirectURL: "https://openase.example.com/api/v1/auth/oidc/callback",
		SessionTTL:       "30m",
		SessionIdleTTL:   "15m",
	})
	if err == nil || !strings.Contains(err.Error(), "issuer_url is required") {
		t.Fatalf("expected missing issuer error, got %v", err)
	}
}

func TestAccessControlStatusAndHelperNormalization(t *testing.T) {
	t.Parallel()

	status, err := ParseAccessControlStatus(" draft ")
	if err != nil {
		t.Fatalf("ParseAccessControlStatus(draft) error = %v", err)
	}
	if status != AccessControlStatusDraft {
		t.Fatalf("status = %q, want %q", status, AccessControlStatusDraft)
	}
	if status.String() != "draft" {
		t.Fatalf("status.String() = %q", status.String())
	}
	if _, err := ParseAccessControlStatus("mystery"); err == nil {
		t.Fatal("expected unsupported access control status error")
	}

	checkedAt := time.Date(2026, time.April, 8, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	activatedAt := time.Date(2026, time.April, 8, 13, 0, 0, 0, time.FixedZone("UTC-5", -5*60*60))
	state, err := ParseAccessControlState(AccessControlStateInput{
		Status: "draft",
		Validation: OIDCValidationMetadataInput{
			CheckedAt: &checkedAt,
			Warnings:  []string{" B ", "a", "b"},
		},
		Activation: OIDCActivationMetadataInput{
			ActivatedAt: &activatedAt,
			ActivatedBy: " admin@example.com ",
			Source:      " test ",
			Message:     " enabled ",
		},
	})
	if err != nil {
		t.Fatalf("ParseAccessControlState(draft metadata) error = %v", err)
	}
	if state.Validation.Status != "not_tested" {
		t.Fatalf("validation status = %q", state.Validation.Status)
	}
	if state.Validation.Message != "No OIDC validation has been recorded yet." {
		t.Fatalf("validation message = %q", state.Validation.Message)
	}
	if state.Validation.CheckedAt == nil || state.Validation.CheckedAt.Location() != time.UTC {
		t.Fatalf("checked_at = %#v, want UTC clone", state.Validation.CheckedAt)
	}
	if got := strings.Join(state.Validation.Warnings, ","); got != "B,a,b" {
		t.Fatalf("validation warnings = %q", got)
	}
	if state.Activation.ActivatedAt == nil || state.Activation.ActivatedAt.Location() != time.UTC {
		t.Fatalf("activated_at = %#v, want UTC clone", state.Activation.ActivatedAt)
	}
	if state.Activation.ActivatedBy != "admin@example.com" {
		t.Fatalf("activated_by = %q", state.Activation.ActivatedBy)
	}
	if state.Activation.Source != "test" {
		t.Fatalf("source = %q", state.Activation.Source)
	}
	if state.Activation.Message != "enabled" {
		t.Fatalf("message = %q", state.Activation.Message)
	}

	if cloneTime(nil) != nil {
		t.Fatal("cloneTime(nil) should return nil")
	}

	if got := normalizeList(nil, false, []string{"openid"}); len(got) != 1 || got[0] != "openid" {
		t.Fatalf("normalizeList fallback = %#v", got)
	}
	if got := normalizeList([]string{" B ", "a", "b", ""}, true, nil); strings.Join(got, ",") != "a,b" {
		t.Fatalf("normalizeList lower+compact = %#v", got)
	}
}

func TestParseAccessControlStateCoversDraftAndActiveErrors(t *testing.T) {
	t.Parallel()

	if _, err := ParseAccessControlState(AccessControlStateInput{Status: "mystery"}); err == nil {
		t.Fatal("expected ParseAccessControlState to reject unknown status")
	}

	_, err := ParseAccessControlState(AccessControlStateInput{
		Status:         "draft",
		SessionTTL:     "nope",
		SessionIdleTTL: "5m",
	})
	if err == nil || !strings.Contains(err.Error(), "session_ttl must be a valid duration") {
		t.Fatalf("expected session_ttl parse error, got %v", err)
	}

	_, err = ParseAccessControlState(AccessControlStateInput{
		Status:         "draft",
		SessionTTL:     "1h",
		SessionIdleTTL: "0s",
	})
	if err != nil {
		t.Fatalf("expected zero idle ttl to be accepted, got %v", err)
	}

	_, err = ParseAccessControlState(AccessControlStateInput{
		Status:         "draft",
		SessionTTL:     "1h",
		SessionIdleTTL: "-1s",
	})
	if err == nil || !strings.Contains(err.Error(), "session_idle_ttl must not be negative") {
		t.Fatalf("expected session_idle_ttl >= 0 error, got %v", err)
	}

	_, err = parseActiveOIDCConfig(AccessControlStateInput{
		IssuerURL:      "https://issuer.example.com",
		ClientID:       "openase",
		ClientSecret:   "secret",
		SessionTTL:     "bad",
		SessionIdleTTL: "5m",
	})
	if err == nil || !strings.Contains(err.Error(), "session_ttl must be a valid duration") {
		t.Fatalf("expected parseActiveOIDCConfig duration error, got %v", err)
	}

	base := AccessControlStateInput{
		IssuerURL:        "https://issuer.example.com",
		ClientID:         "openase",
		ClientSecret:     "secret",
		RedirectMode:     "fixed",
		FixedRedirectURL: "https://openase.example.com/callback",
		Scopes:           []string{"openid"},
		SessionTTL:       "2h",
		SessionIdleTTL:   "15m",
	}
	cases := []struct {
		name   string
		mutate func(*AccessControlStateInput)
		want   string
	}{
		{
			name:   "missing issuer",
			mutate: func(input *AccessControlStateInput) { input.IssuerURL = "" },
			want:   "issuer_url is required",
		},
		{
			name:   "missing client id",
			mutate: func(input *AccessControlStateInput) { input.ClientID = "" },
			want:   "client_id is required",
		},
		{
			name:   "missing client secret",
			mutate: func(input *AccessControlStateInput) { input.ClientSecret = "" },
			want:   "client_secret is required",
		},
		{
			name:   "missing fixed redirect",
			mutate: func(input *AccessControlStateInput) { input.FixedRedirectURL = "" },
			want:   "fixed_redirect_url is required",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			input := base
			tc.mutate(&input)
			_, err := parseActiveOIDCConfig(input)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestOIDCRedirectModeHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		raw              string
		fixedRedirectURL string
		want             OIDCRedirectMode
		wantErr          string
	}{
		{
			name: "empty defaults to auto",
			want: OIDCRedirectModeAuto,
		},
		{
			name:             "empty legacy fixed url infers fixed",
			fixedRedirectURL: "https://openase.example.com/callback",
			want:             OIDCRedirectModeFixed,
		},
		{
			name: "explicit auto",
			raw:  " AUTO ",
			want: OIDCRedirectModeAuto,
		},
		{
			name: "explicit fixed",
			raw:  "fixed",
			want: OIDCRedirectModeFixed,
		},
		{
			name:    "invalid mode",
			raw:     "desktop",
			wantErr: "redirect_mode must be one of auto, fixed",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mode, err := ParseOIDCRedirectMode(tc.raw, tc.fixedRedirectURL)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseOIDCRedirectMode() error = %v", err)
			}
			if mode != tc.want {
				t.Fatalf("mode = %q, want %q", mode, tc.want)
			}
			if mode.String() != string(tc.want) {
				t.Fatalf("mode.String() = %q, want %q", mode.String(), tc.want)
			}
		})
	}
}

func TestDraftAndActiveOIDCRedirectHelpers(t *testing.T) {
	t.Parallel()

	draft, err := parseDraftOIDCConfig(AccessControlStateInput{
		RedirectMode:     "",
		RedirectURL:      " https://legacy.example.com/oidc/callback ",
		FixedRedirectURL: "https://preferred.example.com/oidc/callback",
		EmailClaim:       " custom-email ",
		NameClaim:        " custom-name ",
		UsernameClaim:    " custom-user ",
		GroupsClaim:      " custom-groups ",
		Scopes:           []string{},
	})
	if err != nil {
		t.Fatalf("parseDraftOIDCConfig() error = %v", err)
	}
	if draft.RedirectMode != OIDCRedirectModeFixed {
		t.Fatalf("draft redirect mode = %q", draft.RedirectMode)
	}
	if draft.FixedRedirectURL != "https://preferred.example.com/oidc/callback" {
		t.Fatalf("draft fixed redirect = %q", draft.FixedRedirectURL)
	}
	if draft.Claims.EmailClaim != "custom-email" ||
		draft.Claims.NameClaim != "custom-name" ||
		draft.Claims.UsernameClaim != "custom-user" ||
		draft.Claims.GroupsClaim != "custom-groups" {
		t.Fatalf("draft claims = %#v", draft.Claims)
	}
	if got := strings.Join(draft.Scopes, ","); got != "openid,profile,email,groups" {
		t.Fatalf("draft scopes = %q", got)
	}

	if fixedURL, err := draft.EffectiveRedirectURL("https://ignored.example.com"); err != nil {
		t.Fatalf("draft fixed EffectiveRedirectURL() error = %v", err)
	} else if fixedURL != "https://preferred.example.com/oidc/callback" {
		t.Fatalf("draft fixed EffectiveRedirectURL() = %q", fixedURL)
	}

	autoDraft := DefaultDraftOIDCConfig()
	if autoDraft.SessionPolicy.SessionTTL != 0 || autoDraft.SessionPolicy.SessionIdleTTL != 0 {
		t.Fatalf("default draft session policy = %#v", autoDraft.SessionPolicy)
	}
	autoURL, err := autoDraft.EffectiveRedirectURL("https://desktop.example.com:43123")
	if err != nil {
		t.Fatalf("draft auto EffectiveRedirectURL() error = %v", err)
	}
	if autoURL != "https://desktop.example.com:43123/api/v1/auth/oidc/callback" {
		t.Fatalf("draft auto EffectiveRedirectURL() = %q", autoURL)
	}

	active, err := parseActiveOIDCConfig(AccessControlStateInput{
		IssuerURL:      "https://issuer.example.com",
		ClientID:       "openase",
		ClientSecret:   "secret",
		RedirectMode:   "auto",
		SessionTTL:     "2h",
		SessionIdleTTL: "15m",
	})
	if err != nil {
		t.Fatalf("parseActiveOIDCConfig(auto) error = %v", err)
	}
	activeURL, err := active.EffectiveRedirectURL("https://proxy.example.com")
	if err != nil {
		t.Fatalf("active auto EffectiveRedirectURL() error = %v", err)
	}
	if activeURL != "https://proxy.example.com/api/v1/auth/oidc/callback" {
		t.Fatalf("active auto EffectiveRedirectURL() = %q", activeURL)
	}

	activeFixed, err := parseActiveOIDCConfig(AccessControlStateInput{
		IssuerURL:        "https://issuer.example.com",
		ClientID:         "openase",
		ClientSecret:     "secret",
		RedirectMode:     "fixed",
		FixedRedirectURL: "https://openase.example.com/api/v1/auth/oidc/callback",
		SessionTTL:       "2h",
		SessionIdleTTL:   "15m",
	})
	if err != nil {
		t.Fatalf("parseActiveOIDCConfig(fixed) error = %v", err)
	}
	if fixedURL, err := activeFixed.EffectiveRedirectURL("https://ignored.example.com"); err != nil {
		t.Fatalf("active fixed EffectiveRedirectURL() error = %v", err)
	} else if fixedURL != "https://openase.example.com/api/v1/auth/oidc/callback" {
		t.Fatalf("active fixed EffectiveRedirectURL() = %q", fixedURL)
	}

	if _, err := parseDraftOIDCConfig(AccessControlStateInput{RedirectMode: "bad-mode"}); err == nil ||
		!strings.Contains(err.Error(), "redirect_mode must be one of auto, fixed") {
		t.Fatalf("expected draft redirect mode error, got %v", err)
	}

	if _, err := parseActiveOIDCConfig(AccessControlStateInput{
		IssuerURL:        "https://issuer.example.com",
		ClientID:         "openase",
		ClientSecret:     "secret",
		RedirectMode:     "fixed",
		FixedRedirectURL: "relative/path",
		SessionTTL:       "2h",
		SessionIdleTTL:   "15m",
	}); err == nil || !strings.Contains(err.Error(), "fixed_redirect_url must be a valid absolute URL") {
		t.Fatalf("expected fixed redirect validation error, got %v", err)
	}
}

func TestOIDCRedirectURLParsingHelpers(t *testing.T) {
	t.Parallel()

	parsed, err := parseAbsoluteURL(" https://OpenASE.example.com:9443/base ")
	if err != nil {
		t.Fatalf("parseAbsoluteURL(valid) error = %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "OpenASE.example.com:9443" {
		t.Fatalf("parseAbsoluteURL(valid) = %#v", parsed)
	}

	if _, err := parseAbsoluteURL("/relative/path"); err == nil ||
		!strings.Contains(err.Error(), "absolute url is required") {
		t.Fatalf("expected parseAbsoluteURL(relative) error, got %v", err)
	}
	if _, err := parseAbsoluteURL("https://example.com/%zz"); err == nil {
		t.Fatal("expected parseAbsoluteURL(parse failure) to fail")
	}

	autoURL, err := resolveAutoRedirectURL("HTTPS://OpenASE.example.com:9443/base")
	if err != nil {
		t.Fatalf("resolveAutoRedirectURL(valid) error = %v", err)
	}
	if autoURL != "https://OpenASE.example.com:9443/api/v1/auth/oidc/callback" {
		t.Fatalf("resolveAutoRedirectURL(valid) = %q", autoURL)
	}

	if _, err := resolveAutoRedirectURL("not-a-url"); err == nil ||
		!strings.Contains(err.Error(), "external base url must be a valid absolute URL") {
		t.Fatalf("expected resolveAutoRedirectURL(invalid) error, got %v", err)
	}
}
