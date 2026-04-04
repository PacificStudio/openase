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

func TestBuildAgentEnvironmentScriptUsesRelaxedCodexLoginMatch(t *testing.T) {
	script := buildAgentEnvironmentScript(domain.Machine{})
	if !strings.Contains(script, `claude auth status --json`) {
		t.Fatalf("expected claude auth probe to prefer json status, got %q", script)
	}
	if !strings.Contains(script, `"loggedIn"[[:space:]]*:[[:space:]]*true`) {
		t.Fatalf("expected claude auth probe to check loggedIn json field, got %q", script)
	}
	if !strings.Contains(script, `grep -Eq 'Logged in|Login method:'`) {
		t.Fatalf("expected claude auth probe to keep legacy text fallback, got %q", script)
	}
	if !strings.Contains(script, `login status 2>&1 | grep -q 'Logged in'`) {
		t.Fatalf("expected relaxed codex login match in script, got %q", script)
	}
	if strings.Contains(script, `login status 2>/dev/null`) {
		t.Fatalf("expected codex login match to inspect stderr output, got %q", script)
	}
	if !strings.Contains(script, `${GEMINI_API_KEY:-}`) || !strings.Contains(script, `${GOOGLE_API_KEY:-}`) {
		t.Fatalf("expected gemini api-key probe in script, got %q", script)
	}
	if !strings.Contains(script, `${GOOGLE_CLOUD_PROJECT:-}`) || !strings.Contains(script, `${GOOGLE_CLOUD_LOCATION:-}`) {
		t.Fatalf("expected gemini vertex env probe in script, got %q", script)
	}
	if !strings.Contains(script, `/.gemini/settings.json`) || !strings.Contains(script, `/.gemini/google_accounts.json`) || !strings.Contains(script, `/.gemini/oauth_creds.json`) {
		t.Fatalf("expected gemini oauth cache probe in script, got %q", script)
	}
	if !strings.Contains(script, `"selectedType"[[:space:]]*:[[:space:]]*"\([^"]*\)"`) {
		t.Fatalf("expected gemini selectedType parsing in script, got %q", script)
	}
	if !strings.Contains(script, `"active"[[:space:]]*:[[:space:]]*"[^"]+"`) || !strings.Contains(script, `"refresh_token"[[:space:]]*:[[:space:]]*"[^"]+"`) {
		t.Fatalf("expected gemini oauth credential probes in script, got %q", script)
	}
}

func TestMonitorCollectorCollectFullAuditIncludesGitHubTokenProbe(t *testing.T) {
	collector := &MonitorCollector{
		now: func() time.Time { return time.Date(2026, 3, 28, 12, 30, 0, 0, time.UTC) },
		runLocal: func(_ context.Context, script string) ([]byte, error) {
			if !strings.Contains(script, "export GH_TOKEN='ghu_probe_token'") {
				t.Fatalf("expected GH_TOKEN to be projected into audit script, got %q", script)
			}
			return []byte(
				"git\ttrue\tOpenASE\topenase@example.com\n" +
					"gh_cli\ttrue\tlogged_in\n" +
					"github_token_probe\ttrue\tvalid\ttrue\trepo\tgranted\t\n" +
					"network\ttrue\ttrue\ttrue\n",
			), nil
		},
	}

	audit, err := collector.CollectFullAudit(context.Background(), domain.Machine{
		Name:    domain.LocalMachineName,
		Host:    domain.LocalMachineHost,
		EnvVars: []string{"GH_TOKEN=ghu_probe_token"},
	})
	if err != nil {
		t.Fatalf("CollectFullAudit() error = %v", err)
	}
	if audit.GitHubTokenProbe.State != "valid" || !audit.GitHubTokenProbe.Valid {
		t.Fatalf("expected github token probe in full audit, got %+v", audit.GitHubTokenProbe)
	}
}
