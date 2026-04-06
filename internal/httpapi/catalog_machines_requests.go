package httpapi

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type machinePatchRequest struct {
	Name               *string                               `json:"name"`
	Host               *string                               `json:"host"`
	Port               *int                                  `json:"port"`
	ReachabilityMode   *string                               `json:"reachability_mode"`
	ExecutionMode      *string                               `json:"execution_mode"`
	SSHUser            *string                               `json:"ssh_user"`
	SSHKeyPath         *string                               `json:"ssh_key_path"`
	AdvertisedEndpoint *string                               `json:"advertised_endpoint"`
	DaemonStatus       *domain.MachineDaemonStatusInput      `json:"daemon_status"`
	DetectedOS         *string                               `json:"detected_os"`
	DetectedArch       *string                               `json:"detected_arch"`
	DetectionStatus    *string                               `json:"detection_status"`
	ChannelCredential  *domain.MachineChannelCredentialInput `json:"channel_credential"`
	Description        *string                               `json:"description"`
	Labels             *[]string                             `json:"labels"`
	Status             *string                               `json:"status"`
	WorkspaceRoot      *string                               `json:"workspace_root"`
	AgentCLIPath       *string                               `json:"agent_cli_path"`
	EnvVars            *[]string                             `json:"env_vars"`
}

func parseMachinePatchRequest(
	machineID uuid.UUID,
	current domain.Machine,
	patch machinePatchRequest,
) (domain.UpdateMachine, error) {
	request := domain.MachineInput{
		Name:               current.Name,
		Host:               current.Host,
		Port:               intPointer(current.Port),
		ReachabilityMode:   current.ReachabilityMode.String(),
		ExecutionMode:      current.ExecutionMode.String(),
		SSHUser:            current.SSHUser,
		SSHKeyPath:         current.SSHKeyPath,
		AdvertisedEndpoint: current.AdvertisedEndpoint,
		DaemonStatus: domain.MachineDaemonStatusInput{
			Registered:       boolPointer(current.DaemonStatus.Registered),
			LastRegisteredAt: timeToStringPointer(current.DaemonStatus.LastRegisteredAt),
			CurrentSessionID: cloneStringPointerValue(current.DaemonStatus.CurrentSessionID),
			SessionState:     current.DaemonStatus.SessionState.String(),
		},
		DetectedOS:      current.DetectedOS.String(),
		DetectedArch:    current.DetectedArch.String(),
		DetectionStatus: current.DetectionStatus.String(),
		ChannelCredential: &domain.MachineChannelCredentialInput{
			Kind:          current.ChannelCredential.Kind.String(),
			TokenID:       cloneStringPointerValue(current.ChannelCredential.TokenID),
			CertificateID: cloneStringPointerValue(current.ChannelCredential.CertificateID),
		},
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
	if patch.ReachabilityMode != nil {
		request.ReachabilityMode = *patch.ReachabilityMode
	}
	if patch.ExecutionMode != nil {
		request.ExecutionMode = *patch.ExecutionMode
	}
	if patch.SSHUser != nil {
		request.SSHUser = patch.SSHUser
	}
	if patch.SSHKeyPath != nil {
		request.SSHKeyPath = patch.SSHKeyPath
	}
	if patch.AdvertisedEndpoint != nil {
		request.AdvertisedEndpoint = patch.AdvertisedEndpoint
	}
	if patch.DaemonStatus != nil {
		request.DaemonStatus = *patch.DaemonStatus
	}
	if patch.DetectedOS != nil {
		request.DetectedOS = *patch.DetectedOS
	}
	if patch.DetectedArch != nil {
		request.DetectedArch = *patch.DetectedArch
	}
	if patch.DetectionStatus != nil {
		request.DetectionStatus = *patch.DetectionStatus
	}
	if patch.ChannelCredential != nil {
		request.ChannelCredential = patch.ChannelCredential
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
