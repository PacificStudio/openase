import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import AgentsPage from './agents-page.svelte'
import {
  emptyOutputFixture,
  emptyStepFixture,
  makeAgent,
  makePageData,
  orgFixture,
  projectFixture,
} from './agents-page.test-helpers'

const { listAgentOutput, listAgentSteps, connectEventStream, loadAgentsPageResult } = vi.hoisted(
  () => ({
    listAgentOutput: vi.fn(),
    listAgentSteps: vi.fn(),
    connectEventStream: vi.fn(),
    loadAgentsPageResult: vi.fn(),
  }),
)

vi.mock('$lib/api/openase', () => ({
  listAgentOutput,
  listAgentSteps,
}))

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

  it('opens separate trace and step streams for the output drawer', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(makeAgent()) })
    listAgentOutput.mockResolvedValue(emptyOutputFixture)
    listAgentSteps.mockResolvedValue(emptyStepFixture)
    connectEventStream.mockReturnValue(() => {})

    const { findAllByLabelText } = render(AgentsPage)

    const buttons = await findAllByLabelText('View output')
    await fireEvent.click(buttons[0]!)

    await waitFor(() => {
      expect(connectEventStream).toHaveBeenCalledWith(
        '/api/v1/projects/project-1/agents/agent-1/output/stream',
        expect.any(Object),
      )
      expect(connectEventStream).toHaveBeenCalledWith(
        '/api/v1/projects/project-1/agents/agent-1/steps/stream',
        expect.any(Object),
      )
    })
  })
})
