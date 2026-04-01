import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import WorkflowHookEventEditor from './workflow-hook-event-editor.svelte'

describe('WorkflowHookEventEditor', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('supports add, duplicate, and delete actions for an event', async () => {
    const onAdd = vi.fn()
    const onDuplicate = vi.fn()
    const onDelete = vi.fn()

    const { getByRole } = render(WorkflowHookEventEditor, {
      props: {
        label: 'On claim',
        description: 'Prepare the workspace.',
        rows: [
          {
            id: 'row-1',
            cmd: 'pnpm install',
            timeout: '300',
            onFailure: 'block',
            workdir: 'web',
          },
        ],
        allowWorkdir: true,
        onAdd,
        onChange: vi.fn(),
        onDuplicate,
        onDelete,
      },
    })

    await fireEvent.click(getByRole('button', { name: 'Add row' }))
    await fireEvent.click(getByRole('button', { name: 'Duplicate On claim row 1' }))
    await fireEvent.click(getByRole('button', { name: 'Delete On claim row 1' }))

    expect(onAdd).toHaveBeenCalledTimes(1)
    expect(onDuplicate).toHaveBeenCalledWith(0)
    expect(onDelete).toHaveBeenCalledWith(0)
  })
})
