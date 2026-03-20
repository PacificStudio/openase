import type { LayoutLoad } from './$types'
import type { AgentProvider, Organization, Project } from '$lib/api/contracts'

type OrgResponse = {
  organizations?: Organization[]
}

type ProjectResponse = {
  projects?: Project[]
}

type ProviderResponse = {
  providers?: AgentProvider[]
}

type AgentResponse = {
  agents?: unknown[]
}

function emptyLayoutData() {
  return {
    organizations: [],
    projects: [],
    currentOrg: null,
    currentProject: null,
    providers: [],
    agentCount: 0,
  }
}

function selectById<T extends { id: string }>(items: T[], requestedId: string | null) {
  if (items.length === 0) {
    return null
  }

  if (!requestedId) {
    return items[0]
  }

  return items.find((item) => item.id === requestedId) ?? items[0]
}

async function loadOrganizations(fetcher: typeof fetch) {
  const response = await fetcher('/api/v1/orgs')
  if (!response.ok) {
    return []
  }

  const data = (await response.json()) as OrgResponse
  return data.organizations ?? []
}

async function loadOrgContext(fetcher: typeof fetch, orgId: string) {
  const [projectResponse, providerResponse] = await Promise.all([
    fetcher(`/api/v1/orgs/${orgId}/projects`),
    fetcher(`/api/v1/orgs/${orgId}/providers`),
  ])

  const projectData = projectResponse.ok
    ? ((await projectResponse.json()) as ProjectResponse)
    : { projects: [] }
  const providerData = providerResponse.ok
    ? ((await providerResponse.json()) as ProviderResponse)
    : { providers: [] }

  return {
    projects: projectData.projects ?? [],
    providers: providerData.providers ?? [],
  }
}

async function loadAgentCount(fetcher: typeof fetch, projectId: string | null) {
  if (!projectId) {
    return 0
  }

  const response = await fetcher(`/api/v1/projects/${projectId}/agents`)
  if (!response.ok) {
    return 0
  }

  const data = (await response.json()) as AgentResponse
  return data.agents?.length ?? 0
}

export const load: LayoutLoad = async ({ fetch, url }) => {
  const requestedOrgId = url.searchParams.get('orgId')
  const requestedProjectId = url.searchParams.get('projectId')
  const organizations = await loadOrganizations(fetch)
  const currentOrg = selectById(organizations, requestedOrgId)
  if (!currentOrg) {
    return emptyLayoutData()
  }

  const { projects, providers } = await loadOrgContext(fetch, currentOrg.id)
  const currentProject = selectById(projects, requestedProjectId)
  const agentCount = await loadAgentCount(fetch, currentProject?.id ?? null)

  return {
    organizations,
    projects,
    currentOrg,
    currentProject,
    providers,
    agentCount,
  }
}
