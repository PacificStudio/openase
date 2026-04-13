package chat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/provider"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

var (
	ErrConversationNotFound         = chatrepo.ErrNotFound
	ErrConversationConflict         = chatrepo.ErrConflict
	ErrConversationTurnActive       = chatrepo.ErrTurnAlreadyActive
	ErrConversationTurnNotActive    = fmt.Errorf("%w: project conversation does not have an active turn", domain.ErrConflict)
	ErrConversationInterruptPending = domain.ErrInterruptPending
	ErrPendingInterruptNotFound     = chatrepo.ErrNotFound
	ErrConversationRuntimeAbsent    = fmt.Errorf("chat conversation runtime is unavailable")
)

type projectConversationCatalog interface {
	ListOrganizations(ctx context.Context) ([]catalogdomain.Organization, error)
	ListProjects(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.Project, error)
	GetProject(ctx context.Context, id uuid.UUID) (catalogdomain.Project, error)
	GetMachine(ctx context.Context, id uuid.UUID) (catalogdomain.Machine, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (catalogdomain.AgentProvider, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.AgentProvider, error)
	ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]catalogdomain.ProjectRepo, error)
	ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]catalogdomain.TicketRepoScope, error)
	ListActivityEvents(ctx context.Context, input catalogdomain.ListActivityEvents) (catalogdomain.ActivityEventPage, error)
}

type projectConversationSkillSync interface {
	RefreshSkills(ctx context.Context, input workflowservice.RefreshSkillsInput) (workflowservice.RefreshSkillsResult, error)
}

type projectConversationAgentPlatform interface {
	IssueToken(ctx context.Context, input agentplatform.IssueInput) (agentplatform.IssuedToken, error)
}

type projectConversationSecretManager interface {
	ResolveBoundForRuntime(context.Context, secretsservice.ResolveBoundRuntimeInput) ([]secretsdomain.ResolvedSecret, error)
}

type projectConversationActivityEmitter interface {
	Emit(context.Context, activitysvc.RecordInput) (*catalogdomain.ActivityEvent, error)
}

type liveProjectConversation struct {
	principal domain.ProjectConversationPrincipal
	provider  catalogdomain.AgentProvider
	machine   catalogdomain.Machine
	runtime   Runtime
	codex     projectConversationCodexRuntime
	interrupt projectConversationInterruptRuntime
	turnStop  projectConversationTurnStopRuntime
	workspace provider.AbsolutePath
}

type projectConversationUsageHighWater struct {
	inputTokens         int64
	outputTokens        int64
	cachedInputTokens   int64
	cacheCreationTokens int64
	reasoningTokens     int64
	promptTokens        int64
	candidateTokens     int64
	toolTokens          int64
	totalTokens         int64
	costAmount          float64
	hasCostAmount       bool
}

type projectConversationCodexRuntime interface {
	Runtime
	EnsureSession(ctx context.Context, input RuntimeTurnInput) error
	RespondInterrupt(ctx context.Context, input RuntimeInterruptResponseInput) (TurnStream, error)
	InterruptTurn(ctx context.Context, sessionID SessionID) (RuntimeSessionAnchor, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationInterruptRuntime interface {
	Runtime
	RespondInterrupt(ctx context.Context, input RuntimeInterruptResponseInput) (TurnStream, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationTurnStopRuntime interface {
	Runtime
	InterruptTurn(ctx context.Context, sessionID SessionID) (RuntimeSessionAnchor, error)
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationSessionAnchorer interface {
	SessionAnchor(sessionID SessionID) RuntimeSessionAnchor
}

type projectConversationCore struct {
	conversations projectConversationConversationStore
	entries       projectConversationEntryStore
	interrupts    projectConversationInterruptStore
	runtimeStore  projectConversationRuntimeStore
	catalog       projectConversationCatalog
	tickets       ticketReader
	workflows     workflowReader
	skillSync     projectConversationSkillSync

	localProcessManager provider.AgentCLIProcessManager
	sshPool             *sshinfra.Pool
	platformAPIURL      string
	agentPlatform       projectConversationAgentPlatform
	githubAuth          githubauthservice.TokenResolver
	secretResolver      RuntimeEnvironmentResolver
	secretManager       projectConversationSecretManager
	activityEmitter     projectConversationActivityEmitter

	streamBroker  *projectConversationStreamBroker
	muxBroker     *projectConversationMuxBroker
	turnLocks     userLockRegistry
	promptBuilder *Service
}

type ProjectConversationService struct {
	logger *slog.Logger

	core *projectConversationCore

	runtimeManager  *projectConversationRuntimeManager
	newCodexRuntime func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error)
}

func NewProjectConversationService(
	logger *slog.Logger,
	stores projectConversationStoreSource,
	catalog projectConversationCatalog,
	tickets ticketReader,
	workflows workflowReader,
	localProcessManager provider.AgentCLIProcessManager,
	sshPool *sshinfra.Pool,
) *ProjectConversationService {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	service := &ProjectConversationService{
		logger: logger.With("component", "project-conversation-service"),
		core: &projectConversationCore{
			conversations:       stores,
			entries:             stores,
			interrupts:          stores,
			runtimeStore:        stores,
			catalog:             catalog,
			tickets:             tickets,
			workflows:           workflows,
			localProcessManager: localProcessManager,
			sshPool:             sshPool,
			streamBroker:        newProjectConversationStreamBroker(),
			muxBroker:           newProjectConversationMuxBroker(),
		},
	}
	if syncer, ok := workflows.(projectConversationSkillSync); ok {
		service.core.skillSync = syncer
	}
	service.core.promptBuilder = &Service{
		logger:    service.logger,
		catalog:   catalog,
		tickets:   tickets,
		workflows: workflows,
	}
	service.newCodexRuntime = func(manager provider.AgentCLIProcessManager) (projectConversationCodexRuntime, error) {
		adapter, err := newCodexAdapterForManager(manager)
		if err != nil {
			return nil, err
		}
		runtime := NewCodexRuntime(adapter)
		runtime.ConfigureSecretResolver(service.core.secretResolver)
		return runtime, nil
	}
	service.runtimeManager = newProjectConversationRuntimeManager(
		service.logger,
		catalog,
		service.core.runtimeStore,
		localProcessManager,
		sshPool,
		service.newCodexRuntime,
	)
	service.runtimeManager.ConfigureSkillSync(service.core.skillSync)
	return service
}

func (s *ProjectConversationService) ConfigurePlatformEnvironment(
	apiURL string,
	platform projectConversationAgentPlatform,
) {
	if s == nil {
		return
	}
	s.core.platformAPIURL = strings.TrimSpace(apiURL)
	s.core.agentPlatform = platform
}

func (s *ProjectConversationService) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	if s == nil {
		return
	}
	s.core.githubAuth = resolver
	if s.runtimeManager != nil {
		s.runtimeManager.ConfigureGitHubCredentials(resolver)
	}
}

func (s *ProjectConversationService) ConfigureSecretResolver(resolver RuntimeEnvironmentResolver) {
	if s == nil {
		return
	}
	s.core.secretResolver = resolver
	if s.runtimeManager != nil {
		s.runtimeManager.ConfigureSecretResolver(resolver)
	}
}

func (s *ProjectConversationService) ConfigureSecretManager(manager projectConversationSecretManager) {
	if s == nil {
		return
	}
	s.core.secretManager = manager
}

func (s *ProjectConversationService) ConfigureActivityEmitter(emitter projectConversationActivityEmitter) {
	if s == nil {
		return
	}
	s.core.activityEmitter = emitter
}

func projectConversationTurnLockKey(conversation domain.Conversation) UserID {
	return UserID("conversation:" + conversation.ID.String())
}

func isStableLocalProjectConversationUser(userID UserID) bool {
	return strings.TrimSpace(userID.String()) == strings.TrimSpace(LocalProjectConversationUserID.String())
}
