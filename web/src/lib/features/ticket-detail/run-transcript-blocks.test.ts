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
      adapterType: 'codex-app-server',
      modelName: 'gpt-5.4',
      usage: {
        total: 25,
        input: 20,
        output: 5,
        cachedInput: 3,
        cacheCreation: 2,
        reasoning: 1,
        prompt: 18,
        candidate: 4,
        tool: 2,
      },
      status: 'ended',
      createdAt: '2026-04-03T12:00:00Z',
      terminalAt: '2026-04-03T12:02:00Z',
    }

    const blocks = seedRunBlocks(run)
    expect(blocks.some((block) => block.kind === 'result')).toBe(false)
  })
})
