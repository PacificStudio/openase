package catalog

import repocatalog "github.com/BetterAndBetterII/openase/internal/repo/catalog"

var (
	_ OrganizationService   = (*service)(nil)
	_ MachineService        = (*service)(nil)
	_ ProjectService        = (*service)(nil)
	_ ProjectRepoService    = (*service)(nil)
	_ DashboardQueryService = (*service)(nil)
	_ UsageQueryService     = (*service)(nil)
	_ AgentProviderService  = (*service)(nil)
	_ AgentService          = (*service)(nil)
	_ AgentRunQueryService  = (*service)(nil)
	_ ActivityQueryService  = (*service)(nil)
	_ Service               = (*service)(nil)
)

var (
	_ OrganizationRepository   = (*repocatalog.EntRepository)(nil)
	_ MachineRepository        = (*repocatalog.EntRepository)(nil)
	_ ProjectRepository        = (*repocatalog.EntRepository)(nil)
	_ ProjectRepoRepository    = (*repocatalog.EntRepository)(nil)
	_ DashboardQueryRepository = (*repocatalog.EntRepository)(nil)
	_ UsageQueryRepository     = (*repocatalog.EntRepository)(nil)
	_ AgentProviderRepository  = (*repocatalog.EntRepository)(nil)
	_ AgentRepository          = (*repocatalog.EntRepository)(nil)
	_ AgentRunQueryRepository  = (*repocatalog.EntRepository)(nil)
	_ ActivityQueryRepository  = (*repocatalog.EntRepository)(nil)
	_ Repository               = (*repocatalog.EntRepository)(nil)
)
