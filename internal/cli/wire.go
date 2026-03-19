package cli

import (
	"fmt"
	"log/slog"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func buildEventProvider(cfg config.Config, logger *slog.Logger) (provider.EventProvider, error) {
	driver, err := cfg.ResolvedEventDriver()
	if err != nil {
		return nil, err
	}

	switch driver {
	case config.EventDriverChannel:
		logger.Info("configured event provider", "driver", driver, "mode", cfg.Server.Mode)
		return eventinfra.NewChannelBus(), nil
	case config.EventDriverPGNotify:
		bus, err := eventinfra.NewPGNotifyBus(cfg.Database.DSN, logger)
		if err != nil {
			return nil, err
		}

		logger.Info("configured event provider", "driver", driver, "mode", cfg.Server.Mode)
		return bus, nil
	default:
		return nil, fmt.Errorf("unsupported event driver %q", driver)
	}
}
