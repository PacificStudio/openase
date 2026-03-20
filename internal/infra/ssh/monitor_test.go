package ssh

import (
	"context"
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
