package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/spf13/cobra"
)

var managedServiceName = provider.MustParseServiceName("openase")

const managedServiceDescription = "OpenASE -- Auto Software Engineering Platform"

type upCommandDeps struct {
	resolveConfigPath              func(string) (provider.AbsolutePath, error)
	runSetupWizard                 func(context.Context, io.Writer) error
	buildUserServiceManager        func() (provider.UserServiceManager, error)
	buildManagedServiceInstallSpec func(string) (provider.UserServiceInstallSpec, error)
}

func newUpCommand(options *rootOptions) *cobra.Command {
	return newUpCommandWithDeps(options, upCommandDeps{
		resolveConfigPath:              resolveManagedServiceConfigPath,
		runSetupWizard:                 runDefaultSetupWizard,
		buildUserServiceManager:        buildUserServiceManager,
		buildManagedServiceInstallSpec: buildManagedServiceInstallSpec,
	})
}

func newUpCommandWithDeps(options *rootOptions, deps upCommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start setup wizard on first run, otherwise install or update the user service.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := deps.resolveConfigPath(options.configFile)
			if err != nil {
				return err
			}
			if configPath == "" {
				return deps.runSetupWizard(cmd.Context(), cmd.OutOrStdout())
			}

			manager, err := deps.buildUserServiceManager()
			if err != nil {
				return err
			}

			spec, err := deps.buildManagedServiceInstallSpec(configPath.String())
			if err != nil {
				return err
			}

			if err := manager.Apply(cmd.Context(), spec); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "openase service applied via %s\n", manager.Platform())
			return nil
		},
	}
}

func newDownCommand(_ *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the managed OpenASE user service.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, err := buildUserServiceManager()
			if err != nil {
				return err
			}
			if err := manager.Down(cmd.Context(), managedServiceName); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "openase service stopped via %s\n", manager.Platform())
			return nil
		},
	}
}

func newRestartCommand(_ *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the managed OpenASE user service.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, err := buildUserServiceManager()
			if err != nil {
				return err
			}
			if err := manager.Restart(cmd.Context(), managedServiceName); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "openase service restarted via %s\n", manager.Platform())
			return nil
		},
	}
}

func newLogsCommand(_ *rootOptions) *cobra.Command {
	var follow bool
	var lines int

	command := &cobra.Command{
		Use:   "logs",
		Short: "Tail logs from the managed OpenASE user service.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, err := buildUserServiceManager()
			if err != nil {
				return err
			}

			opts, err := provider.NewUserServiceLogsOptions(lines, follow, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}

			return manager.Logs(cmd.Context(), managedServiceName, opts)
		},
	}

	command.Flags().BoolVarP(&follow, "follow", "f", true, "Keep streaming new log lines.")
	command.Flags().IntVarP(&lines, "lines", "n", 200, "Number of log lines to print before streaming.")

	return command
}

func buildManagedServiceInstallSpec(configFile string) (provider.UserServiceInstallSpec, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return provider.UserServiceInstallSpec{}, fmt.Errorf("resolve current executable: %w", err)
	}
	executablePath, err = filepath.Abs(executablePath)
	if err != nil {
		return provider.UserServiceInstallSpec{}, fmt.Errorf("resolve executable path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return provider.UserServiceInstallSpec{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".openase")
	workingDirectory := provider.MustParseAbsolutePath(baseDir)
	environmentFile := provider.MustParseAbsolutePath(filepath.Join(baseDir, ".env"))
	stdoutPath := provider.MustParseAbsolutePath(filepath.Join(baseDir, "logs", managedServiceName.String()+".stdout.log"))
	stderrPath := provider.MustParseAbsolutePath(filepath.Join(baseDir, "logs", managedServiceName.String()+".stderr.log"))
	programPath := provider.MustParseAbsolutePath(executablePath)

	args := []string{"all-in-one"}
	resolvedConfigPath, err := resolveManagedServiceConfigPath(configFile)
	if err != nil {
		return provider.UserServiceInstallSpec{}, err
	}
	if resolvedConfigPath != "" {
		args = append(args, "--config", resolvedConfigPath.String())
	}

	spec, err := provider.NewUserServiceInstallSpec(
		managedServiceName,
		managedServiceDescription,
		programPath,
		args,
		workingDirectory,
		environmentFile,
		stdoutPath,
		stderrPath,
	)
	if err != nil {
		return provider.UserServiceInstallSpec{}, err
	}

	return spec, nil
}

func resolveManagedServiceConfigPath(configFile string) (provider.AbsolutePath, error) {
	if configFile != "" {
		absolutePath, err := filepath.Abs(configFile)
		if err != nil {
			return "", fmt.Errorf("resolve config path %q: %w", configFile, err)
		}
		info, err := os.Stat(absolutePath)
		if err != nil {
			return "", fmt.Errorf("stat config path %q: %w", absolutePath, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("config path %q must be a file", absolutePath)
		}

		return provider.ParseAbsolutePath(absolutePath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	for _, candidate := range managedServiceConfigCandidates(cwd, homeDir) {
		info, statErr := os.Stat(candidate)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}

			return "", fmt.Errorf("stat config candidate %q: %w", candidate, statErr)
		}
		if info.IsDir() {
			continue
		}

		return provider.ParseAbsolutePath(candidate)
	}

	return "", nil
}

func managedServiceConfigCandidates(cwd string, homeDir string) []string {
	candidates := make([]string, 0, 9)
	for _, extension := range []string{"yaml", "yml", "json", "toml"} {
		candidates = append(candidates, filepath.Join(cwd, "openase."+extension))
	}
	candidates = append(candidates, filepath.Join(homeDir, ".openase", "config.yaml"))
	for _, extension := range []string{"yaml", "yml", "json", "toml"} {
		candidates = append(candidates, filepath.Join(homeDir, ".openase", "openase."+extension))
	}

	return candidates
}
