package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

func TestRuntimeWorkspaceProvisionerSerializesPreparePerMachine(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ctx := context.Background()
	client := openTestEntClient(t)
	provisioner := newRuntimeWorkspaceProvisioner(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, time.Now)
	provisioner.leaseMgr.leaseDuration = 250 * time.Millisecond
	provisioner.leaseMgr.heartbeatInterval = 20 * time.Millisecond
	provisioner.leaseMgr.waitInterval = 5 * time.Millisecond
	provisioner.syncInstructionHub = func(context.Context, catalogdomain.Machine, machinetransport.ResolvedTransport, workspaceinfra.Workspace, string, bool) error {
		return nil
	}

	machine := catalogdomain.Machine{
		ID:   uuid.New(),
		Name: catalogdomain.LocalMachineName,
		Host: catalogdomain.LocalMachineHost,
	}
	launchContextOne := testRuntimeLaunchContext(uuid.New(), "ASE-146A")
	launchContextTwo := testRuntimeLaunchContext(uuid.New(), "ASE-146B")

	var callCount atomic.Int32
	var activePrepare atomic.Int32
	var concurrentPrepareDetected atomic.Bool
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})
	provisioner.prepareWorkspace = func(ctx context.Context, _ catalogdomain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error) {
		if activePrepare.Add(1) > 1 {
			concurrentPrepareDetected.Store(true)
		}
		defer activePrepare.Add(-1)

		callNumber := callCount.Add(1)
		workspacePath := filepath.Join(t.TempDir(), fmt.Sprintf("workspace-%d", callNumber))
		if err := os.MkdirAll(workspacePath, 0o750); err != nil {
			return workspaceinfra.Workspace{}, err
		}

		if callNumber == 1 {
			close(firstEntered)
			select {
			case <-releaseFirst:
			case <-ctx.Done():
				return workspaceinfra.Workspace{}, ctx.Err()
			}
		} else {
			close(secondEntered)
		}

		return workspaceinfra.Workspace{Path: workspacePath, BranchName: request.BranchName}, nil
	}

	firstErrCh := make(chan error, 1)
	go func() {
		_, err := provisioner.prepareTicketWorkspace(ctx, uuid.New(), launchContextOne, machine, false)
		firstErrCh <- err
	}()

	select {
	case <-firstEntered:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first prepare to start")
	}

	secondErrCh := make(chan error, 1)
	go func() {
		_, err := provisioner.prepareTicketWorkspace(ctx, uuid.New(), launchContextTwo, machine, false)
		secondErrCh <- err
	}()

	select {
	case <-secondEntered:
		t.Fatal("second prepare entered before first lease holder released")
	case <-time.After(60 * time.Millisecond):
	}

	close(releaseFirst)

	if err := <-firstErrCh; err != nil {
		t.Fatalf("first prepareTicketWorkspace() error = %v", err)
	}
	if err := <-secondErrCh; err != nil {
		t.Fatalf("second prepareTicketWorkspace() error = %v", err)
	}
	if concurrentPrepareDetected.Load() {
		t.Fatal("expected machine-scoped lease to serialize workspace materialization")
	}
}

func TestRuntimeWorkspaceProvisionerReleasesLeaseAfterPrepareFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ctx := context.Background()
	client := openTestEntClient(t)
	provisioner := newRuntimeWorkspaceProvisioner(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, time.Now)
	provisioner.leaseMgr.leaseDuration = 200 * time.Millisecond
	provisioner.leaseMgr.heartbeatInterval = 20 * time.Millisecond
	provisioner.leaseMgr.waitInterval = 5 * time.Millisecond
	provisioner.syncInstructionHub = func(context.Context, catalogdomain.Machine, machinetransport.ResolvedTransport, workspaceinfra.Workspace, string, bool) error {
		return nil
	}

	machine := catalogdomain.Machine{
		ID:   uuid.New(),
		Name: catalogdomain.LocalMachineName,
		Host: catalogdomain.LocalMachineHost,
	}
	launchContextOne := testRuntimeLaunchContext(uuid.New(), "ASE-146C")
	launchContextTwo := testRuntimeLaunchContext(uuid.New(), "ASE-146D")

	var callCount atomic.Int32
	expectedErr := errors.New("boom")
	provisioner.prepareWorkspace = func(_ context.Context, _ catalogdomain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error) {
		if callCount.Add(1) == 1 {
			return workspaceinfra.Workspace{}, expectedErr
		}
		workspacePath := filepath.Join(t.TempDir(), "workspace-success")
		if err := os.MkdirAll(workspacePath, 0o750); err != nil {
			return workspaceinfra.Workspace{}, err
		}
		return workspaceinfra.Workspace{Path: workspacePath, BranchName: request.BranchName}, nil
	}

	if _, err := provisioner.prepareTicketWorkspace(ctx, uuid.New(), launchContextOne, machine, false); !errors.Is(err, expectedErr) {
		t.Fatalf("first prepareTicketWorkspace() error = %v, want %v", err, expectedErr)
	}

	secondCtx, cancel := context.WithTimeout(ctx, 75*time.Millisecond)
	defer cancel()
	if _, err := provisioner.prepareTicketWorkspace(secondCtx, uuid.New(), launchContextTwo, machine, false); err != nil {
		t.Fatalf("second prepareTicketWorkspace() error = %v, want released lease after failure", err)
	}
}

func testRuntimeLaunchContext(ticketID uuid.UUID, ticketIdentifier string) runtimeLaunchContext {
	return runtimeLaunchContext{
		agent: &ent.Agent{
			ID:   uuid.New(),
			Name: "codex-01",
			Edges: ent.AgentEdges{
				Provider: &ent.AgentProvider{AdapterType: entagentprovider.AdapterTypeCodexAppServer},
			},
		},
		project: &ent.Project{
			ID:   uuid.New(),
			Slug: "payments",
			Edges: ent.ProjectEdges{
				Organization: &ent.Organization{Slug: "acme"},
			},
		},
		ticket: &ent.Ticket{
			ID:         ticketID,
			Identifier: ticketIdentifier,
		},
	}
}
