package catalog

import (
	"context"
	"fmt"

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
	builder := r.client.Machine.UpdateOneID(input.ID).
		SetStatus(toEntMachineStatus(input.Status)).
		SetLastHeartbeatAt(input.LastHeartbeatAt.UTC()).
		SetResources(cloneAnyMap(input.Resources))
	if err := builder.Exec(ctx); err != nil {
		return mapWriteError("record machine probe", err)
	}

	return nil
}

func createLocalMachine(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID) error {
	_, err := tx.Machine.Create().
		SetOrganizationID(organizationID).
		SetName(domain.LocalMachineName).
		SetHost(domain.LocalMachineHost).
		SetPort(22).
		SetDescription("Control-plane local execution host.").
		SetStatus(toEntMachineStatus(domain.MachineStatusOnline)).
		SetResources(map[string]any{
			"transport":    "local",
			"last_success": true,
		}).
		Save(ctx)
	if err != nil {
		return mapWriteError("create local machine", err)
	}

	return nil
}

func machineCreateBuilder(builder *ent.MachineCreate, input domain.CreateMachine) *ent.MachineCreate {
	builder.
		SetOrganizationID(input.OrganizationID).
		SetName(input.Name).
		SetHost(input.Host).
		SetPort(input.Port).
		SetDescription(input.Description).
		SetStatus(toEntMachineStatus(input.Status)).
		SetNillableSSHUser(input.SSHUser).
		SetNillableSSHKeyPath(input.SSHKeyPath).
		SetNillableWorkspaceRoot(input.WorkspaceRoot).
		SetNillableAgentCliPath(input.AgentCLIPath)
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
		SetDescription(input.Description).
		SetStatus(toEntMachineStatus(input.Status))
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

func mapMachines(items []*ent.Machine) []domain.Machine {
	machines := make([]domain.Machine, 0, len(items))
	for _, item := range items {
		machines = append(machines, mapMachine(item))
	}

	return machines
}

func mapMachine(item *ent.Machine) domain.Machine {
	return domain.Machine{
		ID:              item.ID,
		OrganizationID:  item.OrganizationID,
		Name:            item.Name,
		Host:            item.Host,
		Port:            item.Port,
		SSHUser:         optionalString(item.SSHUser),
		SSHKeyPath:      optionalString(item.SSHKeyPath),
		Description:     item.Description,
		Labels:          append([]string(nil), item.Labels...),
		Status:          toDomainMachineStatus(item.Status),
		WorkspaceRoot:   optionalString(item.WorkspaceRoot),
		AgentCLIPath:    optionalString(item.AgentCliPath),
		EnvVars:         append([]string(nil), item.EnvVars...),
		LastHeartbeatAt: cloneTimePointer(item.LastHeartbeatAt),
		Resources:       cloneAnyMap(item.Resources),
	}
}
