package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

type Config struct {
	Server        ServerConfig
	Auth          AuthConfig
	GitHub        GitHubConfig
	Database      DatabaseConfig
	Orchestrator  OrchestratorConfig
	Event         EventConfig
	Observability ObservabilityConfig
	Logging       LoggingConfig
	Metadata      Metadata
}

type Metadata struct {
	ConfigFile string
}

type ServerConfig struct {
	Mode            ServerMode
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type GitHubConfig struct {
	WebhookSecret string
}

type ServerMode string

const (
	ServerModeAllInOne    ServerMode = "all-in-one"
	ServerModeServe       ServerMode = "serve"
	ServerModeOrchestrate ServerMode = "orchestrate"
)

type DatabaseConfig struct {
	DSN string
}

type OrchestratorConfig struct {
	TickInterval time.Duration
}

type EventConfig struct {
	Driver EventDriver
}

type ObservabilityConfig struct {
	Metrics MetricsConfig
	Tracing TraceConfig
}

type MetricsConfig struct {
	Enabled bool
	Export  MetricsExportConfig
}

type MetricsExportConfig struct {
	Prometheus   bool
	OTLPEndpoint string
}

type TraceConfig struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	SampleRatio float64
}

type EventDriver string

const (
	EventDriverAuto     EventDriver = "auto"
	EventDriverChannel  EventDriver = "channel"
	EventDriverPGNotify EventDriver = "pgnotify"
)

type LoggingConfig struct {
	Level  slog.Level
	Format LogFormat
}

type LoadOptions struct {
	ConfigFile string
	Overrides  map[string]any
}

func Load(opts LoadOptions) (Config, error) {
	v := viper.New()
	configureDefaults(v)
	configureEnvironment(v)
	applyOverrides(v, opts.Overrides)

	configFile, err := readConfigFile(v, opts.ConfigFile)
	if err != nil {
		return Config{}, err
	}

	cfg, err := parseConfig(v)
	if err != nil {
		return Config{}, err
	}
	cfg.Metadata.ConfigFile = configFile

	return cfg, nil
}

func configureDefaults(v *viper.Viper) {
	v.SetDefault("server.mode", string(ServerModeAllInOne))
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 40023)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.shutdown_timeout", 10*time.Second)
	configureAuthDefaults(v)
	v.SetDefault("github.webhook_secret", "")
	v.SetDefault("database.dsn", "")
	v.SetDefault("orchestrator.tick_interval", 5*time.Second)
	v.SetDefault("event.driver", string(EventDriverAuto))
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.export.prometheus", false)
	v.SetDefault("observability.metrics.export.otlp_endpoint", "")
	v.SetDefault("observability.tracing.enabled", false)
	v.SetDefault("observability.tracing.endpoint", "")
	v.SetDefault("observability.tracing.service_name", "openase")
	v.SetDefault("observability.tracing.sample_ratio", 1.0)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", string(LogFormatText))
}

func configureEnvironment(v *viper.Viper) {
	v.SetEnvPrefix("OPENASE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
}

func applyOverrides(v *viper.Viper, overrides map[string]any) {
	for key, value := range overrides {
		v.Set(key, value)
	}
}

func readConfigFile(v *viper.Viper, explicitPath string) (string, error) {
	if explicitPath != "" {
		v.SetConfigFile(explicitPath)
		if err := v.ReadInConfig(); err != nil {
			return "", fmt.Errorf("read config file: %w", err)
		}

		return v.ConfigFileUsed(), nil
	}

	v.SetConfigName("config")
	v.AddConfigPath(".")

	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".openase"))
	}

	if err := v.ReadInConfig(); err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if errors.As(err, &configNotFound) {
			return "", nil
		}

		return "", fmt.Errorf("read config file: %w", err)
	}

	return v.ConfigFileUsed(), nil
}

func parseConfig(v *viper.Viper) (Config, error) {
	serverMode, err := parseServerMode(v.Get("server.mode"))
	if err != nil {
		return Config{}, fmt.Errorf("parse server.mode: %w", err)
	}

	serverHost, err := parseNonEmptyString(v.Get("server.host"))
	if err != nil {
		return Config{}, fmt.Errorf("parse server.host: %w", err)
	}

	serverPort, err := parsePort(v.Get("server.port"))
	if err != nil {
		return Config{}, fmt.Errorf("parse server.port: %w", err)
	}

	readTimeout, err := parseDuration(v.Get("server.read_timeout"))
	if err != nil {
		return Config{}, fmt.Errorf("parse server.read_timeout: %w", err)
	}

	writeTimeout, err := parseDuration(v.Get("server.write_timeout"))
	if err != nil {
		return Config{}, fmt.Errorf("parse server.write_timeout: %w", err)
	}

	shutdownTimeout, err := parseDuration(v.Get("server.shutdown_timeout"))
	if err != nil {
		return Config{}, fmt.Errorf("parse server.shutdown_timeout: %w", err)
	}

	authConfig, err := parseAuthConfig(v)
	if err != nil {
		authConfig = AuthConfig{}
	}

	gitHubWebhookSecret, err := parseOptionalString(v.Get("github.webhook_secret"))
	if err != nil {
		return Config{}, fmt.Errorf("parse github.webhook_secret: %w", err)
	}

	databaseDSN, err := parseOptionalString(v.Get("database.dsn"))
	if err != nil {
		return Config{}, fmt.Errorf("parse database.dsn: %w", err)
	}

	tickInterval, err := parseDuration(v.Get("orchestrator.tick_interval"))
	if err != nil {
		return Config{}, fmt.Errorf("parse orchestrator.tick_interval: %w", err)
	}

	eventDriver, err := parseEventDriver(v.Get("event.driver"))
	if err != nil {
		return Config{}, fmt.Errorf("parse event.driver: %w", err)
	}

	metricsEnabled, err := parseBool(v.Get("observability.metrics.enabled"))
	if err != nil {
		return Config{}, fmt.Errorf("parse observability.metrics.enabled: %w", err)
	}

	prometheusEnabled, err := parseBool(v.Get("observability.metrics.export.prometheus"))
	if err != nil {
		return Config{}, fmt.Errorf("parse observability.metrics.export.prometheus: %w", err)
	}

	otlpEndpoint, err := parseOptionalString(v.Get("observability.metrics.export.otlp_endpoint"))
	if err != nil {
		return Config{}, fmt.Errorf("parse observability.metrics.export.otlp_endpoint: %w", err)
	}

	traceEnabled, err := parseBool(v.Get("observability.tracing.enabled"))
	if err != nil {
		return Config{}, fmt.Errorf("parse observability.tracing.enabled: %w", err)
	}

	traceEndpoint, err := parseOptionalString(v.Get("observability.tracing.endpoint"))
	if err != nil {
		return Config{}, fmt.Errorf("parse observability.tracing.endpoint: %w", err)
	}

	traceServiceName, err := parseNonEmptyString(v.Get("observability.tracing.service_name"))
	if err != nil {
		return Config{}, fmt.Errorf("parse observability.tracing.service_name: %w", err)
	}

	traceSampleRatio, err := parseUnitInterval(v.Get("observability.tracing.sample_ratio"))
	if err != nil {
		return Config{}, fmt.Errorf("parse observability.tracing.sample_ratio: %w", err)
	}

	logLevel, err := parseLogLevel(v.Get("log.level"))
	if err != nil {
		return Config{}, fmt.Errorf("parse log.level: %w", err)
	}

	logFormat, err := parseLogFormat(v.Get("log.format"))
	if err != nil {
		return Config{}, fmt.Errorf("parse log.format: %w", err)
	}

	cfg := Config{
		Server: ServerConfig{
			Mode:            serverMode,
			Host:            serverHost,
			Port:            serverPort,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		Auth: authConfig,
		GitHub: GitHubConfig{
			WebhookSecret: gitHubWebhookSecret,
		},
		Database: DatabaseConfig{
			DSN: databaseDSN,
		},
		Orchestrator: OrchestratorConfig{
			TickInterval: tickInterval,
		},
		Event: EventConfig{
			Driver: eventDriver,
		},
		Observability: ObservabilityConfig{
			Metrics: MetricsConfig{
				Enabled: metricsEnabled,
				Export: MetricsExportConfig{
					Prometheus:   prometheusEnabled,
					OTLPEndpoint: otlpEndpoint,
				},
			},
			Tracing: TraceConfig{
				Enabled:     traceEnabled,
				Endpoint:    traceEndpoint,
				ServiceName: traceServiceName,
				SampleRatio: traceSampleRatio,
			},
		},
		Logging: LoggingConfig{
			Level:  logLevel,
			Format: logFormat,
		},
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func parseNonEmptyString(raw any) (string, error) {
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("unsupported string type %T", raw)
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("value must not be empty")
	}

	return trimmed, nil
}

func parseOptionalString(raw any) (string, error) {
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("unsupported string type %T", raw)
	}

	return strings.TrimSpace(value), nil
}

func parsePort(raw any) (int, error) {
	switch value := raw.(type) {
	case int:
		if value < 1 || value > 65535 {
			return 0, fmt.Errorf("port %d out of range", value)
		}
		return value, nil
	case int64:
		return parsePort(int(value))
	case float64:
		return parsePort(int(value))
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, fmt.Errorf("invalid port %q", value)
		}
		return parsePort(parsed)
	default:
		return 0, fmt.Errorf("unsupported port type %T", raw)
	}
}

func parseBool(raw any) (bool, error) {
	switch value := raw.(type) {
	case bool:
		return value, nil
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return false, fmt.Errorf("invalid bool %q", value)
		}
		return parsed, nil
	default:
		return false, fmt.Errorf("unsupported bool type %T", raw)
	}
}

func parseDuration(raw any) (time.Duration, error) {
	switch value := raw.(type) {
	case time.Duration:
		if value <= 0 {
			return 0, fmt.Errorf("duration %s must be positive", value)
		}
		return value, nil
	case string:
		parsed, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", value)
		}
		return parseDuration(parsed)
	default:
		return 0, fmt.Errorf("unsupported duration type %T", raw)
	}
}

func parseUnitInterval(raw any) (float64, error) {
	switch value := raw.(type) {
	case float64:
		if value < 0 || value > 1 {
			return 0, fmt.Errorf("value %v must be between 0 and 1", value)
		}
		return value, nil
	case int:
		return parseUnitInterval(float64(value))
	case int64:
		return parseUnitInterval(float64(value))
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid float %q", value)
		}
		return parseUnitInterval(parsed)
	default:
		return 0, fmt.Errorf("unsupported float type %T", raw)
	}
}

func parseServerMode(raw any) (ServerMode, error) {
	switch value := raw.(type) {
	case ServerMode:
		if value == ServerModeAllInOne || value == ServerModeServe || value == ServerModeOrchestrate {
			return value, nil
		}
	case string:
		mode := ServerMode(strings.ToLower(strings.TrimSpace(value)))
		if mode == ServerModeAllInOne || mode == ServerModeServe || mode == ServerModeOrchestrate {
			return mode, nil
		}
		return "", fmt.Errorf("unsupported server mode %q", value)
	default:
		return "", fmt.Errorf("unsupported server mode type %T", raw)
	}

	return "", fmt.Errorf("unsupported server mode %q", raw)
}

func parseEventDriver(raw any) (EventDriver, error) {
	switch value := raw.(type) {
	case EventDriver:
		if value == EventDriverAuto || value == EventDriverChannel || value == EventDriverPGNotify {
			return value, nil
		}
	case string:
		driver := EventDriver(strings.ToLower(strings.TrimSpace(value)))
		if driver == EventDriverAuto || driver == EventDriverChannel || driver == EventDriverPGNotify {
			return driver, nil
		}
		return "", fmt.Errorf("unsupported event driver %q", value)
	default:
		return "", fmt.Errorf("unsupported event driver type %T", raw)
	}

	return "", fmt.Errorf("unsupported event driver %q", raw)
}

func parseLogLevel(raw any) (slog.Level, error) {
	switch value := raw.(type) {
	case slog.Level:
		return value, nil
	case string:
		var level slog.Level
		if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(value)))); err != nil {
			return 0, fmt.Errorf("invalid slog level %q", value)
		}
		return level, nil
	default:
		return 0, fmt.Errorf("unsupported slog level type %T", raw)
	}
}

func parseLogFormat(raw any) (LogFormat, error) {
	switch value := raw.(type) {
	case LogFormat:
		if value == LogFormatText || value == LogFormatJSON {
			return value, nil
		}
	case string:
		format := LogFormat(strings.ToLower(strings.TrimSpace(value)))
		if format == LogFormatText || format == LogFormatJSON {
			return format, nil
		}
		return "", fmt.Errorf("unsupported log format %q", value)
	}

	return "", fmt.Errorf("unsupported log format type %T", raw)
}

func validateConfig(cfg Config) error {
	if cfg.Event.Driver == EventDriverChannel && cfg.Server.Mode != ServerModeAllInOne {
		return errors.New("event.driver=channel requires server.mode=all-in-one")
	}

	driver, err := cfg.ResolvedEventDriver()
	if err != nil {
		return err
	}
	if driver == EventDriverPGNotify && cfg.Database.DSN == "" {
		return errors.New("database.dsn is required when event.driver resolves to pgnotify")
	}
	return nil
}

func (cfg Config) ResolvedEventDriver() (EventDriver, error) {
	switch cfg.Event.Driver {
	case EventDriverChannel, EventDriverPGNotify:
		return cfg.Event.Driver, nil
	case EventDriverAuto:
		if cfg.Server.Mode == ServerModeAllInOne {
			return EventDriverChannel, nil
		}
		if cfg.Server.Mode == ServerModeServe || cfg.Server.Mode == ServerModeOrchestrate {
			return EventDriverPGNotify, nil
		}
		return "", fmt.Errorf("cannot resolve event driver for server mode %q", cfg.Server.Mode)
	default:
		return "", fmt.Errorf("unsupported event driver %q", cfg.Event.Driver)
	}
}
