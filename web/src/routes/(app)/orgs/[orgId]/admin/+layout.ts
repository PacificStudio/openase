import type { LayoutLoad } from './$types'

export const prerender = false

export const load: LayoutLoad = ({ params, url }) => ({
  organizationId: params.orgId,
  currentPath: url.pathname,
})
