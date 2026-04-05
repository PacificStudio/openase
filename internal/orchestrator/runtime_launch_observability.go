package orchestrator

import (
	"errors"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

type runtimeLaunchFailureStage string

const (
	runtimeLaunchStageContext             runtimeLaunchFailureStage = "launch_context"
	runtimeLaunchStageResolveMachine      runtimeLaunchFailureStage = "resolve_machine"
	runtimeLaunchStageWorkspaceTransport  runtimeLaunchFailureStage = "workspace_transport"
	runtimeLaunchStageWorkspaceRoot       runtimeLaunchFailureStage = "workspace_root"
	runtimeLaunchStageRepoAuth            runtimeLaunchFailureStage = "repo_auth"
	runtimeLaunchStageGitOperation        runtimeLaunchFailureStage = "git_operation"
	runtimeLaunchStageHookOnClaim         runtimeLaunchFailureStage = "hook_on_claim"
	runtimeLaunchStageRuntimeSnapshot     runtimeLaunchFailureStage = "runtime_snapshot"
	runtimeLaunchStagePreflightTransport  runtimeLaunchFailureStage = "preflight_transport"
	runtimeLaunchStageOpenASEPreflight    runtimeLaunchFailureStage = "openase_preflight"
	runtimeLaunchStageAgentCLIPreflight   runtimeLaunchFailureStage = "agent_cli_preflight"
	runtimeLaunchStageBuildInstructions   runtimeLaunchFailureStage = "build_instructions"
	runtimeLaunchStageHookOnStart         runtimeLaunchFailureStage = "hook_on_start"
	runtimeLaunchStageTransportResolve    runtimeLaunchFailureStage = "transport_resolve"
	runtimeLaunchStageProcessStart        runtimeLaunchFailureStage = "process_start"
	runtimeExecutionStageProcessStreaming runtimeLaunchFailureStage = "process_streaming"
)

type runtimeLaunchFailure struct {
	stage         runtimeLaunchFailureStage
	machineID     uuid.UUID
	transportMode string
	workspaceRoot string
	cause         error
}

func (e *runtimeLaunchFailure) Error() string {
	if e == nil {
		return "runtime launch failed"
	}
	if e.cause == nil {
		return "runtime launch failed"
	}
	return e.cause.Error()
}

func (e *runtimeLaunchFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func wrapRuntimeLaunchFailure(
	machine catalogdomain.Machine,
	workspaceRoot string,
	stage runtimeLaunchFailureStage,
	cause error,
) error {
	if cause == nil {
		return nil
	}
	if existing := runtimeLaunchFailureDetails(cause); existing != nil {
		if existing.stage == "" {
			existing.stage = stage
		}
		if existing.machineID == uuid.Nil {
			existing.machineID = machine.ID
		}
		if strings.TrimSpace(existing.transportMode) == "" {
			existing.transportMode = machine.ConnectionMode.String()
		}
		if strings.TrimSpace(existing.workspaceRoot) == "" {
			existing.workspaceRoot = strings.TrimSpace(workspaceRoot)
		}
		return existing
	}
	return &runtimeLaunchFailure{
		stage:         stage,
		machineID:     machine.ID,
		transportMode: machine.ConnectionMode.String(),
		workspaceRoot: strings.TrimSpace(workspaceRoot),
		cause:         cause,
	}
}

func runtimeLaunchFailureDetails(err error) *runtimeLaunchFailure {
	var details *runtimeLaunchFailure
	if errors.As(err, &details) {
		return details
	}
	return nil
}

func mergeRuntimeFailureMetadata(metadata map[string]any, err error) map[string]any {
	cloned := cloneLifecycleMetadata(metadata)
	details := runtimeLaunchFailureDetails(err)
	if details == nil {
		return cloned
	}
	if details.stage != "" {
		cloned["failure_stage"] = string(details.stage)
	}
	if details.machineID != uuid.Nil {
		cloned["machine_id"] = details.machineID.String()
	}
	if strings.TrimSpace(details.transportMode) != "" {
		cloned["transport_mode"] = strings.TrimSpace(details.transportMode)
	}
	if strings.TrimSpace(details.workspaceRoot) != "" {
		cloned["workspace_root"] = strings.TrimSpace(details.workspaceRoot)
	}
	return cloned
}

func classifyRuntimeLaunchWorkspaceStage(err error) runtimeLaunchFailureStage {
	var prepareErr *workspaceinfra.PrepareError
	if !errors.As(err, &prepareErr) {
		return runtimeLaunchStageWorkspaceTransport
	}
	switch prepareErr.Stage {
	case workspaceinfra.PrepareFailureStageWorkspaceRoot:
		return runtimeLaunchStageWorkspaceRoot
	case workspaceinfra.PrepareFailureStageRepoAuth:
		return runtimeLaunchStageRepoAuth
	case workspaceinfra.PrepareFailureStageGitOperation:
		return runtimeLaunchStageGitOperation
	default:
		return runtimeLaunchStageWorkspaceTransport
	}
}

func classifyRuntimeLaunchPreflightStage(err error) runtimeLaunchFailureStage {
	var preflightErr *machinetransport.RuntimePreflightError
	if !errors.As(err, &preflightErr) {
		return runtimeLaunchStagePreflightTransport
	}
	switch preflightErr.Stage {
	case machinetransport.RuntimePreflightStageWorkspace:
		return runtimeLaunchStageWorkspaceRoot
	case machinetransport.RuntimePreflightStageOpenASE:
		return runtimeLaunchStageOpenASEPreflight
	case machinetransport.RuntimePreflightStageAgentCLI:
		return runtimeLaunchStageAgentCLIPreflight
	default:
		return runtimeLaunchStagePreflightTransport
	}
}
