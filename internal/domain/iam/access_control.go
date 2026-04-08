package iam

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type AccessControlStatus string

const (
	AccessControlStatusAbsent AccessControlStatus = "absent"
	AccessControlStatusDraft  AccessControlStatus = "draft"
	AccessControlStatusActive AccessControlStatus = "active"
)

func ParseAccessControlStatus(raw string) (AccessControlStatus, error) {
	switch AccessControlStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case "", AccessControlStatusAbsent:
		return AccessControlStatusAbsent, nil
	case AccessControlStatusDraft:
		return AccessControlStatusDraft, nil
	case AccessControlStatusActive:
		return AccessControlStatusActive, nil
	default:
		return "", fmt.Errorf("unsupported access control status %q", raw)
	}
}

func (s AccessControlStatus) String() string { return string(s) }

type OIDCClaims struct {
	EmailClaim    string
	NameClaim     string
	UsernameClaim string
	GroupsClaim   string
}

type OIDCSessionPolicy struct {
	SessionTTL     time.Duration
	SessionIdleTTL time.Duration
}

type EncryptedSecret struct {
	Algorithm  string    `json:"algorithm"`
	Nonce      string    `json:"nonce"`
	Ciphertext string    `json:"ciphertext"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type OIDCValidationMetadata struct {
	Status                string
	Message               string
	CheckedAt             *time.Time
	IssuerURL             string
	AuthorizationEndpoint string
	TokenEndpoint         string
	RedirectURL           string
	Warnings              []string
}

type OIDCActivationMetadata struct {
	ActivatedAt *time.Time
	ActivatedBy string
	Source      string
	Message     string
}

type DraftOIDCConfig struct {
	IssuerURL            string
	ClientID             string
	ClientSecret         string
	RedirectURL          string
	Scopes               []string
	Claims               OIDCClaims
	AllowedEmailDomains  []string
	BootstrapAdminEmails []string
	SessionPolicy        OIDCSessionPolicy
}

type ActiveOIDCConfig struct {
	IssuerURL            string
	ClientID             string
	ClientSecret         string
	RedirectURL          string
	Scopes               []string
	Claims               OIDCClaims
	AllowedEmailDomains  []string
	BootstrapAdminEmails []string
	SessionPolicy        OIDCSessionPolicy
}

type AccessControlStateInput struct {
	Status               string
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
	SessionTTL           string
	SessionIdleTTL       string
	Validation           OIDCValidationMetadataInput
	Activation           OIDCActivationMetadataInput
}

type OIDCValidationMetadataInput struct {
	Status                string
	Message               string
	CheckedAt             *time.Time
	IssuerURL             string
	AuthorizationEndpoint string
	TokenEndpoint         string
	RedirectURL           string
	Warnings              []string
}

type OIDCActivationMetadataInput struct {
	ActivatedAt *time.Time
	ActivatedBy string
	Source      string
	Message     string
}

type AccessControlState struct {
	Status     AccessControlStatus
	Draft      *DraftOIDCConfig
	Active     *ActiveOIDCConfig
	Validation OIDCValidationMetadata
	Activation OIDCActivationMetadata
}

type RuntimePrincipalKind string

const (
	RuntimePrincipalKindLocal RuntimePrincipalKind = "local_instance_admin"
	RuntimePrincipalKindHuman RuntimePrincipalKind = "human_user"
)

type RuntimeAccessControlState struct {
	Stored                   AccessControlState
	AuthMode                 AuthMode
	LoginRequired            bool
	PrincipalKind            RuntimePrincipalKind
	SessionGovernanceEnabled bool
	ResolvedOIDCConfig       *ActiveOIDCConfig
}

func ParseAccessControlState(input AccessControlStateInput) (AccessControlState, error) {
	status, err := ParseAccessControlStatus(input.Status)
	if err != nil {
		return AccessControlState{}, err
	}

	validation := normalizeOIDCValidationMetadata(input.Validation)
	activation := normalizeOIDCActivationMetadata(input.Activation)
	state := AccessControlState{
		Status:     status,
		Validation: validation,
		Activation: activation,
	}

	if status == AccessControlStatusAbsent {
		return state, nil
	}

	draft, err := parseDraftOIDCConfig(input)
	if err != nil {
		return AccessControlState{}, err
	}
	state.Draft = &draft
	if status == AccessControlStatusDraft {
		return state, nil
	}

	active, err := parseActiveOIDCConfig(input)
	if err != nil {
		return AccessControlState{}, err
	}
	state.Active = &active
	draft = DraftOIDCConfig(active)
	state.Draft = &draft
	return state, nil
}

func (s AccessControlState) ConfiguredAuthMode() AuthMode {
	if s.Status == AccessControlStatusActive {
		return AuthModeOIDC
	}
	return AuthModeDisabled
}

func ResolveRuntimeAccessControlState(stored AccessControlState) RuntimeAccessControlState {
	state := RuntimeAccessControlState{
		Stored:        stored,
		AuthMode:      AuthModeDisabled,
		PrincipalKind: RuntimePrincipalKindLocal,
	}
	if stored.Active == nil {
		return state
	}

	active := *stored.Active
	state.AuthMode = AuthModeOIDC
	state.LoginRequired = true
	state.PrincipalKind = RuntimePrincipalKindHuman
	state.SessionGovernanceEnabled = true
	state.ResolvedOIDCConfig = &active
	return state
}

func DefaultDraftOIDCConfig() DraftOIDCConfig {
	return DraftOIDCConfig{
		RedirectURL: setupDefaultOIDCRedirectURL,
		Scopes:      defaultOIDCScopes(),
		Claims:      defaultOIDCClaims(),
		SessionPolicy: OIDCSessionPolicy{
			SessionTTL:     defaultOIDCSessionTTL,
			SessionIdleTTL: defaultOIDCIdleTTL,
		},
	}
}

const (
	// #nosec G101 -- This is the local default callback URL template, not a credential.
	setupDefaultOIDCRedirectURL = "http://127.0.0.1:19836/api/v1/auth/oidc/callback"
	defaultOIDCSessionTTL       = 8 * time.Hour
	defaultOIDCIdleTTL          = 30 * time.Minute
)

func defaultOIDCScopes() []string {
	return []string{"openid", "profile", "email", "groups"}
}

func defaultOIDCClaims() OIDCClaims {
	return OIDCClaims{
		EmailClaim:    "email",
		NameClaim:     "name",
		UsernameClaim: "preferred_username",
		GroupsClaim:   "groups",
	}
}

func parseDraftOIDCConfig(input AccessControlStateInput) (DraftOIDCConfig, error) {
	template := DefaultDraftOIDCConfig()
	sessionTTL, err := parseAccessControlDuration("session_ttl", input.SessionTTL, template.SessionPolicy.SessionTTL)
	if err != nil {
		return DraftOIDCConfig{}, err
	}
	sessionIdleTTL, err := parseAccessControlDuration("session_idle_ttl", input.SessionIdleTTL, template.SessionPolicy.SessionIdleTTL)
	if err != nil {
		return DraftOIDCConfig{}, err
	}
	if sessionIdleTTL > sessionTTL {
		return DraftOIDCConfig{}, fmt.Errorf("session_idle_ttl must not exceed session_ttl")
	}

	claims := OIDCClaims{
		EmailClaim:    fallbackTrimmed(input.EmailClaim, template.Claims.EmailClaim),
		NameClaim:     fallbackTrimmed(input.NameClaim, template.Claims.NameClaim),
		UsernameClaim: fallbackTrimmed(input.UsernameClaim, template.Claims.UsernameClaim),
		GroupsClaim:   fallbackTrimmed(input.GroupsClaim, template.Claims.GroupsClaim),
	}

	return DraftOIDCConfig{
		IssuerURL:            strings.TrimSpace(input.IssuerURL),
		ClientID:             strings.TrimSpace(input.ClientID),
		ClientSecret:         strings.TrimSpace(input.ClientSecret),
		RedirectURL:          fallbackTrimmed(input.RedirectURL, template.RedirectURL),
		Scopes:               normalizeList(input.Scopes, false, template.Scopes),
		Claims:               claims,
		AllowedEmailDomains:  normalizeList(input.AllowedEmailDomains, true, nil),
		BootstrapAdminEmails: normalizeList(input.BootstrapAdminEmails, true, nil),
		SessionPolicy: OIDCSessionPolicy{
			SessionTTL:     sessionTTL,
			SessionIdleTTL: sessionIdleTTL,
		},
	}, nil
}

func parseActiveOIDCConfig(input AccessControlStateInput) (ActiveOIDCConfig, error) {
	draft, err := parseDraftOIDCConfig(input)
	if err != nil {
		return ActiveOIDCConfig{}, err
	}
	if draft.IssuerURL == "" {
		return ActiveOIDCConfig{}, fmt.Errorf("issuer_url is required for active oidc config")
	}
	if draft.ClientID == "" {
		return ActiveOIDCConfig{}, fmt.Errorf("client_id is required for active oidc config")
	}
	if draft.ClientSecret == "" {
		return ActiveOIDCConfig{}, fmt.Errorf("client_secret is required for active oidc config")
	}
	return ActiveOIDCConfig(draft), nil
}

func parseAccessControlDuration(fieldName string, raw string, fallback time.Duration) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", fieldName, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", fieldName)
	}
	return parsed, nil
}

func normalizeOIDCValidationMetadata(input OIDCValidationMetadataInput) OIDCValidationMetadata {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "not_tested"
	}
	message := strings.TrimSpace(input.Message)
	if message == "" {
		message = "No OIDC validation has been recorded yet."
	}
	warnings := normalizeList(input.Warnings, false, nil)
	return OIDCValidationMetadata{
		Status:                status,
		Message:               message,
		CheckedAt:             cloneTime(input.CheckedAt),
		IssuerURL:             strings.TrimSpace(input.IssuerURL),
		AuthorizationEndpoint: strings.TrimSpace(input.AuthorizationEndpoint),
		TokenEndpoint:         strings.TrimSpace(input.TokenEndpoint),
		RedirectURL:           strings.TrimSpace(input.RedirectURL),
		Warnings:              warnings,
	}
}

func normalizeOIDCActivationMetadata(input OIDCActivationMetadataInput) OIDCActivationMetadata {
	return OIDCActivationMetadata{
		ActivatedAt: cloneTime(input.ActivatedAt),
		ActivatedBy: strings.TrimSpace(input.ActivatedBy),
		Source:      strings.TrimSpace(input.Source),
		Message:     strings.TrimSpace(input.Message),
	}
}

func normalizeList(items []string, lower bool, fallback []string) []string {
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
	if len(result) == 0 && fallback != nil {
		return append([]string(nil), fallback...)
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func fallbackTrimmed(raw string, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func cloneTime(raw *time.Time) *time.Time {
	if raw == nil {
		return nil
	}
	cloned := raw.UTC()
	return &cloned
}
