import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { HRAdvisorSnapshot } from '../types'
import HRAdvisorPanel from './hr-advisor-panel.svelte'

const { requestProjectAssistant, clearProjectAssistantFocus } = vi.hoisted(() => ({
  requestProjectAssistant: vi.fn(),
  clearProjectAssistantFocus: vi.fn(),
}))

vi.mock('$lib/stores/app.svelte', () => ({
  appStore: {
    requestProjectAssistant,
    clearProjectAssistantFocus,
  },
}))

const advisorFixture: HRAdvisorSnapshot = {
  summary: {
    open_tickets: 12,
    coding_tickets: 0,
    failing_tickets: 1,
    blocked_tickets: 2,
    active_agents: 1,
    workflow_count: 2,
    recent_activity_count: 5,
    active_workflow_families: [],
  },
  staffing: {
    developers: 0,
    qa: 0,
    docs: 0,
    security: 0,
    product: 0,
    research: 0,
  },
  recommendations: [],
}

describe('HRAdvisorPanel', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('renders AI-driven summary and actions', async () => {
    const { findByText } = render(HRAdvisorPanel, {
      props: { projectId: 'project-1', advisor: advisorFixture },
    })

    expect(await findByText('AI-driven')).toBeTruthy()
    expect(await findByText('Ask Project AI')).toBeTruthy()
    expect(await findByText('Ask Project AI to create')).toBeTruthy()
    expect(await findByText('12')).toBeTruthy()
    expect(await findByText('2')).toBeTruthy()
  })

  it('requests Project AI advice prompt', async () => {
    const { findByText } = render(HRAdvisorPanel, {
      props: { projectId: 'project-1', advisor: advisorFixture },
    })

    await fireEvent.click((await findByText('Ask Project AI')).closest('button')!)

    expect(clearProjectAssistantFocus).toHaveBeenCalledWith('hr-advisor')
    expect(requestProjectAssistant).toHaveBeenCalledWith(
      expect.stringContaining('recommend what workflow or role changes should happen next'),
    )
  })

  it('requests Project AI creation prompt', async () => {
    const { findByText } = render(HRAdvisorPanel, {
      props: { projectId: 'project-1', advisor: advisorFixture },
    })

    await fireEvent.click((await findByText('Ask Project AI to create')).closest('button')!)

    expect(requestProjectAssistant).toHaveBeenCalledWith(
      expect.stringContaining('If confidence is high, create the missing workflow'),
    )
  })
})
