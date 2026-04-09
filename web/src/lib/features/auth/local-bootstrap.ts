import { ApiError } from '$lib/api/client'
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

export function describeLocalBootstrapRedeemError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.detail || 'Local bootstrap authorization failed.'
  }
  return 'Local bootstrap authorization failed.'
}
