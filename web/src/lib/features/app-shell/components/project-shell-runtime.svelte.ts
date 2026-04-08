import { goto } from '$app/navigation'
import { logoutHumanSession, normalizeReturnTo } from '$lib/api/auth'
import { loadAppContext } from '$lib/api/app-context'
import { ApiError } from '$lib/api/client'
import type { ProjectAIFocus } from '$lib/features/chat'
import { authStore } from '$lib/stores/auth.svelte'
import { appStore } from '$lib/stores/app.svelte'
import {
  organizationPath,
  projectPath,
  type AppRouteContext,
  type ProjectSection,
} from '$lib/stores/app-context'
import { toastStore } from '$lib/stores/toast.svelte'
import {
  replaceSelectionIfChanged,
  selectCurrentOrg,
  selectCurrentProject,
} from './project-shell-state'

type LoadedAppContext = Awaited<ReturnType<typeof loadAppContext>>

export function syncResolvedRouteContext(nextRouteContext: AppRouteContext) {
  const nextOrgId = nextRouteContext.orgId
  const nextProjectId = nextRouteContext.scope === 'project' ? nextRouteContext.projectId : null
  const nextOrg = appStore.resolveOrganization(nextOrgId)
  const nextProject = appStore.resolveProject(nextOrgId, nextProjectId)

  replaceSelectionIfChanged(appStore.currentOrg, nextOrg, (value) => {
    appStore.currentOrg = value
  })
  replaceSelectionIfChanged(appStore.currentProject, nextProject, (value) => {
    appStore.currentProject = value
  })
}

export function applyLoadedAppContext(
  payload: LoadedAppContext,
  nextRouteContext: AppRouteContext,
  nextRouteKey: string,
) {
  appStore.applyAppContext({
    organizations: payload.organizations,
    projects: payload.projects,
    providers: payload.providers,
    agentCount: payload.agentCount,
  })
  const fetchedAt = Date.now()
  appStore.appContextFetchedAt = fetchedAt
  appStore.currentOrg = selectCurrentOrg(payload, nextRouteContext)
  appStore.currentProject = selectCurrentProject(payload, nextRouteContext)

  return { fetchedAt, routeKey: nextRouteKey }
}

export function settingsHrefForRoute(routeContext: AppRouteContext): string {
  return routeContext.scope === 'project'
    ? projectPath(routeContext.orgId, routeContext.projectId, 'settings')
    : ''
}

export function openSettingsForRoute(
  routeContext: AppRouteContext,
  currentSection: ProjectSection,
) {
  if (routeContext.scope === 'project') {
    void goto(projectPath(routeContext.orgId, routeContext.projectId, currentSection))
    return
  }
  if (routeContext.scope === 'org') {
    void goto(organizationPath(routeContext.orgId))
    return
  }
  void goto('/')
}

export async function logoutProjectShellSession(
  currentLocation: string,
  setLogoutPending: (value: boolean) => void,
) {
  setLogoutPending(true)
  const returnTo = normalizeReturnTo(currentLocation)
  const loginRequired = authStore.loginRequired

  try {
    await logoutHumanSession()
  } catch (caughtError) {
    if (!(caughtError instanceof ApiError) || caughtError.status !== 401) {
      toastStore.error(caughtError instanceof ApiError ? caughtError.detail : 'Failed to log out.')
      setLogoutPending(false)
      return
    }
  }

  authStore.clear()
  setLogoutPending(false)
  await goto(loginRequired ? `/login?return_to=${encodeURIComponent(returnTo)}` : '/')
}

export function consumeProjectAssistantRequest(onOpen: (prompt: string) => void) {
  const request = appStore.projectAssistantRequest
  if (!request) {
    return
  }

  const consumed = appStore.consumeProjectAssistantRequest()
  if (consumed) {
    onOpen(consumed.prompt)
  }
}

export function resolveAssistantFocus(
  currentProjectId: string | null | undefined,
  focus: ProjectAIFocus | null | undefined,
): ProjectAIFocus | null {
  if (!currentProjectId || focus?.projectId !== currentProjectId) {
    return null
  }
  return focus
}
