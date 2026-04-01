export const projectSections = [
  'dashboard',
  'tickets',
  'agents',
  'machines',
  'updates',
  'activity',
  'workflows',
  'skills',
  'scheduled-jobs',
  'settings',
] as const

export type ProjectSection = (typeof projectSections)[number]

type AppContextParams = {
  orgId?: string
  projectId?: string
}

export type AppRouteContext =
  | { scope: 'none'; orgId: null; projectId: null }
  | { scope: 'org'; orgId: string; projectId: null }
  | { scope: 'project'; orgId: string; projectId: string }

const projectSectionSet = new Set<string>(projectSections)

export function parseAppRouteContext(params: AppContextParams): AppRouteContext {
  const orgId = params.orgId?.trim()
  const projectId = params.projectId?.trim()

  if (orgId && projectId) {
    return { scope: 'project', orgId, projectId }
  }

  if (orgId) {
    return { scope: 'org', orgId, projectId: null }
  }

  return { scope: 'none', orgId: null, projectId: null }
}

export function organizationPath(orgId: string) {
  return `/orgs/${orgId}`
}

export function projectPath(
  orgId: string,
  projectId: string,
  section: ProjectSection = 'dashboard',
) {
  const basePath = `${organizationPath(orgId)}/projects/${projectId}`
  return section === 'dashboard' ? basePath : `${basePath}/${section}`
}

export function projectSectionFromPathname(
  pathname: string,
  context: Extract<AppRouteContext, { scope: 'project' }>,
): ProjectSection {
  const basePath = projectPath(context.orgId, context.projectId)

  if (!pathname.startsWith(basePath)) {
    return 'dashboard'
  }

  const suffix = pathname.slice(basePath.length).replace(/^\/+/, '')
  if (!suffix) {
    return 'dashboard'
  }

  const firstSegment = suffix.split('/')[0]
  return asProjectSection(firstSegment) ?? 'dashboard'
}

function asProjectSection(value: string): ProjectSection | null {
  return projectSectionSet.has(value) ? (value as ProjectSection) : null
}
