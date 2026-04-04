package chat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	claudecodeadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

type projectConversationRuntimeManager struct {
	logger              *slog.Logger
	catalog             projectConversationCatalog
	runtimeStore        projectConversationRuntimeStore
	skillSync           projectConversationSkillSync
	localProcessManager provider.AgentCLIProcessManager
	sshPool             *sshinfra.Pool
	transports          *machinetransport.Resolver
	githubAuth          githubauthservice.TokenResolver
	newCodexRuntime     func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error)

	mu   sync.Mutex
	live map[uuid.UUID]*liveProjectConversation
}

func newProjectConversationRuntimeManager(
	logger *slog.Logger,
	catalog projectConversationCatalog,
	runtimeStore projectConversationRuntimeStore,
	localProcessManager provider.AgentCLIProcessManager,
	sshPool *sshinfra.Pool,
	newCodexRuntime func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error),
) *projectConversationRuntimeManager {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &projectConversationRuntimeManager{
		logger:              logger,
		catalog:             catalog,
		runtimeStore:        runtimeStore,
		localProcessManager: localProcessManager,
		sshPool:             sshPool,
		transports:          machinetransport.NewResolver(localProcessManager, sshPool),
		newCodexRuntime:     newCodexRuntime,
		live:                map[uuid.UUID]*liveProjectConversation{},
	}
}

func (m *projectConversationRuntimeManager) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	if m == nil {
		return
	}
	m.githubAuth = resolver
}

func (m *projectConversationRuntimeManager) ConfigureSkillSync(syncer projectConversationSkillSync) {
	if m == nil {
		return
	}
	m.skillSync = syncer
}

func (m *projectConversationRuntimeManager) Get(conversationID uuid.UUID) (*liveProjectConversation, bool) {
	if m == nil {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	live := m.live[conversationID]
	return live, live != nil
}

func (m *projectConversationRuntimeManager) WorkspacePath(conversationID uuid.UUID) (provider.AbsolutePath, bool) {
	live, ok := m.Get(conversationID)
	if !ok || live == nil || strings.TrimSpace(live.workspace.String()) == "" {
		return "", false
	}
	return live.workspace, true
}

func (m *projectConversationRuntimeManager) Close(conversationID uuid.UUID) (*liveProjectConversation, bool) {
	if m == nil {
		return nil, false
	}

	m.mu.Lock()
	live := m.live[conversationID]
	delete(m.live, conversationID)
	m.mu.Unlock()

	if live != nil && live.runtime != nil {
		live.runtime.CloseSession(SessionID(conversationID.String()))
	}
	return live, live != nil
}

func (m *projectConversationRuntimeManager) ensureLiveRuntime(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
) (*liveProjectConversation, bool, error) {
	principal, err := m.runtimeStore.EnsurePrincipal(ctx, domain.EnsurePrincipalInput{
		ConversationID: conversation.ID,
		ProjectID:      conversation.ProjectID,
		ProviderID:     conversation.ProviderID,
		Name:           projectConversationPrincipalName(conversation.ID),
	})
	if err != nil {
		return nil, false, fmt.Errorf("ensure project conversation principal: %w", err)
	}

	m.mu.Lock()
	if existing := m.live[conversation.ID]; existing != nil {
		existing.principal = principal
		m.mu.Unlock()
		return existing, true, nil
	}
	m.mu.Unlock()

	machine, err := m.catalog.GetMachine(ctx, providerItem.MachineID)
	if err != nil {
		return nil, false, fmt.Errorf("get chat provider machine: %w", err)
	}
	workspacePath, err := m.ensureConversationWorkspace(ctx, machine, project, providerItem, conversation.ID)
	if err != nil {
		return nil, false, err
	}
	if err := m.runRemoteRuntimePreflight(ctx, machine, providerItem, workspacePath.String()); err != nil {
		return nil, false, err
	}
	manager, err := m.resolveProcessManager(machine)
	if err != nil {
		return nil, false, err
	}

	var runtime Runtime
	var codexRuntime projectConversationCodexRuntime
	var interruptRuntime projectConversationInterruptRuntime
	switch providerItem.AdapterType {
	case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
		if m.newCodexRuntime == nil {
			return nil, false, fmt.Errorf("codex project conversation runtime unavailable")
		}
		codexRuntime, err = m.newCodexRuntime(manager)
		if err != nil {
			return nil, false, err
		}
		runtime = codexRuntime
		interruptRuntime = codexRuntime
	case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
		claudeRuntime := NewClaudeRuntime(claudecodeadapter.NewAdapter(manager))
		runtime = claudeRuntime
		interruptRuntime = claudeRuntime
	case catalogdomain.AgentProviderAdapterTypeGeminiCLI:
		runtime = NewGeminiRuntime(manager)
	default:
		return nil, false, fmt.Errorf("%w: provider=%s", ErrProviderUnsupported, providerItem.AdapterType)
	}

	live := &liveProjectConversation{
		principal: principal,
		provider:  providerItem,
		machine:   machine,
		runtime:   runtime,
		codex:     codexRuntime,
		interrupt: interruptRuntime,
		workspace: workspacePath,
	}

	m.mu.Lock()
	m.live[conversation.ID] = live
	m.mu.Unlock()

	now := time.Now().UTC()
	updatedPrincipal, updateErr := m.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
		PrincipalID:          principal.ID,
		RuntimeState:         domain.RuntimeStateReady,
		CurrentSessionID:     optionalString(conversation.ID.String()),
		CurrentWorkspacePath: optionalString(workspacePath.String()),
		LastHeartbeatAt:      &now,
		CurrentStepStatus:    optionalString("runtime_ready"),
		CurrentStepSummary:   optionalString("Project conversation runtime ready."),
		CurrentStepChangedAt: &now,
	})
	if updateErr == nil {
		live.principal = updatedPrincipal
	}
	return live, false, nil
}

func (m *projectConversationRuntimeManager) ensureConversationWorkspace(
	ctx context.Context,
	machine catalogdomain.Machine,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	conversationID uuid.UUID,
) (provider.AbsolutePath, error) {
	root := ""
	if machine.WorkspaceRoot != nil && strings.TrimSpace(*machine.WorkspaceRoot) != "" {
		root = strings.TrimSpace(*machine.WorkspaceRoot)
	} else if machine.Host == catalogdomain.LocalMachineHost {
		localRoot, err := workspaceinfra.LocalWorkspaceRoot()
		if err != nil {
			return "", err
		}
		root = localRoot
	}
	if root == "" {
		return "", fmt.Errorf("chat provider machine %s is missing workspace_root", machine.Name)
	}

	projectRepos, err := m.catalog.ListProjectRepos(ctx, project.ID)
	if err != nil {
		return "", fmt.Errorf("list project repos for conversation workspace: %w", err)
	}
	request, err := workspaceinfra.ParseSetupRequest(workspaceinfra.SetupInput{
		WorkspaceRoot:    root,
		OrganizationSlug: project.OrganizationID.String(),
		ProjectSlug:      project.Slug,
		AgentName:        projectConversationPrincipalName(conversationID),
		TicketIdentifier: projectConversationWorkspaceName(conversationID),
		Repos:            mapConversationWorkspaceRepos(projectRepos),
	})
	if err != nil {
		return "", fmt.Errorf("build project conversation workspace request: %w", err)
	}
	request, err = m.applyGitHubWorkspaceAuth(ctx, project.ID, request)
	if err != nil {
		return "", fmt.Errorf("prepare chat workspace auth: %w", err)
	}

	transport, err := m.resolveTransport(machine)
	if err != nil {
		return "", err
	}
	workspaceItem, err := transport.PrepareWorkspace(ctx, machine, request)
	if err != nil {
		if machine.Host == catalogdomain.LocalMachineHost {
			return "", fmt.Errorf("prepare local chat workspace: %w", err)
		}
		return "", fmt.Errorf("prepare remote chat workspace: %w", err)
	}
	if err := m.syncConversationWorkspaceSkills(ctx, machine, project.ID, workspaceItem.Path, string(providerItem.AdapterType)); err != nil {
		return "", err
	}
	return provider.ParseAbsolutePath(filepath.Clean(workspaceItem.Path))
}

func (m *projectConversationRuntimeManager) resolveProcessManager(machine catalogdomain.Machine) (provider.AgentCLIProcessManager, error) {
	transport, err := m.resolveTransport(machine)
	if err != nil {
		return nil, err
	}
	return machinetransport.NewProcessManager(transport, machine), nil
}

func (m *projectConversationRuntimeManager) syncConversationWorkspaceSkills(
	ctx context.Context,
	machine catalogdomain.Machine,
	projectID uuid.UUID,
	workspaceRoot string,
	adapterType string,
) error {
	if m == nil || m.skillSync == nil {
		return nil
	}

	if machine.Host == catalogdomain.LocalMachineHost {
		_, err := m.skillSync.RefreshSkills(ctx, workflowservice.RefreshSkillsInput{
			ProjectID:     projectID,
			WorkspaceRoot: workspaceRoot,
			AdapterType:   adapterType,
		})
		if err != nil {
			return fmt.Errorf("refresh local project conversation skills: %w", err)
		}
		return nil
	}

	tempRoot, err := os.MkdirTemp("", "openase-project-conversation-skills-*")
	if err != nil {
		return fmt.Errorf("create temp skills workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()

	_, err = m.skillSync.RefreshSkills(ctx, workflowservice.RefreshSkillsInput{
		ProjectID:     projectID,
		WorkspaceRoot: tempRoot,
		AdapterType:   adapterType,
	})
	if err != nil {
		return fmt.Errorf("refresh remote project conversation skills snapshot: %w", err)
	}
	if err := m.copyConversationWorkspaceArtifactsRemote(ctx, machine, tempRoot, workspaceRoot, adapterType); err != nil {
		return fmt.Errorf("sync remote project conversation skills: %w", err)
	}
	return nil
}

func (m *projectConversationRuntimeManager) copyConversationWorkspaceArtifactsRemote(
	ctx context.Context,
	machine catalogdomain.Machine,
	localRoot string,
	remoteWorkspaceRoot string,
	adapterType string,
) error {
	target, err := workflowservice.ResolveSkillTargetForRuntime(remoteWorkspaceRoot, adapterType)
	if err != nil {
		return err
	}
	relativePaths := conversationWorkspaceArtifactPaths(localRoot, adapterType)
	skillsRelativePath, err := filepath.Rel(remoteWorkspaceRoot, target.SkillsDir)
	if err != nil {
		return fmt.Errorf("derive remote skills relative path: %w", err)
	}
	transport, err := m.resolveTransport(machine)
	if err != nil {
		return err
	}
	return transport.SyncArtifacts(ctx, machine, machinetransport.SyncArtifactsRequest{
		LocalRoot:   localRoot,
		TargetRoot:  remoteWorkspaceRoot,
		Paths:       relativePaths,
		RemovePaths: []string{filepath.ToSlash(skillsRelativePath), ".openase/bin"},
	})
}

func (m *projectConversationRuntimeManager) applyGitHubWorkspaceAuth(
	ctx context.Context,
	projectID uuid.UUID,
	request workspaceinfra.SetupRequest,
) (workspaceinfra.SetupRequest, error) {
	return githubauthservice.ApplyWorkspaceAuth(ctx, m.githubAuth, projectID, request)
}

func (m *projectConversationRuntimeManager) resolveTransport(machine catalogdomain.Machine) (machinetransport.Transport, error) {
	if m == nil || m.transports == nil {
		return nil, fmt.Errorf("chat machine transport resolver unavailable for machine %s", machine.Name)
	}
	return m.transports.Resolve(machine)
}

func (m *projectConversationRuntimeManager) runRemoteRuntimePreflight(
	ctx context.Context,
	machine catalogdomain.Machine,
	providerItem catalogdomain.AgentProvider,
	workspacePath string,
) error {
	if machine.Host == catalogdomain.LocalMachineHost {
		return nil
	}

	transport, err := m.resolveTransport(machine)
	if err != nil {
		return err
	}
	if transport.Mode() != catalogdomain.MachineConnectionModeWSListener {
		return nil
	}

	command := strings.TrimSpace(providerItem.CliCommand)
	if machine.AgentCLIPath != nil && strings.TrimSpace(*machine.AgentCLIPath) != "" {
		command = strings.TrimSpace(*machine.AgentCLIPath)
	}
	return machinetransport.RunRemoteRuntimePreflight(ctx, transport, machine, machinetransport.RuntimePreflightSpec{
		WorkingDirectory: workspacePath,
		AgentCommand:     command,
		Environment:      append([]string(nil), machine.EnvVars...),
	})
}
