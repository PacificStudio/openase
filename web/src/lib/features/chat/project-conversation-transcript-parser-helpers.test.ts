import { describe, expect, it } from 'vitest'

import {
  claudeFailureDetailFromPayload,
  describeClaudeResultFailure,
  mapLegacyClaudeFailureOutput,
} from './project-conversation-transcript-parser-helpers'

const GENERIC =
  "Claude couldn't finish this reply. Try sending your message again."

describe('claudeFailureDetailFromPayload', () => {
  it('maps bare error_during_execution to generic retry guidance', () => {
    const detail = claudeFailureDetailFromPayload({
      type: 'result',
      subtype: 'error_during_execution',
      is_error: true,
    })
    expect(detail).toBe(GENERIC)
    expect(detail).not.toContain('error_during_execution')
  })

  it('maps interrupted execution to session interrupted guidance', () => {
    const detail = claudeFailureDetailFromPayload({
      type: 'result',
      subtype: 'error_during_execution',
      is_error: true,
      terminal_reason: 'aborted_streaming',
      errors: [
        '[ede_diagnostic] result_type=user last_content_type=n/a stop_reason=tool_use',
        'Error: Request was aborted.',
      ],
    })
    expect(detail).toBe(
      "Claude couldn't finish this reply because the session was interrupted. Try sending your message again.",
    )
  })

  it('maps missing resume session to resume guidance', () => {
    const detail = claudeFailureDetailFromPayload({
      type: 'result',
      subtype: 'error_during_execution',
      is_error: true,
      errors: ['thread not found: claude-session-stale'],
    })
    expect(detail).toBe(
      "Claude couldn't resume the previous session. Try sending your message again, or start a new conversation if it keeps failing.",
    )
  })

  it('maps error subtype to pre-finish error guidance', () => {
    const detail = claudeFailureDetailFromPayload({
      type: 'result',
      subtype: 'error',
      is_error: true,
    })
    expect(detail).toBe(
      'Claude reported an error before this reply finished. Try sending your message again.',
    )
  })

  it('maps empty subtype with payload keys to generic retry guidance', () => {
    const detail = claudeFailureDetailFromPayload({
      type: 'result',
      is_error: true,
      session_id: 'claude-session-1',
    })
    expect(detail).toBe(GENERIC)
  })

  it('returns undefined when payload is not a Claude error result', () => {
    expect(
      claudeFailureDetailFromPayload({
        type: 'result',
        subtype: 'success',
      }),
    ).toBeUndefined()
    expect(claudeFailureDetailFromPayload(null)).toBeUndefined()
  })

  it('describeClaudeResultFailure matches claudeFailureDetailFromPayload', () => {
    const raw = {
      type: 'result',
      subtype: 'error_during_execution',
      is_error: true,
    }
    expect(describeClaudeResultFailure(raw)).toBe(claudeFailureDetailFromPayload(raw))
  })
})

describe('mapLegacyClaudeFailureOutput', () => {
  it('maps legacy empty error_during_execution output without exposing subtype', () => {
    const mapped = mapLegacyClaudeFailureOutput(
      'Claude Code reported an empty error_during_execution result.',
    )
    expect(mapped).toBe(GENERIC)
    expect(mapped).not.toContain('error_during_execution')
  })

  it('returns undefined for unknown output', () => {
    expect(mapLegacyClaudeFailureOutput('some other failure')).toBeUndefined()
  })
})