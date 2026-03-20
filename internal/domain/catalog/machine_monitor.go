package catalog

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type MachineReachability struct {
	CheckedAt    time.Time
	Transport    string
	Reachable    bool
	LatencyMS    int64
	FailureCause string
}

type MachineSystemResources struct {
	CollectedAt            time.Time
	CPUCores               int
	CPUUsagePercent        float64
	MemoryTotalGB          float64
	MemoryUsedGB           float64
	MemoryAvailableGB      float64
	MemoryAvailablePercent float64
	DiskTotalGB            float64
	DiskAvailableGB        float64
	DiskAvailablePercent   float64
}

type MachineGPU struct {
	Index              int
	Name               string
	MemoryTotalGB      float64
	MemoryUsedGB       float64
	UtilizationPercent float64
}

type MachineGPUResources struct {
	CollectedAt time.Time
	Available   bool
	GPUs        []MachineGPU
}

func ParseMachineSystemResources(raw string, collectedAt time.Time) (MachineSystemResources, error) {
	values, err := parseMachineMetricLines(raw)
	if err != nil {
		return MachineSystemResources{}, err
	}

	cpuCores, err := parseMetricInt(values, "cpu_cores")
	if err != nil {
		return MachineSystemResources{}, err
	}
	cpuUsagePercent, err := parseMetricFloat(values, "cpu_usage_percent")
	if err != nil {
		return MachineSystemResources{}, err
	}
	memTotalKB, err := parseMetricFloat(values, "memory_total_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}
	memAvailableKB, err := parseMetricFloat(values, "memory_available_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}
	diskTotalKB, err := parseMetricFloat(values, "disk_total_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}
	diskAvailableKB, err := parseMetricFloat(values, "disk_available_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}

	memoryTotalGB := kilobytesToGigabytes(memTotalKB)
	memoryAvailableGB := kilobytesToGigabytes(memAvailableKB)
	memoryUsedGB := roundTwoDecimals(memoryTotalGB - memoryAvailableGB)
	diskTotalGB := kilobytesToGigabytes(diskTotalKB)
	diskAvailableGB := kilobytesToGigabytes(diskAvailableKB)

	return MachineSystemResources{
		CollectedAt:            collectedAt.UTC(),
		CPUCores:               cpuCores,
		CPUUsagePercent:        roundTwoDecimals(cpuUsagePercent),
		MemoryTotalGB:          memoryTotalGB,
		MemoryUsedGB:           memoryUsedGB,
		MemoryAvailableGB:      memoryAvailableGB,
		MemoryAvailablePercent: percentage(memoryAvailableGB, memoryTotalGB),
		DiskTotalGB:            diskTotalGB,
		DiskAvailableGB:        diskAvailableGB,
		DiskAvailablePercent:   percentage(diskAvailableGB, diskTotalGB),
	}, nil
}

func ParseMachineGPUResources(raw string, collectedAt time.Time) (MachineGPUResources, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "no_gpu") {
		return MachineGPUResources{
			CollectedAt: collectedAt.UTC(),
			Available:   false,
			GPUs:        nil,
		}, nil
	}

	reader := csv.NewReader(strings.NewReader(trimmed))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return MachineGPUResources{}, fmt.Errorf("parse gpu metrics csv: %w", err)
	}

	gpus := make([]MachineGPU, 0, len(records))
	for index, record := range records {
		if len(record) != 5 {
			return MachineGPUResources{}, fmt.Errorf("gpu metrics row %d must have 5 columns", index)
		}
		gpuIndex, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu index on row %d: %w", index, err)
		}
		memoryTotalMB, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu memory total on row %d: %w", index, err)
		}
		memoryUsedMB, err := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu memory used on row %d: %w", index, err)
		}
		utilizationPercent, err := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu utilization on row %d: %w", index, err)
		}

		gpus = append(gpus, MachineGPU{
			Index:              gpuIndex,
			Name:               strings.TrimSpace(record[1]),
			MemoryTotalGB:      roundTwoDecimals(memoryTotalMB / 1024.0),
			MemoryUsedGB:       roundTwoDecimals(memoryUsedMB / 1024.0),
			UtilizationPercent: roundTwoDecimals(utilizationPercent),
		})
	}

	return MachineGPUResources{
		CollectedAt: collectedAt.UTC(),
		Available:   true,
		GPUs:        gpus,
	}, nil
}

func parseMachineMetricLines(raw string) (map[string]string, error) {
	values := map[string]string{}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("machine metric line %q must be KEY=VALUE", trimmed)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return values, nil
}

func parseMetricInt(values map[string]string, key string) (int, error) {
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing machine metric %q", key)
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse machine metric %q: %w", key, err)
	}
	return parsed, nil
}

func parseMetricFloat(values map[string]string, key string) (float64, error) {
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing machine metric %q", key)
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("parse machine metric %q: %w", key, err)
	}
	return parsed, nil
}

func kilobytesToGigabytes(value float64) float64 {
	return roundTwoDecimals(value / (1024.0 * 1024.0))
}

func percentage(part float64, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return roundTwoDecimals((part / total) * 100)
}

func roundTwoDecimals(value float64) float64 {
	const factor = 100
	return math.Round(value*factor) / factor
}
