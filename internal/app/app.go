package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/httpapi"
	"golang.org/x/sync/errgroup"
)

type App struct {
	config config.Config
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) *App {
	return &App{
		config: cfg,
		logger: logger,
	}
}

func (a *App) RunServe(ctx context.Context) error {
	server := httpapi.NewServer(a.config.Server, a.logger)
	a.logger.Info("serve runtime ready", "config_file", a.config.Metadata.ConfigFile)

	return server.Run(ctx)
}

func (a *App) RunOrchestrate(ctx context.Context) error {
	a.logger.Info(
		"orchestrator runtime ready",
		"tick_interval", a.config.Orchestrator.TickInterval.String(),
		"config_file", a.config.Metadata.ConfigFile,
	)

	ticker := time.NewTicker(a.config.Orchestrator.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("orchestrator runtime stopping")
			return nil
		case tick := <-ticker.C:
			a.logger.Info("scheduler tick", "time", tick.UTC().Format(time.RFC3339))
		}
	}
}

func (a *App) RunAllInOne(ctx context.Context) error {
	a.logger.Info("all-in-one runtime ready")

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := a.RunServe(groupCtx); err != nil {
			return fmt.Errorf("serve runtime: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		if err := a.RunOrchestrate(groupCtx); err != nil {
			return fmt.Errorf("orchestrate runtime: %w", err)
		}
		return nil
	})

	return group.Wait()
}
