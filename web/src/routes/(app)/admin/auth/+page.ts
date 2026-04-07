import { error } from '@sveltejs/kit'
import type { PageLoad } from './$types'

export const load: PageLoad = async ({ parent }) => {
  const { authSession } = await parent()
  if (authSession.authMode === 'oidc' && !authSession.roles.includes('instance_admin')) {
    throw error(403, 'Instance admin access is required for /admin/auth.')
  }
  return {}
}
