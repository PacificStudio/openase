import { ApiError } from '$lib/api/client'
import { normalizeReturnTo } from '$lib/api/auth'
import { redeemLocalBootstrapAuthorization } from '$lib/api/auth'

export type LocalBootstrapRedeemInput = {
  requestID: string
  code: string
  nonce: string
}

export async function redeemLocalBootstrapBrowserSession(
  input: LocalBootstrapRedeemInput,
): Promise<void> {
  await redeemLocalBootstrapAuthorization(input)
}

export function parseLocalBootstrapRedeemURL(
  raw: string,
  currentOrigin: string,
): string | null {
  const trimmed = raw.trim()
  if (!trimmed) {
    return null
  }

  try {
    const parsed = new URL(trimmed, currentOrigin)
    if (parsed.origin !== currentOrigin || parsed.pathname !== '/local-bootstrap') {
      return null
    }
    for (const key of ['request_id', 'code', 'nonce']) {
      if (!parsed.searchParams.get(key)?.trim()) {
        return null
      }
    }
    parsed.searchParams.set(
      'return_to',
      normalizeReturnTo(parsed.searchParams.get('return_to')),
    )
    return `${parsed.pathname}?${parsed.searchParams.toString()}`
  } catch {
    return null
  }
}

export function describeLocalBootstrapRedeemError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.detail || 'Local bootstrap authorization failed.'
  }
  return 'Local bootstrap authorization failed.'
}
