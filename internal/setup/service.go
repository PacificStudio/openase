package setup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/envfile"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	"github.com/BetterAndBetterII/openase/internal/localdiag"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

var ErrSetupAlreadyCompleted = errors.New("setup already completed")

type DatabaseConnector interface {
	Ping(context.Context, string) error
	Migrate(context.Context, string) error
}

type DockerRunner interface {
	Run(context.Context, string, ...string) (string, error)
}

type Installer interface {
	Initialize(context.Context, InstallInput) error
}

type OrganizationConfig struct {
	Name string
	Slug string
}

type ProjectConfig struct {
	Name string
	Slug string
}

type InstallInput struct {
	Database     DatabaseConfig
	Agents       []AgentOption
	Organization OrganizationConfig
	Project      ProjectConfig
}

type Options struct {
	HomeDir      string
	ConfigPath   string
	Resolver     provider.ExecutableResolver
	Connector    DatabaseConnector
	Installer    Installer
	DockerRunner DockerRunner
	RunCommand   func(context.Context, string, ...string) (string, error)
}

type Service struct {
	homeDir            string
	configPathOverride string
	resolver           provider.ExecutableResolver
	connector          DatabaseConnector
	installer          Installer
	dockerRunner       DockerRunner
	runCommand         func(context.Context, string, ...string) (string, error)
	logger             *slog.Logger
}

type runtimeDatabaseConnector struct{}

type execDockerRunner struct{}

type defaultInstaller struct{}

func (runtimeDatabaseConnector) Ping(ctx context.Context, dsn string) (err error) {
	client, err := openEntClient(dsn)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, client.Close())
	}()

	return client.Schema.Create(ctx)
}

func (runtimeDatabaseConnector) Migrate(ctx context.Context, dsn string) error {
	client, err := database.Open(ctx, dsn)
	if err != nil {
		return err
	}
	return client.Close()
}

func (execDockerRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	//nolint:gosec // setup intentionally invokes the local docker CLI for environment bootstrap
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func NewService(opts Options) (*Service, error) {
	homeDir := strings.TrimSpace(opts.HomeDir)
	if homeDir == "" {
		resolved, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve user home directory: %w", err)
		}
		homeDir = resolved
	}

	resolver := opts.Resolver
	if resolver == nil {
		resolver = executable.NewPathResolver()
	}

	connector := opts.Connector
	if connector == nil {
		connector = runtimeDatabaseConnector{}
	}

	installer := opts.Installer
	if installer == nil {
		installer = defaultInstaller{}
	}

	dockerRunner := opts.DockerRunner
	if dockerRunner == nil {
		dockerRunner = execDockerRunner{}
	}

	runCommand := opts.RunCommand
	configPath := strings.TrimSpace(opts.ConfigPath)

	return &Service{
		homeDir:            homeDir,
		configPathOverride: configPath,
		resolver:           resolver,
		connector:          connector,
		installer:          installer,
		dockerRunner:       dockerRunner,
		runCommand:         runCommand,
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, nil
}

func (s *Service) Bootstrap(ctx context.Context) (Bootstrap, error) {
	configPath, exists := s.existingConfigPath()
	cliDiagnostics := s.detectCLIDiagnostics(ctx)
	agentOptions := detectAgentOptions(cliDiagnostics)

	return Bootstrap{
		ConfigExists: exists,
		ConfigPath:   configPath,
		Sources: []DatabaseSourceOption{
			{
				ID:          DatabaseSourceDocker,
				Name:        "Use Docker To Start Local PostgreSQL",
				Description: "Fastest path: create a local PostgreSQL 16 container and validate it automatically.",
			},
			{
				ID:          DatabaseSourceManual,
				Name:        "Enter Existing PostgreSQL Connection",
				Description: "Provide host, port, database, user, password, and sslmode for an existing database.",
			},
		},
		Agents: agentOptions,
		CLI:    cliDiagnostics,
		Defaults: Defaults{
			ManualDatabase: defaultRawDatabaseInput(),
			DockerDatabase: defaultRawDockerDatabaseInput(),
		},
	}, nil
}

func (s *Service) DesktopPreflight(ctx context.Context) (DesktopPreflightResult, error) {
	result := DesktopPreflightResult{
		Ready:          false,
		ConfigPath:     s.configPathResolved(),
		OpenASEHomeDir: s.openaseHomeDir(),
	}

	if err := s.ensureHomeLayout(); err != nil {
		result.Issues = append(result.Issues, DesktopIssue{
			Code:    DesktopIssueDirectoryUnavailable,
			Title:   "OpenASE data directory is unavailable",
			Message: err.Error(),
			Action:  "Check directory permissions, then retry the desktop setup.",
		})
		return result, nil
	}

	if !fileExists(result.ConfigPath) {
		result.Issues = append(result.Issues, DesktopIssue{
			Code:    DesktopIssueConfigMissing,
			Title:   "OpenASE config is missing",
			Message: fmt.Sprintf("No config file was found at %s.", result.ConfigPath),
			Action:  "Choose an existing PostgreSQL connection or let Docker prepare PostgreSQL for you.",
		})
		return result, nil
	}

	loaded, err := config.Load(config.LoadOptions{ConfigFile: result.ConfigPath})
	if err != nil {
		result.Issues = append(result.Issues, DesktopIssue{
			Code:    DesktopIssueConfigInvalid,
			Title:   "OpenASE config could not be loaded",
			Message: err.Error(),
			Action:  "Update the database settings from the desktop setup flow and write a fresh config.",
		})
		return result, nil
	}

	if err := s.connector.Ping(ctx, loaded.Database.DSN); err != nil {
		result.Issues = append(result.Issues, classifyDesktopSetupIssue(err))
		return result, nil
	}

	result.Ready = true
	return result, nil
}

func (s *Service) DesktopApply(ctx context.Context, raw RawDesktopApplyRequest) (DesktopApplyResult, error) {
	result := DesktopApplyResult{
		Ready:          false,
		ConfigPath:     s.configPathResolved(),
		OpenASEHomeDir: s.openaseHomeDir(),
	}

	source, err := parseDatabaseSourceInput(raw.Database)
	if err != nil {
		result.Issues = append(result.Issues, DesktopIssue{
			Code:    DesktopIssueInputInvalid,
			Title:   "Setup input is incomplete",
			Message: err.Error(),
			Action:  "Review the database fields and try again.",
		})
		return result, nil
	}

	prepared, err := s.PrepareDatabase(ctx, raw.Database)
	if err != nil {
		result.DatabaseSource = source.Type
		result.Issues = append(result.Issues, classifyDesktopSetupIssue(err))
		return result, nil
	}

	completed, err := s.Complete(ctx, RawCompleteRequest{
		Database:       prepared.Config.Raw(),
		AllowOverwrite: raw.AllowOverwrite,
	})
	if err != nil {
		result.DatabaseSource = prepared.Source
		if prepared.Docker != nil {
			result.Docker = prepared.Docker
		}
		result.Issues = append(result.Issues, classifyDesktopSetupIssue(err))
		return result, nil
	}

	result.Ready = true
	result.ConfigPath = completed.ConfigPath
	result.EnvPath = completed.EnvPath
	result.DatabaseSource = prepared.Source
	result.Docker = prepared.Docker
	return result, nil
}

func (s *Service) PrepareDatabase(ctx context.Context, raw RawDatabaseSourceInput) (PreparedDatabase, error) {
	source, err := parseDatabaseSourceInput(raw)
	if err != nil {
		return PreparedDatabase{}, err
	}

	switch source.Type {
	case DatabaseSourceManual:
		if source.Manual == nil {
			return PreparedDatabase{}, fmt.Errorf("database.manual must not be empty")
		}
		if err := s.connector.Ping(ctx, source.Manual.DSN()); err != nil {
			return PreparedDatabase{}, fmt.Errorf("test database connection: %w", err)
		}
		return PreparedDatabase{
			Source: DatabaseSourceManual,
			Config: *source.Manual,
		}, nil
	case DatabaseSourceDocker:
		if source.Docker == nil {
			return PreparedDatabase{}, fmt.Errorf("database.docker must not be empty")
		}
		return s.prepareDockerDatabase(ctx, *source.Docker)
	default:
		return PreparedDatabase{}, fmt.Errorf("unsupported database source %q", source.Type)
	}
}

func (s *Service) Complete(ctx context.Context, raw RawCompleteRequest) (CompleteResult, error) {
	request, err := parseCompleteRequest(raw)
	if err != nil {
		return CompleteResult{}, err
	}

	if _, exists := s.existingConfigPath(); exists && !request.AllowOverwrite {
		return CompleteResult{}, ErrSetupAlreadyCompleted
	}

	if err := s.connector.Ping(ctx, request.Database.DSN()); err != nil {
		return CompleteResult{}, fmt.Errorf("test database connection: %w", err)
	}
	if err := s.connector.Migrate(ctx, request.Database.DSN()); err != nil {
		return CompleteResult{}, fmt.Errorf("migrate database schema: %w", err)
	}

	bootstrap, err := s.Bootstrap(ctx)
	if err != nil {
		return CompleteResult{}, err
	}
	selectedAgents := selectedAvailableAgents(bootstrap.Agents)

	if err := s.installer.Initialize(ctx, InstallInput{
		Database: request.Database,
		Agents:   selectedAgents,
		Organization: OrganizationConfig{
			Name: DefaultOrganizationName,
			Slug: DefaultOrganizationSlug,
		},
		Project: ProjectConfig{
			Name: DefaultProjectName,
			Slug: DefaultProjectSlug,
		},
	}); err != nil {
		return CompleteResult{}, err
	}

	if err := s.ensureHomeLayout(); err != nil {
		return CompleteResult{}, err
	}

	authToken, err := generateAuthToken()
	if err != nil {
		return CompleteResult{}, fmt.Errorf("generate auth token: %w", err)
	}

	if err := s.writeConfigFile(request, selectedAgents); err != nil {
		return CompleteResult{}, err
	}
	if err := s.writeEnvFile(authToken); err != nil {
		return CompleteResult{}, err
	}

	return CompleteResult{
		ConfigPath:       s.configPath(),
		EnvPath:          s.envPath(),
		OrganizationName: DefaultOrganizationName,
		OrganizationSlug: DefaultOrganizationSlug,
		ProjectName:      DefaultProjectName,
		ProjectSlug:      DefaultProjectSlug,
	}, nil
}

func (s *Service) ensureHomeLayout() error {
	baseDir := s.openaseHomeDir()
	for _, dir := range []string{
		baseDir,
		filepath.Join(baseDir, "logs"),
		filepath.Join(baseDir, "workspaces"),
		filepath.Dir(s.configPathResolved()),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	return nil
}

func (s *Service) writeConfigFile(request CompleteRequest, agents []AgentOption) error {
	type serverConfig struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
		Mode string `yaml:"mode"`
	}
	type databaseConfig struct {
		DSN string `yaml:"dsn"`
	}
	type orchestratorConfig struct {
		TickInterval string `yaml:"tick_interval"`
	}
	type eventConfig struct {
		Driver string `yaml:"driver"`
	}
	type observabilityMetricsExportConfig struct {
		Prometheus   bool   `yaml:"prometheus"`
		OTLPEndpoint string `yaml:"otlp_endpoint"`
	}
	type observabilityMetricsConfig struct {
		Enabled bool                             `yaml:"enabled"`
		Export  observabilityMetricsExportConfig `yaml:"export"`
	}
	type observabilityTracingConfig struct {
		Enabled     bool    `yaml:"enabled"`
		Endpoint    string  `yaml:"endpoint"`
		ServiceName string  `yaml:"service_name"`
		SampleRatio float64 `yaml:"sample_ratio"`
	}
	type observabilityConfig struct {
		Metrics observabilityMetricsConfig `yaml:"metrics"`
		Tracing observabilityTracingConfig `yaml:"tracing"`
	}
	type logConfig struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	}
	type setupConfig struct {
		OrganizationName string   `yaml:"organization_name"`
		OrganizationSlug string   `yaml:"organization_slug"`
		ProjectName      string   `yaml:"project_name"`
		ProjectSlug      string   `yaml:"project_slug"`
		Agents           []string `yaml:"agents,omitempty"`
	}
	payload := struct {
		Server        serverConfig        `yaml:"server"`
		Database      databaseConfig      `yaml:"database"`
		Orchestrator  orchestratorConfig  `yaml:"orchestrator"`
		Event         eventConfig         `yaml:"event"`
		Observability observabilityConfig `yaml:"observability"`
		Log           logConfig           `yaml:"log"`
		Setup         setupConfig         `yaml:"setup"`
	}{
		Server: serverConfig{
			Host: "127.0.0.1",
			Port: 19836,
			Mode: "all-in-one",
		},
		Database: databaseConfig{
			DSN: request.Database.DSN(),
		},
		Orchestrator: orchestratorConfig{TickInterval: "5s"},
		Event:        eventConfig{Driver: "auto"},
		Observability: observabilityConfig{
			Metrics: observabilityMetricsConfig{
				Enabled: true,
				Export: observabilityMetricsExportConfig{
					Prometheus:   false,
					OTLPEndpoint: "",
				},
			},
			Tracing: observabilityTracingConfig{
				Enabled:     false,
				Endpoint:    "",
				ServiceName: "openase",
				SampleRatio: 1.0,
			},
		},
		Log: logConfig{Level: "info", Format: "text"},
		Setup: setupConfig{
			OrganizationName: DefaultOrganizationName,
			OrganizationSlug: DefaultOrganizationSlug,
			ProjectName:      DefaultProjectName,
			ProjectSlug:      DefaultProjectSlug,
			Agents:           agentIDs(agents),
		},
	}

	content, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal setup config: %w", err)
	}

	if err := os.WriteFile(s.configPath(), content, 0o600); err != nil {
		return fmt.Errorf("write setup config: %w", err)
	}

	return nil
}

func (s *Service) writeEnvFile(authToken string) error {
	updates := map[string]string{
		"OPENASE_AUTH_TOKEN": authToken,
	}
	if normalizedPath := envfile.NormalizePath(os.Getenv("PATH")); normalizedPath != "" {
		updates["PATH"] = normalizedPath
	}
	if err := envfile.Upsert(s.envPath(), updates); err != nil {
		return fmt.Errorf("write setup env file: %w", err)
	}

	return nil
}

func (s *Service) openaseHomeDir() string {
	return filepath.Join(s.homeDir, ".openase")
}

func (s *Service) configPathResolved() string {
	if strings.TrimSpace(s.configPathOverride) != "" {
		return s.configPathOverride
	}
	return filepath.Join(s.openaseHomeDir(), "config.yaml")
}

func (s *Service) configPath() string {
	return s.configPathResolved()
}

func (s *Service) envPath() string {
	return filepath.Join(s.openaseHomeDir(), ".env")
}

func (s *Service) existingConfigPath() (string, bool) {
	if fileExists(s.configPathResolved()) {
		return s.configPathResolved(), true
	}
	return s.configPathResolved(), false
}

func classifyDesktopSetupIssue(err error) DesktopIssue {
	if err == nil {
		return DesktopIssue{
			Code:    DesktopIssueSetupFailed,
			Title:   "Setup failed",
			Message: "unknown setup error",
			Action:  "Retry the desktop setup.",
		}
	}

	normalized := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, ErrSetupAlreadyCompleted):
		return DesktopIssue{
			Code:    DesktopIssueSetupFailed,
			Title:   "Config already exists",
			Message: err.Error(),
			Action:  "Retry with overwrite enabled or update the existing config manually.",
		}
	case strings.Contains(normalized, "password authentication failed"),
		strings.Contains(normalized, "authentication failed"),
		strings.Contains(normalized, "role") && strings.Contains(normalized, "does not exist"):
		return DesktopIssue{
			Code:    DesktopIssueDatabaseAuthFailed,
			Title:   "PostgreSQL rejected the credentials",
			Message: err.Error(),
			Action:  "Update the username or password and retry the connection test.",
		}
	case strings.Contains(normalized, "docker is not installed"),
		strings.Contains(normalized, "docker daemon is unavailable"),
		strings.Contains(normalized, "docker daemon is not reachable"),
		strings.Contains(normalized, "docker permission denied"):
		return DesktopIssue{
			Code:    DesktopIssueDockerUnavailable,
			Title:   "Docker is unavailable",
			Message: err.Error(),
			Action:  "Install Docker, start the daemon, or switch to an existing PostgreSQL connection.",
		}
	case strings.Contains(normalized, "port 127.0.0.1"),
		strings.Contains(normalized, "port is already allocated"):
		return DesktopIssue{
			Code:    DesktopIssuePortConflict,
			Title:   "PostgreSQL port is already in use",
			Message: err.Error(),
			Action:  "Free the port or pick the existing PostgreSQL path.",
		}
	case strings.Contains(normalized, "not ready within"),
		strings.Contains(normalized, "timed out"):
		return DesktopIssue{
			Code:    DesktopIssueSetupTimeout,
			Title:   "Setup timed out",
			Message: err.Error(),
			Action:  "Retry the setup and inspect the logs if the timeout repeats.",
		}
	case strings.Contains(normalized, "connection refused"),
		strings.Contains(normalized, "dial tcp"),
		strings.Contains(normalized, "no such host"),
		strings.Contains(normalized, "i/o timeout"),
		strings.Contains(normalized, "network is unreachable"),
		strings.Contains(normalized, "test database connection"):
		return DesktopIssue{
			Code:    DesktopIssueDatabaseUnreachable,
			Title:   "PostgreSQL could not be reached",
			Message: err.Error(),
			Action:  "Verify the host, port, database name, and that PostgreSQL is running.",
		}
	default:
		return DesktopIssue{
			Code:    DesktopIssueSetupFailed,
			Title:   "Setup failed",
			Message: err.Error(),
			Action:  "Retry the setup or inspect the logs for more detail.",
		}
	}
}

func (s *Service) detectCLIDiagnostics(ctx context.Context) []CLIDiagnostic {
	reports := localdiag.Inspect(ctx, localdiag.SetupCommandSpecs(), localdiag.Options{
		LookPath:   s.resolver.LookPath,
		RunCommand: s.runCommand,
	})

	diagnostics := make([]CLIDiagnostic, 0, len(reports))
	for _, report := range reports {
		item := CLIDiagnostic{
			ID:      report.ID,
			Name:    report.Name,
			Command: report.Command,
			Path:    report.Path,
			Version: report.Version,
		}
		switch report.Status {
		case localdiag.StatusReady:
			item.Status = "ready"
		case localdiag.StatusVersionError:
			item.Status = "version_error"
			item.Message = "Found on PATH, but version detection failed."
		default:
			item.Status = "missing"
			item.Message = "Not found on PATH."
		}
		diagnostics = append(diagnostics, item)
	}

	return diagnostics
}

func detectAgentOptions(diagnostics []CLIDiagnostic) []AgentOption {
	diagnosticByCommand := make(map[string]CLIDiagnostic, len(diagnostics))
	for _, diagnostic := range diagnostics {
		diagnosticByCommand[diagnostic.Command] = diagnostic
	}

	templates := catalogdomain.BuiltinAgentProviderTemplates()
	options := make([]AgentOption, 0, len(templates))
	for _, template := range templates {
		option := AgentOption{
			ID:          template.ID,
			Name:        template.Name,
			Command:     template.Command,
			AdapterType: template.AdapterType,
			ModelName:   template.ModelName,
		}
		if diagnostic, ok := diagnosticByCommand[template.Command]; ok {
			option.Available = diagnostic.Status == "ready"
			option.Path = diagnostic.Path
			option.Version = diagnostic.Version
		}
		options = append(options, option)
	}

	return options
}

func selectedAvailableAgents(options []AgentOption) []AgentOption {
	selected := make([]AgentOption, 0, len(options))
	for _, option := range options {
		if option.Available {
			selected = append(selected, option)
		}
	}
	return selected
}

func agentIDs(options []AgentOption) []string {
	ids := make([]string, 0, len(options))
	for _, option := range options {
		ids = append(ids, option.ID)
	}
	return ids
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func generateAuthToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	return hex.EncodeToString(token), nil
}

func generateDatabasePassword() (string, error) {
	token := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	return hex.EncodeToString(token), nil
}

func (s *Service) prepareDockerDatabase(ctx context.Context, config DockerDatabaseConfig) (PreparedDatabase, error) {
	dockerPath, err := s.resolver.LookPath("docker")
	if err != nil {
		return PreparedDatabase{}, fmt.Errorf("docker is not installed or not available on PATH; install Docker and retry, or choose manual database setup")
	}

	if _, err := s.dockerRunner.Run(ctx, dockerPath, "info", "--format", "{{.ServerVersion}}"); err != nil {
		return PreparedDatabase{}, classifyDockerCommandError("docker daemon is unavailable", err)
	}

	if err := ensureTCPPortAvailable(config.Port); err != nil {
		return PreparedDatabase{}, err
	}

	exists, err := s.dockerContainerExists(ctx, dockerPath, config.ContainerName)
	if err != nil {
		return PreparedDatabase{}, err
	}
	if exists {
		return PreparedDatabase{}, fmt.Errorf(
			"docker container %q already exists; remove it, rename it, or choose manual database setup",
			config.ContainerName,
		)
	}

	password, err := generateDatabasePassword()
	if err != nil {
		return PreparedDatabase{}, fmt.Errorf("generate database password: %w", err)
	}

	if _, err := s.dockerRunner.Run(ctx, dockerPath, "volume", "create", config.VolumeName); err != nil {
		return PreparedDatabase{}, classifyDockerCommandError("create docker volume", err)
	}

	runArgs := []string{
		"run", "-d",
		"--name", config.ContainerName,
		"--restart", "unless-stopped",
		"-e", "POSTGRES_DB=" + config.DatabaseName,
		"-e", "POSTGRES_USER=" + config.User,
		"-e", "POSTGRES_PASSWORD=" + password,
		"-p", fmt.Sprintf("127.0.0.1:%d:5432", config.Port),
		"-v", fmt.Sprintf("%s:/var/lib/postgresql/data", config.VolumeName),
		config.Image,
	}
	if _, err := s.dockerRunner.Run(ctx, dockerPath, runArgs...); err != nil {
		return PreparedDatabase{}, classifyDockerRunError(config, err)
	}

	databaseConfig := DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     config.Port,
		Name:     config.DatabaseName,
		User:     config.User,
		Password: password,
		SSLMode:  "disable",
	}
	if err := s.waitForDatabase(ctx, databaseConfig, 60*time.Second); err != nil {
		return PreparedDatabase{}, err
	}

	return PreparedDatabase{
		Source: DatabaseSourceDocker,
		Config: databaseConfig,
		Docker: &DockerDatabaseRuntime{
			ContainerName: config.ContainerName,
			DatabaseName:  config.DatabaseName,
			User:          config.User,
			Port:          config.Port,
			VolumeName:    config.VolumeName,
			Image:         config.Image,
		},
	}, nil
}

func ensureTCPPortAvailable(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf(
			"docker PostgreSQL port 127.0.0.1:%d is already in use; stop the conflicting service or choose manual database setup",
			port,
		)
	}
	return listener.Close()
}

func (s *Service) dockerContainerExists(ctx context.Context, dockerPath string, containerName string) (bool, error) {
	output, err := s.dockerRunner.Run(
		ctx,
		dockerPath,
		"ps",
		"-a",
		"--filter",
		fmt.Sprintf("name=^/%s$", containerName),
		"--format",
		"{{.Names}}",
	)
	if err != nil {
		return false, classifyDockerCommandError("inspect docker containers", err)
	}

	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == containerName {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) waitForDatabase(ctx context.Context, config DatabaseConfig, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if err := s.connector.Ping(ctx, config.DSN()); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf(
				"postgres container started, but the database was not ready within %s; check docker logs for %q and retry",
				timeout,
				config.Name,
			)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func classifyDockerCommandError(prefix string, err error) error {
	if err == nil {
		return nil
	}

	normalized := strings.ToLower(err.Error())
	switch {
	case strings.Contains(normalized, "permission denied"):
		return fmt.Errorf("%s: Docker permission denied; ensure your user can access the docker daemon and retry", prefix)
	case strings.Contains(normalized, "cannot connect to the docker daemon"),
		strings.Contains(normalized, "is the docker daemon running"),
		strings.Contains(normalized, "error during connect"),
		strings.Contains(normalized, "docker daemon"),
		strings.Contains(normalized, "connection refused"):
		return fmt.Errorf("%s: Docker daemon is not reachable; start Docker and retry", prefix)
	default:
		return fmt.Errorf("%s: %w", prefix, err)
	}
}

func classifyDockerRunError(config DockerDatabaseConfig, err error) error {
	if err == nil {
		return nil
	}

	normalized := strings.ToLower(err.Error())
	switch {
	case strings.Contains(normalized, "port is already allocated"):
		return fmt.Errorf(
			"docker could not bind 127.0.0.1:%d because the port is already allocated; stop the conflicting service or choose manual database setup",
			config.Port,
		)
	case strings.Contains(normalized, "conflict"):
		return fmt.Errorf(
			"docker container %q already exists; remove it, rename it, or choose manual database setup",
			config.ContainerName,
		)
	case strings.Contains(normalized, "pull access denied"),
		strings.Contains(normalized, "manifest unknown"),
		strings.Contains(normalized, "not found"):
		return fmt.Errorf(
			"docker could not pull image %q; verify the image name or your registry access and retry",
			config.Image,
		)
	default:
		return classifyDockerCommandError("start docker postgres container", err)
	}
}

func (defaultInstaller) Initialize(ctx context.Context, input InstallInput) (err error) {
	client, err := database.Open(ctx, input.Database.DSN())
	if err != nil {
		return fmt.Errorf("open installation database: %w", err)
	}
	defer func() {
		err = errors.Join(err, client.Close())
	}()

	repo := catalogrepo.NewEntRepository(client)
	statusService := ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
	service := catalogservice.New(
		repo,
		executable.NewPathResolver(),
		nil,
		catalogservice.WithProjectStatusBootstrapper(catalogservice.ProjectStatusBootstrapperFunc(func(ctx context.Context, projectID uuid.UUID) error {
			_, err := statusService.ResetToDefaultTemplate(ctx, projectID)
			return err
		})),
	)

	orgCreate, err := catalogdomain.ParseCreateOrganization(catalogdomain.OrganizationInput{
		Name: input.Organization.Name,
		Slug: input.Organization.Slug,
	})
	if err != nil {
		return err
	}
	org, err := service.CreateOrganization(ctx, orgCreate)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}

	providers, err := service.ListAgentProviders(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("list seeded agent providers: %w", err)
	}
	defaultProviderID := selectSetupDefaultProviderID(input.Agents, providers)

	projectCreate, err := catalogdomain.ParseCreateProject(org.ID, catalogdomain.ProjectInput{
		Name:                   input.Project.Name,
		Slug:                   input.Project.Slug,
		Status:                 "In Progress",
		MaxConcurrentAgents:    intPtr(0),
		DefaultAgentProviderID: defaultProviderID,
	})
	if err != nil {
		return err
	}
	if _, err := service.CreateProject(ctx, projectCreate); err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	if defaultProviderID != nil {
		updateOrg, updateErr := catalogdomain.ParseUpdateOrganization(org.ID, catalogdomain.OrganizationInput{
			Name:                   org.Name,
			Slug:                   org.Slug,
			DefaultAgentProviderID: defaultProviderID,
		})
		if updateErr != nil {
			return updateErr
		}
		if _, updateErr = service.UpdateOrganization(ctx, updateOrg); updateErr != nil {
			return fmt.Errorf("update organization default agent provider: %w", updateErr)
		}
	}

	return nil
}

func safeSlug(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", ".", "-")
	slug := replacer.Replace(lower)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "workspace"
	}
	return slug
}

func intPtr(value int) *int {
	return &value
}

func selectSetupDefaultProviderID(
	selectedOptions []AgentOption,
	providers []catalogdomain.AgentProvider,
) *string {
	for _, option := range selectedOptions {
		for _, providerItem := range providers {
			if providerItem.Name == option.Name && providerItem.AdapterType == option.AdapterType {
				id := providerItem.ID.String()
				return &id
			}
		}
	}

	for _, providerItem := range providers {
		if providerItem.Available {
			id := providerItem.ID.String()
			return &id
		}
	}

	return nil
}
