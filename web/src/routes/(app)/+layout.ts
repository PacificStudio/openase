import { parseAppRouteContext, projectSectionFromPathname } from '$lib/stores/app-context'
import type { LayoutLoad } from './$types'

export const load: LayoutLoad = async ({ params, url }) => {
  const routeContext = parseAppRouteContext(params)

  return {
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
