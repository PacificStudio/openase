package otel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const defaultInstrumentationScope = "github.com/BetterAndBetterII/openase"

var otelMetricsComponent = logging.DeclareComponent("otel-metrics")

type MetricsConfig struct {
	ServiceName  string
	Prometheus   bool
	OTLPEndpoint string
}

type MetricsProvider struct {
	logger *slog.Logger
	meter  otelmetric.Meter
	sdk    *metric.MeterProvider

	prometheusHandler http.Handler

	counters   sync.Map
	histograms sync.Map
	gauges     sync.Map
}

func NewMetricsProvider(ctx context.Context, cfg MetricsConfig, logger *slog.Logger) (*MetricsProvider, error) {
	if logger == nil {
		logger = slog.Default()
	}

	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "openase"
	}

	options := []metric.Option{
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	}

	var prometheusHandler http.Handler
	if cfg.Prometheus {
		registry := prometheus.NewRegistry()
		exporter, err := prometheusexporter.New(
			prometheusexporter.WithRegisterer(registry),
			prometheusexporter.WithoutTargetInfo(),
			prometheusexporter.WithoutScopeInfo(),
		)
		if err != nil {
			return nil, fmt.Errorf("build prometheus exporter: %w", err)
		}
		options = append(options, metric.WithReader(exporter))
		prometheusHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		})
	}

	if endpoint := strings.TrimSpace(cfg.OTLPEndpoint); endpoint != "" {
		exporterOptions := []otlpmetrichttp.Option{}
		if strings.Contains(endpoint, "://") {
			exporterOptions = append(exporterOptions, otlpmetrichttp.WithEndpointURL(endpoint))
		} else {
			exporterOptions = append(exporterOptions, otlpmetrichttp.WithEndpoint(endpoint), otlpmetrichttp.WithInsecure())
		}

		exporter, err := otlpmetrichttp.New(ctx, exporterOptions...)
		if err != nil {
			return nil, fmt.Errorf("build otlp metrics exporter: %w", err)
		}
		options = append(options, metric.WithReader(metric.NewPeriodicReader(exporter)))
	}

	sdkProvider := metric.NewMeterProvider(options...)
	return &MetricsProvider{
		logger:            logging.WithComponent(logger, otelMetricsComponent),
		meter:             sdkProvider.Meter(defaultInstrumentationScope),
		sdk:               sdkProvider,
		prometheusHandler: prometheusHandler,
	}, nil
}

func (p *MetricsProvider) Counter(name string, tags provider.Tags) provider.Counter {
	spec, key, err := parseInstrumentSpec(name, tags)
	if err != nil {
		p.logger.Error("build counter instrument", "error", err)
		return provider.NewNoopMetricsProvider().Counter("", nil)
	}

	if existing, ok := p.counters.Load(key); ok {
		return existing.(provider.Counter)
	}

	instrument, err := p.meter.Float64Counter(spec.name)
	if err != nil {
		p.logger.Error("create counter instrument", "name", spec.name, "error", err)
		return provider.NewNoopMetricsProvider().Counter("", nil)
	}

	counter := otelCounter{instrument: instrument, attributes: spec.attributes}
	actual, _ := p.counters.LoadOrStore(key, counter)
	return actual.(provider.Counter)
}

func (p *MetricsProvider) Histogram(name string, tags provider.Tags) provider.Histogram {
	spec, key, err := parseInstrumentSpec(name, tags)
	if err != nil {
		p.logger.Error("build histogram instrument", "error", err)
		return provider.NewNoopMetricsProvider().Histogram("", nil)
	}

	if existing, ok := p.histograms.Load(key); ok {
		return existing.(provider.Histogram)
	}

	instrument, err := p.meter.Float64Histogram(spec.name)
	if err != nil {
		p.logger.Error("create histogram instrument", "name", spec.name, "error", err)
		return provider.NewNoopMetricsProvider().Histogram("", nil)
	}

	histogram := otelHistogram{instrument: instrument, attributes: spec.attributes}
	actual, _ := p.histograms.LoadOrStore(key, histogram)
	return actual.(provider.Histogram)
}

func (p *MetricsProvider) Gauge(name string, tags provider.Tags) provider.Gauge {
	spec, key, err := parseInstrumentSpec(name, tags)
	if err != nil {
		p.logger.Error("build gauge instrument", "error", err)
		return provider.NewNoopMetricsProvider().Gauge("", nil)
	}

	if existing, ok := p.gauges.Load(key); ok {
		return existing.(provider.Gauge)
	}

	instrument, err := p.meter.Float64Gauge(spec.name)
	if err != nil {
		p.logger.Error("create gauge instrument", "name", spec.name, "error", err)
		return provider.NewNoopMetricsProvider().Gauge("", nil)
	}

	gauge := otelGauge{instrument: instrument, attributes: spec.attributes}
	actual, _ := p.gauges.LoadOrStore(key, gauge)
	return actual.(provider.Gauge)
}

func (p *MetricsProvider) PrometheusHandler() http.Handler {
	return p.prometheusHandler
}

func (p *MetricsProvider) Shutdown(ctx context.Context) error {
	return p.sdk.Shutdown(ctx)
}

type otelCounter struct {
	instrument otelmetric.Float64Counter
	attributes []attribute.KeyValue
}

func (c otelCounter) Add(value float64) {
	c.instrument.Add(context.Background(), value, otelmetric.WithAttributes(c.attributes...))
}

type otelHistogram struct {
	instrument otelmetric.Float64Histogram
	attributes []attribute.KeyValue
}

func (h otelHistogram) Record(value float64) {
	h.instrument.Record(context.Background(), value, otelmetric.WithAttributes(h.attributes...))
}

type otelGauge struct {
	instrument otelmetric.Float64Gauge
	attributes []attribute.KeyValue
}

func (g otelGauge) Set(value float64) {
	g.instrument.Record(context.Background(), value, otelmetric.WithAttributes(g.attributes...))
}

type instrumentSpec struct {
	name       string
	attributes []attribute.KeyValue
}

func parseInstrumentSpec(name string, tags provider.Tags) (instrumentSpec, string, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return instrumentSpec{}, "", fmt.Errorf("metric name must not be empty")
	}

	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	attributes := make([]attribute.KeyValue, 0, len(keys))
	parts := []string{trimmedName}
	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return instrumentSpec{}, "", fmt.Errorf("metric tag key must not be empty")
		}
		value := strings.TrimSpace(tags[key])
		attributes = append(attributes, attribute.String(trimmedKey, value))
		parts = append(parts, trimmedKey+"="+value)
	}

	return instrumentSpec{
		name:       trimmedName,
		attributes: attributes,
	}, strings.Join(parts, "|"), nil
}
