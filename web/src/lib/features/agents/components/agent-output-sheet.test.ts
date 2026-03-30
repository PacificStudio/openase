import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { AgentOutputEntry, AgentStepEntry } from '$lib/api/contracts'
import type { AgentInstance } from '../types'
import AgentOutputSheet from './agent-output-sheet.svelte'

const agentFixture: AgentInstance = {
  id: 'agent-1',
  name: 'Codex Worker',
  providerId: 'provider-1',
  providerName: 'Codex',
  modelName: 'gpt-5.4',
  status: 'running',
  runtimePhase: 'executing',
  runtimeControlState: 'active',
  activeRunCount: 1,
  currentTicket: {
    id: 'ticket-1',
    identifier: 'ASE-277',
    title: 'Align runtime event pipeline',
  },
  lastHeartbeat: '2026-03-27T12:00:00Z',
  runtimeStartedAt: '2026-03-27T12:00:00Z',
  sessionId: 'session-1',
  lastError: '',
  currentStepStatus: 'planning',
  currentStepSummary: 'Inspecting runtime events.',
  currentStepChangedAt: '2026-03-27T12:00:00Z',
  todayCompleted: 0,
  todayCost: 0,
}

const outputEntriesFixture: AgentOutputEntry[] = [
  {
    id: 'output-1',
    project_id: 'project-1',
    agent_id: 'agent-1',
    agent_run_id: 'run-1',
    ticket_id: 'ticket-1',
    stream: 'assistant',
    output: 'Inspecting runtime events.',
    created_at: '2026-03-27T12:00:00Z',
  },
]

const stepEntriesFixture: AgentStepEntry[] = [
  {
    id: 'step-1',
    project_id: 'project-1',
    agent_id: 'agent-1',
    ticket_id: 'ticket-1',
    agent_run_id: 'run-1',
    source_trace_event_id: null,
    step_status: 'planning',
    summary: 'Inspect runtime events.',
    created_at: '2026-03-27T12:00:00Z',
  },
]

describe('AgentOutputSheet', () => {
  afterEach(() => {
    cleanup()
  })

  it('does not emit open-change callbacks when only stream props change', async () => {
    const onOpenChange = vi.fn()

    const view = render(AgentOutputSheet, {
      open: true,
      agent: agentFixture,
      entries: [],
      steps: [],
      loading: false,
      error: '',
      streamState: 'connecting',
      onOpenChange,
    })

    expect(onOpenChange).not.toHaveBeenCalled()

    await view.rerender({
      open: true,
      agent: agentFixture,
      entries: outputEntriesFixture,
      steps: stepEntriesFixture,
      loading: false,
      error: '',
      streamState: 'live',
      onOpenChange,
    })

    expect(onOpenChange).not.toHaveBeenCalled()
  })

  it('emits open-change when the sheet actually closes', async () => {
    const onOpenChange = vi.fn()

    const view = render(AgentOutputSheet, {
      open: true,
      agent: agentFixture,
      entries: outputEntriesFixture,
      steps: stepEntriesFixture,
      loading: false,
      error: '',
      streamState: 'live',
      onOpenChange,
    })

    await view.rerender({
      open: false,
      agent: agentFixture,
      entries: outputEntriesFixture,
      steps: stepEntriesFixture,
      loading: false,
      error: '',
      streamState: 'idle',
      onOpenChange,
    })

    expect(onOpenChange).toHaveBeenCalledTimes(1)
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
