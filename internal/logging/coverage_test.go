package logging

import (
	"log/slog"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
)

func TestNewReturnsLoggerForConfiguredFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config config.LoggingConfig
	}{
		{
			name: "json",
			config: config.LoggingConfig{
				Level:  slog.LevelWarn,
				Format: config.LogFormatJSON,
			},
		},
		{
			name: "text fallback",
			config: config.LoggingConfig{
				Level:  slog.LevelInfo,
				Format: config.LogFormat("unknown"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if logger := New(tt.config); logger == nil {
				t.Fatal("New() = nil")
			}
		})
	}
}
