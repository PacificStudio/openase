package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
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
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_PORT", "41000")
	t.Setenv("OPENASE_SERVER_MODE", "serve")
	t.Setenv("OPENASE_GITHUB_WEBHOOK_SECRET", "topsecret")
	t.Setenv("OPENASE_DATABASE_DSN", "postgres://openase:secret@localhost:5432/openase?sslmode=disable")
	t.Setenv("OPENASE_ORCHESTRATOR_TICK_INTERVAL", "2s")
	t.Setenv("OPENASE_EVENT_DRIVER", "pgnotify")
	t.Setenv("OPENASE_OBSERVABILITY_METRICS_ENABLED", "false")
	t.Setenv("OPENASE_OBSERVABILITY_METRICS_EXPORT_PROMETHEUS", "true")
	t.Setenv("OPENASE_OBSERVABILITY_METRICS_EXPORT_OTLP_ENDPOINT", "collector.internal:4318")
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

	if cfg.GitHub.WebhookSecret != "topsecret" {
		t.Fatalf("expected GitHub webhook secret from env, got %q", cfg.GitHub.WebhookSecret)
	}

	if cfg.Database.DSN == "" {
		t.Fatal("expected database dsn from env")
	}

	if cfg.Orchestrator.TickInterval != 2*time.Second {
		t.Fatalf("expected env tick interval, got %s", cfg.Orchestrator.TickInterval)
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

	if cfg.Logging.Format != LogFormatJSON {
		t.Fatalf("expected json log format, got %q", cfg.Logging.Format)
	}

	if cfg.Logging.Level != slog.LevelDebug {
		t.Fatalf("expected debug log level, got %s", cfg.Logging.Level)
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "openase.yaml")
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
database:
  dsn: postgres://openase:secret@localhost:5432/openase?sslmode=disable
orchestrator:
  tick_interval: 3s
event:
  driver: pgnotify
observability:
  metrics:
    enabled: true
    export:
      prometheus: true
      otlp_endpoint: https://collector.example.test/v1/metrics
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

	if cfg.Logging.Level != slog.LevelWarn {
		t.Fatalf("expected warn log level, got %s", cfg.Logging.Level)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_PORT", "70000")

	if _, err := Load(LoadOptions{}); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestLoadRejectsChannelDriverOutsideAllInOne(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_MODE", "serve")
	t.Setenv("OPENASE_EVENT_DRIVER", "channel")

	if _, err := Load(LoadOptions{}); err == nil {
		t.Fatal("expected invalid channel driver error")
	}
}

func TestLoadRejectsMissingDatabaseDSNForResolvedPGNotify(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_MODE", "serve")

	if _, err := Load(LoadOptions{}); err == nil {
		t.Fatal("expected missing database dsn error")
	}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
