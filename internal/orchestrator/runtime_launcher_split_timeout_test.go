package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

func TestRuntimeLauncherStartRuntimeSessionWaitsForWorkspaceLeaseWithoutConsumingSessionStartTimeout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	localMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ(catalogdomain.LocalMachineName),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}

	agentOne, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-lease-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create first agent: %v", err)
	}
	agentTwo, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-lease-02").
		Save(ctx)
	if err != nil {
		t.Fatalf("create second agent: %v", err)
	}

	ticketOne, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-146E").
		SetTitle("First launch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetTargetMachineID(localMachine.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create first ticket: %v", err)
	}
	ticketTwo, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-146F").
		SetTitle("Second launch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetTargetMachineID(localMachine.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create second ticket: %v", err)
	}

	runOne := mustCreateCurrentRun(ctx, t, client, agentOne, workflowItem.ID, ticketOne.ID, entagentrun.StatusLaunching, time.Time{})
	runTwo := mustCreateCurrentRun(ctx, t, client, agentTwo, workflowItem.ID, ticketTwo.ID, entagentrun.StatusLaunching, time.Time{})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
	launcher.ConfigureLaunchTimeouts(500*time.Millisecond, 50*time.Millisecond)
	launcher.adapters = &agentAdapterRegistry{
		adapters: map[entagentprovider.AdapterType]agentAdapter{
			entagentprovider.AdapterTypeCodexAppServer: runtimeImmediateAgentAdapter{},
		},
	}
	provisioner := launcher.ensureWorkspaceProvisioner()
	provisioner.leaseMgr.leaseDuration = 400 * time.Millisecond
	provisioner.leaseMgr.heartbeatInterval = 20 * time.Millisecond
	provisioner.leaseMgr.waitInterval = 5 * time.Millisecond
	firstPrepareStarted := make(chan struct{})
	var prepareCalls int
	provisioner.prepareWorkspace = func(ctx context.Context, _ catalogdomain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error) {
		prepareCalls++
		workspacePath := filepath.Join(t.TempDir(), fmt.Sprintf("workspace-%d", prepareCalls))
		if err := os.MkdirAll(workspacePath, 0o750); err != nil {
			return workspaceinfra.Workspace{}, err
		}
		if prepareCalls == 1 {
			close(firstPrepareStarted)
			select {
			case <-time.After(120 * time.Millisecond):
			case <-ctx.Done():
				return workspaceinfra.Workspace{}, ctx.Err()
			}
		}
		return workspaceinfra.Workspace{Path: workspacePath, BranchName: request.BranchName}, nil
	}
	provisioner.syncInstructionHub = func(context.Context, catalogdomain.Machine, machinetransport.ResolvedTransport, workspaceinfra.Workspace, string, bool) error {
		return nil
	}
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	assignmentOne := runtimeAssignment{ticket: ticketOne, agent: agentOne, run: runOne}
	assignmentTwo := runtimeAssignment{ticket: ticketTwo, agent: agentTwo, run: runTwo}

	type sessionResult struct {
		session agentSession
		err     error
	}
	firstResultCh := make(chan sessionResult, 1)
	go func() {
		session, err := launcher.startRuntimeSession(ctx, assignmentOne)
		firstResultCh <- sessionResult{session: session, err: err}
	}()

	select {
	case <-firstPrepareStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first workspace prepare")
	}

	secondResultCh := make(chan sessionResult, 1)
	go func() {
		session, err := launcher.startRuntimeSession(ctx, assignmentTwo)
		secondResultCh <- sessionResult{session: session, err: err}
	}()

	firstResult := <-firstResultCh
	if firstResult.err != nil {
		t.Fatalf("first startRuntimeSession() error = %v", firstResult.err)
	}
	secondResult := <-secondResultCh
	if secondResult.err != nil {
		t.Fatalf("second startRuntimeSession() error = %v", secondResult.err)
	}

	if sessionID, ok := firstResult.session.SessionID(); !ok || sessionID == "" {
		t.Fatalf("first session missing session id: %#v", firstResult.session)
	}
	if sessionID, ok := secondResult.session.SessionID(); !ok || sessionID == "" {
		t.Fatalf("second session missing session id: %#v", secondResult.session)
	}
}

type runtimeImmediateAgentAdapter struct{}

func (runtimeImmediateAgentAdapter) Start(context.Context, agentSessionStartSpec) (agentSession, error) {
	return runtimeStaticAgentSession{sessionID: uuid.NewString()}, nil
}

func (runtimeImmediateAgentAdapter) Resume(context.Context, agentSessionResumeSpec) (agentSession, error) {
	return runtimeStaticAgentSession{sessionID: uuid.NewString()}, nil
}
