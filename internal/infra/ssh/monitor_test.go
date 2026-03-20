package ssh

import (
	"context"
	"strings"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestMonitorCollectorCollectReachabilityUsesInjectedClockForLatency(t *testing.T) {
	tick := time.Date(2026, 3, 20, 17, 0, 0, 0, time.UTC)
	calls := 0
	collector := &MonitorCollector{
		pool: NewPool("/tmp/openase",
			WithDialer(&fakeDialer{clients: []Client{&fakeClient{}}}),
			WithReadFile(func(string) ([]byte, error) {
				return []byte("key"), nil
			}),
		),
		now: func() time.Time {
			calls++
			switch calls {
			case 1:
				return tick
			case 2:
				return tick
			default:
				return tick.Add(1750 * time.Millisecond)
			}
		},
	}

	reachability, err := collector.CollectReachability(context.Background(), testRemoteMachine())
	if err != nil {
		t.Fatalf("collect reachability: %v", err)
	}
	if !reachability.Reachable {
		t.Fatalf("expected reachability to succeed, got %+v", reachability)
	}
	if reachability.LatencyMS != 1750 {
		t.Fatalf("expected mocked latency, got %+v", reachability)
	}
}

func TestMonitorCollectorCollectReachabilityLocalMachineSkipsPool(t *testing.T) {
	collector := &MonitorCollector{
		now: func() time.Time { return time.Date(2026, 3, 20, 17, 5, 0, 0, time.UTC) },
	}

	reachability, err := collector.CollectReachability(context.Background(), domain.Machine{
		Name: domain.LocalMachineName,
		Host: domain.LocalMachineHost,
	})
	if err != nil {
		t.Fatalf("collect local reachability: %v", err)
	}
	if reachability.Transport != "local" || !reachability.Reachable {
		t.Fatalf("unexpected local reachability result: %+v", reachability)
	}
}

func TestMonitorCollectorCollectAgentEnvironmentInjectsMachineEnvVars(t *testing.T) {
	var capturedScript string
	collector := &MonitorCollector{
		now: func() time.Time { return time.Date(2026, 3, 20, 17, 10, 0, 0, time.UTC) },
		runLocal: func(_ context.Context, script string) ([]byte, error) {
			capturedScript = script
			return []byte(
				"claude_code\tfalse\t\tunknown\tunknown\n" +
					"codex\ttrue\t0.0.1\tunknown\tapi_key\n" +
					"gemini\tfalse\t\tunknown\tunknown\n",
			), nil
		},
	}

	environment, err := collector.CollectAgentEnvironment(context.Background(), domain.Machine{
		Name:    domain.LocalMachineName,
		Host:    domain.LocalMachineHost,
		EnvVars: []string{"OPENAI_API_KEY=sk-test", "PATH=/opt/codex/bin:/usr/bin"},
	})
	if err != nil {
		t.Fatalf("collect agent environment: %v", err)
	}
	if !strings.Contains(capturedScript, "export OPENAI_API_KEY='sk-test'") {
		t.Fatalf("expected OPENAI_API_KEY export in monitor script, got %q", capturedScript)
	}
	if !strings.Contains(capturedScript, "export PATH='/opt/codex/bin:/usr/bin'") {
		t.Fatalf("expected PATH export in monitor script, got %q", capturedScript)
	}
	if environment.CLIs[1].AuthMode != domain.MachineAgentAuthModeAPIKey || !environment.CLIs[1].Ready {
		t.Fatalf("expected codex api-key snapshot to be ready, got %+v", environment.CLIs[1])
	}
}
