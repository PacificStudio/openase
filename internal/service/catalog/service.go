package catalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var (
	ErrNotFound                  = domain.ErrNotFound
	ErrConflict                  = domain.ErrConflict
	ErrInvalidInput              = domain.ErrInvalidInput
	ErrMachineTestingUnavailable = errors.New("machine testing unavailable")
	ErrMachineProbeFailed        = errors.New("machine probe failed")
	ErrMachineHealthUnavailable  = errors.New("machine health refresh unavailable")
)

type MachineTester interface {
	TestConnection(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error)
}

type MachineHealthCollector interface {
	CollectReachability(ctx context.Context, machine domain.Machine) (domain.MachineReachability, error)
	CollectSystemResources(ctx context.Context, machine domain.Machine) (domain.MachineSystemResources, error)
	CollectGPUResources(ctx context.Context, machine domain.Machine) (domain.MachineGPUResources, error)
	CollectAgentEnvironment(ctx context.Context, machine domain.Machine) (domain.MachineAgentEnvironment, error)
	CollectFullAudit(ctx context.Context, machine domain.Machine) (domain.MachineFullAudit, error)
}

type Option func(*service)

type ProjectStatusBootstrapper interface {
	BootstrapProjectStatuses(ctx context.Context, projectID uuid.UUID) error
}

type ProjectStatusBootstrapperFunc func(ctx context.Context, projectID uuid.UUID) error

func (fn ProjectStatusBootstrapperFunc) BootstrapProjectStatuses(ctx context.Context, projectID uuid.UUID) error {
	return fn(ctx, projectID)
}

func WithProjectStatusBootstrapper(bootstrapper ProjectStatusBootstrapper) Option {
	return func(s *service) {
		if bootstrapper == nil {
			return
		}
		s.projectStatusBootstrapper = bootstrapper
	}
}

func WithMachineHealthCollector(collector MachineHealthCollector) Option {
	return func(s *service) {
		if collector == nil {
			return
		}
		s.machineHealthCollector = collector
	}
}

type service struct {
	repo                      Repository
	resolver                  provider.ExecutableResolver
	machineTester             MachineTester
	machineHealthCollector    MachineHealthCollector
	projectStatusBootstrapper ProjectStatusBootstrapper
}

func New(repo Repository, resolver provider.ExecutableResolver, machineTester MachineTester, opts ...Option) Service {
	svc := &service{repo: repo, resolver: resolver, machineTester: machineTester}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc
}

func (s *service) ListOrganizations(ctx context.Context) ([]domain.Organization, error) {
	return s.repo.ListOrganizations(ctx)
}

func (s *service) CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error) {
	item, err := s.repo.CreateOrganization(ctx, input)
	if err != nil {
		return domain.Organization{}, err
	}

	if item.DefaultAgentProviderID != nil {
		return item, nil
	}

	providers, err := s.ListAgentProviders(ctx, item.ID)
	if err != nil {
		return domain.Organization{}, err
	}

	defaultProviderID := preferredAvailableProviderID(providers)
	if defaultProviderID == nil {
		return item, nil
	}

	return s.repo.UpdateOrganization(ctx, domain.UpdateOrganization{
		ID:                     item.ID,
		Name:                   item.Name,
		Slug:                   item.Slug,
		DefaultAgentProviderID: defaultProviderID,
	})
}

func (s *service) GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	return s.repo.GetOrganization(ctx, id)
}

func (s *service) UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	return s.repo.UpdateOrganization(ctx, input)
}

func (s *service) ArchiveOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	return s.repo.ArchiveOrganization(ctx, id)
}

func (s *service) ListMachines(ctx context.Context, organizationID uuid.UUID) ([]domain.Machine, error) {
	return s.repo.ListMachines(ctx, organizationID)
}

func (s *service) CreateMachine(ctx context.Context, input domain.CreateMachine) (domain.Machine, error) {
	return s.repo.CreateMachine(ctx, input)
}

func (s *service) GetMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error) {
	return s.repo.GetMachine(ctx, id)
}

func (s *service) UpdateMachine(ctx context.Context, input domain.UpdateMachine) (domain.Machine, error) {
	return s.repo.UpdateMachine(ctx, input)
}

func (s *service) DeleteMachine(ctx context.Context, id uuid.UUID) (domain.Machine, error) {
	return s.repo.DeleteMachine(ctx, id)
}

func (s *service) GetWorkspaceDashboardSummary(ctx context.Context) (domain.WorkspaceDashboardSummary, error) {
	return s.repo.GetWorkspaceDashboardSummary(ctx)
}

func (s *service) GetOrganizationDashboardSummary(ctx context.Context, organizationID uuid.UUID) (domain.OrganizationDashboardSummary, error) {
	return s.repo.GetOrganizationDashboardSummary(ctx, organizationID)
}

func (s *service) GetOrganizationTokenUsage(
	ctx context.Context,
	input domain.GetOrganizationTokenUsage,
) (domain.OrganizationTokenUsageReport, error) {
	return s.repo.GetOrganizationTokenUsage(ctx, input)
}

func (s *service) GetProjectTokenUsage(
	ctx context.Context,
	input domain.GetProjectTokenUsage,
) (domain.ProjectTokenUsageReport, error) {
	return s.repo.GetProjectTokenUsage(ctx, input)
}

func (s *service) TestMachineConnection(ctx context.Context, id uuid.UUID) (domain.Machine, domain.MachineProbe, error) {
	if s.machineTester == nil {
		return domain.Machine{}, domain.MachineProbe{}, ErrMachineTestingUnavailable
	}

	machine, err := s.repo.GetMachine(ctx, id)
	if err != nil {
		return domain.Machine{}, domain.MachineProbe{}, err
	}

	probe, err := s.machineTester.TestConnection(ctx, machine)
	if err != nil {
		checkedAt := probe.CheckedAt
		if checkedAt.IsZero() {
			checkedAt = time.Now().UTC()
		}
		updateErr := s.repo.RecordMachineProbe(ctx, domain.RecordMachineProbe{
			ID:              id,
			Status:          domainMachineFailureStatus(machine),
			LastHeartbeatAt: checkedAt,
			Resources:       mergeMachineProbeResources(machine.Resources, probe, checkedAt, err),
			DetectedOS:      probe.DetectedOS,
			DetectedArch:    probe.DetectedArch,
			DetectionStatus: probe.DetectionStatus,
		})
		if updateErr != nil {
			return domain.Machine{}, domain.MachineProbe{}, fmt.Errorf("%w: %v (status update failed: %v)", ErrMachineProbeFailed, err, updateErr)
		}
		return domain.Machine{}, domain.MachineProbe{}, fmt.Errorf("%w: %v", ErrMachineProbeFailed, err)
	}

	if err := s.repo.RecordMachineProbe(ctx, domain.RecordMachineProbe{
		ID:              id,
		Status:          domainMachineSuccessStatus(machine),
		LastHeartbeatAt: probe.CheckedAt,
		Resources:       mergeMachineProbeResources(machine.Resources, probe, probe.CheckedAt, nil),
		DetectedOS:      probe.DetectedOS,
		DetectedArch:    probe.DetectedArch,
		DetectionStatus: probe.DetectionStatus,
	}); err != nil {
		return domain.Machine{}, domain.MachineProbe{}, err
	}

	updated, err := s.repo.GetMachine(ctx, id)
	if err != nil {
		return domain.Machine{}, domain.MachineProbe{}, err
	}

	return updated, probe, nil
}

func (s *service) RefreshMachineHealth(ctx context.Context, id uuid.UUID) (domain.Machine, error) {
	if s.machineHealthCollector == nil {
		return domain.Machine{}, ErrMachineHealthUnavailable
	}

	machine, err := s.repo.GetMachine(ctx, id)
	if err != nil {
		return domain.Machine{}, err
	}

	resources := cloneResources(machine.Resources)
	status := machine.Status
	if status != domain.MachineStatusMaintenance {
		status = domain.MachineStatusOnline
	}

	reachability, reachabilityErr := s.machineHealthCollector.CollectReachability(ctx, machine)
	checkedAt := reachability.CheckedAt.UTC()
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
		reachability.CheckedAt = checkedAt
	}
	updateMonitorL1Resources(resources, reachability)

	softReachabilityFailure := reachabilityErr != nil || !reachability.Reachable
	hardReachabilityFailure := false
	if softReachabilityFailure {
		failures := machineMonitorFailures(resources) + 1
		setMachineMonitorFailures(resources, failures)
		if reachabilityErr != nil {
			setMachineMonitorError(resources, "l1", reachabilityErr.Error())
		}
		if machine.Host != domain.LocalMachineHost {
			hardReachabilityFailure = true
		}
	} else {
		setMachineMonitorFailures(resources, 0)
		clearMachineMonitorError(resources, "l1")
	}

	systemProbeFailure := false
	level3ProbeFailure := false
	level4ProbeFailure := false
	level5ProbeFailure := false

	if !softReachabilityFailure && !hardReachabilityFailure {
		systemResources, err := s.machineHealthCollector.CollectSystemResources(ctx, machine)
		if err != nil {
			systemProbeFailure = true
			setMachineMonitorError(resources, "l2", err.Error())
		} else {
			updateMonitorL2Resources(resources, systemResources)
			clearMachineMonitorError(resources, "l2")
		}

		gpuResources, err := s.machineHealthCollector.CollectGPUResources(ctx, machine)
		if err != nil {
			level3ProbeFailure = true
			setMachineMonitorError(resources, "l3", err.Error())
		} else {
			updateMonitorL3Resources(resources, gpuResources)
			clearMachineMonitorError(resources, "l3")
		}

		agentEnvironment, err := s.machineHealthCollector.CollectAgentEnvironment(ctx, machine)
		if err != nil {
			level4ProbeFailure = true
			setMachineMonitorError(resources, "l4", err.Error())
		} else {
			updateMonitorL4Resources(resources, agentEnvironment)
			clearMachineMonitorError(resources, "l4")
		}

		fullAudit, err := s.machineHealthCollector.CollectFullAudit(ctx, machine)
		if err != nil {
			level5ProbeFailure = true
			setMachineMonitorError(resources, "l5", err.Error())
		} else {
			updateMonitorL5Resources(resources, fullAudit)
			clearMachineMonitorError(resources, "l5")
		}
	}

	if machine.Status != domain.MachineStatusMaintenance {
		switch {
		case hardReachabilityFailure:
			status = domain.MachineStatusOffline
		case softReachabilityFailure || systemProbeFailure || level3ProbeFailure || level4ProbeFailure || level5ProbeFailure || machineHasLowDisk(resources):
			status = domain.MachineStatusDegraded
		default:
			status = domain.MachineStatusOnline
		}
	}

	if err := s.repo.RecordMachineProbe(ctx, domain.RecordMachineProbe{
		ID:              machine.ID,
		Status:          status,
		LastHeartbeatAt: checkedAt,
		Resources:       resources,
	}); err != nil {
		return domain.Machine{}, err
	}

	return s.repo.GetMachine(ctx, id)
}

func (s *service) ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx, organizationID)
}

func (s *service) CreateProject(ctx context.Context, input domain.CreateProject) (domain.Project, error) {
	item, err := s.repo.CreateProject(ctx, input)
	if err != nil {
		return domain.Project{}, err
	}
	if s.projectStatusBootstrapper == nil {
		return item, nil
	}
	if err := s.projectStatusBootstrapper.BootstrapProjectStatuses(ctx, item.ID); err != nil {
		return domain.Project{}, fmt.Errorf("seed default project statuses: %w", err)
	}
	return item, nil
}

func (s *service) GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	return s.repo.GetProject(ctx, id)
}

func (s *service) UpdateProject(ctx context.Context, input domain.UpdateProject) (domain.Project, error) {
	return s.repo.UpdateProject(ctx, input)
}

func (s *service) ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	return s.repo.ArchiveProject(ctx, id)
}

func (s *service) ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRepo, error) {
	return s.repo.ListProjectRepos(ctx, projectID)
}

func (s *service) CreateProjectRepo(ctx context.Context, input domain.CreateProjectRepo) (domain.ProjectRepo, error) {
	return s.repo.CreateProjectRepo(ctx, input)
}

func (s *service) GetProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	return s.repo.GetProjectRepo(ctx, projectID, id)
}

func (s *service) UpdateProjectRepo(ctx context.Context, input domain.UpdateProjectRepo) (domain.ProjectRepo, error) {
	return s.repo.UpdateProjectRepo(ctx, input)
}

func (s *service) DeleteProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	return s.repo.DeleteProjectRepo(ctx, projectID, id)
}

func (s *service) ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]domain.TicketRepoScope, error) {
	return s.repo.ListTicketRepoScopes(ctx, projectID, ticketID)
}

func (s *service) CreateTicketRepoScope(ctx context.Context, input domain.CreateTicketRepoScope) (domain.TicketRepoScope, error) {
	return s.repo.CreateTicketRepoScope(ctx, input)
}

func (s *service) GetTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error) {
	return s.repo.GetTicketRepoScope(ctx, projectID, ticketID, id)
}

func (s *service) UpdateTicketRepoScope(ctx context.Context, input domain.UpdateTicketRepoScope) (domain.TicketRepoScope, error) {
	return s.repo.UpdateTicketRepoScope(ctx, input)
}

func (s *service) DeleteTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error) {
	return s.repo.DeleteTicketRepoScope(ctx, projectID, ticketID, id)
}

func domainMachineFailureStatus(machine domain.Machine) domain.MachineStatus {
	if machine.Host == domain.LocalMachineHost {
		return domain.MachineStatusDegraded
	}
	return domain.MachineStatusOffline
}

func domainMachineSuccessStatus(machine domain.Machine) domain.MachineStatus {
	if machine.Status == domain.MachineStatusMaintenance {
		return domain.MachineStatusOnline
	}
	return machine.Status
}

func cloneResources(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = cloneAnyValue(value)
	}

	return cloned
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, nestedValue := range typed {
			cloned[key] = cloneAnyValue(nestedValue)
		}
		return cloned
	case []map[string]any:
		cloned := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloneResources(item))
		}
		return cloned
	case []any:
		cloned := make([]any, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloneAnyValue(item))
		}
		return cloned
	default:
		return value
	}
}

func mergeMachineProbeResources(
	base map[string]any,
	probe domain.MachineProbe,
	checkedAt time.Time,
	probeErr error,
) map[string]any {
	merged := cloneResources(base)
	if probe.Transport != "" {
		merged["transport"] = probe.Transport
	}

	connectionTest := map[string]any{
		"checked_at":   checkedAt.UTC().Format(time.RFC3339),
		"transport":    probe.Transport,
		"last_success": probeErr == nil,
	}
	if probeErr != nil {
		connectionTest["error"] = probeErr.Error()
	}
	if probe.Output != "" {
		connectionTest["output"] = probe.Output
	}
	if len(probe.Resources) > 0 {
		connectionTest["resources"] = cloneResources(probe.Resources)
	}
	merged["connection_test"] = connectionTest

	return merged
}

const (
	lowDiskThresholdGB        = 5.0
	lowMemoryThresholdPercent = 10.0
	fullGPUMemoryThresholdGB  = 0.5
)

func machineMonitorFailures(resources map[string]any) int {
	monitor, ok := nestedMap(resources, "monitor")
	if !ok {
		return 0
	}
	levelMap, ok := nestedMap(monitor, "l1")
	if !ok {
		return 0
	}
	switch raw := levelMap["consecutive_failures"].(type) {
	case int:
		return raw
	case float64:
		return int(raw)
	default:
		return 0
	}
}

func setMachineMonitorFailures(resources map[string]any, failures int) {
	levelMap := ensureMonitorLevel(resources, "l1")
	levelMap["consecutive_failures"] = failures
}

func setMachineMonitorError(resources map[string]any, level string, message string) {
	levelMap := ensureMonitorLevel(resources, level)
	levelMap["error"] = strings.TrimSpace(message)
}

func clearMachineMonitorError(resources map[string]any, level string) {
	levelMap := ensureMonitorLevel(resources, level)
	delete(levelMap, "error")
}

func updateMonitorL1Resources(resources map[string]any, reachability domain.MachineReachability) {
	levelMap := ensureMonitorLevel(resources, "l1")
	levelMap["checked_at"] = reachability.CheckedAt.UTC().Format(time.RFC3339)
	levelMap["transport"] = reachability.Transport
	levelMap["reachable"] = reachability.Reachable
	levelMap["latency_ms"] = reachability.LatencyMS
	if strings.TrimSpace(reachability.FailureCause) != "" {
		levelMap["failure_cause"] = reachability.FailureCause
	} else {
		delete(levelMap, "failure_cause")
	}

	resources["transport"] = reachability.Transport
	resources["checked_at"] = reachability.CheckedAt.UTC().Format(time.RFC3339)
	resources["last_success"] = reachability.Reachable
}

func updateMonitorL2Resources(resources map[string]any, systemResources domain.MachineSystemResources) {
	levelMap := ensureMonitorLevel(resources, "l2")
	levelMap["checked_at"] = systemResources.CollectedAt.UTC().Format(time.RFC3339)
	levelMap["memory_low"] = systemResources.MemoryAvailablePercent < lowMemoryThresholdPercent
	levelMap["disk_low"] = systemResources.DiskAvailableGB < lowDiskThresholdGB

	resources["cpu_cores"] = systemResources.CPUCores
	resources["cpu_usage_percent"] = systemResources.CPUUsagePercent
	resources["memory_total_gb"] = systemResources.MemoryTotalGB
	resources["memory_used_gb"] = systemResources.MemoryUsedGB
	resources["memory_available_gb"] = systemResources.MemoryAvailableGB
	resources["disk_total_gb"] = systemResources.DiskTotalGB
	resources["disk_available_gb"] = systemResources.DiskAvailableGB
	resources["collected_at"] = systemResources.CollectedAt.UTC().Format(time.RFC3339)
}

func updateMonitorL3Resources(resources map[string]any, gpuResources domain.MachineGPUResources) {
	levelMap := ensureMonitorLevel(resources, "l3")
	levelMap["checked_at"] = gpuResources.CollectedAt.UTC().Format(time.RFC3339)
	levelMap["available"] = gpuResources.Available

	if !gpuResources.Available {
		resources["gpu"] = []map[string]any{}
		levelMap["gpu_dispatchable"] = false
		resources["gpu_dispatchable"] = false
		return
	}

	gpus := make([]map[string]any, 0, len(gpuResources.GPUs))
	gpuDispatchable := false
	for _, gpu := range gpuResources.GPUs {
		if gpu.MemoryTotalGB-gpu.MemoryUsedGB > fullGPUMemoryThresholdGB {
			gpuDispatchable = true
		}
		gpus = append(gpus, map[string]any{
			"index":               gpu.Index,
			"name":                gpu.Name,
			"memory_total_gb":     gpu.MemoryTotalGB,
			"memory_used_gb":      gpu.MemoryUsedGB,
			"utilization_percent": gpu.UtilizationPercent,
		})
	}
	slices.SortFunc(gpus, func(left, right map[string]any) int {
		return compareAnyInt(left["index"], right["index"])
	})

	levelMap["gpu_dispatchable"] = gpuDispatchable
	resources["gpu"] = gpus
	resources["gpu_dispatchable"] = gpuDispatchable
}

func updateMonitorL4Resources(resources map[string]any, agentEnvironment domain.MachineAgentEnvironment) {
	levelMap := ensureMonitorLevel(resources, "l4")
	levelMap["checked_at"] = agentEnvironment.CollectedAt.UTC().Format(time.RFC3339)
	levelMap["agent_dispatchable"] = agentEnvironment.Dispatchable

	environmentSummary := make(map[string]any, len(agentEnvironment.CLIs))
	for _, cli := range agentEnvironment.CLIs {
		snapshot := map[string]any{
			"installed":   cli.Installed,
			"version":     cli.Version,
			"auth_status": string(cli.AuthStatus),
			"auth_mode":   string(cli.AuthMode),
			"ready":       cli.Ready,
		}
		levelMap[cli.Name] = cloneResources(snapshot)
		environmentSummary[cli.Name] = snapshot
	}

	resources["agent_dispatchable"] = agentEnvironment.Dispatchable
	resources["agent_environment_checked_at"] = agentEnvironment.CollectedAt.UTC().Format(time.RFC3339)
	resources["agent_environment"] = environmentSummary
}

func updateMonitorL5Resources(resources map[string]any, fullAudit domain.MachineFullAudit) {
	levelMap := ensureMonitorLevel(resources, "l5")
	levelMap["checked_at"] = fullAudit.CollectedAt.UTC().Format(time.RFC3339)

	gitSummary := map[string]any{
		"installed":  fullAudit.Git.Installed,
		"user_name":  fullAudit.Git.UserName,
		"user_email": fullAudit.Git.UserEmail,
	}
	ghSummary := map[string]any{
		"installed":   fullAudit.GitHubCLI.Installed,
		"auth_status": string(fullAudit.GitHubCLI.AuthStatus),
	}
	githubTokenProbe := map[string]any{
		"state":       string(fullAudit.GitHubTokenProbe.State),
		"configured":  fullAudit.GitHubTokenProbe.Configured,
		"valid":       fullAudit.GitHubTokenProbe.Valid,
		"permissions": append([]string(nil), fullAudit.GitHubTokenProbe.Permissions...),
		"repo_access": string(fullAudit.GitHubTokenProbe.RepoAccess),
		"last_error":  fullAudit.GitHubTokenProbe.LastError,
	}
	if fullAudit.GitHubTokenProbe.CheckedAt != nil {
		githubTokenProbe["checked_at"] = fullAudit.GitHubTokenProbe.CheckedAt.UTC().Format(time.RFC3339)
	}
	networkSummary := map[string]any{
		"github_reachable": fullAudit.Network.GitHubReachable,
		"pypi_reachable":   fullAudit.Network.PyPIReachable,
		"npm_reachable":    fullAudit.Network.NPMReachable,
	}

	levelMap["git"] = cloneResources(gitSummary)
	levelMap["gh_cli"] = cloneResources(ghSummary)
	levelMap["github_token_probe"] = cloneResources(githubTokenProbe)
	levelMap["network"] = cloneResources(networkSummary)
	resources["full_audit"] = map[string]any{
		"checked_at":         fullAudit.CollectedAt.UTC().Format(time.RFC3339),
		"git":                gitSummary,
		"gh_cli":             ghSummary,
		"github_token_probe": githubTokenProbe,
		"network":            networkSummary,
	}
}

func machineHasLowDisk(resources map[string]any) bool {
	value, ok := resources["disk_available_gb"]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case float64:
		return typed < lowDiskThresholdGB
	case float32:
		return float64(typed) < lowDiskThresholdGB
	case int:
		return float64(typed) < lowDiskThresholdGB
	default:
		return false
	}
}

func ensureMonitorLevel(resources map[string]any, level string) map[string]any {
	monitor, ok := nestedMap(resources, "monitor")
	if !ok {
		monitor = map[string]any{}
		resources["monitor"] = monitor
	}
	levelMap, ok := nestedMap(monitor, level)
	if !ok {
		levelMap = map[string]any{}
		monitor[level] = levelMap
	}
	return levelMap
}

func nestedMap(resources map[string]any, key string) (map[string]any, bool) {
	raw, ok := resources[key]
	if !ok {
		return nil, false
	}
	typed, ok := raw.(map[string]any)
	return typed, ok
}

func compareAnyInt(left any, right any) int {
	leftValue := anyToInt(left)
	rightValue := anyToInt(right)
	switch {
	case leftValue < rightValue:
		return -1
	case leftValue > rightValue:
		return 1
	default:
		return 0
	}
}

func anyToInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
