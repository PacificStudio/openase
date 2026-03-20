package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	otelinfra "github.com/BetterAndBetterII/openase/internal/infra/otel"
	userserviceinfra "github.com/BetterAndBetterII/openase/internal/infra/userservice"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func buildEventProvider(cfg config.Config, logger *slog.Logger) (provider.EventProvider, error) {
	driver, err := cfg.ResolvedEventDriver()
	if err != nil {
		return nil, err
	}

	switch driver {
	case config.EventDriverChannel:
		logger.Info("configured event provider", "driver", driver, "mode", cfg.Server.Mode)
		return eventinfra.NewChannelBus(), nil
	case config.EventDriverPGNotify:
		bus, err := eventinfra.NewPGNotifyBus(cfg.Database.DSN, logger)
		if err != nil {
			return nil, err
		}

		logger.Info("configured event provider", "driver", driver, "mode", cfg.Server.Mode)
		return bus, nil
	default:
		return nil, fmt.Errorf("unsupported event driver %q", driver)
	}
}

func buildUserServiceManager() (provider.UserServiceManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "linux":
		return userserviceinfra.NewSystemdUserManager(homeDir), nil
	case "darwin":
		return userserviceinfra.NewLaunchdUserManager(homeDir, os.Getuid()), nil
	default:
		return nil, fmt.Errorf("unsupported OS %q for managed user services", runtime.GOOS)
	}
}

type metricsRuntime struct {
	provider          provider.MetricsProvider
	prometheusHandler http.Handler
	shutdown          func(context.Context) error
}

func buildMetricsProvider(cfg config.Config, logger *slog.Logger) (metricsRuntime, error) {
	if !cfg.Observability.Metrics.Enabled {
		return metricsRuntime{
			provider: provider.NewNoopMetricsProvider(),
			shutdown: func(context.Context) error { return nil },
		}, nil
	}

	metricsProvider, err := otelinfra.NewMetricsProvider(context.Background(), otelinfra.MetricsConfig{
		ServiceName:  "openase",
		Prometheus:   cfg.Observability.Metrics.Export.Prometheus,
		OTLPEndpoint: cfg.Observability.Metrics.Export.OTLPEndpoint,
	}, logger)
	if err != nil {
		return metricsRuntime{}, err
	}

	logger.Info(
		"configured metrics provider",
		"enabled", true,
		"prometheus_export", cfg.Observability.Metrics.Export.Prometheus,
		"otlp_endpoint", cfg.Observability.Metrics.Export.OTLPEndpoint,
	)

	return metricsRuntime{
		provider:          metricsProvider,
		prometheusHandler: metricsProvider.PrometheusHandler(),
		shutdown: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return metricsProvider.Shutdown(shutdownCtx)
		},
	}, nil
}

func buildTraceProvider(cfg config.Config, logger *slog.Logger) (provider.TraceProvider, error) {
	if !cfg.Observability.Tracing.Enabled {
		logger.Info("configured trace provider", "exporter", "noop", "service_name", cfg.Observability.Tracing.ServiceName)
		return provider.NewNoopTraceProvider(), nil
	}

	return otelinfra.NewTraceProvider(otelinfra.TraceConfig{
		ServiceName: cfg.Observability.Tracing.ServiceName,
		Endpoint:    cfg.Observability.Tracing.Endpoint,
		SampleRatio: cfg.Observability.Tracing.SampleRatio,
	}, logger)
}
