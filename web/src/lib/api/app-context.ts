import type { AgentProvider, Organization, Project } from './contracts'

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
