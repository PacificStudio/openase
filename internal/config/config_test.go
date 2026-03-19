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
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENASE_SERVER_PORT", "41000")
	t.Setenv("OPENASE_ORCHESTRATOR_TICK_INTERVAL", "2s")
	t.Setenv("OPENASE_LOG_FORMAT", "json")
	t.Setenv("OPENASE_LOG_LEVEL", "debug")

	cfg, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != 41000 {
		t.Fatalf("expected env port, got %d", cfg.Server.Port)
	}

	if cfg.Orchestrator.TickInterval != 2*time.Second {
		t.Fatalf("expected env tick interval, got %s", cfg.Orchestrator.TickInterval)
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
  host: 127.0.0.1
  port: 40123
  read_timeout: 20s
  write_timeout: 25s
  shutdown_timeout: 12s
orchestrator:
  tick_interval: 3s
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

	if cfg.Server.ReadTimeout != 20*time.Second {
		t.Fatalf("expected read timeout 20s, got %s", cfg.Server.ReadTimeout)
	}

	if cfg.Orchestrator.TickInterval != 3*time.Second {
		t.Fatalf("expected tick interval 3s, got %s", cfg.Orchestrator.TickInterval)
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

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
