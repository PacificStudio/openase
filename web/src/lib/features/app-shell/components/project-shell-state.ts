import type { loadAppContext } from '$lib/api/app-context'
import type { Project } from '$lib/api/contracts'
import type { AppRouteContext } from '$lib/stores/app-context'

type LoadedAppContext = Awaited<ReturnType<typeof loadAppContext>>

export function isAppContextFresh(
  lastAppContextKey: string,
  nextRouteKey: string,
  lastAppContextFetchedAt: number,
) {
  return lastAppContextKey === nextRouteKey && Date.now() - lastAppContextFetchedAt < 30_000
}

export function selectCurrentOrg(payload: LoadedAppContext, routeContext: AppRouteContext) {
  if (!routeContext.orgId) {
    return null
  }

  return (
    payload.organizations.find((organization) => organization.id === routeContext.orgId) ?? null
  )
}

export function selectCurrentProject(payload: LoadedAppContext, routeContext: AppRouteContext) {
  if (routeContext.scope !== 'project') {
    return null
  }

  return payload.projects.find((project) => project.id === routeContext.projectId) ?? null
}

export function getProjectHealth(project: Project | null): 'healthy' | 'degraded' | 'critical' {
  const status = project?.status?.toLowerCase()
  if (status === 'healthy' || status === 'active') return 'healthy'
  if (status === 'blocked' || status === 'archived') return 'critical'
  return 'degraded'
}

export function getProjectHealthLabel(
  project: Project | null,
  projectHealth: 'healthy' | 'degraded' | 'critical',
) {
  const status = project?.status ?? ''
  switch (projectHealth) {
    case 'healthy':
      return `All systems healthy${status ? ` (${status})` : ''}`
    case 'degraded':
      return `Project status: ${status || 'degraded'}`
    case 'critical':
      return `Project status: ${status || 'critical'} — may need attention`
  }
}

export function replaceSelectionIfChanged<T extends { id?: string } | null>(
  current: T,
  next: T,
  assign: (value: T) => void,
) {
  if ((current?.id ?? null) === (next?.id ?? null)) {
    return
  }

  assign(next)
}

export function mergeProjectIntoAppContext(
  projects: Project[],
  currentProject: Project | null,
  nextProject: Project,
) {
  const existingIndex = projects.findIndex((project) => project.id === nextProject.id)
  const nextProjects =
    existingIndex >= 0
      ? projects.map((project, index) => (index === existingIndex ? nextProject : project))
      : [...projects, nextProject]

  return {
    projects: nextProjects,
    currentProject: currentProject?.id === nextProject.id ? nextProject : currentProject,
  }
}
