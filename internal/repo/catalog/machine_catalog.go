package catalog

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	"github.com/google/uuid"
)

func (r *EntRepository) ListMachines(ctx context.Context, organizationID uuid.UUID) ([]domain.Machine, error) {
	exists, err := r.organizationIsActive(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("check organization before listing machines: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	items, err := r.client.Machine.Query().
		Where(entmachine.OrganizationID(organizationID)).
		Order(entmachine.ByName()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list machines: %w", err)
	}

	return mapMachines(items), nil
}

func (r *EntRepository) CreateMachine(ctx context.Context, input domain.CreateMachine) (domain.Machine, error) {
	exists, err := r.organizationIsActive(ctx, input.OrganizationID)
	if err != nil {
		return domain.Machine{}, fmt.Errorf("check organization before creating machine: %w", err)
	}
	if !exists {
		return domain.Machine{}, ErrNotFound
	}
	input, err = normalizeCreateMachineDefaults(input)
	if err != nil {
		return domain.Machine{}, fmt.Errorf("normalize machine before create: %w", err)
	}

	item, err := machineCreateBuilder(r.client.Machine.Create(), input).Save(ctx)
	if err != nil {
		return domain.Machine{}, mapWriteError("create machine", err)
	}

	return mapMachine(item), nil
}

func (r *EntRepository) GetMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error) {
	item, err := r.client.Machine.Get(ctx, id)
	if err != nil {
		return domain.Machine{}, mapReadError("get machine", err)
	}

	return mapMachine(item), nil
}

func (r *EntRepository) UpdateMachine(ctx context.Context, input domain.UpdateMachine) (domain.Machine, error) {
	current, err := r.client.Machine.Get(ctx, input.ID)
	if err != nil {
		return domain.Machine{}, mapReadError("get machine for update", err)
	}
	if current.OrganizationID != input.OrganizationID {
		return domain.Machine{}, fmt.Errorf("%w: machine organization mismatch", ErrInvalidInput)
	}
	if current.Name == domain.LocalMachineName && (input.Name != domain.LocalMachineName || input.Host != domain.LocalMachineHost) {
		return domain.Machine{}, fmt.Errorf("%w: local machine name and host are immutable", ErrInvalidInput)
	}
	input, err = normalizeUpdateMachineDefaults(input)
	if err != nil {
		return domain.Machine{}, fmt.Errorf("normalize machine before update: %w", err)
	}

	item, err := machineUpdateBuilder(r.client.Machine.UpdateOneID(input.ID), input).Save(ctx)
	if err != nil {
		return domain.Machine{}, mapWriteError("update machine", err)
	}

	return mapMachine(item), nil
}

func (r *EntRepository) DeleteMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error) {
	item, err := r.client.Machine.Get(ctx, id)
	if err != nil {
		return domain.Machine{}, mapReadError("get machine before delete", err)
	}
	if item.Name == domain.LocalMachineName {
		return domain.Machine{}, fmt.Errorf("%w: local machine cannot be removed", ErrInvalidInput)
	}

	if err := r.client.Machine.DeleteOneID(id).Exec(ctx); err != nil {
		return domain.Machine{}, mapWriteError("delete machine", err)
	}

	return mapMachine(item), nil
}

func (r *EntRepository) RecordMachineProbe(ctx context.Context, input domain.RecordMachineProbe) error {
	detectedOS, err := domain.ParseStoredMachineDetectedOS(input.DetectedOS.String())
	if err != nil {
		return fmt.Errorf("normalize probe detected os: %w", err)
	}
	detectedArch, err := domain.ParseStoredMachineDetectedArch(input.DetectedArch.String())
	if err != nil {
		return fmt.Errorf("normalize probe detected arch: %w", err)
	}
	detectionStatus, err := domain.ParseStoredMachineDetectionStatus(input.DetectionStatus.String())
	if err != nil {
		return fmt.Errorf("normalize probe detection status: %w", err)
	}
	builder := r.client.Machine.UpdateOneID(input.ID).
		SetStatus(toEntMachineStatus(input.Status)).
		SetLastHeartbeatAt(input.LastHeartbeatAt.UTC()).
		SetDetectedOs(entmachine.DetectedOs(detectedOS.String())).
		SetDetectedArch(entmachine.DetectedArch(detectedArch.String())).
		SetDetectionStatus(entmachine.DetectionStatus(detectionStatus.String())).
		SetResources(cloneAnyMap(input.Resources))
	if err := builder.Exec(ctx); err != nil {
		return mapWriteError("record machine probe", err)
	}

	return nil
}

func createLocalMachine(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID) (*ent.Machine, error) {
	item, err := tx.Machine.Create().
		SetOrganizationID(organizationID).
		SetName(domain.LocalMachineName).
		SetHost(domain.LocalMachineHost).
		SetPort(22).
		SetConnectionMode(entmachine.ConnectionModeLocal).
		SetTransportCapabilities(pgarray.StringArray(domainTransportCapabilityStrings(defaultLocalTransportCapabilities()))).
		SetDescription("Control-plane local execution host.").
		SetStatus(toEntMachineStatus(domain.MachineStatusOnline)).
		SetDetectedOs(entmachine.DetectedOs(normalizeDetectedOS(runtime.GOOS).String())).
		SetDetectedArch(entmachine.DetectedArch(normalizeDetectedArch(runtime.GOARCH).String())).
		SetDetectionStatus(entmachine.DetectionStatus(domain.MachineDetectionStatusOK.String())).
		SetResources(map[string]any{
			"transport":    "local",
			"last_success": true,
		}).
		Save(ctx)
	if err != nil {
		return nil, mapWriteError("create local machine", err)
	}

	return item, nil
}

func machineCreateBuilder(builder *ent.MachineCreate, input domain.CreateMachine) *ent.MachineCreate {
	builder.
		SetOrganizationID(input.OrganizationID).
		SetName(input.Name).
		SetHost(input.Host).
		SetPort(input.Port).
		SetConnectionMode(entmachine.ConnectionMode(input.ConnectionMode.String())).
		SetDescription(input.Description).
		SetStatus(toEntMachineStatus(input.Status)).
		SetNillableSSHUser(input.SSHUser).
		SetNillableSSHKeyPath(input.SSHKeyPath).
		SetNillableAdvertisedEndpoint(input.AdvertisedEndpoint).
		SetDaemonRegistered(input.DaemonStatus.Registered).
		SetNillableDaemonLastRegisteredAt(input.DaemonStatus.LastRegisteredAt).
		SetNillableDaemonSessionID(input.DaemonStatus.CurrentSessionID).
		SetDaemonSessionState(entmachine.DaemonSessionState(input.DaemonStatus.SessionState.String())).
		SetDetectedOs(entmachine.DetectedOs(input.DetectedOS.String())).
		SetDetectedArch(entmachine.DetectedArch(input.DetectedArch.String())).
		SetDetectionStatus(entmachine.DetectionStatus(input.DetectionStatus.String())).
		SetChannelCredentialKind(entmachine.ChannelCredentialKind(input.ChannelCredential.Kind.String())).
		SetNillableChannelTokenID(input.ChannelCredential.TokenID).
		SetNillableChannelCertificateID(input.ChannelCredential.CertificateID).
		SetNillableWorkspaceRoot(input.WorkspaceRoot).
		SetNillableAgentCliPath(input.AgentCLIPath)
	if len(input.TransportCapabilities) > 0 {
		builder.SetTransportCapabilities(pgarray.StringArray(domainTransportCapabilityStrings(input.TransportCapabilities)))
	}
	if len(input.Labels) > 0 {
		builder.SetLabels(pgarray.StringArray(input.Labels))
	}
	if len(input.EnvVars) > 0 {
		builder.SetEnvVars(pgarray.StringArray(input.EnvVars))
	}

	return builder
}

func machineUpdateBuilder(builder *ent.MachineUpdateOne, input domain.UpdateMachine) *ent.MachineUpdateOne {
	builder.
		SetName(input.Name).
		SetHost(input.Host).
		SetPort(input.Port).
		SetConnectionMode(entmachine.ConnectionMode(input.ConnectionMode.String())).
		SetDescription(input.Description).
		SetStatus(toEntMachineStatus(input.Status)).
		SetDaemonRegistered(input.DaemonStatus.Registered).
		SetDaemonSessionState(entmachine.DaemonSessionState(input.DaemonStatus.SessionState.String())).
		SetDetectedOs(entmachine.DetectedOs(input.DetectedOS.String())).
		SetDetectedArch(entmachine.DetectedArch(input.DetectedArch.String())).
		SetDetectionStatus(entmachine.DetectionStatus(input.DetectionStatus.String())).
		SetChannelCredentialKind(entmachine.ChannelCredentialKind(input.ChannelCredential.Kind.String()))
	if input.SSHUser != nil {
		builder.SetSSHUser(*input.SSHUser)
	} else {
		builder.ClearSSHUser()
	}
	if input.SSHKeyPath != nil {
		builder.SetSSHKeyPath(*input.SSHKeyPath)
	} else {
		builder.ClearSSHKeyPath()
	}
	if input.AdvertisedEndpoint != nil {
		builder.SetAdvertisedEndpoint(*input.AdvertisedEndpoint)
	} else {
		builder.ClearAdvertisedEndpoint()
	}
	if len(input.TransportCapabilities) > 0 {
		builder.SetTransportCapabilities(pgarray.StringArray(domainTransportCapabilityStrings(input.TransportCapabilities)))
	} else {
		builder.ClearTransportCapabilities()
	}
	if input.DaemonStatus.LastRegisteredAt != nil {
		builder.SetDaemonLastRegisteredAt(*input.DaemonStatus.LastRegisteredAt)
	} else {
		builder.ClearDaemonLastRegisteredAt()
	}
	if input.DaemonStatus.CurrentSessionID != nil {
		builder.SetDaemonSessionID(*input.DaemonStatus.CurrentSessionID)
	} else {
		builder.ClearDaemonSessionID()
	}
	if input.ChannelCredential.TokenID != nil {
		builder.SetChannelTokenID(*input.ChannelCredential.TokenID)
	} else {
		builder.ClearChannelTokenID()
	}
	if input.ChannelCredential.CertificateID != nil {
		builder.SetChannelCertificateID(*input.ChannelCredential.CertificateID)
	} else {
		builder.ClearChannelCertificateID()
	}
	if len(input.Labels) > 0 {
		builder.SetLabels(pgarray.StringArray(input.Labels))
	} else {
		builder.ClearLabels()
	}
	if input.WorkspaceRoot != nil {
		builder.SetWorkspaceRoot(*input.WorkspaceRoot)
	} else {
		builder.ClearWorkspaceRoot()
	}
	if input.AgentCLIPath != nil {
		builder.SetAgentCliPath(*input.AgentCLIPath)
	} else {
		builder.ClearAgentCliPath()
	}
	if len(input.EnvVars) > 0 {
		builder.SetEnvVars(pgarray.StringArray(input.EnvVars))
	} else {
		builder.ClearEnvVars()
	}

	return builder
}

func normalizeCreateMachineDefaults(input domain.CreateMachine) (domain.CreateMachine, error) {
	mode, err := domain.ParseStoredMachineConnectionMode(input.ConnectionMode.String(), input.Host)
	if err != nil {
		return domain.CreateMachine{}, err
	}
	capabilities, err := domain.ParseStoredMachineTransportCapabilities(domainTransportCapabilityStrings(input.TransportCapabilities), mode)
	if err != nil {
		return domain.CreateMachine{}, err
	}
	detectedOS, err := domain.ParseStoredMachineDetectedOS(input.DetectedOS.String())
	if err != nil {
		return domain.CreateMachine{}, err
	}
	detectedArch, err := domain.ParseStoredMachineDetectedArch(input.DetectedArch.String())
	if err != nil {
		return domain.CreateMachine{}, err
	}
	detectionStatus, err := domain.ParseStoredMachineDetectionStatus(input.DetectionStatus.String())
	if err != nil {
		return domain.CreateMachine{}, err
	}
	channelCredentialKind, err := domain.ParseStoredMachineChannelCredentialKind(input.ChannelCredential.Kind.String())
	if err != nil {
		return domain.CreateMachine{}, err
	}
	sessionState, err := domain.ParseStoredMachineSessionState(input.DaemonStatus.SessionState.String())
	if err != nil {
		return domain.CreateMachine{}, err
	}

	input.ConnectionMode = mode
	input.TransportCapabilities = capabilities
	input.DetectedOS = detectedOS
	input.DetectedArch = detectedArch
	input.DetectionStatus = detectionStatus
	input.ChannelCredential.Kind = channelCredentialKind
	input.DaemonStatus.SessionState = sessionState
	return input, nil
}

func normalizeUpdateMachineDefaults(input domain.UpdateMachine) (domain.UpdateMachine, error) {
	createInput, err := normalizeCreateMachineDefaults(domain.CreateMachine{
		OrganizationID:        input.OrganizationID,
		Name:                  input.Name,
		Host:                  input.Host,
		Port:                  input.Port,
		ConnectionMode:        input.ConnectionMode,
		TransportCapabilities: input.TransportCapabilities,
		SSHUser:               input.SSHUser,
		SSHKeyPath:            input.SSHKeyPath,
		AdvertisedEndpoint:    input.AdvertisedEndpoint,
		DaemonStatus:          input.DaemonStatus,
		DetectedOS:            input.DetectedOS,
		DetectedArch:          input.DetectedArch,
		DetectionStatus:       input.DetectionStatus,
		ChannelCredential:     input.ChannelCredential,
		Description:           input.Description,
		Labels:                input.Labels,
		Status:                input.Status,
		WorkspaceRoot:         input.WorkspaceRoot,
		AgentCLIPath:          input.AgentCLIPath,
		EnvVars:               input.EnvVars,
	})
	if err != nil {
		return domain.UpdateMachine{}, err
	}
	input.ConnectionMode = createInput.ConnectionMode
	input.TransportCapabilities = createInput.TransportCapabilities
	input.DaemonStatus = createInput.DaemonStatus
	input.DetectedOS = createInput.DetectedOS
	input.DetectedArch = createInput.DetectedArch
	input.DetectionStatus = createInput.DetectionStatus
	input.ChannelCredential = createInput.ChannelCredential
	return input, nil
}

func mapMachines(items []*ent.Machine) []domain.Machine {
	machines := make([]domain.Machine, 0, len(items))
	for _, item := range items {
		machines = append(machines, mapMachine(item))
	}

	return machines
}

func mapMachine(item *ent.Machine) domain.Machine {
	connectionMode := parseStoredMachineConnectionMode(string(item.ConnectionMode), item.Host)
	transportCapabilities := parseStoredMachineTransportCapabilities(item.TransportCapabilities, connectionMode)
	detectedOS := parseStoredMachineDetectedOS(string(item.DetectedOs))
	detectedArch := parseStoredMachineDetectedArch(string(item.DetectedArch))
	detectionStatus := parseStoredMachineDetectionStatus(string(item.DetectionStatus))
	daemonStatus := domain.MachineDaemonStatus{
		Registered:       item.DaemonRegistered,
		LastRegisteredAt: cloneTimePointer(item.DaemonLastRegisteredAt),
		CurrentSessionID: optionalString(item.DaemonSessionID),
		SessionState:     parseStoredMachineSessionState(string(item.DaemonSessionState)),
	}
	channelCredential := domain.MachineChannelCredential{
		Kind:          parseStoredMachineChannelCredentialKind(string(item.ChannelCredentialKind)),
		TokenID:       optionalString(item.ChannelTokenID),
		CertificateID: optionalString(item.ChannelCertificateID),
	}

	return domain.Machine{
		ID:                    item.ID,
		OrganizationID:        item.OrganizationID,
		Name:                  item.Name,
		Host:                  item.Host,
		Port:                  item.Port,
		ConnectionMode:        connectionMode,
		TransportCapabilities: transportCapabilities,
		SSHUser:               optionalString(item.SSHUser),
		SSHKeyPath:            optionalString(item.SSHKeyPath),
		AdvertisedEndpoint:    optionalString(item.AdvertisedEndpoint),
		DaemonStatus:          daemonStatus,
		DetectedOS:            detectedOS,
		DetectedArch:          detectedArch,
		DetectionStatus:       detectionStatus,
		ChannelCredential:     channelCredential,
		Description:           item.Description,
		Labels:                append([]string(nil), item.Labels...),
		Status:                toDomainMachineStatus(item.Status),
		WorkspaceRoot:         optionalString(item.WorkspaceRoot),
		AgentCLIPath:          optionalString(item.AgentCliPath),
		EnvVars:               append([]string(nil), item.EnvVars...),
		LastHeartbeatAt:       cloneTimePointer(item.LastHeartbeatAt),
		Resources:             cloneAnyMap(item.Resources),
	}
}

func domainTransportCapabilityStrings(items []domain.MachineTransportCapability) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item.String())
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}
	return values
}

func parseStoredMachineConnectionMode(raw string, host string) domain.MachineConnectionMode {
	mode, err := domain.ParseStoredMachineConnectionMode(raw, host)
	if err != nil {
		if host == domain.LocalMachineHost {
			return domain.MachineConnectionModeLocal
		}
		return domain.MachineConnectionModeSSH
	}
	return mode
}

func parseStoredMachineTransportCapabilities(
	raw []string,
	mode domain.MachineConnectionMode,
) []domain.MachineTransportCapability {
	items, err := domain.ParseStoredMachineTransportCapabilities(raw, mode)
	if err != nil {
		fallback, _ := domain.ParseStoredMachineTransportCapabilities(nil, mode)
		return fallback
	}
	return items
}

func defaultLocalTransportCapabilities() []domain.MachineTransportCapability {
	return []domain.MachineTransportCapability{
		domain.MachineTransportCapabilityProbe,
		domain.MachineTransportCapabilityWorkspacePrepare,
		domain.MachineTransportCapabilityArtifactSync,
		domain.MachineTransportCapabilityProcessStreaming,
	}
}

func normalizeDetectedOS(raw string) domain.MachineDetectedOS {
	value, err := domain.ParseStoredMachineDetectedOS(raw)
	if err != nil {
		return domain.MachineDetectedOSUnknown
	}
	return value
}

func normalizeDetectedArch(raw string) domain.MachineDetectedArch {
	value, err := domain.ParseStoredMachineDetectedArch(raw)
	if err != nil {
		return domain.MachineDetectedArchUnknown
	}
	return value
}

func parseStoredMachineDetectedOS(raw string) domain.MachineDetectedOS {
	value, err := domain.ParseStoredMachineDetectedOS(raw)
	if err != nil {
		return domain.MachineDetectedOSUnknown
	}
	return value
}

func parseStoredMachineDetectedArch(raw string) domain.MachineDetectedArch {
	value, err := domain.ParseStoredMachineDetectedArch(raw)
	if err != nil {
		return domain.MachineDetectedArchUnknown
	}
	return value
}

func parseStoredMachineDetectionStatus(raw string) domain.MachineDetectionStatus {
	value, err := domain.ParseStoredMachineDetectionStatus(raw)
	if err != nil {
		return domain.MachineDetectionStatusUnknown
	}
	return value
}

func parseStoredMachineSessionState(raw string) domain.MachineTransportSessionState {
	value, err := domain.ParseStoredMachineSessionState(raw)
	if err != nil {
		return domain.MachineTransportSessionStateUnknown
	}
	return value
}

func parseStoredMachineChannelCredentialKind(raw string) domain.MachineChannelCredentialKind {
	value, err := domain.ParseStoredMachineChannelCredentialKind(raw)
	if err != nil {
		return domain.MachineChannelCredentialKindNone
	}
	return value
}
