package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/BetterAndBetterII/openase/internal/app"
	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	configFile string
}

func NewRootCommand(version string) *cobra.Command {
	options := &rootOptions{}
	rootCmd := &cobra.Command{
		Use:           "openase",
		Short:         "OpenASE is an issue-driven automated software engineering platform.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&options.configFile, "config", "", "Path to an OpenASE config file.")

	rootCmd.AddCommand(newServeCommand(options))
	rootCmd.AddCommand(newOrchestrateCommand(options))
	rootCmd.AddCommand(newAllInOneCommand(options))
	rootCmd.AddCommand(newUpCommand(options))
	rootCmd.AddCommand(newSetupCommand())
	rootCmd.AddCommand(newDownCommand(options))
	rootCmd.AddCommand(newRestartCommand(options))
	rootCmd.AddCommand(newLogsCommand(options))
	rootCmd.AddCommand(newDoctorCommand(options))
	rootCmd.AddCommand(newTicketCommand())
	rootCmd.AddCommand(newProjectCommand())
	rootCmd.AddCommand(newIssueAgentTokenCommand(options))
	rootCmd.AddCommand(newVersionCommand(version))

	return rootCmd
}

func newServeCommand(options *rootOptions) *cobra.Command {
	var host string
	var port int

	command := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API server.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			overrides := map[string]any{
				"server.mode": config.ServerModeServe,
			}
			if cmd.Flags().Changed("host") {
				overrides["server.host"] = host
			}
			if cmd.Flags().Changed("port") {
				overrides["server.port"] = port
			}

			return runWithConfig(cmd.Context(), options.configFile, overrides, func(ctx context.Context, instance *app.App) error {
				return instance.RunServe(ctx)
			})
		},
	}

	command.Flags().StringVar(&host, "host", "", "HTTP listen host override.")
	command.Flags().IntVar(&port, "port", 0, "HTTP listen port override.")

	return command
}

func newOrchestrateCommand(options *rootOptions) *cobra.Command {
	var tickInterval string

	command := &cobra.Command{
		Use:   "orchestrate",
		Short: "Run the orchestration loop.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			overrides := map[string]any{
				"server.mode": config.ServerModeOrchestrate,
			}
			if cmd.Flags().Changed("tick-interval") {
				overrides["orchestrator.tick_interval"] = tickInterval
			}

			return runWithConfig(cmd.Context(), options.configFile, overrides, func(ctx context.Context, instance *app.App) error {
				return instance.RunOrchestrate(ctx)
			})
		},
	}

	command.Flags().StringVar(&tickInterval, "tick-interval", "", "Scheduler tick interval override, for example 5s.")

	return command
}

func newAllInOneCommand(options *rootOptions) *cobra.Command {
	var host string
	var port int
	var tickInterval string

	command := &cobra.Command{
		Use:   "all-in-one",
		Short: "Run the API server and orchestrator in a single process.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			overrides := map[string]any{
				"server.mode": config.ServerModeAllInOne,
			}
			if cmd.Flags().Changed("host") {
				overrides["server.host"] = host
			}
			if cmd.Flags().Changed("port") {
				overrides["server.port"] = port
			}
			if cmd.Flags().Changed("tick-interval") {
				overrides["orchestrator.tick_interval"] = tickInterval
			}

			return runWithConfig(cmd.Context(), options.configFile, overrides, func(ctx context.Context, instance *app.App) error {
				return instance.RunAllInOne(ctx)
			})
		},
	}

	command.Flags().StringVar(&host, "host", "", "HTTP listen host override.")
	command.Flags().IntVar(&port, "port", 0, "HTTP listen port override.")
	command.Flags().StringVar(&tickInterval, "tick-interval", "", "Scheduler tick interval override, for example 5s.")

	return command
}

func newVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the OpenASE CLI version.",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(os.Stdout, version)
			return err
		},
	}
}

func runWithConfig(
	parent context.Context,
	configFile string,
	overrides map[string]any,
	run func(context.Context, *app.App) error,
) error {
	cfg, err := config.Load(config.LoadOptions{
		ConfigFile: configFile,
		Overrides:  overrides,
	})
	if err != nil {
		return err
	}

	logger := logging.New(cfg.Logging)
	slog.SetDefault(logger)

	eventProvider, err := buildEventProvider(cfg, logger)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := eventProvider.Close(); closeErr != nil {
			logger.Error("close event provider", "error", closeErr)
		}
	}()

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, app.New(cfg, logger, eventProvider))
}
