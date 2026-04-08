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
	if draft.Draft.RedirectURL != setupDefaultOIDCRedirectURL {
		t.Fatalf("draft redirect = %q", draft.Draft.RedirectURL)
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
		RedirectURL:          "https://openase.example.com/api/v1/auth/oidc/callback",
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
		Status:         "active",
		IssuerURL:      "https://issuer.example.com",
		ClientID:       "openase",
		RedirectURL:    "https://openase.example.com/api/v1/auth/oidc/callback",
		SessionTTL:     "30m",
		SessionIdleTTL: "2h",
	})
	if err == nil {
		t.Fatal("expected ParseAccessControlState(active) to fail")
	}

	_, err = ParseAccessControlState(AccessControlStateInput{
		Status:         "active",
		ClientID:       "openase",
		ClientSecret:   "secret",
		RedirectURL:    "https://openase.example.com/api/v1/auth/oidc/callback",
		SessionTTL:     "30m",
		SessionIdleTTL: "15m",
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
	if err == nil || !strings.Contains(err.Error(), "session_idle_ttl must be greater than zero") {
		t.Fatalf("expected session_idle_ttl > 0 error, got %v", err)
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
		IssuerURL:      "https://issuer.example.com",
		ClientID:       "openase",
		ClientSecret:   "secret",
		RedirectURL:    "https://openase.example.com/callback",
		Scopes:         []string{"openid"},
		SessionTTL:     "2h",
		SessionIdleTTL: "15m",
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
