package setup

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
)

type Mode string

const (
	ModePersonal   Mode = "personal"
	ModeTeam       Mode = "team"
	ModeEnterprise Mode = "enterprise"
)

type ModeOption struct {
	ID          Mode   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AgentOption struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Command     string                       `json:"command"`
	AdapterType entagentprovider.AdapterType `json:"adapter_type"`
	ModelName   string                       `json:"model_name"`
	Available   bool                         `json:"available"`
	Path        string                       `json:"path,omitempty"`
}

type Bootstrap struct {
	ConfigExists bool          `json:"config_exists"`
	ConfigPath   string        `json:"config_path"`
	Modes        []ModeOption  `json:"modes"`
	Agents       []AgentOption `json:"agents"`
	Defaults     Defaults      `json:"defaults"`
}

type Defaults struct {
	Mode          Mode             `json:"mode"`
	DefaultBranch string           `json:"default_branch"`
	Database      RawDatabaseInput `json:"database"`
}

type RawDatabaseInput struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`
}

type RawProjectInput struct {
	Name            string `json:"name"`
	PrimaryRepoPath string `json:"primary_repo_path"`
	PrimaryRepoURL  string `json:"primary_repo_url"`
	DefaultBranch   string `json:"default_branch"`
}

type RawCompleteRequest struct {
	Mode     string           `json:"mode"`
	Database RawDatabaseInput `json:"database"`
	Agents   []string         `json:"agents"`
	Project  RawProjectInput  `json:"project"`
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type ProjectConfig struct {
	Name            string
	PrimaryRepoPath string
	PrimaryRepoURL  string
	DefaultBranch   string
}

type CompleteRequest struct {
	Mode     Mode
	Database DatabaseConfig
	Agents   []string
	Project  ProjectConfig
}

type DatabaseTestResult struct {
	Message string `json:"message"`
}

type CompleteResult struct {
	ConfigPath      string   `json:"config_path"`
	EnvPath         string   `json:"env_path"`
	PrimaryRepoPath string   `json:"primary_repo_path"`
	ScaffoldedFiles []string `json:"scaffolded_files"`
	ProjectName     string   `json:"project_name"`
	Mode            Mode     `json:"mode"`
}

type parsedAgentSet struct {
	IDs     []string
	Options []AgentOption
}

func parseMode(raw string) (Mode, error) {
	mode := Mode(strings.TrimSpace(strings.ToLower(raw)))
	switch mode {
	case ModePersonal, ModeTeam, ModeEnterprise:
		return mode, nil
	default:
		return "", fmt.Errorf("mode must be one of personal, team, enterprise")
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

func parseProjectInput(raw RawProjectInput) (ProjectConfig, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return ProjectConfig{}, fmt.Errorf("project.name must not be empty")
	}

	repoPath := strings.TrimSpace(raw.PrimaryRepoPath)
	if repoPath == "" {
		return ProjectConfig{}, fmt.Errorf("project.primary_repo_path must not be empty")
	}
	absoluteRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("resolve project.primary_repo_path: %w", err)
	}

	info, err := os.Stat(absoluteRepoPath)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("stat project.primary_repo_path: %w", err)
	}
	if !info.IsDir() {
		return ProjectConfig{}, fmt.Errorf("project.primary_repo_path must be a directory")
	}
	if _, err := os.Stat(filepath.Join(absoluteRepoPath, ".git")); err != nil {
		return ProjectConfig{}, fmt.Errorf("project.primary_repo_path must point to a git repository")
	}

	defaultBranch := strings.TrimSpace(raw.DefaultBranch)
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	return ProjectConfig{
		Name:            name,
		PrimaryRepoPath: absoluteRepoPath,
		PrimaryRepoURL:  strings.TrimSpace(raw.PrimaryRepoURL),
		DefaultBranch:   defaultBranch,
	}, nil
}

func parseCompleteRequest(raw RawCompleteRequest, availableAgents []AgentOption) (CompleteRequest, error) {
	mode, err := parseMode(raw.Mode)
	if err != nil {
		return CompleteRequest{}, err
	}

	database, err := parseDatabaseInput(raw.Database)
	if err != nil {
		return CompleteRequest{}, err
	}

	project, err := parseProjectInput(raw.Project)
	if err != nil {
		return CompleteRequest{}, err
	}

	agents, err := parseAgentIDs(raw.Agents, availableAgents)
	if err != nil {
		return CompleteRequest{}, err
	}

	return CompleteRequest{
		Mode:     mode,
		Database: database,
		Agents:   agents.IDs,
		Project:  project,
	}, nil
}

func parseAgentIDs(raw []string, availableAgents []AgentOption) (parsedAgentSet, error) {
	known := make(map[string]AgentOption, len(availableAgents))
	for _, option := range availableAgents {
		known[option.ID] = option
	}

	seen := make(map[string]struct{}, len(raw))
	ids := make([]string, 0, len(raw))
	options := make([]AgentOption, 0, len(raw))
	for index, item := range raw {
		id := strings.TrimSpace(item)
		if id == "" {
			return parsedAgentSet{}, fmt.Errorf("agents[%d] must not be empty", index)
		}
		option, ok := known[id]
		if !ok {
			return parsedAgentSet{}, fmt.Errorf("agents[%d] references unsupported agent %q", index, id)
		}
		if !option.Available {
			return parsedAgentSet{}, fmt.Errorf("agents[%d] references unavailable agent %q", index, id)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		options = append(options, option)
	}

	return parsedAgentSet{IDs: ids, Options: options}, nil
}
