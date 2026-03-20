package otel

import (
	"context"
	"net/http"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/provider"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceProviderStartsAndExportsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	traceProvider := newTestTraceProvider(exporter)

	ctx := context.Background()
	ctx, span := traceProvider.StartSpan(ctx, "http.request",
		provider.WithSpanKind(provider.SpanKindServer),
		provider.WithSpanAttributes(
			provider.StringAttribute("http.method", http.MethodGet),
			provider.IntAttribute("http.status_code", http.StatusOK),
			provider.BoolAttribute("http.sampled", true),
		),
	)
	if span.TraceID() == "" {
		t.Fatal("expected trace id")
	}
	if span.SpanID() == "" {
		t.Fatal("expected span id")
	}

	carrier := http.Header{}
	traceProvider.InjectHTTPHeaders(ctx, carrier)
	if carrier.Get("Traceparent") == "" {
		t.Fatalf("expected traceparent header, got %v", carrier)
	}

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected one exported span, got %d", len(spans))
	}
	if spans[0].Name != "http.request" {
		t.Fatalf("expected span name http.request, got %q", spans[0].Name)
	}
	if got := attributeValue(t, spans[0].Attributes, "http.method").AsString(); got != http.MethodGet {
		t.Fatalf("expected http.method attribute %q, got %q", http.MethodGet, got)
	}
	if got := attributeValue(t, spans[0].Attributes, "http.status_code").AsInt64(); got != http.StatusOK {
		t.Fatalf("expected http.status_code attribute %d, got %d", http.StatusOK, got)
	}
	if got := attributeValue(t, spans[0].Attributes, "http.sampled").AsBool(); !got {
		t.Fatal("expected http.sampled attribute true")
	}
}

func TestTraceProviderExtractsIncomingTraceContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	traceProvider := newTestTraceProvider(exporter)

	parentCtx, parentSpan := traceProvider.StartSpan(context.Background(), "parent")

	headers := http.Header{}
	traceProvider.InjectHTTPHeaders(parentCtx, headers)

	childCtx := traceProvider.ExtractHTTPContext(context.Background(), headers)
	_, childSpan := traceProvider.StartSpan(childCtx, "child")
	childSpan.End()
	parentSpan.End()

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected two exported spans, got %d", len(spans))
	}
	parent := spanStubByName(t, spans, "parent")
	child := spanStubByName(t, spans, "child")
	if child.Parent.SpanID() != parent.SpanContext.SpanID() {
		t.Fatalf("expected child span parent %s, got %s", parent.SpanContext.SpanID(), child.Parent.SpanID())
	}
}

func newTestTraceProvider(exporter sdktrace.SpanExporter) *traceProvider {
	sdkProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	return &traceProvider{
		provider:   sdkProvider,
		tracer:     sdkProvider.Tracer(instrumentationName),
		propagator: propagator,
	}
}

func attributeValue(t *testing.T, attrs []attribute.KeyValue, key string) attribute.Value {
	t.Helper()

	for _, attr := range attrs {
		if string(attr.Key) == key {
			return attr.Value
		}
	}

	t.Fatalf("expected attribute %q", key)
	return attribute.Value{}
}

func spanStubByName(t *testing.T, spans tracetest.SpanStubs, name string) tracetest.SpanStub {
	t.Helper()

	for _, span := range spans {
		if span.Name == name {
			return span
		}
	}

	t.Fatalf("expected span %q", name)
	return tracetest.SpanStub{}
}
