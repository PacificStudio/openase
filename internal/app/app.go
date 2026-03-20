package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/httpapi"
	claudecodeadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	agentcliruntime "github.com/BetterAndBetterII/openase/internal/infra/agentcli"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	"github.com/BetterAndBetterII/openase/internal/orchestrator"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
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
	client, err := database.Open(ctx, a.config.Database.DSN)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			a.logger.Error("close database", "error", closeErr)
		}
	}()

	if err := a.startRuntimeEventLogging(ctx); err != nil {
		return err
	}

	catalogRepo := catalogrepo.NewEntRepository(client)
	catalogSvc := catalogservice.New(catalogRepo, executable.NewPathResolver())
	notificationSvc := notificationservice.NewService(client, a.logger, http.DefaultClient)
	if err := notificationservice.NewEngine(notificationSvc, a.events, a.logger).Start(ctx); err != nil {
		return err
	}
	workflowSvc, err := workflowservice.NewService(client, a.logger, "")
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			a.logger.Error("close workflow service", "error", closeErr)
		}
	}()
	ticketSvc := ticketservice.NewService(client)
	chatSvc := chatservice.NewService(
		a.logger,
		claudecodeadapter.NewAdapter(agentcliruntime.NewManager(agentcliruntime.ManagerOptions{})),
		catalogSvc,
		ticketSvc,
		workflowSvc,
	)
	server := httpapi.NewServer(
		a.config.Server,
		a.config.GitHub,
		a.logger,
		a.events,
		ticketSvc,
		ticketstatus.NewService(client),
		agentplatform.NewService(client),
		catalogSvc,
		workflowSvc,
		httpapi.WithNotificationService(notificationSvc),
		httpapi.WithChatService(chatSvc),
	)
	driver, err := a.config.ResolvedEventDriver()
	if err != nil {
		return err
	}
	a.logger.Info("serve runtime ready", "config_file", a.config.Metadata.ConfigFile, "event_driver", driver)

	return server.Run(ctx)
}

func (a *App) RunOrchestrate(ctx context.Context) error {
	client, err := database.Open(ctx, a.config.Database.DSN)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			a.logger.Error("close database", "error", closeErr)
		}
	}()

	scheduler := orchestrator.NewScheduler(client, a.logger)
	healthChecker := orchestrator.NewHealthChecker(client, a.logger)
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
			healthReport, healthErr := healthChecker.Run(ctx)
			report, runErr := scheduler.RunTick(ctx)
			payload := map[string]any{
				"mode":          string(a.config.Server.Mode),
				"time":          tick.UTC().Format(time.RFC3339),
				"health_report": healthReport,
				"report":        report,
			}
			combinedErr := joinOrchestratorTickErrors(healthErr, runErr)
			if combinedErr != nil {
				payload["error"] = combinedErr.Error()
				a.logger.Error(
					"orchestrator tick failed",
					"time", tick.UTC().Format(time.RFC3339),
					"error", combinedErr,
				)
			} else {
				a.logger.Info(
					"orchestrator tick completed",
					"time", tick.UTC().Format(time.RFC3339),
					"claims_checked", healthReport.ClaimsChecked,
					"stalled_claims", healthReport.StalledClaims,
					"agents_released", healthReport.AgentsReleased,
					"workflows_scanned", report.WorkflowsScanned,
					"candidates_scanned", report.CandidatesScanned,
					"tickets_dispatched", report.TicketsDispatched,
					"tickets_skipped", report.TicketsSkipped,
				)
			}
			if err := a.publishRuntimeEvent(ctx, runtimeTickType, payload); err != nil {
				return fmt.Errorf("publish scheduler tick: %w", err)
			}
		}
	}
}

func joinOrchestratorTickErrors(healthErr error, schedulerErr error) error {
	if healthErr == nil {
		return schedulerErr
	}
	if schedulerErr == nil {
		return fmt.Errorf("health check: %w", healthErr)
	}

	return fmt.Errorf("health check: %w; scheduler: %v", healthErr, schedulerErr)
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
