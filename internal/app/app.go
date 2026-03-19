package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/httpapi"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"golang.org/x/sync/errgroup"
)

var (
	runtimeEventsTopic = provider.MustParseTopic("runtime.events")
	runtimeStartedType = provider.MustParseEventType("runtime.started")
	runtimeTickType    = provider.MustParseEventType("orchestrator.tick")
)

type App struct {
	config config.Config
	logger *slog.Logger
	events provider.EventProvider
}

func New(cfg config.Config, logger *slog.Logger, events provider.EventProvider) *App {
	return &App{
		config: cfg,
		logger: logger,
		events: events,
	}
}

func (a *App) RunServe(ctx context.Context) error {
	if err := a.startRuntimeEventLogging(ctx); err != nil {
		return err
	}

	server := httpapi.NewServer(a.config.Server, a.logger, a.events)
	driver, err := a.config.ResolvedEventDriver()
	if err != nil {
		return err
	}
	a.logger.Info("serve runtime ready", "config_file", a.config.Metadata.ConfigFile, "event_driver", driver)

	return server.Run(ctx)
}

func (a *App) RunOrchestrate(ctx context.Context) error {
	driver, err := a.config.ResolvedEventDriver()
	if err != nil {
		return err
	}
	a.logger.Info(
		"orchestrator runtime ready",
		"tick_interval", a.config.Orchestrator.TickInterval.String(),
		"config_file", a.config.Metadata.ConfigFile,
		"event_driver", driver,
	)
	if err := a.publishRuntimeEvent(ctx, runtimeStartedType, map[string]string{"mode": string(a.config.Server.Mode)}); err != nil {
		return fmt.Errorf("publish runtime started event: %w", err)
	}

	ticker := time.NewTicker(a.config.Orchestrator.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("orchestrator runtime stopping")
			return nil
		case tick := <-ticker.C:
			if err := a.publishRuntimeEvent(ctx, runtimeTickType, map[string]string{
				"mode": string(a.config.Server.Mode),
				"time": tick.UTC().Format(time.RFC3339),
			}); err != nil {
				return fmt.Errorf("publish scheduler tick: %w", err)
			}
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

func (a *App) publishRuntimeEvent(ctx context.Context, eventType provider.EventType, payload any) error {
	event, err := provider.NewJSONEvent(runtimeEventsTopic, eventType, payload, time.Now())
	if err != nil {
		return err
	}

	return a.events.Publish(ctx, event)
}

func (a *App) startRuntimeEventLogging(ctx context.Context) error {
	stream, err := a.events.Subscribe(ctx, runtimeEventsTopic)
	if err != nil {
		return fmt.Errorf("subscribe runtime events: %w", err)
	}

	go func() {
		for event := range stream {
			a.logger.Info(
				"runtime event received",
				"topic", event.Topic.String(),
				"type", event.Type.String(),
				"published_at", event.PublishedAt.Format(time.RFC3339),
				"payload", string(event.Payload),
			)
		}
	}()

	return nil
}
