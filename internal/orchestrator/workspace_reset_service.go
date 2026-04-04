package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketdomain "github.com/BetterAndBetterII/openase/internal/domain/ticket"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/google/uuid"
)

type TicketWorkspaceResetService struct {
	client      *ent.Client
	logger      *slog.Logger
	provisioner *runtimeWorkspaceProvisioner
}

type ticketWorkspaceCleanupTarget struct {
	machineID     uuid.UUID
	workspaceRoot string
	workspaceIDs  []uuid.UUID
}

type TicketWorkspaceResetConflictError struct {
	TicketID uuid.UUID
	RunID    uuid.UUID
}

func (e TicketWorkspaceResetConflictError) Error() string {
	return fmt.Sprintf("ticket %s still has active run %s", e.TicketID, e.RunID)
}

func (TicketWorkspaceResetConflictError) WorkspaceResetConflict() bool {
	return true
}

func NewTicketWorkspaceResetService(
	client *ent.Client,
	logger *slog.Logger,
	sshPool *sshinfra.Pool,
) *TicketWorkspaceResetService {
	if logger == nil {
		logger = slog.Default()
	}
	now := time.Now
	return &TicketWorkspaceResetService{
		client:      client,
		logger:      logger.With("component", "ticket-workspace-reset"),
		provisioner: newRuntimeWorkspaceProvisioner(client, logger, sshPool, now),
	}
}

func (s *TicketWorkspaceResetService) ResetTicketWorkspace(ctx context.Context, ticketID uuid.UUID) error {
	if s == nil || s.client == nil || s.provisioner == nil {
		return fmt.Errorf("ticket workspace reset unavailable")
	}
	if ticketID == uuid.Nil {
		return ticketdomain.ErrTicketNotFound
	}

	targets, err := s.resolveCleanupTargets(ctx, ticketID)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err := s.resetCleanupTarget(ctx, ticketID, target); err != nil {
			return err
		}
	}

	s.logger.Info("ticket workspace reset", "ticket_id", ticketID, "targets", len(targets))
	return nil
}

func (s *TicketWorkspaceResetService) resolveCleanupTargets(
	ctx context.Context,
	ticketID uuid.UUID,
) ([]ticketWorkspaceCleanupTarget, error) {
	ticketItem, err := s.client.Ticket.Query().
		Where(entticket.IDEQ(ticketID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ticketdomain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("load ticket %s for workspace reset: %w", ticketID, err)
	}
	if ticketItem.CurrentRunID != nil {
		return nil, TicketWorkspaceResetConflictError{
			TicketID: ticketID,
			RunID:    *ticketItem.CurrentRunID,
		}
	}

	workspaces, err := s.client.TicketRepoWorkspace.Query().
		Where(
			entticketrepoworkspace.TicketIDEQ(ticketID),
			entticketrepoworkspace.CleanedAtIsNil(),
		).
		Order(ent.Desc(entticketrepoworkspace.FieldCreatedAt), ent.Asc(entticketrepoworkspace.FieldRepoPath)).
		WithAgentRun(func(query *ent.AgentRunQuery) {
			query.WithProvider()
		}).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load ticket repo workspaces for reset: %w", err)
	}
	if len(workspaces) == 0 {
		return nil, nil
	}

	targets := make([]ticketWorkspaceCleanupTarget, 0, len(workspaces))
	indexByKey := make(map[string]int, len(workspaces))
	for _, item := range workspaces {
		workspaceRoot := strings.TrimSpace(item.WorkspaceRoot)
		if workspaceRoot == "" {
			return nil, fmt.Errorf("ticket repo workspace %s missing workspace root", item.ID)
		}
		if item.Edges.AgentRun == nil || item.Edges.AgentRun.Edges.Provider == nil {
			return nil, fmt.Errorf("ticket repo workspace %s missing provider context", item.ID)
		}

		machineID := item.Edges.AgentRun.Edges.Provider.MachineID
		key := machineID.String() + "|" + workspaceRoot
		targetIndex, exists := indexByKey[key]
		if !exists {
			targetIndex = len(targets)
			indexByKey[key] = targetIndex
			targets = append(targets, ticketWorkspaceCleanupTarget{
				machineID:     machineID,
				workspaceRoot: workspaceRoot,
			})
		}
		targets[targetIndex].workspaceIDs = append(targets[targetIndex].workspaceIDs, item.ID)
	}

	return targets, nil
}

func (s *TicketWorkspaceResetService) resetCleanupTarget(
	ctx context.Context,
	ticketID uuid.UUID,
	target ticketWorkspaceCleanupTarget,
) error {
	if len(target.workspaceIDs) == 0 {
		return nil
	}

	machineItem, err := s.client.Machine.Get(ctx, target.machineID)
	if err != nil {
		return fmt.Errorf("load machine %s for workspace reset: %w", target.machineID, err)
	}
	machine := mapRuntimeMachine(machineItem)
	remote := machine.Host != catalogdomain.LocalMachineHost

	if _, err := s.client.TicketRepoWorkspace.Update().
		Where(entticketrepoworkspace.IDIn(target.workspaceIDs...)).
		SetState(entticketrepoworkspace.StateCleaning).
		ClearLastError().
		Save(ctx); err != nil {
		return fmt.Errorf("mark ticket workspaces cleaning before reset: %w", err)
	}

	if err := s.provisioner.removeWorkspaceRoot(ctx, machine, remote, target.workspaceRoot); err != nil {
		trimmedError := strings.TrimSpace(err.Error())
		if _, updateErr := s.client.TicketRepoWorkspace.Update().
			Where(entticketrepoworkspace.IDIn(target.workspaceIDs...)).
			SetState(entticketrepoworkspace.StateFailed).
			SetLastError(trimmedError).
			Save(ctx); updateErr != nil {
			return fmt.Errorf(
				"reset workspace %s failed: %w (also failed to record cleanup error: %v)",
				target.workspaceRoot,
				err,
				updateErr,
			)
		}
		return err
	}

	cleanedAt := s.provisioner.now().UTC()
	if _, err := s.client.TicketRepoWorkspace.Update().
		Where(entticketrepoworkspace.IDIn(target.workspaceIDs...)).
		SetState(entticketrepoworkspace.StateCleaned).
		SetCleanedAt(cleanedAt).
		ClearLastError().
		Save(ctx); err != nil {
		return fmt.Errorf("mark ticket workspaces cleaned after reset: %w", err)
	}

	s.logger.Info(
		"ticket workspace reset target cleaned",
		"ticket_id", ticketID,
		"machine_id", target.machineID,
		"workspace_root", target.workspaceRoot,
	)
	return nil
}
