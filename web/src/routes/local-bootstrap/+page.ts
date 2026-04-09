import { normalizeReturnTo } from '$lib/api/auth'
import type { PageLoad } from './$types'

export const load: PageLoad = ({ url }) => ({
  requestID: url.searchParams.get('request_id')?.trim() ?? '',
  code: url.searchParams.get('code')?.trim() ?? '',
  nonce: url.searchParams.get('nonce')?.trim() ?? '',
  returnTo: normalizeReturnTo(url.searchParams.get('return_to')),
})
