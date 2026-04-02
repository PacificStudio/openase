import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  resetOrganizationEventBusForTests,
  retainOrganizationEventBus,
  subscribeOrganizationMachineEvents,
} from './org-event-bus'

const { connectEventStream } = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
}))

vi.mock('$lib/api/sse', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/sse')>('$lib/api/sse')
  return {
    ...actual,
    connectEventStream,
  }
})

describe('organizationEventBus', () => {
  afterEach(() => {
    resetOrganizationEventBusForTests()
    vi.clearAllMocks()
  })

  it('uses the machines stream path for machine subscribers', () => {
    const disconnect = vi.fn()
    connectEventStream.mockReturnValue(disconnect)

    const release = retainOrganizationEventBus('org-1', 'machines')
    const unsubscribe = subscribeOrganizationMachineEvents('org-1', vi.fn())

    expect(connectEventStream).toHaveBeenCalledTimes(1)
    expect(connectEventStream).toHaveBeenCalledWith(
      '/api/v1/orgs/org-1/machines/stream',
      expect.objectContaining({
        onEvent: expect.any(Function),
        onStateChange: expect.any(Function),
      }),
    )

    unsubscribe()
    release()
    expect(disconnect).toHaveBeenCalledTimes(1)
  })

  it('uses the providers stream path for provider subscribers', () => {
    const disconnect = vi.fn()
    connectEventStream.mockReturnValue(disconnect)

    const release = retainOrganizationEventBus('org-1', 'providers')

    expect(connectEventStream).toHaveBeenCalledTimes(1)
    expect(connectEventStream.mock.calls[0]?.[0]).toBe('/api/v1/orgs/org-1/providers/stream')

    release()
    expect(disconnect).toHaveBeenCalledTimes(1)
  })
})
