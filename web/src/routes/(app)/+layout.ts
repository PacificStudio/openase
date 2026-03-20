import {
  loadOrganizationContext,
  loadOrganizations,
  loadProjectAgentCount,
} from '$lib/api/app-context'
import type { AgentProvider, Organization, Project } from '$lib/api/contracts'
import {
  parseAppRouteContext,
  projectSectionFromPathname,
  type ProjectSection,
} from '$lib/stores/app-context'
import { error } from '@sveltejs/kit'
import type { LayoutLoad } from './$types'

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
const loadAgentCount = loadProjectAgentCount
