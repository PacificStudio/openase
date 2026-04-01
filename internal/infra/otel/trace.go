package otel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/BetterAndBetterII/openase"

var otelTraceComponent = logging.DeclareComponent("otel-trace")

type TraceConfig struct {
	ServiceName string
	Endpoint    string
	SampleRatio float64
}

func NewTraceProvider(cfg TraceConfig, logger *slog.Logger) (provider.TraceProvider, error) {
	logger = logging.WithComponent(logger, otelTraceComponent)
	resourceAttributes := resource.WithAttributes(
		semconv.ServiceName(cfg.ServiceName),
	)
	res, err := resource.New(context.Background(), resourceAttributes)
	if err != nil {
		return nil, fmt.Errorf("build trace resource: %w", err)
	}

	exporter, exporterName, err := newExporter(cfg)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.TraceIDRatioBased(cfg.SampleRatio)
	sdkProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
		sdktrace.WithBatcher(exporter),
	)
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTracerProvider(sdkProvider)
	otel.SetTextMapPropagator(propagator)
	logger.Info(
		"configured trace provider",
		"exporter", exporterName,
		"service_name", cfg.ServiceName,
		"sample_ratio", cfg.SampleRatio,
	)

	return &traceProvider{
		provider:   sdkProvider,
		tracer:     sdkProvider.Tracer(instrumentationName),
		propagator: propagator,
	}, nil
}

type traceProvider struct {
	provider   oteltrace.TracerProvider
	tracer     oteltrace.Tracer
	propagator propagation.TextMapPropagator
}

func (p *traceProvider) ExtractHTTPContext(ctx context.Context, header http.Header) context.Context {
	return p.propagator.Extract(ctx, propagation.HeaderCarrier(header))
}

func (p *traceProvider) InjectHTTPHeaders(ctx context.Context, header http.Header) {
	p.propagator.Inject(ctx, propagation.HeaderCarrier(header))
}

func (p *traceProvider) StartSpan(ctx context.Context, name string, opts ...provider.SpanStartOption) (context.Context, provider.Span) {
	kind, attrs := provider.ResolveSpanStartOptions(opts...)
	traceOptions := make([]oteltrace.SpanStartOption, 0, 2)
	traceOptions = append(traceOptions, oteltrace.WithSpanKind(mapSpanKind(kind)))
	if len(attrs) > 0 {
		traceOptions = append(traceOptions, oteltrace.WithAttributes(mapAttributes(attrs)...))
	}
	nextCtx, span := p.tracer.Start(ctx, name, traceOptions...)

	return nextCtx, tracedSpan{span: span}
}

func (p *traceProvider) Shutdown(ctx context.Context) error {
	shutdowner, ok := p.provider.(interface {
		Shutdown(context.Context) error
	})
	if !ok {
		return nil
	}

	return shutdowner.Shutdown(ctx)
}

type tracedSpan struct {
	span oteltrace.Span
}

func (s tracedSpan) End() {
	s.span.End()
}

func (s tracedSpan) RecordError(err error) {
	if err == nil {
		return
	}

	s.span.RecordError(err)
}

func (s tracedSpan) SetAttributes(attrs ...provider.SpanAttribute) {
	s.span.SetAttributes(mapAttributes(attrs)...)
}

func (s tracedSpan) SetStatus(code provider.SpanStatusCode, description string) {
	s.span.SetStatus(mapSpanStatus(code), description)
}

func (s tracedSpan) TraceID() string {
	spanContext := s.span.SpanContext()
	if !spanContext.TraceID().IsValid() {
		return ""
	}

	return spanContext.TraceID().String()
}

func (s tracedSpan) SpanID() string {
	spanContext := s.span.SpanContext()
	if !spanContext.SpanID().IsValid() {
		return ""
	}

	return spanContext.SpanID().String()
}

func newExporter(cfg TraceConfig) (sdktrace.SpanExporter, string, error) {
	if cfg.Endpoint == "" {
		exporter, err := stdouttrace.New()
		if err != nil {
			return nil, "", fmt.Errorf("build stdout trace exporter: %w", err)
		}

		return exporter, "stdout", nil
	}

	parsed, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, "", fmt.Errorf("parse tracing endpoint %q: %w", cfg.Endpoint, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, "", fmt.Errorf("tracing endpoint %q must be an absolute URL", cfg.Endpoint)
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Endpoint),
	}
	if parsed.Scheme == "http" {
		options = append(options, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), options...)
	if err != nil {
		return nil, "", fmt.Errorf("build otlp http trace exporter: %w", err)
	}

	return exporter, "otlphttp", nil
}

func mapSpanKind(kind provider.SpanKind) oteltrace.SpanKind {
	switch kind {
	case provider.SpanKindServer:
		return oteltrace.SpanKindServer
	case provider.SpanKindClient:
		return oteltrace.SpanKindClient
	case provider.SpanKindProducer:
		return oteltrace.SpanKindProducer
	case provider.SpanKindConsumer:
		return oteltrace.SpanKindConsumer
	default:
		return oteltrace.SpanKindInternal
	}
}

func mapSpanStatus(code provider.SpanStatusCode) codes.Code {
	switch code {
	case provider.SpanStatusOK:
		return codes.Ok
	case provider.SpanStatusError:
		return codes.Error
	default:
		return codes.Unset
	}
}

func mapAttributes(attrs []provider.SpanAttribute) []attribute.KeyValue {
	values := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		switch attr.Type {
		case provider.SpanAttributeTypeString:
			values = append(values, attribute.String(attr.Key, attr.StringValue))
		case provider.SpanAttributeTypeInt64:
			values = append(values, attribute.Int64(attr.Key, attr.Int64Value))
		case provider.SpanAttributeTypeBool:
			values = append(values, attribute.Bool(attr.Key, attr.BoolValue))
		}
	}

	return values
}
