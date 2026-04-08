import { getAuthSession, normalizeReturnTo } from '$lib/api/auth'
import { redirect } from '@sveltejs/kit'
import type { PageLoad } from './$types'

export const load: PageLoad = async ({ url, fetch }) => {
  const returnTo = normalizeReturnTo(url.searchParams.get('return_to'))
  const authSession = await getAuthSession(fetch)

  if (!authSession.loginRequired) {
    throw redirect(307, returnTo)
  }
  if (authSession.authenticated) {
    throw redirect(307, returnTo)
  }

  return {
    authSession,
    returnTo,
  }
}
