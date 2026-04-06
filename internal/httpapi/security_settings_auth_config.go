package httpapi

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/setup"
	"go.yaml.in/yaml/v3"
)

const securitySettingsDefaultDocsBaseURL = "https://github.com/pacificstudio/openase/blob/main"

type securityAuthBootstrapStateResponse struct {
	Status      string   `json:"status"`
	AdminEmails []string `json:"admin_emails"`
	Summary     string   `json:"summary"`
}

type securityOIDCDraftResponse struct {
	IssuerURL              string   `json:"issuer_url"`
	ClientID               string   `json:"client_id"`
	ClientSecretConfigured bool     `json:"client_secret_configured"`
	RedirectURL            string   `json:"redirect_url"`
	Scopes                 []string `json:"scopes"`
	AllowedEmailDomains    []string `json:"allowed_email_domains"`
	BootstrapAdminEmails   []string `json:"bootstrap_admin_emails"`
}

type securityDocumentationLinkResponse struct {
	Title   string `json:"title"`
	Href    string `json:"href"`
	Summary string `json:"summary"`
}

type securityAuthSettingsResponse struct {
	ActiveMode         string                              `json:"active_mode"`
	ConfiguredMode     string                              `json:"configured_mode"`
	IssuerURL          string                              `json:"issuer_url,omitempty"`
	LocalPrincipal     string                              `json:"local_principal"`
	ModeSummary        string                              `json:"mode_summary"`
	RecommendedMode    string                              `json:"recommended_mode"`
	PublicExposureRisk string                              `json:"public_exposure_risk"`
	Warnings           []string                            `json:"warnings"`
	NextSteps          []string                            `json:"next_steps"`
	ConfigPath         string                              `json:"config_path,omitempty"`
	BootstrapState     securityAuthBootstrapStateResponse  `json:"bootstrap_state"`
	OIDCDraft          securityOIDCDraftResponse           `json:"oidc_draft"`
	Docs               []securityDocumentationLinkResponse `json:"docs"`
}

type securityOIDCTestResultResponse struct {
	Status                string   `json:"status"`
	Message               string   `json:"message"`
	IssuerURL             string   `json:"issuer_url"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	RedirectURL           string   `json:"redirect_url"`
	Warnings              []string `json:"warnings"`
}

type securityOIDCEnableResponse struct {
	Activation securityOIDCActivationResponse `json:"activation"`
	Security   securitySettingsResponse       `json:"security"`
}

type securityOIDCActivationResponse struct {
	Status          string   `json:"status"`
	Message         string   `json:"message"`
	RestartRequired bool     `json:"restart_required"`
	NextSteps       []string `json:"next_steps"`
}

type securityOIDCDraftInput struct {
	IssuerURL            string
	ClientID             string
	ClientSecret         string
	RedirectURL          string
	Scopes               []string
	AllowedEmailDomains  []string
	BootstrapAdminEmails []string
}

type securitySettingsConfigEditor struct {
	path     string
	fallback config.AuthConfig
}

func newSecuritySettingsConfigEditor(path string, homeDir string, fallback config.AuthConfig) securitySettingsConfigEditor {
	resolvedPath := strings.TrimSpace(path)
	if resolvedPath == "" {
		base := strings.TrimSpace(homeDir)
		if base == "" {
			if guessed, err := os.UserHomeDir(); err == nil {
				base = guessed
			}
		}
		if base != "" {
			resolvedPath = filepath.Join(base, ".openase", "config.yaml")
		}
	}
	return securitySettingsConfigEditor{path: resolvedPath, fallback: fallback}
}

func (e securitySettingsConfigEditor) resolvedPath() string {
	return e.path
}

func (e securitySettingsConfigEditor) loadStoredAuth() (config.AuthConfig, error) {
	cfg := e.fallback
	if strings.TrimSpace(cfg.OIDC.RedirectURL) == "" {
		cfg.OIDC.RedirectURL = setup.DefaultOIDCRedirectURL
	}
	if len(cfg.OIDC.Scopes) == 0 {
		cfg.OIDC.Scopes = []string{"openid", "profile", "email", "groups"}
	}
	if strings.TrimSpace(e.path) == "" {
		return cfg, nil
	}

	payload, err := os.ReadFile(e.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return config.AuthConfig{}, fmt.Errorf("read config file: %w", err)
	}

	var doc struct {
		Auth struct {
			Mode string `yaml:"mode"`
			OIDC struct {
				IssuerURL            string   `yaml:"issuer_url"`
				ClientID             string   `yaml:"client_id"`
				ClientSecret         string   `yaml:"client_secret"`
				RedirectURL          string   `yaml:"redirect_url"`
				Scopes               []string `yaml:"scopes"`
				AllowedEmailDomains  []string `yaml:"allowed_email_domains"`
				BootstrapAdminEmails []string `yaml:"bootstrap_admin_emails"`
			} `yaml:"oidc"`
		} `yaml:"auth"`
	}
	if err := yaml.Unmarshal(payload, &doc); err != nil {
		return config.AuthConfig{}, fmt.Errorf("parse config file: %w", err)
	}
	if mode := normalizeAuthMode(doc.Auth.Mode, cfg.Mode); mode != "" {
		cfg.Mode = mode
	}
	cfg.OIDC = mergeOIDCDraftWithConfig(cfg.OIDC, securityOIDCDraftInput{
		IssuerURL:            doc.Auth.OIDC.IssuerURL,
		ClientID:             doc.Auth.OIDC.ClientID,
		ClientSecret:         doc.Auth.OIDC.ClientSecret,
		RedirectURL:          doc.Auth.OIDC.RedirectURL,
		Scopes:               doc.Auth.OIDC.Scopes,
		AllowedEmailDomains:  doc.Auth.OIDC.AllowedEmailDomains,
		BootstrapAdminEmails: doc.Auth.OIDC.BootstrapAdminEmails,
	})
	return cfg, nil
}

func (e securitySettingsConfigEditor) saveDraft(input securityOIDCDraftInput, mode config.AuthMode) (config.AuthConfig, error) {
	if strings.TrimSpace(e.path) == "" {
		return config.AuthConfig{}, errors.New("config file path is unavailable")
	}

	root := map[string]any{}
	if payload, err := os.ReadFile(e.path); err == nil {
		if len(payload) > 0 {
			if err := yaml.Unmarshal(payload, &root); err != nil {
				return config.AuthConfig{}, fmt.Errorf("parse config file: %w", err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return config.AuthConfig{}, fmt.Errorf("read config file: %w", err)
	}
	if root == nil {
		root = map[string]any{}
	}

	authMap := childMap(root, "auth")
	authMap["mode"] = string(mode)
	oidcMap := childMap(authMap, "oidc")
	oidcMap["issuer_url"] = input.IssuerURL
	oidcMap["client_id"] = input.ClientID
	if strings.TrimSpace(input.ClientSecret) != "" {
		oidcMap["client_secret"] = input.ClientSecret
	}
	oidcMap["redirect_url"] = input.RedirectURL
	oidcMap["scopes"] = append([]string(nil), input.Scopes...)
	oidcMap["allowed_email_domains"] = append([]string(nil), input.AllowedEmailDomains...)
	oidcMap["bootstrap_admin_emails"] = append([]string(nil), input.BootstrapAdminEmails...)
	authMap["oidc"] = oidcMap
	root["auth"] = authMap

	content, err := yaml.Marshal(root)
	if err != nil {
		return config.AuthConfig{}, fmt.Errorf("marshal config file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(e.path), 0o700); err != nil {
		return config.AuthConfig{}, fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(e.path, content, 0o600); err != nil {
		return config.AuthConfig{}, fmt.Errorf("write config file: %w", err)
	}
	return e.loadStoredAuth()
}

func childMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key].(map[string]any); ok && existing != nil {
		return existing
	}
	next := map[string]any{}
	parent[key] = next
	return next
}

func normalizeAuthMode(raw string, fallback config.AuthMode) config.AuthMode {
	switch config.AuthMode(strings.ToLower(strings.TrimSpace(raw))) {
	case config.AuthModeDisabled:
		return config.AuthModeDisabled
	case config.AuthModeOIDC:
		return config.AuthModeOIDC
	default:
		return fallback
	}
}

func mergeOIDCDraftWithConfig(base config.OIDCConfig, input securityOIDCDraftInput) config.OIDCConfig {
	next := base
	if strings.TrimSpace(input.IssuerURL) != "" || next.IssuerURL == "" {
		next.IssuerURL = strings.TrimSpace(input.IssuerURL)
	}
	if strings.TrimSpace(input.ClientID) != "" || next.ClientID == "" {
		next.ClientID = strings.TrimSpace(input.ClientID)
	}
	if strings.TrimSpace(input.ClientSecret) != "" || next.ClientSecret == "" {
		next.ClientSecret = strings.TrimSpace(input.ClientSecret)
	}
	if strings.TrimSpace(input.RedirectURL) != "" || next.RedirectURL == "" {
		next.RedirectURL = strings.TrimSpace(input.RedirectURL)
	}
	if input.Scopes != nil {
		next.Scopes = append([]string(nil), input.Scopes...)
	}
	if input.AllowedEmailDomains != nil {
		next.AllowedEmailDomains = append([]string(nil), input.AllowedEmailDomains...)
	}
	if input.BootstrapAdminEmails != nil {
		next.BootstrapAdminEmails = append([]string(nil), input.BootstrapAdminEmails...)
	}
	if strings.TrimSpace(next.RedirectURL) == "" {
		next.RedirectURL = setup.DefaultOIDCRedirectURL
	}
	if len(next.Scopes) == 0 {
		next.Scopes = []string{"openid", "profile", "email", "groups"}
	}
	return next
}

func draftInputFromConfig(cfg config.AuthConfig) securityOIDCDraftInput {
	return securityOIDCDraftInput{
		IssuerURL:            strings.TrimSpace(cfg.OIDC.IssuerURL),
		ClientID:             strings.TrimSpace(cfg.OIDC.ClientID),
		ClientSecret:         strings.TrimSpace(cfg.OIDC.ClientSecret),
		RedirectURL:          strings.TrimSpace(cfg.OIDC.RedirectURL),
		Scopes:               append([]string(nil), cfg.OIDC.Scopes...),
		AllowedEmailDomains:  append([]string(nil), cfg.OIDC.AllowedEmailDomains...),
		BootstrapAdminEmails: append([]string(nil), cfg.OIDC.BootstrapAdminEmails...),
	}
}

func parseSecurityOIDCDraftRequest(raw rawSecurityOIDCDraftRequest, existing securityOIDCDraftInput) securityOIDCDraftInput {
	return securityOIDCDraftInput{
		IssuerURL:            strings.TrimSpace(raw.IssuerURL),
		ClientID:             strings.TrimSpace(raw.ClientID),
		ClientSecret:         preserveSecret(raw.ClientSecret, existing.ClientSecret),
		RedirectURL:          strings.TrimSpace(raw.RedirectURL),
		Scopes:               normalizeStringList(raw.Scopes, false),
		AllowedEmailDomains:  normalizeStringList(raw.AllowedEmailDomains, true),
		BootstrapAdminEmails: normalizeStringList(raw.BootstrapAdminEmails, true),
	}
}

func preserveSecret(raw string, existing string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(existing)
}

func normalizeStringList(items []string, lower bool) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if lower {
			trimmed = strings.ToLower(trimmed)
		}
		result = append(result, trimmed)
	}
	return result
}

func completeOIDCAuthConfig(input securityOIDCDraftInput) (config.AuthConfig, error) {
	parsed, err := setup.ParseAuthInput(setup.RawAuthInput{
		Mode: string(setup.AuthModeOIDC),
		OIDC: &setup.RawOIDCInput{
			IssuerURL:            input.IssuerURL,
			ClientID:             input.ClientID,
			ClientSecret:         input.ClientSecret,
			RedirectURL:          input.RedirectURL,
			Scopes:               strings.Join(input.Scopes, ","),
			AllowedEmailDomains:  strings.Join(input.AllowedEmailDomains, ","),
			BootstrapAdminEmails: strings.Join(input.BootstrapAdminEmails, ","),
			SessionTTL:           setup.DefaultOIDCSessionTTL,
			SessionIdleTTL:       setup.DefaultOIDCIdleTTL,
		},
	})
	if err != nil {
		return config.AuthConfig{}, err
	}
	if parsed.OIDC == nil {
		return config.AuthConfig{}, errors.New("oidc settings are required")
	}
	return config.AuthConfig{
		Mode: config.AuthModeOIDC,
		OIDC: config.OIDCConfig{
			IssuerURL:            parsed.OIDC.IssuerURL,
			ClientID:             parsed.OIDC.ClientID,
			ClientSecret:         parsed.OIDC.ClientSecret,
			RedirectURL:          parsed.OIDC.RedirectURL,
			Scopes:               append([]string(nil), parsed.OIDC.Scopes...),
			EmailClaim:           "email",
			NameClaim:            "name",
			UsernameClaim:        "preferred_username",
			GroupsClaim:          "groups",
			AllowedEmailDomains:  append([]string(nil), parsed.OIDC.AllowedEmailDomains...),
			BootstrapAdminEmails: append([]string(nil), parsed.OIDC.BootstrapAdminEmails...),
			SessionTTL:           8 * time.Hour,
			SessionIdleTTL:       30 * time.Minute,
		},
	}, nil
}

func buildSecurityAuthSettingsResponse(active config.AuthConfig, stored config.AuthConfig, configPath string, host string) securityAuthSettingsResponse {
	publicExposureRisk, warnings := securityPublicExposure(host, active.Mode)
	bootstrap := buildBootstrapState(stored)
	configuredMode := string(stored.Mode)
	if configuredMode == "" {
		configuredMode = string(active.Mode)
	}
	issuerURL := strings.TrimSpace(active.OIDC.IssuerURL)
	if issuerURL == "" {
		issuerURL = strings.TrimSpace(stored.OIDC.IssuerURL)
	}
	response := securityAuthSettingsResponse{
		ActiveMode:         string(active.Mode),
		ConfiguredMode:     configuredMode,
		IssuerURL:          issuerURL,
		LocalPrincipal:     "local_instance_admin:default",
		ModeSummary:        securityModeSummary(active.Mode),
		RecommendedMode:    securityRecommendedMode(active.Mode),
		PublicExposureRisk: publicExposureRisk,
		Warnings:           warnings,
		NextSteps:          securityNextSteps(active.Mode, configuredMode),
		ConfigPath:         configPath,
		BootstrapState:     bootstrap,
		OIDCDraft:          buildSecurityOIDCDraftResponse(stored),
		Docs:               defaultSecurityDocumentationLinks(),
	}
	return response
}

func buildBootstrapState(cfg config.AuthConfig) securityAuthBootstrapStateResponse {
	emails := append([]string(nil), cfg.OIDC.BootstrapAdminEmails...)
	if len(emails) == 0 {
		return securityAuthBootstrapStateResponse{
			Status:      "not_configured",
			AdminEmails: []string{},
			Summary:     "No bootstrap admin emails configured. The first OIDC admin must be granted through another path before rollout.",
		}
	}
	return securityAuthBootstrapStateResponse{
		Status:      "configured",
		AdminEmails: emails,
		Summary:     fmt.Sprintf("%d bootstrap admin email(s) will receive instance_admin on first successful OIDC login.", len(emails)),
	}
}

func buildSecurityOIDCDraftResponse(cfg config.AuthConfig) securityOIDCDraftResponse {
	return securityOIDCDraftResponse{
		IssuerURL:              strings.TrimSpace(cfg.OIDC.IssuerURL),
		ClientID:               strings.TrimSpace(cfg.OIDC.ClientID),
		ClientSecretConfigured: strings.TrimSpace(cfg.OIDC.ClientSecret) != "",
		RedirectURL:            strings.TrimSpace(cfg.OIDC.RedirectURL),
		Scopes:                 append([]string(nil), cfg.OIDC.Scopes...),
		AllowedEmailDomains:    append([]string(nil), cfg.OIDC.AllowedEmailDomains...),
		BootstrapAdminEmails:   append([]string(nil), cfg.OIDC.BootstrapAdminEmails...),
	}
}

func securityModeSummary(mode config.AuthMode) string {
	if mode == config.AuthModeOIDC {
		return "OIDC is active. Browser sessions, RBAC, cached users, memberships, invitations, and auth audit diagnostics are enforced from this control plane."
	}
	return "Disabled mode keeps OpenASE in local single-user operation. The current user keeps local highest privilege without browser login or OIDC dependency."
}

func securityRecommendedMode(mode config.AuthMode) string {
	if mode == config.AuthModeOIDC {
		return "Use OIDC for multi-user or networked deployments, then keep bootstrap admin emails narrow after first login."
	}
	return "Keep disabled mode for personal or local-only use. Move to OIDC + instance_admin when you need real multi-user browser access control."
}

func securityPublicExposure(host string, mode config.AuthMode) (string, []string) {
	trimmed := strings.TrimSpace(host)
	if isLoopbackHost(trimmed) {
		if mode == config.AuthModeDisabled {
			return "local_only", []string{"Disabled mode is appropriate for local-only or single-user use on a loopback-bound instance."}
		}
		return "managed", []string{"OIDC is active on a loopback-bound instance. Keep redirect URLs aligned with the current local server address."}
	}
	if mode == config.AuthModeDisabled {
		return "high", []string{"High risk: auth.mode=disabled on a non-loopback host exposes the control plane without browser login. Enable OIDC before wider rollout."}
	}
	return "managed", []string{"OIDC is active on a non-loopback host. Confirm HTTPS, redirect URLs, allowed domains, and bootstrap admin coverage before rollout."}
}

func securityPublicExposureWarnings(host string, mode config.AuthMode) []string {
	_, warnings := securityPublicExposure(host, mode)
	return warnings
}

func isLoopbackHost(host string) bool {
	if host == "" {
		return true
	}
	if strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func securityNextSteps(active config.AuthMode, configuredMode string) []string {
	if active == config.AuthModeOIDC {
		return []string{
			"Verify instance, organization, and project role bindings for each administrator and operator group.",
			"Review session inventory and audit history after the first production login.",
			"Trim bootstrap admin emails once steady-state RBAC is in place.",
		}
	}
	steps := []string{
		"You can keep disabled mode for local single-user use with no extra IAM overhead.",
		"Save draft OIDC settings, test discovery, then enable OIDC only when you are ready for multi-user browser login.",
	}
	if configuredMode == string(config.AuthModeOIDC) {
		steps = append(steps, "Restart the service to activate the saved OIDC mode and complete the first bootstrap admin sign-in.")
	}
	return steps
}

func defaultSecurityDocumentationLinks() []securityDocumentationLinkResponse {
	return []securityDocumentationLinkResponse{
		{
			Title:   "Mode selection guide",
			Href:    securitySettingsDefaultDocsBaseURL + "/docs/en/human-auth-oidc-rbac.md",
			Summary: "Choose between disabled mode and OIDC, including local-user and instance_admin guidance.",
		},
		{
			Title:   "Dual-mode contract",
			Href:    securitySettingsDefaultDocsBaseURL + "/docs/en/iam-dual-mode-contract.md",
			Summary: "Read the long-term disabled versus OIDC contract and the explicit enable / rollback flow.",
		},
		{
			Title:   "IAM rollout checklist",
			Href:    securitySettingsDefaultDocsBaseURL + "/docs/en/iam-admin-console-rollout.md",
			Summary: "Roll out the full IAM console in stages with migration checks, rollback steps, and validation coverage.",
		},
	}
}
