package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
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

func TestSetupCommandRejectsRemovedLegacyWebFlags(t *testing.T) {
	command := newSetupCommand()
	command.SetArgs([]string{"--web"})

	err := command.Execute()
	if err == nil {
		t.Fatal("Execute() expected unknown flag error for removed --web flag")
	}
	if !strings.Contains(err.Error(), "unknown flag: --web") {
		t.Fatalf("Execute() error = %v", err)
	}
}
