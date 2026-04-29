package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func (s runtimeWorkspacePreparationSlice) buildAgentPlatformAccess(ctx context.Context, launchContext runtimeLaunchContext) (runtimePlatformAccess, error) {
	l := s.launcher
	if l == nil || l.agentPlatform == nil {
		return runtimePlatformAccess{}, nil
	}
	if launchContext.agent == nil || launchContext.project == nil || launchContext.ticket == nil {
		return runtimePlatformAccess{}, fmt.Errorf("runtime launch context is incomplete for platform environment")
	}

	scopeWhitelist := agentplatform.ScopeWhitelist{}
	if launchContext.ticket.WorkflowID != nil {
		workflowItem, err := l.client.Workflow.Get(ctx, *launchContext.ticket.WorkflowID)
		if err != nil {
			return runtimePlatformAccess{}, fmt.Errorf("load workflow %s for agent platform token: %w", *launchContext.ticket.WorkflowID, err)
		}
		scopeWhitelist = agentplatform.ScopeWhitelist{
			Configured: len(workflowItem.PlatformAccessAllowed) > 0,
			Scopes:     append([]string(nil), workflowItem.PlatformAccessAllowed...),
		}
	}

	issued, err := l.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:        launchContext.agent.ID,
		ProjectID:      launchContext.project.ID,
		TicketID:       launchContext.ticket.ID,
		ScopeWhitelist: scopeWhitelist,
	})
	if err != nil {
		return runtimePlatformAccess{}, fmt.Errorf("issue agent platform token: %w", err)
	}
	contractScopes := issued.Scopes
	if len(contractScopes) == 0 {
		contractScopes = agentplatform.DefaultScopesForPrincipalKind(agentplatform.PrincipalKindTicketAgent)
	}

	contractInput := agentplatform.RuntimeContractInput{
		PrincipalKind: agentplatform.PrincipalKindTicketAgent,
		ProjectID:     launchContext.project.ID,
		TicketID:      launchContext.ticket.ID,
		APIURL:        l.platformAPIURL,
		Token:         issued.Token,
		Scopes:        contractScopes,
	}
	return runtimePlatformAccess{
		environment: agentplatform.BuildRuntimeEnvironment(contractInput),
		contract:    agentplatform.BuildCapabilityContract(contractInput),
	}, nil
}

func (s runtimeWorkspacePreparationSlice) ticketRuntimePlatformContract(
	launchContext runtimeLaunchContext,
	scopes []string,
) string {
	l := s.launcher
	if launchContext.project == nil || launchContext.ticket == nil {
		return ""
	}
	return agentplatform.BuildCapabilityContract(agentplatform.RuntimeContractInput{
		PrincipalKind: agentplatform.PrincipalKindTicketAgent,
		ProjectID:     launchContext.project.ID,
		TicketID:      launchContext.ticket.ID,
		APIURL:        l.platformAPIURL,
		Token:         "<runtime-injected>",
		Scopes:        scopes,
	})
}

func (s runtimeWorkspacePreparationSlice) buildRuntimeSecretEnvironment(ctx context.Context, launchContext runtimeLaunchContext) ([]string, error) {
	l := s.launcher
	if l == nil || l.secretManager == nil || launchContext.project == nil || launchContext.ticket == nil || launchContext.agent == nil {
		return nil, nil
	}

	resolved, err := l.secretManager.ResolveBoundForRuntime(ctx, secretsservice.ResolveBoundRuntimeInput{
		ProjectID:  launchContext.project.ID,
		TicketID:   uuidPointer(launchContext.ticket.ID),
		WorkflowID: launchContext.ticket.WorkflowID,
		AgentID:    uuidPointer(launchContext.agent.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("resolve runtime secret bindings: %w", err)
	}
	environment, err := secretsservice.BuildRuntimeEnvironment(resolved)
	if err != nil {
		return nil, fmt.Errorf("build runtime secret environment: %w", err)
	}
	return environment, nil
}

func (s runtimeWorkspacePreparationSlice) buildGitHubOutboundEnvironment(
	ctx context.Context,
	projectID uuid.UUID,
	baseEnvironment []string,
) ([]string, error) {
	l := s.launcher
	if l == nil || l.githubAuth == nil || projectID == uuid.Nil {
		return nil, nil
	}

	resolved, err := l.githubAuth.ResolveProjectCredential(ctx, projectID)
	if err != nil {
		if errors.Is(err, githubauthservice.ErrCredentialNotConfigured) {
			return nil, nil
		}
		return nil, fmt.Errorf("resolve project GitHub credential for agent environment: %w", err)
	}

	token := strings.TrimSpace(resolved.Token)
	if token == "" {
		return nil, nil
	}

	return buildGitHubTokenEnvironment(baseEnvironment, token), nil
}

func (s runtimeWorkspacePreparationSlice) buildDeveloperInstructions(
	ctx context.Context,
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	workspace string,
	runtimeSnapshot workflowservice.RuntimeSnapshot,
	platformContract string,
) (string, error) {
	l := s.launcher
	if l == nil || l.workflow == nil || launchContext.ticket == nil || launchContext.ticket.WorkflowID == nil {
		return "", nil
	}
	if launchContext.agent == nil || launchContext.project == nil {
		return "", fmt.Errorf("runtime launch context is incomplete for harness injection")
	}

	currentMachine, accessibleMachines, err := l.loadMachineAccess(ctx, launchContext.project, machine, workspace)
	if err != nil {
		return "", fmt.Errorf("load project machine access for harness injection: %w", err)
	}

	data, err := l.workflow.BuildHarnessTemplateData(ctx, workflowservice.BuildHarnessTemplateDataInput{
		WorkflowID:         *launchContext.ticket.WorkflowID,
		TicketID:           launchContext.ticket.ID,
		AgentID:            &launchContext.agent.ID,
		Workspace:          strings.TrimSpace(workspace),
		Timestamp:          l.now().UTC(),
		Machine:            currentMachine,
		AccessibleMachines: accessibleMachines,
	})
	if err != nil {
		return "", fmt.Errorf("build workflow harness context for agent launch: %w", err)
	}

	rendered, err := workflowservice.RenderHarnessBody(runtimeSnapshot.Workflow.Content, data)
	if err != nil {
		return "", fmt.Errorf("render workflow harness for agent launch: %w", err)
	}

	return composeWorkflowDeveloperInstructions(rendered, composeWorkflowTicketContext(data), platformContract), nil
}
