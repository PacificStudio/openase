package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/app"
	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/spf13/cobra"
)

func TestRunWithConfigLoadsConfigAndInvokesRunner(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
server:
  mode: all-in-one
event:
  driver: channel
observability:
  metrics:
    enabled: false
  tracing:
    enabled: false
    service_name: openase-test
`)), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	expectedErr := errors.New("stop after coverage")
	called := false
	err := runWithConfig(context.Background(), configPath, nil, func(ctx context.Context, instance *app.App) error {
		called = true
		if ctx == nil {
			t.Fatal("runWithConfig() passed nil context")
		}
		if instance == nil {
			t.Fatal("runWithConfig() passed nil app")
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("runWithConfig() error = %v, want %v", err, expectedErr)
	}
	if !called {
		t.Fatal("runWithConfig() did not invoke runner")
	}
}

func TestModeCommandsReturnConfigLoadErrorsAfterApplyingFlags(t *testing.T) {
	missingConfig := filepath.Join(t.TempDir(), "missing.yaml")
	options := &rootOptions{configFile: missingConfig}

	tests := []struct {
		name    string
		command *cobra.Command
		args    []string
	}{
		{
			name:    "serve",
			command: newServeCommand(options),
			args:    []string{"--host", "127.0.0.2", "--port", "19001"},
		},
		{
			name:    "orchestrate",
			command: newOrchestrateCommand(options),
			args:    []string{"--tick-interval", "2s"},
		},
		{
			name:    "all-in-one",
			command: newAllInOneCommand(options),
			args:    []string{"--host", "127.0.0.3", "--port", "19002", "--tick-interval", "3s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			tt.command.SetOut(&output)
			tt.command.SetErr(&output)
			tt.command.SetArgs(tt.args)

			err := tt.command.ExecuteContext(context.Background())
			if err == nil || !strings.Contains(err.Error(), "read config file") {
				t.Fatalf("ExecuteContext() error = %v, want read config file failure", err)
			}
		})
	}
}

func TestCLIWireBuilders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	eventProvider, err := buildEventProvider(config.Config{
		Server: config.ServerConfig{Mode: config.ServerModeAllInOne},
		Event:  config.EventConfig{Driver: config.EventDriverChannel},
	}, logger)
	if err != nil {
		t.Fatalf("buildEventProvider(channel) error = %v", err)
	}
	if err := eventProvider.Close(); err != nil {
		t.Fatalf("eventProvider.Close() error = %v", err)
	}

	if _, err := buildEventProvider(config.Config{
		Event: config.EventConfig{Driver: config.EventDriver("bogus")},
	}, logger); err == nil || !strings.Contains(err.Error(), "unsupported event driver") {
		t.Fatalf("buildEventProvider(bogus) error = %v", err)
	}

	manager, err := buildUserServiceManager()
	switch runtime.GOOS {
	case "linux", "darwin":
		if err != nil {
			t.Fatalf("buildUserServiceManager() error = %v", err)
		}
		if manager == nil || manager.Platform() == "" {
			t.Fatalf("buildUserServiceManager() = %#v", manager)
		}
	default:
		if err == nil || !strings.Contains(err.Error(), "unsupported OS") {
			t.Fatalf("buildUserServiceManager() error = %v", err)
		}
	}

	metricsRuntime, err := buildMetricsProvider(config.Config{
		Observability: config.ObservabilityConfig{
			Metrics: config.MetricsConfig{Enabled: false},
		},
	}, logger)
	if err != nil {
		t.Fatalf("buildMetricsProvider(disabled) error = %v", err)
	}
	if metricsRuntime.provider == nil || metricsRuntime.shutdown == nil {
		t.Fatalf("buildMetricsProvider(disabled) = %+v", metricsRuntime)
	}
	if err := metricsRuntime.shutdown(context.Background()); err != nil {
		t.Fatalf("metricsRuntime.shutdown() error = %v", err)
	}

	traceProvider, err := buildTraceProvider(config.Config{
		Observability: config.ObservabilityConfig{
			Tracing: config.TraceConfig{
				Enabled:     false,
				ServiceName: "openase-test",
			},
		},
	}, logger)
	if err != nil {
		t.Fatalf("buildTraceProvider(disabled) error = %v", err)
	}
	if traceProvider == nil {
		t.Fatal("buildTraceProvider(disabled) returned nil provider")
	}
	if err := traceProvider.Shutdown(context.Background()); err != nil {
		t.Fatalf("traceProvider.Shutdown() error = %v", err)
	}
}

func TestCLISetupHelpers(t *testing.T) {
	var banner bytes.Buffer
	printSetupWebBanner(&banner, "http://127.0.0.1:19836/setup")
	if !strings.Contains(banner.String(), "http://127.0.0.1:19836/setup") {
		t.Fatalf("printSetupWebBanner() = %q", banner.String())
	}

	t.Setenv("PATH", t.TempDir())
	if err := openBrowser("http://127.0.0.1:19836/setup"); err == nil {
		t.Fatal("openBrowser() expected error when browser launcher is missing")
	}

	var wizard bytes.Buffer
	err := runSetupWebWizard(context.Background(), &wizard, "300.300.300.300", freeCLIPort(t))
	if err == nil {
		t.Fatal("runSetupWebWizard() expected listener error for invalid host")
	}
	if !strings.Contains(wizard.String(), "legacy web setup") {
		t.Fatalf("runSetupWebWizard() output = %q", wizard.String())
	}
}

func freeCLIPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	tcpAddr := listener.Addr().(*net.TCPAddr)
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	return tcpAddr.Port
}
