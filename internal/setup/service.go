package setup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"go.yaml.in/yaml/v3"
)

var ErrSetupAlreadyCompleted = errors.New("setup already completed")

type DatabaseConnector interface {
	Ping(context.Context, string) error
	Migrate(context.Context, string) error
}

type Installer interface {
	Initialize(context.Context, InstallInput) error
}

type InstallInput struct {
	Mode     Mode
	Database DatabaseConfig
	Agents   []AgentOption
	Project  ProjectConfig
}

type Options struct {
	HomeDir   string
	Resolver  provider.ExecutableResolver
	Connector DatabaseConnector
	Installer Installer
}

type Service struct {
	homeDir   string
	resolver  provider.ExecutableResolver
	connector DatabaseConnector
	installer Installer
	logger    *slog.Logger
}

type runtimeDatabaseConnector struct{}

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

type defaultInstaller struct{}

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

	return &Service{
		homeDir:   homeDir,
		resolver:  resolver,
		connector: connector,
		installer: installer,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, nil
}

func (s *Service) Bootstrap(_ context.Context) (Bootstrap, error) {
	configPath, exists := s.existingConfigPath()

	return Bootstrap{
		ConfigExists: exists,
		ConfigPath:   configPath,
		Modes: []ModeOption{
			{ID: ModePersonal, Name: "Personal", Description: "Single developer, fastest path to value."},
			{ID: ModeTeam, Name: "Team", Description: "Shared workspace with multiple contributors."},
			{ID: ModeEnterprise, Name: "Enterprise Pilot", Description: "Pilot track with tighter controls later."},
		},
		Agents: detectAgentOptions(s.resolver),
		Defaults: Defaults{
			Mode:          ModePersonal,
			DefaultBranch: "main",
			Database: RawDatabaseInput{
				Host:    "localhost",
				Port:    5432,
				Name:    "openase",
				User:    "openase",
				SSLMode: "disable",
			},
		},
	}, nil
}

func (s *Service) TestDatabase(ctx context.Context, raw RawDatabaseInput) (DatabaseTestResult, error) {
	databaseConfig, err := parseDatabaseInput(raw)
	if err != nil {
		return DatabaseTestResult{}, err
	}

	if err := s.connector.Ping(ctx, databaseConfig.DSN()); err != nil {
		return DatabaseTestResult{}, fmt.Errorf("test database connection: %w", err)
	}

	return DatabaseTestResult{Message: "Database connection succeeded."}, nil
}

func (s *Service) Complete(ctx context.Context, raw RawCompleteRequest) (CompleteResult, error) {
	if _, exists := s.existingConfigPath(); exists {
		return CompleteResult{}, ErrSetupAlreadyCompleted
	}

	bootstrap, err := s.Bootstrap(ctx)
	if err != nil {
		return CompleteResult{}, err
	}

	request, err := parseCompleteRequest(raw, bootstrap.Agents)
	if err != nil {
		return CompleteResult{}, err
	}

	parsedAgents, err := parseAgentIDs(raw.Agents, bootstrap.Agents)
	if err != nil {
		return CompleteResult{}, err
	}

	if err := s.connector.Migrate(ctx, request.Database.DSN()); err != nil {
		return CompleteResult{}, fmt.Errorf("migrate database schema: %w", err)
	}

	scaffoldedFiles, err := ensurePrimaryRepoScaffold(request.Project.PrimaryRepoPath)
	if err != nil {
		return CompleteResult{}, err
	}

	if err := s.installer.Initialize(ctx, InstallInput{
		Mode:     request.Mode,
		Database: request.Database,
		Agents:   parsedAgents.Options,
		Project:  request.Project,
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

	if err := s.writeConfigFile(request); err != nil {
		return CompleteResult{}, err
	}
	if err := s.writeEnvFile(authToken); err != nil {
		return CompleteResult{}, err
	}

	return CompleteResult{
		ConfigPath:      s.configPath(),
		EnvPath:         s.envPath(),
		PrimaryRepoPath: request.Project.PrimaryRepoPath,
		ScaffoldedFiles: scaffoldedFiles,
		ProjectName:     request.Project.Name,
		Mode:            request.Mode,
	}, nil
}

func (s *Service) ensureHomeLayout() error {
	baseDir := filepath.Join(s.homeDir, ".openase")
	for _, dir := range []string{
		baseDir,
		filepath.Join(baseDir, "logs"),
		filepath.Join(baseDir, "workspaces"),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	return nil
}

func (s *Service) writeConfigFile(request CompleteRequest) error {
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
	type observabilityConfig struct {
		Metrics observabilityMetricsConfig `yaml:"metrics"`
	}
	type logConfig struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	}
	type setupConfig struct {
		Mode            Mode     `yaml:"mode"`
		ProjectName     string   `yaml:"project_name"`
		PrimaryRepoPath string   `yaml:"primary_repo_path"`
		PrimaryRepoURL  string   `yaml:"primary_repo_url,omitempty"`
		DefaultBranch   string   `yaml:"default_branch"`
		Agents          []string `yaml:"agents,omitempty"`
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
		},
		Log: logConfig{Level: "info", Format: "text"},
		Setup: setupConfig{
			Mode:            request.Mode,
			ProjectName:     request.Project.Name,
			PrimaryRepoPath: request.Project.PrimaryRepoPath,
			PrimaryRepoURL:  request.Project.PrimaryRepoURL,
			DefaultBranch:   request.Project.DefaultBranch,
			Agents:          slices.Clone(request.Agents),
		},
	}

	content, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal setup config: %w", err)
	}

	if err := os.WriteFile(s.configPath(), content, 0o600); err != nil {
		return fmt.Errorf("write setup config: %w", err)
	}
	if err := os.WriteFile(s.legacyConfigPath(), content, 0o600); err != nil {
		return fmt.Errorf("write legacy setup config: %w", err)
	}

	return nil
}

func (s *Service) writeEnvFile(authToken string) error {
	content := fmt.Sprintf("OPENASE_AUTH_TOKEN=%s\n", authToken)
	if err := os.WriteFile(s.envPath(), []byte(content), 0o600); err != nil {
		return fmt.Errorf("write setup env file: %w", err)
	}

	return nil
}

func (s *Service) configPath() string {
	return filepath.Join(s.homeDir, ".openase", "config.yaml")
}

func (s *Service) legacyConfigPath() string {
	return filepath.Join(s.homeDir, ".openase", "openase.yaml")
}

func (s *Service) envPath() string {
	return filepath.Join(s.homeDir, ".openase", ".env")
}

func (s *Service) existingConfigPath() (string, bool) {
	for _, candidate := range []string{s.configPath(), s.legacyConfigPath()} {
		if fileExists(candidate) {
			return candidate, true
		}
	}

	return s.configPath(), false
}

func detectAgentOptions(resolver provider.ExecutableResolver) []AgentOption {
	options := []AgentOption{
		{
			ID:          "claude-code",
			Name:        "Claude Code",
			Command:     "claude",
			AdapterType: "claude-code-cli",
			ModelName:   "claude-sonnet-4-5",
		},
		{
			ID:          "codex",
			Name:        "OpenAI Codex",
			Command:     "codex",
			AdapterType: "codex-app-server",
			ModelName:   "gpt-5.3-codex",
		},
		{
			ID:          "gemini",
			Name:        "Gemini CLI",
			Command:     "gemini",
			AdapterType: "gemini-cli",
			ModelName:   "gemini-2.5-pro",
		},
	}

	for index, option := range options {
		if resolver == nil {
			continue
		}
		path, err := resolver.LookPath(option.Command)
		if err == nil {
			options[index].Available = true
			options[index].Path = path
		}
	}

	return options
}

func ensurePrimaryRepoScaffold(repoRoot string) ([]string, error) {
	files := primaryRepoScaffold(repoRoot)
	created := make([]string, 0, len(files))
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.path), 0o750); err != nil {
			return nil, fmt.Errorf("create scaffold directory for %s: %w", file.path, err)
		}
		if fileExists(file.path) {
			continue
		}
		if err := os.WriteFile(file.path, []byte(file.content), file.mode); err != nil {
			return nil, fmt.Errorf("write scaffold file %s: %w", file.path, err)
		}
		created = append(created, file.path)
	}

	return created, nil
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

func (defaultInstaller) Initialize(ctx context.Context, input InstallInput) (err error) {
	client, err := database.Open(ctx, input.Database.DSN())
	if err != nil {
		return fmt.Errorf("open installation database: %w", err)
	}
	defer func() {
		err = errors.Join(err, client.Close())
	}()

	repo := catalogrepo.NewEntRepository(client)
	service := catalogservice.New(repo, executable.NewPathResolver(), nil)

	orgSlug := safeSlug(string(input.Mode) + "-" + input.Project.Name)
	orgCreate, err := catalogdomain.ParseCreateOrganization(catalogdomain.OrganizationInput{
		Name: strings.TrimSpace(strings.ToTitle(strings.ReplaceAll(string(input.Mode), "-", " "))) + " Workspace",
		Slug: orgSlug,
	})
	if err != nil {
		return err
	}
	org, err := service.CreateOrganization(ctx, orgCreate)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}

	var defaultProviderID *string
	for _, option := range input.Agents {
		createProvider, parseErr := catalogdomain.ParseCreateAgentProvider(org.ID, catalogdomain.AgentProviderInput{
			Name:        option.Name,
			AdapterType: string(option.AdapterType),
			ModelName:   option.ModelName,
		})
		if parseErr != nil {
			return parseErr
		}
		providerItem, createErr := service.CreateAgentProvider(ctx, createProvider)
		if createErr != nil {
			return fmt.Errorf("create agent provider %s: %w", option.Name, createErr)
		}
		if defaultProviderID == nil {
			id := providerItem.ID.String()
			defaultProviderID = &id
		}
	}

	projectCreate, err := catalogdomain.ParseCreateProject(org.ID, catalogdomain.ProjectInput{
		Name:                   input.Project.Name,
		Slug:                   safeSlug(input.Project.Name),
		Status:                 "active",
		MaxConcurrentAgents:    intPtr(5),
		DefaultAgentProviderID: defaultProviderID,
	})
	if err != nil {
		return err
	}
	project, err := service.CreateProject(ctx, projectCreate)
	if err != nil {
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

	repositoryURL := input.Project.PrimaryRepoURL
	if repositoryURL == "" {
		repositoryURL = input.Project.PrimaryRepoPath
	}
	isPrimary := true
	projectRepoCreate, err := catalogdomain.ParseCreateProjectRepo(project.ID, catalogdomain.ProjectRepoInput{
		Name:          filepath.Base(input.Project.PrimaryRepoPath),
		RepositoryURL: repositoryURL,
		DefaultBranch: input.Project.DefaultBranch,
		IsPrimary:     &isPrimary,
	})
	if err != nil {
		return err
	}
	if _, err := service.CreateProjectRepo(ctx, projectRepoCreate); err != nil {
		return fmt.Errorf("create project repo: %w", err)
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
