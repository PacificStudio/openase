package otel

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
