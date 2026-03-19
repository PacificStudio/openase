package logging

import (
	"log/slog"
	"os"

	"github.com/BetterAndBetterII/openase/internal/config"
)

func New(cfg config.LoggingConfig) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.Level}

	switch cfg.Format {
	case config.LogFormatJSON:
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	default:
		return slog.New(slog.NewTextHandler(os.Stdout, options))
	}
}
