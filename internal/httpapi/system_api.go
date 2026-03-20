package httpapi

import (
	"net/http"
	"time"

	runtimeobservability "github.com/BetterAndBetterII/openase/internal/runtime/observability"
	"github.com/labstack/echo/v4"
)

type OpenAPISystemMemorySnapshot struct {
	ObservedAt        string `json:"observed_at"`
	Goroutines        int    `json:"goroutines"`
	AllocBytes        uint64 `json:"alloc_bytes"`
	TotalAllocBytes   uint64 `json:"total_alloc_bytes"`
	SysBytes          uint64 `json:"sys_bytes"`
	HeapAllocBytes    uint64 `json:"heap_alloc_bytes"`
	HeapInuseBytes    uint64 `json:"heap_inuse_bytes"`
	HeapIdleBytes     uint64 `json:"heap_idle_bytes"`
	HeapReleasedBytes uint64 `json:"heap_released_bytes"`
	StackInuseBytes   uint64 `json:"stack_inuse_bytes"`
	NextGCBytes       uint64 `json:"next_gc_bytes"`
	GCCycles          uint32 `json:"gc_cycles"`
}

type OpenAPISystemDashboardResponse struct {
	Memory OpenAPISystemMemorySnapshot `json:"memory"`
}

func (s *Server) handleSystemDashboard(c echo.Context) error {
	return c.JSON(http.StatusOK, OpenAPISystemDashboardResponse{
		Memory: encodeSystemMemorySnapshot(s.memoryCollector.Snapshot()),
	})
}

func encodeSystemMemorySnapshot(
	snapshot runtimeobservability.ProcessMemorySnapshot,
) OpenAPISystemMemorySnapshot {
	return OpenAPISystemMemorySnapshot{
		ObservedAt:        snapshot.ObservedAt.UTC().Format(time.RFC3339),
		Goroutines:        snapshot.Goroutines,
		AllocBytes:        snapshot.AllocBytes,
		TotalAllocBytes:   snapshot.TotalAllocBytes,
		SysBytes:          snapshot.SysBytes,
		HeapAllocBytes:    snapshot.HeapAllocBytes,
		HeapInuseBytes:    snapshot.HeapInuseBytes,
		HeapIdleBytes:     snapshot.HeapIdleBytes,
		HeapReleasedBytes: snapshot.HeapReleasedBytes,
		StackInuseBytes:   snapshot.StackInuseBytes,
		NextGCBytes:       snapshot.NextGCBytes,
		GCCycles:          snapshot.GCCycles,
	}
}
