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

export const load: LayoutLoad = async ({ fetch }) => {
  const orgResponse = await fetch('/api/v1/orgs')
  if (!orgResponse.ok) {
    return {
      currentOrg: null,
      currentProject: null,
      providers: [],
      agentCount: 0,
    }
  }

  const orgData = (await orgResponse.json()) as OrgResponse
  const currentOrg = orgData.organizations?.[0] ?? null
  if (!currentOrg) {
    return {
      currentOrg: null,
      currentProject: null,
      providers: [],
      agentCount: 0,
    }
  }

  const [projectResponse, providerResponse] = await Promise.all([
    fetch(`/api/v1/orgs/${currentOrg.id}/projects`),
    fetch(`/api/v1/orgs/${currentOrg.id}/providers`),
  ])

  const projectData = projectResponse.ok
    ? ((await projectResponse.json()) as ProjectResponse)
    : { projects: [] }
  const providerData = providerResponse.ok
    ? ((await providerResponse.json()) as ProviderResponse)
    : { providers: [] }

  const currentProject = projectData.projects?.[0] ?? null
  let agentCount = 0

  if (currentProject) {
    const agentResponse = await fetch(`/api/v1/projects/${currentProject.id}/agents`)
    if (agentResponse.ok) {
      const agentData = (await agentResponse.json()) as AgentResponse
      agentCount = agentData.agents?.length ?? 0
    }
  }

  return {
    currentOrg,
    currentProject,
    providers: providerData.providers ?? [],
    agentCount,
  }
}
