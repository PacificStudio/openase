import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'
import ProjectConversationTranscript from './project-conversation-transcript.svelte'

describe('ProjectConversationTranscript', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders command output directly while hiding noisy task status entries', async () => {
    const { getAllByRole, getByRole, getByText, getAllByText, queryByText } = render(
      ProjectConversationTranscript,
      {
        props: {
          entries: [
            {
              id: 'entry-tool',
              kind: 'tool_call',
              role: 'system',
              tool: 'functions.exec_command',
              arguments: { cmd: 'git status' },
            },
            {
              id: 'entry-status',
              kind: 'task_status',
              role: 'system',
              statusType: 'task_progress',
              title: 'Task progress',
              detail: 'Status: running',
              raw: {
                status: 'running',
                command: 'pnpm test',
                file: 'README.md',
                patch: '@@ -1 +1 @@\n-old line\n+new line',
              },
            },
            {
              id: 'entry-command',
              kind: 'command_output',
              role: 'system',
              stream: 'stdout',
              command: 'git status',
              phase: 'executing',
              snapshot: false,
              content: 'M web/src/app.ts\n',
            },
            {
              id: 'entry-diff',
              kind: 'diff',
              role: 'assistant',
              diff: {
                type: 'diff',
                file: 'README.md',
                hunks: [
                  {
                    oldStart: 1,
                    oldLines: 1,
                    newStart: 1,
                    newLines: 2,
                    lines: [
                      { op: 'context', text: 'old line' },
                      { op: 'add', text: 'new line' },
                    ],
                  },
                ],
              },
            },
            {
              id: 'entry-interrupt',
              kind: 'interrupt',
              role: 'system',
              interruptId: 'interrupt-1',
              provider: 'codex',
              interruptKind: 'command_execution_approval',
              payload: {
                command: 'git status',
                file: 'README.md',
                patch: '@@ -1,1 +1,2 @@\n-old line\n+new line',
              },
              options: [{ id: 'approve_once', label: 'Approve once' }],
              status: 'pending',
            },
          ],
        },
      },
    )

    // only tool calls remain grouped
    expect(getByText('1 item')).toBeTruthy()

    // command_output is rendered directly with the runtime-style command card
    expect(getAllByRole('button', { name: /git status/i })).toHaveLength(2)
    expect(getByText('executing')).toBeTruthy()

    // noisy task status entries are hidden entirely
    expect(queryByText('Task progress')).toBeNull()

    // Standalone entries are still rendered directly
    expect(getAllByText('README.md').length).toBeGreaterThan(0)
    expect(queryByText('+new line')).toBeNull()
    await fireEvent.click(getByRole('button', { name: /README\.md/i }))
    expect(getByText('+new line')).toBeTruthy()
    expect(getByText('Command approval')).toBeTruthy()
    expect(getByText('Approve once')).toBeTruthy()
  })

  it('renders text messages with correct styling', () => {
    const { getByText } = render(ProjectConversationTranscript, {
      props: {
        entries: [
          {
            id: 'entry-user',
            kind: 'text',
            role: 'user',
            content: 'Hello, AI!',
            streaming: false,
          },
          {
            id: 'entry-assistant',
            kind: 'text',
            role: 'assistant',
            content: 'Hello! How can I help?',
            streaming: false,
          },
        ],
      },
    })

    expect(getByText('Hello, AI!')).toBeTruthy()
    expect(getByText('Hello! How can I help?')).toBeTruthy()
  })

  it('rerenders when assistant text entries arrive after the initial render', async () => {
    const { findByText, rerender } = render(ProjectConversationTranscript, {
      props: {
        entries: [
          {
            id: 'entry-user',
            kind: 'text',
            role: 'user',
            content: 'Hello, AI!',
            streaming: false,
          },
        ],
      },
    })

    expect(await findByText('Hello, AI!')).toBeTruthy()

    await rerender({
      entries: [
        {
          id: 'entry-user',
          kind: 'text',
          role: 'user',
          content: 'Hello, AI!',
          streaming: false,
        },
        {
          id: 'entry-assistant',
          kind: 'text',
          role: 'assistant',
          content: 'First streamed reply chunk.',
          streaming: true,
        },
      ],
    })

    expect(await findByText('First streamed reply chunk.')).toBeTruthy()
  })

  it('shows pending indicator when pending is true', () => {
    const { getByText } = render(ProjectConversationTranscript, {
      props: {
        entries: [],
        pending: true,
      },
    })

    expect(getByText('Thinking...')).toBeTruthy()
  })

  it('hides codex thread and task status noise while keeping higher-signal statuses', async () => {
    const { getByRole, getByText, queryByText } = render(ProjectConversationTranscript, {
      props: {
        entries: [
          {
            id: 'entry-thread-status',
            kind: 'task_status',
            role: 'system',
            statusType: 'thread_status',
            title: 'Codex thread status',
            detail: 'waitingOnApproval · waitingOnApproval',
            raw: {
              anchor_kind: 'thread',
              thread_id: 'thread-1',
              status: 'waitingOnApproval',
              active_flags: ['waitingOnApproval'],
            },
          },
          {
            id: 'entry-task-started',
            kind: 'task_status',
            role: 'system',
            statusType: 'task_started',
            title: 'Task started',
            detail: 'Status: inProgress',
            raw: {
              status: 'inProgress',
            },
          },
          {
            id: 'entry-session-state',
            kind: 'task_status',
            role: 'system',
            statusType: 'session_state',
            title: 'Claude session status',
            detail: 'requires_action · approval required · requires_action',
            raw: {
              anchor_kind: 'session',
              status: 'requires_action',
              detail: 'approval required',
              active_flags: ['requires_action'],
            },
          },
        ],
      },
    })

    await fireEvent.click(getByRole('button', { name: /Claude session status/i }))

    expect(queryByText('Codex thread status')).toBeNull()
    expect(queryByText('Task started')).toBeNull()
    expect(getByText('Claude session status')).toBeTruthy()
    expect(getByText('requires_action · approval required · requires_action')).toBeTruthy()
  })

  it('uses the command as the visible label for command output cards', async () => {
    const { getByRole, getAllByText } = render(ProjectConversationTranscript, {
      props: {
        entries: [
          {
            id: 'entry-command',
            kind: 'command_output',
            role: 'system',
            stream: 'command',
            command: 'pnpm vitest run',
            snapshot: false,
            content: 'PASS\n',
          },
        ],
      },
    })

    await fireEvent.click(getByRole('button', { name: /pnpm vitest run/i }))

    expect(getAllByText('pnpm vitest run').length).toBeGreaterThan(0)
  })

  it('folds long command output through the shared runtime-style truncation flow', async () => {
    const content = Array.from({ length: 18 }, (_, index) => `line ${index + 1}`).join('\n')
    const { getByRole, queryByText, getByText } = render(ProjectConversationTranscript, {
      props: {
        entries: [
          {
            id: 'entry-command',
            kind: 'command_output',
            role: 'system',
            stream: 'stdout',
            command: 'pnpm test',
            snapshot: false,
            content,
          },
        ],
      },
    })

    await fireEvent.click(getByRole('button', { name: /pnpm test/i }))

    expect(getByRole('button', { name: /\+8 lines hidden/i })).toBeTruthy()
    expect(queryByText('line 10')).toBeNull()

    await fireEvent.click(getByRole('button', { name: /\+8 lines hidden/i }))

    expect(getByText(/line 10/)).toBeTruthy()
    expect(getByRole('button', { name: /collapse output/i })).toBeTruthy()
  })
})
