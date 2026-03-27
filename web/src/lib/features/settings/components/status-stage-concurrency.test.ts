import { render } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import type { StatusPayload } from '$lib/api/contracts'
import StatusStageConcurrency from './status-stage-concurrency.svelte'

describe('StatusStageConcurrency', () => {
  it('treats omitted max_active_runs values as unlimited capacity', () => {
    const stages = [
      {
        id: 'stage-1',
        project_id: 'project-1',
        key: 'backlog',
        name: 'Backlog',
        position: 1,
        active_runs: 2,
        description: '',
      } as StatusPayload['stages'][number],
    ]

    const { container, getByText } = render(StatusStageConcurrency, {
      props: {
        stages,
      },
    })

    expect(getByText('2 active now, unlimited capacity')).toBeTruthy()
    expect(container.textContent).toContain('Backlog')
    expect(container.textContent).not.toContain('2/')
  })
})
