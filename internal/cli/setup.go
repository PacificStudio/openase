package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	userserviceinfra "github.com/BetterAndBetterII/openase/internal/infra/userservice"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/BetterAndBetterII/openase/internal/setup"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	defaultSetupURL        = "http://127.0.0.1:19836"
	defaultSetupConfigPath = "~/.openase/config.yaml"
)

var errSetupAborted = errors.New("setup aborted")

type setupRuntimeMode string

const (
	setupRuntimeModeConfigOnly         setupRuntimeMode = "config-only"
	setupRuntimeModeManagedUserService setupRuntimeMode = "managed-user-service"
)

type setupFlowDeps struct {
	buildUserServiceManager        func() (provider.UserServiceManager, error)
	buildManagedServiceInstallSpec func(string) (provider.UserServiceInstallSpec, error)
	checkManagedUserServiceSupport func(context.Context) error
	verifyManagedUserService       func(context.Context, provider.ServiceName) error
	buildInstalledSetupService     func(context.Context, string, provider.UserServiceInstallSpec) *installedSetupService
	goos                           string
}

type installedSetupService struct {
	Name          provider.ServiceName
	Platform      string
	InstallSpec   provider.UserServiceInstallSpec
	LaunchdTarget string
	LaunchdPlist  string
}

type setupFlowService interface {
	Bootstrap(context.Context) (setup.Bootstrap, error)
	PrepareDatabase(context.Context, setup.RawDatabaseSourceInput) (setup.PreparedDatabase, error)
	Complete(context.Context, setup.RawCompleteRequest) (setup.CompleteResult, error)
}

type setupFlowOptions struct {
	allowOverwrite bool
}

type setupPrompter struct {
	reader *bufio.Reader
	inFile *os.File
	out    io.Writer
}

func newSetupCommand() *cobra.Command {
	var force bool

	command := &cobra.Command{
		Use:   "setup",
		Short: "Run interactive local setup for OpenASE.",
		Long: strings.TrimSpace(`
Run the interactive local setup flow for OpenASE.

The default flow stays inside the terminal, prepares a PostgreSQL connection,
checks key local CLIs, configures auth mode, optionally installs the current-
user managed service, and writes a runnable ~/.openase/config.yaml plus
~/.openase/.env.
`),
		Example: "openase setup\nopenase setup --force",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSetupFlowCommand(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), setupFlowOptions{
				allowOverwrite: force,
			})
		},
	}

	command.Flags().BoolVar(&force, "force", false, "Overwrite an existing ~/.openase/config.yaml without prompting.")
	command.AddCommand(newSetupDesktopCommand())

	return command
}

func runDefaultSetupWizard(ctx context.Context, out io.Writer) error {
	return runSetupFlowCommand(ctx, os.Stdin, out, setupFlowOptions{})
}

func runSetupFlowCommand(ctx context.Context, in io.Reader, out io.Writer, opts setupFlowOptions) error {
	service, err := newSetupServiceFromEnv()
	if err != nil {
		return err
	}
	err = runSetupFlowWithDeps(ctx, in, out, service, opts, setupFlowDeps{
		buildUserServiceManager:        buildUserServiceManager,
		buildManagedServiceInstallSpec: buildManagedServiceInstallSpec,
		checkManagedUserServiceSupport: checkManagedUserServiceSupport,
		verifyManagedUserService:       verifyManagedUserService,
		buildInstalledSetupService:     buildInstalledSetupService,
		goos:                           runtime.GOOS,
	})
	if err != nil {
		printSetupFailure(out, defaultSetupConfigPath)
		return err
	}
	return nil
}

func newSetupDesktopCommand() *cobra.Command {
	command := &cobra.Command{
		Use:    "desktop",
		Short:  "Run desktop-oriented setup helpers.",
		Hidden: true,
	}
	command.AddCommand(
		newSetupDesktopBootstrapCommand(),
		newSetupDesktopPreflightCommand(),
		newSetupDesktopApplyCommand(),
	)
	return command
}

func newSetupDesktopBootstrapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Print setup bootstrap metadata for the desktop shell.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := newSetupServiceFromEnv()
			if err != nil {
				return err
			}
			result, err := service.Bootstrap(cmd.Context())
			if err != nil {
				return err
			}
			return writeSetupJSON(cmd.OutOrStdout(), result)
		},
	}
}

func newSetupDesktopPreflightCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "preflight",
		Short: "Run the desktop first-launch preflight checks.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := newSetupServiceFromEnv()
			if err != nil {
				return err
			}
			result, err := service.DesktopPreflight(cmd.Context())
			if err != nil {
				return err
			}
			return writeSetupJSON(cmd.OutOrStdout(), result)
		},
	}
}

func newSetupDesktopApplyCommand() *cobra.Command {
	var inputPath string
	command := &cobra.Command{
		Use:   "apply",
		Short: "Apply the desktop setup request from JSON input.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := readInputFile(strings.TrimSpace(inputPath))
			if err != nil {
				return err
			}
			var request setup.RawDesktopApplyRequest
			if err := json.Unmarshal(body, &request); err != nil {
				return fmt.Errorf("decode desktop setup request: %w", err)
			}

			service, err := newSetupServiceFromEnv()
			if err != nil {
				return err
			}
			result, err := service.DesktopApply(cmd.Context(), request)
			if err != nil {
				return err
			}
			return writeSetupJSON(cmd.OutOrStdout(), result)
		},
	}
	command.Flags().StringVar(&inputPath, "input", "", "Read the raw JSON desktop setup request from a file. Use - for stdin.")
	_ = command.MarkFlagRequired("input")
	return command
}

func newSetupServiceFromEnv() (*setup.Service, error) {
	return setup.NewService(setup.Options{
		HomeDir:    resolveSetupHomeOverride(),
		ConfigPath: resolveSetupConfigOverride(),
	})
}

func resolveSetupHomeOverride() string {
	if value := strings.TrimSpace(os.Getenv("OPENASE_SETUP_HOME")); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv("OPENASE_DESKTOP_OPENASE_HOME"))
}

func resolveSetupConfigOverride() string {
	if value := strings.TrimSpace(os.Getenv("OPENASE_SETUP_CONFIG_PATH")); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv("OPENASE_DESKTOP_OPENASE_CONFIG"))
}

func writeSetupJSON(out io.Writer, payload any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func runSetupFlowWithDeps(
	ctx context.Context,
	in io.Reader,
	out io.Writer,
	service setupFlowService,
	opts setupFlowOptions,
	deps setupFlowDeps,
) error {
	bootstrap, err := service.Bootstrap(ctx)
	if err != nil {
		return err
	}

	prompter := newSetupPrompter(in, out)
	printSetupIntro(out, bootstrap)

	allowOverwrite := opts.allowOverwrite
	if bootstrap.ConfigExists && !allowOverwrite {
		confirmed, confirmErr := prompter.confirm(
			fmt.Sprintf("Config already exists at %s. Overwrite it?", bootstrap.ConfigPath),
			false,
		)
		if confirmErr != nil {
			return confirmErr
		}
		if !confirmed {
			_, _ = fmt.Fprintln(out, "\nSetup cancelled. Existing configuration was left unchanged.")
			return nil
		}
		allowOverwrite = true
	}

	prepared, err := promptPreparedDatabase(ctx, prompter, service, bootstrap)
	if err != nil {
		if errors.Is(err, errSetupAborted) {
			_, _ = fmt.Fprintln(out, "\nSetup cancelled.")
			return nil
		}
		return err
	}

	printCLIDiagnostics(out, bootstrap.CLI)
	runtimeMode, err := promptRuntimeMode(ctx, prompter, deps)
	if err != nil {
		if errors.Is(err, errSetupAborted) {
			_, _ = fmt.Fprintln(out, "\nSetup cancelled.")
			return nil
		}
		return err
	}

	goos := deps.targetGOOS()

	printSetupSummary(out, bootstrap, prepared, runtimeMode, goos)

	confirmationLabel := "Write ~/.openase/config.yaml and ~/.openase/.env now?"
	if runtimeMode == setupRuntimeModeManagedUserService {
		confirmationLabel = fmt.Sprintf(
			"Write ~/.openase/config.yaml and ~/.openase/.env, then install the managed OpenASE user service via %s now?",
			setupManagedUserServicePlatformName(goos),
		)
	}
	confirmed, err := prompter.confirm(confirmationLabel, true)
	if err != nil {
		return err
	}
	if !confirmed {
		_, _ = fmt.Fprintln(out, "\nSetup cancelled before writing files.")
		if prepared.Source == setup.DatabaseSourceDocker && prepared.Docker != nil {
			_, _ = fmt.Fprintf(
				out,
				"Local PostgreSQL container %q is still available for reuse.\n",
				prepared.Docker.ContainerName,
			)
		}
		return nil
	}

	result, err := service.Complete(ctx, setup.RawCompleteRequest{
		Database:       prepared.Config.Raw(),
		AllowOverwrite: allowOverwrite,
	})
	if err != nil {
		return err
	}

	var installedService *installedSetupService
	if runtimeMode == setupRuntimeModeManagedUserService {
		installedService, err = installSetupManagedService(ctx, deps, result.ConfigPath)
		if err != nil {
			return err
		}
	}

	printSetupSuccess(out, result, prepared, installedService)
	return nil
}

func promptPreparedDatabase(
	ctx context.Context,
	prompter *setupPrompter,
	service setupFlowService,
	bootstrap setup.Bootstrap,
) (setup.PreparedDatabase, error) {
	for {
		source, err := promptDatabaseSource(prompter, bootstrap.Sources)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}

		switch source {
		case setup.DatabaseSourceManual:
			return promptManualDatabase(ctx, prompter, service, bootstrap.Defaults.ManualDatabase)
		case setup.DatabaseSourceDocker:
			return promptDockerDatabase(ctx, prompter, service, bootstrap.Defaults.DockerDatabase)
		default:
			return setup.PreparedDatabase{}, fmt.Errorf("unsupported database source %q", source)
		}
	}
}

func promptDatabaseSource(prompter *setupPrompter, sources []setup.DatabaseSourceOption) (setup.DatabaseSourceType, error) {
	if len(sources) == 0 {
		return "", fmt.Errorf("no database sources are available")
	}

	options := make([]string, 0, len(sources))
	for _, source := range sources {
		options = append(options, fmt.Sprintf("%s: %s", source.Name, source.Description))
	}
	index, err := prompter.choose("Choose a database source", options, 0)
	if err != nil {
		return "", err
	}

	return sources[index].ID, nil
}

func promptManualDatabase(
	ctx context.Context,
	prompter *setupPrompter,
	service setupFlowService,
	defaults setup.RawDatabaseInput,
) (setup.PreparedDatabase, error) {
	current := defaults
	for {
		var err error
		current.Host, err = prompter.stringValue("PostgreSQL host", current.Host)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}
		current.Port, err = prompter.intValue("PostgreSQL port", current.Port)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}
		current.Name, err = prompter.stringValue("Database name", current.Name)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}
		current.User, err = prompter.stringValue("Database user", current.User)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}
		current.Password, err = prompter.secretValue("Database password", current.Password)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}
		current.SSLMode, err = prompter.selectValue("SSL mode", []string{"disable", "require", "verify-full"}, current.SSLMode)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}

		prepared, err := service.PrepareDatabase(ctx, setup.RawDatabaseSourceInput{
			Type:   string(setup.DatabaseSourceManual),
			Manual: &current,
		})
		if err == nil {
			_, _ = fmt.Fprintln(prompter.out, "\nManual database connection succeeded.")
			return prepared, nil
		}

		_, _ = fmt.Fprintf(prompter.out, "\nDatabase connection failed: %v\n", err)
		retry, confirmErr := prompter.confirm("Edit the database settings and try again?", true)
		if confirmErr != nil {
			return setup.PreparedDatabase{}, confirmErr
		}
		if !retry {
			return setup.PreparedDatabase{}, errSetupAborted
		}
	}
}

func promptDockerDatabase(
	ctx context.Context,
	prompter *setupPrompter,
	service setupFlowService,
	defaults setup.RawDockerDatabaseInput,
) (setup.PreparedDatabase, error) {
	for {
		_, _ = fmt.Fprintln(prompter.out)
		_, _ = fmt.Fprintln(prompter.out, "Docker PostgreSQL defaults:")
		_, _ = fmt.Fprintf(prompter.out, "  Container: %s\n", defaults.ContainerName)
		_, _ = fmt.Fprintf(prompter.out, "  Database:  %s\n", defaults.DatabaseName)
		_, _ = fmt.Fprintf(prompter.out, "  User:      %s\n", defaults.User)
		_, _ = fmt.Fprintf(prompter.out, "  Port:      127.0.0.1:%d -> 5432\n", defaults.Port)
		_, _ = fmt.Fprintf(prompter.out, "  Volume:    %s\n", defaults.VolumeName)
		_, _ = fmt.Fprintf(prompter.out, "  Image:     %s\n", defaults.Image)

		confirmed, err := prompter.confirm("Create and validate this local PostgreSQL container now?", true)
		if err != nil {
			return setup.PreparedDatabase{}, err
		}
		if !confirmed {
			return setup.PreparedDatabase{}, errSetupAborted
		}

		prepared, err := service.PrepareDatabase(ctx, setup.RawDatabaseSourceInput{
			Type:   string(setup.DatabaseSourceDocker),
			Docker: &defaults,
		})
		if err == nil {
			_, _ = fmt.Fprintln(prompter.out, "\nDocker PostgreSQL is ready and the connection succeeded.")
			return prepared, nil
		}

		_, _ = fmt.Fprintf(prompter.out, "\nDocker setup failed: %v\n", err)
		retry, confirmErr := prompter.confirm("Retry Docker database setup?", true)
		if confirmErr != nil {
			return setup.PreparedDatabase{}, confirmErr
		}
		if !retry {
			return setup.PreparedDatabase{}, errSetupAborted
		}
	}
}

func printSetupIntro(out io.Writer, bootstrap setup.Bootstrap) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "OpenASE local setup")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "This flow will:")
	_, _ = fmt.Fprintln(out, "  1. Prepare and validate PostgreSQL")
	_, _ = fmt.Fprintln(out, "  2. Check key local CLIs such as git, codex, and claude")
	_, _ = fmt.Fprintln(out, "  3. Configure local runtime defaults without legacy auth.mode or inline OIDC YAML")
	_, _ = fmt.Fprintln(out, "  4. Choose config-only mode or managed user service mode")
	_, _ = fmt.Fprintln(out, "  5. Write ~/.openase/config.yaml and ~/.openase/.env")
	_, _ = fmt.Fprintln(out, "  6. Initialize the default local workspace metadata")
	if bootstrap.ConfigExists {
		_, _ = fmt.Fprintf(out, "\nExisting config detected: %s\n", bootstrap.ConfigPath)
	}
}

func printCLIDiagnostics(out io.Writer, diagnostics []setup.CLIDiagnostic) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "CLI checks:")
	for _, diagnostic := range diagnostics {
		status := "WARN"
		switch diagnostic.Status {
		case "ready":
			status = "OK  "
		case "version_error":
			status = "WARN"
		case "missing":
			status = "WARN"
		}

		line := diagnostic.Name
		switch diagnostic.Status {
		case "ready":
			line += " - " + diagnostic.Version
			if diagnostic.Path != "" {
				line += " (" + diagnostic.Path + ")"
			}
		case "version_error":
			line += " - found on PATH, but version detection failed"
			if diagnostic.Path != "" {
				line += " (" + diagnostic.Path + ")"
			}
		default:
			line += " - not found on PATH"
		}
		_, _ = fmt.Fprintf(out, "  [%s] %s\n", status, line)
	}
}

func printSetupSummary(
	out io.Writer,
	bootstrap setup.Bootstrap,
	prepared setup.PreparedDatabase,
	runtimeMode setupRuntimeMode,
	goos string,
) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Setup summary:")
	_, _ = fmt.Fprintf(
		out,
		"  Database: %s %s:%d/%s (user=%s, sslmode=%s)\n",
		prepared.Source,
		prepared.Config.Host,
		prepared.Config.Port,
		prepared.Config.Name,
		prepared.Config.User,
		prepared.Config.SSLMode,
	)
	if prepared.Docker != nil {
		_, _ = fmt.Fprintf(out, "  Docker container: %s\n", prepared.Docker.ContainerName)
		_, _ = fmt.Fprintf(out, "  Docker volume:    %s\n", prepared.Docker.VolumeName)
	}
	_, _ = fmt.Fprintln(out, "  Browser auth: local bootstrap link until an active OIDC config is enabled later")
	_, _ = fmt.Fprintf(out, "  Runtime:   %s\n", describeSetupRuntimeMode(runtimeMode, goos))
	_, _ = fmt.Fprintf(out, "  Config path: %s\n", bootstrap.ConfigPath)
}

func printSetupSuccess(
	out io.Writer,
	result setup.CompleteResult,
	prepared setup.PreparedDatabase,
	installedService *installedSetupService,
) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "OpenASE setup completed.")
	_, _ = fmt.Fprintf(out, "Config: %s\n", result.ConfigPath)
	_, _ = fmt.Fprintf(out, "Env:    %s\n", result.EnvPath)
	_, _ = fmt.Fprintf(out, "Org:    %s (%s)\n", result.OrganizationName, result.OrganizationSlug)
	_, _ = fmt.Fprintf(out, "Project:%s (%s)\n", result.ProjectName, result.ProjectSlug)
	_, _ = fmt.Fprintf(
		out,
		"Database:%s %s:%d/%s (user=%s)\n",
		prepared.Source,
		prepared.Config.Host,
		prepared.Config.Port,
		prepared.Config.Name,
		prepared.Config.User,
	)
	_, _ = fmt.Fprintln(out, "Browser auth: local bootstrap link")
	if prepared.Docker != nil {
		_, _ = fmt.Fprintf(out, "Docker container: %s\n", prepared.Docker.ContainerName)
		_, _ = fmt.Fprintf(out, "Docker volume:    %s\n", prepared.Docker.VolumeName)
		_, _ = fmt.Fprintf(out, "Stop container:   docker stop %s\n", prepared.Docker.ContainerName)
		_, _ = fmt.Fprintf(out, "Remove container: docker rm -f %s\n", prepared.Docker.ContainerName)
	}
	if installedService != nil {
		_, _ = fmt.Fprintf(out, "Service:  %s via %s\n", installedService.Name, installedService.Platform)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Browser:")
	_, _ = fmt.Fprintf(out, "  Open %s\n", defaultSetupURL)
	_, _ = fmt.Fprintln(out, "  OpenASE no longer grants anonymous browser admin access.")
	_, _ = fmt.Fprintln(out, "  Generate a one-time local bootstrap link before signing in:")
	_, _ = fmt.Fprintln(out, "    openase auth bootstrap create-link --return-to / --format text")
	_, _ = fmt.Fprintln(out, "  Break-glass repair path if OIDC is misconfigured later:")
	_, _ = fmt.Fprintln(out, "    openase auth break-glass disable-oidc")
	_, _ = fmt.Fprintln(out, "    openase auth bootstrap create-link --return-to /admin/auth --format text")
	_, _ = fmt.Fprintf(out, "Next: openase doctor --config %s\n", result.ConfigPath)
	if installedService != nil {
		printManagedServiceSuccessHints(out, installedService)
	} else {
		_, _ = fmt.Fprintf(out, "Next: openase all-in-one --config %s\n", result.ConfigPath)
	}
}

func printSetupFailure(out io.Writer, configPath string) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Setup did not complete cleanly.")
	_, _ = fmt.Fprintf(out, "Next: openase doctor --config %s\n", configPath)
}

func newSetupPrompter(in io.Reader, out io.Writer) *setupPrompter {
	prompter := &setupPrompter{
		reader: bufio.NewReader(in),
		out:    out,
	}
	if file, ok := in.(*os.File); ok {
		prompter.inFile = file
	}
	return prompter
}

func (p *setupPrompter) choose(label string, options []string, defaultIndex int) (int, error) {
	_, _ = fmt.Fprintln(p.out)
	_, _ = fmt.Fprintln(p.out, label+":")
	for index, option := range options {
		marker := " "
		if index == defaultIndex {
			marker = "*"
		}
		_, _ = fmt.Fprintf(p.out, "  %s %d. %s\n", marker, index+1, option)
	}

	for {
		response, err := p.line(fmt.Sprintf("Enter choice [%d]", defaultIndex+1))
		if err != nil {
			return 0, err
		}
		if response == "" {
			return defaultIndex, nil
		}
		value, err := strconv.Atoi(response)
		if err == nil && value >= 1 && value <= len(options) {
			return value - 1, nil
		}
		_, _ = fmt.Fprintln(p.out, "Please enter a valid option number.")
	}
}

func (p *setupPrompter) stringValue(label string, defaultValue string) (string, error) {
	for {
		response, err := p.lineWithDefault(label, defaultValue)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(response) != "" {
			return strings.TrimSpace(response), nil
		}
		_, _ = fmt.Fprintln(p.out, "Value must not be empty.")
	}
}

func (p *setupPrompter) intValue(label string, defaultValue int) (int, error) {
	for {
		response, err := p.line(fmt.Sprintf("%s [%d]", label, defaultValue))
		if err != nil {
			return 0, err
		}
		if response == "" {
			return defaultValue, nil
		}
		value, err := strconv.Atoi(strings.TrimSpace(response))
		if err == nil {
			return value, nil
		}
		_, _ = fmt.Fprintln(p.out, "Please enter a valid integer.")
	}
}

func (p *setupPrompter) secretValue(label string, currentValue string) (string, error) {
	prompt := label
	if currentValue != "" {
		prompt += " [press Enter to keep existing value]"
	}
	if fd, ok := terminalFileDescriptor(p.inFile); ok && term.IsTerminal(fd) {
		_, _ = fmt.Fprintf(p.out, "%s: ", prompt)
		value, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(p.out)
		if err != nil {
			return "", err
		}
		trimmed := strings.TrimSpace(string(value))
		if trimmed == "" {
			return currentValue, nil
		}
		return trimmed, nil
	}

	response, err := p.line(prompt)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(response) == "" {
		return currentValue, nil
	}
	return strings.TrimSpace(response), nil
}

func (p *setupPrompter) selectValue(label string, values []string, defaultValue string) (string, error) {
	defaultIndex := 0
	for index, value := range values {
		if value == defaultValue {
			defaultIndex = index
			break
		}
	}
	index, err := p.choose(label, values, defaultIndex)
	if err != nil {
		return "", err
	}
	return values[index], nil
}

func (p *setupPrompter) csvValue(label string, defaultValue string) (string, error) {
	return p.lineWithDefault(label, defaultValue)
}

func (p *setupPrompter) confirm(label string, defaultValue bool) (bool, error) {
	suffix := "y/N"
	if defaultValue {
		suffix = "Y/n"
	}

	for {
		response, err := p.line(fmt.Sprintf("%s [%s]", label, suffix))
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(response)) {
		case "":
			return defaultValue, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			_, _ = fmt.Fprintln(p.out, "Please answer yes or no.")
		}
	}
}

func (p *setupPrompter) lineWithDefault(label string, defaultValue string) (string, error) {
	response, err := p.line(fmt.Sprintf("%s [%s]", label, defaultValue))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(response) == "" {
		return defaultValue, nil
	}
	return strings.TrimSpace(response), nil
}

func (p *setupPrompter) line(prompt string) (string, error) {
	_, _ = fmt.Fprintf(p.out, "%s: ", prompt)
	value, err := p.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func terminalFileDescriptor(file *os.File) (int, bool) {
	if file == nil {
		return 0, false
	}

	fd := file.Fd()
	maxInt := uintptr(^uint(0) >> 1)
	if fd > maxInt {
		return 0, false
	}

	return int(fd), true
}

func promptRuntimeMode(ctx context.Context, prompter *setupPrompter, deps setupFlowDeps) (setupRuntimeMode, error) {
	goos := deps.targetGOOS()
	options := []string{
		"Only Write Config: setup writes ~/.openase files and you start openase manually later.",
		setupManagedUserServicePrompt(goos),
	}
	index, err := prompter.choose("Choose how OpenASE should run after setup", options, 0)
	if err != nil {
		return "", err
	}
	if index == 0 {
		return setupRuntimeModeConfigOnly, nil
	}

	if err := deps.checkManagedUserServiceSupport(ctx); err != nil {
		_, _ = fmt.Fprintf(
			prompter.out,
			"\nCurrent machine cannot use the managed OpenASE user service via %s: %v\n",
			setupManagedUserServicePlatformName(goos),
			err,
		)
		fallback, confirmErr := prompter.confirm("Continue with config-only setup instead?", true)
		if confirmErr != nil {
			return "", confirmErr
		}
		if fallback {
			return setupRuntimeModeConfigOnly, nil
		}
		return "", errSetupAborted
	}

	return setupRuntimeModeManagedUserService, nil
}

func installSetupManagedService(
	ctx context.Context,
	deps setupFlowDeps,
	configPath string,
) (*installedSetupService, error) {
	manager, err := deps.buildUserServiceManager()
	if err != nil {
		return nil, fmt.Errorf("setup wrote config files, but failed to build the managed service installer: %w", err)
	}

	spec, err := deps.buildManagedServiceInstallSpec(configPath)
	if err != nil {
		return nil, fmt.Errorf("setup wrote config files, but failed to build the managed service definition: %w", err)
	}
	if err := manager.Apply(ctx, spec); err != nil {
		return nil, fmt.Errorf("setup wrote config files, but failed to install managed user service %q via %s: %w", spec.Name, manager.Platform(), err)
	}
	if err := deps.verifyManagedUserService(ctx, spec.Name); err != nil {
		return nil, fmt.Errorf("setup wrote config files, but %s could not verify service %q: %w", manager.Platform(), spec.Name, err)
	}

	return deps.installedSetupServiceBuilder()(ctx, manager.Platform(), spec), nil
}

func describeSetupRuntimeMode(mode setupRuntimeMode, goos string) string {
	switch mode {
	case setupRuntimeModeManagedUserService:
		return fmt.Sprintf("managed user service via %s", setupManagedUserServicePlatformName(goos))
	default:
		return "config-only"
	}
}

func (d setupFlowDeps) targetGOOS() string {
	if strings.TrimSpace(d.goos) == "" {
		return runtime.GOOS
	}

	return d.goos
}

func (d setupFlowDeps) installedSetupServiceBuilder() func(context.Context, string, provider.UserServiceInstallSpec) *installedSetupService {
	if d.buildInstalledSetupService == nil {
		return buildInstalledSetupService
	}

	return d.buildInstalledSetupService
}

func setupManagedUserServicePlatformName(goos string) string {
	switch goos {
	case "linux":
		return "systemd --user"
	case "darwin":
		return "launchd"
	default:
		return "managed user service"
	}
}

func setupManagedUserServicePrompt(goos string) string {
	switch goos {
	case "linux":
		return "Install Managed User Service (systemd --user): setup also installs the managed OpenASE service for this Linux user."
	case "darwin":
		return "Install Managed User Service (launchd): setup also installs the managed OpenASE service for this macOS user."
	default:
		return "Install Managed User Service: setup also installs the managed OpenASE service for this user when the platform supports it."
	}
}

func checkManagedUserServiceSupport(ctx context.Context) error {
	switch runtime.GOOS {
	case "linux":
		return checkSystemdUserServiceSupport(ctx)
	case "darwin":
		return checkLaunchdUserServiceSupport(ctx)
	default:
		return fmt.Errorf("managed user services are not supported on %s", runtime.GOOS)
	}
}

func verifyManagedUserService(ctx context.Context, name provider.ServiceName) error {
	switch runtime.GOOS {
	case "linux":
		return verifySystemdUserService(ctx, name)
	case "darwin":
		return verifyLaunchdUserService(ctx, name)
	default:
		return fmt.Errorf("managed user services are not supported on %s", runtime.GOOS)
	}
}

func buildInstalledSetupService(ctx context.Context, platform string, spec provider.UserServiceInstallSpec) *installedSetupService {
	service := &installedSetupService{
		Name:        spec.Name,
		Platform:    platform,
		InstallSpec: spec,
	}
	if platform == "launchd" {
		homeDir := filepath.Dir(spec.WorkingDirectory.String())
		service.LaunchdPlist = launchdPlistPath(homeDir, spec.Name)
		ref, err := userserviceinfra.ResolveLaunchdService(ctx, os.Getuid(), spec.Name)
		if err == nil {
			service.LaunchdTarget = ref.Target
		}
	}

	return service
}

func printManagedServiceSuccessHints(out io.Writer, installedService *installedSetupService) {
	switch installedService.Platform {
	case "launchd":
		_, _ = fmt.Fprintf(out, "Next: launchctl print %s\n", installedService.LaunchdTarget)
		_, _ = fmt.Fprintf(out, "Plist: %s\n", installedService.LaunchdPlist)
		_, _ = fmt.Fprintf(out, "Stdout log: %s\n", installedService.InstallSpec.StdoutPath)
		_, _ = fmt.Fprintf(out, "Stderr log: %s\n", installedService.InstallSpec.StderrPath)
		_, _ = fmt.Fprintf(out, "Restart: launchctl kickstart -k %s\n", installedService.LaunchdTarget)
		_, _ = fmt.Fprintf(out, "Stop: launchctl bootout %s\n", installedService.LaunchdTarget)
	default:
		_, _ = fmt.Fprintf(out, "Next: systemctl --user status %s\n", installedService.Name)
		_, _ = fmt.Fprintf(out, "Logs: journalctl --user -u %s -n 200 -f\n", installedService.Name)
		_, _ = fmt.Fprintf(out, "Restart: systemctl --user restart %s\n", installedService.Name)
		_, _ = fmt.Fprintf(out, "Stop: systemctl --user stop %s\n", installedService.Name)
	}
}

func checkSystemdUserServiceSupport(ctx context.Context) error {
	return runSystemctlUser(ctx, "show-environment")
}

func verifySystemdUserService(ctx context.Context, name provider.ServiceName) error {
	if err := runSystemctlUser(ctx, "is-enabled", name.String()); err != nil {
		return err
	}
	if err := runSystemctlUser(ctx, "is-active", name.String()); err != nil {
		return err
	}
	return nil
}

func checkLaunchdUserServiceSupport(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve user home directory for launchd LaunchAgents: %w", err)
	}

	if _, err := userserviceinfra.CheckLaunchdSupport(ctx, homeDir, os.Getuid()); err != nil {
		return err
	}

	return nil
}

func verifyLaunchdUserService(ctx context.Context, name provider.ServiceName) error {
	ref, err := userserviceinfra.ResolveLaunchdService(ctx, os.Getuid(), name)
	if err != nil {
		return err
	}
	if !ref.Loaded {
		return fmt.Errorf("launchd could not find the managed OpenASE service in %s", ref.Domain)
	}

	return nil
}

func launchdServiceLabel(name provider.ServiceName) string {
	return "com." + name.String()
}

func launchdPlistPath(homeDir string, name provider.ServiceName) string {
	return filepath.Join(homeDir, "Library", "LaunchAgents", launchdServiceLabel(name)+".plist")
}

func runSystemctlUser(ctx context.Context, args ...string) error {
	path, err := exec.LookPath("systemctl")
	if err != nil {
		return fmt.Errorf("systemctl is not installed")
	}

	//nolint:gosec // setup intentionally shells out to the local systemd CLI for capability checks
	command := exec.CommandContext(ctx, path, append([]string{"--user"}, args...)...)
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}

	normalized := strings.ToLower(strings.TrimSpace(string(output)))
	if normalized == "" {
		normalized = strings.ToLower(err.Error())
	}
	switch {
	case strings.Contains(normalized, "failed to connect to bus"),
		strings.Contains(normalized, "no medium found"),
		strings.Contains(normalized, "operation not permitted"),
		strings.Contains(normalized, "access denied"):
		return fmt.Errorf("systemd --user is not available for this login session; choose config-only or start a user systemd session")
	case strings.Contains(normalized, "not found"):
		return fmt.Errorf("systemd --user could not find the managed OpenASE service")
	default:
		return fmt.Errorf("systemctl --user %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
}
