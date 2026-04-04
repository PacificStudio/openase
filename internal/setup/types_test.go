package setup

import "testing"

func TestParseAuthInputSupportsDisabledAndOIDC(t *testing.T) {
	t.Run("disabled-default", func(t *testing.T) {
		auth, err := ParseAuthInput(RawAuthInput{})
		if err != nil {
			t.Fatalf("ParseAuthInput() error = %v", err)
		}
		if auth.Mode != AuthModeDisabled || auth.OIDC != nil {
			t.Fatalf("ParseAuthInput() = %+v", auth)
		}
	})

	t.Run("oidc", func(t *testing.T) {
		auth, err := ParseAuthInput(RawAuthInput{
			Mode: string(AuthModeOIDC),
			OIDC: &RawOIDCInput{
				IssuerURL:            "https://login.microsoftonline.com/tenant/v2.0",
				ClientID:             "openase",
				ClientSecret:         "secret",
				RedirectURL:          DefaultOIDCRedirectURL,
				Scopes:               DefaultOIDCScopes,
				BootstrapAdminEmails: "admin@example.com, owner@example.com",
				SessionTTL:           DefaultOIDCSessionTTL,
				SessionIdleTTL:       DefaultOIDCIdleTTL,
			},
		})
		if err != nil {
			t.Fatalf("ParseAuthInput() error = %v", err)
		}
		if auth.Mode != AuthModeOIDC || auth.OIDC == nil {
			t.Fatalf("ParseAuthInput() = %+v", auth)
		}
		if got := len(auth.OIDC.Scopes); got != 4 {
			t.Fatalf("scope count = %d", got)
		}
		if got := len(auth.OIDC.BootstrapAdminEmails); got != 2 {
			t.Fatalf("bootstrap admin email count = %d", got)
		}
	})
}

func TestParseAuthInputRejectsInvalidOIDC(t *testing.T) {
	_, err := ParseAuthInput(RawAuthInput{
		Mode: string(AuthModeOIDC),
		OIDC: &RawOIDCInput{
			IssuerURL:      "https://example.com",
			ClientID:       "openase",
			ClientSecret:   "",
			RedirectURL:    DefaultOIDCRedirectURL,
			Scopes:         DefaultOIDCScopes,
			SessionTTL:     "30m",
			SessionIdleTTL: "1h",
		},
	})
	if err == nil || err.Error() != "auth.oidc.client_secret must not be empty when auth.mode=oidc" {
		t.Fatalf("ParseAuthInput() error = %v", err)
	}
}
