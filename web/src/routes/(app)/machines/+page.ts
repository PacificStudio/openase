import type { PageLoad } from './$types'
import { redirectToDefaultProject } from '../legacy-redirect'

export const load: PageLoad = async ({ fetch }) => {
  return redirectToDefaultProject(fetch, 'machines')
}
