package observability

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

const DefaultProcessMemoryReportInterval = 5 * time.Second

type ProcessMemorySnapshot struct {
	ObservedAt        time.Time
	Goroutines        int
	AllocBytes        uint64
	TotalAllocBytes   uint64
	SysBytes          uint64
	HeapAllocBytes    uint64
	HeapInuseBytes    uint64
	HeapIdleBytes     uint64
	HeapReleasedBytes uint64
	StackInuseBytes   uint64
	NextGCBytes       uint64
	GCCycles          uint32
}

type ProcessMemoryCollector interface {
	Snapshot() ProcessMemorySnapshot
}

type RuntimeProcessMemoryCollector struct{}

func (RuntimeProcessMemoryCollector) Snapshot() ProcessMemorySnapshot {
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	return ProcessMemorySnapshot{
		ObservedAt:        time.Now().UTC(),
		Goroutines:        runtime.NumGoroutine(),
		AllocBytes:        memStats.Alloc,
		TotalAllocBytes:   memStats.TotalAlloc,
		SysBytes:          memStats.Sys,
		HeapAllocBytes:    memStats.HeapAlloc,
		HeapInuseBytes:    memStats.HeapInuse,
		HeapIdleBytes:     memStats.HeapIdle,
		HeapReleasedBytes: memStats.HeapReleased,
		StackInuseBytes:   memStats.StackInuse,
		NextGCBytes:       memStats.NextGC,
		GCCycles:          memStats.NumGC,
	}
}

type ProcessMemoryReporter struct {
	logger    *slog.Logger
	collector ProcessMemoryCollector
	metrics   provider.MetricsProvider
	mode      string
}

func NewProcessMemoryReporter(
	collector ProcessMemoryCollector,
	metrics provider.MetricsProvider,
	mode string,
	logger *slog.Logger,
) *ProcessMemoryReporter {
	if collector == nil {
		collector = RuntimeProcessMemoryCollector{}
	}
	if metrics == nil {
		metrics = provider.NewNoopMetricsProvider()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &ProcessMemoryReporter{
		logger:    logger.With("component", "process-memory-metrics"),
		collector: collector,
		metrics:   metrics,
		mode:      normalizeModeTag(mode),
	}
}

func (r *ProcessMemoryReporter) SnapshotAndReport() ProcessMemorySnapshot {
	snapshot := r.collector.Snapshot()
	tags := provider.Tags{"mode": r.mode}

	r.metrics.Gauge("openase.system.memory.alloc_bytes", tags).Set(float64(snapshot.AllocBytes))
	r.metrics.Gauge("openase.system.memory.total_alloc_bytes", tags).Set(float64(snapshot.TotalAllocBytes))
	r.metrics.Gauge("openase.system.memory.sys_bytes", tags).Set(float64(snapshot.SysBytes))
	r.metrics.Gauge("openase.system.memory.heap_alloc_bytes", tags).Set(float64(snapshot.HeapAllocBytes))
	r.metrics.Gauge("openase.system.memory.heap_inuse_bytes", tags).Set(float64(snapshot.HeapInuseBytes))
	r.metrics.Gauge("openase.system.memory.heap_idle_bytes", tags).Set(float64(snapshot.HeapIdleBytes))
	r.metrics.Gauge("openase.system.memory.heap_released_bytes", tags).Set(float64(snapshot.HeapReleasedBytes))
	r.metrics.Gauge("openase.system.memory.stack_inuse_bytes", tags).Set(float64(snapshot.StackInuseBytes))
	r.metrics.Gauge("openase.system.memory.next_gc_bytes", tags).Set(float64(snapshot.NextGCBytes))
	r.metrics.Gauge("openase.system.memory.gc_cycles", tags).Set(float64(snapshot.GCCycles))
	r.metrics.Gauge("openase.system.runtime.goroutines", tags).Set(float64(snapshot.Goroutines))

	return snapshot
}

func (r *ProcessMemoryReporter) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultProcessMemoryReportInterval
	}

	snapshot := r.SnapshotAndReport()
	r.logger.Info(
		"started process memory reporting",
		"mode", r.mode,
		"interval", interval.String(),
		"heap_inuse_bytes", snapshot.HeapInuseBytes,
		"sys_bytes", snapshot.SysBytes,
	)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.SnapshotAndReport()
			}
		}
	}()
}

func normalizeModeTag(mode string) string {
	trimmed := strings.TrimSpace(mode)
	if trimmed == "" {
		return "unknown"
	}

	return trimmed
}
