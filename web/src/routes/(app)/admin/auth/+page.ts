import { error } from '@sveltejs/kit'
import type { PageLoad } from './$types'

export const load: PageLoad = async ({ parent }) => {
  const { authSession } = await parent()
  if (!authSession.canManageAuth) {
    throw error(403, 'Instance admin access is required for /admin/auth.')
  }
  return {}
}
