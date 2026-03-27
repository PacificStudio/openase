package otel

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/provider"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewTraceProviderSupportsSpanLifecycleAndShutdown(t *testing.T) {
	traceProvider, err := NewTraceProvider(TraceConfig{
		ServiceName: "openase-test",
		SampleRatio: 1,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewTraceProvider() error = %v", err)
	}

	ctx, span := traceProvider.StartSpan(context.Background(), "coverage-span")
	if span.TraceID() == "" || span.SpanID() == "" {
		t.Fatalf("span ids = (%q, %q)", span.TraceID(), span.SpanID())
	}
	span.RecordError(errors.New("boom"))
	span.SetAttributes(
		provider.StringAttribute("key", "value"),
		provider.IntAttribute("count", 2),
		provider.BoolAttribute("ok", true),
	)
	span.SetStatus(provider.SpanStatusError, "boom")
	span.End()

	if err := traceProvider.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestNewExporterAndSpanMappings(t *testing.T) {
	if _, _, err := newExporter(TraceConfig{Endpoint: "://bad"}); err == nil {
		t.Fatal("newExporter(parse failure) expected error")
	}
	if _, _, err := newExporter(TraceConfig{Endpoint: "/relative"}); err == nil {
		t.Fatal("newExporter(relative endpoint) expected error")
	}

	if got := mapSpanKind(provider.SpanKind("unknown")); got != oteltrace.SpanKindInternal {
		t.Fatalf("mapSpanKind(default) = %v", got)
	}
	if got := mapSpanKind(provider.SpanKindProducer); got != oteltrace.SpanKindProducer {
		t.Fatalf("mapSpanKind(producer) = %v", got)
	}
	if got := mapSpanKind(provider.SpanKindConsumer); got != oteltrace.SpanKindConsumer {
		t.Fatalf("mapSpanKind(consumer) = %v", got)
	}
	if got := mapSpanStatus(provider.SpanStatusCode("unknown")); got != codes.Unset {
		t.Fatalf("mapSpanStatus(default) = %v", got)
	}
	if got := mapSpanStatus(provider.SpanStatusError); got != codes.Error {
		t.Fatalf("mapSpanStatus(error) = %v", got)
	}
}

func TestTraceProviderNoopAndHTTPExporterPaths(t *testing.T) {
	exporter, name, err := newExporter(TraceConfig{Endpoint: "http://127.0.0.1:4318/v1/traces"})
	if err != nil {
		t.Fatalf("newExporter(http endpoint) error = %v", err)
	}
	if name != "otlphttp" || exporter == nil {
		t.Fatalf("newExporter(http endpoint) = %q, %#v", name, exporter)
	}

	header := make(http.Header)
	noopProvider := noop.NewTracerProvider()
	traceProvider := &traceProvider{
		provider:   noopProvider,
		tracer:     noopProvider.Tracer("test"),
		propagator: propagation.TraceContext{},
	}
	ctx, span := traceProvider.StartSpan(context.Background(), "noop")
	traceProvider.InjectHTTPHeaders(ctx, header)
	if err := traceProvider.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(noop provider) error = %v", err)
	}
	span.RecordError(nil)
	if span.TraceID() != "" || span.SpanID() != "" {
		t.Fatalf("noop span ids = (%q, %q), want empty", span.TraceID(), span.SpanID())
	}

	traced := tracedSpan{span: oteltrace.SpanFromContext(context.Background())}
	if traced.TraceID() != "" || traced.SpanID() != "" {
		t.Fatalf("invalid tracedSpan ids = (%q, %q), want empty", traced.TraceID(), traced.SpanID())
	}
}
