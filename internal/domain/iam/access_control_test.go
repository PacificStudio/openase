package iam

import (
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
}
