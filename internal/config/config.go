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
	Server       ServerConfig
	Orchestrator OrchestratorConfig
	Logging      LoggingConfig
	Metadata     Metadata
}

type Metadata struct {
	ConfigFile string
}

type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type OrchestratorConfig struct {
	TickInterval time.Duration
}

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
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 40023)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.shutdown_timeout", 10*time.Second)
	v.SetDefault("orchestrator.tick_interval", 5*time.Second)
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

	v.SetConfigName("openase")
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

	tickInterval, err := parseDuration(v.Get("orchestrator.tick_interval"))
	if err != nil {
		return Config{}, fmt.Errorf("parse orchestrator.tick_interval: %w", err)
	}

	logLevel, err := parseLogLevel(v.Get("log.level"))
	if err != nil {
		return Config{}, fmt.Errorf("parse log.level: %w", err)
	}

	logFormat, err := parseLogFormat(v.Get("log.format"))
	if err != nil {
		return Config{}, fmt.Errorf("parse log.format: %w", err)
	}

	return Config{
		Server: ServerConfig{
			Host:            serverHost,
			Port:            serverPort,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		Orchestrator: OrchestratorConfig{
			TickInterval: tickInterval,
		},
		Logging: LoggingConfig{
			Level:  logLevel,
			Format: logFormat,
		},
	}, nil
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
