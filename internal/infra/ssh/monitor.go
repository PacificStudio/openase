package ssh

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

const (
	systemResourceScript = `
cpu_before=$(awk '/^cpu / {print $2+$3+$4+$5+$6+$7+$8, $5}' /proc/stat)
sleep 0.2
cpu_after=$(awk '/^cpu / {print $2+$3+$4+$5+$6+$7+$8, $5}' /proc/stat)
cpu_total_before=$(printf '%s\n' "$cpu_before" | awk '{print $1}')
cpu_idle_before=$(printf '%s\n' "$cpu_before" | awk '{print $2}')
cpu_total_after=$(printf '%s\n' "$cpu_after" | awk '{print $1}')
cpu_idle_after=$(printf '%s\n' "$cpu_after" | awk '{print $2}')
cpu_total_delta=$((cpu_total_after-cpu_total_before))
cpu_idle_delta=$((cpu_idle_after-cpu_idle_before))
cpu_usage=$(awk -v total="$cpu_total_delta" -v idle="$cpu_idle_delta" 'BEGIN { if (total <= 0) { print "0.00"; exit } printf "%.2f", ((total-idle) * 100) / total }')
cpu_cores=$(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc)
memory_total_kb=$(awk '/^MemTotal:/ {print $2}' /proc/meminfo)
memory_available_kb=$(awk '/^MemAvailable:/ {print $2}' /proc/meminfo)
disk_total_kb=$(df -kP / | awk 'NR==2 {print $2}')
disk_available_kb=$(df -kP / | awk 'NR==2 {print $4}')
printf 'cpu_cores=%s\n' "$cpu_cores"
printf 'cpu_usage_percent=%s\n' "$cpu_usage"
printf 'memory_total_kb=%s\n' "$memory_total_kb"
printf 'memory_available_kb=%s\n' "$memory_available_kb"
printf 'disk_total_kb=%s\n' "$disk_total_kb"
printf 'disk_available_kb=%s\n' "$disk_available_kb"
`
	gpuResourceScript = `
if ! command -v nvidia-smi >/dev/null 2>&1; then
  printf 'no_gpu\n'
  exit 0
fi
nvidia-smi --query-gpu=index,name,memory.total,memory.used,utilization.gpu --format=csv,noheader,nounits
`
)

type MonitorCollector struct {
	pool     *Pool
	now      func() time.Time
	runLocal func(context.Context, string) ([]byte, error)
}

func NewMonitorCollector(pool *Pool) *MonitorCollector {
	return &MonitorCollector{
		pool: pool,
		now:  time.Now,
		runLocal: func(ctx context.Context, script string) ([]byte, error) {
			//nolint:gosec // The shell path is fixed and script bodies are package constants.
			return exec.CommandContext(ctx, "sh", "-lc", script).CombinedOutput()
		},
	}
}

func (c *MonitorCollector) CollectReachability(ctx context.Context, machine domain.Machine) (domain.MachineReachability, error) {
	checkedAt := c.now().UTC()
	if machine.Host == domain.LocalMachineHost {
		return domain.MachineReachability{
			CheckedAt: checkedAt,
			Transport: "local",
			Reachable: true,
		}, nil
	}
	if c == nil || c.pool == nil {
		return domain.MachineReachability{
			CheckedAt:    checkedAt,
			Transport:    "ssh",
			FailureCause: "ssh pool unavailable",
		}, fmt.Errorf("ssh pool unavailable")
	}

	startedAt := c.now().UTC()
	_, err := c.pool.Get(ctx, machine)
	latency := c.now().UTC().Sub(startedAt).Milliseconds()
	if err != nil {
		return domain.MachineReachability{
			CheckedAt:    checkedAt,
			Transport:    "ssh",
			LatencyMS:    latency,
			FailureCause: err.Error(),
		}, err
	}

	return domain.MachineReachability{
		CheckedAt: checkedAt,
		Transport: "ssh",
		Reachable: true,
		LatencyMS: latency,
	}, nil
}

func (c *MonitorCollector) CollectSystemResources(ctx context.Context, machine domain.Machine) (domain.MachineSystemResources, error) {
	collectedAt := c.now().UTC()
	output, err := c.runScript(ctx, machine, systemResourceScript)
	if err != nil {
		return domain.MachineSystemResources{}, err
	}

	return domain.ParseMachineSystemResources(string(output), collectedAt)
}

func (c *MonitorCollector) CollectGPUResources(ctx context.Context, machine domain.Machine) (domain.MachineGPUResources, error) {
	collectedAt := c.now().UTC()
	output, err := c.runScript(ctx, machine, gpuResourceScript)
	if err != nil {
		return domain.MachineGPUResources{}, err
	}

	return domain.ParseMachineGPUResources(string(output), collectedAt)
}

func (c *MonitorCollector) runScript(ctx context.Context, machine domain.Machine, script string) ([]byte, error) {
	if machine.Host == domain.LocalMachineHost {
		if c == nil || c.runLocal == nil {
			return nil, fmt.Errorf("local monitor runner unavailable")
		}
		output, err := c.runLocal(ctx, script)
		if err != nil {
			return nil, fmt.Errorf("run local monitor script: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return output, nil
	}
	if c == nil || c.pool == nil {
		return nil, fmt.Errorf("ssh pool unavailable")
	}

	client, err := c.pool.Get(ctx, machine)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("open ssh session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	output, err := session.CombinedOutput("sh -lc " + shellQuote(script))
	if err != nil {
		return nil, fmt.Errorf("run remote monitor script: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return output, nil
}

func shellQuote(raw string) string {
	return "'" + strings.ReplaceAll(raw, "'", `'"'"'`) + "'"
}
