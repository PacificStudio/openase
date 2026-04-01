package httpapi

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type machinePatchRequest struct {
	Name          *string   `json:"name"`
	Host          *string   `json:"host"`
	Port          *int      `json:"port"`
	SSHUser       *string   `json:"ssh_user"`
	SSHKeyPath    *string   `json:"ssh_key_path"`
	Description   *string   `json:"description"`
	Labels        *[]string `json:"labels"`
	Status        *string   `json:"status"`
	WorkspaceRoot *string   `json:"workspace_root"`
	AgentCLIPath  *string   `json:"agent_cli_path"`
	EnvVars       *[]string `json:"env_vars"`
}

func parseMachinePatchRequest(
	machineID uuid.UUID,
	current domain.Machine,
	patch machinePatchRequest,
) (domain.UpdateMachine, error) {
	request := domain.MachineInput{
		Name:          current.Name,
		Host:          current.Host,
		Port:          intPointer(current.Port),
		SSHUser:       current.SSHUser,
		SSHKeyPath:    current.SSHKeyPath,
		Description:   current.Description,
		Labels:        cloneStringSlice(current.Labels),
		Status:        current.Status.String(),
		WorkspaceRoot: current.WorkspaceRoot,
		AgentCLIPath:  current.AgentCLIPath,
		EnvVars:       cloneStringSlice(current.EnvVars),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.Host != nil {
		request.Host = *patch.Host
	}
	if patch.Port != nil {
		request.Port = patch.Port
	}
	if patch.SSHUser != nil {
		request.SSHUser = patch.SSHUser
	}
	if patch.SSHKeyPath != nil {
		request.SSHKeyPath = patch.SSHKeyPath
	}
	if patch.Description != nil {
		request.Description = *patch.Description
	}
	if patch.Labels != nil {
		request.Labels = *patch.Labels
	}
	if patch.Status != nil {
		request.Status = *patch.Status
	}
	if patch.WorkspaceRoot != nil {
		request.WorkspaceRoot = patch.WorkspaceRoot
	}
	if patch.AgentCLIPath != nil {
		request.AgentCLIPath = patch.AgentCLIPath
	}
	if patch.EnvVars != nil {
		request.EnvVars = *patch.EnvVars
	}

	return domain.ParseUpdateMachine(machineID, current.OrganizationID, request)
}
