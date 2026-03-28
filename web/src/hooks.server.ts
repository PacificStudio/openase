import { env } from '$env/dynamic/private'
import type { Handle } from '@sveltejs/kit'
import { handleMockApi } from '$lib/testing/e2e/mock-api'

const e2eMockEnabled = env.OPENASE_E2E_MOCK === '1'
const e2eApiDelayMs = Number(env.OPENASE_E2E_API_DELAY_MS ?? '0')

export const handle: Handle = async ({ event, resolve }) => {
  if (e2eMockEnabled) {
    const mockedResponse = await handleMockApi(event.request, event.url)
    if (mockedResponse) {
      if (e2eApiDelayMs > 0 && !event.url.pathname.endsWith('/stream')) {
        await new Promise((resolveDelay) => setTimeout(resolveDelay, e2eApiDelayMs))
      }
      return mockedResponse
    }
  }

  return resolve(event)
}
