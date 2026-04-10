package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func clearOpenASEEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"OPENASE_SERVER_MODE",
		"OPENASE_SERVER_HOST",
		"OPENASE_SERVER_PORT",
		"OPENASE_SECURITY_CIPHER_SEED",
		"OPENASE_GITHUB_WEBHOOK_SECRET",
		"OPENASE_DATABASE_DSN",
		"OPENASE_ORCHESTRATOR_TICK_INTERVAL",
		"OPENASE_ORCHESTRATOR_WORKSPACE_PREPARE_TIMEOUT",
		"OPENASE_ORCHESTRATOR_AGENT_SESSION_START_TIMEOUT",
		"OPENASE_EVENT_DRIVER",
		"OPENASE_OBSERVABILITY_METRICS_ENABLED",
		"OPENASE_OBSERVABILITY_METRICS_EXPORT_PROMETHEUS",
		"OPENASE_OBSERVABILITY_METRICS_EXPORT_OTLP_ENDPOINT",
		"OPENASE_OBSERVABILITY_TRACING_ENABLED",
		"OPENASE_OBSERVABILITY_TRACING_ENDPOINT",
		"OPENASE_OBSERVABILITY_TRACING_SERVICE_NAME",
		"OPENASE_OBSERVABILITY_TRACING_SAMPLE_RATIO",
		"OPENASE_LOG_FORMAT",
		"OPENASE_LOG_LEVEL",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearOpenASEEnv(t)
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Mode != ServerModeAllInOne {
		t.Fatalf("expected default server mode all-in-one, got %q", cfg.Server.Mode)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected default host, got %q", cfg.Server.Host)
	}

	if cfg.Server.Port != 40023 {
		t.Fatalf("expected default port, got %d", cfg.Server.Port)
	}

	if cfg.Orchestrator.TickInterval != 5*time.Second {
		t.Fatalf("expected default tick interval, got %s", cfg.Orchestrator.TickInterval)
	}
	if cfg.Orchestrator.WorkspacePrepareTimeout != 5*time.Minute {
		t.Fatalf("expected default workspace prepare timeout, got %s", cfg.Orchestrator.WorkspacePrepareTimeout)
	}
	if cfg.Orchestrator.AgentSessionStartTimeout != 30*time.Second {
		t.Fatalf("expected default agent session start timeout, got %s", cfg.Orchestrator.AgentSessionStartTimeout)
	}

	if cfg.Logging.Level != slog.LevelInfo {
		t.Fatalf("expected default log level info, got %s", cfg.Logging.Level)
	}

	if cfg.Logging.Format != LogFormatText {
		t.Fatalf("expected default log format text, got %q", cfg.Logging.Format)
	}

	if !cfg.Observability.Metrics.Enabled {
		t.Fatal("expected metrics to be enabled by default")
	}

	if cfg.Observability.Metrics.Export.Prometheus {
		t.Fatal("expected prometheus export to be disabled by default")
	}

	if cfg.Observability.Tracing.Enabled {
		t.Fatal("expected tracing to be disabled by default")
	}

	if cfg.Observability.Tracing.ServiceName != "openase" {
		t.Fatalf("expected default tracing service name, got %q", cfg.Observability.Tracing.ServiceName)
	}

	if cfg.Observability.Tracing.SampleRatio != 1 {
		t.Fatalf("expected default tracing sample ratio 1, got %v", cfg.Observability.Tracing.SampleRatio)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	clearOpenASEEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_PORT", "41000")
	t.Setenv("OPENASE_SERVER_MODE", "serve")
	t.Setenv("OPENASE_SECURITY_CIPHER_SEED", "shared-cluster-seed")
	t.Setenv("OPENASE_GITHUB_WEBHOOK_SECRET", "topsecret")
	t.Setenv("OPENASE_DATABASE_DSN", "postgres://openase:secret@localhost:5432/openase?sslmode=disable")
	t.Setenv("OPENASE_ORCHESTRATOR_TICK_INTERVAL", "2s")
	t.Setenv("OPENASE_ORCHESTRATOR_WORKSPACE_PREPARE_TIMEOUT", "6m")
	t.Setenv("OPENASE_ORCHESTRATOR_AGENT_SESSION_START_TIMEOUT", "45s")
	t.Setenv("OPENASE_EVENT_DRIVER", "pgnotify")
	t.Setenv("OPENASE_OBSERVABILITY_METRICS_ENABLED", "false")
	t.Setenv("OPENASE_OBSERVABILITY_METRICS_EXPORT_PROMETHEUS", "true")
	t.Setenv("OPENASE_OBSERVABILITY_METRICS_EXPORT_OTLP_ENDPOINT", "collector.internal:4318")
	t.Setenv("OPENASE_OBSERVABILITY_TRACING_ENABLED", "true")
	t.Setenv("OPENASE_OBSERVABILITY_TRACING_ENDPOINT", "http://collector.internal:4318/v1/traces")
	t.Setenv("OPENASE_OBSERVABILITY_TRACING_SERVICE_NAME", "openase-dev")
	t.Setenv("OPENASE_OBSERVABILITY_TRACING_SAMPLE_RATIO", "0.5")
	t.Setenv("OPENASE_LOG_FORMAT", "json")
	t.Setenv("OPENASE_LOG_LEVEL", "debug")

	cfg, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != 41000 {
		t.Fatalf("expected env port, got %d", cfg.Server.Port)
	}

	if cfg.Server.Mode != ServerModeServe {
		t.Fatalf("expected serve mode, got %q", cfg.Server.Mode)
	}

	if cfg.Security.CipherSeed != "shared-cluster-seed" {
		t.Fatalf("expected security cipher seed from env, got %q", cfg.Security.CipherSeed)
	}

	if cfg.GitHub.WebhookSecret != "topsecret" {
		t.Fatalf("expected GitHub webhook secret from env, got %q", cfg.GitHub.WebhookSecret)
	}

	if cfg.Database.DSN == "" {
		t.Fatal("expected database dsn from env")
	}

	if cfg.Orchestrator.TickInterval != 2*time.Second {
		t.Fatalf("expected env tick interval, got %s", cfg.Orchestrator.TickInterval)
	}
	if cfg.Orchestrator.WorkspacePrepareTimeout != 6*time.Minute {
		t.Fatalf("expected env workspace prepare timeout, got %s", cfg.Orchestrator.WorkspacePrepareTimeout)
	}
	if cfg.Orchestrator.AgentSessionStartTimeout != 45*time.Second {
		t.Fatalf("expected env agent session start timeout, got %s", cfg.Orchestrator.AgentSessionStartTimeout)
	}

	if cfg.Event.Driver != EventDriverPGNotify {
		t.Fatalf("expected pgnotify event driver, got %q", cfg.Event.Driver)
	}

	if cfg.Observability.Metrics.Enabled {
		t.Fatal("expected metrics to be disabled from env")
	}

	if !cfg.Observability.Metrics.Export.Prometheus {
		t.Fatal("expected prometheus export to be enabled from env")
	}

	if cfg.Observability.Metrics.Export.OTLPEndpoint != "collector.internal:4318" {
		t.Fatalf("expected OTLP endpoint from env, got %q", cfg.Observability.Metrics.Export.OTLPEndpoint)
	}

	if !cfg.Observability.Tracing.Enabled {
		t.Fatal("expected tracing enabled from env")
	}

	if cfg.Observability.Tracing.Endpoint != "http://collector.internal:4318/v1/traces" {
		t.Fatalf("expected tracing endpoint from env, got %q", cfg.Observability.Tracing.Endpoint)
	}

	if cfg.Observability.Tracing.ServiceName != "openase-dev" {
		t.Fatalf("expected tracing service name from env, got %q", cfg.Observability.Tracing.ServiceName)
	}

	if cfg.Observability.Tracing.SampleRatio != 0.5 {
		t.Fatalf("expected tracing sample ratio 0.5, got %v", cfg.Observability.Tracing.SampleRatio)
	}

	if cfg.Logging.Format != LogFormatJSON {
		t.Fatalf("expected json log format, got %q", cfg.Logging.Format)
	}

	if cfg.Logging.Level != slog.LevelDebug {
		t.Fatalf("expected debug log level, got %s", cfg.Logging.Level)
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	clearOpenASEEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	writeFile(t, configPath, []byte(`
server:
  mode: serve
  host: 127.0.0.1
  port: 40123
  read_timeout: 20s
  write_timeout: 25s
  shutdown_timeout: 12s
github:
  webhook_secret: config-file-secret
security:
  cipher_seed: config-file-shared-seed
database:
  dsn: postgres://openase:secret@localhost:5432/openase?sslmode=disable
orchestrator:
  tick_interval: 3s
  workspace_prepare_timeout: 7m
  agent_session_start_timeout: 40s
event:
  driver: pgnotify
observability:
  metrics:
    enabled: true
    export:
      prometheus: true
      otlp_endpoint: https://collector.example.test/v1/metrics
  tracing:
    enabled: true
    endpoint: http://collector.internal:4318/v1/traces
    service_name: openase-prod
    sample_ratio: 0.25
log:
  level: warn
  format: json
`))

	cfg, err := Load(LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Metadata.ConfigFile != configPath {
		t.Fatalf("expected config file metadata %q, got %q", configPath, cfg.Metadata.ConfigFile)
	}

	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 40123 {
		t.Fatalf("unexpected server config: %+v", cfg.Server)
	}

	if cfg.Server.Mode != ServerModeServe {
		t.Fatalf("expected serve mode, got %q", cfg.Server.Mode)
	}

	if cfg.Security.CipherSeed != "config-file-shared-seed" {
		t.Fatalf("expected config file security cipher seed, got %q", cfg.Security.CipherSeed)
	}

	if cfg.GitHub.WebhookSecret != "config-file-secret" {
		t.Fatalf("expected config file GitHub webhook secret, got %q", cfg.GitHub.WebhookSecret)
	}

	if cfg.Database.DSN == "" {
		t.Fatal("expected config file database dsn")
	}

	if cfg.Server.ReadTimeout != 20*time.Second {
		t.Fatalf("expected read timeout 20s, got %s", cfg.Server.ReadTimeout)
	}

	if cfg.Orchestrator.TickInterval != 3*time.Second {
		t.Fatalf("expected tick interval 3s, got %s", cfg.Orchestrator.TickInterval)
	}
	if cfg.Orchestrator.WorkspacePrepareTimeout != 7*time.Minute {
		t.Fatalf("expected workspace prepare timeout 7m, got %s", cfg.Orchestrator.WorkspacePrepareTimeout)
	}
	if cfg.Orchestrator.AgentSessionStartTimeout != 40*time.Second {
		t.Fatalf("expected agent session start timeout 40s, got %s", cfg.Orchestrator.AgentSessionStartTimeout)
	}

	if cfg.Event.Driver != EventDriverPGNotify {
		t.Fatalf("expected pgnotify driver, got %q", cfg.Event.Driver)
	}

	if !cfg.Observability.Metrics.Enabled {
		t.Fatal("expected metrics enabled from config file")
	}

	if !cfg.Observability.Metrics.Export.Prometheus {
		t.Fatal("expected prometheus export from config file")
	}

	if cfg.Observability.Metrics.Export.OTLPEndpoint != "https://collector.example.test/v1/metrics" {
		t.Fatalf("expected OTLP endpoint from config file, got %q", cfg.Observability.Metrics.Export.OTLPEndpoint)
	}

	if !cfg.Observability.Tracing.Enabled {
		t.Fatal("expected tracing enabled from config file")
	}

	if cfg.Observability.Tracing.Endpoint != "http://collector.internal:4318/v1/traces" {
		t.Fatalf("expected tracing endpoint from config file, got %q", cfg.Observability.Tracing.Endpoint)
	}

	if cfg.Observability.Tracing.ServiceName != "openase-prod" {
		t.Fatalf("expected tracing service name from config file, got %q", cfg.Observability.Tracing.ServiceName)
	}

	if cfg.Observability.Tracing.SampleRatio != 0.25 {
		t.Fatalf("expected tracing sample ratio 0.25, got %v", cfg.Observability.Tracing.SampleRatio)
	}

	if cfg.Logging.Level != slog.LevelWarn {
		t.Fatalf("expected warn log level, got %s", cfg.Logging.Level)
	}
}

func TestParseAuthConfigOIDCNormalizesClaimsAndLists(t *testing.T) {
	v := viper.New()
	configureAuthDefaults(v)
	v.Set("auth.mode", "oidc")
	v.Set("auth.csrf.trusted_origins", " http://LOCALHOST:4173/ , https://Admin.EXAMPLE.com ")
	v.Set("auth.oidc.issuer_url", " https://idp.example.com ")
	v.Set("auth.oidc.client_id", " openase ")
	v.Set("auth.oidc.client_secret", " super-secret ")
	v.Set("auth.oidc.redirect_url", " http://127.0.0.1:19836/api/v1/auth/oidc/callback ")
	v.Set("auth.oidc.scopes", "openid, profile , email , groups")
	v.Set("auth.oidc.allowed_email_domains", "Example.com, ops.example.com ")
	v.Set("auth.oidc.bootstrap_admin_emails", []string{" Admin@Example.com ", "owner@example.com"})
	v.Set("auth.oidc.session_ttl", "8h")
	v.Set("auth.oidc.session_idle_ttl", "30m")

	cfg, err := parseAuthConfig(v)
	if err != nil {
		t.Fatalf("parseAuthConfig() error = %v", err)
	}
	if err := validateAuthConfig(cfg); err != nil {
		t.Fatalf("validateAuthConfig() error = %v", err)
	}

	if cfg.Mode != AuthModeOIDC {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, AuthModeOIDC)
	}
	if got, want := cfg.CSRF.TrustedOrigins, []string{"http://localhost:4173", "https://admin.example.com"}; !slicesEqual(got, want) {
		t.Fatalf("TrustedOrigins = %#v, want %#v", got, want)
	}
	if cfg.OIDC.IssuerURL != "https://idp.example.com" {
		t.Fatalf("IssuerURL = %q", cfg.OIDC.IssuerURL)
	}
	if cfg.OIDC.ClientID != "openase" {
		t.Fatalf("ClientID = %q", cfg.OIDC.ClientID)
	}
	if cfg.OIDC.ClientSecret != "super-secret" {
		t.Fatalf("ClientSecret = %q", cfg.OIDC.ClientSecret)
	}
	if cfg.OIDC.RedirectURL != "http://127.0.0.1:19836/api/v1/auth/oidc/callback" {
		t.Fatalf("RedirectURL = %q", cfg.OIDC.RedirectURL)
	}
	if got, want := cfg.OIDC.Scopes, []string{"openid", "profile", "email", "groups"}; !slicesEqual(got, want) {
		t.Fatalf("Scopes = %#v, want %#v", got, want)
	}
	if got, want := cfg.OIDC.AllowedEmailDomains, []string{"example.com", "ops.example.com"}; !slicesEqual(got, want) {
		t.Fatalf("AllowedEmailDomains = %#v, want %#v", got, want)
	}
	if got, want := cfg.OIDC.BootstrapAdminEmails, []string{"admin@example.com", "owner@example.com"}; !slicesEqual(got, want) {
		t.Fatalf("BootstrapAdminEmails = %#v, want %#v", got, want)
	}
	if cfg.OIDC.SessionTTL != 8*time.Hour {
		t.Fatalf("SessionTTL = %s, want 8h", cfg.OIDC.SessionTTL)
	}
	if cfg.OIDC.SessionIdleTTL != 30*time.Minute {
		t.Fatalf("SessionIdleTTL = %s, want 30m", cfg.OIDC.SessionIdleTTL)
	}
}

func TestParseAuthConfigOIDCDefaultsToNonExpiringSessions(t *testing.T) {
	v := viper.New()
	configureAuthDefaults(v)
	v.Set("auth.mode", "oidc")
	v.Set("auth.oidc.issuer_url", "https://idp.example.com")
	v.Set("auth.oidc.client_id", "openase")
	v.Set("auth.oidc.client_secret", "secret")
	v.Set("auth.oidc.redirect_url", "http://127.0.0.1:19836/api/v1/auth/oidc/callback")

	cfg, err := parseAuthConfig(v)
	if err != nil {
		t.Fatalf("parseAuthConfig() error = %v", err)
	}
	if err := validateAuthConfig(cfg); err != nil {
		t.Fatalf("validateAuthConfig() error = %v", err)
	}
	if cfg.OIDC.SessionTTL != 0 {
		t.Fatalf("SessionTTL = %s, want 0", cfg.OIDC.SessionTTL)
	}
	if cfg.OIDC.SessionIdleTTL != 0 {
		t.Fatalf("SessionIdleTTL = %s, want 0", cfg.OIDC.SessionIdleTTL)
	}
}

func TestValidateAuthConfigRejectsInvalidOIDCSettings(t *testing.T) {
	base := AuthConfig{
		Mode: AuthModeOIDC,
		OIDC: OIDCConfig{
			IssuerURL:    "https://idp.example.com",
			ClientID:     "openase",
			ClientSecret: "secret",
			RedirectURL:  "http://127.0.0.1:19836/api/v1/auth/oidc/callback",
			Scopes:       []string{"openid"},
			SessionTTL:   time.Hour,
		},
	}

	cases := []struct {
		name string
		mut  func(*AuthConfig)
		want string
	}{
		{
			name: "missing issuer",
			mut: func(cfg *AuthConfig) {
				cfg.OIDC.IssuerURL = ""
			},
			want: "auth.oidc.issuer_url is required when auth.mode=oidc",
		},
		{
			name: "missing client id",
			mut: func(cfg *AuthConfig) {
				cfg.OIDC.ClientID = ""
			},
			want: "auth.oidc.client_id is required when auth.mode=oidc",
		},
		{
			name: "idle ttl exceeds ttl",
			mut: func(cfg *AuthConfig) {
				cfg.OIDC.SessionIdleTTL = 2 * time.Hour
			},
			want: "auth.oidc.session_idle_ttl must not exceed auth.oidc.session_ttl",
		},
		{
			name: "invalid csrf trusted origin",
			mut: func(cfg *AuthConfig) {
				cfg.CSRF.TrustedOrigins = []string{"http://localhost:4173/callback"}
			},
			want: "auth.csrf.trusted_origins: invalid origin \"http://localhost:4173/callback\": path is not allowed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mut(&cfg)
			err := validateAuthConfig(cfg)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("validateAuthConfig() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestLoadIgnoresLegacyAuthParsingFailures(t *testing.T) {
	clearOpenASEEnv(t)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configBody := strings.TrimSpace(`
server:
  mode: all-in-one
  host: 127.0.0.1
  port: 19836
database:
  dsn: postgres://openase:secret@127.0.0.1:5432/openase?sslmode=disable
auth:
  mode: definitely-not-supported
  oidc:
    scopes:
      bad: shape
`)
	if err := os.WriteFile(configPath, []byte(configBody+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.DSN == "" {
		t.Fatalf("expected non-auth config to keep loading, got %+v", cfg)
	}
	if cfg.Auth.Mode != "" {
		t.Fatalf("expected legacy auth parse failure to fall back to zero auth config, got %+v", cfg.Auth)
	}
}

func TestParseAuthConfigRejectsInvalidCSRForigin(t *testing.T) {
	v := viper.New()
	configureAuthDefaults(v)
	v.Set("auth.csrf.trusted_origins", []string{"http://localhost:4173/path"})

	cfg, err := parseAuthConfig(v)
	if err != nil {
		t.Fatalf("parseAuthConfig() error = %v", err)
	}
	err = validateAuthConfig(cfg)
	if err == nil || err.Error() != "auth.csrf.trusted_origins: invalid origin \"http://localhost:4173/path\": path is not allowed" {
		t.Fatalf("validateAuthConfig() error = %v", err)
	}
}

func slicesEqual[T comparable](got []T, want []T) bool {
	if len(got) != len(want) {
		return false
	}
	for idx := range got {
		if got[idx] != want[idx] {
			return false
		}
	}
	return true
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	clearOpenASEEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_PORT", "70000")

	if _, err := Load(LoadOptions{}); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestLoadRejectsChannelDriverOutsideAllInOne(t *testing.T) {
	clearOpenASEEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_MODE", "serve")
	t.Setenv("OPENASE_EVENT_DRIVER", "channel")

	if _, err := Load(LoadOptions{}); err == nil {
		t.Fatal("expected invalid channel driver error")
	}
}

func TestLoadRejectsMissingDatabaseDSNForResolvedPGNotify(t *testing.T) {
	clearOpenASEEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_MODE", "serve")

	if _, err := Load(LoadOptions{}); err == nil {
		t.Fatal("expected missing database dsn error")
	}
}

func TestLoadRejectsInvalidTracingSampleRatio(t *testing.T) {
	clearOpenASEEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_OBSERVABILITY_TRACING_SAMPLE_RATIO", "1.2")

	if _, err := Load(LoadOptions{}); err == nil {
		t.Fatal("expected invalid tracing sample ratio error")
	}
}

func TestConfigHelperParsers(t *testing.T) {
	if got, err := parseNonEmptyString("  host  "); err != nil || got != "host" {
		t.Fatalf("parseNonEmptyString() = %q, %v", got, err)
	}
	if _, err := parseNonEmptyString("   "); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("parseNonEmptyString(blank) error = %v", err)
	}
	if _, err := parseNonEmptyString(1); err == nil || !strings.Contains(err.Error(), "unsupported string type") {
		t.Fatalf("parseNonEmptyString(type) error = %v", err)
	}

	if got, err := parseOptionalString("  value  "); err != nil || got != "value" {
		t.Fatalf("parseOptionalString() = %q, %v", got, err)
	}
	if _, err := parseOptionalString(true); err == nil {
		t.Fatalf("parseOptionalString(type) error = %v", err)
	}

	if got, err := parsePort(" 41000 "); err != nil || got != 41000 {
		t.Fatalf("parsePort(string) = %d, %v", got, err)
	}
	if got, err := parsePort(int64(41001)); err != nil || got != 41001 {
		t.Fatalf("parsePort(int64) = %d, %v", got, err)
	}
	if _, err := parsePort(70000); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("parsePort(range) error = %v", err)
	}

	if got, err := parseBool("TRUE"); err != nil || !got {
		t.Fatalf("parseBool(string) = %v, %v", got, err)
	}
	if _, err := parseBool(1); err == nil {
		t.Fatalf("parseBool(type) error = %v", err)
	}

	if got, err := parseDuration("3s"); err != nil || got != 3*time.Second {
		t.Fatalf("parseDuration(string) = %s, %v", got, err)
	}
	if _, err := parseDuration(0 * time.Second); err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("parseDuration(non-positive) error = %v", err)
	}
	if got, err := parseNonNegativeDuration("0s"); err != nil || got != 0 {
		t.Fatalf("parseNonNegativeDuration(zero) = %s, %v", got, err)
	}
	if _, err := parseNonNegativeDuration(-1 * time.Second); err == nil || !strings.Contains(err.Error(), "must not be negative") {
		t.Fatalf("parseNonNegativeDuration(negative) error = %v", err)
	}

	if got, err := parseUnitInterval("0.25"); err != nil || got != 0.25 {
		t.Fatalf("parseUnitInterval(string) = %v, %v", got, err)
	}
	if _, err := parseUnitInterval(2); err == nil || !strings.Contains(err.Error(), "between 0 and 1") {
		t.Fatalf("parseUnitInterval(range) error = %v", err)
	}

	if got, err := parseServerMode(ServerModeServe); err != nil || got != ServerModeServe {
		t.Fatalf("parseServerMode() = %q, %v", got, err)
	}
	if _, err := parseServerMode("weird"); err == nil || !strings.Contains(err.Error(), "unsupported server mode") {
		t.Fatalf("parseServerMode(invalid) error = %v", err)
	}

	if got, err := parseEventDriver(EventDriverChannel); err != nil || got != EventDriverChannel {
		t.Fatalf("parseEventDriver() = %q, %v", got, err)
	}
	if _, err := parseEventDriver("weird"); err == nil || !strings.Contains(err.Error(), "unsupported event driver") {
		t.Fatalf("parseEventDriver(invalid) error = %v", err)
	}

	if got, err := parseLogLevel("WARN"); err != nil || got != slog.LevelWarn {
		t.Fatalf("parseLogLevel() = %v, %v", got, err)
	}
	if _, err := parseLogLevel(true); err == nil {
		t.Fatalf("parseLogLevel(type) error = %v", err)
	}

	if got, err := parseLogFormat(LogFormatJSON); err != nil || got != LogFormatJSON {
		t.Fatalf("parseLogFormat() = %q, %v", got, err)
	}
	if _, err := parseLogFormat("weird"); err == nil || !strings.Contains(err.Error(), "unsupported log format") {
		t.Fatalf("parseLogFormat(invalid) error = %v", err)
	}
}

func TestConfigValidationHelpers(t *testing.T) {
	cfg := Config{
		Server: ServerConfig{Mode: ServerModeAllInOne},
		Event:  EventConfig{Driver: EventDriverAuto},
	}
	driver, err := cfg.ResolvedEventDriver()
	if err != nil || driver != EventDriverChannel {
		t.Fatalf("ResolvedEventDriver(all-in-one) = %q, %v", driver, err)
	}

	cfg = Config{
		Server:   ServerConfig{Mode: ServerModeServe},
		Event:    EventConfig{Driver: EventDriverAuto},
		Database: DatabaseConfig{DSN: "postgres://example"},
	}
	driver, err = cfg.ResolvedEventDriver()
	if err != nil || driver != EventDriverPGNotify {
		t.Fatalf("ResolvedEventDriver(serve) = %q, %v", driver, err)
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("validateConfig(valid) error = %v", err)
	}

	if _, err := (Config{Server: ServerConfig{Mode: "weird"}, Event: EventConfig{Driver: EventDriverAuto}}).ResolvedEventDriver(); err == nil {
		t.Fatal("ResolvedEventDriver(invalid mode) expected error")
	}
	if err := validateConfig(Config{Server: ServerConfig{Mode: ServerModeServe}, Event: EventConfig{Driver: EventDriverChannel}}); err == nil {
		t.Fatal("validateConfig(channel+serve) expected error")
	}
	if err := validateConfig(Config{Server: ServerConfig{Mode: ServerModeServe}, Event: EventConfig{Driver: EventDriverAuto}}); err == nil {
		t.Fatal("validateConfig(pgnotify without dsn) expected error")
	}

	cfg = Config{
		Security: SecurityConfig{CipherSeed: " shared-seed "},
		Database: DatabaseConfig{DSN: "postgres://ignored"},
	}
	if got := cfg.ResolvedSecurityCipherSeed(); got != "shared-seed" {
		t.Fatalf("ResolvedSecurityCipherSeed(explicit) = %q", got)
	}

	cfg = Config{Database: DatabaseConfig{DSN: " postgres://fallback "}}
	if got := cfg.ResolvedSecurityCipherSeed(); got != "postgres://fallback" {
		t.Fatalf("ResolvedSecurityCipherSeed(fallback) = %q", got)
	}
}

func TestConfigFileHelpers(t *testing.T) {
	v := viper.New()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	writeFile(t, configPath, []byte("server:\n  mode: serve\n"))

	if used, err := readConfigFile(v, configPath); err != nil || used != configPath {
		t.Fatalf("readConfigFile(explicit) = %q, %v", used, err)
	}

	empty := t.TempDir()
	t.Setenv("HOME", empty)
	other := viper.New()
	used, err := readConfigFile(other, "")
	if err != nil || used != "" {
		t.Fatalf("readConfigFile(no file) = %q, %v", used, err)
	}

	overrideV := viper.New()
	applyOverrides(overrideV, map[string]any{"server.port": 42000})
	if got := overrideV.GetInt("server.port"); got != 42000 {
		t.Fatalf("applyOverrides() server.port = %d, want 42000", got)
	}
}

func TestConfigHelperParsersCoverAdditionalBranches(t *testing.T) {
	if got, err := parsePort(float64(41002)); err != nil || got != 41002 {
		t.Fatalf("parsePort(float64) = %d, %v", got, err)
	}
	if _, err := parsePort("oops"); err == nil || !strings.Contains(err.Error(), "invalid port") {
		t.Fatalf("parsePort(invalid string) error = %v", err)
	}
	if _, err := parsePort(true); err == nil || !strings.Contains(err.Error(), "unsupported port type") {
		t.Fatalf("parsePort(type) error = %v", err)
	}

	if got, err := parseBool(false); err != nil || got {
		t.Fatalf("parseBool(bool) = %v, %v", got, err)
	}
	if _, err := parseBool("not-bool"); err == nil || !strings.Contains(err.Error(), "invalid bool") {
		t.Fatalf("parseBool(invalid string) error = %v", err)
	}

	if got, err := parseDuration(4 * time.Second); err != nil || got != 4*time.Second {
		t.Fatalf("parseDuration(duration) = %s, %v", got, err)
	}
	if _, err := parseDuration("bad"); err == nil || !strings.Contains(err.Error(), "invalid duration") {
		t.Fatalf("parseDuration(invalid string) error = %v", err)
	}
	if _, err := parseDuration(5); err == nil || !strings.Contains(err.Error(), "unsupported duration type") {
		t.Fatalf("parseDuration(type) error = %v", err)
	}
	if got, err := parseNonNegativeDuration(4 * time.Second); err != nil || got != 4*time.Second {
		t.Fatalf("parseNonNegativeDuration(duration) = %s, %v", got, err)
	}
	if _, err := parseNonNegativeDuration("bad"); err == nil || !strings.Contains(err.Error(), "invalid duration") {
		t.Fatalf("parseNonNegativeDuration(invalid string) error = %v", err)
	}
	if _, err := parseNonNegativeDuration(5); err == nil || !strings.Contains(err.Error(), "unsupported duration type") {
		t.Fatalf("parseNonNegativeDuration(type) error = %v", err)
	}

	if got, err := parseUnitInterval(int64(1)); err != nil || got != 1 {
		t.Fatalf("parseUnitInterval(int64) = %v, %v", got, err)
	}
	if _, err := parseUnitInterval("oops"); err == nil || !strings.Contains(err.Error(), "invalid float") {
		t.Fatalf("parseUnitInterval(invalid string) error = %v", err)
	}
	if _, err := parseUnitInterval(true); err == nil || !strings.Contains(err.Error(), "unsupported float type") {
		t.Fatalf("parseUnitInterval(type) error = %v", err)
	}

	if got, err := parseServerMode(" orchestrate "); err != nil || got != ServerModeOrchestrate {
		t.Fatalf("parseServerMode(string) = %q, %v", got, err)
	}
	if _, err := parseServerMode(true); err == nil || !strings.Contains(err.Error(), "unsupported server mode type") {
		t.Fatalf("parseServerMode(type) error = %v", err)
	}

	if got, err := parseEventDriver(" channel "); err != nil || got != EventDriverChannel {
		t.Fatalf("parseEventDriver(string) = %q, %v", got, err)
	}
	if _, err := parseEventDriver(true); err == nil || !strings.Contains(err.Error(), "unsupported event driver type") {
		t.Fatalf("parseEventDriver(type) error = %v", err)
	}

	if got, err := parseLogLevel(slog.LevelError); err != nil || got != slog.LevelError {
		t.Fatalf("parseLogLevel(level) = %v, %v", got, err)
	}
	if _, err := parseLogLevel("wat"); err == nil || !strings.Contains(err.Error(), "invalid slog level") {
		t.Fatalf("parseLogLevel(invalid string) error = %v", err)
	}

	if got, err := parseLogFormat(" json "); err != nil || got != LogFormatJSON {
		t.Fatalf("parseLogFormat(string) = %q, %v", got, err)
	}
	if _, err := parseLogFormat(true); err == nil || !strings.Contains(err.Error(), "unsupported log format type") {
		t.Fatalf("parseLogFormat(type) error = %v", err)
	}
}

func TestConfigValidationAndFileHelpersCoverFailurePaths(t *testing.T) {
	cfg := Config{
		Server:   ServerConfig{Mode: ServerModeServe},
		Event:    EventConfig{Driver: EventDriverPGNotify},
		Database: DatabaseConfig{DSN: "postgres://example"},
	}
	driver, err := cfg.ResolvedEventDriver()
	if err != nil || driver != EventDriverPGNotify {
		t.Fatalf("ResolvedEventDriver(pgnotify) = %q, %v", driver, err)
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("validateConfig(pgnotify with dsn) error = %v", err)
	}
	if _, err := (Config{Event: EventConfig{Driver: "broken"}}).ResolvedEventDriver(); err == nil || !strings.Contains(err.Error(), "unsupported event driver") {
		t.Fatalf("ResolvedEventDriver(invalid driver) error = %v", err)
	}

	v := viper.New()
	if _, err := readConfigFile(v, filepath.Join(t.TempDir(), "missing.yaml")); err == nil || !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("readConfigFile(missing explicit) error = %v", err)
	}

	dir := t.TempDir()
	badConfigPath := filepath.Join(dir, "config.yaml")
	writeFile(t, badConfigPath, []byte("server:\n  mode: ["))
	other := viper.New()
	if _, err := readConfigFile(other, badConfigPath); err == nil || !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("readConfigFile(invalid yaml) error = %v", err)
	}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
