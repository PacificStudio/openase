import { describe, expect, it } from 'vitest'

import {
  createWorkflowHooksDraft,
  mergeWorkflowHooksPayload,
  parseWorkflowHooksDraft,
} from './workflow-hooks'

describe('workflow hooks draft', () => {
  it('parses workflow and ticket hook groups into API payloads', () => {
    const draft = createWorkflowHooksDraft({
      workflow_hooks: {
        on_activate: [{ cmd: 'claude --version', timeout: 30, on_failure: 'block' }],
      },
      ticket_hooks: {
        on_claim: [
          {
            cmd: 'pnpm install --frozen-lockfile',
            timeout: 300,
            on_failure: 'warn',
            workdir: 'web',
          },
        ],
      },
    })

    const parsed = parseWorkflowHooksDraft(draft)

    expect(parsed).toEqual({
      ok: true,
      value: {
        workflow_hooks: {
          on_activate: [{ cmd: 'claude --version', timeout: 30, on_failure: 'block' }],
        },
        ticket_hooks: {
          on_claim: [
            {
              cmd: 'pnpm install --frozen-lockfile',
              timeout: 300,
              on_failure: 'warn',
              workdir: 'web',
            },
          ],
        },
      },
      validation: {
        rowErrors: {},
        hasErrors: false,
        firstError: '',
      },
    })
  })

  it('rejects rows with missing commands or invalid timeouts', () => {
    const draft = createWorkflowHooksDraft()
    draft.workflowHooks.on_activate = [
      {
        id: 'row-1',
        cmd: '',
        timeout: 'soon',
        onFailure: 'block',
        workdir: '',
      },
    ]

    const parsed = parseWorkflowHooksDraft(draft)

    expect(parsed.ok).toBe(false)
    expect(parsed).toMatchObject({
      error: 'On activate: command is required.',
      validation: {
        hasErrors: true,
        rowErrors: {
          'row-1': {
            cmd: 'Command is required.',
            timeout: 'Timeout must be a whole number.',
          },
        },
      },
    })
  })

  it('preserves unsupported existing hook keys when merging supported edits', () => {
    const merged = mergeWorkflowHooksPayload(
      {
        workflow_hooks: {
          on_reload: [{ cmd: 'echo reload', on_failure: 'ignore' }],
        },
      },
      {
        workflow_hooks: {
          on_deactivate: [{ cmd: 'echo old', on_failure: 'warn' }],
        },
        ticket_hooks: {
          on_custom: [{ cmd: 'echo custom', on_failure: 'ignore' }],
        },
        audit: {
          enabled: true,
        },
      },
    )

    expect(merged).toEqual({
      workflow_hooks: {
        on_deactivate: [{ cmd: 'echo old', on_failure: 'warn' }],
        on_reload: [{ cmd: 'echo reload', on_failure: 'ignore' }],
      },
      ticket_hooks: {
        on_custom: [{ cmd: 'echo custom', on_failure: 'ignore' }],
      },
      audit: {
        enabled: true,
      },
    })
  })
})
