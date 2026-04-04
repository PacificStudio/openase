package setup

import (
	"fmt"
	"net/url"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

type DatabaseSourceType string

const (
	DatabaseSourceManual DatabaseSourceType = "manual"
	DatabaseSourceDocker DatabaseSourceType = "docker"
)

const (
	DefaultOrganizationName = "Local Workspace"
	DefaultOrganizationSlug = "local-workspace"
	DefaultProjectName      = "OpenASE Workspace"
	DefaultProjectSlug      = "openase-workspace"
)

type DatabaseSourceOption struct {
	ID          DatabaseSourceType `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
}

type CLIDiagnostic struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

type AgentOption struct {
	ID          string                                 `json:"id"`
	Name        string                                 `json:"name"`
	Command     string                                 `json:"command"`
	AdapterType catalogdomain.AgentProviderAdapterType `json:"adapter_type"`
	ModelName   string                                 `json:"model_name"`
	Available   bool                                   `json:"available"`
	Path        string                                 `json:"path,omitempty"`
	Version     string                                 `json:"version,omitempty"`
}

type Bootstrap struct {
	ConfigExists bool                   `json:"config_exists"`
	ConfigPath   string                 `json:"config_path"`
	Sources      []DatabaseSourceOption `json:"sources"`
	Agents       []AgentOption          `json:"agents"`
	CLI          []CLIDiagnostic        `json:"cli"`
	Defaults     Defaults               `json:"defaults"`
}

type Defaults struct {
	ManualDatabase RawDatabaseInput       `json:"manual_database"`
	DockerDatabase RawDockerDatabaseInput `json:"docker_database"`
}

type RawDatabaseInput struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`
}

type RawDockerDatabaseInput struct {
	ContainerName string `json:"container_name"`
	DatabaseName  string `json:"database_name"`
	User          string `json:"user"`
	Port          int    `json:"port"`
	VolumeName    string `json:"volume_name"`
	Image         string `json:"image"`
}

type RawDatabaseSourceInput struct {
	Type   string                  `json:"type"`
	Manual *RawDatabaseInput       `json:"manual,omitempty"`
	Docker *RawDockerDatabaseInput `json:"docker,omitempty"`
}

type RawCompleteRequest struct {
	Database       RawDatabaseInput `json:"database"`
	AllowOverwrite bool             `json:"allow_overwrite,omitempty"`
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type DockerDatabaseConfig struct {
	ContainerName string
	DatabaseName  string
	User          string
	Port          int
	VolumeName    string
	Image         string
}

type DatabaseSourceConfig struct {
	Type   DatabaseSourceType
	Manual *DatabaseConfig
	Docker *DockerDatabaseConfig
}

type CompleteRequest struct {
	Database       DatabaseConfig
	AllowOverwrite bool
}

type DatabaseTestResult struct {
	Message string `json:"message"`
}

type PreparedDatabase struct {
	Source DatabaseSourceType
	Config DatabaseConfig
	Docker *DockerDatabaseRuntime
}

type DockerDatabaseRuntime struct {
	ContainerName string `json:"container_name"`
	DatabaseName  string `json:"database_name"`
	User          string `json:"user"`
	Port          int    `json:"port"`
	VolumeName    string `json:"volume_name"`
	Image         string `json:"image"`
}

type CompleteResult struct {
	ConfigPath       string `json:"config_path"`
	EnvPath          string `json:"env_path"`
	OrganizationName string `json:"organization_name"`
	OrganizationSlug string `json:"organization_slug"`
	ProjectName      string `json:"project_name"`
	ProjectSlug      string `json:"project_slug"`
}

func defaultRawDatabaseInput() RawDatabaseInput {
	return RawDatabaseInput{
		Host:    "127.0.0.1",
		Port:    5432,
		Name:    "openase",
		User:    "openase",
		SSLMode: "disable",
	}
}

func defaultRawDockerDatabaseInput() RawDockerDatabaseInput {
	return RawDockerDatabaseInput{
		ContainerName: "openase-local-postgres",
		DatabaseName:  "openase",
		User:          "openase",
		Port:          15432,
		VolumeName:    "openase-local-postgres-data",
		Image:         "postgres:16-alpine",
	}
}

func parseDatabaseSourceType(raw string) (DatabaseSourceType, error) {
	sourceType := DatabaseSourceType(strings.TrimSpace(strings.ToLower(raw)))
	switch sourceType {
	case DatabaseSourceManual, DatabaseSourceDocker:
		return sourceType, nil
	default:
		return "", fmt.Errorf("database.type must be one of manual, docker")
	}
}

func parseDatabaseInput(raw RawDatabaseInput) (DatabaseConfig, error) {
	host := strings.TrimSpace(raw.Host)
	if host == "" {
		return DatabaseConfig{}, fmt.Errorf("database.host must not be empty")
	}

	port := raw.Port
	if port == 0 {
		port = 5432
	}
	if port < 1 || port > 65535 {
		return DatabaseConfig{}, fmt.Errorf("database.port must be between 1 and 65535")
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return DatabaseConfig{}, fmt.Errorf("database.name must not be empty")
	}

	user := strings.TrimSpace(raw.User)
	if user == "" {
		return DatabaseConfig{}, fmt.Errorf("database.user must not be empty")
	}

	sslMode := strings.TrimSpace(strings.ToLower(raw.SSLMode))
	if sslMode == "" {
		sslMode = "disable"
	}
	switch sslMode {
	case "disable", "require", "verify-full":
	default:
		return DatabaseConfig{}, fmt.Errorf("database.ssl_mode must be one of disable, require, verify-full")
	}

	return DatabaseConfig{
		Host:     host,
		Port:     port,
		Name:     name,
		User:     user,
		Password: raw.Password,
		SSLMode:  sslMode,
	}, nil
}

func (c DatabaseConfig) DSN() string {
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:     c.Name,
		RawQuery: "sslmode=" + url.QueryEscape(c.SSLMode),
	}).String()
}

func (c DatabaseConfig) Raw() RawDatabaseInput {
	return RawDatabaseInput{
		Host:     c.Host,
		Port:     c.Port,
		Name:     c.Name,
		User:     c.User,
		Password: c.Password,
		SSLMode:  c.SSLMode,
	}
}

func parseDockerDatabaseInput(raw RawDockerDatabaseInput) (DockerDatabaseConfig, error) {
	defaults := defaultRawDockerDatabaseInput()

	containerName := strings.TrimSpace(raw.ContainerName)
	if containerName == "" {
		containerName = defaults.ContainerName
	}

	databaseName := strings.TrimSpace(raw.DatabaseName)
	if databaseName == "" {
		databaseName = defaults.DatabaseName
	}

	user := strings.TrimSpace(raw.User)
	if user == "" {
		user = defaults.User
	}

	port := raw.Port
	if port == 0 {
		port = defaults.Port
	}
	if port < 1 || port > 65535 {
		return DockerDatabaseConfig{}, fmt.Errorf("database.docker.port must be between 1 and 65535")
	}

	volumeName := strings.TrimSpace(raw.VolumeName)
	if volumeName == "" {
		volumeName = defaults.VolumeName
	}

	image := strings.TrimSpace(raw.Image)
	if image == "" {
		image = defaults.Image
	}

	return DockerDatabaseConfig{
		ContainerName: containerName,
		DatabaseName:  databaseName,
		User:          user,
		Port:          port,
		VolumeName:    volumeName,
		Image:         image,
	}, nil
}

func parseDatabaseSourceInput(raw RawDatabaseSourceInput) (DatabaseSourceConfig, error) {
	sourceType, err := parseDatabaseSourceType(raw.Type)
	if err != nil {
		return DatabaseSourceConfig{}, err
	}

	switch sourceType {
	case DatabaseSourceManual:
		manualRaw := defaultRawDatabaseInput()
		if raw.Manual != nil {
			manualRaw = *raw.Manual
		}
		manual, err := parseDatabaseInput(manualRaw)
		if err != nil {
			return DatabaseSourceConfig{}, err
		}
		return DatabaseSourceConfig{
			Type:   DatabaseSourceManual,
			Manual: &manual,
		}, nil
	case DatabaseSourceDocker:
		dockerRaw := defaultRawDockerDatabaseInput()
		if raw.Docker != nil {
			dockerRaw = *raw.Docker
		}
		docker, err := parseDockerDatabaseInput(dockerRaw)
		if err != nil {
			return DatabaseSourceConfig{}, err
		}
		return DatabaseSourceConfig{
			Type:   DatabaseSourceDocker,
			Docker: &docker,
		}, nil
	default:
		return DatabaseSourceConfig{}, fmt.Errorf("unsupported database source %q", sourceType)
	}
}

func parseCompleteRequest(raw RawCompleteRequest) (CompleteRequest, error) {
	database, err := parseDatabaseInput(raw.Database)
	if err != nil {
		return CompleteRequest{}, err
	}

	return CompleteRequest{
		Database:       database,
		AllowOverwrite: raw.AllowOverwrite,
	}, nil
}
