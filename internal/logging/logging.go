package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/config"
)

// New builds the process logger from runtime logging configuration.
func New(cfg config.LoggingConfig) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.Level}

	switch cfg.Format {
	case config.LogFormatJSON:
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	default:
		return slog.New(slog.NewTextHandler(os.Stdout, options))
	}
}

// DeclareComponent normalizes a component name for package-level reuse.
func DeclareComponent(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		panic("logging component name must not be empty")
	}
	return trimmed
}

// WithComponent returns a logger tagged with the supplied component.
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logger.With("component", DeclareComponent(component))
}
