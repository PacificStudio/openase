import { render } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import StatusConcurrency from './status-concurrency.svelte'

describe('StatusConcurrency', () => {
  it('renders status-level capacity details', () => {
    const statuses = [
      {
        id: 'status-1',
        project_id: 'project-1',
        name: 'Todo',
        stage: 'unstarted',
        color: '#2563eb',
        icon: '',
        is_default: true,
        description: '',
        position: 0,
        active_runs: 2,
        max_active_runs: 3,
      },
    ]

    const { container, getByText } = render(StatusConcurrency, {
      props: {
        statuses,
      },
    })

    expect(getByText('2 active now, capacity 3')).toBeTruthy()
    expect(container.textContent).toContain('Todo')
    expect(container.textContent).toContain('2/3')
  })
})
