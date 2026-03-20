import type { AgentProvider, Organization, Project } from '$lib/api/contracts'
import {
  parseAppRouteContext,
  projectSectionFromPathname,
  type ProjectSection,
} from '$lib/stores/app-context'
import { error } from '@sveltejs/kit'
import type { LayoutLoad } from './$types'

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

type AppLayoutData = {
  organizations: Organization[]
  currentOrg: Organization | null
  currentProject: Project | null
  projects: Project[]
  providers: AgentProvider[]
  agentCount: number
  currentSection: ProjectSection
}

const emptyLayoutData: AppLayoutData = {
  organizations: [],
  currentOrg: null,
  currentProject: null,
  projects: [],
  providers: [],
  agentCount: 0,
  currentSection: 'dashboard',
}

export const load: LayoutLoad = async ({ fetch, params, url }) => {
  const organizations = await loadOrganizations(fetch)
  if (organizations.length === 0 && !params.orgId) {
    return emptyLayoutData
  }

  const routeContext = parseAppRouteContext(params)
  const currentOrg = routeContext.orgId
    ? (organizations.find((organization) => organization.id === routeContext.orgId) ?? null)
    : null

  if (routeContext.scope !== 'none' && !currentOrg) {
    throw error(404, 'Organization not found')
  }

  const { projects, providers } = currentOrg
    ? await loadOrganizationContext(fetch, currentOrg.id)
    : { projects: [], providers: [] }

  const currentProject =
    routeContext.scope === 'project'
      ? (projects.find((project) => project.id === routeContext.projectId) ?? null)
      : null

  if (routeContext.scope === 'project' && !currentProject) {
    throw error(404, 'Project not found')
  }

  const agentCount = currentProject ? await loadAgentCount(fetch, currentProject.id) : 0

  return {
    organizations,
    currentOrg,
    currentProject,
    projects,
    providers,
    agentCount,
    currentSection:
      currentProject && currentOrg
        ? projectSectionFromPathname(url.pathname, {
            scope: 'project',
            orgId: currentOrg.id,
            projectId: currentProject.id,
          })
        : 'dashboard',
  }
}

async function loadOrganizations(fetch: typeof globalThis.fetch) {
  const orgResponse = await fetch('/api/v1/orgs')
  if (!orgResponse.ok) {
    return []
  }

  const orgData = (await orgResponse.json()) as OrgResponse
  return orgData.organizations ?? []
}

async function loadOrganizationContext(fetch: typeof globalThis.fetch, orgId: string) {
  const [projectResponse, providerResponse] = await Promise.all([
    fetch(`/api/v1/orgs/${orgId}/projects`),
    fetch(`/api/v1/orgs/${orgId}/providers`),
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

async function loadAgentCount(fetch: typeof globalThis.fetch, projectId: string) {
  const agentResponse = await fetch(`/api/v1/projects/${projectId}/agents`)
  if (!agentResponse.ok) {
    return 0
  }

  const agentData = (await agentResponse.json()) as AgentResponse
  return agentData.agents?.length ?? 0
}
