package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/httpapi"
	claudecodeadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	codex "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/infra/agentcli"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	"github.com/BetterAndBetterII/openase/internal/orchestrator"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	"github.com/BetterAndBetterII/openase/internal/provider"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	agentplatformrepo "github.com/BetterAndBetterII/openase/internal/repo/agentplatform"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	chatconversationrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	githubauthrepo "github.com/BetterAndBetterII/openase/internal/repo/githubauth"
	humanauthrepo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	machinechannelrepo "github.com/BetterAndBetterII/openase/internal/repo/machinechannel"
	notificationrepo "github.com/BetterAndBetterII/openase/internal/repo/notification"
	scheduledjobrepo "github.com/BetterAndBetterII/openase/internal/repo/scheduledjob"
	secretsrepo "github.com/BetterAndBetterII/openase/internal/repo/secrets"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	runtimeobservability "github.com/BetterAndBetterII/openase/internal/runtime/observability"
	runtimesecretenv "github.com/BetterAndBetterII/openase/internal/runtime/secretenv"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	githubreposervice "github.com/BetterAndBetterII/openase/internal/service/githubrepo"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

var (
	runtimeEventsTopic = provider.MustParseTopic("runtime.events")
	runtimeStartedType = provider.MustParseEventType("runtime.started")
	runtimeTickType    = provider.MustParseEventType("orchestrator.tick")
)

type chatRuntimeEnvironmentResolver struct {
	resolver runtimesecretenv.Resolver
}

func (r chatRuntimeEnvironmentResolver) ResolveProviderEnvironment(
	ctx context.Context,
	input chatservice.RuntimeEnvironmentResolveInput,
) ([]string, error) {
	return runtimesecretenv.AppendResolvedProviderSecrets(ctx, r.resolver, runtimesecretenv.ResolveInput{
		ProjectID:          input.ProjectID,
		ProviderAuthConfig: input.ProviderAuthConfig,
		BaseEnvironment:    input.BaseEnvironment,
		TicketID:           input.TicketID,
		WorkflowID:         input.WorkflowID,
		AgentID:            input.AgentID,
	})
}

type App struct {
	config              config.Config
	logger              *slog.Logger
	events              provider.EventProvider
	trace               provider.TraceProvider
	metrics             provider.MetricsProvider
	metricsHandler      http.Handler
	reverseRuntimeRelay *machinetransport.ReverseRuntimeRelayRegistry
}

func New(
	cfg config.Config,
	logger *slog.Logger,
	events provider.EventProvider,
	trace provider.TraceProvider,
	metrics provider.MetricsProvider,
	metricsHandler http.Handler,
) *App {
	if trace == nil {
		trace = provider.NewNoopTraceProvider()
	}
	if metrics == nil {
		metrics = provider.NewNoopMetricsProvider()
	}

	return &App{
		config:              cfg,
		logger:              logger,
		events:              events,
		trace:               trace,
		metrics:             metrics,
		metricsHandler:      metricsHandler,
		reverseRuntimeRelay: machinetransport.NewReverseRuntimeRelayRegistry(),
	}
}

func (a *App) RunServe(ctx context.Context) error {
	runtimeobservability.NewProcessMemoryReporter(
		runtimeobservability.RuntimeProcessMemoryCollector{},
		a.metrics,
		string(a.config.Server.Mode),
		a.logger,
	).Start(ctx, runtimeobservability.DefaultProcessMemoryReportInterval)

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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve user home directory: %w", err)
	}
	projectsRoot, err := workspaceinfra.LocalProjectStateRoot()
	if err != nil {
		return err
	}
	sshPool := sshinfra.NewPool(filepath.Join(homeDir, ".openase"))
	transportResolver := machinetransport.NewResolver(agentcli.NewManager(agentcli.ManagerOptions{}), sshPool).
		WithReverseRuntimeRelay(a.reverseRuntimeRelay)
	defer func() {
		if closeErr := sshPool.Close(); closeErr != nil {
			a.logger.Error("close ssh pool", "error", closeErr)
		}
	}()

	catalogRepo := catalogrepo.NewEntRepository(client)
	humanAuthRepo := humanauthrepo.NewEntRepository(client)
	humanVisibility := humanauthservice.NewVisibilityResolver(humanAuthRepo)
	githubAuthSvc, err := githubauthservice.New(githubauthrepo.NewEntRepository(client), http.DefaultClient, a.config.ResolvedSecurityCipherSeed())
	if err != nil {
		return err
	}
	secretSvc, err := secretsservice.New(secretsrepo.NewEntRepository(client), a.config.Database.DSN)
	if err != nil {
		return err
	}
	githubRepoSvc := githubreposervice.NewService(githubAuthSvc, http.DefaultClient)
	ticketSvc := ticketservice.NewService(ticketrepo.NewEntRepository(client))
	ticketSvc.ConfigureSSHPool(sshPool)
	ticketSvc.ConfigureTransportResolver(transportResolver)
	ticketSvc.ConfigurePlatformEnvironment(a.agentPlatformAPIURL(), agentplatform.NewService(agentplatformrepo.NewEntRepository(client)))
	ticketSvc.ConfigureActivityEmitter(activitysvc.NewEmitter(activitysvc.EntRecorder{Client: client}, a.events))
	ticketStatusRepo := ticketstatusrepo.NewEntRepository(client)
	ticketStatusSvc := ticketstatus.NewService(ticketStatusRepo)
	catalogSvc := catalogservice.New(
		catalogRepo,
		executable.NewPathResolver(),
		machinetransport.NewTester(transportResolver),
		catalogservice.WithHumanVisibilityResolver(humanVisibility),
		catalogservice.WithMachineHealthCollector(machinetransport.NewMonitorCollector(transportResolver, sshPool)),
		catalogservice.WithProjectStatusBootstrapper(catalogservice.ProjectStatusBootstrapperFunc(func(ctx context.Context, projectID uuid.UUID) error {
			_, err := ticketStatusSvc.ResetToDefaultTemplate(ctx, projectID)
			return err
		})),
	)
	notificationSvc := notificationservice.NewService(notificationrepo.NewEntRepository(client), a.logger, http.DefaultClient)
	if err := notificationservice.NewEngine(notificationSvc, a.events, a.logger).Start(ctx); err != nil {
		return err
	}
	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), a.logger, projectsRoot)
	if err != nil {
		return err
	}
	scheduledJobSvc := scheduledjobservice.NewService(scheduledjobrepo.NewEntRepository(client), ticketSvc, a.logger)
	defer func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			a.logger.Error("close workflow service", "error", closeErr)
		}
	}()
	chatProcessManager := agentcli.NewManager(agentcli.ManagerOptions{})
	codexRuntimeAdapter, err := codex.NewAdapter(codex.AdapterOptions{ProcessManager: chatProcessManager})
	if err != nil {
		return fmt.Errorf("construct chat codex adapter: %w", err)
	}
	chatSecretResolver := chatRuntimeEnvironmentResolver{resolver: secretSvc}
	codexRuntime := chatservice.NewCodexRuntime(codexRuntimeAdapter)
	codexRuntime.ConfigureSecretResolver(chatSecretResolver)
	claudeRuntime := chatservice.NewClaudeRuntime(claudecodeadapter.NewAdapter(chatProcessManager))
	claudeRuntime.ConfigureSecretResolver(chatSecretResolver)
	geminiRuntime := chatservice.NewGeminiRuntime(chatProcessManager)
	geminiRuntime.ConfigureSecretResolver(chatSecretResolver)
	chatSvc := chatservice.NewService(
		a.logger,
		chatservice.NewRuntime(
			claudeRuntime,
			codexRuntime,
			geminiRuntime,
		),
		catalogSvc,
		ticketSvc,
		workflowSvc,
		ticketStatusSvc,
		provider.MustParseAbsolutePath(projectsRoot),
	)
	chatSvc.EnableDurableSessions(filepath.Join(homeDir, ".openase", "chat", "ephemeral-sessions.json"))
	projectConversationSvc := chatservice.NewProjectConversationService(
		a.logger,
		chatconversationrepo.NewEntRepository(client),
		catalogSvc,
		ticketSvc,
		workflowSvc,
		chatProcessManager,
		sshPool,
	)
	projectConversationSvc.ConfigurePlatformEnvironment(a.agentPlatformAPIURL(), agentplatform.NewService(agentplatformrepo.NewEntRepository(client)))
	projectConversationSvc.ConfigureGitHubCredentials(githubAuthSvc)
	projectConversationSvc.ConfigureSecretResolver(chatSecretResolver)
	projectConversationSvc.ConfigureSecretManager(secretSvc)
	projectUpdateSvc := projectupdateservice.NewService(
		client,
		activitysvc.NewEmitter(activitysvc.EntRecorder{Client: client}, a.events),
	)
	ticketWorkspaceResetSvc := orchestrator.NewTicketWorkspaceResetService(client, a.logger, sshPool)
	instanceAuthSvc, err := accesscontrolservice.New(
		accesscontrolrepo.NewEntRepository(client),
		a.config.Database.DSN,
		a.config.Metadata.ConfigFile,
		homeDir,
	)
	if err != nil {
		return fmt.Errorf("construct instance auth service: %w", err)
	}
	if _, err := instanceAuthSvc.Refresh(ctx); err != nil {
		return fmt.Errorf("initialize instance auth runtime state: %w", err)
	}
	humanAuthSvc := humanauthservice.NewService(humanAuthRepo, http.DefaultClient, instanceAuthSvc)
	humanAuthorizer := humanauthservice.NewAuthorizer(humanAuthRepo)
	machineChannelSvc := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client))
	machineSessions := machinechannelservice.NewSessionRegistry(machinechannelservice.DefaultHeartbeatTimeout)
	server := httpapi.NewServer(
		a.config.Server,
		a.config.GitHub,
		a.logger,
		a.events,
		ticketSvc,
		ticketStatusSvc,
		agentplatform.NewService(agentplatformrepo.NewEntRepository(client)),
		catalogSvc,
		workflowSvc,
		httpapi.WithGitHubAuthService(githubAuthSvc),
		httpapi.WithGitHubRepoService(githubRepoSvc),
		httpapi.WithSecretService(secretSvc),
		httpapi.WithInstanceAuthService(instanceAuthSvc),
		httpapi.WithHumanAuthService(humanAuthSvc, humanAuthorizer),
		httpapi.WithRuntimeConfigFile(a.config.Metadata.ConfigFile),
		httpapi.WithHomeDir(homeDir),
		httpapi.WithTraceProvider(a.trace),
		httpapi.WithMetricsProvider(a.metrics),
		httpapi.WithMetricsHandler(a.metricsHandler),
		httpapi.WithScheduledJobService(scheduledJobSvc),
		httpapi.WithNotificationService(notificationSvc),
		httpapi.WithProjectUpdateService(projectUpdateSvc),
		httpapi.WithChatService(chatSvc),
		httpapi.WithProjectConversationService(projectConversationSvc),
		httpapi.WithMachineChannel(machineChannelSvc, machineSessions),
		httpapi.WithReverseRuntimeRelay(a.reverseRuntimeRelay),
		httpapi.WithTicketWorkspaceResetter(ticketWorkspaceResetSvc),
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

	projectsRoot, err := workspaceinfra.LocalProjectStateRoot()
	if err != nil {
		return err
	}

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), a.logger, projectsRoot)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			a.logger.Error("close workflow service", "error", closeErr)
		}
	}()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve user home directory: %w", err)
	}
	sshPool := sshinfra.NewPool(filepath.Join(homeDir, ".openase"))
	transportResolver := machinetransport.NewResolver(agentcli.NewManager(agentcli.ManagerOptions{}), sshPool).
		WithReverseRuntimeRelay(a.reverseRuntimeRelay)
	defer func() {
		if closeErr := sshPool.Close(); closeErr != nil {
			a.logger.Error("close ssh pool", "error", closeErr)
		}
	}()

	githubAuthSvc, err := githubauthservice.New(githubauthrepo.NewEntRepository(client), http.DefaultClient, a.config.ResolvedSecurityCipherSeed())
	if err != nil {
		return err
	}
	secretSvc, err := secretsservice.New(secretsrepo.NewEntRepository(client), a.config.Database.DSN)
	if err != nil {
		return err
	}
	ticketStatusRepo := ticketstatusrepo.NewEntRepository(client)
	scheduler := orchestrator.NewScheduler(client, a.logger, a.events)
	healthChecker := orchestrator.NewHealthChecker(client, a.logger)
	machineMonitor := orchestrator.NewMachineMonitor(client, a.logger, machinetransport.NewMonitorCollector(transportResolver, sshPool))
	machineMonitor.ConfigureEvents(a.events)
	healthChecker.ConfigureEvents(a.events)
	runtimeLauncher := orchestrator.NewRuntimeLauncher(
		client,
		a.logger,
		a.events,
		agentcli.NewManager(agentcli.ManagerOptions{}),
		sshPool,
		workflowSvc,
	)
	runtimeLauncher.ConfigureReverseRuntimeRelay(a.reverseRuntimeRelay)
	runtimeState := orchestrator.NewRuntimeStateStore()
	healthChecker.ConfigureRuntimeState(runtimeState)
	runtimeLauncher.ConfigureRuntimeState(runtimeState)
	runtimeLauncher.ConfigureGitHubCredentials(githubAuthSvc)
	runtimeLauncher.ConfigureSecretResolver(secretSvc)
	runtimeLauncher.ConfigureMetrics(a.metrics)
	runtimeLauncher.ConfigurePlatformEnvironment(a.agentPlatformAPIURL(), agentplatform.NewService(agentplatformrepo.NewEntRepository(client)))
	runtimeLauncher.ConfigureSecretManager(secretSvc)
	runtimeLauncher.ConfigureLaunchTimeouts(
		a.config.Orchestrator.WorkspacePrepareTimeout,
		a.config.Orchestrator.AgentSessionStartTimeout,
	)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtimeLauncher.Close(stopCtx); err != nil {
			a.logger.Warn("close runtime launcher", "error", err)
		}
	}()
	driver, err := a.config.ResolvedEventDriver()
	if err != nil {
		return err
	}
	a.logger.Info(
		"orchestrator runtime ready",
		"tick_interval", a.config.Orchestrator.TickInterval.String(),
		"workspace_prepare_timeout", a.config.Orchestrator.WorkspacePrepareTimeout.String(),
		"agent_session_start_timeout", a.config.Orchestrator.AgentSessionStartTimeout.String(),
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
			tickCtx, span := a.trace.StartSpan(ctx, "orchestrator.tick",
				provider.WithSpanAttributes(
					provider.StringAttribute("runtime.mode", string(a.config.Server.Mode)),
					provider.StringAttribute("tick.time", tick.UTC().Format(time.RFC3339)),
				),
			)
			start := time.Now()
			healthReport, healthErr := healthChecker.Run(tickCtx)
			machineReport, machineErr := machineMonitor.RunTick(tickCtx)
			report, runErr := scheduler.RunTick(tickCtx)
			launchErr := runtimeLauncher.RunTick(tickCtx)
			statusSnapshots, statusErr := ticketstatus.ListStatusRuntimeSnapshots(tickCtx, ticketStatusRepo)
			payload := map[string]any{
				"mode":           string(a.config.Server.Mode),
				"time":           tick.UTC().Format(time.RFC3339),
				"health_report":  healthReport,
				"machine_report": machineReport,
				"report":         report,
			}
			combinedErr := joinOrchestratorTickErrors(healthErr, machineErr, runErr, launchErr, statusErr)
			if combinedErr != nil {
				payload["error"] = combinedErr.Error()
				span.RecordError(combinedErr)
				span.SetStatus(provider.SpanStatusError, combinedErr.Error())
				a.logger.Error(
					"orchestrator tick failed",
					"time", tick.UTC().Format(time.RFC3339),
					"error", combinedErr,
				)
				a.metrics.Counter("openase.orchestrator.tick_total", provider.Tags{
					"mode":   string(a.config.Server.Mode),
					"result": "error",
				}).Add(1)
			} else {
				span.SetStatus(provider.SpanStatusOK, "")
				a.logger.Info(
					"orchestrator tick completed",
					"time", tick.UTC().Format(time.RFC3339),
					"claims_checked", healthReport.ClaimsChecked,
					"stalled_claims", healthReport.StalledClaims,
					"agents_released", healthReport.AgentsReleased,
					"machines_scanned", machineReport.MachinesScanned,
					"machines_updated", machineReport.MachinesUpdated,
					"machine_l1_checks", machineReport.L1Checks,
					"machine_l2_checks", machineReport.L2Checks,
					"machine_l3_checks", machineReport.L3Checks,
					"workflows_scanned", report.WorkflowsScanned,
					"candidates_scanned", report.CandidatesScanned,
					"tickets_dispatched", report.TicketsDispatched,
					"tickets_skipped", report.TicketsSkipped,
				)
				a.metrics.Counter("openase.orchestrator.tick_total", provider.Tags{
					"mode":   string(a.config.Server.Mode),
					"result": "ok",
				}).Add(1)
			}
			a.metrics.Histogram("openase.orchestrator.tick_duration_seconds", provider.Tags{
				"mode": string(a.config.Server.Mode),
			}).Record(time.Since(start).Seconds())
			for _, snapshot := range statusSnapshots {
				tags := provider.Tags{
					"project_id": snapshot.ProjectID.String(),
					"status_id":  snapshot.StatusID.String(),
					"status":     snapshot.Name,
					"limited":    strconv.FormatBool(snapshot.MaxActiveRuns != nil),
				}
				a.metrics.Gauge("openase.ticket.status_active_runs", tags).Set(float64(snapshot.ActiveRuns))
				statusCapacity := 0
				if snapshot.MaxActiveRuns != nil {
					statusCapacity = *snapshot.MaxActiveRuns
				}
				a.metrics.Gauge("openase.ticket.status_capacity", tags).Set(float64(statusCapacity))
			}
			span.SetAttributes(
				provider.IntAttribute("orchestrator.health.claims_checked", healthReport.ClaimsChecked),
				provider.IntAttribute("orchestrator.health.stalled_claims", healthReport.StalledClaims),
				provider.IntAttribute("orchestrator.health.agents_released", healthReport.AgentsReleased),
				provider.IntAttribute("orchestrator.machine.machines_scanned", machineReport.MachinesScanned),
				provider.IntAttribute("orchestrator.machine.machines_updated", machineReport.MachinesUpdated),
				provider.IntAttribute("orchestrator.machine.l1_checks", machineReport.L1Checks),
				provider.IntAttribute("orchestrator.machine.l2_checks", machineReport.L2Checks),
				provider.IntAttribute("orchestrator.machine.l3_checks", machineReport.L3Checks),
				provider.IntAttribute("orchestrator.report.workflows_scanned", report.WorkflowsScanned),
				provider.IntAttribute("orchestrator.report.candidates_scanned", report.CandidatesScanned),
				provider.IntAttribute("orchestrator.report.tickets_dispatched", report.TicketsDispatched),
				provider.IntAttribute("orchestrator.report.tickets_skipped.total", sumSkipCounts(report.TicketsSkipped)),
			)
			for reason, count := range report.TicketsSkipped {
				span.SetAttributes(provider.IntAttribute("orchestrator.report.tickets_skipped."+reason, count))
			}
			publishErr := a.publishRuntimeEvent(ctx, runtimeTickType, payload)
			if publishErr != nil {
				span.RecordError(publishErr)
				span.SetStatus(provider.SpanStatusError, publishErr.Error())
			}
			span.End()
			if publishErr != nil {
				return fmt.Errorf("publish scheduler tick: %w", publishErr)
			}
		}
	}
}

func (a *App) agentPlatformAPIURL() string {
	host := strings.TrimSpace(a.config.Server.Host)
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}

	return "http://" + net.JoinHostPort(host, strconv.Itoa(a.config.Server.Port)) + "/api/v1/platform"
}

func sumSkipCounts(values map[string]int) int {
	total := 0
	for _, value := range values {
		total += value
	}

	return total
}

func joinOrchestratorTickErrors(healthErr error, machineErr error, schedulerErr error, launcherErr error, stageErr error) error {
	errs := make([]error, 0, 5)
	if healthErr != nil {
		errs = append(errs, fmt.Errorf("health check: %w", healthErr))
	}
	if machineErr != nil {
		errs = append(errs, fmt.Errorf("machine monitor: %w", machineErr))
	}
	if schedulerErr != nil {
		errs = append(errs, fmt.Errorf("scheduler: %w", schedulerErr))
	}
	if launcherErr != nil {
		errs = append(errs, fmt.Errorf("runtime launcher: %w", launcherErr))
	}
	if stageErr != nil {
		errs = append(errs, fmt.Errorf("stage metrics: %w", stageErr))
	}
	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
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
