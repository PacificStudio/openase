import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { retainProjectEventBus } from '$lib/features/project-events'
import { i18nStore } from '$lib/i18n/store.svelte'
import { appStore } from '$lib/stores/app.svelte'
import { resetAgentsPageCacheForTests } from '../agents-page-cache'
import AgentsPage from './agents-page.svelte'
import { makeAgent, makePageData, orgFixture, projectFixture } from './agents-page.test-helpers'

const { connectEventStream, loadAgentsPageResult } = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
  loadAgentsPageResult: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({}))

vi.mock('$lib/api/sse', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/sse')>('$lib/api/sse')
  return {
    ...actual,
    connectEventStream,
  }
})

vi.mock('../page-data', () => ({
  loadAgentsPageResult,
}))

describe('AgentsPage reconnect', () => {
  afterEach(() => {
    cleanup()
    resetAgentsPageCacheForTests()
    i18nStore.setLocale('en')
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('reloads runtime state after the shared project stream reconnects', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    const initialData = makePageData(makeAgent({ runtimePhase: 'ready' }))
    const updatedData = makePageData(makeAgent({ runtimePhase: 'executing' }))
    updatedData.agentRuns[0] = { ...updatedData.agentRuns[0], status: 'executing' }

    loadAgentsPageResult
      .mockResolvedValueOnce({ ok: true, data: initialData })
      .mockResolvedValueOnce({ ok: true, data: updatedData })
    connectEventStream.mockReturnValue(() => {})

    const releaseShell = retainProjectEventBus(projectFixture.id)
    const view = render(AgentsPage)

    expect(await view.findByText('Codex Worker')).toBeTruthy()
    await fireEvent.click(view.getByRole('button', { name: 'Expand' }))
    expect(view.getByText('Ready')).toBeTruthy()
    expect(loadAgentsPageResult).toHaveBeenCalledTimes(1)

    const projectBusCall = connectEventStream.mock.calls.find(
      ([url]) => url === `/api/v1/projects/${projectFixture.id}/events/stream`,
    )
    if (!projectBusCall) {
      throw new Error('project event stream was not opened')
    }

    const options = projectBusCall[1] as {
      onStateChange: (state: 'live' | 'idle' | 'connecting' | 'retrying') => void
    }
    options.onStateChange('connecting')
    options.onStateChange('live')
    options.onStateChange('retrying')
    options.onStateChange('live')

    await waitFor(() => {
      expect(loadAgentsPageResult).toHaveBeenCalledTimes(2)
      expect(view.getByText('Executing')).toBeTruthy()
    })

    releaseShell()
  })
})
