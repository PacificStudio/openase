import type { PageLoad } from './$types'
import { redirectToDefaultOrganization } from './legacy-redirect'

export const load: PageLoad = async ({ fetch }) => {
  return redirectToDefaultOrganization(fetch)
}
