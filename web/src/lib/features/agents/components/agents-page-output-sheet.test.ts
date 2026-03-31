import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import type { AgentOutputPayload, AgentStepPayload } from '$lib/api/contracts'
import AgentsPage from './agents-page.svelte'
import {
  emptyOutputFixture,
  emptyStepFixture,
  makeAgent,
  makePageData,
  orgFixture,
  outputEntriesFixture,
  projectFixture,
  stepEntriesFixture,
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

describe('AgentsPage output sheet lifecycle', () => {
  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  const agentVariants = [
    {
      label: 'running agent',
      agent: makeAgent({
        id: 'agent-running',
        status: 'running',
        runtimePhase: 'executing',
        runtimeControlState: 'active',
        activeRunCount: 1,
      }),
    },
    {
      label: 'terminated agent',
      agent: makeAgent({
        id: 'agent-terminated',
        status: 'terminated',
        runtimePhase: 'none',
        runtimeControlState: 'active',
        activeRunCount: 0,
        currentTicket: undefined,
        lastHeartbeat: null,
      }),
    },
    {
      label: 'idle agent',
      agent: makeAgent({
        id: 'agent-idle',
        status: 'idle',
        runtimePhase: 'none',
        runtimeControlState: 'active',
        activeRunCount: 0,
        currentTicket: undefined,
      }),
    },
  ] as const

  for (const { label, agent } of agentVariants) {
    describe(label, () => {
      it('opens sheet, loads data, and closes cleanly', async () => {
        appStore.currentOrg = orgFixture
        appStore.currentProject = projectFixture

        loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(agent) })

        let resolveOutput!: (value: AgentOutputPayload) => void
        let resolveSteps!: (value: AgentStepPayload) => void
        listAgentOutput.mockReturnValue(
          new Promise<AgentOutputPayload>((resolve) => (resolveOutput = resolve)),
        )
        listAgentSteps.mockReturnValue(
          new Promise<AgentStepPayload>((resolve) => (resolveSteps = resolve)),
        )

        const disconnectFns = {
          agentStream: vi.fn(),
          providerStream: vi.fn(),
          outputStream: vi.fn(),
          stepStream: vi.fn(),
        }

        let streamCallIndex = 0
        connectEventStream.mockImplementation(() => {
          const idx = streamCallIndex++
          if (idx === 0) return disconnectFns.agentStream
          if (idx === 1) return disconnectFns.providerStream
          if (idx === 2) return disconnectFns.outputStream
          return disconnectFns.stepStream
        })

        const view = render(AgentsPage)
        const buttons = await view.findAllByLabelText('View output')
        expect(buttons.length).toBeGreaterThanOrEqual(1)

        await fireEvent.click(buttons[0]!)

        await waitFor(() => {
          expect(view.getByText('Loading agent runtime…')).toBeTruthy()
        })

        expect(listAgentOutput).toHaveBeenCalledWith('project-1', agent.id, { limit: 200 })
        expect(listAgentSteps).toHaveBeenCalledWith('project-1', agent.id, { limit: 200 })

        resolveOutput(outputEntriesFixture)
        resolveSteps(stepEntriesFixture)

        await waitFor(() => {
          expect(view.getByText('read_file("/src/main.ts")')).toBeTruthy()
        })

        expect(view.getByText('2 entries')).toBeTruthy()
        expect(view.getByText('Analyzing pipeline structure.')).toBeTruthy()

        await waitFor(() => {
          expect(connectEventStream).toHaveBeenCalledWith(
            `/api/v1/projects/project-1/agents/${agent.id}/output/stream`,
            expect.any(Object),
          )
          expect(connectEventStream).toHaveBeenCalledWith(
            `/api/v1/projects/project-1/agents/${agent.id}/steps/stream`,
            expect.any(Object),
          )
        })

        const closeButtons = view.getAllByRole('button', { name: 'Close' })
        const sheetClose = closeButtons.find(
          (button) => button.closest('[data-slot="sheet-content"]') !== null,
        )
        expect(sheetClose).toBeTruthy()
        await fireEvent.click(sheetClose!)

        await waitFor(() => {
          expect(view.queryByText('Loading agent runtime…')).toBeNull()
          expect(view.queryByText('Inspecting runtime events.')).toBeNull()
        })

        expect(disconnectFns.outputStream).toHaveBeenCalled()
        expect(disconnectFns.stepStream).toHaveBeenCalled()
      })

      it('handles API errors gracefully without freezing', async () => {
        appStore.currentOrg = orgFixture
        appStore.currentProject = projectFixture

        loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(agent) })
        listAgentOutput.mockRejectedValue(new Error('Connection refused'))
        listAgentSteps.mockRejectedValue(new Error('Connection refused'))

        const disconnectFns = { output: vi.fn(), steps: vi.fn() }
        let streamIdx = 0
        connectEventStream.mockImplementation(() => {
          const idx = streamIdx++
          if (idx <= 1) return vi.fn()
          if (idx === 2) return disconnectFns.output
          return disconnectFns.steps
        })

        const view = render(AgentsPage)
        const buttons = await view.findAllByLabelText('View output')
        await fireEvent.click(buttons[0]!)

        await waitFor(() => {
          expect(view.getByText('Failed to load agent output.')).toBeTruthy()
        })

        const closeButtons = view.getAllByRole('button', { name: 'Close' })
        const sheetClose = closeButtons.find(
          (button) => button.closest('[data-slot="sheet-content"]') !== null,
        )
        await fireEvent.click(sheetClose!)

        await waitFor(() => {
          expect(view.queryByText('Failed to load agent output.')).toBeNull()
        })
      })

      it('handles SSE frame arrival after initial load', async () => {
        appStore.currentOrg = orgFixture
        appStore.currentProject = projectFixture

        loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(agent) })
        listAgentOutput.mockResolvedValue(emptyOutputFixture)
        listAgentSteps.mockResolvedValue(emptyStepFixture)

        type FrameHandler = (frame: { event: string; data: string }) => void
        let outputFrameHandler: FrameHandler | undefined
        let stepFrameHandler: FrameHandler | undefined

        let streamIdx = 0
        connectEventStream.mockImplementation((_url: string, opts: { onEvent: FrameHandler }) => {
          const idx = streamIdx++
          if (idx <= 1) return vi.fn()
          if (idx === 2) {
            outputFrameHandler = opts.onEvent
            return vi.fn()
          }
          stepFrameHandler = opts.onEvent
          return vi.fn()
        })

        const view = render(AgentsPage)
        const buttons = await view.findAllByLabelText('View output')
        await fireEvent.click(buttons[0]!)

        await waitFor(() => {
          expect(view.getByText('0 entries')).toBeTruthy()
        })

        outputFrameHandler?.({
          event: 'agent.output',
          data: JSON.stringify({
            payload: {
              entry: {
                id: 'output-live-1',
                project_id: 'project-1',
                agent_id: agent.id,
                agent_run_id: 'run-1',
                ticket_id: 'ticket-1',
                stream: 'assistant',
                output: 'Live streamed output line.',
                created_at: '2026-03-27T12:01:00Z',
              },
            },
          }),
        })

        await waitFor(() => {
          expect(view.getByText('Live streamed output line.')).toBeTruthy()
          expect(view.getByText('1 entries')).toBeTruthy()
        })

        stepFrameHandler?.({
          event: 'agent.step',
          data: JSON.stringify({
            payload: {
              entry: {
                id: 'step-live-1',
                project_id: 'project-1',
                agent_id: agent.id,
                ticket_id: 'ticket-1',
                agent_run_id: 'run-1',
                source_trace_event_id: null,
                step_status: 'executing',
                summary: 'Reading source files.',
                created_at: '2026-03-27T12:01:00Z',
              },
            },
          }),
        })

        await waitFor(() => {
          expect(view.getByText('Reading source files.')).toBeTruthy()
        })
      })
    })
  }
})
