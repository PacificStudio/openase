import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

import WorkflowHookRowEditor from './workflow-hook-row-editor.svelte'

describe('WorkflowHookRowEditor', () => {
  afterEach(() => {
    cleanup()
  })

  it('shows workdir only for ticket hook rows', () => {
    const row = {
      id: 'row-1',
      cmd: 'pnpm install',
      timeout: '300',
      onFailure: 'block' as const,
      workdir: 'web',
    }

    const workflowRow = render(WorkflowHookRowEditor, {
      props: {
        row,
        title: 'On activate row 1',
      },
    })
    expect(workflowRow.queryByLabelText('Workdir')).toBeNull()
    workflowRow.unmount()

    const ticketRow = render(WorkflowHookRowEditor, {
      props: {
        row,
        title: 'On claim row 1',
        allowWorkdir: true,
      },
    })
    expect(ticketRow.getByLabelText('Workdir')).toBeTruthy()
  })
})
