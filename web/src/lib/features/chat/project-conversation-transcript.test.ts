import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'
import ProjectConversationTranscript from './project-conversation-transcript.svelte'

describe('ProjectConversationTranscript', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders structured runtime cards, interrupt details, and diff hunks', () => {
    const { getAllByText, getByText } = render(ProjectConversationTranscript, {
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

    expect(getByText('Tool call')).toBeTruthy()
    expect(getByText('functions.exec_command')).toBeTruthy()
    expect(getByText('Arguments')).toBeTruthy()
    expect(getByText('Command output')).toBeTruthy()
    expect(getByText((content) => content.includes('stdout'))).toBeTruthy()
    expect(getByText('@@ -1,1 +1,2 @@')).toBeTruthy()
    expect(getByText('+new line')).toBeTruthy()
    expect(getByText('Command approval required')).toBeTruthy()
    expect(getAllByText('command').length).toBeGreaterThan(0)
    expect(getByText('git status')).toBeTruthy()
    expect(getByText('Payload details')).toBeTruthy()
  })
})
