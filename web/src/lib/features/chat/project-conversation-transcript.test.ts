import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'
import ProjectConversationTranscript from './project-conversation-transcript.svelte'

describe('ProjectConversationTranscript', () => {
  afterEach(() => {
    cleanup()
  })

  it('groups system entries into a collapsed operation block and renders standalone entries', () => {
    const { getByText, getAllByText, queryByText } = render(ProjectConversationTranscript, {
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
            stream: 'command',
            phase: 'stdout',
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
    })

    // The 3 system entries (tool_call, task_status, command_output) are grouped
    // into a single collapsed operation block with a summary header
    expect(getByText('3 items')).toBeTruthy()

    // Collapsed by default — individual card content is NOT visible
    expect(queryByText('Task progress')).toBeNull()

    // Standalone entries are still rendered directly
    // Diff card
    expect(getAllByText('README.md').length).toBeGreaterThan(0)
    expect(getByText('+new line')).toBeTruthy()

    // Interrupt card
    expect(getByText('Command approval required')).toBeTruthy()
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

  it('shows pending indicator when pending is true', () => {
    const { getByText } = render(ProjectConversationTranscript, {
      props: {
        entries: [],
        pending: true,
      },
    })

    expect(getByText('Thinking...')).toBeTruthy()
  })
})
