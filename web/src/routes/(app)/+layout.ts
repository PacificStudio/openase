import { getAuthSession, normalizeReturnTo } from '$lib/api/auth'
import { parseAppRouteContext, projectSectionFromPathname } from '$lib/stores/app-context'
import { redirect } from '@sveltejs/kit'
import type { LayoutLoad } from './$types'

export const load: LayoutLoad = async ({ params, url, fetch }) => {
  const routeContext = parseAppRouteContext(params)
  const authSession = await getAuthSession(fetch)
  if (authSession.authMode === 'oidc' && !authSession.authenticated) {
    throw redirect(
      307,
      `/login?return_to=${encodeURIComponent(normalizeReturnTo(url.pathname + url.search + url.hash))}`,
    )
  }

  return {
    authSession,
    routeContext,
    currentSection:
      routeContext.scope === 'project'
        ? projectSectionFromPathname(url.pathname, {
            scope: 'project',
            orgId: routeContext.orgId,
            projectId: routeContext.projectId,
          })
        : 'dashboard',
  }
}
