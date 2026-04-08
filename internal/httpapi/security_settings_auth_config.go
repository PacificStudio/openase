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
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
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

type securityAuthSessionPolicyResponse struct {
	SessionTTL     string `json:"session_ttl"`
	SessionIdleTTL string `json:"session_idle_ttl"`
}

type securityAuthValidationDiagnosticsResponse struct {
	Status                string   `json:"status"`
	Message               string   `json:"message"`
	CheckedAt             *string  `json:"checked_at,omitempty"`
	IssuerURL             string   `json:"issuer_url,omitempty"`
	AuthorizationEndpoint string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string   `json:"token_endpoint,omitempty"`
	RedirectURL           string   `json:"redirect_url,omitempty"`
	Warnings              []string `json:"warnings"`
}

type securityAuthSettingsResponse struct {
	ActiveMode         string                                    `json:"active_mode"`
	ConfiguredMode     string                                    `json:"configured_mode"`
	IssuerURL          string                                    `json:"issuer_url,omitempty"`
	LocalPrincipal     string                                    `json:"local_principal"`
	ModeSummary        string                                    `json:"mode_summary"`
	RecommendedMode    string                                    `json:"recommended_mode"`
	PublicExposureRisk string                                    `json:"public_exposure_risk"`
	Warnings           []string                                  `json:"warnings"`
	NextSteps          []string                                  `json:"next_steps"`
	ConfigPath         string                                    `json:"config_path,omitempty"`
	BootstrapState     securityAuthBootstrapStateResponse        `json:"bootstrap_state"`
	SessionPolicy      securityAuthSessionPolicyResponse         `json:"session_policy"`
	LastValidation     securityAuthValidationDiagnosticsResponse `json:"last_validation"`
	OIDCDraft          securityOIDCDraftResponse                 `json:"oidc_draft"`
	Docs               []securityDocumentationLinkResponse       `json:"docs"`
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

type securityOIDCValidationRecord struct {
	Status                string
	Message               string
	CheckedAt             time.Time
	IssuerURL             string
	AuthorizationEndpoint string
	TokenEndpoint         string
	RedirectURL           string
	Warnings              []string
}

type securityStoredAuthState struct {
	Auth           config.AuthConfig
	LastValidation securityOIDCValidationRecord
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

func (e securitySettingsConfigEditor) loadStoredState() (securityStoredAuthState, error) {
	cfg := e.fallback
	if strings.TrimSpace(cfg.OIDC.RedirectURL) == "" {
		cfg.OIDC.RedirectURL = setup.DefaultOIDCRedirectURL
	}
	if len(cfg.OIDC.Scopes) == 0 {
		cfg.OIDC.Scopes = []string{"openid", "profile", "email", "groups"}
	}
	state := securityStoredAuthState{
		Auth:           cfg,
		LastValidation: defaultSecurityOIDCValidationRecord(),
	}
	if strings.TrimSpace(e.path) == "" {
		return state, nil
	}

	payload, err := os.ReadFile(e.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return securityStoredAuthState{}, fmt.Errorf("read config file: %w", err)
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
			LastValidation struct {
				Status                string   `yaml:"status"`
				Message               string   `yaml:"message"`
				CheckedAt             string   `yaml:"checked_at"`
				IssuerURL             string   `yaml:"issuer_url"`
				AuthorizationEndpoint string   `yaml:"authorization_endpoint"`
				TokenEndpoint         string   `yaml:"token_endpoint"`
				RedirectURL           string   `yaml:"redirect_url"`
				Warnings              []string `yaml:"warnings"`
			} `yaml:"last_validation"`
		} `yaml:"auth"`
	}
	if err := yaml.Unmarshal(payload, &doc); err != nil {
		return securityStoredAuthState{}, fmt.Errorf("parse config file: %w", err)
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
	state.Auth = cfg
	state.LastValidation = parseSecurityOIDCValidationRecord(
		doc.Auth.LastValidation.Status,
		doc.Auth.LastValidation.Message,
		doc.Auth.LastValidation.CheckedAt,
		doc.Auth.LastValidation.IssuerURL,
		doc.Auth.LastValidation.AuthorizationEndpoint,
		doc.Auth.LastValidation.TokenEndpoint,
		doc.Auth.LastValidation.RedirectURL,
		doc.Auth.LastValidation.Warnings,
	)
	return state, nil
}

func (e securitySettingsConfigEditor) saveDraft(input securityOIDCDraftInput, mode config.AuthMode) (securityStoredAuthState, error) {
	root, err := e.loadConfigRoot()
	if err != nil {
		return securityStoredAuthState{}, err
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
	return e.writeRoot(root)
}

func (e securitySettingsConfigEditor) saveValidation(record securityOIDCValidationRecord) error {
	root, err := e.loadConfigRoot()
	if err != nil {
		return err
	}
	authMap := childMap(root, "auth")
	lastValidation := childMap(authMap, "last_validation")
	lastValidation["status"] = record.Status
	lastValidation["message"] = record.Message
	if !record.CheckedAt.IsZero() {
		lastValidation["checked_at"] = record.CheckedAt.UTC().Format(time.RFC3339)
	}
	lastValidation["issuer_url"] = record.IssuerURL
	lastValidation["authorization_endpoint"] = record.AuthorizationEndpoint
	lastValidation["token_endpoint"] = record.TokenEndpoint
	lastValidation["redirect_url"] = record.RedirectURL
	lastValidation["warnings"] = append([]string(nil), record.Warnings...)
	authMap["last_validation"] = lastValidation
	root["auth"] = authMap
	_, err = e.writeRoot(root)
	return err
}

func (e securitySettingsConfigEditor) loadConfigRoot() (map[string]any, error) {
	if strings.TrimSpace(e.path) == "" {
		return nil, errors.New("config file path is unavailable")
	}

	root := map[string]any{}
	if payload, err := os.ReadFile(e.path); err == nil {
		if len(payload) > 0 {
			if err := yaml.Unmarshal(payload, &root); err != nil {
				return nil, fmt.Errorf("parse config file: %w", err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	if root == nil {
		root = map[string]any{}
	}
	return root, nil
}

func (e securitySettingsConfigEditor) writeRoot(root map[string]any) (securityStoredAuthState, error) {
	content, err := yaml.Marshal(root)
	if err != nil {
		return securityStoredAuthState{}, fmt.Errorf("marshal config file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(e.path), 0o700); err != nil {
		return securityStoredAuthState{}, fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(e.path, content, 0o600); err != nil {
		return securityStoredAuthState{}, fmt.Errorf("write config file: %w", err)
	}
	return e.loadStoredState()
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

func buildSecurityAuthSettingsResponse(
	active config.AuthConfig,
	stored config.AuthConfig,
	lastValidation securityOIDCValidationRecord,
	configPath string,
	host string,
) securityAuthSettingsResponse {
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
		SessionPolicy:      buildSecuritySessionPolicyResponse(stored, active),
		LastValidation:     buildSecurityValidationDiagnosticsResponse(lastValidation),
		OIDCDraft:          buildSecurityOIDCDraftResponse(stored),
		Docs:               defaultSecurityDocumentationLinks(),
	}
	return response
}

func buildSecuritySessionPolicyResponse(
	stored config.AuthConfig,
	active config.AuthConfig,
) securityAuthSessionPolicyResponse {
	sessionTTL := stored.OIDC.SessionTTL
	if sessionTTL == 0 {
		sessionTTL = active.OIDC.SessionTTL
	}
	if sessionTTL == 0 {
		sessionTTL = 8 * time.Hour
	}
	sessionIdleTTL := stored.OIDC.SessionIdleTTL
	if sessionIdleTTL == 0 {
		sessionIdleTTL = active.OIDC.SessionIdleTTL
	}
	if sessionIdleTTL == 0 {
		sessionIdleTTL = 30 * time.Minute
	}
	return securityAuthSessionPolicyResponse{
		SessionTTL:     sessionTTL.String(),
		SessionIdleTTL: sessionIdleTTL.String(),
	}
}

func buildSecurityValidationDiagnosticsResponse(
	record securityOIDCValidationRecord,
) securityAuthValidationDiagnosticsResponse {
	response := securityAuthValidationDiagnosticsResponse{
		Status:                record.Status,
		Message:               record.Message,
		IssuerURL:             strings.TrimSpace(record.IssuerURL),
		AuthorizationEndpoint: strings.TrimSpace(record.AuthorizationEndpoint),
		TokenEndpoint:         strings.TrimSpace(record.TokenEndpoint),
		RedirectURL:           strings.TrimSpace(record.RedirectURL),
		Warnings:              append([]string(nil), record.Warnings...),
	}
	if response.Status == "" {
		response.Status = "not_tested"
	}
	if strings.TrimSpace(response.Message) == "" {
		response.Message = "No OIDC validation has been recorded yet."
	}
	if !record.CheckedAt.IsZero() {
		value := record.CheckedAt.UTC().Format(time.RFC3339)
		response.CheckedAt = &value
	}
	return response
}

func defaultSecurityOIDCValidationRecord() securityOIDCValidationRecord {
	return securityOIDCValidationRecord{
		Status:   "not_tested",
		Message:  "No OIDC validation has been recorded yet.",
		Warnings: []string{},
	}
}

func parseSecurityOIDCValidationRecord(
	status string,
	message string,
	checkedAt string,
	issuerURL string,
	authorizationEndpoint string,
	tokenEndpoint string,
	redirectURL string,
	warnings []string,
) securityOIDCValidationRecord {
	record := defaultSecurityOIDCValidationRecord()
	if trimmed := strings.TrimSpace(status); trimmed != "" {
		record.Status = trimmed
	}
	if trimmed := strings.TrimSpace(message); trimmed != "" {
		record.Message = trimmed
	}
	if trimmed := strings.TrimSpace(checkedAt); trimmed != "" {
		if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
			record.CheckedAt = parsed
		}
	}
	record.IssuerURL = strings.TrimSpace(issuerURL)
	record.AuthorizationEndpoint = strings.TrimSpace(authorizationEndpoint)
	record.TokenEndpoint = strings.TrimSpace(tokenEndpoint)
	record.RedirectURL = strings.TrimSpace(redirectURL)
	record.Warnings = normalizeStringList(warnings, false)
	return record
}

func securityOIDCValidationSuccessRecord(
	response securityOIDCTestResultResponse,
) securityOIDCValidationRecord {
	return securityOIDCValidationRecord{
		Status:                "ok",
		Message:               strings.TrimSpace(response.Message),
		CheckedAt:             time.Now().UTC(),
		IssuerURL:             strings.TrimSpace(response.IssuerURL),
		AuthorizationEndpoint: strings.TrimSpace(response.AuthorizationEndpoint),
		TokenEndpoint:         strings.TrimSpace(response.TokenEndpoint),
		RedirectURL:           strings.TrimSpace(response.RedirectURL),
		Warnings:              normalizeStringList(response.Warnings, false),
	}
}

func securityOIDCValidationFailureRecord(
	message string,
	redirectURL string,
	host string,
	mode config.AuthMode,
) securityOIDCValidationRecord {
	return securityOIDCValidationRecord{
		Status:      "failed",
		Message:     strings.TrimSpace(message),
		CheckedAt:   time.Now().UTC(),
		RedirectURL: strings.TrimSpace(redirectURL),
		Warnings:    securityPublicExposureWarnings(host, mode),
	}
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

func buildSecurityAuthSettingsResponseFromAccessControl(
	runtimeState iam.RuntimeAccessControlState,
	stored iam.AccessControlState,
	storageLocation string,
	host string,
) securityAuthSettingsResponse {
	activeMode := runtimeConfigAuthMode(runtimeState)
	publicExposureRisk, warnings := securityPublicExposure(host, activeMode)
	configuredMode := stored.ConfiguredAuthMode().String()
	issuerURL := runtimeAccessControlIssuerURL(runtimeState, stored)
	return securityAuthSettingsResponse{
		ActiveMode:         string(activeMode),
		ConfiguredMode:     configuredMode,
		IssuerURL:          issuerURL,
		LocalPrincipal:     "local_instance_admin:default",
		ModeSummary:        securityModeSummary(activeMode),
		RecommendedMode:    securityRecommendedMode(activeMode),
		PublicExposureRisk: publicExposureRisk,
		Warnings:           warnings,
		NextSteps:          securityNextSteps(activeMode, configuredMode),
		ConfigPath:         storageLocation,
		BootstrapState:     buildBootstrapStateFromAccessControl(stored),
		SessionPolicy:      buildSecuritySessionPolicyResponseFromAccessControl(stored, runtimeState),
		LastValidation:     buildSecurityValidationDiagnosticsResponseFromAccessControl(stored.Validation),
		OIDCDraft:          buildSecurityOIDCDraftResponseFromAccessControl(stored),
		Docs:               defaultSecurityDocumentationLinks(),
	}
}

func buildSecuritySessionPolicyResponseFromAccessControl(
	stored iam.AccessControlState,
	runtimeState iam.RuntimeAccessControlState,
) securityAuthSessionPolicyResponse {
	sessionTTL := time.Duration(0)
	sessionIdleTTL := time.Duration(0)
	if runtimeState.ResolvedOIDCConfig != nil {
		sessionTTL = runtimeState.ResolvedOIDCConfig.SessionPolicy.SessionTTL
		sessionIdleTTL = runtimeState.ResolvedOIDCConfig.SessionPolicy.SessionIdleTTL
	}
	switch {
	case stored.Active != nil:
		sessionTTL = stored.Active.SessionPolicy.SessionTTL
		sessionIdleTTL = stored.Active.SessionPolicy.SessionIdleTTL
	case stored.Draft != nil:
		sessionTTL = stored.Draft.SessionPolicy.SessionTTL
		sessionIdleTTL = stored.Draft.SessionPolicy.SessionIdleTTL
	}
	if sessionTTL == 0 {
		sessionTTL = 8 * time.Hour
	}
	if sessionIdleTTL == 0 {
		sessionIdleTTL = 30 * time.Minute
	}
	return securityAuthSessionPolicyResponse{
		SessionTTL:     sessionTTL.String(),
		SessionIdleTTL: sessionIdleTTL.String(),
	}
}

func buildSecurityValidationDiagnosticsResponseFromAccessControl(
	validation iam.OIDCValidationMetadata,
) securityAuthValidationDiagnosticsResponse {
	response := securityAuthValidationDiagnosticsResponse{
		Status:                validation.Status,
		Message:               validation.Message,
		IssuerURL:             strings.TrimSpace(validation.IssuerURL),
		AuthorizationEndpoint: strings.TrimSpace(validation.AuthorizationEndpoint),
		TokenEndpoint:         strings.TrimSpace(validation.TokenEndpoint),
		RedirectURL:           strings.TrimSpace(validation.RedirectURL),
		Warnings:              append([]string(nil), validation.Warnings...),
	}
	if response.Status == "" {
		response.Status = "not_tested"
	}
	if strings.TrimSpace(response.Message) == "" {
		response.Message = "No OIDC validation has been recorded yet."
	}
	if validation.CheckedAt != nil {
		value := validation.CheckedAt.UTC().Format(time.RFC3339)
		response.CheckedAt = &value
	}
	return response
}

func securityOIDCValidationSuccessMetadata(response securityOIDCTestResultResponse) iam.OIDCValidationMetadata {
	now := time.Now().UTC()
	return iam.OIDCValidationMetadata{
		Status:                "ok",
		Message:               strings.TrimSpace(response.Message),
		CheckedAt:             &now,
		IssuerURL:             strings.TrimSpace(response.IssuerURL),
		AuthorizationEndpoint: strings.TrimSpace(response.AuthorizationEndpoint),
		TokenEndpoint:         strings.TrimSpace(response.TokenEndpoint),
		RedirectURL:           strings.TrimSpace(response.RedirectURL),
		Warnings:              normalizeStringList(response.Warnings, false),
	}
}

func securityOIDCValidationFailureMetadata(
	message string,
	redirectURL string,
	host string,
	mode config.AuthMode,
) iam.OIDCValidationMetadata {
	now := time.Now().UTC()
	return iam.OIDCValidationMetadata{
		Status:      "failed",
		Message:     strings.TrimSpace(message),
		CheckedAt:   &now,
		RedirectURL: strings.TrimSpace(redirectURL),
		Warnings:    securityPublicExposureWarnings(host, mode),
	}
}

func buildBootstrapStateFromAccessControl(state iam.AccessControlState) securityAuthBootstrapStateResponse {
	var emails []string
	switch {
	case state.Active != nil:
		emails = append([]string(nil), state.Active.BootstrapAdminEmails...)
	case state.Draft != nil:
		emails = append([]string(nil), state.Draft.BootstrapAdminEmails...)
	default:
		emails = []string{}
	}
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

func buildSecurityOIDCDraftResponseFromAccessControl(state iam.AccessControlState) securityOIDCDraftResponse {
	draft := iam.DefaultDraftOIDCConfig()
	switch {
	case state.Active != nil:
		draft = iam.DraftOIDCConfig(*state.Active)
	case state.Draft != nil:
		draft = *state.Draft
	}
	return securityOIDCDraftResponse{
		IssuerURL:              strings.TrimSpace(draft.IssuerURL),
		ClientID:               strings.TrimSpace(draft.ClientID),
		ClientSecretConfigured: strings.TrimSpace(draft.ClientSecret) != "",
		RedirectURL:            strings.TrimSpace(draft.RedirectURL),
		Scopes:                 append([]string(nil), draft.Scopes...),
		AllowedEmailDomains:    append([]string(nil), draft.AllowedEmailDomains...),
		BootstrapAdminEmails:   append([]string(nil), draft.BootstrapAdminEmails...),
	}
}

func draftOIDCConfigFromRequest(raw rawSecurityOIDCDraftRequest, current iam.AccessControlState) iam.DraftOIDCConfig {
	defaultDraft := iam.DefaultDraftOIDCConfig()
	existing := defaultDraft
	switch {
	case current.Active != nil:
		existing = iam.DraftOIDCConfig(*current.Active)
	case current.Draft != nil:
		existing = *current.Draft
	}
	return iam.DraftOIDCConfig{
		IssuerURL:            strings.TrimSpace(raw.IssuerURL),
		ClientID:             strings.TrimSpace(raw.ClientID),
		ClientSecret:         preserveSecret(raw.ClientSecret, existing.ClientSecret),
		RedirectURL:          strings.TrimSpace(raw.RedirectURL),
		Scopes:               fallbackList(normalizeStringList(raw.Scopes, false), existing.Scopes),
		Claims:               existing.Claims,
		AllowedEmailDomains:  normalizeStringList(raw.AllowedEmailDomains, true),
		BootstrapAdminEmails: normalizeStringList(raw.BootstrapAdminEmails, true),
		SessionPolicy:        existing.SessionPolicy,
	}
}

func activeOIDCConfigFromDraft(draft iam.DraftOIDCConfig) (iam.ActiveOIDCConfig, error) {
	state, err := iam.ParseAccessControlState(iam.AccessControlStateInput{
		Status:               iam.AccessControlStatusActive.String(),
		IssuerURL:            draft.IssuerURL,
		ClientID:             draft.ClientID,
		ClientSecret:         draft.ClientSecret,
		RedirectURL:          draft.RedirectURL,
		Scopes:               draft.Scopes,
		EmailClaim:           draft.Claims.EmailClaim,
		NameClaim:            draft.Claims.NameClaim,
		UsernameClaim:        draft.Claims.UsernameClaim,
		GroupsClaim:          draft.Claims.GroupsClaim,
		AllowedEmailDomains:  draft.AllowedEmailDomains,
		BootstrapAdminEmails: draft.BootstrapAdminEmails,
		SessionTTL:           draft.SessionPolicy.SessionTTL.String(),
		SessionIdleTTL:       draft.SessionPolicy.SessionIdleTTL.String(),
	})
	if err != nil {
		return iam.ActiveOIDCConfig{}, err
	}
	if state.Active == nil {
		return iam.ActiveOIDCConfig{}, errors.New("active oidc config is required")
	}
	return *state.Active, nil
}

func completeOIDCAuthConfigFromAccessControl(active iam.ActiveOIDCConfig) config.AuthConfig {
	return config.AuthConfig{
		Mode: config.AuthModeOIDC,
		OIDC: config.OIDCConfig{
			IssuerURL:            active.IssuerURL,
			ClientID:             active.ClientID,
			ClientSecret:         active.ClientSecret,
			RedirectURL:          active.RedirectURL,
			Scopes:               append([]string(nil), active.Scopes...),
			EmailClaim:           active.Claims.EmailClaim,
			NameClaim:            active.Claims.NameClaim,
			UsernameClaim:        active.Claims.UsernameClaim,
			GroupsClaim:          active.Claims.GroupsClaim,
			AllowedEmailDomains:  append([]string(nil), active.AllowedEmailDomains...),
			BootstrapAdminEmails: append([]string(nil), active.BootstrapAdminEmails...),
			SessionTTL:           active.SessionPolicy.SessionTTL,
			SessionIdleTTL:       active.SessionPolicy.SessionIdleTTL,
		},
	}
}

func fallbackList(items []string, fallback []string) []string {
	if items == nil {
		return append([]string(nil), fallback...)
	}
	return items
}
