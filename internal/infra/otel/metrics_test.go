package otel

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestMetricsProviderExportsCounterHistogramAndGauge(t *testing.T) {
	metricsProvider, err := NewMetricsProvider(context.Background(), MetricsConfig{
		ServiceName: "openase",
		Prometheus:  true,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewMetricsProvider returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := metricsProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}
	})

	metricsProvider.Counter("openase.test.counter_total", map[string]string{"kind": "demo"}).Add(3)
	metricsProvider.Histogram("openase.test.duration_seconds", map[string]string{"kind": "demo"}).Record(0.25)
	metricsProvider.Gauge("openase.test.queue_depth", map[string]string{"state": "ready"}).Set(7)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	metricsProvider.PrometheusHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected scrape status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		`openase_test_counter_total{kind="demo"} 3`,
		`openase_test_duration_seconds_count{kind="demo"} 1`,
		`openase_test_duration_seconds_sum{kind="demo"} 0.25`,
		`openase_test_queue_depth{state="ready"} 7`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected scrape to contain %q, got %q", expected, body)
		}
	}
}

func TestMetricsProviderHelperPathsAndNoopFallbacks(t *testing.T) {
	metricsProvider, err := NewMetricsProvider(context.Background(), MetricsConfig{}, nil)
	if err != nil {
		t.Fatalf("NewMetricsProvider(defaults) error = %v", err)
	}
	t.Cleanup(func() {
		if err := metricsProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown(defaults) error = %v", err)
		}
	})

	counter := metricsProvider.Counter(" openase.test.counter_total ", map[string]string{"kind": "demo"})
	counter.Add(1)
	if reflect.TypeOf(counter).Name() != "otelCounter" {
		t.Fatalf("Counter() type = %T, want otelCounter", counter)
	}
	if reflect.TypeOf(metricsProvider.Counter("openase.test.counter_total", map[string]string{"kind": "demo"})).Name() != "otelCounter" {
		t.Fatalf("Counter() cached type mismatch")
	}

	histogram := metricsProvider.Histogram("openase.test.duration_seconds", map[string]string{"kind": "demo"})
	histogram.Record(0.5)
	if reflect.TypeOf(histogram).Name() != "otelHistogram" {
		t.Fatalf("Histogram() type = %T, want otelHistogram", histogram)
	}
	if reflect.TypeOf(metricsProvider.Histogram("openase.test.duration_seconds", map[string]string{"kind": "demo"})).Name() != "otelHistogram" {
		t.Fatalf("Histogram() cached type mismatch")
	}

	gauge := metricsProvider.Gauge("openase.test.queue_depth", map[string]string{"state": "ready"})
	gauge.Set(3)
	if reflect.TypeOf(gauge).Name() != "otelGauge" {
		t.Fatalf("Gauge() type = %T, want otelGauge", gauge)
	}
	if reflect.TypeOf(metricsProvider.Gauge("openase.test.queue_depth", map[string]string{"state": "ready"})).Name() != "otelGauge" {
		t.Fatalf("Gauge() cached type mismatch")
	}

	metricsProvider.Counter(" ", nil).Add(1)
	metricsProvider.Histogram("openase.bad.histogram", map[string]string{" ": "value"}).Record(1)
	metricsProvider.Gauge("openase.bad.gauge", map[string]string{" ": "value"}).Set(1)

	if _, _, err := parseInstrumentSpec(" ", nil); err == nil || !strings.Contains(err.Error(), "metric name must not be empty") {
		t.Fatalf("parseInstrumentSpec(blank name) error = %v", err)
	}
	if _, _, err := parseInstrumentSpec("metric.name", map[string]string{" ": "value"}); err == nil || !strings.Contains(err.Error(), "metric tag key must not be empty") {
		t.Fatalf("parseInstrumentSpec(blank tag key) error = %v", err)
	}
	spacedKey := " b "
	if _, key, err := parseInstrumentSpec(" metric.name ", map[string]string{spacedKey: " 2 ", "a": "1"}); err != nil || key != "metric.name|b=2|a=1" {
		t.Fatalf("parseInstrumentSpec(sorted) = %q, %v", key, err)
	}
}
