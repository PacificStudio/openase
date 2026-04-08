package httpapi

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
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
	RedirectMode           string   `json:"redirect_mode"`
	FixedRedirectURL       string   `json:"fixed_redirect_url"`
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
		RedirectMode:           draft.RedirectMode.String(),
		FixedRedirectURL:       strings.TrimSpace(draft.FixedRedirectURL),
		Scopes:                 append([]string(nil), draft.Scopes...),
		AllowedEmailDomains:    append([]string(nil), draft.AllowedEmailDomains...),
		BootstrapAdminEmails:   append([]string(nil), draft.BootstrapAdminEmails...),
	}
}

func draftOIDCConfigFromRequest(raw rawSecurityOIDCDraftRequest, current iam.AccessControlState) (iam.DraftOIDCConfig, error) {
	defaultDraft := iam.DefaultDraftOIDCConfig()
	existing := defaultDraft
	switch {
	case current.Active != nil:
		existing = iam.DraftOIDCConfig(*current.Active)
	case current.Draft != nil:
		existing = *current.Draft
	}
	fixedRedirectURL := strings.TrimSpace(raw.FixedRedirectURL)
	if fixedRedirectURL == "" {
		fixedRedirectURL = strings.TrimSpace(raw.RedirectURL)
	}
	redirectMode, err := iam.ParseOIDCRedirectMode(raw.RedirectMode, fixedRedirectURL)
	if err != nil {
		return iam.DraftOIDCConfig{}, err
	}
	return iam.DraftOIDCConfig{
		IssuerURL:            strings.TrimSpace(raw.IssuerURL),
		ClientID:             strings.TrimSpace(raw.ClientID),
		ClientSecret:         preserveSecret(raw.ClientSecret, existing.ClientSecret),
		RedirectMode:         redirectMode,
		FixedRedirectURL:     fixedRedirectURL,
		Scopes:               fallbackList(normalizeStringList(raw.Scopes, false), existing.Scopes),
		Claims:               existing.Claims,
		AllowedEmailDomains:  normalizeStringList(raw.AllowedEmailDomains, true),
		BootstrapAdminEmails: normalizeStringList(raw.BootstrapAdminEmails, true),
		SessionPolicy:        existing.SessionPolicy,
	}, nil
}

func activeOIDCConfigFromDraft(draft iam.DraftOIDCConfig) (iam.ActiveOIDCConfig, error) {
	state, err := iam.ParseAccessControlState(iam.AccessControlStateInput{
		Status:               iam.AccessControlStatusActive.String(),
		IssuerURL:            draft.IssuerURL,
		ClientID:             draft.ClientID,
		ClientSecret:         draft.ClientSecret,
		RedirectMode:         draft.RedirectMode.String(),
		FixedRedirectURL:     draft.FixedRedirectURL,
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
			RedirectURL:          strings.TrimSpace(active.FixedRedirectURL),
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
