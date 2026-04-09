package accesscontrol

import (
	"strings"

	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
)

func RuntimeFallbackInputFromConfig(cfg config.AuthConfig) (iam.AccessControlStateInput, bool) {
	input := iam.AccessControlStateInput{
		Status: iam.AccessControlStatusAbsent.String(),
		Validation: iam.OIDCValidationMetadataInput{
			Status:   "not_tested",
			Message:  "No OIDC validation has been recorded yet.",
			Warnings: []string{},
		},
	}

	switch cfg.Mode {
	case config.AuthModeOIDC:
		input.Status = iam.AccessControlStatusActive.String()
	case "", config.AuthModeDisabled:
		if !hasRuntimeOIDCValues(cfg.OIDC) {
			return input, false
		}
		input.Status = iam.AccessControlStatusDraft.String()
	default:
		if !hasRuntimeOIDCValues(cfg.OIDC) {
			return input, false
		}
		input.Status = iam.AccessControlStatusDraft.String()
	}

	input.IssuerURL = strings.TrimSpace(cfg.OIDC.IssuerURL)
	input.ClientID = strings.TrimSpace(cfg.OIDC.ClientID)
	input.ClientSecret = strings.TrimSpace(cfg.OIDC.ClientSecret)
	input.RedirectURL = strings.TrimSpace(cfg.OIDC.RedirectURL)
	input.FixedRedirectURL = strings.TrimSpace(cfg.OIDC.RedirectURL)
	input.Scopes = append([]string(nil), cfg.OIDC.Scopes...)
	input.EmailClaim = strings.TrimSpace(cfg.OIDC.EmailClaim)
	input.NameClaim = strings.TrimSpace(cfg.OIDC.NameClaim)
	input.UsernameClaim = strings.TrimSpace(cfg.OIDC.UsernameClaim)
	input.GroupsClaim = strings.TrimSpace(cfg.OIDC.GroupsClaim)
	input.AllowedEmailDomains = append([]string(nil), cfg.OIDC.AllowedEmailDomains...)
	input.BootstrapAdminEmails = append([]string(nil), cfg.OIDC.BootstrapAdminEmails...)
	if cfg.OIDC.SessionTTL > 0 {
		input.SessionTTL = cfg.OIDC.SessionTTL.String()
	}
	if cfg.OIDC.SessionIdleTTL > 0 {
		input.SessionIdleTTL = cfg.OIDC.SessionIdleTTL.String()
	}
	return input, true
}

func hasRuntimeOIDCValues(raw config.OIDCConfig) bool {
	return strings.TrimSpace(raw.IssuerURL) != "" ||
		strings.TrimSpace(raw.ClientID) != "" ||
		strings.TrimSpace(raw.ClientSecret) != "" ||
		strings.TrimSpace(raw.RedirectURL) != "" ||
		len(raw.Scopes) > 0 ||
		strings.TrimSpace(raw.EmailClaim) != "" ||
		strings.TrimSpace(raw.NameClaim) != "" ||
		strings.TrimSpace(raw.UsernameClaim) != "" ||
		strings.TrimSpace(raw.GroupsClaim) != "" ||
		len(raw.AllowedEmailDomains) > 0 ||
		len(raw.BootstrapAdminEmails) > 0 ||
		raw.SessionTTL > 0 ||
		raw.SessionIdleTTL > 0
}
