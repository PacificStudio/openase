package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type AuthMode string

const (
	AuthModeDisabled AuthMode = "disabled"
	AuthModeOIDC     AuthMode = "oidc"
)

type AuthConfig struct {
	Mode AuthMode
	OIDC OIDCConfig
	CSRF CSRFConfig
}

type OIDCConfig struct {
	IssuerURL            string
	ClientID             string
	ClientSecret         string
	RedirectURL          string
	Scopes               []string
	EmailClaim           string
	NameClaim            string
	UsernameClaim        string
	GroupsClaim          string
	AllowedEmailDomains  []string
	BootstrapAdminEmails []string
	SessionTTL           time.Duration
	SessionIdleTTL       time.Duration
}

type CSRFConfig struct {
	TrustedOrigins []string
}

func configureAuthDefaults(v *viper.Viper) {
	v.SetDefault("auth.mode", string(AuthModeDisabled))
	v.SetDefault("auth.csrf.trusted_origins", []string{})
	v.SetDefault("auth.oidc.issuer_url", "")
	v.SetDefault("auth.oidc.client_id", "")
	v.SetDefault("auth.oidc.client_secret", "")
	v.SetDefault("auth.oidc.redirect_url", "")
	v.SetDefault("auth.oidc.scopes", []string{"openid", "profile", "email", "groups"})
	v.SetDefault("auth.oidc.email_claim", "email")
	v.SetDefault("auth.oidc.name_claim", "name")
	v.SetDefault("auth.oidc.username_claim", "preferred_username")
	v.SetDefault("auth.oidc.groups_claim", "groups")
	v.SetDefault("auth.oidc.allowed_email_domains", []string{})
	v.SetDefault("auth.oidc.bootstrap_admin_emails", []string{})
	v.SetDefault("auth.oidc.session_ttl", time.Duration(0))
	v.SetDefault("auth.oidc.session_idle_ttl", time.Duration(0))
}

func parseAuthConfig(v *viper.Viper) (AuthConfig, error) {
	mode, err := parseAuthMode(v.Get("auth.mode"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.mode: %w", err)
	}
	trustedOrigins, err := parseStringSlice(v.Get("auth.csrf.trusted_origins"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.csrf.trusted_origins: %w", err)
	}

	scopes, err := parseStringSlice(v.Get("auth.oidc.scopes"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.scopes: %w", err)
	}
	allowedEmailDomains, err := parseStringSlice(v.Get("auth.oidc.allowed_email_domains"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.allowed_email_domains: %w", err)
	}
	bootstrapAdminEmails, err := parseStringSlice(v.Get("auth.oidc.bootstrap_admin_emails"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.bootstrap_admin_emails: %w", err)
	}

	sessionTTL, err := parseNonNegativeDuration(v.Get("auth.oidc.session_ttl"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.session_ttl: %w", err)
	}
	sessionIdleTTL, err := parseNonNegativeDuration(v.Get("auth.oidc.session_idle_ttl"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.session_idle_ttl: %w", err)
	}

	issuerURL, err := parseOptionalString(v.Get("auth.oidc.issuer_url"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.issuer_url: %w", err)
	}
	clientID, err := parseOptionalString(v.Get("auth.oidc.client_id"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.client_id: %w", err)
	}
	clientSecret, err := parseOptionalString(v.Get("auth.oidc.client_secret"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.client_secret: %w", err)
	}
	redirectURL, err := parseOptionalString(v.Get("auth.oidc.redirect_url"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.redirect_url: %w", err)
	}
	emailClaim, err := parseNonEmptyString(v.Get("auth.oidc.email_claim"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.email_claim: %w", err)
	}
	nameClaim, err := parseNonEmptyString(v.Get("auth.oidc.name_claim"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.name_claim: %w", err)
	}
	usernameClaim, err := parseNonEmptyString(v.Get("auth.oidc.username_claim"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.username_claim: %w", err)
	}
	groupsClaim, err := parseNonEmptyString(v.Get("auth.oidc.groups_claim"))
	if err != nil {
		return AuthConfig{}, fmt.Errorf("parse auth.oidc.groups_claim: %w", err)
	}

	return AuthConfig{
		Mode: mode,
		CSRF: CSRFConfig{
			TrustedOrigins: normalizeOriginSlice(trustedOrigins),
		},
		OIDC: OIDCConfig{
			IssuerURL:            issuerURL,
			ClientID:             clientID,
			ClientSecret:         clientSecret,
			RedirectURL:          redirectURL,
			Scopes:               normalizeStringSlice(scopes),
			EmailClaim:           strings.TrimSpace(emailClaim),
			NameClaim:            strings.TrimSpace(nameClaim),
			UsernameClaim:        strings.TrimSpace(usernameClaim),
			GroupsClaim:          strings.TrimSpace(groupsClaim),
			AllowedEmailDomains:  normalizeLowerStringSlice(allowedEmailDomains),
			BootstrapAdminEmails: normalizeLowerStringSlice(bootstrapAdminEmails),
			SessionTTL:           sessionTTL,
			SessionIdleTTL:       sessionIdleTTL,
		},
	}, nil
}

func validateAuthConfig(cfg AuthConfig) error {
	for _, origin := range cfg.CSRF.TrustedOrigins {
		if _, err := NormalizeTrustedOriginForCSRF(origin); err != nil {
			return fmt.Errorf("auth.csrf.trusted_origins: %w", err)
		}
	}

	switch cfg.Mode {
	case "", AuthModeDisabled:
		return nil
	case AuthModeOIDC:
		if strings.TrimSpace(cfg.OIDC.IssuerURL) == "" {
			return errors.New("auth.oidc.issuer_url is required when auth.mode=oidc")
		}
		if strings.TrimSpace(cfg.OIDC.ClientID) == "" {
			return errors.New("auth.oidc.client_id is required when auth.mode=oidc")
		}
		if strings.TrimSpace(cfg.OIDC.ClientSecret) == "" {
			return errors.New("auth.oidc.client_secret is required when auth.mode=oidc")
		}
		if strings.TrimSpace(cfg.OIDC.RedirectURL) == "" {
			return errors.New("auth.oidc.redirect_url is required when auth.mode=oidc")
		}
		if len(cfg.OIDC.Scopes) == 0 {
			return errors.New("auth.oidc.scopes must not be empty when auth.mode=oidc")
		}
		if cfg.OIDC.SessionTTL > 0 && cfg.OIDC.SessionIdleTTL > cfg.OIDC.SessionTTL {
			return errors.New("auth.oidc.session_idle_ttl must not exceed auth.oidc.session_ttl")
		}
		return nil
	default:
		return fmt.Errorf("unsupported auth mode %q", cfg.Mode)
	}
}

func parseAuthMode(raw any) (AuthMode, error) {
	switch value := raw.(type) {
	case AuthMode:
		if value == AuthModeDisabled || value == AuthModeOIDC {
			return value, nil
		}
	case string:
		mode := AuthMode(strings.ToLower(strings.TrimSpace(value)))
		if mode == "" {
			mode = AuthModeDisabled
		}
		if mode == AuthModeDisabled || mode == AuthModeOIDC {
			return mode, nil
		}
		return "", fmt.Errorf("unsupported auth mode %q", value)
	default:
		return "", fmt.Errorf("unsupported auth mode type %T", raw)
	}

	return "", fmt.Errorf("unsupported auth mode %q", raw)
}

func parseStringSlice(raw any) ([]string, error) {
	switch value := raw.(type) {
	case []string:
		return append([]string(nil), value...), nil
	case []any:
		items := make([]string, 0, len(value))
		for _, item := range value {
			typed, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("unsupported list item type %T", item)
			}
			items = append(items, typed)
		}
		return items, nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return []string{}, nil
		}
		parts := strings.Split(trimmed, ",")
		items := make([]string, 0, len(parts))
		items = append(items, parts...)
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported string slice type %T", raw)
	}
}

func normalizeStringSlice(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func normalizeLowerStringSlice(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.ToLower(strings.TrimSpace(item))
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func normalizeOriginSlice(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		origin, err := NormalizeTrustedOriginForCSRF(item)
		if err != nil {
			result = append(result, strings.TrimSpace(item))
			continue
		}
		result = append(result, origin)
	}
	return result
}

// NormalizeTrustedOriginForCSRF canonicalizes a configured/request origin for CSRF checks.
func NormalizeTrustedOriginForCSRF(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("origin must not be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid origin %q: %w", trimmed, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid origin %q: scheme and host are required", trimmed)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("invalid origin %q: userinfo is not allowed", trimmed)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("invalid origin %q: path is not allowed", trimmed)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid origin %q: query and fragment are not allowed", trimmed)
	}

	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("invalid origin %q: host is required", trimmed)
	}
	if port := parsed.Port(); port != "" {
		host = net.JoinHostPort(host, port)
	}

	return scheme + "://" + host, nil
}
