import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import AgentsPage from './agents-page.svelte'
import { makeAgent, makePageData, orgFixture, projectFixture } from './agents-page.test-helpers'

const { connectEventStream, loadAgentsPageResult } = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
  loadAgentsPageResult: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

vi.mock('../page-data', () => ({
  loadAgentsPageResult,
}))

describe('AgentsPage', () => {
  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders agent list after loading', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(makeAgent()) })
    connectEventStream.mockReturnValue(() => {})

    const { findByText } = render(AgentsPage)

    expect(await findByText('Codex Worker')).toBeTruthy()
  })
})
