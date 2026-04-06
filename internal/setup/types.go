package setup

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

type DatabaseSourceType string
type AuthModeType string

const (
	DatabaseSourceManual DatabaseSourceType = "manual"
	DatabaseSourceDocker DatabaseSourceType = "docker"
)

const (
	AuthModeDisabled AuthModeType = "disabled"
	AuthModeOIDC     AuthModeType = "oidc"
)

const (
	DefaultOrganizationName = "Local Workspace"
	DefaultOrganizationSlug = "local-workspace"
	DefaultProjectName      = "OpenASE Workspace"
	DefaultProjectSlug      = "openase-workspace"
	DefaultOIDCRedirectURL  = "http://127.0.0.1:19836/api/v1" + "/auth/oidc/callback"
	DefaultOIDCScopes       = "openid,profile,email,groups"
	DefaultOIDCSessionTTL   = "8h"
	DefaultOIDCIdleTTL      = "30m"
)

type DatabaseSourceOption struct {
	ID          DatabaseSourceType `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
}

type CLIDiagnostic struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

type AuthModeOption struct {
	ID          AuthModeType `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
}

type AgentOption struct {
	ID          string                                 `json:"id"`
	Name        string                                 `json:"name"`
	Command     string                                 `json:"command"`
	AdapterType catalogdomain.AgentProviderAdapterType `json:"adapter_type"`
	ModelName   string                                 `json:"model_name"`
	Available   bool                                   `json:"available"`
	Path        string                                 `json:"path,omitempty"`
	Version     string                                 `json:"version,omitempty"`
}

type Bootstrap struct {
	ConfigExists bool                   `json:"config_exists"`
	ConfigPath   string                 `json:"config_path"`
	Sources      []DatabaseSourceOption `json:"sources"`
	AuthModes    []AuthModeOption       `json:"auth_modes"`
	Agents       []AgentOption          `json:"agents"`
	CLI          []CLIDiagnostic        `json:"cli"`
	Defaults     Defaults               `json:"defaults"`
}

type Defaults struct {
	ManualDatabase RawDatabaseInput       `json:"manual_database"`
	DockerDatabase RawDockerDatabaseInput `json:"docker_database"`
	Auth           RawAuthInput           `json:"auth"`
}

type RawDatabaseInput struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`
}

type RawDockerDatabaseInput struct {
	ContainerName string `json:"container_name"`
	DatabaseName  string `json:"database_name"`
	User          string `json:"user"`
	Port          int    `json:"port"`
	VolumeName    string `json:"volume_name"`
	Image         string `json:"image"`
}

type RawDatabaseSourceInput struct {
	Type   string                  `json:"type"`
	Manual *RawDatabaseInput       `json:"manual,omitempty"`
	Docker *RawDockerDatabaseInput `json:"docker,omitempty"`
}

type RawOIDCInput struct {
	IssuerURL            string `json:"issuer_url"`
	ClientID             string `json:"client_id"`
	ClientSecret         string `json:"client_secret"`
	RedirectURL          string `json:"redirect_url"`
	Scopes               string `json:"scopes"`
	AllowedEmailDomains  string `json:"allowed_email_domains,omitempty"`
	BootstrapAdminEmails string `json:"bootstrap_admin_emails,omitempty"`
	SessionTTL           string `json:"session_ttl,omitempty"`
	SessionIdleTTL       string `json:"session_idle_ttl,omitempty"`
}

type RawAuthInput struct {
	Mode string        `json:"mode"`
	OIDC *RawOIDCInput `json:"oidc,omitempty"`
}

type RawCompleteRequest struct {
	Database       RawDatabaseInput `json:"database"`
	Auth           RawAuthInput     `json:"auth"`
	AllowOverwrite bool             `json:"allow_overwrite,omitempty"`
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type DockerDatabaseConfig struct {
	ContainerName string
	DatabaseName  string
	User          string
	Port          int
	VolumeName    string
	Image         string
}

type DatabaseSourceConfig struct {
	Type   DatabaseSourceType
	Manual *DatabaseConfig
	Docker *DockerDatabaseConfig
}

type CompleteRequest struct {
	Database       DatabaseConfig
	Auth           AuthConfig
	AllowOverwrite bool
}

type PreparedDatabase struct {
	Source DatabaseSourceType
	Config DatabaseConfig
	Docker *DockerDatabaseRuntime
}

type DockerDatabaseRuntime struct {
	ContainerName string `json:"container_name"`
	DatabaseName  string `json:"database_name"`
	User          string `json:"user"`
	Port          int    `json:"port"`
	VolumeName    string `json:"volume_name"`
	Image         string `json:"image"`
}

type OIDCConfig struct {
	IssuerURL            string
	ClientID             string
	ClientSecret         string
	RedirectURL          string
	Scopes               []string
	AllowedEmailDomains  []string
	BootstrapAdminEmails []string
	SessionTTL           string
	SessionIdleTTL       string
}

type AuthConfig struct {
	Mode AuthModeType
	OIDC *OIDCConfig
}

func ParseAuthInput(raw RawAuthInput) (AuthConfig, error) {
	return parseAuthInput(raw)
}

type CompleteResult struct {
	ConfigPath       string `json:"config_path"`
	EnvPath          string `json:"env_path"`
	OrganizationName string `json:"organization_name"`
	OrganizationSlug string `json:"organization_slug"`
	ProjectName      string `json:"project_name"`
	ProjectSlug      string `json:"project_slug"`
}

func defaultRawDatabaseInput() RawDatabaseInput {
	return RawDatabaseInput{
		Host:    "127.0.0.1",
		Port:    5432,
		Name:    "openase",
		User:    "openase",
		SSLMode: "disable",
	}
}

func defaultRawDockerDatabaseInput() RawDockerDatabaseInput {
	return RawDockerDatabaseInput{
		ContainerName: "openase-local-postgres",
		DatabaseName:  "openase",
		User:          "openase",
		Port:          15432,
		VolumeName:    "openase-local-postgres-data",
		Image:         "postgres:16-alpine",
	}
}

func defaultRawAuthInput() RawAuthInput {
	return RawAuthInput{
		Mode: string(AuthModeDisabled),
		OIDC: &RawOIDCInput{
			ClientID:       "openase",
			RedirectURL:    DefaultOIDCRedirectURL,
			Scopes:         DefaultOIDCScopes,
			SessionTTL:     DefaultOIDCSessionTTL,
			SessionIdleTTL: DefaultOIDCIdleTTL,
		},
	}
}

func parseDatabaseSourceType(raw string) (DatabaseSourceType, error) {
	sourceType := DatabaseSourceType(strings.TrimSpace(strings.ToLower(raw)))
	switch sourceType {
	case DatabaseSourceManual, DatabaseSourceDocker:
		return sourceType, nil
	default:
		return "", fmt.Errorf("database.type must be one of manual, docker")
	}
}

func parseAuthModeType(raw string) (AuthModeType, error) {
	mode := AuthModeType(strings.TrimSpace(strings.ToLower(raw)))
	if mode == "" {
		mode = AuthModeDisabled
	}
	switch mode {
	case AuthModeDisabled, AuthModeOIDC:
		return mode, nil
	default:
		return "", fmt.Errorf("auth.mode must be one of disabled, oidc")
	}
}

func parseDatabaseInput(raw RawDatabaseInput) (DatabaseConfig, error) {
	host := strings.TrimSpace(raw.Host)
	if host == "" {
		return DatabaseConfig{}, fmt.Errorf("database.host must not be empty")
	}

	port := raw.Port
	if port == 0 {
		port = 5432
	}
	if port < 1 || port > 65535 {
		return DatabaseConfig{}, fmt.Errorf("database.port must be between 1 and 65535")
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return DatabaseConfig{}, fmt.Errorf("database.name must not be empty")
	}

	user := strings.TrimSpace(raw.User)
	if user == "" {
		return DatabaseConfig{}, fmt.Errorf("database.user must not be empty")
	}

	sslMode := strings.TrimSpace(strings.ToLower(raw.SSLMode))
	if sslMode == "" {
		sslMode = "disable"
	}
	switch sslMode {
	case "disable", "require", "verify-full":
	default:
		return DatabaseConfig{}, fmt.Errorf("database.ssl_mode must be one of disable, require, verify-full")
	}

	return DatabaseConfig{
		Host:     host,
		Port:     port,
		Name:     name,
		User:     user,
		Password: raw.Password,
		SSLMode:  sslMode,
	}, nil
}

func (c DatabaseConfig) DSN() string {
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:     c.Name,
		RawQuery: "sslmode=" + url.QueryEscape(c.SSLMode),
	}).String()
}

func (c DatabaseConfig) Raw() RawDatabaseInput {
	return RawDatabaseInput(c)
}

func (c AuthConfig) Raw() RawAuthInput {
	raw := RawAuthInput{Mode: string(c.Mode)}
	if c.OIDC == nil {
		return raw
	}

	raw.OIDC = &RawOIDCInput{
		IssuerURL:            c.OIDC.IssuerURL,
		ClientID:             c.OIDC.ClientID,
		ClientSecret:         c.OIDC.ClientSecret,
		RedirectURL:          c.OIDC.RedirectURL,
		Scopes:               strings.Join(c.OIDC.Scopes, ","),
		AllowedEmailDomains:  strings.Join(c.OIDC.AllowedEmailDomains, ","),
		BootstrapAdminEmails: strings.Join(c.OIDC.BootstrapAdminEmails, ","),
		SessionTTL:           c.OIDC.SessionTTL,
		SessionIdleTTL:       c.OIDC.SessionIdleTTL,
	}
	return raw
}

func parseDockerDatabaseInput(raw RawDockerDatabaseInput) (DockerDatabaseConfig, error) {
	defaults := defaultRawDockerDatabaseInput()

	containerName := strings.TrimSpace(raw.ContainerName)
	if containerName == "" {
		containerName = defaults.ContainerName
	}

	databaseName := strings.TrimSpace(raw.DatabaseName)
	if databaseName == "" {
		databaseName = defaults.DatabaseName
	}

	user := strings.TrimSpace(raw.User)
	if user == "" {
		user = defaults.User
	}

	port := raw.Port
	if port == 0 {
		port = defaults.Port
	}
	if port < 1 || port > 65535 {
		return DockerDatabaseConfig{}, fmt.Errorf("database.docker.port must be between 1 and 65535")
	}

	volumeName := strings.TrimSpace(raw.VolumeName)
	if volumeName == "" {
		volumeName = defaults.VolumeName
	}

	image := strings.TrimSpace(raw.Image)
	if image == "" {
		image = defaults.Image
	}

	return DockerDatabaseConfig{
		ContainerName: containerName,
		DatabaseName:  databaseName,
		User:          user,
		Port:          port,
		VolumeName:    volumeName,
		Image:         image,
	}, nil
}

func parseDatabaseSourceInput(raw RawDatabaseSourceInput) (DatabaseSourceConfig, error) {
	sourceType, err := parseDatabaseSourceType(raw.Type)
	if err != nil {
		return DatabaseSourceConfig{}, err
	}

	switch sourceType {
	case DatabaseSourceManual:
		manualRaw := defaultRawDatabaseInput()
		if raw.Manual != nil {
			manualRaw = *raw.Manual
		}
		manual, err := parseDatabaseInput(manualRaw)
		if err != nil {
			return DatabaseSourceConfig{}, err
		}
		return DatabaseSourceConfig{
			Type:   DatabaseSourceManual,
			Manual: &manual,
		}, nil
	case DatabaseSourceDocker:
		dockerRaw := defaultRawDockerDatabaseInput()
		if raw.Docker != nil {
			dockerRaw = *raw.Docker
		}
		docker, err := parseDockerDatabaseInput(dockerRaw)
		if err != nil {
			return DatabaseSourceConfig{}, err
		}
		return DatabaseSourceConfig{
			Type:   DatabaseSourceDocker,
			Docker: &docker,
		}, nil
	default:
		return DatabaseSourceConfig{}, fmt.Errorf("unsupported database source %q", sourceType)
	}
}

func parseCompleteRequest(raw RawCompleteRequest) (CompleteRequest, error) {
	database, err := parseDatabaseInput(raw.Database)
	if err != nil {
		return CompleteRequest{}, err
	}
	auth, err := parseAuthInput(raw.Auth)
	if err != nil {
		return CompleteRequest{}, err
	}

	return CompleteRequest{
		Database:       database,
		Auth:           auth,
		AllowOverwrite: raw.AllowOverwrite,
	}, nil
}

func parseAuthInput(raw RawAuthInput) (AuthConfig, error) {
	mode, err := parseAuthModeType(raw.Mode)
	if err != nil {
		return AuthConfig{}, err
	}
	if mode == AuthModeDisabled {
		return AuthConfig{Mode: AuthModeDisabled}, nil
	}

	oidcRaw := defaultRawAuthInput().OIDC
	if raw.OIDC != nil {
		oidcRaw = raw.OIDC
	}
	if oidcRaw == nil {
		return AuthConfig{}, fmt.Errorf("auth.oidc must be provided when auth.mode=oidc")
	}

	oidc, err := parseOIDCInput(*oidcRaw)
	if err != nil {
		return AuthConfig{}, err
	}

	return AuthConfig{
		Mode: AuthModeOIDC,
		OIDC: &oidc,
	}, nil
}

func parseOIDCInput(raw RawOIDCInput) (OIDCConfig, error) {
	defaults := defaultRawAuthInput().OIDC
	if defaults == nil {
		return OIDCConfig{}, fmt.Errorf("oidc defaults are unavailable")
	}

	issuerURL := strings.TrimSpace(raw.IssuerURL)
	if issuerURL == "" {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.issuer_url must not be empty when auth.mode=oidc")
	}

	clientID := strings.TrimSpace(raw.ClientID)
	if clientID == "" {
		clientID = defaults.ClientID
	}
	if clientID == "" {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.client_id must not be empty when auth.mode=oidc")
	}

	clientSecret := strings.TrimSpace(raw.ClientSecret)
	if clientSecret == "" {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.client_secret must not be empty when auth.mode=oidc")
	}

	redirectURL := strings.TrimSpace(raw.RedirectURL)
	if redirectURL == "" {
		redirectURL = defaults.RedirectURL
	}
	if redirectURL == "" {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.redirect_url must not be empty when auth.mode=oidc")
	}

	scopesInput := strings.TrimSpace(raw.Scopes)
	if scopesInput == "" {
		scopesInput = defaults.Scopes
	}
	scopes := normalizeCSV(scopesInput)
	if len(scopes) == 0 {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.scopes must not be empty when auth.mode=oidc")
	}

	sessionTTL := strings.TrimSpace(raw.SessionTTL)
	if sessionTTL == "" {
		sessionTTL = defaults.SessionTTL
	}
	if _, err := time.ParseDuration(sessionTTL); err != nil {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.session_ttl must be a valid duration: %w", err)
	}

	sessionIdleTTL := strings.TrimSpace(raw.SessionIdleTTL)
	if sessionIdleTTL == "" {
		sessionIdleTTL = defaults.SessionIdleTTL
	}
	idleDuration, err := time.ParseDuration(sessionIdleTTL)
	if err != nil {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.session_idle_ttl must be a valid duration: %w", err)
	}

	sessionDuration, err := time.ParseDuration(sessionTTL)
	if err != nil {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.session_ttl must be a valid duration: %w", err)
	}
	if idleDuration > sessionDuration {
		return OIDCConfig{}, fmt.Errorf("auth.oidc.session_idle_ttl must not exceed auth.oidc.session_ttl")
	}

	return OIDCConfig{
		IssuerURL:            issuerURL,
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		RedirectURL:          redirectURL,
		Scopes:               scopes,
		AllowedEmailDomains:  normalizeCSV(raw.AllowedEmailDomains),
		BootstrapAdminEmails: normalizeCSV(raw.BootstrapAdminEmails),
		SessionTTL:           sessionTTL,
		SessionIdleTTL:       sessionIdleTTL,
	}, nil
}

func normalizeCSV(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}

	parts := strings.Split(trimmed, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	return items
}
