package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/setup"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	defaultSetupHost = "127.0.0.1"
	defaultSetupPort = 19836
)

var errSetupAborted = errors.New("setup aborted")

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
	var host string
	var port int
	var web bool
	var force bool

	command := &cobra.Command{
		Use:   "setup",
		Short: "Run interactive local setup for OpenASE.",
		Long: strings.TrimSpace(`
Run the interactive local setup flow for OpenASE.

The default flow stays inside the terminal, prepares a PostgreSQL connection,
checks key local CLIs, and writes a runnable ~/.openase/config.yaml plus
~/.openase/.env. Use --web only for legacy browser-based troubleshooting.
`),
		Example: "openase setup\nopenase setup --force\nopenase setup --web --host 127.0.0.1 --port 19836",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if web {
				return runSetupWebWizard(cmd.Context(), cmd.OutOrStdout(), host, port)
			}
			return runSetupFlowCommand(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), setupFlowOptions{
				allowOverwrite: force,
			})
		},
	}

	command.Flags().BoolVar(&force, "force", false, "Overwrite an existing ~/.openase/config.yaml without prompting.")
	command.Flags().BoolVar(&web, "web", false, "Use the legacy browser-based setup flow.")
	command.Flags().StringVar(&host, "host", defaultSetupHost, "HTTP listen host for --web mode.")
	command.Flags().IntVar(&port, "port", defaultSetupPort, "HTTP listen port for --web mode.")
	_ = command.Flags().MarkHidden("web")
	_ = command.Flags().MarkHidden("host")
	_ = command.Flags().MarkHidden("port")

	return command
}

func runDefaultSetupWizard(ctx context.Context, out io.Writer) error {
	return runSetupFlowCommand(ctx, os.Stdin, out, setupFlowOptions{})
}

func runSetupFlowCommand(ctx context.Context, in io.Reader, out io.Writer, opts setupFlowOptions) error {
	service, err := setup.NewService(setup.Options{})
	if err != nil {
		return err
	}
	return runSetupFlow(ctx, in, out, service, opts)
}

func runSetupFlow(ctx context.Context, in io.Reader, out io.Writer, service setupFlowService, opts setupFlowOptions) error {
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
	printSetupSummary(out, bootstrap, prepared)

	confirmed, err := prompter.confirm("Write ~/.openase/config.yaml and ~/.openase/.env now?", true)
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

	printSetupSuccess(out, result, prepared)
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
	_, _ = fmt.Fprintln(out, "  3. Write ~/.openase/config.yaml and ~/.openase/.env")
	_, _ = fmt.Fprintln(out, "  4. Initialize the default local workspace metadata")
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

func printSetupSummary(out io.Writer, bootstrap setup.Bootstrap, prepared setup.PreparedDatabase) {
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
	_, _ = fmt.Fprintf(out, "  Config path: %s\n", bootstrap.ConfigPath)
}

func printSetupSuccess(out io.Writer, result setup.CompleteResult, prepared setup.PreparedDatabase) {
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
	if prepared.Docker != nil {
		_, _ = fmt.Fprintf(out, "Docker container: %s\n", prepared.Docker.ContainerName)
		_, _ = fmt.Fprintf(out, "Docker volume:    %s\n", prepared.Docker.VolumeName)
		_, _ = fmt.Fprintf(out, "Stop container:   docker stop %s\n", prepared.Docker.ContainerName)
		_, _ = fmt.Fprintf(out, "Remove container: docker rm -f %s\n", prepared.Docker.ContainerName)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintf(out, "Next: openase doctor --config %s\n", result.ConfigPath)
	_, _ = fmt.Fprintf(out, "Next: openase all-in-one --config %s\n", result.ConfigPath)
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

func runSetupWebWizard(ctx context.Context, out io.Writer, host string, port int) error {
	service, err := setup.NewService(setup.Options{})
	if err != nil {
		return err
	}

	server := setup.NewServer(setup.ServerOptions{
		Host:    host,
		Port:    port,
		Service: service,
	})
	address := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/setup"

	printSetupWebBanner(out, address)
	_ = openBrowser(address)

	return server.Run(ctx)
}

func printSetupWebBanner(out io.Writer, address string) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  OpenASE legacy web setup")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintf(out, "  Browser flow: %s\n", address)
	_, _ = fmt.Fprintln(out, "  This path is deprecated; prefer `openase setup` without --web.")
	_, _ = fmt.Fprintln(out)
}

func openBrowser(url string) error {
	var name string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		name = "open"
		args = []string{url}
	default:
		name = "xdg-open"
		args = []string{url}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	//nolint:gosec // setup intentionally launches the platform-specific browser opener
	return exec.CommandContext(ctx, name, args...).Start()
}
