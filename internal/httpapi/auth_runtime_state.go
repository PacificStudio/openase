package httpapi

import (
	"net/http"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	"github.com/labstack/echo/v4"
)

func (s *Server) currentRuntimeAccessControlState(c echo.Context) (iam.RuntimeAccessControlState, error) {
	if s.instanceAuthService == nil {
		return bootstrapRuntimeAccessControlState(s.auth), nil
	}
	stored, err := s.readInstanceAccessControlState(c)
	if err != nil {
		return iam.RuntimeAccessControlState{}, err
	}
	return iam.ResolveRuntimeAccessControlState(stored.State), nil
}

func bootstrapRuntimeAccessControlState(cfg config.AuthConfig) iam.RuntimeAccessControlState {
	state := iam.RuntimeAccessControlState{
		AuthMode:      iam.AuthModeDisabled,
		PrincipalKind: iam.RuntimePrincipalKindLocal,
	}
	if cfg.Mode != config.AuthModeOIDC {
		return state
	}

	state.AuthMode = iam.AuthModeOIDC
	state.LoginRequired = true
	state.PrincipalKind = iam.RuntimePrincipalKindHuman
	state.SessionGovernanceEnabled = true

	if strings.TrimSpace(cfg.OIDC.IssuerURL) == "" ||
		strings.TrimSpace(cfg.OIDC.ClientID) == "" ||
		strings.TrimSpace(cfg.OIDC.ClientSecret) == "" {
		return state
	}

	active, err := activeOIDCConfigFromDraft(iam.DraftOIDCConfig{
		IssuerURL:        strings.TrimSpace(cfg.OIDC.IssuerURL),
		ClientID:         strings.TrimSpace(cfg.OIDC.ClientID),
		ClientSecret:     strings.TrimSpace(cfg.OIDC.ClientSecret),
		RedirectMode:     iam.OIDCRedirectModeFixed,
		FixedRedirectURL: strings.TrimSpace(cfg.OIDC.RedirectURL),
		Scopes:           append([]string(nil), cfg.OIDC.Scopes...),
		Claims: iam.OIDCClaims{
			EmailClaim:    strings.TrimSpace(cfg.OIDC.EmailClaim),
			NameClaim:     strings.TrimSpace(cfg.OIDC.NameClaim),
			UsernameClaim: strings.TrimSpace(cfg.OIDC.UsernameClaim),
			GroupsClaim:   strings.TrimSpace(cfg.OIDC.GroupsClaim),
		},
		AllowedEmailDomains:  append([]string(nil), cfg.OIDC.AllowedEmailDomains...),
		BootstrapAdminEmails: append([]string(nil), cfg.OIDC.BootstrapAdminEmails...),
		SessionPolicy: iam.OIDCSessionPolicy{
			SessionTTL:     cfg.OIDC.SessionTTL,
			SessionIdleTTL: cfg.OIDC.SessionIdleTTL,
		},
	})
	if err == nil {
		state.ResolvedOIDCConfig = &active
	}
	return state
}

func runtimeConfigAuthMode(state iam.RuntimeAccessControlState) config.AuthMode {
	if state.AuthMode == iam.AuthModeOIDC {
		return config.AuthModeOIDC
	}
	return config.AuthModeDisabled
}

func runtimeAccessControlIssuerURL(state iam.RuntimeAccessControlState, stored iam.AccessControlState) string {
	if state.ResolvedOIDCConfig != nil {
		return strings.TrimSpace(state.ResolvedOIDCConfig.IssuerURL)
	}
	switch {
	case stored.Active != nil:
		return strings.TrimSpace(stored.Active.IssuerURL)
	case stored.Draft != nil:
		return strings.TrimSpace(stored.Draft.IssuerURL)
	default:
		return ""
	}
}

func writeAuthRuntimeUnavailable(c echo.Context, code string, err error) error {
	return writeAPIError(c, http.StatusInternalServerError, code, err.Error())
}
