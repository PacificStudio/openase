package observability

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	otelinfra "github.com/BetterAndBetterII/openase/internal/infra/otel"
)

type staticMemoryCollector struct {
	snapshot ProcessMemorySnapshot
}

func (c staticMemoryCollector) Snapshot() ProcessMemorySnapshot {
	return c.snapshot
}

func TestProcessMemoryReporterExportsSnapshot(t *testing.T) {
	metricsProvider, err := otelinfra.NewMetricsProvider(context.Background(), otelinfra.MetricsConfig{
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

	reporter := NewProcessMemoryReporter(
		staticMemoryCollector{snapshot: ProcessMemorySnapshot{
			Goroutines:        17,
			AllocBytes:        2048,
			TotalAllocBytes:   4096,
			SysBytes:          8192,
			HeapAllocBytes:    1024,
			HeapInuseBytes:    1536,
			HeapIdleBytes:     2560,
			HeapReleasedBytes: 512,
			StackInuseBytes:   768,
			NextGCBytes:       6144,
			GCCycles:          9,
		}},
		metricsProvider,
		"serve",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	reporter.SnapshotAndReport()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	metricsProvider.PrometheusHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected scrape status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		`openase_system_memory_alloc_bytes{mode="serve"} 2048`,
		`openase_system_memory_heap_inuse_bytes{mode="serve"} 1536`,
		`openase_system_memory_next_gc_bytes{mode="serve"} 6144`,
		`openase_system_memory_gc_cycles{mode="serve"} 9`,
		`openase_system_runtime_goroutines{mode="serve"} 17`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected scrape to contain %q, got %q", expected, body)
		}
	}
}
