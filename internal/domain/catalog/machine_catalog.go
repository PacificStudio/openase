package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	LocalMachineName = "local"
	LocalMachineHost = "local"
)

type Machine struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Name            string
	Host            string
	Port            int
	SSHUser         *string
	SSHKeyPath      *string
	Description     string
	Labels          []string
	Status          MachineStatus
	WorkspaceRoot   *string
	MirrorRoot      *string
	AgentCLIPath    *string
	EnvVars         []string
	LastHeartbeatAt *time.Time
	Resources       map[string]any
}

type MachineProbe struct {
	CheckedAt time.Time
	Transport string
	Output    string
	Resources map[string]any
}

type MachineInput struct {
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Port          *int     `json:"port"`
	SSHUser       *string  `json:"ssh_user"`
	SSHKeyPath    *string  `json:"ssh_key_path"`
	Description   string   `json:"description"`
	Labels        []string `json:"labels"`
	Status        string   `json:"status"`
	WorkspaceRoot *string  `json:"workspace_root"`
	MirrorRoot    *string  `json:"mirror_root"`
	AgentCLIPath  *string  `json:"agent_cli_path"`
	EnvVars       []string `json:"env_vars"`
}

type CreateMachine struct {
	OrganizationID uuid.UUID
	Name           string
	Host           string
	Port           int
	SSHUser        *string
	SSHKeyPath     *string
	Description    string
	Labels         []string
	Status         MachineStatus
	WorkspaceRoot  *string
	MirrorRoot     *string
	AgentCLIPath   *string
	EnvVars        []string
}

type UpdateMachine struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Host           string
	Port           int
	SSHUser        *string
	SSHKeyPath     *string
	Description    string
	Labels         []string
	Status         MachineStatus
	WorkspaceRoot  *string
	MirrorRoot     *string
	AgentCLIPath   *string
	EnvVars        []string
}

type RecordMachineProbe struct {
	ID              uuid.UUID
	Status          MachineStatus
	LastHeartbeatAt time.Time
	Resources       map[string]any
}

func ParseCreateMachine(organizationID uuid.UUID, raw MachineInput) (CreateMachine, error) {
	name, err := parseMachineName(raw.Name)
	if err != nil {
		return CreateMachine{}, err
	}

	host, err := parseMachineHost(raw.Host)
	if err != nil {
		return CreateMachine{}, err
	}

	if host == LocalMachineHost && name != LocalMachineName {
		return CreateMachine{}, fmt.Errorf("local machine must use name %q", LocalMachineName)
	}
	if name == LocalMachineName && host != LocalMachineHost {
		return CreateMachine{}, fmt.Errorf("machine %q must use host %q", LocalMachineName, LocalMachineHost)
	}

	port, err := parseMachinePort(raw.Port)
	if err != nil {
		return CreateMachine{}, err
	}

	sshUser := parseOptionalText(raw.SSHUser)
	sshKeyPath := parseOptionalText(raw.SSHKeyPath)
	if host != LocalMachineHost {
		if sshUser == nil {
			return CreateMachine{}, fmt.Errorf("ssh_user must not be empty for remote machines")
		}
		if sshKeyPath == nil {
			return CreateMachine{}, fmt.Errorf("ssh_key_path must not be empty for remote machines")
		}
	}

	labels, err := parseLabels(raw.Labels)
	if err != nil {
		return CreateMachine{}, err
	}

	status, err := parseMachineStatus(raw.Status, host == LocalMachineHost)
	if err != nil {
		return CreateMachine{}, err
	}

	envVars, err := parseMachineEnvVars(raw.EnvVars)
	if err != nil {
		return CreateMachine{}, err
	}

	return CreateMachine{
		OrganizationID: organizationID,
		Name:           name,
		Host:           host,
		Port:           port,
		SSHUser:        sshUser,
		SSHKeyPath:     sshKeyPath,
		Description:    strings.TrimSpace(raw.Description),
		Labels:         labels,
		Status:         status,
		WorkspaceRoot:  parseOptionalText(raw.WorkspaceRoot),
		MirrorRoot:     parseOptionalText(raw.MirrorRoot),
		AgentCLIPath:   parseOptionalText(raw.AgentCLIPath),
		EnvVars:        envVars,
	}, nil
}

func ParseUpdateMachine(id uuid.UUID, organizationID uuid.UUID, raw MachineInput) (UpdateMachine, error) {
	input, err := ParseCreateMachine(organizationID, raw)
	if err != nil {
		return UpdateMachine{}, err
	}

	return UpdateMachine{
		ID:             id,
		OrganizationID: input.OrganizationID,
		Name:           input.Name,
		Host:           input.Host,
		Port:           input.Port,
		SSHUser:        input.SSHUser,
		SSHKeyPath:     input.SSHKeyPath,
		Description:    input.Description,
		Labels:         input.Labels,
		Status:         input.Status,
		WorkspaceRoot:  input.WorkspaceRoot,
		MirrorRoot:     input.MirrorRoot,
		AgentCLIPath:   input.AgentCLIPath,
		EnvVars:        input.EnvVars,
	}, nil
}

func parseMachineName(raw string) (string, error) {
	return parseName("name", raw)
}

func parseMachineHost(raw string) (string, error) {
	host, err := parseTrimmedRequired("host", raw)
	if err != nil {
		return "", err
	}

	if strings.Contains(host, " ") {
		return "", fmt.Errorf("host must not contain spaces")
	}

	return host, nil
}

func parseMachinePort(raw *int) (int, error) {
	if raw == nil {
		return 22, nil
	}
	if *raw <= 0 || *raw > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}

	return *raw, nil
}

func parseMachineStatus(raw string, isLocal bool) (MachineStatus, error) {
	if strings.TrimSpace(raw) == "" {
		if isLocal {
			return MachineStatusOnline, nil
		}
		return MachineStatusMaintenance, nil
	}

	status := MachineStatus(strings.ToLower(strings.TrimSpace(raw)))
	if !status.IsValid() {
		return "", fmt.Errorf("status must be one of online, offline, degraded, maintenance")
	}

	return status, nil
}

func parseMachineEnvVars(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	items := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for index, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("env_vars[%d] must not be empty", index)
		}
		if !strings.Contains(trimmed, "=") {
			return nil, fmt.Errorf("env_vars[%d] must be in KEY=VALUE format", index)
		}
		key, value, _ := strings.Cut(trimmed, "=")
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("env_vars[%d] must include a non-empty key", index)
		}
		normalized := strings.TrimSpace(key) + "=" + value
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}

	return items, nil
}
