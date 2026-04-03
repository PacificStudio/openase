import { describe, expect, it } from 'vitest'

import { buildLifecycleBlock, seedRunBlocks } from './run-transcript-blocks'
import type { TicketRun } from './types'

describe('ticket run terminal semantics', () => {
  it('does not render terminated lifecycle events as transcript result blocks', () => {
    const block = buildLifecycleBlock({
      eventType: 'agent.terminated',
      message: 'Agent product-manager-01 terminated its runtime session.',
      createdAt: '2026-04-03T12:03:00Z',
    })

    expect(block).toBeNull()
  })

  it('does not seed ended runs with a synthetic result block', () => {
    const run: TicketRun = {
      id: 'run-1',
      attemptNumber: 1,
      agentId: 'agent-1',
      agentName: 'Runner',
      provider: 'Codex',
      status: 'ended',
      createdAt: '2026-04-03T12:00:00Z',
      terminalAt: '2026-04-03T12:02:00Z',
    }

    const blocks = seedRunBlocks(run)
    expect(blocks.some((block) => block.kind === 'result')).toBe(false)
  })
})
