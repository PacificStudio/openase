import type { AgentProvider, AppContextPayload, Organization, Project } from './contracts'

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

type AppFetch = typeof globalThis.fetch

export async function loadOrganizations(fetch: AppFetch) {
  const orgResponse = await fetch('/api/v1/orgs')
  if (!orgResponse.ok) {
    return []
  }

  const orgData = (await orgResponse.json()) as OrgResponse
  return orgData.organizations ?? []
}

export async function loadAppContext(
  fetch: AppFetch,
  input?: {
    orgId?: string | null
    projectId?: string | null
  },
) {
  const response = await fetch(
    withQuery('/api/v1/app-context', {
      org_id: input?.orgId ?? undefined,
      project_id: input?.projectId ?? undefined,
    }),
  )
  if (!response.ok) {
    return {
      organizations: [] as Organization[],
      projects: [] as Project[],
      providers: [] as AgentProvider[],
      agentCount: 0,
    }
  }

  const payload = (await response.json()) as AppContextPayload
  return {
    organizations: payload.organizations ?? [],
    projects: payload.projects ?? [],
    providers: payload.providers ?? [],
    agentCount: payload.agent_count ?? 0,
  }
}

export async function loadOrganizationProjects(fetch: AppFetch, orgId: string) {
  const projectResponse = await fetch(`/api/v1/orgs/${orgId}/projects`)
  if (!projectResponse.ok) {
    return []
  }

  const projectData = (await projectResponse.json()) as ProjectResponse
  return projectData.projects ?? []
}

export async function loadOrganizationContext(fetch: AppFetch, orgId: string) {
  const [projects, providers] = await Promise.all([
    loadOrganizationProjects(fetch, orgId),
    loadOrganizationProviders(fetch, orgId),
  ])

  return { projects, providers }
}

export async function loadProjectAgentCount(fetch: AppFetch, projectId: string) {
  const agentResponse = await fetch(`/api/v1/projects/${projectId}/agents`)
  if (!agentResponse.ok) {
    return 0
  }

  const agentData = (await agentResponse.json()) as AgentResponse
  return agentData.agents?.length ?? 0
}

async function loadOrganizationProviders(fetch: AppFetch, orgId: string) {
  const providerResponse = await fetch(`/api/v1/orgs/${orgId}/providers`)
  if (!providerResponse.ok) {
    return []
  }

  const providerData = (await providerResponse.json()) as ProviderResponse
  return providerData.providers ?? []
}

function withQuery(path: string, params: Record<string, string | undefined>) {
  const url = new URL(path, 'http://openase.local')
  for (const [key, value] of Object.entries(params)) {
    if (!value) continue
    url.searchParams.set(key, value)
  }
  return `${url.pathname}${url.search}`
}
